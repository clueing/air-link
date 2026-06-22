# AirLink Logo 生成脚本
# 使用此脚本生成 favicon.ico 和 logo.png

Write-Host "AirLink Logo 生成工具" -ForegroundColor Cyan
Write-Host "=====================`n"

$webDir = "internal/server/web"
$svgLogo = "$webDir/logo.svg"
$svgFavicon = "$webDir/favicon.svg"

# 检查 SVG 文件是否存在
if (-not (Test-Path $svgLogo) -or -not (Test-Path $svgFavicon)) {
    Write-Host "错误: SVG 文件不存在" -ForegroundColor Red
    exit 1
}

Write-Host "方案 1: 使用在线工具 (推荐)" -ForegroundColor Green
Write-Host "----------------------------------------"
Write-Host "1. 生成 PNG (logo.png, 512x512):"
Write-Host "   访问: https://cloudconvert.com/svg-to-png"
Write-Host "   上传: $svgLogo"
Write-Host "   设置: 宽度=512, 高度=512, 保持透明"
Write-Host "   下载: 保存为 $webDir/logo.png"
Write-Host ""
Write-Host "2. 生成 ICO (favicon.ico, 32x32):"
Write-Host "   访问: https://convertio.co/svg-ico/"
Write-Host "   上传: $svgFavicon"
Write-Host "   设置: 32x32 像素"
Write-Host "   下载: 保存为 $webDir/favicon.ico"
Write-Host ""

Write-Host "`n方案 2: 使用 ImageMagick (本地)" -ForegroundColor Green
Write-Host "----------------------------------------"

# 检查 ImageMagick 是否安装
$magick = Get-Command magick -ErrorAction SilentlyContinue
if ($magick) {
    Write-Host "检测到 ImageMagick，可以直接生成..." -ForegroundColor Yellow
    Write-Host ""

    $generate = Read-Host "是否立即生成? (y/n)"
    if ($generate -eq 'y') {
        Write-Host "生成 logo.png (512x512)..." -ForegroundColor Cyan
        & magick convert -background none -size 512x512 $svgLogo "$webDir/logo.png"

        Write-Host "生成 favicon.ico (32x32)..." -ForegroundColor Cyan
        & magick convert -background none -size 32x32 $svgFavicon "$webDir/favicon.ico"

        Write-Host "`n✅ 生成完成!" -ForegroundColor Green
        Write-Host "   - $webDir/logo.png"
        Write-Host "   - $webDir/favicon.ico"
    }
} else {
    Write-Host "ImageMagick 未安装" -ForegroundColor Yellow
    Write-Host "安装方法:"
    Write-Host "  winget install ImageMagick.ImageMagick"
    Write-Host "  或访问: https://imagemagick.org/script/download.php"
}

Write-Host "`n方案 3: 使用 Inkscape (本地)" -ForegroundColor Green
Write-Host "----------------------------------------"
$inkscape = Get-Command inkscape -ErrorAction SilentlyContinue
if ($inkscape) {
    Write-Host "检测到 Inkscape" -ForegroundColor Yellow
    Write-Host ""

    $generate = Read-Host "使用 Inkscape 生成? (y/n)"
    if ($generate -eq 'y') {
        Write-Host "生成 logo.png (512x512)..." -ForegroundColor Cyan
        & inkscape $svgLogo --export-type=png --export-filename="$webDir/logo.png" --export-width=512 --export-height=512

        Write-Host "生成 favicon 临时 PNG..." -ForegroundColor Cyan
        & inkscape $svgFavicon --export-type=png --export-filename="$webDir/favicon-temp.png" --export-width=32 --export-height=32

        if (Test-Path "$webDir/favicon-temp.png") {
            Write-Host "转换为 ICO..." -ForegroundColor Cyan
            if ($magick) {
                & magick convert "$webDir/favicon-temp.png" "$webDir/favicon.ico"
                Remove-Item "$webDir/favicon-temp.png"
            } else {
                Write-Host "需要 ImageMagick 来转换为 ICO 格式" -ForegroundColor Yellow
                Write-Host "临时文件保存在: $webDir/favicon-temp.png"
            }
        }

        Write-Host "`n✅ 生成完成!" -ForegroundColor Green
    }
} else {
    Write-Host "Inkscape 未安装" -ForegroundColor Yellow
    Write-Host "安装方法:"
    Write-Host "  winget install Inkscape.Inkscape"
    Write-Host "  或访问: https://inkscape.org/release/"
}

Write-Host "`n方案 4: 在线批量转换 (最简单)" -ForegroundColor Green
Write-Host "----------------------------------------"
Write-Host "推荐网站:"
Write-Host "  - https://www.aconvert.com/cn/icon/svg-to-ico/"
Write-Host "  - https://favicon.io/favicon-converter/"
Write-Host "  - https://realfavicongenerator.net/"
Write-Host ""

Write-Host "`n完成后，记得更新 HTML 中的引用:" -ForegroundColor Cyan
Write-Host '  <link rel="icon" type="image/x-icon" href="/favicon.ico" />'
Write-Host '  <link rel="icon" type="image/png" sizes="512x512" href="/logo.png" />'
