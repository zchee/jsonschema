package jsonschema

import (
	"regexp"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
var pkgPathCanonicalizer = strings.NewReplacer(
	"github.com/eino-contrib/jsonschema_test", "github.com/invopop/jsonschema_test",
	"github.com/eino-contrib/jsonschema", "github.com/invopop/jsonschema",
)

// ToSnakeCase converts the provided string into snake case using dashes.
// This is useful for Schema IDs and definitions to be coherent with
// common JSON Schema examples.
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
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
