package auth

import (
	"fmt"
	"os"
)

// Environment overrides. Both are required together.
//
// Named LEETUI_* rather than LEETCODE_* on purpose: the cookie names are generic enough
// that a browser extension, a scraper, or another LeetCode tool could plausibly export
// them, and inheriting somebody else's session by accident is a bad way to find out.
const (
	EnvSession = "LEETUI_SESSION"
	EnvCSRF    = "LEETUI_CSRF"
)

// loadEnv reads credentials from the environment.
//
// The bool distinguishes "not configured" from "configured and complete". One variable
// without the other is neither: it is a typo in a shell profile or a CI secret that
// failed to inject, and both are worth failing loudly on. Falling through to the next
// backend would leave the user signed in as somebody they did not intend, with nothing
// on screen to explain why their override did not take.
func loadEnv() (Credentials, bool, error) {
	session := Clean(os.Getenv(EnvSession))
	csrf := Clean(os.Getenv(EnvCSRF))

	switch {
	case session == "" && csrf == "":
		return Credentials{}, false, nil
	case session == "":
		return Credentials{}, false, fmt.Errorf("%s is set but %s is not; set both or neither", EnvCSRF, EnvSession)
	case csrf == "":
		return Credentials{}, false, fmt.Errorf("%s is set but %s is not; set both or neither", EnvSession, EnvCSRF)
	}
	return Credentials{Session: session, CSRF: csrf}, true, nil
}
