//go:build !windows

package build

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

type commandRunner struct{}

func newProcessRunner() processRunner { return commandRunner{} }

func (commandRunner) Run(ctx context.Context, command, dir string, emit func(string, string)) processOutcome {
	startedAt := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return processOutcome{StartTime: startedAt, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return processOutcome{StartTime: startedAt, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return processOutcome{StartTime: startedAt, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}
	var output sync.WaitGroup
	output.Add(2)
	go streamOutput(stdout, StreamStdout, emit, &output)
	go streamOutput(stderr, StreamStderr, emit, &output)
	waitErr := cmd.Wait()
	output.Wait()
	outcome := processOutcome{
		StartTime: startedAt,
		EndTime:   time.Now(),
		Started:   true,
		Cancelled: errors.Is(ctx.Err(), context.Canceled),
		Err:       waitErr,
		Technical: waitErr,
		ExitCode:  0,
	}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		outcome.ExitCode = exitErr.ExitCode()
	}
	if outcome.Cancelled {
		outcome.ExitCode = -1
	}
	return outcome
}
