"""Module for handling realtime (group chat) communication phases.

リアルタイム（グループチャット方式）通信フェーズを処理するモジュール.

サーバからTALK_START/WHISPER_STARTを受信すると、このモジュールが
非同期的にブロードキャストの受信・LLMへの発言判断問い合わせ・発言送信を行う。
TALK_END/WHISPER_ENDを受信するとフェーズを終了し、制御を呼び出し元に返す。
"""

from __future__ import annotations

import logging
import threading
import time
from dataclasses import dataclass, field
from queue import Empty, Queue
from typing import TYPE_CHECKING

from aiwolf_nlp_common.client import Client
from aiwolf_nlp_common.packet import Packet, Request, Talk

if TYPE_CHECKING:
    from agent.agent import Agent

logger = logging.getLogger(__name__)

# リアルタイムリクエストの集合
REALTIME_REQUESTS = {
    Request.TALK_START,
    Request.TALK_BROADCAST,
    Request.TALK_END,
    Request.WHISPER_START,
    Request.WHISPER_BROADCAST,
    Request.WHISPER_END,
}

# LLMの判断結果: 聞くだけ（発言しない）
LISTEN = "LISTEN"


def is_realtime_request(request: Request) -> bool:
    """Check if the request is a realtime communication request.

    リクエストがリアルタイム通信リクエストかどうかを判定する.
    """
    return request in REALTIME_REQUESTS


@dataclass
class RealtimePhaseState:
    """State for a realtime communication phase.

    リアルタイム通信フェーズの状態管理.
    """

    is_talk: bool  # True=talk, False=whisper
    remain_count: int = 0
    talks_in_phase: list[Talk] = field(default_factory=list)
    ended: bool = False
    last_speak_time: float = 0.0


class RealtimeHandler:
    """Handler for realtime communication phases.

    リアルタイム通信フェーズのハンドラ.
    WebSocketからのブロードキャスト受信とLLMによる発言判断を管理する.
    """

    def __init__(
        self,
        client: Client,
        agent: Agent,
        poll_interval: float = 0.5,
        speak_cooldown: float = 3.0,
    ) -> None:
        """Initialize the realtime handler.

        Args:
            client: Client connected to the server.
            agent: The agent instance.
            poll_interval: How often to check for new messages and decide to speak (seconds).
            speak_cooldown: Minimum time between consecutive speaks (seconds).
        """
        self.client = client
        self.agent = agent
        self.poll_interval = poll_interval
        self.speak_cooldown = speak_cooldown
        self._recv_queue: Queue[Packet] = Queue()
        self._stop_event = threading.Event()
        self._recv_thread: threading.Thread | None = None
        self._send_lock = threading.Lock()

    def handle_phase(self, start_packet: Packet) -> Packet | None:
        """Handle a complete realtime communication phase.

        リアルタイム通信フェーズ全体を処理する.
        TALK_START/WHISPER_STARTのパケットを受け取り、
        TALK_END/WHISPER_ENDを受信するまでループする.

        Args:
            start_packet: The START packet.

        Returns:
            The next non-realtime Packet (if received after END), or None.
        """
        is_talk = start_packet.request == Request.TALK_START
        end_request = Request.TALK_END if is_talk else Request.WHISPER_END

        state = RealtimePhaseState(is_talk=is_talk)

        # START パケットから初期状態を取得
        self._apply_packet(start_packet, state)

        self.agent.agent_logger.logger.info(
            "リアルタイムフェーズ開始: %s, remain_count=%d",
            "TALK" if is_talk else "WHISPER",
            state.remain_count,
        )

        # 受信スレッドを起動
        self._stop_event.clear()
        self._recv_thread = threading.Thread(target=self._receiver_loop, daemon=True)
        self._recv_thread.start()

        next_packet: Packet | None = None

        try:
            # メインの発言判断ループ
            while not state.ended:
                # キューからパケットを処理
                new_packets = self._drain_queue()
                for packet in new_packets:
                    if packet.request == end_request:
                        state.ended = True
                        self.agent.agent_logger.logger.info("リアルタイムフェーズ終了通知を受信しました")
                        break
                    if packet.request in (Request.TALK_BROADCAST, Request.WHISPER_BROADCAST):
                        self._apply_packet(packet, state)
                    elif not is_realtime_request(packet.request):
                        # リアルタイム以外のパケット（予期しないが安全に処理）
                        next_packet = packet
                        state.ended = True
                        break

                if state.ended:
                    break

                # 発言判断
                if state.remain_count > 0 and self._should_consider_speaking(state):
                    response = self._decide_and_speak(state)
                    if response == "Over":
                        with self._send_lock:
                            self.client.send("Over")
                        self.agent.agent_logger.logger.info("Overを送信しました")
                        # Over後もブロードキャストは受信し続ける（END まで）
                        state.remain_count = 0
                    elif response and response != LISTEN:
                        with self._send_lock:
                            self.client.send(response)
                        state.remain_count -= 1
                        state.last_speak_time = time.time()
                        self.agent.agent_logger.logger.info(
                            "リアルタイム発言: %s (残り%d回)",
                            response,
                            state.remain_count,
                        )

                # 短い待機
                time.sleep(self.poll_interval)

        finally:
            # 受信スレッドを停止
            self._stop_event.set()
            if self._recv_thread is not None:
                self._recv_thread.join(timeout=3.0)

        # エージェントのカウンタを更新（従来モードのプロンプトとの互換性のため）
        if is_talk:
            self.agent.sent_talk_count = len(self.agent.talk_history)
        else:
            self.agent.sent_whisper_count = len(self.agent.whisper_history)

        return next_packet

    def _apply_packet(self, packet: Packet, state: RealtimePhaseState) -> None:
        """Apply a packet's data to agent and state.

        START パケット（フルinfo）と BROADCAST パケット（最小info）の両方に対応する.
        BROADCASTのinfoは game_id, day, agent, remain_count のみで、
        status_map や role_map を持たないため、フルinfoを上書きしてはいけない.
        """
        # info を更新
        if packet.info:
            if packet.info.remain_count is not None:
                state.remain_count = packet.info.remain_count

            # status_map が存在する → フルinfo（STARTパケット）→ 丸ごと設定
            # status_map が空 → 最小info（BROADCASTパケット）→ remain_count のみ更新
            if packet.info.status_map:
                self.agent.info = packet.info
            elif self.agent.info is not None:
                self.agent.info.remain_count = packet.info.remain_count
            else:
                self.agent.info = packet.info

        # setting を更新
        if packet.setting:
            self.agent.setting = packet.setting

        # トーク/ウィスパー履歴を追加
        talks = packet.talk_history if state.is_talk else packet.whisper_history
        if talks:
            for talk in talks:
                if state.is_talk:
                    self.agent.talk_history.append(talk)
                else:
                    self.agent.whisper_history.append(talk)
                state.talks_in_phase.append(talk)

                self.agent.agent_logger.logger.info(
                    "ブロードキャスト受信: %s: %s",
                    talk.agent,
                    talk.text,
                )

    def _should_consider_speaking(self, state: RealtimePhaseState) -> bool:
        """Check if enough time has passed since last speak to consider speaking again."""
        if state.remain_count <= 0:
            return False
        elapsed = time.time() - state.last_speak_time
        return elapsed >= self.speak_cooldown

    def _decide_and_speak(self, state: RealtimePhaseState) -> str | None:
        """Ask the LLM whether to speak and what to say.

        LLMに発言判断を問い合わせる.

        Returns:
            The message to send, "Over", LISTEN, or None.
        """
        return self.agent.talk_realtime(state.talks_in_phase, state.is_talk, state.remain_count)

    def _drain_queue(self) -> list[Packet]:
        """Drain all pending packets from the receive queue."""
        packets: list[Packet] = []
        while True:
            try:
                packets.append(self._recv_queue.get_nowait())
            except Empty:
                break
        return packets

    def _receiver_loop(self) -> None:
        """Background thread: receive packets from server and enqueue them.

        client.receive() を短いタイムアウトで呼び出し、Packetをキューに入れる.
        """
        try:
            self.client.socket.settimeout(0.5)
        except Exception:  # noqa: BLE001
            pass

        while not self._stop_event.is_set():
            try:
                packet = self.client.receive()
                self._recv_queue.put(packet)
            except Exception:  # noqa: BLE001
                # タイムアウトやその他のエラー → 継続
                if self._stop_event.is_set():
                    break
                continue

        # ソケットのタイムアウトをクリア（メインスレッドに制御を返す前に）
        try:
            self.client.socket.settimeout(None)
        except Exception:  # noqa: BLE001
            pass
