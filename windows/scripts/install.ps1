# 安裝虛擬螢幕驅動、套用 1920x1440 設定與自訂 EDID、建立免 UAC 的拔插排程工作、安裝開關程式。
# 需要系統管理員權限。可重複執行（會更新設定並重新載入驅動）。
param(
    # 排程工作要以哪個使用者執行；預設為目前登入桌面的使用者
    [string]$User = (Get-CimInstance Win32_ComputerSystem).UserName,
    # 安裝完成後執行檢查，並以一般權限開啟開關程式（install.cmd 使用）
    [switch]$Launch
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
Assert-Admin

# 1. 下載驅動
& (Join-Path $PSScriptRoot 'download-vdd.ps1')
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'ARM64' } else { 'x86' }  # x86 資料夾內是 amd64 驅動
$driverDir = Join-Path $VendorDir "extracted\SignedDrivers\$arch\VDD"
$inf = Join-Path $driverDir 'MttVDD.inf'
$devcon = Join-Path $VendorDir 'extracted\Dependencies\devcon.exe'

# 2. 設定檔與 EDID（先備份原有檔案）
#    標記檔存在表示目前的檔案是本專案放的，重複安裝時不能再備份，否則移除時會「還原」成本專案的檔案
New-Item -ItemType Directory -Force $VddConfigDir | Out-Null
if (-not (Test-Path $InstallMarker)) {
    foreach ($name in 'vdd_settings.xml', 'user_edid.bin') {
        $path = Join-Path $VddConfigDir $name
        if (Test-Path $path) {
            Copy-Item $path "$path.go-virtual-display.bak" -Force
            Write-Host "已備份原有的 $name"
        }
    }
    Set-Content $InstallMarker 'go-virtual-display'
}
Copy-Item (Join-Path $PSScriptRoot 'vdd_settings.xml') (Join-Path $VddConfigDir 'vdd_settings.xml') -Force
Copy-Item $EdidFile (Join-Path $VddConfigDir 'user_edid.bin') -Force
New-Item -Force $VddRegKey | Out-Null
Set-ItemProperty $VddRegKey -Name VDDPATH -Value $VddConfigDir
Write-Host "已套用設定與 EDID 到 $VddConfigDir"

# 3. 安裝驅動，或在已安裝時重新載入
if (Get-VddDevice) {
    Write-Host '驅動已安裝，更新驅動套件並重新載入裝置 ...'
    pnputil /add-driver "$inf" /install | Out-Null
    pnputil /restart-device /deviceid "$VddHardwareId" | Out-Null
} else {
    Write-Host '安裝 Virtual Display Driver ...'
    & $devcon install "$inf" "$VddHardwareId"
    if ($LASTEXITCODE -ne 0) { throw "devcon 安裝失敗（代碼 $LASTEXITCODE）" }
}

# 4. 拔插用排程工作：以最高權限執行，之後按開關不會跳 UAC
#    有裝 Synergy 等服務時，再加一個動作：等螢幕建立好後重新啟動它（排程工作的多個動作會依序執行）
$services = @($RestartServices | Where-Object { Get-Service -Name $_ -ErrorAction SilentlyContinue })
foreach ($t in @(@{ Name = 'Plug'; Verb = '/enable-device' }, @{ Name = 'Unplug'; Verb = '/disable-device' })) {
    $action = @(New-ScheduledTaskAction -Execute "$env:SystemRoot\System32\conhost.exe" `
        -Argument "--headless $env:SystemRoot\System32\pnputil.exe $($t.Verb) /deviceid `"$VddHardwareId`"")
    if ($services) {
        $names = ($services | ForEach-Object { "'$_'" }) -join ','
        $action += New-ScheduledTaskAction -Execute "$env:SystemRoot\System32\conhost.exe" `
            -Argument "--headless $env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe -NoProfile -NonInteractive -Command `"Start-Sleep -Seconds 3; Restart-Service -Name $names -Force`""
    }
    $principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 1)
    Register-ScheduledTask -TaskPath $TaskPath -TaskName $t.Name -Action $action -Principal $principal -Settings $settings -Force | Out-Null
}
Write-Host "已建立排程工作 $($TaskPath)Plug、$($TaskPath)Unplug（使用者：$User）"
if ($services) { Write-Host "拔插後會自動重新啟動：$($services -join '、')" }

# 5. 開關程式
$exe = Join-Path $RepoRoot 'dist\windows\vdswitch.exe'
if (-not (Test-Path $exe) -and (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host '找不到開關程式執行檔，用 Go 編譯 ...'
    & (Join-Path $RepoRoot 'build.ps1')
}
if (Test-Path $exe) {
    Get-Process vdswitch -ErrorAction SilentlyContinue | Stop-Process -Force
    New-Item -ItemType Directory -Force $AppDir | Out-Null
    Copy-Item $exe $AppDir -Force
    # WScript.Shell 用系統 ANSI 字碼頁處理路徑，字碼頁沒有的中文字（例如日文系統的「虛」）會變成 ?，
    # 所以先用英文檔名建立，再用 PowerShell 改名
    $programs = Join-Path $env:ProgramData 'Microsoft\Windows\Start Menu\Programs'
    $lnk = Join-Path $programs '虛擬螢幕開關.lnk'
    $tmp = Join-Path $programs 'go-virtual-display.lnk'
    $sc = (New-Object -ComObject WScript.Shell).CreateShortcut($tmp)
    $sc.TargetPath = Join-Path $AppDir 'vdswitch.exe'
    $sc.Save()
    Move-Item $tmp $lnk -Force
    Write-Host "已安裝開關程式到 $AppDir，開始功能表捷徑：虛擬螢幕開關"
} else {
    Write-Host "找不到 $exe，也沒有安裝 Go，略過開關程式"
}

Write-Host ''
if (-not $Launch) {
    Write-Host '完成。執行 verify.ps1 檢查結果。'
    return
}
Write-Host '安裝完成，檢查結果：'
Start-Sleep -Seconds 3  # 等驅動把螢幕建立好
& (Join-Path $PSScriptRoot 'verify.ps1')
$installed = Join-Path $AppDir 'vdswitch.exe'
if (Test-Path $installed) {
    # 透過 explorer 啟動，開關程式才會以一般權限執行，而不是繼承這裡的系統管理員權限
    Start-Process explorer.exe -ArgumentList "`"$installed`""
    Write-Host '已開啟虛擬螢幕開關（桌面右下角與系統列）'
}
