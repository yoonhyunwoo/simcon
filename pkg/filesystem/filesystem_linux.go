package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// FileSystem handles container filesystem operations
type FileSystem struct {
	RootPath string
}

// NewFileSystem creates a new filesystem manager
func NewFileSystem(containerID string) *FileSystem {
	return &FileSystem{
		RootPath: filepath.Join("/var/lib/simcon", containerID),
	}
}

// CreateRootFS creates the root filesystem for the container
func (fs *FileSystem) CreateRootFS() error {
	if err := os.MkdirAll(fs.RootPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs: %v", err)
	}

	// Create basic directory structure
	dirs := []string{
		"bin",
		"dev",
		"etc",
		"proc",
		"sys",
		"usr",
		"var",
	}

	for _, dir := range dirs {
		path := filepath.Join(fs.RootPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}

// MountProc mounts the proc filesystem
func (fs *FileSystem) MountProc() error {
	procPath := filepath.Join(fs.RootPath, "proc")
	return syscall.Mount("proc", procPath, "proc", 0, "")
}

// MountSys mounts the sys filesystem
func (fs *FileSystem) MountSys() error {
	sysPath := filepath.Join(fs.RootPath, "sys")
	return syscall.Mount("sysfs", sysPath, "sysfs", 0, "")
}

// Cleanup removes the container filesystem
func (fs *FileSystem) Cleanup() error {
	return os.RemoveAll(fs.RootPath)
}
