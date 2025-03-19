* `simcon-oci create`
  * Create container (without running process)
  * Load config.json from bundle directory
  * Mount rootfs
  * Initialize namespaces (PID, NET, etc.) and cgroups
  * Create runtime state directory (/var/lib/simcon/oci/containers/<container-id>/)
* simcon-oci init
```
* ✅ runc init 과정에서 수행하는 주요 작업 순서
  세션 키링(Session Keyring) 설정

사용자 네임스페이스 사용 여부에 따라 키링 권한을 다르게 설정하고 세션 키링에 참여합니다.
JoinSessionKeyring, ModKeyringPerm 함수 호출로 키링 생성 및 권한 설정.
네트워크 및 라우팅 설정

setupNetwork() 호출: 네트워크 인터페이스를 설정합니다.
setupRoute() 호출: 라우팅 테이블을 구성합니다.
SELinux 라벨링 초기화

selinux.GetEnabled()로 SELinux 활성화 상태 확인.
selinux.SetKeyLabel()과 selinux.SetExecLabel()로 프로세스 및 키에 라벨을 설정합니다.
루트 파일 시스템 준비

prepareRootfs() 호출하여 루트 파일 시스템을 마운트 및 설정합니다.
콘솔 설정

setupConsole()을 통해 새 터미널(콘솔) 생성.
system.Setctty()로 컨트롤 터미널을 지정합니다.
PIDFD(프로세스 파일 디스크립터) 설정

setupPidfd() 호출로 pidfd 전달 및 확인.
루트 파일 시스템 최종 설정

finalizeRootfs()로 루트 파일 시스템을 최종 마운트하고 정리.
호스트 이름 및 도메인 이름 설정

unix.Sethostname() 및 unix.Setdomainname()을 통해 네임스페이스 내 호스트, 도메인 이름 설정.
AppArmor 프로파일 적용

apparmor.ApplyProfile() 호출로 AppArmor 보안 프로파일을 적용.
시스템 파라미터(Sysctl) 설정

writeSystemProperty() 호출로 sysctl 값들을 커널에 설정합니다.
특정 파일 시스템 경로 보호

readonlyPath() 및 maskPath()를 통해 일부 경로를 읽기 전용 또는 마스킹 처리합니다.
부모 프로세스 종료 감지(PDeathSignal)

system.GetParentDeathSignal()을 통해 부모 프로세스 사망 신호를 감지하고 복구합니다.
NoNewPrivileges 적용

unix.Prctl() 호출로 PR_SET_NO_NEW_PRIVS 설정, 권한 상승 방지.
스케줄링 및 IO 우선순위 설정

setupScheduler() 및 setupIOPriority() 호출.
부모 프로세스에게 준비 완료 신호 전송

syncParentReady() 호출로 부모 프로세스에 준비 완료 알림.
Seccomp 적용

seccomp.InitSeccomp() 및 syncParentSeccomp() 호출하여 Seccomp 정책을 적용하여 시스템 콜 제한.
네임스페이스 최종 설정

finalizeNamespace()로 사용자, 그룹, 기타 네임스페이스를 최종 적용.
StartContainer 훅 실행

컨테이너 시작 훅(Hook) 실행 및 환경 변수 적용.
부모 PID 확인 및 안전성 검증

unix.Getppid() 값이 초기 부모 프로세스와 같은지 확인.
실행 파일 경로 확인 및 exec 준비

exec.LookPath()로 컨테이너에서 실행할 명령어 확인.
setupPersonality()로 시스템 콜 인터페이스의 동작을 변경하는 personality 설정.
FIFO 동기화 및 exec 신호 전송

FIFO 파일을 열고 실행 신호를 보냄.
utils.ProcThreadSelfFd()와 unix.Open() 사용.
StartContainer 훅 추가 실행

컨테이너 상태 정보(specs.StateCreated) 전달하고 훅 실행.
불필요한 파일 디스크립터 닫기

utils.UnsafeCloseFrom() 호출로 컨테이너 내부에서 필요 없는 FD를 닫아 보안 강화 (CVE-2024-21626 관련).
최종적으로 컨테이너 프로세스 실행

system.Exec() 호출하여 컨테이너 내 사용자 프로세스를 execve()로 실행하고, init 프로세스 종료.
```
