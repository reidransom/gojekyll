//go:build !windows

package site

import (
	"os"

	"golang.org/x/sys/unix"
)

func withRemoteThemeLock(path string, action func() error) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }()
	return action()
}
