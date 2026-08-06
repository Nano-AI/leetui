package auth

import (
	"context"
	"errors"
	"fmt"
)

// Import reads the LeetCode session out of one browser.
//
// Returns ErrNoBrowserCookies when that browser has no LeetCode session — the user is
// signed out there, or signed in somewhere else.
func Import(ctx context.Context, b Browser) (Credentials, error) {
	var c Credentials

	// The live database is locked while the browser runs, so work on a copy.
	tmp, cleanup, err := copyToTemp(b.cookiePath)
	if err != nil {
		return c, fmt.Errorf("copy %s cookie database: %w", b.Name, err)
	}
	defer cleanup()

	db, err := openReadOnly(tmp)
	if err != nil {
		return c, fmt.Errorf("open %s cookie database: %w", b.Name, err)
	}
	defer db.Close()

	if b.firefox {
		return readFirefox(ctx, db)
	}
	return readChromium(ctx, db, b)
}

// ImportAny tries every detected browser and returns the first usable session.
//
// Errors are collected rather than returned immediately: one browser being locked or
// signed out should not stop the others from being tried.
func ImportAny(ctx context.Context) (Credentials, Browser, error) {
	browsers := DetectBrowsers()
	if len(browsers) == 0 {
		return Credentials{}, Browser{}, errors.New("no supported browser found")
	}

	var problems []error
	for _, b := range browsers {
		c, err := Import(ctx, b)
		if err != nil {
			if !errors.Is(err, ErrNoBrowserCookies) {
				problems = append(problems, fmt.Errorf("%s: %w", b.Label(), err))
			}
			continue
		}
		if c.Valid() {
			return c, b, nil
		}
	}

	if len(problems) > 0 {
		return Credentials{}, Browser{}, errors.Join(problems...)
	}
	return Credentials{}, Browser{}, ErrNoBrowserCookies
}
