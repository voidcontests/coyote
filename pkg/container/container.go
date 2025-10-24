package container

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Context holds common parameters for container operations
type Context struct {
	client      *client.Client
	containerID string
}

// ProcessResult stores different information about result of executed command
type ProcessResult struct {
	Ok       bool
	ExitCode int
	Stdout   string
	Stderr   string
}

// New creates and starts new docker container with security options and return attached Context
func New(ctx context.Context, c *client.Client) (*Context, error) {
	secopts, err := readSecurityOpts()
	if err != nil {
		return nil, err
	}

	resp, err := c.ContainerCreate(ctx, &container.Config{
		Image: "ghcr.io/voidcontests/runner:latest",
		Cmd:   []string{"sleep", "3600"},
	}, &container.HostConfig{
		SecurityOpt: []string{
			fmt.Sprintf("seccomp=%s", secopts),
		},
	}, nil, nil, "")

	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	err = c.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &Context{
		client:      c,
		containerID: resp.ID,
	}, nil
}

// Flush force remove attached container to Context
func (cc *Context) Flush(ctx context.Context) error {
	return cc.client.ContainerRemove(ctx, cc.containerID, container.RemoveOptions{Force: true})
}

// WriteFile writes a file into container file system
func (cc *Context) WriteFile(ctx context.Context, path string, content string) error {
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", b64, path)
	_, err := cc.Execute(ctx, cmd)
	return err
}

// Execute executes provided `cmd` inside the container
func (cc *Context) Execute(ctx context.Context, cmd string) (ProcessResult, error) {
	execopts, err := cc.client.ContainerExecCreate(ctx, cc.containerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/bash", "-c", cmd},
	})
	if err != nil {
		return ProcessResult{}, fmt.Errorf("failed to create exec: %w", err)
	}

	hr, err := cc.client.ContainerExecAttach(ctx, execopts.ID, container.ExecAttachOptions{})
	if err != nil {
		return ProcessResult{}, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer hr.Close()

	var execution struct {
		stdout bytes.Buffer
		stderr bytes.Buffer
	}

	done := make(chan error, 1)

	go func() {
		reader := &reader{ctx: ctx, r: hr.Reader}
		_, err := stdcopy.StdCopy(&execution.stdout, &execution.stderr, reader)
		if err != nil && !errors.Is(err, io.EOF) {
			done <- err
		} else {
			done <- nil
		}
	}()

	select {
	case <-ctx.Done():
		return ProcessResult{
			Ok:       false,
			ExitCode: -1,
			Stdout:   execution.stdout.String(),
			Stderr:   execution.stderr.String(),
		}, ctx.Err()
	case err := <-done:
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			return ProcessResult{}, fmt.Errorf("failed to read program output: %w", err)
		}
	}

	ei, err := cc.client.ContainerExecInspect(ctx, execopts.ID)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("failed to inspect execution: %w", err)
	}

	return ProcessResult{
		Ok:       ei.ExitCode == 0,
		ExitCode: ei.ExitCode,
		Stdout:   execution.stdout.String(),
		Stderr:   execution.stderr.String(),
	}, nil
}

// Execute executes provided `cmd` inside the container with given timeout
func (cc *Context) ExecuteWithTimeout(ctx context.Context, cmd string, timeout time.Duration) (ProcessResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return cc.Execute(ctx, cmd)
}

type reader struct {
	ctx context.Context
	r   io.Reader
}

func (r *reader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	return r.r.Read(p)
}

func readSecurityOpts() (string, error) {
	bytes, err := os.ReadFile("seccomp.json")
	if err != nil {
		return "", fmt.Errorf("failed to read seccomp.json: %w", err)
	}

	return string(bytes), nil
}
