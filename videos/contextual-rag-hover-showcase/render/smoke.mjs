// Real-time smoke test of the full flow against the mock API (no fake clock).
// node render/smoke.mjs [outdir] [--channel=chromium]
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const FIX = path.join(here, '..', 'fixtures');
const out = process.argv[2] || '/tmp/smoke';
const channel = (process.argv.find((a) => a.startsWith('--channel=')) || '').split('=')[1];
fs.mkdirSync(out, { recursive: true });
await fetch('http://localhost:5000/__mock/reset', { method: 'POST' });
await fetch('http://localhost:5000/__mock/mode?gated=0', { method: 'POST' });

const browser = await chromium.launch(channel ? { channel } : {});
const page = await browser.newPage({ viewport: { width: 1440, height: 810 } });
const errors = [];
page.on('pageerror', (e) => errors.push('pageerror ' + e.message));
page.on('console', (m) => m.type() === 'error' && errors.push('console ' + m.text()));
page.on('requestfailed', (r) => errors.push('requestfailed ' + r.url()));
await page.goto('http://localhost:3000/', { waitUntil: 'networkidle' });
await page.screenshot({ path: `${out}/01-hero.png` });

// drop both files on the dropzone via a real DataTransfer
const files = ['Q3_Report.pdf', 'Monthly_Revenue.csv'].map((n) => ({
  name: n, type: n.endsWith('.pdf') ? 'application/pdf' : 'text/csv',
  b64: fs.readFileSync(path.join(FIX, n)).toString('base64'),
}));
const dropzone = page.locator('text=Upload Dokumen').locator('xpath=ancestor::div[contains(@class,"border-dashed")]');
const dt = await page.evaluateHandle((files) => {
  const dt = new DataTransfer();
  for (const f of files) {
    const bin = Uint8Array.from(atob(f.b64), (c) => c.charCodeAt(0));
    dt.items.add(new File([bin], f.name, { type: f.type }));
  }
  return dt;
}, files);
await dropzone.dispatchEvent('dragover', { dataTransfer: dt });
await dropzone.dispatchEvent('drop', { dataTransfer: dt });
await page.waitForSelector('text=Mulai Proses & Chat');
await page.screenshot({ path: `${out}/02-selected.png` });
await page.click('text=Mulai Proses & Chat');
await page.waitForTimeout(300);
await page.screenshot({ path: `${out}/03-progress.png` });
await page.waitForSelector('text=Uploaded Document');
await page.waitForSelector('button[title="Filter by document"]');
await page.screenshot({ path: `${out}/04-chat.png` });

await page.click('button[title="Filter by document"]');
await page.locator('label', { hasText: 'Q3_Report.pdf' }).click();
await page.screenshot({ path: `${out}/05-filter.png` });
await page.mouse.click(700, 300); // click outside closes popover
await page.fill('textarea', 'What are the key Q3 results?');
await page.keyboard.press('Enter');
await page.waitForSelector('text=References');
await page.waitForTimeout(800);
await page.screenshot({ path: `${out}/06-answer.png` });
await page.locator('button[title="View Q3_Report.pdf"]').first().click();
await page.waitForTimeout(2500);
await page.screenshot({ path: `${out}/07-pdf.png` });
await page.locator('button[aria-label="Close panel"]:visible').click();
await page.waitForTimeout(800);
await page.click('button[title^="Filtered"]');
await page.locator('label', { hasText: 'Q3_Report.pdf' }).click();
await page.locator('label', { hasText: 'Monthly_Revenue.csv' }).click();
await page.screenshot({ path: `${out}/08-filter-csv.png` });
await page.mouse.click(700, 300);
await page.fill('textarea', 'Chart monthly revenue for Q3.');
await page.keyboard.press('Enter');
await page.waitForSelector('img[alt="Data Visualization"]');
await page.waitForTimeout(1200);
await page.screenshot({ path: `${out}/09-chart.png` });
const log = await (await fetch('http://localhost:5000/__mock/log')).json();
fs.writeFileSync(`${out}/mock-log.json`, JSON.stringify(log, null, 2));
console.log('errors:', errors);
console.log(log.log.filter((l) => l.path === '/api/chat').map((l) => JSON.stringify(l.body)).join('\n'));
await browser.close();
