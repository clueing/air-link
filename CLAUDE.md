# AirLink 技术设计文档

## 项目概述

AirLink 是一个基于网页的 P2P 文件和消息传输工具，支持 PIN 码配对连接，实现类似 AirDrop 的体验。

### 核心特性

- **无需注册登录**：打开即用，输入 PIN 即可连接
- **P2P 直连**：文件和消息端到端直传，不经服务器
- **PIN 码房间**：通过 6 位 PIN 码配对，支持局域网和跨网络传输
- **二维码加入**：扫码即可自动加入房间，移动端更方便
- **一键分享**：复制带 PIN 的链接，发送给对方即可连接
- **跨平台**：支持 Windows/macOS/Linux，单个可执行文件

## 技术架构

### 整体架构

```
┌─────────────┐         WebSocket         ┌─────────────┐
│  浏览器 A   │ ←──────信令交换──────→    │ Go 信令服务器 │
│  (前端)     │                           │             │
└─────────────┘                           └─────────────┘
      ↓                                          ↑
   WebRTC P2P (文件/消息直传)              WebSocket
      ↓                                          ↓
┌─────────────┐                           ┌─────────────┐
│  浏览器 B   │ ←──────信令交换──────→    │  浏览器 C   │
│  (前端)     │                           │  (前端)     │
└─────────────┘                           └─────────────┘
      ↖                                        ↗
        ╲                                    ╱
         ╲        WebRTC P2P (Mesh)        ╱
          ╲                                ╱
           ↘────────────────────────────↗
```

### 技术栈

#### Go 后端
- **Web 框架**：标准库 `net/http`
- **WebSocket**：`gorilla/websocket`
- **配置管理**：`viper`（支持 YAML/环境变量/命令行）
- **mDNS 服务**：`hashicorp/mdns`（服务注册，非扫描）
- **日志**：`logrus` 或 `zap`

#### 前端
- **纯原生 JavaScript**：无构建步骤，直接运行
- **WebRTC**：原生 API（完全控制）
- **二维码**：QRCode.js（本地化）
- **UI**：基于设计稿的自定义 CSS

## 核心功能设计

### 1. 设备连接

#### 1.1 PIN 码房间系统

**连接方式：**
1. **创建房间**：Host 设备创建房间，生成 6 位 PIN 码和二维码
2. **加入房间**：
   - 扫描二维码（自动填充 PIN 并加入）
   - 复制链接发送给对方（URL 带 `?pin=123456` 参数）
   - 手动输入 PIN 码

**PIN 生成规则：**
- 6 位数字（100000-999999）
- 随机生成，避免连续或重复模式
- 有效期：5 分钟（可配置）

**二维码功能：**
- 使用 QRCode.js 生成二维码
- 二维码内容：`https://your-domain/?pin=123456`
- 扫码后自动跳转并加入房间
- 移动端友好，白色背景、圆角、阴影效果

**房间生命周期：**
1. **创建**：Host 设备创建房间 → 生成 PIN → 显示等待界面（含二维码）
2. **加入**：其他设备输入 PIN / 扫码 / 点击链接 → 验证成功 → 加入房间
3. **过期**：5 分钟后 PIN 失效，但已连接设备继续工作
4. **销毁**：所有设备离开后，房间自动销毁

**安全机制：**
- 速率限制：同一设备 ID 每分钟最多 5 次尝试
- 锁定机制：5 次失败后锁定 10 分钟
- 内存管理：房间数据仅在内存中，不持久化

### 2. WebRTC 连接

#### 2.1 连接拓扑

**Mesh 全连接模式：**
- N 个设备需要建立 N×(N-1)/2 个 P2P 连接
- 任意两设备可直接传输文件，无需中转
- 去中心化：Host 离开后房间继续存在

**连接建立流程：**
```
设备 A 加入房间
  ↓
服务器推送现有设备列表给 A
  ↓
A 向每个现有设备发起 WebRTC Offer
  ↓
信令通过服务器中继：A → Server → B
  ↓
B 返回 Answer：B → Server → A
  ↓
ICE 候选交换（可能多次）
  ↓
P2P 连接建立成功
```

#### 2.2 ICE 配置

**STUN/TURN 服务器：**
- **默认 STUN**：`stun:stun.l.google.com:19302`
- **用户可配置**：前端设置界面 + 后端配置文件
- **TURN**：可选，用户自行添加

**ICE 策略：**
- `iceTransportPolicy: 'all'`：优先尝试直连，自动降级到 TURN
- 连接超时：30 秒
- 失败处理：显示友好错误消息 + 诊断信息（开发者模式）

### 3. 消息与文件传输

#### 3.1 DataChannel 协议

**消息类型：**

```json
// 1. 文本消息
{
  "type": "text",
  "sender_id": "uuid",
  "sender_name": "MacBook Pro",
  "content": "你好",
  "timestamp": 1234567890
}

// 2. 文件元数据（开始传输）
{
  "type": "file_meta",
  "file_id": "uuid",
  "name": "document.pdf",
  "size": 1048576,
  "mime_type": "application/pdf",
  "total_chunks": 64
}

// 3. 文件分片（二进制）
// 前 8 字节：[4 字节 file_id_hash][4 字节 chunk_index]
// 后续字节：分片数据（最多 16KB）

// 4. 传输控制
{
  "type": "transfer_control",
  "file_id": "uuid",
  "action": "pause" | "resume" | "cancel"
}

// 5. ACK 确认
{
  "type": "ack",
  "file_id": "uuid",
  "chunk_index": 42
}
```

#### 3.2 文件分片与传输

**分片策略：**
- **分片大小**：16KB（所有浏览器兼容）
- **并发控制**：
  - 发送队列：最多同时传输 2-3 个文件
  - 接收队列：独立计数，不限制
  - 超出并发限制的文件进入等待队列（FIFO）

**断点续传（会话内）：**
- 传输状态存储在内存中
- 连接断开后自动重连（最多 3 次）
- 重连后继续传输未完成的分片
- 刷新页面后重新开始（不持久化）

**进度管理：**
```javascript
// 发送端状态
{
  file_id: "uuid",
  total_chunks: 64,
  sent_chunks: [0, 1, 2, ...],  // 已发送的分片索引
  acked_chunks: [0, 1, ...],     // 已确认的分片索引
  status: "sending" | "paused" | "completed" | "failed"
}

// 接收端状态
{
  file_id: "uuid",
  total_chunks: 64,
  received_chunks: [0, 1, 2, ...],
  chunks_data: Map<index, ArrayBuffer>,
  status: "receiving" | "completed"
}
```

### 4. 设备管理

#### 4.1 设备标识

**设备 ID：**
- 持久化 UUID，存储在 `localStorage`
- 首次访问时生成，清除浏览器数据后重新生成
- 用于速率限制、设备去重

**设备名称：**
- 默认生成：`平台类型 #随机后缀`
  - 例如：`Windows PC #A3B2`, `MacBook #7F8E`, `iPhone #D1C4`
- 用户可自定义编辑
- 允许重名（通过 device_id 区分）

**平台检测规则：**
```javascript
const ua = navigator.userAgent;
const platform = navigator.platform;

if (/android/i.test(ua)) return 'Android 设备';
if (/iPad|iPhone|iPod/.test(platform)) return 'iOS 设备';
if (/Win/.test(platform)) return 'Windows PC';
if (/Mac/.test(platform)) return 'MacBook';  // 或 iMac, Mac mini
if (/Linux/.test(platform)) return 'Linux 工作站';
```

#### 4.2 在线状态同步

**心跳机制：**
- 前端每 30 秒发送一次心跳 ping
- 服务器 60 秒未收到心跳 → 标记设备离线
- 离线通知推送给房间内所有设备

**设备列表更新：**
```json
// 服务器推送消息
{
  "type": "device_update",
  "action": "join" | "leave",
  "device": {
    "device_id": "uuid",
    "device_name": "MacBook Pro",
    "status": "online"
  },
  "total_devices": 3
}
```

### 5. 安全与隐私

#### 5.1 传输加密

- **WebRTC DTLS**：默认启用，端到端加密
- **信令安全**：WebSocket 可选 TLS（wss://）
- **无服务器存储**：文件和消息不经过服务器，不持久化

#### 5.2 PIN 安全

**防暴力破解：**
```go
// 速率限制配置
type RateLimit struct {
    MaxAttempts   int           // 5 次
    WindowTime    time.Duration // 1 分钟
    LockoutTime   time.Duration // 10 分钟
}

// 尝试记录
type AttemptRecord struct {
    DeviceID      string
    Attempts      int
    LastAttempt   time.Time
    LockedUntil   time.Time
}
```

**PIN 验证流程：**
1. 检查设备是否被锁定
2. 检查 PIN 是否存在且未过期
3. 记录尝试次数
4. 验证失败 → 累计失败次数
5. 达到阈值 → 锁定设备

#### 5.3 恶意文件防护

**用户侧防护：**
- 显示文件类型、大小、发送者
- 用户自行判断是否接受
- 浏览器原生下载安全检查（Windows Defender、macOS Gatekeeper）

**不实施的防护：**
- 文件内容扫描（性能开销大，隐私风险）
- 文件类型白名单（限制合法使用场景）

## 协议定义

### WebSocket 信令协议

#### 客户端 → 服务器

```json
// 1. 连接握手
{
  "type": "hello",
  "device_id": "uuid",
  "device_name": "MacBook Pro"
}

// 2. 创建房间
{
  "type": "create_room",
  "device_id": "uuid",
  "device_name": "MacBook Pro"
}

// 3. 加入房间
{
  "type": "join_room",
  "pin": "123456",
  "device_id": "uuid",
  "device_name": "MacBook Pro"
}

// 4. 扫描局域网
{
  "type": "scan_lan"
}

// 5. WebRTC 信令
{
  "type": "webrtc_signal",
  "to_device_id": "uuid",
  "signal": {
    "type": "offer" | "answer" | "ice",
    "sdp": "...",  // offer/answer
    "candidate": {...}  // ICE candidate
  }
}

// 6. 心跳
{
  "type": "ping"
}

// 7. 离开房间
{
  "type": "leave_room"
}
```

#### 服务器 → 客户端

```json
// 1. 握手确认
{
  "type": "hello_ack",
  "server_version": "1.0.0"
}

// 2. 房间创建成功
{
  "type": "room_created",
  "pin": "123456",
  "expires_at": 1234567890
}

// 3. 加入房间成功
{
  "type": "room_joined",
  "pin": "123456",
  "devices": [
    {"device_id": "uuid", "device_name": "Windows PC"}
  ]
}

// 4. 局域网扫描结果
{
  "type": "lan_scan_result",
  "devices": [
    {"device_id": "uuid", "device_name": "iMac", "ip": "192.168.1.100"}
  ]
}

// 5. WebRTC 信令中继
{
  "type": "webrtc_signal",
  "from_device_id": "uuid",
  "signal": {...}
}

// 6. 设备上线/离线
{
  "type": "device_update",
  "action": "join" | "leave",
  "device": {"device_id": "uuid", "device_name": "MacBook"}
}

// 7. 错误
{
  "type": "error",
  "code": "INVALID_PIN" | "ROOM_EXPIRED" | "RATE_LIMITED",
  "message": "PIN 无效或已过期"
}

// 8. 心跳响应
{
  "type": "pong"
}
```

## 配置管理

### 配置文件格式

**airlink.yaml**

```yaml
# 服务器配置
server:
  port: 8080                    # HTTP/WebSocket 端口
  host: "0.0.0.0"              # 监听地址
  auto_open_browser: true       # 启动时自动打开浏览器

# WebRTC 配置
webrtc:
  stun_servers:
    - "stun:stun.l.google.com:19302"
    - "stun:stun1.l.google.com:19302"
  turn_servers: []               # 可选，用户自行配置
    # - url: "turn:your-turn-server.com:3478"
    #   username: "user"
    #   credential: "pass"

# PIN 房间配置
room:
  pin_expiry: 300                # PIN 有效期（秒）
  max_devices: 10                # 单个房间最多设备数

# 安全配置
security:
  rate_limit:
    max_attempts: 5              # 最大尝试次数
    window_time: 60              # 时间窗口（秒）
    lockout_time: 600            # 锁定时间（秒）

# 日志配置
logging:
  level: "info"                  # debug, info, warn, error
  file: ""                       # 空字符串表示仅输出到控制台
```

### 命令行参数

```bash
./airlink [flags]

Flags:
  --port int          服务器端口 (default 8080)
  --config string     配置文件路径 (default "./airlink.yaml")
  --no-browser        不自动打开浏览器
  --debug             开启调试模式
  --version           显示版本信息
  --help              显示帮助信息
```

### 配置优先级

命令行参数 > 环境变量 > 配置文件 > 默认值

## 项目结构

```
air-link/
├── cmd/
│   └── airlink/
│       └── main.go              # 程序入口
├── internal/
│   ├── server/
│   │   ├── server.go            # HTTP 服务器
│   │   └── websocket.go         # WebSocket 处理
│   ├── signaling/
│   │   ├── room.go              # 房间管理
│   │   ├── device.go            # 设备管理
│   │   └── relay.go             # 信令中继
│   ├── discovery/
│   │   ├── mdns.go              # mDNS 发现
│   │   └── probe.go             # HTTP 探测
│   ├── security/
│   │   └── ratelimit.go         # 速率限制
│   └── config/
│       └── config.go            # 配置管理
├── web/
│   ├── index.html               # 主页面
│   ├── app.js                   # 应用逻辑
│   └── style.css                # 样式（基于设计稿）
├── airlink.yaml                 # 默认配置
├── go.mod
├── go.sum
├── Makefile                     # 构建脚本
├── CLAUDE.md                    # 本文档
└── README.md                    # 用户文档
```

## 错误处理与用户反馈

### 错误类型与提示

| 错误场景 | 错误码 | 用户提示 | 开发者信息 |
|---------|--------|---------|-----------|
| PIN 不存在 | `INVALID_PIN` | PIN 无效或已过期 | PIN not found in memory |
| 速率限制 | `RATE_LIMITED` | 尝试次数过多，请稍后再试 | Device locked until {time} |
| 房间已满 | `ROOM_FULL` | 房间已达人数上限 | Max {n} devices per room |
| WebRTC 超时 | `CONNECTION_TIMEOUT` | 连接超时，请检查网络 | ICE gathering timeout |
| 服务器断开 | `SERVER_DISCONNECTED` | 与服务器的连接已断开 | WebSocket closed |
| 文件传输失败 | `TRANSFER_FAILED` | 文件传输失败，点击重试 | DataChannel error: {reason} |

### 自动重试机制

**WebSocket 重连：**
- 断开后立即重连（最多 3 次）
- 指数退避：1s, 2s, 4s
- 失败后显示"无法连接到服务器"

**WebRTC 重连：**
- 连接断开后自动重建（最多 3 次）
- 重连期间暂停文件传输
- 重连成功后自动续传

**文件传输重试：**
- 单个分片失败 → 自动重发（最多 3 次）
- 整个文件失败 → 显示"传输失败"+ 重试按钮

### 开发者模式

**激活方式：**
- 按下 `Ctrl+Shift+D`（或 `Cmd+Shift+D`）
- 或在控制台输入 `window.enableDebugMode()`

**调试面板内容：**
- WebRTC 连接状态（ICE connection state, gathering state）
- ICE 候选列表（local, remote, selected pair）
- 传输统计（发送/接收字节数、丢包率、RTT）
- DataChannel 状态（bufferedAmount, readyState）
- 详细日志（WebSocket 消息、信令交换、错误堆栈）

## 构建与部署

### 开发环境

```bash
# 安装依赖
go mod download

# 运行开发服务器
go run cmd/airlink/main.go --debug

# 或使用 Makefile
make dev
```

### 生产构建

```bash
# 跨平台构建
make build-all

# 生成以下文件：
# dist/airlink-windows-amd64.exe
# dist/airlink-darwin-amd64
# dist/airlink-darwin-arm64
# dist/airlink-linux-amd64
```

**构建配置：**
- 静态编译：`CGO_ENABLED=0`
- 内嵌资源：使用 `embed` 包内嵌 `web/` 目录
- 压缩：使用 `upx` 压缩可执行文件（可选）
- 版本信息：通过 `-ldflags` 注入 git commit hash

### 部署方式

**1. 本地运行（推荐）**
```bash
# 下载可执行文件
wget https://github.com/user/air-link/releases/download/v1.0.0/airlink-linux-amd64

# 添加执行权限
chmod +x airlink-linux-amd64

# 运行
./airlink-linux-amd64
```

**2. Docker 部署**
```bash
docker run -p 8080:8080 -v ./airlink.yaml:/app/airlink.yaml airlink:latest
```

**3. 系统服务（可选）**
```bash
# systemd (Linux)
sudo systemctl enable airlink
sudo systemctl start airlink
```

## 性能指标

### 目标性能

- **连接建立时间**：< 3 秒（局域网）、< 10 秒（公网）
- **文件传输速度**：接近网络带宽（局域网 100MB/s+，公网受限于上行带宽）
- **并发支持**：单个房间最多 10 个设备，服务器支持 100+ 个房间
- **内存占用**：Go 服务器 < 50MB，前端 < 100MB

### 优化策略

- **分片传输**：16KB 分片，减少内存占用
- **流控**：监控 DataChannel `bufferedAmount`，避免发送过快
- **并发控制**：限制同时传输文件数，避免带宽竞争
- **连接复用**：Mesh 拓扑，设备间直连，减少中继

## 已知限制

1. **浏览器兼容性**：需要支持 WebRTC 的现代浏览器（Chrome 56+, Firefox 52+, Safari 11+）
2. **NAT 穿透**：极少数复杂 NAT 环境下可能无法建立 P2P 连接（需要 TURN 服务器）
3. **文件大小**：理论上无限制，但超大文件（10GB+）可能因浏览器内存限制导致性能下降
4. **断点续传**：仅支持会话内续传，刷新页面后需重新传输
5. **移动端**：浏览器后台可能导致连接断开，需要用户保持前台运行

## 后续增强方向

### 短期（MVP 后）
- [ ] IndexedDB 存储传输状态，实现跨会话断点续传
- [ ] 托盘 GUI，一键启动，显示状态
- [ ] 移动端适配优化（PWA，Service Worker）
- [ ] 文件预览优化（更多文件类型支持）

### 中期
- [ ] 多语言支持（i18n）
- [ ] 自定义主题
- [ ] 传输历史记录
- [ ] 二维码扫描加入房间（移动端）

### 长期
- [ ] 端到端加密（应用层，可选）
- [ ] 语音/视频通话支持
- [ ] 屏幕共享
- [ ] 协同编辑（文本文档）

## Git 提交规范

使用 Conventional Commits 格式：

```
<type>(<scope>): <subject>

类型（type）：
- feat: 新功能
- fix: 修复 bug
- docs: 文档更新
- style: 代码格式调整（不影响功能）
- refactor: 重构
- perf: 性能优化
- test: 测试相关
- chore: 构建/工具链相关

示例：
feat(signaling): 实现 WebSocket 信令服务
fix(webrtc): 修复 ICE 候选收集超时问题
docs(readme): 更新部署文档
```

**提交要求：**
- 不添加 Co-Authored-By
- 不在 commit message 中列明细
- 合理的提交粒度（一个逻辑功能一个 commit）
- 中文描述，清晰简洁

## 参考资料

- [WebRTC API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [RFC 8445 - ICE](https://datatracker.ietf.org/doc/html/rfc8445)
- [RFC 6762 - mDNS](https://datatracker.ietf.org/doc/html/rfc6762)
- [Conventional Commits](https://www.conventionalcommits.org/)
