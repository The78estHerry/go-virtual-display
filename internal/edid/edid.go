// Package edid 組裝與解析 EDID 1.4 基本區塊（128 bytes）。
package edid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Size 是 EDID 基本區塊的長度。
const Size = 128

var header = []byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}

// Timing 是一組詳細時序（Detailed Timing Descriptor）。
type Timing struct {
	PixelClockKHz                int
	HActive, HFront, HSync       int
	HBack                        int
	VActive, VFront, VSync       int
	VBack                        int
	HSyncPositive, VSyncPositive bool
}

// HTotal 回傳水平總像素數。
func (t Timing) HTotal() int { return t.HActive + t.HFront + t.HSync + t.HBack }

// VTotal 回傳垂直總行數。
func (t Timing) VTotal() int { return t.VActive + t.VFront + t.VSync + t.VBack }

// RefreshHz 回傳更新率（Hz）。
func (t Timing) RefreshHz() float64 {
	return float64(t.PixelClockKHz) * 1000 / float64(t.HTotal()*t.VTotal())
}

// HFreqKHz 回傳水平掃描頻率（kHz）。
func (t Timing) HFreqKHz() float64 { return float64(t.PixelClockKHz) / float64(t.HTotal()) }

// DMT1920x1440at60 是 VESA DMT 的 1920×1440@60Hz 時序。
var DMT1920x1440at60 = Timing{
	PixelClockKHz: 234000,
	HActive:       1920, HFront: 128, HSync: 208, HBack: 344,
	VActive: 1440, VFront: 1, VSync: 3, VBack: 56,
	HSyncPositive: false, VSyncPositive: true,
}

// Params 是產生 EDID 所需的資訊。
type Params struct {
	Manufacturer string // 3 個大寫英文字母（PNP ID）
	ProductCode  uint16
	Serial       uint32
	Week, Year   int    // 製造週、年份（西元）
	Name         string // 螢幕名稱，最多 13 字元
	SerialText   string // 序號字串，最多 13 字元；空字串則不放
	WidthCM      int    // 實體尺寸
	HeightCM     int
	Timing       Timing // 首選（原生）時序
}

// Default 是本專案使用的 1920×1440 虛擬螢幕參數。
var Default = Params{
	Manufacturer: "VDS",
	ProductCode:  0x1440,
	Serial:       1,
	Week:         44,
	Year:         2026,
	Name:         "VirtualDisp",
	SerialText:   "GVD-1920x1440",
	WidthCM:      40,
	HeightCM:     30,
	Timing:       DMT1920x1440at60,
}

// Build 依參數產生 128 bytes 的 EDID。
func Build(p Params) ([]byte, error) {
	if err := validate(p); err != nil {
		return nil, err
	}
	e := make([]byte, Size)
	copy(e, header)

	m := p.Manufacturer
	id := uint16(m[0]-'A'+1)<<10 | uint16(m[1]-'A'+1)<<5 | uint16(m[2]-'A'+1)
	binary.BigEndian.PutUint16(e[8:], id)
	binary.LittleEndian.PutUint16(e[10:], p.ProductCode)
	binary.LittleEndian.PutUint32(e[12:], p.Serial)
	e[16] = byte(p.Week)
	e[17] = byte(p.Year - 1990)
	e[18], e[19] = 1, 4

	e[20] = 0xA5 // 數位輸入、每色 8 bit、DisplayPort
	e[21], e[22] = byte(p.WidthCM), byte(p.HeightCM)
	e[23] = 120  // gamma 2.2
	e[24] = 0x07 // sRGB 預設色域、首選時序為原生、連續頻率（搭配 Range Limits）
	putChromaticity(e[25:35])

	// 沒有 established timings；standard timings 放 1920×1440@60，其餘未使用
	for i := 38; i < 54; i += 2 {
		e[i], e[i+1] = 0x01, 0x01
	}
	if st, ok := standardTiming(p.Timing); ok {
		e[38], e[39] = st[0], st[1]
	}

	putDTD(e[54:72], p.Timing, p.WidthCM*10, p.HeightCM*10)
	putRangeLimits(e[72:90], p.Timing)
	putText(e[90:108], 0xFC, p.Name)
	if p.SerialText != "" {
		putText(e[108:126], 0xFF, p.SerialText)
	} else {
		e[108+3] = 0x10 // dummy descriptor
	}
	e[126] = 0 // 沒有擴充區塊
	e[127] = checksum(e[:127])
	return e, nil
}

func validate(p Params) error {
	if len(p.Manufacturer) != 3 {
		return errors.New("manufacturer 必須是 3 個字母")
	}
	for _, c := range []byte(p.Manufacturer) {
		if c < 'A' || c > 'Z' {
			return errors.New("manufacturer 只能用大寫英文字母")
		}
	}
	if len(p.Name) > 13 || len(p.SerialText) > 13 {
		return errors.New("名稱與序號字串最多 13 字元")
	}
	if p.Year < 1990 || p.Year > 1990+255 || p.Week < 0 || p.Week > 54 {
		return errors.New("製造年份或週數超出範圍")
	}
	t := p.Timing
	if t.PixelClockKHz <= 0 || t.PixelClockKHz/10 > 0xFFFF {
		return errors.New("pixel clock 超出範圍")
	}
	if t.HActive > 4095 || t.HTotal()-t.HActive > 4095 || t.VActive > 4095 || t.VTotal()-t.VActive > 4095 {
		return errors.New("解析度或 blanking 超出 DTD 可表示的範圍")
	}
	if t.HFront > 1023 || t.HSync > 1023 || t.VFront > 63 || t.VSync > 63 {
		return errors.New("porch 或 sync 寬度超出 DTD 可表示的範圍")
	}
	return nil
}

func checksum(b []byte) byte {
	var s byte
	for _, v := range b {
		s += v
	}
	return -s
}

// putChromaticity 寫入 sRGB 原色與白點座標（10 bit）。
func putChromaticity(b []byte) {
	enc := func(v float64) int { return int(v*1024 + 0.5) }
	rx, ry := enc(0.640), enc(0.330)
	gx, gy := enc(0.300), enc(0.600)
	bx, by := enc(0.150), enc(0.060)
	wx, wy := enc(0.3127), enc(0.3290)
	lo := func(v int) byte { return byte(v & 3) }
	b[0] = lo(rx)<<6 | lo(ry)<<4 | lo(gx)<<2 | lo(gy)
	b[1] = lo(bx)<<6 | lo(by)<<4 | lo(wx)<<2 | lo(wy)
	for i, v := range []int{rx, ry, gx, gy, bx, by, wx, wy} {
		b[2+i] = byte(v >> 2)
	}
}

func standardTiming(t Timing) ([2]byte, bool) {
	hz := int(t.RefreshHz() + 0.5)
	if t.HActive%8 != 0 || t.HActive < 256 || t.HActive > 2288 || hz < 60 || hz > 123 {
		return [2]byte{}, false
	}
	var aspect byte
	switch {
	case t.HActive*10 == t.VActive*16: // 16:10
		aspect = 0
	case t.HActive*3 == t.VActive*4: // 4:3
		aspect = 1
	case t.HActive*4 == t.VActive*5: // 5:4
		aspect = 2
	case t.HActive*9 == t.VActive*16: // 16:9
		aspect = 3
	default:
		return [2]byte{}, false
	}
	return [2]byte{byte(t.HActive/8 - 31), aspect<<6 | byte(hz-60)}, true
}

func putDTD(b []byte, t Timing, wmm, hmm int) {
	binary.LittleEndian.PutUint16(b, uint16(t.PixelClockKHz/10))
	hb := t.HTotal() - t.HActive
	vb := t.VTotal() - t.VActive
	b[2] = byte(t.HActive)
	b[3] = byte(hb)
	b[4] = byte(t.HActive>>8)<<4 | byte(hb>>8)
	b[5] = byte(t.VActive)
	b[6] = byte(vb)
	b[7] = byte(t.VActive>>8)<<4 | byte(vb>>8)
	b[8] = byte(t.HFront)
	b[9] = byte(t.HSync)
	b[10] = byte(t.VFront&0xF)<<4 | byte(t.VSync&0xF)
	b[11] = byte(t.HFront>>8)<<6 | byte(t.HSync>>8)<<4 | byte(t.VFront>>4)<<2 | byte(t.VSync>>4)
	b[12] = byte(wmm)
	b[13] = byte(hmm)
	b[14] = byte(wmm>>8)<<4 | byte(hmm>>8)
	flags := byte(0x18) // 數位、分離同步
	if t.VSyncPositive {
		flags |= 0x04
	}
	if t.HSyncPositive {
		flags |= 0x02
	}
	b[17] = flags
}

func putRangeLimits(b []byte, t Timing) {
	b[3] = 0xFD
	hz := int(t.RefreshHz() + 0.5)
	khz := int(t.HFreqKHz() + 0.5)
	b[5] = byte(hz - 10)
	b[6] = byte(hz + 15)
	b[7] = byte(khz - 30)
	b[8] = byte(khz + 10)
	b[9] = byte((t.PixelClockKHz+9999)/10000 + 1) // 單位 10 MHz，無條件進位並留餘裕
	b[10] = 0x01                                  // 只提供範圍限制，不支援 GTF/CVT 公式
	b[11] = 0x0A
	for i := 12; i < 18; i++ {
		b[i] = 0x20
	}
}

func putText(b []byte, tag byte, s string) {
	b[3] = tag
	for i := 5; i < 18; i++ {
		b[i] = 0x20
	}
	n := copy(b[5:], s)
	if n < 13 {
		b[5+n] = 0x0A
	}
}

// Info 是從 EDID 解析出的摘要。
type Info struct {
	Manufacturer string
	ProductCode  uint16
	Serial       uint32
	Version      string
	Name         string
	SerialText   string
	WidthCM      int
	HeightCM     int
	Preferred    Timing
}

// Parse 驗證並解析 128 bytes 的 EDID 基本區塊。
func Parse(e []byte) (Info, error) {
	var info Info
	if len(e) < Size {
		return info, fmt.Errorf("長度 %d bytes，至少要 %d", len(e), Size)
	}
	if !bytes.Equal(e[:8], header) {
		return info, errors.New("header 不正確")
	}
	var s byte
	for _, v := range e[:Size] {
		s += v
	}
	if s != 0 {
		return info, fmt.Errorf("checksum 錯誤（總和 mod 256 = %d）", s)
	}
	id := binary.BigEndian.Uint16(e[8:])
	info.Manufacturer = string([]byte{
		byte(id>>10&0x1F) + 'A' - 1, byte(id>>5&0x1F) + 'A' - 1, byte(id&0x1F) + 'A' - 1,
	})
	info.ProductCode = binary.LittleEndian.Uint16(e[10:])
	info.Serial = binary.LittleEndian.Uint32(e[12:])
	info.Version = fmt.Sprintf("%d.%d", e[18], e[19])
	info.WidthCM, info.HeightCM = int(e[21]), int(e[22])

	first := true
	for off := 54; off < 126; off += 18 {
		d := e[off : off+18]
		if d[0] != 0 || d[1] != 0 {
			if first {
				info.Preferred = parseDTD(d)
				first = false
			}
			continue
		}
		switch d[3] {
		case 0xFC:
			info.Name = parseText(d)
		case 0xFF:
			info.SerialText = parseText(d)
		}
	}
	if first {
		return info, errors.New("找不到詳細時序描述子")
	}
	return info, nil
}

func parseDTD(d []byte) Timing {
	hb := int(d[3]) | int(d[4]&0xF)<<8
	vb := int(d[6]) | int(d[7]&0xF)<<8
	t := Timing{
		PixelClockKHz: int(binary.LittleEndian.Uint16(d)) * 10,
		HActive:       int(d[2]) | int(d[4]>>4)<<8,
		VActive:       int(d[5]) | int(d[7]>>4)<<8,
		HFront:        int(d[8]) | int(d[11]>>6)<<8,
		HSync:         int(d[9]) | int(d[11]>>4&3)<<8,
		VFront:        int(d[10]>>4) | int(d[11]>>2&3)<<4,
		VSync:         int(d[10]&0xF) | int(d[11]&3)<<4,
	}
	t.HBack = hb - t.HFront - t.HSync
	t.VBack = vb - t.VFront - t.VSync
	if d[17]&0x18 == 0x18 {
		t.VSyncPositive = d[17]&0x04 != 0
		t.HSyncPositive = d[17]&0x02 != 0
	}
	return t
}

func parseText(d []byte) string {
	s := string(d[5:18])
	if i := strings.IndexByte(s, 0x0A); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " ")
}
