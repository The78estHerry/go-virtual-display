//go:build !windows && !linux

package display

import (
	"errors"

	"github.com/The78estHerry/go-virtual-display/internal/i18n"
)

func newController() (Controller, error) {
	return nil, errors.New(i18n.T("this_operating_system_is_not_supported"))
}
