package container

import (
	"path/filepath"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
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

//TODO : Process resources
func resources(spec specs.Spec) error {
	cgroupPath := spec.Linux.CgroupsPath
	// set device
	devicesfile := filepath.Join(cgroupPath, "cgroup.devices.allow")
	for _, resource := range spec.Linux.Resources.Devices {
		if !resource.Allow {
			continue
		}
		if resource.
	}
}
