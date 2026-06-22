// AirLink 应用集成层
class AirLinkApp {
  constructor() {
    this.signaling = null;
    this.webrtc = new WebRTCManager();
    this.fileTransfer = null; // 文件传输管理器
    this.currentRoom = null;
    this.devices = new Map(); // deviceId -> device info
    this.isHost = false;

    // 回调
    this.onRoomCreatedCallback = null;
    this.onRoomJoinedCallback = null;
    this.onDeviceJoinedCallback = null;
    this.onDeviceLeftCallback = null;
    this.onMessageReceivedCallback = null;
    this.onErrorCallback = null;
  }

  // 初始化
  async init() {
    // 获取配置
    const config = await this.fetchConfig();

    // 设置 ICE 服务器
    this.webrtc.setIceServers(config.stun_servers, config.turn_servers || []);

    // 连接信令服务器
    const wsUrl = `ws://${window.location.host}/ws`;
    this.signaling = new SignalingClient(wsUrl);

    // 注册信令消息处理器
    this.setupSignalingHandlers();

    // 注册 WebRTC 回调
    this.setupWebRTCHandlers();

    // 连接
    await this.signaling.connect();

    // 启动心跳
    this.startHeartbeat();

    console.log('AirLink 初始化完成');
    return this.signaling.getDeviceInfo();
  }

  // 获取配置
  async fetchConfig() {
    try {
      const response = await fetch('/api/config');
      return await response.json();
    } catch (error) {
      console.error('获取配置失败:', error);
      // 使用默认配置
      return {
        stun_servers: ['stun:stun.l.google.com:19302'],
        turn_servers: []
      };
    }
  }

  // 设置信令消息处理器
  setupSignalingHandlers() {
    // 房间创建成功
    this.signaling.on('room_created', (msg) => {
      this.currentRoom = {
        pin: msg.data.pin,
        expiresAt: msg.data.expires_at
      };
      this.isHost = true;
      this.devices.set(this.signaling.deviceId, {
        id: this.signaling.deviceId,
        name: this.signaling.deviceName,
        isYou: true
      });

      if (this.onRoomCreatedCallback) {
        this.onRoomCreatedCallback(this.currentRoom);
      }
    });

    // 加入房间成功
    this.signaling.on('room_joined', (msg) => {
      this.currentRoom = {
        pin: msg.data.pin
      };
      this.isHost = false;

      // 添加自己
      this.devices.set(this.signaling.deviceId, {
        id: this.signaling.deviceId,
        name: this.signaling.deviceName,
        isYou: true
      });

      // 添加现有设备
      if (msg.data.devices) {
        msg.data.devices.forEach(device => {
          this.devices.set(device.device_id, {
            id: device.device_id,
            name: device.device_name,
            isYou: false
          });
        });
      }

      // 向所有现有设备发起连接
      this.connectToAllDevices();

      if (this.onRoomJoinedCallback) {
        this.onRoomJoinedCallback(this.currentRoom, Array.from(this.devices.values()));
      }
    });

    // 设备加入/离开
    this.signaling.on('device_update', (msg) => {
      const action = msg.data.action;
      const device = msg.data.device;

      if (action === 'join') {
        this.devices.set(device.device_id, {
          id: device.device_id,
          name: device.device_name,
          isYou: false
        });

        // 发起 WebRTC 连接
        this.connectToDevice(device.device_id);

        if (this.onDeviceJoinedCallback) {
          this.onDeviceJoinedCallback(device);
        }
      } else if (action === 'leave') {
        this.devices.delete(device.device_id);
        this.webrtc.closeConnection(device.device_id);

        if (this.onDeviceLeftCallback) {
          this.onDeviceLeftCallback(device);
        }
      }
    });

    // WebRTC 信令
    this.signaling.on('webrtc_signal', async (msg) => {
      const fromDeviceId = msg.device_id;
      const signal = msg.signal;

      try {
        if (signal.type === 'offer') {
          // 收到 offer
          let pc = this.webrtc.connections.get(fromDeviceId);

          // Offer 冲突解决：使用字典序比较 device ID
          if (pc && pc.signalingState === 'have-local-offer') {
            console.log('检测到 offer 冲突，使用字典序解决');
            const isImpolite = this.signaling.deviceId > fromDeviceId;

            if (isImpolite) {
              // 我是 impolite 方，忽略对方的 offer
              console.log('我是 impolite 方，忽略收到的 offer');
              return;
            } else {
              // 我是 polite 方，回滚自己的 offer，接受对方的 offer
              console.log('我是 polite 方，回滚本地 offer');
              await pc.setLocalDescription({ type: 'rollback' });
            }
          }

          // 创建或获取连接
          pc = await this.webrtc.createConnection(fromDeviceId, false);

          // 设置远程描述
          await pc.setRemoteDescription(new RTCSessionDescription(signal));

          // 创建并发送 answer
          const answer = await pc.createAnswer();
          await pc.setLocalDescription(answer);

          this.signaling.sendSignal(fromDeviceId, {
            type: 'answer',
            sdp: answer.sdp
          });

        } else if (signal.type === 'answer') {
          // 收到 answer
          const pc = this.webrtc.connections.get(fromDeviceId);

          if (!pc) {
            console.warn('收到 answer 但连接不存在');
            return;
          }

          // 只有在 have-local-offer 状态下才能设置 remote answer
          if (pc.signalingState === 'have-local-offer') {
            await pc.setRemoteDescription(new RTCSessionDescription(signal));
          } else {
            console.warn(`收到 answer 但状态不对: ${pc.signalingState}，忽略`);
          }

        } else if (signal.type === 'ice') {
          // 收到 ICE 候选
          await this.webrtc.addIceCandidate(fromDeviceId, signal.candidate);
        }
      } catch (error) {
        console.error('处理 WebRTC 信令失败:', error);
      }
    });

    // 错误
    this.signaling.on('error', (msg) => {
      console.error('信令错误:', msg.data);
      if (this.onErrorCallback) {
        this.onErrorCallback(msg.data);
      }
    });
  }

  // 设置 WebRTC 回调
  setupWebRTCHandlers() {
    // ICE 候选
    this.webrtc.onIceCandidateHandler((deviceId, candidate) => {
      this.signaling.sendSignal(deviceId, {
        type: 'ice',
        candidate: candidate
      });
    });

    // 连接状态变化
    this.webrtc.onConnectionState((deviceId, state) => {
      console.log(`设备 ${deviceId} 连接状态: ${state}`);

      if (state === 'connected') {
        console.log(`与 ${deviceId} 建立 P2P 连接`);
      }
    });

    // 文本消息
    this.webrtc.onMessage((deviceId, message) => {
      console.log('收到消息:', message);
      if (this.onMessageReceivedCallback) {
        this.onMessageReceivedCallback(deviceId, message);
      }
    });
  }

  // 设置文件传输管理器
  setFileTransferManager(fileTransfer) {
    this.fileTransfer = fileTransfer;
  }

  // 连接到所有设备
  async connectToAllDevices() {
    for (const [deviceId, device] of this.devices.entries()) {
      if (!device.isYou) {
        await this.connectToDevice(deviceId);
      }
    }
  }

  // 连接到指定设备
  async connectToDevice(deviceId) {
    try {
      // 创建连接
      await this.webrtc.createConnection(deviceId, true);

      // 创建 offer
      const offer = await this.webrtc.createOffer(deviceId);

      // 发送 offer
      this.signaling.sendSignal(deviceId, offer);

      console.log(`已向 ${deviceId} 发送 offer`);
    } catch (error) {
      console.error(`连接到 ${deviceId} 失败:`, error);
    }
  }

  // 创建房间
  async createRoom() {
    return this.signaling.createRoom();
  }

  // 加入房间
  async joinRoom(pin) {
    return this.signaling.joinRoom(pin);
  }

  // 离开房间
  leaveRoom() {
    this.signaling.leaveRoom();
    this.webrtc.closeAll();
    this.currentRoom = null;
    this.devices.clear();
    this.isHost = false;
  }

  // 发送文本消息
  sendTextMessage(text) {
    const message = {
      type: 'text',
      sender_id: this.signaling.deviceId,
      sender_name: this.signaling.deviceName,
      content: text,
      timestamp: Date.now()
    };

    // 广播到所有设备
    return this.webrtc.broadcastMessage(message);
  }

  // 更新设备名称
  updateDeviceName(newName) {
    this.signaling.updateDeviceName(newName);
  }

  // 启动心跳
  startHeartbeat() {
    setInterval(() => {
      if (this.signaling.connected) {
        this.signaling.ping();
      }
    }, 30000); // 30 秒
  }

  // 获取设备列表
  getDevices() {
    return Array.from(this.devices.values());
  }

  // 获取当前房间信息
  getRoomInfo() {
    return this.currentRoom;
  }

  // 获取连接统计（调试用）
  async getConnectionStats(deviceId) {
    return await this.webrtc.getStats(deviceId);
  }

  // 设置回调
  onRoomCreated(callback) {
    this.onRoomCreatedCallback = callback;
  }

  onRoomJoined(callback) {
    this.onRoomJoinedCallback = callback;
  }

  onDeviceJoined(callback) {
    this.onDeviceJoinedCallback = callback;
  }

  onDeviceLeft(callback) {
    this.onDeviceLeftCallback = callback;
  }

  onMessageReceived(callback) {
    this.onMessageReceivedCallback = callback;
  }

  onError(callback) {
    this.onErrorCallback = callback;
  }
}

// 导出
window.AirLinkApp = AirLinkApp;
