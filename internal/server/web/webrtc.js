// WebRTC 连接管理器
class WebRTCManager {
  constructor() {
    this.connections = new Map(); // deviceId -> RTCPeerConnection
    this.dataChannels = new Map(); // deviceId -> RTCDataChannel
    this.onMessageCallback = null;
    this.onFileCallback = null;
    this.onConnectionStateCallback = null;
    this.iceServers = [];
  }

  // 设置 ICE 服务器
  setIceServers(stunServers, turnServers = []) {
    this.iceServers = [
      ...stunServers.map(url => ({ urls: url })),
      ...turnServers.map(server => ({
        urls: server.url,
        username: server.username,
        credential: server.credential
      }))
    ];
  }

  // 创建 P2P 连接
  async createConnection(deviceId, isInitiator) {
    // 如果连接已存在且稳定，返回现有连接
    if (this.connections.has(deviceId)) {
      const existingPc = this.connections.get(deviceId);
      const existingState = existingPc.signalingState;

      console.log(`连接已存在且状态为 ${existingState}: ${deviceId}`);
      return existingPc;
    }

    const config = {
      iceServers: this.iceServers,
      iceTransportPolicy: 'all' // 优先直连，自动降级
    };

    const pc = new RTCPeerConnection(config);
    this.connections.set(deviceId, pc);

    // ICE 候选收集
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.onIceCandidate(deviceId, event.candidate);
      }
    };

    // 连接状态变化
    pc.onconnectionstatechange = () => {
      console.log(`连接状态 [${deviceId}]: ${pc.connectionState}`);
      if (this.onConnectionStateCallback) {
        this.onConnectionStateCallback(deviceId, pc.connectionState);
      }

      // 连接失败或断开，尝试重连
      if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
        this.handleConnectionFailure(deviceId);
      }
    };

    // ICE 连接状态
    pc.oniceconnectionstatechange = () => {
      console.log(`ICE 状态 [${deviceId}]: ${pc.iceConnectionState}`);
    };

    // 数据通道
    if (isInitiator) {
      // 发起方创建 DataChannel
      const dc = pc.createDataChannel('airlink', {
        ordered: true
      });
      this.setupDataChannel(deviceId, dc);
    } else {
      // 接收方监听 DataChannel
      pc.ondatachannel = (event) => {
        this.setupDataChannel(deviceId, event.channel);
      };
    }

    return pc;
  }

  // 设置 DataChannel
  setupDataChannel(deviceId, channel) {
    this.dataChannels.set(deviceId, channel);

    channel.onopen = () => {
      console.log(`DataChannel 已打开: ${deviceId}`);
    };

    channel.onclose = () => {
      console.log(`DataChannel 已关闭: ${deviceId}`);
      this.dataChannels.delete(deviceId);
    };

    channel.onerror = (error) => {
      console.error(`DataChannel 错误 [${deviceId}]:`, error);
    };

    channel.onmessage = (event) => {
      this.handleDataChannelMessage(deviceId, event.data);
    };

    // 设置二进制类型
    channel.binaryType = 'arraybuffer';
  }

  // 创建 Offer
  async createOffer(deviceId) {
    const pc = this.connections.get(deviceId);
    if (!pc) {
      throw new Error(`连接不存在: ${deviceId}`);
    }

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);

    return {
      type: 'offer',
      sdp: offer.sdp
    };
  }

  // 创建 Answer
  async createAnswer(deviceId, offer) {
    const pc = this.connections.get(deviceId);
    if (!pc) {
      throw new Error(`连接不存在: ${deviceId}`);
    }

    await pc.setRemoteDescription(new RTCSessionDescription(offer));
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);

    return {
      type: 'answer',
      sdp: answer.sdp
    };
  }

  // 处理 Answer
  async handleAnswer(deviceId, answer) {
    const pc = this.connections.get(deviceId);
    if (!pc) {
      throw new Error(`连接不存在: ${deviceId}`);
    }

    await pc.setRemoteDescription(new RTCSessionDescription(answer));
  }

  // 添加 ICE 候选
  async addIceCandidate(deviceId, candidate) {
    const pc = this.connections.get(deviceId);
    if (!pc) {
      console.warn(`连接不存在，忽略 ICE 候选: ${deviceId}`);
      return;
    }

    // 检查 remote description 是否已设置
    if (!pc.remoteDescription) {
      console.warn(`远程描述未设置，暂缓添加 ICE 候选: ${deviceId}`);
      // ICE 候选会在 remote description 设置后自动重试
      return;
    }

    try {
      await pc.addIceCandidate(new RTCIceCandidate(candidate));
    } catch (error) {
      console.error(`添加 ICE 候选失败 [${deviceId}]:`, error.message);
    }
  }

  // 处理 DataChannel 消息
  handleDataChannelMessage(deviceId, data) {
    // 判断是文本还是二进制
    if (typeof data === 'string') {
      // JSON 消息
      try {
        const msg = JSON.parse(data);
        if (this.onMessageCallback) {
          this.onMessageCallback(deviceId, msg);
        }
      } catch (error) {
        console.error('解析消息失败:', error);
      }
    } else {
      // 二进制数据（文件分片）
      if (this.onFileCallback) {
        this.onFileCallback(deviceId, data);
      }
    }
  }

  // 发送文本消息
  sendMessage(deviceId, message) {
    const channel = this.dataChannels.get(deviceId);
    if (!channel || channel.readyState !== 'open') {
      console.error(`DataChannel 未就绪: ${deviceId}`);
      return false;
    }

    try {
      channel.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error(`发送消息失败 [${deviceId}]:`, error);
      return false;
    }
  }

  // 发送二进制数据
  sendBinary(deviceId, data) {
    const channel = this.dataChannels.get(deviceId);
    if (!channel || channel.readyState !== 'open') {
      console.error(`DataChannel 未就绪: ${deviceId}`);
      return false;
    }

    try {
      // 检查缓冲区
      if (channel.bufferedAmount > 16 * 1024 * 1024) {
        console.warn(`发送缓冲区过大 [${deviceId}]: ${channel.bufferedAmount}`);
        return false;
      }

      channel.send(data);
      return true;
    } catch (error) {
      console.error(`发送二进制数据失败 [${deviceId}]:`, error);
      return false;
    }
  }

  // 广播消息到所有设备
  broadcastMessage(message, excludeDeviceId = null) {
    let successCount = 0;
    for (const [deviceId, channel] of this.dataChannels.entries()) {
      if (deviceId !== excludeDeviceId && channel.readyState === 'open') {
        if (this.sendMessage(deviceId, message)) {
          successCount++;
        }
      }
    }
    return successCount;
  }

  // 处理连接失败
  async handleConnectionFailure(deviceId) {
    const pc = this.connections.get(deviceId);
    if (!pc) return;

    // 简单重连机制（会话内）
    console.log(`尝试重连: ${deviceId}`);
    // 实际重连逻辑由上层应用处理
  }

  // 关闭连接
  closeConnection(deviceId) {
    const channel = this.dataChannels.get(deviceId);
    if (channel) {
      channel.close();
      this.dataChannels.delete(deviceId);
    }

    const pc = this.connections.get(deviceId);
    if (pc) {
      pc.close();
      this.connections.delete(deviceId);
    }

    console.log(`连接已关闭: ${deviceId}`);
  }

  // 关闭所有连接
  closeAll() {
    for (const deviceId of this.connections.keys()) {
      this.closeConnection(deviceId);
    }
  }

  // 获取连接状态
  getConnectionState(deviceId) {
    const pc = this.connections.get(deviceId);
    if (!pc) return 'closed';
    return pc.connectionState;
  }

  // 获取 DataChannel 状态
  getDataChannelState(deviceId) {
    const channel = this.dataChannels.get(deviceId);
    if (!channel) return 'closed';
    return channel.readyState;
  }

  // 获取连接统计信息（用于调试）
  async getStats(deviceId) {
    const pc = this.connections.get(deviceId);
    if (!pc) return null;

    const stats = await pc.getStats();
    const result = {
      connection: pc.connectionState,
      ice: pc.iceConnectionState,
      candidates: [],
      dataChannel: {}
    };

    stats.forEach(report => {
      if (report.type === 'candidate-pair' && report.state === 'succeeded') {
        result.candidates.push({
          local: report.localCandidateId,
          remote: report.remoteCandidateId,
          bytesSent: report.bytesSent,
          bytesReceived: report.bytesReceived,
          currentRtt: report.currentRoundTripTime
        });
      }

      if (report.type === 'data-channel') {
        result.dataChannel = {
          state: report.state,
          messagesSent: report.messagesSent,
          messagesReceived: report.messagesReceived,
          bytesSent: report.bytesSent,
          bytesReceived: report.bytesReceived
        };
      }
    });

    return result;
  }

  // 设置回调
  onMessage(callback) {
    this.onMessageCallback = callback;
  }

  onFile(callback) {
    this.onFileCallback = callback;
  }

  onConnectionState(callback) {
    this.onConnectionStateCallback = callback;
  }

  onIceCandidate(deviceId, candidate) {
    // 由上层应用通过信令服务器发送
    if (this.onIceCandidateCallback) {
      this.onIceCandidateCallback(deviceId, candidate);
    }
  }

  onIceCandidateHandler(callback) {
    this.onIceCandidateCallback = callback;
  }
}

// 导出
window.WebRTCManager = WebRTCManager;
