package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Runner interface {
	Run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, error)
	RunShell(ctx context.Context, dir string, timeout time.Duration, command string) (string, error)
}

type DefaultRunner struct{}

func (r DefaultRunner) Run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, error) {
	cmdCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	if cancel != nil {
		defer cancel()
	}

	// #nosec G204 - command and arguments are dynamically provided to execute git commands
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("timeout running %s", strings.Join(args, " "))
	}
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return string(output), fmt.Errorf("%v: %s", err, trimmed)
		}
		return string(output), err
	}
	return string(output), nil
}

func (r DefaultRunner) RunShell(ctx context.Context, dir string, timeout time.Duration, command string) (string, error) {
	// #nosec G204 - shell execution is a core feature for custom recommendations
	return r.Run(ctx, dir, timeout, "sh", "-c", command)
}

var GlobalRunner Runner = DefaultRunner{}
