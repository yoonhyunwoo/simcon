package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/yoonhyunwoo/simcon/pkg/cgroups"
	"golang.org/x/sys/unix"
)

// Container represents an OCI container
type Container struct {
	ID          string
	Bundle      string
	Process     *Process
	State       *ContainerState
	Spec        *specs.Spec
	InitProcess *InitProcess
}

// Process represents a container process
type Process struct {
	ID           int
	Args         []string
	Env          []string
	User         *User
	Capabilities *Capabilities
}

// User represents process user and group
type User struct {
	UID            uint32
	GID            uint32
	AdditionalGids []uint32
}

// Capabilities represents process capabilities
type Capabilities struct {
	Bounding    []string
	Effective   []string
	Inheritable []string
	Permitted   []string
	Ambient     []string
}

// State represents container state
type State struct {
	Version     string
	ID          string
	Status      string
	PID         int
	Bundle      string
	Annotations map[string]string
}

// Mount represents a filesystem mount
type Mount struct {
	Source      string
	Destination string
	Type        string
	Options     []string
}

// Hook represents a container lifecycle hook
type Hook struct {
	Path    string
	Args    []string
	Env     []string
	Timeout int
}

// ContainerManager handles container operations
type ContainerManager interface {
	Create(id, bundle string) (*Container, error)
	Start(id string) error
	Kill(id string, signal int) error
	Delete(id string) error
	State(id string) (*State, error)
}

// MountManager handles mount operations
type MountManager interface {
	Mount(mounts []Mount) error
	Unmount(mounts []Mount) error
}

// HookManager handles hook operations
type HookManager interface {
	ExecuteHooks(hooks []Hook) error
}

// CgroupManager handles cgroup operations
type CgroupManager interface {
	CreateCgroup(id string) error
	SetResources(id string, resources *Resources) error
	AddProcess(id string, pid int) error
	DeleteCgroup(id string) error
}

// Resources represents cgroup resources
type Resources struct {
	CPU     *CPU
	Memory  *Memory
	Pids    *Pids
	BlockIO *BlockIO
}

// CPU represents CPU resource limits
type CPU struct {
	Shares          uint64
	Quota           int64
	Period          uint64
	RealtimeRuntime int64
	RealtimePeriod  uint64
	Cpus            string
	Mems            string
}

// Memory represents memory resource limits
type Memory struct {
	Limit            int64
	Reservation      int64
	Swap             int64
	Kernel           int64
	KernelTCP        int64
	Swappiness       uint64
	DisableOOMKiller bool
}

// Pids represents process ID limits
type Pids struct {
	Limit int64
}

// BlockIO represents block IO resource limits
type BlockIO struct {
	Weight            uint16
	LeafWeight        uint16
	WeightDevice      []WeightDevice
	ThrottleReadBps   []ThrottleDevice
	ThrottleWriteBps  []ThrottleDevice
	ThrottleReadIOPS  []ThrottleDevice
	ThrottleWriteIOPS []ThrottleDevice
}

// WeightDevice represents a block IO weight device
type WeightDevice struct {
	Major      int64
	Minor      int64
	Weight     uint16
	LeafWeight uint16
}

// ThrottleDevice represents a block IO throttle device
type ThrottleDevice struct {
	Major int64
	Minor int64
	Rate  uint64
}

// NewContainer creates a new container instance from an OCI bundle
func NewContainer(id, bundle string) (*Container, error) {
	spec, err := loadSpec(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %v", err)
	}

	stateManager := NewStateManager()
	state, err := stateManager.CreateState(id, bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %v", err)
	}

	container := &Container{
		ID:     id,
		Bundle: bundle,
		Process: &Process{
			ID:   -1,
			Args: spec.Process.Args,
			Env:  spec.Process.Env,
			User: &User{
				UID:            spec.Process.User.UID,
				GID:            spec.Process.User.GID,
				AdditionalGids: spec.Process.User.AdditionalGids,
			},
			Capabilities: &Capabilities{
				Bounding:    spec.Process.Capabilities.Bounding,
				Effective:   spec.Process.Capabilities.Effective,
				Inheritable: spec.Process.Capabilities.Inheritable,
				Permitted:   spec.Process.Capabilities.Permitted,
				Ambient:     spec.Process.Capabilities.Ambient,
			},
		},
		State: state,
		Spec:  spec,
	}

	container.InitProcess = NewInitProcess(container)
	return container, nil
}

// Create creates a new container instance
func (c *Container) Create() error {
	stateManager := NewStateManager()

	// Create cgroup
	cgroupManager := cgroups.NewCgroupManager(c.ID)
	err := cgroupManager.Create()

	if c.Spec.Linux.Resources.Memory != nil && c.Spec.Linux.Resources.Memory.Limit != nil {
		cgroupManager.SetMemoryLimit(*c.Spec.Linux.Resources.Memory.Limit)
	}
	if c.Spec.Linux.Resources.CPU != nil && c.Spec.Linux.Resources.CPU.Shares != nil {
		cgroupManager.SetCPULimit(int(*c.Spec.Linux.Resources.CPU.Shares))
	}
	if c.Spec.Linux.Resources.Pids != nil {
		cgroupManager.SetPidsLimit(int(c.Spec.Linux.Resources.Pids.Limit))
	}
	if c.Spec.Linux.Resources.BlockIO != nil && c.Spec.Linux.Resources.BlockIO.Weight != nil {
		cgroupManager.SetBlockIO(int(*c.Spec.Linux.Resources.BlockIO.Weight))
	}
	if c.Spec.Linux.Resources.Network != nil && c.Spec.Linux.Resources.Network.ClassID != nil {
		cgroupManager.SetNetwork(uint32(*c.Spec.Linux.Resources.Network.ClassID))
	}
	if c.Spec.Linux.Resources.Devices != nil {
		cgroupManager.SetDevices(c.Spec.Linux.Resources.Devices)
	}
	if c.Spec.Linux.Resources.HugepageLimits != nil {
		cgroupManager.SetHugepages(c.Spec.Linux.Resources.HugepageLimits)
	}
	if c.Spec.Linux.Resources.Rdma != nil {
		cgroupManager.SetRdma(c.Spec.Linux.Resources.Rdma)
	}
	if c.Spec.Linux.Resources.Unified != nil {
		cgroupManager.SetUnified(c.Spec.Linux.Resources.Unified)
	}

	if err != nil {
		return fmt.Errorf("failed to create cgroup: %v", err)
	}

	// Setup mounts
	if c.Spec.Mounts != nil {
		for _, mount := range c.Spec.Mounts {
			if err := setupMount(mount); err != nil {
				return fmt.Errorf("failed to setup mount %s: %v", mount.Destination, err)
			}
		}
	}

	// Setup security configurations
	if c.Spec.Process != nil {
		// Setup capabilities
		if c.Spec.Process.Capabilities != nil {
			if err := setupCapabilities(c.Spec.Process.Capabilities); err != nil {
				return fmt.Errorf("failed to setup capabilities: %v", err)
			}
		}

		// Setup seccomp
		if c.Spec.Linux != nil && c.Spec.Linux.Seccomp != nil {
			if err := setupSeccomp(c.Spec.Linux.Seccomp); err != nil {
				return fmt.Errorf("failed to setup seccomp: %v", err)
			}
		}

		// Setup rlimits
		if c.Spec.Process.Rlimits != nil {
			if err := setupRlimits(c.Spec.Process.Rlimits); err != nil {
				return fmt.Errorf("failed to setup rlimits: %v", err)
			}
		}
	}

	// Execute prestart hooks
	if err := c.executeHooks(c.Spec.Hooks.Prestart); err != nil {
		return fmt.Errorf("prestart hooks failed: %v", err)
	}

	// Execute createRuntime hooks
	if err := c.executeHooks(c.Spec.Hooks.CreateRuntime); err != nil {
		return fmt.Errorf("createRuntime hooks failed: %v", err)
	}

	// Execute createContainer hooks
	if err := c.executeHooks(c.Spec.Hooks.CreateContainer); err != nil {
		return fmt.Errorf("createContainer hooks failed: %v", err)
	}

	// Start init process
	if err := c.InitProcess.Start(); err != nil {
		return fmt.Errorf("failed to start init process: %v", err)
	}

	c.State.Status = StateCreated

	// Execute poststart hooks
	if err := c.executeHooks(c.Spec.Hooks.Poststart); err != nil {
		// Log warning but continue
		fmt.Printf("warning: poststart hooks failed: %v\n", err)
	}

	return stateManager.UpdateState(c.State)
}

// Start starts the container process
func (c *Container) Start() error {
	if c.State.Status != StateCreated {
		return fmt.Errorf("container must be in created state to start")
	}

	stateManager := NewStateManager()

	// Execute startContainer hooks
	if err := c.executeHooks(c.Spec.Hooks.StartContainer); err != nil {
		return fmt.Errorf("startContainer hooks failed: %v", err)
	}

	// Start the container process
	cmd := exec.Command(c.Spec.Process.Args[0], c.Spec.Process.Args[1:]...)
	cmd.SysProcAttr = &unix.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	c.Process.ID = cmd.Process.Pid
	c.State.Status = StateRunning

	// Execute poststart hooks
	if err := c.executeHooks(c.Spec.Hooks.Poststart); err != nil {
		// Log warning but continue
		fmt.Printf("warning: poststart hooks failed: %v\n", err)
	}

	return stateManager.UpdateState(c.State)
}

// Kill sends a signal to the container process
func (c *Container) Kill(signal unix.Signal) error {
	if c.State.Status != StateCreated && c.State.Status != StateRunning {
		return fmt.Errorf("container must be in created or running state to kill")
	}

	if c.Process.ID == -1 {
		return fmt.Errorf("container is not running")
	}

	return unix.Kill(c.Process.ID, signal)
}

// Delete removes the container
func (c *Container) Delete() error {
	if c.State.Status != StateStopped {
		return fmt.Errorf("container must be in stopped state to delete")
	}

	stateManager := NewStateManager()

	// Execute poststop hooks
	if err := c.executeHooks(c.Spec.Hooks.Poststop); err != nil {
		// Log warning but continue
		fmt.Printf("warning: poststop hooks failed: %v\n", err)
	}

	return stateManager.DeleteState(c.ID)
}

// loadSpec loads the OCI spec from the bundle
func loadSpec(bundle string) (*specs.Spec, error) {
	configPath := filepath.Join(bundle, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.json: %v", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %v", err)
	}

	return &spec, nil
}

// executeHooks executes a list of hooks
func (c *Container) executeHooks(hooks []specs.Hook) error {
	for _, hook := range hooks {
		cmd := exec.Command(hook.Path, hook.Args...)
		cmd.Env = hook.Env
		cmd.Dir = c.Bundle

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %s failed: %v", hook.Path, err)
		}
	}
	return nil
}

// setupMount sets up a filesystem mount
func setupMount(mount specs.Mount) error {
	// Implementation for mounting filesystems
	// This is a placeholder for the actual implementation
	return nil
}

// setupCapabilities sets up process capabilities
func setupCapabilities(caps *specs.LinuxCapabilities) error {
	// Implementation for setting up capabilities
	// This is a placeholder for the actual implementation
	return nil
}

// setupSeccomp sets up seccomp filters
func setupSeccomp(seccomp *specs.LinuxSeccomp) error {
	// Implementation for setting up seccomp
	// This is a placeholder for the actual implementation
	return nil
}

// setupRlimits sets up resource limits
func setupRlimits(rlimits []specs.POSIXRlimit) error {
	// Implementation for setting up resource limits
	// This is a placeholder for the actual implementation
	return nil
}
