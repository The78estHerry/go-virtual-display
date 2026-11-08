package ui

import (
	"os"
	"testing"
	"time"
	"unsafe"
)

// 用不常見的組合鍵測試，避免和使用者電腦上的快捷鍵衝突
const testHotkey = "Ctrl+Alt+Shift+F11"

func TestHotkeyRegisterAndConflict(t *testing.T) {
	h, _ := ParseHotkey(testHotkey)
	m := startHotkey(h, func() {})
	if _, err := m.Status(); err != nil {
		t.Fatalf("註冊 %s 失敗：%v", h, err)
	}
	// 同一組再註冊一次（模擬被其他程式占用）要回報錯誤
	other := startHotkey(h, func() {})
	if _, err := other.Status(); err == nil {
		t.Fatal("重複註冊應該失敗")
	}
	// 換成另一組成功後，原本那組就空出來了
	h2, _ := ParseHotkey("Ctrl+Alt+Shift+F10")
	if err := m.Set(h2); err != nil {
		t.Fatalf("改成 %s 失敗：%v", h2, err)
	}
	if err := other.Set(h); err != nil {
		t.Fatalf("空出來的 %s 應該可以註冊：%v", h, err)
	}
	// 改成被占用的組合時失敗，並保留原本的
	if err := m.Set(h); err == nil {
		t.Fatal("改成被占用的組合應該失敗")
	}
	if cur, err := m.Status(); cur != h2 || err != nil {
		t.Fatalf("失敗後應該保留 %s，得到 %s %v", h2, cur, err)
	}
	m.Suspend()
	m.Resume()
	if _, err := m.Status(); err != nil {
		t.Fatalf("Resume 後應該重新註冊：%v", err)
	}
	m.Suspend()
	other.Suspend()
}

// TestHotkeyPress 用 SendInput 模擬按下快捷鍵，會送出真的按鍵，所以要設環境變數才執行。
func TestHotkeyPress(t *testing.T) {
	if os.Getenv("GO_VIRTUAL_DISPLAY_TEST_PRESS") == "" {
		t.Skip("設 GO_VIRTUAL_DISPLAY_TEST_PRESS=1 才執行")
	}
	h, _ := ParseHotkey(testHotkey)
	pressed := make(chan struct{}, 1)
	m := startHotkey(h, func() { pressed <- struct{}{} })
	defer m.Suspend()
	if _, err := m.Status(); err != nil {
		t.Fatal(err)
	}
	vk, _ := virtualKey(h.Key)
	keys := []uint16{0x11, 0x12, 0x10, uint16(vk)} // VK_CONTROL、VK_MENU、VK_SHIFT
	sendKeys(keys, false)
	sendKeys(keys, true)
	select {
	case <-pressed:
	case <-time.After(2 * time.Second):
		t.Fatal("沒有收到快捷鍵")
	}
}

type keyInput struct {
	Type  uint32
	_     uint32
	Vk    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	Extra uintptr
	_     [8]byte
}

func sendKeys(vks []uint16, up bool) {
	in := make([]keyInput, len(vks))
	for i, vk := range vks {
		in[i] = keyInput{Type: 1, Vk: vk}
		if up {
			in[i].Flags = 2 // KEYEVENTF_KEYUP
			in[i].Vk = vks[len(vks)-1-i]
		}
	}
	user32.NewProc("SendInput").Call(uintptr(len(in)), uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
}
