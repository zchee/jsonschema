package jsonschema

import (
	"testing"

	"github.com/eino-contrib/jsonschema/examples"
)

// Benchmarks target the hottest reflection helpers so we can measure parsing
// and name mangling changes precisely.

var (
	benchTagString = "required,minLength=1,maxLength=20,pattern=^foo\\,bar$,,example=joe,example=lucy,default=alex"
	benchNames     = []string{"SimpleHTTPServer", "HTTPRequestHandler", "IPAddress", "UserProfile", "GeoLocation"}
)

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

func BenchmarkToSnakeCase(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToSnakeCase(benchNames[i%len(benchNames)])
	}
}
