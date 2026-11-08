# 從 assets\icon.svg、unplug.svg 產生 assets\icon.ico、unplug.ico（16～256 px 多種尺寸）。
# 用 Windows 內建的 Microsoft Edge（headless）把 SVG 轉成透明背景的 PNG，再組成 ICO。
# 再用 llvm-mingw 的 windres 編成 cmd\vdswitch\rsrc_windows_amd64.syso，讓 vdswitch.exe 帶有圖示。
# 只有改了 SVG 時才需要執行；產生的 .ico 和 .syso 都會 commit 進 repo，平常編譯不需要這些工具。
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$edge = "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
if (-not (Test-Path $edge)) { throw "找不到 Microsoft Edge：$edge" }
$sizes = 16, 20, 24, 32, 40, 48, 64, 256

# Build-Icon 把 $SvgName 轉成多尺寸的 $IcoName，回傳 256 px PNG 的暫存路徑
function Build-Icon([string]$SvgName, [string]$IcoName) {
    $tmp = Join-Path $env:TEMP "go-virtual-display-icon-$IcoName"
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory $tmp | Out-Null
    $svg = Get-Content (Join-Path $PSScriptRoot $SvgName) -Raw -Encoding UTF8

    $pngs = foreach ($s in $sizes) {
        $html = Join-Path $tmp "$s.html"
        $png = Join-Path $tmp "$s.png"
        $page = "<!doctype html><html><head><style>html,body{margin:0;background:transparent}svg{display:block;width:${s}px;height:${s}px}</style></head><body>$svg</body></html>"
        [IO.File]::WriteAllText($html, $page, (New-Object Text.UTF8Encoding $false))
        $edgeArgs = @('--headless', '--disable-gpu', '--hide-scrollbars', '--default-background-color=00000000',
            "--window-size=$s,$s", "--screenshot=$png", "--user-data-dir=$tmp\profile", ([Uri]$html).AbsoluteUri)
        Start-Process $edge -ArgumentList $edgeArgs -Wait -WindowStyle Hidden
        if (-not (Test-Path $png)) { throw "Edge 沒有產生 $png" }
        $png
    }

    # ICO：檔頭＋每個尺寸一筆目錄，影像直接放 PNG（Windows Vista 以後支援）
    $data = $pngs | ForEach-Object { , [IO.File]::ReadAllBytes($_) }
    $ms = New-Object IO.MemoryStream
    $w = New-Object IO.BinaryWriter $ms
    $w.Write([uint16]0); $w.Write([uint16]1); $w.Write([uint16]$sizes.Count)
    $offset = 6 + 16 * $sizes.Count
    for ($i = 0; $i -lt $sizes.Count; $i++) {
        $s = $sizes[$i]
        $dim = if ($s -ge 256) { 0 } else { $s }  # 256 寫成 0
        $w.Write([byte]$dim); $w.Write([byte]$dim); $w.Write([byte]0); $w.Write([byte]0)
        $w.Write([uint16]1); $w.Write([uint16]32)
        $w.Write([uint32]$data[$i].Length); $w.Write([uint32]$offset)
        $offset += $data[$i].Length
    }
    foreach ($d in $data) { $w.Write($d) }
    $w.Flush()
    [IO.File]::WriteAllBytes((Join-Path $PSScriptRoot $IcoName), $ms.ToArray())
    Write-Host "完成：assets\$IcoName（$($sizes -join '、') px）"
    $tmp
}

# icon.svg：程式圖示（資源 ID 1）。unplug.svg：「拔掉虛擬螢幕」捷徑的圖示（資源 ID 2，0.4.4 起）
$tmp = Build-Icon 'icon.svg' 'icon.ico'
Copy-Item (Join-Path $tmp '256.png') (Join-Path $PSScriptRoot 'icon-256.png') -Force
Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
Write-Host '完成：assets\icon-256.png'
$tmp = Build-Icon 'unplug.svg' 'unplug.ico'
Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue

$windres = Get-Command x86_64-w64-mingw32-windres -ErrorAction SilentlyContinue
if (-not $windres) { throw '找不到 x86_64-w64-mingw32-windres（llvm-mingw），無法產生 .syso' }
Push-Location (Join-Path $PSScriptRoot '..\cmd\vdswitch')
& $windres.Source -O coff -i vdswitch.rc -o rsrc_windows_amd64.syso
$code = $LASTEXITCODE
Pop-Location
if ($code -ne 0) { throw 'windres 失敗' }
Write-Host '完成：cmd\vdswitch\rsrc_windows_amd64.syso'
