# 取得 Virtual Display Driver，最後輸出驅動檔案的根目錄（底下是 SignedDrivers\、Dependencies\）。
# 1. Release 安裝包內附的（scripts\vdd\），雜湊都對就直接用：不需要網路，下載點消失也能安裝
# 2. 沒有內附（原始碼）或檔案不對時，下載固定版本到 downloads\，驗證 SHA-256 後解壓縮。已下載且雜湊正確時略過下載
# 不需要系統管理員權限。失敗時丟出例外，呼叫端（install.ps1）要在改動系統之前呼叫。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

if ($VddBundleDir -and (Test-Path $VddBundleDir)) {
    if (Test-VddFiles $VddBundleDir) {
        Write-Host (T "Using the bundled Virtual Display Driver $VddVersion (SHA-256 verified)" "使用安裝包內附的 Virtual Display Driver $VddVersion（SHA-256 正確）")
        return $VddBundleDir
    }
    Write-Host (T 'Warning: the bundled driver files are missing or damaged; downloading instead.' '注意：安裝包內附的驅動檔案不完整或已損壞，改成下載。') -ForegroundColor Yellow
}

$zip = Join-Path $VendorDir "VDD.Control.$VddVersion.zip"
$extracted = Join-Path $VendorDir 'extracted'

New-Item -ItemType Directory -Force $VendorDir | Out-Null
if (-not (Test-Path $zip) -or (Get-FileHash $zip -Algorithm SHA256).Hash -ne $VddSha256) {
    Write-Host (T "Downloading Virtual Display Driver $VddVersion ..." "下載 Virtual Display Driver $VddVersion ...")
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $ProgressPreference = 'SilentlyContinue'  # Windows PowerShell 5.1 顯示進度條時下載會慢上數十倍
    try {
        Invoke-WebRequest -Uri $VddUrl -OutFile $zip -UseBasicParsing
    } catch {
        Remove-Item $zip -Force -ErrorAction SilentlyContinue
        throw ((T "Could not download the driver (check the network connection): " "無法下載驅動（請檢查網路連線）：") + $_.Exception.Message)
    }
}
$hash = (Get-FileHash $zip -Algorithm SHA256).Hash
if ($hash -ne $VddSha256) {
    Remove-Item $zip -Force
    throw (T "SHA-256 mismatch ($hash); the download was deleted." "SHA-256 不符（$hash），已刪除下載的檔案。")
}
Write-Host (T "SHA-256 OK: $hash" "SHA-256 正確：$hash")

if (-not (Test-VddFiles $extracted)) {
    Remove-Item $extracted -Recurse -Force -ErrorAction SilentlyContinue
    Expand-Archive -Path $zip -DestinationPath $extracted -Force
    if (-not (Test-VddFiles $extracted)) { throw (T 'The downloaded driver files do not match the expected SHA-256.' '下載的驅動檔案和預期的 SHA-256 不符。') }
}
Write-Host (T "Extracted to $extracted" "已解壓縮到 $extracted")
return $extracted
