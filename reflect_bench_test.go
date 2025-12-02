package jsonschema

import "testing"

func BenchmarkReflectTestUser(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(TestUser{})
	}
}

func BenchmarkReflectEmbedded(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectDoNotReference(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{DoNotReference: true}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectExpandedStruct(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{ExpandedStruct: true}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectWithComments(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{
		CommentMap: map[string]string{
			"github.com/invopop/jsonschema.Outer":           "outer comment",
			"github.com/invopop/jsonschema.Outer.Text":      "text comment",
			"github.com/invopop/jsonschema.Outer.Inner":     "inner comment",
			"github.com/invopop/jsonschema.Inner.Foo":       "foo comment",
			"github.com/invopop/jsonschema.Text":            "text type comment",
			"github.com/invopop/jsonschema.TextNamed":       "text named comment",
			"github.com/invopop/jsonschema.TextNamed.Field": "unused",
		},
	}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(Outer{})
	}
}
