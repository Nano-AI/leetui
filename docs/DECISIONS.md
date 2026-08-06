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

## D-005 — REVISED 2026-08-06: vendor the drivers, do not import leetgo

> **This decision was reversed after measuring it.** The original text is kept below,
> because the interface it produced is still the right shape and still in use.
>
> **What changed.** Importing `github.com/j178/leetgo/lang` compiles **144 packages** and
> takes the module graph from 82 to 168. Among them: a full JavaScript interpreter
> (`dop251/goja`), a second SQLite driver, `fsnotify`, `viper`/`afero`/`cast`, and
> interactive survey prompts. None of that serves local code execution — it is a CLI
> application's dependency tree, arriving because `lang/` reaches into the rest of the
> app.
>
> **New decision.** Take **D-005's own option 4**: vendor the per-language driver
> runtimes into `internal/runner/`, with MIT attribution, and write our own execution
> layer. leetgo's Python runtime is 164 lines; the value was always the driver text and
> the comparator knowledge, never the Go code wrapped around it.
>
> **Kept.** The `runner.Runner` / `runner.Generator` interfaces stay exactly as they
> are. They were designed as a firewall against leetgo's unstable API; they now serve as
> the seam between the TUI and however a language happens to execute. Writing them first
> is what made this reversal a substitution instead of a rewrite.
>
> **Cost accepted.** We own the four drivers outright: upstream fixes must be hand-ported,
> and comparator edge cases (in-place mutation, unordered results, float tolerance,
> design problems) are ours to discover. That was already true — leetgo does not encode
> them either, which is why D-003 promises a one-key remote verify on any local failure.
>
> **Attribution.** Vendored driver sources carry a header naming leetgo and its MIT
> licence. `docs/` and the README credit it.

**Original decision (superseded).** `go get github.com/j178/leetgo` at a **pinned tag**. Every call crosses our own `runner.Runner` / `runner.Generator` interfaces. No leetgo type appears outside `internal/runner/leetgo*.go`.

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

### D-006a — Company packs sync ONE AT A TIME, not in bulk

Settled 2026-08-06, once the endpoints were probed live rather than guessed at.

The roadmap said "invert company → problem, ~500 requests". The real registry is **984 companies**, and each has **five** timeframes (`thirty-days`, `three-months`, `six-months`, `more-than-six-months`, `all` — verified, and the only five that resolve). A blanket sync is ~5,000 requests: forty minutes at the shared rate limit, to answer a question the user asked about one company.

**Decision.** The registry syncs wholesale — it is one request and it works **signed out**. Packs sync on demand, when a company and timeframe are chosen. Google's largest pack is 2,335 problems, which is 24 requests and a few seconds.

**Endpoints.** `companyTags` (registry, public) · `favoriteDetailV2` (pack size, public) · `favoriteQuestionList` with `favoriteSlug: "<company>-<timeframe>"` (the problems, **premium**).

**The gate is not an error.** A free account gets HTTP 200 and an empty `questions` array with `totalLength` still set. `CompanyPage` compares the two and raises `ErrPremiumRequired`; a caller that skipped that check would report "this company asks nothing". Likewise an unknown company returns `null`, not an error, which is why `PackSize` runs first and maps it to `ErrNotFound`.

### D-006b — Rejected: piggybacking company tags on `questionData`

`question.companyTagStats` exists and would ride along on the statement fetch for free, filling `ASKED BY` for every problem browsed without a single extra request.

**Not taken.** It buckets by a *different* vocabulary — numeric keys for 0-6 months, 6-12 months, 1-2 years — which does not line up with the five named windows the packs use. Two timeframe vocabularies in one `timeframe` column would make the data untrustworthy for the sake of saving requests we are not spending anyway.

---

## D-007 — Editorials render via HTML → markdown → Glamour; images are bracket links

**Decision.** LeetCode editorial HTML is converted to markdown and rendered with Glamour (Chroma-highlighted code, styled tables/quotes) using a custom theme matching `DESIGN.md`.

- **LaTeX** → Unicode approximation (`O(n log n)`, `≤`, `Σ`, `√`, subscripts).
- **Images** → an inline bracket marker, e.g. `[▸ img 1 — dp state table]`. Enter opens it in the system viewer/browser. **This is the default on every terminal.**
- **Inline graphics** (Kitty / iTerm2 protocol) are an **opt-in config flag**, never auto-detected into the default path.

**Why.** The bracket is the floor: it works in every terminal, degrades to nothing worse than a label, and is what the user asked for. Inline images are a bonus for terminals that support them, not a dependency.

**Cost.** Diagram-heavy editorials read worse than in a browser. Mitigated by `o` → open the full editorial in a browser.

### D-007a — Correction: editorials are markdown, and go through a different door

The heading above says "HTML → markdown → Glamour". That is right for **problem statements**. It is wrong for editorials, and the difference was only visible once a real one was fetched.

`question.solution.content` is **markdown already**, with HTML embedded in it: `<iframe>` playground embeds holding the reference implementations, Vimeo players, `<div>` figure wrappers, and `$$…$$` display math. Running it through `HTMLToMarkdown` parses the prose as one text node and loses every heading.

**So the pipeline inverts.** `render.Editorial` keeps the markdown and reduces the embedded HTML to the same bracket markers statements use — `[▸ 1 — code]`, `[▸ 2 — video]` — labelled by what the URL points at, because the reader is deciding whether to leave the terminal for it. Everything else in D-007 holds: same markers, same number keys, same LaTeX approximation.

One consequence for `latex.go`: `$$…$$` needed its own rule **before** the inline `$…$` one. Given `$$O(n^2)$$` the inline pattern matches the middle and leaves a stray dollar on each side.

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

**Rule.** `README.md` is derived and IS regenerated. It must be written from **markdown**, never from the pane's rendered form — Glamour's output is wrapped to a pane width and full of ANSI escape codes, which are garbage in a file people open in an editor. `prepare` converts the statement itself rather than reusing `detailMD`, for exactly this reason.

### D-010a — One earned exception to never-overwrite: `testcases.txt`

`testcases.txt` is user-editable and so falls under the never-overwrite rule. There is a single exception, and `workspace.ReplaceTestcases` exists to make it explicit rather than letting a caller quietly widen the rule.

Expected answers are scraped from the statement prose (see `runner/testcase.go` — LeetCode's API gives inputs but no answers). Feeding that scraper the *rendered* statement matched nothing, so leetui wrote a full set of cases with every answer blank, and `createIfMissing` then preserved that file forever: every local run reported *"none had an expected answer to check against"* and no amount of re-running fixed it.

**The exception.** A `testcases.txt` in which **every** case has an empty expected answer holds nothing a person could have typed, so it is replaced. One non-empty answer anywhere and the file is left alone. Anything else still goes through `WriteTestcases`.

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

### D-012a — The editor goes beside leetui, not on top of it

Settled 2026-08-06, after the first real solving session raised two complaints: *"how am I supposed to read the problem and work on it at the same time?"* and *"running test cases after exiting a problem is annoying."* Both are the same cause. `tea.ExecProcess` suspends leetui completely, so the statement disappears exactly when you start writing code against it, and the file watcher — which exists precisely to re-run on save — is dead for the whole editing session.

**Decision.** Three routes, best first, chosen by what the terminal can actually do:

| route | when | leetui |
|---|---|---|
| **pane** | inside tmux / zellij / WezTerm / Kitty | stays up beside the editor |
| **detached** | a GUI editor | stays up; the editor has its own window |
| **takeover** | a terminal editor with nowhere to go | suspended, as before |

Only the last one suspends anything, and it is the only one that has to.

**Detection is by environment variable**, not by looking for the binary. `$TMUX` is set only *inside* a session, so its presence is proof; `tmux` being on `PATH` says nothing about whether this terminal is in one.

**GUI editors lose `--wait`.** Those flags exist so `ExecProcess` does not resume the TUI over a file nobody has typed in. When leetui is staying on screen, blocking is exactly wrong — it would freeze the statement and the watcher for as long as the window is open.

**In takeover, `README.md` opens alongside** so the problem is still readable, but only for editors whose split flag is known (`nvim -O`, `hx --vsplit`). A second file opened *stacked* is a buffer in the way, not a statement to read, so editors without a known flag get one file.

**And the tests run when the editor exits**, if the file changed. That path is the only one where the watcher saw nothing.

**One save must run the tests once.** The watcher and the after-edit run share one timestamp map, so whichever notices a save first advances it and the other finds nothing to do. Two copies of that bookkeeping would run everything twice.

Config: `editor_pane`, `run_after_edit`, `open_statement` — all default true, all independently switchable.

---

## D-013 — Keymap: vim-first, arrows always work, fully remappable

**Decision.** `hjkl` / `gg` / `G` / `n` / `N`, `/` search, `:` command palette, `?` help overlay. Arrows and PgUp/PgDn work everywhere as an unadvertised fallback. Mouse enabled for click and scroll. Every binding overridable in `config.toml`.

**Why.** Vim bindings for speed, arrows so nobody is walled out, palette for discoverability. The three cost little together because they resolve to the same action layer.

**Invariant.** Keys are declared in one keymap table, never hardcoded at a call site — that is what makes remapping and the `?` overlay both work without duplication.

---

## D-014 — A solution file is two regions: scaffolding, and the marked code

Settled 2026-08-06, after the first real edit session.

A LeetCode starter snippet is **not a compilable file**. It has no imports, no package clause, and no definition of `ListNode` or `TreeNode`, because the judge supplies all of that around it. Written to disk verbatim — which is what leetui did — the result is a buffer the editor lights up red: clangd cannot resolve `vector<int>`, pyright cannot resolve `List[int]`. The local runner compiled it anyway, because the generated `main` includes the driver *before* textually including `solution.cpp`, which hid the problem from the tests and from nobody else.

**Decision.** The file leetui writes has two regions:

```cpp
// 1. Two Sum · Easy
// https://leetcode.com/problems/two-sum/
//
// Everything above the marker is local scaffolding, for your editor
// and the local runner. Only the marked region is submitted.

#include "leetui_driver.h"        // ← scaffolding

// @leetui code=start
class Solution { ... };           // ← what LeetCode gets
// @leetui code=end
```

Same idea as vscode-leetcode's `@lc code=start`, and **those markers are read too**, so a workspace built with that extension submits correctly here without being rewritten.

**Per language.** C++ includes the driver header (`#pragma once`, so the generated main including both is fine). Go gets `package main` — the driver shares the package, which is what puts the node types in scope. Python imports `typing`, plus the node types **from the driver**. Java gets `java.util.*`. Languages with no local driver get nothing but the header; inventing imports for them would be guessing.

**Invariant — scaffolding REFERENCES the driver's types, never redefines them.** Python's driver serializes with `isinstance(value, (ListNode, TreeNode))` against its own classes. A solution that declared its own `ListNode` would return an object the driver could not serialize, and the failure would present as a wrong answer rather than as a wiring mistake.

**Submit sends the marked region only.** This also fixes a latent bug: Go solutions were being submitted with their `package main`, and LeetCode wraps Go submissions in a package of its own. Unmarked files keep working — the fallback sends the whole file, minus a leading Go package clause, which is the one piece of scaffolding the judge rejects rather than tolerates.

**Upgrading old folders is additive.** A file without markers is *wrapped*: its existing content becomes the marked region byte for byte, and lines are added above and below. Nothing the user typed can be lost, which is what makes rewriting a file they may have edited defensible at all (see D-010a for the other earned exception). A Go package clause moves out of the marked region as part of the wrap. A file that already has markers is untouched, so this is a no-op after the first edit.

**Cost.** The solution file is no longer something you can paste straight into leetcode.com without deleting a few lines. Worth it: the file is edited far more often than it is pasted, and `o` opens the problem in a browser when pasting is what you want.

---

## D-015 — A CLI seam, not a second frontend

Settled 2026-08-06, weighing "should this be an nvim plugin / VS Code extension instead?"

**The numbers decided it.** The TUI is ~5,000 lines — 41% of the tree. The core (`leetcode`, `store`, `syncer`, `runner`, `render`, `workspace`, `solve`) is ~7,100 and already frontend-agnostic. A VS Code extension throws away both — different language, webview UI, and a mature free incumbent in vscode-leetcode. An nvim plugin keeps the core but discards the board, search, company browser, editorial pane, and the flip, rebuilt in Lua.

**Decision.** Neither. `leetui` with no arguments stays the product; a small set of subcommands exposes the same core to anything that can run a process:

```sh
leetui pull two-sum        # folder, statement, scaffolded solution, test cases
leetui run                 # from inside the folder
leetui run %               # from an editor: the buffer's path
leetui submit two-sum
leetui path two-sum        # for scripting
```

An nvim plugin then becomes a keymap rather than a project, and it does not fork anything:

```lua
vim.keymap.set("n", "<leader>lr", ":!leetui run %<CR>")
```

**A problem argument accepts four shapes** — slug, folder name, a path to either, or omitted to mean the working directory. Supporting only the slug would make the editor case a lookup exercise instead of one line, which is the entire point.

**Exit codes are part of the contract**, because that is what an editor branches on: `0` worked (and every case passed), `1` ran and was wrong, `2` could not run.

**Output is plain text.** No colour, no box drawing. It lands in an editor's message area as often as a terminal, and neither renders a Departure Board.

**What this forced, and was worth it on its own:** `prepare` moved out of `internal/tui` into `internal/solve`. Laying out a problem folder was never a TUI concern, and two copies would have drifted.

---

## D-016 — leetui is the LeetCode side of the desk, not the editor

Settled 2026-08-06, from the user's own framing: *"All UI components / LeetCode components are contained in the CLI itself, but the code is through nvim and vscode."*

**Decision.** leetui owns browsing, search, company packs, the statement, editorials, running, and submitting. It does **not** own editing. The user already has an editor configured the way they like it, usually already open in the next pane.

**What this demotes.** The three-route editor launcher (D-012a) solves *"leetui needs to hand you to an editor."* Under this model it rarely fires: you are already in one. `e` stays as the fast way to find the file the first time and for anyone who wants it, but it is a convenience, not the path.

**What this promotes.** The file watcher becomes THE mechanism rather than a nicety. Edit anywhere, save, leetui re-runs. That already worked — it was just invisible, which is the same as being off.

**What it required.** leetui must **say where the file is**. Without a path there is nothing to open in the other pane and the whole arrangement collapses back into pressing `e` and hoping. Hence the workbench strip: the solution's path, the language, and one line saying whether a save will re-run.

The strip asks `workspace.Path` for that path rather than assembling the folder name itself. A second copy of the convention would drift, and a strip that points at the wrong file is worse than no strip.

**It also has to be honest about when saves do nothing.** A judge-only language has no local driver, so "watching" would promise a re-run that never comes; it says `judge only · s to submit` instead. Same for a file that does not exist yet, and for `watch_solution = false` — which reads `watch off`, never "watching off", because a skimmed "watching…" says the opposite of what it means.

**Cost.** A four-line strip of vertical space in the side column, always present. It earns it: it is the only thing on screen that connects leetui to the editor that is doing the actual work.

---

## D-017 — Difficulty borrows LeetCode's own colours

Settled 2026-08-06, at the user's request. Read off leetcode.com's dark theme in a browser rather than guessed:

| | LeetCode variable | value |
|---|---|---|
| Easy | `--difficulty-easy` → `--dark-teal-60` | `#1CBABA` |
| Medium | `--difficulty-medium` → `--dark-yellow-60` | `#FFB700` |
| Hard | `--difficulty-hard` → `--dark-red-60` | `#F63737` |

**This overrules DESIGN.md's strongest rule** — that green and red belong to the judge alone, so that a verdict lands. It is worth it: everyone who has used LeetCode already reads teal/amber/red as easy/medium/hard with no legend, which no invented weight ramp achieves. The previous scheme (dim/bone/amber by weight) was legible but had to be learned.

**The dilution is real and is contained.** Difficulty appears only in a three-character column and in the problem heading — never beside the submission queue, where the flip lands. Nothing else in the app may borrow these three. Hard's red (`#F63737`) and the judge's Wrong Answer red (`#D65A5A`) are close enough that putting them in one pane would blunt both.

**Reverses if.** A verdict ever fails to read at a glance because of it.

---

## D-018 — Two screens: the list, then the problem

Settled 2026-08-06. The statement pane used to be on screen permanently, taking 40% of the width beneath the list.

**Decision.** Browsing and solving are separate screens. `enter` opens, `esc` returns.

- **Browse** is the list and nothing else, full width. leetcode.com/problemset is a table and nothing else, for the same reason: while you are looking *for* a problem, one you have not chosen to read is in the way, and it costs the list the width it wants for titles, tags, and companies.
- **Solve** is the statement plus the working column — the solution's path, run results, submissions. The statement earns its space here because reading it is now the job.

**Consequences worth stating:**

*Verbs that act on one problem open it first.* `e`, `r`, `s`, `d` all put their answer somewhere only the solve screen shows — the result panel, the queue, the statement pane. Run from the list without opening it and the keypress is silent, which reads as broken rather than as "wrong screen". This was a real bug the tests caught: `d` on the list set the editorial flag and displayed nothing.

*The cursor never moves as a side effect.* Coming back from a problem lands exactly where you left, or browsing becomes a game of finding your place again.

*Number keys are decided by the SCREEN, not by focus.* On the list they filter difficulty; on a problem they open markers. You can see which screen you are on; you could not reliably see which pane had focus.

*`tab` does nothing on the list.* One pane means focus has nowhere to go, and moving it somewhere invisible makes the next keypress behave unexpectedly.

*The hint strip differs per screen.* Offering `r run` while someone scrolls four thousand problems pushes the one key they need — `enter` — off the end.

---

## D-019 — `f` creates the file and hands you the path

Settled 2026-08-06. `e` lays the file out AND launches an editor. Under D-016 the editor is already open somewhere else, so what is actually wanted is the first half alone: the file exists, scaffolded and ready, and its path is on screen to open by hand.

**Not "touch".** The file arrives with its imports, package clause, driver include, and marked region (D-014), plus the folder's README and test cases. Present is not the same as openable.

**The language is always asked.** It is the one decision creating a file involves, and the picker opens on the language you used last — so the fast path is `f` `enter`, and the choice is never taken away. That memory is `last_lang` in config, separate from `default_lang`: one is a preference set once, the other a habit the app observes, and the habit wins at startup.

**The path goes in a toast, not the status line.** The status line comments on what you just did; a toast hands you something you now need — a path you are about to type into another window. It floats over the top right, where nothing but glanceable rail metadata lives, and any keypress dismisses it.

Compositing needed ANSI-aware cutting: a rendered line is characters interleaved with colour codes, so "column 80" means the eightieth *printable* cell with the active styling carried across the seam. `ansi.Truncate`/`TruncateLeft` from `charmbracelet/x/ansi` do it, and lipgloss already depends on that module. The toast sits flush right — an inset stranded the pane's own corner beside it, and a lone `╮` floating next to a box reads as a rendering fault.

**Wording turns on whether the file existed.** "Created" over a file you have been working in for an hour would be alarming, and it is the same keypress; that case says "ready" instead.

---

## D-020 — Two corrections from first use

Both found by someone reading the board for the first time, which is the only way either would have surfaced.

**The acceptance column is a percentage.** It was a three-cell sparkline, on the reasoning (recorded in board.go, and wrong) that a bar and a "55.1%" label are the same fact twice. They are not. `58%` says what it is; `█▆▂` needs a legend nobody has, and the first question asked of the board was what it meant. Clever loses to legible on a scanning surface. No decimal — the column is scanned, and 57% versus 57.9% never changes a decision.

**A verdict now updates the board.** `status` had exactly one writer, the list sync, so a solve stayed invisible until the next full re-sync: you submit, the judge says Accepted, and the row still reads TRIED. `store.SetStatus` closes that, and it only ever moves progress FORWARD — a later wrong answer must not downgrade a solved problem, because LeetCode keeps your best result and so should this.

---

## D-021 — Two more corrections from first use

**Rows are banded.** The board had vertical rules and nothing horizontal, so tracing from a title across to its state column meant counting cells. `#15171D` on alternate rows, `#2A2D36` under the cursor, which outranks the band so the selection stays unmistakable.

Painting an already-styled row needed care: lipgloss closes every cell with a full reset, so a background set on the finished string survives only to the first cell and then gives out. `components.Paint` re-asserts it after each reset, and derives the escape from lipgloss rather than writing it by hand — so under `NO_COLOR` or a plain `TERM` it comes back empty and painting becomes a no-op instead of a special case.

**LOCKED depends on the account, not the problem.** It read straight off `PaidOnly`, so a Premium subscriber was told their problems were locked — the column claimed to report the reader's own state while actually reporting a property of the problem. With a subscription nothing is locked, and leetcode.com shows no lock either. The detail pane had the identical bug and showed a dead end instead of a statement that was on its way. Signed out still counts as locked, because signed out you genuinely cannot read it.

---

## D-022 — A todo list, writable from outside the app

Settled 2026-08-06. "A list of problems I want to do", plus a CLI so an agent can fill it.

**Its own table, not a column on `problems`.** The two have different owners: `problems` is a cache of LeetCode's data that a re-sync rewrites wholesale, and a list someone curated by hand must never be collateral damage of a refresh. There is deliberately no foreign key either — an agent may queue a problem this machine has not synced yet, and the entry has to survive until it has.

**Every operation is idempotent.** Adding twice is not an error and does not move the item; removing something absent is not an error. A tool that must check before it acts is a tool that races. Re-adding updates the note but keeps the original position: the list is a queue, and correcting a typo should not send the oldest item to the back of it.

**Ordered oldest first**, for the same reason — the thing added three weeks ago is the one most in danger of being forgotten.

**`--json` is a first-class output**, always an array and never `null`, so a loop over it needs no special case. `docs/AGENTS.md` documents the shape as stable.

**In the app:** `m` marks the row under the cursor, `M` filters to the list. One key for both directions of the mark, because marking happens while scanning and remembering which of two keys applies would cost more thought than the action is worth. The mark is written to memory first and to SQLite in the background, so it appears on the same frame as the keypress.

### D-022a — `LEETUI_CONFIG_DIR`, and why it exists

Added the same day, after the test suite **overwrote a live install's `config.toml`**: a test set `Workspace` to a `t.TempDir()`, something called `Config.Save()`, and `Dir()` resolved to the developer's real config — persisting a temp path and blanking their configured editor along with it. It went unnoticed until a later `leetui run` reported a file inside `/var/folders/.../TestCreateDoesNotArmTheWatcher…/`.

`Dir()` now honours `LEETUI_CONFIG_DIR`, and the TUI test harness sets it for every test. A process that can be told where its state lives is one that can be tested without touching anybody's. It is also a fair way to keep more than one profile, which is what `docs/AGENTS.md` recommends for agents.

---

## Open items

- [ ] Which browsers browser-cookie-import supports at v1 (Chrome only, or + Firefox/Arc/Brave)
- [ ] Whether `notes.md` is committed to the GitHub repo or gitignored by default
- [ ] Rate-limit budget: requests/sec ceiling for the company sync, empirically determined
- [ ] Whether premium study plans get a dedicated view or fold into company packs
