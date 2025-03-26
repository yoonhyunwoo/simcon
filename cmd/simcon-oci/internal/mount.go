package container

import (
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func Mount(spec *specs.Spec) error {
	for _, mount := range spec.Mounts {
		options, data := parseMountOptions(mount.Options)
		err := unix.Mount(mount.Source, mount.Destination, mount.Type, options, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseMountOptions(options []string) (uintptr, string) {
	var flags uintptr
	var dataList []string

	for _, opt := range options {
		switch opt {
		case "ro":
			flags |= unix.MS_RDONLY
		case "nosuid":
			flags |= unix.MS_NOSUID
		case "noexec":
			flags |= unix.MS_NOEXEC
		case "nodev":
			flags |= unix.MS_NODEV
		case "relatime":
			flags |= unix.MS_RELATIME
		case "noatime":
			flags |= unix.MS_NOATIME
		case "bind":
			flags |= unix.MS_BIND
		case "rbind":
			flags |= unix.MS_BIND | unix.MS_REC
		case "remount":
			flags |= unix.MS_REMOUNT
		case "shared":
			flags |= unix.MS_SHARED
		case "private":
			flags |= unix.MS_PRIVATE
		case "slave":
			flags |= unix.MS_SLAVE
		case "unbindable":
			flags |= unix.MS_UNBINDABLE
		case "sync":
			flags |= unix.MS_SYNCHRONOUS
		case "dirsync":
			flags |= unix.MS_DIRSYNC
		case "mand":
			flags |= unix.MS_MANDLOCK
		case "lazytime":
			flags |= unix.MS_LAZYTIME

		default:
			dataList = append(dataList, opt)
		}
	}

	data := strings.Join(dataList, ",")
	return flags, data
}
