# AI Werewolf Realtime System

AI人狼リアルタイム対戦システム。Go製ゲームサーバ、Python製LLMエージェント、SvelteKit製ビューアの3コンポーネントで構成される。

## 構成

```
realtime_system/
  aiwolf-nlp-server/     # ゲームサーバ (Go / Gin)
  aiwolf-nlp-agent-llm/  # LLMエージェント (Python / LangChain)
  aiwolf-nlp-viewer/     # ビューア (SvelteKit / DaisyUI)
  aiwolf-nlp-common/     # 共通ライブラリ (Python)
```

## 前提条件

**Docker を使う場合（推奨）:**

- Docker 20.10+
- Docker Compose v2+
- LLM APIキー (Google Gemini / OpenAI / Ollama のいずれか)

**ローカルで直接実行する場合:**

- Go 1.24+
- Python 3.11+ / [uv](https://docs.astral.sh/uv/) (推奨)
- Node.js 22+ / pnpm 10+
- LLM APIキー (Google Gemini / OpenAI / Ollama のいずれか)

---

## 初回セットアップ

### Docker の場合（推奨）

```bash
# 1. リポジトリをクローン
git clone https://github.com/iggy157/aiwolf-nlp-realtime-system.git
cd aiwolf-nlp-realtime-system

# 2. APIキーを設定
cp .env.example .env
vim .env   # 使用するプロバイダのAPIキーを入力

# 3. LLMプロバイダを変更する場合は docker/agent.yml を編集（デフォルトは Google Gemini）
# vim docker/agent.yml   # llm.type を openai や ollama に変更

# 4. Docker イメージをビルド（初回のみ、数分かかる）
docker compose build
```

**`.env` の中身**（使用するプロバイダのキーだけ設定すればよい）:

```
GOOGLE_API_KEY=AIza...        # Google Gemini を使う場合（デフォルト）
OPENAI_API_KEY=sk-proj-...    # OpenAI を使う場合
# Ollama の場合はAPIキー不要（事前に ollama run llama3.1 を実行）
```

**`docker/agent.yml` の LLM 設定**（プロバイダを変更する場合のみ編集）:

```yaml
llm:
  type: google    # google / openai / ollama から選択
```

> Docker のデフォルトはリアルタイムモード（日本語）。ターン制で使う場合は [実行方法 > Docker > ターン制モード](#docker-ターン制モード) を参照。

### ローカルの場合

```bash
# 1. リポジトリをクローン
git clone https://github.com/iggy157/aiwolf-nlp-realtime-system.git
cd aiwolf-nlp-realtime-system

# 2. エージェントの設定ファイルを作成（以下から1つ選ぶ）
cd aiwolf-nlp-agent-llm

# リアルタイムモード・日本語の場合（推奨）
cp config/config.realtime.yml.example config/config.yml

# ターン制・日本語の場合
# cp config/config.jp.yml.example config/config.yml

# ターン制・英語の場合
# cp config/config.en.yml.example config/config.yml

# 3. APIキーを設定
cp config/.env.example config/.env
vim config/.env   # 使用するプロバイダのAPIキーを入力

# 4. LLMプロバイダを変更する場合は config.yml を編集（デフォルトは Google Gemini）
# vim config/config.yml   # llm.type を openai や ollama に変更

# 5. Python 依存関係をインストール
uv sync
cd ..

# 6. ビューアの依存関係をインストール
cd aiwolf-nlp-viewer
pnpm install
cd ..
```

**テンプレートと対応するサーバ設定の組み合わせ:**

| エージェントテンプレート | 内容 | サーバ起動時の設定ファイル |
|---|---|---|
| `config.realtime.yml.example` | リアルタイムモード（日本語） | `realtime_5.yml` |
| `config.jp.yml.example` | ターン制モード（日本語） | `default_viewer_5.yml` or `default_5.yml` |
| `config.en.yml.example` | ターン制モード（英語） | `default_en_5.yml` |

> `default_viewer_5.yml` はビューアからの操作（手動開始・エージェント起動・リアルタイム観戦）に対応したターン制設定。`default_5.yml` はエージェント接続時に自動開始する従来の設定。

**`config/.env` の中身**（使用するプロバイダのキーだけ設定すればよい）:

```
GOOGLE_API_KEY=AIza...        # Google Gemini を使う場合（デフォルト）
OPENAI_API_KEY=sk-proj-...    # OpenAI を使う場合
# Ollama の場合はAPIキー不要（事前に ollama run llama3.1 を実行）
```

**`config/config.yml` の LLM 設定**（プロバイダを変更する場合のみ編集）:

```yaml
llm:
  type: google    # google / openai / ollama から選択
```

---

## 実行方法

### Docker

> Docker 環境ではビューアからのエージェント起動（agent_spawner）は無効。エージェントは `docker compose` コマンドで起動する。

#### リアルタイムモード（デフォルト）

**ビューアから操作（推奨）:**

```bash
docker compose up -d
docker compose --profile with-agent up agent
```

1. ブラウザで `http://localhost:5173/control` を開く
2. サーバURL `http://localhost:8080` を入力して「接続」
3. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
4. `http://localhost:5173/realtime` でゲームをリアルタイム観戦

**コマンドのみで操作:**

```bash
docker compose --profile with-agent up -d
sleep 5 && curl -X POST http://localhost:8080/api/game/start
docker compose logs -f agent
```

#### ターン制モード {#docker-ターン制モード}

環境変数 `SERVER_CONFIG` と `AGENT_CONFIG` でサーバ・エージェントの設定ファイルを切り替える。

**ビューアから操作（推奨）:**

```bash
SERVER_CONFIG=docker/server.default_viewer_5.yml AGENT_CONFIG=docker/agent.turnbased.yml docker compose up -d
SERVER_CONFIG=docker/server.default_viewer_5.yml AGENT_CONFIG=docker/agent.turnbased.yml docker compose --profile with-agent up agent
```

1. ブラウザで `http://localhost:5173/control` を開く
2. サーバURL `http://localhost:8080` を入力して「接続」
3. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
4. `http://localhost:5173/realtime` でゲームをリアルタイム観戦

**コマンドのみで操作:**

```bash
SERVER_CONFIG=docker/server.default_viewer_5.yml AGENT_CONFIG=docker/agent.turnbased.yml docker compose --profile with-agent up -d
sleep 5 && curl -X POST http://localhost:8080/api/game/start
docker compose logs -f agent
```

> 環境変数を省略するとリアルタイムモード（デフォルト）で起動する。

#### 停止

```bash
docker compose --profile with-agent down
```

#### その他よく使うコマンド

```bash
docker compose logs -f server     # サーバログ確認
docker compose logs -f agent      # エージェントログ確認
docker compose build              # イメージ再ビルド（コード変更後）
```

### ローカル

#### リアルタイムモード + ビューアから操作（推奨）

```bash
# ターミナル1: サーバを起動
cd aiwolf-nlp-server
go run main.go -c config/realtime_5.yml

# ターミナル2: ビューアを起動
cd aiwolf-nlp-viewer
pnpm dev
```

1. ブラウザで `http://localhost:5173/control` を開く
2. サーバURL `http://localhost:8080` を入力して「接続」
3. エージェント起動パネルでチーム名・モデルを選択して「起動」（5体分）
4. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
5. `http://localhost:5173/realtime` でゲームをリアルタイム観戦

> ビューアからのエージェント起動は `agent_spawner` 機能を使用。`realtime_5.yml` ではデフォルトで有効。

#### リアルタイムモード + コマンドのみで操作

```bash
# ターミナル1: サーバを起動
cd aiwolf-nlp-server
go run main.go -c config/realtime_5.yml

# ターミナル2: エージェントを起動
cd aiwolf-nlp-agent-llm
uv run python src/main.py -c config/config.yml

# ターミナル3: エージェント接続後にゲームを開始（manual_start: true のため手動）
curl -X POST http://localhost:8080/api/game/start
```

> ビューアで観戦したい場合は別ターミナルで `cd aiwolf-nlp-viewer && pnpm dev` → `http://localhost:5173/realtime`

#### ターン制モード + ビューアから操作

```bash
# ターミナル1: サーバを起動
cd aiwolf-nlp-server
go run main.go -c config/default_viewer_5.yml

# ターミナル2: ビューアを起動
cd aiwolf-nlp-viewer
pnpm dev
```

1. ブラウザで `http://localhost:5173/control` を開く
2. サーバURL `http://localhost:8080` を入力して「接続」
3. エージェント起動パネルでチーム名・モデルを選択して「起動」（5体分）
4. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
5. `http://localhost:5173/realtime` でゲームをリアルタイム観戦

#### ターン制モード + コマンドのみ（日本語）

```bash
# ターミナル1: サーバを起動
cd aiwolf-nlp-server
go run main.go -c config/default_5.yml

# ターミナル2: エージェントを起動（5人揃うと自動開始、curl 不要）
cd aiwolf-nlp-agent-llm
uv run python src/main.py -c config/config.yml
```

#### ターン制モード + コマンドのみ（英語）

```bash
# ターミナル1: サーバを起動
cd aiwolf-nlp-server
go run main.go -c config/default_en_5.yml

# ターミナル2: エージェントを起動（5人揃うと自動開始、curl 不要）
cd aiwolf-nlp-agent-llm
uv run python src/main.py -c config/config.yml
```

---

## 2回目以降の日常コマンド

初回セットアップ完了後は、設定変更がなければ起動コマンドだけでよい。

**Docker + リアルタイム + ビューア操作:**

```bash
docker compose up -d
docker compose --profile with-agent up agent
# → ブラウザで http://localhost:5173/control を開いて操作
# → 停止: docker compose --profile with-agent down
```

**Docker + リアルタイム + コマンドのみ:**

```bash
docker compose --profile with-agent up -d
sleep 5 && curl -X POST http://localhost:8080/api/game/start
# → 停止: docker compose --profile with-agent down
```

**Docker + ターン制 + ビューア操作:**

```bash
SERVER_CONFIG=docker/server.default_viewer_5.yml AGENT_CONFIG=docker/agent.turnbased.yml docker compose up -d
SERVER_CONFIG=docker/server.default_viewer_5.yml AGENT_CONFIG=docker/agent.turnbased.yml docker compose --profile with-agent up agent
# → ブラウザで http://localhost:5173/control を開いて操作
# → 停止: docker compose --profile with-agent down
```

**ローカル + ビューア操作（リアルタイム）:**

```bash
# ターミナル1
cd aiwolf-nlp-server && go run main.go -c config/realtime_5.yml
# ターミナル2
cd aiwolf-nlp-viewer && pnpm dev
# → ブラウザで http://localhost:5173/control を開いて操作
```

**ローカル + コマンドのみ（リアルタイム）:**

```bash
# ターミナル1
cd aiwolf-nlp-server && go run main.go -c config/realtime_5.yml
# ターミナル2
cd aiwolf-nlp-agent-llm && uv run python src/main.py -c config/config.yml
# ターミナル3（エージェント接続後）
curl -X POST http://localhost:8080/api/game/start
```

**ローカル + ターン制 + ビューア操作:**

```bash
# ターミナル1
cd aiwolf-nlp-server && go run main.go -c config/default_viewer_5.yml
# ターミナル2
cd aiwolf-nlp-viewer && pnpm dev
# → ブラウザで http://localhost:5173/control を開いて操作
```

**ローカル + ターン制 + コマンドのみ:**

```bash
# ターミナル1（日本語: default_5.yml / 英語: default_en_5.yml）
cd aiwolf-nlp-server && go run main.go -c config/default_5.yml
# ターミナル2（自動開始）
cd aiwolf-nlp-agent-llm && uv run python src/main.py -c config/config.yml
```

---

## カスタマイズ

### LLMプロバイダを変更する

設定ファイルを編集（ローカル: `aiwolf-nlp-agent-llm/config/config.yml` / Docker: `docker/agent.yml` or `docker/agent.turnbased.yml`）:

```yaml
llm:
  type: openai    # google / openai / ollama から選択
```

対応するAPIキーを設定（ローカル: `config/.env` / Docker: `.env`）。Ollama はAPIキー不要。

使用するモデルやtemperatureは各プロバイダセクションで変更:

```yaml
google:
  model: gemini-2.0-flash       # デフォルト: gemini-2.0-flash-lite
  temperature: 0.5              # デフォルト: 0.7

openai:
  model: gpt-4o                 # デフォルト: gpt-4o-mini
  temperature: 0.7

ollama:
  model: llama3.1               # デフォルト: llama3.1
  temperature: 0.7
  base_url: http://localhost:11434
```

> Docker で Ollama を使う場合は `base_url` を `http://host.docker.internal:11434`（macOS/Windows）または `http://172.17.0.1:11434`（Linux）に変更する。

### コントロールパネルを使わない（自動開始モード）

サーバ設定で `manual_start` を無効にする:

```yaml
server:
  manual_start: false   # または行を削除
```

エージェントが必要人数分接続した時点でゲームが自動開始される。

### ビューアからのエージェント起動を無効にする

サーバ設定で `agent_spawner` を無効にする:

```yaml
agent_spawner:
  enable: false
```

### リモートアクセスを有効にする

サーバ設定:

```yaml
server:
  web_socket:
    host: 0.0.0.0     # デフォルト 127.0.0.1 → 全インターフェースでリッスン
    port: 8080
```

エージェント側の接続先も変更:

```yaml
web_socket:
  url: ws://<サーバのIPアドレス>:8080/ws
```

### エージェント数を変更する

5人戦 → 13人戦に変更する場合、サーバ設定で `default_13.yml` を使うか、設定ファイル内で:

```yaml
game:
  agent_count: 13
```

ロール配分も合わせて変更:

```yaml
logic:
  roles:
    13:
      WEREWOLF: 3
      POSSESSED: 1
      SEER: 1
      BODYGUARD: 1
      MEDIUM: 1
      VILLAGER: 6
```

### リアルタイム通信のパラメータ調整

サーバ設定:

```yaml
game:
  realtime:
    phase_timeout: 120s    # フェーズ全体の制限時間
    silence_timeout: 15s   # 全員が沈黙したら自動終了
    rate_limit: 2s         # 最小発言間隔
  talk:
    max_count:
      per_agent: 10        # 1日あたりの最大発言回数/エージェント
      per_day: 50          # 1日あたりの最大発言回数/全体
    max_length:
      per_talk: 200        # 1発言あたりの最大文字数
```

エージェント設定:

```yaml
realtime:
  poll_interval: 0.5       # ブロードキャスト確認間隔（秒）
  speak_cooldown: 3.0      # 連続発言の最小間隔（秒）
```

### 接続認証を有効にする

サーバ設定:

```yaml
server:
  authentication:
    enable: true
```

環境変数で秘密鍵を設定:

```bash
export SECRET_KEY=your-secret-key
```

ビューアの `/token` ページでトークンを生成し、エージェント設定に追加:

```yaml
web_socket:
  token: <生成したトークン>
```

---

## リファレンス

### Docker ファイル構成

```
docker-compose.yml                  # サービス定義
.env                                # APIキー（.env.example からコピー）
docker/
  server.realtime_5.yml             # サーバ設定：リアルタイムモード（デフォルト）
  server.default_viewer_5.yml       # サーバ設定：ターン制 + ビューア操作
  agent.yml                         # エージェント設定：リアルタイムモード（デフォルト）
  agent.turnbased.yml               # エージェント設定：ターン制モード
```

| サービス | 内容 | ポート |
|---|---|---|
| `server` | ゲームサーバ (Go) | 8080 |
| `viewer` | ビューア (SvelteKit) | 5173 |
| `agent` | LLMエージェント (Python) | なし |

`agent` は `profiles: [with-agent]` が設定されており、`docker compose up` だけでは起動しない。

環境変数 `SERVER_CONFIG` と `AGENT_CONFIG` で設定ファイルを切り替え可能（デフォルトはリアルタイムモード）:

| 環境変数 | デフォルト値 | ターン制の場合 |
|---|---|---|
| `SERVER_CONFIG` | `docker/server.realtime_5.yml` | `docker/server.default_viewer_5.yml` |
| `AGENT_CONFIG` | `docker/agent.yml` | `docker/agent.turnbased.yml` |

- Docker 環境では `agent_spawner` は無効（エージェントは `docker compose` で起動）
- ビューアの `src/` と `static/` はボリュームマウント（ホットリロード可）
- サーバログは Docker volume（`server-log`）に保存

### サーバ設定ファイル

`aiwolf-nlp-server/config/` 配下:

| ファイル | 内容 | ビューア操作 |
|---|---|---|
| `realtime_5.yml` | リアルタイム5人戦（グループチャット方式） | 対応 |
| `default_viewer_5.yml` | ターン制5人戦 + ビューア操作対応 | 対応 |
| `default_5.yml` | 標準5人戦（ターン制・自動開始） | 観戦のみ |
| `default_13.yml` | 標準13人戦（ターン制・自動開始） | 観戦のみ |
| `default_en_5.yml` | 標準5人戦（英語・ターン制・自動開始） | 観戦のみ |
| `default_en_13.yml` | 標準13人戦（英語・ターン制・自動開始） | 観戦のみ |

「ビューア操作 対応」= `manual_start`・`agent_spawner`・`realtime_broadcaster` が有効。コントロールパネルからエージェント起動・ゲーム開始・リアルタイム観戦がすべてできる。

### ビューアの各ページ

| パス | 内容 |
|---|---|
| `/` | ホーム（各ページへのリンク） |
| `/control` | コントロールパネル（サーバ管理、エージェント起動、ゲーム制御） |
| `/realtime` | リアルタイムログ表示（WebSocket接続でゲーム進行を視覚化） |
| `/archive` | アーカイブログ表示（LOG形式ファイルの読み込み） |
| `/agent` | エージェントUI（ブラウザからエージェントとして参加） |
| `/token` | 認証トークン生成 |
| `/statistics` | 統計情報 |

### 管理用 REST API

| メソッド | パス | 内容 |
|---|---|---|
| `GET` | `/api/status` | サーバ状態（待機部屋、ゲーム一覧、コスト、プロセス） |
| `POST` | `/api/game/start` | ゲーム開始（`manual_start: true` 時） |
| `POST` | `/api/game/:id/pause` | 一時停止（フェーズ境界で停止） |
| `POST` | `/api/game/:id/resume` | 再開 |
| `POST` | `/api/cost/report` | コストレポート受信（エージェントから自動送信） |
| `POST` | `/api/agent/spawn` | エージェントプロセス起動（`agent_spawner` 有効時） |
| `GET` | `/api/agent/processes` | 起動済みプロセス一覧 |
| `POST` | `/api/agent/:id/stop` | プロセス停止 |

**一時停止の仕組み:** `/api/game/:id/pause` を呼ぶと、現在のフェーズが完了した時点でゲームが停止する。`/api/game/:id/resume` で続行。

**コスト追跡:** エージェントはゲーム終了時（FINISH）に自動的に `/api/cost/report` にコストをPOST送信する。`/api/status` にゲームごとの集計が含まれる。

### 対応LLMモデルとコスト

| プロバイダ | モデル | 入力 ($/1M tokens) | 出力 ($/1M tokens) |
|---|---|---|---|
| Google | gemini-2.0-flash-lite | 0.02 | 0.10 |
| Google | gemini-2.0-flash | 0.10 | 0.40 |
| Google | gemini-2.5-flash | 0.15 | 0.60 |
| OpenAI | gpt-4o-mini | 0.15 | 0.60 |
| OpenAI | gpt-4o | 2.50 | 10.00 |
| Ollama | 全モデル | 0.00 | 0.00 |

---

## トラブルシューティング

### サーバに接続できない

- `host: 127.0.0.1` → 同一マシンからのみ接続可能。リモートなら `0.0.0.0` に変更
- ファイアウォールでポートが開放されていることを確認

### エージェントが接続後すぐにゲームが始まる

- サーバ設定で `manual_start: true` になっていることを確認

### コストが表示されない

- ゲームが正常に終了（FINISH）していることを確認（途中で kill するとレポートが送信されない）
- `.env` にAPIキーが正しく設定されていることを確認

### spawner で起動したエージェントが接続しない

- `agent_spawner.agent_dir` のパスが正しいことを確認（サーバからの相対パス）
- `agent_spawner.python_cmd` が実行可能なことを確認（`uv run python` or `python`）
- `agent_spawner.config_template` がエージェントディレクトリからの相対パスであることを確認

### ビューアのビルドエラー

```bash
cd aiwolf-nlp-viewer
pnpm install   # 依存関係を再インストール
pnpm dev
```

---

## ライセンス

各コンポーネントのリポジトリを参照:

- [aiwolf-nlp-server](https://github.com/aiwolfdial/aiwolf-nlp-server)
- [aiwolf-nlp-agent-llm](https://github.com/aiwolfdial/aiwolf-nlp-agent-llm)
- [aiwolf-nlp-viewer](https://github.com/aiwolfdial/aiwolf-nlp-viewer)
