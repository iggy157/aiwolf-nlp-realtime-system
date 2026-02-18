package core

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

// SpawnedProcess はspawnされたエージェントプロセスの情報です
type SpawnedProcess struct {
	ID      string `json:"id"`
	Team    string `json:"team"`
	Count   int    `json:"count"`
	Model   string `json:"model"`
	LLMType string `json:"llm_type"`
	Status  string `json:"status"` // "running", "stopped", "error"
	cmd     *exec.Cmd
	mu      sync.Mutex
}

// SpawnRequest はエージェントspawnのリクエストです
type SpawnRequest struct {
	Team        string  `json:"team"`
	Count       int     `json:"count"`
	LLMType     string  `json:"llm_type"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

// registerSpawnRoutes はエージェントspawn関連のルートを登録します
func (s *Server) registerSpawnRoutes(api *gin.RouterGroup) {
	if !s.config.AgentSpawner.Enable {
		return
	}
	api.POST("/agent/spawn", s.handleAgentSpawn)
	api.GET("/agent/processes", s.handleAgentProcesses)
	api.POST("/agent/:id/stop", s.handleAgentStop)
}

// handleAgentSpawn はエージェントプロセスをspawnします
func (s *Server) handleAgentSpawn(c *gin.Context) {
	var req SpawnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不正なリクエスト"})
		return
	}

	if req.Team == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team は必須です"})
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.LLMType == "" {
		req.LLMType = "google"
	}
	if req.Model == "" {
		switch req.LLMType {
		case "openai":
			req.Model = "gpt-4o-mini"
		case "google":
			req.Model = "gemini-2.0-flash-lite"
		case "ollama":
			req.Model = "llama3.1"
		}
	}
	if req.Temperature <= 0 {
		req.Temperature = 0.7
	}

	// テンプレート設定を読み込む
	templatePath := s.config.AgentSpawner.ConfigTemplate
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(s.config.AgentSpawner.AgentDir, templatePath)
	}
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("設定テンプレートの読み込みに失敗: %v", err)})
		return
	}

	// テンプレートをパースして上書き
	var config map[string]interface{}
	if err := yaml.Unmarshal(templateData, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("設定テンプレートのパースに失敗: %v", err)})
		return
	}

	// WebSocket URLを自身のサーバに向ける
	wsURL := fmt.Sprintf("ws://%s:%d/ws", s.config.Server.WebSocket.Host, s.config.Server.WebSocket.Port)
	if s.config.Server.WebSocket.Host == "" || s.config.Server.WebSocket.Host == "0.0.0.0" {
		wsURL = fmt.Sprintf("ws://127.0.0.1:%d/ws", s.config.Server.WebSocket.Port)
	}

	if ws, ok := config["web_socket"].(map[interface{}]interface{}); ok {
		ws["url"] = wsURL
		ws["auto_reconnect"] = false
	}
	if agent, ok := config["agent"].(map[interface{}]interface{}); ok {
		agent["team"] = req.Team
		agent["num"] = req.Count
	}
	if llm, ok := config["llm"].(map[interface{}]interface{}); ok {
		llm["type"] = req.LLMType
	}

	// LLMプロバイダ設定を上書き
	providerKey := req.LLMType
	if provider, ok := config[providerKey].(map[interface{}]interface{}); ok {
		provider["model"] = req.Model
		provider["temperature"] = req.Temperature
	}

	// 一時設定ファイルを書き出す
	tmpFile, err := os.CreateTemp("", "aiwolf-agent-*.yml")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "一時ファイルの作成に失敗"})
		return
	}
	configBytes, _ := yaml.Marshal(config)
	tmpFile.Write(configBytes)
	tmpFile.Close()

	// プロセスをspawn
	pythonCmd := s.config.AgentSpawner.PythonCmd
	if pythonCmd == "" {
		pythonCmd = "python"
	}

	// コマンドを構築 (例: "uv run python" → ["uv", "run", "python"])
	cmdParts := strings.Fields(pythonCmd)
	cmdArgs := append(cmdParts[1:], "src/main.py", "-c", tmpFile.Name())
	cmd := exec.Command(cmdParts[0], cmdArgs...)
	cmd.Dir = s.config.AgentSpawner.AgentDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.Remove(tmpFile.Name())
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("プロセスの起動に失敗: %v", err)})
		return
	}

	processID := fmt.Sprintf("proc-%d", cmd.Process.Pid)
	proc := &SpawnedProcess{
		ID:      processID,
		Team:    req.Team,
		Count:   req.Count,
		Model:   req.Model,
		LLMType: req.LLMType,
		Status:  "running",
		cmd:     cmd,
	}
	s.spawnedProcesses.Store(processID, proc)

	// プロセス終了を監視
	go func() {
		err := cmd.Wait()
		proc.mu.Lock()
		if err != nil {
			proc.Status = "error"
		} else {
			proc.Status = "stopped"
		}
		proc.mu.Unlock()
		os.Remove(tmpFile.Name())
		slog.Info("spawnしたエージェントプロセスが終了しました", "id", processID, "team", req.Team)
	}()

	slog.Info("エージェントプロセスをspawnしました", "id", processID, "team", req.Team, "count", req.Count, "model", req.Model)
	c.JSON(http.StatusOK, gin.H{
		"id":      processID,
		"team":    req.Team,
		"count":   req.Count,
		"message": "エージェントプロセスを起動しました",
	})
}

// handleAgentProcesses はspawnされたプロセス一覧を返します
func (s *Server) handleAgentProcesses(c *gin.Context) {
	var processes []SpawnedProcess
	s.spawnedProcesses.Range(func(key, value any) bool {
		proc := value.(*SpawnedProcess)
		proc.mu.Lock()
		processes = append(processes, SpawnedProcess{
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
	if processes == nil {
		processes = []SpawnedProcess{}
	}
	c.JSON(http.StatusOK, processes)
}

// handleAgentStop はspawnされたプロセスを停止します
func (s *Server) handleAgentStop(c *gin.Context) {
	id := c.Param("id")
	value, ok := s.spawnedProcesses.Load(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "プロセスが見つかりません"})
		return
	}
	proc := value.(*SpawnedProcess)
	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "プロセスは実行中ではありません"})
		return
	}

	if proc.cmd.Process != nil {
		proc.cmd.Process.Kill()
		proc.Status = "stopped"
	}

	c.JSON(http.StatusOK, gin.H{"message": "プロセスを停止しました", "id": id})
}
