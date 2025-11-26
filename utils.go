package jsonschema

import (
	"regexp"
	"strings"
	"unicode"

	orderedmap "github.com/wk8/go-ordered-map/v2"
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

	runes := []rune(str)
	var b strings.Builder
	b.Grow(len(str) + len(runes))

	for i, r := range runes {
		if shouldInsertDash(runes, i) {
			b.WriteByte('-')
		}
		b.WriteRune(toLowerRune(r))
	}

	return b.String()
}

func shouldInsertDash(runes []rune, idx int) bool {
	if idx == 0 {
		return false
	}

	prev := runes[idx-1]
	curr := runes[idx]

	prevLower := isLower(prev)
	prevDigit := isDigit(prev)
	currUpper := isUpper(curr)
	currDigit := isDigit(curr)

	if currUpper {
		nextLower := idx+1 < len(runes) && isLower(runes[idx+1])
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

// NewProperties is a helper method to instantiate a new properties ordered
// map.
func NewProperties() *orderedmap.OrderedMap[string, *Schema] {
	return orderedmap.New[string, *Schema]()
}
