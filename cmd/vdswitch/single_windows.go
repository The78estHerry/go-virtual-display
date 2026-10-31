package main

import (
	"golang.org/x/sys/windows"
)

const (
	mutexName = `Local\go-virtual-display-vdswitch`
	eventName = `Local\go-virtual-display-show`
)

// acquireSingleInstance 以具名 mutex 確保只執行一份；
// 第二份啟動時觸發具名 event，請第一份顯示小工具後結束。
func acquireSingleInstance() (<-chan struct{}, bool) {
	mname, _ := windows.UTF16PtrFromString(mutexName)
	ename, _ := windows.UTF16PtrFromString(eventName)
	_, err := windows.CreateMutex(nil, false, mname)
	if err == windows.ERROR_ALREADY_EXISTS {
		if ev, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, ename); err == nil {
			windows.SetEvent(ev)
			windows.CloseHandle(ev)
		}
		return nil, false
	}
	ev, err := windows.CreateEvent(nil, 0, 0, ename) // 自動重設
	if err != nil {
		return nil, true
	}
	show := make(chan struct{}, 1)
	go func() {
		for {
			if s, _ := windows.WaitForSingleObject(ev, windows.INFINITE); s == windows.WAIT_OBJECT_0 {
				select {
				case show <- struct{}{}:
				default:
				}
			}
		}
	}()
	return show, true
}
