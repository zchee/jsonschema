package jsonschema

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/eino-contrib/jsonschema/examples"
	"github.com/stretchr/testify/require"
)

func TestCommentsSchemaGeneration(t *testing.T) {
	tests := []struct {
		typ       any
		reflector *Reflector
		fixture   string
	}{
		{
			typ:       &examples.User{},
			reflector: prepareCommentReflector(t),
			fixture:   "fixtures/go_comments.json",
		},
		{
			typ:       &examples.User{},
			reflector: prepareCommentReflector(t, WithFullComment()),
			fixture:   "fixtures/go_comments_full.json",
		},
		{
			typ:       &examples.User{},
			reflector: prepareCustomCommentReflector(t),
			fixture:   "fixtures/custom_comments.json",
		},
	}
	for _, tt := range tests {
		name := strings.TrimSuffix(filepath.Base(tt.fixture), ".json")
		t.Run(name, func(t *testing.T) {
			compareSchemaOutput(t,
				tt.fixture, tt.reflector, tt.typ,
			)
		})
	}
}

func prepareCommentReflector(t *testing.T, opts ...CommentOption) *Reflector {
	t.Helper()
	r := new(Reflector)
	err := r.AddGoComments("github.com/invopop/jsonschema", "./examples", opts...)
	require.NoError(t, err, "did not expect error while adding comments")
	return r
}

func prepareCustomCommentReflector(t *testing.T) *Reflector {
	t.Helper()
	r := new(Reflector)
	r.LookupComment = func(t reflect.Type, f string) string {
		if t != reflect.TypeFor[examples.User]() {
			// To test the interaction between a custom LookupComment function and the
			// AddGoComments function, we only override comments for the User type.
			return ""
		}
		if f == "" {
			return fmt.Sprintf("Go type %s, defined in package %s.", t.Name(), t.PkgPath())
		}
		return fmt.Sprintf("Field %s of Go type %s.%s.", f, t.PkgPath(), t.Name())
	}
	// Also add the Go comments.
	err := r.AddGoComments("github.com/invopop/jsonschema", "./examples")
	require.NoError(t, err, "did not expect error while adding comments")
	return r
}
