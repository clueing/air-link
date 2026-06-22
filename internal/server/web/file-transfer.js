// 文件传输管理器
class FileTransferManager {
  constructor(webrtcManager) {
    this.webrtc = webrtcManager;
    this.sendQueue = []; // 待发送队列
    this.receiveQueue = []; // 接收中的文件
    this.activeSends = new Map(); // fileId -> SendState
    this.activeReceives = new Map(); // fileId -> ReceiveState
    this.maxConcurrentSends = 2; // 最多同时发送 2 个文件
    this.maxConcurrentReceives = 3; // 最多同时接收 3 个文件
    this.chunkSize = 16 * 1024; // 16KB
    this.maxRetries = 3; // 最大重试次数

    // 回调
    this.onProgressCallback = null;
    this.onCompleteCallback = null;
    this.onErrorCallback = null;

    // 注册 WebRTC 文件消息处理
    this.webrtc.onFile((deviceId, data) => {
      this.handleFileData(deviceId, data);
    });
  }

  // 发送文件
  async sendFile(file, toDeviceId, senderName) {
    const fileId = this.generateFileId();
    const totalChunks = Math.ceil(file.size / this.chunkSize);

    const sendState = {
      fileId,
      file,
      toDeviceId,
      senderName, // 保存发送者名称
      totalChunks,
      sentChunks: new Set(),
      ackedChunks: new Set(),
      currentChunk: 0,
      status: 'queued',
      retries: 0,
      startTime: null,
      progress: 0
    };

    // 添加到队列
    this.sendQueue.push(sendState);

    // 触发队列处理
    this.processSendQueue();

    return fileId;
  }

  // 处理发送队列
  async processSendQueue() {
    // 检查是否可以启动新的发送
    if (this.activeSends.size >= this.maxConcurrentSends) {
      return;
    }

    // 取出队列中的第一个
    const sendState = this.sendQueue.shift();
    if (!sendState) {
      return;
    }

    // 标记为活动状态
    this.activeSends.set(sendState.fileId, sendState);
    sendState.status = 'sending';
    sendState.startTime = Date.now();

    try {
      // 发送文件元数据
      await this.sendFileMetadata(sendState);

      // 开始发送分片
      await this.sendChunks(sendState);
    } catch (error) {
      console.error('文件发送失败:', error);

      // 触发错误回调
      if (this.onErrorCallback) {
        this.onErrorCallback(sendState.fileId, error);
      }

      // 从活动列表移除
      this.activeSends.delete(sendState.fileId);

      // 继续处理队列
      this.processSendQueue();
    }
  }

  // 发送文件元数据
  async sendFileMetadata(sendState) {
    const metadata = {
      type: 'file_meta',
      file_id: sendState.fileId,
      name: sendState.file.name,
      size: sendState.file.size,
      mime_type: sendState.file.type,
      total_chunks: sendState.totalChunks,
      sender_name: sendState.senderName || '未知' // 添加发送者名称
    };

    // 等待 DataChannel 就绪
    await this.waitForDataChannel(sendState.toDeviceId);

    this.webrtc.sendMessage(sendState.toDeviceId, metadata);
  }

  // 等待 DataChannel 就绪
  async waitForDataChannel(deviceId, timeout = 10000) {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      const channel = this.webrtc.dataChannels.get(deviceId);
      if (channel && channel.readyState === 'open') {
        return true;
      }
      // 等待 100ms 后重试
      await new Promise(resolve => setTimeout(resolve, 100));
    }

    throw new Error(`DataChannel 超时未就绪: ${deviceId}`);
  }

  // 发送分片
  async sendChunks(sendState) {
    const { file, fileId, toDeviceId, totalChunks } = sendState;

    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
      // 检查是否已发送
      if (sendState.sentChunks.has(chunkIndex)) {
        continue;
      }

      // 读取分片
      const start = chunkIndex * this.chunkSize;
      const end = Math.min(start + this.chunkSize, file.size);
      const chunk = file.slice(start, end);
      const arrayBuffer = await chunk.arrayBuffer();

      // 构建二进制包：[4字节 fileId hash][4字节 chunkIndex][数据]
      const header = new ArrayBuffer(8);
      const headerView = new DataView(header);
      const fileIdHash = this.hashString(fileId);
      headerView.setUint32(0, fileIdHash, false);
      headerView.setUint32(4, chunkIndex, false);

      // 合并 header 和 chunk data
      const packet = this.concatArrayBuffers(header, arrayBuffer);

      // 发送
      let sent = false;
      for (let retry = 0; retry < this.maxRetries; retry++) {
        if (this.webrtc.sendBinary(toDeviceId, packet)) {
          sent = true;
          sendState.sentChunks.add(chunkIndex);
          break;
        }
        // 等待一下再重试
        await this.sleep(100);
      }

      if (!sent) {
        // 发送失败
        sendState.status = 'failed';
        this.activeSends.delete(fileId);
        if (this.onErrorCallback) {
          this.onErrorCallback(fileId, 'send_failed');
        }
        return;
      }

      // 更新进度
      sendState.progress = (sendState.sentChunks.size / totalChunks) * 100;
      if (this.onProgressCallback) {
        this.onProgressCallback(fileId, sendState.progress, 'send');
      }

      // 流控：检查缓冲区
      await this.waitForBuffer(toDeviceId);
    }

    // 所有分片发送完成
    sendState.status = 'completed';
    sendState.progress = 100;
    this.activeSends.delete(fileId);

    if (this.onCompleteCallback) {
      this.onCompleteCallback(fileId, 'send');
    }

    // 处理队列中的下一个
    this.processSendQueue();
  }

  // 处理接收到的文件数据
  handleFileData(deviceId, data) {
    // 解析 header
    const headerView = new DataView(data, 0, 8);
    const fileIdHash = headerView.getUint32(0, false);
    const chunkIndex = headerView.getUint32(4, false);

    // 查找对应的接收状态
    let receiveState = null;
    for (const [fileId, state] of this.activeReceives.entries()) {
      if (this.hashString(fileId) === fileIdHash) {
        receiveState = state;
        break;
      }
    }

    if (!receiveState) {
      console.warn(`收到未知文件的分片: hash=${fileIdHash}, chunk=${chunkIndex}`);
      return;
    }

    // 提取数据
    const chunkData = data.slice(8);
    receiveState.chunks.set(chunkIndex, chunkData);
    receiveState.receivedChunks.add(chunkIndex);

    // 更新进度
    receiveState.progress = (receiveState.receivedChunks.size / receiveState.totalChunks) * 100;
    if (this.onProgressCallback) {
      this.onProgressCallback(receiveState.fileId, receiveState.progress, 'receive');
    }

    // 检查是否接收完成
    if (receiveState.receivedChunks.size === receiveState.totalChunks) {
      this.completeReceive(receiveState);
    }
  }

  // 完成接收
  completeReceive(receiveState) {
    // 合并所有分片
    const chunks = [];
    for (let i = 0; i < receiveState.totalChunks; i++) {
      const chunk = receiveState.chunks.get(i);
      if (!chunk) {
        console.error(`缺少分片 ${i}`);
        return;
      }
      chunks.push(chunk);
    }

    const blob = new Blob(chunks, { type: receiveState.mimeType });
    receiveState.blob = blob;
    receiveState.status = 'completed';
    receiveState.progress = 100;

    this.activeReceives.delete(receiveState.fileId);

    if (this.onCompleteCallback) {
      this.onCompleteCallback(receiveState.fileId, 'receive', {
        name: receiveState.name,
        size: receiveState.size,
        blob: blob
      });
    }
  }

  // 处理文件元数据（从 WebRTC 消息）
  handleFileMetadata(deviceId, metadata) {
    const receiveState = {
      fileId: metadata.file_id,
      name: metadata.name,
      size: metadata.size,
      mimeType: metadata.mime_type,
      totalChunks: metadata.total_chunks,
      senderName: metadata.sender_name || '未知', // 保存发送者名称
      chunks: new Map(),
      receivedChunks: new Set(),
      status: 'receiving',
      progress: 0,
      fromDeviceId: deviceId
    };

    this.activeReceives.set(receiveState.fileId, receiveState);

    // 通知开始接收
    if (this.onProgressCallback) {
      this.onProgressCallback(receiveState.fileId, 0, 'receive', {
        name: receiveState.name,
        size: receiveState.size,
        senderName: receiveState.senderName // 传递发送者名称
      });
    }
  }

  // 取消发送
  cancelSend(fileId) {
    const sendState = this.activeSends.get(fileId);
    if (sendState) {
      sendState.status = 'cancelled';
      this.activeSends.delete(fileId);
      this.processSendQueue();
    }

    // 从队列中移除
    this.sendQueue = this.sendQueue.filter(s => s.fileId !== fileId);
  }

  // 等待缓冲区
  async waitForBuffer(deviceId) {
    const channel = this.webrtc.dataChannels.get(deviceId);
    if (!channel) return;

    const maxBuffer = 8 * 1024 * 1024; // 8MB
    while (channel.bufferedAmount > maxBuffer) {
      await this.sleep(50);
    }
  }

  // 生成文件 ID
  generateFileId() {
    return 'file_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
  }

  // 字符串哈希（简单实现）
  hashString(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32bit integer
    }
    return Math.abs(hash);
  }

  // 合并 ArrayBuffer
  concatArrayBuffers(buffer1, buffer2) {
    const tmp = new Uint8Array(buffer1.byteLength + buffer2.byteLength);
    tmp.set(new Uint8Array(buffer1), 0);
    tmp.set(new Uint8Array(buffer2), buffer1.byteLength);
    return tmp.buffer;
  }

  // 睡眠
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // 获取发送状态
  getSendState(fileId) {
    return this.activeSends.get(fileId);
  }

  // 获取接收状态
  getReceiveState(fileId) {
    return this.activeReceives.get(fileId);
  }

  // 获取队列信息
  getQueueInfo() {
    return {
      sendQueue: this.sendQueue.length,
      activeSends: this.activeSends.size,
      activeReceives: this.activeReceives.size
    };
  }

  // 设置回调
  onProgress(callback) {
    this.onProgressCallback = callback;
  }

  onComplete(callback) {
    this.onCompleteCallback = callback;
  }

  onError(callback) {
    this.onErrorCallback = callback;
  }
}

// 导出
window.FileTransferManager = FileTransferManager;
