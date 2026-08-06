package store

import (
	"context"
	"testing"
)

// TestNumericSearch: typing a problem number must find that problem. Full-text search
// cannot answer "146" — the number lives in a column, not in the indexed text.
func TestNumericSearch(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	if err := s.UpsertSummaries(ctx, sample()); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		query string
		want  int // expected top hit's numeric ID; 0 means no results
	}{
		{"146", 146},
		{"1", 1}, // exact match outranks the prefix match on 146
		{"14", 146},
		{"42", 42},
		{"9999", 0},
	}

	for _, tc := range cases {
		rows, err := s.Query(ctx, Filter{Text: tc.query})
		if err != nil {
			t.Errorf("query %q: %v", tc.query, err)
			continue
		}
		if tc.want == 0 {
			if len(rows) != 0 {
				t.Errorf("query %q returned %d rows, want none", tc.query, len(rows))
			}
			continue
		}
		if len(rows) == 0 {
			t.Errorf("query %q returned nothing, want problem %d", tc.query, tc.want)
			continue
		}
		if rows[0].NumericID != tc.want {
			t.Errorf("query %q top hit = %d, want %d", tc.query, rows[0].NumericID, tc.want)
		}
	}
}

func TestNumericQueryDetection(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"146", "146"},
		{"0042", "42"}, // leading zeros are how the board displays them
		{" 7 ", "7"},
		{"two sum", ""},
		{"3sum", ""},  // starts with a digit but is a title
		{"12345", ""}, // longer than any problem number
		{"", ""},
	} {
		if got := numericQuery(tc.in); got != tc.want {
			t.Errorf("numericQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
