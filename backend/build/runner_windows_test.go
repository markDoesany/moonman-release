//go:build windows

package build

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWindowsCommandRunnerStreamsBothOutputStreams(t *testing.T) {
	runner := newProcessRunner()
	var output []string
	outcome := runner.Run(context.Background(), `echo stdout && echo stderr 1>&2`, t.TempDir(), func(stream, text string) {
		output = append(output, stream+":"+text)
	})

	if outcome.Err != nil {
		t.Fatalf("command failed: %v", outcome.Err)
	}
	if !containsOutput(output, StreamStdout, "stdout") || !containsOutput(output, StreamStderr, "stderr") {
		t.Fatalf("expected stdout and stderr, got %v", output)
	}
}

func TestWindowsCommandRunnerCancelsProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := newProcessRunner()
	done := make(chan processOutcome, 1)
	go func() {
		done <- runner.Run(ctx, `ping 127.0.0.1 -n 30 > nul`, t.TempDir(), func(string, string) {})
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case outcome := <-done:
		if !outcome.Cancelled {
			t.Fatalf("outcome was not marked cancelled: %+v", outcome)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled process did not exit")
	}
}

func containsOutput(output []string, stream, text string) bool {
	for _, line := range output {
		if strings.HasPrefix(line, stream+":") && strings.Contains(line, text) {
			return true
		}
	}
	return false
}
