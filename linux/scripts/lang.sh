# shellcheck shell=bash
# 介面語言（install.sh、uninstall.sh、verify.sh 共用，以 source 載入）。
# 順序：已設定的 GVD_LANG（install.sh 傳給 root 階段）→ /etc/go-virtual-display.conf 的 LANGUAGE（安裝時選的）
# → 系統語系：中文用繁體中文，其他一律英文。
gvd_detect_lang() {
    local l
    l=$(sed -n 's/^LANGUAGE=//p' /etc/go-virtual-display.conf 2>/dev/null | head -n 1)
    case "${l:-${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}}" in
        zh*) echo zh-TW ;;
        *) echo en ;;
    esac
}
case "${GVD_LANG:-}" in
    en | zh-TW) ;;
    *) GVD_LANG=$(gvd_detect_lang) ;;
esac

# t <英文> <繁體中文>：依介面語言印出其中一個
t() {
    if [ "$GVD_LANG" = zh-TW ]; then printf '%s\n' "$2"; else printf '%s\n' "$1"; fi
}
