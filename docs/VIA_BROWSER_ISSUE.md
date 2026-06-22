# Via 浏览器（iOS WebView）兼容性问题

## 问题描述

Via 浏览器在 iOS 上使用系统 WebView（WKWebView），可以加入房间但接收不到消息。

## 原因分析

### 1. WebView 的 WebRTC 限制

iOS 系统 WebView（WKWebView）对 WebRTC 的支持有限：
- ✅ 支持基本的 RTCPeerConnection API
- ⚠️ 对 DataChannel 的支持可能不完整
- ❌ 某些 WebRTC 特性被禁用或限制

### 2. Via 浏览器特性

Via 浏览器：
- 使用系统 WebView 渲染网页
- 不是完整的浏览器引擎
- WebRTC 功能受限于系统 WebView

### 3. 可能的具体问题

#### A. DataChannel 消息接收
```javascript
// 可能在 WebView 中不触发
channel.onmessage = (event) => {
  console.log('收到消息'); // 可能不触发
};
```

#### B. SCTP 协议支持
DataChannel 使用 SCTP 协议，WebView 可能不完全支持。

#### C. 二进制数据处理
```javascript
// 可能在 WebView 中有问题
channel.binaryType = 'arraybuffer';
```

## 测试结果

| 浏览器 | 加入房间 | 发送消息 | 接收消息 | 文件传输 |
|--------|---------|---------|---------|---------|
| Safari | ✅ | ✅ | ✅ | ✅ |
| Chrome (iOS) | ✅ | ✅ | ✅ | ✅ |
| Firefox (iOS) | ✅ | ✅ | ✅ | ✅ |
| Via | ✅ | ? | ❌ | ❌ |
| 其他 WebView | ⚠️ | ⚠️ | ⚠️ | ⚠️ |

**注意**：iOS 上的 Chrome、Firefox 等也使用 WebView，但它们是完整的浏览器应用，可能有额外的权限和优化。

## 解决方案

### 方案 1：使用 Safari（推荐）

**iOS 上推荐使用 Safari**：
- ✅ 完整的 WebRTC 支持
- ✅ 最佳性能
- ✅ 所有功能正常

**操作步骤**：
1. 在 Via 中复制 URL
2. 在 Safari 中打开
3. 正常使用 AirLink

### 方案 2：添加降级方案（未实现）

可以检测 WebView 并使用 WebSocket 降级：

```javascript
// 检测是否为 WebView
function isWebView() {
  const ua = navigator.userAgent;
  return /WebView|wv|Android.*Version/i.test(ua);
}

// 如果是 WebView，使用 WebSocket 转发消息
if (isWebView()) {
  console.log('检测到 WebView，使用 WebSocket 模式');
  // 改用服务器转发消息，而不是 P2P
}
```

**优点**：兼容性好  
**缺点**：
- 需要服务器转发（失去 P2P 优势）
- 增加服务器负载
- 增加延迟

### 方案 3：提示用户切换浏览器

在检测到 WebView 时显示提示：

```javascript
if (isWebView()) {
  showToast('检测到 WebView，部分功能可能不可用，建议使用 Safari');
}
```

## 建议

### 对用户
1. **iOS 设备使用 Safari**（最佳体验）
2. 或使用 Chrome/Firefox 浏览器应用
3. 避免使用 Via 等基于 WebView 的浏览器

### 对开发者
1. **当前不修复**
   - WebView 限制是系统级的
   - 修复成本高（需要服务器转发）
   - 影响少数用户

2. **添加浏览器检测和提示**
   ```javascript
   // 检测并提示
   if (isWebView() || isBrowserWithLimitedWebRTC()) {
     showWarning('当前浏览器可能不支持完整的 P2P 功能，建议使用 Safari');
   }
   ```

3. **文档说明**
   - 在 README 中说明浏览器要求
   - 提供浏览器兼容性表

## 浏览器兼容性要求

### 推荐浏览器

**桌面端：**
- ✅ Chrome 56+
- ✅ Firefox 52+
- ✅ Edge 79+
- ✅ Safari 11+

**移动端：**
- ✅ iOS Safari 11+
- ✅ iOS Chrome 56+
- ✅ Android Chrome 56+
- ✅ Android Firefox 52+

### 不推荐

- ❌ Via 浏览器
- ❌ UC 浏览器（部分版本）
- ❌ 微信/QQ 内置浏览器
- ❌ 其他使用系统 WebView 的轻量级浏览器

## 检测代码

可以添加到应用初始化：

```javascript
function checkBrowserCompatibility() {
  // 检测 WebRTC 支持
  if (!window.RTCPeerConnection) {
    return { ok: false, reason: '不支持 WebRTC' };
  }

  // 检测 DataChannel
  try {
    const pc = new RTCPeerConnection();
    const dc = pc.createDataChannel('test');
    pc.close();
  } catch (e) {
    return { ok: false, reason: '不支持 DataChannel' };
  }

  // 检测是否为 WebView
  const ua = navigator.userAgent;
  const isWebView = /WebView|wv/i.test(ua);
  const isVia = /Via/i.test(ua);

  if (isWebView || isVia) {
    return { 
      ok: false, 
      reason: '当前浏览器可能不完全支持 P2P 功能',
      suggestion: '建议使用 Safari、Chrome 或 Firefox'
    };
  }

  return { ok: true };
}

// 初始化时检查
const compat = checkBrowserCompatibility();
if (!compat.ok) {
  showToast(compat.reason + (compat.suggestion ? '\n' + compat.suggestion : ''));
}
```

## 总结

**Via 浏览器的问题是已知的 WebView 限制，不是 AirLink 的 bug。**

**推荐解决方案：**
1. ✅ 用户：使用 Safari（iOS）或 Chrome（Android）
2. ⚠️ 开发者：添加浏览器检测和提示
3. ❌ 不推荐：实现 WebSocket 降级（成本高）

---

**最后更新**: 2024-06-22  
**状态**: 已知限制，建议用户使用完整浏览器
