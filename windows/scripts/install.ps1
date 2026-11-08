# 安裝虛擬螢幕驅動、套用 1920x1440 設定與自訂 EDID、建立免 UAC 的拔插排程工作、安裝開關程式。
# 需要系統管理員權限。可重複執行（會更新設定並重新載入驅動）。
param(
    # 排程工作要以哪個使用者執行；預設為目前登入桌面的使用者
    [string]$User = (Get-CimInstance Win32_ComputerSystem).UserName,
    # 安裝完成後執行檢查，並以一般權限開啟開關程式（install.cmd 使用）
    [switch]$Launch,
    # 介面語言；不指定時讓使用者選，預設依系統顯示語言（或上次安裝時的選擇）
    [ValidateSet('en', 'zh-TW')][string]$Language
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
Assert-Admin

# 已經安裝過：告訴使用者目前和要安裝的版本，同意後先移除舊版（保留解析度清單和語言），再全新安裝。
# 實際移除要等到驅動取得成功之後，否則沒有網路時舊版被移除、新版又裝不起來
$reinstall = $false
# 0.1.0 沒有登記到「設定 > 應用程式」，所以也看程式資料夾。不看 VDD 裝置：可能是使用者自己另外裝的，不能幫他移除
$installedVersion = (Get-ItemProperty $UninstallKey -Name DisplayVersion -ErrorAction SilentlyContinue).DisplayVersion
if ($installedVersion -or (Test-Path $AppDir)) {
    $shown = if ($installedVersion) { $installedVersion } else { T 'unknown (0.1.0 or earlier)' '不明（0.1.0 以前）' }
    Write-Host ''
    Write-Host (T 'Virtual Display Switch is already installed.' '虛擬螢幕開關已經安裝了。') -ForegroundColor Yellow
    Write-Host ((T '  Installed version: ' '  目前版本：') + $shown)
    Write-Host ((T '  This package:      ' '  要安裝的版本：') + $AppVersion)
    $iv = $null; $pv = $null
    if ([version]::TryParse("$installedVersion", [ref]$iv) -and [version]::TryParse($AppVersion, [ref]$pv)) {
        if ($iv -eq $pv) { Write-Host (T '  (Same version: this will reinstall it.)' '  （版本相同，會重新安裝）') }
        elseif ($iv -gt $pv) { Write-Host (T '  Warning: the installed version is newer; this will downgrade it.' '  注意：目前的版本比較新，繼續會降版。') -ForegroundColor Yellow }
    }
    Write-Host (T '  Your resolutions, language and "Start at sign-in" setting are kept.' '  你新增的解析度、選的語言和「登入時自動啟動」會保留。')
    Write-Host ''
    Write-Host (T 'Uninstall the current version and install this one? (y/N) : ' '要先移除目前的版本，再安裝這個版本嗎？(y/N) : ') -NoNewline
    $answer = [Console]::ReadLine()
    if ("$answer".Trim() -notmatch '^(y|yes)$') {
        Write-Host (T 'Cancelled. Nothing was changed.' '已取消，沒有做任何變更。')
        return
    }
    $reinstall = $true
}

# 1. 取得驅動（內附的或下載），在改動系統之前完成。失敗時直接中止，已安裝的版本保持原樣
Write-Host ''
try {
    $VddRoot = & (Join-Path $PSScriptRoot 'download-vdd.ps1')
} catch {
    throw ((T 'Could not get the driver; nothing was changed. ' '無法取得驅動，沒有做任何變更。') + $_.Exception.Message)
}

if ($reinstall) {
    Write-Host ''
    Write-Host (T 'Uninstalling the current version ...' '移除目前的版本 ...')
    & (Join-Path $PSScriptRoot 'uninstall.ps1') -KeepUserSettings
    Write-Host ''
    Write-Host ((T 'Installing version ' '安裝版本 ') + $AppVersion + ' ...')
}

# 0. 介面語言（安裝過程、開關程式、移除都用這個）
if (-not $Language) {
    $default = if ($Lang -eq 'zh-TW') { '2' } else { '1' }
    # 不用 Read-Host：Windows PowerShell 5.1 在 UTF-8 主控台（chcp 65001）用 Read-Host 時，
    # 前面的中文會被畫錯位置。改成先印完選單，再用 Console.ReadLine 讀取
    Write-Host ''
    Write-Host 'Select language / 選擇語言'
    Write-Host '  1) English'
    Write-Host '  2) 繁體中文 (Traditional Chinese)'
    Write-Host ''
    Write-Host "Enter 1 or 2, or press Enter for $default : " -NoNewline
    $answer = [Console]::ReadLine()
    $Language = switch ("$answer".Trim()) { '1' { 'en' } '2' { 'zh-TW' } default { $Lang } }
}
$Lang = $Language
New-Item -Force $LanguageKey | Out-Null
Set-ItemProperty $LanguageKey -Name Language -Value $Lang
Write-Host (T 'Language: English' '語言：繁體中文')
Write-Host ''

# 驅動檔案的位置（第 1 步取得的）
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'ARM64' } else { 'x86' }  # x86 資料夾內是 amd64 驅動
$driverDir = Join-Path $VddRoot "SignedDrivers\$arch\VDD"
$inf = Join-Path $driverDir 'MttVDD.inf'
$devcon = Join-Path $VddRoot 'Dependencies\devcon.exe'

# 2. 設定檔與 EDID（先備份原有檔案）
#    標記檔存在表示目前的檔案是本專案放的，重複安裝時不能再備份，否則移除時會「還原」成本專案的檔案
New-Item -ItemType Directory -Force $VddConfigDir | Out-Null
if (-not (Test-Path $InstallMarker)) {
    foreach ($name in 'vdd_settings.xml', 'user_edid.bin') {
        $path = Join-Path $VddConfigDir $name
        if (Test-Path $path) {
            Copy-Item $path "$path.go-virtual-display.bak" -Force
            Write-Host (T "Backed up the existing $name" "已備份原有的 $name")
        }
    }
    Set-Content $InstallMarker 'go-virtual-display'
}
Copy-Item (Join-Path $PSScriptRoot 'vdd_settings.xml') (Join-Path $VddConfigDir 'vdd_settings.xml') -Force
New-Item -Force $VddRegKey | Out-Null
Set-ItemProperty $VddRegKey -Name VDDPATH -Value $VddConfigDir
# 依使用者存的解析度產生 EDID 並改寫設定（新安裝是 1920x1080；重新安裝保留原本選的）。沒有執行檔時用預設 EDID
$applied = $false
if (Test-Path $AppExe) {
    $p = Start-Process $AppExe -ArgumentList '--apply' -Wait -PassThru
    $applied = $p.ExitCode -eq 0
    if (-not $applied) { Write-Host (T 'Could not apply the saved resolution; using the default 1920x1080' '無法套用存下的解析度，改用預設的 1920x1080') -ForegroundColor Yellow }
}
if (-not $applied) { Copy-Item $EdidFile (Join-Path $VddConfigDir 'user_edid.bin') -Force }
Write-Host (T "Applied settings and EDID to $VddConfigDir" "已套用設定與 EDID 到 $VddConfigDir")

# 3. 安裝驅動，或在已安裝時重新載入
if (Get-VddDevice) {
    Write-Host (T 'Driver already installed; updating the driver package and reloading the device ...' '驅動已安裝，更新驅動套件並重新載入裝置 ...')
    pnputil /add-driver "$inf" /install | Out-Null
    pnputil /restart-device /deviceid "$VddHardwareId" | Out-Null
} else {
    Write-Host (T 'Installing Virtual Display Driver ...' '安裝 Virtual Display Driver ...')
    & $devcon install "$inf" "$VddHardwareId"
    if ($LASTEXITCODE -ne 0) { throw (T "devcon install failed (code $LASTEXITCODE)" "devcon 安裝失敗（代碼 $LASTEXITCODE）") }
}
Restart-DisplayDependentServices

# 4. 排程工作：以最高權限執行，之後按開關、切換解析度都不會跳 UAC
#    Plug／Unplug 執行 vdswitch.exe --plug／--unplug 啟用或停用裝置（0.4.4 起用 SetupAPI；
#    之前的 pnputil /enable-device 在停用狀態下重新開機後會失敗，conhost 又會吞掉結束碼）
#    Apply 執行 vdswitch.exe --apply 套用使用者選的解析度
#    RestartServices：開關程式偵測到螢幕配置改變（拔插、切換解析度、在 Windows 設定裡調整排列）
#    並穩定後執行它，重新啟動 Synergy 等服務。一律建立，執行時才檢查服務在不在，
#    所以之後才安裝或移除 Synergy 都不必重新安裝本程式
$tasks = @(
    @{ Name = 'Plug'; Exe = (Join-Path $AppDir 'vdswitch.exe'); Arg = '--plug' },
    @{ Name = 'Unplug'; Exe = (Join-Path $AppDir 'vdswitch.exe'); Arg = '--unplug' },
    @{ Name = 'Apply'; Exe = (Join-Path $AppDir 'vdswitch.exe'); Arg = '--apply' },
    @{ Name = 'RestartServices'; Exe = "$env:SystemRoot\System32\conhost.exe"
        Arg = "--headless $env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe -NoProfile -NonInteractive -Command `"Get-Service -Name $(($RestartServices | ForEach-Object { "'$_'" }) -join ',') -ErrorAction SilentlyContinue | Restart-Service -Force`"" })
foreach ($t in $tasks) {
    $action = New-ScheduledTaskAction -Execute $t.Exe -Argument $t.Arg
    $principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Highest
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 1)
    Register-ScheduledTask -TaskPath $TaskPath -TaskName $t.Name -Action $action -Principal $principal -Settings $settings -Force | Out-Null
}
$taskNames = ($tasks | ForEach-Object { $_.Name })
Write-Host (T "Created scheduled tasks $($TaskPath)$($taskNames -join ', ') (user: $User)" "已建立排程工作 $($TaskPath)$($taskNames -join '、')（使用者：$User）")
Write-Host ((T 'After the display layout changes, these services are restarted if installed: ' '螢幕配置改變後，會自動重新啟動這些服務（有安裝的話）：') + ($RestartServices -join ', '))

# 5. 開關程式
$exe = $AppExe
if (-not $Packaged -and -not (Test-Path $exe) -and (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host (T 'Switch executable not found; building it with Go ...' '找不到開關程式執行檔，用 Go 編譯 ...')
    & (Join-Path $RepoRoot 'build.ps1')
}
if (Test-Path $exe) {
    Get-Process vdswitch -ErrorAction SilentlyContinue | Stop-Process -Force
    New-Item -ItemType Directory -Force $AppDir | Out-Null
    Copy-Item $exe $AppDir -Force
    # 從網路下載的安裝包會帶「來自網路」標記，解除後開啟時才不會被 SmartScreen 擋下
    Unblock-File (Join-Path $AppDir 'vdswitch.exe')
    # 開始功能表與桌面捷徑。檔名一律用英文，中文檔名在非中文系統上會出問題。
    # 中文介面時在「註解」放中文名稱，滑鼠提示會顯示，Windows 搜尋也有機會比對到
    $programs = Join-Path $env:ProgramData 'Microsoft\Windows\Start Menu\Programs'
    Remove-Item (Join-Path $programs $LegacyShortcut) -Force -ErrorAction SilentlyContinue
    foreach ($dir in $programs, $DesktopDir) {
        New-Shortcut (Join-Path $dir "$AppName.lnk") (Join-Path $AppDir 'vdswitch.exe') `
            (T 'Plug in or unplug the virtual display and switch resolutions' '虛擬螢幕開關：插入或拔掉虛擬螢幕、切換解析度')
    }
    # 「拔掉」捷徑只放開始功能表：主螢幕看不到時盲打搜尋用，放桌面反而容易誤點。
    # 圖示用 vdswitch.exe 的資源 ID 2（assets\unplug.svg：螢幕熄掉＋紅色拔開插頭），和程式圖示區分
    New-Shortcut (Join-Path $programs "$UnplugShortcutName.lnk") (Join-Path $AppDir 'vdswitch.exe') `
        (T 'Unplug the virtual display right away' '立即拔掉虛擬螢幕') '--request-unplug' -IconIndex -2
    $zhShortcut = Join-Path $programs "$UnplugShortcutNameZh.lnk"
    Remove-Item $zhShortcut -Force -ErrorAction SilentlyContinue  # 重新安裝時改成英文介面的話不留
    if ($Lang -eq 'zh-TW') {
        # 參數要和英文捷徑不同（多一個 zh，程式會忽略）：目標和參數完全相同的捷徑，
        # 開始功能表只會列出其中一個，中文捷徑就搜尋不到
        New-Shortcut $zhShortcut (Join-Path $AppDir 'vdswitch.exe') '立即拔掉虛擬螢幕' '--request-unplug zh' -IconIndex -2
    }
    Write-Host (T "Installed the switch to $AppDir; shortcuts on the Start menu and desktop: $AppName" "已安裝開關程式到 $AppDir，開始功能表與桌面捷徑：$AppName")
    Write-Host (T "If the virtual display is the main display and you cannot see the switch:" '虛擬螢幕是主螢幕、看不到開關時：')
    Write-Host (T "  - Ctrl+Alt+Shift+V moves the widget next to the mouse pointer (can be changed from the tray menu)" '  - Ctrl+Alt+Shift+V：把小工具移到滑鼠游標旁邊（可在系統列選單更改）')
    Write-Host (T "  - Press Win, type `"unplug`" and press Enter to unplug the virtual display ($UnplugShortcutName)" "  - 按 Win、輸入「unplug」或「拔掉」、按 Enter：拔掉虛擬螢幕（$UnplugShortcutNameZh）")
} else {
    Write-Host (T "$exe not found and Go is not installed; skipping the switch" "找不到 $exe，也沒有安裝 Go，略過開關程式")
}

# 6. 移除用的檔案複製到安裝位置，並登記到「設定 > 應用程式」，安裝資料夾刪掉後仍能移除
Remove-Item $UninstallerDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $UninstallerDir 'scripts') | Out-Null
Copy-Item (Join-Path (Split-Path -Parent $PSScriptRoot) 'uninstall.cmd') $UninstallerDir
Copy-Item (Join-Path $PSScriptRoot 'common.ps1'), (Join-Path $PSScriptRoot 'uninstall.ps1'), (Join-Path $PSScriptRoot 'launch-uninstall.ps1') (Join-Path $UninstallerDir 'scripts')
Get-ChildItem $UninstallerDir -Recurse -File | Unblock-File
$launcher = Join-Path $UninstallerDir 'scripts\launch-uninstall.ps1'
New-Item -Force $UninstallKey | Out-Null
$entry = @{
    DisplayName     = (T 'Virtual Display Switch (go-virtual-display)' '虛擬螢幕開關（go-virtual-display）')
    DisplayVersion  = $AppVersion
    Publisher       = 'go-virtual-display'
    InstallLocation = $AppDir
    UninstallString = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$launcher`""
    URLInfoAbout    = 'https://github.com/The78estHerry/go-virtual-display'
}
if (Test-Path (Join-Path $AppDir 'vdswitch.exe')) { $entry.DisplayIcon = Join-Path $AppDir 'vdswitch.exe' }
foreach ($k in $entry.Keys) { Set-ItemProperty $UninstallKey -Name $k -Value $entry[$k] }
Set-ItemProperty $UninstallKey -Name NoModify -Value 1 -Type DWord
Set-ItemProperty $UninstallKey -Name NoRepair -Value 1 -Type DWord
$sizeKB = [int]((Get-ChildItem $AppDir -Recurse -File | Measure-Object Length -Sum).Sum / 1KB)
Set-ItemProperty $UninstallKey -Name EstimatedSize -Value $sizeKB -Type DWord
Write-Host (T 'Registered in Settings > Apps; you can also uninstall from the tray icon (Uninstall…)' '已登記到「設定 > 應用程式」，也可從系統列圖示的「解除安裝…」移除')

Write-Host ''
if (-not $Launch) {
    Write-Host (T 'Done. Run verify.ps1 to check the result.' '完成。執行 verify.ps1 檢查結果。')
    return
}
Write-Host (T 'Installation complete. Checking:' '安裝完成，檢查結果：')
Start-Sleep -Seconds 3  # 等驅動把螢幕建立好
& (Join-Path $PSScriptRoot 'verify.ps1')
$installed = Join-Path $AppDir 'vdswitch.exe'
if (Test-Path $installed) {
    # 透過 explorer 啟動，開關程式才會以一般權限執行，而不是繼承這裡的系統管理員權限
    Start-Process explorer.exe -ArgumentList "`"$installed`""
    Write-Host (T 'Opened Virtual Display Switch (bottom-right of the desktop and the tray)' '已開啟虛擬螢幕開關（桌面右下角與系統列）')
}
