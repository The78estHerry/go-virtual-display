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

### 一鍵安裝

點兩下 **`windows\install.cmd`**，UAC 詢問時按「是」，接著全部自動完成：

1. 下載驅動並驗證 SHA-256
2. 把 `vdd_settings.xml` 和 EDID 放到 `C:\VirtualDisplayDriver`（原本有檔案的話會先備份）
3. 安裝驅動
4. 建立拔插用的排程工作（`\go-virtual-display\Plug`、`\go-virtual-display\Unplug`），之後按開關不會再跳 UAC。有裝 Synergy 的話，每次拔插後會自動重新啟動它（見下方說明）
5. 把開關程式裝到 `C:\Program Files\go-virtual-display`，並在開始功能表建立「虛擬螢幕開關」。如果還沒編譯但有裝 Go，會先自動編譯
6. 執行檢查，並打開開關程式

可以重複執行，例如改了 EDID 之後再跑一次就會更新。

### 一鍵移除

點兩下 **`windows\uninstall.cmd`**，UAC 詢問時按「是」。這會：
- 移除驅動、排程工作、開關程式和登入時自動啟動的設定
- 還原原本的 VDD 設定檔

### 手動執行（進階）

一鍵安裝背後執行的是 `windows\` 裡的腳本，也可以在系統管理員 PowerShell 裡個別執行：

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1                # 編譯（需要 Go）
powershell -ExecutionPolicy Bypass -File .\windows\install.ps1      # 安裝
powershell -ExecutionPolicy Bypass -File .\windows\verify.ps1       # 檢查（不需要系統管理員）
powershell -ExecutionPolicy Bypass -File .\windows\uninstall.ps1    # 移除
```

### 搭配 Synergy

Synergy 子機只在啟動時讀取整個桌面的大小。插入或拔掉虛擬螢幕後，它會用舊的大小計算滑鼠位置，從主機移過來的滑鼠會落到錯的螢幕上。所以安裝時如果偵測到 Synergy 的服務（`Synergy Core Daemon`），拔插用的排程工作會在拔插 3 秒後自動重新啟動它，Synergy 會斷線一下再自動連回。

Synergy 是在本專案之後才安裝的話，重跑一次 `windows\install.cmd` 就會加上。其他需要重新啟動的服務可以加到 `windows\common.ps1` 的 `$RestartServices`。

> ⚠️ 更新顯示卡或晶片組驅動之前，建議先移除虛擬螢幕驅動（VDD 官方建議）。如果開機後黑畫面，請進安全模式移除。

## Linux

### 需求

- 顯示卡上有一個**沒接螢幕**的接頭（例如 HDMI-A-1）。沒有實體顯示卡的虛擬機或伺服器不適用。
- polkit（`pkexec`）
- 系統列圖示需要桌面支援 StatusNotifierItem：KDE 直接支援，GNOME 要裝 AppIndicator 擴充套件
- 自己編譯時需要 Go 和 Gio 的相依套件（見 `Makefile` 開頭）
- 開機參數的自動設定支援 GRUB；initramfs 的自動設定支援 initramfs-tools（Debian／Ubuntu）和 dracut（Fedora 等）

### 一鍵安裝

用一般使用者在終端機執行（不用加 sudo，需要時會自己詢問密碼）：

```bash
linux/install.sh
```

接著全部自動完成：

1. 如果還沒編譯但有裝 Go，會先編譯開關程式
2. 自動挑一個沒接螢幕的外接接頭（HDMI、DP 等，不會選筆電內建面板）。要指定的話用 `linux/install.sh card0-HDMI-A-1`
3. 安裝 EDID、拔插小幫手和 polkit 規則，之後按開關不用輸入密碼
4. 把 EDID 加進 initramfs
5. 在 `/etc/default/grub` 加上開機參數，並重新產生 GRUB 設定。第一次會備份成 `grub.go-virtual-display.bak`
6. 系統有 debugfs 的話直接套用一次，不用重開機。沒有的話要重新開機
7. 打開開關程式

可以重複執行，不會重複加參數。換接頭時會把舊接頭的參數清掉。

### 一鍵移除

```bash
linux/uninstall.sh
```

這會移除所有安裝的檔案、開機參數、initramfs 設定和登入時自動啟動的設定，並把接頭恢復成自動偵測。只會移除本專案加的參數，你原本的開機參數不會被動到。重新開機後就完全恢復原狀。

檢查用 `linux/verify.sh`。

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
windows/              Windows 一鍵安裝／移除（install.cmd、uninstall.cmd）、背後的 PowerShell 腳本與 VDD 設定
linux/                Linux 一鍵安裝／移除（install.sh、uninstall.sh）、檢查、小幫手、polkit 規則、initramfs hook
build.ps1 / Makefile  編譯
```

## 另一種做法（沒有採用）

也可以自己修改 Microsoft 的 IddSampleDriver，把 EDID 寫死在驅動裡。這樣每個位元組都能自己控制，但必須自己編譯驅動。沒有正式簽章時要開測試簽章模式，而測試簽章模式需要關閉 Secure Boot，畫面上也會出現浮水印。Virtual Display Driver 已簽章，又支援自訂 EDID，所以採用它。
