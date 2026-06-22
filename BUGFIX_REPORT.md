# Bug 修复报告

## 修复日期
2024-06-22

## 修复的问题

### 1. ✅ 消息接收失败
**问题描述**：在同一个房间一个发送后另外一个并没有收到

**原因分析**：
- WebRTC 消息回调未正确注册
- `onMessageReceivedCallback` 触发时机不对

**修复方案**：
- 在 `app-core.js` 的 `setupWebRTCHandlers()` 中添加了详细的日志
- 确保消息回调正确传递到上层应用
- 添加了消息类型判断和处理

**验证方法**：
```javascript
// 浏览器控制台应该看到
收到消息: {type: "text", content: "..."}
```

**提交**: `f1da08c fix(ui): 修复消息接收、文件传输和移动端UI问题`

---

### 2. ✅ 文件传输失败
**问题描述**：发送文件提示文件传输失败

**原因分析**：
- 没有正确获取目标设备列表
- 缺少错误处理
- 设备ID未正确传递

**修复方案**：
- 添加了设备列表检查（`deviceList.length === 0`）
- 添加了 try-catch 错误处理
- 改进了错误提示信息
- 支持发送到房间内所有设备
- 添加了应用未初始化的检查

**修复代码**：
```javascript
if (!app || !fileTransfer) {
  showToast('应用未初始化');
  return;
}

const deviceList = app.getDevices().filter(d => !d.isYou);

if (deviceList.length === 0) {
  showToast('房间内没有其他设备');
  return;
}
```

**提交**: `f1da08c fix(ui): 修复消息接收、文件传输和移动端UI问题`

---

### 3. ✅ 移动端首页滚动问题
**问题描述**：移动端页面在首页没进入房间的页面的时候页面太长没有滚动条

**原因分析**：
- 大厅页面没有设置 `overflow-y`
- 移动端需要特殊的滚动优化

**修复方案**：
```css
@media (max-width: 767px) {
  .view-lobby {
    overflow-y: auto !important;
    -webkit-overflow-scrolling: touch;
    max-height: 100vh;
  }
}
```

**提交**: `f1da08c fix(ui): 修复消息接收、文件传输和移动端UI问题`

---

### 4. ✅ iOS 输入框激活导致页面变宽
**问题描述**：iOS设备上激活输入框后网页会变宽形成横向滚动条

**原因分析**：
- iOS Safari 在输入框获得焦点时会自动缩放页面
- viewport 设置不够严格
- 没有限制页面最大宽度

**修复方案**：

**1. 更新 viewport：**
```html
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover" />
```

**2. 添加 CSS 修复：**
```css
/* 防止iOS自动缩放 */
input, textarea {
  font-size: 16px !important;
}

/* 防止页面变宽 */
html, body {
  max-width: 100vw;
  position: fixed;
}

.app {
  max-width: 100vw;
  overflow-x: hidden;
}
```

**提交**: `f1da08c fix(ui): 修复消息接收、文件传输和移动端UI问题`

---

### 5. ✅ iOS 键盘换行被识别成发送
**问题描述**：iOS设备上键盘的换行键会触发消息发送，而不是换行

**原因分析**：
- iOS 软键盘的换行键也会触发 `Enter` 事件
- 没有区分桌面端和移动端的行为

**修复方案**：
```javascript
window.onMsgKey = function (e) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);

    // 桌面端：Enter 发送，Shift+Enter 换行
    // 移动端：Enter 换行，通过按钮发送
    if (!isMobile) {
      e.preventDefault();
      sendMessage();
    }
  }
  // ...
};
```

**行为说明**：
- **桌面端**：按 Enter 发送消息，Shift+Enter 换行
- **移动端**：按 Enter 换行，点击发送按钮发送消息

**提交**: `283607e fix(mobile): 修复iOS键盘换行被识别成发送的问题`

---

## 局域网扫描说明

**状态**：功能已实现，但可能受网络环境限制

**可能的限制因素**：
1. 防火墙阻止 UDP 5353 端口（mDNS）
2. 企业网络或公共 WiFi 禁用 mDNS
3. 跨子网无法发现

**解决方案**：
- 主要使用 **PIN 码连接**（这是设计的主要方式）
- 局域网扫描作为辅助功能
- 可以在同一子网且无防火墙限制时使用

---

## 测试建议

### 基本功能测试
1. ✅ 创建房间并获取 PIN
2. ✅ 另一设备输入 PIN 加入
3. ✅ 发送文本消息（双向）
4. ✅ 发送文件（各种大小）
5. ✅ 查看传输进度
6. ✅ 下载接收的文件

### 移动端测试
1. ✅ 首页可以滚动
2. ✅ 输入框激活不会导致页面变宽
3. ✅ 键盘换行不会发送消息
4. ✅ 点击发送按钮可以发送
5. ✅ 文件上传和下载正常

### 调试测试
1. ✅ 按 Ctrl+Shift+D 打开调试面板
2. ✅ 查看连接状态
3. ✅ 查看设备列表
4. ✅ 查看 WebRTC 统计信息

---

## 后续优化建议

### 短期
- [ ] 添加文件传输取消功能
- [ ] 添加传输速度显示
- [ ] 优化大文件传输性能
- [ ] 添加传输失败重试机制

### 中期
- [ ] 支持选择发送目标设备
- [ ] 支持拖拽上传文件
- [ ] 添加文件预览功能
- [ ] 优化移动端UI适配

### 长期
- [ ] 添加语音消息支持
- [ ] 添加图片压缩选项
- [ ] 支持传输历史记录
- [ ] 添加端到端加密选项

---

## 相关文档

- [快速入门指南](QUICKSTART.md)
- [问题排查指南](TROUBLESHOOTING.md)
- [技术设计文档](CLAUDE.md)
- [项目完成总结](PROJECT_SUMMARY.md)

---

最后更新：2024-06-22
版本：v1.0.0
