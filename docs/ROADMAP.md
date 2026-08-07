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

## Phase 3 — Premium ✅ **COMPLETE**

- [x] **Premium detection + lock states** — the rail badges the account, and every gated
      surface names what it is withholding: a locked statement, a locked editorial (with
      its public title and video flag), and a company pull that came back gated. Never a
      raw error, never a silently missing feature (D-006)
- [x] **Editorials** (`d`) — toggles the detail pane between statement and write-up. The
      pane follows the cursor, so reading editorials down a list works. See **D-007a**:
      editorials are markdown with embedded HTML, *not* HTML, and needed their own
      converter — `[TOC]` stripped, playground and video embeds reduced to numbered
      markers, `$$…$$` display math approximated in Unicode
- [x] **Company registry** — 984 companies in one request, and it works **signed out**,
      so a free account can still see which lists exist and how large each one is
- [x] **Company packs** (`c`) — browse-by-company with type-to-filter, then a timeframe.
      The board filters to the pack and sorts by **frequency**, which is what makes it a
      priority list rather than a numbered one. Five timeframes, verified against the
      live API: 30 days / 3 months / 6 months / over 6 months / all time
- [x] **Premium-only filter** (`p`) — cycles all → premium → free. A company pack is
      also the only place some premium problems appear locally, so a pack seeds problems
      the list sync has not reached
- [x] **Timer / stopwatch in the rail** (`t` `T`) — mirrors LeetCode's own

**Milestone met.** Premium parity minus mock assessments (D-006). Verified against the
live API signed out — see `TestLivePremiumSchema`, which is what catches LeetCode moving
a field.

**Packs sync one at a time, on purpose.** The roadmap's original "~500 requests" was off:
984 companies × 5 timeframes is ~5,000 requests and forty minutes at the rate limit, to
answer a question about one company. See **D-006a** for the reasoning and the endpoints,
and **D-006b** for the `companyTagStats` shortcut that was rejected.

Carried forward:
- Premium study plans (Top Interview 150 and friends) are `favoriteQuestionList` under a
  different slug — the same plumbing, no view yet
- The `ASKED BY` column shows company slugs rather than display names, so Meta reads as
  `facebook` there. Consistent with the tag column, which is also slugs

## Interlude — CLI seam ✅ **COMPLETE**

- [x] `internal/solve` — problem-folder layout extracted from the TUI, shared by both
      frontends. `Locate` resolves a slug, folder name, path, or the working directory
- [x] `leetui pull / run / submit / path` — plain-text output, editor-shaped arguments,
      exit codes an editor can branch on (D-015)
- [x] Workspace `.gitignore` for generated drivers, binaries, and `__pycache__`, so
      Phase 4 starts from a clean repository

**Why this and not an nvim plugin or a VS Code extension.** See D-015: the TUI is 41% of
the tree, and a second frontend discards either it or the Go core entirely. A CLI makes an
nvim plugin a keymap instead of a project.

## Interlude — first-use fixes and the todo list ✅ **COMPLETE**

Everything here came from actually using the app, which is the only way any of it would
have surfaced.

- [x] **Two screens** (D-018) — browsing is the full-width list, `enter` opens a problem,
      `esc` comes back to exactly where you were. The statement no longer sits on screen
      taking 40% of the width while you are looking *for* something
- [x] **LeetCode's difficulty colours** (D-017) — teal / amber / red, read off their dark
      theme. Overrules the green-and-red rule, deliberately and in one contained place
- [x] **The workbench strip** (D-016) — the solution's path and whether a save re-runs.
      leetui is the LeetCode side of the desk; the editing happens in your own pane
- [x] **`f` creates the file** (D-019) and toasts its path, without launching anything
- [x] **`ACC` is a percentage** (D-020) — the sparkline needed a legend nobody had
- [x] **A verdict updates the board** (D-020) — `status` had one writer, the list sync
- [x] **Banded rows** (D-021) — vertical rules could not carry the eye across a row
- [x] **LOCKED depends on the account** (D-021), not on the problem
- [x] **Todo list** (D-022) — `m` / `M` in the app, `leetui todo` from a script, with
      `--json` for agents. See `docs/AGENTS.md`
- [x] **`LEETUI_CONFIG_DIR`** (D-022a) — after the test suite overwrote a live config
- [x] **State is a glyph** (D-023) — `✓` `◐` `⊘` under a `STATE` header, with an ASCII
      set for terminals that cannot be trusted with the width of `✓`

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
- [ ] 80×24 responsive collapse
- [ ] Opt-in inline images (Kitty / iTerm2)
- [ ] Local JS/TS runner (before Java — see D-004)

## Phase 6 — Terminal compatibility

- [ ] **Finish ASCII mode.** `config.PreferASCII` and `theme.Glyphs` landed with D-023 and
      cover the state and todo marks, but the frame is still box-drawing (`╭─┼─╯`). A
      terminal that cannot render `✓` cannot render those either, so the promise is
      currently half kept. `components.Frame` needs an ASCII rule set
- [ ] `NO_COLOR`, 256-colour and monochrome profiles
- [ ] `--ascii` flag alongside the config key

## Phase 7 — The website

The last piece. Nothing here is started.

- [ ] Decide what it is *for* — a landing page for the tool, or a place your solved
      problems and notes are published from. These are different projects
- [ ] If it publishes: the workspace already holds a `README.md` per problem with the
      statement, and `notes.md` the user wrote. Phase 4's git integration is what would
      get them somewhere a site can read
- [ ] Install story: `go install`, a Homebrew tap, and prebuilt binaries per platform
- [ ] Screenshots or an asciinema cast — the board and the flip are the pitch

**Open question, worth settling before any code:** a site that publishes solutions is a
different thing from a site that advertises the tool. The first needs the git phase and a
decision about what is public; the second needs a domain and half a day. Do not start
until which one is chosen.

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
| `$EDITOR` | unset in the shell. leetui's own config sets `editor = "nvim"`, which takes precedence, so leetui is covered. Setting `EDITOR` in the shell profile is still worth doing for everything else. |
| config path | `~/Library/Application Support/leetui/config.toml` on macOS — **not** `~/.config`, which is what `os.UserConfigDir()` returns here. `LEETUI_CONFIG_DIR` overrides it (D-022a). |
