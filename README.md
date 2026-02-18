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

## Docker で起動する（推奨）

Docker を使うと Go / Python / Node.js の個別インストールが不要になる。

### クイックスタート A: ビューアから操作（推奨）

ビューアのコントロールパネルでゲーム開始・停止を制御する方法。

```bash
# 1. APIキーを設定
cp .env.example .env
vim .env   # 使用するプロバイダのキーを入力

# 2. サーバとビューアを起動
docker compose up -d

# 3. エージェントを起動（別ターミナルで実行）
docker compose --profile with-agent up agent
```

4. ブラウザで `http://localhost:5173/control` を開く
5. サーバURL `http://localhost:8080` を入力して「接続」
6. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
7. `http://localhost:5173/realtime` でゲームをリアルタイム観戦

> Docker 環境ではビューアからのエージェント起動（agent_spawner）は無効。エージェントは手順3の `docker compose` コマンドで起動する。

### クイックスタート B: コマンドのみで操作

ビューアを使わず、ターミナルだけで実行する方法。

```bash
# 1. APIキーを設定
cp .env.example .env
vim .env

# 2. 全サービスを一括起動（サーバ + ビューア + エージェント）
docker compose --profile with-agent up -d

# 3. ゲームを開始（manual_start: true のためAPIで開始指示を送る）
#    エージェントがサーバに接続するまで数秒待ってから実行
curl -X POST http://localhost:8080/api/game/start

# 4. ログでゲーム進行を確認
docker compose logs -f agent

# 5. 全て停止
docker compose --profile with-agent down
```

> `docker/server.realtime_5.yml` で `manual_start: false` に変更すると、エージェントが5人揃った時点でゲームが自動開始される。その場合、手順3の `curl` は不要。

### ファイル構成

```
docker-compose.yml           # サービス定義
.env                         # APIキー（.env.example からコピー）
docker/
  server.realtime_5.yml      # サーバ設定（Docker用）
  agent.yml                  # エージェント設定（Docker用）
```

### 各サービスの説明

| サービス | 内容 | ポート |
|---|---|---|
| `server` | ゲームサーバ (Go) | 8080 |
| `viewer` | ビューア (SvelteKit) | 5173 |
| `agent` | LLMエージェント (Python) | なし |

`agent` サービスは `profiles: [with-agent]` が設定されており、`docker compose up` だけでは起動しない。エージェントの起動タイミングを制御するため、別途明示的に起動する。

### よく使うコマンド

```bash
# サーバ + ビューアを起動
docker compose up -d

# エージェントも含めて全て起動
docker compose --profile with-agent up -d

# エージェントだけ起動（サーバ起動後に）
docker compose --profile with-agent up agent -d

# ログを確認
docker compose logs -f server
docker compose logs -f agent

# 全て停止
docker compose --profile with-agent down

# イメージを再ビルド（コード変更後）
docker compose build
```

### Docker 設定のカスタマイズ

Docker 用の設定ファイルは `docker/` ディレクトリにある。

**LLMプロバイダを変更する:**

`docker/agent.yml` を編集:

```yaml
llm:
  type: openai    # google → openai に変更
```

`.env` に対応するAPIキーを設定。

**Ollama（ローカルモデル）を使う場合:**

`docker/agent.yml`:

```yaml
llm:
  type: ollama
ollama:
  # ホストで実行中の Ollama に接続
  base_url: http://host.docker.internal:11434   # macOS/Windows
  # base_url: http://172.17.0.1:11434           # Linux
```

事前にホスト側で `ollama run llama3.1` を実行しておく。

**エージェント数やゲームパラメータを変更する:**

`docker/server.realtime_5.yml` と `docker/agent.yml` を編集する。設定項目の詳細は後述の「カスタマイズ」セクションを参照。

### Docker の注意事項

- Docker 環境では `agent_spawner`（ビューアからのエージェント起動）は無効。エージェントは `docker compose` コマンドで起動する。
- ビューアの `src/` と `static/` はボリュームマウントされており、ホスト側でファイルを編集するとホットリロードが反映される。
- サーバのログは Docker volume（`server-log`）に保存される。`docker compose logs server` でも確認可能。

---

## ローカルで起動する

Docker を使わずに直接実行する場合の手順。

### クイックスタート A: ビューアから操作（推奨）

ビューアのコントロールパネルからエージェント起動・ゲーム制御をすべて行う方法。ターミナルで起動するのはサーバとビューアだけ。

```bash
# 1. サーバを起動（ターミナル1）
cd aiwolf-nlp-server
go run main.go -c config/realtime_5.yml

# 2. ビューアを起動（ターミナル2）
cd aiwolf-nlp-viewer
pnpm install   # 初回のみ
pnpm dev

# 3. エージェントのAPIキーを設定（初回のみ）
cd aiwolf-nlp-agent-llm
cp config/.env.example .env
vim .env   # 使用するプロバイダのキーを入力
```

4. ブラウザで `http://localhost:5173/control` を開く
5. サーバURL `http://localhost:8080` を入力して「接続」
6. エージェント起動パネルでチーム名・モデルを選択して「起動」（5体分）
7. 待機部屋に5人が表示されたら「ゲーム開始」ボタンを押す
8. `http://localhost:5173/realtime` でゲームをリアルタイム観戦
9. ゲーム中は「一時停止」/「再開」で制御、コスト欄で累積コスト確認

> ビューアからのエージェント起動は `agent_spawner` 機能を使用。`realtime_5.yml` ではデフォルトで有効。

### クイックスタート B: コマンドのみで操作

ビューアを使わず、ターミナルだけで実行する方法。

```bash
# 1. エージェントのAPIキーを設定（初回のみ）
cd aiwolf-nlp-agent-llm
cp config/.env.example .env
vim .env   # 使用するプロバイダのキーを入力

# 2. サーバを起動（ターミナル1）
cd aiwolf-nlp-server
go run main.go -c config/realtime_5.yml

# 3. エージェントを起動（ターミナル2）
cd aiwolf-nlp-agent-llm
uv run python src/main.py -c config/config.yml

# 4. ゲームを開始（manual_start: true のためAPIで開始指示を送る）
#    エージェントがサーバに接続するまで数秒待ってから実行
curl -X POST http://localhost:8080/api/game/start

# 5. ゲーム進行はエージェントのターミナルログで確認
```

> `realtime_5.yml` で `manual_start: false` に変更すると、エージェントが5人揃った時点でゲームが自動開始される。その場合、手順4の `curl` は不要。

> ビューアは任意。観戦したい場合はビューアも起動して `http://localhost:5173/realtime` にアクセスする。

### 設定ファイル

`aiwolf-nlp-server/config/` 配下に設定ファイルがある。

| ファイル | 内容 |
|---|---|
| `default_5.yml` | 標準5人戦（ターン制） |
| `default_13.yml` | 標準13人戦（ターン制） |
| `realtime_5.yml` | リアルタイム5人戦（グループチャット方式） |

`realtime_5.yml` はデフォルトで以下がすべて有効になっている:

- リアルタイムモード（グループチャット方式）
- コントロールパネル連携（`manual_start: true`）
- ビューアからのエージェント起動（`agent_spawner`）

追加の設定変更なしでそのまま使える。

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

---

## カスタマイズ

### LLMプロバイダを変更する

エージェント設定 `aiwolf-nlp-agent-llm/config/config.yml`:

```yaml
llm:
  type: openai    # google → openai に変更
```

`.env` に対応するAPIキーを追加:

```
OPENAI_API_KEY=sk-...
```

Ollama（ローカルモデル）を使う場合:

```yaml
llm:
  type: ollama
```

APIキー不要。事前に `ollama run llama3.1` でモデルをダウンロードしておく。

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

### ターン制モード（従来方式）に変更する

サーバ設定で `realtime_5.yml` の代わりに `default_5.yml` を使う:

```bash
go run main.go -c config/default_5.yml
```

`default_5.yml` はリアルタイムモード無効・`manual_start` なし。エージェントが接続すると自動でゲームが開始される。

エージェント側も対応する設定に切り替える:

```bash
uv run python src/main.py -c config/config.jp.yml.example
```

### コントロールパネルを使わない（自動開始モード）

サーバ設定で `manual_start` を無効にする:

```yaml
server:
  manual_start: false   # または行を削除
```

この場合、エージェントが必要人数分接続した時点でゲームが自動的に開始される。コントロールパネルの「ゲーム開始」ボタンは不要。

### ビューアからのエージェント起動を無効にする

サーバ設定で `agent_spawner` を無効にする:

```yaml
agent_spawner:
  enable: false
```

コントロールパネルのエージェント起動パネルが非表示になる。エージェントは従来通りコマンドラインから起動する。

### リモートアクセスを有効にする

デフォルトではサーバは `127.0.0.1`（ローカルのみ）にバインドされる。リモートマシンからアクセスする場合:

```yaml
server:
  web_socket:
    host: 0.0.0.0     # 全インターフェースでリッスン
    port: 8080
```

エージェント側の接続先も変更:

```yaml
web_socket:
  url: ws://<サーバのIPアドレス>:8080/ws
```

### エージェント数を変更する

5人戦（デフォルト）→ 13人戦に変更する場合、サーバ設定で `default_13.yml` を使うか、設定ファイル内で:

```yaml
game:
  agent_count: 13
```

ロール配分も合わせて変更が必要:

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
    phase_timeout: 120s    # フェーズ全体の制限時間（大きくすると議論時間が伸びる）
    silence_timeout: 15s   # 全員が沈黙したら自動終了（小さくするとテンポが上がる）
    rate_limit: 2s         # 最小発言間隔（小さくするとスパム的になる）
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
  poll_interval: 0.5       # ブロードキャスト確認間隔（秒、小さいほどレスポンスが速い）
  speak_cooldown: 3.0      # 連続発言の最小間隔（秒、LLMの呼び出し頻度に影響）
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
  url: ws://127.0.0.1:8080/ws
  token: <生成したトークン>
```

---

## 管理用 REST API

サーバ起動中は常に利用可能:

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

### 一時停止の仕組み

`/api/game/:id/pause` を呼ぶと、現在進行中のフェーズが完了した時点でゲームが停止する。フェーズの途中（エージェントの応答待ち中など）では停止しない。`/api/game/:id/resume` で続行。

### コスト追跡

エージェントはゲーム終了時（FINISH）に自動的に `/api/cost/report` にトークン使用量とコストをHTTP POSTで送信する。`/api/status` のレスポンスにゲームごとの集計が含まれる。

---

## 対応LLMモデルとコスト

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

- ゲームが正常に終了（FINISH）していることを確認（途中でプロセスを kill するとレポートが送信されない）
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
