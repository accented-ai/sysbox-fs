package sysctlprobe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	Command = "probe-shm-userns"
	// Go runtime failures use exit status 2.
	unsupportedExitCode = 3
)

func Supported() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/proc/self/exe", Command)
	cmd.Env = []string{"GOMAXPROCS=1"}
	cmd.WaitDelay = time.Second
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:                 syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 165536, Size: 65536}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 165536, Size: 65536}},
		Credential:                 &syscall.Credential{Uid: 0, Gid: 0, NoSetGroups: true},
		GidMappingsEnableSetgroups: false,
	}
	return run(ctx, cmd)
}

func run(ctx context.Context, cmd *exec.Cmd) (bool, error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return false, fmt.Errorf("SHM namespace probe: %w", ctx.Err())
	}
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == unsupportedExitCode {
		return false, nil
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return false, fmt.Errorf("SHM namespace probe: %w: %s", err, detail)
	}
	return false, fmt.Errorf("SHM namespace probe: %w", err)
}

// Run checks access only after exec has initialized a fresh Go runtime.
func Run() int {
	supported, err := inspect(func(path string, flags int, mode os.FileMode) (io.Closer, error) {
		return os.OpenFile(path, flags, mode)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !supported {
		return unsupportedExitCode
	}
	return 0
}

func inspect(open func(string, int, os.FileMode) (io.Closer, error)) (bool, error) {
	for _, path := range []string{
		"/proc/sys/kernel/shmmni",
		"/proc/sys/kernel/shmmax",
		"/proc/sys/kernel/shmall",
	} {
		file, err := open(path, os.O_RDWR, 0)
		if err != nil {
			if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
				return false, nil
			}
			return false, fmt.Errorf("open %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return false, fmt.Errorf("close %s: %w", path, err)
		}
	}
	return true, nil
}
