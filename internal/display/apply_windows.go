package display

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

const applyTimeout = 30 * time.Second

// ApplyActive 觸發最高權限排程工作 Apply（執行 vdswitch.exe --apply），並等它寫回結果。
func (winController) ApplyActive() error {
	start := time.Now()
	if err := runTask("Apply"); err != nil {
		return err
	}
	if err := waitTaskResult(applyResultFile, start, applyTimeout); err != errNoResult {
		return err
	}
	return errors.New(i18n.T("Timed out while applying the resolution", "套用解析度逾時"))
}

// ApplyElevated 由 vdswitch.exe --apply 呼叫（以系統管理員身分執行）：
// 依 profiles.json 使用中的解析度產生 EDID、改寫 VDD 設定，插入狀態下重新啟動裝置。
// 只從使用者的設定讀取三個整數並驗證範圍，寫入的位置固定。結果同時寫到 apply-result.json。
func ApplyElevated() error {
	err := applyElevated()
	writeTaskResult(applyResultFile, err)
	return err
}

func applyElevated() error {
	p := profile.Load().ActiveProfile()
	if err := p.Validate(); err != nil {
		return err
	}
	b, err := p.EDID()
	if err != nil {
		return err
	}
	dir := vddConfigDir()
	if err := writeFileAtomic(filepath.Join(dir, "user_edid.bin"), b); err != nil {
		return err
	}
	if err := setVDDResolution(filepath.Join(dir, "vdd_settings.xml"), p); err != nil {
		return err
	}
	if st, _ := (winController{}).Status(); st == Plugged {
		cmd := exec.Command("pnputil.exe", "/restart-device", "/deviceid", hardwareID)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(i18n.T("Cannot restart the virtual display: %v", "無法重新啟動虛擬螢幕：%v"), err)
		}
	}
	return nil
}

// vddConfigDir 回傳 VDD 的設定目錄（安裝腳本寫在登錄檔 VDDPATH）。
func vddConfigDir() string {
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\MikeTheTech\VirtualDisplayDriver`, registry.QUERY_VALUE|registry.WOW64_64KEY); err == nil {
		defer k.Close()
		if v, _, err := k.GetStringValue("VDDPATH"); err == nil && v != "" {
			return v
		}
	}
	return `C:\VirtualDisplayDriver`
}

var (
	reResolutions = regexp.MustCompile(`(?s)<resolutions>.*?</resolutions>`)
	reRefresh     = regexp.MustCompile(`<g_refresh_rate>\s*\d+\s*</g_refresh_rate>`)
)

// setVDDResolution 把 vdd_settings.xml 的解析度清單和 global 更新率改成只有 p 這一組。
func setVDDResolution(path string, p profile.Profile) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	if !reResolutions.MatchString(s) || !reRefresh.MatchString(s) {
		return fmt.Errorf(i18n.T("Unexpected format in %s; please reinstall", "%s 的格式不符，請重新安裝"), path)
	}
	s = reResolutions.ReplaceAllLiteralString(s, fmt.Sprintf(
		"<resolutions>\n        <resolution>\n            <width>%d</width>\n            <height>%d</height>\n            <refresh_rate>%d</refresh_rate>\n        </resolution>\n    </resolutions>",
		p.Width, p.Height, p.Refresh))
	first := true
	s = reRefresh.ReplaceAllStringFunc(s, func(string) string {
		if first {
			first = false
			return fmt.Sprintf("<g_refresh_rate>%d</g_refresh_rate>", p.Refresh)
		}
		return "" // 只留一個 global 更新率
	})
	// 格式錯誤的 XML 驅動會整個不讀，結果是不建立螢幕、也沒有任何錯誤訊息，所以寫入前先檢查
	if err := checkXML([]byte(s)); err != nil {
		return fmt.Errorf(i18n.T("Invalid XML in %s: %v; please reinstall", "%s 的 XML 格式錯誤：%v，請重新安裝"), path, err)
	}
	return writeFileAtomic(path, []byte(s))
}

// checkXML 確認 b 是格式正確的 XML（例如註解裡不能有 "--"）。
func checkXML(b []byte) error {
	d := xml.NewDecoder(bytes.NewReader(b))
	for {
		if _, err := d.Token(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func writeFileAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
