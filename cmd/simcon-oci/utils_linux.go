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

// validateArgs 검증 함수는 CLI 인자 개수를 검증합니다.
// checkType에 따라 정확한 개수, 최소 개수, 최대 개수를 확인합니다.
func validateArgs(ctx *cli.Context, expectedCount, checkType int) error {
	var err error
	commandName := ctx.Command.Name

	switch checkType {
	case exactArgs:
		if ctx.NArg() != expectedCount {
			err = fmt.Errorf("%s: %q 명령은 정확히 %d개의 인자가 필요합니다.", os.Args[0], commandName, expectedCount)
		}
	case minArgs:
		if ctx.NArg() < expectedCount {
			err = fmt.Errorf("%s: %q 명령은 최소 %d개의 인자가 필요합니다.", os.Args[0], commandName, expectedCount)
		}
	case maxArgs:
		if ctx.NArg() > expectedCount {
			err = fmt.Errorf("%s: %q 명령은 최대 %d개의 인자를 허용합니다.", os.Args[0], commandName, expectedCount)
		}
	}

	if err != nil {
		fmt.Println("Incorrect Usage.\n")
		_ = cli.ShowCommandHelp(ctx, commandName)
		return err
	}
	return nil
}

// create는 새로운 컨테이너를 생성하고 초기 상태를 저장합니다.
func create(ctx *cli.Context) (int, error) {
	containerID := ctx.Args().First()
	bundlePath := ctx.Args().Get(1)
	configPath := filepath.Join(bundlePath, configFile)

	// OCI 스펙 config 파일 로딩
	config, err := loadConfig(configPath)
	if err != nil {
		return -1, err
	}

	// 컨테이너 메타데이터 저장 디렉토리 생성
	containerMetadataDir := filepath.Join(dataDir, containerID)
	if err := os.MkdirAll(containerMetadataDir, 0755); err != nil {
		return -1, err
	}

	// 컨테이너 상태 파일 생성
	stateFilePath := filepath.Join(containerMetadataDir, "state.json")
	if _, err := os.Create(stateFilePath); err != nil {
		return -1, err
	}

	// 컨테이너 프로세스 시작
	pid, err := forkProcess(config, containerMetadataDir)
	if err != nil {
		return -1, err
	}

	// 컨테이너 상태 저장
	configBytes, err := json.Marshal(config)
	if err != nil {
		return -1, err
	}

	containerState := specs.State{
		Version: ociVersion,
		ID:      containerID,
		Status:  specs.StateCreated,
		Pid:     pid,
		Bundle:  bundlePath,
		Annotations: map[string]string{
			"created": time.Now().String(),
			"config":  string(configBytes),
		},
	}

	stateFile, err := os.OpenFile(stateFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return -1, err
	}

	if err := json.NewEncoder(stateFile).Encode(containerState); err != nil {
		return -1, err
	}

	// TODO: cgroups 설정
	// TODO: 네트워크 설정

	return 1, nil
}

func state(ctx *cli.Context) (*specs.State, error) {

}

// forkProcess는 새로운 네임스페이스와 함께 init 프로세스를 시작합니다.
func forkProcess(config *specs.Spec, containerMetadataDir string) (int, error) {
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()

	cmd := exec.Command("/proc/self/exe", "init", containerMetadataDir)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // Hostname namespace
			syscall.CLONE_NEWPID | // PID namespace
			syscall.CLONE_NEWNS | // Mount namespace
			syscall.CLONE_NEWNET, // Network namespace
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("init 프로세스 시작 실패: %v\n", err)
		return -1, err
	}

	pid := cmd.Process.Pid

	// TODO: 네트워크 초기화
	// TODO: overlayFS 초기화
	// TODO: pivot_root 처리

	// 부모 프로세스가 기다리도록 SIGSTOP 전송
	if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		fmt.Printf("SIGSTOP 전송 실패: %v\n", err)
		return -1, err
	}

	return pid, nil
}

// containerInit는 init 프로세스에서 실행됩니다.
func containerInit(ctx *cli.Context) error {
	fmt.Println("컨테이너 init 프로세스 실행 중...")
	return nil
}

// pivotRoot는 현재 프로세스의 root filesystem을 변경합니다.
func pivotRootfs(newRootfs string) error {
	oldRootFD, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: "/", Err: err}
	}
	defer unix.Close(oldRootFD)

	newRootFD, err := unix.Open(newRootfs, unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: newRootfs, Err: err}
	}
	defer unix.Close(newRootFD)

	if err := unix.Fchdir(newRootFD); err != nil {
		return &os.PathError{Op: "fchdir", Path: strconv.Itoa(newRootFD), Err: err}
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return &os.PathError{Op: "pivot_root", Path: ".", Err: err}
	}

	if err := unix.Fchdir(oldRootFD); err != nil {
		return &os.PathError{Op: "fchdir", Path: strconv.Itoa(oldRootFD), Err: err}
	}

	// 슬레이브 마운트로 변경하여 호스트에 영향이 가지 않게 함
	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return err
	}

	// 이전 루트를 언마운트 (detach 방식)
	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return err
	}

	// 최종적으로 new root로 이동
	if err := unix.Chdir("/"); err != nil {
		return &os.PathError{Op: "chdir", Path: "/", Err: err}
	}

	return nil
}
