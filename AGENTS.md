# AGENTS.md

Contributor guide for the `jsonschema` Go library — a JSON Schema generator from Go types via reflection.

## Project Structure

```
.
├── *.go              # Core library source files
├── *_test.go         # Test files alongside source
├── fixtures/         # JSON test fixtures for schema comparison
├── examples/         # Example Go types for testing/documentation
├── vendor/           # Vendored dependencies
└── .github/workflows # CI/CD workflows (lint, test, release)
```

Key files:
- `schema.go` — Schema type definition (JSON Schema Draft 2020-12)
- `reflect.go` — Core reflection logic for type-to-schema conversion
- `reflect_comments.go` — Go comment extraction for descriptions
- `id.go` — Schema ID generation utilities

## Build, Test, and Development Commands

```bash
# Run all tests with race detection
go test -race ./...

# Run tests and update fixtures (when schema output changes intentionally)
go test -update ./...

# Run linter (matches CI)
golangci-lint run

# Download dependencies
go mod download
```

## Coding Style & Naming Conventions

- **Go version**: 1.18+ required (generics used)
- **Formatting**: `gofmt` and `goimports` enforced
- **Type aliases**: Use `any` instead of `interface{}`
- **Linting**: golangci-lint v1.62 with config in `.golangci.yml`
  - Key linters: `govet`, `errcheck`, `goimports`, `revive`, `gocyclo`, `dupl`
  - Cyclomatic complexity threshold: 20
  - Duplicate code threshold: 100 lines
- **Naming**: Follow standard Go conventions; struct tags use `json:` and `jsonschema:`

## Testing Guidelines

- **Framework**: `github.com/stretchr/testify` (assert/require)
- **Pattern**: Fixture-based — expected JSON schemas stored in `fixtures/*.json`
- **Updating fixtures**: Run `go test -update` when intentional schema changes occur
- **Coverage**: Race detection enabled; coverage uploaded to Codecov
- **Test flags**:
  - `-update` — regenerate fixture files
  - `-compare` — write failed outputs to `.out.json` for debugging

Example test pattern:
```go
func TestFeature(t *testing.T) {
    r := new(Reflector)
    schema := r.Reflect(&MyType{})
    compareSchemaWithFixture(t, "my_feature.json", schema)
}
```

## Commit & Pull Request Guidelines

- **Commit style**: Conventional Commits
  - `feat:` — new features
  - `fix:` — bug fixes
  - `docs:` — documentation changes
- **Message format**: Short imperative summary (e.g., `feat: add TypeEnhanced support`)
- **PR requirements**:
  - All CI checks must pass (lint + test)
  - Update/add fixtures if schema output changes
  - Include test coverage for new functionality
