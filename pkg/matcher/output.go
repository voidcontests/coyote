package matcher

import (
	"strings"
)

type OutputMatcher struct{}

func NewOutputMatcher() *OutputMatcher {
	return &OutputMatcher{}
}

func (m *OutputMatcher) Match(actual, expected string) bool {
	suffix := "\n"
	return strings.TrimSuffix(actual, suffix) == strings.TrimSuffix(expected, suffix)
}
