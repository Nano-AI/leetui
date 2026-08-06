# Architecture

Read `DECISIONS.md` first — this file explains *how* the code is arranged;
that one explains *why* each choice was made.

---

## File size

**Keep files in the 100–200 line range.** One file, one job. A file that has grown past
~200 lines is doing more than one thing and should be split along the seam that made it
grow. `internal/tui/doc.go` carries that package's file map; add to it when you add a
file.

Current: 187 files, ~19,807 lines, mean 105, max 212.

## Package map

```
cmd/leetui/                 main.go dispatch · tui.go the app · cli.go the subcommands
                            cmd_solve.go · cmd_submit.go · report.go plain-text output
internal/
  config/     config.go types · keymap.go bindings · defaults.go · load.go · resolve.go
  auth/       auth.go paste+keychain · browser.go types · detect.go · import.go
              chromium.go · chromium_crypto.go · firefox.go · cookiedb.go
  leetcode/   client.go construction · transport.go GraphQL+errors · api.go queries
              queries.go documents · models.go wire types · submit.go judge · rest.go
              company.go packs+timeframes · editorial.go · queries_premium.go
  store/      open.go · schema.go migrations · state.go checkpoints · fts.go
              types.go rows · write.go upserts · query.go filters · order.go sorts
              get.go · stats.go · company.go registry+packs · editorial.go cache
  syncer/     syncer.go · progress.go · problems.go the resumable job · detail.go
              companies.go registry+one pack · editorial.go
  render/     html.go converter · walk.go dispatch · inline.go · blocks.go
              whitespace.go · text.go · latex.go · glamour.go theme
              editorial.go markdown-with-HTML, a separate door (D-007a)
  runner/     interfaces + language registry, vendored per-language drivers (D-005)
              scaffold.go the two-region solution file · extract.go what gets submitted
              testcase.go example inputs + scraped answers · overrides.go comparators
  solve/      prepare.go problem-folder layout · run.go generate+execute
              locate.go slug from a slug, folder, path, or cwd (D-015)
  editor/     editor.go detection and launch arguments (D-012)
              pane.go tmux/zellij/WezTerm/Kitty splits (D-012a)
  workspace/  problem-folder layout on disk (D-010), never-overwrite writes
  vcs/        (Phase 4) git shell-out: status, commit-on-accepted, push
  tui/
    theme/       tokens.go — the only place hex values appear
                 type.go treatments · verdict.go · difficulty.go
    components/  frame.go the bezel · grid.go rows and rules · flap.go the signature
                 sparkline.go
    (see internal/tui/doc.go for the file map of state and rendering)
```

`internal/` is deliberate: nothing here is a public API, and Go's visibility rule
enforces that.

---

## Data flow

```
                  ┌──────────────┐
   user keypress ─┤ tui.Model    │
                  │ Update(msg)  │
                  └──┬────────┬──┘
                     │        │ tea.Cmd (async, never blocks Update)
                     │        ▼
                     │   ┌─────────────────────────────┐
                     │   │ store (SQLite)  ← fast path │  search, browse, cached bodies
                     │   │ leetcode (HTTP) ← slow path │  submit, sync, editorials
                     │   │ runner  (exec)  ← subprocess│  local run
                     │   │ vcs     (exec)  ← subprocess│  commit, push
                     │   └───────────┬─────────────────┘
                     │               │ tea.Msg
                     └───────────────┘
```

**Rule: `Update` never blocks.** Every network call, subprocess, and disk sync returns a
`tea.Cmd` and comes back as a `tea.Msg`. This is what keeps the flip animation smooth
while a submission is in flight.

**Rule: views read the store, not the network.** Browsing and searching hit SQLite only.
The network populates SQLite through explicit sync jobs. This is what makes search feel
instant and what keeps the rate limiter (D-008) meaningful.

---

## Key seams

### `runner` — the execution seam (D-005)

```go
type Generator interface { Generate(ctx, p Problem, lang Lang, dir string) error }
type Runner interface {
    Run(ctx, dir string, lang Lang, cases []TestCase, rule Rule) (Result, error)
    Supports(lang Lang) bool   // false → caller falls back to remote judge (D-004)
}
```

**D-005 was reversed.** leetgo is no longer imported — importing it compiled 144 extra
packages, including a JavaScript interpreter and a second SQLite driver. The drivers are
vendored under `runner/drivers/` instead, with MIT attribution. The interfaces above are
unchanged, which is what made the reversal cheap. Read D-005 before reconsidering.

### The solution file has two regions (D-014)

`scaffold.go` writes it, `extract.go` reads it back:

```
scaffolding    imports, package clause, driver include — for the editor and the
               local compiler. Never leaves the machine.
@leetui code=start
   ...         exactly what the judge receives
@leetui code=end
```

**Invariant.** Scaffolding *references* the driver's `ListNode` / `TreeNode`, never
redefines them. Python's driver serializes with `isinstance` against its own classes, so a
duplicate declaration fails to serialize and presents as a wrong answer.

### `leetcode.Client` — one rate limiter, no exceptions

All outbound LeetCode traffic passes through a single limiter, including the company
sync. Session expiry surfaces as a typed `ErrSessionExpired` so the TUI can prompt
re-auth in place instead of failing the user's action outright.

### `store` — the fast path

Schema holds problems, tags, companies (with frequency + timeframe), submissions,
sync checkpoints, and an FTS5 virtual table over titles/slugs/statements/editorials.
Sync jobs are **resumable**: checkpoints are written as they go so an interrupted
company sync resumes rather than restarts.

### `theme` — the only place hex lives

`tui/theme` exports the nine tokens from `DESIGN.md` and the treatments built from them.
A hex literal anywhere else in the tree is a bug.

---

## Concurrency

- One Bubbletea program, single-threaded `Update`.
- Network and subprocess work happens inside `tea.Cmd` goroutines.
- Long jobs (company sync, full problem sync) run as a background worker that emits
  progress `tea.Msg`s. Cancellable via context; checkpointed so cancel is not data loss.
- SQLite is opened in WAL mode; writes are serialized through the sync worker.

---

## Error handling

Errors are UI states, not panics. Three tiers:

1. **Recoverable, actionable** → inline message + a key to fix it
   (`Session expired. Press a to re-authenticate.`)
2. **Recoverable, informational** → status line, auto-dismissed
   (`Company sync paused — rate limited. Resuming in 30s.`)
3. **Fatal** → tear down the alt-screen cleanly, print to stderr, non-zero exit.

Never leave the terminal in alt-screen or with mouse mode on. Restore on every exit path,
including panic (`defer` + `recover` at the top of `main`).

---

## Security invariants

- Cookies live in the OS keychain. Never in `config.toml`, never in logs, never in an
  error string.
- HTTP debug logging redacts `Cookie` and `x-csrftoken`.
- `vcs` verifies repo, branch, and remote before committing (D-011). Push is never
  automatic.
- Subprocess execution (`$EDITOR`, compilers, `git`) uses argument slices — never a
  shell string built from problem titles or user input.
