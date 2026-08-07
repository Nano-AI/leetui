package vcs

import (
	"strconv"
	"strings"
)

// Parsing git's machine-readable output.
//
// Everything here is fed NUL-separated (-z). Without it git C-quotes any path holding a
// space or a non-ASCII byte, and the caller would have to unquote it — the workspace's
// own folders are safe, but a user's repository may contain anything.

// parsePorcelain reads `git status --porcelain=v2 --branch -z`.
//
// Header lines are `# key value`. Entry lines start with a type byte:
//
//	1  ordinary change      2  rename/copy      u  unmerged      ?  untracked
//
// Type 2 is followed by a second NUL-terminated field holding the original path, which
// is why this walks the records rather than ranging over them.
func parsePorcelain(out string) Status {
	var s Status
	recs := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")

	for i := 0; i < len(recs); i++ {
		rec := recs[i]
		if rec == "" {
			continue
		}
		switch rec[0] {
		case '#':
			s.applyHeader(rec)
		case '1', '2':
			if c, ok := parseEntry(rec); ok {
				s.Changes = append(s.Changes, c)
			}
			if rec[0] == '2' {
				i++ // consume the original path
			}
		case 'u':
			if c, ok := parseEntry(rec); ok {
				// Unmerged paths are both sides at once; report them as needing
				// attention rather than pretending one half is settled.
				c.Staged, c.Unstaged = true, true
				s.Changes = append(s.Changes, c)
			}
		case '?':
			s.Changes = append(s.Changes, Change{
				Path: strings.TrimSpace(rec[1:]), Untracked: true,
			})
		}
	}
	return s
}

// applyHeader reads one `# key value` line.
func (s *Status) applyHeader(rec string) {
	fields := strings.Fields(rec)
	if len(fields) < 3 {
		return
	}
	switch fields[1] {
	case "branch.head":
		if fields[2] == "(detached)" {
			s.Detached = true
			return
		}
		s.Branch = fields[2]
	case "branch.upstream":
		s.Upstream = fields[2]
	case "branch.ab":
		// "+3 -0" — signed, and git omits the line entirely with no upstream.
		s.Ahead = atoiSigned(fields[2])
		if len(fields) > 3 {
			s.Behind = atoiSigned(fields[3])
		}
	}
}

// parseEntry reads a `1`/`2`/`u` line. The path is the last space-separated field for
// type 1 and u; for type 2 the score sits between the hashes and the path, so the path
// is still last.
func parseEntry(rec string) (Change, bool) {
	fields := strings.Fields(rec)
	if len(fields) < 3 {
		return Change{}, false
	}
	xy := fields[1]
	if len(xy) < 2 {
		return Change{}, false
	}
	return Change{
		Path:     fields[len(fields)-1],
		Staged:   xy[0] != '.',
		Unstaged: xy[1] != '.',
	}, true
}

// atoiSigned reads git's "+3" / "-0" counts as a magnitude.
func atoiSigned(s string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(s, "+"))
	if err != nil {
		return 0
	}
	if n < 0 {
		return -n
	}
	return n
}

// parseLog reads `git log --format=%h%x00%s%x00%cr -z`.
//
// With -z the records are NUL-separated AND the entries are NUL-separated, so commits
// arrive as a flat run of three fields each.
func parseLog(out string) []Commit {
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var out2 []Commit
	for i := 0; i+2 < len(fields); i += 3 {
		short := strings.TrimSpace(fields[i])
		if short == "" {
			continue
		}
		out2 = append(out2, Commit{
			Short:   short,
			Subject: fields[i+1],
			When:    fields[i+2],
		})
	}
	return out2
}
