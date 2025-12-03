package jsonschema

import (
	"reflect"
	"testing"
)

type wideStruct struct {
	F01 string `json:"f01"`
	F02 string `json:"f02"`
	F03 string `json:"f03"`
	F04 string `json:"f04"`
	F05 string `json:"f05"`
	F06 string `json:"f06"`
	F07 string `json:"f07"`
	F08 string `json:"f08"`
	F09 string `json:"f09"`
	F10 string `json:"f10"`
	F11 string `json:"f11"`
	F12 string `json:"f12"`
	F13 string `json:"f13"`
	F14 string `json:"f14"`
	F15 string `json:"f15"`
	F16 string `json:"f16"`
	F17 string `json:"f17"`
	F18 string `json:"f18"`
	F19 string `json:"f19"`
	F20 string `json:"f20"`
	F21 string `json:"f21"`
	F22 string `json:"f22"`
	F23 string `json:"f23"`
	F24 string `json:"f24"`
	F25 string `json:"f25"`
	F26 string `json:"f26"`
	F27 string `json:"f27"`
	F28 string `json:"f28"`
	F29 string `json:"f29"`
	F30 string `json:"f30"`
	F31 string `json:"f31"`
	F32 string `json:"f32"`
}

func BenchmarkReflectTestUser(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	for b.Loop() {
		_ = r.Reflect(TestUser{})
	}
}

func BenchmarkReflectEmbedded(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	for b.Loop() {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectDoNotReference(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{DoNotReference: true}
	for b.Loop() {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectExpandedStruct(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{ExpandedStruct: true}
	for b.Loop() {
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
	for b.Loop() {
		_ = r.Reflect(Outer{})
	}
}

func BenchmarkReflectWideStruct(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	ws := wideStruct{}
	for b.Loop() {
		_ = r.Reflect(ws)
	}
}

func BenchmarkReflectRepeatedType(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	t := reflect.TypeOf(TestUser{})
	for b.Loop() {
		_ = r.ReflectFromType(t)
	}
}
