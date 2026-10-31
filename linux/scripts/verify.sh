#!/usr/bin/env bash
# 檢查虛擬螢幕：接頭狀態、可用的解析度、系統讀到的 EDID 是否與 edid/edid_1920x1440.bin 相同。不需要 root。
set -uo pipefail

REPO=$(cd "$(dirname "$0")/../.." && pwd)
# shellcheck source=lang.sh
. "$REPO/linux/scripts/lang.sh"
CONNECTOR=$(sed -n 's/^CONNECTOR=//p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1)
if [ -z "$CONNECTOR" ]; then
    t "[X] /etc/go-virtual-display.conf not found. Please run install.sh first." "[X] 找不到 /etc/go-virtual-display.conf，請先執行 install.sh"
    exit 1
fi
DIR="/sys/class/drm/$CONNECTOR"
ok=0

t "[i] Connector $CONNECTOR: $(cat "$DIR/status")" "[i] 接頭 $CONNECTOR：$(cat "$DIR/status")"
if grep -qx '1920x1440' "$DIR/modes" 2>/dev/null; then
    t "[O] Available modes include 1920x1440" "[O] 可用解析度包含 1920x1440"
else
    modes=$(tr '\n' ' ' < "$DIR/modes" 2>/dev/null)
    t "[!] Available modes do not include 1920x1440 (unplugged, or the kernel parameters are not in effect yet): $modes" \
        "[!] 可用解析度沒有 1920x1440（拔掉狀態或開機參數尚未生效）：$modes"
    ok=1
fi
if [ -s "$DIR/edid" ] && cmp -s <(head -c 128 "$DIR/edid") "$REPO/edid/edid_1920x1440.bin"; then
    t "[O] The EDID read by the system is identical to edid_1920x1440.bin" "[O] 系統讀到的 EDID 與 edid_1920x1440.bin 相同"
else
    t "[X] The EDID read by the system is different or empty" "[X] 系統讀到的 EDID 不同或是空的"
    ok=1
fi
if command -v xrandr >/dev/null && [ -n "${DISPLAY:-}" ]; then
    echo "[i] xrandr:"
    xrandr --query | grep -A1 ' connected' | sed 's/^/    /'
fi
exit $ok
