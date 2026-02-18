# aiwolf-nlp-agent-llm

[README in English](/README.en.md)

人狼知能コンテスト（自然言語部門） のLLMを用いたサンプルエージェントです。

## 環境構築

> [!IMPORTANT]
> Python 3.11以上が必要です。

```bash
git clone https://github.com/aiwolfdial/aiwolf-nlp-agent-llm.git
cd aiwolf-nlp-agent-llm
cp config/.env.example config/.env
python -m venv .venv
source .venv/bin/activate
pip install -e .
```

### 日本語のプロンプトを使用したい場合
```bash
cp config/config.jp.yml.example config/config.yml
```

### 英語のプロンプトを使用したい場合
```bash
cp config/config.en.yml.example config/config.yml
```

### リアルタイム通信モードを使用したい場合

```bash
cp config/config.realtime.yml.example config/config.yml
```

## リアルタイム通信モード

サーバ側でリアルタイムモードが有効な場合、TALK/WHISPERフェーズがグループチャット方式で動作します。エージェントは自由なタイミングで発言でき、他のエージェントの発言をリアルタイムで受信します。

### 仕組み

1. サーバから `TALK_START` を受信するとリアルタイムフェーズに入る
2. 受信スレッドがバックグラウンドでブロードキャストを受信し続ける
3. メインスレッドが定期的にLLMに「今話すべきか？」を問い合わせる
4. LLMの判断に基づき、発言 / 聞くだけ(`LISTEN`) / 終了(`Over`) を選択する
5. サーバから `TALK_END` を受信するとフェーズ終了、従来の同期モードに戻る

### 設定

```yaml
realtime:
  poll_interval: 0.5     # ブロードキャスト確認間隔（秒）
  speak_cooldown: 3.0    # 連続発言の最小間隔（秒）
```

| パラメータ | 説明 | 推奨値 |
|---|---|---|
| `poll_interval` | 新着メッセージの確認とLLM判断の間隔 | 0.5s |
| `speak_cooldown` | 連続発言を防ぐ最小間隔。サーバの `rate_limit` 以上に設定する | 3.0s |
| `llm.sleep_time` | リアルタイムモードではレスポンス速度重視のため0推奨 | 0 |

### プロンプト

設定ファイルの `prompt` セクションに `talk_realtime` / `whisper_realtime` を定義します。未定義の場合は従来の `talk` / `whisper` プロンプトにフォールバックします。

> [!IMPORTANT]
> リアルタイムモードを使用するには `aiwolf-nlp-common` の更新版が必要です。
> ```bash
> pip install -e ../aiwolf-nlp-common
> ```

## その他

実行方法や設定などその他については[aiwolf-nlp-agent](https://github.com/aiwolfdial/aiwolf-nlp-agent)をご確認ください。
