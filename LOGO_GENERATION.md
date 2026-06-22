# AirLink Logo 生成说明

## 文件说明

- `logo.svg` - 高分辨率 LOGO (200x200)
- `favicon.svg` - Favicon 版本 (100x100)

## 生成 PNG 和 ICO

### 方案 1：在线工具（推荐，最简单）

#### 生成 logo.png (512x512)
1. 访问 [CloudConvert](https://cloudconvert.com/svg-to-png)
2. 上传 `logo.svg`
3. 设置：
   - 宽度：512px
   - 高度：512px
   - ✅ 保持透明背景
4. 下载，保存为 `logo.png`

#### 生成 favicon.ico (32x32)
1. 访问 [Convertio](https://convertio.co/svg-ico/)
2. 上传 `favicon.svg`
3. 设置：32x32 像素
4. 下载，保存为 `favicon.ico`

### 方案 2：使用 PowerShell 脚本

```powershell
# 运行生成脚本
.\generate-logo.ps1
```

脚本会自动检测已安装的工具（ImageMagick、Inkscape）并生成文件。

### 方案 3：手动使用 ImageMagick

```powershell
# 安装 ImageMagick
winget install ImageMagick.ImageMagick

# 生成 logo.png
magick convert -background none -size 512x512 logo.svg logo.png

# 生成 favicon.ico
magick convert -background none -size 32x32 favicon.svg favicon.ico
```

### 方案 4：使用 Inkscape

```powershell
# 安装 Inkscape
winget install Inkscape.Inkscape

# 生成 logo.png
inkscape logo.svg --export-type=png --export-filename=logo.png --export-width=512

# 生成 favicon PNG (需要再转 ICO)
inkscape favicon.svg --export-type=png --export-filename=favicon.png --export-width=32
```

## 文件规格

### logo.png
- **尺寸**: 512x512 像素
- **格式**: PNG
- **背景**: 透明
- **用途**: 
  - 移动端添加到主屏幕图标
  - 社交媒体分享图标
  - PWA 应用图标

### favicon.ico
- **尺寸**: 32x32 像素（多尺寸可选：16x16, 32x32, 48x48）
- **格式**: ICO
- **背景**: 透明
- **用途**: 
  - 浏览器标签页图标
  - 书签图标
  - Windows 任务栏图标

## 更新 HTML

生成文件后，更新 `index.html`：

```html
<head>
  <meta charset="utf-8" />
  <title>AirLink — P2P 文件传输</title>
  
  <!-- Favicon -->
  <link rel="icon" type="image/x-icon" href="/favicon.ico" />
  <link rel="icon" type="image/png" sizes="512x512" href="/logo.png" />
  <link rel="apple-touch-icon" href="/logo.png" />
  
  <!-- PWA Manifest (可选) -->
  <link rel="manifest" href="/manifest.json" />
</head>
```

## PWA Manifest (可选)

如果要支持 PWA，创建 `manifest.json`：

```json
{
  "name": "AirLink",
  "short_name": "AirLink",
  "description": "P2P 文件传输",
  "icons": [
    {
      "src": "/logo.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ],
  "start_url": "/",
  "display": "standalone",
  "theme_color": "#3e63dd",
  "background_color": "#ffffff"
}
```

## 验证

生成后检查：
- ✅ 圆角外围应该是透明的
- ✅ PNG 文件大小应该在 5-20KB
- ✅ ICO 文件大小应该在 1-5KB
- ✅ 在浏览器中显示正常

## 在线工具推荐

1. **Favicon Generator**: https://favicon.io/favicon-converter/
2. **RealFaviconGenerator**: https://realfavicongenerator.net/
3. **CloudConvert**: https://cloudconvert.com/svg-to-png
4. **Convertio**: https://convertio.co/svg-ico/

---

**提示**: 推荐使用在线工具，最简单快捷！
