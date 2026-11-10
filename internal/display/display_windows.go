package display

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
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
	set, data, err := findDevice()
	if err != nil {
		return Unknown, err
	}
	defer set.Close()
	var status, problem uint32
	if err := windows.CM_Get_DevNode_Status(&status, &problem, data.DevInst, 0); err != nil {
		return Unknown, err
	}
	switch {
	case status&windows.DN_HAS_PROBLEM != 0 && problem == cmProbDisabled:
		return Unplugged, nil
	case status&windows.DN_STARTED != 0:
		return Plugged, nil
	default:
		return Unknown, fmt.Errorf(i18n.T("virtual_display_device_error_code"), problem)
	}
}

// findDevice 在 root 列舉的裝置裡找出 VDD（硬體 ID Root\MttVDD），略過已移除只留紀錄的裝置。
// 呼叫端要 Close 回傳的 set。
func findDevice() (windows.DevInfo, *windows.DevInfoData, error) {
	set, err := windows.SetupDiGetClassDevsEx(nil, "ROOT", 0, windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return 0, nil, err
	}
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
		if windows.CM_Get_DevNode_Status(&status, &problem, data.DevInst, 0) != nil {
			continue // 已移除但還留著紀錄的裝置
		}
		return set, data, nil
	}
	set.Close()
	return 0, nil, ErrNotInstalled
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

func (winController) Plug() error   { return runSwitchTask("Plug") }
func (winController) Unplug() error { return runSwitchTask("Unplug") }

// runTask 觸發安裝時建立的最高權限排程工作，避免每次都跳 UAC。
func runTask(name string) error {
	cmd := exec.Command("schtasks.exe", "/run", "/tn", taskPath+name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(i18n.T("cannot_run_scheduled_task_please_reinstall"), taskPath, name, err)
	}
	return nil
}
