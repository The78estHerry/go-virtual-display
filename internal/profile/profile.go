// Package profile 管理使用者存的解析度（每一組產生一份 EDID），以及目前要給虛擬螢幕用哪一組。
//
// 清單存在 <使用者設定目錄>/go-virtual-display/profiles.json。Windows 上套用 EDID 的最高權限排程工作
// 以同一個使用者執行，所以讀得到同一份檔案。
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/The78estHerry/go-virtual-display/internal/edid"
	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

// Profile 是一組解析度與更新率。
type Profile struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Refresh int `json:"refresh"`
}

// Default 是剛安裝時唯一的一組。
var Default = Profile{1920, 1080, 60}

// Presets 是管理視窗裡可以直接加入的常見解析度。
var Presets = []Profile{
	{1920, 1080, 60}, {2560, 1440, 60}, {3840, 2160, 60}, {1280, 720, 60}, // 16:9
	{1920, 1200, 60}, {2560, 1600, 60}, {3840, 2400, 60}, // 16:10
	{640, 480, 60}, {1024, 768, 60}, {1600, 1200, 60}, {1920, 1440, 60}, {1280, 1024, 60}, // 4:3、5:4
	{2560, 1080, 60}, // 21:9
	{1080, 1920, 60}, // 直式
}

// 手動輸入的範圍：寬高受 DTD 的 12 bit 限制，更新率限制在常見範圍
const (
	MinSize, MaxSize       = 320, 4095
	MinRefresh, MaxRefresh = 24, 240
)

// Key 是用來比對與存檔的識別字串，例如 1920x1080@60。
func (p Profile) Key() string { return fmt.Sprintf("%dx%d@%d", p.Width, p.Height, p.Refresh) }

// Label 是顯示用的名稱，例如 1920×1080 @60Hz。
func (p Profile) Label() string { return fmt.Sprintf("%d×%d @%dHz", p.Width, p.Height, p.Refresh) }

// Size 是不含更新率的名稱，例如 1920×1080。
func (p Profile) Size() string { return fmt.Sprintf("%d×%d", p.Width, p.Height) }

// Timing 回傳這組解析度的時序：1920×1440@60 沿用 VESA DMT（和 0.2.0 以前相同），
// 其餘用 CVT reduced blanking，再微調成更新率剛好是整數（CVT 原本會是 59.93 Hz 這類數值，
// 和 Virtual Display Driver 設定檔的整數更新率不一致時，驅動不會建立螢幕）。
func (p Profile) Timing() (edid.Timing, error) {
	if p == (Profile{1920, 1440, 60}) {
		return edid.DMT1920x1440at60, nil
	}
	t, err := edid.CVTReducedBlanking(p.Width, p.Height, float64(p.Refresh))
	if err != nil {
		return t, err
	}
	return edid.ExactRefresh(t, p.Refresh)
}

// Validate 檢查是否在可以產生 EDID 的範圍內。
func (p Profile) Validate() error {
	if p.Width < MinSize || p.Width > MaxSize || p.Height < MinSize || p.Height > MaxSize {
		return fmt.Errorf(i18n.T("width_and_height_must_be_between"), MinSize, MaxSize)
	}
	if p.Refresh < MinRefresh || p.Refresh > MaxRefresh {
		return fmt.Errorf(i18n.T("refresh_rate_must_be_between_and"), MinRefresh, MaxRefresh)
	}
	if _, err := p.EDID(); err != nil {
		return errors.New(i18n.T("this_resolution_and_refresh_rate_are"))
	}
	return nil
}

// EDID 產生這組解析度的 128 bytes EDID。
func (p Profile) EDID() ([]byte, error) {
	t, err := p.Timing()
	if err != nil {
		return nil, err
	}
	params := edid.Default
	params.Timing = t
	params.SerialText = fmt.Sprintf("GVD-%dx%d", p.Width, p.Height) // 驅動會改掉廠商／產品代碼，靠序號區分每種解析度
	params.WidthCM, params.HeightCM = physicalCM(p.Width), physicalCM(p.Height)
	return edid.Build(params)
}

// physicalCM 以 96 DPI 換算實體尺寸，Windows 預設縮放就會是 100%。
func physicalCM(px int) int {
	return min(max(int(math.Round(float64(px)*2.54/96)), 1), 255)
}

// List 是存檔內容：所有存下的解析度，以及目前使用中的那一組。
type List struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
}

// ActiveProfile 回傳使用中的那一組；找不到時回傳第一組。
func (l List) ActiveProfile() Profile {
	for _, p := range l.Profiles {
		if p.Key() == l.Active {
			return p
		}
	}
	return l.Profiles[0]
}

// Has 回傳清單裡是否已有這一組。
func (l List) Has(p Profile) bool {
	return slices.ContainsFunc(l.Profiles, func(q Profile) bool { return q.Key() == p.Key() })
}

var mu sync.Mutex

// Path 回傳 profiles.json 的位置。
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "go-virtual-display", "profiles.json"), nil
}

// Load 讀取清單；檔案不存在、損壞或內容不合法時，回傳只有 Default 的清單。
func Load() List {
	def := List{Active: Default.Key(), Profiles: []Profile{Default}}
	path, err := Path()
	if err != nil {
		return def
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	var l List
	if json.Unmarshal(b, &l) != nil {
		return def
	}
	l.Profiles = slices.DeleteFunc(l.Profiles, func(p Profile) bool { return p.Validate() != nil })
	if len(l.Profiles) == 0 {
		return def
	}
	l.Active = l.ActiveProfile().Key()
	return l
}

// Update 讀取清單、以 f 修改後寫回；f 回傳錯誤時不寫入。
func Update(f func(*List) error) (List, error) {
	mu.Lock()
	defer mu.Unlock()
	l := Load()
	if err := f(&l); err != nil {
		return l, err
	}
	path, err := Path()
	if err != nil {
		return l, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return l, err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return l, err
	}
	return l, os.WriteFile(path, b, 0o644)
}

// Add 加入一組解析度（已存在時不重複加入）。
func Add(p Profile) (List, error) {
	if err := p.Validate(); err != nil {
		return Load(), err
	}
	return Update(func(l *List) error {
		if !l.Has(p) {
			l.Profiles = append(l.Profiles, p)
		}
		return nil
	})
}

// Remove 刪除一組解析度；使用中的不能刪。
func Remove(p Profile) (List, error) {
	return Update(func(l *List) error {
		if p.Key() == l.Active {
			return errors.New(i18n.T("the_resolution_in_use_cannot_be"))
		}
		l.Profiles = slices.DeleteFunc(l.Profiles, func(q Profile) bool { return q.Key() == p.Key() })
		return nil
	})
}

// SetActive 把某一組設為使用中（不在清單裡時一併加入）。
func SetActive(p Profile) (List, error) {
	if err := p.Validate(); err != nil {
		return Load(), err
	}
	return Update(func(l *List) error {
		if !l.Has(p) {
			l.Profiles = append(l.Profiles, p)
		}
		l.Active = p.Key()
		return nil
	})
}
