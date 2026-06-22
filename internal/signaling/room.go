package signaling

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// Device 设备信息
type Device struct {
	ID         string    `json:"device_id"`
	Name       string    `json:"device_name"`
	ConnTime   time.Time `json:"-"`
	LastPing   time.Time `json:"-"`
	Connection interface{} `json:"-"` // WebSocket 连接
}

// Room 房间
type Room struct {
	PIN       string
	CreatedAt time.Time
	ExpiresAt time.Time
	Devices   map[string]*Device
	mu        sync.RWMutex
}

// NewRoom 创建房间
func NewRoom(pin string, expiryDuration time.Duration) *Room {
	now := time.Now()
	return &Room{
		PIN:       pin,
		CreatedAt: now,
		ExpiresAt: now.Add(expiryDuration),
		Devices:   make(map[string]*Device),
	}
}

// AddDevice 添加设备
func (r *Room) AddDevice(device *Device) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查设备是否已存在
	if _, exists := r.Devices[device.ID]; exists {
		return false
	}

	r.Devices[device.ID] = device
	return true
}

// RemoveDevice 移除设备
func (r *Room) RemoveDevice(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.Devices, deviceID)
}

// GetDevice 获取设备
func (r *Room) GetDevice(deviceID string) (*Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, exists := r.Devices[deviceID]
	return device, exists
}

// GetDevices 获取所有设备
func (r *Room) GetDevices() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]*Device, 0, len(r.Devices))
	for _, device := range r.Devices {
		devices = append(devices, device)
	}
	return devices
}

// DeviceCount 获取设备数量
func (r *Room) DeviceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Devices)
}

// IsExpired 检查是否过期
func (r *Room) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsEmpty 检查是否为空
func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Devices) == 0
}

// RoomManager 房间管理器
type RoomManager struct {
	rooms          map[string]*Room
	deviceToRoom   map[string]string // device_id -> pin
	mu             sync.RWMutex
	maxDevices     int
	expiryDuration time.Duration
}

// NewRoomManager 创建房间管理器
func NewRoomManager(maxDevices int, expiryDuration time.Duration) *RoomManager {
	rm := &RoomManager{
		rooms:          make(map[string]*Room),
		deviceToRoom:   make(map[string]string),
		maxDevices:     maxDevices,
		expiryDuration: expiryDuration,
	}

	// 启动清理协程
	go rm.cleanup()

	return rm
}

// CreateRoom 创建房间
func (rm *RoomManager) CreateRoom() (*Room, error) {
	pin := generatePIN()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 确保 PIN 唯一
	for i := 0; i < 10; i++ {
		if _, exists := rm.rooms[pin]; !exists {
			room := NewRoom(pin, rm.expiryDuration)
			rm.rooms[pin] = room
			return room, nil
		}
		pin = generatePIN()
	}

	return nil, fmt.Errorf("无法生成唯一 PIN")
}

// GetRoom 获取房间
func (rm *RoomManager) GetRoom(pin string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[pin]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if room.IsExpired() {
		return nil, false
	}

	return room, true
}

// JoinRoom 加入房间
func (rm *RoomManager) JoinRoom(pin string, device *Device) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[pin]
	if !exists {
		return fmt.Errorf("房间不存在")
	}

	if room.IsExpired() {
		return fmt.Errorf("房间已过期")
	}

	if room.DeviceCount() >= rm.maxDevices {
		return fmt.Errorf("房间已满")
	}

	if !room.AddDevice(device) {
		return fmt.Errorf("设备已在房间中")
	}

	rm.deviceToRoom[device.ID] = pin
	return nil
}

// LeaveRoom 离开房间
func (rm *RoomManager) LeaveRoom(deviceID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	pin, exists := rm.deviceToRoom[deviceID]
	if !exists {
		return
	}

	delete(rm.deviceToRoom, deviceID)

	room, exists := rm.rooms[pin]
	if !exists {
		return
	}

	room.RemoveDevice(deviceID)

	// 如果房间为空，删除房间
	if room.IsEmpty() {
		delete(rm.rooms, pin)
	}
}

// GetDeviceRoom 获取设备所在房间
func (rm *RoomManager) GetDeviceRoom(deviceID string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	pin, exists := rm.deviceToRoom[deviceID]
	if !exists {
		return nil, false
	}

	room, exists := rm.rooms[pin]
	return room, exists
}

// cleanup 定期清理过期房间
func (rm *RoomManager) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rm.mu.Lock()
		for pin, room := range rm.rooms {
			if room.IsExpired() && room.IsEmpty() {
				delete(rm.rooms, pin)
			}
		}
		rm.mu.Unlock()
	}
}

// generatePIN 生成 6 位 PIN
func generatePIN() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}
