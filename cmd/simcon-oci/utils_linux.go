package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli/v2"
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

	configPath := fmt.Sprintf("/containers/%s/config.json", containerID)

	config, err := loadConfig(configPath)

	if err != nil {
		return -1, err
	}

	// 컨테이너 상태 디렉토리 생성
	containerDir := fmt.Sprintf("/containers/%s", containerID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return -1, err
	}

	//cgroups 설정

	// Networks 설정

	// 컨테이너 생성 (자식 프로세스)
	pid, err := forkContainerProcess(config, containerDir)
	if err != nil {
		return -1, err
	}

	// PID 저장
	pidFile := fmt.Sprintf("%s/pidfile", containerDir)
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		return -1, err
	}

	return pid, nil
}

func forkContainerProcess(config *specs.Spec, containerDir string) (int, error) {
	// simcon-oci process를 복제
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
	runtime.GOMAXPROCS(1)
	// 쓰레드 잠금
	runtime.LockOSThread()

	containerDir := context.Args().First()

	if containerDir == "" {
		return fmt.Errorf("containerDir Args require")
	}

	// 1. rootfs 마운트 준비
	// 2. pivot_root 실행
	// 3. /proc, /sys 마운트 등 환경 준비
	// 4. exec로 실제 컨테이너 프로세스 실행

	if err != nil {
		fmt.Printf("init 실패: %v\n", err)
		return err
	}

	return nil
}
