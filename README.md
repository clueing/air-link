# AirLink

**AirLink** 是一个基于网页的 P2P 文件和消息传输工具，支持局域网自动发现和公网 PIN 配对连接，实现类似 AirDrop 的体验。

## 特性

- ✨ **无需注册登录**：打开即用，输入 PIN 即可连接
- 🔒 **P2P 直连**：文件和消息端到端直传，不经服务器
- 🌐 **局域网发现**：自动发现同一局域网内的设备
- 🌍 **公网支持**：通过 PIN 码配对，支持跨网络传输
- 💻 **跨平台**：支持 Windows/macOS/Linux，单个可执行文件
- 📱 **移动端兼容**：浏览器访问，支持手机和平板

## 快速开始

### 下载

前往 [Releases](https://github.com/yourusername/air-link/releases) 页面下载对应平台的可执行文件。

### 5 分钟快速入门

详细的快速入门指南请查看：**[📖 快速入门指南](docs/QUICKSTART.md)**

### 简要步骤

1. 启动 AirLink（自动打开浏览器）
2. 点击"创建房间"，获得 6 位 PIN 码
3. 其他设备输入 PIN 码加入
4. 开始传输文件和消息

## 📚 文档

- **[快速入门指南](docs/QUICKSTART.md)** - 详细的使用教程
- **[问题排查](docs/TROUBLESHOOTING.md)** - 常见问题解决方案
- **[测试指南](docs/TEST_GUIDE.md)** - 如何测试 AirLink
- **[局域网扫描说明](docs/LAN_SCAN_FAQ.md)** - 局域网扫描功能详解
- **[更多文档](docs/README.md)** - 完整文档索引

## 浏览器兼容性

### ✅ 推荐浏览器

- Chrome 56+
- Firefox 52+
- Safari 11+
- Edge 79+

### ⚠️ 已知限制

- **Via 浏览器**（iOS WebView）- 部分功能受限，详见 [Via 浏览器兼容性](docs/VIA_BROWSER_ISSUE.md)
- 其他 WebView 浏览器可能存在类似问题

### 🔒 隐私说明

AirLink 使用 P2P 技术，连接时会交换 IP 地址：
- 只与房间内设备共享
- 服务器不存储 IP 地址
- 建议只与信任的人共享 PIN 码
- 如需隐藏 IP，请使用 VPN

## 配置

### 配置文件

创建 `airlink.yaml` 配置文件：

```yaml
# 服务器配置
server:
  port: 8080
  host: "0.0.0.0"
  auto_open_browser: true

# WebRTC 配置
webrtc:
  stun_servers:
    - "stun:stun.l.google.com:19302"
  turn_servers: []

# PIN 房间配置
room:
  pin_expiry: 300  # 5 分钟
  max_devices: 10

# 安全配置
security:
  rate_limit:
    max_attempts: 5
    window_time: 60
    lockout_time: 600

# 局域网发现
discovery:
  mdns_enabled: true
  http_probe_enabled: true
  service_name: "_airlink._tcp"

# 日志配置
logging:
  level: "info"
  file: ""
```

### 命令行参数

```bash
airlink [flags]

Flags:
  --port int          服务器端口 (default 8080)
  --config string     配置文件路径 (default "./airlink.yaml")
  --no-browser        不自动打开浏览器
  --debug             开启调试模式
  --version           显示版本信息
  --help              显示帮助信息
```

示例：

```bash
# 使用自定义端口
./airlink --port 9000

# 不自动打开浏览器
./airlink --no-browser

# 调试模式
./airlink --debug

# 使用自定义配置文件
./airlink --config /path/to/config.yaml
```

## 开发

### 环境要求

- Go 1.21+
- Node.js（可选，仅用于前端开发）

### 构建

```bash
# 克隆仓库
git clone https://github.com/yourusername/air-link.git
cd air-link

# 安装依赖
go mod download

# 开发运行
make dev

# 构建当前平台
make build

# 构建所有平台
make build-all

# 运行测试
make test
```

### 项目结构

```
air-link/
├── cmd/airlink/          # 主程序入口
├── internal/
│   ├── config/           # 配置管理
│   ├── server/           # HTTP/WebSocket 服务
│   ├── signaling/        # 信令逻辑
│   ├── discovery/        # 局域网发现
│   └── security/         # 安全模块
├── internal/server/web/  # 前端资源
│   ├── index.html
│   ├── webrtc.js
│   ├── signaling.js
│   ├── file-transfer.js
│   └── app-core.js
├── airlink.yaml          # 默认配置
├── CLAUDE.md             # 技术文档
├── Makefile
└── README.md
```

## 技术架构

- **后端**：Go + gorilla/websocket + hashicorp/mdns
- **前端**：原生 JavaScript + WebRTC API
- **信令**：WebSocket（纯 JSON）
- **传输**：WebRTC DataChannel（P2P）
- **发现**：mDNS + HTTP 探测

详细技术文档请参阅 [CLAUDE.md](CLAUDE.md)。

## 安全性

- WebRTC 默认使用 DTLS 加密（端到端）
- PIN 码有速率限制，防止暴力破解
- 文件不经过服务器，不存储
- 信令服务器仅中继连接信息

## 常见问题

### 无法连接到其他设备？

1. 检查防火墙设置，确保允许端口 8080
2. 确认两台设备在同一局域网内，或使用正确的 PIN
3. 如果在复杂 NAT 环境下，可能需要配置 TURN 服务器

### 文件传输速度慢？

- 局域网内传输速度取决于网络带宽
- 公网传输速度取决于上传带宽（对方的下载速度）
- 尝试关闭其他占用带宽的程序

### 浏览器兼容性？

支持现代浏览器：
- Chrome 56+
- Firefox 52+
- Safari 11+
- Edge 79+

## 许可证

[MIT License](LICENSE)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

- GitHub: https://github.com/yourusername/air-link
- Issues: https://github.com/yourusername/air-link/issues
