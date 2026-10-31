#!/usr/bin/env bash
# 在容器裡以 root 模擬測試 Linux 腳本：把系統路徑換成 $F 底下的假目錄。
# /src 是 repo、/data/e2560.bin 是 2560x1440 的 EDID（都由 linux/tests/run.ps1 唯讀掛載）。
set -uo pipefail
F=/fake
PASS=0
FAIL=0
ok() { echo "PASS: $1"; PASS=$((PASS + 1)); }
ng() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); }
check() { if eval "$2"; then ok "$1"; else ng "$1"; fi; }

# ---------- 準備：複製腳本並改寫路徑 ----------
rm -rf /t "$F"
mkdir -p /t/linux/scripts /t/edid /t/dist/linux
cp /src/linux/install.sh /src/linux/uninstall.sh /t/linux/
cp /src/linux/scripts/* /t/linux/scripts/
cp /src/edid/*.bin /t/edid/
for f in /t/linux/*.sh /t/linux/scripts/*; do
    sed -i -e "s#/sys/class/drm#$F/sys/class/drm#g" -e "s#/sys/kernel/debug#$F/debug#g" \
        -e "s#/lib/firmware#$F/fw#g" -e "s#/etc/#$F/etc/#g" -e "s#/run/#$F/run/#g" \
        -e "s#/usr/local/#$F/usr/local/#g" -e "s#/usr/share/#$F/usr/share/#g" "$f"
done
# 假的 vdswitch：--write-edid 寫出 2560x1440 的 EDID（模擬使用者選過解析度）
cat > /t/dist/linux/vdswitch << 'EOF'
#!/bin/sh
if [ "$1" = --write-edid ]; then cp /tmp/e2560.bin "$2"; echo 2560x1440@60; exit 0; fi
exit 0
EOF
chmod +x /t/dist/linux/vdswitch

mkdir -p "$F/sys/class/drm/card0-HDMI-A-1" "$F/sys/class/drm/card0-eDP-1" "$F/etc/default" "$F/run" "$F/fw/edid" "$F/etc/initramfs-tools/hooks"
echo disconnected > "$F/sys/class/drm/card0-HDMI-A-1/status"
echo connected > "$F/sys/class/drm/card0-eDP-1/status"
# 0.3.0 安裝後的狀態：舊檔名的 EDID 和開機參數
cp /t/edid/edid_1920x1440.bin "$F/fw/edid/edid_1920x1440.bin"
echo 'CONNECTOR=card0-HDMI-A-1' > "$F/etc/go-virtual-display.conf"
cat > "$F/etc/default/grub" << 'EOF'
GRUB_DEFAULT=0
GRUB_CMDLINE_LINUX_DEFAULT="quiet splash video=DP-1:1280x720 drm.edid_firmware=HDMI-A-1:edid/edid_1920x1440.bin video=HDMI-A-1:e"
GRUB_CMDLINE_LINUX=""
EOF
sed -i "s#GRUB_DEFAULT_FILE=/etc/default/grub#GRUB_DEFAULT_FILE=$F/etc/default/grub#" /t/linux/install.sh /t/linux/uninstall.sh
grep -q "$F/etc/default/grub" /t/linux/install.sh || ng "grub path rewrite"
# 假的系統指令
mkdir -p /stub
for c in update-grub update-initramfs; do
    printf '#!/bin/sh\necho "%s $*" >> %s/run/calls\n' "$c" "$F" > "/stub/$c"
    chmod +x "/stub/$c"
done
printf '#!/bin/sh\nexit 0\n' > /stub/pkill && chmod +x /stub/pkill
export PATH=/stub:$PATH

# 2560x1440 的 EDID（用 make-edid 產生後放進 /tmp）
cp /data/e2560.bin /tmp/e2560.bin

H=/t/linux/scripts/vdswitch-helper
FW=$F/fw/edid/go-virtual-display.bin

# ---------- 1. 從 0.3.0 升級安裝（root 階段，帶入使用者的 EDID） ----------
cp /tmp/e2560.bin /tmp/user.bin
bash /t/linux/install.sh --lang en --edid /tmp/user.bin > /tmp/install1.log 2>&1
check "install exit 0" "[ \$? -eq 0 ]"
check "new firmware = user's EDID (2560)" "cmp -s $FW /tmp/e2560.bin"
check "old firmware removed" "[ ! -e $F/fw/edid/edid_1920x1440.bin ]"
check "connector kept from conf" "grep -q '^CONNECTOR=card0-HDMI-A-1$' $F/etc/go-virtual-display.conf"
check "grub has new params once" "[ \$(grep -o 'drm.edid_firmware=HDMI-A-1:edid/go-virtual-display.bin' $F/etc/default/grub | wc -l) -eq 1 ]"
check "grub old param removed" "! grep -q edid_1920x1440 $F/etc/default/grub"
check "grub video=HDMI-A-1:e once" "[ \$(grep -o 'video=HDMI-A-1:e' $F/etc/default/grub | wc -l) -eq 1 ]"
check "grub user's params kept" "grep -q 'quiet splash video=DP-1:1280x720' $F/etc/default/grub"
check "initramfs hook uses new name" "grep -q 'go-virtual-display.bin' $F/etc/initramfs-tools/hooks/go-virtual-display"
check "update-grub and update-initramfs called" "grep -q update-grub $F/run/calls && grep -q update-initramfs $F/run/calls"
check "helper installed" "[ -x $F/usr/local/libexec/go-virtual-display/vdswitch-helper ]"
grep 'GRUB_CMDLINE_LINUX_DEFAULT' $F/etc/default/grub

# ---------- 2. 重新安裝（沒有帶 EDID）：保留原本的 ----------
bash /t/linux/install.sh --lang en > /tmp/install2.log 2>&1
check "reinstall keeps existing firmware" "cmp -s $FW /tmp/e2560.bin"
check "reinstall: params not duplicated" "[ \$(grep -o 'go-virtual-display.bin' $F/etc/default/grub | wc -l) -eq 1 ]"

# ---------- 3. 小幫手 edid：沒有 debugfs ----------
echo connected > "$F/sys/class/drm/card0-HDMI-A-1/status"
: > $F/run/calls
out=$(sh $H edid < /t/edid/edid_1920x1080.bin 2>&1)
check "helper edid → reboot-required without debugfs ($out)" "[ '$out' = reboot-required ]"
check "firmware replaced with 1080" "cmp -s $FW /t/edid/edid_1920x1080.bin"
check "firmware mode 644" "[ \$(stat -c %a $FW) = 644 ]"
sleep 1
check "background initramfs rebuild ran" "grep -q 'update-initramfs -u' $F/run/calls"
check "no temp files left" "[ \$(ls $F/fw/edid | wc -l) -eq 1 ]"

# ---------- 4. 小幫手 edid：有 debugfs ----------
mkdir -p "$F/debug/dri/0/HDMI-A-1"
: > "$F/debug/dri/0/HDMI-A-1/edid_override"
out=$(sh $H edid < /tmp/e2560.bin 2>&1)
check "helper edid → applied with debugfs ($out)" "[ '$out' = applied ]"
check "override written" "cmp -s $F/debug/dri/0/HDMI-A-1/edid_override /tmp/e2560.bin"
check "connector re-plugged (status ends on)" "[ \"\$(cat $F/sys/class/drm/card0-HDMI-A-1/status)\" = on ]"

# ---------- 5. 小幫手 edid：錯誤的輸入都要拒絕，且不動到檔案 ----------
cp $FW /tmp/before.bin
head -c 127 /tmp/e2560.bin > /tmp/short.bin
{ cat /tmp/e2560.bin; printf 'x'; } > /tmp/long.bin
{ printf '\001'; tail -c 127 /tmp/e2560.bin; } > /tmp/badhdr.bin
{ head -c 127 /tmp/e2560.bin; printf '\377'; } > /tmp/badsum.bin
for bad in short long badhdr badsum; do
    out=$(sh $H edid < /tmp/$bad.bin 2>&1)
    rc=$?
    check "helper rejects $bad (rc=$rc: $out)" "[ $rc -ne 0 ] && cmp -s $FW /tmp/before.bin"
done
check "no temp files after rejects" "[ \$(ls $F/fw/edid | wc -l) -eq 1 ]"
out=$(sh $H bogus 2>&1); rc=$?
check "helper rejects unknown command" "[ $rc -eq 2 ]"

# ---------- 6. on／off 仍然正常 ----------
sh $H off && check "helper off" "[ \"\$(cat $F/sys/class/drm/card0-HDMI-A-1/status)\" = off ]"

# ---------- 7. verify.sh 讀出的解析度 ----------
cp /tmp/e2560.bin "$F/sys/class/drm/card0-HDMI-A-1/edid"
printf '2560x1440\n1920x1080\n' > "$F/sys/class/drm/card0-HDMI-A-1/modes"
echo connected > "$F/sys/class/drm/card0-HDMI-A-1/status"
bash /t/linux/scripts/verify.sh > /tmp/verify.log 2>&1
rc=$?
cat /tmp/verify.log
check "verify passes (rc=$rc)" "[ $rc -eq 0 ] && grep -q 'Installed EDID: 2560x1440' /tmp/verify.log"

# ---------- 8. 移除 ----------
mkdir -p /root/.config/go-virtual-display && echo '{}' > /root/.config/go-virtual-display/profiles.json
touch "$FW.abc123"
bash /t/linux/uninstall.sh > /tmp/uninstall.log 2>&1
check "uninstall exit 0" "[ \$? -eq 0 ]"
check "firmware removed" "[ ! -e $FW ] && [ ! -e $FW.abc123 ]"
check "conf removed" "[ ! -e $F/etc/go-virtual-display.conf ]"
check "grub params removed, user's kept" "grep -q 'GRUB_CMDLINE_LINUX_DEFAULT=\"quiet splash video=DP-1:1280x720\"' $F/etc/default/grub"
check "hook removed" "[ ! -e $F/etc/initramfs-tools/hooks/go-virtual-display ]"
check "helper dir removed" "[ ! -e $F/usr/local/libexec/go-virtual-display ]"
check "connector reset to detect" "[ \"\$(cat $F/sys/class/drm/card0-HDMI-A-1/status)\" = detect ]"
grep 'GRUB_CMDLINE_LINUX_DEFAULT' $F/etc/default/grub

# ---------- 9. 安裝包（扁平配置）找得到檔案 ----------
rm -rf /p && mkdir -p /p/scripts
cp /t/linux/install.sh /t/linux/uninstall.sh /p/
cp /t/linux/scripts/* /p/scripts/
cp /t/edid/edid_1920x1080.bin /p/scripts/
echo 'CONNECTOR=card0-HDMI-A-1' > "$F/etc/go-virtual-display.conf"
echo disconnected > "$F/sys/class/drm/card0-HDMI-A-1/status"
bash /p/install.sh --lang zh-TW > /tmp/install3.log 2>&1
check "packaged install exit 0" "[ \$? -eq 0 ]"
check "packaged: default 1080 EDID installed" "cmp -s $FW /t/edid/edid_1920x1080.bin"
check "packaged: zh-TW messages" "grep -q '安裝完成' /tmp/install3.log"
check "packaged: no switch → skipped message" "grep -q '略過開關程式' /tmp/install3.log"

echo
echo "RESULT: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ]
