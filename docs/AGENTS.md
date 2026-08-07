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
drawing. `todo` additionally offers `--json`.

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

---

## Configuration

`LEETUI_CONFIG_DIR` relocates the config directory, which is how you give an agent its own
profile without touching the user's:

```sh
LEETUI_CONFIG_DIR=/tmp/agent-profile leetui todo --json
```

The database lives in `~/.local/share/leetui/`. Session cookies are in the OS keychain,
never on disk, and never printed — `leetui` will not hand you a session token.

---

## What this surface will not do

- **Sign in.** Credentials go through the interactive app, into the OS keychain. There is
  no flag that takes a cookie.
- **Print credentials.** Nothing here outputs a session token, and debug logging redacts
  them.
- **Submit without being asked.** See above; it is real and public.
- **Push.** There is no command for it. Publishing happens from the interactive app, from
  a keypress, behind a confirmation naming the remote — never from a script.
- **Sync the whole problem set.** That is thousands of throttled requests; press `S` in
  the app. Individual problems are fetched on demand by `pull`, `run`, and `todo add`.
