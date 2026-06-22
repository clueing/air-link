# AirLink Nginx 配置优化建议

## 当前问题

你的配置中 WebSocket 升级头设置有误：
```nginx
proxy_set_header Connection upgrade;      # ❌ 错误：值应该是 "upgrade"
proxy_set_header Upgrade $http_upgrade;   # ✅ 正确
```

## 修复方案

### 完整的优化配置

```nginx
server {
    listen 80;
    listen 443 ssl http2;
    server_name send.lutao.de;
    
    # SSL 证书
    ssl_certificate /www/sites/send.lutao.de/ssl/fullchain.pem;
    ssl_certificate_key /www/sites/send.lutao.de/ssl/privkey.pem;
    ssl_protocols TLSv1.3 TLSv1.2;
    ssl_ciphers ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # 日志
    access_log /www/sites/send.lutao.de/log/access.log main;
    error_log /www/sites/send.lutao.de/log/error.log;
    
    # HTTP 强制跳转 HTTPS
    if ($scheme = http) {
        return 301 https://$host$request_uri;
    }
    
    # Let's Encrypt 验证
    location ^~ /.well-knownacme-challenge {
        allow all;
        root /usr/share/nginx/html;
    }
    
    # 隐藏敏感文件
    location ~ ^/(\.user.ini|\.htaccess|\.git|\.env|\.svn|\.project|LICENSE|README.md) {
        return 404;
    }
    
    # 主要代理配置
    location / {
        # 基本代理头
        proxy_pass http://127.0.0.1:8012;
        proxy_http_version 1.1;
        
        # 请求头
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $server_name;
        
        # WebSocket 升级头（关键！）
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # 超时设置（WebSocket 需要长连接）
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
        
        # 缓冲设置
        proxy_buffering off;
    }
    
    # 安全头
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
}
```

## 关键修改点

### 1. ✅ 修复 WebSocket 升级头

**修改前（错误）：**
```nginx
proxy_set_header Connection upgrade;      # ❌ 缺少引号
proxy_set_header Upgrade $http_upgrade;
```

**修改后（正确）：**
```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";    # ✅ 加引号
```

### 2. ✅ 添加超时设置

WebSocket 是长连接，需要增加超时时间：

```nginx
proxy_connect_timeout 60s;
proxy_send_timeout 300s;    # 5 分钟（PIN 码有效期）
proxy_read_timeout 300s;
```

### 3. ✅ 禁用代理缓冲

WebSocket 需要实时传输：

```nginx
proxy_buffering off;
```

### 4. ✅ 移动 http2 位置

```nginx
# 错误位置
http2 on;

# 正确位置
listen 443 ssl http2;  # 直接在 listen 指令中
```

## 应用配置

### 方法 1：直接修改配置文件

```bash
# 编辑配置
nano /www/sites/send.lutao.de/nginx.conf

# 测试配置
nginx -t

# 重载配置
nginx -s reload
```

### 方法 2：使用面板（宝塔/1Panel）

1. 进入站点设置
2. 点击"配置文件"
3. 替换为上面的优化配置
4. 保存并重启 Nginx

## 验证 WebSocket

配置完成后，访问：
```
https://send.lutao.de
```

打开浏览器控制台，应该看到：
```
✅ WebSocket 已连接
```

如果还是报错，检查：
1. AirLink 后端是否正常运行（`ps aux | grep airlink`）
2. 端口 8012 是否正确
3. 防火墙是否开放

## 后端启动命令

确保 AirLink 后端正确启动：

```bash
# 使用 systemd（推荐）
sudo systemctl start airlink
sudo systemctl enable airlink

# 或直接运行
./airlink --port 8012 --no-browser

# 检查是否运行
curl http://127.0.0.1:8012
```

## 故障排查

### 1. 检查 WebSocket 连接

```bash
# 查看 Nginx 错误日志
tail -f /www/sites/send.lutao.de/log/error.log

# 查看 AirLink 后端日志
journalctl -u airlink -f
```

### 2. 测试本地连接

```bash
# 测试后端是否运行
curl http://127.0.0.1:8012

# 应该返回 HTML 内容
```

### 3. 测试 WebSocket

使用浏览器控制台：
```javascript
const ws = new WebSocket('wss://send.lutao.de/ws');
ws.onopen = () => console.log('连接成功');
ws.onerror = (e) => console.error('连接失败', e);
```

---

**重要提示**：修改后记得重启 Nginx！
