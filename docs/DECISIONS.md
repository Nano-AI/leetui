# Decision Record

Every decision here was settled deliberately. **Do not re-litigate without the user reopening it.**
Format: decision, rationale, what it costs, what would reverse it.

Settled: 2026-08-06.

---

## D-001 — Language & TUI framework: Go + Bubbletea/Lipgloss

**Decision.** Go, with `charmbracelet/bubbletea` (Elm-architecture event loop), `lipgloss` (styling), `bubbles` (list/textinput/viewport/spinner), `glamour` (markdown).

**Why.** Single static binary and `go install` distribution. The Elm msg loop maps cleanly onto the app's shape — everything here is an async network or subprocess result arriving later. Lipgloss styling is expressive enough for the visual direction in `DESIGN.md`. `leetgo` (see D-005) proves LeetCode's GraphQL is comfortable from Go.

**Cost.** Less visually capable than Python + Textual (no real CSS engine, no built-in animation timeline). Motion has to be hand-driven via `tea.Tick`.

**Reverses if.** Never, realistically. This is a one-way door; treat it as fixed.

---

## D-002 — Auth: paste cookie, with browser auto-import as convenience

**Decision.** Primary path is pasting `LEETCODE_SESSION` + `csrftoken`. Secondary is reading the Chrome/Firefox cookie store on demand. Credentials go in the **OS keychain**, never plaintext config.

**Why.** LeetCode has no public API and no OAuth for third parties. Cookie paste is the only path that works identically on every OS with zero platform-specific code, so it is the floor. Browser import removes the friction for the common case.

**Cost.** Sessions expire (~2 weeks). The app must detect 401/403 and re-prompt gracefully rather than failing mid-flow. Browser import needs macOS Keychain / Windows DPAPI / Linux keyring handling per browser, and breaks when Chrome rotates its encryption scheme — so it must always be optional and always degrade to paste.

**Reverses if.** LeetCode ships a real API token. (They will not.)

**Security invariant.** Cookies never touch `config.toml`, never get logged, never appear in an error message or a crash dump. Any HTTP debug logging must redact `Cookie` and `x-csrftoken` headers.

---

## D-003 — Tests run locally; submission goes to the judge

**Decision.** `run` executes against a **local** driver. `submit` goes to LeetCode's judge.

**Why.** The tight loop should not cost a 2–6s network round-trip or burn rate limit that the company sync (D-008) needs. Correctness of record still comes from the real judge.

**Cost.** We own a judge harness — custom types (`ListNode`, `TreeNode`, `Node` with random pointer), argument/return serialization, and comparator semantics that `metaData` does not encode: in-place mutation problems, order-insensitive answers, float tolerance, design problems (constructor + op/arg sequence).

**Mitigation.** D-005 buys most of this. Where local disagrees with expectation, the UI shows a diff and offers a one-key remote verify — local failure is never presented as authoritative.

---

## D-004 — Local execution ships for Go, Python, C++, Rust. Java and JS/TS are remote-only.

**Decision.** Local run: **Go, Python3, C++, Rust** — exactly the set `leetgo` already implements. Java and JS/TS get full browse/edit/submit/sync, but `run` for them hits the judge.

**Why.** `leetgo/lang` contains `go.go`, `python.go`, `cpp.go`, `rust.go` and nothing else; its own docs state local testing is these four because "local testing requires more work to implement for each language." Writing Java (compile + classpath + JDK probe) and JS/TS drivers from scratch is roughly the two hardest weeks in the project for two of six languages.

**Cost.** Java and JS/TS users get a slower loop. Rust is supported despite not being requested — it was free.

**Note.** *Codegen* templates still ship for all 20+ languages leetgo generates; only *local execution* is limited to four.

**Reverses if.** JS/TS is the cheap one to add later (JSON-native, no compile step, custom types are plain classes) — do that before Java. Upstream both to leetgo if written.

---

## D-005 — leetgo is a pinned dependency behind our own adapter interface

**Decision.** `go get github.com/j178/leetgo` at a **pinned tag**. Every call crosses our own `runner.Runner` / `runner.Generator` interfaces. No leetgo type appears outside `internal/runner/leetgo*.go`.

**Why.** leetgo is MIT-licensed and has no `internal/` directory, so `lang/`, `leetcode/`, and `testutils/` are legitimately importable. But it is an **application, not a library** — its exported API carries no stability guarantee and will break under us.

**Cost.** An adapter layer that is pure overhead on day one.

**Reverses if.** Upstream breaks us twice, or diverges from what we need. Then hard-fork `lang/` + `testutils/` into the repo (MIT permits, attribution required) — with the adapter in place that is a one-file swap, not a refactor. **The adapter exists specifically to keep this cheap. Do not let leetgo types leak.**

---

## D-006 — Full premium parity, minus mock assessments

**Decision.** Every premium surface the user's cookie unlocks is a first-class part of the TUI: premium-locked problems, company tags + frequency, **company-specific problem packs** (pick "Google", work through Google's list — the website's core premium loop), editorials, premium study plans and lists.

**Explicitly out of scope:** mock interviews / timed assessments.

**In scope, separately:** a **timer/stopwatch** in the header rail, mirroring LeetCode's own top-right timer. This is a plain timer, not assessment scoring.

**Why.** The user has Premium and stated the goal as parity. The app cannot *grant* premium — it only surfaces what the session already unlocks.

**Cost.** Large GraphQL surface to reverse and keep working.

**Degradation.** Without premium, gated panes show a lock and a one-line statement of what they would contain. Never a raw error, never a silently hidden feature.

---

## D-007 — Editorials render via HTML → markdown → Glamour; images are bracket links

**Decision.** LeetCode editorial HTML is converted to markdown and rendered with Glamour (Chroma-highlighted code, styled tables/quotes) using a custom theme matching `DESIGN.md`.

- **LaTeX** → Unicode approximation (`O(n log n)`, `≤`, `Σ`, `√`, subscripts).
- **Images** → an inline bracket marker, e.g. `[▸ img 1 — dp state table]`. Enter opens it in the system viewer/browser. **This is the default on every terminal.**
- **Inline graphics** (Kitty / iTerm2 protocol) are an **opt-in config flag**, never auto-detected into the default path.

**Why.** The bracket is the floor: it works in every terminal, degrades to nothing worse than a label, and is what the user asked for. Inline images are a bonus for terminals that support them, not a dependency.

**Cost.** Diagram-heavy editorials read worse than in a browser. Mitigated by `o` → open the full editorial in a browser.

---

## D-008 — Company data syncs by inverting company → problem

**Decision.** Walk the premium company list (~500 entries) via `companyTag`, and build the **reverse** problem→companies index locally. Throttled background job, resumable on interrupt, refreshed weekly.

**Why.** Fetching company tags per problem is ~3,600 requests and a plausible rate-limit or account flag. Querying by company is the same data in ~7× fewer requests, and frequency + timeframe buckets (6mo / 1yr / 2yr / all) come along for free. It also directly powers the company packs in D-006.

**Cost.** Two representations to keep consistent. First sync is long — it must be backgrounded, interruptible, resumable, and show real progress.

**Invariant.** All LeetCode HTTP goes through one rate limiter. Never bypass it for "just one request."

---

## D-009 — SQLite + FTS5 via `modernc.org/sqlite`

**Verified 2026-08-06:** FTS5 is compiled into `modernc.org/sqlite` and works. The
resulting binary is 25 MB and links only system libraries — the single static binary
from D-001 survives.


**Decision.** Local store is SQLite with an FTS5 index over problem titles, slugs, statements, and editorials. Full sync on first run, delta after.

**Driver: `modernc.org/sqlite` (pure Go).** **Not** `mattn/go-sqlite3` — that needs cgo, which forfeits the single static binary that motivated D-001.

**Why.** Instant fuzzy title/slug/id lookup plus faceted filters (tag, difficulty, status, acceptance, company, frequency) is a query engine. Writing that by hand over an in-memory JSON snapshot is real work that SQLite already does correctly.

**Cost.** `modernc.org/sqlite` is slower than the cgo driver and has a large generated codebase. At 3,600 rows this is irrelevant.

---

## D-010 — Disk layout is problem-first

**Decision.**

```
<workspace>/
  0001-two-sum/
    README.md        # problem statement (markdown, synced)
    solution.py
    solution.go
    notes.md         # user's own notes, never overwritten by sync
    testcases.txt    # leetgo-format cases, editable
  0146-lru-cache/
    ...
```

Zero-padded 4-digit ID + slug.

**Why.** GitHub renders `README.md` per directory, so browsing the repo shows each problem statement inline. Zero-padding sorts correctly. One problem's work — all languages, notes, cases — stays together. Matches the LeetHub convention, so the repo is legible to anyone who lands on it.

**Cost.** Per-language linting/CI is awkward compared to a language-first tree.

**Reverses if.** Effectively never — this is a history-rewriting migration once solutions are committed. **Treat as fixed.**

**Rule.** `notes.md` is user-owned. Sync may create it, must never overwrite it.

---

## D-011 — Git: commit on Accepted, push on demand

**Decision.** An `Accepted` verdict auto-commits. Pushing is always an explicit keypress. Implemented by shelling out to the `git` binary.

**Why.** Auto-commit captures the moment work is provably correct. Explicit push means the app never surprise-publishes to a remote. Shelling out inherits the user's credentials, commit signing, hooks, and `includeIf` config — reimplementing that with go-git is strictly worse.

**Commit message convention** (see also user memory):

```
solve(0146): lru cache — go, 58ms, beats 91.2%

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
```

Short, specific, dev syntax. **The user is always the primary author.** Never add a generated-with footer or a link to a Claude Code session.

**Guard.** Before committing: verify the workspace is a repo, on the expected branch, and that the remote is what the user configured. Refuse and surface a message rather than committing somewhere unexpected.

---

## D-012 — Editing is delegated to `$EDITOR`, plus a file watcher

**Decision.** `e` suspends the TUI and execs `$EDITOR` (falling back `$VISUAL` → `nvim` → `vim` → `nano`), resuming on exit. Independently, the solution file is **watched** — saving it from any editor in any other pane triggers a re-run.

**Why.** Writing a text editor (undo stack, syntax highlighting, autoindent, selection, search) is a project, not a feature of this one. Users keep their own config and LSP.

**Cost.** Terminal suspend/resume needs care with Bubbletea's alt-screen and mouse modes (`tea.ExecProcess`).

**Note.** `$EDITOR` and `$VISUAL` are both unset on this machine — the fallback chain matters, and first-run setup should offer to set one.

---

## D-013 — Keymap: vim-first, arrows always work, fully remappable

**Decision.** `hjkl` / `gg` / `G` / `n` / `N`, `/` search, `:` command palette, `?` help overlay. Arrows and PgUp/PgDn work everywhere as an unadvertised fallback. Mouse enabled for click and scroll. Every binding overridable in `config.toml`.

**Why.** Vim bindings for speed, arrows so nobody is walled out, palette for discoverability. The three cost little together because they resolve to the same action layer.

**Invariant.** Keys are declared in one keymap table, never hardcoded at a call site — that is what makes remapping and the `?` overlay both work without duplication.

---

## Open items

- [ ] Which browsers browser-cookie-import supports at v1 (Chrome only, or + Firefox/Arc/Brave)
- [ ] Whether `notes.md` is committed to the GitHub repo or gitignored by default
- [ ] Rate-limit budget: requests/sec ceiling for the company sync, empirically determined
- [ ] Whether premium study plans get a dedicated view or fold into company packs
