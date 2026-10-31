package display

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
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
	return "", fmt.Errorf(i18n.T("No valid CONNECTOR setting in %s", "%s 裡沒有有效的 CONNECTOR 設定"), configFile)
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
	return Unknown, fmt.Errorf(i18n.T("Unknown connector status: %q", "接頭狀態不明：%q"), strings.TrimSpace(string(b)))
}

func (linuxController) Plug() error   { return runHelper("on") }
func (linuxController) Unplug() error { return runHelper("off") }

// runHelper 透過 polkit（pkexec）以 root 執行小幫手；install.sh 已設定為使用中的本機使用者免密碼。
func runHelper(arg string) error {
	out, err := exec.Command("pkexec", helperPath, arg).CombinedOutput()
	if err != nil {
		return fmt.Errorf(i18n.T("Switch failed: %v %s", "切換失敗：%v %s"), err, strings.TrimSpace(string(out)))
	}
	return nil
}
