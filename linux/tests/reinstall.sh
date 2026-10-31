#!/usr/bin/env bash
# 測試 install.sh 的「已安裝時先提醒」：以一般使用者 tester 執行（真的 sudo，免密碼），用 script 提供終端機。
# 系統路徑換成 $F 底下的假目錄。/src 是 repo、/data/e2560.bin 是 2560x1440 的 EDID（由 linux/tests/run.ps1 掛載）。
set -uo pipefail
F=/fake
PASS=0
FAIL=0
ok() { echo "PASS: $1"; PASS=$((PASS + 1)); }
ng() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); }
check() { if eval "$2"; then ok "$1"; else ng "$1"; fi; }

export DEBIAN_FRONTEND=noninteractive
apt-get -qq update > /dev/null && apt-get install -y -qq sudo > /dev/null 2>&1
useradd -m tester
echo 'tester ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/tester

# ---------- 準備：原始碼配置（linux/、edid/、dist/linux/、windows/scripts/common.ps1）並改寫路徑 ----------
rm -rf /t "$F"
mkdir -p /t/linux/scripts /t/edid /t/dist/linux /t/windows/scripts
cp /src/linux/install.sh /src/linux/uninstall.sh /t/linux/
cp /src/linux/scripts/* /t/linux/scripts/
cp /src/edid/*.bin /t/edid/
cp /src/windows/scripts/common.ps1 /t/windows/scripts/
VER=$(sed -n "s/^\$AppVersion *= *'\([^']*\)'.*/\1/p" /t/windows/scripts/common.ps1)
for f in /t/linux/*.sh /t/linux/scripts/*; do
    sed -i -e "s#/sys/class/drm#$F/sys/class/drm#g" -e "s#/sys/kernel/debug#$F/debug#g" \
        -e "s#/lib/firmware#$F/fw#g" -e "s#/etc/#$F/etc/#g" -e "s#/run/#$F/run/#g" \
        -e "s#/usr/local/#$F/usr/local/#g" -e "s#/usr/share/#$F/usr/share/#g" "$f"
done
sed -i "s#GRUB_DEFAULT_FILE=/etc/default/grub#GRUB_DEFAULT_FILE=$F/etc/default/grub#" /t/linux/install.sh /t/linux/uninstall.sh
cp /data/e2560.bin /tmp/e2560.bin && chmod 644 /tmp/e2560.bin
cat > /t/dist/linux/vdswitch << 'EOF'
#!/bin/sh
if [ "$1" = --write-edid ]; then cp /tmp/e2560.bin "$2"; echo 2560x1440@60; exit 0; fi
exit 0
EOF
chmod -R a+rX /t && chmod +x /t/dist/linux/vdswitch

mkdir -p "$F/sys/class/drm/card0-HDMI-A-1" "$F/sys/class/drm/card0-DP-2" "$F/etc/default" "$F/run" "$F/fw/edid" "$F/etc/initramfs-tools/hooks"
echo disconnected > "$F/sys/class/drm/card0-HDMI-A-1/status"
echo disconnected > "$F/sys/class/drm/card0-DP-2/status"
echo 'GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"' > "$F/etc/default/grub"
chmod -R a+rX "$F"
# 假的系統指令放在 sudo 的 secure_path 裡
for c in update-grub update-initramfs; do
    printf '#!/bin/sh\necho "%s $*" >> %s/run/calls\n' "$c" "$F" > "/usr/local/sbin/$c"
    chmod +x "/usr/local/sbin/$c"
done
printf '#!/bin/sh\nexit 0\n' > /usr/local/sbin/pkill && chmod +x /usr/local/sbin/pkill

# 以 tester 在終端機裡執行 install.sh，input 是要回答的內容
run_tty() { # $1=輸入 $2=紀錄檔 其餘=install.sh 參數
    local input=$1 log=$2
    shift 2
    printf "$input" | sudo -u tester -H script -qec "bash /t/linux/install.sh $*" /dev/null > "$log" 2>&1
}
CONF=$F/etc/go-virtual-display.conf
FW=$F/fw/edid/go-virtual-display.bin
CFG=/home/tester/.config

# ---------- 1. 第一次安裝（沒有設定檔）：不會問「已安裝」，接頭指定 DP-2 ----------
run_tty '1\n' /tmp/i1.log card0-DP-2
check "first install: no already-installed prompt" "! grep -q 'already installed' /tmp/i1.log"
check "first install: conf has VERSION=$VER" "grep -qx 'VERSION=$VER' $CONF"
check "first install: connector DP-2" "grep -qx 'CONNECTOR=card0-DP-2' $CONF"
check "first install: firmware = user's EDID" "cmp -s $FW /tmp/e2560.bin"

# 使用者設定
sudo -u tester mkdir -p $CFG/go-virtual-display $CFG/autostart
echo '{"active":"2560x1440@60"}' | sudo -u tester tee $CFG/go-virtual-display/profiles.json > /dev/null
sudo -u tester touch $CFG/autostart/go-virtual-display.desktop

# ---------- 2. 再裝一次，回答 n：取消，什麼都不動 ----------
cp $CONF /tmp/conf.before
: > $F/run/calls
run_tty 'n\n' /tmp/i2.log
cat /tmp/i2.log | sed 's/^/    | /'
check "prompt shown" "grep -q 'Virtual Display Switch is already installed' /tmp/i2.log"
check "shows installed $VER and package $VER" "grep -q \"Installed version: $VER\" /tmp/i2.log && grep -q \"This package:      $VER\" /tmp/i2.log"
check "same version note" "grep -q 'Same version' /tmp/i2.log"
check "n → cancelled" "grep -q 'Cancelled. Nothing was changed.' /tmp/i2.log"
check "n → conf unchanged, no grub/initramfs calls" "cmp -s $CONF /tmp/conf.before && [ ! -s $F/run/calls ]"

# ---------- 3. 回答 y：先移除再安裝，保留接頭、語言、使用者設定 ----------
sed -i 's/^LANGUAGE=.*/LANGUAGE=zh-TW/' $CONF
: > $F/run/calls
run_tty 'y\n\n' /tmp/i3.log
cat /tmp/i3.log | sed 's/^/    | /'
check "y → uninstall then install messages (zh-TW)" "grep -q '移除目前的版本' /tmp/i3.log && grep -q \"安裝版本 $VER\" /tmp/i3.log"
check "y → uninstall's 'complete' message suppressed" "! grep -q '移除完成' /tmp/i3.log"
check "y → connector DP-2 kept" "grep -qx 'CONNECTOR=card0-DP-2' $CONF"
check "y → language zh-TW kept (Enter = previous)" "grep -qx 'LANGUAGE=zh-TW' $CONF"
check "y → profiles.json kept" "[ -f $CFG/go-virtual-display/profiles.json ]"
check "y → autostart kept" "[ -f $CFG/autostart/go-virtual-display.desktop ]"
check "y → grub regenerated twice (uninstall + install)" "[ \$(grep -c update-grub $F/run/calls) -eq 2 ]"
check "y → grub params once" "[ \$(grep -o 'go-virtual-display.bin' $F/etc/default/grub | wc -l) -eq 1 ]"
check "y → firmware reinstalled" "cmp -s $FW /tmp/e2560.bin"

# ---------- 4. 目前的版本比較新：降版提示 ----------
sed -i 's/^VERSION=.*/VERSION=9.9.9/' $CONF
sed -i 's/^LANGUAGE=.*/LANGUAGE=en/' $CONF
run_tty 'n\n' /tmp/i4.log
check "downgrade warning" "grep -q 'this will downgrade it' /tmp/i4.log"

# ---------- 5. 舊版本號比較舊：沒有特別提示 ----------
sed -i 's/^VERSION=.*/VERSION=0.3.9/' $CONF
run_tty 'n\n' /tmp/i5.log
check "upgrade: no same/downgrade note" "! grep -q 'Same version' /tmp/i5.log && ! grep -q downgrade /tmp/i5.log && grep -q 'Installed version: 0.3.9' /tmp/i5.log"

# ---------- 6. 0.4.0 以前沒有 VERSION ----------
sed -i '/^VERSION=/d' $CONF
run_tty 'n\n' /tmp/i6.log
check "no VERSION → unknown (0.4.0 or earlier)" "grep -q 'Installed version: unknown (0.4.0 or earlier)' /tmp/i6.log"

# ---------- 7. 沒有終端機：直接覆蓋安裝 ----------
: > $F/run/calls
sudo -u tester -H bash /t/linux/install.sh < /dev/null > /tmp/i7.log 2>&1
check "no tty → overwrite install, no uninstall" "grep -q 'installing $VER over it' /tmp/i7.log && [ \$(grep -c update-grub $F/run/calls) -eq 1 ]"
check "no tty → VERSION written again" "grep -qx 'VERSION=$VER' $CONF"

# ---------- 8. 一般移除：使用者設定刪掉 ----------
sudo -u tester -H bash /t/linux/uninstall.sh > /tmp/u.log 2>&1
check "plain uninstall removes user settings" "[ ! -e $CFG/go-virtual-display ] && [ ! -e $CFG/autostart/go-virtual-display.desktop ]"
check "plain uninstall shows complete message" "grep -q 'Uninstall complete' /tmp/u.log"
check "plain uninstall removes conf" "[ ! -e $CONF ]"

echo
echo "RESULT: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ]
