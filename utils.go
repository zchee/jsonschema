package jsonschema

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	matchFirstCap        = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap          = regexp.MustCompile("([a-z0-9])([A-Z])")
	pkgPathCanonicalizer = strings.NewReplacer(
		"github.com/eino-contrib/jsonschema_test", "github.com/invopop/jsonschema_test",
		"github.com/eino-contrib/jsonschema", "github.com/invopop/jsonschema",
	)
)

// ToSnakeCase converts the provided string into snake case using dashes.
// This is useful for Schema IDs and definitions to be coherent with
// common JSON Schema examples.
func ToSnakeCase(str string) string {
	if str == "" {
		return ""
	}

	// ASCII fast path avoids rune decoding and reduces allocations for common identifiers.
	if isASCII(str) {
		var b strings.Builder
		b.Grow(len(str) + len(str)/2 + 1)

		var prevLower, prevDigit bool
		for i := 0; i < len(str); i++ {
			c := str[i]
			currUpper := 'A' <= c && c <= 'Z'
			currDigit := '0' <= c && c <= '9'
			if currUpper {
				nextLower := i+1 < len(str) && ('a' <= str[i+1] && str[i+1] <= 'z')
				if i > 0 && (prevLower || prevDigit || nextLower) {
					b.WriteByte('-')
				}
				c += 'a' - 'A'
			} else if currDigit {
				if i > 0 && prevLower {
					b.WriteByte('-')
				}
			}

			b.WriteByte(c)
			prevLower = 'a' <= c && c <= 'z'
			prevDigit = currDigit
		}

		return b.String()
	}

	var b strings.Builder
	// Heuristic: input length + ~50% headroom for dashes/UTF-8 growth.
	b.Grow(len(str) + len(str)/2 + 1)

	reader := strings.NewReader(str)
	curr, _, _ := reader.ReadRune()
	next, _, nextErr := reader.ReadRune()
	hasNext := nextErr == nil
	var prev rune
	prevValid := false

	for {
		if shouldInsertDashRunes(prevValid, prev, curr, hasNext, next) {
			b.WriteByte('-')
		}
		b.WriteRune(toLowerRune(curr))

		if !hasNext {
			break
		}

		prev, prevValid = curr, true
		curr = next
		next, _, nextErr = reader.ReadRune()
		hasNext = nextErr == nil
	}

	return b.String()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func shouldInsertDashRunes(prevValid bool, prev, curr rune, hasNext bool, next rune) bool {
	if !prevValid {
		return false
	}

	prevLower := isLower(prev)
	prevDigit := isDigit(prev)
	currUpper := isUpper(curr)
	currDigit := isDigit(curr)

	if currUpper {
		nextLower := hasNext && isLower(next)
		return prevLower || prevDigit || nextLower
	}

	if currDigit {
		return prevLower
	}

	return false
}

func isUpper(r rune) bool { return 'A' <= r && r <= 'Z' }

func isLower(r rune) bool { return 'a' <= r && r <= 'z' }

func isDigit(r rune) bool { return '0' <= r && r <= '9' }

func toLowerRune(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	if r <= unicode.MaxASCII {
		return r
	}

	return unicode.ToLower(r)
}

// canonicalPkgPath normalizes known module path aliases to their canonical form.
func canonicalPkgPath(path string) string {
	return pkgPathCanonicalizer.Replace(path)
}

func canonicalizeCommentText(text string) string {
	if text == "" {
		return text
	}
	return pkgPathCanonicalizer.Replace(text)
}
