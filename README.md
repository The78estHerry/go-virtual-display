<p align="center"><img src="assets/icon-256.png" width="160" alt="go-virtual-display 圖示：夢的泡泡裡想著一台彩虹螢幕"></p>

# go-virtual-display

**繁體中文** | [English](README.en.md) | [日本語](README.ja.md)

[![PayPal](https://img.shields.io/badge/PayPal-paypal.me%2FThe78ester-00457C?logo=paypal&logoColor=white)](https://paypal.me/The78ester)

💖 如果這個工具對你有幫助，歡迎透過 [PayPal](https://paypal.me/The78ester) 贊助支持開發。

讓電腦在**沒有接實體螢幕**的情況下，以為接著一台螢幕，而且這台螢幕回報的是我們自己產生的 EDID。解析度可以自己選、新增和刪除，預設是 1920×1080。另外附一個「拔插開關」：系統列圖示加桌面小工具，按一下就能插入或拔掉這台虛擬螢幕。

適合用在無頭主機的遠端桌面或串流、錄影，以及需要特定解析度的測試。

支援 **Windows 10／11** 和 **Linux**（需要顯示卡上有一個空的接頭）。macOS 不支援。

Windows 版支援 **Synergy 3**（滑鼠鍵盤共用軟體）：螢幕配置改變後自動重新啟動 Synergy，並在系統列和小工具上顯示 Synergy 是否在運作（見「[搭配 Synergy 3](#搭配-synergy-3)」）。

## 運作方式

| | Windows | Linux |
|---|---|---|
| 虛擬螢幕來源 | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver)（已簽章的開源 IddCx 驅動，MIT） | 核心內建的 DRM EDID 覆寫，讓空接頭被當成接著螢幕 |
| 自訂 EDID | 驅動讀取 `C:\VirtualDisplayDriver\user_edid.bin` | 開機參數 `drm.edid_firmware=` 指向 `/lib/firmware/edid/go-virtual-display.bin`；切換解析度時寫入 debugfs 的 `edid_override` 立即生效 |
| 拔插 | 停用／啟用驅動裝置（`pnputil`），由「最高權限」排程工作執行，按開關不跳 UAC | 寫入 `/sys/class/drm/<接頭>/status`（`on`／`off`），由 polkit 允許使用中的使用者免密碼執行 |

驅動本體**不放在這個 repo**。
- **Windows 的 Release 安裝包**已附固定版本（25.7.23）的驅動檔案，安裝時驗證 SHA-256，**不需要網路**，原本的下載點消失也能安裝。
  - 只附需要的檔案：amd64／ARM64 的驅動和 `devcon.exe`，內容和官方 zip 完全相同，簽章仍然有效。
  - 授權聲明在安裝包的 `scripts\vdd\THIRD-PARTY-NOTICES.txt`（見「[第三方軟體](#第三方軟體)」）。
- **從原始碼安裝**時，會下載官方的 zip 並驗證 SHA-256。
- 每個驅動檔案的 SHA-256 都寫在 `windows\scripts\common.ps1`，安裝包裡的檔案被改過也會被發現。

## EDID 內容

每一組解析度產生一份 EDID（`internal/profile`）：

- EDID 1.4，128 bytes，沒有擴充區塊
- 首選時序用 VESA CVT reduced blanking，也就是現代 LCD 常用的時序。1920×1440@60 例外，沿用 VESA DMT，和舊版相同
- 名稱 `VirtualDisp`，序號字串 `GVD-<寬>x<高>`（例如 `GVD-1920x1080`），sRGB
- 實體尺寸以 96 DPI 換算（1920×1080 → 51×29 cm），Windows 預設縮放會是 100%
- Virtual Display Driver 會把廠商／產品代碼換成 `MTT`／`0x1337`，所以每種解析度靠序號字串區分。Windows 會分別記住每種解析度的排列和縮放

repo 裡有兩份產好的 EDID：
- `edid/edid_1920x1080.bin`：安裝時的預設（Windows、Linux）
- `edid/edid_1920x1440.bin`：0.3.0 以前的 Linux 版使用，留著當作 DMT 時序的參考

```
go run ./cmd/make-edid -w 2560 -h 1440 -hz 60 -o out.bin   # 產生指定解析度
go run ./cmd/make-edid -check edid/edid_1920x1080.bin      # 檢查並列出內容
```

## 解析度

- **切換**：系統列圖示按右鍵 →「解析度」，點一下要用的解析度。虛擬螢幕會短暫重新啟動，Windows 有裝 Synergy 的話也會自動重新啟動。在拔掉狀態下切換，下次插入時生效。
- **管理**：同一個子選單的「管理解析度…」會打開視窗：
  - 已存的解析度可以「使用」或「刪除」，使用中的不能刪
  - 常見解析度按一下就加入（都是 60 Hz）：1920×1080、2560×1440、3840×2160、1280×720、1920×1200、2560×1600、3840×2400、640×480、1024×768、1600×1200、1920×1440、1280×1024、2560×1080、1080×1920
  - 也可以手動輸入寬、高、更新率。寬高 320–4095，更新率 24–240 Hz；太高而無法用 EDID 表示時會顯示錯誤
- 剛安裝時只有 1920×1080@60。清單存在 `%APPDATA%\go-virtual-display\profiles.json`（Linux 是 `~/.config/go-virtual-display/profiles.json`）。覆蓋安裝（升級或重裝）時會保留；移除時會一併刪掉。
- **Windows**：切換由最高權限排程工作 `\go-virtual-display\Apply` 執行 `vdswitch.exe --apply`，它會產生 EDID、改寫 `C:\VirtualDisplayDriver` 的設定，所以不會跳 UAC。
- **Linux**：開關程式產生 EDID，透過 polkit 交給小幫手（`vdswitch-helper edid`，免密碼）。小幫手檢查長度、標頭和 checksum 後換掉 `/lib/firmware/edid/go-virtual-display.bin`，並在背景重建 initramfs。開機參數固定指向這個檔名，所以不用改 GRUB。
  - 系統有 debugfs（`/sys/kernel/debug/dri/<卡>/<接頭>/edid_override`）時立即生效
  - 沒有的話（例如開了 Secure Boot 的 lockdown），小工具會顯示「重新開機後套用」

## Windows

### 需求

- Windows 10／11 x64（ARM64 也有驅動，但 Windows 11 24H2 以後可能需要測試簽章模式）
- [Microsoft Visual C++ Redistributable](https://learn.microsoft.com/cpp/windows/latest-supported-vc-redist)（缺少時驅動會報 `vcruntime140.dll not found`）
- 沒裝 Go 的話，從 Gitea 的 Releases 下載 `go-virtual-display-<版本>-windows.zip`，裡面已附編好的開關程式。自己編譯則需要 Go 1.26 以上

### 一鍵安裝

點兩下 **`windows\install.cmd`**，UAC 詢問時按「是」，接著全部自動完成：

1. 選擇介面語言（English／繁體中文），直接按 Enter 用預設值（見下方「介面語言」）
2. 取得驅動並驗證 SHA-256：安裝包用內附的，原始碼才下載。這一步失敗時會直接停止，不會改動系統（重新安裝時也不會先移除舊版）
3. 把 `vdd_settings.xml` 和 EDID 放到 `C:\VirtualDisplayDriver`（原本有檔案的話會先備份）。EDID 依你存的解析度產生，新安裝是 1920×1080
4. 安裝驅動
5. 建立排程工作（`\go-virtual-display\Plug`、`Unplug`、`Apply`、`RestartServices`），之後按開關、切換解析度都不會再跳 UAC。有裝 Synergy 的話，螢幕配置改變後會自動重新啟動它（見下方說明）
6. 把開關程式裝到 `C:\Program Files\go-virtual-display`，並在開始功能表和桌面建立「Virtual Display Switch」捷徑（想釘選到工作列的話，在捷徑上按右鍵選「釘選到工作列」）。如果還沒編譯但有裝 Go，會先自動編譯
7. 複製移除用的檔案，並登記到「設定 > 應用程式」
8. 執行檢查，並打開開關程式

**已經安裝過時**（升級或重裝），安裝程式會先顯示目前的版本和要安裝的版本（降版時會提醒），問你要不要繼續：
- 回答 `y`：先移除目前的版本，再全新安裝。你新增的解析度、選的語言和「登入時自動啟動」會保留
- 直接按 Enter 或其他回答：取消，不做任何變更

### 移除

三種方式擇一，都會跳 UAC 詢問，按「是」：
- 在系統列圖示按右鍵，選「**解除安裝…**」
- 到「**設定 > 應用程式**」，找到「虛擬螢幕開關（go-virtual-display）」（英文介面是「Virtual Display Switch (go-virtual-display)」）並移除
- 點兩下 `windows\uninstall.cmd`

安裝時會把移除用的檔案複製到 `C:\Program Files\go-virtual-display\uninstaller`，所以就算刪掉當初的安裝資料夾，前兩種方式仍然可以移除。移除會：
- 移除驅動、排程工作、開關程式和登入時自動啟動的設定
- 移除 Windows 留下的虛擬螢幕裝置紀錄、縮放設定和 VDD 的登錄值
- 還原原本的 VDD 設定檔。安裝前沒有的話，就連同 `C:\VirtualDisplayDriver` 資料夾一起刪掉
- 刪掉你的設定：新增的解析度、小工具位置、選的語言（`%APPDATA%\go-virtual-display`、`HKLM\SOFTWARE\go-virtual-display`）。只想升級的話，直接執行新版的 `install.cmd` 就好，不用先移除

### 手動執行（進階）

一鍵安裝背後執行的是 `windows\scripts\` 裡的腳本，也可以在系統管理員 PowerShell 裡個別執行：

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1                # 編譯（需要 Go）
powershell -ExecutionPolicy Bypass -File .\windows\scripts\install.ps1      # 安裝
powershell -ExecutionPolicy Bypass -File .\windows\scripts\verify.ps1       # 檢查（不需要系統管理員）
powershell -ExecutionPolicy Bypass -File .\windows\scripts\uninstall.ps1    # 移除
```

### 搭配 Synergy 3

在 Synergy 3（服務名稱 `Synergy Core Daemon`）的子機上實測過。

- **自動重新啟動**：Synergy 子機只在啟動時讀取整個桌面的大小。插入或拔掉虛擬螢幕後，它會用舊的大小計算滑鼠位置，從主機移過來的滑鼠會落到錯的螢幕上。在「設定 > 顯示器」調整螢幕排列後也一樣，滑鼠會卡在舊的邊緣。
  - 開關程式偵測到螢幕配置改變（拔插、切換解析度、調整排列、接上其他螢幕），穩定約 2 秒後，透過排程工作 `RestartServices` 重新啟動 Synergy。Synergy 會斷線一下再自動連回。
  - 這個排程工作一律會建立，執行時才檢查 Synergy 在不在，所以之後才安裝或移除 Synergy 都不必重新安裝本程式。
  - 開關程式沒有執行時不會處理。
- **狀態顯示**：有安裝 Synergy 時，小工具的狀態下面會多一行「Synergy 3 執行中／未執行」，系統列選單也有「Synergy 3：執行中／未執行」（數字是 Synergy 的主要版號，從 `Synergy.exe` 的版本讀取，讀不到時只寫 Synergy），前面都有 Synergy 的圖示。執行中是彩色，未執行是灰色，沒安裝就不顯示。
  - 「執行中」是指 Synergy 的服務和實際共用滑鼠鍵盤的 `synergy-core.exe` 都在執行。從 Synergy 的系統列結束 Synergy 時，服務仍然會留著，但 `synergy-core.exe` 會停掉，這時顯示「未執行」。
  - Synergy 未執行時，螢幕配置改變也不會重新啟動它，避免把你關掉的 Synergy 又打開。
  - 圖示是執行時從你電腦上已安裝的 Synergy 讀取的，本專案沒有附帶 Synergy 的圖檔。
- 其他也只在啟動時讀取桌面大小的服務，可以加到 `windows\scripts\common.ps1` 的 `$RestartServices`。

本專案與 Synergy 的開發商 Symless 沒有任何關係，也沒有得到他們的認可或贊助（見「[商標](#商標)」）。

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
linux/install.sh        # 原始碼
./install.sh            # Release 安裝包（go-virtual-display-<版本>-linux-amd64.tar.gz 解壓縮後）
```

接著全部自動完成：

1. 如果還沒編譯但有裝 Go，會先編譯開關程式（只有原始碼需要；安裝包已附編好的）
2. 自動挑一個沒接螢幕的外接接頭（HDMI、DP 等，不會選筆電內建面板）。要指定的話用 `linux/install.sh card0-HDMI-A-1`
3. 依你存的解析度產生 EDID（新安裝是 1920×1080），安裝 EDID、小幫手和 polkit 規則，之後拔插、切換解析度都不用輸入密碼
4. 把 EDID 加進 initramfs
5. 在 `/etc/default/grub` 加上開機參數，並重新產生 GRUB 設定。第一次會備份成 `grub.go-virtual-display.bak`
6. 系統有 debugfs 的話直接套用一次，不用重開機。沒有的話要重新開機
7. 打開開關程式

**已經安裝過時**（升級或重裝），會先顯示目前的版本和要安裝的版本（降版時會提醒），問你要不要繼續：
- 回答 `y`：先移除目前的版本，再全新安裝。你新增的解析度、選的語言、接頭和「登入時自動啟動」都會保留
- 直接按 Enter 或其他回答：取消，不做任何變更
- 沒有終端機可以詢問時（例如從其他程式執行），直接覆蓋安裝

可以重複執行，不會重複加參數，也會保留你選的解析度。換接頭時會把舊接頭的參數清掉。從 0.3.0 以前的版本升級時，會把舊的 `edid_1920x1440.bin` 和它的開機參數換成新的檔名。

### 一鍵移除

```bash
linux/uninstall.sh
```

這會移除所有安裝的檔案、開機參數、initramfs 設定、登入時自動啟動的設定和你存的解析度（`~/.config/go-virtual-display`），並把接頭恢復成自動偵測。只會移除本專案加的參數，你原本的開機參數不會被動到。重新開機後就完全恢復原狀。

檢查用 `linux/scripts/verify.sh`。

## 圖示

程式圖示是「夢的泡泡裡想著一台螢幕」，螢幕畫面是彩虹與雲。它嵌在 `vdswitch.exe` 裡，「安裝的程式」、開始功能表和檔案總管都會顯示。系統列圖示仍是會隨狀態變色的螢幕：綠色插入、灰色拔掉、橘色切換中、紅色錯誤。

改圖示時編輯 `assets/icon.svg`，再執行 `assets/make-icon.ps1`。它會用 Windows 內建的 Edge 把 SVG 轉成各種尺寸，組成 `icon.ico`，再用 llvm-mingw 的 windres 產生 `cmd/vdswitch/rsrc_windows_amd64.syso`。產物都會 commit，所以平常編譯不需要這些工具。

## 介面語言

開關程式、安裝和移除的訊息都有英文和繁體中文兩種。

- **預設**：依系統的顯示語言決定。只有中文（不分繁簡）預設繁體中文，其他一律英文。
- **安裝時可以選**：Windows 的 `install.cmd`、Linux 的 `install.sh` 一開始會詢問，直接按 Enter 用預設值。重新安裝時，預設是上次選的語言。
- **選擇存在**：Windows 是登錄檔 `HKLM\SOFTWARE\go-virtual-display` 的 `Language`，Linux 是 `/etc/go-virtual-display.conf` 的 `LANGUAGE`（`en` 或 `zh-TW`）。移除時一併刪除。
- 檔名一律用英文（包括開始功能表捷徑「Virtual Display Switch」和安裝包裡的 `README.txt`），避免在非中文系統上出問題。

## 拔插開關（vdswitch）

- **桌面小工具**：
  - 一張小卡片，右邊是滑動開關，按一下就切換
  - 綠色表示已插入，灰色表示已拔掉，橘色表示切換中，紅色表示發生錯誤（卡片上會顯示原因）
  - 拖曳左側文字區可以移動卡片
  - Windows 會記住卡片位置；Linux 由視窗管理員決定位置
  - Windows：有安裝 Synergy 時，狀態下面多一行 Synergy 的圖示和「執行中／未執行」，卡片會高一點（見「[搭配 Synergy 3](#搭配-synergy-3)」）
  - Windows：卡片所在的螢幕被拔掉時（例如放在設為主螢幕的虛擬螢幕上），卡片會暫時移到主螢幕右下角，那個螢幕插回來後再回到原位。暫時移開期間自己拖曳過的話，就留在新的位置
- **系統列圖示**：
  - 圖示顏色和小工具相同
  - **左鍵點一下切換**
  - 右鍵選單：Synergy 狀態（只有 Windows，有安裝 Synergy 時）、插入／拔掉、解析度（只有 Windows）、顯示桌面小工具、小工具固定在最上層、登入時自動啟動、解除安裝…（只有 Windows）、結束
- 狀態每 2 秒同步一次。從其他地方（裝置管理員、VDD 的控制程式）改變狀態時，兩邊都會跟著更新。
- 只會執行一份。重複開啟時會把小工具叫出來。
- 設定存在 `%APPDATA%\go-virtual-display\widget.json`（Linux 是 `~/.config/go-virtual-display/widget.json`）。

## 專案結構

```
cmd/make-edid/        產生／檢查 EDID
cmd/vdswitch/         拔插開關主程式（含單一執行個體）
internal/edid/        EDID 1.4 組裝與解析（含測試）
internal/i18n/        介面語言判斷與英文／繁中翻譯
internal/profile/     解析度清單（profiles.json）與每組解析度的 EDID
internal/display/     各平台的狀態查詢與拔插
internal/ui/          系統列（fyne.io/systray）、小工具（Gio）、設定、自動啟動
edid/                 產生好的 EDID
assets/               程式圖示：icon.svg（原始檔）、icon.ico、make-icon.ps1（產生 .ico 與 cmd/vdswitch 的 .syso）
windows/              Windows 一鍵安裝／移除：只要點 install.cmd、uninstall.cmd
  scripts/            背後的 PowerShell 腳本與 VDD 設定
linux/                Linux 一鍵安裝／移除：只要執行 install.sh、uninstall.sh
  scripts/            檢查、小幫手、polkit 規則、initramfs hook、.desktop
build.ps1 / Makefile  編譯
release.ps1           產生 Release 安裝包（步驟與內容見 RELEASING.md）
```

## 另一種做法（沒有採用）

也可以自己修改 Microsoft 的 IddSampleDriver，把 EDID 寫死在驅動裡。這樣每個位元組都能自己控制，但必須自己編譯驅動。沒有正式簽章時要開測試簽章模式，而測試簽章模式需要關閉 Secure Boot，畫面上也會出現浮水印。Virtual Display Driver 已簽章，又支援自訂 EDID，所以採用它。

## 授權

本專案以 [MIT 授權](LICENSE) 釋出，可以自由使用、修改和散布，但不提供任何保證。內附的第三方軟體依各自的授權（見「[第三方軟體](#第三方軟體)」）。

如果這個工具對你有幫助，歡迎透過 [PayPal](https://paypal.me/The78ester) 贊助支持開發。

## 商標

- Synergy 是 Symless Ltd. 的商標，Synergy 的圖示（LOGO）屬於 Symless Ltd.。本專案只是為了相容而提到它的名稱，與 Symless 沒有任何關係，也沒有得到他們的認可或贊助。
- 本專案不包含、也不散布 Synergy 的圖示。開關程式顯示的 Synergy 圖示，是在執行時從使用者自己電腦上已安裝的 Synergy 程式讀取的。
- Windows 是 Microsoft Corporation 的商標。其他產品名稱屬於各自的擁有者。

## 第三方軟體

Windows 的 Release 安裝包內附以下檔案，內容未經修改，取自 Virtual Display Driver 官方發佈的 `VDD.Control.25.7.23.zip`：

| 檔案 | 軟體 | 授權 |
|---|---|---|
| `scripts\vdd\SignedDrivers\*\VDD\*` | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver) | MIT |
| `scripts\vdd\Dependencies\devcon.exe` | DevCon（Microsoft 簽章，[原始碼](https://github.com/microsoft/Windows-driver-samples/tree/main/setup/devcon)） | MS-PL |

授權全文在 `windows\scripts\THIRD-PARTY-NOTICES.txt`（安裝包裡是 `scripts\vdd\THIRD-PARTY-NOTICES.txt`）。本專案與這些軟體的作者沒有任何關係。
