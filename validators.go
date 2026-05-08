package termactions

import (
	"fmt"
	"math"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// --- Text validators -------------------------------------
// These validators work with both [Text] and [MultilineText] prompts.
// They accept func(string) (string, bool) matching the WithValidator signature.

// ValidateTextChain combines multiple text validators into one.
// Validators are run in order — the first failure stops the chain.
// If no validators are provided, the chain will always pass.
//
//	termactions.Text().WithValidator(termactions.ValidateTextChain(
//	    termactions.ValidateTextRequired(),
//	    termactions.ValidateTextMinMaxLength(3, 50),
//	    termactions.ValidateTextASCIIAlphanumeric(),
//	))
func ValidateTextChain(validators ...func(string) (string, bool)) func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, v := range validators {
			if msg, ok := v(s); !ok {
				return msg, false
			}
		}
		return "", true
	}
}

// ValidateTextRequired fails if the input is empty or whitespace only.
func ValidateTextRequired() func(string) (string, bool) {
	return func(s string) (string, bool) {
		if strings.TrimSpace(s) == "" {
			return "required", false
		}
		return "", true
	}
}

// ValidateTextMinLength fails if the input is shorter than n characters.
// Length is measured in Unicode code points, not bytes.
// If n is zero, the validator will always pass.
func ValidateTextMinLength(n int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if len([]rune(s)) < n {
			return fmt.Sprintf("must be at least %d characters", n), false
		}
		return "", true
	}
}

// ValidateTextMaxLength fails if the input is longer than n characters.
// Length is measured in Unicode code points, not bytes.
// If n is zero, the validator will pass on empty input and fail otherwise.
func ValidateTextMaxLength(n int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if len([]rune(s)) > n {
			return fmt.Sprintf("must be at most %d characters", n), false
		}
		return "", true
	}
}

// ValidateTextMinMaxLength fails if the input length is outside the range [min, max].
// Length is measured in Unicode code points, not bytes.
// If min is greater than max, the validator will never pass.
func ValidateTextMinMaxLength(min, max int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		l := len([]rune(s))
		if l < min || l > max {
			return fmt.Sprintf("must be %d–%d characters", min, max), false
		}
		return "", true
	}
}

// ValidateTextEmail fails if the input is not a valid email address.
// Validation is based on RFC 5322 and accepts formats like "Name <email@example.com>".
// For strict address-only input, ensure the label does not prompt for display names.
func ValidateTextEmail() func(string) (string, bool) {
	return func(s string) (string, bool) {
		if _, err := mail.ParseAddress(s); err != nil {
			return "must be a valid email address", false
		}
		return "", true
	}
}

// ValidateTextURL fails if the input is not a valid URL with a scheme and host.
// Both scheme and host are required, e.g. "https://example.com".
// Relative URLs and schemeless inputs like "example.com" are rejected.
func ValidateTextURL() func(string) (string, bool) {
	return func(s string) (string, bool) {
		u, err := url.ParseRequestURI(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "must be a valid URL", false
		}
		return "", true
	}
}

// ValidateTextASCIINumeric fails if the input contains any non-digit characters.
// Only ASCII digits 0–9 are accepted.
func ValidateTextASCIINumeric() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if r < '0' || r > '9' {
				return "must contain digits only", false
			}
		}
		return "", true
	}
}

// ValidateTextASCIIAlphanumeric fails if the input contains any non-alphanumeric characters.
// Only ASCII letters a–z, A–Z and digits 0–9 are accepted. Accented and non-ASCII letters are rejected.
func ValidateTextASCIIAlphanumeric() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				return "must contain letters and digits only", false
			}
		}
		return "", true
	}
}

// ValidateTextUnicodeNumeric fails if the input contains any non-digit characters.
// Accepts digits from all Unicode scripts, not just ASCII 0–9.
func ValidateTextUnicodeNumeric() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return "must contain digits only", false
			}
		}
		return "", true
	}
}

// ValidateTextUnicodeAlphanumeric fails if the input contains any non-alphanumeric characters.
// Accepts letters and digits from all Unicode scripts including accented characters.
func ValidateTextUnicodeAlphanumeric() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return "must contain letters and digits only", false
			}
		}
		return "", true
	}
}

// ValidateTextRegex fails if the input does not match the given regular expression.
// msg is the error message shown to the user on failure.
// Panics if pattern is not a valid regular expression.
func ValidateTextRegex(pattern, msg string) func(string) (string, bool) {
	re := regexp.MustCompile(pattern)
	return func(s string) (string, bool) {
		if !re.MatchString(s) {
			return msg, false
		}
		return "", true
	}
}

// ValidateTextMin fails if the input, parsed as a number, is less than n.
// Non-numeric input is also rejected.
func ValidateTextMin(n float64) func(string) (string, bool) {
	return func(s string) (string, bool) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return "must be a number", false
		}
		if v < n {
			return fmt.Sprintf("must be at least %s", formatFloat(n)), false
		}
		return "", true
	}
}

// ValidateTextMax fails if the input, parsed as a number, is greater than n.
// Non-numeric input is also rejected.
func ValidateTextMax(n float64) func(string) (string, bool) {
	return func(s string) (string, bool) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return "must be a number", false
		}
		if v > n {
			return fmt.Sprintf("must be at most %s", formatFloat(n)), false
		}
		return "", true
	}
}

// ValidateTextMinMax fails if the input, parsed as a number, is outside the range [min, max].
// If min is greater than max, the validator will never pass.
func ValidateTextMinMax(min, max float64) func(string) (string, bool) {
	return func(s string) (string, bool) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return "must be a number", false
		}
		if v < min || v > max {
			return fmt.Sprintf("must be between %s and %s", formatFloat(min), formatFloat(max)), false
		}
		return "", true
	}
}

// ValidateTextIPAddr fails if the input is not a valid IPv4 or IPv6 address.
// Both formats are accepted, e.g. "192.168.1.1" or "::1".
func ValidateTextIPAddr() func(string) (string, bool) {
	return func(s string) (string, bool) {
		if net.ParseIP(s) == nil {
			return "must be a valid IP address", false
		}
		return "", true
	}
}

// ValidateTextPortNumber fails if the input is not a valid port number.
// Valid range is 1–65535; port 0 is reserved and not accepted.
func ValidateTextPortNumber() func(string) (string, bool) {
	return func(s string) (string, bool) {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 65535 {
			return "must be a valid port number (1–65535)", false
		}
		return "", true
	}
}

// ValidateTextNoSpaces fails if the input contains any space characters ( ).
func ValidateTextNoSpaces() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if r == ' ' {
				return "must not contain spaces", false
			}
		}
		return "", true
	}
}

// ValidateTextNoWhitespace fails if the input contains any whitespace characters
// including spaces, tabs, newlines, and other Unicode whitespace.
func ValidateTextNoWhitespace() func(string) (string, bool) {
	return func(s string) (string, bool) {
		for _, r := range s {
			if unicode.IsSpace(r) {
				return "must not contain whitespace", false
			}
		}
		return "", true
	}
}

// ValidateTextStartsWith fails if the input does not start with the given prefix.
// If prefix is empty, the validator will always pass.
func ValidateTextStartsWith(prefix string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if !strings.HasPrefix(s, prefix) {
			return fmt.Sprintf("must start with %q", prefix), false
		}
		return "", true
	}
}

// ValidateTextEndsWith fails if the input does not end with the given suffix.
// If suffix is empty, the validator will always pass.
func ValidateTextEndsWith(suffix string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if !strings.HasSuffix(s, suffix) {
			return fmt.Sprintf("must end with %q", suffix), false
		}
		return "", true
	}
}

// ValidateTextOneOf fails if the input does not exactly match one of the allowed values.
// If options is empty, the validator will never pass.
func ValidateTextOneOf(options ...string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if slices.Contains(options, s) {
			return "", true
		}
		return fmt.Sprintf("must be one of: %s", strings.Join(options, ", ")), false
	}
}

// ValidateTextMatches fails if the input does not match the value pointed to by other.
// Useful for confirm-style prompts such as password confirmation.
//
//	pass, _ := termactions.Secret().WithLabel("Password").Render()
//	termactions.Secret().
//	    WithLabel("Confirm password").
//	    WithValidator(termactions.ValidateTextMatches(&pass)).
//	    Render()
func ValidateTextMatches(other *string) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if s != *other {
			return "does not match", false
		}
		return "", true
	}
}

// --- MultilineText validators ----------------------------
// These validators are specific to [MultilineText] prompts.
// For general text validators (required, length, regex, etc.)
// use the ValidateText* functions above — they work with both
// [Text] and [MultilineText].

// ValidateMultilineTextMinLines fails if the input has fewer than n lines.
// Empty input is treated as zero lines.
// If n is zero, the validator will always pass.
func ValidateMultilineTextMinLines(n int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if strings.TrimSpace(s) == "" {
			if n > 0 {
				return fmt.Sprintf("must be at least %d %s", n, pluralLine(n)), false
			}
			return "", true
		}
		count := strings.Count(s, "\n") + 1
		if count < n {
			return fmt.Sprintf("must be at least %d %s", n, pluralLine(n)), false
		}
		return "", true
	}
}

// ValidateMultilineTextMaxLines fails if the input has more than n lines.
// Empty input is treated as zero lines.
// If n is zero, the validator will pass on empty input and fail otherwise.
func ValidateMultilineTextMaxLines(n int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		if strings.TrimSpace(s) == "" {
			return "", true
		}
		count := strings.Count(s, "\n") + 1
		if count > n {
			return fmt.Sprintf("must be at most %d %s", n, pluralLine(n)), false
		}
		return "", true
	}
}

// ValidateMultilineTextMinMaxLines fails if the line count is outside the range [min, max].
// Empty input is treated as zero lines.
// min must not be greater than max; if it is, the validator will never pass.
func ValidateMultilineTextMinMaxLines(min, max int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		count := 0
		if strings.TrimSpace(s) != "" {
			count = strings.Count(s, "\n") + 1
		}
		if count < min || count > max {
			return fmt.Sprintf("must be %d–%d %s", min, max, pluralLine(max)), false
		}
		return "", true
	}
}

// --- Select validators ------------------------------------

// ValidateSelectRequired fails if no choice has been made (zero value Choice).
func ValidateSelectRequired() func(Choice) (string, bool) {
	return func(c Choice) (string, bool) {
		if c == (Choice{}) {
			return "selection required", false
		}
		return "", true
	}
}

// --- MultiSelect validators -------------------------------

// ValidateMultiSelectChain combines multiple MultiSelect validators into one.
// Validators are run in order — the first failure stops the chain.
// If no validators are provided, the chain will always pass.
//
//	termactions.MultiSelect().WithValidator(termactions.ValidateMultiSelectChain(
//	    termactions.ValidateMultiSelectRequired(),
//	    termactions.ValidateMultiSelectMinMax(1, 3),
//	))
func ValidateMultiSelectChain(validators ...func([]Choice) (string, bool)) func([]Choice) (string, bool) {
	return func(choices []Choice) (string, bool) {
		for _, v := range validators {
			if msg, ok := v(choices); !ok {
				return msg, false
			}
		}
		return "", true
	}
}

// ValidateMultiSelectRequired fails if no choices have been selected.
func ValidateMultiSelectRequired() func([]Choice) (string, bool) {
	return func(choices []Choice) (string, bool) {
		if len(choices) == 0 {
			return "selection required", false
		}
		return "", true
	}
}

// ValidateMultiSelectMin fails if fewer than n choices are selected.
// If n exceeds the number of available choices, the validator will never pass.
// If n is zero, the validator will always pass.
func ValidateMultiSelectMin(n int) func([]Choice) (string, bool) {
	return func(choices []Choice) (string, bool) {
		if len(choices) < n {
			return fmt.Sprintf("select at least %d %s", n, pluralChoice(n)), false
		}
		return "", true
	}
}

// ValidateMultiSelectMax fails if more than n choices are selected.
// If n exceeds the number of available choices, the validator will never fail.
// If n is zero, the validator will pass on empty selection and fail otherwise.
func ValidateMultiSelectMax(n int) func([]Choice) (string, bool) {
	return func(choices []Choice) (string, bool) {
		if len(choices) > n {
			return fmt.Sprintf("select at most %d %s", n, pluralChoice(n)), false
		}
		return "", true
	}
}

// ValidateMultiSelectMinMax fails if the number of selected choices is outside [min, max].
// min must not be greater than max; if it is, the validator will never pass.
func ValidateMultiSelectMinMax(min, max int) func([]Choice) (string, bool) {
	return func(choices []Choice) (string, bool) {
		n := len(choices)
		if n < min || n > max {
			return fmt.Sprintf("select between %d and %d %s", min, max, pluralChoice(max)), false
		}
		return "", true
	}
}

// --- helpers ---------------------------------------------

// formatFloat formats a float64 as an integer string if it has no fractional
// part, otherwise as a decimal string.
func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return strconv.Itoa(int(f))
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// pluralChoice returns "choice" or "choices" based on the given number.
func pluralChoice(n int) string {
	if n == 1 {
		return "choice"
	}
	return "choices"
}

// pluralLine returns "line" or "lines" based on the given number.
func pluralLine(n int) string {
	if n == 1 {
		return "line"
	}
	return "lines"
}
