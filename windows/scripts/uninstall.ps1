# 移除虛擬螢幕驅動、排程工作、開關程式與開機啟動項，並還原原本的 VDD 設定檔。
# 需要系統管理員權限。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
Assert-Admin

Get-Process vdswitch -ErrorAction SilentlyContinue | Stop-Process -Force

if (Get-VddDevice) {
    pnputil /remove-device /deviceid "$VddHardwareId" | Out-Null
    Write-Host '已移除虛擬螢幕裝置'
    Restart-DisplayDependentServices
}
Get-WindowsDriver -Online | Where-Object { $_.OriginalFileName -like '*\mttvdd.inf' } | ForEach-Object {
    pnputil /delete-driver $_.Driver /uninstall /force | Out-Null
    Write-Host "已刪除驅動套件 $($_.Driver)"
}
# 驅動移除後，Windows 仍會留著虛擬螢幕的「已不存在的螢幕」裝置與它的縮放設定
Get-PnpDevice -ErrorAction SilentlyContinue | Where-Object { $_.InstanceId -like "$VddMonitorId\*" } | ForEach-Object {
    pnputil /remove-device "$($_.InstanceId)" | Out-Null
    Write-Host "已移除殘留的螢幕裝置 $($_.InstanceId)"
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
Write-Host '已刪除排程工作'

Remove-ItemProperty $RunKey -Name $RunValue -ErrorAction SilentlyContinue
Remove-Item $AppDir -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $env:ProgramData 'Microsoft\Windows\Start Menu\Programs\虛擬螢幕開關.lnk') -Force -ErrorAction SilentlyContinue
Write-Host '已移除開關程式與開機啟動項'

foreach ($name in 'vdd_settings.xml', 'user_edid.bin') {
    $path = Join-Path $VddConfigDir $name
    if (Test-Path "$path.go-virtual-display.bak") {
        Move-Item "$path.go-virtual-display.bak" $path -Force
        Write-Host "已還原原有的 $name"
    } else {
        Remove-Item $path -Force -ErrorAction SilentlyContinue
    }
}
Remove-Item $InstallMarker -Force -ErrorAction SilentlyContinue
if ((Test-Path $VddConfigDir) -and -not (Get-ChildItem $VddConfigDir -Force)) {
    Remove-Item $VddConfigDir -Force
}
Write-Host '完成。'
