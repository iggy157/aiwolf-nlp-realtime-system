# aiwolf-nlp-viewer リアルタイム通信対応 + チャットUI刷新

## 変更ファイル

| ファイル | 種別 | 内容 |
|---|---|---|
| `src/lib/types/agent.ts` | **変更** | Request enumに6値追加、`getAgentAvatar()`・`getAgentColor()` ユーティリティ追加 |
| `src/lib/utils/agent-socket.ts` | **変更** | リアルタイムパケット処理、BROADCAST時のinfo上書き防止、`isRealtimePhase`状態管理 |
| `src/routes/agent/+page.svelte` | **変更** | リアルタイム状態をTalkColumn/ActionBarに伝搬、レイアウト調整 |
| `src/routes/agent/ChatBubble.svelte` | **書き換え** | LINE風チャットバブルにリデザイン |
| `src/routes/agent/TalkColumn.svelte` | **書き換え** | グループチャット風コンテナ + 自動スクロール + LIVEインジケーター + i18n化 |
| `src/routes/agent/AgentColumn.svelte` | **変更** | レイアウトサイズを親コンテナ追従に変更 |
| `src/routes/agent/ActionBar.svelte` | **変更** | リアルタイムモード用入力UI追加（LIVE表示・残り発言回数・タイマー非表示） |
| `src/i18n/ja.json` | **変更** | リアルタイムRequest翻訳6件 + 新UIキー5件追加 |
| `src/i18n/en.json` | **変更** | リアルタイムRequest翻訳6件 + 新UIキー5件追加 |

## UIの変更点

### チャットバブル（ChatBubble.svelte）
- **エージェント別カラーバブル** — 13色のパレットでエージェントごとに異なる背景色
- **アバターアイコン** — Agent[XX] 形式でも画像を自動割り当て（番号→male/female画像）
- **連続発言のグルーピング** — 同一エージェントの連続メッセージはアバター・名前を省略
- **Over/Skip** — 小さなシステムメッセージ風のピル型表示

### チャットコンテナ（TalkColumn.svelte）
- **日付区切り線** — タブ切り替え → インライン区切り（チャットアプリ風）
- **自動スクロール** — 新メッセージ受信時にスムーズスクロール（上スクロール中は停止）
- **LIVEインジケーター** — リアルタイムフェーズ中は緑パルスアニメーション
- **i18n対応** — ハードコード日本語を全て翻訳キーに置換

### ActionBar（リアルタイムモード）
- タイマー/プログレスバー → **LIVE表示** + **残り発言回数バッジ**
- Over ボタン + テキスト入力 + 送信ボタンのシンプルなUI
- 文字数計算をリアルタイムフェーズ用に修正（TALK設定を参照）

### レイアウト
- AgentColumn を固定幅340pxに変更、チャットカラムにより多くのスペース