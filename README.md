# go-virtual-display

讓電腦在**沒有接實體螢幕**的情況下，以為接著一台 **1920×1440（4:3）** 的螢幕，而且這台螢幕回報的是我們自己產生的 EDID。另外附一個「拔插開關」：系統列圖示加桌面小工具，按一下就能插入或拔掉這台虛擬螢幕。

適合用在無頭主機的遠端桌面或串流、錄影，以及需要固定 4:3 畫面的測試。

支援 **Windows 10／11** 和 **Linux**（需要顯示卡上有一個空的接頭）。macOS 不支援。

## 運作方式

| | Windows | Linux |
|---|---|---|
| 虛擬螢幕來源 | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver)（已簽章的開源 IddCx 驅動，MIT） | 核心內建的 DRM EDID 覆寫，讓空接頭被當成接著螢幕 |
| 自訂 EDID | 驅動讀取 `C:\VirtualDisplayDriver\user_edid.bin` | 開機參數 `drm.edid_firmware=` |
| 拔插 | 停用／啟用驅動裝置（`pnputil`），由「最高權限」排程工作執行，按開關不跳 UAC | 寫入 `/sys/class/drm/<接頭>/status`（`on`／`off`），由 polkit 允許使用中的使用者免密碼執行 |

驅動本體**不放在這個 repo**，安裝腳本會下載固定版本（25.7.23）並驗證 SHA-256。

## EDID 內容

`edid/edid_1920x1440.bin`，由 `cmd/make-edid` 產生：

- EDID 1.4，128 bytes，沒有擴充區塊
- 首選時序：VESA DMT 1920×1440 @ 60 Hz（pixel clock 234.00 MHz；H 1920/128/208/344；V 1440/1/3/56）
- 廠商代碼 `VDS`、產品代碼 `0x1440`、名稱 `VirtualDisp`、序號字串 `GVD-1920x1440`、尺寸 40×30 cm、sRGB

```
go run ./cmd/make-edid -o edid/edid_1920x1440.bin      # 重新產生
go run ./cmd/make-edid -check edid/edid_1920x1440.bin  # 檢查並列出內容
```

要改名稱或時序，修改 `internal/edid/edid.go` 的 `Default`。

## Windows

### 需求

- Windows 10／11 x64（ARM64 也有驅動，但 Windows 11 24H2 以後可能需要測試簽章模式）
- [Microsoft Visual C++ Redistributable](https://learn.microsoft.com/cpp/windows/latest-supported-vc-redist)（缺少時驅動會報 `vcruntime140.dll not found`）
- 自己編譯時需要 Go 1.26 以上

### 安裝

1. 編譯（或使用已編好的 `dist\windows\vdswitch.exe`）：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\build.ps1
   ```
2. 以**系統管理員身分**開啟 PowerShell，執行：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\windows\install.ps1
   ```
   這一步會：
   - 下載驅動
   - 把 `vdd_settings.xml` 和 EDID 放到 `C:\VirtualDisplayDriver`（原本有檔案的話會先備份）
   - 安裝驅動
   - 建立排程工作 `\go-virtual-display\Plug`、`\go-virtual-display\Unplug`
   - 把開關程式裝到 `C:\Program Files\go-virtual-display`，並在開始功能表建立「虛擬螢幕開關」
3. 檢查（不需要系統管理員）：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\windows\verify.ps1
   ```

### 移除

以系統管理員身分執行 `windows\uninstall.ps1`。這會：
- 移除驅動、排程工作、開關程式和登入時自動啟動的設定
- 還原原本的 VDD 設定檔

> ⚠️ 更新顯示卡或晶片組驅動之前，建議先移除虛擬螢幕驅動（VDD 官方建議）。如果開機後黑畫面，請進安全模式移除。

## Linux

### 需求

- 顯示卡上有一個**沒接螢幕**的接頭（例如 HDMI-A-1）。沒有實體顯示卡的虛擬機或伺服器不適用。
- polkit（`pkexec`）
- 系統列圖示需要桌面支援 StatusNotifierItem：KDE 直接支援，GNOME 要裝 AppIndicator 擴充套件
- 自己編譯時需要 Go 和 Gio 的相依套件（見 `Makefile` 開頭）

### 安裝

```bash
make linux                                   # 編譯到 dist/linux/
sudo linux/install.sh                        # 列出接頭，找狀態是 disconnected 的
sudo linux/install.sh card0-HDMI-A-1         # 安裝
```

安裝腳本會印出要加到開機參數的設定，例如：

```
drm.edid_firmware=HDMI-A-1:edid/edid_1920x1440.bin video=HDMI-A-1:e
```

把它加到 `/etc/default/grub` 的 `GRUB_CMDLINE_LINUX_DEFAULT`，再執行 `update-grub`，然後重新開機。如果顯示卡驅動在 initramfs 階段就載入，還要重新產生 initramfs。有 debugfs 的系統，安裝時會先直接套用一次，不用重開機也能測試。

用 `linux/verify.sh` 檢查；移除用 `sudo linux/uninstall.sh`，之後再把開機參數刪掉。

## 拔插開關（vdswitch）

- **桌面小工具**：
  - 一張小卡片，右邊是滑動開關，按一下就切換
  - 綠色表示已插入，灰色表示已拔掉，橘色表示切換中，紅色表示發生錯誤（卡片上會顯示原因）
  - 拖曳左側文字區可以移動卡片
  - Windows 會記住卡片位置；Linux 由視窗管理員決定位置
- **系統列圖示**：
  - 圖示顏色和小工具相同
  - **左鍵點一下切換**
  - 右鍵選單：插入／拔掉、顯示桌面小工具、小工具固定在最上層、登入時自動啟動、結束
- 狀態每 2 秒同步一次。從其他地方（裝置管理員、VDD 的控制程式）改變狀態時，兩邊都會跟著更新。
- 只會執行一份。重複開啟時會把小工具叫出來。
- 設定存在 `%APPDATA%\go-virtual-display\widget.json`（Linux 是 `~/.config/go-virtual-display/widget.json`）。

## 專案結構

```
cmd/make-edid/        產生／檢查 EDID
cmd/vdswitch/         拔插開關主程式（含單一執行個體）
internal/edid/        EDID 1.4 組裝與解析（含測試）
internal/display/     各平台的狀態查詢與拔插
internal/ui/          系統列（fyne.io/systray）、小工具（Gio）、設定、自動啟動
edid/                 產生好的 EDID
windows/              Windows 安裝／驗證／移除腳本與 VDD 設定
linux/                Linux 安裝／驗證／移除腳本、小幫手、polkit 規則
build.ps1 / Makefile  編譯
```

## 另一種做法（沒有採用）

也可以自己修改 Microsoft 的 IddSampleDriver，把 EDID 寫死在驅動裡。這樣每個位元組都能自己控制，但必須自己編譯驅動。沒有正式簽章時要開測試簽章模式，而測試簽章模式需要關閉 Secure Boot，畫面上也會出現浮水印。Virtual Display Driver 已簽章，又支援自訂 EDID，所以採用它。
