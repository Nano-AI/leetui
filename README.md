# leetui

A terminal client for LeetCode. Browse, search, solve, run, and submit without leaving
the terminal — with full Premium parity and optional GitHub sync.

**Status: Phase 1 complete — browse and search work against real LeetCode data.**
Sync 4,013 problems locally, search them instantly offline, read statements rendered in
the terminal. Solving, running, submitting, and the premium surfaces are next.
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
| `0` `esc` | clear filters |
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
- **Local execution: Go, Python, C++, Rust** — via `leetgo` behind an adapter. Java and JS/TS are remote-only
- **Full Premium parity** — company packs, editorials, premium problems. No mock assessments
- **SQLite + FTS5** (`modernc.org/sqlite`, pure Go) for instant offline search
- **Company sync inverts company→problem** — ~500 requests instead of ~3,600
- **Disk layout is problem-first** — `0146-lru-cache/{solution.go, README.md, notes.md}`
- **Git** — auto-commit on Accepted, push only on an explicit keypress
- **Editing** — delegate to `$EDITOR`, plus a file watcher for external editors
- **Keys** — vim-first, arrows always work, everything remappable

---

## Layout

```
cmd/leetui/          entrypoint
internal/
  config/  auth/  leetcode/  store/  runner/  render/  workspace/  vcs/
  tui/
    theme/       the only place hex values are allowed to appear
    components/  flap (the signature), sparkline
    app.go       root model, layout, global keys
docs/                DECISIONS · DESIGN · ARCHITECTURE · ROADMAP
```

---

## Credits

Local codegen and execution build on [`leetgo`](https://github.com/j178/leetgo) (MIT).
TUI stack is [Charm](https://charm.sh): Bubbletea, Lipgloss, Bubbles, Glamour.

leetui is an unofficial client and is not affiliated with LeetCode.
