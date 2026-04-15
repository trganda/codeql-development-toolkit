package executil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Runner executes external commands and always captures stdout/stderr.
type Runner struct {
	Binary string
	Stdin  io.Reader
}

// Result contains captured command output.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// RunError includes command details and captured stderr for failed runs.
type RunError struct {
	Binary   string
	Args     []string
	ExitCode int
	Stderr   string
	Cause    error
}

func (e *RunError) Error() string {
	cmd := e.Binary
	if len(e.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", e.Binary, strings.Join(e.Args, " "))
	}
	if e.Stderr != "" {
		if e.ExitCode > 0 {
			return fmt.Sprintf("command failed (exit %d): %s: %s", e.ExitCode, cmd, e.Stderr)
		}
		return fmt.Sprintf("command failed: %s: %s", cmd, e.Stderr)
	}
	if e.ExitCode > 0 {
		return fmt.Sprintf("command failed (exit %d): %s", e.ExitCode, cmd)
	}
	return fmt.Sprintf("command failed: %s", cmd)
}

func (e *RunError) Unwrap() error {
	return e.Cause
}

// NewRunner creates a command runner for a binary.
func NewRunner(binary string) *Runner {
	return &Runner{Binary: binary}
}

// Run executes a command and captures both stdout and stderr.
func (r *Runner) Run(args ...string) (*Result, error) {
	cmd := exec.Command(r.Binary, args...)
	if r.Stdin != nil {
		cmd.Stdin = r.Stdin
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := &Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}

	if err == nil {
		return res, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
	}

	return res, &RunError{
		Binary:   r.Binary,
		Args:     append([]string(nil), args...),
		ExitCode: res.ExitCode,
		Stderr:   strings.TrimSpace(stderr.String()),
		Cause:    err,
	}
}

func (r *Result) StdoutString() string {
	return string(r.Stdout)
}

func (r *Result) StderrString() string {
	return string(r.Stderr)
}
