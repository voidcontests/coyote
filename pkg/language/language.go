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

func Get(name string) (Language, error) {
	lang, ok := languages[name]
	if !ok {
		return Language{}, ErrUnknownLanguage
	}
	return lang, nil
}

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

func GetExecutionCommand(l Language, source, input, output string) (string, bool) {
	switch l.Name {
	case CPP:
		return fmt.Sprintf("cat %s | %s", input, output), true
	case Python:
		return fmt.Sprintf("cat %s | python3 %s", input, source), true
	}

	return "", false
}
