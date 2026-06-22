package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	WebRTC   WebRTCConfig   `mapstructure:"webrtc"`
	Room     RoomConfig     `mapstructure:"room"`
	Security SecurityConfig `mapstructure:"security"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Host            string `mapstructure:"host"`
	AutoOpenBrowser bool   `mapstructure:"auto_open_browser"`
}

// WebRTCConfig WebRTC 配置
type WebRTCConfig struct {
	STUNServers []string      `mapstructure:"stun_servers"`
	TURNServers []TURNServer  `mapstructure:"turn_servers"`
}

// TURNServer TURN 服务器配置
type TURNServer struct {
	URL        string `mapstructure:"url"`
	Username   string `mapstructure:"username"`
	Credential string `mapstructure:"credential"`
}

// RoomConfig 房间配置
type RoomConfig struct {
	PINExpiry  int `mapstructure:"pin_expiry"`   // 秒
	MaxDevices int `mapstructure:"max_devices"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	MaxAttempts  int `mapstructure:"max_attempts"`
	WindowTime   int `mapstructure:"window_time"`   // 秒
	LockoutTime  int `mapstructure:"lockout_time"`  // 秒
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	// 设置默认值
	setDefaults()

	// 设置配置文件
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("airlink")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.airlink")
	}

	// 读取环境变量
	viper.SetEnvPrefix("AIRLINK")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// 解析配置
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务器
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.auto_open_browser", true)

	// WebRTC
	viper.SetDefault("webrtc.stun_servers", []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun.miwifi.com:3478",
		"stun:stun.chat.bilibili.com:3478",
		"stun:stun.hitv.com:3478",
		"stun:stun.voipbuster.com:3478",
		"stun:stun.sipnet.net:3478",
	})
	viper.SetDefault("webrtc.turn_servers", []TURNServer{})

	// 房间
	viper.SetDefault("room.pin_expiry", 300)
	viper.SetDefault("room.max_devices", 10)

	// 安全
	viper.SetDefault("security.rate_limit.max_attempts", 5)
	viper.SetDefault("security.rate_limit.window_time", 60)
	viper.SetDefault("security.rate_limit.lockout_time", 600)

	// 日志
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "")
}

// GetPINExpiryDuration 获取 PIN 过期时间
func (c *Config) GetPINExpiryDuration() time.Duration {
	return time.Duration(c.Room.PINExpiry) * time.Second
}

// GetRateLimitWindowDuration 获取速率限制时间窗口
func (c *Config) GetRateLimitWindowDuration() time.Duration {
	return time.Duration(c.Security.RateLimit.WindowTime) * time.Second
}

// GetRateLimitLockoutDuration 获取速率限制锁定时间
func (c *Config) GetRateLimitLockoutDuration() time.Duration {
	return time.Duration(c.Security.RateLimit.LockoutTime) * time.Second
}
