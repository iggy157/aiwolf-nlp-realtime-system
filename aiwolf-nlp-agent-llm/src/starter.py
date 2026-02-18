"""Module for launching agents.

エージェントを起動するためのモジュール.
従来のターン制プロトコルとリアルタイム通信プロトコルの両方に対応する.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING, Any

from utils.agent_utils import init_agent_from_packet
from utils.realtime_handler import RealtimeHandler, is_realtime_request

if TYPE_CHECKING:
    from agent.agent import Agent

from time import sleep

from aiwolf_nlp_common.client import Client
from aiwolf_nlp_common.packet import Request

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)
console_handler = logging.StreamHandler()
formatter = logging.Formatter(
    "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
console_handler.setFormatter(formatter)
logger.addHandler(console_handler)


def create_client(config: dict[str, Any]) -> Client:
    """Create a client.

    クライアントの作成.

    Args:
        config (dict[str, Any]): Configuration dictionary / 設定辞書

    Returns:
        Client: Created client instance / 作成されたクライアントインスタンス
    """
    return Client(
        url=str(config["web_socket"]["url"]),
        token=(str(config["web_socket"]["token"]) if config["web_socket"]["token"] else None),
    )


def connect_to_server(client: Client, name: str) -> None:
    """Handle connection to the server.

    サーバーへの接続処理.

    Args:
        client (Client): Client instance / クライアントインスタンス
        name (str): Agent name / エージェント名
    """
    while True:
        try:
            client.connect()
            logger.info("エージェント %s がゲームサーバに接続しました", name)
            break
        except Exception as ex:  # noqa: BLE001
            logger.warning(
                "エージェント %s がゲームサーバに接続できませんでした",
                name,
            )
            logger.warning(ex)
            logger.info("再接続を試みます")
            sleep(15)


def handle_game_session(
    client: Client,
    config: dict[str, Any],
    name: str,
) -> None:
    """Handle game session.

    ゲームセッションの処理.
    従来のターン制とリアルタイム通信の両方に対応する.

    Args:
        client (Client): Client instance / クライアントインスタンス
        config (dict[str, Any]): Configuration dictionary / 設定辞書
        name (str): Agent name / エージェント名
    """
    agent: Agent | None = None
    realtime_handler: RealtimeHandler | None = None

    while True:
        packet = client.receive()

        # リアルタイム通信リクエストの場合
        if is_realtime_request(packet.request):
            if not agent:
                raise ValueError("エージェントが初期化されていません")
            # STARTリクエストの場合、リアルタイムハンドラに委譲
            if packet.request in (Request.TALK_START, Request.WHISPER_START):
                if realtime_handler is None:
                    realtime_handler = RealtimeHandler(
                        client=client,
                        agent=agent,
                        poll_interval=float(config.get("realtime", {}).get("poll_interval", 0.5)),
                        speak_cooldown=float(config.get("realtime", {}).get("speak_cooldown", 3.0)),
                    )
                next_packet = realtime_handler.handle_phase(packet)
                # リアルタイムフェーズ中に非リアルタイムパケットを受信した場合
                if next_packet is not None:
                    packet = next_packet
                    # 下の通常処理に落とす
                else:
                    continue
            else:
                # BROADCAST/END が直接来た場合（通常はハンドラ内で処理済み）
                logger.warning("予期しないリアルタイムリクエストをスキップします: %s", packet.request)
                continue

        # 通常のリクエスト処理（従来のターン制プロトコル）
        if packet.request == Request.NAME:
            client.send(name)
            continue
        if packet.request == Request.INITIALIZE:
            agent = init_agent_from_packet(config, name, packet)
            realtime_handler = None  # 新しいゲームごとにリセット
        if not agent:
            raise ValueError("エージェントが初期化されていません")
        agent.set_packet(packet)
        req = agent.action()
        agent.agent_logger.packet(agent.request, req)
        if req:
            client.send(req)
        if packet.request == Request.FINISH:
            # ゲーム終了時にコストレポートをサーバに送信
            if agent.info and agent.cost_tracker.call_count > 0:
                agent.cost_tracker.report_to_server(
                    ws_url=str(config["web_socket"]["url"]),
                    game_id=agent.info.game_id,
                    agent_name=agent.info.agent,
                    team_name=str(config["agent"]["team"]),
                )
                logger.info(
                    "エージェント %s のコスト: $%.6f (入力: %d tokens, 出力: %d tokens, %d calls)",
                    name,
                    agent.cost_tracker.total_cost,
                    agent.cost_tracker.total_input_tokens,
                    agent.cost_tracker.total_output_tokens,
                    agent.cost_tracker.call_count,
                )
            break


def connect(config: dict[str, Any], idx: int = 1) -> None:
    """Launch an agent.

    エージェントを起動する.

    Args:
        config (dict[str, Any]): Configuration dictionary / 設定辞書
        idx (int): Agent index (default: 1) / エージェントインデックス (デフォルト: 1)
    """
    name = str(config["agent"]["team"]) + str(idx)
    while True:
        try:
            client = create_client(config)
            connect_to_server(client, name)
            try:
                handle_game_session(client, config, name)
            finally:
                client.close()
                logger.info("エージェント %s とゲームサーバの接続を切断しました", name)
        except Exception as ex:  # noqa: BLE001
            logger.warning(
                "エージェント %s がエラーで終了しました",
                name,
            )
            logger.warning(ex)

        if not bool(config["web_socket"]["auto_reconnect"]):
            break
