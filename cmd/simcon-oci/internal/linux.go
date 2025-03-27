package container

import (
	"fmt"
	"os"
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

// TODO : Process resources
func resources(spec *specs.Spec) error {
	cgroupPath := spec.Linux.CgroupsPath
	// set device
	devicefile := filepath.Join(cgroupPath, "cgroup.devices.allow")
	devicesCgroup, err := os.Create(devicefile)
	if err != nil {
		return err
	}
	defer devicesCgroup.Close()

	for _, resource := range spec.Linux.Resources.Devices {
		if !resource.Allow {
			continue
		}
		row := fmt.Sprintf("%s %d:%d %s", resource.Type, resource.Major, resource.Minor, resource.Access)
		_, err = devicesCgroup.WriteString(row)
		if err != nil {
			return err
		}
	}

	return nil
}

func setNamespaces(namespaces []specs.LinuxNamespace) uintptr {
	var cloneFlags uintptr
	for _, namespace := range namespaces {
		switch namespace.Type {
		case specs.PIDNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWPID
		case specs.UTSNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWUTS
		case specs.IPCNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWIPC
		case specs.NetworkNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWNET
		case specs.MountNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWNS
		case specs.CgroupNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWCGROUP
		case specs.TimeNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWTIME
		case specs.UserNamespace:
			cloneFlags = cloneFlags | syscall.CLONE_NEWUSER
		}
	}
	return cloneFlags
}
