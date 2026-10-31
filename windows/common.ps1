# 各腳本共用的常數與函式（以 dot-source 載入）。

$VddVersion   = '25.7.23'
$VddUrl       = "https://github.com/VirtualDrivers/Virtual-Display-Driver/releases/download/$VddVersion/VDD.Control.$VddVersion.zip"
$VddSha256    = 'A701F2272E9FCF382849B24F913C6DD07597B3B1116525F2E90182F019609154'
$VddHardwareId = 'Root\MttVDD'
$VddConfigDir = 'C:\VirtualDisplayDriver'
$VddRegKey    = 'HKLM:\SOFTWARE\MikeTheTech\VirtualDisplayDriver'
$VddMonitorId = 'DISPLAY\MTT1337'  # 驅動建立的螢幕裝置（驅動會把 EDID 的廠商／產品代碼換成 MTT／0x1337）
$InstallMarker = Join-Path $VddConfigDir '.go-virtual-display'

$RepoRoot     = Split-Path -Parent $PSScriptRoot
$VendorDir    = Join-Path $RepoRoot "downloads\vdd-$VddVersion"
$EdidFile     = Join-Path $RepoRoot 'edid\edid_1920x1440.bin'
$AppDir       = Join-Path $env:ProgramFiles 'go-virtual-display'
$TaskPath     = '\go-virtual-display\'
$RunKey       = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$RunValue     = 'go-virtual-display'
# 拔插後要重新啟動的服務：這些程式只在啟動時讀取桌面大小，螢幕增減後滑鼠位置會算錯
$RestartServices = @('Synergy Core Daemon')

function Assert-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not (New-Object Security.Principal.WindowsPrincipal $id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw '請用「以系統管理員身分執行」的 PowerShell 執行這個腳本。'
    }
}

function Get-VddDevice {
    Get-PnpDevice -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -contains $VddHardwareId }
}
