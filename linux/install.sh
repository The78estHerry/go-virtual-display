#!/usr/bin/env bash
# 一鍵安裝：把一個沒接螢幕的顯示卡接頭設成「接著一台螢幕」（使用自訂 EDID，解析度可在開關程式切換），
# 安裝免密碼的小幫手與開關程式，並自動加上開機參數。
#
# 用法：install.sh            自動挑選沒接螢幕的外接接頭（HDMI／DP 等）
#       install.sh <接頭>     指定接頭，例如 card0-HDMI-A-1
# 以一般使用者執行即可，需要 root 的部分會自動透過 sudo 執行。
# 介面語言：互動執行時讓使用者選，預設依系統語系（中文用繁體中文，其他英文）。
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
SCRIPTS=$HERE/scripts
# 兩種配置：Release 安裝包把預設 EDID 和開關程式放在 scripts/ 裡；原始碼則各在 edid/、dist/linux/
if [ -f "$SCRIPTS/edid_1920x1080.bin" ]; then
    DEFAULT_EDID=$SCRIPTS/edid_1920x1080.bin
    APP=$SCRIPTS/vdswitch
    REPO=""
else
    REPO=$(cd "$HERE/.." && pwd)
    DEFAULT_EDID=$REPO/edid/edid_1920x1080.bin
    APP=$REPO/dist/linux/vdswitch
fi
CONF=/etc/go-virtual-display.conf
LIBEXEC=/usr/local/libexec/go-virtual-display
# 開機參數固定指向這個檔名，切換解析度時只換檔案內容（0.3.0 以前是 edid_1920x1440.bin）
EDID_NAME=go-virtual-display.bin
OLD_EDID_NAME=edid_1920x1440.bin
FIRMWARE=/lib/firmware/edid/$EDID_NAME
GRUB_DEFAULT_FILE=/etc/default/grub
# 版本號：安裝包裡是 release-linux.ps1 寫的 scripts/VERSION；原始碼則讀 windows/scripts/common.ps1 的 $AppVersion（唯一的來源）
APP_VERSION=""
if [ -f "$SCRIPTS/VERSION" ]; then
    APP_VERSION=$(head -n 1 "$SCRIPTS/VERSION" | tr -d '\r')
elif [ -n "$REPO" ] && [ -f "$REPO/windows/scripts/common.ps1" ]; then
    APP_VERSION=$(sed -n "s/^\$AppVersion *= *'\([^']*\)'.*/\1/p" "$REPO/windows/scripts/common.ps1" | head -n 1)
fi
APP_VERSION=${APP_VERSION:-unknown}

# root 階段的參數由一般使用者階段帶入：--lang <語言>、--edid <依使用者解析度產生的 EDID>
EDID_IN=""
while [ $# -gt 0 ]; do
    case "$1" in
        --lang)
            GVD_LANG=$2
            shift 2
            ;;
        --edid)
            EDID_IN=$2
            shift 2
            ;;
        *) break ;;
    esac
done
# shellcheck source=scripts/lang.sh
. "$SCRIPTS/lang.sh"

list_connectors() {
    for d in /sys/class/drm/card*-*; do
        [ -f "$d/status" ] && printf '  %-24s %s\n' "$(basename "$d")" "$(cat "$d/status")"
    done
}

# 挑第一個沒接螢幕的外接接頭；排除筆電內建面板（eDP、LVDS、DSI）與虛擬接頭
pick_connector() {
    for d in /sys/class/drm/card*-*; do
        local n
        n=$(basename "$d")
        [ -f "$d/status" ] || continue
        case "$n" in *-eDP-* | *-LVDS-* | *-DSI-* | *-Writeback-* | *-Virtual-*) continue ;; esac
        if [ "$(cat "$d/status")" = disconnected ]; then
            echo "$n"
            return 0
        fi
    done
    return 1
}

# ---------- 一般使用者階段：已安裝時先確認、選語言、編譯、提權、開啟開關程式 ----------
if [ "$(id -u)" -ne 0 ]; then
    # 已經安裝過：告訴使用者目前和要安裝的版本，同意後先移除舊版（保留解析度清單和語言），再全新安裝。
    # 0.4.0 以前沒有在設定檔記錄版本
    if [ -f "$CONF" ]; then
        installed=$(sed -n 's/^VERSION=//p' "$CONF" | head -n 1)
        shown=${installed:-$(t 'unknown (0.4.0 or earlier)' '不明（0.4.0 以前）')}
        if [ -t 0 ]; then
            echo
            t "Virtual Display Switch is already installed." "虛擬螢幕開關已經安裝了。"
            t "  Installed version: $shown" "  目前版本：$shown"
            t "  This package:      $APP_VERSION" "  要安裝的版本：$APP_VERSION"
            if [ -n "$installed" ] && [ "$APP_VERSION" != unknown ]; then
                if [ "$installed" = "$APP_VERSION" ]; then
                    t "  (Same version: this will reinstall it.)" "  （版本相同，會重新安裝）"
                elif [ "$(printf '%s\n%s\n' "$installed" "$APP_VERSION" | sort -V | tail -n 1)" = "$installed" ]; then
                    t "  Warning: the installed version is newer; this will downgrade it." "  注意：目前的版本比較新，繼續會降版。"
                fi
            fi
            t "  Your resolution list and language are kept." "  你新增的解析度和選的語言會保留。"
            echo
            read -r -p "$(t 'Uninstall the current version and install this one? (y/N) : ' '要先移除目前的版本，再安裝這個版本嗎？(y/N) : ')" answer || true
            case "$answer" in
                y | Y | yes | Yes | YES) ;;
                *)
                    t "Cancelled. Nothing was changed." "已取消，沒有做任何變更。"
                    exit 0
                    ;;
            esac
            # 移除會刪掉設定檔，先記下原本的接頭；沒有指定接頭時沿用它
            old_connector=$(sed -n 's/^CONNECTOR=\(card[0-9]*-[A-Za-z0-9-]*\)$/\1/p' "$CONF" | head -n 1)
            echo
            t "==> Uninstalling the current version ..." "==> 移除目前的版本 ..."
            bash "$HERE/uninstall.sh" --keep-user-settings
            echo
            t "==> Installing version $APP_VERSION ..." "==> 安裝版本 $APP_VERSION ..."
            if [ $# -eq 0 ] && [ -n "$old_connector" ]; then
                set -- "$old_connector"
            fi
        else
            # 沒有終端機可以詢問時，直接覆蓋安裝（install.sh 可以重複執行）
            t "==> Already installed ($shown); installing $APP_VERSION over it ..." "==> 已經安裝（$shown），直接覆蓋安裝 $APP_VERSION ..."
        fi
    fi
    if [ -t 0 ]; then
        default=1
        [ "$GVD_LANG" = zh-TW ] && default=2
        echo "Select language / 選擇語言"
        echo "  1) English"
        echo "  2) 繁體中文"
        read -r -p "Enter 1 or 2 [default: $default]: " answer || true
        case "${answer:-$default}" in
            1) GVD_LANG=en ;;
            2) GVD_LANG=zh-TW ;;
        esac
        echo
    fi
    if [ -n "$REPO" ] && [ ! -x "$APP" ] && command -v go >/dev/null; then
        t "==> Building the switch ..." "==> 編譯開關程式 ..."
        if ! (cd "$REPO" && make linux); then
            t "Build failed; continuing without the switch (see the top of the Makefile for required packages)." \
                "編譯失敗，繼續安裝但略過開關程式（缺少的套件見 Makefile 開頭）。"
        fi
    fi
    # 依這個使用者存的解析度產生 EDID（沒存過就是預設的 1920x1080），重新安裝時保留原本選的解析度
    edid_args=()
    if [ -x "$APP" ]; then
        user_edid=$(mktemp)
        trap 'rm -f "$user_edid"' EXIT
        if active=$("$APP" --write-edid "$user_edid" 2>/dev/null); then
            edid_args=(--edid "$user_edid")
            t "==> Resolution: $active" "==> 解析度：$active"
        fi
    fi
    t "==> Administrator permission is required. Please enter your password:" "==> 需要系統管理員權限，請輸入密碼："
    sudo bash "$0" --lang "$GVD_LANG" "${edid_args[@]}" "$@"
    if [ -x /usr/local/bin/vdswitch ] && [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
        pkill -x vdswitch 2>/dev/null || true
        setsid /usr/local/bin/vdswitch >/dev/null 2>&1 < /dev/null &
        t "==> Opened Virtual Display Switch (tray and desktop widget)" "==> 已開啟虛擬螢幕開關（系統列與桌面小工具）"
    fi
    exit 0
fi

# ---------- root 階段 ----------
# 只移除本專案加上的參數：指向我們 EDID（含 0.3.0 以前的檔名）的 drm.edid_firmware，以及指定接頭的 video=<接頭>:e
strip_params() {
    [ -n "$1" ] || return 0
    sed -i -E "/^GRUB_CMDLINE_LINUX_DEFAULT=/ { s/ ?drm\.edid_firmware=[^\" ]*(go-virtual-display|edid_1920x1440)\.bin//g; s/ ?video=$1:e//g; s/=\" /=\"/ }" "$GRUB_DEFAULT_FILE"
}
OLD_CONNECTOR=$(sed -n 's/^CONNECTOR=\(card[0-9]*-[A-Za-z0-9-]*\)$/\1/p' "$CONF" 2>/dev/null | head -n 1 || true)
OLD_KCONN=${OLD_CONNECTOR#*-}
CONNECTOR=${1:-$OLD_CONNECTOR} # 重新安裝時沿用原本的接頭
if [ -z "$CONNECTOR" ]; then
    if ! CONNECTOR=$(pick_connector); then
        t "No free external connector found. Current connectors:" "找不到沒接螢幕的外接接頭。目前的接頭：" >&2
        list_connectors >&2
        t "Use install.sh <connector> to choose one. Machines without a physical GPU (VMs, servers) are not supported." \
            "可以用 install.sh <接頭> 指定。沒有實體顯示卡的機器（VM、伺服器）不適用。" >&2
        exit 1
    fi
    t "==> Selected connector $CONNECTOR" "==> 自動選擇接頭 $CONNECTOR"
fi
if ! [[ "$CONNECTOR" =~ ^card[0-9]+-[A-Za-z0-9-]+$ ]] || [ ! -f "/sys/class/drm/$CONNECTOR/status" ]; then
    t "Connector $CONNECTOR not found. Current connectors:" "找不到接頭 $CONNECTOR。目前的接頭：" >&2
    list_connectors >&2
    exit 1
fi
CARDNUM=${CONNECTOR%%-*}
CARDNUM=${CARDNUM#card}
KCONN=${CONNECTOR#*-} # 核心參數用的名稱，例如 HDMI-A-1
PARAMS="drm.edid_firmware=$KCONN:edid/$EDID_NAME video=$KCONN:e"

t "==> Installing the EDID, helper and polkit rule" "==> 安裝 EDID、小幫手與 polkit 規則"
# EDID 的優先順序：一般使用者階段依使用者的解析度產生的 → 重新安裝時原本的 → 預設的 1920x1080
if [ -n "$EDID_IN" ] && [ -f "$EDID_IN" ] && [ "$(wc -c < "$EDID_IN")" -eq 128 ]; then
    install -D -m 644 "$EDID_IN" "$FIRMWARE"
elif [ ! -f "$FIRMWARE" ]; then
    install -D -m 644 "$DEFAULT_EDID" "$FIRMWARE"
fi
rm -f "/lib/firmware/edid/$OLD_EDID_NAME" # 0.3.0 以前的檔名
printf 'CONNECTOR=%s\nLANGUAGE=%s\nVERSION=%s\n' "$CONNECTOR" "$GVD_LANG" "$APP_VERSION" > "$CONF"
chmod 644 "$CONF"
install -D -m 755 "$SCRIPTS/vdswitch-helper" "$LIBEXEC/vdswitch-helper"
install -D -m 644 "$SCRIPTS/io.littleherry.vdswitch.policy" /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy

if [ -x "$APP" ]; then
    install -D -m 755 "$APP" /usr/local/bin/vdswitch
    install -D -m 644 "$SCRIPTS/vdswitch.desktop" /usr/share/applications/vdswitch.desktop
    t "==> Installed the switch to /usr/local/bin/vdswitch" "==> 已安裝開關程式 /usr/local/bin/vdswitch"
else
    t "==> $APP not found; skipping the switch" "==> 找不到 $APP，略過開關程式"
fi

# 顯示卡驅動若在 initramfs 階段載入，EDID 也必須在 initramfs 裡
t "==> Adding the EDID to the initramfs" "==> 把 EDID 加進 initramfs"
if command -v update-initramfs >/dev/null; then
    install -D -m 755 "$SCRIPTS/initramfs-hook" /etc/initramfs-tools/hooks/go-virtual-display
    update-initramfs -u
elif command -v dracut >/dev/null; then
    echo "install_items+=\" $FIRMWARE \"" > /etc/dracut.conf.d/go-virtual-display.conf
    dracut -f
else
    t "    Cannot do this automatically (neither initramfs-tools nor dracut). If it does not work after a reboot, add $FIRMWARE to your initramfs manually." \
        "    無法自動處理（不是 initramfs-tools 或 dracut）。若重開機後沒有生效，請手動把 $FIRMWARE 加進 initramfs。"
fi

t "==> Setting kernel parameters: $PARAMS" "==> 設定開機參數：$PARAMS"
BOOT_OK=0
if [ -f "$GRUB_DEFAULT_FILE" ]; then
    [ -f "$GRUB_DEFAULT_FILE.go-virtual-display.bak" ] || cp "$GRUB_DEFAULT_FILE" "$GRUB_DEFAULT_FILE.go-virtual-display.bak"
    # 先移除舊的設定（含換接頭前的）再加上新的，重複安裝不會累積
    strip_params "$OLD_KCONN"
    strip_params "$KCONN"
    # PARAMS 含有 /，所以這裡用 | 當 sed 的分隔符號
    sed -i -E "/^GRUB_CMDLINE_LINUX_DEFAULT=/ { s|\"\$| $PARAMS\"|; s|=\" |=\"| }" "$GRUB_DEFAULT_FILE"
    if command -v update-grub >/dev/null; then
        update-grub && BOOT_OK=1
    elif command -v grub2-mkconfig >/dev/null; then
        grub2-mkconfig -o /boot/grub2/grub.cfg && BOOT_OK=1
    elif command -v grub-mkconfig >/dev/null; then
        grub-mkconfig -o /boot/grub/grub.cfg && BOOT_OK=1
    fi
fi
if [ "$BOOT_OK" -ne 1 ]; then
    t "    GRUB not found. Please add these kernel parameters manually: $PARAMS" "    找不到 GRUB，請手動把下列參數加到開機參數：$PARAMS"
fi

# 有 debugfs 時不用重開機，直接套用一次
OVERRIDE="/sys/kernel/debug/dri/$CARDNUM/$KCONN/edid_override"
APPLIED=0
if [ -w "$OVERRIDE" ] && cat "$FIRMWARE" > "$OVERRIDE" 2>/dev/null; then
    # 已經插入時（重新安裝）要先拔再插，核心才會重新讀取解析度
    if [ "$(cat "/sys/class/drm/$CONNECTOR/status")" = connected ]; then
        echo off > "/sys/class/drm/$CONNECTOR/status"
        sleep 1
    fi
    echo on > "/sys/class/drm/$CONNECTOR/status"
    APPLIED=1
    t "==> Applied the EDID and plugged in the virtual display" "==> 已直接套用 EDID 並插入虛擬螢幕"
fi

echo
t "Installation complete (connector $CONNECTOR)." "安裝完成（接頭 $CONNECTOR）。"
if [ "$APPLIED" -ne 1 ]; then
    t "Please reboot for the settings to take effect, then check with $SCRIPTS/verify.sh." "請重新開機讓設定生效，之後可用 $SCRIPTS/verify.sh 檢查。"
else
    t "It will also take effect automatically after a reboot. Check with $SCRIPTS/verify.sh." "重新開機後也會自動生效。可用 $SCRIPTS/verify.sh 檢查。"
fi
