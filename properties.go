package jsonschema

import (
	jsonv1 "github.com/goccy/go-json"
)

// Properties preserves insertion order and provides map-like access for schema properties.
// It is optimized for repeated lookups and deterministic iteration without the overhead of
// a general-purpose ordered map.
type Properties struct {
	order  []string
	values map[string]*Schema
}

func NewProperties() *Properties {
	return &Properties{
		values: make(map[string]*Schema),
	}
}

func NewPropertiesCap(capacity int) *Properties {
	return &Properties{
		order:  make([]string, 0, capacity),
		values: make(map[string]*Schema, capacity),
	}
}

func (p *Properties) Len() int {
	if p == nil {
		return 0
	}
	return len(p.values)
}

func (p *Properties) Set(key string, value *Schema) {
	if p.values == nil {
		p.values = make(map[string]*Schema)
	}
	if _, exists := p.values[key]; !exists {
		p.order = append(p.order, key)
	}
	p.values[key] = value
}

func (p *Properties) Get(key string) (*Schema, bool) {
	if p == nil {
		return nil, false
	}
	v, ok := p.values[key]
	return v, ok
}

func (p *Properties) Delete(key string) {
	if p == nil {
		return
	}
	if _, ok := p.values[key]; !ok {
		return
	}
	delete(p.values, key)
	for i, k := range p.order {
		if k == key {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
}

func (p *Properties) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}

	// Precompute size roughly: '{' '}' and separators.
	buf := make([]byte, 0, len(p.order)*32)
	buf = append(buf, '{')
	for i, k := range p.order {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyBytes, err := jsonv1.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		valBytes, err := jsonv1.Marshal(p.values[k])
		if err != nil {
			return nil, err
		}
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}')
	return buf, nil
}

func (p *Properties) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var m map[string]*Schema
	if err := jsonv1.Unmarshal(data, &m); err != nil {
		return err
	}
	if p.values == nil {
		p.values = make(map[string]*Schema, len(m))
	}
	for k, v := range m {
		p.order = append(p.order, k)
		p.values[k] = v
	}
	return nil
}
