# 各腳本共用的常數與函式（以 dot-source 載入）。

$VddVersion   = '25.7.23'
$VddUrl       = "https://github.com/VirtualDrivers/Virtual-Display-Driver/releases/download/$VddVersion/VDD.Control.$VddVersion.zip"
$VddSha256    = 'A701F2272E9FCF382849B24F913C6DD07597B3B1116525F2E90182F019609154'
# Release 安裝包內附的驅動檔案（scripts\vdd\，配置和 zip 裡相同），讓沒有網路或下載點消失時也能安裝。
# 只取需要的檔案，內容和 zip 裡完全相同（簽章仍然有效）。雜湊取自用 $VddSha256 驗證過的 zip，
# 寫在 repo 裡而不是打包時產生，安裝包被改過也會被發現。升級 VDD 時要一起更新
$VddFilesSha256 = [ordered]@{
    'SignedDrivers\x86\VDD\MttVDD.inf'   = '550D211FE481E74DFE3F9D724ED78BE48B3A9113405965D683D9373E8D672F5D'
    'SignedDrivers\x86\VDD\MttVDD.dll'   = 'C9CA837F57A98FBD43BC416A7F535A95843626E7759EAF85CF0CD7CE334DBB05'
    'SignedDrivers\x86\VDD\mttvdd.cat'   = '08A0093FC9B2E32B287A6F8A77CA4DE0A31830D29FC33D2B13A918DC859468F6'
    'SignedDrivers\ARM64\VDD\MttVDD.inf' = 'A8B8D1C939EE49BB44B4CC3EF9A820B9943B1DBFA456D5B6880B6F106B51011B'
    'SignedDrivers\ARM64\VDD\MttVDD.dll' = '9BD5429A71F2103309B291C18D97DAA5E2D8E1A35794F375AE718F1F1EB27C4F'
    'SignedDrivers\ARM64\VDD\mttvdd.cat' = '9E62DD01DE5D80773819E6C92C76DACB175B4632E0D551026975DA5F815B5A97'
    'Dependencies\devcon.exe'              = '77E87E7C4E23D6B9A4F5D86D15ED9EC61A16A242FB299C390AF6F179225BFA05'
}
$VddHardwareId = 'Root\MttVDD'
$VddConfigDir = 'C:\VirtualDisplayDriver'
$VddRegKey    = 'HKLM:\SOFTWARE\MikeTheTech\VirtualDisplayDriver'
$VddMonitorId = 'DISPLAY\MTT1337'  # 驅動建立的螢幕裝置（驅動會把 EDID 的廠商／產品代碼換成 MTT／0x1337）
$InstallMarker = Join-Path $VddConfigDir '.go-virtual-display'

# 兩種配置：Release 安裝包把預設 EDID 和 vdswitch.exe 放在 scripts\ 裡；原始碼則各在 edid\、dist\windows\
# 預設 EDID（1920×1080）只在沒有 vdswitch.exe 時使用；有的話由 vdswitch.exe --apply 依使用者選的解析度產生
$Packaged     = Test-Path (Join-Path $PSScriptRoot 'edid_1920x1080.bin')
if ($Packaged) {
    $VendorDir = Join-Path $PSScriptRoot "downloads\vdd-$VddVersion"
    $VddBundleDir = Join-Path $PSScriptRoot 'vdd'  # 內附的驅動
    $EdidFile  = Join-Path $PSScriptRoot 'edid_1920x1080.bin'
    $AppExe    = Join-Path $PSScriptRoot 'vdswitch.exe'
} else {
    $RepoRoot  = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    $VendorDir = Join-Path $RepoRoot "downloads\vdd-$VddVersion"
    $VddBundleDir = $null  # 原始碼沒有內附驅動，一律下載
    $EdidFile  = Join-Path $RepoRoot 'edid\edid_1920x1080.bin'
    $AppExe    = Join-Path $RepoRoot 'dist\windows\vdswitch.exe'
}
$AppVersion   = '0.4.4'  # 發佈版本；release.ps1、release-linux.ps1 以此為準
$AppName      = 'Virtual Display Switch'  # 開始功能表捷徑名稱（檔名一律英文）
# 開始功能表的「拔掉」捷徑（0.4.4 起）：虛擬螢幕是主螢幕、開關都在看不到的螢幕上時，
# 可以盲打「Win → unplug → Enter」拔掉。執行 vdswitch.exe --request-unplug，透過排程工作不跳 UAC
$UnplugShortcutName = 'Unplug Virtual Display'
# 中文介面時多建一個中文名稱的捷徑：開始功能表搜尋只比對名稱、不比對註解，英文名稱用中文搜不到。
# 這是「檔名一律英文」的例外（使用者決定），只在中文介面建立。捷徑用 IShellLinkW 建立，中文檔名沒有字碼頁問題
$UnplugShortcutNameZh = '拔掉虛擬螢幕'
$AppDir       = Join-Path $env:ProgramFiles 'go-virtual-display'
$DesktopDir   = [Environment]::GetFolderPath('Desktop')  # 安裝與移除都以同一個使用者的系統管理員身分執行
$UninstallerDir = Join-Path $AppDir 'uninstaller'  # 安裝資料夾刪掉後仍能移除
$UninstallKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\go-virtual-display'  # 「設定 > 應用程式」
$TaskPath     = '\go-virtual-display\'
$RunKey       = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$RunValue     = 'go-virtual-display'
# 螢幕配置改變後要重新啟動的服務：這些程式只在啟動時讀取桌面大小，螢幕增減後滑鼠位置會算錯
# Synergy 的服務名稱也寫在 internal/display/synergy_windows.go（系統列和小工具的 Synergy 圖示），改名時兩邊一起改
$RestartServices = @('Synergy Core Daemon')
$LanguageKey  = 'HKLM:\SOFTWARE\go-virtual-display'  # 安裝時選的介面語言（Language = en／zh-TW），開關程式也讀這裡
$LegacyShortcut = '虛擬螢幕開關.lnk'  # 0.1.x 用的中文捷徑名稱，安裝與移除時一併刪除

# 系統顯示語言是中文時回傳 zh-TW，其他一律 en。
# PowerShell 的 Get-UICulture 可能和實際顯示語言不同（例如顯示語言是中文卻回傳 en-US），所以直接查 API。
function Get-SystemLanguage {
    if (-not ('GoVirtualDisplay.Lang' -as [type])) {
        Add-Type -Namespace GoVirtualDisplay -Name Lang -MemberDefinition '[DllImport("kernel32.dll")] public static extern ushort GetUserDefaultUILanguage();'
    }
    if (([GoVirtualDisplay.Lang]::GetUserDefaultUILanguage() -band 0x3ff) -eq 0x04) { 'zh-TW' } else { 'en' }
}
$Lang = (Get-ItemProperty $LanguageKey -Name Language -ErrorAction SilentlyContinue).Language
if ($Lang -notin 'en', 'zh-TW') { $Lang = Get-SystemLanguage }

# 依介面語言回傳英文或繁體中文的字串
function T([string]$En, [string]$Zh) { if ($Lang -eq 'zh-TW') { $Zh } else { $En } }

function Assert-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not (New-Object Security.Principal.WindowsPrincipal $id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw (T 'Please run this script from PowerShell started with "Run as administrator".' '請用「以系統管理員身分執行」的 PowerShell 執行這個腳本。')
    }
}

# 建立捷徑。不用 WScript.Shell：它以 ANSI 字碼頁處理字串，字碼頁沒有的中文字（例如日文系統的「虛」）會變成 ?。
# 這裡直接用 Unicode 版的 IShellLinkW。
# $IconIndex 不是 0 時改用 $Target 裡的其他圖示；負數表示資源 ID（例如 -2 是 vdswitch.rc 的 2 ICON）。
function New-Shortcut([string]$Path, [string]$Target, [string]$Description, [string]$Arguments = '', [int]$IconIndex = 0) {
    if (-not ('GoVirtualDisplay.ShellLink' -as [type])) {
        Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;
using System.Text;
namespace GoVirtualDisplay {
    [ComImport, Guid("000214F9-0000-0000-C000-000000000046"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IShellLinkW {
        void GetPath([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder f, int cch, IntPtr fd, uint flags);
        void GetIDList(out IntPtr ppidl);
        void SetIDList(IntPtr pidl);
        void GetDescription([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder s, int cch);
        void SetDescription([MarshalAs(UnmanagedType.LPWStr)] string s);
        void GetWorkingDirectory([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder s, int cch);
        void SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string s);
        void GetArguments([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder s, int cch);
        void SetArguments([MarshalAs(UnmanagedType.LPWStr)] string s);
        void GetHotkey(out short k);
        void SetHotkey(short k);
        void GetShowCmd(out int c);
        void SetShowCmd(int c);
        void GetIconLocation([Out, MarshalAs(UnmanagedType.LPWStr)] StringBuilder s, int cch, out int i);
        void SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string s, int i);
        void SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string s, uint r);
        void Resolve(IntPtr hwnd, uint flags);
        void SetPath([MarshalAs(UnmanagedType.LPWStr)] string s);
    }
    [ComImport, Guid("00021401-0000-0000-C000-000000000046")]
    class CShellLink { }
    public static class ShellLink {
        public static void Create(string path, string target, string description, string arguments, int iconIndex) {
            var link = (IShellLinkW)new CShellLink();
            link.SetPath(target);
            link.SetArguments(arguments);
            if (iconIndex != 0) { link.SetIconLocation(target, iconIndex); }
            link.SetWorkingDirectory(System.IO.Path.GetDirectoryName(target));
            link.SetDescription(description);
            ((IPersistFile)link).Save(path, true);
        }
    }
}
'@
    }
    [GoVirtualDisplay.ShellLink]::Create($Path, $Target, $Description, $Arguments, $IconIndex)
}

# Test-VddFiles 檢查 $root 底下的驅動檔案是否齊全且雜湊都對（內附的或下載解壓縮的都適用）
function Test-VddFiles([string]$root) {
    if (-not $root -or -not (Test-Path $root)) { return $false }
    foreach ($f in $VddFilesSha256.Keys) {
        $p = Join-Path $root $f
        if (-not (Test-Path $p) -or (Get-FileHash $p -Algorithm SHA256).Hash -ne $VddFilesSha256[$f]) { return $false }
    }
    return $true
}

function Get-VddDevice {
    Get-PnpDevice -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -contains $VddHardwareId }
}

# 安裝／移除驅動也會增減螢幕，和拔插一樣要重新啟動 $RestartServices
function Restart-DisplayDependentServices {
    $services = @($RestartServices | Where-Object { Get-Service -Name $_ -ErrorAction SilentlyContinue })
    if (-not $services) { return }
    Start-Sleep -Seconds 3  # 等螢幕建立或移除完成
    Restart-Service -Name $services -Force
    Write-Host ((T 'Restarted: ' '已重新啟動：') + ($services -join ', '))
}
