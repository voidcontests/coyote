package language

import (
	"runner/internal/domain"
)

type Provider struct {
	languages map[string]domain.Language
}

func NewProvider() *Provider {
	return &Provider{
		languages: map[string]domain.Language{
			domain.LanguageC: {
				Name:      domain.LanguageC,
				Kind:      domain.Compiled,
				Extension: "c",
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
