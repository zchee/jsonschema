// Package jsonschema uses reflection to generate JSON Schemas from Go types [1].
//
// If json tags are present on struct fields, they will be used to infer
// property names and if a property is required (omitempty is present).
//
// [1] http://json-schema.org/latest/json-schema-validation.html
package jsonschema

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"maps"
	"net"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
)

// customSchemaImpl is used to detect if the type provides it's own
// custom Schema Type definition to use instead. Very useful for situations
// where there are custom JSON Marshal and Unmarshal methods.
type customSchemaImpl interface {
	JSONSchema() *Schema
}

// Function to be run after the schema has been generated.
// this will let you modify a schema afterwards
type extendSchemaImpl interface {
	JSONSchemaExtend(*Schema)
}

// If the object to be reflected defines a `JSONSchemaAlias` method, its type will
// be used instead of the original type.
type aliasSchemaImpl interface {
	JSONSchemaAlias() any
}

// If an object to be reflected defines a `JSONSchemaPropertyAlias` method,
// it will be called for each property to determine if another object
// should be used for the contents.
type propertyAliasSchemaImpl interface {
	JSONSchemaProperty(prop string) any
}

var customAliasSchema = reflect.TypeFor[aliasSchemaImpl]()
var customPropertyAliasSchema = reflect.TypeFor[propertyAliasSchemaImpl]()

var customType = reflect.TypeFor[customSchemaImpl]()
var extendType = reflect.TypeFor[extendSchemaImpl]()

// customSchemaGetFieldDocString
type customSchemaGetFieldDocString interface {
	GetFieldDocString(fieldName string) string
}

type customGetFieldDocString func(fieldName string) string

var customStructGetFieldDocString = reflect.TypeFor[customSchemaGetFieldDocString]()

// Reflect reflects to Schema from a value using the default Reflector
func Reflect(v any) *Schema {
	return ReflectFromType(reflect.TypeOf(v))
}

// ReflectFromType generates root schema using the default Reflector
func ReflectFromType(t reflect.Type) *Schema {
	r := &Reflector{}
	return r.ReflectFromType(t)
}

// A Reflector reflects values into a Schema.
type Reflector struct {
	// BaseSchemaID defines the URI that will be used as a base to determine Schema
	// IDs for models. For example, a base Schema ID of `https://invopop.com/schemas`
	// when defined with a struct called `User{}`, will result in a schema with an
	// ID set to `https://invopop.com/schemas/user`.
	//
	// If no `BaseSchemaID` is provided, we'll take the type's complete package path
	// and use that as a base instead. Set `Anonymous` to try if you do not want to
	// include a schema ID.
	BaseSchemaID ID

	// Anonymous when true will hide the auto-generated Schema ID and provide what is
	// known as an "anonymous schema". As a rule, this is not recommended.
	Anonymous bool

	// AssignAnchor when true will use the original struct's name as an anchor inside
	// every definition, including the root schema. These can be useful for having a
	// reference to the original struct's name in CamelCase instead of the snake-case used
	// by default for URI compatibility.
	//
	// Anchors do not appear to be widely used out in the wild, so at this time the
	// anchors themselves will not be used inside generated schema.
	AssignAnchor bool

	// AllowAdditionalProperties will cause the Reflector to generate a schema
	// without additionalProperties set to 'false' for all struct types. This means
	// the presence of additional keys in JSON objects will not cause validation
	// to fail. Note said additional keys will simply be dropped when the
	// validated JSON is unmarshaled.
	AllowAdditionalProperties bool

	// RequiredFromJSONSchemaTags will cause the Reflector to generate a schema
	// that requires any key tagged with `jsonschema:required`, overriding the
	// default of requiring any key *not* tagged with `json:,omitempty`.
	RequiredFromJSONSchemaTags bool

	// Do not reference definitions. This will remove the top-level $defs map and
	// instead cause the entire structure of types to be output in one tree. The
	// list of type definitions (`$defs`) will not be included.
	DoNotReference bool

	// ExpandedStruct when true will include the reflected type's definition in the
	// root as opposed to a definition with a reference.
	ExpandedStruct bool

	// FieldNameTag will change the tag used to get field names. json tags are used by default.
	FieldNameTag string

	// IgnoredTypes defines a slice of types that should be ignored in the schema,
	// switching to just allowing additional properties instead.
	IgnoredTypes []any

	// Lookup allows a function to be defined that will provide a custom mapping of
	// types to Schema IDs. This allows existing schema documents to be referenced
	// by their ID instead of being embedded into the current schema definitions.
	// Reflected types will never be pointers, only underlying elements.
	Lookup func(reflect.Type) ID

	// Mapper is a function that can be used to map custom Go types to jsonschema schemas.
	Mapper func(reflect.Type) *Schema

	// Namer allows customizing of type names. The default is to use the type's name
	// provided by the reflect package.
	Namer func(reflect.Type) string

	// KeyNamer allows customizing of key names.
	// The default is to use the key's name as is, or the json tag if present.
	// If a json tag is present, KeyNamer will receive the tag's name as an argument, not the original key name.
	KeyNamer func(string) string

	// AdditionalFields allows adding structfields for a given type
	AdditionalFields func(reflect.Type) []reflect.StructField

	// LookupComment allows customizing comment lookup. Given a reflect.Type and optionally
	// a field name, it should return the comment string associated with this type or field.
	//
	// If the field name is empty, it should return the type's comment; otherwise, the field's
	// comment should be returned. If no comment is found, an empty string should be returned.
	//
	// When set, this function is called before the below CommentMap lookup mechanism. However,
	// if it returns an empty string, the CommentMap is still consulted.
	LookupComment func(reflect.Type, string) string

	// CommentMap is a dictionary of fully qualified go types and fields to comment
	// strings that will be used if a description has not already been provided in
	// the tags. Types and fields are added to the package path using "." as a
	// separator.
	//
	// Type descriptions should be defined like:
	//
	//   map[string]string{"github.com/invopop/jsonschema.Reflector": "A Reflector reflects values into a Schema."}
	//
	// And Fields defined as:
	//
	//   map[string]string{"github.com/invopop/jsonschema.Reflector.DoNotReference": "Do not reference definitions."}
	//
	// See also: AddGoComments, LookupComment
	CommentMap map[string]string

	// EnableSchemaCache toggles per-Reflector schema memoization. When true,
	// schemas are cached per type and reflector configuration fingerprint, and
	// each call returns a cloned copy to avoid mutation sharing. Default is
	// false to preserve existing behaviour.
	EnableSchemaCache bool

	// MaxSchemaCacheEntries bounds the schema cache size when EnableSchemaCache
	// is true. Zero means unbounded (current behaviour). A small positive value
	// limits memory growth at the cost of occasional cache misses (FIFO
	// eviction).
	MaxSchemaCacheEntries int

	schemaCache *xsync.MapOf[schemaCacheKey, *Schema]

	schemaCacheOrder []schemaCacheKey
	schemaCacheMu    sync.Mutex
}

// Reflect reflects to Schema from a value.
func (r *Reflector) Reflect(v any) *Schema {
	return r.ReflectFromType(reflect.TypeOf(v))
}

// ReflectFromType generates root schema
func (r *Reflector) ReflectFromType(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem() // re-assign from pointer
	}

	if r.EnableSchemaCache {
		if cached := r.loadSchemaFromCache(t); cached != nil {
			return cached
		}
	}

	name := r.typeName(t)

	s := new(Schema)
	definitions := Definitions{}
	s.Definitions = definitions
	bs := r.reflectTypeToSchemaWithID(definitions, t)
	if r.ExpandedStruct {
		*s = *definitions[name]
		delete(definitions, name)
	} else {
		*s = *bs
	}

	// Attempt to set the schema ID
	if !r.Anonymous && s.ID == EmptyID {
		baseSchemaID := r.BaseSchemaID
		if baseSchemaID == EmptyID {
			id := ID("https://" + t.PkgPath())
			if err := id.Validate(); err == nil {
				// it's okay to silently ignore URL errors
				baseSchemaID = id
			}
		}
		if baseSchemaID != EmptyID {
			s.ID = baseSchemaID.Add(ToSnakeCase(name))
		}
	}

	s.Version = Version
	if !r.DoNotReference {
		s.Definitions = definitions
	}

	if r.EnableSchemaCache {
		r.storeSchemaInCache(t, s)
	}

	return s
}

// Available Go defined types for JSON Schema Validation.
// RFC draft-wright-json-schema-validation-00, section 7.3
var (
	timeType = reflect.TypeFor[time.Time]() // date-time RFC section 7.3.1
	ipType   = reflect.TypeFor[net.IP]()    // ipv4 and ipv6 RFC section 7.3.4, 7.3.5
	uriType  = reflect.TypeFor[url.URL]()   // uri RFC section 7.3.6
)

// Byte slices will be encoded as base64
var byteSliceType = reflect.TypeFor[[]byte]()

// Except for json.RawMessage
var rawMessageType = reflect.TypeFor[json.RawMessage]()

// Go code generated from protobuf enum types should fulfil this interface.
type protoEnum interface {
	EnumDescriptor() ([]byte, []int)
}

var protoEnumType = reflect.TypeFor[protoEnum]()

// SetBaseSchemaID is a helper use to be able to set the reflectors base
// schema ID from a string as opposed to then ID instance.
func (r *Reflector) SetBaseSchemaID(id string) {
	r.BaseSchemaID = ID(id)
}

func (r *Reflector) refOrReflectTypeToSchema(definitions Definitions, t reflect.Type) *Schema {
	id := r.lookupID(t)
	if id != EmptyID {
		return &Schema{
			Ref: id.String(),
		}
	}

	// Already added to definitions?
	if def := r.refDefinition(definitions, t); def != nil {
		return def
	}

	return r.reflectTypeToSchemaWithID(definitions, t)
}

func (r *Reflector) reflectTypeToSchemaWithID(defs Definitions, t reflect.Type) *Schema {
	s := r.reflectTypeToSchema(defs, t)
	if s != nil {
		if r.Lookup != nil {
			id := r.Lookup(t)
			if id != EmptyID {
				s.ID = id
			}
		}
	}
	return s
}

func (r *Reflector) reflectTypeToSchema(definitions Definitions, t reflect.Type) *Schema {
	// only try to reflect non-pointers
	if t.Kind() == reflect.Ptr {
		return r.refOrReflectTypeToSchema(definitions, t.Elem())
	}

	// Check if the there is an alias method that provides an object
	// that we should use instead of this one.
	if t.Implements(customAliasSchema) {
		v := reflect.New(t)
		o := v.Interface().(aliasSchemaImpl)
		t = reflect.TypeOf(o.JSONSchemaAlias())
		return r.refOrReflectTypeToSchema(definitions, t)
	}

	// Do any pre-definitions exist?
	if r.Mapper != nil {
		if t := r.Mapper(t); t != nil {
			return t
		}
	}
	if rt := r.reflectCustomSchema(definitions, t); rt != nil {
		return rt
	}

	// Prepare a base to which details can be added
	st := new(Schema)

	// jsonpb will marshal protobuf enum options as either strings or integers.
	// It will unmarshal either.
	if t.Implements(protoEnumType) {
		st.OneOf = []*Schema{
			{Type: "string"},
			{Type: "integer"},
		}
		return st
	}

	// Defined format types for JSON Schema Validation
	// RFC draft-wright-json-schema-validation-00, section 7.3
	// TODO email RFC section 7.3.2, hostname RFC section 7.3.3, uriref RFC section 7.3.7
	if t == ipType {
		// TODO differentiate ipv4 and ipv6 RFC section 7.3.4, 7.3.5
		st.Type = "string"
		st.Format = "ipv4"
		return st
	}

	switch t.Kind() {
	case reflect.Struct:
		r.reflectStruct(definitions, t, st)

	case reflect.Slice, reflect.Array:
		r.reflectSliceOrArray(definitions, t, st)

	case reflect.Map:
		r.reflectMap(definitions, t, st)

	case reflect.Interface:
		// empty

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		st.Type = "integer"

	case reflect.Float32, reflect.Float64:
		st.Type = "number"

	case reflect.Bool:
		st.Type = "boolean"

	case reflect.String:
		st.Type = "string"

	default:
		panic("unsupported type " + t.String())
	}

	r.reflectSchemaExtend(definitions, t, st)

	// Always try to reference the definition which may have just been created
	if def := r.refDefinition(definitions, t); def != nil {
		return def
	}

	return st
}

func (r *Reflector) reflectCustomSchema(definitions Definitions, t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		return r.reflectCustomSchema(definitions, t.Elem())
	}

	if t.Implements(customType) {
		v := reflect.New(t)
		o := v.Interface().(customSchemaImpl)
		st := o.JSONSchema()
		r.addDefinition(definitions, t, st)
		if ref := r.refDefinition(definitions, t); ref != nil {
			return ref
		}
		return st
	}

	return nil
}

func (r *Reflector) reflectSchemaExtend(definitions Definitions, t reflect.Type, s *Schema) *Schema {
	if t.Implements(extendType) {
		v := reflect.New(t)
		o := v.Interface().(extendSchemaImpl)
		o.JSONSchemaExtend(s)
		if ref := r.refDefinition(definitions, t); ref != nil {
			return ref
		}
	}

	return s
}

func (r *Reflector) reflectSliceOrArray(definitions Definitions, t reflect.Type, st *Schema) {
	if t == rawMessageType {
		return
	}

	r.addDefinition(definitions, t, st)

	if st.Description == "" {
		st.Description = r.lookupComment(t, "")
	}

	if t.Kind() == reflect.Array {
		l := uint64(t.Len())
		st.MinItems = &l
		st.MaxItems = &l
	}
	if t.Kind() == reflect.Slice && t.Elem() == byteSliceType.Elem() {
		st.Type = "string"
		// NOTE: ContentMediaType is not set here
		st.ContentEncoding = "base64"
	} else {
		st.Type = "array"
		st.Items = r.refOrReflectTypeToSchema(definitions, t.Elem())
	}
}

func (r *Reflector) reflectMap(definitions Definitions, t reflect.Type, st *Schema) {
	r.addDefinition(definitions, t, st)

	st.Type = "object"
	if st.Description == "" {
		st.Description = r.lookupComment(t, "")
	}

	switch t.Key().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		st.PatternProperties = map[string]*Schema{
			"^[0-9]+$": r.refOrReflectTypeToSchema(definitions, t.Elem()),
		}
		st.AdditionalProperties = FalseSchema
		return
	}
	if t.Elem().Kind() != reflect.Interface {
		st.AdditionalProperties = r.refOrReflectTypeToSchema(definitions, t.Elem())
	}
}

// Reflects a struct to a JSON Schema type.
func (r *Reflector) reflectStruct(definitions Definitions, t reflect.Type, s *Schema) {
	// Handle special types
	switch t {
	case timeType: // date-time RFC section 7.3.1
		s.Type = "string"
		s.Format = "date-time"
		return
	case uriType: // uri RFC section 7.3.6
		s.Type = "string"
		s.Format = "uri"
		return
	}

	r.addDefinition(definitions, t, s)
	s.Type = "object"
	fieldTagsCache := r.fieldTagsForType(t)
	propCap := r.estimatePropertyCapacity(t, fieldTagsCache, nil)
	s.Properties = NewProperties(propCap)
	s.Description = r.lookupComment(t, "")
	if r.AssignAnchor {
		s.Anchor = t.Name()
	}
	if !r.AllowAdditionalProperties && s.AdditionalProperties == nil {
		s.AdditionalProperties = FalseSchema
	}

	ignored := false
	for _, it := range r.IgnoredTypes {
		if reflect.TypeOf(it) == t {
			ignored = true
			break
		}
	}
	if !ignored {
		propCap := r.estimatePropertyCapacity(t, fieldTagsCache, nil)
		r.reflectStructFields(s, definitions, t, fieldTagsCache, propCap)
	}
}

func (r *Reflector) reflectStructFields(st *Schema, definitions Definitions, t reflect.Type, fieldTagsCache []fieldTags, requiredCap int) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	if st.Required == nil {
		capHint := t.NumField()
		const requiredPreallocThreshold = 10
		if capHint < requiredPreallocThreshold {
			capHint = requiredPreallocThreshold
		}
		if requiredCap > capHint {
			capHint = requiredCap
		}
		st.Required = make([]string, 0, capHint)
	}

	if fieldTagsCache == nil {
		fieldTagsCache = r.fieldTagsForType(t)
	}

	var getFieldDocString customGetFieldDocString
	if t.Implements(customStructGetFieldDocString) {
		v := reflect.New(t)
		o := v.Interface().(customSchemaGetFieldDocString)
		getFieldDocString = o.GetFieldDocString
	}

	customPropertyMethod := func(string) any {
		return nil
	}
	if t.Implements(customPropertyAliasSchema) {
		v := reflect.New(t)
		o := v.Interface().(propertyAliasSchemaImpl)
		customPropertyMethod = o.JSONSchemaProperty
	}

	handleField := func(f reflect.StructField, tags fieldTags) {
		name, shouldEmbed, required, nullable := r.reflectFieldName(f, tags)
		// if anonymous and exported type should be processed recursively
		// current type should inherit properties of anonymous one
		if name == "" {
			if shouldEmbed {
				r.reflectStructFields(st, definitions, f.Type, nil, 0)
			}
			return
		}

		// If a JSONSchemaAlias(prop string) method is defined, attempt to use
		// the provided object's type instead of the field's type.
		var property *Schema
		if alias := customPropertyMethod(name); alias != nil {
			property = r.refOrReflectTypeToSchema(definitions, reflect.TypeOf(alias))
		} else {
			property = r.refOrReflectTypeToSchema(definitions, f.Type)
		}

		property.structKeywordsFromTags(f, st, name, tags)
		if property.Description == "" {
			property.Description = r.lookupComment(t, f.Name)
		}
		if getFieldDocString != nil {
			property.Description = getFieldDocString(f.Name)
		}

		if nullable {
			property = &Schema{
				OneOf: []*Schema{
					property,
					{
						Type: "null",
					},
				},
			}
		}

		st.Properties.Set(name, property)
		if required {
			st.Required = appendUniqueString(st.Required, name)
		}
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		handleField(f, fieldTagsCache[i])
	}
	if r.AdditionalFields != nil {
		if af := r.AdditionalFields(t); af != nil {
			for _, sf := range af {
				handleField(sf, r.parseFieldTagsForField(sf))
			}
		}
	}
}

type fieldTags struct {
	jsonTags        []string
	jsonschemaTags  []string
	jsonschemaExtra []string
}

type fieldTagsKey struct {
	t       reflect.Type
	tagName string
}

var cachedFieldTags = xsync.NewMapOf[fieldTagsKey, []fieldTags]()

func (r *Reflector) estimatePropertyCapacity(t reflect.Type, tags []fieldTags, seen map[reflect.Type]bool) int {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return 0
	}
	if tags == nil || len(tags) != t.NumField() {
		tags = r.fieldTagsForType(t)
	}
	if seen == nil {
		seen = make(map[reflect.Type]bool)
	}
	if seen[t] {
		return 0
	}
	seen[t] = true

	cap := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, shouldEmbed, _, _ := r.reflectFieldName(f, tags[i])
		if name != "" {
			cap++
			continue
		}
		if shouldEmbed {
			embeddedTags := r.fieldTagsForType(f.Type)
			cap += r.estimatePropertyCapacity(f.Type, embeddedTags, seen)
		}
	}

	if r.AdditionalFields != nil {
		if af := r.AdditionalFields(t); af != nil {
			for _, sf := range af {
				tags := r.parseFieldTagsForField(sf)
				name, shouldEmbed, _, _ := r.reflectFieldName(sf, tags)
				if name != "" {
					cap++
					continue
				}
				if shouldEmbed {
					embeddedTags := r.fieldTagsForType(sf.Type)
					cap += r.estimatePropertyCapacity(sf.Type, embeddedTags, seen)
				}
			}
		}
	}

	return cap
}

func (r *Reflector) fieldTagsForType(t reflect.Type) []fieldTags {
	if t.Kind() != reflect.Struct {
		return nil
	}
	key := fieldTagsKey{
		t:       t,
		tagName: r.fieldNameTag(),
	}
	if tags, ok := cachedFieldTags.Load(key); ok {
		return tags
	}
	tags := make([]fieldTags, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tags[i] = r.parseFieldTagsForField(t.Field(i))
	}
	cachedFieldTags.Store(key, tags)
	return tags
}

func (r *Reflector) parseFieldTagsForField(f reflect.StructField) fieldTags {
	return fieldTags{
		jsonTags:        strings.Split(f.Tag.Get(r.fieldNameTag()), ","),
		jsonschemaTags:  splitOnUnescapedCommas(f.Tag.Get("jsonschema")),
		jsonschemaExtra: strings.Split(f.Tag.Get("jsonschema_extras"), ","),
	}
}

func appendUniqueString(base []string, value string) []string {
	if slices.Contains(base, value) {
		return base
	}
	return append(base, value)
}

// addDefinition will append the provided schema. If needed, an ID and anchor will also be added.
func (r *Reflector) addDefinition(definitions Definitions, t reflect.Type, s *Schema) {
	name := r.typeName(t)
	if name == "" {
		return
	}
	definitions[name] = s
}

// refDefinition will provide a schema with a reference to an existing definition.
func (r *Reflector) refDefinition(definitions Definitions, t reflect.Type) *Schema {
	if r.DoNotReference {
		return nil
	}
	name := r.typeName(t)
	if name == "" {
		return nil
	}
	if _, ok := definitions[name]; !ok {
		return nil
	}
	return &Schema{
		Ref: "#/$defs/" + name,
	}
}

func (r *Reflector) lookupID(t reflect.Type) ID {
	if r.Lookup != nil {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		return r.Lookup(t)

	}
	return EmptyID
}

type schemaCacheKey struct {
	t           reflect.Type
	fingerprint uint64
}

func (r *Reflector) loadSchemaFromCache(t reflect.Type) *Schema {
	cache := r.getSchemaCache()
	key := schemaCacheKey{
		t:           t,
		fingerprint: r.cacheFingerprint(),
	}
	if cache == nil {
		return nil
	}
	if cached, ok := cache.Load(key); ok {
		r.promoteCacheKey(key)
		return cloneSchema(cached)
	}
	return nil
}

func (r *Reflector) storeSchemaInCache(t reflect.Type, s *Schema) {
	cache := r.getSchemaCache()
	if cache == nil {
		return
	}
	key := schemaCacheKey{
		t:           t,
		fingerprint: r.cacheFingerprint(),
	}
	if _, exists := cache.Load(key); exists {
		return
	}
	cache.Store(key, cloneSchema(s))

	if r.MaxSchemaCacheEntries > 0 {
		r.schemaCacheMu.Lock()
		r.schemaCacheOrder = append(r.schemaCacheOrder, key)
		r.evictIfNeeded(cache)
		r.schemaCacheMu.Unlock()
	}
}

func (r *Reflector) promoteCacheKey(key schemaCacheKey) {
	if r.MaxSchemaCacheEntries == 0 || r.schemaCache == nil {
		return
	}
	r.schemaCacheMu.Lock()
	defer r.schemaCacheMu.Unlock()
	for i, k := range r.schemaCacheOrder {
		if k == key {
			// move to end
			copy(r.schemaCacheOrder[i:], r.schemaCacheOrder[i+1:])
			r.schemaCacheOrder[len(r.schemaCacheOrder)-1] = key
			return
		}
	}
}

func (r *Reflector) evictIfNeeded(cache *xsync.MapOf[schemaCacheKey, *Schema]) {
	for r.MaxSchemaCacheEntries > 0 && len(r.schemaCacheOrder) > r.MaxSchemaCacheEntries {
		evict := r.schemaCacheOrder[0]
		r.schemaCacheOrder = r.schemaCacheOrder[1:]
		cache.Delete(evict)
	}
}

func (r *Reflector) getSchemaCache() *xsync.MapOf[schemaCacheKey, *Schema] {
	if !r.EnableSchemaCache {
		return nil
	}
	if r.schemaCache == nil {
		r.schemaCache = xsync.NewMapOf[schemaCacheKey, *Schema]()
	}
	return r.schemaCache
}

func (r *Reflector) cacheFingerprint() uint64 {
	h := fnv.New64a()
	writeBool := func(b bool) {
		if b {
			h.Write([]byte{1})
			return
		}
		h.Write([]byte{0})
	}

	writeBool(r.Anonymous)
	writeBool(r.AssignAnchor)
	writeBool(r.AllowAdditionalProperties)
	writeBool(r.RequiredFromJSONSchemaTags)
	writeBool(r.DoNotReference)
	writeBool(r.ExpandedStruct)
	h.Write([]byte(r.FieldNameTag))
	h.Write([]byte(r.BaseSchemaID))

	writeUintptr := func(ptr uintptr) {
		h.Write([]byte(strconv.FormatUint(uint64(ptr), 10)))
	}

	if r.Lookup != nil {
		writeUintptr(reflect.ValueOf(r.Lookup).Pointer())
	}
	if r.Mapper != nil {
		writeUintptr(reflect.ValueOf(r.Mapper).Pointer())
	}
	if r.Namer != nil {
		writeUintptr(reflect.ValueOf(r.Namer).Pointer())
	}
	if r.KeyNamer != nil {
		writeUintptr(reflect.ValueOf(r.KeyNamer).Pointer())
	}
	if r.AdditionalFields != nil {
		writeUintptr(reflect.ValueOf(r.AdditionalFields).Pointer())
	}
	if r.LookupComment != nil {
		writeUintptr(reflect.ValueOf(r.LookupComment).Pointer())
	}

	if r.CommentMap != nil {
		h.Write([]byte(strconv.FormatInt(int64(len(r.CommentMap)), 10)))
		writeUintptr(reflect.ValueOf(r.CommentMap).Pointer())
	}

	return h.Sum64()
}

func (t *Schema) structKeywordsFromTags(f reflect.StructField, parent *Schema, propertyName string, tags fieldTags) {
	t.Description = f.Tag.Get("jsonschema_description")

	schemaTags := tags.jsonschemaTags
	schemaTags = t.genericKeywords(schemaTags, parent, propertyName)

	switch t.Type {
	case "string":
		t.stringKeywords(schemaTags)
	case "number":
		t.numericalKeywords(schemaTags)
	case "integer":
		t.numericalKeywords(schemaTags)
	case "array":
		t.arrayKeywords(schemaTags)
	case "boolean":
		t.booleanKeywords(schemaTags)
	}
	t.extraKeywords(tags.jsonschemaExtra)
}

// read struct tags for generic keywords
func (t *Schema) genericKeywords(tags []string, parent *Schema, propertyName string) []string { //nolint:gocyclo
	var unprocessed []string
	for _, tag := range tags {
		name, val, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		switch name {
		case "title":
			t.Title = val
		case "description":
			t.Description = val
		case "type":
			t.Type = val
		case "anchor":
			t.Anchor = val
		case "oneof_required":
			var typeFound *Schema
			for i := range parent.OneOf {
				if parent.OneOf[i].Title == val {
					typeFound = parent.OneOf[i]
				}
			}
			if typeFound == nil {
				typeFound = &Schema{
					Title:    val,
					Required: []string{},
				}
				parent.OneOf = append(parent.OneOf, typeFound)
			}
			typeFound.Required = append(typeFound.Required, propertyName)
		case "anyof_required":
			var typeFound *Schema
			for i := range parent.AnyOf {
				if parent.AnyOf[i].Title == val {
					typeFound = parent.AnyOf[i]
				}
			}
			if typeFound == nil {
				typeFound = &Schema{
					Title:    val,
					Required: []string{},
				}
				parent.AnyOf = append(parent.AnyOf, typeFound)
			}
			typeFound.Required = append(typeFound.Required, propertyName)
		case "oneof_ref":
			subSchema := t
			if t.Items != nil {
				subSchema = t.Items
			}
			if subSchema.OneOf == nil {
				subSchema.OneOf = make([]*Schema, 0, 1)
			}
			subSchema.Ref = ""
			for r := range strings.SplitSeq(val, ";") {
				subSchema.OneOf = append(subSchema.OneOf, &Schema{Ref: r})
			}
		case "oneof_type":
			if t.OneOf == nil {
				t.OneOf = make([]*Schema, 0, 1)
			}
			t.Type = ""
			for ty := range strings.SplitSeq(val, ";") {
				t.OneOf = append(t.OneOf, &Schema{Type: ty})
			}
		case "anyof_ref":
			subSchema := t
			if t.Items != nil {
				subSchema = t.Items
			}
			if subSchema.AnyOf == nil {
				subSchema.AnyOf = make([]*Schema, 0, 1)
			}
			subSchema.Ref = ""
			for r := range strings.SplitSeq(val, ";") {
				subSchema.AnyOf = append(subSchema.AnyOf, &Schema{Ref: r})
			}
		case "anyof_type":
			if t.AnyOf == nil {
				t.AnyOf = make([]*Schema, 0, 1)
			}
			t.Type = ""
			for ty := range strings.SplitSeq(val, ";") {
				t.AnyOf = append(t.AnyOf, &Schema{Type: ty})
			}
		default:
			unprocessed = append(unprocessed, tag)
		}
	}
	return unprocessed
}

// read struct tags for boolean type keywords
func (t *Schema) booleanKeywords(tags []string) {
	for _, tag := range tags {
		name, val, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		if name == "default" {
			if val == "true" {
				t.Default = true
			} else if val == "false" {
				t.Default = false
			}
		}
	}
}

// read struct tags for string type keywords
func (t *Schema) stringKeywords(tags []string) {
	for _, tag := range tags {
		name, val, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		switch name {
		case "minLength":
			t.MinLength = parseUint(val)
		case "maxLength":
			t.MaxLength = parseUint(val)
		case "pattern":
			t.Pattern = val
		case "format":
			t.Format = val
		case "readOnly":
			i, _ := strconv.ParseBool(val)
			t.ReadOnly = i
		case "writeOnly":
			i, _ := strconv.ParseBool(val)
			t.WriteOnly = i
		case "default":
			t.Default = val
		case "example":
			t.Examples = append(t.Examples, val)
		case "enum":
			t.Enum = append(t.Enum, val)
		}
	}
}

// read struct tags for numerical type keywords
func (t *Schema) numericalKeywords(tags []string) {
	for _, tag := range tags {
		name, val, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		switch name {
		case "multipleOf":
			t.MultipleOf, _ = toJSONNumber(val)
		case "minimum":
			t.Minimum, _ = toJSONNumber(val)
		case "maximum":
			t.Maximum, _ = toJSONNumber(val)
		case "exclusiveMaximum":
			t.ExclusiveMaximum, _ = toJSONNumber(val)
		case "exclusiveMinimum":
			t.ExclusiveMinimum, _ = toJSONNumber(val)
		case "default":
			if num, ok := toJSONNumber(val); ok {
				t.Default = num
			}
		case "example":
			if num, ok := toJSONNumber(val); ok {
				t.Examples = append(t.Examples, num)
			}
		case "enum":
			if num, ok := toJSONNumber(val); ok {
				t.Enum = append(t.Enum, num)
			}
		}
	}
}

// read struct tags for object type keywords
// func (t *Type) objectKeywords(tags []string) {
//     for _, tag := range tags{
//         nameValue := strings.Split(tag, "=")
//         name, val := nameValue[0], nameValue[1]
//         switch name{
//             case "dependencies":
//                 t.Dependencies = val
//                 break;
//             case "patternProperties":
//                 t.PatternProperties = val
//                 break;
//         }
//     }
// }

// read struct tags for array type keywords
func (t *Schema) arrayKeywords(tags []string) {
	var defaultValues []any

	var unprocessed []string
	for _, tag := range tags {
		name, val, ok := strings.Cut(tag, "=")
		if !ok {
			continue
		}
		switch name {
		case "minItems":
			t.MinItems = parseUint(val)
		case "maxItems":
			t.MaxItems = parseUint(val)
		case "uniqueItems":
			t.UniqueItems = true
		case "default":
			defaultValues = append(defaultValues, val)
		case "format":
			t.Items.Format = val
		case "pattern":
			t.Items.Pattern = val
		default:
			unprocessed = append(unprocessed, tag) // left for further processing by underlying type
		}
	}
	if len(defaultValues) > 0 {
		t.Default = defaultValues
	}

	if len(unprocessed) == 0 {
		// we don't have anything else to process
		return
	}

	switch t.Items.Type {
	case "string":
		t.Items.stringKeywords(unprocessed)
	case "number":
		t.Items.numericalKeywords(unprocessed)
	case "integer":
		t.Items.numericalKeywords(unprocessed)
	case "array":
		// explicitly don't support traversal for the [][]..., as it's unclear where the array tags belong
	case "boolean":
		t.Items.booleanKeywords(unprocessed)
	}
}

func (t *Schema) extraKeywords(tags []string) {
	for _, tag := range tags {
		if key, val, ok := strings.Cut(tag, "="); ok {
			t.setExtra(key, val)
		}
	}
}

func (t *Schema) setExtra(key, val string) {
	if t.Extras == nil {
		t.Extras = map[string]any{}
	}
	if existingVal, ok := t.Extras[key]; ok {
		switch existingVal := existingVal.(type) {
		case string:
			t.Extras[key] = []string{existingVal, val}
		case []string:
			t.Extras[key] = append(existingVal, val)
		case int:
			t.Extras[key], _ = strconv.Atoi(val)
		case bool:
			t.Extras[key] = (val == "true" || val == "t")
		}
	} else {
		switch key {
		case "minimum":
			t.Extras[key], _ = strconv.Atoi(val)
		default:
			var x any
			if val == "true" {
				x = true
			} else if val == "false" {
				x = false
			} else {
				x = val
			}
			t.Extras[key] = x
		}
	}
}

func requiredFromJSONTags(tags []string, val *bool) {
	if ignoredByJSONTags(tags) {
		return
	}

	for _, tag := range tags[1:] {
		switch tag {
		case "omitempty", "omitzero":
			*val = false
			return
		}
	}
	*val = true
}

func requiredFromJSONSchemaTags(tags []string, val *bool) {
	if ignoredByJSONSchemaTags(tags) {
		return
	}
	for _, tag := range tags {
		if tag == "required" {
			*val = true
		}
	}
}

func nullableFromJSONSchemaTags(tags []string) bool {
	if ignoredByJSONSchemaTags(tags) {
		return false
	}
	return slices.Contains(tags, "nullable")
}

func ignoredByJSONTags(tags []string) bool {
	return tags[0] == "-"
}

func ignoredByJSONSchemaTags(tags []string) bool {
	return tags[0] == "-"
}

func inlinedByJSONTags(tags []string) bool {
	return slices.Contains(tags[1:], "inline")
}

// toJSONNumber converts string to *json.Number.
// It'll aso return whether the number is valid.
func toJSONNumber(s string) (json.Number, bool) {
	num := json.Number(s)
	if _, err := num.Int64(); err == nil {
		return num, true
	}
	if _, err := num.Float64(); err == nil {
		return num, true
	}
	return json.Number(""), false
}

func parseUint(num string) *uint64 {
	val, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return nil
	}
	return &val
}

func (r *Reflector) fieldNameTag() string {
	if r.FieldNameTag != "" {
		return r.FieldNameTag
	}
	return "json"
}

func (r *Reflector) reflectFieldName(f reflect.StructField, tags fieldTags) (string, bool, bool, bool) {
	jsonTags := tags.jsonTags

	if ignoredByJSONTags(jsonTags) {
		return "", false, false, false
	}

	schemaTags := tags.jsonschemaTags
	if ignoredByJSONSchemaTags(schemaTags) {
		return "", false, false, false
	}

	var required bool
	if !r.RequiredFromJSONSchemaTags {
		requiredFromJSONTags(jsonTags, &required)
	}
	requiredFromJSONSchemaTags(schemaTags, &required)

	nullable := nullableFromJSONSchemaTags(schemaTags)

	if f.Anonymous && jsonTags[0] == "" {
		// As per JSON Marshal rules, anonymous structs are inherited
		if f.Type.Kind() == reflect.Struct {
			return "", true, false, false
		}

		// As per JSON Marshal rules, anonymous pointer to structs are inherited
		if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
			return "", true, false, false
		}
	}

	// As per JSON Marshal rules, inline nested structs that have `inline` tag.
	if inlinedByJSONTags(jsonTags) {
		return "", true, false, false
	}

	// Try to determine the name from the different combos
	name := f.Name
	if jsonTags[0] != "" {
		name = jsonTags[0]
	}
	if !f.Anonymous && f.PkgPath != "" {
		// field not anonymous and not export has no export name
		name = ""
	} else if r.KeyNamer != nil {
		name = r.KeyNamer(name)
	}

	return name, false, required, nullable
}

// UnmarshalJSON is used to parse a schema object or boolean.
func (t *Schema) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("true")) {
		*t = *TrueSchema
		return nil
	} else if bytes.Equal(data, []byte("false")) {
		*t = *FalseSchema
		return nil
	}
	type SchemaAlt Schema
	aux := &struct {
		*SchemaAlt
	}{
		SchemaAlt: (*SchemaAlt)(t),
	}
	return json.Unmarshal(data, aux)
}

// MarshalJSON is used to serialize a schema object or boolean.
func (t *Schema) MarshalJSON() ([]byte, error) {
	if t.boolean != nil {
		if *t.boolean {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	}
	if reflect.DeepEqual(&Schema{}, t) {
		// Don't bother returning empty schemas
		return []byte("true"), nil
	}
	type SchemaAlt Schema
	b, err := json.Marshal((*SchemaAlt)(t))
	if err != nil {
		return nil, err
	}
	if len(t.Extras) == 0 {
		return b, nil
	}
	m, err := json.Marshal(t.Extras)
	if err != nil {
		return nil, err
	}
	if len(b) == 2 {
		return m, nil
	}
	b[len(b)-1] = ','
	return append(b, m[1:]...), nil
}

func (r *Reflector) typeName(t reflect.Type) string {
	if r.Namer != nil {
		if name := r.Namer(t); name != "" {
			return name
		}
	}
	return t.Name()
}

// Split on commas that are not preceded by `\`.
// This way, we prevent splitting regexes
func splitOnUnescapedCommas(tagString string) []string {
	if tagString == "" {
		return []string{""}
	}

	parts := make([]string, 0, 4)
	buf := make([]byte, 0, len(tagString))
	prevEsc := false

	for i := 0; i < len(tagString); i++ {
		ch := tagString[i]
		if ch == ',' {
			if prevEsc {
				// Remove escaping backslash and treat comma literally.
				buf = buf[:len(buf)-1]
				buf = append(buf, ',')
				prevEsc = false
				continue
			}
			parts = append(parts, string(buf))
			buf = buf[:0]
			prevEsc = false
			continue
		}

		buf = append(buf, ch)
		prevEsc = ch == '\\'
	}

	parts = append(parts, string(buf))
	return parts
}

func fullyQualifiedTypeName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

func cloneSchema(s *Schema) *Schema {
	if s == nil {
		return nil
	}

	ns := *s

	if s.AllOf != nil {
		ns.AllOf = cloneSchemaSlice(s.AllOf)
	}
	if s.AnyOf != nil {
		ns.AnyOf = cloneSchemaSlice(s.AnyOf)
	}
	if s.OneOf != nil {
		ns.OneOf = cloneSchemaSlice(s.OneOf)
	}
	if s.Not != nil {
		ns.Not = cloneSchema(s.Not)
	}
	if s.If != nil {
		ns.If = cloneSchema(s.If)
	}
	if s.Then != nil {
		ns.Then = cloneSchema(s.Then)
	}
	if s.Else != nil {
		ns.Else = cloneSchema(s.Else)
	}
	if s.DependentSchemas != nil {
		ns.DependentSchemas = make(map[string]*Schema, len(s.DependentSchemas))
		for k, v := range s.DependentSchemas {
			ns.DependentSchemas[k] = cloneSchema(v)
		}
	}
	if s.PrefixItems != nil {
		ns.PrefixItems = cloneSchemaSlice(s.PrefixItems)
	}
	if s.Items != nil {
		ns.Items = cloneSchema(s.Items)
	}
	if s.Contains != nil {
		ns.Contains = cloneSchema(s.Contains)
	}
	if s.Properties != nil {
		props := NewProperties(s.Properties.Len())
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			props.Set(pair.Key, cloneSchema(pair.Value))
		}
		ns.Properties = props
	}
	if s.PatternProperties != nil {
		ns.PatternProperties = make(map[string]*Schema, len(s.PatternProperties))
		for k, v := range s.PatternProperties {
			ns.PatternProperties[k] = cloneSchema(v)
		}
	}
	if s.AdditionalProperties != nil {
		ns.AdditionalProperties = cloneSchema(s.AdditionalProperties)
	}
	if s.PropertyNames != nil {
		ns.PropertyNames = cloneSchema(s.PropertyNames)
	}
	if s.Enum != nil {
		enumCopy := make([]any, len(s.Enum))
		copy(enumCopy, s.Enum)
		ns.Enum = enumCopy
	}
	if s.Required != nil {
		ns.Required = copyStringSlice(s.Required)
	}
	if s.DependentRequired != nil {
		ns.DependentRequired = make(map[string][]string, len(s.DependentRequired))
		for k, v := range s.DependentRequired {
			ns.DependentRequired[k] = copyStringSlice(v)
		}
	}
	if s.ContentSchema != nil {
		ns.ContentSchema = cloneSchema(s.ContentSchema)
	}
	if s.Examples != nil {
		examples := make([]any, len(s.Examples))
		copy(examples, s.Examples)
		ns.Examples = examples
	}
	if s.Definitions != nil {
		defs := make(Definitions, len(s.Definitions))
		for k, v := range s.Definitions {
			defs[k] = cloneSchema(v)
		}
		ns.Definitions = defs
	}
	if s.Extras != nil {
		extras := make(map[string]any, len(s.Extras))
		maps.Copy(extras, s.Extras)
		ns.Extras = extras
	}

	return &ns
}

func cloneSchemaSlice(in []*Schema) []*Schema {
	if len(in) == 0 {
		return nil
	}
	out := make([]*Schema, len(in))
	for i := range in {
		out[i] = cloneSchema(in[i])
	}
	return out
}

func copyStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
