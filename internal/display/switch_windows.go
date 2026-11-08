package display

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

// 拔插由最高權限排程工作 Plug／Unplug 執行 vdswitch.exe --plug／--unplug（0.4.4 起）。
//
// 0.4.3 以前排程工作是 `conhost.exe --headless pnputil.exe /enable-device ...`，有兩個問題：
//   - 裝置在停用狀態下重新開機後，pnputil /enable-device 會失敗（1167「The device is not connected」），
//     用 SetupAPI 的 DIF_PROPERTYCHANGE（和 Enable-PnpDevice、裝置管理員相同）才啟用得了。
//   - conhost --headless 不會傳回子程序的結束碼，失敗時開關程式只看到「逾時」。
// 所以改成由 vdswitch.exe 自己呼叫 SetupAPI，並把結果寫到 switch-result.json。

const (
	switchResultFile    = "switch-result.json"
	switchResultTimeout = 10 * time.Second
	applyResultFile     = "apply-result.json"
	resultInterval      = 300 * time.Millisecond
)

// taskResult 是以最高權限執行的子命令寫給開關程式看的結果。
type taskResult struct {
	Time  time.Time `json:"time"`
	Error string    `json:"error,omitempty"`
}

func taskResultPath(name string) (string, error) {
	p, err := profile.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), name), nil
}

// writeTaskResult 把 err 寫到結果檔，寫不進去也不影響子命令本身的結果。
func writeTaskResult(name string, err error) {
	r := taskResult{Time: time.Now()}
	if err != nil {
		r.Error = err.Error()
	}
	path, perr := taskResultPath(name)
	if perr != nil {
		return
	}
	if b, merr := json.Marshal(r); merr == nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, b, 0o644)
	}
}

// waitTaskResult 等待 start 之後寫入的結果。等不到時回傳 errNoResult。
func waitTaskResult(name string, start time.Time, timeout time.Duration) error {
	path, err := taskResultPath(name)
	if err != nil {
		return err
	}
	for deadline := start.Add(timeout); time.Now().Before(deadline); time.Sleep(resultInterval) {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var r taskResult
		if json.Unmarshal(b, &r) != nil || r.Time.Before(start) {
			continue // 還是上一次的結果，或正在寫入
		}
		if r.Error != "" {
			return errors.New(r.Error)
		}
		return nil
	}
	return errNoResult
}

var errNoResult = errors.New("no result")

// runSwitchTask 觸發 Plug／Unplug 排程工作並等它寫回結果。
// 等不到結果（例如還是 0.4.3 以前建立的排程工作）時回傳 nil，交給呼叫端輪詢狀態。
func runSwitchTask(name string) error {
	start := time.Now()
	if err := runTask(name); err != nil {
		return err
	}
	if err := waitTaskResult(switchResultFile, start, switchResultTimeout); err != nil && err != errNoResult {
		return err
	}
	return nil
}

// SetEnabledElevated 由 vdswitch.exe --plug／--unplug 呼叫（以系統管理員身分執行）：
// 啟用或停用 VDD 裝置，結果同時寫到 switch-result.json。
func SetEnabledElevated(enable bool) error {
	err := setEnabled(enable)
	writeTaskResult(switchResultFile, err)
	return err
}

func setEnabled(enable bool) error {
	set, data, err := findDevice()
	if err != nil {
		return err
	}
	defer set.Close()
	params := windows.PropChangeParams{
		ClassInstallHeader: *windows.MakeClassInstallHeader(windows.DIF_PROPERTYCHANGE),
		StateChange:        windows.DICS_DISABLE,
		Scope:              windows.DICS_FLAG_GLOBAL,
	}
	if enable {
		params.StateChange = windows.DICS_ENABLE
	}
	if err := set.SetClassInstallParams(data, &params.ClassInstallHeader, uint32(unsafe.Sizeof(params))); err != nil {
		return switchError(enable, err)
	}
	if err := set.CallClassInstaller(windows.DIF_PROPERTYCHANGE, data); err != nil {
		return switchError(enable, err)
	}
	if p, err := set.DeviceInstallParams(data); err == nil && p.Flags&(windows.DI_NEEDREBOOT|windows.DI_NEEDRESTART) != 0 {
		return errors.New(i18n.T("Windows needs a restart to finish this change", "需要重新開機才能完成切換"))
	}
	return nil
}

func switchError(enable bool, err error) error {
	if enable {
		return fmt.Errorf(i18n.T("Cannot plug in the virtual display: %v", "無法插入虛擬螢幕：%v"), err)
	}
	return fmt.Errorf(i18n.T("Cannot unplug the virtual display: %v", "無法拔掉虛擬螢幕：%v"), err)
}
