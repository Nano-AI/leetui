package runner

import (
	"encoding/json"
	"math/big"
)

// exactNumber stores the reduced rational spelling of a JSON number. The distinct
// type keeps numbers separate from JSON strings while preserving equivalence of
// 1, 1.0, and 1e0 without rounding large integer answers through float64.
type exactNumber string

func (n exactNumber) rat() *big.Rat {
	r, _ := new(big.Rat).SetString(string(n))
	return r
}

func exactNumbers(v any) any {
	switch v := v.(type) {
	case json.Number:
		if r, ok := new(big.Rat).SetString(string(v)); ok {
			return exactNumber(r.RatString())
		}
		return v
	case []any:
		for i := range v {
			v[i] = exactNumbers(v[i])
		}
		return v
	case map[string]any:
		for k := range v {
			v[k] = exactNumbers(v[k])
		}
		return v
	default:
		return v
	}
}
