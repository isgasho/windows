//+build !windows

package dns

import "errors"

func Enabled() (bool, error) {
	return false, errors.New("not implemented")
}

func Enable() error {
	return errors.New("not implemented")
}

func Disable() error {
	return errors.New("not implemented")
}
