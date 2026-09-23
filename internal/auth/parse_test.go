package auth

import "testing"

// TestClean covers the shapes a real paste arrives in.
//
// This is the function that makes the two-field form work at all. The single-field form
// matched on the cookie NAMES, so copying the two values out of the devtools cookie
// table, which is exactly what the app's own instructions told you to do, produced two
// bare values and the unhelpful "found neither LEETCODE_SESSION nor csrftoken".
func TestClean(t *testing.T) {
	const val = "eyJhbGciOiJIUzI1NiJ9.payload.sig"

	for _, tc := range []struct {
		name, in, want string
	}{
		{"bare value", val, val},
		{"surrounding whitespace", "  " + val + "  ", val},
		{"trailing newline from a paste", val + "\n", val},
		{"name= prefix", "LEETCODE_SESSION=" + val, val},
		{"name: prefix, as in a header dump", "csrftoken: abc123", "abc123"},
		{"name = value with spaces", "LEETCODE_SESSION = " + val, val},
		{"trailing semicolon from a cookie header", val + ";", val},
		{"name, value and semicolon together", "LEETCODE_SESSION=" + val + ";", val},
		{"double quotes, as in JSON", `"` + val + `"`, val},
		{"single quotes, as in a shell", "'" + val + "'", val},
		{"empty", "", ""},
		{"whitespace only", "   ", ""},

		// A lone quote is either part of the value or a sign the paste is truncated.
		// Removing it would quietly change the secret, and the user cannot see it.
		{"unmatched leading quote is left alone", `"` + val, `"` + val},

		// Only the two names we know are stripped. A value that happens to contain
		// "something=" must survive intact.
		{"unrelated prefix is not stripped", "sessionid=" + val, "sessionid=" + val},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Clean(tc.in); got != tc.want {
				t.Errorf("Clean(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSniff is all-or-nothing on purpose: half a credential distributed across the form
// is worse than leaving the paste alone, because the user cannot see which half moved.
func TestSniff(t *testing.T) {
	const sess = "eyJhbGciOiJIUzI1NiJ9.payload.sig"
	const csrf = "abc123DEF456"

	for _, tc := range []struct {
		name, in string
		want     bool
	}{
		{"cookie header", "LEETCODE_SESSION=" + sess + "; csrftoken=" + csrf, true},
		{"separate lines", "LEETCODE_SESSION=" + sess + "\ncsrftoken=" + csrf, true},
		{"json export", `{"LEETCODE_SESSION":"` + sess + `","csrftoken":"` + csrf + `"}`, true},
		{"session only", "LEETCODE_SESSION=" + sess, false},
		{"csrf only", "csrftoken=" + csrf, false},
		{"a bare value with no names", sess, false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Sniff(tc.in)
			if ok != tc.want {
				t.Fatalf("Sniff(%q) ok = %v, want %v", tc.in, ok, tc.want)
			}
			if ok && (got.Session != sess || got.CSRF != csrf) {
				t.Errorf("Sniff extracted %v, want both cookies intact", got)
			}
		})
	}
}

// TestRedactLeaksNothingUsable is a standing check on the one function allowed to put a
// secret near a screen.
func TestRedactLeaksNothingUsable(t *testing.T) {
	const secret = "eyJhbGciOiJIUzI1NiJ9.averylongpayloadthatmustnotappear.signature"
	got := Redact(secret)
	if len(got) > 24 {
		t.Errorf("Redact returned %d characters: %q", len(got), got)
	}
	if got == secret {
		t.Fatal("Redact returned the secret unchanged")
	}
	if Redact("") != "<empty>" {
		t.Error("Redact lost its empty case")
	}
	if Redact("short") != "<redacted>" {
		t.Error("a short secret was not fully redacted")
	}
}
