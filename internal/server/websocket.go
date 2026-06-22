package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clue/air-link/internal/security"
	"github.com/clue/air-link/internal/signaling"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（局域网使用）
	},
}

// Message WebSocket 消息
type Message struct {
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data,omitempty"`
	DeviceID string                 `json:"device_id,omitempty"`
	ToDevice string                 `json:"to_device_id,omitempty"`
	Signal   interface{}            `json:"signal,omitempty"`
}

// WSHandler WebSocket 处理器
type WSHandler struct {
	roomManager  *signaling.RoomManager
	rateLimiter  *security.RateLimiter
	logger       *logrus.Logger
	stunServers  []string
}

// NewWSHandler 创建 WebSocket 处理器
func NewWSHandler(
	roomManager *signaling.RoomManager,
	rateLimiter *security.RateLimiter,
	stunServers []string,
	logger *logrus.Logger,
) *WSHandler {
	return &WSHandler{
		roomManager:  roomManager,
		rateLimiter:  rateLimiter,
		logger:       logger,
		stunServers:  stunServers,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	var currentDevice *signaling.Device
	deviceID := ""

	// 心跳检测
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 消息处理
	for {
		select {
		case <-ticker.C:
			// 发送 ping
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.logger.Debugf("发送 ping 失败: %v", err)
				goto cleanup
			}

		default:
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Errorf("读取消息失败: %v", err)
				}
				goto cleanup
			}

			h.logger.Debugf("收到消息: %s from %s", msg.Type, msg.DeviceID)

			switch msg.Type {
			case "hello":
				// 连接握手
				deviceID = getStringField(msg.Data, "device_id")
				deviceName := getStringField(msg.Data, "device_name")

				currentDevice = &signaling.Device{
					ID:         deviceID,
					Name:       deviceName,
					ConnTime:   time.Now(),
					LastPing:   time.Now(),
					Connection: conn,
				}

				h.sendMessage(conn, Message{
					Type: "hello_ack",
					Data: map[string]interface{}{
						"server_version": "1.0.0",
						"stun_servers":   h.stunServers,
					},
				})

			case "create_room":
				// 创建房间
				if currentDevice == nil {
					h.sendError(conn, "NOT_AUTHENTICATED", "请先发送 hello 消息")
					continue
				}

				room, err := h.roomManager.CreateRoom()
				if err != nil {
					h.sendError(conn, "CREATE_FAILED", err.Error())
					continue
				}

				if err := h.roomManager.JoinRoom(room.PIN, currentDevice); err != nil {
					h.sendError(conn, "JOIN_FAILED", err.Error())
					continue
				}

				h.sendMessage(conn, Message{
					Type: "room_created",
					Data: map[string]interface{}{
						"pin":        room.PIN,
						"expires_at": room.ExpiresAt.Unix(),
					},
				})

				h.logger.Infof("设备 %s 创建房间 %s", currentDevice.Name, room.PIN)

			case "join_room":
				// 加入房间
				if currentDevice == nil {
					h.sendError(conn, "NOT_AUTHENTICATED", "请先发送 hello 消息")
					continue
				}

				pin := getStringField(msg.Data, "pin")

				// 速率限制检查
				allowed, lockedUntil := h.rateLimiter.CheckAllowed(deviceID)
				if !allowed {
					h.sendError(conn, "RATE_LIMITED", fmt.Sprintf("尝试次数过多，请等待至 %s", lockedUntil.Format("15:04:05")))
					continue
				}

				// 获取房间
				room, exists := h.roomManager.GetRoom(pin)
				if !exists {
					h.rateLimiter.RecordAttempt(deviceID, false)
					h.sendError(conn, "INVALID_PIN", "PIN 无效或已过期")
					continue
				}

				// 加入房间
				if err := h.roomManager.JoinRoom(pin, currentDevice); err != nil {
					h.rateLimiter.RecordAttempt(deviceID, false)
					h.sendError(conn, "JOIN_FAILED", err.Error())
					continue
				}

				h.rateLimiter.RecordAttempt(deviceID, true)

				// 获取房间内其他设备
				devices := room.GetDevices()
				deviceList := make([]map[string]interface{}, 0)
				for _, d := range devices {
					if d.ID != currentDevice.ID {
						deviceList = append(deviceList, map[string]interface{}{
							"device_id":   d.ID,
							"device_name": d.Name,
						})
					}
				}

				// 发送加入成功消息
				h.sendMessage(conn, Message{
					Type: "room_joined",
					Data: map[string]interface{}{
						"pin":     pin,
						"devices": deviceList,
					},
				})

				// 通知房间内其他设备
				h.broadcastToRoom(room, currentDevice.ID, Message{
					Type: "device_update",
					Data: map[string]interface{}{
						"action": "join",
						"device": map[string]interface{}{
							"device_id":   currentDevice.ID,
							"device_name": currentDevice.Name,
						},
						"total_devices": room.DeviceCount(),
					},
				})

				h.logger.Infof("设备 %s 加入房间 %s", currentDevice.Name, pin)

			case "webrtc_signal":
				// WebRTC 信令中继
				if currentDevice == nil {
					h.sendError(conn, "NOT_AUTHENTICATED", "请先发送 hello 消息")
					continue
				}

				room, exists := h.roomManager.GetDeviceRoom(currentDevice.ID)
				if !exists {
					h.sendError(conn, "NOT_IN_ROOM", "未加入任何房间")
					continue
				}

				toDeviceID := msg.ToDevice
				toDevice, exists := room.GetDevice(toDeviceID)
				if !exists {
					h.sendError(conn, "DEVICE_NOT_FOUND", "目标设备不存在")
					continue
				}

				// 转发信令
				toConn, ok := toDevice.Connection.(*websocket.Conn)
				if !ok {
					h.sendError(conn, "INVALID_CONNECTION", "目标设备连接无效")
					continue
				}

				h.sendMessage(toConn, Message{
					Type:     "webrtc_signal",
					DeviceID: currentDevice.ID,
					Signal:   msg.Signal,
				})

			case "ping":
				// 心跳响应
				if currentDevice != nil {
					currentDevice.LastPing = time.Now()
				}
				h.sendMessage(conn, Message{Type: "pong"})

			case "leave_room":
				// 离开房间
				if currentDevice != nil {
					h.handleDeviceLeave(currentDevice)
				}
				h.sendMessage(conn, Message{Type: "left_room"})

			default:
				h.logger.Warnf("未知消息类型: %s", msg.Type)
			}
		}
	}

cleanup:
	// 清理连接
	if currentDevice != nil {
		h.handleDeviceLeave(currentDevice)
	}
}

// handleDeviceLeave 处理设备离开
func (h *WSHandler) handleDeviceLeave(device *signaling.Device) {
	room, exists := h.roomManager.GetDeviceRoom(device.ID)
	if !exists {
		return
	}

	h.roomManager.LeaveRoom(device.ID)

	// 通知房间内其他设备
	h.broadcastToRoom(room, device.ID, Message{
		Type: "device_update",
		Data: map[string]interface{}{
			"action": "leave",
			"device": map[string]interface{}{
				"device_id":   device.ID,
				"device_name": device.Name,
			},
			"total_devices": room.DeviceCount(),
		},
	})

	h.logger.Infof("设备 %s 离开房间 %s", device.Name, room.PIN)
}

// broadcastToRoom 向房间内所有设备（除了排除的设备）广播消息
func (h *WSHandler) broadcastToRoom(room *signaling.Room, excludeDeviceID string, msg Message) {
	devices := room.GetDevices()
	for _, device := range devices {
		if device.ID == excludeDeviceID {
			continue
		}

		conn, ok := device.Connection.(*websocket.Conn)
		if !ok {
			continue
		}

		h.sendMessage(conn, msg)
	}
}

// sendMessage 发送消息
func (h *WSHandler) sendMessage(conn *websocket.Conn, msg Message) {
	if err := conn.WriteJSON(msg); err != nil {
		h.logger.Errorf("发送消息失败: %v", err)
	}
}

// sendError 发送错误消息
func (h *WSHandler) sendError(conn *websocket.Conn, code, message string) {
	h.sendMessage(conn, Message{
		Type: "error",
		Data: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// getStringField 获取字符串字段
func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
