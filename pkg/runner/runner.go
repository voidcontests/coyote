package runner

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/voidcontests/coyote/internal/domain"
)

type runner struct {
	image string
}

func New(image string) domain.Runner {
	return &runner{
		image: image,
	}
}

func (r *runner) Execute(req domain.ExecutionRequest) (domain.ExecutionReport, error) {
	command := r.buildExecuteCommand(req)
	if command == "" {
		return domain.ExecutionReport{}, domain.ErrUnknownLanguage
	}

	return r.isolate(command)
}

func (r *runner) Cleanup(filebase string) error {
	command := fmt.Sprintf(`find /tmp/sandbox -type f -name "%s.*" -delete`, filebase)
	_, err := r.isolate(command)
	return err
}

func (r *runner) buildExecuteCommand(req domain.ExecutionRequest) string {
	switch req.Language {
	case domain.LanguageC:
		return fmt.Sprintf(
			`mkdir -p /tmp/sandbox && echo '%s' > /tmp/sandbox/%s.c && gcc -std=c17 -Wall -Wextra -Werror -pedantic -o /tmp/sandbox/%s.out /tmp/sandbox/%s.c && echo "%s" | /tmp/sandbox/%s.out`,
			req.Code, req.Filebase, req.Filebase, req.Filebase, req.Input, req.Filebase,
		)
	case domain.LanguagePython:
		return fmt.Sprintf(
			`mkdir -p /tmp/sandbox && echo '%s' > /tmp/sandbox/%s.py && echo "%s" | python3 /tmp/sandbox/%s.py`,
			req.Code, req.Filebase, req.Input, req.Filebase,
		)
	default:
		return ""
	}
}

func (r *runner) isolate(command string) (domain.ExecutionReport, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"--cpus=0.5",
		"--memory=128m",
		"--pids-limit=50",
		"--network=none",
		r.image,
		"bash", "-c", command,
	)

	var report domain.ExecutionReport
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			report.ExitCode = exitErr.ExitCode()
			report.Stderr = string(exitErr.Stderr)
		} else {
			slog.Error("failed to execute command", slog.String("error", err.Error()))
			return domain.ExecutionReport{}, err
		}
	}
	report.Stdout = string(out)

	return report, nil
}
