# 移除虛擬螢幕驅動、排程工作、開關程式與開機啟動項，並還原原本的 VDD 設定檔。
# 需要系統管理員權限。
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
Assert-Admin

Get-Process vdswitch -ErrorAction SilentlyContinue | Stop-Process -Force

if (Get-VddDevice) {
    pnputil /remove-device /deviceid "$VddHardwareId" | Out-Null
    Write-Host '已移除虛擬螢幕裝置'
}
Get-WindowsDriver -Online | Where-Object { $_.OriginalFileName -like '*\mttvdd.inf' } | ForEach-Object {
    pnputil /delete-driver $_.Driver /uninstall /force | Out-Null
    Write-Host "已刪除驅動套件 $($_.Driver)"
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
if ((Test-Path $VddConfigDir) -and -not (Get-ChildItem $VddConfigDir -Force)) {
    Remove-Item $VddConfigDir -Force
}
Write-Host '完成。'
