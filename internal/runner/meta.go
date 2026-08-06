package runner

import (
	"encoding/json"
	"fmt"
)

// Meta is LeetCode's `metaData` — the only structured description of a solution's shape
// that exists.
//
// It gives parameter names, parameter types, and the return type. It does NOT say
// whether a problem mutates its input in place, accepts answers in any order, or
// tolerates floating-point error. Those live in the override table (overrides.go),
// because there is nowhere else to get them.
type Meta struct {
	Name   string  `json:"name"`
	Params []Param `json:"params"`
	Return struct {
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"return"`

	// Manual marks problems whose driver LeetCode writes by hand.
	Manual bool `json:"manual"`

	// Classname and Methods are present on design problems — a class plus a sequence of
	// operations, rather than one function. They need a different driver shape.
	Classname   string          `json:"classname"`
	Constructor json.RawMessage `json:"constructor"`
	Methods     json.RawMessage `json:"methods"`
}

// Param is one argument.
type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParseMeta decodes a metaData string.
func ParseMeta(raw string) (Meta, error) {
	var m Meta
	if raw == "" {
		return m, fmt.Errorf("problem has no metaData; local run is not possible")
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return m, fmt.Errorf("parse metaData: %w", err)
	}
	if m.Name == "" && m.Classname == "" {
		return m, fmt.Errorf("metaData names neither a function nor a class")
	}
	return m, nil
}

// IsDesign reports whether this is a class-with-operations problem.
//
// These are the ones most likely to disagree with the judge, so the runner declines them
// rather than reporting a result it cannot stand behind.
func (m Meta) IsDesign() bool {
	return m.Classname != "" || len(m.Methods) > 0
}

// ParamTypes returns the parameter types in order, ready to hand to a driver.
func (m Meta) ParamTypes() []string {
	out := make([]string, len(m.Params))
	for i, p := range m.Params {
		out[i] = p.Type
	}
	return out
}

// ParamNames returns the parameter names in order.
func (m Meta) ParamNames() []string {
	out := make([]string, len(m.Params))
	for i, p := range m.Params {
		out[i] = p.Name
	}
	return out
}
