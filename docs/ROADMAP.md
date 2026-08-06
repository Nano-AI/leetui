# Roadmap

Status of each phase. **Update this file as work lands** — it is how a new session
learns what already exists without reading the whole tree.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

---

## Phase 0 — Foundation

- [x] Grill and settle architecture → `DECISIONS.md`
- [x] Visual direction → `DESIGN.md`
- [x] Package map → `ARCHITECTURE.md`
- [x] Go 1.26.5 installed (was missing)
- [x] Go module + Charm deps (bubbletea, lipgloss, bubbles, glamour)
- [x] `tui/theme` — Departure Board tokens, type treatments, verdict + difficulty rendering
- [x] `tui/components` — the flip (signature), acceptance sparkline
- [x] Bubbletea shell: rail + board + detail + queue, `tab` focus, `?` help, timer
- [x] Responsive collapse at 100 / 80 columns
- [x] Render tests at every breakpoint + flip-settles guard
- [x] `config` — TOML load/save, XDG paths, config-driven keymap table

## Phase 1 — Read-only client ✅ **CHECKPOINT — ready to test**

- [x] `auth` — cookie paste (accepts any paste shape) + OS keychain storage
- [x] `leetcode` — GraphQL client, single rate limiter, typed errors, redacted logging
- [x] `store` — schema, migrations, FTS5, faceted queries
- [x] `syncer` — resumable checkpointed problem sync, 429 backoff, cancellable
- [x] `render` — HTML → markdown → Glamour, LaTeX→Unicode, bracket images
- [x] Board view: browse, `/` search, difficulty + status filters
- [x] Detail view: statement via Glamour, premium lock state
- [x] Auth view, sync progress in the rail, timer

**Milestone met.** Verified against the live API: 4,013 problems reported, sync and
resume work, search is instant and relevance-ranked, statements render.

Verify with:

```sh
go test ./...                                          # offline
LEETUI_LIVE=1 go test ./internal/syncer ./internal/tui -v   # against leetcode.com
```

### Design revision (after first real-terminal review)

The first pass was borderless and read as soup. Fixed:

- [x] Every pane carries a bezel; the board is a real grid with column rules,
      `├─┼─┤` under the header and `┴` closing the bottom
- [x] **Removed difficulty-by-bracket-mass on problem IDs** — it duplicated the DIF
      column and made the left margin ragged. IDs are now a plain zero-padded field
      with an amber bar marking the selected row
- [x] Row state is a word (`SOLVED` / `TRIED` / `LOCKED`), not a glyph needing a legend
- [x] Search moved into its own framed panel, with matched terms highlighted in titles
- [x] **First run syncs itself** — an empty database has one useful next action, so the
      app takes it instead of asking for a keypress. `esc` drops to the board early
- [x] `TestBoardIsAClosedGrid` guards that every line is enclosed by the frame

Not yet done in this phase:
- Browser cookie auto-import (paste works; auto-import is the convenience half of D-002)
- Delta sync (a re-sync currently walks the whole list)
- Tag and company filters exist in the store but have no UI yet

## Phase 2 — Solve loop ✅ **COMPLETE**

- [x] `workspace` — problem-first folders (D-010). Solutions, notes, and testcases are
      created but never overwritten; only the derived README regenerates
- [x] `runner` — `Runner`/`Generator` interfaces, language registry, metaData parsing,
      comparator, and the override table for what metaData cannot express
- [x] **Python driver, vendored** — D-005 reversed after measuring the import at 144
      compiled packages. Zero new dependencies. See D-005 for the full reasoning
- [x] `$EDITOR` delegation via `tea.ExecProcess`
- [x] Language picker (`l`) — only what the problem offers, marking local vs judge
- [x] **Editor detection and picker (`E`)** — finds Neovim, Vim, Helix, Kakoune, Emacs,
      Micro, Nano, VS Code / Insiders / Cursor / Windsurf, Zed, Sublime, BBEdit,
      TextMate, and the JetBrains launchers. Checks PATH *and* macOS app bundles, since
      Sublime and JetBrains often ship their CLI helper unlinked. GUI editors carry
      their `--wait` flag or the TUI would repaint over an untouched file. The choice
      persists to `config.toml`, and pressing `e` with nothing configured opens the
      picker rather than guessing
- [x] Local run (`r`) with per-case timeout and crash isolation
- [x] Submit (`s`), judge poll, and the flip driving real verdicts
- [x] **Go and C++ drivers** — vendored alongside Python. `Lang.Local` reflects which
      drivers EXIST rather than which are planned, so the picker never promises a run
      that would fail
- [x] **Result diff panel** — a failure shows input, expected, and actual stacked;
      passing cases collapse to one line. A result from another problem is never shown
- [x] File watcher — polls a modification time (no fsnotify dependency); the first
      sighting only records it, so opening an old solution does not fire a run.
      `watch_solution` in config turns it off
- [x] Design problems (LRU Cache, Min Stack, Trie) — Python runs them; Go and C++
      still decline rather than guessing a shape

**Milestone met.** Solve a problem end to end without leaving the terminal, in Python,
Go, or C++. Every other language edits and submits normally and runs on the judge.

Local execution verified against: two-sum, longest-common-prefix, reverse-linked-list,
maximum-depth-of-binary-tree, remove-duplicates (in-place), and LRU Cache (design).

Carried forward:
- Rust has a language entry but no driver, and `rustc` is not installed here
- Go and C++ decline design problems
- The comparator override table is seeded, not exhaustive — an uncurated mismatch says
  "check on the judge" rather than claiming a wrong answer (D-003)

## Phase 3 — Premium

- [ ] Premium detection + graceful lock states
- [ ] Editorials — HTML → markdown → Glamour, LaTeX approximation
- [ ] Company sync (invert company → problem, throttled, resumable)
- [ ] **Company packs** — pick "Google" and work through Google's list, the way the
      website's premium loop works. The schema is already in place
      (`companies`, `problem_companies` with frequency and timeframe) and the board
      already has an `ASKED BY` column reading from it; what is missing is the sync
      job and a browse-by-company view. Timeframe buckets (6mo / 1yr / 2yr / all)
      come along with the inverted sync for free.
- [ ] Premium-only problems and lists
- [ ] Timer / stopwatch in the rail

**Milestone:** premium parity minus mock assessments (D-006).

## Phase 4 — Git

- [ ] `vcs` — status, guarded commit-on-Accepted, explicit push
- [ ] README.md generation per problem folder
- [ ] Git pane

## Phase 5 — Polish

- [ ] Settings view (all config editable in-app)
- [ ] **`:` command palette that sets config** — `:set default_lang go`,
      `:set workspace ~/code/leetcode`, `:sync companies`. One typed surface over the
      same action table the keymap uses (D-013), so every action is reachable by name
      and discoverable without memorising a key. Writes through to `config.toml`, so
      what you set in the palette survives the session.
- [ ] Keymap remapping UI
- [ ] `NO_COLOR`, `--ascii`, `--no-motion`, 256-color fallback
- [ ] 80×24 responsive collapse
- [ ] Opt-in inline images (Kitty / iTerm2)
- [ ] Local JS/TS runner (before Java — see D-004)

---

## Environment notes

Verified 2026-08-06 on this machine:

| Tool | State |
|------|-------|
| go | 1.26.5 (installed this session via brew) |
| git | 2.50.1 |
| gh | 2.92.0 |
| python3 | 3.14.4 |
| clang++ | 21.0.0 |
| node | 25.9.0 |
| deno | 2.7.14 |
| rustc | **not installed** — D-004 lists Rust as free from leetgo, but local Rust runs need the toolchain. Go/Python/C++ are the working set here until `rustup` is installed. |
| java | pre-9 (rejects `--version`) — remote-only anyway per D-004 |
| nvim | present at /opt/homebrew/bin/nvim |
| `$EDITOR` | unset in the shell. `~/.config/leetui/config.toml` sets `editor = "nvim"`, which takes precedence, so leetui is covered. Setting `EDITOR` in the shell profile is still worth doing for everything else. |
