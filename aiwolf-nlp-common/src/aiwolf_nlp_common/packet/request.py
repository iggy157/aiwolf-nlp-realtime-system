from __future__ import annotations

from enum import Enum


class Request(str, Enum):
    """リクエストの種類を示す列挙型.

    Attributes:
        NAME (str): 名前リクエスト.
        TALK (str): トークリクエスト.
        WHISPER (str): 囁きリクエスト.
        VOTE (str): 投票リクエスト.
        DIVINE (str): 占いリクエスト.
        GUARD (str): 護衛リクエスト.
        ATTACK (str): 襲撃リクエスト.
        INITIALIZE (str): ゲーム開始リクエスト.
        DAILY_INITIALIZE (str): 昼開始リクエスト.
        DAILY_FINISH (str): 昼終了リクエスト.
        FINISH (str): ゲーム終了リクエスト.
        TALK_START (str): リアルタイムトーク開始リクエスト.
        TALK_BROADCAST (str): リアルタイムトークブロードキャスト.
        TALK_END (str): リアルタイムトーク終了リクエスト.
        WHISPER_START (str): リアルタイム囁き開始リクエスト.
        WHISPER_BROADCAST (str): リアルタイム囁きブロードキャスト.
        WHISPER_END (str): リアルタイム囁き終了リクエスト.
    """

    NAME = "NAME"
    TALK = "TALK"
    WHISPER = "WHISPER"
    VOTE = "VOTE"
    DIVINE = "DIVINE"
    GUARD = "GUARD"
    ATTACK = "ATTACK"
    INITIALIZE = "INITIALIZE"
    DAILY_INITIALIZE = "DAILY_INITIALIZE"
    DAILY_FINISH = "DAILY_FINISH"
    FINISH = "FINISH"
    TALK_START = "TALK_START"
    TALK_BROADCAST = "TALK_BROADCAST"
    TALK_END = "TALK_END"
    WHISPER_START = "WHISPER_START"
    WHISPER_BROADCAST = "WHISPER_BROADCAST"
    WHISPER_END = "WHISPER_END"