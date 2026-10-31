# Linux 編譯。Gio 在 Linux 需要 cgo 與以下套件（Debian/Ubuntu）：
#   sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev \
#        libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev
.PHONY: linux edid test

linux: test
	mkdir -p dist/linux
	go build -trimpath -ldflags '-s -w' -o dist/linux/vdswitch ./cmd/vdswitch
	go build -trimpath -ldflags '-s -w' -o dist/linux/make-edid ./cmd/make-edid

edid:
	go run ./cmd/make-edid -o edid/edid_1920x1440.bin

test:
	go test ./...
