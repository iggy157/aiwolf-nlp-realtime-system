package core

import (
	"net/http"

	"github.com/aiwolfdial/aiwolf-nlp-server/logic"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/gin-gonic/gin"
)

// API レスポンス型

// CostReport はエージェントから送信されるコストレポートです
type CostReport struct {
	GameID       string  `json:"game_id"`
	Agent        string  `json:"agent"`
	Team         string  `json:"team"`
	Model        string  `json:"model"`
	LLMType      string  `json:"llm_type"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	InputCost    float64 `json:"input_cost"`
	OutputCost   float64 `json:"output_cost"`
	TotalCost    float64 `json:"total_cost"`
	CallCount    int     `json:"call_count"`
}

// GameCostSummary はゲームごとのコスト集計です
type GameCostSummary struct {
	GameID      string       `json:"game_id"`
	Agents      []CostReport `json:"agents"`
	TotalCost   float64      `json:"total_cost"`
	TotalInput  int          `json:"total_input_tokens"`
	TotalOutput int          `json:"total_output_tokens"`
}

type gameInfo struct {
	ID        string                  `json:"id"`
	Day       int                     `json:"day"`
	IsDaytime bool                    `json:"is_daytime"`
	Phase     string                  `json:"phase"`
	Paused    bool                    `json:"paused"`
	Finished  bool                    `json:"finished"`
	WinSide   string                  `json:"win_side,omitempty"`
	Agents    []logic.AgentStatusInfo `json:"agents"`
}

type statusResponse struct {
	ServerVersion  string `json:"server_version"`
	ManualStart    bool   `json:"manual_start"`
	SpawnerEnabled bool   `json:"spawner_enabled"`
	WaitingRoom    struct {
		Required int        `json:"required"`
		Teams    []TeamInfo `json:"teams"`
		Total    int        `json:"total"`
	} `json:"waiting_room"`
	Games     []gameInfo        `json:"games"`
	Costs     []GameCostSummary `json:"costs"`
	Processes []struct {
		ID      string `json:"id"`
		Team    string `json:"team"`
		Count   int    `json:"count"`
		Model   string `json:"model"`
		LLMType string `json:"llm_type"`
		Status  string `json:"status"`
	} `json:"processes"`
}

// registerAPIRoutes は管理用REST APIルートを登録します
func (s *Server) registerAPIRoutes(router *gin.Engine) {
	api := router.Group("/api")
	api.GET("/status", s.handleStatus)
	api.POST("/game/start", s.handleGameStart)
	api.POST("/game/:id/pause", s.handleGamePause)
	api.POST("/game/:id/resume", s.handleGameResume)
	api.POST("/cost/report", s.handleCostReport)
	s.registerSpawnRoutes(api)
}

// handleStatus はサーバの現在の状態を返します
func (s *Server) handleStatus(c *gin.Context) {
	teams := s.waitingRoom.ListTeams()
	total := 0
	for _, t := range teams {
		total += t.Count
	}

	var games []gameInfo
	s.games.Range(func(key, value any) bool {
		game, ok := value.(*logic.Game)
		if !ok {
			return true
		}
		winSide := ""
		if game.IsFinished() {
			winSide = string(game.GetWinSide())
		}
		games = append(games, gameInfo{
			ID:        game.GetID(),
			Day:       game.GetDay(),
			IsDaytime: game.GetIsDaytime(),
			Phase:     game.GetPhase(),
			Paused:    game.IsPaused(),
			Finished:  game.IsFinished(),
			WinSide:   winSide,
			Agents:    game.GetAgentStatusInfos(),
		})
		return true
	})

	resp := statusResponse{
		ServerVersion:  Version.Version,
		ManualStart:    s.config.Server.ManualStart,
		SpawnerEnabled: s.config.AgentSpawner.Enable,
	}
	resp.WaitingRoom.Required = s.config.Game.AgentCount
	if teams == nil {
		resp.WaitingRoom.Teams = []TeamInfo{}
	} else {
		resp.WaitingRoom.Teams = teams
	}
	resp.WaitingRoom.Total = total
	if games == nil {
		resp.Games = []gameInfo{}
	} else {
		resp.Games = games
	}

	// コスト集計
	var costs []GameCostSummary
	s.costReports.Range(func(key, value any) bool {
		gameID := key.(string)
		reports := value.([]CostReport)
		summary := GameCostSummary{
			GameID: gameID,
			Agents: reports,
		}
		for _, r := range reports {
			summary.TotalCost += r.TotalCost
			summary.TotalInput += r.InputTokens
			summary.TotalOutput += r.OutputTokens
		}
		costs = append(costs, summary)
		return true
	})
	if costs == nil {
		resp.Costs = []GameCostSummary{}
	} else {
		resp.Costs = costs
	}

	// spawnされたプロセス一覧
	type procInfo struct {
		ID      string `json:"id"`
		Team    string `json:"team"`
		Count   int    `json:"count"`
		Model   string `json:"model"`
		LLMType string `json:"llm_type"`
		Status  string `json:"status"`
	}
	var procs []procInfo
	s.spawnedProcesses.Range(func(key, value any) bool {
		proc := value.(*SpawnedProcess)
		proc.mu.Lock()
		procs = append(procs, procInfo{
			ID:      proc.ID,
			Team:    proc.Team,
			Count:   proc.Count,
			Model:   proc.Model,
			LLMType: proc.LLMType,
			Status:  proc.Status,
		})
		proc.mu.Unlock()
		return true
	})
	// レスポンス型に合わせて変換
	for _, p := range procs {
		resp.Processes = append(resp.Processes, struct {
			ID      string `json:"id"`
			Team    string `json:"team"`
			Count   int    `json:"count"`
			Model   string `json:"model"`
			LLMType string `json:"llm_type"`
			Status  string `json:"status"`
		}(p))
	}
	if resp.Processes == nil {
		resp.Processes = make([]struct {
			ID      string `json:"id"`
			Team    string `json:"team"`
			Count   int    `json:"count"`
			Model   string `json:"model"`
			LLMType string `json:"llm_type"`
			Status  string `json:"status"`
		}, 0)
	}

	c.JSON(http.StatusOK, resp)
}

// handleGameStart は待機部屋のエージェントでゲームを開始します
func (s *Server) handleGameStart(c *gin.Context) {
	var game *logic.Game

	if s.config.Matching.IsOptimize {
		s.waitingRoom.connections.Range(func(key, value any) bool {
			team := key.(string)
			s.matchOptimizer.updateTeam(team)
			return true
		})
		matches := s.matchOptimizer.getMatches()
		roleMapConns, err := s.waitingRoom.GetConnectionsWithMatchOptimizer(matches)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		game = logic.NewGameWithRole(&s.config, s.gameSetting, roleMapConns)
	} else {
		connections, err := s.waitingRoom.GetConnections()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		game = logic.NewGame(&s.config, s.gameSetting, connections)
	}

	if s.jsonLogger != nil {
		game.SetJSONLogger(s.jsonLogger)
	}
	if s.gameLogger != nil {
		game.SetGameLogger(s.gameLogger)
	}
	if s.realtimeBroadcaster != nil {
		game.SetRealtimeBroadcaster(s.realtimeBroadcaster)
	}
	if s.ttsBroadcaster != nil {
		game.SetTTSBroadcaster(s.ttsBroadcaster)
	}
	s.games.Store(game.GetID(), game)

	go func() {
		winSide := game.Start()
		if s.config.Matching.IsOptimize {
			if winSide != model.T_NONE {
				s.matchOptimizer.setMatchEnd(game.GetRoleTeamNamesMap())
			} else {
				s.matchOptimizer.setMatchWeight(game.GetRoleTeamNamesMap(), 0)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"id":      game.GetID(),
		"message": "ゲームを開始しました",
	})
}

// handleGamePause は指定されたゲームを一時停止します
func (s *Server) handleGamePause(c *gin.Context) {
	id := c.Param("id")
	value, ok := s.games.Load(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "ゲームが見つかりません"})
		return
	}
	game, ok := value.(*logic.Game)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部エラー"})
		return
	}
	if game.IsFinished() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ゲームは既に終了しています"})
		return
	}
	game.Pause()
	c.JSON(http.StatusOK, gin.H{"message": "ゲームを一時停止しました", "id": id})
}

// handleGameResume は指定されたゲームを再開します
func (s *Server) handleGameResume(c *gin.Context) {
	id := c.Param("id")
	value, ok := s.games.Load(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "ゲームが見つかりません"})
		return
	}
	game, ok := value.(*logic.Game)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部エラー"})
		return
	}
	game.Resume()
	c.JSON(http.StatusOK, gin.H{"message": "ゲームを再開しました", "id": id})
}

// handleCostReport はエージェントからのコストレポートを受信します
func (s *Server) handleCostReport(c *gin.Context) {
	var report CostReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不正なリクエスト"})
		return
	}
	if report.GameID == "" || report.Agent == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game_id と agent は必須です"})
		return
	}

	// ゲームIDごとにレポートを蓄積
	for {
		value, loaded := s.costReports.LoadOrStore(report.GameID, []CostReport{report})
		if !loaded {
			break
		}
		existing := value.([]CostReport)
		updated := append(existing, report)
		if s.costReports.CompareAndSwap(report.GameID, value, updated) {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "コストレポートを受信しました"})
}
