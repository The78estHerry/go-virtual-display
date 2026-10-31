//go:build !windows && !linux

package display

import (
	"errors"

	"gitea.littleherry.work/The78estWorld/go-virtual-display/internal/i18n"
)

func newController() (Controller, error) {
	return nil, errors.New(i18n.T("This operating system is not supported yet", "這個作業系統還不支援"))
}
