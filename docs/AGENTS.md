# leetui for agents

A stable, scriptable surface over leetui's core. Everything here is safe to call from an
automated tool, and the one command that is not read-only says so loudly.

leetui with no arguments opens an interactive terminal app — **never invoke that from an
agent.** It takes over the terminal and waits for a human. Use the subcommands below.

---

## The contract

**Exit codes are the API.** Branch on these, not on the text:

| code | meaning |
|------|---------|
| `0` | it worked — and for `run`, every test case passed |
| `1` | it ran and the answer was wrong |
| `2` | it could not run at all: unknown problem, missing toolchain, not signed in |

**Naming a problem.** Every command that takes one accepts four shapes, so you rarely need
to look anything up:

```
two-sum                          the slug
0001-two-sum                     the folder name
~/leetcode/0001-two-sum          a path to the folder
~/leetcode/0001-two-sum/solution.py   a path to a file inside it
                                 omitted — means the current directory
```

**Output is plain text on stdout**, progress and errors on stderr. No colour, no box
drawing. `todo` and `mark` additionally offer `--json`.

**Flags may appear anywhere**, before or after the problem:

```sh
leetui todo add two-sum --note "from the JD"     # works
leetui todo add --note "from the JD" two-sum     # also works
```

---

## Commands

### `leetui todo` — the list of problems to work through

The reason this surface exists. Queue problems from anywhere and they are waiting on the
board next time a human opens leetui.

```sh
leetui todo                                  # list, human-readable
leetui todo --json                           # list, for parsing
leetui todo add two-sum
leetui todo add two-sum --note "why it is here"
leetui todo add two-sum 3sum valid-parentheses   # several at once
leetui todo rm two-sum
leetui todo clear
```

**Every operation is idempotent.** Adding something twice is not an error and does not
move it in the queue; removing something absent is not an error either. You never need to
check before you act, which means you never race.

The JSON is a stable array — never `null`, so a loop over it needs no special case:

```json
[
  {
    "slug": "two-sum",
    "title": "Two Sum",
    "difficulty": "Easy",
    "id": 1,
    "status": "ac",
    "note": "from the JD",
    "added_at": "2026-08-06T23:42:32Z",
    "url": "https://leetcode.com/problems/two-sum/"
  }
]
```

`status` is `ac` (solved), `notac` (attempted), or absent (untouched). `title`,
`difficulty`, `id`, and `url` are omitted for a problem this machine has not synced yet —
the entry is still valid, there is just less known about it.

The list is ordered **oldest first**. It is a queue: the thing added three weeks ago is
the one most in danger of being forgotten.

### `leetui mark` — which problems were worth doing

A verdict per problem, kept so a **second pass** through a study plan or a company pack
can skip what taught nothing and revisit what did.

This is the surface to use when a human asks you to triage a list — "go through Top
Interview 150 and flag the ones worth redoing". It is the one place where your judgment is
written down somewhere the app will show them.

```sh
leetui mark up two-sum                       # worth doing again
leetui mark down 3sum                        # not worth another pass
leetui mark up two-sum --note "hash-map insight"
leetui mark up two-sum 3sum lru-cache        # several at once
leetui mark clear two-sum
leetui mark                                  # list, human-readable
leetui mark --json                           # list, for parsing
leetui mark --json --up                      # only the important ones
leetui mark --json --down                    # only the written-off ones
```

`important` / `unimportant` and bare `+` / `-` are accepted as synonyms for `up` / `down`.

**Three states, not a score.** A problem is `up`, `down`, or unmarked. There is no
counter to increment, so you never have to read a value before writing one.

**The mark reorders the board.** `up` floats a problem to the top of every ordering and
gilds it; `down` sinks it to the bottom and greys it; unmarked sits between them.

**`down` demotes; it does not delete.** A problem you mark down stays in the collection,
stays searchable, and stays in its study plan. This matters for how you should use it:
marking something `down` is cheap and reversible, so triaging aggressively costs the user
nothing. It is not a destructive operation and does not need confirming.

**A mark is not a todo.** They answer different questions and neither touches the other:

| | means | when solved |
|---|---|---|
| `leetui todo` | "get to this" — a queue | you take it off |
| `leetui mark` | "this was worth doing" — a judgment | **it stays** |

"Solved **and** worth doing again" is the state that makes a second pass possible, and the
todo list cannot express it. That is why this exists separately.

**Every operation is idempotent.** Marking twice is not an error. Flipping `up` to `down`
takes one call, not a clear first. Clearing something unmarked is not an error. You never
need to check before you act.

**An empty `--note` never erases an existing one**, so a bulk pass cannot wipe a reason a
human wrote. Pass a real note to replace it.

You may mark a problem this machine has not synced yet; the verdict survives until it is.

The JSON is a stable array — never `null`:

```json
[
  {
    "slug": "two-sum",
    "mark": "up",
    "title": "Two Sum",
    "difficulty": "Easy",
    "id": 1,
    "status": "ac",
    "note": "hash-map insight",
    "marked_at": "2026-08-09T22:22:54Z",
    "url": "https://leetcode.com/problems/two-sum/"
  }
]
```

`mark` is `up` or `down` — always present, always one of those two. The remaining fields
behave exactly as `todo`'s do, including being omitted for an unsynced problem.

Ordered **newest first**, the opposite of the todo list. A todo is a queue where the
oldest item is most at risk of being forgotten; a mark is a judgment, and the most recent
one is the one still being acted on.

**In the app**, `+` and `-` set the verdict on the row under the cursor and `i` cycles the
board through all → important → unimportant, in a column headed `MARK`.

A mark you write from here reaches the database at once, but the board does not poll it —
an already-open leetui picks it up the next time it reloads its rows, which any search,
filter change, sync, or restart does. The same is true of `todo`. If a human is watching
while you triage, tell them to press `i` twice to see the marks land.

### `leetui pull <problem>` — lay out the files

Creates the problem folder: `README.md` with the statement, a scaffolded solution file,
and `testcases.txt` seeded from the examples. Prints the title and the solution's path.

```sh
leetui pull two-sum
leetui pull two-sum --lang cpp
```

Safe to re-run. It never overwrites a solution you have edited.

### `leetui run [problem|file]` — run the tests locally

```sh
leetui run                       # from inside the problem folder
leetui run two-sum
leetui run path/to/solution.py   # language inferred from the extension
leetui run two-sum --lang cpp
```

Prepares first, so this works on a machine that has never seen the problem. Local
execution covers **Python, Go, and C++**; anything else exits `2` and tells you to submit
instead.

`--watch` stays open and re-runs on every save, clearing between runs so the pane shows
one result. It is for a human in a terminal pane, **not for an agent** — it never exits on
its own.

A failing case prints input, expected, and actual:

```
case 3  FAIL
  in    "(]"
  want  false
  got   true

2 of 5 cases mismatched.
```

**A local mismatch is not authoritative.** LeetCode's metadata cannot express in-place
mutation, order-insensitive answers, or float tolerance, so for problems without a curated
comparator the output says so and points at the judge. Treat exit `1` as "look at this",
not as "definitely wrong".

### `leetui submit [problem|file]` — send it to the judge

```sh
leetui submit two-sum
```

**This is the one command with outside effects.** It creates a real submission on the
user's LeetCode account, which is visible in their public submission history and cannot be
undone. Do not call it speculatively, in a loop, or to "check" an answer — `run` is for
checking. Get explicit confirmation from the user before an agent submits on their behalf.

Requires a signed-in session; exits `2` if there is none. Only the region between the
`@leetui code=start` / `@leetui code=end` markers is sent.

**An accepted submission commits.** If the workspace is a git repository, `submit` commits
the solution, statement, test cases, and notes on an `Accepted` verdict, and prints a line
to stderr saying so. It **never pushes** — nothing in this surface reaches a remote.

The commit is best effort and never changes the exit code: a repository that will not take
one is a line on stderr and nothing more. Turn it off with `commit_on_accepted = false`
under `[git]`, or by giving the agent its own `LEETUI_CONFIG_DIR` profile.

### `leetui path <problem>` — print the folder

```sh
cd "$(leetui path two-sum)"
```

Read-only. Does not create anything.

---

## Working with the files

A solution file has two regions:

```python
# 20. Valid Parentheses · Easy
# https://leetcode.com/problems/valid-parentheses/
#
# Everything above the marker is local scaffolding, for your editor
# and the local runner. Only the marked region is submitted.

from typing import Any, Dict, List, Optional, Set, Tuple

# @leetui code=start
class Solution:
    def isValid(self, s: str) -> bool:
        ...
# @leetui code=end
```

**Edit only between the markers.** Everything above is imports and a package clause that
LeetCode supplies itself and would reject as duplicates. The markers are what separate
them; removing them makes the whole file the submission.

`testcases.txt` is plain and hand-editable — input lines, a line reading `output:`, the
expected value, then a blank line between cases. Adding your own cases there is supported
and they are never overwritten.

---

## A worked loop

Queue from a job description, then work the list:

```sh
leetui todo add two-sum group-anagrams lru-cache --note "phone screen prep"

leetui todo --json | jq -r '.[] | select(.status != "ac") | .slug' | while read -r slug; do
  leetui pull "$slug"
  # ... write a solution into the marked region ...
  if leetui run "$slug"; then
    echo "$slug passes locally"
  fi
done
```

Note the loop calls `run`, not `submit`. Submitting is the user's decision.

## A second worked loop — triage, then redo

The pattern `mark` was built for. A human works through Top Interview 150, an agent records
which ones were worth the time, and a later pass re-does only those.

Triaging, one problem at a time as they are reviewed:

```sh
leetui mark up   lru-cache      --note "eviction order is the whole problem"
leetui mark down remove-element --note "same as the one before it"
```

Then the re-do pass — solved **and** still marked important, which is exactly the query
the todo list could not express:

```sh
leetui mark --json --up \
  | jq -r '.[] | select(.status == "ac") | .slug' \
  | while read -r slug; do
      leetui pull "$slug"      # fresh scaffold; your old solution is never overwritten
      echo "redo: $slug"
    done
```

To see what is left untriaged in a plan, diff the marks against what the human has solved:

```sh
leetui mark --json | jq -r '.[].slug' | sort > /tmp/marked
# ... compare against the plan's slugs; anything absent has no verdict yet
```

---

## Configuration

`LEETUI_DEBUG=1` writes a request trace to `~/.local/share/leetui/debug.log` — what was
sent, what came back, and the full body of anything that failed to decode. Credentials are
reduced to `abcd…wxyz(837)` form before they reach it, so the file is safe to attach to a
bug report. Use it when a command fails in a way its error message does not explain.

`LEETUI_CONFIG_DIR` relocates the config directory, which is how you give an agent its own
profile without touching the user's:

```sh
LEETUI_CONFIG_DIR=/tmp/agent-profile leetui todo --json
```

The database lives in `~/.local/share/leetui/`.
Session cookies go to the best store the machine has: the OS keychain, then a credential helper you configure, then a `0600` file (D-002a).
`leetui doctor` reports which one is in use.
They are never printed either way — `leetui` will not hand you a session token.

### Signing in without a keychain

`leetui doctor` has an `auth` section that reports which store is in use and what else this machine offers.

If it says the credentials are in a `0600` file, that is the last resort and it is working as designed — but there is a better option if you have `pass`, `gpg`, `age`, or anything else that can hold a secret.
Point `[auth] helper` at a command implementing Docker's credential-helper protocol:

```toml
[auth]
helper = "leetui-credential-pass"
```

The whole implementation, against `pass`:

```sh
#!/bin/sh
# leetui-credential-pass — put this on your PATH, chmod +x
case "$1" in
  store) pass insert -m -f leetui/credentials >/dev/null ;;
  get)   pass show leetui/credentials ;;
  erase) pass rm -f leetui/credentials >/dev/null ;;
esac
```

`get` writes `{"session":"…","csrftoken":"…","username":"…"}` to stdout and exits non-zero when it holds nothing yet.
`store` reads that same JSON on stdin. Anything the helper writes to stderr is shown to the user on failure; stdin and stdout are never logged.

For CI, or a shell that already sources secrets from somewhere, skip storage entirely:

```sh
export LEETUI_SESSION="…"
export LEETUI_CSRF="…"
```

Both are required together, and they take precedence over everything on disk.

---

## What this surface will not do

- **Sign in silently.** `leetui login` exists, for machines where the full-screen app is
  awkward to reach, and it takes `--session`/`--csrf`/`--stdin`. What it will not do is
  store anything it has not first verified against LeetCode, or fall back to a weaker
  credential store without saying so on stdout.
- **Print credentials.** Nothing here outputs a session token, and debug logging redacts
  them.
- **Submit without being asked.** See above; it is real and public.
- **Push.** There is no command for it. Publishing happens from the interactive app, from
  a keypress, behind a confirmation naming the remote — never from a script.
- **Sync the whole problem set.** That is thousands of throttled requests; press `S` in
  the app. Individual problems are fetched on demand by `pull`, `run`, `todo add`, and
  `mark up`/`mark down`.
- **Pull a study plan.** There is no subcommand for it — press `P` in the app. `mark`
  works on any problem regardless of which plan surfaced it, so triage does not need one.
