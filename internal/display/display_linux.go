package display

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

const (
	configFile = "/etc/go-virtual-display.conf" // install.sh 寫入 CONNECTOR=card0-HDMI-A-1
	helperPath = "/usr/local/libexec/go-virtual-display/vdswitch-helper"
)

type linuxController struct {
	statusPath string
}

func newController() (Controller, error) {
	conn, err := readConnector()
	if err != nil {
		return nil, err
	}
	return linuxController{statusPath: filepath.Join("/sys/class/drm", conn, "status")}, nil
}

func readConnector() (string, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return "", ErrNotInstalled
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if ok && k == "CONNECTOR" && v != "" && !strings.ContainsAny(v, "/.") {
			return v, nil
		}
	}
	return "", fmt.Errorf(i18n.T("no_valid_connector_setting_in"), configFile)
}

// Status 讀取 DRM 接頭的狀態，不需要 root。
func (c linuxController) Status() (State, error) {
	b, err := os.ReadFile(c.statusPath)
	if err != nil {
		return Unknown, ErrNotInstalled
	}
	switch strings.TrimSpace(string(b)) {
	case "connected":
		return Plugged, nil
	case "disconnected":
		return Unplugged, nil
	}
	return Unknown, fmt.Errorf(i18n.T("unknown_connector_status"), strings.TrimSpace(string(b)))
}

func (linuxController) Plug() error   { _, err := runHelper(nil, "on"); return err }
func (linuxController) Unplug() error { _, err := runHelper(nil, "off"); return err }

// ApplyActive 依 profiles.json 使用中的解析度產生 EDID，從 stdin 交給小幫手：
// 小幫手驗證後寫到 /lib/firmware/edid/go-virtual-display.bin，有 debugfs 時立即套用，並在背景重建 initramfs。
// 開機參數固定指向這個檔名，所以換解析度不用改 GRUB。
func (linuxController) ApplyActive() error {
	p := profile.Load().ActiveProfile()
	if err := p.Validate(); err != nil {
		return err
	}
	b, err := p.EDID()
	if err != nil {
		return err
	}
	out, err := runHelper(b, "edid")
	if err != nil {
		return err
	}
	if strings.Contains(out, "reboot-required") {
		return ErrRebootRequired
	}
	return nil
}

// runHelper 透過 polkit（pkexec）以 root 執行小幫手；install.sh 已設定為使用中的本機使用者免密碼。
// stdin 不是 nil 時會送給小幫手。回傳小幫手的輸出。
func runHelper(stdin []byte, arg string) (string, error) {
	cmd := exec.Command("pkexec", helperPath, arg)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(i18n.T("switch_failed"), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
