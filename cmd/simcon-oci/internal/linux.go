package container

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func maskPath(path string) error {
	return unix.Mount("/dev/null", path, "", syscall.MS_BIND, "")
}

func readonlyPath(path string) error {
	if err := syscall.Mount(path, path, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}
	if err := syscall.Mount("", path, "", syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_BIND, ""); err != nil {
		return err
	}
	return nil
}
