# 發佈版本

給要打包、發佈新版本的人。所有專案共用的打包準則在最後一節。

## 版本號

- 用語意化版本 `主.次.修`，例如 `0.1.0`。tag 名稱加 `v`，例如 `v0.1.0`。
- 修 bug 加修訂號，加功能加次版號。1.0.0 之前可以有不相容的變更。小幅度的功能調整可由使用者決定只加修訂號。
- 版本號只寫在 `windows/scripts/common.ps1` 的 `$AppVersion`：`release.ps1` 以它命名安裝包，「設定 > 應用程式」也顯示它。發新版前先改這裡。

## 步驟

1. 把 `windows/scripts/common.ps1` 的 `$AppVersion` 改成新版本，確認要發佈的修改都已 commit 並推上 Gitea，工作目錄乾淨（`git status`）。
2. 產生安裝包：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\release.ps1
   ```
   它會先跑 `build.ps1`（測試、編譯、重產 EDID），再組出 `dist\release\go-virtual-display-<版本>-windows.zip`，最後列出包進去的檔案。
3. 核對包進去的檔案和下面的清單一致，沒有多也沒有少。
   改了 `assets/icon.svg` 的話，先執行 `assets/make-icon.ps1` 更新 `icon.ico` 和 `.syso`，並一起 commit。
4. 解壓縮到一個新資料夾，實際點 `install.cmd` 安裝、拔插，再移除一輪。三種移除方式（系統列、設定 > 應用程式、`uninstall.cmd`）有改到的都要試，移除後檢查系統是否乾淨。
5. 打 annotated tag 並推上去：
   ```
   git tag -a v0.1.0 -m "v0.1.0：<一句話摘要>"
   git push origin v0.1.0
   ```
6. 在 Gitea 建 Release（網頁或 API），說明寫：
   - 下載哪個檔案、怎麼安裝
   - 這版的內容
   - 測試狀態：哪些平台實機測過，哪些沒有
   - 已知問題
7. 上傳安裝包當作 Release 附件。**不要上傳原始碼壓縮檔**，Gitea 會自動附上。

用 API 時，Release 說明的中文要先寫成 UTF-8 檔案，再用 `curl --data-binary @檔案` 送出。直接放在命令列參數裡會變成亂碼。

## 各平台安裝包的內容

### Windows：`go-virtual-display-<版本>-windows.zip`

```
go-virtual-display-<版本>-windows/
├── install.cmd               一鍵安裝（windows/install.cmd）
├── uninstall.cmd             一鍵移除（windows/uninstall.cmd）
├── README.txt                使用說明，英文在前、繁中在後（release.ps1 產生，UTF-8 with BOM＋CRLF）
└── scripts/
    ├── common.ps1            ┐
    ├── download-vdd.ps1      │
    ├── install.ps1           ├ windows/scripts/
    ├── launch-uninstall.ps1  │
    ├── uninstall.ps1         │
    ├── verify.ps1            │
    ├── vdd_settings.xml      ┘
    ├── edid_1920x1080.bin    edid/edid_1920x1080.bin（沒有 vdswitch.exe 時才用的預設 EDID）
    └── vdswitch.exe          dist/windows/vdswitch.exe（build.ps1 編譯）
```

- 腳本靠 `scripts\edid_1920x1080.bin` 是否存在判斷自己在安裝包裡（`common.ps1` 的 `$Packaged`）。在安裝包裡時，EDID 和開關程式從 `scripts\` 讀取，驅動下載到 `scripts\downloads\`。
- **不放**：
  - Go 原始碼、`go.mod`、`build.ps1`、`release.ps1`、`Makefile`
  - `make-edid.exe`：使用者用不到
  - `linux/`
  - `README.md`、`CLAUDE.md`、`RELEASING.md`：改放精簡的 `README.txt`
  - Virtual Display Driver 本體：安裝時才下載並驗證 SHA-256，不重新散布
- 新增會在安裝時用到的檔案時，要同時改 `release.ps1` 和這份清單。

### Linux：`go-virtual-display-<版本>-linux-amd64.tar.gz`

Linux 版的開關程式（Gio）需要 cgo，Windows 上無法直接編譯，所以用 Docker 的 Linux 容器編譯並打包：

```powershell
powershell -ExecutionPolicy Bypass -File .\release-linux.ps1 -Ref v0.3.0   # 預設 HEAD
```

- 需要 Docker Desktop（Linux 容器）。用 `debian:bullseye`（glibc 2.31）編譯，Ubuntu 20.04／Debian 11 以後都能執行。Go 從 go.dev 下載並比對 SHA-256。
- Debian 11 的安全更新已經搬到 `archive.debian.org`，腳本會改寫 apt 來源。
- 用 tar 才能保留執行權限。檔案擁有者固定為 root（0/0）。
- 0.4.0 起是扁平配置（和 Windows 一樣），`install.sh` 依 `scripts/edid_1920x1080.bin` 是否存在判斷自己在安裝包裡：

```
go-virtual-display-<版本>-linux-amd64/
├── install.sh                一鍵安裝（linux/install.sh）
├── uninstall.sh              一鍵移除（linux/uninstall.sh）
├── README.txt                使用說明，英文在前、繁中在後（release-linux.ps1 產生，UTF-8＋LF）
└── scripts/                  linux/scripts/ 的所有檔案，加上：
    ├── edid_1920x1080.bin    edid/edid_1920x1080.bin（預設 EDID）
    └── vdswitch              開關程式（容器內編譯）
```

- 0.3.0 的腳本只認得原始碼的目錄配置，`release-linux.ps1 -Ref v0.3.0` 會自動改用這個配置：

```
go-virtual-display-<版本>-linux-amd64/
├── README.txt                使用說明，英文在前、繁中在後（release-linux.ps1 產生，UTF-8＋LF）
├── linux/
│   ├── install.sh            一鍵安裝
│   ├── uninstall.sh          一鍵移除
│   └── scripts/              verify.sh、vdswitch-helper、lang.sh、initramfs-hook、polkit 規則、.desktop
├── edid/edid_1920x1440.bin
└── dist/linux/vdswitch       開關程式（容器內編譯）
```

- 打包後，用 `ubuntu:22.04` 容器檢查：腳本 `bash -n`、`readelf -d` 看依賴的函式庫、`ldd` 沒有 not found。

## 版本紀錄

| 版本 | 日期 | 安裝包 | 摘要 |
|---|---|---|---|
| 0.1.0 | 2026-10-31 | windows | 第一個版本。Windows 已從安裝包實機測試（安裝、拔插、移除乾淨），Linux 未測 |
| 0.2.0 | 2026-10-31 | windows | 英文／繁中雙語介面，安裝時選擇，預設依系統顯示語言；檔名一律英文；可從系統列「解除安裝…」或「設定 > 應用程式」移除，安裝資料夾刪掉也能移除（原訂 0.1.1，未發佈，併入本版）。已從安裝包實測：英文／繁中介面、系統列與「設定 > 應用程式」移除都乾淨 |

## 打包準則（所有專案共用）

1. **一個平台一個安裝包**，檔名 `<專案>-<版本>-<平台>[-<架構>].<副檔名>`。
2. **只放執行需要的檔案**：編譯好的程式、安裝／移除腳本、執行時要讀的設定與資料、一份精簡的使用說明。原始碼、建置腳本、開發文件、其他平台的檔案、測試檔一律不放。
3. **使用者一眼看得出要點哪個**：最上層只放入口（安裝、移除）和說明，其餘收進子資料夾。
4. **第三方元件**能在安裝時下載就不重新散布，並固定版本、驗證雜湊值。
5. **用腳本產生安裝包**（例如 `release.ps1`），不手動組。腳本最後列出包進去的檔案。
6. **在專案裡記錄**每個平台的安裝包內容（像本檔），新增或移除檔案時一起更新。
7. **發佈前從安裝包實際裝一次**，不要只在原始碼目錄測試。
8. **檔名一律用英文**：安裝包裡的檔案、資料夾、說明檔、捷徑都是。使用者介面要提供英文版，依系統顯示語言決定預設（只有中文語系預設繁中），安裝時讓使用者選。
