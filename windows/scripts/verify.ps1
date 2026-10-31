# 檢查虛擬螢幕：裝置狀態、系統讀到的 EDID 是否與 edid\edid_1920x1440.bin 相同、目前的螢幕解析度。
# 不需要系統管理員權限。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$ok = $true

$dev = Get-VddDevice
if (-not $dev) {
    Write-Host (T '[X] Virtual Display Driver device not found' '[X] 找不到 Virtual Display Driver 裝置') -ForegroundColor Red
    exit 1
}
Write-Host (T "[i] Device: $($dev.FriendlyName) ($($dev.InstanceId)), status $($dev.Status)" "[i] 裝置：$($dev.FriendlyName)（$($dev.InstanceId)），狀態 $($dev.Status)")
if ($dev.Status -ne 'OK') {
    Write-Host (T '[!] The device is not enabled (currently unplugged); the checks below may not find the display' '[!] 裝置不是啟用狀態（目前是拔掉），以下檢查可能找不到螢幕') -ForegroundColor Yellow
}

$want = [IO.File]::ReadAllBytes($EdidFile)
# VDD 會把 byte 8–11（廠商代碼、產品代碼）換成自己的 MTT／0x1337，再重算 byte 127 的 checksum，比對時略過這幾個 byte
$ignored = 8, 9, 10, 11, 127
$found = $false
foreach ($key in Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Enum\DISPLAY' -ErrorAction SilentlyContinue) {
    foreach ($inst in Get-ChildItem $key.PSPath) {
        $edid = (Get-ItemProperty (Join-Path $inst.PSPath 'Device Parameters') -Name EDID -ErrorAction SilentlyContinue).EDID
        if (-not $edid -or $edid.Length -lt 128) { continue }
        $diff = @(0..127 | Where-Object { $edid[$_] -ne $want[$_] })
        $where = "DISPLAY\$($key.PSChildName)\$($inst.PSChildName)"
        if ($diff.Count -eq 0) {
            Write-Host (T "[O] The EDID recorded by Windows is identical to edid_1920x1440.bin ($where)" "[O] 系統記錄的 EDID 與 edid_1920x1440.bin 完全相同（$where）") -ForegroundColor Green
            $found = $true
        } elseif (-not ($diff | Where-Object { $_ -notin $ignored })) {
            Write-Host (T "[O] The EDID recorded by Windows matches edid_1920x1440.bin; only the vendor/product ID was replaced by the driver with $($key.PSChildName) ($where)" "[O] 系統記錄的 EDID 與 edid_1920x1440.bin 相同，只有廠商／產品代碼被驅動換成 $($key.PSChildName)（$where）") -ForegroundColor Green
            $found = $true
        }
    }
}
if (-not $found) {
    if ($dev.Status -eq 'OK') {
        Write-Host (T '[X] No matching EDID found in the registry' '[X] 登錄檔中找不到相同的 EDID') -ForegroundColor Red
        $ok = $false
    } else {
        Write-Host (T '[!] This EDID is not in the registry yet; it appears after the virtual display has been plugged in once' '[!] 登錄檔中還沒有這份 EDID：虛擬螢幕插入過一次之後才會出現') -ForegroundColor Yellow
    }
}

Add-Type -AssemblyName System.Windows.Forms
$screens = [System.Windows.Forms.Screen]::AllScreens
foreach ($s in $screens) {
    Write-Host ((T '[i] Display {0}: {1}x{2}{3}' '[i] 螢幕 {0}：{1}x{2}{3}') -f $s.DeviceName, $s.Bounds.Width, $s.Bounds.Height, $(if ($s.Primary) { T ' (primary)' '（主螢幕）' } else { '' }))
}
if (-not ($screens | Where-Object { $_.Bounds.Width -eq 1920 -and $_.Bounds.Height -eq 1440 })) {
    Write-Host (T '[!] No 1920x1440 display (with display scaling on, the scaled size is shown here)' '[!] 沒有 1920x1440 的螢幕（若有開啟顯示縮放，這裡顯示的是縮放後的大小）') -ForegroundColor Yellow
}

$tasks = Get-ScheduledTask -TaskPath $TaskPath -ErrorAction SilentlyContinue
Write-Host ((T '[i] Scheduled tasks: ' '[i] 排程工作：') + (($tasks | ForEach-Object TaskName) -join ', '))
if (-not $ok) { exit 1 }
