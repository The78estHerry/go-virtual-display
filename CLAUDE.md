# CLAUDE.md — go-virtual-display

給 Claude 的專案說明與建置紀錄。使用者看的說明在 `README.md`。所有文件、註解、commit 訊息一律用台灣繁體中文。

## 目的

讓 Windows（以及 Linux）在沒有實體螢幕時，以為接著一台回報自訂 EDID 的螢幕（解析度可自選，預設 1920×1080：Windows 0.3.0 起、Linux 0.4.0 起；Linux 0.3.0 以前固定 1920×1440），並提供系統列圖示＋桌面小工具形式的一鍵軟體拔插開關。

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
| `internal/edid` | EDID 1.4 組裝（`Build`）、解析（`Parse`）、`CVTReducedBlanking`（cvt.go）、舊版固定參數 `Default`（DMT 1920×1440@60，0.3.0 以前的 Linux 版用） |
| `assets/` | 程式圖示：`icon.svg`（方案 A1，原始檔）、`icon.ico`、`icon-256.png`、`make-icon.ps1`（用 Edge headless 轉 PNG 再組 ICO，用 windres 產生 `cmd/vdswitch/rsrc_windows_amd64.syso`） |
| `internal/profile` | 解析度清單 `profiles.json`（`Load`／`Add`／`Remove`／`SetActive`）、14 組 `Presets`、每組的 `EDID()`、`Validate()` |
| `cmd/make-edid` | `-o` 產生、`-check` 檢查並印出 hex |
| `edid/edid_1920x1080.bin`、`edid_1920x1440.bin` | 產好的 EDID，commit 進 repo：前者是安裝時的預設（Windows、Linux），後者是 0.3.0 以前 Linux 版使用的（和 0.2.0 以前完全相同） |
| `internal/display` | `Controller` 介面（`Status`/`Plug`/`Unplug`），各平台各一個實作 |
| `internal/ui` | `Model`（狀態、輪詢、切換後等待）、`tray.go`、`widget.go`、`settings.go`、`autostart_*.go`、`platform_*.go`、`icon.go`（執行時繪製圖示） |
| `internal/i18n` | 介面語言：`T(英文, 繁中)`。順序是環境變數 `GO_VIRTUAL_DISPLAY_LANG`（測試用）→ 安裝時選的 → 系統顯示語言 |
| `cmd/vdswitch` | 主程式＋單一執行個體（Windows：具名 mutex＋event；Linux：flock） |
| `windows/install.cmd`／`uninstall.cmd` | 一鍵安裝／移除：用 `net session` 判斷權限，不足時以 `Start-Process -Verb RunAs` 重新啟動自己，再呼叫 `windows\scripts\install.ps1 -Launch`／`uninstall.ps1`。只放 ASCII，因為 cmd 用系統字碼頁讀檔 |
| `windows/scripts/` | `common.ps1`（常數）、`download-vdd.ps1`、`install.ps1`、`verify.ps1`、`uninstall.ps1`、`vdd_settings.xml` |
| `linux/` | `install.sh`、`uninstall.sh`（一鍵：一般使用者執行，root 部分自己 `sudo bash "$0"`）。其餘在 `linux/scripts/`：`verify.sh`、`vdswitch-helper`、polkit 規則、`initramfs-hook`、`.desktop`。`linux/tests/`：Docker 模擬測試（`run.ps1` 一鍵執行） |
| `downloads/` | 下載的 VDD（gitignore） |
| `dist/` | 編譯產物（gitignore），`dist/release/` 是安裝包 |
| `release.ps1`、`RELEASING.md` | 產生 Release 安裝包；發佈步驟與各平台安裝包的檔案清單 |
| `release-linux.ps1` | 用 Docker（`debian:bullseye`）編譯 Linux 版並打包成 tar.gz，`-Ref` 指定版本 |

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
  - 排程工作 `RestartServices` 一律建立（0.4.2 起），動作是 `Get-Service <$RestartServices> -ErrorAction SilentlyContinue | Restart-Service -Force`，執行時才檢查服務在不在，之後才裝或移除 Synergy 都不必重裝。開關程式（`internal/ui/layout_windows.go`）每秒用 `EnumDisplayMonitors` 比對螢幕配置，改變並穩定 2 秒後執行它。0.4.1 以前是附加在 Plug／Unplug／Apply 後面的第二個動作（`Start-Sleep 3`），但在 Windows 設定裡調整排列時沒人重新啟動，Synergy 的滑鼠會卡在舊邊緣，0.4.2 改成這樣。
  - 狀態用 SetupAPI＋`CM_Get_DevNode_Status` 讀取：問題代碼 22（`CM_PROB_DISABLED`）表示拔掉，`DN_STARTED` 表示插入。不解析 pnputil 的輸出，因為輸出會依系統語系變成中文。
- **Synergy 狀態圖示**（0.4.2 起，`internal/display/synergy_windows.go`、`internal/ui/synergy_icon_windows.go`）：
  - 讀服務狀態不能用 `svc/mgr.Connect`（要求完整權限，一般使用者打不開），要用 `OpenSCManager(SC_MANAGER_CONNECT)`＋`OpenService(SERVICE_QUERY_STATUS|SERVICE_QUERY_CONFIG)`。
  - 圖示在執行時用 `PrivateExtractIconsW` 從使用者安裝的 `Synergy.exe` 讀取（`GetIconInfo`＋`GetDIBits` 轉成 NRGBA），**不把 Synergy 的 LOGO 放進 repo 或安裝包**：那是 Symless 的商標。README 三個語言版本都有「商標」段落說明。
  - 系統列的 Synergy 項目不能 `Disable()`：停用時 Windows 會把選單圖示畫成灰色，看不出執行中。
  - **只看服務狀態不夠**：Synergy 3 從它自己的系統列結束時，只停掉 `synergy-core.exe` 和系統列程式，服務（`synergy-daemon.exe`）仍然 Running（紀錄：`tray process exited, stopping core process`）。所以「執行中」＝服務 Running 且 `synergy-core.exe` 存在（`CreateToolhelp32Snapshot` 列程序）。
  - Synergy 有安裝但未執行時，`watchDisplayLayout` 不執行 `RestartServices`，避免把使用者關掉的 Synergy 帶起來（重新啟動服務會帶起 core）。
  - 版號：只顯示主要版號（「Synergy 3」）。讀 `Synergy.exe` 的產品版本（3.6.3）；`synergy-core.exe`／`synergy-daemon.exe` 是 1.20.3（核心元件的版號），`synergy-service.exe` 是 Node.js 的 22.x，都不能用。依檔案修改時間快取。
  - 小工具：有裝 Synergy 時狀態下面多一行（圖示＋「Synergy 執行中／未執行」），高度 64 → 84 dp，裝或移除 Synergy 時用 `win.Option` 改高度（另開 goroutine 呼叫）。
  - 服務名稱在 Go 的 `synergyService` 和 `common.ps1` 的 `$RestartServices` 各寫一次，改名時兩邊一起改。
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
- **`vdd_settings.xml` 的註解裡不能寫 `--`**（例如 `--apply`）。XML 格式錯誤時，VDD 不會報錯，只是不建立螢幕，非常難查。
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
- 2026-10-31 晚上：加了 `ExactRefresh` 之後，虛擬螢幕還是沒出現，**問題還沒解決**。目前狀態（0.3.0 安裝中、維持插入）：
  - VDD 裝置 `Root\MttVDD` 是 Started（OK），WUDFHost 有在跑，但沒有 `DISPLAY\MTT1337` 螢幕裝置，事件記錄沒有錯誤。
  - `logging=true` 也沒有產生 `C:\VirtualDisplayDriver\Logs`。控制管道 `\\.\pipe\MTTVirtualDisplayPipe` 送 GETSETTINGS 回傳 0 bytes。
  - 安裝後的 XML 和 0.2.0 幾乎一樣。用 Apply 工作切到 1920×1440（0.2.0 的 DMT 時序）也一樣沒有螢幕，所以應該不是 EDID 時序的問題。
  - 測試時我把 `%APPDATA%\go-virtual-display\profiles.json` 的 active 改成 `1920x1440@60`（清單有 1920×1080 和 1920×1440），已請使用者之後切回 1920×1080。
  - 主要推測：今天反覆安裝、移除了約 10 次，Windows 的 indirect display 狀態卡住了（上次開機是 10/26 17:44）。使用者正要**重開機（不移除）**。
  - 最後一次確認虛擬螢幕正常，是 17:20 前後（0.1.x 的 Synergy 測試）。也請使用者回想 0.2.0 測試時有沒有看到虛擬螢幕。
  - **重開機後接續**：
    1. 請使用者看「設定 > 顯示器」有沒有虛擬螢幕，我也檢查 `Get-PnpDevice` 有沒有 `DISPLAY\MTT1337`。
    2. 有出現：是系統狀態問題，在 README／安裝包說明加上「異常時請重開機」，接著讓使用者驗證 0.3.0 的其餘項目。
    3. 還是沒有：想辦法拿到 VDD 的記錄（登錄檔設定或控制管道），或拿 v0.2.0 安裝包對照，逐一找出差異。
- 2026-11-01：**找到真正原因**。重開機後 0.3.0 還是沒有螢幕，使用者改裝 0.1.1 就正常。
  - 在 0.1.1 上逐一替換設定檔測試（以系統管理員執行的腳本，換檔→`pnputil /restart-device`→看有沒有 `DISPLAY\MTT1337`→還原）：
    - LF 換行、logging 行尾註解、1080 的 XML＋0.3.0 的 EDID：都正常
    - 0.3.0 的 XML（不論 LF 或 CRLF）：沒有螢幕
  - 原因：0.3.0 的 `vdd_settings.xml` 第 3 行註解寫了 `vdswitch.exe --apply`。**XML 註解不能有 `--`**，整個檔案格式錯誤，驅動不讀設定，所以不建立螢幕、也不寫記錄、沒有任何錯誤。之前推測的 EDID 時序和系統狀態都不是原因（`ExactRefresh` 仍保留，讓更新率剛好是整數）。
  - 修正：拿掉註解裡的 `--`，logging 改回 false。`setVDDResolution` 寫入前用 `encoding/xml` 檢查格式，測試也會檢查範本格式。`.gitattributes` 加上 `*.xml text eol=crlf`，0.3.0 之前的安裝包是 CRLF，之後被改成 LF 了（實測 LF 也能用，只是保持一致）。
  - 重新產生測試用安裝包 `dist\release\go-virtual-display-0.3.0-windows.zip`，等使用者移除 0.1.1 後重裝測試。
  - 0.3.0 還沒驗證的項目：切換解析度和管理視窗、桌面捷徑和搜尋「虛擬螢幕」、新圖示、安裝時語言選單的排版（`59b0f01` 改用 `Console.ReadLine`）、移除是否乾淨。通過後照 `RELEASING.md` 打 v0.3.0 tag 並發佈。
  - 另外要查：0.1.1 重開機後插入時，虛擬螢幕變成主螢幕（在 (0,0)，實體螢幕被排到右邊）。之後裝 0.3.0 時實體螢幕維持主螢幕，可能是 Windows 沿用上次的排列，先觀察。
- 2026-11-01：使用者用修正後的安裝包（英文介面）安裝 0.3.0，虛擬螢幕正常出現。驗證通過：切換解析度、管理視窗、桌面捷徑、新圖示，移除後全部乾淨。英文介面搜尋「Virtual Display Switch」找得到，搜尋中文找不到，這是預期的行為。
- 2026-11-01：使用者要求在已安裝時重複安裝要先提醒：
  - `install.ps1` 一開始先看「應用程式」登記的 `DisplayVersion`，或程式資料夾是否存在（0.1.0 沒有登記）。不看 VDD 裝置，因為可能是使用者自己另外裝的。
  - 已安裝時顯示目前版本和要安裝的版本（相同版本或降版會另外提示），問 `y/N`。回答 y 才執行套件裡的 `uninstall.ps1 -KeepUserSettings`，再全新安裝；其他回答就取消。
  - `-KeepUserSettings`：不刪 `%APPDATA%\go-virtual-display`（解析度清單、小工具位置）和 `HKLM\SOFTWARE\go-virtual-display`（語言）。一般移除仍會全部刪掉。
  - 使用者實測通過：提醒有出現，重新安裝後解析度還在。之後一般移除，檢查全部乾淨（`%APPDATA%`、語言機碼也刪了，`%TEMP%` 沒有殘留），Synergy 正常。
  - Git Credential Manager 裡的 claude-bot token 在 10/31 23:35 之後不見了（原因不明，可能是某次驗證失敗被 GCM 自動刪除）。使用者提供新 token（scope：write:repository／organization／package，沒有 read:user），用 `git credential approve` 存回去。Claude Code 裡有 `GCM_INTERACTIVE=never`，不會跳登入視窗，token 不見時 push 會一直卡住，要請使用者提供 token 或在一般終端機登入。
- 2026-11-01：發佈 **v0.3.0**（annotated tag＋Gitea Release id 14，附件 `go-virtual-display-0.3.0-windows.zip`，SHA-256 `325396E4…9C1F0082`）。
- 2026-11-01：依使用者要求，用 v0.3.0 的程式碼補上 Linux 安裝包 `go-virtual-display-0.3.0-linux-amd64.tar.gz`（SHA-256 `1c6bf10f…be717af8`），附加到同一個 Release，並更新 Release 說明。
  - 新增 `release-linux.ps1`：用 Docker Desktop 的 `debian:bullseye` 編譯（glibc 2.31）。Debian 11 的安全更新已搬到 `archive.debian.org`，要改 apt 來源。
  - 安裝包沿用原始碼的目錄配置（`linux/`、`edid/`、`dist/linux/vdswitch`），0.3.0 的腳本不用改。
  - 只在 `ubuntu:22.04` 容器裡驗證過：腳本語法、`ldd` 沒有缺少的函式庫、執行檔可以啟動（沒有桌面所以無法顯示）。**還沒有實機測試**。
  - 已知問題：0.3.0 的 Linux 版系統列文字顯示 `profile.Default` 的 1920×1080，實際是 1920×1440。
  - Git Bash 裡的 Python 讀 stdin、印出中文會用 cp932，看起來像亂碼。要用 `PYTHONUTF8=1` 並從檔案讀寫，不要從 stdin 讀中文。
- 使用者問 icon 三個方案的比較網頁：上一個工作階段的暫存資料夾已經清掉，只剩定案的 A1（`assets/icon.svg`）。
- 2026-11-01：使用者要求 Gitea 的 README 顯示程式圖示：README 最上方放 `assets/icon-256.png`（相對路徑，Gitea 會直接顯示）。
- 2026-11-01：0.4.0 開發：**Linux 版也能切換解析度**（`$AppVersion` 已改成 0.4.0，還沒發佈）。
  - 開關程式的解析度選單和管理視窗本來就是跨平台的（看 `ProfileApplier`），Linux 只要實作 `ApplyActive`。
  - `ApplyActive`：產生 EDID，`pkexec vdswitch-helper edid` 從 **stdin** 傳入。用 stdin 而不是檔案路徑，root 就不用讀使用者指定的路徑。
  - 小幫手 `edid`：讀 129 bytes 判斷是否剛好 128、檢查 header 和 checksum，寫到 `/lib/firmware/edid/go-virtual-display.bin`。
    - 有 debugfs 時寫進 `edid_override`；插入中的話先 `off` 再 `on`，因為 force 狀態沒變時核心不會重新讀取模式。
    - 印出 `applied` 或 `reboot-required`。
    - 背景用 `setsid flock … update-initramfs -u`（或 `dracut -f`）重建 initramfs。輸出要全部導走，否則 Go 的 `CombinedOutput` 會等它結束。
  - `reboot-required` 對應 `display.ErrRebootRequired`：不算失敗，不還原使用中的解析度。`Snapshot.RebootPending` 讓小工具顯示「重新開機後套用」。不能放在 `Err`，因為 `Err` 每 2 秒的輪詢會被蓋掉。
  - `vdswitch --write-edid <檔案>`（Linux）：install.sh 的一般使用者階段用它依 `~/.config/go-virtual-display/profiles.json` 產生 EDID，用 `--edid` 傳給 root 階段。優先順序：使用者的 → 重新安裝時原本的 → 預設的 1080。
  - 升級：`strip_params` 和 uninstall 用 `(go-virtual-display|edid_1920x1440)\.bin` 同時清掉新舊檔名，並刪掉舊的 `edid_1920x1440.bin`。uninstall 也刪 `~/.config/go-virtual-display`，和 Windows 一致。
  - 安裝包改成扁平配置：`install.sh`、`uninstall.sh` 放最上層，其餘在 `scripts/`。`install.sh` 依 `scripts/edid_1920x1080.bin` 是否存在判斷自己在安裝包裡。`release-linux.ps1` 依版本自動選配置，`-Ref v0.3.0` 仍會產生舊配置。
  - 測試（都在 Docker 的 `ubuntu:22.04`）：
    - 模擬測試：把 `/sys`、`/lib/firmware`、`/etc` 等換成假目錄，40 項全部通過。涵蓋從 0.3.0 升級、重新安裝、小幫手的立即套用和需要重開機兩種情況、拒絕錯誤的 EDID、verify、移除、安裝包配置。
    - `release-linux.ps1` 在容器裡用 cgo 編譯成功，所以 Linux 的介面程式碼也能編譯。
    - 真的執行檔 `--write-edid`：預設產生 1080，有 profiles.json 時依使用者選的產生。
    - **沒測到**：一般使用者階段的 sudo 流程、真正的 DRM、debugfs、polkit、Gio 的畫面。
  - 測試用安裝包：`dist\release\go-virtual-display-0.4.0-linux-amd64.tar.gz`。
- 2026-11-01：依使用者要求先發佈 **v0.4.0**，Release 說明標明 Linux 沒有實機測過（Gitea Release id 15）。附件：
  - `go-virtual-display-0.4.0-windows.zip`（SHA-256 `75E9D119…FE3ACD43`）：Windows 只改了共用的 `model.go`，沒有重新實機測試
  - `go-virtual-display-0.4.0-linux-amd64.tar.gz`（SHA-256 `2292AD6D…01BFB18E`）
- 2026-11-01：0.4.1 開發：Linux 的 `install.sh` 也加上「已安裝時先提醒」（`$AppVersion` 已改成 0.4.1，還沒發佈）。
  - 版本號：安裝包是 `release-linux.ps1` 寫的 `scripts/VERSION`；原始碼則用 sed 讀 `windows/scripts/common.ps1` 的 `$AppVersion`，仍然只有一個來源。安裝時寫進 `/etc/go-virtual-display.conf` 的 `VERSION=`，0.4.0 以前沒有記錄，顯示「不明（0.4.0 以前）」。
  - 判斷已安裝：`/etc/go-virtual-display.conf` 存在（每個版本都會寫）。在一般使用者階段詢問，比較版本用 `sort -V`。
  - 回答 y：先記下原本的接頭，執行同一個資料夾裡的 `uninstall.sh --keep-user-settings`，再安裝。沒有指定接頭時沿用原本的，因為移除會刪掉設定檔。語言在 `lang.sh` 載入時已從設定檔讀出，所以語言選單的預設值仍是原本的。
  - `--keep-user-settings` 保留 `~/.config/go-virtual-display` 和登入時自動啟動，並且不印「移除完成」。Windows 的 `-KeepUserSettings` 仍會刪掉 Run 機碼，兩邊不一致；Linux 保留是因為它也算使用者的設定。
  - 沒有終端機（`[ -t 0 ]` 不成立）時不詢問，直接覆蓋安裝（install.sh 可以重複執行）。
  - 測試：`linux/tests/run.ps1` 在 Docker 跑兩支腳本。`simulate.sh` 40 項（原本的），`reinstall.sh` 26 項（新增）：一般使用者＋真的 sudo、用 `script` 提供終端機，涵蓋 n 取消、y 移除再安裝（保留接頭、語言、設定）、降版、升級、沒有 VERSION、沒有終端機、一般移除。全部通過。
- 2026-11-01：依使用者要求發佈 **v0.4.1**（Gitea Release id 16），Release 說明標明 Linux 沒有實機測過。附件 SHA-256：Windows `CCBBC194…EB45058B`、Linux `A4A6C176…EA2A9980`。
- 2026-11-01：0.4.2 開發：使用者要求重新安裝時兩邊都保留「登入時自動啟動」。Windows 的 `uninstall.ps1 -KeepUserSettings` 改成不刪 Run 機碼（指向 Program Files 裡的 vdswitch.exe，重新安裝後路徑相同）。兩邊的提示文字和 README 都改成「解析度、語言和登入時自動啟動會保留」。**Windows 待實機確認**：設好自動啟動 → 再執行 install.cmd 回答 y → Run 機碼 `HKCU\...\Run\go-virtual-display` 還在。
- **2026-11-01 收工時的狀態（下次從這裡接續）**：
  - 已發佈：v0.3.0（Windows＋補上的 Linux）、v0.4.0、v0.4.1。Release 說明都標明 Linux 沒有實機測過。
  - main 上是 **0.4.2（開發中，未發佈）**，只改了 Windows 重新安裝時保留自動啟動，已推上 Gitea。測試用安裝包 `dist\release\go-virtual-display-0.4.2-windows.zip` 已產生。
  - 使用者這台電腦：收工時沒有安裝。11/1 下午使用者又裝了一版（0.4.1 或 0.4.2 測試包）在用，虛擬螢幕插入中。Docker Desktop 是這次為了編譯 Linux 版啟動的，可能還開著。
  - Gitea token：使用者在 11/1 提供了新的 token，已用 `git credential approve` 存進 Credential Manager（claude-bot）。曾建議使用者撤銷並換新，因為 token 貼在對話裡；換新的話要再存一次。
- 2026-11-01：0.4.2 再修三個 Windows 問題（使用者回報，已實機測試通過）：
  - **小工具跟著被拔掉的螢幕消失**：虛擬螢幕設為主螢幕、小工具在它上面時拔掉，小工具的座標落在已不存在的螢幕上。`platform_windows.go` 的 `widgetPlacer` 每秒檢查：視窗至少一半在某螢幕工作區內才算看得到，否則暫時移到主螢幕右下角，原本的螢幕回來時移回原位（使用者選的行為）。暫時移開期間位置變了就當成使用者拖過，不再移回。隱藏或結束時存的是原本的位置。
  - **在「設定 > 顯示器」調整排列後 Synergy 滑鼠卡在邊緣**（本機滑鼠正常，使用者分別測過）：改由開關程式偵測螢幕配置改變後執行新的 `RestartServices` 排程工作，見上面的「Windows 拔插」。
  - **系統列「顯示桌面小工具」有時沒勾**（小工具明明顯示著）：`app.go` 先啟動 systray 再建立小工具，選單先建好時 `a.widget.Visible()` 還是 false。改成先建立小工具。`widget.json` 本來就會在重新安裝時保留，不是設定遺失。
- 2026-11-01：0.4.2 再加兩項（使用者要求）：
  - `RestartServices` 一律建立、執行時才檢查服務，見「Windows 拔插」。
  - Synergy 狀態圖示：有安裝就在小工具標題旁和系統列選單顯示，服務 Running 是彩色、否則灰色，沒安裝不顯示（使用者選的行為）。見「Synergy 狀態圖示」。
  - README 改成三語：`README.md`（繁中，主要版本）、`README.en.md`、`README.ja.md`，最上方互相連結，翻譯版註明以中文版為準。標示支援 Synergy 3，並加上商標聲明。**改 README 時三個版本要一起更新。**
- 2026-11-01：0.4.2 再加：**Windows 安裝包內附驅動，可離線安裝**（使用者擔心沒網路或下載點消失）。
  - 只附需要的檔案（`scripts\vdd\`，配置和 zip 相同）：x86（其實是 amd64）和 ARM64 的 `MttVDD.inf`／`.dll`／`mttvdd.cat`，加上 `devcon.exe`（Microsoft 簽章，使用者同意一起附）。壓縮後安裝包只多約 0.1 MB。
  - 每個檔案的雜湊寫在 `common.ps1` 的 `$VddFilesSha256`，`Test-VddFiles` 檢查。`download-vdd.ps1` 先用內附的，不對才下載，最後輸出驅動根目錄（`$VddRoot`）。
  - 授權：VDD 是 MIT；devcon 的原始碼（Windows-driver-samples）是 **MS-PL**，不是 MIT。全文在 `windows/scripts/THIRD-PARTY-NOTICES.txt`，打包到 `scripts\vdd\`。
  - **修正重新安裝的順序**：之前是回答 y → 移除舊版 → 下載驅動，沒網路時會兩頭落空。改成先取得驅動，成功才移除舊版。
  - 在 scratchpad 模擬過：下載網址改壞時仍用內附的成功；內附檔案改壞時警告並改成下載，下載失敗就中止。`.cat` 簽章驗證仍是 Valid。
- 2026-11-01：發佈 **v0.4.2**（Gitea Release id 17），Release 說明標明 Linux 沒有實機測過、這版 Linux 只重新打包。附件 SHA-256：Windows `9FC384DF…5C35D933`、Linux `F0935191…15CE91EC`（2026-11-02 加上 LICENSE 後重新打包替換，程式碼和實測的版本相同；原本是 `339F3B0C…`／`2D6DCDC9…`）。
  - 上傳後以 token 下載回來比對雜湊一致。**不帶 token 下載會拿到別的內容**（雜湊不同），比對時要帶 `Authorization`。
  - `RELEASING.md` 補上 0.3.0～0.4.2 的版本紀錄；打包準則第 4 條改成：預設下載，必要時（使用者決定）可以內附，但內容不改、雜湊寫死在 repo、附授權全文。
  - Linux 安裝包在 `ubuntu:22.04` 檢查：腳本語法正確；`ldd` 缺 10 個桌面函式庫（EGL、X11、Wayland、xkbcommon 等，空容器本來就沒有），裝上之後是 0。
- 2026-11-01～02：為了之後可能同步到 GitHub 讓其他人下載、贊助，做了以下準備：
  - **授權**：新增 `LICENSE`（MIT，著作權人 The78ester）。兩個安裝包最上層都附上（`release.ps1`、`release-linux.ps1`，後者用 `[ -f LICENSE ]` 判斷，打包 0.4.2 以前的 tag 不會出錯）。三份 README 都加了「授權」段落和一句贊助說明。
  - **贊助**：新增 `.github/FUNDING.yml`，用 PayPal（`custom: ["https://paypal.me/The78ester"]`）。
  - 給過的建議（使用者還沒決定）：
    - CLAUDE.md 有本機與 token 的紀錄，公開前要移除或改寫，必要時清理 git 歷史。
    - module 路徑和網址都寫死 Gitea，要先決定主要放在哪裡。
    - 程式碼簽章可考慮 SignPath.io 或 Azure Trusted Signing，否則會有 SmartScreen 和防毒軟體警告。
    - 補 SECURITY.md、issue 範本、免責聲明。
  - **v0.4.2 重新打包**（使用者要求這版就附 LICENSE）：程式碼和實測的版本相同，只差文件和打包腳本。替換了 Release 附件（新雜湊見上一則），並把 `v0.4.2` tag 從 `1ed2f75` 移到 `e6bd74b`，讓原始碼壓縮檔也有 LICENSE。
    - 強制推送 tag 會被自動模式擋下，要使用者明確要求後才能執行。
  - **Release 說明改成三語**：v0.1.0～v0.4.2 六個 Release 全部改完，規則寫在 `RELEASING.md` 第 6 步。
    - 以中文為主，英文、日文放在 `<details>` 折疊區塊；中文內容維持原文不改。
    - 程式介面沒有日文，所以日文版引用介面文字時寫英文原文。
    - 原稿在 scratchpad，用 API 的 PATCH 更新，再讀回來比對內容一致。
- 2026-11-02～03：**同步到 GitHub**（`The78estHerry/go-virtual-display`），發佈 0.4.3。
  - 敏感資料檢查：repo 和 git 歷史裡沒有 token、密碼、憑證，也沒有刪除過檔案。只有名稱、網域、信箱、電腦規格。
  - 使用者的決定：
    - 不隱藏 Gitea，歷史原封不動推上去（舊 commit 裡看得到 Gitea 網址、信箱、舊版 CLAUDE.md）。
    - CLAUDE.md 照常公開。
    - Gitea 繼續同步，兩邊都發 Release。
    - 和 Claude 的對話紀錄要不要公開，之後再決定。
  - **對話紀錄（`~/.claude/projects/...`）不能直接公開**：裡面有使用者貼過的 Gitea token 和 GitHub token、gmail、`git credential fill` 的輸出、Synergy 設定與紀錄、截圖、其他專案（claude-secretary、KYOSAITAI）的內容。要公開的話，先轉成遮罩過的 Markdown，給使用者看過。
  - 0.4.3 的改動：module 路徑、`install.ps1` 的 `URLInfoAbout`、兩個 release 腳本寫的 README.txt 原始碼網址改成 GitHub；`FUNDING.yml` 啟用 PayPal（`custom`）；三份 README 的贊助說明加上 PayPal 連結。功能沒有變更。
  - polkit ID `io.littleherry.vdswitch` 不改：改名的話，已安裝的使用者升級時要多處理舊檔。
  - 發佈 **v0.4.3**（2026-11-03）：使用者用安裝包實測通過（支援連結開啟 GitHub、拔插正常）。
    - 推上 GitHub 的有 main 和 v0.1.0～v0.4.3 全部 tag；GitHub 只有 v0.4.3 有 Release，舊版的 Release 只在 Gitea。
    - GitHub Release id 432617847，Gitea Release id 18，標題沿用「v0.4.3」這種只有版本號的格式。
    - 附件 SHA-256：Windows `858849E7…63D30616`、Linux `BECEA322…26EFB9D9B`。兩邊下載回來比對都一致。
    - 第一次推 GitHub 時 403：fine-grained token 的 Contents 權限要設成 Read and write。改權限不會換掉 token。
  - GitHub token 是使用者貼在對話裡的，已建議發佈完成後撤銷並換新，新的請使用者在自己的終端機用 `git credential approve` 輸入。
- 2026-11-04：依使用者要求，三份 README 最上面（語言連結下面）加上 PayPal 徽章和一句贊助說明。
- 2026-11-04：依使用者要求，commit 信箱從舊的網域信箱（敏感字眼，不寫出來，見 gitea-to-github 的 rules.md 第 5 節）改成 `littleherry@gmail.com`，**連歷史一起改寫**：
  - 用 `git filter-branch --env-filter` 改寫全部 commit（電腦上沒有 filter-repo），內容和日期不變，只有雜湊變了。annotated tag 用原本的訊息和日期重新建立，tagger 也換成新信箱。
  - 強制推送 main 和 v0.1.0～v0.4.3 到 GitHub 和 Gitea。Release 跟著 tag 名稱，安裝包附件不變；原始碼壓縮檔的內容相同。
  - 上面紀錄裡的 commit 雜湊已換成新的。改寫前的備份是 scratchpad 的 `before-email-rewrite.bundle`（工作階段結束就會清掉）。
  - 全域的 `~/.claude/settings.json`（`GIT_AUTHOR_EMAIL`／`GIT_COMMITTER_EMAIL`）和 `~/.claude/CLAUDE.md` 也改成新信箱，**所有專案**之後的 commit 都會用它。
  - 同一天再改寫一次：之前的 CLAUDE.md 紀錄寫出了舊信箱，連同其他 10 個 repo 的歷史一起清掉（經過在 gitea-to-github 的 `cases/history-email-cleanup.md`）。本專案只有最後兩個 commit 的雜湊改變，tag 和 Release 都沒動。
- 2026-11-04：依使用者要求，贊助只用 PayPal（`https://paypal.me/The78ester`，不帶 locale 參數）：三份 README 的徽章、最上方和授權段落的贊助句，`FUNDING.yml` 的 `custom`，RELEASING.md 的 0.4.3 紀錄，v0.4.3 的 Release 說明（GitHub、Gitea，其餘內容和附件不變）。go-screen-burn-in-fixer 同步改；gitea-to-github 的 `readme-style.md` 是之後所有 GitHub 專案的樣式。**不要用 GitHub Sponsors：不支援台灣。**
- 2026-11-04：使用者設定好 Ko-fi（`https://ko-fi.com/the78ester`）。三份 README 加上 Ko-fi 徽章，贊助句改成「PayPal 或 Ko-fi」，`FUNDING.yml` 加 `ko_fi: the78ester`。go-screen-burn-in-fixer 和 readme-style.md 同步改。v0.4.3 的 Release 說明（GitHub、Gitea）三語也加上 Ko-fi。
- 待辦：
  1. （已完成）v0.4.2 已發佈。
  2. Linux 實機測試：**使用者決定暫時不測**（2026-11-01），Release 說明一律標註「Linux 沒有實機測過」即可，不要主動提議測試。之後要測時需要使用者的 Linux 機器（WSL 沒有 DRM），測試前可先跑 `linux\tests\run.ps1`。
  3. 使用者曾問 icon 三個方案的比較網頁，舊的已不在。若需要可依記錄重畫 A1／A2／A3 並發佈成 Artifact 保存（還沒答應要做）。

## 給之後的 Claude

- 發新版本照 `RELEASING.md` 做：先改 `common.ps1` 的 `$AppVersion`，再用 `release.ps1` 產生精簡的 Windows 安裝包，各平台要包哪些檔案也記在那裡。改了安裝時用到的檔案，要同時更新 `release.ps1` 和 `RELEASING.md` 的清單。

- 改了 EDID 參數要重新產生 `.bin` 並一起 commit，`windows/scripts/verify.ps1` 和 `linux/scripts/verify.sh` 都是拿它來比對。
- 升級 VDD 版本時，要同時更新 `windows/scripts/common.ps1` 的版本、zip 的 SHA-256 和 `$VddFilesSha256`（內附的 7 個檔案），`THIRD-PARTY-NOTICES.txt` 的版本和雜湊，並重新讀新版的 `vdd_settings.xml` 格式和 INF 的硬體 ID。
- commit 一律以 claude-bot 身分。0.4.3 起要推兩邊：
  - `github`：`https://github.com/The78estHerry/go-virtual-display`，主要位置，公開。token 是使用者的 fine-grained token，存在 Credential Manager（`host=github.com`、`username=The78estHerry`）。
  - `origin`：`gitea.littleherry.work/The78estWorld/go-virtual-display`，要登入才能看，同步一份。
  - 發新版時兩邊都要建 Release，步驟見 `RELEASING.md`。
- module 路徑是 `github.com/The78estHerry/go-virtual-display`（0.4.3 起；之前是 Gitea 的路徑）。程式碼和安裝包裡給使用者看的連結都要指向 GitHub，因為其他人打不開 Gitea。
