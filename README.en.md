<p align="center"><img src="assets/icon-256.png" width="160" alt="go-virtual-display icon: a dream bubble thinking of a rainbow monitor"></p>

# go-virtual-display

[繁體中文](README.md) | **English** | [日本語](README.ja.md)

> The Traditional Chinese README is the primary version. If this translation differs from it, the Chinese version is correct.

Makes your computer believe a monitor is connected even when **no physical monitor is attached**, and that monitor reports an EDID we generate ourselves. You can pick, add and remove resolutions; the default is 1920×1080. It also comes with a "plug switch": a tray icon plus a desktop widget that plug in or unplug this virtual monitor with one click.

Useful for remote desktop, streaming or recording on headless machines, and for tests that need a specific resolution.

Supports **Windows 10/11** and **Linux** (needs a free connector on the graphics card). macOS is not supported.

The Windows version supports **Synergy 3** (keyboard and mouse sharing software): it restarts Synergy automatically after the display layout changes, and shows in the tray and the widget whether Synergy is running (see [Using with Synergy 3](#using-with-synergy-3)).

## How it works

| | Windows | Linux |
|---|---|---|
| Virtual monitor source | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver) (signed open-source IddCx driver, MIT) | The kernel's built-in DRM EDID override, which makes a free connector look connected |
| Custom EDID | The driver reads `C:\VirtualDisplayDriver\user_edid.bin` | Boot parameter `drm.edid_firmware=` points to `/lib/firmware/edid/go-virtual-display.bin`; switching resolution writes debugfs `edid_override` to apply it immediately |
| Plug / unplug | Disable/enable the driver device (`pnputil`), run by a "highest privileges" scheduled task, so the switch never shows UAC | Write `on`/`off` to `/sys/class/drm/<connector>/status`, allowed without a password for the active user via polkit |

The driver itself is **not stored in this repo**.
- **The Windows release package** includes the driver files of a pinned version (25.7.23) and verifies their SHA-256 at install time, so **no internet connection is needed**, and it still installs even if the original download disappears.
  - Only the needed files are included: the amd64/ARM64 drivers and `devcon.exe`, identical to the official zip, so their signatures stay valid.
  - The license notices are in `scripts\vdd\THIRD-PARTY-NOTICES.txt` in the package (see [Third-party software](#third-party-software)).
- **When installing from source**, the official zip is downloaded and its SHA-256 verified.
- The SHA-256 of every driver file is written in `windows\scripts\common.ps1`, so tampering with the files in the package is detected.

## EDID contents

One EDID is generated per resolution (`internal/profile`):

- EDID 1.4, 128 bytes, no extension blocks
- The preferred timing uses VESA CVT reduced blanking, the timing modern LCDs commonly use. 1920×1440@60 is the exception and keeps VESA DMT, as in older versions
- Name `VirtualDisp`, serial string `GVD-<width>x<height>` (e.g. `GVD-1920x1080`), sRGB
- Physical size is computed at 96 DPI (1920×1080 → 51×29 cm), so Windows defaults to 100% scaling
- Virtual Display Driver replaces the vendor/product codes with `MTT`/`0x1337`, so resolutions are told apart by the serial string. Windows remembers the arrangement and scaling of each resolution separately

The repo contains two prebuilt EDIDs:
- `edid/edid_1920x1080.bin`: the default at install time (Windows, Linux)
- `edid/edid_1920x1440.bin`: used by the Linux version before 0.3.0, kept as a DMT timing reference

```
go run ./cmd/make-edid -w 2560 -h 1440 -hz 60 -o out.bin   # generate a given resolution
go run ./cmd/make-edid -check edid/edid_1920x1080.bin      # check and list the contents
```

## Resolutions

- **Switch**: right-click the tray icon → "Resolution", then click the one you want. The virtual monitor restarts briefly; on Windows, Synergy is restarted too if installed. Switching while unplugged takes effect at the next plug-in.
- **Manage**: "Manage resolutions…" in the same submenu opens a window:
  - Saved resolutions can be "Use"d or "Delete"d; the active one cannot be deleted
  - Common resolutions can be added with one click (all 60 Hz): 1920×1080, 2560×1440, 3840×2160, 1280×720, 1920×1200, 2560×1600, 3840×2400, 640×480, 1024×768, 1600×1200, 1920×1440, 1280×1024, 2560×1080, 1080×1920
  - You can also type width, height and refresh rate. Width/height 320–4095, refresh rate 24–240 Hz; an error is shown when the value is too high to describe in an EDID
- A fresh install has only 1920×1080@60. The list is stored in `%APPDATA%\go-virtual-display\profiles.json` (Linux: `~/.config/go-virtual-display/profiles.json`). It is kept when installing over an existing install (upgrade or reinstall) and deleted on uninstall.
- **Windows**: switching runs `vdswitch.exe --apply` through the highest-privileges scheduled task `\go-virtual-display\Apply`. It generates the EDID and rewrites the settings in `C:\VirtualDisplayDriver`, so no UAC prompt appears.
- **Linux**: the switch program generates the EDID and hands it to a helper through polkit (`vdswitch-helper edid`, no password). The helper checks the length, header and checksum, replaces `/lib/firmware/edid/go-virtual-display.bin`, and rebuilds the initramfs in the background. The boot parameter always points to this file name, so GRUB does not need to change.
  - With debugfs (`/sys/kernel/debug/dri/<card>/<connector>/edid_override`) it applies immediately
  - Without it (e.g. Secure Boot lockdown), the widget shows "Reboot to apply"

## Windows

### Requirements

- Windows 10/11 x64 (there is also an ARM64 driver, but Windows 11 24H2 and later may need test-signing mode)
- [Microsoft Visual C++ Redistributable](https://learn.microsoft.com/cpp/windows/latest-supported-vc-redist) (without it the driver reports `vcruntime140.dll not found`)
- Without Go, download `go-virtual-display-<version>-windows.zip` from the Gitea Releases; it includes the prebuilt switch program. Building it yourself needs Go 1.26 or later

### One-click install

Double-click **`windows\install.cmd`** and click "Yes" at the UAC prompt. Everything else is automatic:

1. Choose the interface language (English / Traditional Chinese); press Enter for the default (see "Interface language" below)
2. Get the driver and verify its SHA-256: the package uses the bundled files; only a source install downloads. If this step fails, installation stops without changing anything (when reinstalling, the old version is not removed either)
3. Put `vdd_settings.xml` and the EDID into `C:\VirtualDisplayDriver` (existing files are backed up first). The EDID is generated from your saved resolution; a fresh install uses 1920×1080
4. Install the driver
5. Create scheduled tasks (`\go-virtual-display\Plug`, `Unplug`, `Apply`, `RestartServices`) so the switch and resolution changes never show UAC again. If Synergy is installed, it is restarted automatically after the display layout changes (see below)
6. Install the switch program to `C:\Program Files\go-virtual-display` and create "Virtual Display Switch" shortcuts in the Start menu and on the desktop (to pin it to the taskbar, right-click the shortcut and choose "Pin to taskbar"). If it is not built yet but Go is installed, it is built first
7. Copy the uninstall files and register in "Settings > Apps"
8. Run the checks and open the switch program

**If it is already installed** (upgrade or reinstall), the installer shows the current and the new version (with a warning when downgrading) and asks whether to continue:
- Answer `y`: uninstall the current version, then install fresh. Your added resolutions, chosen language and "Start at sign-in" are kept
- Press Enter or answer anything else: cancel without changing anything

### Uninstall

Use any one of these; each shows a UAC prompt, click "Yes":
- Right-click the tray icon and choose "**Uninstall…**"
- In "**Settings > Apps**", find "Virtual Display Switch (go-virtual-display)" (in the Chinese interface: 「虛擬螢幕開關（go-virtual-display）」) and uninstall it
- Double-click `windows\uninstall.cmd`

The installer copies the uninstall files to `C:\Program Files\go-virtual-display\uninstaller`, so the first two methods still work even if you delete the original install folder. Uninstalling:
- Removes the driver, scheduled tasks, the switch program and the start-at-sign-in setting
- Removes the virtual monitor device records, scaling settings and VDD registry values left by Windows
- Restores the original VDD settings file. If there was none before installing, `C:\VirtualDisplayDriver` is deleted entirely
- Deletes your settings: added resolutions, widget position and chosen language (`%APPDATA%\go-virtual-display`, `HKLM\SOFTWARE\go-virtual-display`). To upgrade, just run the new `install.cmd`; there is no need to uninstall first

### Running manually (advanced)

The one-click install runs the scripts in `windows\scripts\`. You can also run them one by one in an administrator PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1                # build (needs Go)
powershell -ExecutionPolicy Bypass -File .\windows\scripts\install.ps1      # install
powershell -ExecutionPolicy Bypass -File .\windows\scripts\verify.ps1       # check (no administrator needed)
powershell -ExecutionPolicy Bypass -File .\windows\scripts\uninstall.ps1    # uninstall
```

### Using with Synergy 3

Tested on a Synergy 3 client (service name `Synergy Core Daemon`).

- **Automatic restart**: a Synergy client reads the size of the whole desktop only when it starts. After the virtual monitor is plugged in or unplugged, it keeps calculating the mouse position with the old size, and a mouse coming over from the server lands on the wrong monitor. The same happens after rearranging monitors in "Settings > Display": the mouse gets stuck at the old edge.
  - When the switch program detects a display layout change (plug/unplug, resolution switch, rearranging, connecting another monitor), it waits about 2 seconds for things to settle and then restarts Synergy through the scheduled task `RestartServices`. Synergy disconnects briefly and reconnects on its own.
  - This task is always created and checks whether Synergy exists when it runs, so installing or removing Synergy later does not require reinstalling this program.
  - Nothing happens while the switch program is not running.
- **Status display**: when Synergy is installed, the widget gets an extra line under the status, "Synergy 3 Running" / "Synergy 3 Not running", and the tray menu shows "Synergy 3: Running" / "Synergy 3: Not running" (the number is Synergy's major version, read from `Synergy.exe`; just "Synergy" if it cannot be read), both with the Synergy icon. The icon is in color while running and gray when not running. Nothing is shown when Synergy is not installed.
  - "Running" means both the Synergy service and `synergy-core.exe`, which actually shares the keyboard and mouse, are running. When you quit Synergy from its own tray icon, the service stays but `synergy-core.exe` stops, so it shows "Not running".
  - While Synergy is not running, display layout changes do not restart it, so a Synergy you closed is not started again.
  - The icon is read at run time from the Synergy installed on your computer. This project does not ship any Synergy image files.
- Other services that also read the desktop size only at startup can be added to `$RestartServices` in `windows\scripts\common.ps1`.

This project is not affiliated with, endorsed by or sponsored by Symless, the developer of Synergy (see [Trademarks](#trademarks)).

> ⚠️ Before updating graphics or chipset drivers, uninstalling the virtual display driver first is recommended (VDD's official advice). If the screen is black after booting, uninstall it in Safe Mode.

## Linux

### Requirements

- A connector on the graphics card with **no monitor attached** (e.g. HDMI-A-1). Virtual machines or servers without a real graphics card are not supported.
- polkit (`pkexec`)
- The tray icon needs desktop support for StatusNotifierItem: KDE supports it directly, GNOME needs the AppIndicator extension
- Building it yourself needs Go and Gio's dependencies (see the top of the `Makefile`)
- Automatic boot-parameter setup supports GRUB; automatic initramfs setup supports initramfs-tools (Debian/Ubuntu) and dracut (Fedora etc.)

### One-click install

Run it in a terminal as a normal user (no sudo needed; it asks for your password when necessary):

```bash
linux/install.sh        # from source
./install.sh            # from the release package (after extracting go-virtual-display-<version>-linux-amd64.tar.gz)
```

Everything else is automatic:

1. If the switch program is not built yet but Go is installed, it is built first (source only; the package includes a prebuilt one)
2. A free external connector (HDMI, DP etc., never a laptop's built-in panel) is chosen automatically. To choose one yourself, use `linux/install.sh card0-HDMI-A-1`
3. The EDID is generated from your saved resolution (a fresh install uses 1920×1080). The EDID, the helper and the polkit rule are installed, so plugging and switching resolution never need a password
4. The EDID is added to the initramfs
5. The boot parameters are added to `/etc/default/grub` and the GRUB configuration is regenerated. The first time, a backup is saved as `grub.go-virtual-display.bak`
6. With debugfs it is applied once right away, no reboot needed. Otherwise reboot
7. The switch program opens

**If it is already installed** (upgrade or reinstall), it shows the current and the new version (with a warning when downgrading) and asks whether to continue:
- Answer `y`: uninstall the current version, then install fresh. Your added resolutions, chosen language, connector and "Start at sign-in" are all kept
- Press Enter or answer anything else: cancel without changing anything
- When there is no terminal to ask (e.g. run from another program), it installs over the existing one

It can be run repeatedly without adding parameters twice, and keeps your chosen resolution. When the connector changes, the old connector's parameters are removed. When upgrading from a version before 0.3.0, the old `edid_1920x1440.bin` and its boot parameter are replaced with the new file name.

### One-click uninstall

```bash
linux/uninstall.sh
```

This removes all installed files, boot parameters, initramfs settings, the start-at-sign-in setting and your saved resolutions (`~/.config/go-virtual-display`), and sets the connector back to auto-detection. Only the parameters this project added are removed; your own boot parameters are untouched. After a reboot everything is back to how it was.

Use `linux/scripts/verify.sh` to check.

## Icon

The program icon is "a dream bubble thinking of a monitor", with a rainbow and clouds on the screen. It is embedded in `vdswitch.exe` and appears in "Installed apps", the Start menu and File Explorer. The tray icon is still a monitor whose color follows the state: green plugged in, gray unplugged, orange switching, red error.

To change the icon, edit `assets/icon.svg` and run `assets/make-icon.ps1`. It uses the built-in Edge to render the SVG at each size, assembles `icon.ico`, and uses llvm-mingw's windres to produce `cmd/vdswitch/rsrc_windows_amd64.syso`. The outputs are committed, so normal builds do not need these tools.

## Interface language

The switch program and the install and uninstall messages are available in English and Traditional Chinese.

- **Default**: based on the system display language. Only Chinese (Traditional or Simplified) defaults to Traditional Chinese; everything else uses English.
- **Choose at install time**: Windows `install.cmd` and Linux `install.sh` ask at the start; press Enter for the default. When reinstalling, the default is your previous choice.
- **Where it is stored**: on Windows in `Language` under `HKLM\SOFTWARE\go-virtual-display`, on Linux in `LANGUAGE` in `/etc/go-virtual-display.conf` (`en` or `zh-TW`). It is deleted on uninstall.
- File names are always English (including the Start menu shortcut "Virtual Display Switch" and `README.txt` in the package) to avoid problems on non-Chinese systems.

## Plug switch (vdswitch)

- **Desktop widget**:
  - A small card with a slide switch on the right; click to toggle
  - Green means plugged in, gray unplugged, orange switching, red an error (the reason is shown on the card)
  - Drag the text area on the left to move the card
  - Windows remembers the card's position; on Linux the window manager decides
  - Windows: when Synergy is installed, an extra line under the status shows the Synergy icon and "Running" / "Not running", making the card a little taller (see [Using with Synergy 3](#using-with-synergy-3))
  - Windows: when the monitor the card is on gets unplugged (e.g. it is on the virtual monitor set as the main display), the card moves temporarily to the bottom-right of the main display, and returns to its place when that monitor is plugged back in. If you drag it in the meantime, it stays where you put it
- **Tray icon**:
  - Same colors as the widget
  - **Left-click to toggle**
  - Right-click menu: Synergy status (Windows only, when Synergy is installed), Plug in / Unplug, Resolution (Windows only), Show desktop widget, Keep widget on top, Start at sign-in, Uninstall… (Windows only), Exit
- The state is synced every 2 seconds. When it is changed elsewhere (Device Manager, VDD's control program), both follow.
- Only one instance runs. Opening it again brings up the widget.
- Settings are stored in `%APPDATA%\go-virtual-display\widget.json` (Linux: `~/.config/go-virtual-display/widget.json`).

## Project layout

```
cmd/make-edid/        generate / check EDIDs
cmd/vdswitch/         plug switch program (with single instance)
internal/edid/        EDID 1.4 assembly and parsing (with tests)
internal/i18n/        interface language detection and English / Traditional Chinese text
internal/profile/     resolution list (profiles.json) and the EDID of each resolution
internal/display/     per-platform status query and plug/unplug
internal/ui/          tray (fyne.io/systray), widget (Gio), settings, autostart
edid/                 prebuilt EDIDs
assets/               program icon: icon.svg (source), icon.ico, make-icon.ps1 (builds .ico and the .syso for cmd/vdswitch)
windows/              Windows one-click install/uninstall: just run install.cmd, uninstall.cmd
  scripts/            the PowerShell scripts and VDD settings behind them
linux/                Linux one-click install/uninstall: just run install.sh, uninstall.sh
  scripts/            checks, helper, polkit rule, initramfs hook, .desktop
build.ps1 / Makefile  build
release.ps1           build release packages (steps and contents in RELEASING.md)
```

## Alternative (not used)

You could also modify Microsoft's IddSampleDriver and hard-code the EDID in the driver. That gives full control over every byte, but you must build the driver yourself. Without a proper signature you need test-signing mode, which requires turning off Secure Boot and shows a watermark on screen. Virtual Display Driver is signed and supports custom EDIDs, so it was chosen instead.

## License

This project is released under the [MIT License](LICENSE): you may use, modify and distribute it freely, without any warranty. The bundled third-party software is under its own licenses (see [Third-party software](#third-party-software)).

If this tool helps you, sponsorship to support development is welcome (the "Sponsor" button on the GitHub page, once it is available).

## Trademarks

- Synergy is a trademark of Symless Ltd., and the Synergy icon (logo) belongs to Symless Ltd. This project mentions the name only for compatibility and is not affiliated with, endorsed by or sponsored by Symless.
- This project does not include or distribute the Synergy icon. The Synergy icon shown by the switch program is read at run time from the Synergy program already installed on the user's own computer.
- Windows is a trademark of Microsoft Corporation. Other product names belong to their respective owners.

## Third-party software

The Windows release package includes the following files, unmodified, taken from Virtual Display Driver's official `VDD.Control.25.7.23.zip`:

| Files | Software | License |
|---|---|---|
| `scripts\vdd\SignedDrivers\*\VDD\*` | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver) | MIT |
| `scripts\vdd\Dependencies\devcon.exe` | DevCon (signed by Microsoft, [source](https://github.com/microsoft/Windows-driver-samples/tree/main/setup/devcon)) | MS-PL |

The full license texts are in `windows\scripts\THIRD-PARTY-NOTICES.txt` (`scripts\vdd\THIRD-PARTY-NOTICES.txt` in the package). This project is not affiliated with the authors of this software.
