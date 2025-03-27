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

func setResources(spec *specs.Spec) error {
	if spec.Linux == nil || spec.Linux.Resources == nil {
		return nil
	}

	cgroupPath := spec.Linux.CgroupsPath
	res := spec.Linux.Resources

	// Memory
	if res.Memory != nil {
		if res.Memory.Limit != nil {
			err := writeFile(filepath.Join(cgroupPath, "memory.max"), fmt.Sprintf("%d", *res.Memory.Limit))
			if err != nil {
				return err
			}
		}
		if res.Memory.Swap != nil {
			err := writeFile(filepath.Join(cgroupPath, "memory.swap.max"), fmt.Sprintf("%d", *res.Memory.Swap))
			if err != nil {
				return err
			}
		}
		if res.Memory.Swappiness != nil {
			err := writeFile(filepath.Join(cgroupPath, "memory.swappiness"), fmt.Sprintf("%d", *res.Memory.Swappiness))
			if err != nil {
				return err
			}
		}
	}

	// CPU
	if res.CPU != nil {
		if res.CPU.Quota != nil && res.CPU.Period != nil {
			value := fmt.Sprintf("%d %d", *res.CPU.Quota, *res.CPU.Period)
			err := writeFile(filepath.Join(cgroupPath, "cpu.max"), value)
			if err != nil {
				return err
			}
		}
		if res.CPU.Shares != nil {
			err := writeFile(filepath.Join(cgroupPath, "cpu.weight"), fmt.Sprintf("%d", *res.CPU.Shares))
			if err != nil {
				return err
			}
		}
	}

	// PIDs
	if res.Pids != nil {
		err := writeFile(filepath.Join(cgroupPath, "pids.max"), fmt.Sprintf("%d", res.Pids.Limit))
		if err != nil {
			return err
		}
	}

	// Block IO
	if res.BlockIO != nil {
		if res.BlockIO.Weight != nil {
			err := writeFile(filepath.Join(cgroupPath, "io.weight"), fmt.Sprintf("%d", *res.BlockIO.Weight))
			if err != nil {
				return err
			}
		}
		// 생략: device 별 블록 IO 제한 처리 (추가 구현 가능)
	}

	// Hugepages
	for _, hp := range res.HugepageLimits {
		file := fmt.Sprintf("hugetlb.%s.max", hp.Pagesize)
		err := writeFile(filepath.Join(cgroupPath, file), fmt.Sprintf("%d", hp.Limit))
		if err != nil {
			return err
		}
	}

	// Network
	if res.Network != nil {
		if res.Network.ClassID != nil {
			err := writeFile(filepath.Join(cgroupPath, "net_cls.classid"), fmt.Sprintf("%d", *res.Network.ClassID))
			if err != nil {
				return err
			}
		}
		for _, prio := range res.Network.Priorities {
			line := fmt.Sprintf("%s %d", prio.Name, prio.Priority)
			err := writeFile(filepath.Join(cgroupPath, "net_prio.ifpriomap"), line)
			if err != nil {
				return err
			}
		}
	}

	// RDMA
	if res.Rdma != nil {
		for dev, limit := range res.Rdma {
			if limit.HcaHandles != nil {
				err := writeFile(filepath.Join(cgroupPath, "rdma."+dev+".hca_handle.max"), fmt.Sprintf("%d", *limit.HcaHandles))
				if err != nil {
					return err
				}
			}
			if limit.HcaObjects != nil {
				err := writeFile(filepath.Join(cgroupPath, "rdma."+dev+".hca_object.max"), fmt.Sprintf("%d", *limit.HcaObjects))
				if err != nil {
					return err
				}
			}
		}
	}

	// Unified
	for key, value := range res.Unified {
		err := writeFile(filepath.Join(cgroupPath, key), value)
		if err != nil {
			return err
		}
	}

	// Devices
	devPath := filepath.Join(cgroupPath, "cgroup.devices.allow")
	f, err := os.OpenFile(devPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, dev := range res.Devices {
		if !dev.Allow {
			continue
		}
		row := fmt.Sprintf("%s %d:%d %s\n", dev.Type, *dev.Major, *dev.Minor, dev.Access)
		_, err := f.WriteString(row)
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

// writeFile 유틸 함수
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
