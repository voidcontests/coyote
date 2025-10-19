package domain

import "errors"

var (
	ErrUnknownLanguage     = errors.New("unknown language")
	ErrNotCompiledLanguage = errors.New("language is not compiled")
)

type LanguageKind uint8

const (
	Compiled LanguageKind = 1 << iota
	Interpreted
)

const (
	LanguageCPP    = "cpp"
	LanguagePython = "python"
)

type Language struct {
	Name      string
	Kind      LanguageKind
	Extension string
}
