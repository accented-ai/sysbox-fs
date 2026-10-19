package sysctlprobe

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

type probeFile struct{ close func() error }

func (f probeFile) Close() error { return f.close() }

func TestInspect(t *testing.T) {
	for _, failure := range []error{nil, syscall.EPERM, syscall.EACCES, syscall.EIO} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			opened, closed := 0, 0
			paths := map[string]bool{
				"/proc/sys/kernel/shmmni": true,
				"/proc/sys/kernel/shmmax": true,
				"/proc/sys/kernel/shmall": true,
			}
			supported, err := inspect(func(path string, flags int, mode os.FileMode) (io.Closer, error) {
				if !paths[path] {
					t.Fatalf("unexpected or repeated sysctl: %s", path)
				}
				delete(paths, path)
				if flags != os.O_RDWR || mode != 0 {
					t.Fatalf("unexpected open flags/mode: %d/%v", flags, mode)
				}
				opened++
				if path == "/proc/sys/kernel/shmmax" && failure != nil {
					return nil, &os.PathError{Op: "open", Path: path, Err: failure}
				}
				return probeFile{close: func() error { closed++; return nil }}, nil
			})
			if supported != (failure == nil) || (err != nil) != (failure == syscall.EIO) {
				t.Fatalf("unexpected result: supported=%v error=%v", supported, err)
			}
			if failure == syscall.EIO && !errors.Is(err, failure) {
				t.Fatalf("lost I/O error: %v", err)
			}
			if failure == nil && (opened != 3 || closed != 3) || failure != nil && (opened != 2 || closed != 1) {
				t.Fatalf("unexpected file ownership: opened=%d closed=%d", opened, closed)
			}
		})
	}
}

func helperCommand(ctx context.Context, mode string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProbeProcess$", "--", mode)
	cmd.Env = []string{"SYSBOXFS_PROBE_TEST=1", "GOMAXPROCS=1"}
	cmd.WaitDelay = time.Second
	return cmd
}

func TestRunResults(t *testing.T) {
	for _, mode := range []string{"success", "unsupported", "failure", "panic"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			supported, err := run(ctx, helperCommand(ctx, mode))
			if supported != (mode == "success") || (err != nil) != (mode == "failure" || mode == "panic") {
				t.Fatalf("unexpected result: supported=%v error=%v", supported, err)
			}
			if mode == "failure" && !strings.Contains(err.Error(), "probe failed") {
				t.Fatalf("lost child diagnostic: %v", err)
			}
			if mode == "panic" && !strings.Contains(err.Error(), "probe panicked") {
				t.Fatalf("lost panic diagnostic: %v", err)
			}
		})
	}
}

func TestRunCancelsAndReaps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := helperCommand(ctx, "block")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := run(ctx, cmd); done <- err }()
	if ready, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || ready != "ready\n" {
		t.Fatalf("child handshake: %q, %v", ready, err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cancellation: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe did not return after cancellation")
	}
	var status syscall.WaitStatus
	if _, err := syscall.Wait4(cmd.Process.Pid, &status, syscall.WNOHANG, nil); !errors.Is(err, syscall.ECHILD) {
		t.Fatalf("child was not reaped: %v", err)
	}
}

func TestProbeProcess(t *testing.T) {
	if os.Getenv("SYSBOXFS_PROBE_TEST") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "success":
		os.Exit(0)
	case "unsupported":
		os.Exit(unsupportedExitCode)
	case "failure":
		fmt.Fprintln(os.Stderr, "probe failed")
		os.Exit(1)
	case "panic":
		panic("probe panicked")
	case "block":
		fmt.Println("ready")
		var input [1]byte
		os.Stdin.Read(input[:])
		os.Exit(4)
	}
	os.Exit(5)
}
