package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"golang.org/x/term"
)

// Sign-in outside the TUI.
//
// The docs used to say "there is no flag that takes a cookie", and on a machine where
// the full-screen app is awkward to reach (a headless box, a container, an SSH session
// with a broken terminal) that meant there was no way to sign in at all. It also meant
// no way to debug sign-in without the UI in the way, which is exactly the situation a
// keychain failure creates.
//
// Every route ends at the same two steps as the TUI: ask LeetCode whether the cookies
// work, and only then store them. A `login` that reported success on a typo would be
// worse than no `login` at all, because a script would believe it.

// loginTimeout bounds the verification round trip.
const loginTimeout = 20 * time.Second

func runLogin(a *app, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui login", flag.ContinueOnError)
	session := fs.String("session", "", "the LEETCODE_SESSION cookie")
	csrf := fs.String("csrf", "", "the csrftoken cookie")
	stdin := fs.Bool("stdin", false, "read a cookie header, cURL command, or JSON blob from stdin")
	showStatus := fs.Bool("status", false, "report who is signed in and where the credentials are kept")

	if err := fs.Parse(args); err != nil {
		return exitProblem, err
	}

	if *showStatus {
		return loginStatus(a, os.Stdout)
	}

	creds, err := gatherCredentials(*session, *csrf, *stdin)
	if err != nil {
		return exitProblem, err
	}

	return storeVerified(creds, os.Stdout)
}

func runLogout(a *app, args []string) (int, error) {
	fs := flag.NewFlagSet("leetui logout", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return exitProblem, err
	}
	if err := auth.Clear(); err != nil {
		return exitProblem, err
	}
	fmt.Fprintln(os.Stdout, "Signed out.")
	return exitOK, nil
}

// gatherCredentials collects the two cookies from whichever route the user chose.
func gatherCredentials(session, csrf string, stdin bool) (auth.Credentials, error) {
	switch {
	case stdin:
		blob, err := io.ReadAll(os.Stdin)
		if err != nil {
			return auth.Credentials{}, fmt.Errorf("read stdin: %w", err)
		}
		// Parse rather than Clean: a piped blob is a whole cookie header or a `pass`
		// entry, and the point of --stdin is not having to split it first.
		return auth.Parse(string(blob))

	case session != "" || csrf != "":
		c := auth.Credentials{Session: auth.Clean(session), CSRF: auth.Clean(csrf)}
		if c.Session == "" {
			return c, errors.New("--csrf given without --session")
		}
		if c.CSRF == "" {
			return c, errors.New("--session given without --csrf")
		}
		return c, nil

	default:
		return promptCredentials()
	}
}

// promptCredentials asks for the two cookies on a terminal.
//
// Echo is off, because this is a shared-screen risk in exactly the way the TUI's masked
// fields are. Unlike the TUI there is no reveal toggle: a shell has scrollback, and a
// session token echoed into it outlives the moment the user wanted to see it.
func promptCredentials() (auth.Credentials, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return auth.Credentials{}, errors.New("not a terminal: use --stdin, or --session and --csrf")
	}

	fmt.Fprintln(os.Stderr, "On leetcode.com: devtools → Application → Cookies")
	fmt.Fprintln(os.Stderr, "Input is hidden. A whole cookie header works in the first prompt.")
	fmt.Fprintln(os.Stderr)

	first, err := promptSecret(fd, "LEETCODE_SESSION: ")
	if err != nil {
		return auth.Credentials{}, err
	}
	// Same courtesy as the form: if what was pasted holds both cookies, take both
	// rather than asking again for one already in hand.
	if c, ok := auth.Sniff(first); ok {
		return c, nil
	}

	second, err := promptSecret(fd, "csrftoken:        ")
	if err != nil {
		return auth.Credentials{}, err
	}

	c := auth.Credentials{Session: auth.Clean(first), CSRF: auth.Clean(second)}
	if !c.Valid() {
		return c, errors.New("both cookies are required")
	}
	return c, nil
}

func promptSecret(fd int, label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", strings.TrimSpace(label), err)
	}
	return string(b), nil
}

// storeVerified checks the cookies against LeetCode, then stores them, then says where.
func storeVerified(c auth.Credentials, w io.Writer) (int, error) {
	if !c.Valid() {
		return exitProblem, errors.New("both LEETCODE_SESSION and csrftoken are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	st, err := leetcode.New(leetcode.WithCredentials(c)).Status(ctx)
	if err != nil {
		return exitProblem, fmt.Errorf("could not reach LeetCode to check these cookies: %w", err)
	}
	if !st.IsSignedIn {
		return exitProblem, errors.New("LeetCode rejected these cookies; they may have expired")
	}

	c.Username = st.Username
	backend, err := auth.Store(c)
	if err != nil {
		return exitProblem, err
	}

	fmt.Fprintf(w, "Signed in as %s.\n", st.Username)
	reportBackend(w, backend)
	return exitOK, nil
}

// loginStatus answers "am I signed in, and where does that live".
func loginStatus(a *app, w io.Writer) (int, error) {
	creds, backend, err := auth.Load()
	if errors.Is(err, auth.ErrNoCredentials) {
		fmt.Fprintln(w, "Not signed in. Run: leetui login")
		return exitOK, nil
	}
	if err != nil {
		return exitProblem, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	st, err := leetcode.New(leetcode.WithCredentials(creds)).Status(ctx)
	switch {
	case err != nil:
		// Say what is stored anyway. "Cannot reach LeetCode" and "your session is
		// dead" are different problems, and only one of them is fixed by signing in.
		fmt.Fprintf(w, "Stored credentials found, but LeetCode could not be reached: %v\n", err)
	case !st.IsSignedIn:
		fmt.Fprintln(w, "Stored session is no longer valid. Run: leetui login")
	default:
		fmt.Fprintf(w, "Signed in as %s.\n", st.Username)
	}

	reportBackend(w, backend)
	return exitOK, nil
}

// reportBackend names the store, and says what it means when the answer is a file.
//
// This is the line that has to exist. A tool that falls back from the keychain and does
// not say so leaves the user believing something about their own machine that is not
// true, which is the complaint that has followed the GitHub CLI's silent fallback for
// years. Saying it costs two lines.
func reportBackend(w io.Writer, b auth.Backend) {
	switch b {
	case auth.BackendEnv:
		fmt.Fprintf(w, "Credentials come from %s and %s; nothing is stored on disk.\n",
			auth.EnvSession, auth.EnvCSRF)
		return
	case auth.BackendKeyring, auth.BackendHelper:
		fmt.Fprintf(w, "Kept in %s.\n", b.Describe())
		return
	}

	path, err := auth.FilePath()
	if err != nil {
		fmt.Fprintf(w, "Kept in %s.\n", b.Describe())
		return
	}
	fmt.Fprintf(w, "Kept in %s: %s\n", b.Describe(), path)
	fmt.Fprintln(w, "No OS keychain was available. To use one instead, start a Secret Service")
	fmt.Fprintln(w, "(gnome-keyring, kwallet), or set [auth] helper in config.toml.")
}
