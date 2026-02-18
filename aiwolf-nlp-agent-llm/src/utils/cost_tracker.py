"""Module for tracking LLM token usage and costs.

LLMのトークン使用量とコストを追跡するモジュール.
"""

from __future__ import annotations

import logging
import urllib.parse
from dataclasses import dataclass, field
from typing import Any

import requests

logger = logging.getLogger(__name__)

# モデルごとの料金表 (USD per 1M tokens)
# https://openai.com/api/pricing/
# https://ai.google.dev/pricing
COST_TABLE: dict[str, tuple[float, float]] = {
    # OpenAI (input, output) per 1M tokens
    "gpt-4o": (2.50, 10.00),
    "gpt-4o-mini": (0.15, 0.60),
    "gpt-4o-2024-11-20": (2.50, 10.00),
    "gpt-4.1": (2.00, 8.00),
    "gpt-4.1-mini": (0.40, 1.60),
    "gpt-4.1-nano": (0.10, 0.40),
    "o3-mini": (1.10, 4.40),
    "o4-mini": (1.10, 4.40),
    # Google Gemini
    "gemini-2.0-flash": (0.10, 0.40),
    "gemini-2.0-flash-lite": (0.02, 0.10),
    "gemini-2.5-flash": (0.15, 0.60),
    "gemini-2.5-pro": (1.25, 10.00),
    "gemini-1.5-flash": (0.075, 0.30),
    "gemini-1.5-pro": (1.25, 5.00),
    # Ollama (ローカル実行のため無料)
    "ollama": (0.0, 0.0),
}


@dataclass
class TokenUsage:
    """Single LLM call token usage record.

    1回のLLM呼び出しのトークン使用量レコード.
    """

    input_tokens: int = 0
    output_tokens: int = 0
    input_cost: float = 0.0
    output_cost: float = 0.0


@dataclass
class CostTracker:
    """Tracks cumulative token usage and costs for an agent.

    エージェントの累積トークン使用量とコストを追跡する.
    """

    model_name: str = ""
    llm_type: str = ""
    total_input_tokens: int = 0
    total_output_tokens: int = 0
    total_input_cost: float = 0.0
    total_output_cost: float = 0.0
    call_count: int = 0
    history: list[TokenUsage] = field(default_factory=list)

    def configure(self, llm_type: str, model_name: str) -> None:
        """Configure the tracker with model information.

        モデル情報でトラッカーを設定する.

        Args:
            llm_type: LLM provider type / LLMプロバイダタイプ
            model_name: Model name / モデル名
        """
        self.llm_type = llm_type
        self.model_name = model_name

    def _get_cost_per_million(self) -> tuple[float, float]:
        """Get cost per 1M tokens for the current model.

        現在のモデルの100万トークンあたりのコストを取得する.

        Returns:
            (input_cost_per_1M, output_cost_per_1M)
        """
        if self.llm_type == "ollama":
            return (0.0, 0.0)
        # 完全一致 → プレフィックス一致で検索
        if self.model_name in COST_TABLE:
            return COST_TABLE[self.model_name]
        for key, val in COST_TABLE.items():
            if self.model_name.startswith(key):
                return val
        logger.warning("コスト表にモデル %s が見つかりません。0として計算します。", self.model_name)
        return (0.0, 0.0)

    def track(self, response_metadata: dict[str, Any] | None) -> TokenUsage:
        """Track token usage from a single LLM response.

        1回のLLM応答からトークン使用量を追跡する.

        Args:
            response_metadata: Response metadata from langchain AIMessage / langchain AIMessageのレスポンスメタデータ

        Returns:
            TokenUsage: Token usage for this call / この呼び出しのトークン使用量
        """
        usage = TokenUsage()
        if not response_metadata:
            self.call_count += 1
            self.history.append(usage)
            return usage

        # OpenAI形式
        token_usage = response_metadata.get("token_usage") or response_metadata.get("usage", {})
        if isinstance(token_usage, dict):
            usage.input_tokens = token_usage.get("prompt_tokens", 0) or token_usage.get("input_tokens", 0)
            usage.output_tokens = token_usage.get("completion_tokens", 0) or token_usage.get("output_tokens", 0)

        # Google Gemini形式
        usage_metadata = response_metadata.get("usage_metadata", {})
        if isinstance(usage_metadata, dict):
            usage.input_tokens = usage.input_tokens or usage_metadata.get("prompt_token_count", 0) or usage_metadata.get("input_tokens", 0)
            usage.output_tokens = usage.output_tokens or usage_metadata.get("candidates_token_count", 0) or usage_metadata.get("output_tokens", 0)

        # コスト計算
        input_per_m, output_per_m = self._get_cost_per_million()
        usage.input_cost = (usage.input_tokens / 1_000_000) * input_per_m
        usage.output_cost = (usage.output_tokens / 1_000_000) * output_per_m

        # 累積
        self.total_input_tokens += usage.input_tokens
        self.total_output_tokens += usage.output_tokens
        self.total_input_cost += usage.input_cost
        self.total_output_cost += usage.output_cost
        self.call_count += 1
        self.history.append(usage)

        return usage

    @property
    def total_cost(self) -> float:
        """Total cost in USD. / 合計コスト（USD）."""
        return self.total_input_cost + self.total_output_cost

    @property
    def total_tokens(self) -> int:
        """Total tokens used. / 合計トークン使用量."""
        return self.total_input_tokens + self.total_output_tokens

    def to_report(self, game_id: str, agent_name: str, team_name: str) -> dict[str, Any]:
        """Generate a cost report dict for sending to server.

        サーバ送信用のコストレポート辞書を生成する.

        Args:
            game_id: Game ID / ゲームID
            agent_name: Agent name (e.g. Agent[01]) / エージェント名
            team_name: Team name / チーム名

        Returns:
            dict: Cost report payload / コストレポートのペイロード
        """
        return {
            "game_id": game_id,
            "agent": agent_name,
            "team": team_name,
            "model": self.model_name,
            "llm_type": self.llm_type,
            "input_tokens": self.total_input_tokens,
            "output_tokens": self.total_output_tokens,
            "input_cost": round(self.total_input_cost, 8),
            "output_cost": round(self.total_output_cost, 8),
            "total_cost": round(self.total_cost, 8),
            "call_count": self.call_count,
        }

    def report_to_server(self, ws_url: str, game_id: str, agent_name: str, team_name: str) -> None:
        """Send cost report to the server via HTTP POST.

        HTTP POSTでサーバにコストレポートを送信する.

        Args:
            ws_url: WebSocket URL (e.g. ws://localhost:8080/ws) / WebSocket URL
            game_id: Game ID / ゲームID
            agent_name: Agent name / エージェント名
            team_name: Team name / チーム名
        """
        try:
            parsed = urllib.parse.urlparse(ws_url)
            scheme = "https" if parsed.scheme == "wss" else "http"
            http_base = f"{scheme}://{parsed.hostname}"
            if parsed.port:
                http_base += f":{parsed.port}"
            url = f"{http_base}/api/cost/report"

            payload = self.to_report(game_id, agent_name, team_name)
            resp = requests.post(url, json=payload, timeout=5)

            if resp.status_code == 200:
                logger.info("コストレポートを送信しました: %s ($%.6f)", agent_name, self.total_cost)
            else:
                logger.warning("コストレポートの送信に失敗しました: HTTP %d", resp.status_code)
        except Exception:
            logger.warning("コストレポートの送信に失敗しました (接続エラー)")
