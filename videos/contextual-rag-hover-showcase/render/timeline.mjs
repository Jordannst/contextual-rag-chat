// Showcase timeline. All times in seconds on a 60 fps grid (frame i -> t = i / 60).
// The renderer evaluates everything here as a pure function of t plus anchors (element rects
// measured from the real app), so every render of the same inputs yields identical frames.

export const FPS = 60;
export const DURATION = 30;
export const FRAMES = FPS * DURATION; // 1800, last sampled frame is t = 29.9833

// ---- Geometry -------------------------------------------------------------------------------
// App viewport 1280x720 (above Tailwind's lg breakpoint), window = 28 px title bar + app.
// World scale 1.3 keeps the app iframe on integer screen pixels: x 128..1792, y 90..1026.
export const APP_W = 1280, APP_H = 720, TITLE_H = 28;
export const WIN_W = APP_W, WIN_H = APP_H + TITLE_H;
export const S0 = 1.3;
export const T0 = { x: 128, y: 90 - TITLE_H * S0 }; // 53.6
export const FRAME = { w: 1920, h: 1080 };
export const HOME = { cx: (FRAME.w / 2 - T0.x) / S0, cy: (FRAME.h / 2 - T0.y) / S0, z: 1 };

export function camToTransform(c) {
  const s = S0 * c.z;
  return { s, tx: FRAME.w / 2 - s * c.cx, ty: FRAME.h / 2 - s * c.cy };
}
export const toScreen = (tr, p) => ({ x: tr.tx + tr.s * p.x, y: tr.ty + tr.s * p.y });
export const toLocal = (tr, p) => ({ x: (p.x - tr.tx) / tr.s, y: (p.y - tr.ty) / tr.s });

// ---- Easing -------------------------------------------------------------------------------
export const clamp01 = (u) => Math.min(1, Math.max(0, u));
export const easeInOutCubic = (u) => (u < 0.5 ? 4 * u * u * u : 1 - Math.pow(-2 * u + 2, 3) / 2);
export const easeInOutQuint = (u) => (u < 0.5 ? 16 * u ** 5 : 1 - Math.pow(-2 * u + 2, 5) / 2);
export const easeOutCubic = (u) => 1 - Math.pow(1 - u, 3);
export const smootherstep = (u) => u * u * u * (u * (u * 6 - 15) + 10);
// Hand-like move: zero velocity at both ends, peak speed a little before the midpoint.
export const easeHand = (u) => smootherstep(Math.pow(clamp01(u), 0.82));
export const ramp = (t, a, b, ease = smootherstep) => ease(clamp01((t - a) / (b - a)));
export const window01 = (t, a, b, c, d) => ramp(t, a, b) * (1 - ramp(t, c, d));

// ---- Camera ---------------------------------------------------------------------------------
// Each segment eases from the previous resting state to `to` over [t0, t1]; gaps are holds.
// `to` can be { home: true }, { anchor, z } (center on rect) or { anchor, fit } (fit rect).
// Targets are resolved once per segment (never clamped per frame) and kept inside the window.
export const CAMERA = [
  { t0: 0.45, t1: 1.95, to: { anchor: 'uploadCardSelected', fit: 0.2, dy: 34, maxZ: 1.3 }, ease: easeInOutCubic },
  { t0: 3.55, t1: 4.75, to: { anchor: 'chatColumn', z: 1.28, dy: 40 }, ease: easeInOutCubic },
  { t0: 4.95, t1: 6.15, to: { anchor: 'inputZone', z: 1.6, dy: -125 }, ease: easeInOutCubic },
  { t0: 8.05, t1: 9.25, to: { anchor: 'qa1', fit: 0.1, maxZ: 2.55 }, ease: easeInOutCubic },
  { t0: 11.85, t1: 13.05, to: { anchor: 'splitView', fit: 0.04 }, ease: easeInOutCubic },
  { t0: 13.55, t1: 14.75, to: { anchor: 'pdfFocus', fit: 0.08, dx: 14 }, ease: easeInOutCubic },
  { t0: 16.05, t1: 17.05, to: { anchor: 'splitView', fit: 0.04 }, ease: easeInOutCubic },
  { t0: 17.55, t1: 18.55, to: { anchor: 'inputZone', z: 1.6, dy: -140 }, ease: easeInOutCubic },
  { t0: 23.9, t1: 25.0, to: { anchor: 'a2', fit: 0.07 }, ease: easeInOutCubic },
  { t0: 27.85, t1: 29.5, to: { home: true }, ease: easeInOutQuint },
];

// ---- Cursor ---------------------------------------------------------------------------------
// The cursor lives in window-local coordinates (it rides along with the camera). It starts and
// ends at the same off-frame point (bottom right), so the loop seam never shows a jump.
export const CURSOR_HOME_SCREEN = { x: 2010, y: 1150 };

export const MOVES = [
  { t0: 0.2, t1: 1.45, to: { anchor: 'uploadCard', fx: 0.56, fy: 0.42 }, bend: -0.16 },
  { t0: 1.95, t1: 2.5, to: { live: 'startBtn', fx: 0.5, fy: 0.5 }, bend: 0.12 },
  { t0: 4.05, t1: 4.7, to: { live: 'filterBtn' }, bend: -0.1 },
  { t0: 4.95, t1: 5.3, to: { live: 'labelPdf', fx: 0.3 }, bend: 0.12 },
  { t0: 5.5, t1: 5.88, to: { live: 'textarea', fx: 0.62 }, bend: -0.1 },
  { t0: 7.56, t1: 7.92, to: { live: 'sendBtn' }, bend: 0.12 },
  { t0: 8.2, t1: 9.0, to: { live: 'restSpot1' }, bend: -0.12 },
  { t0: 11.1, t1: 11.65, to: { live: 'citation', fx: 0.55 }, bend: 0.14 },
  { t0: 12.1, t1: 12.75, to: { live: 'restSpot2' }, bend: -0.1 },
  { t0: 16.65, t1: 17.25, to: { live: 'closeBtn' }, bend: 0.12 },
  { t0: 17.65, t1: 18.15, to: { live: 'filterBtn' }, bend: -0.12 },
  { t0: 18.35, t1: 18.65, to: { live: 'labelPdf', fx: 0.3 }, bend: 0.12 },
  { t0: 18.85, t1: 19.12, to: { live: 'labelCsv', fx: 0.3 }, bend: 0.1 },
  { t0: 19.35, t1: 19.72, to: { live: 'textarea', fx: 0.62 }, bend: -0.1 },
  { t0: 21.38, t1: 21.74, to: { live: 'sendBtn' }, bend: 0.12 },
  { t0: 22.02, t1: 22.82, to: { live: 'restSpot3' }, bend: -0.12 },
  { t0: 27.95, t1: 29.55, to: { home: true }, bend: 0.1 },
];

export const CLICKS = [2.62, 4.82, 5.4, 5.98, 8.0, 11.8, 17.38, 18.27, 18.76, 19.22, 19.82, 21.84];

// ---- App actions (beyond clicks) --------------------------------------------------------------
export const Q1 = 'What are the key Q3 results?';
export const Q2 = 'Chart monthly revenue for Q3.';

export function typingSchedule(text, t0, seed) {
  // Deterministic, slightly irregular rhythm (LCG jitter), longer after spaces/punctuation.
  let s = seed >>> 0;
  const rnd = () => ((s = (Math.imul(s, 1664525) + 1013904223) >>> 0) / 4294967296);
  const out = [];
  let t = t0;
  for (const ch of text) {
    out.push({ t, ch });
    t += 0.036 + rnd() * 0.022 + (ch === ' ' ? 0.022 : 0) + (/[?.]/.test(ch) ? 0.03 : 0);
  }
  return out;
}

export const ACTIONS = [
  { t: 1.28, type: 'dragover' },
  { t: 1.6, type: 'drop' },
  { t: 2.3, type: 'measure', names: ['uploadCardSelected'] },
  { t: 3.15, type: 'release', expect: 'progress1' },
  { t: 3.62, type: 'release', expect: 'chatview' },
  { t: 3.64, type: 'wait', expect: 'filterBtn' },
  { t: 3.66, type: 'measure', names: ['chatColumn', 'inputZone'] },
  ...typingSchedule(Q1, 6.08, 7).map((k) => ({ t: k.t, type: 'key', ch: k.ch })),
  { t: 8.18, type: 'release', expect: 'meta' },
  { t: 8.38, type: 'release', expect: 'text:Q3 revenue totaled' },
  { t: 8.62, type: 'release', expect: 'text:Rp454 million' },
  { t: 8.86, type: 'release', expect: 'text:September reached' },
  { t: 9.1, type: 'release', expect: 'text:Rp186 million' },
  { t: 9.34, type: 'release', expect: 'text:from July' },
  { t: 9.58, type: 'release', expect: 'citation' },
  { t: 9.72, type: 'release', expect: 'idle' },
  { t: 9.75, type: 'measure', names: ['qa1', 'a1', 'a1Strong', 'citation'] },
  { t: 13.2, type: 'measure', names: ['splitView', 'pdfFrame', 'pdfFocus'] },
  ...typingSchedule(Q2, 19.95, 11).map((k) => ({ t: k.t, type: 'key', ch: k.ch })),
  { t: 22.0, type: 'release', expect: 'meta' },
  { t: 22.2, type: 'release', expect: 'text:September led at' },
  { t: 22.44, type: 'release', expect: 'text:Rp186 million' },
  { t: 22.68, type: 'release', expect: 'text:Revenue rose' },
  { t: 22.92, type: 'release', expect: 'text:from July to September' },
  { t: 23.3, type: 'release', expect: 'chart' },
  { t: 23.42, type: 'release', expect: 'idle' },
  { t: 23.86, type: 'measure', names: ['a2'] },
  { t: 25.2, type: 'extract', anchor: 'a2' },
];

// ---- Presentation layers ----------------------------------------------------------------------
export const CAPTIONS = [
  { text: 'Ask your documents', t0: 0.95, t1: 7.35 },
  { text: 'Check the source', t0: 11.55, t1: 13.45 },
  { text: 'Turn data into insight', t0: 17.75, t1: 21.6 },
];

// Soft marker highlights. `rects` resolves from anchors; times stagger per item.
export const PDF_PT_W = 595.2756;
export const PDF_MARKS = [
  // PDF points (from top-left), matching fixtures/make_fixtures.py layout.
  { x: 56, y: 353, w: 483.3, h: 26, t: 14.55 }, // Q3 total row
  { x: 56, y: 327, w: 483.3, h: 26, t: 14.85 }, // September row
  { x: 52, y: 457, w: 186, h: 20, t: 15.15 },   // "September vs. July: +55%"
];
export const MARKS = {
  answer: { anchor: 'a1Strong', t: 10.05, step: 0.26, out: [11.0, 11.35] },
  citationRing: { anchor: 'citation', t0: 11.2, t1: 11.82 },
  pdf: { out: [16.0, 16.45] },
};

export const EXTRACT = { t: 25.2, lift: [25.25, 25.9], drop: [27.05, 27.55], fade: [27.55, 27.8] };
// Loop reset inside the window: the chat dims under a veil in the app's background colour,
// then the captured opening state fades in on top (no double exposure of the two states).
export const VEIL = [28.3, 28.75];
export const RESTORE = [28.7, 29.25];
