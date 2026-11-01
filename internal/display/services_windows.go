package display

import (
	"os/exec"
	"syscall"
)

// restartTask 是安裝時偵測到 Synergy 等服務才會建立的排程工作，以最高權限重新啟動那些服務。
const restartTask = "RestartServices"

// HasServiceRestart 回報是否有重新啟動服務的排程工作（沒裝 Synergy 等服務時沒有）。
func HasServiceRestart() bool {
	cmd := exec.Command("schtasks.exe", "/query", "/tn", taskPath+restartTask)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Run() == nil
}

// RestartDependentServices 重新啟動螢幕配置改變後需要重新讀取桌面大小的服務（例如 Synergy）。
func RestartDependentServices() error { return runTask(restartTask) }
