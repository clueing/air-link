# 测试说明

## 关键修复

**问题**：加入房间时双方同时发起连接导致状态冲突

**修复**：加入方不再主动发起连接，等待现有设备主动连接

## 测试步骤

### 1. 启动服务器
```powershell
go run .\cmd\airlink\main.go
```

### 2. 三设备测试

**设备 A（创建房间）：**
1. 打开 http://localhost:8080
2. 点击"创建房间"
3. 记下 PIN 码（如 123456）

**设备 B（第一个加入）：**
1. 打开 http://localhost:8080
2. 输入 PIN 码
3. 点击"加入"
4. 等待连接建立

**设备 C（第二个加入）：**
1. 打开 http://localhost:8080
2. 输入相同的 PIN 码
3. 点击"加入"
4. 等待连接建立

### 3. 验证功能

✅ **检查连接状态**
- 所有设备的调试面板（Ctrl+Shift+D）显示 "connected"
- DataChannel 显示 "open"

✅ **发送文本消息**
- 任意设备发送消息
- 其他所有设备都应该收到

✅ **发送文件**
- 任意设备选择文件发送
- 其他所有设备都应该收到并能下载

### 4. 预期结果

**浏览器控制台日志（正常）：**
```
WebSocket 已连接
AirLink 初始化完成
收到消息: room_joined (或 room_created)
收到消息: device_update
已向 xxx 发送 offer (仅现有设备)
收到消息: webrtc_signal
ICE 状态: checking → connected
连接状态: connecting → connected
DataChannel 已打开
```

**不应该看到的错误：**
- ❌ Called in wrong state: have-remote-offer
- ❌ DataChannel 未就绪
- ❌ 远程描述未设置（ICE 候选添加时）

### 5. 已知正常日志

**警告（正常）：**
```
远程描述未设置，暂缓添加 ICE 候选
```
这是正常的，因为 ICE 候选可能在 offer/answer 交换完成前到达。

**连接已存在（正常）：**
```
连接已存在且状态为 stable
```
这说明连接已经建立成功。

## 故障排除

### 如果仍然有错误

1. **清除浏览器缓存**
   ```
   Ctrl+Shift+Delete → 清除缓存
   或
   Ctrl+F5 强制刷新
   ```

2. **重启服务器**
   ```powershell
   # 停止当前服务器 (Ctrl+C)
   go run .\cmd\airlink\main.go
   ```

3. **检查 Git 版本**
   ```powershell
   git log --oneline -1
   ```
   应该看到：
   ```
   c91ea6f fix(webrtc): 修复加入房间时的连接冲突，加入方不再主动发起连接
   ```

## 成功标志

- ✅ 3 个设备都能加入同一房间
- ✅ 没有 "wrong state" 错误
- ✅ 所有 DataChannel 都显示 "已打开"
- ✅ 消息和文件可以在所有设备间传输

## 如果测试成功

恭喜！AirLink 现在完全可用。

**核心功能：**
- ✅ PIN 码房间
- ✅ 多设备连接（3+ 设备）
- ✅ 实时消息
- ✅ P2P 文件传输
- ✅ 移动端支持

**下一步：**
- 构建生产版本：`make build-all`
- 部署使用

---

**版本**: v1.0.2  
**最后更新**: 2024-06-22  
**状态**: ✅ 已修复所有已知问题
