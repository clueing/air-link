package security

import (
	"sync"
	"time"
)

// AttemptRecord 尝试记录
type AttemptRecord struct {
	DeviceID    string
	Attempts    int
	LastAttempt time.Time
	LockedUntil time.Time
}

// RateLimiter 速率限制器
type RateLimiter struct {
	mu              sync.RWMutex
	records         map[string]*AttemptRecord
	maxAttempts     int
	windowDuration  time.Duration
	lockoutDuration time.Duration
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxAttempts int, windowDuration, lockoutDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records:         make(map[string]*AttemptRecord),
		maxAttempts:     maxAttempts,
		windowDuration:  windowDuration,
		lockoutDuration: lockoutDuration,
	}

	// 启动清理协程
	go rl.cleanup()

	return rl
}

// CheckAllowed 检查是否允许尝试
func (rl *RateLimiter) CheckAllowed(deviceID string) (allowed bool, lockedUntil time.Time) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.records[deviceID]
	if !exists {
		return true, time.Time{}
	}

	now := time.Now()

	// 检查是否在锁定期
	if !record.LockedUntil.IsZero() && now.Before(record.LockedUntil) {
		return false, record.LockedUntil
	}

	// 检查是否超出时间窗口
	if now.Sub(record.LastAttempt) > rl.windowDuration {
		return true, time.Time{}
	}

	// 检查是否超过最大尝试次数
	if record.Attempts >= rl.maxAttempts {
		return false, record.LockedUntil
	}

	return true, time.Time{}
}

// RecordAttempt 记录尝试
func (rl *RateLimiter) RecordAttempt(deviceID string, success bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.records[deviceID]

	if !exists {
		record = &AttemptRecord{
			DeviceID: deviceID,
		}
		rl.records[deviceID] = record
	}

	// 成功则重置
	if success {
		record.Attempts = 0
		record.LockedUntil = time.Time{}
		record.LastAttempt = now
		return
	}

	// 检查是否超出时间窗口，是则重置计数
	if now.Sub(record.LastAttempt) > rl.windowDuration {
		record.Attempts = 1
		record.LastAttempt = now
		return
	}

	// 累加失败次数
	record.Attempts++
	record.LastAttempt = now

	// 达到阈值则锁定
	if record.Attempts >= rl.maxAttempts {
		record.LockedUntil = now.Add(rl.lockoutDuration)
	}
}

// Reset 重置设备的记录
func (rl *RateLimiter) Reset(deviceID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.records, deviceID)
}

// cleanup 定期清理过期记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for deviceID, record := range rl.records {
			// 清理超过锁定时间且超过窗口时间的记录
			if !record.LockedUntil.IsZero() && now.After(record.LockedUntil) {
				if now.Sub(record.LastAttempt) > rl.windowDuration {
					delete(rl.records, deviceID)
				}
			} else if now.Sub(record.LastAttempt) > rl.windowDuration*2 {
				delete(rl.records, deviceID)
			}
		}
		rl.mu.Unlock()
	}
}
