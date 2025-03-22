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

// create는 새로운 컨테이너를 생성하고 초기 상태를 저장합니다.
func create(context *cli.Context) (int, error) {
	containerID := context.Args().First()
	bundlePath := context.Args().Get(1)
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
	stateFilePath := filepath.Join(containerMetadataDir, stateFile)
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

	if err := setupCgroups(containerState, config); err != nil {
		return -1, err
	}
	// TODO: 네트워크 설정

	stateFile, err := os.OpenFile(stateFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return -1, err
	}

	if err := json.NewEncoder(stateFile).Encode(containerState); err != nil {
		return -1, err
	}

	return 1, nil
}

func state(context *cli.Context) (specs.State, error) {
	containerID := context.Args().First()
	containerMetadataDir := filepath.Join(dataDir, containerID)
	stateFilePath := filepath.Join(containerMetadataDir, stateFile)
	containerState := specs.State{}

	sData, err := os.Open(stateFilePath)
	if err != nil {
		return containerState, err
	}
	defer sData.Close()

	json.NewDecoder(sData).Decode(&containerState)

	return containerState, nil
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
func containerInit(context *cli.Context) error {
	fmt.Println("컨테이너 init 프로세스 실행 중...")
	return nil
}

func setupCgroups(state specs.State, config specs.Spec) error {
	// cgroup 기본 경로 설정 (상수로 정의되어 있다고 가정)
	cgroupPath := filepath.Join(cgroupBasePath, state.ID)

	// 디렉토리 생성
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	// 프로세스를 cgroup에 추가
	if err := os.WriteFile(
		filepath.Join(cgroupPath, "cgroup.procs"),
		[]byte(fmt.Sprintf("%d", state.Pid)),
		0644,
	); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	// 리소스 제한 설정 부분
	if config.Linux != nil && config.Linux.Resources != nil {
		// 1. 메모리 제한 설정
		if config.Linux.Resources.Memory != nil {
			// 메모리 제한
			if config.Linux.Resources.Memory.Limit != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "memory.max"),
					[]byte(fmt.Sprintf("%d", *config.Linux.Resources.Memory.Limit)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set memory limit: %w", err)
				}
			}

			// 메모리 예약(reservation)
			if config.Linux.Resources.Memory.Reservation != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "memory.low"),
					[]byte(fmt.Sprintf("%d", *config.Linux.Resources.Memory.Reservation)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set memory reservation: %w", err)
				}
			}

			// 스왑 제한
			if config.Linux.Resources.Memory.Swap != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "memory.swap.max"),
					[]byte(fmt.Sprintf("%d", *config.Linux.Resources.Memory.Swap)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set swap limit: %w", err)
				}
			}
		}

		// 2. CPU 제한 설정
		if config.Linux.Resources.CPU != nil {
			// CPU 쿼터(quota)
			if config.Linux.Resources.CPU.Quota != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpu.max"),
					[]byte(fmt.Sprintf("%d max", *config.Linux.Resources.CPU.Quota)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set CPU quota: %w", err)
				}
			}

			// CPU 주기(period)
			if config.Linux.Resources.CPU.Period != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpu.max"),
					[]byte(fmt.Sprintf("max %d", *config.Linux.Resources.CPU.Period)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set CPU period: %w", err)
				}
			}

			// CPU 쿼터 + 주기를 함께 설정
			if config.Linux.Resources.CPU.Quota != nil && config.Linux.Resources.CPU.Period != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpu.max"),
					[]byte(fmt.Sprintf("%d %d", *config.Linux.Resources.CPU.Quota, *config.Linux.Resources.CPU.Period)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set CPU quota and period: %w", err)
				}
			}

			// CPU 가중치(weight)
			if config.Linux.Resources.CPU.Shares != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpu.weight"),
					[]byte(fmt.Sprintf("%d", *config.Linux.Resources.CPU.Shares)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set CPU weight: %w", err)
				}
			}

			// CPU 셋(cpuset) - 특정 CPU에 할당
			if config.Linux.Resources.CPU.Cpus != "" {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpuset.cpus"),
					[]byte(config.Linux.Resources.CPU.Cpus),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set CPU set: %w", err)
				}
			}

			// 메모리 노드 설정(NUMA)
			if config.Linux.Resources.CPU.Mems != "" {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "cpuset.mems"),
					[]byte(config.Linux.Resources.CPU.Mems),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set memory nodes: %w", err)
				}
			}
		}

		// 3. 프로세스 수 제한
		if config.Linux.Resources.Pids != nil {
			if err := os.WriteFile(
				filepath.Join(cgroupPath, "pids.max"),
				[]byte(fmt.Sprintf("%d", config.Linux.Resources.Pids.Limit)),
				0644,
			); err != nil {
				return fmt.Errorf("failed to set pids limit: %w", err)
			}
		}

		// 4. 블록 I/O 제한
		if config.Linux.Resources.BlockIO != nil {
			// 가중치 설정
			if config.Linux.Resources.BlockIO.Weight != nil {
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "io.weight"),
					[]byte(fmt.Sprintf("%d", *config.Linux.Resources.BlockIO.Weight)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set I/O weight: %w", err)
				}
			}

			// 장치별 스로틀링 설정
			for _, device := range config.Linux.Resources.BlockIO.ThrottleReadBpsDevice {
				deviceStr := fmt.Sprintf("%d:%d", device.Major, device.Minor)
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "io.max"),
					[]byte(fmt.Sprintf("%s rbps=%d", deviceStr, device.Rate)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set read BPS throttle: %w", err)
				}
			}

			for _, device := range config.Linux.Resources.BlockIO.ThrottleWriteBpsDevice {
				deviceStr := fmt.Sprintf("%d:%d", device.Major, device.Minor)
				if err := os.WriteFile(
					filepath.Join(cgroupPath, "io.max"),
					[]byte(fmt.Sprintf("%s wbps=%d", deviceStr, device.Rate)),
					0644,
				); err != nil {
					return fmt.Errorf("failed to set write BPS throttle: %w", err)
				}
			}
		}

		// 5. 허거 페이지(Huge Page) 제한
		for _, limit := range config.Linux.Resources.HugepageLimits {
			if err := os.WriteFile(
				filepath.Join(cgroupPath, fmt.Sprintf("hugetlb.%s.max", limit.Pagesize)),
				[]byte(fmt.Sprintf("%d", limit.Limit)),
				0644,
			); err != nil {
				return fmt.Errorf("failed to set hugepage limit for %s: %w", limit.Pagesize, err)
			}
		}

		// 6. 네트워크 우선순위 설정
		if config.Linux.Resources.Network != nil && config.Linux.Resources.Network.ClassID != nil {
			if err := os.WriteFile(
				filepath.Join(cgroupPath, "net_cls.classid"),
				[]byte(fmt.Sprintf("%d", *config.Linux.Resources.Network.ClassID)),
				0644,
			); err != nil {
				return fmt.Errorf("failed to set network class ID: %w", err)
			}
		}

		// 7. Unified cgroup 설정 (cgroup v2 전용)
		for key, value := range config.Linux.Resources.Unified {
			if err := os.WriteFile(
				filepath.Join(cgroupPath, key),
				[]byte(value),
				0644,
			); err != nil {
				return fmt.Errorf("failed to set unified cgroup parameter %s: %w", key, err)
			}
		}
	}

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

// validateArgs 검증 함수는 CLI 인자 개수를 검증합니다.
// checkType에 따라 정확한 개수, 최소 개수, 최대 개수를 확인합니다.
func validateArgs(context *cli.Context, expectedCount, checkType int) error {
	var err error
	commandName := context.Command.Name

	switch checkType {
	case exactArgs:
		if context.NArg() != expectedCount {
			err = fmt.Errorf("%s: %q 명령은 정확히 %d개의 인자가 필요합니다.", os.Args[0], commandName, expectedCount)
		}
	case minArgs:
		if context.NArg() < expectedCount {
			err = fmt.Errorf("%s: %q 명령은 최소 %d개의 인자가 필요합니다.", os.Args[0], commandName, expectedCount)
		}
	case maxArgs:
		if context.NArg() > expectedCount {
			err = fmt.Errorf("%s: %q 명령은 최대 %d개의 인자를 허용합니다.", os.Args[0], commandName, expectedCount)
		}
	}

	if err != nil {
		fmt.Println("Incorrect Usage.\n")
		_ = cli.ShowCommandHelp(context, commandName)
		return err
	}
	return nil
}
