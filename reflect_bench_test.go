package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/zchee/jsonschema/examples"
)

// Benchmarks target the hottest reflection helpers so we can measure parsing
// and name mangling changes precisely.

var (
	benchTagString = "required,minLength=1,maxLength=20,pattern=^foo\\,bar$,,example=joe,example=lucy,default=alex"
	benchNames     = []string{"SimpleHTTPServer", "HTTPRequestHandler", "IPAddress", "UserProfile", "GeoLocation"}
)

type heavyTagged struct {
	Name    string   `json:"name" jsonschema:"title=Name,description=full user name,minLength=1,maxLength=64,pattern=^[a-zA-Z]+$,example=alice,example=bob"`
	Age     int      `json:"age" jsonschema:"minimum=0,maximum=130,default=42,example=18,example=80"`
	Aliases []string `json:"aliases" jsonschema:"minItems=1,uniqueItems=true,default=nick"`
	Scores  []int    `json:"scores" jsonschema:"minItems=1,maxItems=5,default=1,default=2"`
}

func BenchmarkReflectUser(b *testing.B) {
	reflector := &Reflector{}
	user := &examples.User{}
	b.ReportAllocs()

	for b.Loop() {
		_ = reflector.Reflect(user)
	}
}

func BenchmarkSplitOnUnescapedCommas(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		splitOnUnescapedCommas(benchTagString)
	}
}

func BenchmarkReflectHeavyTags(b *testing.B) {
	reflector := &Reflector{}
	b.ReportAllocs()

	for b.Loop() {
		_ = reflector.Reflect(&heavyTagged{})
	}
}

func BenchmarkToSnakeCase(b *testing.B) {
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		ToSnakeCase(benchNames[i%len(benchNames)])
	}
}

func BenchmarkMarshalSchema(b *testing.B) {
	reflector := &Reflector{}
	schema := reflector.Reflect(&heavyTagged{})

	b.ReportAllocs()

	for b.Loop() {
		if _, err := json.Marshal(schema); err != nil {
			b.Fatal(err)
		}
	}
}
