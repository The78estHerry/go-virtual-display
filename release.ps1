# 產生 Windows 版 Release 安裝包 dist\release\go-virtual-display-<版本>-windows.zip。
# 只放安裝需要的檔案：一鍵安裝／移除、PowerShell 腳本、VDD 設定、EDID、編好的開關程式、使用說明。
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
Copy-Item edid\edid_1920x1440.bin, dist\windows\vdswitch.exe $stage\scripts

$readme = @"
go-virtual-display $Version（Windows）

讓電腦在沒有接實體螢幕時，以為接著一台 1920x1440 的螢幕（使用自訂 EDID），
並附系統列圖示與桌面小工具的一鍵拔插開關。

安裝：點兩下 install.cmd，UAC 詢問時按「是」。
      會自動下載已簽章的 Virtual Display Driver $((Select-String -Path windows\scripts\common.ps1 -Pattern "VddVersion\s*=\s*'(.+)'").Matches[0].Groups[1].Value) 並驗證 SHA-256。
移除：在系統列圖示按右鍵選「解除安裝…」，或到「設定 > 應用程式」移除「虛擬螢幕開關」。
      也可以點兩下 uninstall.cmd。安裝後就算刪掉這個資料夾，前兩種方式仍然可以移除。

需求：Windows 10／11 x64、Microsoft Visual C++ Redistributable。
有裝 Synergy 的話，每次拔插後會自動重新啟動它。

scripts 資料夾是安裝用的檔案，不需要直接開啟。
原始碼與完整說明：https://gitea.littleherry.work/The78estWorld/go-virtual-display
"@
# 用 UTF-8 with BOM 和 CRLF，記事本才會正確顯示中文
[IO.File]::WriteAllText((Join-Path $stage '說明.txt'), ($readme -replace "`r?`n", "`r`n"), (New-Object Text.UTF8Encoding $true))

$zip = Join-Path 'dist\release' "$name.zip"
Remove-Item $zip -Force -ErrorAction SilentlyContinue
# 不用 Compress-Archive：Windows PowerShell 5.1 版會把路徑分隔寫成 \
Add-Type -AssemblyName System.IO.Compression.FileSystem
[IO.Compression.ZipFile]::CreateFromDirectory((Resolve-Path $stage).Path, (Join-Path (Resolve-Path 'dist\release').Path "$name.zip"), 'Optimal', $true)
Write-Host "完成：$zip"
Get-ChildItem $stage -Recurse -File | ForEach-Object { '  ' + $_.FullName.Substring((Resolve-Path $stage).Path.Length + 1) }
