# aiwolf-nlp-server

人狼知能コンテスト（自然言語部門） のゲームサーバ。

> モノレポ（realtime_system）での使い方は[ルートREADME](../README.md)を参照。

## ドキュメント

- [設定ファイルについて](/doc/ja/config.md)
- [ゲームロジックの実装について](/doc/ja/logic.md)
- [プロトコルの実装について](/doc/ja/protocol.md)

## スタンドアロン実行（バイナリ）

サーバ単体で使用する場合、ビルド済みバイナリをダウンロードして実行できる。

サンプルエージェントについては [aiwolfdial/aiwolf-nlp-agent](https://github.com/aiwolfdial/aiwolf-nlp-agent) を参照。

デフォルトのサーバアドレスは `ws://127.0.0.1:8080/ws`。同じチーム名のエージェント同士のみをマッチングさせる自己対戦モードがデフォルトで有効。

### Linux

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-linux-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-linux-amd64
./aiwolf-nlp-server-linux-amd64 -c ./default_5.yml
```

### Windows

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-windows-amd64.exe
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
.\aiwolf-nlp-server-windows-amd64.exe -c .\default_5.yml
```

### macOS (Intel)

> [!NOTE]
> 開発元が不明なアプリケーションとしてブロックされる場合があります。\
> 下記サイトを参考に、実行許可を与えてください。
> <https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-amd64
./aiwolf-nlp-server-darwin-amd64 -c ./default_5.yml
```

### macOS (Apple Silicon)

> [!NOTE]
> 開発元が不明なアプリケーションとしてブロックされる場合があります。\
> 下記サイトを参考に、実行許可を与えてください。
> <https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-arm64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-arm64
./aiwolf-nlp-server-darwin-arm64 -c ./default_5.yml
```
