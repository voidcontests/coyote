package judge

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/voidcontests/coyote/internal/domain/verdict"
)

type Result struct {
	Verdict string // OK, WA or PE
	Message string
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

func Integer(actual, expected string) Result {
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

	for i := range actualTokens {
		actual := actualTokens[i]
		expected := expectedTokens[i]

		if actual != "YES" && actual != "NO" {
			return Result{Verdict: verdict.PE, Message: fmt.Sprintf("Token %d: expected YES or NO, found %q", i+1, actual)}
		}

		if actual != expected {
			return Result{
				Verdict: verdict.WA,
				Message: fmt.Sprintf("Token %d: expected %s, found %s", i+1, expected, actual),
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
