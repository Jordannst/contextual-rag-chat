// Renders the demo analysis chart (Q3 Revenue, IDR million) to chart_q3_revenue.png
// using headless Chromium. Values come straight from Monthly_Revenue.csv.
// Run: node make_chart.mjs
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const rows = fs.readFileSync(path.join(here, 'Monthly_Revenue.csv'), 'utf8')
  .trim().split('\n').slice(1).map((l) => {
    const [month, v] = l.split(',');
    return { month, millions: Number(v) / 1e6 };
  });
if (rows.map((r) => r.millions).join('/') !== '120/148/186') throw new Error('unexpected CSV data');

const W = 560, H = 350;
const pad = { l: 58, r: 20, t: 64, b: 50 };
const yMax = 200;
const plotW = W - pad.l - pad.r, plotH = H - pad.t - pad.b;
const y = (v) => pad.t + plotH * (1 - v / yMax);
const slot = plotW / rows.length, barW = 104;

let svg = '';
for (const t of [0, 50, 100, 150, 200]) {
  svg += `<line x1="${pad.l}" x2="${W - pad.r}" y1="${y(t)}" y2="${y(t)}" stroke="${t === 0 ? '#9CA3AF' : '#E5E7EB'}" stroke-width="1"/>`;
  svg += `<text x="${pad.l - 10}" y="${y(t) + 4}" text-anchor="end" class="tick">${t}</text>`;
}
rows.forEach((r, i) => {
  const cx = pad.l + slot * i + slot / 2;
  const last = i === rows.length - 1;
  const fill = last ? '#2563EB' : '#B8C4DA';
  svg += `<rect x="${cx - barW / 2}" y="${y(r.millions)}" width="${barW}" height="${y(0) - y(r.millions)}" rx="5" fill="${fill}"/>`;
  svg += `<rect x="${cx - barW / 2}" y="${y(0) - 5}" width="${barW}" height="5" fill="${fill}"/>`;
  svg += `<text x="${cx}" y="${y(r.millions) - 11}" text-anchor="middle" class="val${last ? ' hi' : ''}">${r.millions}</text>`;
  svg += `<text x="${cx}" y="${y(0) + 30}" text-anchor="middle" class="cat${last ? ' hi2' : ''}">${r.month}</text>`;
});
svg += `<text x="${pad.l - 2}" y="34" class="title">Q3 Revenue</text>`;
svg += `<text x="${W - pad.r}" y="33" text-anchor="end" class="unit">IDR million</text>`;
svg += `<text transform="translate(16 ${pad.t + plotH / 2}) rotate(-90)" text-anchor="middle" class="axis">Revenue (IDR million)</text>`;

const html = `<!doctype html><html><head><style>
html,body{margin:0;background:#fff}
svg{display:block;font-family:'Inter',sans-serif}
.tick{font-size:13px;fill:#6B7280}
.cat{font-size:19px;fill:#374151;font-weight:600}
.hi2{fill:#111827;font-weight:700}
.val{font-size:25px;fill:#4B5563;font-weight:700}
.hi{fill:#1D4ED8}
.title{font-size:22px;fill:#111827;font-weight:700}
.unit{font-size:15px;fill:#6B7280;font-weight:500}
.axis{font-size:11.5px;fill:#6B7280}
</style></head><body><svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">${svg}</svg></body></html>`;

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: W, height: H }, deviceScaleFactor: 2 });
await page.setContent(html);
await page.evaluate(() => document.fonts.ready);
await page.locator('svg').screenshot({ path: path.join(here, 'chart_q3_revenue.png') });
await browser.close();
console.log('wrote chart_q3_revenue.png', W * 2, 'x', H * 2);
