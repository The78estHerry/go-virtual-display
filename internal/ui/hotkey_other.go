//go:build !windows

package ui

// Linux 沒有統一的全域快捷鍵 API（X11 和 Wayland 不同，Wayland 甚至不允許），目前不支援。

const hotkeySupported = false

type hotkeyMgr struct{}

func startHotkey(Hotkey, func()) *hotkeyMgr { return &hotkeyMgr{} }
func (*hotkeyMgr) Set(Hotkey) error         { return nil }
func (*hotkeyMgr) Suspend()                 {}
func (*hotkeyMgr) Resume()                  {}
func (*hotkeyMgr) Status() (Hotkey, error)  { return Hotkey{}, nil }
