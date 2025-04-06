package runner

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runner/internal/runner/language"
)

type Report struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Compile compiles source file into binary for future executing
func Compile(filebase, lang string) (Report, error) {
	l, ok := language.Get(lang)
	if !ok {
		return Report{}, language.ErrUnknownLanguage
	}

	if l.Kind != language.Compiled {
		return Report{}, language.ErrNotCompiledLanguage
	}

	command := getCompilationCommand(lang, filebase)

	return isolate(command)
}

// Exec executes either compiled binary, or interprets file
// in case of interpreted language
func Exec(filebase, lang string, timeLimitMS int, input string) (Report, error) {
	command := getExecuteCommand(lang, filebase, input, timeLimitMS)
	if command == "" {
		return Report{}, language.ErrUnknownLanguage
	}

	return isolate(command)
}

// Flush removes all files by filebase - request timestamp
func Flush(filebase string) error {
	command := fmt.Sprintf(`find /sandbox -type f -name "%s.*" -delete`, filebase)

	var err error
	_, err = isolate(command)
	return err
}

func getCompilationCommand(lang, filebase string) string {
	switch lang {
	case language.C:
		return fmt.Sprintf(`gcc -std=c17 -o %s.out /sandbox/%s.c`, filebase, filebase)
	}

	return ""
}

func getExecuteCommand(lang, filebase, input string, timeLimitMS int) string {
	switch lang {
	case language.C:
		return fmt.Sprintf(`echo "%s" | timeout %0.3fs /sandbox/%s.out`, input, float64(timeLimitMS)/1000, filebase)
	case language.Python:
		return fmt.Sprintf(`echo "%s" | timeout %0.3fs python3 /sandbox/%s.py`, input, float64(timeLimitMS)/1000, filebase)
	}

	return ""
}

func isolate(command string) (Report, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"--cpus=0.5",
		"--memory=128m",
		"--memory-swap=256m",
		"--pids-limit=50",
		"--read-only",
		"--network=none",
		"-v", "./files:/sandbox",
		"jus1d/void-runner:latest",
		"bash", "-c", command,
	)

	var r Report
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			r.ExitCode = ee.ExitCode()
			r.Stderr = string(ee.Stderr)
		} else {
			slog.Error("can't execute command", slog.String("error", err.Error()))
			return Report{}, err
		}
	}
	r.Stdout = string(out)

	return r, nil
}
