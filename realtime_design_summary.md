# リアルタイム通信対応 — 全体設計書

## 1. 背景と目的

従来のターン制トークでは、13人ゲームにおいてAgent Aが発言した後、Agent Bが「A、どう思う？」と聞いても、残り11人が順番に発言し終えるまでAは返答できない。

リアルタイム通信では、全エージェントがグループチャットのように自由なタイミングで発言でき、質問への即座の応答や自然な議論が可能になる。

---

## 2. リポジトリ別 修正内容

### 2.1 aiwolf-nlp-server（Go）

サーバ側。メッセージブローカーとして動作し、発言のブロードキャストと安全制限の適用を行う。

| ファイル | 種別 | 内容 |
|---|---|---|
| `logic/realtime_communication.go` | **新規** | リアルタイム通信のメインロジック（399行） |
| `logic/communication.go` | **変更** | `doTalk()` / `doWhisper()` に `if realtime.enable` 分岐追加（6行） |
| `model/request.go` | **変更** | 6つの新リクエストタイプ定数と `RequestFromString()` 更新 |
| `model/config.go` | **変更** | `RealtimeConfig` 構造体追加、`GameConfig.Realtime` フィールド追加 |
| `model/agent.go` | **変更** | `SendNonBlocking()` メソッド追加（応答不要のブロードキャスト送信用） |
| `config/realtime_5.yml` | **新規** | リアルタイムモード用サンプル設定 |
| `doc/ja/realtime_protocol.md` | **新規** | プロトコル仕様書 |

**サーバ側の設定項目：**
```yaml
game:
  realtime:
    enable: true           # true でリアルタイムモード有効
    phase_timeout: 120s    # フェーズ全体の制限時間
    silence_timeout: 15s   # 全員沈黙時の自動終了
    rate_limit: 2s         # 1エージェントの最小発言間隔
  talk:
    max_count:
      per_agent: 10        # 1エージェントあたりの発言上限
      per_day: 50          # 全体の発言合計上限
    max_length:
      per_talk: 200        # 1発言の最大文字数
```

**サーバ側の安全機構：**
- レートリミット（スパム防止）
- 発言回数上限（per_agent / per_day）
- 文字数制限（超過分は切り捨て）
- フェーズタイムアウト / サイレンスタイムアウト
- ゼロ値フォールバック（未設定時にデフォルト値を適用）
- stale message ドレイン（フェーズ終了後のバッファ残留メッセージを読み捨て）

---

### 2.2 aiwolf-nlp-common（Python）

共通パッケージ。サーバとエージェント間のプロトコル定義。

| ファイル | 種別 | 内容 |
|---|---|---|
| `src/aiwolf_nlp_common/packet/request.py` | **変更** | `Request` enum に6値追加 |
| `src/aiwolf_nlp_common/packet/request.pyi` | **変更** | 型スタブを同期 |

**追加された enum 値：**
```python
TALK_START = "TALK_START"
TALK_BROADCAST = "TALK_BROADCAST"
TALK_END = "TALK_END"
WHISPER_START = "WHISPER_START"
WHISPER_BROADCAST = "WHISPER_BROADCAST"
WHISPER_END = "WHISPER_END"
```

これにより `Packet.from_dict()` が新リクエストタイプを自動的にパースでき、エージェント側で生JSON解析が不要になる。

---

### 2.3 aiwolf-nlp-agent-llm（Python）

エージェント側。LLMを使った「話すべきか、聞くべきか」の判断が核心。

| ファイル | 種別 | 内容 |
|---|---|---|
| `src/utils/realtime_handler.py` | **新規** | リアルタイムフェーズのハンドラ（266行） |
| `src/starter.py` | **変更** | リアルタイムリクエスト検出 → ハンドラ委譲の分岐追加 |
| `src/agent/agent.py` | **変更** | `talk_realtime()` メソッド追加（約70行） |
| `config/config.realtime.yml.example` | **新規** | リアルタイム用設定（`talk_realtime` プロンプト付き） |

**エージェント側の設定項目：**
```yaml
realtime:
  poll_interval: 0.5     # ブロードキャスト確認間隔（秒）
  speak_cooldown: 3.0    # 連続発言の最小間隔（秒）
```

---

## 3. プロトコルフロー

### 3.1 ゲーム全体の流れ（変更なしの部分含む）

```
NAME           ← 従来通り（同期リクエスト/レスポンス）
INITIALIZE     ← 従来通り
│
├── Day 0
│   DAILY_INITIALIZE  ← 従来通り
│   TALK_START  ────── ★ リアルタイムフェーズ（ここが新規）
│   TALK_END    ──────
│   DAILY_FINISH      ← 従来通り
│   VOTE              ← 従来通り（同期）
│   DIVINE / GUARD    ← 従来通り（同期）
│   ATTACK            ← 従来通り（同期）
│
├── Day 1 ...
│
FINISH         ← 従来通り
```

リアルタイム化されるのは **TALK/WHISPER フェーズのみ**。
投票・占い・護衛・襲撃は従来通りの同期リクエスト/レスポンス。

---

### 3.2 リアルタイムフェーズの詳細フロー

```
                    サーバ                                      エージェント
                      │                                              │
                      │──── TALK_START ─────────────────────────────→│
                      │     {request, info, setting, talk_history}   │
                      │                                              │
                      │                                   ┌──────────┤
                      │                                   │ 受信スレッド起動
                      │                                   │ LLM発言判断ループ開始
                      │                                   └──────────┤
                      │                                              │
              ┌───────│←───── "おはようございます" ───────────────────│ Agent[01] が発言
              │       │                                              │
              │ ブロード│──── TALK_BROADCAST ────────────────────────→│ 全員
              │ キャスト│     {talk_history: [{agent:"Agent[01]",     │
              │       │       text:"おはようございます"}]}            │
              │       │                                              │
              └───────│                                              │
                      │                                   ┌──────────┤
                      │                                   │ LLMに問い合わせ:
                      │                                   │ 「今話すべき？」
                      │                                   │  → 発言内容 or LISTEN
                      │                                   └──────────┤
                      │                                              │
                      │←───── "私は占い師です" ──────────────────────│ Agent[03] が発言
                      │                                              │
                      │──── TALK_BROADCAST ────────────────────────→│ 全員
                      │                                              │
                      │←───── "Agent[03]、証拠は？" ────────────────│ Agent[02] が発言
                      │                                              │
                      │──── TALK_BROADCAST ────────────────────────→│ 全員
                      │                                              │
                      │←───── "Agent[05]を占って人狼でした" ────────│ Agent[03] が即座に返答
                      │                                              │
                      │──── TALK_BROADCAST ────────────────────────→│ 全員
                      │                                              │
                      │           ...（会話が続く）...                │
                      │                                              │
                      │     ┌─ 終了条件 ──────────────────┐         │
                      │     │ ・全員が "Over" を送信       │         │
                      │     │ ・phase_timeout (120s) 到達  │         │
                      │     │ ・silence_timeout (15s) 到達 │         │
                      │     │ ・per_day 上限到達           │         │
                      │     └─────────────────────────────┘         │
                      │                                              │
                      │──── TALK_END ───────────────────────────────→│
                      │                                              │
                      │     ┌─ stale message ドレイン ────┐         │
                      │     │ バッファ残留メッセージを      │         │
                      │     │ 読み捨て（最大2秒）          │         │
                      │     └─────────────────────────────┘         │
                      │                                              │
                      │──── VOTE ──────────────────────────────────→│  従来の同期モードに戻る
                      │←───── "Agent[05]" ──────────────────────────│
                      │                                              │
```

---

### 3.3 エージェント内部の処理フロー

```
starter.py
  │
  │ client.receive()
  │    → Packet(request=TALK_START)
  │
  │ is_realtime_request(packet.request) == True
  │
  └─→ RealtimeHandler.handle_phase(packet)
        │
        ├── _apply_packet(): info, setting, talk_history を Agent に反映
        │
        ├── 受信スレッド起動 ───────────────────────────────────────┐
        │                                                           │
        │   メインループ:                              受信スレッド: │
        │   ┌────────────────────────────┐    client.receive()     │
        │   │ 1. Queue からドレイン       │      → Queue に put     │
        │   │ 2. BROADCAST → 履歴に追加   │                         │
        │   │ 3. TALK_END → 終了          │                         │
        │   │ 4. speak_cooldown 経過?     │                         │
        │   │    → agent.talk_realtime()  │                         │
        │   │      LLM に問い合わせ        │                         │
        │   │      ├─ 発言内容             │                         │
        │   │      │  → client.send()     │                         │
        │   │      ├─ "LISTEN"            │                         │
        │   │      │  → 何もしない         │                         │
        │   │      └─ "Over"              │                         │
        │   │         → client.send()     │                         │
        │   │ 5. poll_interval 待機       │                         │
        │   └────────────────────────────┘                         │
        │                                                           │
        ├── TALK_END 受信 → 受信スレッド停止 ──────────────────────┘
        │
        └── return → starter.py に制御を返す
              │
              │ 次の client.receive()
              │    → Packet(request=VOTE)  ← 従来の同期処理
```

---

### 3.4 LLM 発言判断プロンプト

`talk_realtime` プロンプトで、LLMに会話の流れと残り発言回数を渡し、3択で返答させる。

```
あなたはAgent[03]です。リアルタイムのグループチャットで会話中です。
残り発言回数: 7回

--- 会話の流れ ---
Agent[01]: おはようございます
Agent[02]: 昨日の投票結果を見てどう思いますか？
--- ここまで ---

【発言すべき場面】
- 自分の名前が呼ばれた、質問された
- 重要な情報（占い結果など）を共有すべき
- 議論が停滞していて話題を提供すべき
...

発言する場合: 発言内容のみを出力してください。
聞く場合: 「LISTEN」とだけ出力してください。
もう話すことがない場合: 「Over」とだけ出力してください。
```

**LLMの返答例：**
- `"私は占い師です。Agent[05]を占った結果、人狼でした。"` → サーバに送信
- `"LISTEN"` → 何もしない（次のループで再判断）
- `"Over"` → Over送信、以降は受信のみ

---

## 4. パケット構造

### TALK_START（サーバ → エージェント）
```json
{
  "request": "TALK_START",
  "info": {
    "game_id": "01JX...",
    "day": 1,
    "agent": "Agent[03]",
    "remain_count": 10,
    "status_map": {"Agent[01]": "ALIVE", ...},
    "role_map": {"Agent[03]": "SEER"}
  },
  "setting": { ... },
  "talk_history": [
    {"idx": 0, "day": 1, "turn": 0, "agent": "Agent[01]", "text": "..."}
  ]
}
```

### TALK_BROADCAST（サーバ → 全エージェント）
```json
{
  "request": "TALK_BROADCAST",
  "info": {
    "agent": "Agent[02]",
    "remain_count": 8
  },
  "talk_history": [
    {"idx": 5, "day": 1, "turn": 0, "agent": "Agent[03]", "text": "私は占い師です"}
  ]
}
```

### エージェントの発言（エージェント → サーバ）
```
私は占い師です\n
```
（生テキスト。JSONではない。従来のTALKレスポンスと同じ形式。）

### TALK_END（サーバ → 全エージェント）
```json
{
  "request": "TALK_END"
}
```

WHISPER_START / WHISPER_BROADCAST / WHISPER_END も同様の構造。

---

## 5. 並行性モデル

### サーバ側（Go）
```
メインgoroutine          Agent[01]リスナー   Agent[02]リスナー   ...
  │                         │                   │
  │ ←── msgChan ──────────←│                   │
  │ ←── msgChan ────────────────────────────←  │
  │                         │                   │
  │ ブロードキャスト送信     │                   │
  │ （全エージェントに       │                   │
  │  順次 SendNonBlocking）  │                   │
  │                         │                   │
  │ close(done) ──────────→│ 停止              │ 停止
  │ wg.Wait()               │                   │
  │ TALK_END 送信            │                   │
  │ drainAgentBuffers()      │                   │
```

- gorilla/websocket: 1リーダー + 1ライター が安全に並行可能
- メインgoroutineが全write、各リスナーgoroutineがread担当

### エージェント側（Python）
```
メインスレッド（発言判断）         受信スレッド
  │                                  │
  │ ←── Queue<Packet> ──────────── │ client.receive()
  │                                  │
  │ client.send() ──────→ WebSocket  │
  │                                  │
  │ stop_event.set() ──────────────→│ 停止
```

- websocket-client: recv と send は別スレッドから呼び出し可能
- send のみ `_socket_lock` で保護（将来的な安全マージン）

---

## 6. 従来モードとの互換性

| 条件 | 動作 |
|---|---|
| サーバ `realtime.enable: false` | 従来の TALK/WHISPER を送信 → エージェントは通常パスを通る |
| サーバ `realtime.enable: true` + 旧エージェント | エージェントが TALK_START を未知のリクエストとして処理 → **動作しない**（common更新が必要） |
| サーバ `realtime.enable: true` + 新エージェント | リアルタイムモードで動作 |
| 新サーバ + 新エージェント + `enable: false` | 完全に従来通り |

**デプロイ順序：** common → agent → server の順で更新する。
（commonとagentを先に更新しても、サーバが従来モードなら影響なし。）

---

## 7. 設定チューニングガイド

| パラメータ | 場所 | 推奨値 | 説明 |
|---|---|---|---|
| `phase_timeout` | サーバ | 120s | 長すぎると1日が長くなる。短すぎると議論が切れる |
| `silence_timeout` | サーバ | 15s | 全員が考え中でも15秒で打ち切られるので注意 |
| `rate_limit` | サーバ | 2s | LLMの応答速度に合わせる。速いモデルなら長めに |
| `per_agent` | サーバ | 10 | 1日あたりの発言回数。多すぎるとLLM呼び出し回数が増加 |
| `speak_cooldown` | エージェント | 3.0s | サーバの `rate_limit` 以上に設定する |
| `poll_interval` | エージェント | 0.5s | 短いほどレスポンス良好だがCPU負荷増 |
| `llm.sleep_time` | エージェント | 0 | リアルタイムモードでは0推奨 |
