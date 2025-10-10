package runner

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runner/internal/lib/sl"
	"runner/internal/runner/language"
)

type Report struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Compile compiles source file into binary for future executing
func Compile(filebase, lang, code string) (Report, error) {
	l, ok := language.Get(lang)
	if !ok {
		return Report{}, language.ErrUnknownLanguage
	}

	if l.Kind != language.Compiled {
		return Report{}, language.ErrNotCompiledLanguage
	}

	command := getCompilationCommand(lang, filebase, code)

	return isolate(command)
}

// Exec executes either compiled binary, or interprets file
// in case of interpreted language
func Exec(filebase, lang, code string, timeLimitMS int, input string) (Report, error) {
	command := getExecuteCommand(lang, filebase, code, input, timeLimitMS)
	if command == "" {
		return Report{}, language.ErrUnknownLanguage
	}

	return isolate(command)
}

// Flush removes all files by filebase - request timestamp
func Flush(filebase string) error {
	command := fmt.Sprintf(`find /tmp/sandbox -type f -name "%s.*" -delete`, filebase)

	var err error
	_, err = isolate(command)
	return err
}

func getCompilationCommand(lang, filebase, code string) string {
	switch lang {
	case language.C:
		// Use echo to create the file and then compile it
		return fmt.Sprintf(`mkdir -p /tmp/sandbox && echo '%s' > /tmp/sandbox/%s.c && gcc -std=c17 -Wall -Wextra -Werror -pedantic -o /tmp/sandbox/%s.out /tmp/sandbox/%s.c`, code, filebase, filebase, filebase)
	}

	return ""
}

func getExecuteCommand(lang, filebase, code, input string, timeLimitMS int) string {
	switch lang {
	case language.C:
		// For C, combine compilation and execution in one command
		return fmt.Sprintf(`mkdir -p /tmp/sandbox && echo '%s' > /tmp/sandbox/%s.c && gcc -std=c17 -Wall -Wextra -Werror -pedantic -o /tmp/sandbox/%s.out /tmp/sandbox/%s.c && echo "%s" | timeout %ds /tmp/sandbox/%s.out`, code, filebase, filebase, filebase, input, timeLimitMS/1000, filebase)
	case language.Python:
		// Use echo to create the file and then execute it
		return fmt.Sprintf(`mkdir -p /tmp/sandbox && echo '%s' > /tmp/sandbox/%s.py && echo "%s" | timeout %ds python3 /tmp/sandbox/%s.py`, code, filebase, input, timeLimitMS/1000, filebase)
	}

	return ""
}

func isolate(command string) (Report, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"--cpus=0.5",
		"--memory=128m",
		"--pids-limit=50",
		"--network=none",
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
			slog.Error("can't execute command", sl.Err(err))
			return Report{}, err
		}
	}
	r.Stdout = string(out)

	return r, nil
}
