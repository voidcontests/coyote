package judge

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/voidcontests/coyote/internal/domain/checker"
	"github.com/voidcontests/coyote/internal/domain/verdict"
)

type Result struct {
	Verdict string // OK, WA or PE
	Message string
}

func Check(chckr, actual, expected string) Result {
	switch chckr {
	case checker.Full:
		return Strict(actual, expected)
	case checker.Floats4:
		return Floats4(actual, expected)
	case checker.Floats6:
		return Floats6(actual, expected)
	case checker.Floats9:
		return Floats9(actual, expected)
	case checker.Integers:
		return Integers(actual, expected)
	case checker.Yesno:
		return Yesno(actual, expected)
	default: // fallback to default one if unknow - tokens
		return Tokens(actual, expected)
	}
}

func Tokens(actual, expected string) Result {
	actualTokens := tokenize(actual)
	expectedTokens := tokenize(expected)

	if len(actualTokens) != len(expectedTokens) {
		return Result{
			Verdict: verdict.WA,
			Message: fmt.Sprintf("Expected %d token(s), found %d", len(expectedTokens), len(actualTokens)),
		}
	}

	for i := range actualTokens {
		if actualTokens[i] != expectedTokens[i] {
			return Result{
				Verdict: verdict.WA,
				Message: fmt.Sprintf("Token %d: expected %q, found %q", i+1, expectedTokens[i], actualTokens[i]),
			}
		}
	}

	return Result{Verdict: verdict.OK, Message: fmt.Sprintf("%d token(s)", len(actualTokens))}
}

func Integers(actual, expected string) Result {
	actualNums, err1 := parseIntegers(actual)
	expectedNums, err2 := parseIntegers(expected)

	if err1 != nil {
		return Result{Verdict: verdict.PE, Message: fmt.Sprintf("Cannot parse participant output: %v", err1)}
	}
	if err2 != nil {
		return Result{Verdict: verdict.PE, Message: fmt.Sprintf("Cannot parse jury output: %v", err2)}
	}

	if len(actualNums) != len(expectedNums) {
		return Result{
			Verdict: verdict.WA,
			Message: fmt.Sprintf("Expected %d number(s), found %d", len(expectedNums), len(actualNums)),
		}
	}

	for i := range actualNums {
		if actualNums[i] != expectedNums[i] {
			return Result{
				Verdict: verdict.WA,
				Message: fmt.Sprintf("Number %d: expected %d, found %d", i+1, expectedNums[i], actualNums[i]),
			}
		}
	}

	return Result{Verdict: verdict.OK, Message: fmt.Sprintf("%d number(s)", len(actualNums))}
}

func Floats(actual, expected string, epsilon float64) Result {
	actualFloats, err1 := parseFloats(actual)
	expectedFloats, err2 := parseFloats(expected)

	if err1 != nil {
		return Result{Verdict: verdict.PE, Message: fmt.Sprintf("Cannot parse participant output: %v", err1)}
	}
	if err2 != nil {
		return Result{Verdict: verdict.PE, Message: fmt.Sprintf("Cannot parse jury output: %v", err2)}
	}

	if len(actualFloats) != len(expectedFloats) {
		return Result{
			Verdict: verdict.WA,
			Message: fmt.Sprintf("Expected %d number(s), found %d", len(expectedFloats), len(actualFloats)),
		}
	}

	for i := range actualFloats {
		if !floatsEqual(actualFloats[i], expectedFloats[i], epsilon) {
			return Result{
				Verdict: verdict.WA,
				Message: fmt.Sprintf("Number %d: expected %.10f, found %.10f, error exceeds %.0e",
					i+1, expectedFloats[i], actualFloats[i], epsilon),
			}
		}
	}

	return Result{Verdict: verdict.OK, Message: fmt.Sprintf("%d number(s)", len(actualFloats))}
}

func Floats4(actual, expected string) Result { return Floats(actual, expected, 1e-4) }
func Floats6(actual, expected string) Result { return Floats(actual, expected, 1e-6) }
func Floats9(actual, expected string) Result { return Floats(actual, expected, 1e-9) }

func Yesno(actual, expected string) Result {
	actualTokens := tokenize(strings.ToUpper(actual))
	expectedTokens := tokenize(strings.ToUpper(expected))

	if len(actualTokens) != len(expectedTokens) {
		return Result{
			Verdict: verdict.WA,
			Message: fmt.Sprintf("Expected %d token(s), found %d", len(expectedTokens), len(actualTokens)),
		}
	}

	normalize := func(token string) string {
		switch token {
		case "Y":
			return "YES"
		case "N":
			return "NO"
		case "YES":
			return "YES"
		case "NO":
			return "NO"
		default:
			return token
		}
	}

	for i := range actualTokens {
		actualNorm := normalize(actualTokens[i])
		expectedNorm := normalize(expectedTokens[i])

		if actualNorm != "YES" && actualNorm != "NO" {
			return Result{
				Verdict: verdict.PE,
				Message: fmt.Sprintf("Token %d: expected YES or NO (or Y/N), found %q", i+1, actualTokens[i]),
			}
		}

		if actualNorm != expectedNorm {
			return Result{
				Verdict: verdict.WA,
				Message: fmt.Sprintf("Token %d: expected %s, found %s", i+1, expectedTokens[i], actualTokens[i]),
			}
		}
	}

	return Result{Verdict: verdict.OK, Message: fmt.Sprintf("%d token(s)", len(actualTokens))}
}

func Strict(actual, expected string) Result {
	if actual == expected {
		return Result{Verdict: verdict.OK, Message: "Full match"}
	}
	return Result{Verdict: verdict.WA, Message: "Output differs (including whitespace)"}
}

func tokenize(s string) []string {
	return strings.Fields(s)
}

func parseIntegers(s string) ([]int64, error) {
	tokens := tokenize(s)
	result := make([]int64, 0, len(tokens))

	for _, token := range tokens {
		num, err := strconv.ParseInt(token, 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, num)
	}

	return result, nil
}

func parseFloats(s string) ([]float64, error) {
	tokens := tokenize(s)
	result := make([]float64, 0, len(tokens))

	for _, token := range tokens {
		num, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, num)
	}

	return result, nil
}

func floatsEqual(a, b, epsilon float64) bool {
	diff := math.Abs(a - b)

	if diff <= epsilon {
		return true
	}

	return diff <= epsilon*math.Max(math.Abs(a), math.Abs(b))
}
