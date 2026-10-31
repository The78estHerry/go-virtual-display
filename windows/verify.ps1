# 檢查虛擬螢幕：裝置狀態、系統讀到的 EDID 是否與 edid\edid_1920x1440.bin 相同、目前的螢幕解析度。
# 不需要系統管理員權限。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$ok = $true

$dev = Get-VddDevice
if (-not $dev) {
    Write-Host '[X] 找不到 Virtual Display Driver 裝置' -ForegroundColor Red
    exit 1
}
Write-Host "[i] 裝置：$($dev.FriendlyName)（$($dev.InstanceId)），狀態 $($dev.Status)"
if ($dev.Status -ne 'OK') {
    Write-Host '[!] 裝置不是啟用狀態（目前是拔掉），以下檢查可能找不到螢幕' -ForegroundColor Yellow
}

$want = [IO.File]::ReadAllBytes($EdidFile)
$found = $false
foreach ($key in Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Enum\DISPLAY' -ErrorAction SilentlyContinue) {
    foreach ($inst in Get-ChildItem $key.PSPath) {
        $edid = (Get-ItemProperty (Join-Path $inst.PSPath 'Device Parameters') -Name EDID -ErrorAction SilentlyContinue).EDID
        if ($edid -and $edid.Length -ge 128 -and [Convert]::ToBase64String($edid[0..127]) -eq [Convert]::ToBase64String($want)) {
            Write-Host "[O] 系統記錄的 EDID 與 edid_1920x1440.bin 相同（DISPLAY\$($key.PSChildName)\$($inst.PSChildName)）" -ForegroundColor Green
            $found = $true
        }
    }
}
if (-not $found) {
    Write-Host '[X] 登錄檔中找不到相同的 EDID' -ForegroundColor Red
    $ok = $false
}

Add-Type -AssemblyName System.Windows.Forms
$screens = [System.Windows.Forms.Screen]::AllScreens
foreach ($s in $screens) {
    Write-Host ("[i] 螢幕 {0}：{1}x{2}{3}" -f $s.DeviceName, $s.Bounds.Width, $s.Bounds.Height, $(if ($s.Primary) { '（主螢幕）' } else { '' }))
}
if (-not ($screens | Where-Object { $_.Bounds.Width -eq 1920 -and $_.Bounds.Height -eq 1440 })) {
    Write-Host '[!] 沒有 1920x1440 的螢幕（若有開啟顯示縮放，這裡顯示的是縮放後的大小）' -ForegroundColor Yellow
}

$tasks = Get-ScheduledTask -TaskPath $TaskPath -ErrorAction SilentlyContinue
Write-Host "[i] 排程工作：$(($tasks | ForEach-Object TaskName) -join '、')"
if (-not $ok) { exit 1 }
