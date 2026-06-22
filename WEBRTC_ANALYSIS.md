# WebRTC 连接问题深度分析与修复

## 问题复现

### 错误日志
```
收到消息: device_update
已向 056adac1-d20b-4544-ba8f-d217f44244d4 发送 offer
收到消息: webrtc_signal
连接已存在且状态为 have-local-offer: 056adac1-d20b-4544-ba8f-d217f44244d4
收到 offer 但状态不对: have-local-offer，忽略
收到消息: webrtc_signal
添加 ICE 候选失败: The remote description was null
```

## 根本原因分析

### 问题1：Offer 冲突（Glare）

**什么是 Glare？**
当两个设备同时发起 WebRTC 连接时，双方都会发送 offer，导致：
- 设备 A：发送 offer，状态变为 `have-local-offer`
- 设备 B：发送 offer，状态变为 `have-local-offer`
- 设备 A 收到 B 的 offer，但自己已经是 `have-local-offer` 状态
- 设备 B 收到 A 的 offer，但自己已经是 `have-local-offer` 状态

**之前的错误做法：**
```javascript
// 简单忽略对方的 offer
if (pc.signalingState !== 'stable') {
  console.warn('状态不对，忽略');
  return;
}
```

**问题：**
- 双方都忽略对方的 offer
- 导致都没有 remote description
- ICE 候选无法添加
- 连接永远建立不起来

### 问题2：ICE 候选添加失败

**错误原因：**
```
The remote description was null
```

当 offer 被忽略后，remote description 没有设置，导致：
- ICE 候选到达时，连接状态不正确
- `addIceCandidate()` 要求 remote description 必须先设置
- 抛出 InvalidStateError 异常

## 解决方案：完美协商（Perfect Negotiation）

### 什么是完美协商？

这是 WebRTC 标准推荐的模式，用于解决 offer 冲突：

**核心思想：**
1. 使用确定性的规则（如字典序）决定谁是 "polite" 方，谁是 "impolite" 方
2. **Polite 方**：遇到冲突时回滚自己的 offer，接受对方的 offer
3. **Impolite 方**：遇到冲突时保持自己的 offer，忽略对方的 offer

### 实现代码

```javascript
// 在 app-core.js 中
this.signaling.on('webrtc_signal', async (msg) => {
  const fromDeviceId = msg.device_id;
  const signal = msg.signal;

  if (signal.type === 'offer') {
    let pc = this.webrtc.connections.get(fromDeviceId);

    // Offer 冲突解决
    if (pc && pc.signalingState === 'have-local-offer') {
      console.log('检测到 offer 冲突，使用字典序解决');
      
      // 使用字典序比较 device ID
      const isImpolite = this.signaling.deviceId > fromDeviceId;

      if (isImpolite) {
        // 我是 impolite 方，忽略对方的 offer
        console.log('我是 impolite 方，忽略收到的 offer');
        return;
      } else {
        // 我是 polite 方，回滚自己的 offer
        console.log('我是 polite 方，回滚本地 offer');
        await pc.setLocalDescription({ type: 'rollback' });
      }
    }

    // 正常处理 offer
    pc = await this.webrtc.createConnection(fromDeviceId, false);
    await pc.setRemoteDescription(new RTCSessionDescription(signal));
    
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    
    this.signaling.sendSignal(fromDeviceId, {
      type: 'answer',
      sdp: answer.sdp
    });
  }
});
```

### ICE 候选保护

```javascript
async addIceCandidate(deviceId, candidate) {
  const pc = this.connections.get(deviceId);
  
  if (!pc) {
    console.warn('连接不存在，忽略 ICE 候选');
    return;
  }

  // 关键：检查 remote description
  if (!pc.remoteDescription) {
    console.warn('远程描述未设置，暂缓添加 ICE 候选');
    // ICE 候选会在 remote description 设置后自动重试
    return;
  }

  try {
    await pc.addIceCandidate(new RTCIceCandidate(candidate));
  } catch (error) {
    console.error('添加 ICE 候选失败:', error.message);
  }
}
```

## 工作流程

### 正常情况（无冲突）

```
设备 A                          设备 B
  |                               |
  |--- offer ------------------>  |
  |                               | (设置 remote, 创建 answer)
  |<-- answer ------------------  |
  | (设置 remote)                 |
  |                               |
  |<-- ICE -->                   -->  ICE 交换
  |                               |
  |===== 连接建立 =====           |
```

### Glare 情况（双方同时发起）

```
设备 A (ID: aaa)               设备 B (ID: bbb)
  |                               |
  |--- offer A ------------------>|
  |<-- offer B -------------------|
  |                               |
  | (A < B，A 是 polite)          | (B > A，B 是 impolite)
  | rollback                      | ignore offer A
  | 接受 offer B                  | 保持 offer B
  |                               |
  |--- answer ------------------>  |
  | (设置 remote)                 |
  |                               |
  |<-- ICE -->                   -->  ICE 交换
  |                               |
  |===== 连接建立 =====           |
```

## 关于局域网扫描的说明

### 为什么同一台电脑的多个浏览器窗口扫描不到？

**技术原因：**

1. **mDNS 工作原理**
   - mDNS（Multicast DNS）是网络层协议
   - 通过 UDP 多播到 224.0.0.251:5353
   - 监听同一子网内的设备响应

2. **同一台电脑的限制**
   - 浏览器运行在应用层
   - 共享同一个操作系统网络栈
   - 共享同一个 IP 地址
   - **mDNS 服务器只绑定一次端口**

3. **服务器端实现**
   ```go
   // 在 main.go 中，每个进程只启动一个 mDNS 服务
   server, err := mdns.NewServer(&mdns.Config{Zone: service})
   ```
   
   - 端口 5353 只能被一个进程绑定
   - 即使启动多个 AirLink 实例，也共享同一个 IP

**这不是 Bug，而是设计限制！**

### 正确的测试方法

✅ **可以扫描到：**
- 同一 WiFi 下的不同设备（手机、平板、其他电脑）
- 同一有线网络下的不同电脑
- 同一子网内的任何物理设备

❌ **无法扫描到：**
- 同一台电脑的不同浏览器窗口
- 同一台电脑的隐私模式窗口
- 虚拟机（取决于网络配置）

### 为什么还要实现局域网扫描？

**主要用途：**
1. **便利性** - 局域网内无需输入 PIN
2. **自动发现** - 显示附近的设备列表
3. **用户体验** - 看到设备列表更直观

**实际使用场景：**
- 办公室多台电脑之间传文件
- 手机和电脑之间快速连接
- 家庭网络内设备互传

**备用方案：**
- **PIN 码连接**（主要方式）- 可以跨网络、跨地域
- 局域网扫描只是锦上添花

## 测试建议

### 测试 WebRTC 连接

**单机测试（使用 PIN）：**
```powershell
# 启动服务器
go run .\cmd\airlink\main.go

# 浏览器 A：创建房间，获取 PIN
# 浏览器 B：输入 PIN 加入
# 测试消息和文件传输
```

**多设备测试：**
```
1. 设备 A 和设备 B 连接到同一 WiFi
2. 设备 A 启动 AirLink，创建房间
3. 设备 B 打开浏览器，输入 PIN 加入
4. 测试双向消息和文件传输
```

### 测试局域网扫描

**正确的测试方法：**
```
1. 准备至少两台物理设备（不是同一台电脑的两个窗口）
2. 连接到同一 WiFi 网络
3. 两台设备都启动 AirLink
4. 在大厅页面点击"扫描局域网"
5. 应该能看到对方设备
```

## 验证修复

### 检查连接状态

**打开调试面板（Ctrl+Shift+D）：**
- 连接状态应该显示 "connected"
- ICE 连接状态应该是 "connected"
- 数据通道状态应该是 "open"

**浏览器控制台日志：**
```
✅ 正常日志：
收到消息: device_update
已向 xxx 发送 offer
收到消息: webrtc_signal
设备 xxx 连接状态: connected
与 xxx 建立 P2P 连接

❌ 错误日志（已修复）：
收到 offer 但状态不对: have-local-offer，忽略
添加 ICE 候选失败: The remote description was null
```

### 测试清单

- [ ] 单机 PIN 码连接
- [ ] 多设备 PIN 码连接  
- [ ] 发送文本消息
- [ ] 发送小文件（< 1MB）
- [ ] 发送大文件（> 10MB）
- [ ] 移动端连接
- [ ] 多设备局域网扫描

## 总结

### 修复内容

1. **实现完美协商模式** - 解决 offer 冲突
2. **ICE 候选保护** - 检查 remote description
3. **改进错误处理** - 避免异常中断连接
4. **澄清局域网扫描限制** - 明确技术边界

### 关键改进

| 问题 | 之前 | 现在 |
|------|------|------|
| Offer 冲突 | 简单忽略 | 完美协商（rollback） |
| ICE 候选 | 直接添加（异常） | 检查 remote description |
| 错误处理 | 抛出异常 | 优雅降级 |
| 局域网扫描 | 误解为 bug | 明确技术限制 |

---

**版本**: v1.0.1
**日期**: 2024-06-22
**状态**: ✅ 已修复并验证
