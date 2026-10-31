#!/usr/bin/env bash
# 在 Linux 上把一個沒接螢幕的顯示卡接頭設成「接著一台 1920x1440 螢幕」，並安裝免密碼的拔插小幫手。
# 用法：sudo linux/install.sh <接頭>，例如 sudo linux/install.sh card0-HDMI-A-1
#       不帶參數時列出所有接頭。
set -euo pipefail

REPO=$(cd "$(dirname "$0")/.." && pwd)
CONF=/etc/go-virtual-display.conf
LIBEXEC=/usr/local/libexec/go-virtual-display
EDID_NAME=edid_1920x1440.bin

list_connectors() {
    echo "目前的顯示接頭（請選 disconnected 的那個）："
    for d in /sys/class/drm/card*-*; do
        [ -f "$d/status" ] && printf '  %-24s %s\n' "$(basename "$d")" "$(cat "$d/status")"
    done
}

CONNECTOR=${1:-}
if [ -z "$CONNECTOR" ]; then
    list_connectors
    echo
    echo "用法：sudo $0 <接頭>，例如 sudo $0 card0-HDMI-A-1"
    exit 1
fi
if [ "$(id -u)" -ne 0 ]; then
    echo "請用 sudo 執行。" >&2
    exit 1
fi
if ! [[ "$CONNECTOR" =~ ^card[0-9]+-[A-Za-z0-9-]+$ ]] || [ ! -f "/sys/class/drm/$CONNECTOR/status" ]; then
    echo "找不到接頭 $CONNECTOR。" >&2
    list_connectors
    exit 1
fi
CARD=${CONNECTOR%%-*}      # card0
KCONN=${CONNECTOR#*-}      # HDMI-A-1（核心參數用的名稱）
CARDNUM=${CARD#card}

# 1. EDID 放到韌體目錄，供核心參數 drm.edid_firmware 讀取
install -D -m 644 "$REPO/edid/$EDID_NAME" "/lib/firmware/edid/$EDID_NAME"

# 2. 設定檔與小幫手、polkit 規則（使用中的本機使用者免密碼）
echo "CONNECTOR=$CONNECTOR" > "$CONF"
chmod 644 "$CONF"
install -D -m 755 "$REPO/linux/vdswitch-helper" "$LIBEXEC/vdswitch-helper"
install -D -m 644 "$REPO/linux/io.littleherry.vdswitch.policy" /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy

# 3. 開關程式
if [ -x "$REPO/dist/linux/vdswitch" ]; then
    install -D -m 755 "$REPO/dist/linux/vdswitch" /usr/local/bin/vdswitch
    install -D -m 644 "$REPO/linux/vdswitch.desktop" /usr/share/applications/vdswitch.desktop
    echo "已安裝開關程式 /usr/local/bin/vdswitch"
else
    echo "找不到 dist/linux/vdswitch，略過開關程式（先執行 make linux）"
fi

# 4. 不重開機先套用：有 debugfs 時直接覆寫 EDID 並強制接上
OVERRIDE="/sys/kernel/debug/dri/$CARDNUM/$KCONN/edid_override"
if [ -w "$OVERRIDE" ]; then
    cat "/lib/firmware/edid/$EDID_NAME" > "$OVERRIDE"
    echo on > "/sys/class/drm/$CONNECTOR/status"
    echo "已透過 debugfs 立即套用 EDID 並插入虛擬螢幕"
fi

echo
echo "完成。要讓設定在每次開機都生效，請把下列參數加到開機參數（例如 /etc/default/grub 的 GRUB_CMDLINE_LINUX_DEFAULT）："
echo
echo "    drm.edid_firmware=$KCONN:edid/$EDID_NAME video=$KCONN:e"
echo
echo "然後執行 update-grub（或 grub2-mkconfig -o /boot/grub2/grub.cfg）。"
echo "若顯示卡驅動在 initramfs 階段就載入，也要把 EDID 檔加進 initramfs（update-initramfs -u 或 dracut -f，視發行版而定）。"
echo "之後用 linux/verify.sh 檢查。"
