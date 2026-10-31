#!/usr/bin/env bash
# 一鍵移除：移除 install.sh 安裝的所有檔案、開機參數與 initramfs 設定，並把接頭恢復為自動偵測。
# 也會刪掉使用者的設定（存下的解析度、小工具位置），和 Windows 版一致。
# 以一般使用者執行即可，需要 root 的部分會自動透過 sudo 執行。
#   --keep-user-settings  保留使用者的設定（存下的解析度、小工具位置、登入時自動啟動）。install.sh 覆蓋安裝時先移除舊版會用到
set -euo pipefail

KEEP_USER_SETTINGS=0
[ "${1:-}" = --keep-user-settings ] && KEEP_USER_SETTINGS=1

GRUB_DEFAULT_FILE=/etc/default/grub
# shellcheck source=scripts/lang.sh
. "$(cd "$(dirname "$0")" && pwd)/scripts/lang.sh" # 設定檔刪掉前先讀出安裝時選的語言

if [ "$(id -u)" -ne 0 ]; then
    pkill -x vdswitch 2>/dev/null || true
    if [ "$KEEP_USER_SETTINGS" -ne 1 ]; then
        rm -f "${XDG_CONFIG_HOME:-$HOME/.config}/autostart/go-virtual-display.desktop"
        rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/go-virtual-display" # 存下的解析度、小工具位置
    fi
    t "==> Administrator permission is required. Please enter your password:" "==> 需要系統管理員權限，請輸入密碼："
    exec sudo bash "$0" "$@"
fi

CONNECTOR=$(sed -n 's/^CONNECTOR=\(card[0-9]*-[A-Za-z0-9-]*\)$/\1/p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1 || true)
if [ -n "$CONNECTOR" ] && [ -f "/sys/class/drm/$CONNECTOR/status" ]; then
    echo detect > "/sys/class/drm/$CONNECTOR/status"
fi
pkill -x vdswitch 2>/dev/null || true

t "==> Removing files" "==> 移除檔案"
rm -f /etc/go-virtual-display.conf \
    /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy \
    /usr/local/bin/vdswitch \
    /usr/share/applications/vdswitch.desktop \
    /lib/firmware/edid/go-virtual-display.bin \
    /lib/firmware/edid/edid_1920x1440.bin
rm -f /lib/firmware/edid/go-virtual-display.bin.* # 小幫手寫入時中斷留下的暫存檔
rm -rf /usr/local/libexec/go-virtual-display

t "==> Removing kernel parameters" "==> 移除開機參數"
if [ -f "$GRUB_DEFAULT_FILE" ]; then
    # 只移除本專案加上的參數（含 0.3.0 以前的 EDID 檔名）
    KCONN=${CONNECTOR#*-}
    sed -i -E "/^GRUB_CMDLINE_LINUX_DEFAULT=/ { s/ ?drm\.edid_firmware=[^\" ]*(go-virtual-display|edid_1920x1440)\.bin//g; s/ ?video=${KCONN:-NONE}:e//g; s/=\" /=\"/ }" "$GRUB_DEFAULT_FILE"
    rm -f "$GRUB_DEFAULT_FILE.go-virtual-display.bak"
    if command -v update-grub >/dev/null; then
        update-grub
    elif command -v grub2-mkconfig >/dev/null; then
        grub2-mkconfig -o /boot/grub2/grub.cfg
    elif command -v grub-mkconfig >/dev/null; then
        grub-mkconfig -o /boot/grub/grub.cfg
    fi
fi

t "==> Removing initramfs settings" "==> 移除 initramfs 設定"
if [ -f /etc/initramfs-tools/hooks/go-virtual-display ]; then
    rm -f /etc/initramfs-tools/hooks/go-virtual-display
    update-initramfs -u
fi
if [ -f /etc/dracut.conf.d/go-virtual-display.conf ]; then
    rm -f /etc/dracut.conf.d/go-virtual-display.conf
    dracut -f
fi

if [ "$KEEP_USER_SETTINGS" -ne 1 ]; then
    echo
    t "Uninstall complete. Everything is back to the original state after a reboot." "移除完成。重新開機後就完全恢復原狀。"
fi
