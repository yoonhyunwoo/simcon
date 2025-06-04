package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// InitProcess represents the container's init process
type InitProcess struct {
	Container *Container
	cmd       *exec.Cmd
}

// NewInitProcess creates a new init process
func NewInitProcess(container *Container) *InitProcess {
	return &InitProcess{
		Container: container,
	}
}

// Start starts the init process
func (p *InitProcess) Start() error {
	p.cmd = exec.Command("/proc/self/exe", "init")

	// Set up namespace flags based on the container spec
	var cloneFlags uintptr
	if p.Container.Spec.Linux != nil && p.Container.Spec.Linux.Namespaces != nil {
		for _, ns := range p.Container.Spec.Linux.Namespaces {
			switch ns.Type {
			case "pid":
				cloneFlags |= unix.CLONE_NEWPID
			case "network":
				cloneFlags |= unix.CLONE_NEWNET
			case "mount":
				cloneFlags |= unix.CLONE_NEWNS
			case "uts":
				cloneFlags |= unix.CLONE_NEWUTS
			case "ipc":
				cloneFlags |= unix.CLONE_NEWIPC
			case "user":
				cloneFlags |= unix.CLONE_NEWUSER
			}
		}
	}

	p.cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: cloneFlags,
	}
	p.cmd.Env = append(os.Environ(), fmt.Sprintf("_SIMCON_BUNDLE=%s", p.Container.Bundle))

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start init process: %v", err)
	}

	p.Container.Process.ID = p.cmd.Process.Pid

	return nil
}

// Wait waits for the init process to complete
func (p *InitProcess) Wait() error {
	return p.cmd.Wait()
}

// SetupMounts sets up the container mounts
func (p *InitProcess) SetupMounts() error {
	// First, mount proc
	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount proc: %v", err)
	}

	// Then mount other filesystems
	if p.Container.Spec.Mounts != nil {
		for _, mount := range p.Container.Spec.Mounts {
			if err := unix.Mount(mount.Source, mount.Destination, mount.Type, 0, ""); err != nil {
				return fmt.Errorf("failed to mount %s: %v", mount.Destination, err)
			}
		}
	}
	return nil
}

// SetupSecurity sets up the container security configurations
func (p *InitProcess) SetupSecurity() error {
	if p.Container.Spec.Process == nil {
		return nil
	}

	// Setup capabilities
	if p.Container.Spec.Process.Capabilities != nil {
		if err := p.setupCapabilities(); err != nil {
			return fmt.Errorf("failed to setup capabilities: %v", err)
		}
	}

	// Setup seccomp
	if p.Container.Spec.Linux != nil && p.Container.Spec.Linux.Seccomp != nil {
		if err := p.setupSeccomp(); err != nil {
			return fmt.Errorf("failed to setup seccomp: %v", err)
		}
	}

	// Setup rlimits
	if p.Container.Spec.Process.Rlimits != nil {
		if err := p.setupRlimits(); err != nil {
			return fmt.Errorf("failed to setup rlimits: %v", err)
		}
	}

	return nil
}

func (p *InitProcess) setupCapabilities() error {

	return nil
}

func (p *InitProcess) setupSeccomp() error {
	// Implementation for setting up seccomp
	// This is a placeholder for the actual implementation
	return nil
}

func (p *InitProcess) setupRlimits() error {
	// Implementation for setting up rlimits
	// This is a placeholder for the actual implementation
	return nil
}

func (p *InitProcess) setupHostname() error {
	if err := unix.Sethostname([]byte(p.Container.Spec.Hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %v", err)
	}
	return nil
}

func (p *InitProcess) Chroot() error {
	if err := unix.Chroot("/"); err != nil {
		return fmt.Errorf("failed to chroot: %v", err)
	}
	return nil
}

// CreateProcess creates a new process but doesn't start it
func (p *InitProcess) CreateProcess() error {
	if p.Container.Spec.Process == nil {
		return fmt.Errorf("no process specified in container spec")
	}

	if err := p.Chroot(); err != nil {
		return fmt.Errorf("failed to chroot: %v", err)
	}

	if err := p.setupHostname(); err != nil {
		return fmt.Errorf("failed to setup hostname: %v", err)
	}

	cmd := exec.Command(p.Container.Spec.Process.Args[0], p.Container.Spec.Process.Args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %v", err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Printf("process exited with error: %v", err)
		}
	}()

	return nil
}

// StartProcess starts the created process
func (p *InitProcess) StartProcess() error {
	if err := syscall.Kill(p.Container.Process.ID, syscall.SIGCONT); err != nil {
		return fmt.Errorf("failed to send SIGCONT: %v", err)
	}

	return nil
}
