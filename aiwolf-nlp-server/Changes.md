# aiwolf-nlp-server リアルタイム通信対応

## 変更概要

トーク/ウィスパーフェーズをターン制からリアルタイム（グループチャット方式）に変更可能にしました。
設定ファイルの `realtime.enable: true` で切り替えられます。

## 変更ファイル一覧

### 新規ファイル
| ファイル | 説明 |
|---|---|
| `logic/realtime_communication.go` | リアルタイム通信のメインロジック（約400行） |
| `config/realtime_5.yml` | リアルタイムモード用のサンプル設定ファイル |
| `doc/ja/realtime_protocol.md` | リアルタイムプロトコルのドキュメント |

### 変更ファイル
| ファイル | 変更内容 |
|---|---|
| `logic/communication.go` | `doTalk()` / `doWhisper()` にリアルタイムモードへのルーティング追加 |
| `model/request.go` | 6つの新リクエストタイプ追加（TALK_START, TALK_BROADCAST, TALK_END, WHISPER_*） |
| `model/config.go` | `RealtimeConfig` 構造体追加、`GameConfig` に `Realtime` フィールド追加 |
| `model/agent.go` | `SendNonBlocking()` メソッド追加（ブロードキャスト用） |

### 変更なし
投票、占い、護衛、襲撃、ゲーム進行ロジック、接続管理、マッチング等は変更なし。

## バグ修正（レビュー指摘対応）

### #1 [重大] stale message問題の修正
- `drainAgentBuffers()` を追加
- TALK_END送信後、各エージェントのWebSocketバッファに残った未読メッセージを読み捨てる
- これにより次のフェーズ（VOTE等）でstaleメッセージが誤読されることを防止

### #3 [中] ゼロタイムアウトでフェーズ即終了する問題の修正
- `phase_timeout` / `silence_timeout` が未設定（ゼロ値）の場合にデフォルト値を適用
- デフォルト: phase_timeout=120s, silence_timeout=30s

### #4 [低] PerDay合計発言数制限の適用
- 全エージェントの発言合計数をカウントし、`max_count.per_day` に達したらフェーズを終了

### #2 [中] HasError データ競合（未修正）
- 既存の Agent 構造体の設計に由来する問題のため、今回のスコープ外
- 将来的に atomic または sync.Mutex での同期が必要

## 使い方

既存リポジトリに対して、上記ファイルを上書き/追加してください。

```bash
# リアルタイムモードで起動
./aiwolf-nlp-server -c ./config/realtime_5.yml
```

## 通信フロー

```
[従来]  サーバ→A: TALK  A→サーバ: 発言  サーバ→B: TALK  B→サーバ: 発言 ...

[リアルタイム]
  サーバ → 全員: TALK_START（ゲーム状態）
  
  エージェントが自分の判断で発言:
    A → サーバ: 「私は占い師です」
    サーバ → 全員: TALK_BROADCAST（Aの発言）
    B → サーバ: 「Aさん、証拠は？」
    サーバ → 全員: TALK_BROADCAST（Bの発言）
    A → サーバ: 「Agent[03]を占って人狼でした」
    サーバ → 全員: TALK_BROADCAST（Aの発言）
    ...
  
  サーバ → 全員: TALK_END
```