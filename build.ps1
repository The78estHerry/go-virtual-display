# 編譯 Windows 版執行檔到 dist\windows\，並重新產生、檢查 EDID。Linux 版請在 Linux 上執行 make linux。
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

go test ./...
if ($LASTEXITCODE -ne 0) { throw '測試失敗' }
New-Item -ItemType Directory -Force dist\windows | Out-Null
go build -trimpath -ldflags '-H=windowsgui -s -w' -o dist\windows\vdswitch.exe .\cmd\vdswitch
if ($LASTEXITCODE -ne 0) { throw 'vdswitch 編譯失敗' }
go build -trimpath -ldflags '-s -w' -o dist\windows\make-edid.exe .\cmd\make-edid
if ($LASTEXITCODE -ne 0) { throw 'make-edid 編譯失敗' }
.\dist\windows\make-edid.exe -w 1920 -h 1080 -o edid\edid_1920x1080.bin  # Windows 安裝時的預設
.\dist\windows\make-edid.exe -o edid\edid_1920x1440.bin  # Linux 版仍使用
Write-Host '完成：dist\windows\vdswitch.exe、dist\windows\make-edid.exe'
