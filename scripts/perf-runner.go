//go:build ignore

// perf-runner is the standard-library fallback for hosts without GNU time.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/perf-runner.go METRICS_FILE COMMAND ARG...")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[2], os.Args[3:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	started := time.Now()
	err := command.Run()
	elapsed := time.Since(started).Seconds()
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintln(os.Stderr, "performance command exceeded 30 seconds")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "performance command failed: %v\n", err)
		os.Exit(1)
	}

	usage, ok := command.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok {
		fmt.Fprintln(os.Stderr, "process resource usage is unavailable")
		os.Exit(1)
	}
	maxRSSKiB := usage.Maxrss
	if runtime.GOOS == "darwin" {
		maxRSSKiB /= 1024
	}

	metrics := fmt.Sprintf("%.3f %d\n", elapsed, maxRSSKiB)
	if err := os.WriteFile(os.Args[1], []byte(metrics), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write performance metrics: %v\n", err)
		os.Exit(1)
	}
}
