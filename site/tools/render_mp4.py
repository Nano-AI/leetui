#!/usr/bin/env python3
"""Turn the recorded session into an MP4, for places that will not run a page.

The site plays `demo.json` as live text, which is sharp and small and none of
which helps on LinkedIn, in a README, or anywhere a video is the only thing that
will autoplay. So: replay the frame diffs back into whole frames, hand each one
to Chrome to rasterise, and let ffmpeg carry the timing.

Chrome is the rasteriser because it is the engine the site is checked in, so the
video and the page cannot disagree about what the program looked like. ffmpeg
gets a concat list rather than a fixed frame rate, because the recording holds
still for a second on a screen worth reading and then moves in 90 ms steps.

    ./site/tools/render_mp4.py site/shots/demo.json -o site/shots/demo.mp4

Needs Chrome and ffmpeg. Both are checked before any work starts.
"""

import json, argparse, subprocess, sys, shutil, os, tempfile, html

CHROME_DEFAULT = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

PAGE = """<!doctype html><meta charset=utf-8>
<style>
  html,body{{margin:0;padding:0;background:{bg};}}
  #s{{
    font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,Consolas,monospace;
    font-size:{fs}px; line-height:1.2; white-space:pre;
    color:#E6E3DA; background:{bg};
    padding:{pad}px; width:max-content;
    font-variant-ligatures:none;
  }}
  #s div{{height:1.2em}}
  #s i{{font-style:normal}}
  #s .g{{display:inline-block;width:1ch;text-align:center;vertical-align:top}}
  {palette}
</style>
<div id=s>{body}</div>
"""


def frames_of(cast):
    """Replay the diffs into a list of complete screens."""
    lines = [""] * cast["rows"]
    out = []
    for f in cast["frames"]:
        for k, v in f["l"].items():
            lines[int(k)] = v
        out.append((list(lines), f["d"]))
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cast", nargs="?", default="site/shots/demo.json")
    ap.add_argument("-o", "--out", default="site/shots/demo.mp4")
    ap.add_argument("--font-size", type=float, default=17.0)
    ap.add_argument("--pad", type=int, default=22)
    ap.add_argument("--scale", type=int, default=2, help="device pixel ratio")
    ap.add_argument("--chrome", default=os.environ.get("CHROME", CHROME_DEFAULT))
    ap.add_argument("--keep", action="store_true", help="leave the PNG frames on disk")
    a = ap.parse_args()

    if not os.path.exists(a.chrome):
        sys.exit("no Chrome at %s (set CHROME=...)" % a.chrome)
    if not shutil.which("ffmpeg"):
        sys.exit("ffmpeg is not on PATH")

    cast = json.load(open(a.cast))
    palette = "\n  ".join("#s .p%d{%s}" % (i, css) for i, css in enumerate(cast["styles"]))
    screens = frames_of(cast)

    # Every frame is the same size, so measure once off the first one. A cell is
    # 0.6 em wide in every monospace face Chrome will pick here; the height comes
    # from the 1.2 line box.
    w = round(cast["cols"] * a.font_size * 0.6) + a.pad * 2
    h = round(cast["rows"] * a.font_size * 1.2) + a.pad * 2

    work = tempfile.mkdtemp(prefix="leetui-mp4-")
    listing = []

    for i, (lines, delay) in enumerate(screens):
        body = "".join("<div>%s</div>" % (l or "&nbsp;") for l in lines)
        page = PAGE.format(bg=cast.get("bg", "#0F1014"), fs=a.font_size,
                           pad=a.pad, palette=palette, body=body)
        src = os.path.join(work, "f%03d.html" % i)
        png = os.path.join(work, "f%03d.png" % i)
        open(src, "w").write(page)
        subprocess.run([a.chrome, "--headless", "--disable-gpu",
                        "--force-device-scale-factor=%d" % a.scale,
                        "--window-size=%d,%d" % (w, h),
                        "--virtual-time-budget=1500",
                        "--screenshot=" + png, "file://" + src],
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if not os.path.exists(png):
            sys.exit("Chrome produced no frame %d" % i)
        listing.append("file '%s'\nduration %.3f\n" % (png, delay / 1000.0))
        sys.stderr.write("\rframe %d/%d" % (i + 1, len(screens)))
        sys.stderr.flush()
    sys.stderr.write("\n")

    # The concat demuxer ignores the last entry's duration, so the final frame is
    # named twice: once with its hold, once to end on.
    listing.append("file '%s'\n" % png)
    lst = os.path.join(work, "frames.txt")
    open(lst, "w").write("".join(listing))

    subprocess.run(["ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", lst,
                    # yuv420p and even dimensions, or half the players on the
                    # internet show a grey rectangle.
                    "-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2",
                    "-c:v", "libx264", "-pix_fmt", "yuv420p",
                    "-movflags", "+faststart", "-r", "30", a.out],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    if not a.keep:
        shutil.rmtree(work, ignore_errors=True)
    else:
        print("frames in %s" % work, file=sys.stderr)

    size = os.path.getsize(a.out) // 1024
    print("%s  %d KB  %d frames  %.1fs  %dx%d"
          % (a.out, size, len(screens),
             sum(d for _, d in screens) / 1000.0, w * a.scale, h * a.scale),
          file=sys.stderr)


if __name__ == "__main__":
    main()
