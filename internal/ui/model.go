// Package ui 是拔插開關的介面：系統列圖示與桌面小工具，兩者共用同一個 Model。
package ui

import (
	"sync"
	"time"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/display"
)

const (
	pollInterval   = 2 * time.Second
	switchTimeout  = 10 * time.Second
	switchInterval = 300 * time.Millisecond
)

// Snapshot 是某一刻的狀態，供介面繪製。
type Snapshot struct {
	State display.State
	Busy  bool  // 切換中
	Err   error // 最近一次查詢或切換的錯誤
}

// Model 保存虛擬螢幕狀態，定期查詢並在改變時通知所有介面。
type Model struct {
	ctrl display.Controller

	mu        sync.Mutex
	snap      Snapshot
	listeners []func(Snapshot)
}

// NewModel 建立 Model 並開始定期查詢狀態。ctrl 可以是 nil（平台初始化失敗時），此時 initErr 會顯示在介面上。
func NewModel(ctrl display.Controller, initErr error) *Model {
	m := &Model{ctrl: ctrl, snap: Snapshot{Err: initErr}}
	if ctrl != nil {
		m.refresh()
		go func() {
			for range time.Tick(pollInterval) {
				m.refresh()
			}
		}()
	}
	return m
}

// Subscribe 註冊狀態改變時的回呼，並立即以目前狀態呼叫一次。
func (m *Model) Subscribe(f func(Snapshot)) {
	m.mu.Lock()
	m.listeners = append(m.listeners, f)
	s := m.snap
	m.mu.Unlock()
	f(s)
}

// Snapshot 回傳目前狀態。
func (m *Model) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap
}

func (m *Model) set(update func(*Snapshot)) {
	m.mu.Lock()
	old := m.snap
	update(&m.snap)
	s := m.snap
	ls := append([]func(Snapshot){}, m.listeners...)
	m.mu.Unlock()
	if s.State != old.State || s.Busy != old.Busy || errText(s.Err) != errText(old.Err) {
		for _, f := range ls {
			f(s)
		}
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (m *Model) refresh() {
	if m.Snapshot().Busy {
		return
	}
	st, err := m.ctrl.Status()
	m.set(func(s *Snapshot) {
		if !s.Busy {
			s.State, s.Err = st, err
		}
	})
}

// Toggle 切換插入／拔掉；切換中再按會被忽略。
func (m *Model) Toggle() {
	if m.ctrl == nil {
		return
	}
	m.mu.Lock()
	if m.snap.Busy || m.snap.State == display.Unknown {
		m.mu.Unlock()
		return
	}
	target := display.Plugged
	if m.snap.State == display.Plugged {
		target = display.Unplugged
	}
	m.mu.Unlock()
	m.set(func(s *Snapshot) { s.Busy, s.Err = true, nil })

	go func() {
		var err error
		if target == display.Plugged {
			err = m.ctrl.Plug()
		} else {
			err = m.ctrl.Unplug()
		}
		st := display.Unknown
		if err == nil {
			err = errTimeout
			for deadline := time.Now().Add(switchTimeout); time.Now().Before(deadline); time.Sleep(switchInterval) {
				if s, e := m.ctrl.Status(); e == nil && s == target {
					st, err = s, nil
					break
				}
			}
		}
		if err != nil {
			st, _ = m.ctrl.Status()
		}
		m.set(func(s *Snapshot) { s.State, s.Busy, s.Err = st, false, err })
	}()
}

type timeoutError struct{}

func (timeoutError) Error() string { return "切換逾時，狀態沒有改變" }

var errTimeout error = timeoutError{}
