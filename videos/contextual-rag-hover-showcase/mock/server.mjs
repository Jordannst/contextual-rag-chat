// Controlled demo API for the Contextual RAG Chat showcase.
// Listens on http://localhost:5000 (the URL hardcoded in the frontend) and answers only
// the endpoints the capture flow uses, with fictional Nara Studio data. No AI, DB or credentials.
//
// Two pacing modes:
//   auto   (default) – uploads answer after a short delay, SSE events stream on a timer.
//   gated  – every upload response and every SSE event waits for POST /__mock/release,
//            so the frame-by-frame renderer decides exactly on which frame each one lands.
//
// Control endpoints (not part of the app contract):
//   POST /__mock/reset            -> clears documents, sessions, request log
//   POST /__mock/mode?gated=1|0   -> switch pacing
//   POST /__mock/release          -> releases the oldest pending gated item, returns its label
//   GET  /__mock/pending          -> labels of pending gated items
//   GET  /__mock/log              -> recorded requests (method, path, body summary)
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const FIX = path.join(here, '..', 'fixtures');
const PORT = Number(process.env.MOCK_PORT || 5000);
const SESSION_ID = 101;

const CHART_B64 = fs.readFileSync(path.join(FIX, 'chart_q3_revenue.png')).toString('base64');

// Scripted responses. Chunks are split on natural word groups and never inside a **bold** pair
// or the citation, so the real Markdown renderer never shows half-formed syntax.
const PDF_ANSWER = [
  'Q3 revenue totaled ',
  '**Rp454 million**. ',
  'September reached ',
  '**Rp186 million**, ',
  'up **55%** from July. ',
  '(Q3_Report.pdf)',
];
const CSV_ANSWER = [
  'September led at ',
  '**Rp186 million**. ',
  'Revenue rose **55%** ',
  'from July to September.',
];

const state = { docs: [], sessions: [], messages: [], log: [], gated: false, queue: [] };

function reset() {
  state.docs = [];
  state.sessions = [];
  state.messages = [];
  state.log = [];
  for (const q of state.queue) q.resolve();
  state.queue = [];
}

// Wait for a release (gated) or a fixed delay (auto).
function gate(label, autoDelayMs) {
  if (!state.gated) return new Promise((r) => setTimeout(r, autoDelayMs));
  return new Promise((resolve) => state.queue.push({ label, resolve }));
}

function cors(res) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
}

function json(res, code, obj) {
  cors(res);
  res.writeHead(code, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(obj));
}

function readBody(req) {
  return new Promise((resolve) => {
    const parts = [];
    req.on('data', (c) => parts.push(c));
    req.on('end', () => resolve(Buffer.concat(parts)));
  });
}

// Minimal multipart parser: returns { field, filename, size } of the first file part.
function parseMultipart(buf, contentType) {
  const m = /boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType || '');
  if (!m) return null;
  const boundary = '--' + (m[1] || m[2]);
  const text = buf.toString('latin1');
  for (const part of text.split(boundary)) {
    const head = part.slice(0, part.indexOf('\r\n\r\n'));
    const name = /name="([^"]*)"/.exec(head);
    const fn = /filename="([^"]*)"/.exec(head);
    if (name && fn) {
      const bodyStart = part.indexOf('\r\n\r\n') + 4;
      const size = part.length - bodyStart - 2; // trailing CRLF
      return { field: name[1], filename: fn[1], size };
    }
  }
  return null;
}

function sse(res, event, data) {
  res.write((event ? `event: ${event}\n` : '') + `data: ${JSON.stringify(data)}\n\n`);
}

async function handleChat(req, res, body) {
  let payload;
  try {
    payload = JSON.parse(body.toString('utf8'));
  } catch {
    return json(res, 400, { error: 'Invalid request body' });
  }
  const selected = payload.selectedFiles || [];
  const isData = selected.some((f) => /\.(csv|xlsx|xls)$/i.test(f));
  state.log[state.log.length - 1].body = payload;

  cors(res);
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    Connection: 'keep-alive',
  });
  res.flushHeaders?.();

  const isNew = !state.sessions.length;
  if (isNew) {
    state.sessions.unshift({ id: SESSION_ID, title: payload.question, created_at: '2026-10-07T10:00:00Z' });
  }
  const meta = isData
    ? { type: 'metadata', sources: ['Monthly_Revenue.csv'], analysis: true, sessionId: SESSION_ID }
    : { type: 'metadata', sources: ['Q3_Report.pdf'], sourceIds: [1], sessionId: SESSION_ID };

  await gate('metadata', 250);
  sse(res, 'metadata', meta);
  const chunks = isData ? CSV_ANSWER : PDF_ANSWER;
  for (let i = 0; i < chunks.length; i++) {
    await gate(`chunk ${i + 1}/${chunks.length}`, 140);
    sse(res, null, { type: 'chunk', chunk: chunks[i] });
  }
  if (isData) {
    await gate('chart', 300);
    sse(res, 'chart', { type: 'chart', chartData: CHART_B64, index: 0 });
  }
  await gate('done', 80);
  const full = chunks.join('');
  sse(res, 'done', { type: 'done', sessionId: SESSION_ID, fullLength: full.length, ...(isData ? { analysis: true, chartCount: 1 } : {}) });
  res.end();
  state.messages.push(
    { id: state.messages.length + 1, role: 'user', content: payload.question, created_at: '2026-10-07T10:00:00Z' },
    { id: state.messages.length + 2, role: 'model', content: full, created_at: '2026-10-07T10:00:01Z' },
  );
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const p = decodeURIComponent(url.pathname);
  if (req.method === 'OPTIONS') {
    cors(res);
    res.writeHead(204);
    return res.end();
  }
  const body = await readBody(req);
  if (!p.startsWith('/__mock')) state.log.push({ method: req.method, path: p });

  // ---- control
  if (p === '/__mock/reset') { reset(); return json(res, 200, { ok: true }); }
  if (p === '/__mock/mode') { state.gated = url.searchParams.get('gated') === '1'; return json(res, 200, { gated: state.gated }); }
  if (p === '/__mock/pending') return json(res, 200, { pending: state.queue.map((q) => q.label) });
  if (p === '/__mock/log') return json(res, 200, { log: state.log, docs: state.docs });
  if (p === '/__mock/release') {
    const q = state.queue.shift();
    if (q) q.resolve();
    return json(res, 200, { released: q ? q.label : null, pending: state.queue.length });
  }

  // ---- app contract
  if (req.method === 'GET' && p === '/api/chat/suggestions') return json(res, 200, { questions: [] });
  if (req.method === 'GET' && p === '/api/documents') return json(res, 200, { documents: state.docs, count: state.docs.length });
  if (req.method === 'GET' && p === '/api/sessions') return json(res, 200, { sessions: state.sessions });
  if (req.method === 'GET' && /^\/api\/sessions\/\d+$/.test(p)) return json(res, 200, { messages: state.messages });
  if (req.method === 'POST' && p === '/api/upload') {
    const file = parseMultipart(body, req.headers['content-type']);
    if (!file || file.field !== 'document') return json(res, 400, { error: 'No file uploaded', message: 'Field "document" is required' });
    state.log[state.log.length - 1].body = file;
    await gate(`upload ${file.filename}`, 450);
    if (!state.docs.includes(file.filename)) state.docs.push(file.filename);
    return json(res, 200, {
      fileName: file.filename,
      filePath: `uploads/${file.filename}`,
      message: 'File berhasil diupload, divektorisasi, dan disimpan ke database (3 chunks)',
      chunksCount: 3,
    });
  }
  if (req.method === 'POST' && p === '/api/chat') return handleChat(req, res, body);
  if (req.method === 'GET' && p.startsWith('/api/files/')) {
    const name = path.basename(p.slice('/api/files/'.length));
    const fp = path.join(FIX, name);
    if (!state.docs.includes(name) || !fs.existsSync(fp)) return json(res, 404, { error: 'File not found' });
    cors(res);
    const type = name.endsWith('.pdf') ? 'application/pdf' : name.endsWith('.csv') ? 'text/csv' : 'application/octet-stream';
    res.writeHead(200, { 'Content-Type': type, 'Content-Disposition': `inline; filename="${name}"` });
    return res.end(fs.readFileSync(fp));
  }
  console.warn('[mock] unhandled', req.method, p);
  return json(res, 404, { error: 'Not handled by demo mock' });
});

server.listen(PORT, () => console.log(`[mock] Contextual RAG demo API on http://localhost:${PORT}`));
