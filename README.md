# leetui

A terminal client for LeetCode. Browse, search, solve, run, and submit without leaving
the terminal — with full Premium parity and optional GitHub sync.

**Status: v0.1.0 released.** All seven phases complete. Install with Homebrew, `go
install`, or a prebuilt binary for macOS, Linux, or Windows.
Sync 4,013 problems locally and search them instantly offline. Edit in your editor, run
against the examples without leaving the terminal (Python, Go, C++, JS/TS), and submit to the
judge. Company packs, editorials, and the premium filter are in. An accepted solution
commits itself; pushing is yours to press. See [`docs/ROADMAP.md`](docs/ROADMAP.md).

---

## Read these first

New session working on this repo? Read in this order — they exist so decisions are not
re-derived or re-asked:

| File | What it holds |
|------|---------------|
| [`docs/DECISIONS.md`](docs/DECISIONS.md) | Every settled architecture decision, why, what it costs, what would reverse it. **Treat as settled.** |
| [`docs/DESIGN.md`](docs/DESIGN.md) | The Departure Board visual system — palette, type treatments, motion, copy rules |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Package map, data flow, key seams, invariants |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Phase status. **Update as work lands.** |
| [`docs/AGENTS.md`](docs/AGENTS.md) | The scriptable CLI surface — exit codes, JSON, and what not to automate |

---

## Run it

```sh
go run ./cmd/leetui
```

First run shows an empty board. Press **`S`** to sync — about 4,000 problems, a couple of
minutes at the default 2 requests/second. After that everything is local and instant.

Press **`a`** to sign in with your LeetCode session cookies if you want your solved
status and premium content. Browsing and search work signed out.

| Key | Does |
|-----|------|
| `j` `k` / arrows | move |
| `tab` | cycle pane focus |
| `/` | search titles, tags, and statements |
| `1` `2` `3` | toggle easy / medium / hard (on the list) |
| `enter` `esc` | open a problem / back to the list |
| `1`–`9` | open marker N (on a problem) |
| `u` | cycle all → unsolved → solved |
| `p` | cycle all → premium → free |
| `m` `M` | mark a problem / show just your list |
| `0` `esc` | clear filters |
| `f` | create the solution file and show its path |
| `e` `r` `s` | edit / run locally / submit |
| `l` `E` | choose language / editor |
| `c` | browse company lists, then a timeframe |
| `d` | read the official editorial |
| `S` | sync (press again to pause — it resumes where it stopped) |
| `a` | sign in |
| `o` | open the problem in your browser |
| `t` `T` | start-stop timer / reset |
| `?` `q` | help / quit |

### From an editor, or a script

`leetui` with no arguments opens the app. The subcommands drive the same core, so an
editor never needs a plugin:

```sh
leetui pull two-sum        # folder, statement, scaffolded solution, test cases
leetui run                 # from inside the problem folder
leetui run path/to/solution.py
leetui submit two-sum
leetui path two-sum        # prints the folder, for scripting

leetui todo add two-sum --note "from the JD"
leetui todo --json         # stable array, for agents

leetui run --watch         # stays open, re-runs on save — put it in its own pane
leetui image two-sum 1     # draw a figure, on kitty / iTerm2 / WezTerm / Ghostty
leetui doctor              # what this machine can do, and how to fix what it cannot

leetui --debug             # trace requests to ~/.local/share/leetui/debug.log
LEETUI_DEBUG=1 leetui run  # the same, for a subcommand
```

The trace records what was sent, what came back, and the whole body of anything that
failed to decode. Session cookies are redacted before they reach it, so it is safe to
attach to a bug report.

A problem can be a slug, a folder name, a path to either, or omitted to mean the current
directory — so from inside nvim the buffer you are looking at is enough:

```lua
vim.keymap.set("n", "<leader>lr", ":!leetui run %<CR>")
vim.keymap.set("n", "<leader>ls", ":!leetui submit %<CR>")
```

Exit codes are the contract: `0` passed, `1` ran and was wrong, `2` could not run. The
full surface, including what an agent should *not* automate, is in
[`docs/AGENTS.md`](docs/AGENTS.md).

### A list of problems to get to

`m` marks the problem under the cursor; `M` shows just your list, oldest first. The same
list is writable from a script, so an agent can queue problems from a job description and
they are waiting on the board next time you open it:

```sh
leetui todo add two-sum group-anagrams --note "phone screen prep"
```

### Your solutions, in git

Make the workspace a repository and leetui keeps it current:

```sh
cd ~/leetcode && git init
```

From then on an **Accepted** verdict commits itself — the solution, the statement, the
test cases, and your notes — with a subject you would have written yourself:

```
solve(0146): lru cache — go, 58ms, beats 91.2%
```

No co-author trailer and no session link: it is your work, and `git log` on a solutions
repo is something people show other people.

**Pushing is never automatic.** Press `v` for the repository view — branch, what is
uncommitted, what the remote has not got, and the last few commits. `p` there asks once,
naming the remote's URL, and publishes only when you answer `y`. leetui will not pick a
remote for you, and it never runs `git init` on your behalf.

Turn any of it off in `config.toml`:

```toml
[git]
commit_on_accepted = true   # false to commit by hand
commit_notes       = true   # false to keep notes.md out of the repo
branch             = "main" # refuse to commit anywhere else
remote             = ""     # empty means the branch's own upstream
```

---

## Install

One line, nothing installed first:

```sh
curl -fsSL https://raw.githubusercontent.com/Nano-AI/leetui/main/install.sh | sh
```

It picks the binary for your machine, **verifies it against the release's own
checksums**, and installs it somewhere on your PATH. [Read it first](install.sh) — that
goes for every install script on the internet, including this one.

```sh
brew tap Nano-AI/tap
brew trust nano-ai/tap    # Homebrew gates third-party taps
brew install leetui
```

With Go:

```sh
go install github.com/Nano-AI/leetui/cmd/leetui@latest
```

`go install` puts binaries in `$(go env GOPATH)/bin`, which is **often not on your PATH**.
If `leetui` is not found afterwards, add it:

```sh
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Without either, take a prebuilt binary from
[Releases](https://github.com/Nano-AI/leetui/releases) — macOS (Apple silicon and Intel),
Linux (x86-64 and arm64), and Windows. One static file, no runtime to install.

```sh
leetui doctor    # what this machine can and cannot do, and how to fix it
```

---

## A demo

[`docs/demo.cast`](docs/demo.cast) is an asciinema recording — search, open, reveal tags,
settings, help:

```sh
asciinema play docs/demo.cast
```

It is generated from the real `View()`, driven through the real `Update()`, so it cannot
drift from the product the way a hand-recorded session does:

```sh
LEETUI_CAST=1 go test ./internal/tui -run TestRecordCast
```

---

## Working on leetui

```sh
go test ./...                                              # offline suite
LEETUI_LIVE=1 go test ./internal/syncer ./internal/tui -v  # against leetcode.com
go run ./cmd/leetui --no-motion                            # animation off
```

Config is written to `~/.config/leetui/config.toml`; the database lives in
`~/.local/share/leetui/`. Session cookies go to the OS keychain, never to disk.

---

## Shape of the thing

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ L E E T U I                                                              ⏱ --:--:-- ┊ signed out │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─ PROBLEMS ─────────────────────────────────────────────────────────────────────────────────── 4 ─╮
│ #      │ TODO │ PROBLEM                                  │ DIF │ ACC  │ STATE │ ASKED BY         │
├────────┼──────┼──────────────────────────────────────────┼─────┼──────┼───────┼──────────────────┤
│ ▌ 0001 │      │ Two Sum                                  │ ESY │ 58%  │   ✓   │                  │
│   0042 │      │ Trapping Rain Water                      │ HRD │ 61%  │       │                  │
│   0146 │      │ LRU Cache                                │ MED │ 43%  │   ◐   │                  │
│   1650 │      │ Lowest Common Ancestor of a Binary Tree… │ MED │ 80%  │   ⊘   │                  │
│        │      │                                          │     │      │       │                  │
╰────────┴──────┴──────────────────────────────────────────┴─────┴──────┴───────┴──────────────────╯
 enter open ┊ m mark ┊ M my list ┊ / search ┊ c companies ┊ 1·2·3 difficulty ┊ u unsolved ┊ S sync
```

Browsing is the list and nothing else, full width — the same reason
leetcode.com/problemset is a table and nothing else. `DIF` uses LeetCode's own
colours, teal / amber / red, so it reads without a legend. `STATE` is `✓` solved,
`◐` tried, `⊘` locked; rows alternate a faint band so the eye can track across
one. Terminals that cannot be trusted with the width of `✓` get `x` `~` `-`
automatically.

`enter` opens the problem; `esc` comes back to exactly where you were.

```
╭─ PROBLEM ───────────────── 42.7% ACCEPTED ─╮╭─ SOLUTION ────────── GO ─╮
│ 146. LRU Cache  MED                        ││ ~/leetcode/0146-lru-…/   │
│ design ┊ hash-table ┊ linked-list          ││ watching · saves re-run  │
│ ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ │╰──────────────────────────╯
│ Design a data structure that follows the   │╭─ RUN ────────── 1 FAILED ─╮
│ constraints of a Least Recently Used cache ││ case 1  FAIL             │
│                                            ││ want [0,1]   got [9,9]   │
╰────────────────────────────────────────────╯╰──────────────────────────╯
 e edit ┊ r run ┊ s submit ┊ d editorial ┊ l lang ┊ esc back
```


Press **`c`** for the premium loop the website is built around — pick a company, pick how
recently it asked, work the list. Typing narrows immediately; there is only one thing to
do with a keystroke in a list of 984 companies.

```
╭─ COMPANY ───────────────────────────────────── 3 OF 984 ─╮
│ goo                                                      │
╰──────────────────────────────────────────────────────────╯
╭─ LISTS ──────────────────────────────────────── PREMIUM ─╮
│ ▌ Google                             199 stored of 2335  │
│   Goldman Sachs                            412 problems  │
│   Google Cloud                              38 problems  │
╰──────────────────────────────────────────────────────────╯
 type to narrow ┊ ↑↓ move ┊ enter pick a timeframe ┊ esc back
```

The chosen pack filters the board and sorts it by **frequency**, so what that company asks
most sits at the top. That ordering is the whole reason a pack beats a tag filter.

Solution files are written so your editor can actually resolve them — LeetCode's snippets
have no imports, no package clause, and no `ListNode`, because the judge supplies all of
that. Only the marked region is submitted:

```cpp
// 1. Two Sum · Easy
// https://leetcode.com/problems/two-sum/
//
// Everything above the marker is local scaffolding, for your editor
// and the local runner. Only the marked region is submitted.

#include "leetui_driver.h"

// @leetui code=start
class Solution {
public:
    vector<int> twoSum(vector<int>& nums, int target) {
        
    }
};
// @leetui code=end
```

vscode-leetcode's `@lc code=start` markers are read too, so a workspace built with that
extension works here unchanged.

**leetui is the LeetCode side of the desk, not the editor.** It owns browsing, search,
company packs, statements, editorials, running, and submitting. The code happens in
whatever you already have open — nvim in the next tmux pane, a VS Code terminal, an editor
on another monitor. So the side column always names the file and says whether a save will
re-run it:

```
╭─ SOLUTION ───────────────────────── PYTHON3 ─╮
│ ~/leetcode/0001-two-sum/solution.py          │
│ watching · saves re-run the tests            │
╰──────────────────────────────────────────────╯
```

Open that path wherever you like. Save. leetui re-runs the tests and shows the diff,
without you switching back to it.

**Reading and writing at the same time.** Inside tmux (or zellij, WezTerm, Kitty), `e`
opens your editor in a pane *beside* leetui rather than taking the terminal — the
statement stays on screen and saving re-runs the tests, so you never switch back. A GUI
editor gets the same deal via its own window. With a terminal editor and nowhere to put a
pane, leetui hands over the terminal, opens `README.md` alongside the solution, and runs
the tests when you quit. Turn any of it off with `editor_pane`, `open_statement`, and
`run_after_edit`.

A submission queue is a departure board — work is submitted, sits pending, gets judged,
resolves. So verdicts **flip** into place rather than appearing. That flip is the app's
entire motion budget and its only loading state; there are no spinners in leetui.

Amber is the system speaking. Bone is content. **Green and red belong to the judge alone**
— nothing else in the app is ever green or red, which is what makes a verdict land.

---

## Decisions at a glance

Full reasoning in [`docs/DECISIONS.md`](docs/DECISIONS.md).

- **Go + Bubbletea/Lipgloss/Glamour** — single static binary
- **Auth** — paste `LEETCODE_SESSION` + `csrftoken`, browser import as convenience, stored in the OS keychain
- **Run local, submit remote** — tight loop stays offline, correctness of record comes from the judge
- **Local execution: Python, Go, C++, JavaScript, TypeScript** — vendored drivers, zero new dependencies. Everything else edits and submits normally and runs on the judge
- **Full Premium parity** — company packs, editorials, premium problems. No mock assessments
- **SQLite + FTS5** (`modernc.org/sqlite`, pure Go) for instant offline search
- **Company packs sync one at a time** — 984 companies × 5 timeframes is ~5,000 requests; one pack is ~24
- **Solution files carry editor scaffolding** — imports and driver types above `@leetui code=start`; only the marked region is submitted
- **Disk layout is problem-first** — `0146-lru-cache/{solution.go, README.md, notes.md}`
- **Git** — auto-commit on Accepted; push only from the repository view, behind a confirmation that names the remote's URL
- **Editing** — delegate to `$EDITOR`, in a pane beside leetui where the terminal allows it; a file watcher re-runs on save
- **Two screens** — the list is the list; `enter` opens a problem, `esc` comes back
- **A todo list you can script** — `m` in the app, `leetui todo add` from anywhere
- **Difficulty in LeetCode's own teal / amber / red** — reads without a legend
- **Every column has a header** — a glyph is only a glyph when something names it
- **Keys** — vim-first, arrows always work, everything remappable

---

## Layout

```
cmd/leetui/          entrypoint
internal/
  config/  auth/  leetcode/  store/  syncer/  render/  editor/  workspace/
  solve/             the seam both frontends use: prepare, run, commit
  vcs/               git, shelled out — status, guarded commit, explicit push
  runner/            interfaces, language registry, and vendored drivers/
  tui/
    theme/           the only place hex values are allowed to appear
    components/      frame (the bezel), flap (the signature), sparkline
    doc.go           the file map for this package — read it before adding one
docs/                DECISIONS · DESIGN · ARCHITECTURE · ROADMAP
```

---

## Credits

MIT licensed — see [`LICENSE`](LICENSE), and [`NOTICE`](NOTICE) for what was adapted in.

Local codegen and execution build on [`leetgo`](https://github.com/j178/leetgo) (MIT).
TUI stack is [Charm](https://charm.sh): Bubbletea, Lipgloss, Bubbles, Glamour.

leetui is an unofficial client and is not affiliated with LeetCode.
