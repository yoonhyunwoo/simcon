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

	if err := writeJSONFile(stateFilePath, containerState); err != nil {
		return -1, err
	}

	return 1, nil
}

func state(context *cli.Context) (specs.State, error) {
	containerID := context.Args().First()
	stateFilePath := filepath.Join(dataDir, containerID, stateFile)

	var containerState specs.State
	if err := readJSONFile(stateFilePath, &containerState); err != nil {
		return specs.State{}, err
	}

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
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWCGROUP,
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("init 프로세스 시작 실패: %v\n", err)
		return -1, err
	}

	pid := cmd.Process.Pid

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

func setupCgroups(state specs.State, config *specs.Spec) error {
	cgroupPath := filepath.Join(cgroupBasePath, state.ID)

	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	if err := writeCgroupFile(filepath.Join(cgroupPath, "cgroup.procs"), state.Pid); err != nil {
		return err
	}

	if config.Linux == nil || config.Linux.Resources == nil {
		return nil
	}

	res := config.Linux.Resources

	if mem := res.Memory; mem != nil {
		if mem.Limit != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "memory.max"), *mem.Limit); err != nil {
				return err
			}
		}
		if mem.Reservation != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "memory.low"), *mem.Reservation); err != nil {
				return err
			}
		}
		if mem.Swap != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "memory.swap.max"), *mem.Swap); err != nil {
				return err
			}
		}
	}

	if cpu := res.CPU; cpu != nil {
		if cpu.Quota != nil && cpu.Period != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpu.max"),
				fmt.Sprintf("%d %d", *cpu.Quota, *cpu.Period)); err != nil {
				return err
			}
		} else if cpu.Quota != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpu.max"),
				fmt.Sprintf("%d max", *cpu.Quota)); err != nil {
				return err
			}
		} else if cpu.Period != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpu.max"),
				fmt.Sprintf("max %d", *cpu.Period)); err != nil {
				return err
			}
		}

		if cpu.Shares != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpu.weight"), *cpu.Shares); err != nil {
				return err
			}
		}
		if cpu.Cpus != "" {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpuset.cpus"), cpu.Cpus); err != nil {
				return err
			}
		}
		if cpu.Mems != "" {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "cpuset.mems"), cpu.Mems); err != nil {
				return err
			}
		}
	}

	if pids := res.Pids; pids != nil {
		if err := writeCgroupFile(filepath.Join(cgroupPath, "pids.max"), pids.Limit); err != nil {
			return err
		}
	}

	if blkio := res.BlockIO; blkio != nil {
		if blkio.Weight != nil {
			if err := writeCgroupFile(filepath.Join(cgroupPath, "io.weight"), *blkio.Weight); err != nil {
				return err
			}
		}

		for _, device := range blkio.ThrottleReadBpsDevice {
			deviceStr := fmt.Sprintf("%d:%d rbps=%d", device.Major, device.Minor, device.Rate)
			if err := writeCgroupFile(filepath.Join(cgroupPath, "io.max"), deviceStr); err != nil {
				return err
			}
		}

		for _, device := range blkio.ThrottleWriteBpsDevice {
			deviceStr := fmt.Sprintf("%d:%d wbps=%d", device.Major, device.Minor, device.Rate)
			if err := writeCgroupFile(filepath.Join(cgroupPath, "io.max"), deviceStr); err != nil {
				return err
			}
		}
	}

	for _, limit := range res.HugepageLimits {
		filename := fmt.Sprintf("hugetlb.%s.max", limit.Pagesize)
		if err := writeCgroupFile(filepath.Join(cgroupPath, filename), limit.Limit); err != nil {
			return err
		}
	}

	if net := res.Network; net != nil && net.ClassID != nil {
		if err := writeCgroupFile(filepath.Join(cgroupPath, "net_cls.classid"), *net.ClassID); err != nil {
			return err
		}
	}

	for key, value := range res.Unified {
		if err := writeCgroupFile(filepath.Join(cgroupPath, key), value); err != nil {
			return err
		}
	}

	return nil
}

func setupNetwork(state specs.State, config *specs.Spec) error {
	// 네트워크 설정은 향후 구현 예정
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

	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return err
	}

	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return err
	}

	if err := unix.Chdir("/"); err != nil {
		return &os.PathError{Op: "chdir", Path: "/", Err: err}
	}

	return nil
}

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

func writeCgroupFile(path string, content interface{}) error {
	data := []byte(fmt.Sprintf("%v", content))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write to %s: %w", path, err)
	}
	return nil
}

func readJSONFile(filePath string, v interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(v)
}

func writeJSONFile(filePath string, v interface{}) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(v)
}
