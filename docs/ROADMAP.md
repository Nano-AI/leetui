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

## Phase 4 — Git ✅ **COMPLETE**

- [x] **`vcs`** — a thin, tested layer over the `git` binary (D-011): porcelain-v2 status
      with ahead/behind, a guarded commit, and an explicit push. Shells out so the user's
      credentials, signing, hooks, and `includeIf` config all still apply
- [x] **Commit on Accepted** — from the app *and* from `leetui submit`, through one
      `solve.Commit` so the two produce identical history. Refuses on the wrong branch or
      a detached HEAD, commits only the paths it was asked about, and stays quiet when the
      workspace is not a repository or the files already match `HEAD`
- [x] **No trailers on solution commits** (D-024) — those record the user solving a
      problem, and a co-author line would be a false attribution on their repository
- [x] **README.md generation per problem folder** — already landed with D-010; the
      statement is regenerated on every sync, `notes.md` never is
- [x] **Git pane** (`v`, D-025) — branch, upstream, what is uncommitted in words rather
      than porcelain codes, and the last eight commits. Push lives in here and nowhere
      else, behind a confirmation that names the remote's **URL**

**Not done, deliberately:** leetui never runs `git init` for you. A stray `.git` in a
directory someone did not mean to version is a real annoyance, and the default workspace
is inside their home directory. The pane states the situation and prints the two commands.

## Phase 5 — Polish ✅ **COMPLETE**

- [x] **Settings view** (`V`) and the **`:` command palette** — two views onto ONE
      registry (`internal/config/settings.go`), so they cannot disagree about what
      exists. `:set default_lang go`, `tab` completes to the longest common prefix,
      booleans toggle on a bare name. Both write through to `config.toml`
- [x] **Keymap remapping** — `:set keys.run x`, over the same registry. A duplicate
      binding is refused rather than resolved: it would silently shadow one of the two
      actions, and which one wins depends on map order
- [x] **Tags and hints are spoilers** — both hidden by default, `z` and `Z` reveal.
      "hash-table" beside a problem answers the question the problem is asking, and
      reading it is not a decision. Search by tag still works, which is the use that
      spoils nothing
- [x] **The help scrolls** rather than dropping bindings. Two columns bought one
      doubling and then stopped; this screen grows every time a feature does
- [x] **80×24 responsive collapse** — the floor every emulator still defaults to, so
      `TestFramesFitTerminal` runs EVERY screen at it rather than a sample. That found
      two: the settings list did not fit at all, and the command palette drew two rows
      the layout had not reserved, pushing the board's last row off the bottom
- [x] **Inline images (Kitty / iTerm2)** — `leetui image <problem> [n]`, plus tested
      encoders for both protocols and the tmux passthrough wrapper. `leetui doctor`
      reports when tmux is swallowing the escape, which is otherwise completely silent
- [x] **JPEG is re-encoded** — kitty's `f=100` means PNG, LeetCode serves plenty of
      JPEGs, and handing one over draws nothing and reports nothing. Found by running it
- [ ] **Images INSIDE the statement pane.** Deliberately not done: an image escape
      occupies cells the layout must have reserved, and a protocol the terminal
      mis-measures shears the frame — the same failure as an unwrapped long line, but
      impossible to recover from without quitting. As a command there is no frame to
      shear, and it composes into a tmux pane the way `run --watch` does
- [x] **Local JS/TS runner** (D-004 said this was the cheap one; it was). One driver and
      one binary for both: Node strips TypeScript's types itself from v23, so `solution.ts`
      needs no transpiler, no tsconfig and no second toolchain. JavaScript also takes
      **design problems**, which Go and C++ decline — a design problem in JS is a
      constructor function and its prototype, with no type declarations to reconstruct

## Phase 6 — Terminal compatibility ✅ **COMPLETE**

- [x] **ASCII mode is finished** (D-026) — `theme.Charset` covers the bezel, the column
      rules, the dashed divider, the inline separators, and the cursor bar, alongside the
      glyphs D-023 already handled. `TestASCIIModeDrawsNoBoxCharacters` renders every
      screen and fails on any of the 18 characters it is meant to replace
- [x] **`NO_COLOR`, 256-colour and monochrome profiles** — `theme.DetectProfile`. Nothing
      in the app was ever encoded in colour alone, so 256-colour needs nothing but
      quantising. Monochrome loses the amber bezel that marks focus, which was the one
      real gap: the focused pane's title takes a marker instead
- [x] **`--ascii` flag** alongside `ui.ascii`. The flag wins outright — it is the escape
      hatch for a terminal the detection reads wrong, and someone who typed it has decided

## Phase 7 — The website ✅ **the page is live; the rest needs a purchase or a tag**

The last piece. The page is up; the install story around it is not.

- [x] **Decide what it is for** — a landing page for the tool. See below
- [x] **The page**, in `site/`, deployed to Pages by Actions
- [~] **A domain** — the workflow writes a `CNAME` from the `LEETUI_DOMAIN` repository
      variable, so pointing the site at one is: buy it, set the variable, add the DNS
      record. Left unset the site keeps working at `nano-ai.github.io/leetui`. Buying a
      domain is the one step nothing here can do
- [x] **Prebuilt binaries per platform** — `.github/workflows/release.yml` builds five
      targets on a `v*` tag, with a checksums file. `CGO_ENABLED=0` throughout, which is
      D-009's pure-Go SQLite being cashed in: every target cross-compiles from one runner
      with no C toolchain. Verified locally on all five before the workflow was written
- [x] **`--version`**, stamped by the release build
- [x] **A Homebrew tap, automated** — the formula lives at
      `packaging/homebrew/leetui.rb`, and the release workflow fills its four sha256
      lines from the checksums it just computed and pushes it to the tap. Doing that by
      hand is four copy-pastes per release, which is four chances to ship a formula that
      will not install. **Two manual steps remain, both one-time:** create
      `Nano-AI/homebrew-tap`, and set a `HOMEBREW_TAP_TOKEN` secret. Without the secret
      the step is skipped, so a release never fails on a tap that does not exist yet
- [x] **LICENSE — MIT**, with `NOTICE` carrying leetgo's. MIT because the vendored
      drivers are MIT and the README already credited them as such, so anything more
      restrictive would have been incompatible with code already in the tree. Easily
      changed before a tag if the owner wants otherwise; nothing depends on it yet
- [x] **Screenshot** — captured from the real `View()` into the README, so it cannot
      drift from the product. Re-take with `LEETUI_SHOTS=1 go test ./internal/tui`
- [x] **An asciinema cast** — `docs/demo.cast`, generated from the real `View()` through
      the real `Update()`. No TTY was needed: the v2 format is a JSON header and one array
      per write, and a recorder is only a program that timestamps what a process wrote.
      Regenerate with `LEETUI_CAST=1 go test ./internal/tui -run TestRecordCast`, so it
      cannot go stale the way a hand-recorded session does

**Settled 2026-08-06: this repository's site advertises the tool.** The two are different
projects and they belong in different repositories. A site that *publishes solutions*
reads a workspace — `~/leetcode`, with a `README.md` and `notes.md` per problem — so it
belongs there, next to the thing it publishes, and it needs a separate decision about what
is public. Nothing about that is blocked now that Phase 4 has landed; it is just not this.

`site/index.html` is one self-contained file with no build step, deployed by
`.github/workflows/pages.yml` on changes to `site/` only. It applies the Departure Board
system rather than reinventing one: the six tokens, one mono family, hierarchy from weight
→ case → colour → indentation, and green used exactly once — on the verdict, which is the
whole reason a verdict lands.

**One manual step:** GitHub → Settings → Pages → Source: **GitHub Actions**. The workflow
cannot enable it for you.

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
