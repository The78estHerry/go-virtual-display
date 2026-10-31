#!/usr/bin/env bash
# 移除 install.sh 安裝的檔案，並把接頭恢復為自動偵測。開機參數需要自己從 GRUB 設定移除。
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "請用 sudo 執行。" >&2
    exit 1
fi
CONNECTOR=$(sed -n 's/^CONNECTOR=//p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1)
if [ -n "$CONNECTOR" ] && [ -f "/sys/class/drm/$CONNECTOR/status" ]; then
    echo detect > "/sys/class/drm/$CONNECTOR/status"
fi
pkill -x vdswitch 2>/dev/null || true
rm -f /etc/go-virtual-display.conf \
    /usr/share/polkit-1/actions/io.littleherry.vdswitch.policy \
    /usr/local/bin/vdswitch \
    /usr/share/applications/vdswitch.desktop \
    /lib/firmware/edid/edid_1920x1440.bin
rm -rf /usr/local/libexec/go-virtual-display
echo "已移除。請記得從開機參數刪掉 drm.edid_firmware=... 與 video=...:e，並重新產生 GRUB 設定。"
echo "各使用者的 ~/.config/autostart/go-virtual-display.desktop 可在開關程式選單取消「登入時自動啟動」。"
