package matcher

import (
	"strings"
)

func Match(actual, expected string) bool {
	suffix := "\n"
	return strings.TrimSuffix(actual, suffix) == strings.TrimSuffix(expected, suffix)
}
