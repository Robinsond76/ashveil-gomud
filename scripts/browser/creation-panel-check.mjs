// Phase 72a browser check: drives the real window-creation.js and the
// sprite tint (through creation-panel-harness.html with the real web client
// core and sprite loader) in Chromium or Firefox with Playwright.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/creation-panel-check.mjs [outdir]
//
// It checks that a step shows as a labelled dialog with real buttons, that a
// click sends the same input a telnet player types, that markup in a server
// string is text, that the figure is drawn in the chosen skin and hair, that
// "Type instead" tucks the panel away and a new step brings it back, and that
// the panel closes when creation ends. BROWSER=firefox selects Firefox.
import { createRequire } from 'node:module';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const pw = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const outdir = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json' };
const server = http.createServer((req, res) => {
  if (req.url === '/favicon.ico') { res.writeHead(204); res.end(); return; }
  const file = path.join(root, decodeURIComponent(req.url.split('?')[0]));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const url = 'http://127.0.0.1:' + server.address().port + '/scripts/browser/creation-panel-harness.html';

const kind = process.env.BROWSER === 'firefox' ? pw.firefox : pw.chromium;
const browser = await kind.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1024, height: 800 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto(url);
await page.waitForFunction(() => window.Sprites && Sprites.status && window.CreationPanel);

const step = {
  active: true, mode: 'new', step: 'looks', key: 'skin', kind: 'choice',
  title: 'What is your skin tone? <img src=x onerror="window.__xss=1">', number: 3, total: 12,
  options: [
    { id: 'pale', name: 'Pale', color: '#e8cdb5' },
    { id: 'dark', name: 'Dark', color: '#5a3a28' },
    { id: 'bad', name: 'Odd', color: 'red;background:url(x)' },
  ],
  preview: 'You are tall and lean.', lineage: 'adventurer', skin: '#e8cdb5', hair: '#2a1a10', canback: true,
};

const open = () => page.evaluate(() => document.getElementById('creation-backdrop').classList.contains('open'));
check(!(await open()), 'hidden before any step');
await page.evaluate(s => window.gmcp('Char.Creation', s), step);
check(await open(), 'a step opens the panel');
const dialog = page.locator('#creation-panel');
check(await dialog.getAttribute('role') === 'dialog' && await dialog.getAttribute('aria-modal') === 'true', 'a modal dialog');
check((await page.locator('#creation-heading').textContent()).length > 0, 'a heading labels the dialog');
check((await page.locator('.creation-count').textContent()) === 'Question 3 of 12', 'question counter');
check(await page.locator('#creation-question').textContent() === step.title && await page.locator('#creation-panel img').count() === 0, 'markup in the question is text');
check(await page.evaluate(() => window.__xss === undefined), 'no script ran');
check(await page.locator('.creation-options button').count() === 3, 'one real button per answer');
check(await page.locator('.creation-swatch').count() === 2, 'a bad colour makes no swatch');
check(await page.evaluate(() => document.activeElement && document.activeElement.closest('.creation-options') !== null), 'focus on the first answer');
check((await page.locator('[role=status]').textContent()).startsWith('Question 3 of 12'), 'the step is announced');

// The figure is drawn tinted: compare with an untinted figure.
await page.waitForFunction(() => document.querySelector('.creation-preview-art'), null, { timeout: 5000 }).catch(() => {});
const hasFigure = await page.locator('.creation-preview-art').count() === 1;
check(hasFigure, 'the figure is drawn');
if (hasFigure) {
  const sig = () => page.evaluate(() => {
    const cv = document.querySelector('.creation-preview-art');
    return Array.from(cv.getContext('2d').getImageData(0, 0, cv.width, cv.height).data).join(',');
  });
  const a = await sig();
  await page.evaluate(s => window.gmcp('Char.Creation', { ...s, skin: '#5a3a28', hair: '#d8b868', key: 'hair', number: 4 }), step);
  await page.waitForTimeout(200);
  const b = await sig();
  check(a !== b, 'a different skin and hair repaint the figure');
}
if (outdir) { await page.screenshot({ path: path.join(outdir, 'creation-desktop.png') }); }

// A click sends the telnet input.
await page.evaluate(s => window.gmcp('Char.Creation', s), step);
await page.locator('.creation-options button').nth(1).click();
check(JSON.stringify(await page.evaluate(() => window.sent)) === '["2"]', 'a click sends the option number');
await page.getByRole('button', { name: /Back/ }).click();
check((await page.evaluate(() => window.sent)).pop() === 'back', 'Back sends back');

// Type instead tucks away; a new step brings it back; the tab reopens it.
await page.getByRole('button', { name: 'Type instead' }).click();
check(!(await open()) && await page.locator('#creation-tab.open').count() === 1, 'Type instead hides the panel and offers a way back');
check(await page.evaluate(() => !document.body.classList.contains('creation-open')), 'the client is usable while it is tucked away');
await page.locator('#creation-tab').click();
check(await open(), 'the tab reopens the panel');
await page.getByRole('button', { name: 'Type instead' }).click();
await page.evaluate(s => window.gmcp('Char.Creation', { ...s, key: 'hair', number: 4, title: 'Hair colour?' }), step);
check(await open(), 'a new question brings the panel back');

// A text step, with a backstory.
await page.evaluate(s => window.gmcp('Char.Creation', { ...s, kind: 'text', key: 'line', options: [], title: 'A line of your own?', backstory: 'You grew up on the river.', canskip: true }), step);
await page.locator('.creation-line input').fill('hello there');
await page.getByRole('button', { name: 'Use this line' }).click();
check((await page.evaluate(() => window.sent)).pop() === 'hello there', 'a typed line is sent as typed');
await page.getByRole('button', { name: 'No line' }).click();
check((await page.evaluate(() => window.sent)).pop() === 'none', 'No line sends none');
check(await page.locator('.creation-backstory').textContent() === 'You grew up on the river.', 'the backstory shows');
await page.getByRole('button', { name: 'Skip for now' }).click();
check((await page.evaluate(() => window.sent)).pop() === 'skip', 'Skip sends skip');

// Narrow layout.
await page.setViewportSize({ width: 360, height: 700 });
check(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), 'no horizontal scroll at 360px');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'creation-narrow.png') }); }

// End.
await page.evaluate(() => window.gmcp('Char.Creation', { active: false }));
check(!(await open()) && await page.locator('#creation-panel *').count() === 0, 'the panel closes and empties when creation ends');

await browser.close();
server.close();
console.log(failures ? failures + ' failure(s)' : 'all ok');
process.exit(failures ? 1 : 0);
