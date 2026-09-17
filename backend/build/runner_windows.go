//go:build windows

package build

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type commandRunner struct{}

func newProcessRunner() processRunner { return commandRunner{} }

func (commandRunner) Run(ctx context.Context, command, dir string, emit func(string, string)) processOutcome {
	startedAt := time.Now()
	cmd := exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", command)
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

	job, jobErr := newKillOnCloseJob()
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		if job != 0 {
			_ = windows.CloseHandle(job)
		}
		return processOutcome{StartTime: startedAt, EndTime: time.Now(), ExitCode: -1, Err: err, Technical: err}
	}

	if jobErr == nil {
		assignErr := cmd.Process.WithHandle(func(handle uintptr) {
			jobErr = windows.AssignProcessToJobObject(job, windows.Handle(handle))
		})
		if assignErr != nil {
			jobErr = assignErr
		}
	}
	if jobErr != nil && job != 0 {
		_ = windows.CloseHandle(job)
		job = 0
	}

	stopWatcher := make(chan struct{})
	if job != 0 {
		go func() {
			select {
			case <-ctx.Done():
				_ = windows.TerminateJobObject(job, 1)
			case <-stopWatcher:
			}
		}()
	}

	var output sync.WaitGroup
	output.Add(2)
	go streamOutput(stdout, StreamStdout, emit, &output)
	go streamOutput(stderr, StreamStderr, emit, &output)
	waitErr := cmd.Wait()
	output.Wait()
	close(stopWatcher)
	if job != 0 {
		_ = windows.CloseHandle(job)
	}

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
	if jobErr != nil {
		outcome.Technical = jobErr
	}
	return outcome
}

func newKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}
