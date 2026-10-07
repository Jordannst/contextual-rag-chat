# Contextual RAG Chat — showcase video report

## Source

- Repo: `Jordannst/contextual-rag-chat`, branch `master`
- SHA used: **`39fd9b8c7fbccaeb8c8b6f03198b319b1269c27d`** (same as the brief's baseline; `origin/master` was checked at start and had not moved)
- The app source was **not modified**. Everything lives in `videos/contextual-rag-hover-showcase/`.

## Deliverables

| File | Spec | Size / bitrate |
| --- | --- | --- |
| `out/contextual-rag-showcase-master-1080p60.mp4` | H.264 High, 1920×1080, 60/1 CFR, yuv420p bt709, 1800 frames, 30.000 s, no audio, faststart | 24,023,573 B (22.9 MiB), 6.41 Mbps |
| `out/contextual-rag-showcase-web-720p60.mp4` | H.264 High, 1280×720, 60/1 CFR, yuv420p bt709, 1800 frames, 30.000 s, no audio, faststart, GOP 2 s, CRF 20 capped at 2.2 Mbps | 3,985,993 B (3.80 MiB), **1.06 Mbps** |
| `out/poster-first-frame.jpg` | 1280×720, frame 0 of the web file | 57,668 B |

Duration: **30.0 s exactly** (target kept; nothing needed the 32–34 s allowance).

Web budget: the first encode at CRF 27 was only 1.8 MB (0.47 Mbps). Text sharpness mattered more,
so the budget was spent on quality. Compared at the same settings: CRF 24 → 2.5 MB, CRF 22 → 3.1 MB,
CRF 20 → 3.9 MB (SSIM vs. mezzanine 0.9873 / 0.9879 / 0.9884; CRF 27 was 0.9858). CRF 20 lands inside
both the 3–5 MB and 0.8–1.3 Mbps targets.

## Final storyline (as rendered)

| Time | What happens | Camera / presentation |
| --- | --- | --- |
| 0.0–0.2 | Opening: whole macOS window on the wallpaper, upload page ready (thumbnail = poster) | Static, cursor off-frame |
| 0.2–1.6 | Cursor drags **Q3_Report.pdf** + **Monthly_Revenue.csv** in from bottom-right; dropzone switches to its real drag state; drop → real file list (2.8 KB / 0.1 KB) | Push-in to upload card; caption *Ask your documents* |
| 2.0–3.6 | Click **Mulai Proses & Chat (2 file)** → real progress `Mengunggah 1 dari 2 file… 50%` → chat view with both attachments + confirmation | Hold, then reframe on chat |
| 4.1–6.0 | Open filter, tick **Q3_Report.pdf** only, click into the input (closes popover) | Push to input + popover |
| 6.1–8.0 | Type *What are the key Q3 results?* (irregular deterministic rhythm), click send | — |
| 8.2–9.7 | Answer streams in word groups; citation + References appear | Follows the Q/A pair (z 2.55) |
| 10.0–11.3 | Soft amber marks on **Rp454 million**, **Rp186 million**, **55%**; reading hold | Static hold |
| 11.2–11.8 | Blue ring on the inline citation, cursor clicks `(Q3_Report.pdf)` | caption *Check the source* |
| 11.8–13.4 | Real PDF panel springs open beside the answer | Pan/fit: answer + source together |
| 13.6–16.0 | Push into the PDF: marks on the Q3 total row, September row, "+55%" | Reading hold ~1.5 s |
| 16.0–17.5 | Back to answer + source, cursor closes the panel (slides out) | |
| 17.7–19.8 | Filter again: untick PDF, tick **Monthly_Revenue.csv**, click input | caption *Turn data into insight* |
| 19.9–21.9 | Type *Chart monthly revenue for Q3.*, send | |
| 22.0–23.4 | Analysis text streams, then the chart PNG lands in the same bubble | |
| 23.9–25.0 | Frame the full result bubble (text + chart) | Fit to bubble |
| 25.2–27.6 | Result bubble (captured from the real UI) lifts slightly with a shadow, window dims/shrinks a touch, then settles back | Static |
| 27.9–29.5 | Pull back to the full window; chat dims under a veil while the captured opening state fades in; cursor returns off-frame | Home framing |
| 29.5–30.0 | Hold = frame 0 state → seamless loop | |

## What is real, what is mock, what is composited

- **Real UI capture:** every pixel inside the window comes from the production build of the repo
  (`next build` + `next start`), driven by real input: mouse moves/clicks via Playwright, keyboard
  typing, DOM drag & drop events carrying real `File` objects with the fixture bytes. Labels,
  fonts (bundled Inter, `next/font` mono fonts), progress, popover, citation, References,
  PDF panel, chart bubble are all the app's own.
- **Mock data / AI:** `mock/server.mjs` on :5000 returns scripted responses for the fictional
  Nara Studio data. The SSE timing is staged by the renderer (each event is released on a chosen
  frame) — it says nothing about real AI/backend speed. No Gemini/Cohere/DB/credentials were used.
  The chart is a mock PNG (`fixtures/make_chart.mjs`) delivered through the real `event: chart` contract.
- **Composited presentation layers (stage page, not product features):** wallpaper, macOS window
  chrome (title bar + traffic lights), camera moves, the drawn cursor + click ring, the drag chips,
  captions, amber/blue highlight marks, the lifted result card (a capture of the real bubble at
  that frame), the dim/veil and the end-of-loop restore of the opening state (a capture of frame 0).

## Capture-only adjustments (app source untouched)

1. **PDF fallback.** Chromium's built-in PDF viewer renders asynchronously on wall-clock time
   (seconds, with a thumbnail sidebar and colour-fringed text in headless), so it cannot be stepped
   frame by frame. During capture the panel iframe's request for `/api/files/Q3_Report.pdf` is
   answered with `stage/pdf-fallback.html`, which shows a 220 dpi raster of the **same** fixture PDF.
   The panel's real header, buttons and footer are kept; no viewer toolbar is imitated. The mock
   still serves the real PDF bytes at that URL for any other client.
2. **Deterministic time.** Playwright's fake clock drives timers/rAF/`performance.now`; every CSS
   transition/animation is pinned to the frame time each frame.
3. **framer-motion on its JS path.** `Element.prototype.animate` is hidden in the capture browser so
   framer-motion does not use WAAPI. Its WAAPI opacity animations could not be stepped externally
   (the PDF panel vanished in one frame on close); the JS path animates identically to the live app
   (verified: opacity 0.98→0.10 with the slide-out).
4. **Smooth scroll.** The chat container's `scroll-behavior: smooth` runs on wall-clock time; it is
   replaced during capture by a critically damped spring on the virtual clock with the same intent
   (stick to the newest message).
5. Browser flags: `--hide-scrollbars` (macOS-style overlay scrollbars), `--force-color-profile=srgb`,
   `--font-render-hinting=none`, site isolation off (keeps the PDF iframe in-process).

## QA performed on the final files

All evidence is in `qa/final/` (generated by `render/qa_final.sh`).

| Check | Method | Result |
| --- | --- | --- |
| Codec / format | `ffprobe -count_frames` (`qa/final/ffprobe.txt`) | Both: H.264 High, yuv420p, bt709, `r_frame_rate = avg_frame_rate = 60/1`, **1800 frames**, **30.000 s**, 1 stream (no audio) — **PASS** |
| Faststart | top-level atom walk (`qa/final/atoms.txt`) | `ftyp moov free mdat` for master and web — **PASS** |
| Real 60 fps motion | per-frame MAD analysis, 480×270 (`render/qa_motion.py`) | 0 windows of alternating duplicate frames (no 30→60 doubling); every moving segment changes on every frame except 5–6 isolated sub-pixel scroll-tail frames (see limitations). Captured at 60 fps on a virtual clock — not interpolated, not re-timed — **PASS** |
| Seam | lossless mezzanine frame 1799 → frame 0 | MAD **0.0007**/255, 0.000 % of pixels change > 8 levels (normal frame-to-frame drift is 0.02) — identical. Encoded files: 0.19 (master) / 0.34 (web) = keyframe-vs-P-frame codec difference only — **PASS** |
| Loop 3× | `qa/final/loop3x-web.mp4` + strip of frames around both joins (`seam-strip-3x.png`) | Joins visually identical, cursor off-frame at rest on both sides, camera/wallpaper at rest — **PASS** |
| Black frames / flashes | mean-luma scan of all 1800 frames | No frame below luma 12 (darkest 19.6). Luma jumps > 6/frame only at: upload click (card dims, native), PDF panel open/close (multi-frame slide), chart PNG arrival (native, one frame), last 2 frames of the dissolve — no white flash, no fade-to-black — **PASS (with noted chart pop)** |
| PDF panel close | frames 1046–1057 (`panel-close-frames.png`) | Slides + fades over ~10 frames (was a 1-frame vanish before the framer-motion fix) — **PASS** |
| Determinism | two independent renders, frames 1680–1697 | max pixel difference ≤ 2/255 — **PASS** |
| Card legibility 480 / 360 px | `card-480px.png`, `card-360px.png` (11 focus moments) | Recognisable at 360 px in focus moments: Rp454 million / Rp186 million / 55% (highlighted), citation `(Q3_Report.pdf)`, PDF rows 186,000,000 / 454,000,000 / +55%, chart 120 / 148 / 186 + months. The typed question is small at 360 px in the input shot but readable in the following answer framing. Opening/overview frames are not claimed legible. — **PASS for focus moments** |
| Framing | full 1 s contact sheet review | Traffic lights/title visible in overview; popover, input, citation, panel header & close button in frame when used; captions don't cover input/citation/numbers/PDF (PDF caption ends before the PDF push-in) — **PASS** |
| Data consistency | fixtures + asserts | 120 + 148 + 186 = 454; (186 − 120)/120 = 55 %. Same figures in PDF, CSV, answers, chart (IDR million), references — **PASS** |
| UI / network | renderer listeners (page errors, console errors, failed requests, HTTP ≥ 400) | **0 page errors, 0 network issues** in the final render; request bodies matched the contract: `selectedFiles: ["Q3_Report.pdf"]` then `["Monthly_Revenue.csv"]`, real `history`, `sessionId: 101` on the second question — **PASS** |
| Poster | poster vs frame 0 of web file | 1280×720, mean diff 1.6/255 (JPEG only), same state/framing — **PASS** |
| Web budget | size / bitrate | 3.80 MiB, 1.06 Mbps (target 3–5 MB, 0.8–1.3 Mbps) — **PASS**. Network playback: estimate only (see limitations). |
| Playback viewing | contact sheets + frame strips of the final encodes | Reviewed as stills/strips at many points; no interactive player is available in this container, so "watched 3×" is evidenced by the seam/motion analysis of the 3× file plus frame strips, not by eye in real time. |


## Known limitations

- **Chart appears in one frame** (frame 1398): the app has no chart reveal animation, so the PNG pops
  into the bubble. The camera then frames it; no fake grow animation was added.
- **Sub-pixel scroll tails:** at the end of a few chat scroll settles (≈9.8 s, 23.2 s, 23.9 s) motion
  drops below 1 CSS px/frame and Chromium snaps scroll offsets to whole pixels, so 5–6 single frames
  repeat their predecessor. This is not a 30→60 fps pattern (alternating-duplicate windows: 0).
- **Hover reveals:** when the cursor clicks the inline citation, the app's own copy/regenerate
  buttons fade in on the bubble (genuine hover behaviour; not used as a scene).
- **Reference videos:** only the two prompt files of `yihui-dev/awesome-opus5-5-videos` could be read
  (raw GitHub). The example videos (skillry.dev / x.com) are blocked by this environment's network
  proxy (HTTP 403), so their motion was not watched.
- **Network playback test:** not performed — the environment has no throttled real-network playback
  setup. Estimate only: 3.99 MB at ~1.06 Mbps needs ~1.1 s on a 30 Mbps link and ~6.4 s on 5 Mbps; with
  faststart, playback can begin once the first GOP (~2 s) is buffered. This is arithmetic, not a test.
- The final ending (28.0–30.0 s) was re-captured with an adjusted dissolve and spliced onto frames 0–1679 of the same render (identical up to ≤ 2/255, see Determinism).
- Render speed: ~45 min for 1800 frames in this container (software raster of the app's blurred
  background at zoomed scales).

## Assets and licences

- Wallpaper: original SVG gradients/waves drawn in `stage/stage.html` (no external asset).
- Cursor, drag chips, captions: drawn in the stage (Inter, already used by the app).
- Fonts: Inter (bundled with the app via `@fontsource-variable/inter`, OFL) and the app's
  `next/font` Geist Mono / JetBrains Mono (OFL), fetched at build time.
- Fixtures: generated by the scripts in `fixtures/` — fictional data, marked "Fictional demo data".
- No music, no audio track.
