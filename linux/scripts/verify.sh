#!/usr/bin/env bash
# 檢查虛擬螢幕：接頭狀態、可用的解析度、系統讀到的 EDID 是否與安裝的 EDID 相同。不需要 root。
# 安裝的 EDID 是 /lib/firmware/edid/go-virtual-display.bin，解析度切換時由小幫手換掉。
set -uo pipefail

# shellcheck source=lang.sh
. "$(cd "$(dirname "$0")" && pwd)/lang.sh"
FIRMWARE=/lib/firmware/edid/go-virtual-display.bin
CONNECTOR=$(sed -n 's/^CONNECTOR=//p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1)
if [ -z "$CONNECTOR" ]; then
    t "[X] /etc/go-virtual-display.conf not found. Please run install.sh first." "[X] 找不到 /etc/go-virtual-display.conf，請先執行 install.sh"
    exit 1
fi
if [ ! -s "$FIRMWARE" ]; then
    t "[X] $FIRMWARE not found. Please run install.sh again." "[X] 找不到 $FIRMWARE，請重新執行 install.sh"
    exit 1
fi
DIR="/sys/class/drm/$CONNECTOR"
ok=0

# 從 EDID 第一個 detailed timing（byte 54 起）讀出解析度
read -r -a b <<< "$(od -An -tu1 -v -j 54 -N 8 "$FIRMWARE")"
WIDTH=$((b[2] + ((b[4] >> 4) << 8)))
HEIGHT=$((b[5] + ((b[7] >> 4) << 8)))
MODE="${WIDTH}x${HEIGHT}"
t "[i] Installed EDID: $MODE" "[i] 安裝的 EDID：$MODE"

t "[i] Connector $CONNECTOR: $(cat "$DIR/status")" "[i] 接頭 $CONNECTOR：$(cat "$DIR/status")"
if grep -qx "$MODE" "$DIR/modes" 2>/dev/null; then
    t "[O] Available modes include $MODE" "[O] 可用解析度包含 $MODE"
else
    modes=$(tr '\n' ' ' < "$DIR/modes" 2>/dev/null)
    t "[!] Available modes do not include $MODE (unplugged, or not in effect until a reboot): $modes" \
        "[!] 可用解析度沒有 $MODE（拔掉狀態，或要重新開機才會生效）：$modes"
    ok=1
fi
if [ -s "$DIR/edid" ] && cmp -s <(head -c 128 "$DIR/edid") "$FIRMWARE"; then
    t "[O] The EDID read by the system is identical to the installed EDID" "[O] 系統讀到的 EDID 與安裝的 EDID 相同"
else
    t "[X] The EDID read by the system is different or empty (a reboot may be needed)" "[X] 系統讀到的 EDID 不同或是空的（可能要重新開機）"
    ok=1
fi
if command -v xrandr >/dev/null && [ -n "${DISPLAY:-}" ]; then
    echo "[i] xrandr:"
    xrandr --query | grep -A1 ' connected' | sed 's/^/    /'
fi
exit $ok
