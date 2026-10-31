package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procMessageBox = user32.NewProc("MessageBoxW")

const (
	mbYesNo        = 0x00000004
	mbIconQuestion = 0x00000020
	mbDefButton2   = 0x00000100
	mbTopMost      = 0x00040000
	idYes          = 6
)

// uninstallLauncher 回傳安裝時複製到 Program Files 的移除腳本；找不到（例如直接從原始碼執行）時回傳空字串。
func uninstallLauncher() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	p := filepath.Join(filepath.Dir(exe), "uninstaller", "scripts", "launch-uninstall.ps1")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func uninstallSupported() bool { return uninstallLauncher() != "" }

// startUninstall 確認後啟動移除腳本；腳本會跳 UAC，並先關掉這個程式。回傳是否已啟動。
func startUninstall() bool {
	title, _ := windows.UTF16PtrFromString("解除安裝虛擬螢幕開關")
	text, _ := windows.UTF16PtrFromString("要移除虛擬螢幕驅動、開關程式與所有設定嗎？\n\n接下來會詢問系統管理員權限。")
	r, _, _ := procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)),
		mbYesNo|mbIconQuestion|mbDefButton2|mbTopMost)
	if r != idYes {
		return false
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden",
		"-File", uninstallLauncher())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd.Start() == nil
}
