# 局域网扫描故障排查

## 问题：iOS 设备也无法被扫描到

如果你在不同设备之间也无法扫描到，按以下步骤排查：

## 步骤 1：查看服务器日志

使用 `--debug` 模式运行：

```powershell
go run .\cmd\airlink\main.go --debug
```

点击"扫描局域网"后，查看控制台输出：

### 正常日志示例
```
INFO  收到扫描请求
INFO  开始 mDNS 扫描，服务名: _airlink._tcp
INFO  mDNS 扫描完成，找到 2 个设备
INFO  发现设备: iPhone-Alice (uuid-xxx) at 192.168.1.100:8080
INFO  发现设备: MacBook-Bob (uuid-yyy) at 192.168.1.101:8080
INFO  最终返回 2 个设备
```

### 异常日志示例
```
INFO  收到扫描请求
INFO  开始 mDNS 扫描，服务名: _airlink._tcp
ERROR mDNS 查询失败: permission denied
INFO  mDNS 扫描完成，找到 0 个设备
INFO  mDNS 未发现设备，开始 HTTP 探测
INFO  最终返回 0 个设备
```

## 步骤 2：检查网络连接

### 确认在同一网络

**电脑端：**
```powershell
ipconfig
```

查看 IPv4 地址，例如：`192.168.1.100`

**iOS 端：**
设置 → WiFi → 点击已连接的网络 → 查看 IP 地址

**确认：**
- 两个设备的 IP 地址前三段相同（如 `192.168.1.x`）
- 都连接到同一个 WiFi 名称

### 测试网络连通性

在电脑上 ping iOS 设备的 IP：

```powershell
ping 192.168.1.100
```

如果不通，说明网络隔离了。

## 步骤 3：检查防火墙

### Windows 防火墙

**临时关闭测试：**
```powershell
# 需要管理员权限
Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled False
```

如果关闭后可以扫描到，说明是防火墙问题。

**添加防火墙规则：**
```powershell
# 允许 AirLink 程序
New-NetFirewallRule -DisplayName "AirLink" -Direction Inbound -Program "C:\path\to\airlink.exe" -Action Allow

# 允许 UDP 5353 (mDNS)
New-NetFirewallRule -DisplayName "mDNS" -Direction Inbound -Protocol UDP -LocalPort 5353 -Action Allow

# 允许 TCP 8080 (HTTP)
New-NetFirewallRule -DisplayName "AirLink HTTP" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow
```

### macOS 防火墙

系统偏好设置 → 安全性与隐私 → 防火墙 → 防火墙选项

确保"阻止所有传入连接"未选中。

### iOS

iOS 默认不阻止局域网流量，但公共 WiFi 可能有隔离。

## 步骤 4：检查 WiFi 网络类型

### AP 隔离（Client Isolation）

某些路由器启用了 AP 隔离，导致设备之间无法通信。

**测试方法：**
1. 电脑和 iOS 都连接到路由器
2. 尝试从电脑 ping iOS 的 IP

如果 ping 不通，可能是 AP 隔离。

**解决方案：**
- 登录路由器管理界面
- 关闭"AP 隔离"、"客户端隔离"选项
- 或使用 PIN 码连接（不受影响）

### 公共 WiFi / 访客网络

公共 WiFi 通常启用了设备隔离，无法使用 mDNS。

**解决方案：**
- 使用个人热点
- 或使用 PIN 码连接

## 步骤 5：检查端口占用

mDNS 使用 UDP 5353 端口，可能被其他程序占用。

**Windows：**
```powershell
netstat -ano | findstr 5353
```

**如果看到输出，说明端口被占用。**

可能的占用程序：
- iTunes / Bonjour 服务
- Spotify
- 其他 mDNS 服务

**解决方案：**
- 临时关闭这些程序
- 或重启电脑

## 步骤 6：使用 PIN 码作为备选

如果局域网扫描实在不工作，使用 PIN 码连接：

**优点：**
- 不受防火墙影响
- 不受网络隔离影响
- 可以跨网络使用
- 更安全

**步骤：**
```
1. iOS 设备打开 AirLink
2. 点击"创建房间"
3. 记下 6 位 PIN 码

4. 电脑打开 AirLink
5. 输入 PIN 码
6. 点击"加入"

7. 连接建立，可以传输文件
```

## 常见场景与解决方案

### 场景 1：家庭 WiFi
**问题**：Windows 防火墙阻止  
**解决**：添加防火墙规则（见步骤 3）

### 场景 2：公司 WiFi
**问题**：网络策略禁止 mDNS  
**解决**：使用 PIN 码

### 场景 3：公共 WiFi / 酒店
**问题**：AP 隔离  
**解决**：
- 使用手机热点
- 或使用 PIN 码（如果都能访问互联网）

### 场景 4：iOS 热点共享
**问题**：iOS 设备作为热点时，自己无法扫描到连接的设备  
**解决**：使用 PIN 码

## 调试清单

使用这个清单逐项检查：

- [ ] 两个设备都启动了 AirLink
- [ ] 两个设备连接到同一个 WiFi（IP 前三段相同）
- [ ] 可以互相 ping 通
- [ ] Windows 防火墙已允许 AirLink
- [ ] 路由器未启用 AP 隔离
- [ ] 端口 5353 和 8080 未被占用
- [ ] 服务器日志显示"mDNS 扫描完成"
- [ ] 尝试使用 `--debug` 模式查看详细日志

如果以上都正常但还是扫描不到，**使用 PIN 码连接是最可靠的方案。**

## 技术限制总结

| 限制 | 影响 | 解决方案 |
|------|------|----------|
| 同一台电脑 | ✓ 无法扫描 | 使用 PIN 码 |
| 防火墙 | ✓ 可能阻止 | 添加规则 |
| AP 隔离 | ✓ 无法扫描 | 使用 PIN 码 |
| 公共 WiFi | ✓ 通常被阻止 | 使用 PIN 码 |
| 跨子网 | ✓ 无法扫描 | 使用 PIN 码 |

**结论：PIN 码是最可靠的连接方式！**

## 快速测试

如果你想快速验证 mDNS 是否工作，可以使用系统工具：

### Windows
```powershell
# 安装 Bonjour SDK 后
dns-sd -B _airlink._tcp local
```

### macOS
```bash
dns-sd -B _airlink._tcp local
```

### Linux
```bash
avahi-browse -a
```

如果这些工具都扫描不到，说明是网络环境问题，不是 AirLink 的 bug。

---

**记住：PIN 码连接是主要方式，局域网扫描只是辅助功能！**
