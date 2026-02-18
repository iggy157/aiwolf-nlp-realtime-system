# Phase 1: ゲーム制御API + ビューア管理パネル

## 概要

ビューアからサーバを制御して、ゲームの開始・一時停止・再開ができるようにする。

## アーキテクチャ

```
Viewer (Control Panel)
  │ HTTP (polling 2s)
  ├── GET  /api/status        → サーバ状態取得
  ├── POST /api/game/start    → ゲーム開始
  ├── POST /api/game/:id/pause  → 一時停止
  └── POST /api/game/:id/resume → 再開
  │
Server (Go / Gin)
  ├── 管理REST API (core/api.go) [新規]
  ├── Game.checkPause() — フェーズ境界で一時停止チェック (sync.Cond)
  └── manual_start: true — エージェント接続時に自動開始しない
```

## サーバ側変更 (aiwolf-nlp-server)

### 新規ファイル

| ファイル | 内容 |
|---|---|
| `core/api.go` | REST API ハンドラ (`/api/status`, `/api/game/start`, `/api/game/:id/pause`, `/api/game/:id/resume`) |

### 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `model/config.go` | `ServerConfig.ManualStart bool` 追加 |
| `core/server.go` | `registerAPIRoutes()` 呼び出し追加、`ManualStart` 時に自動ゲーム開始をスキップ |
| `core/waiting_room.go` | `ListTeams()`, `TotalCount()` メソッド追加（API用） |
| `logic/game.go` | `sync.Cond` ベースの pause/resume 機構、`currentPhase` フェーズ追跡 |
| `logic/common.go` | `checkPause()`, `Pause()`, `Resume()`, `IsPaused()`, `GetPhase()`, `GetDay()`, `GetIsDaytime()`, `GetWinSide()`, `GetAgentStatusInfos()` 追加 |

### 一時停止の仕組み

```
Game.Start()
  └── for { checkPause() → progressDay() → checkPause() → progressNight() }
              │
progressDay()
  └── for phase { checkPause() → executePhase() }
              │
checkPause()
  └── pauseMu.Lock() → for g.paused { pauseCond.Wait() } → Unlock()
```

`checkPause()` はフェーズの境界（昼→夜、トーク→投票 等）でのみ呼ばれる。  
フェーズの途中（例: エージェントの応答待ち）では停止しない。  
これにより「現在のフェーズが終わったら停止」という自然な一時停止が実現される。

### 設定例

```yaml
server:
  web_socket:
    host: 127.0.0.1
    port: 8080
  manual_start: true  # ← 追加: APIからのゲーム開始を待つ
```

`manual_start: false`（デフォルト）の場合は従来通りの自動開始動作。

### APIレスポンス例

```json
// GET /api/status
{
  "server_version": "1.0.0",
  "manual_start": true,
  "waiting_room": {
    "required": 5,
    "teams": [
      {"name": "team-a", "count": 1},
      {"name": "team-b", "count": 1}
    ],
    "total": 2
  },
  "games": [
    {
      "id": "01JXYZ...",
      "day": 2,
      "is_daytime": true,
      "phase": "talk",
      "paused": false,
      "finished": false,
      "agents": [
        {"idx": 1, "name": "Agent[01]", "team": "team-a", "role": "VILLAGER", "alive": true, "has_error": false}
      ]
    }
  ]
}
```

## ビューア側変更 (aiwolf-nlp-viewer)

### 新規ファイル

| ファイル | 内容 |
|---|---|
| `src/routes/control/+page.svelte` | コントロールパネルページ |

### 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `src/i18n/ja.json` | `control.*` 翻訳キー追加（重複セクション除去含む） |
| `src/i18n/en.json` | `control.*` 翻訳キー追加（重複セクション除去含む） |

### コントロールパネル UI

**接続バー**: サーバURL入力 → 接続/切断ボタン → ステータスインジケーター

**待機部屋カード**:
- チーム一覧（バッジ表示）
- `N / M` カウンター（全員揃うとバッジが緑に）
- ▶ ゲーム開始ボタン（人数が揃った時のみ表示）

**アクティブゲームカード**:
- LIVEインジケーター or 一時停止アイコン
- `N日目 昼/夜` + フェーズ名バッジ
- ⏸ 一時停止 / ▶ 再開ボタン
- エージェントグリッド（チーム名・役職・生死・エラー状態）

**終了済みゲーム**: 折りたたみ式（勝利チーム色分け）

## 使い方

1. サーバ設定に `manual_start: true` を追加して起動
2. エージェントを通常通り接続（待機部屋に入る）
3. ビューアの `/control` にアクセスしてサーバURLを入力、接続
4. 待機部屋の人数が揃ったら「ゲーム開始」ボタンをクリック
5. ゲーム中は「一時停止」でフェーズ境界で停止、「再開」で続行

---

# Phase 2: トークン/コスト追跡

## 概要

エージェントのLLM呼び出しごとにトークン使用量とコストを追跡し、ビューアに表示する。

## データフロー

```
Agent (Python)
  │ LLM呼び出し → AIMessage.response_metadata からトークン数取得
  │ CostTracker に蓄積
  │
  ├── FINISH時に HTTP POST /api/cost/report
  ▼
Server (Go)
  │ costReports (sync.Map) にゲーム別・エージェント別で蓄積
  │
  ├── GET /api/status の costs フィールドで公開
  ▼
Viewer (Control Panel)
  └── コスト集計テーブル表示
```

## エージェント側変更 (aiwolf-nlp-agent-llm)

### 新規ファイル

| ファイル | 内容 |
|---|---|
| `src/utils/cost_tracker.py` | `CostTracker` クラス + モデルコスト対応表 + サーバ送信 |

### 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `src/agent/agent.py` | `CostTracker` 統合。`_send_message_to_llm()` と `talk_realtime()` で `llm_model.invoke()` を直接呼びメタデータ取得。`StrOutputParser` チェーン廃止 |
| `src/starter.py` | FINISH時にコストレポートをサーバに HTTP POST |

### CostTracker の仕組み

```python
# LLM呼び出し時（agent.py）
ai_message = self.llm_model.invoke(self.llm_message_history)
self.cost_tracker.track(ai_message.response_metadata)

# FINISH時（starter.py）
agent.cost_tracker.report_to_server(ws_url, game_id, agent_name, team_name)
```

### コスト対応表（抜粋）

| モデル | 入力 ($/1M tokens) | 出力 ($/1M tokens) |
|---|---|---|
| gpt-4o | 2.50 | 10.00 |
| gpt-4o-mini | 0.15 | 0.60 |
| gemini-2.0-flash | 0.10 | 0.40 |
| gemini-2.0-flash-lite | 0.02 | 0.10 |
| ollama (全モデル) | 0.00 | 0.00 |

## サーバ側変更

| ファイル | 変更内容 |
|---|---|
| `core/api.go` | `POST /api/cost/report` エンドポイント追加。`GET /api/status` に `costs` フィールド追加 |
| `core/server.go` | `costReports sync.Map` 追加 |

### APIレスポンス（コスト部分）

```json
{
  "costs": [
    {
      "game_id": "01JXYZ...",
      "agents": [
        {
          "agent": "Agent[01]",
          "team": "team-a",
          "model": "gemini-2.0-flash-lite",
          "input_tokens": 12500,
          "output_tokens": 3200,
          "total_cost": 0.000570
        }
      ],
      "total_cost": 0.002850,
      "total_input_tokens": 62500,
      "total_output_tokens": 16000
    }
  ]
}
```

## ビューア側変更

| ファイル | 変更内容 |
|---|---|
| `src/routes/control/+page.svelte` | コスト集計カード追加（エージェント別テーブル + 合計バッジ） |
| `src/i18n/ja.json` / `en.json` | コスト関連の翻訳キー 8件追加 |

### コスト表示UI

- **合計コストバッジ** — ヘッダー右にゲーム全体のUSDコスト表示
- **エージェント別テーブル** — チーム名、モデル名、入力/出力トークン、呼び出し回数、コスト
- **スマートフォーマット** — トークンは K/M 単位、コストは有効桁数に応じて $0.0001〜$0.00 表示

---

# Phase 3: エージェントプロセスspawn

## 概要

ビューアからエージェントプロセスを起動・停止できるようにする。サーバがPythonエージェントのサブプロセスを管理する。

## アーキテクチャ

```
Viewer (Control Panel)
  │ POST /api/agent/spawn {team, count, llm_type, model}
  │ POST /api/agent/:id/stop
  ▼
Server (Go)
  │ 1. 設定テンプレートYAMLを読み込み
  │ 2. パラメータを上書き（チーム名、モデル、WS URL等）
  │ 3. 一時ファイルに書き出し
  │ 4. exec.Command でPythonプロセスをspawn
  │ 5. プロセス終了を監視（goroutine）
  ▼
Agent Process (Python)
  └── 通常通り main.py -c <temp_config.yml> で起動
```

## サーバ側変更

### 新規ファイル

| ファイル | 内容 |
|---|---|
| `core/spawner.go` | `SpawnedProcess` 管理 + spawn/list/stop ハンドラ |

### 変更ファイル

| ファイル | 変更内容 |
|---|---|
| `model/config.go` | `AgentSpawnerConfig` 型追加（`enable`, `agent_dir`, `python_cmd`, `config_template`） |
| `core/server.go` | `spawnedProcesses sync.Map` 追加 |
| `core/api.go` | `registerSpawnRoutes()` 呼び出し追加。`GET /api/status` に `spawner_enabled` と `processes` フィールド追加 |

### 設定例

```yaml
server:
  web_socket:
    host: 127.0.0.1
    port: 8080
  manual_start: true

agent_spawner:
  enable: true
  agent_dir: ../aiwolf-nlp-agent-llm
  python_cmd: uv run python     # or just "python"
  config_template: config/config.jp.yml.example
```

### API

| メソッド | パス | 内容 |
|---|---|---|
| `POST` | `/api/agent/spawn` | エージェントプロセスをspawn |
| `GET` | `/api/agent/processes` | spawnされたプロセス一覧 |
| `POST` | `/api/agent/:id/stop` | プロセスを停止 |

### SpawnRequest

```json
{
  "team": "team-a",
  "count": 5,
  "llm_type": "google",
  "model": "gemini-2.0-flash-lite",
  "temperature": 0.7
}
```

## ビューア側変更

| ファイル | 変更内容 |
|---|---|
| `src/routes/control/+page.svelte` | エージェント起動パネル追加（チーム名・数・LLM・モデル選択 + 起動ボタン + プロセス一覧） |
| `src/i18n/ja.json` / `en.json` | spawn関連の翻訳キー 6件追加 |

### Spawn UI

- **起動フォーム** — チーム名、数、LLM種別（ドロップダウン）、モデル名（テキスト入力）
- LLM種別変更時にデフォルトモデルを自動設定
- **プロセス一覧** — チーム名、モデル、ステータス（running/stopped/error）、停止ボタン
- 色分け: running=緑、error=赤、stopped=グレー

---

# 全体の使い方（Phase 1〜3 統合）

```
1. サーバ設定
   manual_start: true
   agent_spawner:
     enable: true
     agent_dir: ../aiwolf-nlp-agent-llm
     python_cmd: uv run python
     config_template: config/config.jp.yml.example

2. サーバ起動
   $ go run main.go -c config/realtime_5.yml

3. ビューアで /control にアクセス、サーバURLを入力して接続

4. エージェント起動パネルから:
   - チーム名: team-a, 数: 5, LLM: Google, モデル: gemini-2.0-flash-lite
   - 「起動」ボタンクリック

5. 待機部屋に5人揃ったら「ゲーム開始」ボタン

6. ゲーム進行中:
   - 「一時停止」→ フェーズ境界で停止
   - 「再開」→ 続行
   - コスト欄にリアルタイムで累積コスト表示

7. ゲーム終了 → コスト集計テーブルに最終結果
```