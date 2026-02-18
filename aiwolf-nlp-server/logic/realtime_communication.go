package logic

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// realtimeMessage はエージェントからのリアルタイムメッセージを表します
type realtimeMessage struct {
	agent *model.Agent
	text  string
}

// リアルタイム通信のデフォルト値
const (
	defaultPhaseTimeout   = 120 * time.Second
	defaultSilenceTimeout = 30 * time.Second
	defaultDrainTimeout   = 2 * time.Second
)

// conductRealtimeCommunication はリアルタイム（グループチャット方式）でトーク/ウィスパーを行います
// エージェントは任意のタイミングで発言でき、発言は即座に全員にブロードキャストされます
func (g *Game) conductRealtimeCommunication(request model.Request) {
	var agents []*model.Agent
	var talkSetting *model.TalkSetting
	var talkList *[]model.Talk
	var startRequest model.Request
	var broadcastRequest model.Request
	var endRequest model.Request

	switch request {
	case model.R_TALK:
		agents = g.getAliveAgents()
		talkSetting = &g.setting.Talk.TalkSetting
		talkList = &g.getCurrentGameStatus().Talks
		startRequest = model.R_TALK_START
		broadcastRequest = model.R_TALK_BROADCAST
		endRequest = model.R_TALK_END
	case model.R_WHISPER:
		agents = g.getAliveWerewolves()
		talkSetting = &g.setting.Whisper.TalkSetting
		talkList = &g.getCurrentGameStatus().Whispers
		startRequest = model.R_WHISPER_START
		broadcastRequest = model.R_WHISPER_BROADCAST
		endRequest = model.R_WHISPER_END
	default:
		return
	}

	if len(agents) < 2 {
		slog.Warn("エージェント数が2未満のため、リアルタイム通信を行いません", "id", g.id, "agentNum", len(agents))
		return
	}

	// #3: タイムアウト値のゼロ値チェック・デフォルト適用
	phaseTimeoutDuration := g.config.Game.Realtime.PhaseTimeout
	if phaseTimeoutDuration <= 0 {
		phaseTimeoutDuration = defaultPhaseTimeout
		slog.Warn("phase_timeoutが未設定のため、デフォルト値を適用します", "id", g.id, "default", phaseTimeoutDuration)
	}
	silenceTimeoutDuration := g.config.Game.Realtime.SilenceTimeout
	if silenceTimeoutDuration <= 0 {
		silenceTimeoutDuration = defaultSilenceTimeout
		slog.Warn("silence_timeoutが未設定のため、デフォルト値を適用します", "id", g.id, "default", silenceTimeoutDuration)
	}

	slog.Info("リアルタイム通信フェーズを開始します", "id", g.id, "day", g.currentDay, "request", request.Type, "agents", len(agents))

	// エージェントごとの残り発言回数とOVER状態を管理
	remainCount := make(map[*model.Agent]int)
	overMap := make(map[*model.Agent]bool)
	lastSpeakTime := make(map[*model.Agent]time.Time)
	for _, agent := range agents {
		remainCount[agent] = talkSetting.MaxCount.PerAgent
		overMap[agent] = false
	}

	// #4: 全体の発言合計カウンタ
	totalTalkCount := 0
	perDay := talkSetting.MaxCount.PerDay

	// 全エージェントにフェーズ開始を通知（ゲーム状態を含む）
	for _, agent := range agents {
		info := g.buildInfo(agent)
		rc := remainCount[agent]
		info.RemainCount = &rc
		startPacket := model.Packet{
			Request: &startRequest,
			Info:    &info,
			Setting: g.setting,
		}
		if request == model.R_TALK {
			startPacket.TalkHistory = talkList
		} else {
			startPacket.WhisperHistory = talkList
		}
		if err := agent.SendNonBlocking(startPacket); err != nil {
			slog.Error("フェーズ開始パケットの送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
		}
	}

	// メッセージ受信チャネルと終了シグナル
	msgChan := make(chan realtimeMessage, 100)
	done := make(chan struct{})
	var wg sync.WaitGroup

	// 各エージェントのリスナーgoroutineを起動
	for _, agent := range agents {
		wg.Add(1)
		go g.startAgentListener(agent, msgChan, done, &wg)
	}

	// メインイベントループ
	phaseTimeout := time.After(phaseTimeoutDuration)
	silenceTimer := time.NewTimer(silenceTimeoutDuration)
	defer silenceTimer.Stop()

	idx := len(*talkList) // 既存のトークの続きからインデックスを開始
	rateLimit := g.config.Game.Realtime.RateLimit

loop:
	for {
		select {
		case msg := <-msgChan:
			// エラー状態またはOVER済みのエージェントからのメッセージは無視
			if msg.agent.HasError || overMap[msg.agent] {
				continue
			}

			text := msg.text

			// OVER処理
			if text == model.T_OVER {
				overMap[msg.agent] = true
				slog.Info("エージェントがOVERしました", "id", g.id, "agent", msg.agent.String())
				if g.allOver(overMap) {
					slog.Info("全エージェントがOVERしたため、フェーズを終了します", "id", g.id)
					break loop
				}
				continue
			}

			// SKIP処理
			if text == model.T_SKIP || text == model.T_FORCE_SKIP {
				slog.Info("エージェントがスキップしました", "id", g.id, "agent", msg.agent.String())
				continue
			}

			// レートリミットチェック
			if rateLimit > 0 {
				if last, ok := lastSpeakTime[msg.agent]; ok {
					if time.Since(last) < rateLimit {
						slog.Warn("レートリミットに達したため、発言を無視しました", "id", g.id, "agent", msg.agent.String())
						continue
					}
				}
			}

			// 残り発言回数チェック
			if remainCount[msg.agent] <= 0 {
				slog.Warn("発言回数が上限に達したため、発言を無視しました", "id", g.id, "agent", msg.agent.String())
				continue
			}
			remainCount[msg.agent]--

			// 文字数制限（per_talk）
			if talkSetting.MaxLength.PerTalk != nil && *talkSetting.MaxLength.PerTalk > 0 {
				textLen := utf8.RuneCountInString(text)
				if textLen > *talkSetting.MaxLength.PerTalk {
					runes := []rune(text)
					text = string(runes[:*talkSetting.MaxLength.PerTalk])
					slog.Warn("発言が最大文字数を超えたため、切り捨てました", "id", g.id, "agent", msg.agent.String())
				}
			}

			// 空テキストチェック
			if utf8.RuneCountInString(text) == 0 {
				slog.Warn("文字数が0のため、発言を無視しました", "id", g.id, "agent", msg.agent.String())
				continue
			}

			// 実際にブロードキャストされる発言のみカウント
			totalTalkCount++

			lastSpeakTime[msg.agent] = time.Now()

			// サイレンスタイマーをリセット
			if !silenceTimer.Stop() {
				select {
				case <-silenceTimer.C:
				default:
				}
			}
			silenceTimer.Reset(silenceTimeoutDuration)

			// トークエントリを作成
			talk := model.Talk{
				Idx:   idx,
				Day:   g.currentDay,
				Turn:  0,
				Agent: *msg.agent,
				Text:  text,
			}
			idx++
			*talkList = append(*talkList, talk)

			slog.Info("リアルタイム発言を受信しました", "id", g.id, "agent", msg.agent.String(), "text", text, "remainCount", remainCount[msg.agent])

			// 全エージェントにブロードキャスト
			broadcastTalks := []model.Talk{talk}
			for _, agent := range agents {
				rc := remainCount[agent]
				broadcastPacket := model.Packet{
					Request: &broadcastRequest,
				}
				if request == model.R_TALK {
					broadcastPacket.TalkHistory = &broadcastTalks
				} else {
					broadcastPacket.WhisperHistory = &broadcastTalks
				}
				// 受信者の残り回数を通知
				broadcastPacket.Info = &model.Info{
					GameID:      g.id,
					Day:         g.currentDay,
					Agent:       agent,
					RemainCount: &rc,
				}
				if err := agent.SendNonBlocking(broadcastPacket); err != nil {
					slog.Error("ブロードキャストの送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
				}
			}

			// ログ記録
			if g.gameLogger != nil {
				if request == model.R_TALK {
					g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,talk,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
				} else {
					g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,whisper,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
				}
			}

			// リアルタイムブロードキャスター（ビューア用）
			if g.realtimeBroadcaster != nil {
				packet := g.getRealtimeBroadcastPacket()
				if request == model.R_TALK {
					packet.Event = "トーク"
				} else {
					packet.Event = "囁き"
				}
				packet.Message = &talk.Text
				packet.BubbleIdx = &msg.agent.Idx
				g.realtimeBroadcaster.Broadcast(packet)
			}

			// TTS
			if g.ttsBroadcaster != nil && msg.agent.Profile != nil {
				g.ttsBroadcaster.BroadcastText(g.id, talk.Text, msg.agent.Profile.VoiceID)
			}

			// #4: PerDay上限到達チェック
			if perDay > 0 && totalTalkCount >= perDay {
				slog.Info("全体の発言合計数が上限に達したため、フェーズを終了します", "id", g.id, "totalTalkCount", totalTalkCount)
				break loop
			}

		case <-phaseTimeout:
			slog.Info("フェーズタイムアウトに達したため、フェーズを終了します", "id", g.id)
			break loop

		case <-silenceTimer.C:
			slog.Info("サイレンスタイムアウトに達したため、フェーズを終了します", "id", g.id)
			break loop
		}
	}

	// リスナーgoroutineを停止
	close(done)
	wg.Wait()

	// 全エージェントにフェーズ終了を通知
	for _, agent := range agents {
		endPacket := model.Packet{
			Request: &endRequest,
		}
		if err := agent.SendNonBlocking(endPacket); err != nil {
			slog.Error("フェーズ終了パケットの送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
		}
	}

	// #1: stale message対策 - WebSocketバッファに残った未読メッセージをドレインする
	// TALK_END送信後、エージェントがTALK_ENDを受信・処理するまでの間に
	// 送信されたメッセージがバッファに残る可能性がある。
	// これを読み捨てないと、次のフェーズ（VOTE等）で SendPacket の ReadMessage が
	// staleメッセージをレスポンスとして誤読してしまう。
	g.drainAgentBuffers(agents)

	// lastIdxMapを更新（DAILY_FINISHで差分が空になるようにする）
	for _, agent := range agents {
		if request == model.R_TALK {
			g.lastTalkIdxMap[agent] = len(*talkList)
		} else {
			g.lastWhisperIdxMap[agent] = len(*talkList)
		}
	}

	slog.Info("リアルタイム通信フェーズを終了しました", "id", g.id, "day", g.currentDay, "totalTalks", len(*talkList))
}

// startAgentListener は1エージェント分のWebSocket読み取りgoroutineです
func (g *Game) startAgentListener(agent *model.Agent, msgChan chan<- realtimeMessage, done <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-done:
			agent.Connection.SetReadDeadline(time.Time{}) // デッドラインをクリア
			return
		default:
			// 短いデッドラインを設定してdoneチャネルを定期的にチェックできるようにする
			agent.Connection.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, msg, err := agent.Connection.ReadMessage()
			if err != nil {
				// タイムアウトの場合はdoneをチェックして再試行
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// 本当のエラー
				slog.Warn("リアルタイムリスナーでエラーが発生しました", "id", g.id, "agent", agent.String(), "error", err)
				agent.HasError = true
				return
			}
			text := strings.TrimRight(string(msg), "\n")
			select {
			case msgChan <- realtimeMessage{agent: agent, text: text}:
			case <-done:
				agent.Connection.SetReadDeadline(time.Time{})
				return
			}
		}
	}
}

// allOver は全エージェントがOVERしたかどうかを判定します
func (g *Game) allOver(overMap map[*model.Agent]bool) bool {
	for _, isOver := range overMap {
		if !isOver {
			return false
		}
	}
	return true
}

// drainAgentBuffers はフェーズ終了後にWebSocketバッファに残った
// staleメッセージを読み捨てます。
// 各エージェントに対して短いデッドラインでReadMessageを繰り返し、
// タイムアウトするまでバッファ内のメッセージを消費します。
func (g *Game) drainAgentBuffers(agents []*model.Agent) {
	var wg sync.WaitGroup
	for _, agent := range agents {
		if agent.HasError {
			continue
		}
		wg.Add(1)
		go func(a *model.Agent) {
			defer wg.Done()
			drained := 0
			deadline := time.Now().Add(defaultDrainTimeout)
			for time.Now().Before(deadline) {
				a.Connection.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
				_, msg, err := a.Connection.ReadMessage()
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						// タイムアウト = バッファが空になった
						break
					}
					// 接続エラー
					slog.Warn("ドレイン中にエラーが発生しました", "id", g.id, "agent", a.String(), "error", err)
					break
				}
				drained++
				text := strings.TrimRight(string(msg), "\n")
				slog.Info("staleメッセージをドレインしました", "id", g.id, "agent", a.String(), "text", text)
			}
			// デッドラインをクリアして次のフェーズに備える
			a.Connection.SetReadDeadline(time.Time{})
			if drained > 0 {
				slog.Info("ドレイン完了", "id", g.id, "agent", a.String(), "drained", drained)
			}
		}(agent)
	}
	wg.Wait()
}
