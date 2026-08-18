#!/usr/bin/env bash
# Take the site's screenshots by driving the real leetui inside tmux.
#
# Nothing on the site is drawn by hand. Every board, picker and pane in
# site/shots/ came out of this script, which starts the binary, presses the
# keys a reader would press, and hands the screen to ansi2svg.py.
#
# Re-run it after any change to the interface. It needs a synced database and
# a signed-in session, same as a user would have.
#
# The SVG is the screen and nothing else. The window around it (titlebar,
# buttons, shadow) is drawn by site/index.html, so one frame style covers
# every shot and none of them arrive with a second frame baked in.
#
#   ./site/tools/capture.sh
set -euo pipefail

cd "$(dirname "$0")/../.."
OUT=site/shots
SESSION=leetui-shot
BIN=${LEETUI:-leetui}
mkdir -p "$OUT"

start() { # start <cols> <rows>
  tmux kill-session -t "$SESSION" 2>/dev/null || true
  tmux new-session -d -s "$SESSION" -x "$1" -y "$2"
  tmux send-keys -t "$SESSION" "clear; TERM=xterm-256color $BIN" Enter
  sleep 3
}

keys() { for k in "$@"; do tmux send-keys -t "$SESSION" "$k"; sleep 0.45; done; sleep 1.2; }

shot() { # shot <name> [ansi2svg args...]
  local name=$1; shift
  tmux capture-pane -t "$SESSION" -e -p \
    | python3 site/tools/ansi2svg.py -o "$OUT/$name.svg" "$@"
}

# The board. The one screen that has to be the real 4,013-row thing.
start 158 42
shot board

# A problem, open. Statement on the left; on the right the solution file that
# is really on this disk, so the pane shows a path and a language, not a stub.
#
# G first: "valid paren" matches half a dozen problems and the cursor lands on
# the top one, which is 32. 20 is the one with a solution beside it.
keys / v a l i d Space p a r e n Enter G Enter
shot detail

# Search runs against the local database, so it answers while you type.
keys Escape Escape / t r e e
shot search --crop 1:22

# The keys screen needs the width or its two columns wrap into each other.
start 200 44
keys ?
shot keys

# The command line, as a script or an agent sees it. Real runs, real exit codes.
shell() { # shell <name> <command> <settle-seconds> [ansi2svg args...]
  local name=$1 cmd=$2 wait=$3; shift 3
  tmux kill-session -t "$SESSION" 2>/dev/null || true
  tmux new-session -d -s "$SESSION" -x 104 -y 30 -c "$WORK"
  tmux send-keys -t "$SESSION" "clear; PS1='\$ '; export TERM=xterm-256color" Enter
  sleep 1
  tmux send-keys -t "$SESSION" "clear" Enter; sleep 0.5
  tmux send-keys -t "$SESSION" "$cmd" Enter; sleep "$wait"
  tmux capture-pane -t "$SESSION" -e -p \
    | python3 site/tools/ansi2svg.py -o "$OUT/$name.svg" "$@"
}

WORK=${LEETUI_WORKSPACE:-$HOME/leetcode}
shell doctor "leetui doctor" 4
shell run    "cd 0020-valid-parentheses && leetui run" 6 --crop 2:12

tmux kill-session -t "$SESSION" 2>/dev/null || true

# The captures come off a real machine with a real account on it. The board is
# meant to show a signed-in session — that is the point — but nobody needs this
# machine's home directory.
python3 - "$OUT" <<'SCRUB'
import pathlib, re, sys, os
home = os.path.expanduser('~')
for f in pathlib.Path(sys.argv[1]).glob('*.svg'):
    t = f.read_text()
    n = t.replace(home, '/Users/you')
    if n != t:
        f.write_text(n)
        print(f'scrubbed {f.name}')
SCRUB

# ---------------------------------------------------------------------------
# The demo.
#
# One recorded session of somebody using the thing: search narrowing as the
# keys land, the company picker, the study plans, a problem opening. Every
# frame is the real screen. record.py keeps only the lines that changed, which
# is why fifteen seconds of a 120x34 terminal fits in a few tens of KB.
#
# 120 columns rather than 158: the whole board still fits, and at the width the
# page gives the hero the glyphs stay readable.
# ---------------------------------------------------------------------------

REC=$(mktemp)

frame() { # frame <ms-until-next>
  printf '\036%s\n' "$1" >> "$REC"
  tmux capture-pane -t "$SESSION" -e -p >> "$REC"
}

press() { # press <key> [frames]
  tmux send-keys -t "$SESSION" "$1"
  local n=${2:-3} i
  for i in $(seq 1 "$n"); do sleep 0.09; frame 90; done
}

hold() { sleep 0.25; frame "$1"; }   # one frame, held long enough to read

tmux kill-session -t "$SESSION" 2>/dev/null || true
tmux new-session -d -s "$SESSION" -x 120 -y 34
tmux send-keys -t "$SESSION" "clear; TERM=xterm-256color $BIN" Enter
sleep 3

hold 1100                                  # the board, sitting there
press / 2; for k in t r e e; do press "$k" 2; done
hold 1300                                  # 378 matches
press Escape 2; press Escape 2
press c 3; hold 1100                       # companies
for k in g o o; do press "$k" 2; done
hold 1200
press Escape 2; press Escape 2
press P 3; hold 1400                       # study plans
press Escape 2; press Escape 2
for k in j j j j j; do press "$k" 2; done  # walk the board
press Enter 4; hold 1800                   # a problem, open
press Escape 3; hold 900

tmux kill-session -t "$SESSION" 2>/dev/null || true
python3 site/tools/record.py --cols 120 --rows 34 -o "$OUT/demo.json" < "$REC"
rm -f "$REC"

# A PNG of the board, for the README and the link preview. GitHub renders
# an SVG through a proxy that drops the font, and a raster is the only thing
# a link unfurler will show at all. Chrome is the rasteriser because it is
# the same engine the site is checked in; skipped if it is not installed.
CHROME=${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}
if [ -x "$CHROME" ]; then
  tmp=$(mktemp -d)
  printf '%s' "<!doctype html><meta charset=utf-8><style>html,body{margin:0;background:#0F1014}img{display:block;width:1363px}</style><img src=\"file://$PWD/$OUT/board.svg\">" > "$tmp/f.html"
  "$CHROME" --headless --disable-gpu --force-device-scale-factor=2 \
    --window-size=1363,776 --virtual-time-budget=4000 \
    --screenshot="$OUT/board.png" "file://$tmp/f.html" 2>/dev/null
  rm -rf "$tmp"
  echo "board.png  $(du -h "$OUT/board.png" | cut -f1)"
else
  echo "skipped board.png: no Chrome at $CHROME" >&2
fi

ls -l "$OUT"
