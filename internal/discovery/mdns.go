package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/sirupsen/logrus"
)

// Service mDNS 服务
type Service struct {
	server      *mdns.Server
	serviceName string
	port        int
	deviceID    string
	deviceName  string
	logger      *logrus.Logger
}

// NewService 创建 mDNS 服务
func NewService(serviceName string, port int, deviceID, deviceName string, logger *logrus.Logger) *Service {
	return &Service{
		serviceName: serviceName,
		port:        port,
		deviceID:    deviceID,
		deviceName:  deviceName,
		logger:      logger,
	}
}

// Start 启动 mDNS 服务
func (s *Service) Start() error {
	// 获取主机名
	host, err := getHostname()
	if err != nil {
		return fmt.Errorf("获取主机名失败: %w", err)
	}

	// 获取本机 IP
	ips, err := getLocalIPs()
	if err != nil {
		return fmt.Errorf("获取本机 IP 失败: %w", err)
	}

	// 创建服务信息
	info := []string{
		fmt.Sprintf("device_id=%s", s.deviceID),
		fmt.Sprintf("device_name=%s", s.deviceName),
	}

	service, err := mdns.NewMDNSService(
		host,
		s.serviceName,
		"",
		"",
		s.port,
		ips,
		info,
	)
	if err != nil {
		return fmt.Errorf("创建 mDNS 服务失败: %w", err)
	}

	// 启动 mDNS 服务器
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("启动 mDNS 服务器失败: %w", err)
	}

	s.server = server
	s.logger.Infof("mDNS 服务已启动: %s 端口 %d", s.serviceName, s.port)

	return nil
}

// Stop 停止 mDNS 服务
func (s *Service) Stop() error {
	if s.server != nil {
		if err := s.server.Shutdown(); err != nil {
			return err
		}
		s.logger.Info("mDNS 服务已停止")
	}
	return nil
}

// DiscoveredDevice 发现的设备
type DiscoveredDevice struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
}

// Scan 扫描局域网设备
func Scan(serviceName string, timeout time.Duration, logger *logrus.Logger) ([]*DiscoveredDevice, error) {
	entries := make(chan *mdns.ServiceEntry, 10)
	devices := make([]*DiscoveredDevice, 0)

	// 启动查询
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		if err := mdns.Query(&mdns.QueryParam{
			Service: serviceName,
			Domain:  "local",
			Timeout: timeout,
			Entries: entries,
		}); err != nil {
			logger.Errorf("mDNS 查询失败: %v", err)
		}
		close(entries)
	}()

	// 收集结果
	for {
		select {
		case entry := <-entries:
			if entry == nil {
				return devices, nil
			}

			// 解析 TXT 记录
			deviceID := ""
			deviceName := ""
			for _, info := range entry.InfoFields {
				if len(info) > 10 && info[:10] == "device_id=" {
					deviceID = info[10:]
				}
				if len(info) > 12 && info[:12] == "device_name=" {
					deviceName = info[12:]
				}
			}

			// 获取 IP
			ip := ""
			if entry.AddrV4 != nil {
				ip = entry.AddrV4.String()
			} else if entry.AddrV6 != nil {
				ip = entry.AddrV6.String()
			}

			if deviceID != "" && ip != "" {
				devices = append(devices, &DiscoveredDevice{
					DeviceID:   deviceID,
					DeviceName: deviceName,
					IP:         ip,
					Port:       entry.Port,
				})
				logger.Debugf("发现设备: %s (%s) at %s:%d", deviceName, deviceID, ip, entry.Port)
			}

		case <-ctx.Done():
			return devices, nil
		}
	}
}

// getHostname 获取主机名
func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "airlink-host", nil // 使用默认值
	}
	return hostname, nil
}

// getLocalIPs 获取本机 IP 地址
func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("未找到本机 IP 地址")
	}

	return ips, nil
}
