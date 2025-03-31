package runtime

import (
	"reflect"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func SetUser(spec *specs.Spec) error {
	if err := unix.Setuid(int(spec.Process.User.UID)); err != nil {
		return err
	}
	if err := unix.Setgid(int(spec.Process.User.GID)); err != nil {
		return err
	}

	additionalGids := make([]int, len(spec.Process.User.AdditionalGids))
	for i, gid := range spec.Process.User.AdditionalGids {
		additionalGids[i] = int(gid)
	}

	if err := unix.Setgroups(additionalGids); err != nil {
		return err
	}

	unix.Umask(int(*spec.Process.User.Umask))

	if spec.Process.User.Username != "" {
		if err := unix.Setenv("USER", spec.Process.User.Username); err != nil {
			return err
		}
	}

	return nil

}

func SerEnv(spec *specs.Spec) error {
	for _, env := range spec.Process.Env {
		key, value, _ := strings.Cut(env, "=")
		unix.Setenv(key, value)
	}
	return nil
}

func SeㅅRlimits(spec *specs.Spec) error {

	var rlimitMap = map[string]int{
		"RLIMIT_CPU":        unix.RLIMIT_CPU,
		"RLIMIT_FSIZE":      unix.RLIMIT_FSIZE,
		"RLIMIT_DATA":       unix.RLIMIT_DATA,
		"RLIMIT_STACK":      unix.RLIMIT_STACK,
		"RLIMIT_CORE":       unix.RLIMIT_CORE,
		"RLIMIT_RSS":        unix.RLIMIT_RSS,
		"RLIMIT_NPROC":      unix.RLIMIT_NPROC,
		"RLIMIT_NOFILE":     unix.RLIMIT_NOFILE,
		"RLIMIT_MEMLOCK":    unix.RLIMIT_MEMLOCK,
		"RLIMIT_AS":         unix.RLIMIT_AS,
		"RLIMIT_LOCKS":      unix.RLIMIT_LOCKS,
		"RLIMIT_SIGPENDING": unix.RLIMIT_SIGPENDING,
		"RLIMIT_MSGQUEUE":   unix.RLIMIT_MSGQUEUE,
		"RLIMIT_NICE":       unix.RLIMIT_NICE,
		"RLIMIT_RTPRIO":     unix.RLIMIT_RTPRIO,
		"RLIMIT_RTTIME":     unix.RLIMIT_RTTIME,
	}

	for _, rLimit := range spec.Process.Rlimits {
		unix.Setrlimit(rlimitMap[rLimit.Type], &unix.Rlimit{
			Cur: rLimit.Soft,
			Max: rLimit.Hard,
		})

	}
}
