# 從已安裝的位置（C:\Program Files\go-virtual-display\uninstaller）啟動移除。
# 系統列的「解除安裝…」和「設定 > 應用程式」都呼叫這個腳本。
# 移除會刪掉 Program Files 裡的資料夾，所以先把移除用的檔案複製到暫存資料夾，從那裡以系統管理員身分執行，結束後再刪掉暫存資料夾。
$ErrorActionPreference = 'Stop'
$src = Split-Path -Parent $PSScriptRoot  # uninstaller\
$tmp = Join-Path $env:TEMP 'go-virtual-display-uninstall'
Set-Location $env:TEMP
Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
Copy-Item $src $tmp -Recurse
try {
    # uninstall.cmd 已是系統管理員身分時會直接執行，最後 pause 讓使用者看結果
    Start-Process cmd.exe -Verb RunAs -Wait -ArgumentList '/c', "`"$(Join-Path $tmp 'uninstall.cmd')`""
} catch {
    # 使用者在 UAC 按了「否」
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
