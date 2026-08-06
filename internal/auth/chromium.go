package auth

import (
	"context"
	"database/sql"
	"fmt"
)

// readChromium pulls and decrypts the two cookies leetui needs.
func readChromium(ctx context.Context, db *sql.DB, b Browser) (Credentials, error) {
	var c Credentials

	rows, err := db.QueryContext(ctx, `
		SELECT name, value, encrypted_value
		FROM cookies
		WHERE host_key LIKE '%leetcode.com'
		  AND name IN ('LEETCODE_SESSION', 'csrftoken')`)
	if err != nil {
		return c, fmt.Errorf("query cookies: %w", err)
	}
	defer rows.Close()

	type raw struct {
		name  string
		plain string
		enc   []byte
	}
	var found []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.name, &r.plain, &r.enc); err != nil {
			return c, fmt.Errorf("scan cookie: %w", err)
		}
		found = append(found, r)
	}
	if err := rows.Err(); err != nil {
		return c, err
	}
	if len(found) == 0 {
		return c, ErrNoBrowserCookies
	}

	// Only fetch the decryption key if something actually needs decrypting — that way a
	// profile with plaintext cookies never triggers a keychain prompt.
	var key []byte
	needsKey := false
	for _, r := range found {
		if r.plain == "" && len(r.enc) > 0 {
			needsKey = true
		}
	}
	if needsKey {
		key, err = chromiumKey(b)
		if err != nil {
			return c, err
		}
	}

	for _, r := range found {
		value := r.plain
		if value == "" && len(r.enc) > 0 {
			value, err = decryptChromium(r.enc, key)
			if err != nil {
				return c, fmt.Errorf("decrypt %s: %w", r.name, err)
			}
		}
		switch r.name {
		case "LEETCODE_SESSION":
			c.Session = value
		case "csrftoken":
			c.CSRF = value
		}
	}

	if !c.Valid() {
		return c, ErrNoBrowserCookies
	}
	return c, nil
}
