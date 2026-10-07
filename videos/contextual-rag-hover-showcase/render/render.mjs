// Deterministic 60 fps renderer for the Contextual RAG Chat showcase.
//
// The real Next.js app runs inside the stage page (same origin, via route interception).
// For every frame i (t = i/60) the renderer:
//   1. performs the app actions due at t (real mouse clicks, key presses, drag/drop events,
//      releasing the next gated mock-API response) and waits until the app has reacted;
//   2. moves the real mouse to the presentation cursor, advances the app's fake clock to t
//      (timers, rAF, performance.now) and pins every CSS/WAAPI animation to t;
//   3. renders the stage (camera, wallpaper, cursor, captions, highlights) as a pure function of t;
//   4. captures the frame and pipes it to ffmpeg.
//
// Usage:
//   node render/render.mjs --out out/mezz.mkv                 full render (lossless x264rgb)
//   node render/render.mjs --stills qa/stills --at 1,5.5,9.8  PNG stills at given times
//   node render/render.mjs --out qa/prev.mkv --from 10 --to 18 preview a range
//   --every N   capture every Nth frame only (contact sheets);  --anchors file (default render/anchors.json)
import { chromium } from 'playwright';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import * as TL from './timeline.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.join(here, '..');
const FIX = path.join(ROOT, 'fixtures');
const STAGE = path.join(ROOT, 'stage');
const MOCK = 'http://localhost:5000';
const APP = 'http://localhost:3000';

// ---- CLI ----------------------------------------------------------------------------------
const args = Object.fromEntries(process.argv.slice(2).reduce((acc, a, i, arr) => {
  if (a.startsWith('--')) acc.push([a.slice(2), arr[i + 1] && !arr[i + 1].startsWith('--') ? arr[i + 1] : true]);
  return acc;
}, []));
const OUT = args.out || null;
const STILLS = args.stills || null;
const AT = args.at ? String(args.at).split(',').map(Number) : [];
const FROM = args.from !== undefined ? Number(args.from) : 0;
const TO = args.to !== undefined ? Number(args.to) : TL.DURATION;
const EVERY = Number(args.every || 1);
const ANCHORS_FILE = args.anchors || path.join(here, 'anchors.json');
const LAST_FRAME = Math.min(TL.FRAMES, Math.ceil(TO * TL.FPS)); // exclusive
const stillFrames = new Set(AT.map((t) => Math.round(t * TL.FPS)));
const wantFrame = (i) => stillFrames.has(i) || (OUT && i >= Math.round(FROM * TL.FPS) && i < LAST_FRAME && (i - Math.round(FROM * TL.FPS)) % EVERY === 0);
const lastNeeded = Math.max(OUT ? LAST_FRAME - 1 : 0, ...[...stillFrames]);

const fileAnchors = fs.existsSync(ANCHORS_FILE) ? JSON.parse(fs.readFileSync(ANCHORS_FILE, 'utf8')) : {};
const anchors = { ...fileAnchors }; // camera reads these; live measurements refresh them
const measured = {};
// persist measured anchors (early camera targets need them before they exist in a run)
const saveAnchors = () => fs.writeFileSync(ANCHORS_FILE, JSON.stringify({ ...fileAnchors, ...measured }, null, 1));
const log = (...a) => console.log(`[render]`, ...a);

// ---- Browser & stage ------------------------------------------------------------------------
await fetch(`${MOCK}/__mock/reset`, { method: 'POST' });
await fetch(`${MOCK}/__mock/mode?gated=1`, { method: 'POST' });

const browser = await chromium.launch({
  channel: 'chromium',
  args: ['--hide-scrollbars', '--force-color-profile=srgb', '--font-render-hinting=none',
    '--disable-features=IsolateOrigins,site-per-process', '--disable-site-isolation-trials'],
});
const context = await browser.newContext({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 1 });
const netIssues = [];
context.on('requestfailed', (r) => netIssues.push(`failed ${r.method()} ${r.url()} ${r.failure()?.errorText}`));
context.on('response', (r) => { if (r.status() >= 400) netIssues.push(`HTTP ${r.status()} ${r.url()}`); });
await context.route(`${APP}/__stage`, (r) => r.fulfill({ contentType: 'text/html', body: fs.readFileSync(path.join(STAGE, 'stage.html')) }));
// Capture-only PDF fallback (see stage/pdf-fallback.html). Only the iframe *document* request is
// answered here; the mock still serves the real PDF bytes for any other client.
await context.route(`${MOCK}/api/files/Q3_Report.pdf`, (r) => r.request().resourceType() === 'document'
  ? r.fulfill({ contentType: 'text/html', body: fs.readFileSync(path.join(STAGE, 'pdf-fallback.html')) })
  : r.continue());
await context.route(`${MOCK}/api/files/q3_report_page1.png`, (r) => r.fulfill({ contentType: 'image/png', body: fs.readFileSync(path.join(FIX, 'q3_report_page1.png')) }));

const T_BASE = new Date('2026-10-07T10:00:00+07:00').getTime();
await context.clock.install({ time: T_BASE });
const page = await context.newPage();
const pageErrors = [];
page.on('pageerror', (e) => pageErrors.push(e.message));
page.on('console', (m) => { if (m.type() === 'error') pageErrors.push('console: ' + m.text()); });
await page.goto(`${APP}/__stage`);
const appFrame = await (await page.waitForSelector('#app')).contentFrame();
await appFrame.waitForSelector('text=Upload Dokumen');
await page.waitForLoadState('networkidle');
await context.clock.pauseAt(T_BASE + 5000);
await page.evaluate(() => document.fonts.ready);
await appFrame.evaluate(() => document.fonts.ready);
const cdp = await context.newCDPSession(page);

// ---- Capture helpers injected into the app frame -----------------------------------------------
await appFrame.evaluate(({ files }) => {
  const starts = new WeakMap();
  const nativeTop = Object.getOwnPropertyDescriptor(Element.prototype, 'scrollTop');
  const cap = (window.__cap = {});
  // Deterministic stand-in for `scroll-behavior: smooth` (which runs on wall-clock time):
  // the chat container's programmatic scrolls become targets that a critically damped
  // spring follows on the virtual clock. Same intent as the app (stick to the bottom).
  Element.prototype.scrollIntoView = function () {
    const c = this.closest('[style*="scroll-behavior"]');
    if (c) c.__want = Infinity;
  };
  function patch(el) {
    if (el.__patched) return;
    el.__patched = true;
    el.style.scrollBehavior = 'auto';
    el.__pos = nativeTop.get.call(el); el.__vel = 0; el.__want = Infinity;
    Object.defineProperty(el, 'scrollTop', {
      configurable: true,
      get() { return nativeTop.get.call(this); },
      set(v) { const max = this.scrollHeight - this.clientHeight; this.__want = v >= max - 2 ? Infinity : v; },
    });
  }
  cap.step = (vt, dt) => {
    const el = document.querySelector('[style*="scroll-behavior"]');
    if (el) {
      patch(el);
      const max = el.scrollHeight - el.clientHeight;
      const target = Math.min(el.__want, max);
      const w = 2 * Math.PI * 1.7; // ~0.35 s settle
      const n = 4, h = dt / 1000 / n;
      for (let k = 0; k < n; k++) {
        const acc = w * w * (target - el.__pos) - 2 * w * el.__vel;
        el.__vel += acc * h; el.__pos += el.__vel * h;
      }
      if (Math.abs(target - el.__pos) < 0.3 && Math.abs(el.__vel) < 2) { el.__pos = target; el.__vel = 0; }
      el.__pos = Math.max(0, Math.min(max, el.__pos));
      nativeTop.set.call(el, el.__pos);
    }
    // Pin every CSS transition/animation and WAAPI animation to the virtual clock.
    for (const a of document.getAnimations()) {
      if (typeof CSSAnimation !== 'undefined' && a instanceof CSSAnimation && a.effect && a.effect.target) {
        const names = getComputedStyle(a.effect.target).animationName.split(',').map((x) => x.trim());
        if (!names.includes(a.animationName)) { a.cancel(); continue; }
      }
      let s = starts.get(a);
      if (s === undefined) { s = vt; starts.set(a, s); }
      if (a.playState === 'running') a.pause();
      const ct = vt - s;
      const end = a.effect ? a.effect.getComputedTiming().endTime : Infinity;
      if (Number.isFinite(end) && ct >= end) { try { a.finish(); } catch {} } else a.currentTime = ct;
    }
  };
  // Real files for drag & drop (same bytes as fixtures/)
  cap.dt = () => {
    const dt = new DataTransfer();
    for (const f of files) dt.items.add(new File([Uint8Array.from(atob(f.b64), (c) => c.charCodeAt(0))], f.name, { type: f.type }));
    return dt;
  };
  const card = () => document.querySelector('.border-dashed');
  cap.drag = (type) => {
    const ev = new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: cap.dt() });
    card().dispatchEvent(ev);
  };
  // ---- element finders (app coordinates)
  const vis = (el) => { if (!el) return null; const r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0 ? el : null; };
  const all = (sel) => [...document.querySelectorAll(sel)].filter(vis);
  const bubble = (text) => {
    // deepest element whose text contains `text`, then its chat bubble
    const hits = [...document.querySelectorAll('p, strong, span, div')].filter((e) => e.textContent.includes(text));
    for (const h of hits.reverse()) { const b = h.closest('[class*="rounded-3xl"]'); if (b) return b; }
    return null;
  };
  const R = (el) => { const r = el.getBoundingClientRect(); return { x: r.left, y: r.top, w: r.width, h: r.height }; };
  const union = (...rs) => { rs = rs.filter(Boolean); const x = Math.min(...rs.map((r) => r.x)), y = Math.min(...rs.map((r) => r.y)); return { x, y, w: Math.max(...rs.map((r) => r.x + r.w)) - x, h: Math.max(...rs.map((r) => r.y + r.h)) - y }; };
  const byText = (sel, text) => all(sel).find((e) => e.textContent.includes(text));
  const pdfFrame = () => all('iframe[title="Q3_Report.pdf"]')[0];
  const PT = 595.2756;
  const pdfRect = (fr, x, y, w, h) => { const k = (fr.w - 32) / PT; return { x: fr.x + 16 + x * k, y: fr.y + 16 + y * k, w: w * k, h: h * k }; };
  const F = {
    uploadCard: () => R(card()),
    uploadCardSelected: () => R(card()),
    startBtn: () => R(byText('button', 'Mulai Proses')),
    filterBtn: () => R(all('button[title="Filter by document"], button[title^="Filtered"]')[0]),
    labelPdf: () => R(byText('label', 'Q3_Report.pdf')),
    labelCsv: () => R(byText('label', 'Monthly_Revenue.csv')),
    textarea: () => R(document.querySelector('textarea')),
    sendBtn: () => R(document.querySelector('form button[type="submit"]')),
    inputZone: () => R(document.querySelector('form')),
    chatColumn: () => union(R(document.querySelector('[style*="scroll-behavior"] > div')), R(document.querySelector('form'))),
    citation: () => R(all('button[title="View Q3_Report.pdf"]')[0].closest('span')),
    closeBtn: () => R(all('button[aria-label="Close panel"]')[0]),
    a1: () => R(bubble('Rp454 million')),
    qa1: () => union(R(bubble('What are the key Q3 results?')), R(bubble('Rp454 million'))),
    a1Strong: () => [...bubble('Rp454 million').querySelectorAll('strong')].map(R),
    pdfFrame: () => R(pdfFrame()),
    pdfFocus: () => pdfRect(R(pdfFrame()), 40, 136, 515, 350),
    splitView: () => { const fr = R(pdfFrame()); return union(R(bubble('Rp454 million')), pdfRect(fr, 30, 30, 535, 455)); },
    qa2text: () => union(R(bubble('Chart monthly revenue for Q3.')), R(bubble('September led at'))),
    a2: () => R(bubble('September led at')),
    restSpot1: () => { const f = R(document.querySelector('form')); return { x: f.x + f.w + 70, y: f.y - 150, w: 0, h: 0 }; },
    restSpot3: () => { const f = R(document.querySelector('form')); return { x: f.x + f.w + 70, y: f.y - 150, w: 0, h: 0 }; },
    restSpot2: () => { const fr = R(pdfFrame()); const f = R(document.querySelector('form')); return { x: fr.x - 70, y: f.y - 70, w: 0, h: 0 }; },
  };
  cap.measure = (name) => { try { return F[name](); } catch (e) { return { error: String(e) }; } };
  // ---- predicates for gated releases
  const aiText = () => { const b = [...document.querySelectorAll('[class*="rounded-3xl"]')].pop(); return b ? b.textContent : ''; };
  cap.check = (what) => {
    if (what === 'progress1') return document.body.textContent.includes('Mengunggah 1 dari 2');
    if (what === 'chatview') return document.body.textContent.includes('Uploaded Document');
    if (what === 'filterBtn') return !!all('button[title="Filter by document"], button[title^="Filtered"]').length;
    if (what === 'meta') return true;
    if (what === 'citation') return !!all('button[title="View Q3_Report.pdf"]').length && document.body.textContent.includes('References');
    if (what === 'chart') { const i = document.querySelector('img[alt="Data Visualization"]'); return !!(i && i.complete && i.naturalWidth); }
    if (what === 'idle') { const ta = document.querySelector('textarea'); return !!ta && !ta.disabled; }
    if (what.startsWith('text:')) return aiText().includes(what.slice(5).replace(/\*\*/g, ''));
    return false;
  };
}, {
  files: ['Q3_Report.pdf', 'Monthly_Revenue.csv'].map((n) => ({
    name: n, type: n.endsWith('.pdf') ? 'application/pdf' : 'text/csv',
    b64: fs.readFileSync(path.join(FIX, n)).toString('base64'),
  })),
});

const waitFor = async (what, ms = 8000) => {
  const t0 = Date.now();
  while (Date.now() - t0 < ms) {
    if (await appFrame.evaluate((w) => window.__cap.check(w), what)) return true;
    await new Promise((r) => setTimeout(r, 15));
  }
  throw new Error(`timeout waiting for ${what}`);
};
const toLocalRect = (r) => (r && !r.error ? (Array.isArray(r) ? r.map(toLocalRect) : { ...r, y: r.y + TL.TITLE_H }) : null);
const measure = async (name) => toLocalRect(await appFrame.evaluate((n) => window.__cap.measure(n), name));

// ---- Virtual clock ---------------------------------------------------------------------------------
let vt = -2000; // ms relative to frame 0, integer ticks (starts with a 2 s settle pre-roll)
async function advanceTo(ms, dtForScroll) {
  const target = Math.round(ms);
  const d = target - vt;
  await appFrame.evaluate(async ({ d, vt, dt }) => {
    if (d > 0) await globalThis.__pwClock.controller.runFor(d);
    window.__cap.step(vt, dt);
  }, { d, vt: target, dt: dtForScroll });
  vt = target;
}
// settle the opening state (load animations finish) on the virtual clock
for (let k = 1; k <= 120; k++) await advanceTo(-2000 + k * (2000 / 120), 1000 / 60);
anchors.uploadCard = await measure('uploadCard');

// ---- Camera ------------------------------------------------------------------------------------------
function resolveCam(to) {
  if (to.home) return { ...TL.HOME };
  const r = anchors[to.anchor];
  if (!r) return null;
  let z = to.z;
  if (to.fit !== undefined) {
    z = Math.min((TL.FRAME.w * (1 - to.fit)) / (r.w * TL.S0), (TL.FRAME.h * (1 - to.fit)) / (r.h * TL.S0));
    if (to.maxZ) z = Math.min(z, to.maxZ);
  }
  z = Math.max(1, z);
  let cx = r.x + r.w / 2 + (to.dx || 0), cy = r.y + r.h / 2 + (to.dy || 0);
  const s = TL.S0 * z, hw = TL.FRAME.w / 2 / s, hh = TL.FRAME.h / 2 / s;
  // keep the view inside the window (resolved once per target, never per frame)
  const fit = (c, half, size) => (2 * half >= size ? size / 2 : Math.min(size - half, Math.max(half, c)));
  if (z > 1.0001) { cx = fit(cx, hw, TL.WIN_W); cy = fit(cy, hh, TL.WIN_H); } else { cx = TL.HOME.cx; cy = TL.HOME.cy; }
  return { cx, cy, z };
}
// targets resolve when their segment starts (live measurements from earlier in this run win)
const camTargets = [];
function cameraAt(t) {
  let cur = { ...TL.HOME };
  for (let k = 0; k < TL.CAMERA.length; k++) {
    const seg = TL.CAMERA[k];
    if (t > seg.t0 && camTargets[k] === undefined) camTargets[k] = resolveCam(seg.to);
    const to = camTargets[k] || cur;
    if (t >= seg.t1) { cur = to; continue; }
    if (t > seg.t0) {
      const u = seg.ease((t - seg.t0) / (seg.t1 - seg.t0));
      return { cx: cur.cx + (to.cx - cur.cx) * u, cy: cur.cy + (to.cy - cur.cy) * u, z: Math.exp(Math.log(cur.z) + (Math.log(to.z) - Math.log(cur.z)) * u) };
    }
    break;
  }
  return cur;
}

// ---- Cursor -------------------------------------------------------------------------------------------
const HOME_TR = TL.camToTransform(TL.HOME);
const cursorHome = TL.toLocal(HOME_TR, TL.CURSOR_HOME_SCREEN);
const moveTargets = [];
async function resolveMoveTarget(k) {
  const to = TL.MOVES[k].to;
  if (to.home) return cursorHome;
  let r = to.live ? await measure(to.live) : anchors[to.anchor];
  for (let k = 0; !r && to.live && k < 60; k++) { await new Promise((res) => setTimeout(res, 20)); r = await measure(to.live); }
  if (!r) {
    await page.screenshot({ path: path.join(ROOT, 'qa', 'error.png') });
    throw new Error(`no rect for move ${k} (${to.live || to.anchor}): ${JSON.stringify(await appFrame.evaluate((n) => window.__cap.measure(n), to.live || to.anchor))}`);
  }
  return { x: r.x + r.w * (to.fx ?? 0.5), y: r.y + r.h * (to.fy ?? 0.5) };
}
function cursorAt(t) {
  let p = cursorHome;
  for (let k = 0; k < TL.MOVES.length; k++) {
    const m = TL.MOVES[k], to = moveTargets[k];
    if (!to) break;
    if (t >= m.t1) { p = to; continue; }
    if (t > m.t0) {
      const u = TL.easeHand((t - m.t0) / (m.t1 - m.t0));
      const dx = to.x - p.x, dy = to.y - p.y;
      const c = { x: (p.x + to.x) / 2 - dy * m.bend, y: (p.y + to.y) / 2 + dx * m.bend };
      const a = (1 - u) * (1 - u), b = 2 * u * (1 - u), d = u * u;
      return { x: a * p.x + b * c.x + d * to.x, y: a * p.y + b * c.y + d * to.y };
    }
    break;
  }
  return p;
}

// ---- Stage state ----------------------------------------------------------------------------------------
let extractRect = null;
function stageState(t, cam, cur) {
  let tr = TL.camToTransform(cam);
  // emphasis: lift the extracted result card, calm the window
  const lift = TL.ramp(t, ...TL.EXTRACT.lift) * (1 - TL.ramp(t, ...TL.EXTRACT.drop));
  let extract = null;
  if (extractRect && t >= TL.EXTRACT.t && t < TL.EXTRACT.fade[1]) {
    const cx = extractRect.x + extractRect.w / 2, cy = extractRect.y + extractRect.h / 2;
    const m = 1 - 0.025 * lift;
    tr = { s: tr.s * m, tx: cx + (tr.tx - cx) * m, ty: cy + (tr.ty - cy) * m };
    extract = { visible: true, ...extractRect, radius: 24 * extractRect.s, lift, scale: 1 + 0.03 * lift, dy: -12 * lift, opacity: 1 - TL.ramp(t, ...TL.EXTRACT.fade) };
  }
  const sp = TL.toScreen(tr, cur);
  // click feedback
  let press = 1, ring = null;
  for (const c of TL.CLICKS) {
    if (t >= c && t < c + 0.26) press = Math.min(press, t < c + 0.08 ? 1 - 0.12 * TL.ramp(t, c, c + 0.08) : 0.88 + 0.12 * TL.ramp(t, c + 0.08, c + 0.26));
    if (t >= c && t < c + 0.45) {
      const u = (t - c) / 0.45;
      ring = { x: sp.x, y: sp.y, r: 5 + 19 * TL.easeOutCubic(u), opacity: 0.7 * Math.pow(1 - u, 1.6) };
    }
  }
  const cursorScale = 1.32 * Math.pow(cam.z, 0.28) * press;
  // dragged file chips (screen space), lagging the cursor slightly
  let chips = null;
  if (t < 1.95) {
    const lag = TL.toScreen(tr, cursorAt(Math.max(0, t - 0.05)));
    const drop = TL.ramp(t, 1.6, 1.88, TL.easeInOutCubic);
    chips = { x: lag.x + 16 - 30 * drop, y: lag.y + 22 - 10 * drop, scale: 1 - 0.45 * drop, fan: 1 - TL.ramp(t, 1.2, 1.6), opacity: 1 - drop };
  }
  // captions
  let caption = null;
  for (const c of TL.CAPTIONS) {
    const o = TL.window01(t, c.t0, c.t0 + 0.4, c.t1 - 0.35, c.t1);
    if (o > 0) caption = { text: c.text, opacity: o, dy: 10 * (1 - TL.ramp(t, c.t0, c.t0 + 0.5)) };
  }
  // highlights (window-local)
  const marks = [];
  const A = TL.MARKS.answer;
  (anchors[A.anchor] || []).forEach((r, i) => {
    const t0 = A.t + i * A.step;
    const o = TL.window01(t, t0, t0 + 0.12, ...A.out);
    if (o > 0) marks.push({ x: r.x - 3, y: r.y - 1, w: r.w + 6, h: r.h + 2, opacity: o, grow: TL.ramp(t, t0, t0 + 0.3, TL.easeOutCubic) });
  });
  const cr = anchors[TL.MARKS.citationRing.anchor];
  if (cr) {
    const o = TL.window01(t, TL.MARKS.citationRing.t0, TL.MARKS.citationRing.t0 + 0.2, TL.MARKS.citationRing.t1 - 0.15, TL.MARKS.citationRing.t1);
    if (o > 0) marks.push({ kind: 'ring', x: cr.x - 3, y: cr.y - 3, w: cr.w + 6, h: cr.h + 6, opacity: o });
  }
  const fr = anchors.pdfFrame;
  if (fr) {
    const k = (fr.w - 32) / TL.PDF_PT_W;
    for (const m of TL.PDF_MARKS) {
      const o = TL.window01(t, m.t, m.t + 0.12, ...TL.MARKS.pdf.out);
      if (o > 0) marks.push({ x: fr.x + 16 + m.x * k, y: fr.y + 16 + m.y * k, w: m.w * k, h: m.h * k, opacity: o, grow: TL.ramp(t, m.t, m.t + 0.32, TL.easeOutCubic) });
    }
  }
  return {
    t, cam: tr, dim: 1 - 0.26 * lift, restore: TL.ramp(t, ...TL.RESTORE),
    cursor: { x: sp.x, y: sp.y, scale: cursorScale, opacity: 1 }, ring, chips, caption, marks, extract,
    mouse: sp,
  };
}

// ---- Output ----------------------------------------------------------------------------------------------
let ff = null;
if (OUT) {
  fs.mkdirSync(path.dirname(path.resolve(OUT)), { recursive: true });
  ff = spawn('ffmpeg', ['-y', '-loglevel', 'error', '-f', 'image2pipe', '-framerate', String(TL.FPS), '-c:v', 'png', '-i', '-',
    '-c:v', 'libx264rgb', '-qp', '0', '-preset', 'ultrafast', '-r', String(TL.FPS), OUT], { stdio: ['pipe', 'inherit', 'inherit'] });
}
if (STILLS) fs.mkdirSync(STILLS, { recursive: true });
const shot = async (clip) => Buffer.from((await cdp.send('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false, optimizeForSpeed: true, ...(clip ? { clip: { ...clip, scale: 1 } } : {}) })).data, 'base64');
const write = (buf) => new Promise((res) => (ff.stdin.write(buf) ? res() : ff.stdin.once('drain', res)));

// ---- Main loop ---------------------------------------------------------------------------------------------
const actions = [...TL.ACTIONS].sort((a, b) => a.t - b.t);
let ai = 0, moveIdx = 0, clickIdx = 0;
let mouseDown = false, lastMouse = null;
const started = Date.now();
for (let i = 0; i <= lastNeeded; i++) {
  const t = i / TL.FPS;
  // resolve cursor move targets when their move starts (live measurement of the real element)
  // finish a pending click first (mouse up fires 'click'), so moves starting this frame see its result
  if (mouseDown && t >= mouseDown + 0.08) { await page.mouse.up(); mouseDown = false; clickIdx++; await new Promise((r) => setTimeout(r, 40)); }
  // (live targets are re-measured every frame until the move lands, so a layout that is still
  //  settling — e.g. the chat column widening after the PDF panel closes — is followed smoothly)
  while (moveIdx < TL.MOVES.length && t >= TL.MOVES[moveIdx].t0) { moveTargets[moveIdx] = await resolveMoveTarget(moveIdx); moveIdx++; }
  for (let k = 0; k < moveIdx; k++) {
    const m = TL.MOVES[k];
    if (m.to.live && t > m.t0 && t - 1 / TL.FPS < m.t1) moveTargets[k] = await resolveMoveTarget(k);
  }
  const cam = cameraAt(t);
  const cur = cursorAt(t);
  let st = stageState(t, cam, cur);
  // real mouse follows the presentation cursor
  const mx = Math.min(1919, Math.max(0, st.mouse.x)), my = Math.min(1079, Math.max(0, st.mouse.y));
  if (!lastMouse || Math.abs(lastMouse.x - mx) > 0.01 || Math.abs(lastMouse.y - my) > 0.01) { await page.mouse.move(mx, my); lastMouse = { x: mx, y: my }; }
  // clicks: press on the click frame, release 0.08 s later
  while (clickIdx < TL.CLICKS.length && t >= TL.CLICKS[clickIdx] && !mouseDown) { await page.mouse.down(); mouseDown = TL.CLICKS[clickIdx]; }
  // scheduled app actions
  while (ai < actions.length && t >= actions[ai].t) {
    const a = actions[ai++];
    if (a.type === 'key') await page.keyboard.type(a.ch);
    else if (a.type === 'dragover') { await appFrame.evaluate(() => { window.__cap.drag('dragenter'); window.__cap.drag('dragover'); }); }
    else if (a.type === 'drop') { await appFrame.evaluate(() => window.__cap.drag('drop')); await appFrame.waitForSelector('text=Mulai Proses'); }
    else if (a.type === 'wait') await waitFor(a.expect);
    else if (a.type === 'release') {
      const t0 = Date.now();
      while (!(await (await fetch(`${MOCK}/__mock/pending`)).json()).pending.length) {
        if (Date.now() - t0 > 8000) throw new Error(`nothing pending at t=${t} for ${a.expect}`);
        await new Promise((r) => setTimeout(r, 10));
      }
      await fetch(`${MOCK}/__mock/release`, { method: 'POST' });
      await waitFor(a.expect);
      await new Promise((r) => setTimeout(r, 25));
    } else if (a.type === 'measure') {
      for (const n of a.names) { measured[n] = await measure(n); if (measured[n]) anchors[n] = measured[n]; }
      saveAnchors();
    } else if (a.type === 'extract') {
      const r = await measure(a.anchor); // live, at the exact capture frame
      if (!r) throw new Error(`extract: anchor ${a.anchor} missing`);
      const tr = TL.camToTransform(cam);
      const p = TL.toScreen(tr, r);
      const x = Math.floor(p.x), y = Math.floor(p.y);
      extractRect = { x, y, w: Math.ceil(p.x + r.w * tr.s) - x, h: Math.ceil(p.y + r.h * tr.s) - y, s: tr.s };
      st = stageState(t, cam, cur);
      await page.evaluate((s) => window.stage.render({ ...s, extract: null }), st);
      const png = await shot({ x: extractRect.x, y: extractRect.y, width: extractRect.w, height: extractRect.h });
      await page.evaluate((src) => window.stage.setExtract(src), 'data:image/png;base64,' + png.toString('base64'));
    }
    await new Promise((r) => setTimeout(r, 5));
  }
  await advanceTo(t * 1000, 1000 / TL.FPS);
  st = stageState(t, cam, cur);
  await page.evaluate((s) => window.stage.render(s), st);
  if (i === 0) {
    // the opening state of the app, used to restore it seamlessly at the end of the loop
    const png = await shot({ x: 128, y: 90, width: 1664, height: 936 });
    await page.evaluate((src) => window.stage.setRestore(src), 'data:image/png;base64,' + png.toString('base64'));
  }
  if (wantFrame(i)) {
    const buf = await shot();
    if (ff) await write(buf);
    if (STILLS && stillFrames.has(i)) fs.writeFileSync(path.join(STILLS, `t${t.toFixed(2).padStart(5, '0')}.png`), buf);
  }
  if (i % 120 === 0) log(`frame ${i}/${lastNeeded} t=${t.toFixed(2)} ${((Date.now() - started) / 1000).toFixed(0)}s`);
}
if (ff) { ff.stdin.end(); await new Promise((r) => ff.on('close', r)); }

saveAnchors();
const changed = Object.keys(measured).filter((k) => JSON.stringify(measured[k]) !== JSON.stringify(fileAnchors[k]));
log('anchors changed vs file:', changed.length ? changed.join(', ') : 'none');
const mockLog = await (await fetch(`${MOCK}/__mock/log`)).json();
fs.writeFileSync(path.join(path.dirname(ANCHORS_FILE), 'last-run-network.json'), JSON.stringify({ netIssues, pageErrors, mock: mockLog }, null, 1));
log('page errors:', pageErrors.length, 'network issues:', netIssues.length);
log(`done in ${((Date.now() - started) / 1000).toFixed(0)}s`);
await browser.close();
