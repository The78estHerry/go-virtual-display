// make-edid 產生或檢查 1920×1440 虛擬螢幕用的 EDID。
//
//	make-edid -o edid/edid_1920x1440.bin
//	make-edid -check edid/edid_1920x1440.bin
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/edid"
)

func main() {
	out := flag.String("o", "", "輸出的 .bin 路徑")
	check := flag.String("check", "", "要檢查的 EDID 檔案")
	flag.Parse()

	switch {
	case *check != "":
		b, err := os.ReadFile(*check)
		if err != nil {
			fail(err)
		}
		if len(b) != edid.Size {
			fmt.Printf("注意：檔案長度 %d bytes（只檢查前 %d bytes 的基本區塊）\n", len(b), edid.Size)
		}
		report(b)
	case *out != "":
		b, err := edid.Build(edid.Default)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("已寫入 %s\n", *out)
		report(b)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func report(b []byte) {
	info, err := edid.Parse(b)
	if err != nil {
		fail(err)
	}
	t := info.Preferred
	fmt.Printf("EDID %s，%d bytes，checksum 正確\n", info.Version, edid.Size)
	fmt.Printf("廠商 %s、產品代碼 0x%04X、序號 %d、名稱 %q、序號字串 %q、尺寸 %d×%d cm\n",
		info.Manufacturer, info.ProductCode, info.Serial, info.Name, info.SerialText, info.WidthCM, info.HeightCM)
	fmt.Printf("首選時序：%d×%d @ %.2f Hz，pixel clock %.2f MHz，水平 %.2f kHz\n",
		t.HActive, t.VActive, t.RefreshHz(), float64(t.PixelClockKHz)/1000, t.HFreqKHz())
	fmt.Printf("  H %d / %d / %d / %d（總 %d），V %d / %d / %d / %d（總 %d）\n",
		t.HActive, t.HFront, t.HSync, t.HBack, t.HTotal(), t.VActive, t.VFront, t.VSync, t.VBack, t.VTotal())
	for i := 0; i < len(b) && i < edid.Size; i += 16 {
		var sb strings.Builder
		for _, v := range b[i : i+16] {
			fmt.Fprintf(&sb, "%02X ", v)
		}
		fmt.Printf("  %02X: %s\n", i, strings.TrimSpace(sb.String()))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "錯誤：", err)
	os.Exit(1)
}
