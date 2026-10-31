//go:build !windows && !linux

package display

import "errors"

func newController() (Controller, error) {
	return nil, errors.New("這個作業系統還不支援")
}
