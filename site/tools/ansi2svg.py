#!/usr/bin/env python3
"""Turn a captured terminal screen into an SVG.

The site shows the real program, not a drawing of it. `capture.sh` drives leetui
inside tmux and pipes `tmux capture-pane -e` here; what comes out is the same grid
of cells the terminal drew, in the same colours, as vector text.

SVG rather than PNG because the board is type: a raster of a 160-column grid is
either huge or mushy, and this is 30 KB and sharp at any zoom.

Grid alignment does not depend on the reader having any particular font. Every run
carries a textLength, so the glyphs are spaced to the cell whatever the fallback is.
"""

import re, sys, html, argparse, unicodedata

SGR = re.compile(r'\x1b\[([0-9;:]*)m')
OSC = re.compile(r'\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)')
CSI = re.compile(r'\x1b\[[0-9;?]*[A-Za-z]')

# xterm's first sixteen, for the rare 38;5;n under 16.
BASE16 = ['#000000','#cd0000','#00cd00','#cdcd00','#0000ee','#cd00cd','#00cdcd','#e5e5e5',
          '#7f7f7f','#ff0000','#00ff00','#ffff00','#5c5cff','#ff00ff','#00ffff','#ffffff']


def xterm256(n):
    if n < 16:
        return BASE16[n]
    if n < 232:
        n -= 16
        r, g, b = n // 36, (n // 6) % 6, n % 6
        lvl = lambda v: 0 if v == 0 else 55 + 40 * v
        return '#%02x%02x%02x' % (lvl(r), lvl(g), lvl(b))
    v = 8 + (n - 232) * 10
    return '#%02x%02x%02x' % (v, v, v)


class Pen:
    __slots__ = ('fg', 'bg', 'bold', 'italic', 'underline', 'reverse', 'faint')

    def __init__(self):
        self.reset()

    def reset(self):
        self.fg = self.bg = None
        self.bold = self.italic = self.underline = self.reverse = self.faint = False

    def copy(self):
        p = Pen()
        for s in Pen.__slots__:
            setattr(p, s, getattr(self, s))
        return p

    def key(self):
        return (self.fg, self.bg, self.bold, self.italic, self.underline, self.faint)

    def apply(self, params):
        # Empty parameter string means SGR 0.
        codes = [int(c) if c else 0 for c in params.replace(':', ';').split(';')] or [0]
        i = 0
        while i < len(codes):
            c = codes[i]
            if c == 0:
                self.reset()
            elif c == 1:
                self.bold = True
            elif c == 2:
                self.faint = True
            elif c == 3:
                self.italic = True
            elif c == 4:
                self.underline = True
            elif c == 7:
                self.reverse = True
            elif c == 22:
                self.bold = self.faint = False
            elif c == 23:
                self.italic = False
            elif c == 24:
                self.underline = False
            elif c == 27:
                self.reverse = False
            elif c == 39:
                self.fg = None
            elif c == 49:
                self.bg = None
            elif 30 <= c <= 37:
                self.fg = BASE16[c - 30]
            elif 90 <= c <= 97:
                self.fg = BASE16[c - 90 + 8]
            elif 40 <= c <= 47:
                self.bg = BASE16[c - 40]
            elif 100 <= c <= 107:
                self.bg = BASE16[c - 100 + 8]
            elif c in (38, 48):
                target = 'fg' if c == 38 else 'bg'
                if i + 1 < len(codes) and codes[i + 1] == 2:
                    r, g, b = (codes + [0, 0, 0])[i + 2:i + 5]
                    setattr(self, target, '#%02x%02x%02x' % (r & 255, g & 255, b & 255))
                    i += 4
                elif i + 1 < len(codes) and codes[i + 1] == 5:
                    setattr(self, target, xterm256(codes[i + 2] if i + 2 < len(codes) else 0))
                    i += 2
            i += 1


def width(ch):
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in 'WF' else 1


def parse(text):
    """Read the capture into rows of (char, pen) cells."""
    rows = []
    pen = Pen()
    for line in text.split('\n'):
        line = OSC.sub('', line)
        cells = []
        pos = 0
        for m in SGR.finditer(line):
            for ch in CSI.sub('', line[pos:m.start()]):
                w = width(ch)
                if w == 0:
                    continue
                cells.append((ch, pen.copy()))
                for _ in range(w - 1):
                    cells.append(('', pen.copy()))
            pen.apply(m.group(1))
            pos = m.end()
        for ch in CSI.sub('', line[pos:]):
            w = width(ch)
            if w == 0:
                continue
            cells.append((ch, pen.copy()))
            for _ in range(w - 1):
                cells.append(('', pen.copy()))
        rows.append(cells)
    while rows and not ''.join(c for c, _ in rows[-1]).strip():
        rows.pop()
    return rows


FONT = ("ui-monospace,'SF Mono',SFMono-Regular,'JetBrains Mono','Cascadia Mono',"
        "Menlo,Consolas,'DejaVu Sans Mono',monospace")


DEFAULT_FG = '#E6E3DA'


def render(rows, cw=8.4, ch=17.0, fs=14.0, pad=18.0, bg='#0F1014', radius=0, chrome=False):
    cols = max((len(r) for r in rows), default=0)
    top = pad + (26 if chrome else 0)
    w = cols * cw + pad * 2
    h = len(rows) * ch + top + pad
    out = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w:.1f} {h:.1f}" '
           f'width="{w:.1f}" height="{h:.1f}" font-family="{FONT}" font-size="{fs}" '
           f'shape-rendering="crispEdges" role="img">']
    out.append(f'<rect width="{w:.1f}" height="{h:.1f}" rx="{radius}" fill="{bg}"/>')

    if chrome:
        out.append(f'<g><circle cx="{pad+6:.1f}" cy="17" r="5.5" fill="#3A3D46"/>'
                   f'<circle cx="{pad+24:.1f}" cy="17" r="5.5" fill="#3A3D46"/>'
                   f'<circle cx="{pad+42:.1f}" cy="17" r="5.5" fill="#3A3D46"/></g>')

    # Background runs first, so no glyph is ever painted under a later rect.
    out.append('<g shape-rendering="crispEdges">')
    for y, row in enumerate(rows):
        run_bg, run_x, run_n = None, 0, 0
        for x, (c, p) in enumerate(row):
            # Reverse video swaps the pair, and either half of that pair can be
            # the terminal default — a cursor drawn as \e[7m over unstyled text is
            # exactly that case. Resolve the defaults before swapping, or the cell
            # comes out with no background and dark-on-dark text, i.e. invisible.
            b = (p.fg or DEFAULT_FG) if p.reverse else p.bg
            if b == run_bg:
                run_n += 1
                continue
            if run_bg and run_n:
                out.append(f'<rect x="{pad+run_x*cw:.2f}" y="{top+y*ch:.2f}" '
                           f'width="{run_n*cw:.2f}" height="{ch:.2f}" fill="{run_bg}"/>')
            run_bg, run_x, run_n = b, x, 1
        if run_bg and run_n:
            out.append(f'<rect x="{pad+run_x*cw:.2f}" y="{top+y*ch:.2f}" '
                       f'width="{run_n*cw:.2f}" height="{ch:.2f}" fill="{run_bg}"/>')
    out.append('</g>')

    out.append('<g shape-rendering="geometricPrecision">')
    for y, row in enumerate(rows):
        base = top + y * ch + fs * 0.78
        run, run_x, run_key, run_pen = [], 0, None, None
        def flush():
            if not run or not ''.join(run).strip():
                return
            p = run_pen
            fg = (p.bg or bg) if p.reverse else (p.fg or DEFAULT_FG)
            attrs = [f'x="{pad+run_x*cw:.2f}"', f'y="{base:.2f}"',
                     f'fill="{fg}"', f'textLength="{len(run)*cw:.2f}"',
                     'lengthAdjust="spacing"',
                     # Both spellings: xml:space is what SVG 1.1 renderers read,
                     # white-space is what SVG 2 replaced it with. Without one of
                     # them the leading spaces of an indented run collapse and the
                     # textLength stretches four glyphs across eight cells.
                     'xml:space="preserve"', 'style="white-space:pre"']
            if p.bold:
                attrs.append('font-weight="700"')
            if p.faint:
                attrs.append('opacity=".55"')
            if p.italic:
                attrs.append('font-style="italic"')
            if p.underline:
                attrs.append('text-decoration="underline"')
            out.append(f'<text {" ".join(attrs)}>{html.escape("".join(run))}</text>')
        for x, (c, p) in enumerate(row):
            k = p.key() + (p.reverse,)
            if k != run_key:
                flush()
                run, run_x, run_key, run_pen = [], x, k, p
            run.append(c or ' ')
        flush()
    out.append('</g></svg>')
    return '\n'.join(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('-o', '--out', default='-')
    ap.add_argument('--chrome', action='store_true', help='draw a window titlebar')
    ap.add_argument('--radius', type=float, default=0)
    ap.add_argument('--crop', default='', help='rows to keep, as START:END (1-based)')
    a = ap.parse_args()
    rows = parse(sys.stdin.buffer.read().decode('utf-8', 'replace'))
    if a.crop:
        s, _, e = a.crop.partition(':')
        rows = rows[(int(s) - 1 if s else 0):(int(e) if e else None)]
    svg = render(rows, chrome=a.chrome, radius=a.radius)
    if a.out == '-':
        sys.stdout.write(svg)
    else:
        open(a.out, 'w').write(svg)
        print(f'{a.out}  {len(svg)//1024} KB  {len(rows)} rows', file=sys.stderr)


if __name__ == '__main__':
    main()
