#!/usr/bin/env bash
# 一鍵移除：移除 install.sh 安裝的所有檔案、開機參數與 initramfs 設定，並把接頭恢復為自動偵測。
# 以一般使用者執行即可，需要 root 的部分會自動透過 sudo 執行。
set -euo pipefail

GRUB_DEFAULT_FILE=/etc/default/grub

if [ "$(id -u)" -ne 0 ]; then
    pkill -x vdswitch 2>/dev/null || true
    rm -f "${XDG_CONFIG_HOME:-$HOME/.config}/autostart/go-virtual-display.desktop"
    echo "==> 需要系統管理員權限，請輸入密碼："
    exec sudo bash "$0" "$@"
fi

CONNECTOR=$(sed -n 's/^CONNECTOR=\(card[0-9]*-[A-Za-z0-9-]*\)$/\1/p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1 || true)
if [ -n "$CONNECTOR" ] && [ -f "/sys/class/drm/$CONNECTOR/status" ]; then
    echo detect > "/sys/class/drm/$CONNECTOR/status"
fi
pkill -x vdswitch 2>/dev/null || true

echo "==> 移除檔案"
rm -f /etc/go-virtual-display.conf \
    /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy \
    /usr/local/bin/vdswitch \
    /usr/share/applications/vdswitch.desktop \
    /lib/firmware/edid/edid_1920x1440.bin
rm -rf /usr/local/libexec/go-virtual-display

echo "==> 移除開機參數"
if [ -f "$GRUB_DEFAULT_FILE" ]; then
    # 只移除本專案加上的參數
    KCONN=${CONNECTOR#*-}
    sed -i -E "/^GRUB_CMDLINE_LINUX_DEFAULT=/ { s/ ?drm\.edid_firmware=[^\" ]*edid_1920x1440\.bin//g; s/ ?video=${KCONN:-NONE}:e//g; s/=\" /=\"/ }" "$GRUB_DEFAULT_FILE"
    rm -f "$GRUB_DEFAULT_FILE.go-virtual-display.bak"
    if command -v update-grub >/dev/null; then
        update-grub
    elif command -v grub2-mkconfig >/dev/null; then
        grub2-mkconfig -o /boot/grub2/grub.cfg
    elif command -v grub-mkconfig >/dev/null; then
        grub-mkconfig -o /boot/grub/grub.cfg
    fi
fi

echo "==> 移除 initramfs 設定"
if [ -f /etc/initramfs-tools/hooks/go-virtual-display ]; then
    rm -f /etc/initramfs-tools/hooks/go-virtual-display
    update-initramfs -u
fi
if [ -f /etc/dracut.conf.d/go-virtual-display.conf ]; then
    rm -f /etc/dracut.conf.d/go-virtual-display.conf
    dracut -f
fi

echo
echo "移除完成。重新開機後就完全恢復原狀。"
