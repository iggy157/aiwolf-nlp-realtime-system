# aiwolf-nlp-agent-llm

人狼知能コンテスト（自然言語部門） のLLMを用いたサンプルエージェント。

> セットアップ・実行方法は[ルートREADME](../README.md)を参照。

## リアルタイム通信モード

サーバ側でリアルタイムモードが有効な場合、TALK/WHISPERフェーズがグループチャット方式で動作する。エージェントは自由なタイミングで発言でき、他のエージェントの発言をリアルタイムで受信する。

### 仕組み

1. サーバから `TALK_START` を受信するとリアルタイムフェーズに入る
2. 受信スレッドがバックグラウンドでブロードキャストを受信し続ける
3. メインスレッドが定期的にLLMに「今話すべきか？」を問い合わせる
4. LLMの判断に基づき、発言 / 聞くだけ(`LISTEN`) / 終了(`Over`) を選択する
5. サーバから `TALK_END` を受信するとフェーズ終了、従来の同期モードに戻る

### 設定パラメータ

| パラメータ | 説明 | 推奨値 |
|---|---|---|
| `realtime.poll_interval` | 新着メッセージの確認とLLM判断の間隔 | 0.5s |
| `realtime.speak_cooldown` | 連続発言を防ぐ最小間隔。サーバの `rate_limit` 以上に設定する | 3.0s |
| `llm.sleep_time` | リアルタイムモードではレスポンス速度重視のため0推奨 | 0 |

### プロンプト

設定ファイルの `prompt` セクションに `talk_realtime` / `whisper_realtime` を定義する。未定義の場合は従来の `talk` / `whisper` プロンプトにフォールバックする。

## 設定テンプレート

`config/` 配下に3種類のテンプレートがある:

| テンプレート | 内容 |
|---|---|
| `config.realtime.yml.example` | リアルタイムモード（日本語） |
| `config.jp.yml.example` | ターン制モード（日本語） |
| `config.en.yml.example` | ターン制モード（英語） |

## その他

プロトコルや元となるエージェントについては [aiwolf-nlp-agent](https://github.com/aiwolfdial/aiwolf-nlp-agent) を参照。
