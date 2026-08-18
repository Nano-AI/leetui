#!/usr/bin/env python3
"""Record a session of the real program into something a web page can replay.

`capture.sh` drives leetui inside tmux and snapshots the screen every few dozen
milliseconds. This turns that pile of snapshots into one small JSON file: a
palette of the styles used, then one frame per snapshot holding only the lines
that changed since the last one.

Only the lines that changed, because a terminal session mostly holds still. A
120x34 screen is 34 lines; pressing a key usually repaints two of them. Sending
all 34 every frame costs about thirty times more for the same picture.

Frames arrive as `--- <ms> ---` separated blocks on stdin.

    cat frames.txt | record.py -o site/shots/demo.json
"""

import sys, json, html, argparse

sys.path.insert(0, __file__.rsplit('/', 1)[0])
from ansi2svg import parse, DEFAULT_FG  # noqa: E402

BG = '#0F1014'


class Palette:
    """Distinct cell styles, numbered.

    A screen full of amber-on-ink uses one style for thousands of cells, so the
    lines can carry a small integer instead of a colour pair.
    """

    def __init__(self):
        self.index = {}
        self.list = []

    def id(self, pen):
        fg = (pen.bg or BG) if pen.reverse else (pen.fg or DEFAULT_FG)
        bg = (pen.fg or DEFAULT_FG) if pen.reverse else pen.bg
        key = (fg, bg, pen.bold, pen.faint)
        if key not in self.index:
            css = 'color:' + fg
            if bg:
                css += ';background:' + bg
            if pen.bold:
                css += ';font-weight:700'
            if pen.faint:
                css += ';opacity:.55'
            self.index[key] = len(self.list)
            self.list.append(css)
        return self.index[key]


def one_cell(ch):
    """True for glyphs the page's monospace font is trusted to draw one cell wide.

    ASCII and the box-drawing / block-element ranges are in every mono face. Anything
    else (⏱ ⟳ ◆ …) may fall back to a symbol font with its own advance, and one
    wide glyph shoves the rest of the line off the grid. Those get their own span
    so the CSS can pin them to a column.
    """
    o = ord(ch)
    return o < 0x7f or 0x2500 <= o <= 0x259F


def line_html(cells, palette, cols):
    """One row as spans, runs of one style merged."""
    out, run, run_id = [], [], None

    def flush():
        if run:
            out.append('<i class=p%d>%s</i>' % (run_id, html.escape(''.join(run))))

    for ch, pen in cells:
        sid = palette.id(pen)
        ch = ch or ' '
        if not one_cell(ch):
            flush()
            run.clear()
            run_id = None
            out.append('<i class="p%d g">%s</i>' % (sid, html.escape(ch)))
            continue
        if sid != run_id:
            flush()
            run.clear()
            run_id = sid
        run.append(ch)
    flush()
    text = ''.join(out)
    # Trailing blanks cost bytes and draw nothing.
    return text if text.strip('<i class=p0></i> ') or run_id else ''


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('-o', '--out', default='-')
    ap.add_argument('--cols', type=int, default=120)
    ap.add_argument('--rows', type=int, default=34)
    a = ap.parse_args()

    raw = sys.stdin.buffer.read().decode('utf-8', 'replace')
    blocks = raw.split('\x1e')          # record separator, written by capture.sh
    palette = Palette()
    frames, previous = [], {}

    for block in blocks:
        if not block.strip():
            continue
        delay, _, screen = block.partition('\n')
        try:
            delay = int(delay.strip())
        except ValueError:
            continue

        rows = parse(screen)
        rows = rows[:a.rows] + [[]] * max(0, a.rows - len(rows))

        changed = {}
        for i, cells in enumerate(rows):
            markup = line_html(cells, palette, a.cols)
            if previous.get(i) != markup:
                changed[str(i)] = markup
                previous[i] = markup

        # A frame that repaints nothing is still time passing, so it is kept.
        frames.append({'d': delay, 'l': changed})

    doc = {'cols': a.cols, 'rows': a.rows, 'bg': BG,
           'styles': palette.list, 'frames': frames}
    text = json.dumps(doc, separators=(',', ':'))

    if a.out == '-':
        sys.stdout.write(text)
    else:
        open(a.out, 'w').write(text)
        painted = sum(len(f['l']) for f in frames)
        print('%s  %d KB  %d frames  %d lines painted  %d styles'
              % (a.out, len(text) // 1024, len(frames), painted, len(palette.list)),
              file=sys.stderr)


if __name__ == '__main__':
    main()
