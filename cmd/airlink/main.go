package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/clue/air-link/internal/config"
	"github.com/clue/air-link/internal/server"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	// 命令行参数
	var (
		configPath  = flag.String("config", "", "配置文件路径")
		port        = flag.Int("port", 0, "服务器端口")
		noBrowser   = flag.Bool("no-browser", false, "不自动打开浏览器")
		showVersion = flag.Bool("version", false, "显示版本信息")
		debug       = flag.Bool("debug", false, "开启调试模式")
	)
	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("AirLink %s (%s)\n", version, commit)
		return
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *noBrowser {
		cfg.Server.AutoOpenBrowser = false
	}

	// 配置日志
	logger := logrus.New()
	if *debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		level, err := logrus.ParseLevel(cfg.Logging.Level)
		if err != nil {
			level = logrus.InfoLevel
		}
		logger.SetLevel(level)
	}

	// 日志格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 日志文件
	if cfg.Logging.File != "" {
		file, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Warnf("无法打开日志文件: %v", err)
		} else {
			logger.SetOutput(file)
			defer file.Close()
		}
	}

	// 生成设备 ID（实际部署时应该持久化）
	deviceID := uuid.New().String()
	deviceName := getDefaultDeviceName()

	logger.Infof("AirLink %s 启动中...", version)
	logger.Infof("设备 ID: %s", deviceID)
	logger.Infof("设备名称: %s", deviceName)

	// 创建服务器
	srv := server.NewServer(cfg, deviceID, deviceName, logger)

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("正在关闭服务器...")
	if err := srv.Stop(); err != nil {
		logger.Errorf("关闭服务器失败: %v", err)
	}

	logger.Info("服务器已关闭")
}

// getDefaultDeviceName 获取默认设备名称
func getDefaultDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "AirLink 设备"
	}
	return hostname
}
