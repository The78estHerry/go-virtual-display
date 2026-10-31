# 在 Docker 的 ubuntu:22.04 容器裡模擬測試 Linux 腳本（不需要 Linux 電腦，也不會動到本機）。
# 系統路徑（/sys、/lib/firmware、/etc 等）會換成容器裡的假目錄，所以測不到真正的 DRM、debugfs、polkit。
#   simulate.sh   安裝（含從 0.3.0 升級）、重新安裝、小幫手 edid、verify、移除、安裝包配置
#   reinstall.sh  已安裝時先提醒：以一般使用者＋真的 sudo、用 script 提供終端機
#
# 用法：powershell -ExecutionPolicy Bypass -File .\linux\tests\run.ps1
# 需要 Docker Desktop（Linux 容器）和 Go（產生測試用的 EDID）。
$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
Set-Location $repo

$data = Join-Path $repo 'dist\linux-test'
New-Item -ItemType Directory -Force $data | Out-Null
go run ./cmd/make-edid -w 2560 -h 1440 -hz 60 -o (Join-Path $data 'e2560.bin') | Out-Null
if ($LASTEXITCODE -ne 0) { throw '產生測試用 EDID 失敗' }

$failed = @()
foreach ($t in 'simulate.sh', 'reinstall.sh') {
    Write-Host "==> $t"
    docker run --rm -v "${repo}:/src:ro" -v "${data}:/data:ro" ubuntu:22.04 bash "/src/linux/tests/$t"
    if ($LASTEXITCODE -ne 0) { $failed += $t }
}
Remove-Item $data -Recurse -Force
if ($failed) { throw "失敗：$($failed -join ', ')" }
Write-Host '全部通過'
