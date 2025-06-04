package cgroups

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// CgroupManager handles cgroup operations
type CgroupManager struct {
	Path string
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager(containerID string) *CgroupManager {
	return &CgroupManager{
		Path: filepath.Join("/sys/fs/cgroup", containerID),
	}
}

// Create creates a new cgroup
func (m *CgroupManager) Create() error {
	if err := os.MkdirAll(m.Path, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %v", err)
	}
	return nil
}

// SetMemoryLimit sets the memory limit for the cgroup in bytes
func (m *CgroupManager) SetMemoryLimit(limitInBytes int64) error {
	path := filepath.Join(m.Path, "memory.limit_in_bytes")
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", limitInBytes)), 0644)
}

// SetCPULimit sets the CPU shares for the cgroup (relative weight)
func (m *CgroupManager) SetCPULimit(shares int) error {
	path := filepath.Join(m.Path, "cpu.shares")
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", shares)), 0644)
}

// SetPidsLimit sets the maximum number of processes allowed in the cgroup
func (m *CgroupManager) SetPidsLimit(maxPids int) error {
	path := filepath.Join(m.Path, "pids.max")
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", maxPids)), 0644)
}

// SetBlockIO sets the block IO weight for the cgroup (10-1000)
func (m *CgroupManager) SetBlockIO(weight int) error {
	if weight < 10 || weight > 1000 {
		return fmt.Errorf("block IO weight must be between 10 and 1000")
	}
	path := filepath.Join(m.Path, "blkio.weight")
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", weight)), 0644)
}

// SetNetwork sets the network class ID for the cgroup
func (m *CgroupManager) SetNetwork(classID uint32) error {
	path := filepath.Join(m.Path, "net_cls.classid")
	return os.WriteFile(path, []byte(fmt.Sprintf("0x%x", classID)), 0644)
}

// SetDevices sets the device access permissions for the cgroup
func (m *CgroupManager) SetDevices(devices []specs.LinuxDeviceCgroup) error {
	for _, device := range devices {
		rule := fmt.Sprintf("%s %d:%d %s", device.Type, device.Major, device.Minor, device.Access)
		path := filepath.Join(m.Path, "devices.allow")
		if err := os.WriteFile(path, []byte(rule), 0644); err != nil {
			return fmt.Errorf("failed to set device rule %s: %v", rule, err)
		}
	}
	return nil
}

// SetHugepages sets the hugepages limit for the cgroup
func (m *CgroupManager) SetHugepages(limits []specs.LinuxHugepageLimit) error {
	for _, limit := range limits {
		path := filepath.Join(m.Path, fmt.Sprintf("hugetlb.%s.limit_in_bytes", limit.Pagesize))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("%d", limit.Limit)), 0644); err != nil {
			return fmt.Errorf("failed to set hugepage limit for %s: %v", limit.Pagesize, err)
		}
	}
	return nil
}

// SetRdma sets the RDMA device limits for the cgroup
func (m *CgroupManager) SetRdma(rdma map[string]specs.LinuxRdma) error {
	for device, limit := range rdma {
		path := filepath.Join(m.Path, fmt.Sprintf("rdma.%s.hca_handles", device))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("%d", limit.HcaHandles)), 0644); err != nil {
			return fmt.Errorf("failed to set RDMA limit for %s: %v", device, err)
		}
	}
	return nil
}

// SetUnified sets the unified cgroup limits
func (m *CgroupManager) SetUnified(unified map[string]string) error {
	for key, value := range unified {
		path := filepath.Join(m.Path, key)
		if err := os.WriteFile(path, []byte(value), 0644); err != nil {
			return fmt.Errorf("failed to set unified limit %s: %v", key, err)
		}
	}
	return nil
}

// AddProcess adds a process to the cgroup
func (m *CgroupManager) AddProcess(pid int) error {
	path := filepath.Join(m.Path, "tasks")
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// Remove removes the cgroup
func (m *CgroupManager) Remove() error {
	return os.RemoveAll(m.Path)
}
