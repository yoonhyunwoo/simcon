package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
)

const (
	exactArgs = iota
	minArgs
	maxArgs
)

func checkArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name
	switch checkType {
	case exactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case minArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case maxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}

	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		_ = cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

func createContainer(context *cli.Context) (int, error) {
	containerID := context.Args().First()
	bundlePath := context.Args().Get(1)
	configPath := filepath.Join(bundlePath, configFile)
	config, err := loadConfig(configPath)
	if err != nil {
		return -1, err
	}
	//rootfs := filepath.Join(bundlePath, config.Root.Path)

	containerDir := filepath.Join(dataDir, containerID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return -1, err
	}

	stateFile := filepath.Join(containerDir, "state.json")
	_, err = os.Create(stateFile)
	if err != nil {
		return -1, err
	}

	pid, err := initContainerProcess(config, containerDir)

	containerState := specs.State{
		Version: ociVersion,
		ID:      containerID,
		Status:  specs.StateCreated,
		Pid:     pid,
		Bundle:  bundlePath,
		Annotations: map[string]string{
			"created": time.Now().String(),
			"config":  fmt.Sprintf("%v", config),
		},
	}

	sData, err := os.OpenFile(stateFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return -1, err
	}

	if err := json.NewEncoder(sData).Encode(containerState); err != nil {
		return -1, err
	}

	//cgroups 설정

	// Networks 설정
	return 1, nil

}

func initContainerProcess(config *specs.Spec, containerDir string) (int, error) {
	runtime.GOMAXPROCS(1)
	// 쓰레드 잠금
	runtime.LockOSThread()

	cmd := exec.Command("/proc/self/exe", "init", containerDir)

	// main process stdio setup
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// hostname namespace
		Cloneflags: syscall.CLONE_NEWUTS |
			// new pic namespace
			syscall.CLONE_NEWPID |
			// new mount namespace
			syscall.CLONE_NEWNS |
			// new network namespace
			syscall.CLONE_NEWNET,
	}

	// 실행
	if err := cmd.Start(); err != nil {
		return -1, err
	}

	pid := cmd.Process.Pid

	// Network 설정

	// overlayFS 설정

	// pivot_root 설정

	if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return -1, err
	}

	return pid, nil
}

func containerInitProcess(context *cli.Context) error {
	fmt.Println("conmtainer init process..")
	return nil
}

// pivotRoot will call pivot_root such that rootfs becomes the new root
// filesystem, and everything else is cleaned up.
func pivotRoot(rootfs string) error {
	// While the documentation may claim otherwise, pivot_root(".", ".") is
	// actually valid. What this results in is / being the new root but
	// /proc/self/cwd being the old root. Since we can play around with the cwd
	// with pivot_root this allows us to pivot without creating directories in
	// the rootfs. Shout-outs to the LXC developers for giving us this idea.

	oldroot, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: "/", Err: err}
	}
	defer unix.Close(oldroot) //nolint: errcheck

	newroot, err := unix.Open(rootfs, unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: rootfs, Err: err}
	}
	defer unix.Close(newroot) //nolint: errcheck

	// Change to the new root so that the pivot_root actually acts on it.
	if err := unix.Fchdir(newroot); err != nil {
		return &os.PathError{Op: "fchdir", Path: "fd " + strconv.Itoa(newroot), Err: err}
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return &os.PathError{Op: "pivot_root", Path: ".", Err: err}
	}

	// Currently our "." is oldroot (according to the current kernel code).
	// However, purely for safety, we will fchdir(oldroot) since there isn't
	// really any guarantee from the kernel what /proc/self/cwd will be after a
	// pivot_root(2).

	if err := unix.Fchdir(oldroot); err != nil {
		return &os.PathError{Op: "fchdir", Path: "fd " + strconv.Itoa(oldroot), Err: err}
	}

	// Make oldroot rslave to make sure our unmounts don't propagate to the
	// host (and thus bork the machine). We don't use rprivate because this is
	// known to cause issues due to races where we still have a reference to a
	// mount while a process in the host namespace are trying to operate on
	// something they think has no mounts (devicemapper in particular).

	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return err
	}
	// Perform the unmount. MNT_DETACH allows us to unmount /proc/self/cwd.
	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return err
	}

	// Switch back to our shiny new root.
	if err := unix.Chdir("/"); err != nil {
		return &os.PathError{Op: "chdir", Path: "/", Err: err}
	}
	return nil
}
