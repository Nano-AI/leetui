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
SOCKET="leetui-capture-$$"
INNER="${SOCKET}-inner"
OUTER="${SOCKET}-outer"
# Use private servers; a failed capture must not kill a reader's tmux session.
tmux() { command tmux -L "$SOCKET" "$@"; }
cleanup() {
  command tmux -L "$SOCKET" kill-server 2>/dev/null || true
  command tmux -L "$INNER" kill-server 2>/dev/null || true
  command tmux -L "$OUTER" kill-server 2>/dev/null || true
  [ -z "${REC:-}" ] || rm -f "$REC"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf -v BIN_Q '%q' "$BIN"
mkdir -p "$OUT"

start() { # start <cols> <rows>
  tmux kill-session -t "$SESSION" 2>/dev/null || true
  tmux new-session -d -s "$SESSION" -x "$1" -y "$2"
  tmux send-keys -t "$SESSION" "clear; TERM=xterm-256color $BIN_Q" Enter
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

# The editor split: leetui on the left, the real nvim on the right, holding the
# real solution file. `e` asks tmux for the pane, so the only way to photograph
# it is to photograph tmux itself — hence one session nested inside another. The
# outer one sees the inner window already composited, divider and all, which is
# what a reader would see.
editor_shot() {
  local I="command tmux -L $INNER" O="command tmux -L $OUTER" c work_q
  printf -v work_q '%q' "$WORK/0020-valid-parentheses"
  $O kill-server 2>/dev/null || true; $I kill-server 2>/dev/null || true
  $O new-session -d -s cap -x 150 -y 28
  $O send-keys -t cap "clear; TERM=xterm-256color tmux -L $INNER new-session -s ed -x 150 -y 28" Enter
  sleep 2.5

  # The inner tmux is scenery, so it says nothing: its status bar carries the
  # session name and this machine's hostname, and its active-pane border is a
  # green that belongs to the judge (D-017) and to nothing else on this page.
  $I set -t ed -g status off
  $I set -t ed -g pane-border-style "fg=#2A2D36"
  $I set -t ed -g pane-active-border-style "fg=#2A2D36"

  $I send-keys -t ed "cd $work_q && clear && TERM=xterm-256color $BIN_Q" Enter
  sleep 4
  for c in / v a l i d Space p a r e n Enter; do $I send-keys -t ed "$c"; sleep 0.45; done
  sleep 0.7; $I send-keys -t ed G; sleep 0.7
  $I send-keys -t ed Enter; sleep 2.5
  $I send-keys -t ed e; sleep 6            # nvim needs a moment to paint

  # Three things about a real editor do not survive being photographed:
  #  - relativenumber prints "1 1 2 3" around the cursor, which reads as a bug
  #  - the statusline is powerline glyphs from a Nerd Font, and a browser
  #    falling back to a symbol font draws them as boxes at the wrong width
  #  - tmux trims trailing blanks, so a painted background stops at each line's
  #    last character and the block comes out ragged; matching the terminal's
  #    own background leaves nothing to trim
  $I send-keys -t ed ':set number norelativenumber laststatus=0 signcolumn=no' Enter
  sleep 1
  $I send-keys -t ed ':hi Normal guibg=NONE ctermbg=NONE | hi EndOfBuffer guibg=NONE ctermbg=NONE | hi CursorLine guibg=NONE ctermbg=NONE | hi LineNr guibg=NONE ctermbg=NONE' Enter
  sleep 2

  $O capture-pane -t cap -e -p | python3 site/tools/ansi2svg.py -o "$OUT/editor.svg"
  $I kill-server 2>/dev/null || true; $O kill-server 2>/dev/null || true
}
WORK=${LEETUI_WORKSPACE:-$HOME/leetcode}
editor_shot

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

shell doctor "$BIN_Q doctor" 4
shell run    "cd 0020-valid-parentheses && $BIN_Q run" 6 --crop 2:12

tmux kill-session -t "$SESSION" 2>/dev/null || true

# The captures come off a real machine with a real account on it. The board is
# meant to show a signed-in session — that is the point — but nobody needs this
# machine's home directory.
python3 - "$OUT" <<'SCRUB'
import pathlib, re, sys, os, subprocess
home = os.path.expanduser('~')
host = subprocess.run(['hostname'], capture_output=True, text=True).stdout.strip()
for f in pathlib.Path(sys.argv[1]).glob('*.svg'):
    t = f.read_text()
    n = t.replace(home, '/Users/you')
    # Belt and braces: the shots are driven so nothing prints a hostname, but a
    # prompt or a status line that starts doing so should not reach the site.
    for h in filter(None, [host, host.split('.')[0]]):
        n = n.replace(h, 'this-machine')
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
tmux send-keys -t "$SESSION" "clear; TERM=xterm-256color $BIN_Q" Enter
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
