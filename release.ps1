# 產生 Windows 版 Release 安裝包 dist\release\go-virtual-display-<版本>-windows.zip。
# 只放安裝需要的檔案：一鍵安裝／移除、PowerShell 腳本、VDD 設定、EDID、編好的開關程式、使用說明，
# 以及 Virtual Display Driver 需要的檔案（scripts\vdd\，可離線安裝；雜湊以 common.ps1 的 $VddFilesSha256 為準）。
# 版本號以 windows\scripts\common.ps1 的 $AppVersion 為準（「設定 > 應用程式」顯示的也是它）。
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
. .\windows\scripts\common.ps1
$Version = $AppVersion

& .\build.ps1

$name = "go-virtual-display-$Version-windows"
$stage = Join-Path 'dist\release' $name
Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $stage 'scripts') | Out-Null

Copy-Item windows\install.cmd, windows\uninstall.cmd $stage
Copy-Item windows\scripts\*.ps1, windows\scripts\vdd_settings.xml $stage\scripts
Copy-Item edid\edid_1920x1080.bin, dist\windows\vdswitch.exe $stage\scripts

# 內附驅動：從驗證過的 zip 只取需要的檔案，內容不改。每個檔案都要符合 repo 裡的雜湊，否則不打包
$vddSrc = & .\windows\scripts\download-vdd.ps1
$vddDst = Join-Path $stage 'scripts\vdd'
foreach ($f in $VddFilesSha256.Keys) {
    $to = Join-Path $vddDst $f
    New-Item -ItemType Directory -Force (Split-Path $to) | Out-Null
    Copy-Item (Join-Path $vddSrc $f) $to
}
if (-not (Test-VddFiles $vddDst)) { throw 'Bundled driver files do not match $VddFilesSha256' }
Copy-Item windows\scripts\THIRD-PARTY-NOTICES.txt $vddDst

$readme = @"
go-virtual-display $Version (Windows)

[English]

Makes Windows believe a monitor with a custom EDID is connected, even
when no physical monitor is attached. Includes a one-click plug/unplug switch
(tray icon and desktop widget).

Install:   Double-click install.cmd and click "Yes" in the UAC prompt.
           You can choose English or Traditional Chinese during installation.
           The signed Virtual Display Driver $VddVersion is included (its
           SHA-256 is verified), so no internet connection is needed.
           Its licenses are in scripts\vdd\THIRD-PARTY-NOTICES.txt.
Upgrade:   Just run the new install.cmd. If a version is already installed,
           it shows both versions and asks before uninstalling the old one.
           Your resolutions, language and "Start at sign-in" are kept.
Uninstall: Right-click the tray icon and choose "Uninstall...", or remove
           "Virtual Display Switch" in Settings > Apps. You can also
           double-click uninstall.cmd. The first two still work after this
           folder has been deleted. Uninstalling also deletes your
           resolutions and language choice.

Resolution: The default is 1920x1080. Right-click the tray icon and choose
           "Resolution" to switch, or "Manage resolutions..." to add (common
           presets or your own width/height/refresh) and delete resolutions.

Requires Windows 10/11 x64 and the Microsoft Visual C++ Redistributable.
Supports Synergy 3: if installed, it is restarted automatically after the
display layout changes, and its status is shown in the tray menu and widget.
The scripts folder contains installer files; you do not need to open it.

[繁體中文]

讓電腦在沒有接實體螢幕時，以為接著一台螢幕（使用自訂 EDID，解析度可自選），
並附系統列圖示與桌面小工具的一鍵拔插開關。

安裝：點兩下 install.cmd，UAC 詢問時按「是」。安裝時可選英文或繁體中文。
      已附已簽章的 Virtual Display Driver $VddVersion（會驗證 SHA-256），不需要網路。
      它的授權聲明在 scripts\vdd\THIRD-PARTY-NOTICES.txt。
升級：直接執行新版的 install.cmd。已經安裝過時，會顯示目前和要安裝的版本，
      確認後先移除舊版再安裝。你新增的解析度、選的語言和「登入時自動啟動」會保留。
移除：在系統列圖示按右鍵選「解除安裝…」，或到「設定 > 應用程式」移除「虛擬螢幕開關」。
      也可以點兩下 uninstall.cmd。安裝後就算刪掉這個資料夾，前兩種方式仍然可以移除。
      移除時會一併刪掉你新增的解析度和選的語言。

解析度：預設 1920x1080。在系統列圖示按右鍵選「解析度」切換，或選「管理解析度…」
        新增（常見解析度或自己輸入寬、高、更新率）和刪除。

需求：Windows 10／11 x64、Microsoft Visual C++ Redistributable。
支援 Synergy 3：有安裝的話，螢幕配置改變後會自動重新啟動它，系統列和小工具會顯示它的狀態。
scripts 資料夾是安裝用的檔案，不需要直接開啟。

Source / 原始碼：https://gitea.littleherry.work/The78estWorld/go-virtual-display
"@
# 檔名一律英文；內容用 UTF-8 with BOM 和 CRLF，記事本才會正確顯示中文
[IO.File]::WriteAllText((Join-Path $stage 'README.txt'), ($readme -replace "`r?`n", "`r`n"), (New-Object Text.UTF8Encoding $true))

$zip = Join-Path 'dist\release' "$name.zip"
Remove-Item $zip -Force -ErrorAction SilentlyContinue
# 不用 Compress-Archive：Windows PowerShell 5.1 版會把路徑分隔寫成 \
Add-Type -AssemblyName System.IO.Compression.FileSystem
[IO.Compression.ZipFile]::CreateFromDirectory((Resolve-Path $stage).Path, (Join-Path (Resolve-Path 'dist\release').Path "$name.zip"), 'Optimal', $true)
Write-Host "Done: $zip"
Get-ChildItem $stage -Recurse -File | ForEach-Object { '  ' + $_.FullName.Substring((Resolve-Path $stage).Path.Length + 1) }
