// WebSocket 信令客户端
class SignalingClient {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.deviceId = this.getOrCreateDeviceId();
    this.deviceName = this.getOrCreateDeviceName();
    this.messageHandlers = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 3;
    this.reconnectDelay = 1000;
    this.connected = false;
  }

  // 连接 WebSocket
  connect() {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket 已连接');
          this.connected = true;
          this.reconnectAttempts = 0;

          // 发送 hello 消息
          this.sendMessage({
            type: 'hello',
            data: {
              device_id: this.deviceId,
              device_name: this.deviceName
            }
          });

          resolve();
        };

        this.ws.onclose = () => {
          console.log('WebSocket 已断开');
          this.connected = false;
          this.handleDisconnect();
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket 错误:', error);
          reject(error);
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

      } catch (error) {
        reject(error);
      }
    });
  }

  // 处理断开连接
  handleDisconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      console.log(`${delay}ms 后重连 (尝试 ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

      setTimeout(() => {
        this.connect().catch(err => {
          console.error('重连失败:', err);
        });
      }, delay);
    } else {
      console.error('重连失败次数过多，已放弃');
      this.triggerHandler('disconnect', { reason: 'max_attempts' });
    }
  }

  // 发送消息
  sendMessage(message) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.error('WebSocket 未连接');
      return false;
    }

    try {
      this.ws.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('发送消息失败:', error);
      return false;
    }
  }

  // 处理收到的消息
  handleMessage(data) {
    try {
      const message = JSON.parse(data);
      console.log('收到消息:', message.type);

      this.triggerHandler(message.type, message);
    } catch (error) {
      console.error('解析消息失败:', error);
    }
  }

  // 注册消息处理器
  on(type, handler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, []);
    }
    this.messageHandlers.get(type).push(handler);
  }

  // 触发消息处理器
  triggerHandler(type, message) {
    const handlers = this.messageHandlers.get(type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(message);
        } catch (error) {
          console.error(`处理消息 ${type} 失败:`, error);
        }
      });
    }
  }

  // 创建房间
  createRoom() {
    return this.sendMessage({
      type: 'create_room',
      data: {
        device_id: this.deviceId,
        device_name: this.deviceName
      }
    });
  }

  // 加入房间
  joinRoom(pin) {
    return this.sendMessage({
      type: 'join_room',
      data: {
        pin: pin,
        device_id: this.deviceId,
        device_name: this.deviceName
      }
    });
  }

  // 发送 WebRTC 信令
  sendSignal(toDeviceId, signal) {
    return this.sendMessage({
      type: 'webrtc_signal',
      to_device_id: toDeviceId,
      signal: signal
    });
  }

  // 离开房间
  leaveRoom() {
    return this.sendMessage({
      type: 'leave_room'
    });
  }

  // 发送心跳
  ping() {
    return this.sendMessage({
      type: 'ping'
    });
  }

  // 关闭连接
  close() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
      this.connected = false;
    }
  }

  // 获取或创建设备 ID
  getOrCreateDeviceId() {
    const key = 'airlink-device-id';
    let deviceId = localStorage.getItem(key);
    if (!deviceId) {
      deviceId = this.generateUUID();
      localStorage.setItem(key, deviceId);
    }
    return deviceId;
  }

  // 获取或创建设备名称
  getOrCreateDeviceName() {
    const key = 'airlink-device-name';
    let deviceName = localStorage.getItem(key);
    if (!deviceName) {
      deviceName = this.generateDefaultName();
      localStorage.setItem(key, deviceName);
    }
    return deviceName;
  }

  // 生成默认设备名称
  generateDefaultName() {
    const ua = navigator.userAgent || '';
    const platform = navigator.platform || '';
    let baseName = '我的设备';

    if (/android/i.test(ua)) {
      baseName = 'Android 设备';
    } else if (/iPad|iPhone|iPod/.test(platform) || /iPad|iPhone|iPod/.test(ua)) {
      baseName = 'iOS 设备';
    } else if (/Win/.test(platform)) {
      baseName = 'Windows PC';
    } else if (/Mac/.test(platform)) {
      const names = ['MacBook', 'iMac', 'Mac mini'];
      baseName = names[Math.floor(Math.random() * names.length)];
    } else if (/Linux/.test(platform)) {
      baseName = 'Linux 工作站';
    }

    // 添加随机后缀
    const suffix = this.generateRandomSuffix();
    return `${baseName} #${suffix}`;
  }

  // 生成随机后缀
  generateRandomSuffix() {
    const chars = '0123456789ABCDEF';
    let result = '';
    for (let i = 0; i < 4; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  // 生成 UUID
  generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }

  // 更新设备名称
  updateDeviceName(newName) {
    this.deviceName = newName;
    localStorage.setItem('airlink-device-name', newName);
  }

  // 获取设备信息
  getDeviceInfo() {
    return {
      id: this.deviceId,
      name: this.deviceName
    };
  }
}

// 导出
window.SignalingClient = SignalingClient;
