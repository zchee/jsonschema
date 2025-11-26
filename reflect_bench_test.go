package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/eino-contrib/jsonschema/examples"
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reflector.Reflect(user)
	}
}

func BenchmarkSplitOnUnescapedCommas(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitOnUnescapedCommas(benchTagString)
	}
}

func BenchmarkReflectHeavyTags(b *testing.B) {
	reflector := &Reflector{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reflector.Reflect(&heavyTagged{})
	}
}

func BenchmarkToSnakeCase(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToSnakeCase(benchNames[i%len(benchNames)])
	}
}

func BenchmarkMarshalSchema(b *testing.B) {
	reflector := &Reflector{}
	schema := reflector.Reflect(&heavyTagged{})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(schema); err != nil {
			b.Fatal(err)
		}
	}
}
