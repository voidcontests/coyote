package language

import (
	"github.com/voidcontests/coyote/internal/domain"
)

type Provider struct {
	languages map[string]domain.Language
}

func NewProvider() *Provider {
	return &Provider{
		languages: map[string]domain.Language{
			domain.LanguageCPP: {
				Name:      domain.LanguageCPP,
				Kind:      domain.Compiled,
				Extension: "cpp",
			},
			domain.LanguagePython: {
				Name:      domain.LanguagePython,
				Kind:      domain.Interpreted,
				Extension: "py",
			},
		},
	}
}

func (p *Provider) GetLanguage(name string) (domain.Language, error) {
	lang, ok := p.languages[name]
	if !ok {
		return domain.Language{}, domain.ErrUnknownLanguage
	}
	return lang, nil
}

func (p *Provider) RegisterLanguage(lang domain.Language) {
	p.languages[lang.Name] = lang
}
