#!/usr/bin/env bash
# Encode deliverables from the lossless mezzanine (out/mezz.mkv, RGB, 1920x1080, 60 fps CFR).
#   render/encode.sh [mezz] [web_crf]
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"
MEZZ="${1:-$HERE/out/mezz.mkv}"
WEB_CRF="${2:-27}"
OUT="$HERE/out"
COLOR=(-color_primaries bt709 -color_trc bt709 -colorspace bt709 -color_range tv)
TOYUV="scale=out_color_matrix=bt709:out_range=tv:flags=lanczos+accurate_rnd+full_chroma_int,format=yuv420p"

# Master: 1920x1080p60, high quality, no audio
ffmpeg -y -loglevel error -i "$MEZZ" -an \
  -vf "$TOYUV" -c:v libx264 -preset slow -crf 14 -profile:v high -level 4.2 \
  -g 120 -keyint_min 60 -r 60 -fps_mode cfr "${COLOR[@]}" -movflags +faststart \
  "$OUT/contextual-rag-showcase-master-1080p60.mp4"

# Web: 1280x720p60 H.264 CFR yuv420p, faststart, GOP 2 s, capped CRF to avoid bitrate spikes
ffmpeg -y -loglevel error -i "$MEZZ" -an \
  -vf "scale=1280:720:flags=lanczos+accurate_rnd+full_chroma_int,scale=out_color_matrix=bt709:out_range=tv,format=yuv420p" \
  -c:v libx264 -preset veryslow -crf "$WEB_CRF" -maxrate 2200k -bufsize 4400k -profile:v high -level 4.2 \
  -tune animation -g 120 -keyint_min 60 -sc_threshold 0 -r 60 -fps_mode cfr "${COLOR[@]}" -movflags +faststart \
  "$OUT/contextual-rag-showcase-web-720p60.mp4"

# Poster: first frame of the final web file, 1280x720
ffmpeg -y -loglevel error -i "$OUT/contextual-rag-showcase-web-720p60.mp4" -frames:v 1 -q:v 2 "$OUT/poster-first-frame.jpg"

ls -l "$OUT"/*.mp4 "$OUT"/*.jpg
