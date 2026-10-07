# Rendering the Contextual RAG Chat showcase

Everything lives in `videos/contextual-rag-hover-showcase/`. The app source is untouched; the
renderer drives the real production build of the app against a local demo API.

## Tools used (versions in this environment)

| Tool | Version | Role |
| --- | --- | --- |
| Node.js / npm | 22.22.0 / 10.9.4 | App build, mock API, renderer |
| Next.js (repo) | 16.0.5 | `next build` + `next start` of the real app |
| Playwright + Chromium | 1.56.1 / Chromium 141.0.7390.37 (`channel: 'chromium'`, new headless) | App automation + frame capture |
| ffmpeg / libx264 | 6.1.1 (Ubuntu build) | Mezzanine + deliverable encodes |
| Python 3 + reportlab, numpy, Pillow | 5.0.1 / 2.5.3 / 12.3.0 | PDF fixture, QA analysis, contact sheets |
| poppler `pdftoppm` | 24.02.0 | Raster of the PDF fixture (capture fallback) |

## 1. Install and build (repo root)

```bash
npm ci                     # frontend deps from package-lock.json
npx next build             # production build (fonts via next/font are fetched at build time)
cd videos/contextual-rag-hover-showcase
npm install                # only playwright@1.56.1 for the tooling (browser is pre-installed)
```

## 2. Fixtures

```bash
cd videos/contextual-rag-hover-showcase/fixtures
python3 make_fixtures.py   # Q3_Report.pdf + Monthly_Revenue.csv (asserts 454 total and +55%)
node make_chart.mjs        # chart_q3_revenue.png (reads the CSV; 1120x700, 2x density)
pdftoppm -r 220 -png -singlefile Q3_Report.pdf q3_report_page1   # raster for the PDF panel fallback
```

## 3. Services

```bash
render/services.sh start          # mock API on :5000 (mock/server.mjs) + `next start -p 3000`
render/services.sh restart-mock   # after changing fixtures
render/services.sh stop
```

The mock answers `GET /api/chat/suggestions`, `GET /api/documents`, `POST /api/upload` (multipart
field `document`), `POST /api/chat` (SSE), `GET /api/files/:name`, `GET /api/sessions`,
`GET /api/sessions/:id`, with CORS + OPTIONS. `POST /__mock/mode?gated=1` makes every upload
response and SSE event wait for `POST /__mock/release`, which the renderer calls on exact frames.

## 4. Render

```bash
# stills for review (also refreshes render/anchors.json, the measured element rects the camera uses)
node render/render.mjs --stills qa/stills --at 0,2,9.8,13.5,24.6,29.98
# preview a range
node render/render.mjs --out qa/preview.mkv --from 10 --to 18
# full render -> lossless RGB mezzanine (1800 frames @ 60 fps)
node render/render.mjs --out out/mezz.mkv
```

Run a stills pass first when the timeline or the app layout changed: camera targets come from
`render/anchors.json`, which every run rewrites with fresh measurements (the log prints
`anchors changed vs file: none` once converged).

### Re-capturing only a range

The renderer always replays the app from frame 0 (state must be identical), but captures only the
requested range. The final ending was re-captured and spliced like this:

```bash
node render/render.mjs --out qa/tail.mkv --from 28.0 --to 30
ffmpeg -i out/mezz.mkv -i qa/tail.mkv -filter_complex \
  "[0:v]trim=end_frame=1680,setpts=PTS-STARTPTS[a];[1:v]setpts=PTS-STARTPTS[b];[a][b]concat=n=2:v=1[v]" \
  -map "[v]" -c:v libx264rgb -qp 0 -preset ultrafast -r 60 out/mezz-final.mkv && mv out/mezz-final.mkv out/mezz.mkv
```

Timing: ~45 min for a full 1800-frame render in this 4-core, GPU-less container.

## 5. Encode deliverables

```bash
render/encode.sh out/mezz.mkv 20
```

Writes `out/contextual-rag-showcase-master-1080p60.mp4`, `out/contextual-rag-showcase-web-720p60.mp4`
and `out/poster-first-frame.jpg` (first frame of the web file).

## 6. QA

```bash
render/qa_final.sh        # metadata, faststart, motion/seam, 3x loop, 480/360 px sheets, poster check -> qa/final/
```

See `REPORT.md` for the checks that were run and their results.
