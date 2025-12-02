package jsonschema

import "testing"

func BenchmarkReflectTestUser(b *testing.B) {
	b.ReportAllocs()
	r := &Reflector{}
	for i := 0; i < b.N; i++ {
		_ = r.Reflect(TestUser{})
	}
}
