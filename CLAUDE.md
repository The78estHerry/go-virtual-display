# CLAUDE.md — go-virtual-display

給 Claude 的專案說明與建置紀錄。使用者看的說明在 `README.md`。所有文件、註解、commit 訊息一律用台灣繁體中文。

## 目的

讓 Windows（以及 Linux）在沒有實體螢幕時，以為接著一台回報自訂 EDID 的螢幕（Windows 0.3.0 起解析度可自選，預設 1920×1080；Linux 固定 1920×1440），並提供系統列圖示＋桌面小工具形式的一鍵軟體拔插開關。

## 已做的決定與原因

- **Windows 用現成的 Virtual Display Driver（VDD）＋自訂 EDID**（「路線 A」）。另一個選項是自己改 Microsoft IddSampleDriver（「路線 B」），沒採用，因為未簽章的驅動要開測試簽章模式，而這需要關 Secure Boot，畫面還有浮水印。VDD 已簽章，又支援 `user_edid.bin`。
- **Linux 一起做，Mac 不做**。
  - Linux 用核心內建的 `drm.edid_firmware` 加 sysfs `status`（`on`／`off`），完全不需要額外驅動。
  - Mac 只能透過私有 API `CGVirtualDisplay` 建立虛擬螢幕，無法吃原始 EDID，所以不做。
- **語言**：
  - 開關程式和 EDID 產生器用 Go，跨平台共用。使用者偏好 Go，其次 Python，能用 shell 解決更好。
  - 安裝腳本用各平台內建的 shell：Windows PowerShell 5.1、bash。
- **命名**：Go 專案一律加 `go-` 前綴，所以資料夾、Gitea repo、Go module 都叫 go-virtual-display。
- **介面語言**：英文和繁體中文（0.2.0 起）。依系統顯示語言決定預設（只有中文語系預設繁中，其他英文），安裝時讓使用者選。
  - 原因：使用者指出中文在很多電腦上無法使用。
  - 檔名一律英文，這條規則通用所有專案（見記憶 english-file-names）。
  - 原始碼註解、開發文件、commit 訊息仍用繁中，`make-edid` 是開發工具也維持中文。
- **開關介面**：系統列圖示＋桌面小工具（開關式），不做桌面捷徑。
  - 系統列用 `fyne.io/systray`，小工具用 Gio（`gioui.org`）。
  - 在 Windows 上兩者都不需要 cgo。

## 檔案

| 路徑 | 用途 |
|---|---|
| `internal/edid` | EDID 1.4 組裝（`Build`）、解析（`Parse`）、`CVTReducedBlanking`（cvt.go）、舊版固定參數 `Default`（DMT 1920×1440@60，Linux 版用） |
| `assets/` | 程式圖示：`icon.svg`（方案 A1，原始檔）、`icon.ico`、`icon-256.png`、`make-icon.ps1`（用 Edge headless 轉 PNG 再組 ICO，用 windres 產生 `cmd/vdswitch/rsrc_windows_amd64.syso`） |
| `internal/profile` | 解析度清單 `profiles.json`（`Load`／`Add`／`Remove`／`SetActive`）、14 組 `Presets`、每組的 `EDID()`、`Validate()` |
| `cmd/make-edid` | `-o` 產生、`-check` 檢查並印出 hex |
| `edid/edid_1920x1080.bin`、`edid_1920x1440.bin` | 產好的 EDID，commit 進 repo：前者是 Windows 安裝的預設，後者是 Linux 版固定使用（和 0.2.0 以前完全相同） |
| `internal/display` | `Controller` 介面（`Status`/`Plug`/`Unplug`），各平台各一個實作 |
| `internal/ui` | `Model`（狀態、輪詢、切換後等待）、`tray.go`、`widget.go`、`settings.go`、`autostart_*.go`、`platform_*.go`、`icon.go`（執行時繪製圖示） |
| `internal/i18n` | 介面語言：`T(英文, 繁中)`。順序是環境變數 `GO_VIRTUAL_DISPLAY_LANG`（測試用）→ 安裝時選的 → 系統顯示語言 |
| `cmd/vdswitch` | 主程式＋單一執行個體（Windows：具名 mutex＋event；Linux：flock） |
| `windows/install.cmd`／`uninstall.cmd` | 一鍵安裝／移除：用 `net session` 判斷權限，不足時以 `Start-Process -Verb RunAs` 重新啟動自己，再呼叫 `windows\scripts\install.ps1 -Launch`／`uninstall.ps1`。只放 ASCII，因為 cmd 用系統字碼頁讀檔 |
| `windows/scripts/` | `common.ps1`（常數）、`download-vdd.ps1`、`install.ps1`、`verify.ps1`、`uninstall.ps1`、`vdd_settings.xml` |
| `linux/` | `install.sh`、`uninstall.sh`（一鍵：一般使用者執行，root 部分自己 `sudo bash "$0"`）。其餘在 `linux/scripts/`：`verify.sh`、`vdswitch-helper`、polkit 規則、`initramfs-hook`、`.desktop` |
| `downloads/` | 下載的 VDD（gitignore） |
| `dist/` | 編譯產物（gitignore），`dist/release/` 是安裝包 |
| `release.ps1`、`RELEASING.md` | 產生 Release 安裝包；發佈步驟與各平台安裝包的檔案清單 |

## 常用指令

```
go test ./...
powershell -ExecutionPolicy Bypass -File .\build.ps1          # Windows 編譯 + 重產 EDID
GOOS=linux go vet ./internal/display ./internal/edid ./cmd/make-edid   # Linux 端在 Windows 上能檢查到的部分
make linux                                                    # 只能在 Linux 上（Gio 需要 cgo）
```

## 查到的關鍵事實（不要重查）

- **VDD 的版本和下載**：
  - GitHub Releases API 只列得到 2024 年的兩個舊版。
  - 正式版本 25.7.23 是從 winget manifest 找到的：`https://github.com/VirtualDrivers/Virtual-Display-Driver/releases/download/25.7.23/VDD.Control.25.7.23.zip`，SHA-256 `A701F227…9609154`（完整值在 `windows/common.ps1`）。
  - zip 裡的 `SignedDrivers/x86/VDD/` 實際上是 amd64 驅動（INF 寫 NTamd64），另外附 `Dependencies/devcon.exe`。
- **VDD 驅動的行為**（從 DLL 字串和 INF 確認）：
  - 硬體 ID `Root\MttVDD`，是 root-enumerated 裝置，所以要用 `devcon install MttVDD.inf Root\MttVDD` 建立，`pnputil` 無法建立 root 裝置。
  - 設定目錄預設 `C:\VirtualDisplayDriver`，可用登錄檔 `HKLM\SOFTWARE\MikeTheTech\VirtualDisplayDriver\VDDPATH` 指定。
  - 自訂 EDID：在 `vdd_settings.xml` 設 `<CustomEdid>true</CustomEdid>`，檔案命名為 `user_edid.bin` 放在設定目錄。驅動會檢查 checksum。
  - **驅動會改掉 EDID 的 byte 8–11**：廠商／產品代碼一律換成 `MTT`／`0x1337`，再重算 byte 127 的 checksum，其餘 byte 不變。`PreventSpoof=false` 也一樣。所以螢幕裝置是 `DISPLAY\MTT1337\...`，`verify.ps1` 比對時略過這幾個 byte。
  - 解析度仍以 XML 的 `<resolutions>` 為準，EDID 不會「模擬解析度」，所以 XML 只留一組（`vdswitch.exe --apply` 改寫成使用中的解析度），global refresh 只留 60。
  - 有 named pipe `\\.\pipe\MTTVirtualDisplayPipe`，可用來重新載入等，目前沒用到。
- **Windows 拔插**：
  - 用 `pnputil /enable-device|/disable-device /deviceid "Root\MttVDD"`，Windows 11 的 pnputil 支援 `/deviceid`。
  - 排程工作的動作用 `conhost.exe --headless pnputil.exe ...`，避免閃出主控台視窗。
  - 偵測到 `$RestartServices`（目前只有 `Synergy Core Daemon`）裡的服務時，排程工作多一個動作：`Start-Sleep 3` 後 `Restart-Service`。排程工作的多個動作會依序執行。
  - 狀態用 SetupAPI＋`CM_Get_DevNode_Status` 讀取：問題代碼 22（`CM_PROB_DISABLED`）表示拔掉，`DN_STARTED` 表示插入。不解析 pnputil 的輸出，因為輸出會依系統語系變成中文。
- **使用者的電腦**：
  - Windows 11 Pro 26200，zh-TW 語系，1920×1080 螢幕。
  - **系統 ANSI 字碼頁是 932（日文 Shift-JIS）**，不是 950。路徑分隔符號在主控台顯示成 ¥。
  - 顯示卡是 AMD Mobility Radeon HD 5000（很舊）。VDD 的算繪可能要靠 WARP，若虛擬螢幕黑畫面要往這裡查。

## 踩過的坑

- **Gio 事件迴圈裡不能同步呼叫會送訊息給該視窗的 Win32 API**（`ShowWindow`、`SetWindowPos` 等）。Gio 的視窗執行緒會等迴圈處理完事件，兩邊互等就卡死，視窗變成無回應。`widget.go` 收到 `ViewEvent` 時用 `go setupWidgetWindow(...)` 另開 goroutine。
- **systray 在 Windows 上建立視窗和處理訊息必須在同一個執行緒**。`fyne.io/systray` 的 `RunWithExternalLoop` 會在任意 goroutine 跑 `GetMessage`，不能用。改成在 `runtime.LockOSThread()` 的 goroutine 裡呼叫 `systray.Run`，Gio 的 `app.Main()` 留在 main goroutine。
- **模組根目錄不能有 `vendor/` 資料夾**，Go 會當成 vendoring 而報錯，所以下載目錄叫 `downloads/`。
- **PowerShell 5.1 讀取沒有 BOM 的 UTF-8 檔案會當成 ANSI（cp950）**，中文會變亂碼。所有 `.ps1` 都存成 UTF-8 with BOM＋CRLF，`.gitattributes` 固定換行。
- **`schtasks` 的錯誤訊息用系統字碼頁編碼**（不是 UTF-8），所以不放進錯誤字串，只回報 exit code。
- **`WScript.Shell` 的 `CreateShortcut` 用 ANSI 字碼頁處理路徑**。使用者的字碼頁是 932，「虛」不在裡面，變成 `?` 後 `Save()` 報 FileNotFoundException。0.2.0 起捷徑檔名一律英文（`Virtual Display Switch.lnk`），安裝與移除時也會刪掉 0.1.x 的中文捷徑。
- **PowerShell 的 `Get-UICulture` 不可靠**。使用者的顯示語言是繁中（`GetUserDefaultUILanguage` = 0x0404），`Get-UICulture` 卻回傳 en-US。所以腳本和 Go 都直接呼叫 `GetUserDefaultUILanguage`，取低 10 位元等於 0x04（LANG_CHINESE）就算中文。
- **Git Bash 的 `sed -i` 會把 CRLF 檔案改成 LF**。改 `.ps1`、`.cmd` 不要用 `sed -i`，改完要確認仍是 UTF-8 with BOM＋CRLF。
- **Gio 預設字型沒有中文**。Windows 版在中文介面才載入 `msjh.ttc`（微軟正黑體），並關閉系統字型掃描，啟動比較快；英文介面只用 Go 字型。Linux 版靠系統字型補中文。英文字比較寬，英文介面的小工具寬度是 288 dp（中文 248 dp）。

## 進度紀錄

- 2026-10-31：規劃（路線比較→選 A＋自訂 EDID→加入拔插開關→改用 Go→加入 Linux→改名 go-virtual-display）。完成以下項目：
  - EDID 產生器與測試
  - Windows 腳本
  - Linux 腳本
  - vdswitch（Windows 版已實際執行：小工具在右下角、不在工作列、圓角、中文正常；驅動未安裝時正確顯示錯誤）
- 2026-10-31：使用者覺得安裝步驟太多，加入一鍵安裝／移除：
  - 新增 `install.cmd`、`uninstall.cmd`
  - `install.ps1 -Launch`：缺執行檔時自動編譯；裝完先執行 verify，再透過 `explorer.exe` 以一般權限開啟開關程式，避免它繼承系統管理員權限
  - `uninstall.ps1` 會一併刪掉排程工作留下的空資料夾
- 2026-10-31：依使用者要求把 `install.cmd`／`uninstall.cmd` 搬進 `windows/`（只有 Windows 能用）。Linux 也改成一鍵：
  - **自動選接頭**：第一個 disconnected 的外接接頭，排除 eDP／LVDS／DSI。重新安裝時沿用 `/etc/go-virtual-display.conf` 的接頭
  - **自動處理 initramfs**：initramfs-tools 用 hook，dracut 用 `install_items`
  - **自動改 GRUB**：改 `/etc/default/grub` 的 `GRUB_CMDLINE_LINUX_DEFAULT`，第一次先備份成 `.go-virtual-display.bak`
    - 只移除本專案加的參數：指向我們 EDID 的 `drm.edid_firmware`，以及指定接頭的 `video=<接頭>:e`，使用者原有的 `video=...` 不會被動到
    - 加參數的 sed 要用 `|` 當分隔符號，因為參數裡有 `/`（第一版就是在這裡出錯）
    - 在 Git Bash 用假的 grub 檔模擬過：初次安裝、重複安裝、換接頭、移除都正確
  - **自動開啟開關程式**：裝完以一般使用者身分 `setsid` 開啟
- 2026-10-31：第一次實機執行 `windows\install.cmd`：驅動下載、安裝、排程工作都成功，但建立開始功能表捷徑時失敗（見踩過的坑的 WScript.Shell），已修正，待使用者重跑。
- 2026-10-31：重跑後安裝成功，小工具和拔插都正常。使用者同時用 Synergy 3（這台是子機，設在主機左邊）和 RustDesk：
  - **Synergy**：插入虛擬螢幕（排在真實螢幕左邊）後，從主機移過來的滑鼠落進虛擬螢幕的右邊緣，而不是真實螢幕。
    - 原因：Synergy 子機沒有在顯示設定改變後更新桌面大小。左邊界已變成 −1920，寬度卻還是舊的，算出的右邊緣變成 −1。
    - 使用者重新啟動 Synergy 後就正常，所以讓拔插的排程工作自動重新啟動 `Synergy Core Daemon` 服務，它會帶起 `synergy-core.exe`。
    - Synergy 的紀錄在 `C:\ProgramData\Synergy\logs\synergy.log`，設定在 `C:\ProgramData\Synergy\config\synergy.conf`。
  - **RustDesk**：遠端畫面上看不到本機的滑鼠移動。沒有虛擬螢幕時也一樣，所以是 RustDesk 本身的問題，不在本專案範圍。
- 2026-10-31：重跑安裝後 verify 出現紅字「登錄檔中找不到相同的 EDID」。原因是驅動改了廠商代碼（見上面的關鍵事實），所以 verify 改成略過 byte 8–11 和 127。
- 2026-10-31：使用者執行 uninstall 後檢查殘留，發現四項：
  - `C:\VirtualDisplayDriver` 裡還有本專案的檔案。原因是第二次安裝把第一次放的檔案當成「原有檔案」備份了，移除時又「還原」回去。
    - 修正：安裝時放一個標記檔 `.go-virtual-display`，標記檔存在就不再備份。
  - 已不存在的螢幕裝置 `DISPLAY\MTT1337\...`：移除時改用 `pnputil /remove-device` 刪掉。
  - `GraphicsDrivers\ScaleFactors` 裡 `MTT1337...` 的縮放設定：移除時一併刪掉。
  - 登錄檔 `HKLM\SOFTWARE\MikeTheTech\VirtualDisplayDriver\VDDPATH`：移除時刪掉，機碼空了也刪。
  - 沒有處理 `GraphicsDrivers\Configuration`／`Connectivity` 裡的螢幕排列紀錄：只有 SYSTEM 能讀，是 Windows 自己管理的快取，手動刪有風險。
- 2026-10-31：依使用者要求，`windows/`、`linux/` 最上層只留一鍵安裝／移除的兩個檔案，其餘搬進各自的 `scripts/`，讓使用者一眼看出要點哪個。
- 2026-10-31 17:09：使用者用新版 `uninstall.cmd` 移除後檢查，以下全部清除乾淨，Synergy 服務正常執行：VDD 裝置、驅動套件、殘留的 MTT1337 螢幕裝置、縮放設定、`C:\VirtualDisplayDriver`、`HKLM\SOFTWARE\MikeTheTech`、程式目錄、開始功能表捷徑、開機啟動、排程工作資料夾。
- 2026-10-31：實測自動重啟 Synergy：用開關拔插後都正常。但安裝完驅動自動插入的那一次沒有重啟，又出現老問題。
  - 修正：`common.ps1` 加上 `Restart-DisplayDependentServices`，裝好驅動後和移除裝置後都呼叫。
  - 第二次移除後檢查，同樣全部清乾淨。
- 2026-10-31 17:20 前後：重新安裝後實測，安裝完第一次自動插入時 Synergy 也正常。Synergy 紀錄裡安裝（17:21）、拔插、移除（17:24）後都有重新連線。移除後再檢查，同樣全部清乾淨。
- 2026-10-31：發佈 **v0.1.0**（annotated tag＋Gitea Release）。附件 `go-virtual-display-0.1.0-windows.zip` 是 `git archive` 的原始碼加上 `dist/windows/vdswitch.exe`，沒裝 Go 也能一鍵安裝。
  - 使用者認為放了太多用不到的檔案，要求只放安裝需要的。新增 `release.ps1` 和 `RELEASING.md`（打包步驟、安裝包內容、所有專案共用的打包準則），`common.ps1` 改成也能在安裝包配置下執行（`$Packaged`）。重包後 v0.1.0 的 tag 移到新 commit，並替換 Release 附件。
  - `download-vdd.ps1` 關掉進度條：Windows PowerShell 5.1 顯示進度條時下載非常慢。
- 2026-10-31 17:50：使用者從 Gitea 下載精簡版 v0.1.0 安裝包，解壓縮到桌面，安裝、測試、移除。移除後檢查全部清乾淨，Synergy 正常。
- 2026-10-31：0.1.1 小改版：使用者擔心刪掉安裝資料夾就無法移除，要求從系統列移除；討論後也登記到「設定 > 應用程式」（只是一筆登錄值，不是 MSI）。
  - 安裝時把 `uninstall.cmd`、`common.ps1`、`uninstall.ps1`、`launch-uninstall.ps1` 複製到 `C:\Program Files\go-virtual-display\uninstaller`，並寫入 `HKLM\...\Uninstall\go-virtual-display`。
  - 系統列的「解除安裝…」（`internal/ui/uninstall_windows.go`）先跳 MessageBox 確認，再執行 `launch-uninstall.ps1`。「設定 > 應用程式」的 UninstallString 也指向它。
  - `launch-uninstall.ps1` 先把 `uninstaller\` 複製到 `%TEMP%\go-virtual-display-uninstall`，再用 `Start-Process cmd -Verb RunAs -Wait` 執行裡面的 `uninstall.cmd`，結束後刪掉暫存資料夾。原因：移除會刪掉 Program Files 裡的資料夾，而 cmd 是逐行從磁碟讀批次檔的，執行中的 `.cmd` 被刪掉會出錯。
  - 版本號改成只寫在 `common.ps1` 的 `$AppVersion`，`release.ps1` 不再需要 `-Version`。
- 2026-10-31：0.1.1 還沒實測，使用者就要求做英文版，版本改成 0.2.0，0.1.1 的內容併入。
  - 新增 `internal/i18n`，程式所有介面文字改成 `i18n.T(英文, 繁中)`。
  - Windows 腳本用 `common.ps1` 的 `T` 函式。`install.ps1` 一開始讓使用者選語言（`-Language` 可直接指定），存到 `HKLM\SOFTWARE\go-virtual-display\Language`。
  - Linux 腳本共用 `linux/scripts/lang.sh` 的 `t` 函式。`install.sh` 互動執行時讓使用者選語言，用 `--lang` 傳給 root 階段，存進 `/etc/go-virtual-display.conf` 的 `LANGUAGE`。`vdswitch-helper` 只用英文。
  - 檔名改成英文：安裝包的 `說明.txt` 改成雙語的 `README.txt`，開始功能表捷徑改成 `Virtual Display Switch.lnk`。
  - 測試：Go vet／test 通過。用強制英文的副本跑過 `verify.ps1`、`install.ps1` 的開頭，輸出是英文。`lang.sh` 在各種 `LANG` 下判斷正確。小工具的英文畫面沒能在這台截圖確認：測試時小工具沒有出現，全螢幕截圖又會拍到使用者其他視窗，所以停止截圖。
- 2026-10-31 18:33：使用者從 0.2.0 安裝包安裝測試後，從「設定 > 應用程式」移除。紀錄顯示流程正確：Program Files 的 `launch-uninstall.ps1` → `%TEMP%` 副本的 `uninstall.ps1` → 結束後暫存資料夾已刪除。移除後檢查全部乾淨，包括語言登錄機碼、「應用程式」登記、新舊捷徑。
- 2026-10-31 18:36：使用者確認英文和繁中介面都正常，小工具的字沒有被截斷。再用系統列的「Uninstall…」移除，檢查同樣全部乾淨。發佈 **v0.2.0**。
- 2026-10-31：0.3.0 開發：可新增、切換、刪除解析度（只做 Windows，使用者要求 Linux 等 Windows 確認後再做）。
  - 預設清單由使用者選定 14 組（都是 60 Hz）；剛安裝只有 1920×1080@60。
  - 介面：系統列「解析度」子選單＋「管理解析度」視窗（`internal/ui/manager.go`）。手動輸入可填更新率。
  - 套用：開關程式存 `profiles.json` 後觸發排程工作 `Apply`。它以最高權限執行 `vdswitch.exe --apply`（`internal/display/apply_windows.go`）：產生 EDID、用 regexp 改寫 XML 的 `<resolutions>` 和 `<g_refresh_rate>`、插入狀態下 `pnputil /restart-device`，再把結果寫到 `apply-result.json`。開關程式輪詢這個檔案，最多 30 秒。
  - 系統列子選單的分隔線無法移除（`AddSeparator` 沒有回傳值），所以「管理解析度…」放最上面、分隔線固定，下面的解析度項目用 `Remove()` 重建。
  - 修正 Range Limits：水平頻率超過 255 kHz 時用 EDID 1.4 的 offset 旗標（之前會溢位）。
  - 移除時一併刪掉 `%APPDATA%\go-virtual-display`：之前會留下 widget.json，這次修正。
  - 使用者另外要求設計「夢境裡想著一個螢幕」的程式圖示，給「安裝的程式」使用。
    - 第一輪提了三個方案，使用者選 A（雲朵泡泡），要求拿掉小泡泡、螢幕畫面要彩色夢幻（提到彩虹小馬：有版權的角色不能用，改提原創版本）。
    - 第二輪提 A1 彩虹與雲、A2 原創彩虹獨角獸、A3 全像彩光，使用者定案 **A1**。
    - 嵌入方式：`go run github.com/tc-hib/go-winres` 被權限分類器擋下（執行外部下載的程式），改用電腦上已安裝的 llvm-mingw `windres` 把 `vdswitch.rc` 編成 `.syso`。
    - 圖示會自動出現在「安裝的程式」（DisplayIcon 指向 vdswitch.exe）和開始功能表捷徑上。系統列維持會變色的狀態圖示。
- 2026-10-31：0.3.0 第一次實測，插入後沒有出現虛擬螢幕：VDD 裝置是 Started，但沒有 `DISPLAY\MTT1337` 螢幕裝置，事件記錄也沒有錯誤。
  - 推測原因：CVT reduced blanking 的 1920×1080 實際更新率是 59.93 Hz，VDD 設定檔寫的是整數 60。0.2.0 的 1920×1440 用 DMT，剛好 60.000 Hz，所以正常。
  - 修正：新增 `edid.ExactRefresh`，微調總像素和總行數，讓「總像素 × 總行數 × 更新率」是 10000 的倍數，pixel clock 用的是 10 kHz 單位。例如 1920×1080 變成 2080×1125、140.40 MHz、剛好 60 Hz。測試要求每組預設都剛好是整數 Hz。
  - 為了確認原因，這個測試版在 `vdd_settings.xml` 開啟 `<logging>true</logging>`，記錄應該在 `C:\VirtualDisplayDriver\Logs`。`uninstall.ps1` 在沒有還原原本設定時，會一併刪掉 `Logs`。**發佈前要改回 false。**
  - 使用者在我重現測試時執行了移除，裝置當場消失，這是移除造成的，不是 bug。
- 2026-10-31：使用者關掉程式後不知道怎麼重新打開，原因是開始功能表捷徑是英文名稱，中文系統搜尋不到。
  - 安裝時多建一個桌面捷徑，移除時一併刪除。
  - 兩個捷徑都加上註解，中文介面時是中文。
  - 工作列無法由程式自動釘選（Windows 10／11 不允許），請使用者自己右鍵釘選。
  - **`WScript.Shell` 連捷徑註解也用 ANSI 字碼頁處理**，「虛」會變成 `?`。改用 `common.ps1` 的 `New-Shortcut`，透過 Add-Type 的 C# 直接呼叫 `IShellLinkW`。
- 待辦：
  - 0.3.0 發佈前把 `vdd_settings.xml` 的 logging 改回 false
  - Linux 版的解析度功能（Windows 確認後）：
    - 韌體檔名改成固定的 `go-virtual-display.bin`，開機參數不用隨解析度改
    - helper 新增 `edid <檔案>`：驗證 128 bytes、header、checksum
    - 用 debugfs 立即套用，背景重建 initramfs
    - `strip_params` 同時清掉舊檔名的參數
  - Linux 實機測試（需要使用者的 Linux 機器；WSL 沒有 DRM 而且這台沒裝 WSL）

## 給之後的 Claude

- 發新版本照 `RELEASING.md` 做：先改 `common.ps1` 的 `$AppVersion`，再用 `release.ps1` 產生精簡的 Windows 安裝包，各平台要包哪些檔案也記在那裡。改了安裝時用到的檔案，要同時更新 `release.ps1` 和 `RELEASING.md` 的清單。

- 改了 EDID 參數要重新產生 `.bin` 並一起 commit，`windows/scripts/verify.ps1` 和 `linux/scripts/verify.sh` 都是拿它來比對。
- 升級 VDD 版本時，要同時更新 `windows/scripts/common.ps1` 的版本和 SHA-256，並重新讀新版的 `vdd_settings.xml` 格式和 INF 的硬體 ID。
- commit 一律以 claude-bot 身分，推到 `gitea.littleherry.work/The78estWorld/go-virtual-display`。
