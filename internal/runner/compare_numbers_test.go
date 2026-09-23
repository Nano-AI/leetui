package runner

import "testing"

func TestComparePreservesLargeNumbers(t *testing.T) {
	for _, rule := range []Rule{DefaultRule(), {Unordered: true}, {Unordered: true, OrderedRows: true}, {FloatTolerance: 1e-5}} {
		for _, pair := range [][2]string{
			{"9007199254740992", "9007199254740993"},
			{"[9223372036854775806]", "[9223372036854775807]"},
			{"[[-9007199254740992]]", "[[-9007199254740993]]"},
		} {
			if Compare(pair[0], pair[1], rule) {
				t.Errorf("unequal numbers passed: %v under %+v", pair, rule)
			}
		}
		if !Compare("[9007199254740993, 1.0]", "[9007199254740993.0, 1e0]", rule) {
			t.Errorf("equivalent numeric spellings rejected under %+v", rule)
		}
	}
	if _, ok := parseJSON("1 2"); ok {
		t.Error("multiple JSON values accepted")
	}
}
