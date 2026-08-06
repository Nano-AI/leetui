# Departure Board — visual system

The design direction for leetui. Every color, glyph, and motion decision in the codebase
derives from this file. If something on screen is not justified here, it does not belong.

---

## The thesis

**A submission queue is a departure board.** Work is submitted, sits pending, gets judged,
resolves. That is a split-flap airport board, exactly. So the app *is* one: amber phosphor
on flap-black, fixed-width cells that read as physical flaps, and status that **flips**
into place rather than appearing.

This is drawn from the subject rather than applied to it. A terminal's monospace grid is
already a grid of fixed cells — the metaphor costs nothing to render and everything else
falls out of it.

---

## Color

Six system tokens. Three verdict tokens. Nothing else.

| Token      | Hex       | Role |
|------------|-----------|------|
| `ink`      | `#0F1014` | Page base — the board's shadow box |
| `flap`     | `#1B1D24` | Flap face — panel and row backgrounds |
| `hinge`    | `#2A2D36` | The seam between flaps — every rule, border, divider |
| `amber`    | `#E8A33D` | **The system's voice** — labels, selection, headers, timer, wordmark |
| `bone`     | `#E6E3DA` | **Content** — problem text, code, editorial prose |
| `dim`      | `#6B7080` | Secondary — metadata, disabled, hints |

| Verdict    | Hex       | Meaning |
|------------|-----------|---------|
| `ac`       | `#4FB477` | Accepted |
| `wa`       | `#D65A5A` | Wrong Answer, Runtime Error, Compile Error |
| `tle`      | `#C9822E` | Time Limit Exceeded, Memory Limit Exceeded |

### The discipline

> **Amber is the system speaking. Bone is content. Green and red belong to the judge alone.**
>
> **One exception, added 2026-08-06 (D-017):** difficulty borrows LeetCode's own teal /
> amber / red. Everyone who has used LeetCode reads those without a legend, which no
> invented ramp achieves. It is contained to the three-character `DIF` column and the
> problem heading — never beside the submission queue, where the flip lands. Nothing else
> in the app may borrow them.

Nothing else in this application is ever green or red. Not a success toast, not a
progress bar, not a git status. Progress bars are amber. Git status is bone and dim.
Difficulty is the single exception above, and it is the last one.

This is the whole reason a verdict lands. Spend the boldness in one place.

### Difficulty, in LeetCode's colours

**Revised 2026-08-06 (D-017).** Difficulty was a weight ramp on the `dim`→`bone`→`amber`
axis, so that green and red stayed the judge's. It now uses LeetCode's own palette, read
off their dark theme:

```
ESY   #1CBABA  teal,  bold
MED   #FFB700  amber, bold
HRD   #F63737  red,   bold
```

The borrowed palette is legible with no legend to anyone who has used the site, which the
weight ramp never was — it had to be learned. That is worth the one dent in the colour
rule.

It stays a three-character tag in a fixed-width column, and it is the **only** place
difficulty is encoded.

An earlier pass also thickened a half-block bracket around the problem ID. It is gone —
see [Retired: difficulty-by-bracket-mass](#retired-difficulty-by-bracket-mass) under
Structure for why, and do not reintroduce it.

---

## Type

The terminal gives one family, so the three "typefaces" are three *treatments*.
They must stay visually distinct or the system collapses.

**Display — "flap face".** Uppercase, one space between every character, bold.
Used **only** for verdicts and the wordmark. Nothing else ever gets letterspaced.

```
A C C E P T E D          W R O N G   A N S W E R          L E E T U I
```

**Body.** Terminal default weight, `bone`. Problem statements, editorial prose, code.
Glamour's theme maps onto this. Never letterspaced, never uppercased.

**Utility.** Uppercase, `dim`, no letterspacing. Column headers, eyebrows, key hints.

```
SUBMISSION QUEUE      RUNTIME     COMPANY     6 MONTHS
```

Scale: the terminal has one size, so hierarchy comes from **weight → case → color →
indentation**, applied in that order. Never fake a size with box-drawing characters.

---

## Structure

**The board is a framed fixture.** A split-flap board is a housing with a bezel, a header
strip, and rigid column dividers — so every pane carries a frame, and the board is a real
grid with vertical rules between columns.

An earlier pass left everything borderless and it read as soup. Frames are not a
concession to convention here; the metaphor requires them.

- **`╭─╮ │ ╰─╯`** — rounded bezels around every pane. A board housing is a moulded
  object, not a cut sheet.
- **`│` between columns, `├─┼─┤` under the header, `┴` closing the bottom** — the grid
  closes on all four sides. Column rules run the full height even when rows run out; a
  grid that stops halfway reads as broken rather than empty.
- **`▌ 0146`** — a plain zero-padded figure, like a flight number. A rigid four-digit
  field keeps the left margin dead straight. The amber bar in the first column marks the
  selected row, costs no width, and disturbs no alignment.
- ~~**`▁▂▃▄▅▆▇█`** — sparkline for acceptance rate.~~ **Retired 2026-08-06 (D-020).** The
  first person to read the board asked what it meant, which is the whole answer: the
  column now says `58%`. Clever loses to legible on a scanning surface. The component
  survives for anything that genuinely wants a bar rather than a number.
- **Focused pane** is marked by its bezel turning `amber`. Not by a background shift, not
  by a border-weight change.
- **Rows are banded.** `#15171D` on every other row, `#2A2D36` under the cursor. Added
  2026-08-06 (D-021) because the vertical rules separate the columns but do nothing to
  carry the eye ACROSS a row — tracing a title to its state column meant counting. The
  band is doing its job when it is never noticed, only that the row stays together.

### Retired: difficulty-by-bracket-mass

The first pass wrapped each ID in half-block brackets whose *mass* encoded difficulty
(`▏0001▕` / `▐0146▌` / `██0042██`). It was wrong twice over. The `DIF` column already
states difficulty, so it was the same fact twice — decoration wearing structure's
clothes. And three different bracket widths made the left edge ragged and noisy.

**Difficulty is stated once, by `Difficulty.Render()`.** Do not encode it a second time.

### Row state is a word

`SOLVED` / `TRIED` / `LOCKED` in the state column, not a glyph. A `✓` needs a legend;
`SOLVED` does not. `LOCKED` covers premium problems this session cannot open, which is
what the reader most needs to know about a row they cannot use.

Solved is **bone and bold, never green** — green belongs to the judge alone.

### No numbered markers

No `01 / 02 / 03` eyebrows anywhere. Problems already carry LeetCode's own numbering, and
that number means something — inventing a second sequence beside it is noise pretending
to be structure.

---

## Signature: the flip

**One animation. Used consistently. Nowhere else.**

Any cell whose state resolves does not repaint — it flips. Six frames at 40ms, shearing
through the half-block ramp before settling on the new text:

```
frame 0   ▔▔▔▔▔▔▔▔     (old text collapsing)
frame 1   ▀▀▀▀▀▀▀▀
frame 2   ▚▚▚▚▚▚▚▚     (mid-flip, amber)
frame 3   ▄▄▄▄▄▄▄▄
frame 4   ▁▁▁▁▁▁▁▁
frame 5   ACCEPTED     (settles, verdict color)
```

Driven by `tea.Tick`. Applies to:
- submission verdicts (`PENDING` → `JUDGING` → `ACCEPTED`)
- problem status changes (`—` → `✓`)
- sync progress counters

**There are no spinners in this application.** A pending job renders as a flap held
mid-flip. That is the loading state, everywhere, and it is the only one.

### Motion budget

The flip is the entire motion budget. No fades, no slides, no pulsing, no easing curves
on layout. If a new animation seems necessary, the answer is that the flip should cover it.

Honor `reduce_motion = true` in config, `NO_COLOR`, and `--no-motion`: the flip resolves
instantly to its final frame. The app must be fully usable and legible with motion off —
motion is never the only channel carrying information.

---

## Layout

```
╭──────────────────────────────────────────────────────────────────────╮
│ L E E T U I                          ⏱ 12:04:33 ┊ ◆ premium ┊ ada   │  rail
╰──────────────────────────────────────────────────────────────────────╯
╭─ PROBLEMS ──────────────────────────── medium ┊ "cache" ┊ 12 ─╮
│ #      │ PROBLEM                      │ DIF │ AC  │ STATE   │    utility
├────────┼──────────────────────────────┼─────┼─────┼─────────┤    tee joints
│ ▌ 0146 │ LRU Cache                    │ MED │ ▇▃▁ │ SOLVED  │    selected
│   0460 │ LFU Cache                    │ HRD │ █▄▁ │ TRIED   │
│   1650 │ Lowest Common Ancestor III   │ MED │ ██▅ │ LOCKED  │    premium
│        │                              │     │     │         │    rules run
╰────────┴──────────────────────────────┴─────┴─────┴─────────╯    to the bezel
╭─ PROBLEM ──────────── 42.7% ACCEPTED ─╮╭─ SUBMISSIONS ────────╮
│ 146. LRU Cache  MED                   ││ 0001 python3 ▚▚▚▚▚▚  │  mid-flip
│ design ┊ hash-table ┊ linked-list     ││ 0146 go      A C C E… │
│ ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ ││ 0042 cpp     W R O N… │
│ Design a data structure that follows  ││                       │
│ `[▸ img 1 — eviction order]`          ││                       │
╰───────────────────────────────────────╯╰───────────────────────╯
 / search ┊ 1·2·3 difficulty ┊ S sync ┊ ? keys ┊ q quit              dim
```

Note what the grid does and does not carry: the ID is a plain zero-padded field, the
amber `▌` is the *only* selection marker, and difficulty appears once, in `DIF`.

**Rail** is persistent: wordmark, timer/stopwatch (D-006), streak, premium indicator.
**Board** is the browse surface — problem list, company pack, search results, all the same widget.
**Detail** renders statement or editorial through Glamour.
**Queue** is the departure board proper — the thing the whole design is named for.

`tab` cycles focus; the focused pane's rule goes amber. Panes collapse by width: under
100 columns the queue becomes a single status line in the rail; under 80 the board and
detail become full-screen views switched by `tab`.

---

## Copy

Interface words follow the same discipline as the visuals.

- **Verdicts use LeetCode's own vocabulary**, unabbreviated in the queue when width allows:
  `ACCEPTED`, `WRONG ANSWER`, `TIME LIMIT EXCEEDED`. This is the domain's real language;
  translating it into friendlier words would make the app harder to reconcile with the site.
- **Actions keep one name throughout.** The key hint says `submit`, the confirm says
  `Submit 0001 as python3?`, the queue row says `SUBMITTED`. Never `send`, `upload`, `push`.
- **Errors state what happened and what to do**, in the interface's voice, without apology:
  `Session expired. Press a to re-authenticate.`
  Not `Sorry! Something went wrong :(`
- **Empty states are invitations:**
  `No company packs yet. Press S to sync — about 4 minutes.`
  Not `No data`.
- **Local-run failures never claim authority:**
  `Local check failed. Press v to verify on the judge.`

Sentence case for prose. Uppercase reserved for the utility treatment and verdicts.

---

## Quality floor

Not announced in the UI, simply true:

- Renders correctly at 80×24. Degrades by collapsing panes, never by truncating mid-glyph.
- Every action reachable by keyboard; focus is always visibly indicated.
- `NO_COLOR` produces a legible monochrome app — verdicts fall back to the display
  treatment alone (`A C C E P T E D`), which is why verdicts are letterspaced and not
  merely colored.
- Truecolor preferred, 256-color palette mapped as fallback, detected via `$COLORTERM`.
- Unicode half-blocks and box-drawing degrade to ASCII (`|`, `-`, `+`, `#`) under
  `--ascii` for terminals with poor font coverage.
