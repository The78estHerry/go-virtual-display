# 發佈版本

給要打包、發佈新版本的人。所有專案共用的打包準則在最後一節。

## 版本號

- 用語意化版本 `主.次.修`，例如 `0.1.0`。tag 名稱加 `v`，例如 `v0.1.0`。
- 修 bug 加修訂號，加功能加次版號。1.0.0 之前可以有不相容的變更。

## 步驟

1. 確認要發佈的修改都已 commit 並推上 Gitea，工作目錄乾淨（`git status`）。
2. 產生安裝包：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\release.ps1 -Version 0.1.0
   ```
   它會先跑 `build.ps1`（測試、編譯、重產 EDID），再組出 `dist\release\go-virtual-display-<版本>-windows.zip`，最後列出包進去的檔案。
3. 核對包進去的檔案和下面的清單一致，沒有多也沒有少。
4. 解壓縮到一個新資料夾，實際點 `install.cmd` 安裝、拔插、`uninstall.cmd` 移除一輪。
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
├── 說明.txt                  使用說明（release.ps1 產生，UTF-8 with BOM＋CRLF）
└── scripts/
    ├── common.ps1            ┐
    ├── download-vdd.ps1      │
    ├── install.ps1           ├ windows/scripts/
    ├── uninstall.ps1         │
    ├── verify.ps1            │
    ├── vdd_settings.xml      ┘
    ├── edid_1920x1440.bin    edid/edid_1920x1440.bin
    └── vdswitch.exe          dist/windows/vdswitch.exe（build.ps1 編譯）
```

- 腳本靠 `scripts\edid_1920x1440.bin` 是否存在判斷自己在安裝包裡（`common.ps1` 的 `$Packaged`）。在安裝包裡時，EDID 和開關程式從 `scripts\` 讀取，驅動下載到 `scripts\downloads\`。
- **不放**：
  - Go 原始碼、`go.mod`、`build.ps1`、`release.ps1`、`Makefile`
  - `make-edid.exe`：使用者用不到
  - `linux/`
  - `README.md`、`CLAUDE.md`、`RELEASING.md`：改放精簡的 `說明.txt`
  - Virtual Display Driver 本體：安裝時才下載並驗證 SHA-256，不重新散布
- 新增會在安裝時用到的檔案時，要同時改 `release.ps1` 和這份清單。

### Linux：尚未提供

Linux 版的開關程式需要 cgo，要在 Linux 上編譯，目前也還沒實機測試，所以還沒有 Linux 安裝包。之後要做時，比照 Windows 的原則：
- 只放 `install.sh`、`uninstall.sh`、`scripts/` 裡的檔案、EDID、編好的 `vdswitch`
- 腳本改成能從安裝包的配置找到 EDID 和執行檔
- 檔名 `go-virtual-display-<版本>-linux-amd64.tar.gz`，用 tar 才能保留執行權限

## 版本紀錄

| 版本 | 日期 | 安裝包 | 摘要 |
|---|---|---|---|
| 0.1.0 | 2026-10-31 | windows | 第一個版本。Windows 已實機測試，Linux 未測 |

## 打包準則（所有專案共用）

1. **一個平台一個安裝包**，檔名 `<專案>-<版本>-<平台>[-<架構>].<副檔名>`。
2. **只放執行需要的檔案**：編譯好的程式、安裝／移除腳本、執行時要讀的設定與資料、一份精簡的使用說明。原始碼、建置腳本、開發文件、其他平台的檔案、測試檔一律不放。
3. **使用者一眼看得出要點哪個**：最上層只放入口（安裝、移除）和說明，其餘收進子資料夾。
4. **第三方元件**能在安裝時下載就不重新散布，並固定版本、驗證雜湊值。
5. **用腳本產生安裝包**（例如 `release.ps1`），不手動組。腳本最後列出包進去的檔案。
6. **在專案裡記錄**每個平台的安裝包內容（像本檔），新增或移除檔案時一起更新。
7. **發佈前從安裝包實際裝一次**，不要只在原始碼目錄測試。
