"""Motion / flash / seam analysis of a rendered video.

python3 render/qa_motion.py <video.mp4> <report.json>

Decodes the video to 480x270 grayscale and computes per-frame mean luma and the mean absolute
difference (MAD) to the previous frame. Reports:
  * isolated duplicates: a frame identical to its predecessor while both neighbours move
    (the signature of 30 fps footage doubled to 60, or dropped/stuck frames);
  * longest run of motion (consecutive changed frames) and share of moving frames;
  * flashes / black frames: luma jumps or very dark frames;
  * seam: the wrap-around difference (last -> first) next to the typical in-motion difference.
"""
import json
import subprocess
import sys

import numpy as np

W, H = 480, 270


def frames(path):
    cmd = ["ffmpeg", "-loglevel", "error", "-i", path, "-vf", f"scale={W}:{H}:flags=area,format=gray", "-f", "rawvideo", "-"]
    p = subprocess.Popen(cmd, stdout=subprocess.PIPE)
    n = W * H
    while True:
        b = p.stdout.read(n)
        if len(b) < n:
            break
        yield np.frombuffer(b, np.uint8).astype(np.int16).reshape(H, W)


def main():
    path, out = sys.argv[1], sys.argv[2]
    fs = list(frames(path))
    n = len(fs)
    luma = np.array([f.mean() for f in fs])
    mad = np.array([0.0] + [np.abs(fs[i] - fs[i - 1]).mean() for i in range(1, n)])
    wrap = float(np.abs(fs[0] - fs[-1]).mean())
    eps = 0.02  # below this a frame counts as "unchanged" (encoder noise floor at this scale)
    moving = mad > eps
    iso_dupes = [i for i in range(2, n - 1) if mad[i] <= eps and mad[i - 1] > 0.15 and mad[i + 1] > 0.15]
    # alternating pattern check over sliding windows of 12 frames
    alt = 0
    for i in range(1, n - 12):
        w = moving[i:i + 12]
        if w[::2].all() and not w[1::2].any() or w[1::2].all() and not w[::2].any():
            alt += 1
    flashes = [i for i in range(1, n) if abs(luma[i] - luma[i - 1]) > 6.0]
    dark = [i for i in range(n) if luma[i] < 12]
    m = mad[1:][moving[1:]]
    report = {
        "file": path,
        "frames": n,
        "moving_frames": int(moving.sum()),
        "static_frames": int(n - moving.sum()),
        "isolated_duplicate_frames_in_motion": iso_dupes,
        "alternating_dup_windows": alt,
        "luma_min": round(float(luma.min()), 2), "luma_max": round(float(luma.max()), 2),
        "luma_jumps_gt6": flashes,
        "dark_frames_lt12": dark,
        "mad_in_motion_median": round(float(np.median(m)), 4) if len(m) else 0,
        "mad_in_motion_p95": round(float(np.percentile(m, 95)), 4) if len(m) else 0,
        "mad_max": round(float(mad.max()), 4), "mad_max_frame": int(mad.argmax()),
        "seam": {
            "last_to_first_mad": round(wrap, 4),
            "mad_last_frames": [round(float(x), 4) for x in mad[-4:]],
            "mad_first_frames": [round(float(x), 4) for x in mad[1:5]],
        },
        "top_mad_frames": [[int(i), round(float(mad[i]), 3)] for i in np.argsort(mad)[::-1][:12]],
    }
    with open(out, "w") as f:
        json.dump(report, f, indent=1)
    np.save(out.replace(".json", "-mad.npy"), mad)
    print(json.dumps({k: v for k, v in report.items() if k not in ("top_mad_frames",)}, indent=1)[:2500])
    print("top MAD frames:", report["top_mad_frames"])


if __name__ == "__main__":
    main()
