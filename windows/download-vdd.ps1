# 下載固定版本的 Virtual Display Driver 到 downloads\，驗證 SHA-256 後解壓縮。
# 不需要系統管理員權限。已下載且雜湊正確時直接略過。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

$zip = Join-Path $VendorDir "VDD.Control.$VddVersion.zip"
$extracted = Join-Path $VendorDir 'extracted'

New-Item -ItemType Directory -Force $VendorDir | Out-Null
if (-not (Test-Path $zip) -or (Get-FileHash $zip -Algorithm SHA256).Hash -ne $VddSha256) {
    Write-Host "下載 Virtual Display Driver $VddVersion ..."
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $VddUrl -OutFile $zip -UseBasicParsing
}
$hash = (Get-FileHash $zip -Algorithm SHA256).Hash
if ($hash -ne $VddSha256) {
    Remove-Item $zip -Force
    throw "SHA-256 不符（$hash），已刪除下載的檔案。"
}
Write-Host "SHA-256 正確：$hash"

if (-not (Test-Path (Join-Path $extracted 'SignedDrivers'))) {
    Expand-Archive -Path $zip -DestinationPath $extracted -Force
}
Write-Host "已解壓縮到 $extracted"
