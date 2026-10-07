"""Contact sheet from a folder of PNG stills (or a list of files).

python3 render/sheet.py <dir> <out.png> [--width 640] [--cols 3]
Each tile is labelled with its timestamp (from the t<sec>.png filename).
"""
import glob
import os
import sys

from PIL import Image, ImageDraw


def main():
    src, out = sys.argv[1], sys.argv[2]
    width = int(sys.argv[sys.argv.index("--width") + 1]) if "--width" in sys.argv else 640
    cols = int(sys.argv[sys.argv.index("--cols") + 1]) if "--cols" in sys.argv else 3
    files = sorted(glob.glob(os.path.join(src, "*.png"))) if os.path.isdir(src) else src.split(",")
    height = round(width * 9 / 16)
    rows = (len(files) + cols - 1) // cols
    sheet = Image.new("RGB", (width * cols, height * rows), "white")
    for i, f in enumerate(files):
        im = Image.open(f).convert("RGB").resize((width, height), Image.LANCZOS)
        d = ImageDraw.Draw(im)
        label = os.path.basename(f)[1:-4] if os.path.basename(f).startswith("t") else os.path.basename(f)
        d.rectangle([0, 0, 8 + 7 * len(label), 20], fill="black")
        d.text((4, 4), label, fill="white")
        sheet.paste(im, ((i % cols) * width, (i // cols) * height))
    sheet.save(out)


if __name__ == "__main__":
    main()
