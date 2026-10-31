#!/usr/bin/env bash
# 一鍵安裝：把一個沒接螢幕的顯示卡接頭設成「接著一台 1920x1440 螢幕」（使用自訂 EDID），
# 安裝免密碼的拔插小幫手與開關程式，並自動加上開機參數。
#
# 用法：linux/install.sh            自動挑選沒接螢幕的外接接頭（HDMI／DP 等）
#       linux/install.sh <接頭>     指定接頭，例如 card0-HDMI-A-1
# 以一般使用者執行即可，需要 root 的部分會自動透過 sudo 執行。
# 介面語言：互動執行時讓使用者選，預設依系統語系（中文用繁體中文，其他英文）。
set -euo pipefail

REPO=$(cd "$(dirname "$0")/.." && pwd)
CONF=/etc/go-virtual-display.conf
LIBEXEC=/usr/local/libexec/go-virtual-display
EDID_NAME=edid_1920x1440.bin
GRUB_DEFAULT_FILE=/etc/default/grub

# root 階段由一般使用者階段帶入 --lang
if [ "${1:-}" = --lang ]; then
    GVD_LANG=$2
    shift 2
fi
# shellcheck source=scripts/lang.sh
. "$REPO/linux/scripts/lang.sh"

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

# ---------- 一般使用者階段：選語言、編譯、提權、開啟開關程式 ----------
if [ "$(id -u)" -ne 0 ]; then
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
    if [ ! -x "$REPO/dist/linux/vdswitch" ] && command -v go >/dev/null; then
        t "==> Building the switch ..." "==> 編譯開關程式 ..."
        if ! (cd "$REPO" && make linux); then
            t "Build failed; continuing without the switch (see the top of the Makefile for required packages)." \
                "編譯失敗，繼續安裝但略過開關程式（缺少的套件見 Makefile 開頭）。"
        fi
    fi
    t "==> Administrator permission is required. Please enter your password:" "==> 需要系統管理員權限，請輸入密碼："
    sudo bash "$0" --lang "$GVD_LANG" "$@"
    if [ -x /usr/local/bin/vdswitch ] && [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
        pkill -x vdswitch 2>/dev/null || true
        setsid /usr/local/bin/vdswitch >/dev/null 2>&1 < /dev/null &
        t "==> Opened Virtual Display Switch (tray and desktop widget)" "==> 已開啟虛擬螢幕開關（系統列與桌面小工具）"
    fi
    exit 0
fi

# ---------- root 階段 ----------
# 只移除本專案加上的參數：指向我們 EDID 的 drm.edid_firmware，以及指定接頭的 video=<接頭>:e
strip_params() {
    [ -n "$1" ] || return 0
    sed -i -E "/^GRUB_CMDLINE_LINUX_DEFAULT=/ { s/ ?drm\.edid_firmware=[^\" ]*$EDID_NAME//g; s/ ?video=$1:e//g; s/=\" /=\"/ }" "$GRUB_DEFAULT_FILE"
}
OLD_CONNECTOR=$(sed -n 's/^CONNECTOR=\(card[0-9]*-[A-Za-z0-9-]*\)$/\1/p' "$CONF" 2>/dev/null | head -n 1 || true)
OLD_KCONN=${OLD_CONNECTOR#*-}
CONNECTOR=${1:-$OLD_CONNECTOR} # 重新安裝時沿用原本的接頭
if [ -z "$CONNECTOR" ]; then
    if ! CONNECTOR=$(pick_connector); then
        t "No free external connector found. Current connectors:" "找不到沒接螢幕的外接接頭。目前的接頭：" >&2
        list_connectors >&2
        t "Use linux/install.sh <connector> to choose one. Machines without a physical GPU (VMs, servers) are not supported." \
            "可以用 linux/install.sh <接頭> 指定。沒有實體顯示卡的機器（VM、伺服器）不適用。" >&2
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
install -D -m 644 "$REPO/edid/$EDID_NAME" "/lib/firmware/edid/$EDID_NAME"
printf 'CONNECTOR=%s\nLANGUAGE=%s\n' "$CONNECTOR" "$GVD_LANG" > "$CONF"
chmod 644 "$CONF"
install -D -m 755 "$REPO/linux/scripts/vdswitch-helper" "$LIBEXEC/vdswitch-helper"
install -D -m 644 "$REPO/linux/scripts/io.littleherry.vdswitch.policy" /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy

if [ -x "$REPO/dist/linux/vdswitch" ]; then
    install -D -m 755 "$REPO/dist/linux/vdswitch" /usr/local/bin/vdswitch
    install -D -m 644 "$REPO/linux/scripts/vdswitch.desktop" /usr/share/applications/vdswitch.desktop
    t "==> Installed the switch to /usr/local/bin/vdswitch" "==> 已安裝開關程式 /usr/local/bin/vdswitch"
else
    t "==> dist/linux/vdswitch not found; skipping the switch" "==> 找不到 dist/linux/vdswitch，略過開關程式"
fi

# 顯示卡驅動若在 initramfs 階段載入，EDID 也必須在 initramfs 裡
t "==> Adding the EDID to the initramfs" "==> 把 EDID 加進 initramfs"
if command -v update-initramfs >/dev/null; then
    install -D -m 755 "$REPO/linux/scripts/initramfs-hook" /etc/initramfs-tools/hooks/go-virtual-display
    update-initramfs -u
elif command -v dracut >/dev/null; then
    echo "install_items+=\" /lib/firmware/edid/$EDID_NAME \"" > /etc/dracut.conf.d/go-virtual-display.conf
    dracut -f
else
    t "    Cannot do this automatically (neither initramfs-tools nor dracut). If it does not work after a reboot, add /lib/firmware/edid/$EDID_NAME to your initramfs manually." \
        "    無法自動處理（不是 initramfs-tools 或 dracut）。若重開機後沒有生效，請手動把 /lib/firmware/edid/$EDID_NAME 加進 initramfs。"
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
if [ -w "$OVERRIDE" ]; then
    cat "/lib/firmware/edid/$EDID_NAME" > "$OVERRIDE"
    echo on > "/sys/class/drm/$CONNECTOR/status"
    APPLIED=1
    t "==> Applied the EDID and plugged in the virtual display" "==> 已直接套用 EDID 並插入虛擬螢幕"
fi

echo
t "Installation complete (connector $CONNECTOR)." "安裝完成（接頭 $CONNECTOR）。"
if [ "$APPLIED" -ne 1 ]; then
    t "Please reboot for the settings to take effect, then check with linux/scripts/verify.sh." "請重新開機讓設定生效，之後可用 linux/scripts/verify.sh 檢查。"
else
    t "It will also take effect automatically after a reboot. Check with linux/scripts/verify.sh." "重新開機後也會自動生效。可用 linux/scripts/verify.sh 檢查。"
fi
