// 开发者调试面板
class DebugPanel {
  constructor(app) {
    this.app = app;
    this.enabled = false;
    this.panel = null;
    this.updateInterval = null;
  }

  // 启用调试面板
  enable() {
    if (this.enabled) return;
    this.enabled = true;

    // 创建面板
    this.createPanel();

    // 开始更新
    this.startUpdate();

    console.log('调试面板已启用');
  }

  // 禁用调试面板
  disable() {
    if (!this.enabled) return;
    this.enabled = false;

    // 移除面板
    if (this.panel) {
      this.panel.remove();
      this.panel = null;
    }

    // 停止更新
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }

    console.log('调试面板已禁用');
  }

  // 切换调试面板
  toggle() {
    if (this.enabled) {
      this.disable();
    } else {
      this.enable();
    }
  }

  // 创建面板
  createPanel() {
    const panel = document.createElement('div');
    panel.id = 'debug-panel';
    panel.style.cssText = `
      position: fixed;
      bottom: 20px;
      right: 20px;
      width: 400px;
      max-height: 600px;
      background: var(--panel-strong);
      border: 1px solid var(--line);
      border-radius: var(--radius-md);
      box-shadow: var(--shadow-lg);
      z-index: 9999;
      font-family: var(--font-mono);
      font-size: 11px;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    `;

    panel.innerHTML = `
      <div style="padding: 12px; border-bottom: 1px solid var(--line); display: flex; align-items: center; justify-content: space-between; background: var(--bg-subtle);">
        <span style="font-weight: 600; color: var(--text);">🐛 调试面板</span>
        <button onclick="window.debugPanel.disable()" style="background: none; border: none; cursor: pointer; font-size: 16px; color: var(--text-faint);">✕</button>
      </div>
      <div id="debug-content" style="padding: 12px; overflow-y: auto; flex: 1;">
        <div id="debug-info"></div>
      </div>
      <div style="padding: 8px; border-top: 1px solid var(--line); display: flex; gap: 8px; background: var(--bg-subtle);">
        <button onclick="window.debugPanel.refresh()" style="flex: 1; padding: 6px; background: var(--accent-soft); color: var(--accent); border: 1px solid var(--accent-border); border-radius: 6px; cursor: pointer; font-size: 11px; font-family: var(--font);">刷新</button>
        <button onclick="window.debugPanel.copyLogs()" style="flex: 1; padding: 6px; background: var(--accent-soft); color: var(--accent); border: 1px solid var(--accent-border); border-radius: 6px; cursor: pointer; font-size: 11px; font-family: var(--font);">复制日志</button>
      </div>
    `;

    document.body.appendChild(panel);
    this.panel = panel;

    // 使面板可拖动
    this.makeDraggable(panel);
  }

  // 使面板可拖动
  makeDraggable(element) {
    let isDragging = false;
    let currentX;
    let currentY;
    let initialX;
    let initialY;

    const header = element.querySelector('div:first-child');

    header.addEventListener('mousedown', (e) => {
      isDragging = true;
      initialX = e.clientX - element.offsetLeft;
      initialY = e.clientY - element.offsetTop;
      header.style.cursor = 'grabbing';
    });

    document.addEventListener('mousemove', (e) => {
      if (!isDragging) return;

      e.preventDefault();
      currentX = e.clientX - initialX;
      currentY = e.clientY - initialY;

      element.style.left = currentX + 'px';
      element.style.top = currentY + 'px';
      element.style.right = 'auto';
      element.style.bottom = 'auto';
    });

    document.addEventListener('mouseup', () => {
      isDragging = false;
      header.style.cursor = 'grab';
    });

    header.style.cursor = 'grab';
  }

  // 开始更新
  startUpdate() {
    this.updateInfo();
    this.updateInterval = setInterval(() => {
      this.updateInfo();
    }, 1000);
  }

  // 更新信息
  async updateInfo() {
    if (!this.enabled || !this.panel) return;

    const info = await this.gatherInfo();
    const content = document.getElementById('debug-info');
    if (content) {
      content.innerHTML = this.formatInfo(info);
    }
  }

  // 收集信息
  async gatherInfo() {
    const info = {
      timestamp: new Date().toLocaleString('zh-CN'),
      signaling: {},
      webrtc: {},
      devices: [],
      room: {},
      fileTransfer: {}
    };

    // 信令状态
    if (this.app.signaling) {
      info.signaling = {
        connected: this.app.signaling.connected,
        deviceId: this.app.signaling.deviceId,
        deviceName: this.app.signaling.deviceName
      };
    }

    // WebRTC 连接
    if (this.app.webrtc) {
      info.webrtc = {
        connections: this.app.webrtc.connections.size,
        dataChannels: this.app.webrtc.dataChannels.size
      };

      // 每个连接的详细状态
      const connections = [];
      for (const [deviceId, pc] of this.app.webrtc.connections.entries()) {
        const stats = await this.app.webrtc.getStats(deviceId);
        connections.push({
          deviceId: deviceId.substring(0, 8),
          state: pc.connectionState,
          iceState: pc.iceConnectionState,
          stats: stats
        });
      }
      info.webrtc.details = connections;
    }

    // 设备列表
    info.devices = this.app.getDevices().map(d => ({
      name: d.name,
      isYou: d.isYou
    }));

    // 房间信息
    const room = this.app.getRoomInfo();
    if (room) {
      info.room = {
        pin: room.pin,
        isHost: this.app.isHost
      };
    }

    return info;
  }

  // 格式化信息
  formatInfo(info) {
    let html = `<div style="color: var(--text-faint); margin-bottom: 8px;">${info.timestamp}</div>`;

    // 信令状态
    html += this.section('信令状态', `
      <div>连接: ${this.badge(info.signaling.connected ? '已连接' : '断开', info.signaling.connected ? 'success' : 'danger')}</div>
      <div>设备 ID: ${this.code(info.signaling.deviceId?.substring(0, 8) || 'N/A')}</div>
      <div>设备名称: ${this.code(info.signaling.deviceName || 'N/A')}</div>
    `);

    // WebRTC 状态
    html += this.section('WebRTC 状态', `
      <div>活动连接: ${this.badge(info.webrtc.connections || 0)}</div>
      <div>数据通道: ${this.badge(info.webrtc.dataChannels || 0)}</div>
    `);

    // 连接详情
    if (info.webrtc.details && info.webrtc.details.length > 0) {
      html += this.section('连接详情', info.webrtc.details.map(conn => `
        <div style="margin-bottom: 8px; padding: 8px; background: var(--bg-subtle); border-radius: 6px;">
          <div>设备: ${this.code(conn.deviceId)}</div>
          <div>状态: ${this.badge(conn.state, conn.state === 'connected' ? 'success' : 'warning')}</div>
          <div>ICE: ${this.badge(conn.iceState)}</div>
          ${conn.stats ? `
            <div style="margin-top: 4px; font-size: 10px; color: var(--text-faint);">
              ${conn.stats.dataChannel ? `
                <div>发送: ${conn.stats.dataChannel.bytesSent || 0} B</div>
                <div>接收: ${conn.stats.dataChannel.bytesReceived || 0} B</div>
              ` : ''}
            </div>
          ` : ''}
        </div>
      `).join(''));
    }

    // 房间信息
    if (info.room.pin) {
      html += this.section('房间信息', `
        <div>PIN: ${this.code(info.room.pin)}</div>
        <div>角色: ${this.badge(info.room.isHost ? '主持人' : '成员')}</div>
      `);
    }

    // 设备列表
    if (info.devices.length > 0) {
      html += this.section('设备列表', info.devices.map(d => `
        <div>${d.name} ${d.isYou ? this.badge('你', 'accent') : ''}</div>
      `).join(''));
    }

    return html;
  }

  // 工具函数
  section(title, content) {
    return `
      <div style="margin-bottom: 12px;">
        <div style="font-weight: 600; color: var(--text); margin-bottom: 6px; font-size: 12px;">${title}</div>
        <div style="color: var(--text-soft); line-height: 1.6;">${content}</div>
      </div>
    `;
  }

  badge(text, type = 'default') {
    const colors = {
      success: 'var(--success)',
      danger: 'var(--danger)',
      warning: 'var(--warning-text)',
      accent: 'var(--accent)',
      default: 'var(--text-faint)'
    };
    return `<span style="display: inline-block; padding: 2px 6px; background: ${colors[type]}22; color: ${colors[type]}; border-radius: 4px; font-size: 10px;">${text}</span>`;
  }

  code(text) {
    return `<code style="background: var(--bg-subtle); padding: 2px 4px; border-radius: 3px; color: var(--accent);">${text}</code>`;
  }

  // 刷新
  refresh() {
    this.updateInfo();
  }

  // 复制日志
  async copyLogs() {
    const info = await this.gatherInfo();
    const text = JSON.stringify(info, null, 2);

    try {
      await navigator.clipboard.writeText(text);
      alert('日志已复制到剪贴板');
    } catch (error) {
      console.error('复制失败:', error);
      alert('复制失败');
    }
  }
}

// 导出
window.DebugPanel = DebugPanel;
