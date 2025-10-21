package language

import (
	"errors"
	"fmt"
)

const (
	CPP    = "cpp"
	Python = "python"
)

var (
	ErrUnknownLanguage     = errors.New("unknown language")
	ErrNotCompiledLanguage = errors.New("language is not compiled")
)

type Language struct {
	Name       string
	IsCompiled bool
	Extension  string
}

var languages = map[string]Language{
	CPP: {
		Name:       CPP,
		IsCompiled: true,
		Extension:  "cpp",
	},
	Python: {
		Name:       Python,
		IsCompiled: false,
		Extension:  "py",
	},
}

// Get returns a language based on it's name
func Get(name string) (Language, bool) {
	lang, ok := languages[name]
	if !ok {
		return Language{}, false
	}
	return lang, true
}

// GetCompilationCommand returns a `cmd` to compile a source code of a compiled language
func GetCompilationCommand(l Language, source, output string) (string, bool) {
	if !l.IsCompiled {
		return "", false
	}

	switch l.Name {
	case CPP:
		return fmt.Sprintf("g++ -std=c++17 -Wall -Wextra -o %s %s", output, source), true
	}

	return "", false
}

// GetExecutionCommand returns a `cmd` to execute source code of interpreted language or execute compiled binary, is language is a compiled one
func GetExecutionCommand(l Language, source, input, output string) (string, bool) {
	switch l.Name {
	case CPP:
		return fmt.Sprintf("cat %s | %s", input, output), true
	case Python:
		return fmt.Sprintf("cat %s | python3 %s", input, source), true
	}

	return "", false
}
