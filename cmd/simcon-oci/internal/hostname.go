package container

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func setHostname(spec *specs.Spec) error {
	err := unix.Sethostname([]byte(spec.Hostname))
	if err != nil {
		return err
	}
	return nil
}
