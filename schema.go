package jsonschema

import (
	jsonv1 "encoding/json"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Version is the JSON Schema version.
var Version = "https://json-schema.org/draft/2020-12/schema"

// Schema represents a JSON Schema object type.
// RFC draft-bhutton-json-schema-00 section 4.3
type Schema struct {
	// RFC draft-bhutton-json-schema-00
	Version     string      `json:"$schema,omitzero,omitempty"`     // section 8.1.1
	ID          ID          `json:"$id,omitzero,omitempty"`         // section 8.2.1
	Anchor      string      `json:"$anchor,omitzero,omitempty"`     // section 8.2.2
	Ref         string      `json:"$ref,omitzero,omitempty"`        // section 8.2.3.1
	DynamicRef  string      `json:"$dynamicRef,omitzero,omitempty"` // section 8.2.3.2
	Definitions Definitions `json:"$defs,omitzero,omitempty"`       // section 8.2.4
	Comments    string      `json:"$comment,omitzero,omitempty"`    // section 8.3
	// RFC draft-bhutton-json-schema-00 section 10.2.1 (Sub-schemas with logic)
	AllOf []*Schema `json:"allOf,omitzero,omitempty"` // section 10.2.1.1
	AnyOf []*Schema `json:"anyOf,omitzero,omitempty"` // section 10.2.1.2
	OneOf []*Schema `json:"oneOf,omitzero,omitempty"` // section 10.2.1.3
	Not   *Schema   `json:"not,omitzero,omitempty"`   // section 10.2.1.4
	// RFC draft-bhutton-json-schema-00 section 10.2.2 (Apply sub-schemas conditionally)
	If               *Schema            `json:"if,omitzero,omitempty"`               // section 10.2.2.1
	Then             *Schema            `json:"then,omitzero,omitempty"`             // section 10.2.2.2
	Else             *Schema            `json:"else,omitzero,omitempty"`             // section 10.2.2.3
	DependentSchemas map[string]*Schema `json:"dependentSchemas,omitzero,omitempty"` // section 10.2.2.4
	// RFC draft-bhutton-json-schema-00 section 10.3.1 (arrays)
	PrefixItems []*Schema `json:"prefixItems,omitzero,omitempty"` // section 10.3.1.1
	Items       *Schema   `json:"items,omitzero,omitempty"`       // section 10.3.1.2  (replaces additionalItems)
	Contains    *Schema   `json:"contains,omitzero,omitempty"`    // section 10.3.1.3
	// RFC draft-bhutton-json-schema-00 section 10.3.2 (sub-schemas)
	Properties           *orderedmap.OrderedMap[string, *Schema] `json:"properties"`                              // section 10.3.2.1
	PatternProperties    map[string]*Schema                      `json:"patternProperties,omitzero,omitempty"`    // section 10.3.2.2
	AdditionalProperties *Schema                                 `json:"additionalProperties,omitzero,omitempty"` // section 10.3.2.3
	PropertyNames        *Schema                                 `json:"propertyNames,omitzero,omitempty"`        // section 10.3.2.4

	// Type is the instance data model type (RFC draft-bhutton-json-schema-validation-00, section 6).
	// The keyword in JSON Schema is "type".
	Type string `json:"-"` // section 6.1.1

	// TypeEnhanced is an enhanced description of the instance data model type (RFC draft-bhutton-json-schema-validation-00, section 6).
	// The keyword in JSON Schema is "type".
	// In most cases, you won't need it.
	// You can only set one of Type and TypeEnhanced.
	// When marshalling to JSON Schema string, if TypeEnhanced is not nil, the `type` keyword will be of type Array<String>.
	// When unmarshalling from JSON Schema string, if the `type` keyword is of type Array<String>, it will be unmarshalled into the TypeEnhanced field.
	// Optional.
	TypeEnhanced []string `json:"-"` // section 6.1.1

	Enum              []any               `json:"enum,omitzero,omitempty"`              // section 6.1.2
	Const             any                 `json:"const,omitzero,omitempty"`             // section 6.1.3
	MultipleOf        jsonv1.Number       `json:"multipleOf,omitzero,omitempty"`        // section 6.2.1
	Maximum           jsonv1.Number       `json:"maximum,omitzero,omitempty"`           // section 6.2.2
	ExclusiveMaximum  jsonv1.Number       `json:"exclusiveMaximum,omitzero,omitempty"`  // section 6.2.3
	Minimum           jsonv1.Number       `json:"minimum,omitzero,omitempty"`           // section 6.2.4
	ExclusiveMinimum  jsonv1.Number       `json:"exclusiveMinimum,omitzero,omitempty"`  // section 6.2.5
	MaxLength         *uint64             `json:"maxLength,omitzero,omitempty"`         // section 6.3.1
	MinLength         *uint64             `json:"minLength,omitzero,omitempty"`         // section 6.3.2
	Pattern           string              `json:"pattern,omitzero,omitempty"`           // section 6.3.3
	MaxItems          *uint64             `json:"maxItems,omitzero,omitempty"`          // section 6.4.1
	MinItems          *uint64             `json:"minItems,omitzero,omitempty"`          // section 6.4.2
	UniqueItems       bool                `json:"uniqueItems,omitzero,omitempty"`       // section 6.4.3
	MaxContains       *uint64             `json:"maxContains,omitzero,omitempty"`       // section 6.4.4
	MinContains       *uint64             `json:"minContains,omitzero,omitempty"`       // section 6.4.5
	MaxProperties     *uint64             `json:"maxProperties,omitzero,omitempty"`     // section 6.5.1
	MinProperties     *uint64             `json:"minProperties,omitzero,omitempty"`     // section 6.5.2
	Required          []string            `json:"required,omitzero,omitempty"`          // section 6.5.3
	DependentRequired map[string][]string `json:"dependentRequired,omitzero,omitempty"` // section 6.5.4
	// RFC draft-bhutton-json-schema-validation-00, section 7
	Format string `json:"format,omitzero,omitempty"`
	// RFC draft-bhutton-json-schema-validation-00, section 8
	ContentEncoding  string  `json:"contentEncoding,omitzero,omitempty"`  // section 8.3
	ContentMediaType string  `json:"contentMediaType,omitzero,omitempty"` // section 8.4
	ContentSchema    *Schema `json:"contentSchema,omitzero,omitempty"`    // section 8.5
	// RFC draft-bhutton-json-schema-validation-00, section 9
	Title       string `json:"title,omitzero,omitempty"`       // section 9.1
	Description string `json:"description,omitzero,omitempty"` // section 9.1
	Default     any    `json:"default,omitzero,omitempty"`     // section 9.2
	Deprecated  bool   `json:"deprecated,omitzero,omitempty"`  // section 9.3
	ReadOnly    bool   `json:"readOnly,omitzero,omitempty"`    // section 9.4
	WriteOnly   bool   `json:"writeOnly,omitzero,omitempty"`   // section 9.4
	Examples    []any  `json:"examples,omitzero,omitempty"`    // section 9.5

	Extras map[string]any `json:"-"`

	// Special boolean representation of the Schema - section 4.3.2
	boolean *bool
}

type typeUnion struct {
	Type         string
	TypeEnhanced []string
}

var (
	// TrueSchema defines a schema with a true value
	TrueSchema = &Schema{boolean: &[]bool{true}[0]}
	// FalseSchema defines a schema with a false value
	FalseSchema = &Schema{boolean: &[]bool{false}[0]}
)

// Definitions hold schema definitions.
// http://json-schema.org/latest/json-schema-validation.html#rfc.section.5.26
// RFC draft-wright-json-schema-validation-00, section 5.26
type Definitions map[string]*Schema
