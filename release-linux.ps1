# 產生 Linux 安裝包 dist\release\go-virtual-display-<版本>-linux-amd64.tar.gz。
# Linux 版的開關程式（Gio）需要 cgo，Windows 上無法直接編譯，所以用 Docker 的 Linux 容器編譯並打包。
# 用 Debian 11（glibc 2.31）編譯，Ubuntu 20.04／Debian 11 以後的發行版都能執行。
#
# 用法：powershell -ExecutionPolicy Bypass -File .\release-linux.ps1 [-Ref v0.3.0]
#   -Ref：要打包的 git 版本（tag 或 commit），預設 HEAD。版本號取自該版本的 windows/scripts/common.ps1。
# 需要 Docker Desktop（Linux 容器）。
param(
    [string]$Ref = 'HEAD',
    [string]$GoVersion = '1.27.0'
)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$common = git show "${Ref}:windows/scripts/common.ps1"
if ($LASTEXITCODE -ne 0) { throw "找不到 $Ref" }
$Version = ([regex]::Match(($common -join "`n"), "\`$AppVersion\s*=\s*'([^']+)'")).Groups[1].Value
if (-not $Version) { throw "$Ref 的 common.ps1 裡找不到 `$AppVersion" }
$name = "go-virtual-display-$Version-linux-amd64"
Write-Host "打包 $Ref（版本 $Version）→ dist\release\$name.tar.gz"

# 把指定版本的原始碼匯出到暫存資料夾，在容器裡編譯
$work = Join-Path $PSScriptRoot 'dist\release-linux'
Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $work 'src') | Out-Null
git archive --format=tar -o (Join-Path $work 'src.tar') $Ref
if ($LASTEXITCODE -ne 0) { throw 'git archive 失敗' }
New-Item -ItemType Directory -Force (Join-Path $PSScriptRoot 'dist\release') | Out-Null

# 0.4.0 起 install.sh 能從安裝包的扁平配置（install.sh、uninstall.sh 在最上層，其餘在 scripts/）找檔案；
# 0.3.0 只認得原始碼的配置（linux/、edid/、dist/linux/）
$flat = ((git show "${Ref}:linux/install.sh") -join "`n") -match 'SCRIPTS/edid_1920x1080\.bin'
$layout = if ($flat) { 'flat' } else { 'repo' }
Write-Host "安裝包配置：$layout"

# 安裝包裡的 README.txt：英文在前、繁中在後
if ($flat) {
$readme = @"
go-virtual-display $Version (Linux, amd64)

[English]

Makes Linux believe a monitor with a custom EDID is connected to an unused
connector on your graphics card (HDMI, DisplayPort, ...). Includes a one-click
plug/unplug switch (tray icon and desktop widget).

Install:    Run ./install.sh as a normal user (it asks for your sudo
            password). It picks a free external connector automatically, or
            use: ./install.sh card0-HDMI-A-1
            Reboot afterwards if it says so.
Resolution: The default is 1920x1080. Right-click the tray icon and choose
            "Resolution" to switch, or "Manage resolutions..." to add and
            delete resolutions. It applies immediately when debugfs is
            available; otherwise after a reboot.
Check:      scripts/verify.sh
Uninstall:  ./uninstall.sh, then reboot. This also deletes your saved
            resolutions.

Requires an unused connector on a physical GPU (not VMs), GRUB for the kernel
parameters, and glibc 2.31 or later (Ubuntu 20.04 / Debian 11 or newer).
The scripts folder contains installer files; you do not need to open it.

[繁體中文]

讓 Linux 以為顯示卡上一個沒用到的接頭（HDMI、DisplayPort 等）接著一台螢幕
（使用自訂 EDID），並附系統列圖示與桌面小工具的一鍵拔插開關。

安裝：以一般使用者執行 ./install.sh（會詢問 sudo 密碼）。會自動挑選沒接螢幕的
      外接接頭，也可以指定：./install.sh card0-HDMI-A-1
      若提示需要重新開機，請重新開機。
解析度：預設 1920x1080。在系統列圖示按右鍵選「解析度」切換，或選「管理解析度…」
        新增和刪除。有 debugfs 時立即生效，否則重新開機後生效。
檢查：scripts/verify.sh
移除：執行 ./uninstall.sh，再重新開機。會一併刪掉你存的解析度。

需求：實體顯示卡上有空的接頭（VM 不適用）、使用 GRUB 開機、
      glibc 2.31 以上（Ubuntu 20.04／Debian 11 以後）。
scripts 資料夾是安裝用的檔案，不需要直接開啟。

Source / 原始碼：https://github.com/The78estHerry/go-virtual-display
"@
} else {
$readme = @"
go-virtual-display $Version (Linux, amd64)

[English]

Makes Linux believe a 1920x1440 monitor with a custom EDID is connected to an
unused connector on your graphics card (HDMI, DisplayPort, ...). Includes a
one-click plug/unplug switch (tray icon and desktop widget).

Install:   Run linux/install.sh as a normal user (it asks for your sudo
           password). It picks a free external connector automatically, or
           use: linux/install.sh card0-HDMI-A-1
           Reboot afterwards if it says so.
Check:     linux/scripts/verify.sh
Uninstall: linux/uninstall.sh, then reboot.

Requires an unused connector on a physical GPU (not VMs), GRUB for the kernel
parameters, and glibc 2.31 or later (Ubuntu 20.04 / Debian 11 or newer).
On Linux the resolution is fixed at 1920x1440 in this version.
This Linux package has not been tested on real hardware yet.

[繁體中文]

讓 Linux 以為顯示卡上一個沒用到的接頭（HDMI、DisplayPort 等）接著一台
1920x1440 的螢幕（使用自訂 EDID），並附系統列圖示與桌面小工具的一鍵拔插開關。

安裝：以一般使用者執行 linux/install.sh（會詢問 sudo 密碼）。會自動挑選沒接螢幕的
      外接接頭，也可以指定：linux/install.sh card0-HDMI-A-1
      若提示需要重新開機，請重新開機。
檢查：linux/scripts/verify.sh
移除：執行 linux/uninstall.sh，再重新開機。

需求：實體顯示卡上有空的接頭（VM 不適用）、使用 GRUB 開機、
      glibc 2.31 以上（Ubuntu 20.04／Debian 11 以後）。
這一版的 Linux 解析度固定 1920x1440。
Linux 安裝包還沒有在實機上測試過。

Source / 原始碼：https://github.com/The78estHerry/go-virtual-display
"@
}
# 用 LF 換行、UTF-8 without BOM，Linux 的終端機和編輯器才會正常顯示
[IO.File]::WriteAllText((Join-Path $work 'README.txt'), ($readme -replace "`r`n", "`n"), (New-Object Text.UTF8Encoding $false))

# 容器裡執行的步驟。LAYOUT=flat 是 0.4.0 起的扁平配置；repo 沿用原始碼的目錄配置（0.3.0）
$build = @'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
# Debian 11 的安全更新已經搬到 archive.debian.org，原本的位址會 404
sed -i -E 's#https?://(deb|security)\.debian\.org/debian-security#http://archive.debian.org/debian-security#' /etc/apt/sources.list /etc/apt/sources.list.d/* 2>/dev/null || true
apt-get update -qq
apt-get install -y -qq --no-install-recommends ca-certificates curl gcc libc6-dev pkg-config \
    libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev \
    libffi-dev libxcursor-dev libvulkan-dev python3 >/dev/null
tgz=go${GO_VERSION}.linux-amd64.tar.gz
want=$(curl -fsSL 'https://go.dev/dl/?mode=json&include=all' | python3 -c "import json,sys; v='go'+sys.argv[1]; print(next(f['sha256'] for r in json.load(sys.stdin) if r['version']==v for f in r['files'] if f['filename']==sys.argv[2]))" "$GO_VERSION" "$tgz")
curl -fsSL -o /tmp/go.tgz "https://go.dev/dl/$tgz"
echo "$want  /tmp/go.tgz" | sha256sum -c - >/dev/null
tar -C /usr/local -xzf /tmp/go.tgz
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOFLAGS=-mod=mod

mkdir -p /src && tar -C /src -xf /work/src.tar && cd /src
go vet ./internal/display ./internal/edid ./internal/profile ./cmd/make-edid
go test ./internal/edid ./internal/profile
CGO_ENABLED=1 go build -trimpath -ldflags '-s -w' -o /tmp/vdswitch ./cmd/vdswitch

stage=/tmp/$NAME
if [ "$LAYOUT" = flat ]; then
    mkdir -p "$stage/scripts"
    cp linux/install.sh linux/uninstall.sh "$stage/"
    cp linux/scripts/* "$stage/scripts/"
    cp edid/edid_1920x1080.bin "$stage/scripts/"
    printf '%s\n' "$VERSION" > "$stage/scripts/VERSION" # install.sh 顯示與記錄版本用（0.4.1 起）
    chmod 644 "$stage/scripts/VERSION"
    install -m 755 /tmp/vdswitch "$stage/scripts/vdswitch"
    cp /work/README.txt "$stage/"
    if [ -f LICENSE ]; then cp LICENSE "$stage/"; fi   # 0.4.2 之後才有 LICENSE
    chmod 755 "$stage"/*.sh "$stage"/scripts/verify.sh "$stage"/scripts/vdswitch-helper "$stage"/scripts/initramfs-hook
    chmod 644 "$stage"/scripts/lang.sh "$stage"/scripts/*.policy "$stage"/scripts/*.desktop "$stage"/scripts/*.bin "$stage"/README.txt
else
    mkdir -p "$stage/dist/linux" "$stage/edid"
    cp -r linux "$stage/"
    cp edid/edid_1920x1440.bin "$stage/edid/"
    install -m 755 /tmp/vdswitch "$stage/dist/linux/vdswitch"
    cp /work/README.txt "$stage/"
    chmod 755 "$stage"/linux/*.sh "$stage"/linux/scripts/verify.sh "$stage"/linux/scripts/vdswitch-helper "$stage"/linux/scripts/initramfs-hook
    chmod 644 "$stage"/linux/scripts/lang.sh "$stage"/linux/scripts/*.policy "$stage"/linux/scripts/*.desktop "$stage"/edid/*.bin "$stage"/README.txt
fi
tar -C /tmp --owner=0 --group=0 --numeric-owner -czf "/out/$NAME.tar.gz" "$NAME"
tar -tvzf "/out/$NAME.tar.gz"
'@
[IO.File]::WriteAllText((Join-Path $work 'build.sh'), ($build -replace "`r`n", "`n"), (New-Object Text.UTF8Encoding $false))

docker run --rm --platform linux/amd64 `
    -e "GO_VERSION=$GoVersion" -e "NAME=$name" -e "LAYOUT=$layout" -e "VERSION=$Version" `
    -v "${work}:/work:ro" -v "$(Join-Path $PSScriptRoot 'dist\release'):/out" `
    debian:bullseye bash /work/build.sh
if ($LASTEXITCODE -ne 0) { throw "容器內編譯失敗（代碼 $LASTEXITCODE）" }
Remove-Item $work -Recurse -Force
Write-Host "完成：dist\release\$name.tar.gz"
