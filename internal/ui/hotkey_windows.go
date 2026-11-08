package ui

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"gioui.org/io/key"
	"golang.org/x/sys/windows"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

var (
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procGetMessage        = user32.NewProc("GetMessageW")
	procPeekMessage       = user32.NewProc("PeekMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
)

const (
	modAlt      = 0x0001
	modControl  = 0x0002
	modShift    = 0x0004
	modWin      = 0x0008
	modNoRepeat = 0x4000
	wmHotkey    = 0x0312
	wmApp       = 0x8000
	hotkeyID    = 1
	pmNoRemove  = 0
)

type winMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
	_       uint32
}

const hotkeySupported = true

// hotkeyMgr 在一個鎖定的執行緒上用 RegisterHotKey 註冊全域快捷鍵：
// hwnd 傳 0 時 WM_HOTKEY 會送到註冊的執行緒，所以註冊、取消和收訊息都要在同一個執行緒。
// 其他 goroutine 透過 PostThreadMessage 請它換快捷鍵。
type hotkeyMgr struct {
	onPress func()
	tid     uint32

	mu      sync.Mutex
	current Hotkey // 設定的快捷鍵（暫停時不變）
	err     error  // 註冊失敗的原因（例如被其他程式占用）
	req     Hotkey // 下一次要註冊的（零值表示取消註冊）
	reply   chan error
}

func startHotkey(h Hotkey, onPress func()) *hotkeyMgr {
	m := &hotkeyMgr{onPress: onPress, current: h, reply: make(chan error)}
	ready := make(chan struct{})
	go m.loop(h, ready)
	<-ready
	return m
}

func (m *hotkeyMgr) loop(first Hotkey, ready chan<- struct{}) {
	runtime.LockOSThread()
	m.tid = windows.GetCurrentThreadId()
	var msg winMsg
	// 先呼叫一次 PeekMessage 讓這個執行緒建立訊息佇列，PostThreadMessage 才送得進來
	procPeekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmNoRemove)
	err := register(first)
	m.mu.Lock()
	m.err = err
	m.mu.Unlock()
	close(ready)
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		switch msg.Message {
		case wmHotkey:
			if msg.WParam == hotkeyID {
				go m.onPress()
			}
		case wmApp:
			procUnregisterHotKey.Call(0, hotkeyID)
			m.mu.Lock()
			req := m.req
			m.mu.Unlock()
			var err error
			if !req.IsZero() {
				err = register(req)
			}
			m.reply <- err
		}
	}
}

func register(h Hotkey) error {
	vk, ok := virtualKey(h.Key)
	if !ok {
		return fmt.Errorf("unsupported key %q", h.Key)
	}
	var mods uintptr = modNoRepeat
	if h.Mods&key.ModCtrl != 0 {
		mods |= modControl
	}
	if h.Mods&key.ModAlt != 0 {
		mods |= modAlt
	}
	if h.Mods&key.ModShift != 0 {
		mods |= modShift
	}
	if h.Mods&key.ModSuper != 0 {
		mods |= modWin
	}
	if r, _, e := procRegisterHotKey.Call(0, hotkeyID, mods, uintptr(vk)); r == 0 {
		return fmt.Errorf(i18n.T("%s is already used by another program (%v)", "%s 已被其他程式使用（%v）"), h, e)
	}
	return nil
}

// virtualKey 把 A～Z、0～9、F1～F12 轉成 Windows 的虛擬鍵碼。
func virtualKey(k string) (uint32, bool) {
	if !validHotkeyKey(k) {
		return 0, false
	}
	if len(k) == 1 {
		return uint32(k[0]), true // VK_A～VK_Z、VK_0～VK_9 和 ASCII 相同
	}
	n := 0
	for _, c := range k[1:] {
		n = n*10 + int(c-'0')
	}
	return uint32(0x70 + n - 1), true // VK_F1 = 0x70
}

// apply 請快捷鍵執行緒改註冊 h（零值表示取消註冊），等它完成。
func (m *hotkeyMgr) apply(h Hotkey) error {
	m.mu.Lock()
	m.req = h
	m.mu.Unlock()
	if r, _, e := procPostThreadMessage.Call(uintptr(m.tid), wmApp, 0, 0); r == 0 {
		return e
	}
	return <-m.reply
}

// Set 換成新的快捷鍵。註冊失敗時恢復原本的，並回傳錯誤。
func (m *hotkeyMgr) Set(h Hotkey) error {
	if err := m.apply(h); err != nil {
		m.mu.Lock()
		old := m.current
		m.mu.Unlock()
		oldErr := m.apply(old)
		m.mu.Lock()
		m.err = oldErr
		m.mu.Unlock()
		return err
	}
	m.mu.Lock()
	m.current, m.err = h, nil
	m.mu.Unlock()
	return nil
}

// Suspend 暫時取消註冊（設定快捷鍵的視窗開著時，按目前的組合鍵才收得到）；Resume 恢復。
func (m *hotkeyMgr) Suspend() { _ = m.apply(Hotkey{}) }

func (m *hotkeyMgr) Resume() {
	m.mu.Lock()
	cur := m.current
	m.mu.Unlock()
	err := m.apply(cur)
	m.mu.Lock()
	m.err = err
	m.mu.Unlock()
}

// Status 回傳設定的快捷鍵，以及註冊失敗的原因（nil 表示可以用）。
func (m *hotkeyMgr) Status() (Hotkey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current, m.err
}
