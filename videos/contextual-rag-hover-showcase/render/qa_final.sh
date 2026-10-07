#!/usr/bin/env bash
# QA on the final deliverables. Writes evidence into qa/final/.
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$HERE/out"; Q="$HERE/qa/final"
WEB="$OUT/contextual-rag-showcase-web-720p60.mp4"
MASTER="$OUT/contextual-rag-showcase-master-1080p60.mp4"
rm -rf "$Q"; mkdir -p "$Q"

# 1) media metadata (stream + format + exact frame count)
for f in "$MASTER" "$WEB"; do
  echo "== $(basename "$f")"
  ffprobe -v error -count_frames -show_entries \
    stream=codec_type,codec_name,profile,width,height,pix_fmt,r_frame_rate,avg_frame_rate,nb_read_frames,color_space,color_primaries,color_transfer \
    -show_entries format=duration,size,bit_rate,nb_streams -of default=nw=1 "$f"
done > "$Q/ffprobe.txt"

# 2) faststart: top-level atom order (moov must precede mdat)
python3 - "$MASTER" "$WEB" > "$Q/atoms.txt" <<'EOF'
import struct, sys
for path in sys.argv[1:]:
    atoms = []
    with open(path, 'rb') as f:
        while True:
            h = f.read(8)
            if len(h) < 8: break
            size, kind = struct.unpack('>I4s', h)
            if size == 1: size = struct.unpack('>Q', f.read(8))[0]; f.seek(size - 16, 1)
            else: f.seek(size - 8, 1)
            atoms.append(kind.decode('latin1'))
    print(path.split('/')[-1], ' '.join(atoms), '-> faststart' if atoms.index('moov') < atoms.index('mdat') else '-> NOT faststart')
EOF

# 3) motion / flash / seam analysis on the final web file and master
python3 "$HERE/render/qa_motion.py" "$WEB" "$Q/motion-web.json" > "$Q/motion-web.txt"
python3 "$HERE/render/qa_motion.py" "$MASTER" "$Q/motion-master.json" > "$Q/motion-master.txt"

# 4) seam evidence: last 3 frames + first 3 frames of the web file, and a 3x loop clip
ffmpeg -loglevel error -sseof -0.06 -i "$WEB" -vsync 0 "$Q/seam_end_%d.png"
ffmpeg -loglevel error -i "$WEB" -frames:v 3 "$Q/seam_start_%d.png"
printf "file '%s'\nfile '%s'\nfile '%s'\n" "$WEB" "$WEB" "$WEB" > "$Q/concat.txt"
ffmpeg -loglevel error -y -f concat -safe 0 -i "$Q/concat.txt" -c copy "$Q/loop3x-web.mp4"
python3 "$HERE/render/qa_motion.py" "$Q/loop3x-web.mp4" "$Q/motion-loop3x.json" > "$Q/motion-loop3x.txt"
# seam strip around both joins of the 3x clip (frames 1797..1802 and 3597..3602)
mkdir -p "$Q/seam_strip"
ffmpeg -loglevel error -i "$Q/loop3x-web.mp4" -vf "select='between(n\,1796\,1803)+between(n\,3596\,3603)',scale=480:-1" -vsync 0 "$Q/seam_strip/f%02d.png"
python3 "$HERE/render/sheet.py" "$Q/seam_strip" "$Q/seam-strip-3x.png" --width 480 --cols 4

# 5) project-card readability sheets at 480 and 360 px wide (focus moments)
mkdir -p "$Q/focus"
for t in 0.00 1.95 7.40 9.90 10.60 13.40 15.40 19.10 20.95 25.60 26.80; do
  ffmpeg -loglevel error -ss "$t" -i "$WEB" -frames:v 1 "$Q/focus/t$t.png"
done
python3 "$HERE/render/sheet.py" "$Q/focus" "$Q/card-480px.png" --width 480 --cols 3
python3 "$HERE/render/sheet.py" "$Q/focus" "$Q/card-360px.png" --width 360 --cols 3

# 6) poster vs first frame of the web file
python3 - "$OUT/poster-first-frame.jpg" "$Q/seam_start_1.png" > "$Q/poster-check.txt" <<'EOF'
import sys
import numpy as np
from PIL import Image
a = np.asarray(Image.open(sys.argv[1]).convert('RGB'), dtype=np.int16)
b = np.asarray(Image.open(sys.argv[2]).convert('RGB'), dtype=np.int16)
print('poster size', a.shape[1], 'x', a.shape[0], '| first frame', b.shape[1], 'x', b.shape[0])
print('mean abs diff (0-255):', round(float(np.abs(a - b).mean()), 3), '| max:', int(np.abs(a - b).max()))
EOF
ls -l "$OUT" | grep -v mezz
cat "$Q/atoms.txt" "$Q/poster-check.txt"
