# aiwolf-nlp-common

人狼知能コンテスト（自然言語部門） のエージェント向けの共通パッケージです。\
ゲームサーバから送信されるJSON形式のデータをオブジェクトに変換するためのパッケージです。

```python
import json

from aiwolf_nlp_common.packet import Packet

value = json.loads(
    """{"request":"INITIALIZE"}""",
)
packet = Packet.from_dict(value)

print(packet.request) # Request.INITIALIZE
```

詳細については下記のプロトコルの説明やゲームサーバのソースコードを参考にしてください。\
[プロトコルの実装について](https://github.com/aiwolfdial/aiwolf-nlp-server/blob/main/doc/ja/config.md)

## リアルタイム通信対応

リアルタイム通信プロトコル用に、`Request` enum に以下の6値が追加されています。

| 値 | 説明 |
|---|---|
| `TALK_START` | リアルタイムトークフェーズの開始 |
| `TALK_BROADCAST` | トーク発言のブロードキャスト |
| `TALK_END` | リアルタイムトークフェーズの終了 |
| `WHISPER_START` | リアルタイム囁きフェーズの開始 |
| `WHISPER_BROADCAST` | 囁き発言のブロードキャスト |
| `WHISPER_END` | リアルタイム囁きフェーズの終了 |

`Packet.from_dict()` はこれらのリクエストタイプを自動的にパースします。

```python
import json
from aiwolf_nlp_common.packet import Packet, Request

value = json.loads('{"request":"TALK_START","info":{...},"setting":{...}}')
packet = Packet.from_dict(value)
print(packet.request)  # Request.TALK_START
```

> [!NOTE]
> リアルタイム通信を使用する場合は、この更新版パッケージのインストールが必要です。\
> サーバ側で `realtime.enable: false` の場合、これらの新しいリクエストタイプは送信されないため、従来のエージェントへの影響はありません。

## インストール方法

```bash
python -m pip install aiwolf-nlp-common
```

## 運営向け

パッケージ管理ツールとしてuvの使用を推奨します。

```bash
git clone https://github.com/aiwolfdial/aiwolf-nlp-common.git
cd aiwolf-nlp-common
uv venv
uv sync
```

### パッケージのビルド

```bash
pyright --createstub aiwolf_nlp_common
uv build
```

### パッケージの配布

#### PyPI

```bash
uv publish --token <PyPIのアクセストークン>
```

#### TestPyPI

```bash
uv publish --publish-url https://test.pypi.org/legacy/ --token <TestPyPIのアクセストークン>
```

uvを使用しない場合については、パッケージ化と配布については下記のページを参考にしてください。\
[Packaging and distributing projects](https://packaging.python.org/en/latest/guides/distributing-packages-using-setuptools/)
