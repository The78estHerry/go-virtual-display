package display

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

const (
	hardwareID     = `Root\MttVDD` // Virtual Display Driver 的硬體 ID
	taskPath       = `\go-virtual-display\`
	cmProbDisabled = 22 // CM_PROB_DISABLED：裝置被停用
	createNoWindow = 0x08000000
)

type winController struct{}

func newController() (Controller, error) { return winController{}, nil }

// Status 透過 SetupAPI 找到 VDD 裝置並讀取 devnode 狀態，不需要系統管理員權限。
func (winController) Status() (State, error) {
	set, err := windows.SetupDiGetClassDevsEx(nil, "ROOT", 0, windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return Unknown, err
	}
	defer set.Close()
	for i := 0; ; i++ {
		data, err := set.EnumDeviceInfo(i)
		if err != nil {
			break // ERROR_NO_MORE_ITEMS
		}
		prop, err := set.DeviceRegistryProperty(data, windows.SPDRP_HARDWAREID)
		if err != nil || !hasID(prop) {
			continue
		}
		var status, problem uint32
		if err := windows.CM_Get_DevNode_Status(&status, &problem, data.DevInst, 0); err != nil {
			continue // 已移除但還留著紀錄的裝置
		}
		switch {
		case status&windows.DN_HAS_PROBLEM != 0 && problem == cmProbDisabled:
			return Unplugged, nil
		case status&windows.DN_STARTED != 0:
			return Plugged, nil
		default:
			return Unknown, fmt.Errorf(i18n.T("Virtual display device error (code %d)", "虛擬螢幕裝置有問題（代碼 %d）"), problem)
		}
	}
	return Unknown, ErrNotInstalled
}

func hasID(prop interface{}) bool {
	ids, _ := prop.([]string)
	for _, id := range ids {
		if strings.EqualFold(id, hardwareID) {
			return true
		}
	}
	return false
}

func (winController) Plug() error   { return runTask("Plug") }
func (winController) Unplug() error { return runTask("Unplug") }

// runTask 觸發安裝時建立的最高權限排程工作，避免每次都跳 UAC。
func runTask(name string) error {
	cmd := exec.Command("schtasks.exe", "/run", "/tn", taskPath+name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(i18n.T("Cannot run scheduled task %s%s (please reinstall): %v", "無法執行排程工作 %s%s（請重新安裝）：%v"), taskPath, name, err)
	}
	return nil
}
