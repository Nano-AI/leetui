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

**Decision.** Primary path is entering `LEETCODE_SESSION` + `csrftoken`, one field each.
Secondary is reading the Chrome/Firefox cookie store on demand.
Credentials go to the best store the machine has: OS keychain, then a user-configured credential helper, then a `0600` file.
Nothing is stored until LeetCode has confirmed the cookies work.

**Why.** LeetCode has no public API and no OAuth for third parties.
Cookie entry is the only path that works identically on every OS with zero platform-specific code, so it is the floor.
Browser import removes the friction for the common case.

**Cost.** Sessions expire (~2 weeks).
The app must detect 401/403 and re-prompt gracefully rather than failing mid-flow.
Browser import needs macOS Keychain / Windows DPAPI / Linux keyring handling per browser, and breaks when Chrome rotates its encryption scheme, so it must always be optional and always degrade to manual entry.

**Reverses if.** LeetCode ships a real API token. (They will not.)

**Security invariant.** Cookies never get logged, never appear in an error message or a crash dump, and never reach `config.toml`. Any HTTP debug logging must redact `Cookie` and `x-csrftoken` headers. Where a credential is written at rest is governed by D-002a, and a store weaker than the keychain must be disclosed to the user every time it is used.

### D-002a — Storage: a fallback chain that is never silent

**Amends D-002**, which said credentials go in the OS keychain and nowhere else.

**Problem.** On Linux the keychain is the D-Bus Secret Service, and plenty of ordinary machines do not have one: a headless box, a container, an SSH session, a tiling WM that ships no keyring agent.
On those, `keyring.Set` fails and sign-in cannot complete at all.
The user saw a raw godbus error about `org.freedesktop.secrets`, truncated to the width of the sign-in panel.
"Cannot sign in on this machine" is not a defensible reading of an invariant.

**Decision.** Store to the best backend available, in order:

1. **OS keychain.** Preferred, needs no configuration.
2. **Credential helper.** A command in `[auth] helper`, following Docker's `get`/`store`/`erase` protocol.
3. **Plaintext file.** `credentials.json` in the config dir, mode `0600`. Last resort.

A tier is skipped only when it is genuinely unavailable.
A keychain that exists and refuses the write is an error, not a reason to downgrade.
`LEETUI_SESSION`/`LEETUI_CSRF` override everything at load time, for CI and for `pass`-driven shells.

**Why not encrypt the file ourselves?** Because the key would have to live beside the ciphertext, and anyone who can read one can read the other.
It raises no attacker's cost while letting the tool claim a security property it does not have.
Docker reached the same conclusion: it moved from base64-in-`config.json` not to self-encryption but to **external credential helpers**, which is why tier 2 exists.
A helper is the only way to get real encryption without a keychain, because the key ends up in a GPG agent, a passphrase, or a TPM.

**Why a fallback at all, given the GitHub CLI's reputation for this?** Because the objection to `gh` is not that it falls back.
It is that it falls back *silently* ([cli/cli#10108](https://github.com/cli/cli/issues/10108), [#8954](https://github.com/cli/cli/issues/8954), [#7757](https://github.com/cli/cli/issues/7757), [#13317](https://github.com/cli/cli/issues/13317) are all "I had no idea, and nothing told me").
So the disclosure is the load-bearing part of this decision, not the chain.
`auth.Store` and `auth.Load` return a `Backend` that every caller must handle, and it is surfaced in three places: the sign-in panel says where the cookies will go *before* they are typed, the confirmation says where they went, and `leetui doctor` has an `auth` section that names the file and both ways off it.

**Cost.** A `0600` file is readable by anything running as that user.
That is a genuine reduction against a keychain and the reason it is last, disclosed every time, and never chosen while a better tier works.

### D-002b — Verify before storing

**Decision.** Sign-in asks LeetCode `Status` with the candidate cookies and stores only if `isSignedIn`.

**Why.** Storing whatever parsed and reporting success meant a typo'd or expired cookie produced "Signed in", then failed later mid-action as "Session expired", by which point the user had no reason to connect the failure to what they typed.
It also silently replaced a working session with a broken one.
Verification runs on a throwaway client, so an unverified session never becomes the one the app is using and a failure has nothing to roll back.

**Bonus.** `UserStatus.Username` comes back from the same call, which is the only thing that ever populated `Credentials.Username`.

**Cost.** One round trip on the sign-in path, and sign-in now requires network. Both are acceptable: the credentials are useless without network anyway.


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

**The dot has a header.** It shipped under a blank one, and the first question asked was what it meant — the same failure as the acceptance sparkline one commit earlier. A glyph is only allowed to be a glyph when something else names it; the column header is `TODO` and `TestEveryColumnIsLabelled` now fails the build if any column ships without one.

**In the app:** `m` marks the row under the cursor, `M` filters to the list. One key for both directions of the mark, because marking happens while scanning and remembering which of two keys applies would cost more thought than the action is worth. The mark is written to memory first and to SQLite in the background, so it appears on the same frame as the keypress.

### D-022a — `LEETUI_CONFIG_DIR`, and why it exists

Added the same day, after the test suite **overwrote a live install's `config.toml`**: a test set `Workspace` to a `t.TempDir()`, something called `Config.Save()`, and `Dir()` resolved to the developer's real config — persisting a temp path and blanking their configured editor along with it. It went unnoticed until a later `leetui run` reported a file inside `/var/folders/.../TestCreateDoesNotArmTheWatcher…/`.

`Dir()` now honours `LEETUI_CONFIG_DIR`, and the TUI test harness sets it for every test. A process that can be told where its state lives is one that can be tested without touching anybody's. It is also a fair way to keep more than one profile, which is what `docs/AGENTS.md` recommends for agents.

---

## D-023 — The state column is a glyph, under a header

Settled 2026-08-06, chosen from options side by side.

```
✓  solved      U+2713
◐  tried       U+25D0 — half filled, for partly done
⊘  locked      U+2298 — only ever drawn without premium
```

**Why now, having just argued the opposite twice.** D-020 and D-022 both retired unlabelled glyphs, and the rule that came out of them was not "words beat glyphs" — it was **a glyph is only allowed to be a glyph when something else names it**. The column is headed `STATE`. With the header, `✓ ✓ ◐ ⊘ ✓` scans down in a way `SOLVED SOLVED TRIED LOCKED SOLVED` never did, and it costs two fewer cells. Remove that header and this decision is void.

**The width risk, and the fallback.** `✓` and `⊘` are East-Asian **Ambiguous**: a terminal may legitimately draw them two cells wide, and in a grid ruled to the column that shears every row below. So `theme.Glyphs()` has an ASCII set (`x` `~` `-` `*`), selected once at startup by `config.PreferASCII` from:

- `ui.ascii` in config, which always wins
- a locale that is not UTF-8 — the strongest signal, since the terminal is not being told to decode multi-byte characters at all
- `TERM` of `linux` (the kernel console, 256 glyphs), `dumb`, `vt100`, `vt220`, or unset

An unset locale is **not** treated as failure: macOS terminals routinely leave `LANG` unset while handling UTF-8 perfectly, and the config flag covers anyone this gets wrong.

`TestGridSurvivesAmbiguousWidth` renders the board in both modes and asserts every row measures exactly the terminal width, which is the failure this is all guarding against.

**Known gap.** The frame itself is still box-drawing (`╭─┼─╯`) in ASCII mode. A terminal that genuinely cannot render `✓` cannot render those either, so ASCII mode is currently half a promise. Completing it is a Phase 6 item and is recorded there rather than left implied.

---

## D-024 — Solution commits carry no trailers

Settled 2026-08-06, building Phase 4.

D-011 fixed the commit convention and showed a `Co-Authored-By` line with it. Implementing it made clear that line belongs to **this repository's** commits, not to the ones leetui writes into a user's workspace. Those record a person solving a LeetCode problem. leetui typed none of it and neither did any model, so there is nothing to co-author.

```
solve(0146): lru cache — go, 58ms, beats 91.2%
```

Subject, and a body only if the user wrote a note. No `Co-Authored-By`, no generated-with footer, no session link. A false attribution on someone else's repository is worse than a missing one, and `git log` on a solutions repo is something people show other people. `TestNoTrailers` fails the build if any of those strings reappears.

**Figures are omitted when absent, never guessed.** A real 0.0% percentile and a missing one are the same value in the judge's payload, so zero prints nothing — claiming to beat nobody is worse than staying quiet.

---

## D-025 — Push lives in its own mode, behind a named destination

Settled 2026-08-06.

D-011 said pushing is always an explicit keypress. That is necessary but not sufficient: `p` on the board is one fat-finger away from publishing someone's solutions, and "origin" on a confirmation prompt tells nobody *which* account is about to receive them.

So push is reachable **only** from the repository view (`v`), and inside it takes a second key on a confirmation that names **the remote's URL**. Two deliberate steps, and the second is pressed while looking at the destination.

Three supporting rules, each from a way this can go wrong:

- **leetui never picks a remote.** Configured remote, else the upstream's, else the only one — a workspace with both a personal and a work remote gets no push target and says so.
- **`git` runs with `GIT_TERMINAL_PROMPT=0`.** leetui draws on the alternate screen; a credential prompt there is invisible and hangs the app with no way to answer. Failing with "could not authenticate — push from your own shell" is recoverable, and a hung TUI is not.
- **A refused `p` says why.** Detached HEAD, nothing committed, no remote, and already-in-sync are four different situations with four different fixes. A key that silently does nothing reads as a broken key (the same lesson as D-020).

**Committing keeps the opposite temperament.** An accepted verdict commits without asking, and stays silent when the workspace is not a repository or the files already match `HEAD` — neither is something the user did wrong, and a warning after an Accepted verdict would land as if it were.

---

## D-026 — One charset, chosen once

Settled 2026-08-07, finishing Phase 6.

D-023 gave the state column an ASCII fallback and left the frame in box-drawing. That made ASCII mode **half a promise**: a terminal that cannot render `✓` cannot render `╭─┼─╯` either, and a bezel of question marks is worse than no bezel at all.

`theme.Charset` now covers everything non-ASCII the interface draws — bezel corners and edges, column rules and their joints, the dashed divider, the inline separators (`┊`, `·`), and the cursor bar. One set, selected by the same `theme.ASCII` the glyphs use, so a terminal is never asked to draw one half and not the other.

**Every joint is `+` in the ASCII set.** Distinguishing `┬` from `┼` from `┴` would mean inventing a convention the reader has to learn, and the grid's structure is already carried by alignment — the joints only have to not look broken.

`TestASCIIModeDrawsNoBoxCharacters` renders the board, the solve screen, help and the repository view and fails on any of the eighteen characters this is meant to replace. That test found the last one: the cursor bar was still a literal `▌` inside `theme.ID`.

**Colour degrades separately.** `theme.DetectProfile` reports truecolor / 256 / 16 / none. 256-colour needs nothing — it quantises and amber stays amber. Nothing in this app was ever encoded in colour alone, so a verdict keeps its letterspacing, `PASS` keeps saying `PASS`, and `ESY`/`MED`/`HRD` are words before they are colours.

**The one real gap was focus.** DESIGN.md says the focused pane is marked by its bezel turning amber and never by a border-style change. That assumes colour exists; on a monochrome terminal it leaves no indicator at all, and "which pane does `tab` move" becomes unanswerable. So monochrome is the stated exception: the focused pane's title takes a marker. Nowhere else needed one.

---

## D-027 — One settable registry, two views

Settled 2026-08-07, finishing Phase 5.

The roadmap asked for a settings view, a `:` command palette, and a keymap remapping UI as three items. They are one: a table of what can be changed, and two ways to look at it.

`internal/config/settings.go` holds every option — its name, what it means, how to read it, how to write it. `:set` types into it, the settings screen lists it, and keybindings resolve through it on demand rather than being duplicated into it (D-013 already owns the action list; a second copy would drift). Adding an option means adding a row; nothing else needs to know.

**Both write through to `config.toml`.** A preference you have to set again every morning is not a preference.

**A duplicate binding is refused, not resolved.** `:set keys.submit x` when `x` is already `run` fails. Silently accepting it would shadow one of the two actions, and which one wins depends on Go's map iteration order — so the same config would behave differently between runs.

### D-027a — Tags and hints are spoilers

Added the same day, from use.

`hash-table` printed beside a problem **answers the question the problem is asking**, and reading it is not a decision — by the time you have registered the word while deciding whether to attempt the thing, it is too late. Same for hints, which exist to tell you the approach.

Both are hidden by default. `z` and `Z` reveal them, and the line still states how many exist, because a blank space reads as "there are none" rather than "there are three you have not looked at".

**Searching by tag is untouched.** That is the use that spoils nothing: you go looking for graph problems, you are not told this one is a graph problem. **Company names are never hidden** — which company asked says nothing about how to solve it, and it is the entire point of a company pack.

---

## D-028 — A debug log, off by default, redacted always

Settled 2026-08-07.

leetui talks to an API nobody documents. When it misbehaved the user got one error string and nothing to inspect afterwards — which is how a decode bug that failed **every submission** was found by a screenshot rather than by a trace, and explained in one line once it could be reproduced.

`leetui --debug`, or `LEETUI_DEBUG=1` for a subcommand, writes to `~/.local/share/leetui/debug.log`. The environment variable exists because an editor owns the command line for `:!leetui run %` and there is nowhere to put a flag.

**What it records:** the operation, path, and which session; the status and byte count of the response. Not the body — a successful sync is four thousand problems, and a log nobody can read is a log nobody reads.

**Except on a decode failure, where the body IS the diagnosis.** "cannot unmarshal string into Go struct field" names the field and says nothing about what arrived. Both the GraphQL and the REST paths log up to 2 KB of it.

**Appends, never truncates.** The interesting run is often the one before the one you are looking at.

### The invariant

D-002 says a session cookie never reaches a log, and the whole point of a debug log is that it gets pasted into bug reports. Redaction happens at the source: `Client.Debugf` receives traces with the session already reduced to `auth.Redact` form — first four characters, last four, and the length. Eight characters of an 837-character JWT, of which the first four are the header prefix every JWT shares.

That is enough to tell two sessions apart in a trace and useless to anyone who obtains it. `TestDebugTraceNeverCarriesTheSession` drives a real request through the traced path and fails if the token appears — and fails if nothing was traced at all, so it cannot pass vacuously.

**Nothing in the logging path may format a header itself.** That is how a session ends up in a file someone attaches to an issue.

---

## D-029 — The shared half of a driver lives in globals/

Settled 2026-08-07, at the user's suggestion, which was right.

Each language's driver splits in two: a part generated from **this** problem's metaData — argument decoding, the call, the return shape — and a part that is **byte-identical for every problem**. The second half was written into each problem folder, which put ~19KB of scaffolding beside every solution and buried the four files you edit under four you never open.

```
globals/leetui_driver.h          written once
globals/_leetui_driver.py
0001-two-sum/leetui_main.cpp     #include "../globals/leetui_driver.h"
```

**The objection was clangd, and a relative path answers it.** I argued against sharing on the grounds that a shared directory needs `-I` and a `compile_commands.json`, so every `solution.cpp` would show red squiggles in an editor — and the whole product thesis is that you edit in your own editor. The user pointed out that `"../globals/…"` is *relative*: a quoted include resolves against the including file's own directory. No flag, no compile database, and clangd follows it with zero configuration. Verified by compiling one before changing anything.

Python gets the same treatment through `sys.path`, and its per-folder copy is deleted — left in place it would **shadow** the shared one, hiding a stale driver behind a current one.

**Existing folders are migrated.** Every problem created before this says `#include "leetui_driver.h"`, and doing nothing would greet each of them with `fatal error: file not found`. That line is one leetui wrote, above the `@leetui code=start` marker, in the region the docs and `scaffold.go` both call scaffolding rather than code — so rewriting exactly it is in scope, and nothing below the marker is read. The stale copy is removed at the same time: left there the old include would still resolve, and the migration would look optional until the day it was not.

**What this costs.** A problem folder is no longer self-contained — you cannot zip one and compile it elsewhere without `globals/`. Nobody does that, and it was never a promise.

---

## D-030 — Every escape gets its own tmux wrapper

Settled 2026-08-07, after images failed silently in kitty inside tmux.

tmux drops sequences it does not recognise unless `allow-passthrough` is on, and even then each payload must be wrapped in its own DCS with every `ESC` doubled. The first implementation wrapped **the whole image in one DCS**.

That works for a thumbnail and fails for anything real. A 100 KB PNG is ~33 kitty chunks; tmux caps what a single DCS may carry, and the overflow is dropped — no error, no picture, nothing in any log. Precisely the silent failure the graphics detection exists to prevent, reintroduced one layer down.

Each escape is now wrapped individually. `TestPassthroughWrapsEachChunk` builds a three-chunk image and asserts one wrapper per chunk, because the single-wrapper version passed every test that only used a small one.

**Verified against a real figure inside tmux:** 27 KB out, seven chunks, seven wrappers, every inner `ESC` doubled.

**`allow-passthrough` must be on, and tmux must be RESTARTED** — reloading the config is not enough for an already-running server. `leetui doctor` reports the state and prints the fix, since nothing else ever will.

---

## D-031 — Study plans get their own view, one key, one request

Settled 2026-08-09. This closes the open item that asked whether plans get a dedicated view or fold into company packs.

**They get their own view, on `P`.** The two look like the same feature and are not, and the wire format is what settles it.

| | registry | contents | shape | ordering |
|---|---|---|---|---|
| Company pack | free, one request | **Premium** | 984 × 5 timeframes, paged at 100 | frequency |
| Study plan | needs discovery | **free, signed out** | ~22 plans, no timeframe | curriculum |

**A plan arrives whole.** `studyPlanV2Detail(planSlug:)` returns every question and every chapter in a single response — Top Interview 150 is 150 questions across 23 subgroups, one round trip, no `limit`/`skip`. Google's pack is 24 requests. That is why `SetPlan` has no paging loop and no half-written state to defend against: the plan lands or it does not.

**The free/premium axis is inverted, and the useful half is free.** A company pack lets a free account see that Google's list exists and nothing inside it. A study plan lets a free account work all 150 problems of Top Interview 150. Verified signed out.

**Folding plans into `c` was rejected.** The timeframe step is meaningless for a plan, so one picker would have to branch on which kind of list was selected, and the board would have to explain an "asked by" column for something nobody asked. Two concepts, two keys.

### The registry is a union, because neither source is the catalogue

`studyPlansV2ByTag` works signed out but its tag vocabulary is a short curated set — `interview`, `beginner`, `intermediate`, `database`, `dynamic-programming`. Topic tags do not work: `array` and `graph` return nothing. The sweep misses Top 100 Liked, Binary Search, Graph Theory and every premium plan.

So the registry is `SeedPlans()` ∪ tag sweep, each entry confirmed by a real detail fetch. Seventeen seeds is seventeen requests, which is affordable only because a plan is one request each. `UpsertPlans` **merges** rather than replaces for exactly this reason: a sweep that returns less than usual must not delete plans the seed list knows.

`studyPlansV2ByUpc` is the website's "Ongoing" row and is the one operation here that needs a session. It is not used — progress is computed locally against `problems.status`, which works for a free account.

**A seed slug that dies costs one plan, not the registry.** `TestLiveStudyPlanSlugs` is what turns that into a fixable fact. It has already earned its keep: the website's URL says `sql-50` and the plan slug is **`top-sql-50`**.

### The plan endpoint disagrees with every other endpoint

`problemsetQuestionList` answers `"Easy"` and `"ac"`. `studyPlanV2Detail` answers `"EASY"` and `"SOLVED"`.

Stored raw, every plan row would render Easy — `difficultyOf` falls through to Easy for anything unrecognised — and none would ever count as solved, so the picker's progress would read 0/150 forever. Both are normalised once, at the client boundary, so nothing downstream branches on where a question came from. An unrecognised value is **left alone**, not guessed at: a new enum member should surface as an odd-looking row rather than a silent Easy.

### Two things share a slot

`plan_rank` is the flattened position, and `Sort: "plan"` reads it. Chapters are a column, not a second picker: the running order is the plan, and hiding 22 of 23 chapters behind another keypress hides the shape of the thing.

That column is **the same slot as ASKED BY**, relabelled CHAPTER. A plan and a pack are mutually exclusive, so it is never asked to be both, and the board already drops this column first when narrow — a seventh column would just be dropped more often. A row with no chapter renders empty rather than falling back to companies: answering a different question than the header is the mistake D-023 already made twice.

---

## D-032 — Importance is its own table, ternary, and not the todo list

Settled 2026-08-09, from a real need: work through Top Interview 150 once, then come back
and redo only the problems that taught something.

**The todo list is the wrong shape for this, and it is worth being precise about why.**

| | question | when solved |
|---|---|---|
| `todo` | "get to this" — a queue | you take it off |
| `marks` | "this was worth doing" — a judgment | **it stays** |

The state that matters is **solved AND worth doing again**. A queue that empties as you
finish it cannot hold that, and a `priority` column bolted onto `todo` would force
"important" to imply "still outstanding" — which is precisely backwards for a second pass.

So: its own table, and no foreign key, for the same reason `todo` has none (D-022). An
agent may mark a problem this machine has not synced, and the verdict must survive until
it has. It also has to survive a re-sync, which rewrites the problems cache wholesale;
`TestMarksSurviveAResync` is what holds that.

**Ternary, not a score.** Up, down, or no opinion. A stacking counter was considered and
rejected: every write becomes read-modify-write, which is exactly the race D-022 avoided
by making todo idempotent. An agent marking in bulk would have to read each current value
to reach a target, and two agents would clobber each other. Three states need no read.

**Two keys, not one.** `m` toggles a todo with a single key because the state is binary. A
verdict has three states, and one key cycling up → down → clear would make "mark this
important" cost a variable number of presses that depends on state you must read off the
screen first. `+` and `-` each set their own direction; pressing the one a row already
carries withdraws it.

**`i` cycles the filter** through all → important → unimportant, and sorts by verdict while
it is on. One key there, because filtering is deliberate and done while looking at the
result.

**Unmarked sorts in the MIDDLE**, not last. "No opinion" genuinely sits between "worth
redoing" and "written off", and on a board of 4,013 problems where a dozen are marked,
burying everything unmarked underneath the handful marked down would make the sort useless.

**An empty note never erases an existing one.** A bulk agent pass and a human who wrote a
reason must not be in a race the bulk pass wins.

**The glyphs are the keys.** `+` and `-` are ASCII in both glyph sets, unlike every other
board mark. They cannot be drawn two cells wide the way an Ambiguous-width arrow could
(the hazard `theme/glyphs.go` exists to manage), and a column headed `MARK` showing `+`
needs no legend — the glyph *is* the keystroke that produced it.

**The column drops third**, after companies and before state. Solved-or-not is the more
fundamental fact about a row, and a verdict you recorded yourself is one `i` away.

### D-032a — `-` demotes, it never deletes

Corrected the same day, from use. The first cut only drew a `-` glyph and left the row
exactly where it was, which made the mark decorative: you still had to read past
everything you had already written off.

**Unimportant now sinks to the bottom of every sort and renders grey throughout** — title,
difficulty tag, and state glyph, not just the mark column. Sinking moves it out of the
way; draining the colour is what stops it pulling the eye on the way past. Neither one
alone does the job.

**It is never removed.** The row stays in the result set, stays searchable, stays in its
study plan and its company pack. That is the whole distinction between a judgment and a
delete, and only a judgment can be revised later — press `-` again and it comes straight
back up.

**Demotion is a prefix on every sort**, not a sort of its own:

```sql
CASE WHEN EXISTS (SELECT 1 FROM marks mk
  WHERE mk.problem_slug = p.slug AND mk.mark = 'down') THEN 1 ELSE 0 END, <the real sort>
```

So the rule holds under problem number, title, acceptance, difficulty, a pack's frequency,
a plan's curriculum order, and search relevance, without any of them knowing about it. It
carries no bind argument, which is what makes prefixing safe — the remaining placeholders
bind in the same order they did before. With nothing marked down the prefix is constant
for every row and changes no ordering at all; `TestDemotionChangesNothingWithoutMarks`
holds that, because a demotion rule that quietly reshuffles an unmarked board would be a
regression in every existing view.

~~**Important is NOT promoted to the top.** Only the downs move.~~ **Reversed the same
day — see D-032b.** The argument was that floating the ups would fight a study plan's
curriculum order. It does, and that turns out to be the point: on a redo pass the plan's
order is not what you are there for.

**The cursor beats dimming.** A row you deliberately moved onto is rendered normally
whatever you decided about it earlier — the alternative is a selected row you cannot read.

**Search terms are not highlighted on a dimmed row.** Amber on grey would be the single
brightest thing in a row whose entire job is to recede.

`Difficulty.Tag()` was split out of `Difficulty.Render()` for this. Re-styling an
already-rendered string does not work — lipgloss writes escape codes into the output, and
wrapping those in more escape codes leaves the inner colour intact — so the dim path needs
the raw text to colour for itself.

### D-032b — and `+` promotes, symmetrically

Reverses D-032a's "only the downs move", after a session of real use.

**Up floats, unmarked sits in the middle, down sinks — in every sort.** One `markRank`
prefix on the ORDER BY does all three, replacing the demote-only version.

The objection to floating was that it fights a study plan's curriculum order. It does.
That is what a redo pass wants: the second time through Top Interview 150 you are not
working the curriculum, you are working the shortlist, and burying it under the plan's
running order means scrolling for it. The curriculum is still there the moment the marks
come off.

**Unmarked stays in the middle**, for the reason D-032a gave: on a board of four thousand
with a dozen marked, sorting everything unmarked below the handful marked down would make
the rule useless.

**Gilded, not bold.** An important row draws its number and title in amber. It does not
get bold, because bold belongs to the cursor, and a board with a dozen bold rows on it has
no cursor. The difficulty tag keeps its own colour either way — that is semantic, and
overwriting it would trade information for emphasis.

Dimming and gilding are mirror images and the cursor outranks both. `theme.Pad4` was
exported alongside `Difficulty.Tag` for the same reason: the gold path needs the raw
number to colour itself.

---

## D-033 — The one place leetui is pleased with you

Settled 2026-08-09. An Accepted verdict sweeps through a colour band and settles into its
normal green, and a result that beat the field says so in two words.

**This is a deliberate exception to a rule the codebase states out loud.**
`components/flap.go` says the flip is "the app's entire motion budget… if a new animation
seems necessary somewhere, the answer is that the flip should cover it." The flip cannot
cover this: it is the mechanism by which a verdict *arrives*, and it has to look the same
whether the news is good or bad. Celebrating is a different message and needs a different
channel.

It earns the exception by being **rare, brief, and self-cancelling** — only on Accepted,
about a second, and it settles into exactly the frame that would have been there anyway.
The final palette entry before it stops is AC green, so the animation resolves into the
calm state rather than snapping back to it.

**Three levels, not a boolean.** `ui.celebrate` is `off`, `subtle`, or `full`, default
`full`. "No animation" and "no acknowledgement at all" are different requests: someone on
a slow SSH link wants the badge without the frames. At `subtle` no frames are ever
scheduled, which makes it genuinely cheaper rather than merely quieter.

**`ui.reduce_motion` outranks `full`** and demotes it to `subtle`. There is one answer to
"will this screen move", and it is the accessibility setting. It demotes to `subtle`
rather than `off` because asking for no animation is not asking to stop being told you did
well.

**Tiers read the judge's own percentiles**, so a badge means what the website means:

| tier | condition | badge |
|---|---|---|
| pass | Accepted | — |
| fast | beats >50% on one axis | `✦ FAST` |
| double | beats >50% on **both** | `✦ DOUBLE 50` |
| elite | beats ≥90% on both | `✦ TOP 10` |

A better result sweeps for longer, which is the cheapest way to make the rare thing feel
rare. **A zero percentile means "not reported", not "beats nobody"** — LeetCode omits them
for some problems and languages — so a missing figure can never demote a result below
`pass`, and the stats line never prints `beats 0%` for one.

**The badge is information, the motion is decoration.** The badge is the percentiles said
in two words, so it survives at `subtle`; only `full` adds frames. Nothing is ever
conveyed by the animation alone.

**The sweep colours visible characters, not rune positions.** Verdicts are letterspaced by
`theme.Display`, so indexing by position stepped the palette by two and dropped half of
it — the gradient came out coarse and stripey until the counter skipped spaces.

**The sweep never overlaps the flip.** It only replaces a settled verdict, so the two
animations cannot run over each other and the flip keeps its job of being the thing that
resolves. The sweep is keyed to a flap ID, so a second submission landing mid-animation
cannot leave colour running on the wrong row.

---

## D-034 — The site shows the program, not a drawing of it

Settled 2026-08-18. Every screenshot on `site/` is produced by `site/tools/capture.sh`,
which starts the real binary in tmux, presses the keys a reader would press, and pipes
`tmux capture-pane -e` through `site/tools/ansi2svg.py`. Nothing on the page is drawn by
hand. Re-run the script after any change to the interface.

**Why not a hand-built HTML mock.** The site used to reproduce the board in a table with
five invented rows. It drifted from the app immediately, and it undersold it: the real
board is 158 columns of dense, aligned, colour-coded data, and that density is the
product. A mock cannot show density it does not have.

**SVG, not PNG.** The board is type. A raster of a 158-column grid is either enormous or
mushy, and reading it is the entire point. The converter emits one `<text>` per styled run
with a `textLength`, so the grid stays aligned whatever monospace font the reader has, and
`board.svg` is 80 KB and sharp at any zoom. One PNG is rasterised as well, for the README
and the link preview: GitHub proxies SVGs through a sanitiser and no unfurler renders them.

**The window frame belongs to the page, not the capture.** The SVG is the screen and
nothing else. `index.html` draws the titlebar around it, so one frame style covers every
shot, and the frame is what tells a reader where the page stops and the program starts.
That boundary is the whole job: without it the captures read as more page.

**The hero is a recorded session, not a still.** `capture.sh` also drives a scripted
run through the app and snapshots the screen every 90 ms; `record.py` turns that into a
style palette plus one frame per snapshot carrying **only the lines that changed**. A
terminal mostly holds still, so a keystroke repaints two rows out of thirty-four, and
thirteen seconds of a 120-column screen comes to about 10 KB over the wire. The player
in `index.html` keeps an array of lines, patches the ones a frame names, and waits that
frame's own delay.

Not a GIF or a video: both would be an order of magnitude larger, neither is sharp at
two different widths, and neither leaves the text selectable. Not an embedded player
either. The recording is 120 columns rather than 158 because the whole board still fits
and the glyphs stay readable at the width the page gives the hero.

The still is what the markup ships; the player replaces it once the JSON arrives, so the
page opens on a real screenshot whatever happens to the script. Playback stops when the
window scrolls off screen or the tab goes to the background, and never starts under
`prefers-reduced-motion`.

**The editor split is photographed by nesting tmux.** `e` asks tmux for the pane, so
there is no single pane holding both halves and `capture-pane` cannot see the split. The
capture runs one tmux session inside another: the inner one is what leetui splits, and the
outer one sees that window already composited, divider and all, which is what a reader
would see. `editor.svg` is leetui and a real Neovim on the same screen, holding the file
that is really on disk.

**An MP4 exists for places that will not run a page.** `render_mp4.py` replays the frame
diffs back into whole frames, hands each to Chrome, and lets ffmpeg carry the timing from
a concat list rather than a fixed frame rate, because the recording holds still for a
second on a screen worth reading and then moves in 90 ms steps. Chrome is the rasteriser
so the video and the page cannot disagree about what the program looked like. It is not
part of `capture.sh`: it takes a minute per run, needs ffmpeg, and nothing on the site
uses it. Run it when a README or a post needs a video.

**The demo is why there is no company-and-plans section.** It shows both pickers
narrowing under real keystrokes, which is a better argument than two stills, and the
counters above it already carry the figures.

**Captures are never scaled up past 1:1.** Each figure carries the SVG's own width in
`--nat` and the frame stops there. A short `leetui run` and the full board then put the
same glyph on screen at the same size. Scaling each to fill its column made one page out
of five differently-sized terminals.

**The flip is the page's only animation, and it is the physical one.** In the terminal a
flip is the five half-block glyphs of `components/flap.go`, because that is every frame a
character cell has. A browser has the other 355 degrees, so the leaf actually falls. It
runs in two places, the counters and the verdict, and nothing else moves. Every value is
correct in the markup before a frame runs.

**The captures come off a real machine with a real account.** That is deliberate: a
signed-in board with solved rows on it is the thing being sold. The script scrubs `$HOME`
to `/Users/you` on the way out, which is the only detail nobody else needs.

---

## D-035 — A void return names the mutated argument by itself

Settled 2026-08-19, after `move-zeroes` would not compile.

**Decision.** Where the override table is silent, a non-design, non-`manual` solution that returns `void` is taken to answer through its **first argument**. `AnswerArg` in `internal/runner/overrides.go` makes the call, and all four generators go through it.

**Why.** A function that returns nothing has no other channel. The alternative — what the code did before — was to print `"null"`, which no test case can ever match, so every uncurated void problem failed with output the user could not learn anything from. Inferring the argument turns a guaranteed dead end into a very likely pass, and where the guess is wrong the user at least sees their own data.

The override table (D-003) stays the first authority in both directions. A curated argument wins, and so does a curated **opt-out**: `squares-of-a-sorted-array` and `remove-nth-node-from-end-of-list` sit in the table at `-1` on purpose, and inference does not second-guess them.

**Cost.** It is a guess made on the user's behalf, and one family gets it wrong. `delete-node-in-a-linked-list` is handed the node to delete and judged on a list head it never gives us; printing from that node prints the tail. LeetCode marks that family `manual`, which `metaData` has always carried and nothing read until now, so it is excluded — but the flag is LeetCode's, not ours, and a problem it forgets to mark will be guessed at.

**Mitigation.** `HasOverride` still returns false for an inferred problem, so a mismatch is phrased "check on the judge" rather than a wrong answer, exactly as D-003 requires. A guess never speaks with a curated rule's confidence. Where a guess is found to be wrong, the fix is a table entry, which is the mechanism that already exists.

---

## D-036 — A contest is a curated list with a clock, and its own judge

Settled 2026-08-30, the morning of Weekly Contest 517.

**Decision.** Contests are the third curated list — `C`, next to `P` for plans and `c` for
companies — filtering the board and sorting it by the contest's own running order. What
makes them their own feature rather than a saved filter is the two things a list cannot
have: a countdown, and a different judge.

### The endpoint that scores is not the endpoint that judges

This is the whole reason the feature exists, and it is the one thing that cannot be got
wrong. `POST /problems/{slug}/submit/` during a live contest **is accepted, is judged,
returns Accepted, and scores nothing.** The contest never sees it. Only
`POST /contest/api/{contest}/problems/{slug}/submit/` counts, and the two return the
same shape, so a wrong call looks exactly like a right one until the standings do not
move — an hour later, when nothing can be done about it.

So `SubmitContest` is a separate method from `Submit` and `leetui contest submit` is a
separate verb from `leetui submit`. Neither guesses. A single command that inferred "is a
contest running?" would be right most of the time, and the times it was wrong would cost
a rank nobody could recover.

`postJSON` grew a sibling, `postJSONTo`, taking the Referer explicitly: a contest
submission is posted from the contest's copy of the problem page, and LeetCode rejects a
submission whose Referer does not plausibly name where it came from.

### Empty is the normal answer, and must never be believed

`contestQuestionList` returns an **empty list** for a contest that has not opened — the
same response as a slug that does not exist. Three places had to be taught this:

- `SetContest` treats an empty list as a **no-op, not a wipe.** Refreshing is the normal
  way to use this feature (it is how the problems appear at the start), and a refresh that
  erased the four problems being worked on would be the worst bug this app could have.
- The syncer reports the count and says nothing about what it means; it does not own the
  clock.
- The phase is what distinguishes "not open yet" from "no such contest", and only the
  caller knows the time.

`TestSetContestEmptyDoesNotWipe` is the guard, and it is the most important test here.

### Times are read, never assumed

`startTime` is a Unix **second** and `duration` is in **seconds**. LeetCode sends
milliseconds on other endpoints, and reading this one as milliseconds puts every contest
in 1970 while the countdown still appears to work. `TestStartTimeIsSecondsNotMillis` pins
it against the real value observed for Weekly Contest 517.

Ninety minutes is not hardcoded anywhere. It is what both weeklies and biweeklies run
today, and a constant would be silently wrong the first time LeetCode ran something else.

`PhaseAt` and `Remaining` take a `now` rather than reading the wall clock, so the rail can
tick them and a test can pin them. `Remaining` never returns a negative: `-1m23s` on a
countdown reads as a broken clock, and it is the state every ended contest would sit in.

### Credit is the running order, and is NOT a difficulty

The contest API gives no difficulty field. It gives `credit` — 3/4/5/6 across a weekly's
four problems — which is the scoring weight and also the order they are meant to be read
in. The board sorts on it.

Mapping it onto Easy/Medium/Medium/Hard was rejected. It looks obvious and it is a guess:
a weekly's second problem is routinely an Easy and its third routinely a Hard. So
`ContestQuestion.Summary()` leaves difficulty **blank**, which renders as unknown. A blank
is honest; a wrong difficulty is a lie the board repeats every redraw, and the real one
arrives with the next problem-list sync anyway.

### `problem_contests` has no foreign key, and stores its own `question_id`

Both follow from the same fact: **a live contest's problems are not in the problem set.**
They are added when it ends. A foreign key to `problems` would reject the exact rows this
feature exists to store — the todo/marks bargain of D-022, for the same reason.

And `question_id` is stored on the contest row rather than joined for, because during a
contest there is nowhere else on the machine it exists, and without it no submission can
be built at all.

### The CLI matters more here than the app

Ninety minutes is not the time to learn a screen. `leetui contest pull <slug>` lays out
all four folders in one command, so an editor can be open on the first problem before the
timer starts, and a problem that fails to lay out does not stop the others — three folders
beats an error and none. Where the API cannot supply a statement, the error names
`leetcode.com/contest/<slug>/`, because during a contest a fallback that works beats a
diagnosis that is correct.

### Cloudflare guards `/contest/api/`, and a session is the key

Worth recording because it looks like a wall and is not. Every `/contest/api/` path
answers **403 with a Cloudflare challenge to an anonymous client**, regardless of user
agent — which is what makes it look like the REST contest API needs a headless browser or
a stealth driver. It does not. **With the session cookie attached it answers 200.** The
challenge is for clients with no session at all, not for this one.

That was established by probing with the app's own credentials rather than guessed:

| path | signed out | signed in |
|---|---|---|
| `/contest/api/info/{slug}/` | 403, CF challenge | **200, JSON** |
| `/contest/api/{c}/problems/{s}/submit/` (GET) | — | **405 Method Not Allowed** |
| `/problems/{s}/submit/` (GET) | — | 405 Method Not Allowed |

**The 405 is the useful one.** A wrong path returns 404; 405 means the path exists and
POST is what it wants — and the contest endpoint behaves identically to the ordinary
submit endpoint that this app has shipped for months. That is as far as the submit URL can
be verified without making a real scoring submission to a live contest, which is not a
thing to test with. So: the URL is confirmed, the request body is the same one the working
endpoint takes, and only the round trip itself is unproven.

### Registration IS checked, and it is the most valuable thing the tool says

An earlier draft of this entry said registration could not be checked, because
`ContestNode` has no `registered` field. That was true of GraphQL and wrong about the API:
**`/contest/api/info/{slug}/` returns `registered` for the calling account**, and it
answers before a contest opens, which is exactly when it is worth knowing.

This matters more than anything else here. An unregistered submission is judged, comes
back **Accepted**, and scores nothing — the same silent failure the separate submit
endpoint exists to prevent, one level up, and one nobody notices until the standings do
not move. So `leetui contest` and `leetui contest pull` both check it and say so loudly,
naming the page with the button on it. LeetCode has no registration endpoint, so this is
reported, not fixed.

The same response carries the **real difficulty** (`1`/`2`/`3`), which is why `Summary()`
passes a difficulty through instead of always leaving it blank. The rule that it is never
*derived from credit* is unchanged — it is either the real one or empty.

**Both are enrichment, never load-bearing.** They need a session; the browse path is
GraphQL and works signed out exactly as before. A failed enrichment is silence, not an
error: "not registered" and "could not ask" must not look the same, which is why
`RegistrationKnown` exists alongside `Registered`.

---

## Open items

- [ ] Which browsers browser-cookie-import supports at v1 (Chrome only, or + Firefox/Arc/Brave)
- [x] Whether `notes.md` is committed to the GitHub repo or gitignored by default — **committed, opt-out via `git.commit_notes`** (D-024). It is the user's own writing and the reason a solutions repo is worth reading; anyone who would rather not publish a half-finished thought turns it off.
- [~] Rate-limit budget. **Measured 2026-08-07:** a full 4,013-problem sync at **8 req/s**
  completed in 19.9s with no 429 and a 4.7 MB peak heap. That is evidence the shipped
  default of 2 is conservative, not proof of a ceiling — finding the actual limit means
  provoking one, which risks an account. `internal/syncer/cost_test.go` re-runs the
  measurement.
- [x] Whether premium study plans get a dedicated view or fold into company packs — **their own view on `P`** (D-031). They are not premium, they are not timeframed, and they arrive in one request; the only thing they share with a pack is the picker widget.
