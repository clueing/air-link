# AirLink 问题排查指南

## 已修复的问题

### ✅ 1. 消息接收问题
**问题**：在同一个房间一个发送后另外一个并没有收到

**原因**：WebRTC 消息回调未正确注册

**修复**：
- 在 `app-core.js` 中添加了消息接收的日志
- 确保 `onMessageReceivedCallback` 正确触发
- 添加了文件数据的处理回调

**验证方法**：
1. 两个设备加入同一房间
2. 发送文本消息
3. 打开浏览器控制台，应该看到 "收到消息:" 日志
4. 对方应该立即收到消息

### ✅ 2. 文件传输失败
**问题**：发送文件提示文件传输失败

**原因**：
- 没有正确的目标设备
- 错误处理不完善

**修复**：
- 添加了设备列表检查
- 改进了错误提示
- 添加了 try-catch 错误处理
- 支持发送到多个设备

**验证方法**：
1. 确保至少有两个设备在房间内
2. 点击附件图标选择文件
3. 查看传输进度条
4. 对方应该收到文件并可以下载

### ✅ 3. 移动端首页滚动
**问题**：移动端页面在首页没进入房间的页面的时候页面太长没有滚动条

**修复**：
- 添加了 `.view-lobby` 的 `overflow-y: auto`
- 启用了 `-webkit-overflow-scrolling: touch` 以支持iOS流畅滚动
- 设置 `max-height: 100vh` 限制高度

**验证方法**：
1. 在手机上打开页面
2. 首页应该可以上下滚动
3. 所有内容都可见

### ✅ 4. iOS 输入框激活问题
**问题**：iOS设备上激活输入框后网页会变宽形成横向滚动条

**修复**：
- 更新 viewport：`maximum-scale=1, user-scalable=no`
- 强制输入框 `font-size: 16px` 防止 iOS 自动缩放
- 添加 `max-width: 100vw` 防止页面变宽
- 设置 `overflow-x: hidden` 禁止横向滚动

**验证方法**：
1. 在 iOS 设备上打开页面
2. 点击聊天输入框
3. 键盘弹出时页面不应该变宽
4. 不应该出现横向滚动条

## 当前已知限制

### 局域网扫描
**状态**：功能已实现，但可能受限于以下因素

**可能的原因**：
1. **防火墙阻止 mDNS**
   - Windows Defender 防火墙可能阻止 UDP 5353 端口
   - 解决方法：允许 airlink.exe 通过防火墙

2. **网络隔离**
   - 企业网络或公共 WiFi 可能禁用 mDNS
   - 解决方法：使用 PIN 码连接

3. **跨子网问题**
   - mDNS 不跨越子网边界
   - 解决方法：确保设备在同一子网

**手动测试**：
```powershell
# Windows 测试 mDNS
ping airlink-host.local

# 检查防火墙规则
Get-NetFirewallRule | Where-Object {$_.DisplayName -like "*airlink*"}
```

**临时解决方案**：使用 PIN 码连接（这是主要连接方式）

## 调试技巧

### 1. 启用调试面板
按 `Ctrl+Shift+D`（Windows）或 `Cmd+Shift+D`（macOS）

查看：
- WebRTC 连接状态
- ICE 候选信息
- 数据通道状态
- 设备列表

### 2. 浏览器控制台
按 `F12` 打开开发者工具，查看：
- 错误信息
- WebRTC 日志
- 网络请求

关键日志：
```
收到消息: {type: "text", content: "..."}
设备 xxx 连接状态: connected
与 xxx 建立 P2P 连接
收到文件数据: 16384 bytes
```

### 3. 服务器日志
使用 `--debug` 模式运行：
```powershell
go run .\cmd\airlink\main.go --debug
```

查看：
- WebSocket 连接
- 信令交换
- mDNS 服务状态
- 房间管理

### 4. 网络检查

**检查 WebSocket 连接**：
```javascript
// 在浏览器控制台
console.log(app.signaling.connected)  // 应该是 true
```

**检查 WebRTC 连接**：
```javascript
console.log(app.webrtc.connections.size)  // 应该 > 0
console.log(app.webrtc.dataChannels.size) // 应该 > 0
```

**检查设备列表**：
```javascript
console.log(app.getDevices())
```

## 常见问题

### Q: 为什么消息发送失败？
A: 检查：
1. WebRTC 连接是否建立（查看调试面板）
2. 对方设备是否在线
3. 浏览器控制台是否有错误

### Q: 为什么文件传输很慢？
A: 
- **局域网**：应该接近网络速度（100MB/s+）
- **公网**：受限于上传带宽（通常较慢）
- 检查是否使用了 TURN 中继（会很慢）

### Q: 为什么连接不上？
A: 检查：
1. PIN 码是否正确（6位数字）
2. PIN 是否过期（5分钟有效期）
3. 网络是否正常
4. 服务器是否运行

### Q: 移动端为什么还是有问题？
A: 
1. 清除浏览器缓存后重试
2. 确保使用最新版本的浏览器
3. iOS 需要 Safari 11+ 或 Chrome 56+
4. Android 需要 Chrome 56+ 或 Firefox 52+

## 性能优化建议

### 局域网传输
- 使用 5GHz WiFi 而不是 2.4GHz
- 避免网络拥挤
- 关闭其他占用带宽的应用

### 公网传输
- 检查上传带宽（通常是瓶颈）
- 考虑配置自己的 TURN 服务器
- 减小文件大小或压缩后传输

### 浏览器性能
- 关闭不必要的标签页
- 避免同时传输过多文件
- 定期刷新页面释放内存

## 报告问题

如果遇到新问题，请提供：
1. 浏览器版本和操作系统
2. 控制台错误信息
3. 调试面板截图
4. 服务器日志（如果有）
5. 复现步骤

---

最后更新：2024-06-22
