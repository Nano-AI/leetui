package auth

import (
	"context"
	"database/sql"
	"fmt"
)

func readFirefox(ctx context.Context, db *sql.DB) (Credentials, error) {
	var c Credentials

	rows, err := db.QueryContext(ctx, `
		SELECT name, value FROM moz_cookies
		WHERE host LIKE '%leetcode.com'
		  AND name IN ('LEETCODE_SESSION', 'csrftoken')`)
	if err != nil {
		return c, fmt.Errorf("query cookies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return c, fmt.Errorf("scan cookie: %w", err)
		}
		switch name {
		case "LEETCODE_SESSION":
			c.Session = value
		case "csrftoken":
			c.CSRF = value
		}
	}
	if err := rows.Err(); err != nil {
		return c, err
	}
	if !c.Valid() {
		return c, ErrNoBrowserCookies
	}
	return c, nil
}
