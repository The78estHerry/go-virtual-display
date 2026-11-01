package display

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// synergyCore 是實際共用滑鼠鍵盤的程式。Synergy 3 從系統列結束時只會停掉它和系統列程式，
// 服務（synergy-daemon.exe）仍然是 Running，所以只看服務會一直顯示運作中。
const synergyCore = "synergy-core.exe"

// synergyService 是 Synergy 的服務名稱，和 windows/scripts/common.ps1 的 $RestartServices 一致，改名時兩邊一起改。
const synergyService = "Synergy Core Daemon"

// Synergy 回傳 Synergy 的狀態和安裝資料夾（讀取圖示用）。服務和 synergy-core.exe 都在執行才算運作中。
// 不用 svc/mgr.Connect：它要求完整權限，一般使用者打不開。只要求查詢權限，一般使用者就能讀。
func Synergy() (SynergyState, string) {
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return SynergyNone, ""
	}
	defer windows.CloseServiceHandle(scm)
	name, _ := windows.UTF16PtrFromString(synergyService)
	svc, err := windows.OpenService(scm, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return SynergyNone, ""
	}
	defer windows.CloseServiceHandle(svc)

	state := SynergyStopped
	var st windows.SERVICE_STATUS
	if windows.QueryServiceStatus(svc, &st) == nil && st.CurrentState == windows.SERVICE_RUNNING && processRunning(synergyCore) {
		state = SynergyRunning
	}
	return state, serviceDir(svc)
}

// SynergyMajor 回傳 Synergy 的主要版號（例如 3），讀不到時回傳 0。
// 讀 Synergy.exe 的產品版本：其他執行檔（synergy-core.exe 等）是核心元件自己的版號（例如 1.20.3），不能用。
// 依檔案修改時間快取，Synergy 升級後會重新讀取。
func SynergyMajor(dir string) int {
	if dir == "" {
		return 0
	}
	path := filepath.Join(dir, "Synergy.exe")
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	majorMu.Lock()
	defer majorMu.Unlock()
	if majorCache.path == path && majorCache.mod.Equal(fi.ModTime()) {
		return majorCache.major
	}
	major := fileMajorVersion(path)
	majorCache.path, majorCache.mod, majorCache.major = path, fi.ModTime(), major
	return major
}

var (
	majorMu    sync.Mutex
	majorCache struct {
		path  string
		mod   time.Time
		major int
	}
)

// fileMajorVersion 讀出執行檔版本資源裡產品版本的主要版號。
func fileMajorVersion(path string) int {
	var zero windows.Handle
	size, err := windows.GetFileVersionInfoSize(path, &zero)
	if err != nil || size == 0 {
		return 0
	}
	buf := make([]byte, size)
	if windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])) != nil {
		return 0
	}
	var fixed *windows.VS_FIXEDFILEINFO
	var n uint32
	if windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\`, unsafe.Pointer(&fixed), &n) != nil || fixed == nil {
		return 0
	}
	return int(fixed.ProductVersionMS >> 16)
}

// processRunning 檢查是否有這個名稱的程序（不分大小寫）。列出程序名稱不需要特別權限。
func processRunning(name string) bool {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snap)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = windows.Process32First(snap, &pe); err == nil; err = windows.Process32Next(snap, &pe) {
		if strings.EqualFold(windows.UTF16ToString(pe.ExeFile[:]), name) {
			return true
		}
	}
	return false
}

// serviceDir 從服務的執行檔路徑（可能有引號和參數）取出所在資料夾。
func serviceDir(svc windows.Handle) string {
	var need uint32
	_ = windows.QueryServiceConfig(svc, nil, 0, &need)
	if need == 0 {
		return ""
	}
	buf := make([]byte, need)
	cfg := (*windows.QUERY_SERVICE_CONFIG)(unsafe.Pointer(&buf[0]))
	if windows.QueryServiceConfig(svc, cfg, need, &need) != nil {
		return ""
	}
	return dirOfCommand(windows.UTF16PtrToString(cfg.BinaryPathName))
}

// dirOfCommand 取出命令列第一個參數（執行檔）所在的資料夾。
func dirOfCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	var exe string
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			exe = cmd[1 : end+1]
		}
	} else if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		exe = cmd[:i+4] // 沒有引號時路徑可能有空白，以 .exe 為界
	} else {
		exe = cmd
	}
	if exe == "" {
		return ""
	}
	return filepath.Dir(exe)
}
