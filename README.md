# leetui

A terminal client for LeetCode. Browse, search, solve, run, and submit without leaving
the terminal — with full Premium parity and optional GitHub sync.

**Status: Phase 3 complete — browse, solve, and the premium surfaces all work.**
Sync 4,013 problems locally and search them instantly offline. Edit in your editor, run
against the examples without leaving the terminal (Python, Go, C++), and submit to the
judge. Company packs, editorials, and the premium filter are in. Git integration is next.
See [`docs/ROADMAP.md`](docs/ROADMAP.md).

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
| `1` `2` `3` | toggle easy / medium / hard (board focus) |
| `1`–`9` | open image N (detail focus) |
| `u` | cycle all → unsolved → solved |
| `p` | cycle all → premium → free |
| `0` `esc` | clear filters |
| `e` `r` `s` | edit / run locally / submit |
| `l` `E` | choose language / editor |
| `c` | browse company lists, then a timeframe |
| `d` | read the official editorial |
| `S` | sync (press again to pause — it resumes where it stopped) |
| `a` | sign in |
| `o` | open the problem in your browser |
| `t` `T` | start-stop timer / reset |
| `?` `q` | help / quit |

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
╭──────────────────────────────────────────────────────────────────────────────╮
│ L E E T U I                                ⏱ 00:12:04 ┊ ◆ premium ┊ ada     │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ PROBLEMS ─────────────────────────────────────────────── "cache" ┊ 12 ─╮
│ #      │ PROBLEM                              │ DIF │ AC  │ STATE   │
├────────┼──────────────────────────────────────┼─────┼─────┼─────────┤
│ ▌ 0146 │ LRU Cache                            │ MED │ ▇▃▁ │ SOLVED  │
│   0460 │ LFU Cache                            │ HRD │ █▄▁ │ TRIED   │
│   1650 │ Lowest Common Ancestor of a Tree III │ MED │ ██▅ │ LOCKED  │
╰────────┴──────────────────────────────────────┴─────┴─────┴─────────╯
╭─ PROBLEM ───────────────── 42.7% ACCEPTED ─╮╭─ SUBMISSIONS ─────────────╮
│ 146. LRU Cache  MED                        ││ 0146 go       ▚▚▚▚▚▚▚▚▚▚  │
│ design ┊ hash-table ┊ linked-list          ││ 0042 cpp      A C C E P T… │
│ ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ ││ 0001 python3  W R O N G  … │
│ Design a data structure that follows the   ││                            │
│ constraints of a Least Recently Used cache ││                            │
╰────────────────────────────────────────────╯╰────────────────────────────╯
 / search ┊ 1·2·3 difficulty ┊ u unsolved ┊ S sync ┊ ? keys ┊ q quit
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
- **Local execution: Python, Go, C++** — vendored drivers, zero new dependencies. Everything else edits and submits normally and runs on the judge
- **Full Premium parity** — company packs, editorials, premium problems. No mock assessments
- **SQLite + FTS5** (`modernc.org/sqlite`, pure Go) for instant offline search
- **Company packs sync one at a time** — 984 companies × 5 timeframes is ~5,000 requests; one pack is ~24
- **Solution files carry editor scaffolding** — imports and driver types above `@leetui code=start`; only the marked region is submitted
- **Disk layout is problem-first** — `0146-lru-cache/{solution.go, README.md, notes.md}`
- **Git** — auto-commit on Accepted, push only on an explicit keypress
- **Editing** — delegate to `$EDITOR`, in a pane beside leetui where the terminal allows it; a file watcher re-runs on save
- **Keys** — vim-first, arrows always work, everything remappable

---

## Layout

```
cmd/leetui/          entrypoint
internal/
  config/  auth/  leetcode/  store/  syncer/  render/  editor/  workspace/
  runner/            interfaces, language registry, and vendored drivers/
  tui/
    theme/           the only place hex values are allowed to appear
    components/      frame (the bezel), flap (the signature), sparkline
    doc.go           the file map for this package — read it before adding one
docs/                DECISIONS · DESIGN · ARCHITECTURE · ROADMAP
```

---

## Credits

Local codegen and execution build on [`leetgo`](https://github.com/j178/leetgo) (MIT).
TUI stack is [Charm](https://charm.sh): Bubbletea, Lipgloss, Bubbles, Glamour.

leetui is an unofficial client and is not affiliated with LeetCode.
