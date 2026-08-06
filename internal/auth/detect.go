package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// chromiumFamily describes where each Chromium browser keeps its data, and which
// keychain entry holds its encryption key.
var chromiumFamily = []struct {
	name            string
	macDir          string // relative to ~/Library/Application Support
	linuxDir        string // relative to ~/.config
	keychainService string
	keychainAccount string
}{
	{"Chrome", "Google/Chrome", "google-chrome", "Chrome Safe Storage", "Chrome"},
	{"Brave", "BraveSoftware/Brave-Browser", "BraveSoftware/Brave-Browser", "Brave Safe Storage", "Brave"},
	{"Edge", "Microsoft Edge", "microsoft-edge", "Microsoft Edge Safe Storage", "Microsoft Edge"},
	{"Arc", "Arc/User Data", "", "Arc Safe Storage", "Arc"},
	{"Chromium", "Chromium", "chromium", "Chromium Safe Storage", "Chromium"},
	{"Vivaldi", "Vivaldi", "vivaldi", "Vivaldi Safe Storage", "Vivaldi"},
	{"Opera", "com.operasoftware.Opera", "opera", "Opera Safe Storage", "Opera"},
}

// DetectBrowsers lists browser profiles that have a cookie database on disk.
//
// This is filesystem stats only — no keychain, so it never triggers a permission
// prompt. Presence of a database is not proof of a LeetCode session; Import reports
// that.
func DetectBrowsers() []Browser {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var found []Browser
	for _, fam := range chromiumFamily {
		root := chromiumRoot(home, fam.macDir, fam.linuxDir)
		if root == "" {
			continue
		}
		for _, profile := range chromiumProfiles(root) {
			path := filepath.Join(root, profile, "Cookies")
			// Chrome moved the database under Network/ in 2021; older installs and some
			// forks still use the original location.
			if !fileExists(path) {
				path = filepath.Join(root, profile, "Network", "Cookies")
			}
			if !fileExists(path) {
				continue
			}
			found = append(found, Browser{
				Name:            fam.name,
				Profile:         profile,
				cookiePath:      path,
				keychainService: fam.keychainService,
				keychainAccount: fam.keychainAccount,
			})
		}
	}
	return append(found, detectFirefox(home)...)
}

func chromiumRoot(home, macDir, linuxDir string) string {
	var root string
	switch runtime.GOOS {
	case "darwin":
		if macDir == "" {
			return ""
		}
		root = filepath.Join(home, "Library", "Application Support", macDir)
	case "linux":
		if linuxDir == "" {
			return ""
		}
		root = filepath.Join(home, ".config", linuxDir)
	default:
		return ""
	}
	if !dirExists(root) {
		return ""
	}
	return root
}

// chromiumProfiles lists "Default" and any "Profile N" directories.
func chromiumProfiles(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var profiles []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if n := e.Name(); n == "Default" || strings.HasPrefix(n, "Profile ") {
			profiles = append(profiles, n)
		}
	}
	sort.Strings(profiles)
	return profiles
}

func detectFirefox(home string) []Browser {
	var root string
	switch runtime.GOOS {
	case "darwin":
		root = filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	case "linux":
		root = filepath.Join(home, ".mozilla", "firefox")
	default:
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var found []Browser
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name(), "cookies.sqlite")
		if !fileExists(path) {
			continue
		}
		found = append(found, Browser{
			Name:       "Firefox",
			Profile:    e.Name(),
			cookiePath: path,
			firefox:    true,
		})
	}
	return found
}
