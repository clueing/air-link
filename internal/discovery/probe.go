package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProbeResult HTTP 探测结果
type ProbeResult struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

// HTTPProbe HTTP 探测器
type HTTPProbe struct {
	timeout time.Duration
	logger  *logrus.Logger
}

// NewHTTPProbe 创建 HTTP 探测器
func NewHTTPProbe(timeout time.Duration, logger *logrus.Logger) *HTTPProbe {
	return &HTTPProbe{
		timeout: timeout,
		logger:  logger,
	}
}

// Scan 扫描当前子网
func (p *HTTPProbe) Scan(port int, maxConcurrent int) ([]*ProbeResult, error) {
	// 获取本机 IP
	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("获取本机 IP 失败: %w", err)
	}

	// 计算子网范围 (x.x.x.1 - x.x.x.254)
	baseIP := localIP.Mask(localIP.DefaultMask())
	targets := make([]string, 0, 254)
	for i := 1; i <= 254; i++ {
		ip := net.IPv4(baseIP[0], baseIP[1], baseIP[2], byte(i))
		if !ip.Equal(localIP) {
			targets = append(targets, ip.String())
		}
	}

	// 并发探测
	results := make([]*ProbeResult, 0)
	resultsChan := make(chan *ProbeResult, len(targets))
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, ip := range targets {
		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := p.probe(ctx, targetIP, port)
			if result != nil {
				resultsChan <- result
			}
		}(ip)
	}

	// 等待所有探测完成
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// 收集结果
	for result := range resultsChan {
		results = append(results, result)
		p.logger.Debugf("探测到设备: %s at %s:%d", result.DeviceName, result.IP, result.Port)
	}

	return results, nil
}

// probe 探测单个 IP
func (p *HTTPProbe) probe(ctx context.Context, ip string, port int) *ProbeResult {
	url := fmt.Sprintf("http://%s:%d/api/ping", ip, port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	client := &http.Client{
		Timeout: p.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// 解析响应（假设返回 JSON）
	// 这里简化处理，实际应该解析 JSON
	deviceID := resp.Header.Get("X-Device-ID")
	deviceName := resp.Header.Get("X-Device-Name")

	if deviceID == "" {
		return nil
	}

	return &ProbeResult{
		IP:         ip,
		Port:       port,
		DeviceID:   deviceID,
		DeviceName: deviceName,
	}
}

// getLocalIP 获取本机 IP
func getLocalIP() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到本机 IP")
}
