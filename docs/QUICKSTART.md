# AirLink 快速入门

## 🚀 快速开始（开发环境）

### 1. 启动服务器

```powershell
# 开发模式运行（自动打开浏览器）
go run .\cmd\airlink\main.go

# 或者不自动打开浏览器
go run .\cmd\airlink\main.go --no-browser

# 使用自定义端口
go run .\cmd\airlink\main.go --port 9000

# 调试模式
go run .\cmd\airlink\main.go --debug
```

### 2. 访问应用

服务器启动后，在浏览器中访问：
```
http://localhost:8080
```

### 3. 使用步骤

#### 创建房间
1. 点击"创建房间"按钮
2. 页面会显示一个 6 位 PIN 码（如：123456）
3. 将 PIN 码分享给需要连接的设备

#### 加入房间
1. 在另一台设备上打开 AirLink
2. 输入 6 位 PIN 码
3. 点击"加入"按钮
4. 等待连接建立（通常 3-10 秒）

#### 发送消息
1. 连接成功后，在聊天框输入文本
2. 按 Enter 发送
3. 对方实时收到消息

#### 发送文件
1. 点击聊天框左侧的"附件"图标 📎
2. 选择要发送的文件
3. 文件自动通过 P2P 传输
4. 可以看到实时传输进度
5. 对方收到后点击"保存"下载文件

#### 调试面板
- 按 `Ctrl+Shift+D`（Windows/Linux）或 `Cmd+Shift+D`（macOS）
- 打开开发者调试面板
- 查看连接状态、WebRTC 统计等信息

## 📦 构建可执行文件

### 构建当前平台
```powershell
# 使用 Makefile
make build

# 或直接使用 go build
go build -o dist/airlink.exe .\cmd\airlink\main.go
```

### 跨平台构建
```powershell
# 构建所有平台（Windows/macOS/Linux）
make build-all
```

生成的文件在 `dist/` 目录：
- `airlink-windows-amd64.exe` - Windows 64位
- `airlink-darwin-amd64` - macOS Intel
- `airlink-darwin-arm64` - macOS Apple Silicon
- `airlink-linux-amd64` - Linux 64位

## ⚙️ 配置

### 默认配置
编辑 `airlink.yaml` 文件：

```yaml
server:
  port: 8080                    # 修改端口
  auto_open_browser: false       # 是否自动打开浏览器

webrtc:
  stun_servers:                 # STUN 服务器列表
    - "stun:stun.l.google.com:19302"
    - "stun:stun.miwifi.com:3478"

room:
  pin_expiry: 300               # PIN 有效期（秒）
  max_devices: 10               # 房间最多设备数

security:
  rate_limit:
    max_attempts: 5             # 最大尝试次数
    window_time: 60             # 时间窗口（秒）
    lockout_time: 600           # 锁定时间（秒）
```

### 环境变量
也可以使用环境变量（优先级高于配置文件）：

```powershell
$env:AIRLINK_SERVER_PORT = "9000"
go run .\cmd\airlink\main.go
```

## 🔍 功能测试

### 测试局域网连接
1. 在两台同一局域网的设备上启动 AirLink
2. 设备 A 创建房间，获得 PIN
3. 设备 B 输入 PIN 加入
4. 发送消息和文件测试

### 测试公网连接
1. 在不同网络的两台设备上启动
2. 使用 PIN 连接
3. 测试连接速度和稳定性

### 测试局域网发现
1. 在大厅页面点击"扫描局域网"
2. 查看发现的设备列表
3. 点击"连接"直接连接（功能开发中）

## 🐛 故障排查

### 无法连接到服务器
- 检查防火墙是否允许端口 8080
- 确认服务器是否正在运行
- 尝试使用 `--debug` 模式查看详细日志

### PIN 码无效
- 检查 PIN 是否输入正确（6 位数字）
- PIN 码有 5 分钟有效期，过期需重新创建
- 检查网络连接是否正常

### 文件传输失败
- 检查 WebRTC 连接是否建立成功
- 打开调试面板（Ctrl+Shift+D）查看连接状态
- 尝试发送较小的文件测试
- 检查浏览器控制台是否有错误

### 连接速度慢
- 局域网传输应该很快（接近网络速度）
- 公网传输受上传带宽限制
- 尝试关闭其他占用带宽的程序
- 检查是否使用了 TURN 中继（较慢）

## 📚 更多信息

- **技术文档**：查看 `CLAUDE.md`
- **项目总结**：查看 `PROJECT_SUMMARY.md`
- **完整文档**：查看 `README.md`

## 🎯 下一步

1. ✅ 测试基本功能（创建/加入房间、发送消息/文件）
2. ✅ 测试局域网发现功能
3. ✅ 测试跨网络连接
4. ✅ 构建生产版本
5. ✅ 部署使用

---

**享受 P2P 文件传输的便捷！** 🎉
