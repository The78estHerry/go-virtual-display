// Package ui 是拔插開關的介面：系統列圖示與桌面小工具，兩者共用同一個 Model。
package ui

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/The78estHerry/go-virtual-display/internal/display"
	"github.com/The78estHerry/go-virtual-display/internal/i18n"
	"github.com/The78estHerry/go-virtual-display/internal/profile"
)

const (
	pollInterval   = 2 * time.Second
	switchTimeout  = 10 * time.Second
	switchInterval = 300 * time.Millisecond
)

// Snapshot 是某一刻的狀態，供介面繪製。
type Snapshot struct {
	State    display.State
	Busy     bool         // 切換中（插入／拔掉或套用解析度）
	Applying bool         // 正在套用解析度
	Err      error        // 最近一次查詢或切換的錯誤
	Profiles profile.List // 存下的解析度與使用中的那一組
	// RebootPending 表示選了新的解析度，但要重新開機才會生效（Linux 沒有 debugfs 時）。
	// 不放在 Err：Err 每次輪詢都會被狀態查詢的結果蓋掉
	RebootPending bool
	// Synergy 服務的狀態與安裝資料夾（顯示 Synergy 圖示用，只有 Windows 會偵測）
	Synergy      display.SynergyState
	SynergyDir   string
	SynergyMajor int // 主要版號，例如 3；讀不到時是 0
}

// profilesKey 用來判斷解析度清單或使用中的那一組是否改變。
func profilesKey(l profile.List) string {
	var b strings.Builder
	b.WriteString(l.Active)
	for _, p := range l.Profiles {
		b.WriteString("|" + p.Key())
	}
	return b.String()
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
	m := &Model{ctrl: ctrl, snap: Snapshot{Err: initErr, Profiles: profile.Load()}}
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
	if s.State != old.State || s.Busy != old.Busy || s.Applying != old.Applying || s.RebootPending != old.RebootPending ||
		errText(s.Err) != errText(old.Err) || profilesKey(s.Profiles) != profilesKey(old.Profiles) ||
		s.Synergy != old.Synergy || s.SynergyDir != old.SynergyDir || s.SynergyMajor != old.SynergyMajor {
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
	syn, synDir := display.Synergy()
	synMajor := display.SynergyMajor(synDir)
	m.set(func(s *Snapshot) { s.Synergy, s.SynergyDir, s.SynergyMajor = syn, synDir, synMajor })
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

func (timeoutError) Error() string {
	return i18n.T("timed_out_the_state_did_not")
}

var errTimeout error = timeoutError{}

// ProfilesSupported 回傳這個平台能不能切換解析度（Windows、Linux）。
func (m *Model) ProfilesSupported() bool {
	_, ok := m.ctrl.(display.ProfileApplier)
	return ok
}

// UseProfile 讓虛擬螢幕改用 p 這組解析度；套用失敗時恢復原本使用中的那一組。
func (m *Model) UseProfile(p profile.Profile) {
	ap, ok := m.ctrl.(display.ProfileApplier)
	if !ok {
		return
	}
	m.mu.Lock()
	if m.snap.Busy || m.snap.Profiles.Active == p.Key() {
		m.mu.Unlock()
		return
	}
	prev := m.snap.Profiles.ActiveProfile()
	m.mu.Unlock()

	l, err := profile.SetActive(p)
	if err != nil {
		m.set(func(s *Snapshot) { s.Err = err })
		return
	}
	m.set(func(s *Snapshot) { s.Busy, s.Applying, s.Err, s.Profiles = true, true, nil, l })
	go func() {
		err := ap.ApplyActive()
		pending := false
		if errors.Is(err, display.ErrRebootRequired) {
			pending, err = true, nil // 已存好，只是要重新開機才生效：不還原使用中的解析度
		} else if err != nil {
			l, _ = profile.SetActive(prev)
		}
		st, serr := m.ctrl.Status()
		if err == nil {
			err = serr
		}
		m.set(func(s *Snapshot) {
			s.Busy, s.Applying, s.State, s.Err, s.Profiles = false, false, st, err, l
			if err == nil {
				s.RebootPending = pending
			}
		})
	}()
}

// AddProfile 把 p 加入清單。
func (m *Model) AddProfile(p profile.Profile) error {
	l, err := profile.Add(p)
	m.set(func(s *Snapshot) { s.Profiles = l })
	return err
}

// RemoveProfile 從清單刪除 p（使用中的不能刪）。
func (m *Model) RemoveProfile(p profile.Profile) error {
	l, err := profile.Remove(p)
	m.set(func(s *Snapshot) { s.Profiles = l })
	return err
}
