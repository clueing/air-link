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
	"github.com/clue/air-link/internal/discovery"
	"github.com/clue/air-link/internal/security"
	"github.com/clue/air-link/internal/signaling"
	"github.com/sirupsen/logrus"
)

//go:embed web
var webFS embed.FS

// Server HTTP 服务器
type Server struct {
	config        *config.Config
	roomManager   *signaling.RoomManager
	rateLimiter   *security.RateLimiter
	wsHandler     *WSHandler
	mdnsService   *discovery.Service
	httpProbe     *discovery.HTTPProbe
	logger        *logrus.Logger
	deviceID      string
	deviceName    string
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

	// 创建 mDNS 服务
	var mdnsService *discovery.Service
	if cfg.Discovery.MDNSEnabled {
		mdnsService = discovery.NewService(
			cfg.Discovery.ServiceName,
			cfg.Server.Port,
			deviceID,
			deviceName,
			logger,
		)
	}

	// 创建 HTTP 探测器
	var httpProbe *discovery.HTTPProbe
	if cfg.Discovery.HTTPProbeEnabled {
		httpProbe = discovery.NewHTTPProbe(500*time.Millisecond, logger)
	}

	return &Server{
		config:      cfg,
		roomManager: roomManager,
		rateLimiter: rateLimiter,
		wsHandler:   wsHandler,
		mdnsService: mdnsService,
		httpProbe:   httpProbe,
		logger:      logger,
		deviceID:    deviceID,
		deviceName:  deviceName,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动 mDNS 服务
	if s.mdnsService != nil {
		if err := s.mdnsService.Start(); err != nil {
			s.logger.Warnf("mDNS 服务启动失败: %v", err)
		}
	}

	// 设置路由
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc("/ws", s.wsHandler.HandleWebSocket)

	// API 端点
	mux.HandleFunc("/api/ping", s.handlePing)
	mux.HandleFunc("/api/scan", s.handleScan)
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
	if s.mdnsService != nil {
		return s.mdnsService.Stop()
	}
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

// handleScan 处理扫描请求
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	devices := make([]interface{}, 0)

	// mDNS 扫描
	if s.mdnsService != nil {
		mdnsDevices, err := discovery.Scan(
			s.config.Discovery.ServiceName,
			2*time.Second,
			s.logger,
		)
		if err != nil {
			s.logger.Errorf("mDNS 扫描失败: %v", err)
		} else {
			for _, d := range mdnsDevices {
				if d.DeviceID != s.deviceID {
					devices = append(devices, d)
				}
			}
		}
	}

	// HTTP 探测（如果 mDNS 未发现设备）
	if len(devices) == 0 && s.httpProbe != nil {
		probeResults, err := s.httpProbe.Scan(s.config.Server.Port, 20)
		if err != nil {
			s.logger.Errorf("HTTP 探测失败: %v", err)
		} else {
			for _, r := range probeResults {
				if r.DeviceID != s.deviceID {
					devices = append(devices, r)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"devices": devices,
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
