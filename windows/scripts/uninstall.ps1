# 移除虛擬螢幕驅動、排程工作、開關程式與開機啟動項，並還原原本的 VDD 設定檔。
# 需要系統管理員權限。
param(
    # 保留使用者的設定（解析度清單、小工具位置、介面語言、登入時自動啟動）。install.ps1 覆蓋安裝時先移除舊版會用到
    # 自動啟動的 Run 機碼指向 Program Files 裡的 vdswitch.exe，重新安裝後路徑相同，所以可以留著
    [switch]$KeepUserSettings
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
Assert-Admin

Get-Process vdswitch -ErrorAction SilentlyContinue | Stop-Process -Force

if (Get-VddDevice) {
    pnputil /remove-device /deviceid "$VddHardwareId" | Out-Null
    Write-Host (T 'Removed the virtual display device' '已移除虛擬螢幕裝置')
    Restart-DisplayDependentServices
}
Get-WindowsDriver -Online | Where-Object { $_.OriginalFileName -like '*\mttvdd.inf' } | ForEach-Object {
    pnputil /delete-driver $_.Driver /uninstall /force | Out-Null
    Write-Host (T "Deleted driver package $($_.Driver)" "已刪除驅動套件 $($_.Driver)")
}
# 驅動移除後，Windows 仍會留著虛擬螢幕的「已不存在的螢幕」裝置與它的縮放設定
Get-PnpDevice -ErrorAction SilentlyContinue | Where-Object { $_.InstanceId -like "$VddMonitorId\*" } | ForEach-Object {
    pnputil /remove-device "$($_.InstanceId)" | Out-Null
    Write-Host (T "Removed leftover monitor device $($_.InstanceId)" "已移除殘留的螢幕裝置 $($_.InstanceId)")
}
$scaleKey = 'HKLM:\SYSTEM\CurrentControlSet\Control\GraphicsDrivers\ScaleFactors'
Get-ChildItem $scaleKey -ErrorAction SilentlyContinue | Where-Object { $_.PSChildName -like "$(Split-Path $VddMonitorId -Leaf)*" } |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
# VDD 的設定目錄登錄值（驅動套件已刪除，留著沒有用）
if (Test-Path $VddRegKey) {
    Remove-ItemProperty $VddRegKey -Name VDDPATH -ErrorAction SilentlyContinue
    foreach ($k in $VddRegKey, (Split-Path $VddRegKey)) {
        $item = Get-Item $k -ErrorAction SilentlyContinue
        if ($item -and $item.SubKeyCount -eq 0 -and $item.ValueCount -eq 0) { Remove-Item $k -Force }
    }
}

Get-ScheduledTask -TaskPath $TaskPath -ErrorAction SilentlyContinue | Unregister-ScheduledTask -Confirm:$false
try {
    $svc = New-Object -ComObject Schedule.Service
    $svc.Connect()
    $svc.GetFolder('\').DeleteFolder($TaskPath.Trim('\'), 0)  # 刪掉留下的空資料夾
} catch {}
Write-Host (T 'Deleted scheduled tasks' '已刪除排程工作')

if (-not $KeepUserSettings) { Remove-ItemProperty $RunKey -Name $RunValue -ErrorAction SilentlyContinue }
Remove-Item $AppDir -Recurse -Force -ErrorAction SilentlyContinue  # 含 uninstaller\
Remove-Item $UninstallKey -Recurse -Force -ErrorAction SilentlyContinue
$programs = Join-Path $env:ProgramData 'Microsoft\Windows\Start Menu\Programs'
Remove-Item (Join-Path $programs "$AppName.lnk"), (Join-Path $programs "$UnplugShortcutName.lnk"), (Join-Path $programs $LegacyShortcut), (Join-Path $DesktopDir "$AppName.lnk") -Force -ErrorAction SilentlyContinue
if (-not $KeepUserSettings) {
    Remove-Item $LanguageKey -Recurse -Force -ErrorAction SilentlyContinue
    # 使用者的設定：存下的解析度、小工具位置、套用結果（移除以同一個使用者的系統管理員身分執行）
    Remove-Item (Join-Path $env:APPDATA 'go-virtual-display') -Recurse -Force -ErrorAction SilentlyContinue
}
if ($KeepUserSettings) {
    Write-Host (T 'Removed the switch (your settings are kept)' '已移除開關程式（保留你的設定）')
} else {
    Write-Host (T 'Removed the switch and its sign-in startup entry' '已移除開關程式與開機啟動項')
}

foreach ($name in 'vdd_settings.xml', 'user_edid.bin') {
    $path = Join-Path $VddConfigDir $name
    if (Test-Path "$path.go-virtual-display.bak") {
        Move-Item "$path.go-virtual-display.bak" $path -Force
        Write-Host (T "Restored the original $name" "已還原原有的 $name")
    } else {
        Remove-Item $path -Force -ErrorAction SilentlyContinue
    }
}
Remove-Item $InstallMarker -Force -ErrorAction SilentlyContinue
# 開啟 VDD 記錄（vdd_settings.xml 的 logging）時，驅動會在設定目錄留下記錄檔
if (-not (Test-Path (Join-Path $VddConfigDir 'vdd_settings.xml'))) { Remove-Item (Join-Path $VddConfigDir 'Logs') -Recurse -Force -ErrorAction SilentlyContinue }
if ((Test-Path $VddConfigDir) -and -not (Get-ChildItem $VddConfigDir -Force)) {
    Remove-Item $VddConfigDir -Force
}
Write-Host (T 'Done.' '完成。')
