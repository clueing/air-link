package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/clue/air-link/internal/config"
	"github.com/clue/air-link/internal/security"
	"github.com/clue/air-link/internal/signaling"
	"github.com/sirupsen/logrus"
)

//go:embed web
var webFS embed.FS

// Server HTTP 服务器
type Server struct {
	config      *config.Config
	roomManager *signaling.RoomManager
	rateLimiter *security.RateLimiter
	wsHandler   *WSHandler
	logger      *logrus.Logger
	deviceID    string
	deviceName  string
}

// NewServer 创建服务器
func NewServer(cfg *config.Config, deviceID, deviceName string, logger *logrus.Logger) *Server {
	// 创建房间管理器
	roomManager := signaling.NewRoomManager(
		cfg.Room.MaxDevices,
		cfg.GetPINExpiryDuration(),
	)

	// 创建速率限制器
	rateLimiter := security.NewRateLimiter(
		cfg.Security.RateLimit.MaxAttempts,
		cfg.GetRateLimitWindowDuration(),
		cfg.GetRateLimitLockoutDuration(),
	)

	// 创建 WebSocket 处理器
	wsHandler := NewWSHandler(
		roomManager,
		rateLimiter,
		cfg.WebRTC.STUNServers,
		logger,
	)

	return &Server{
		config:      cfg,
		roomManager: roomManager,
		rateLimiter: rateLimiter,
		wsHandler:   wsHandler,
		logger:      logger,
		deviceID:    deviceID,
		deviceName:  deviceName,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 设置路由
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc("/ws", s.wsHandler.HandleWebSocket)

	// API 端点
	mux.HandleFunc("/api/ping", s.handlePing)
	mux.HandleFunc("/api/config", s.handleConfig)

	// 静态文件
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("加载静态文件失败: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(webContent)))

	// 启动 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.logger.Infof("服务器启动: http://%s", addr)

	// 自动打开浏览器
	if s.config.Server.AutoOpenBrowser {
		go s.openBrowser(fmt.Sprintf("http://localhost:%d", s.config.Server.Port))
	}

	return http.ListenAndServe(addr, mux)
}

// Stop 停止服务器
func (s *Server) Stop() error {
	return nil
}

// handlePing 处理 ping 请求
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Device-ID", s.deviceID)
	w.Header().Set("X-Device-Name", s.deviceName)
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id":   s.deviceID,
		"device_name": s.deviceName,
		"version":     "1.0.0",
	})
}

// handleConfig 处理配置请求
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stun_servers": s.config.WebRTC.STUNServers,
		"turn_servers": s.config.WebRTC.TURNServers,
	})
}

// openBrowser 打开浏览器
func (s *Server) openBrowser(url string) {
	time.Sleep(500 * time.Millisecond) // 等待服务器启动

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		s.logger.Warnf("无法自动打开浏览器: %v", err)
	}
}
