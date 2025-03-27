package runtime

import (
	"golang.org/x/sys/unix"
)

func setHostname(hostname string) error {
	err := unix.Sethostname([]byte(hostname))
	if err != nil {
		return err
	}
	return nil
}
