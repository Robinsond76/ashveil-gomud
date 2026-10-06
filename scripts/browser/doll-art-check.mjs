// Phase 39d art check: a Doll Master company on the battle screen. The
// Doll Master lineage and its advanced classes draw their own sheets, and a
// doll draws as the wooden puppet (not a silhouette); the sheets load.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/doll-art-check.mjs [screenshot.png]
import { createRequire } from 'node:module';
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const shot = process.argv[2];
const root = path.resolve(here, '..', '..');

// The sprite manifest is fetched, so serve the repository over HTTP.
const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json' };
const server = http.createServer((req, res) => {
  const file = path.join(root, decodeURIComponent(req.url.split('?')[0]));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const base = 'http://127.0.0.1:' + server.address().port;

let failures = 0;
const check = (ok, what) => { if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); } };

const member = (key, id, name, archetype, cell, role, cls, className) => ({
  key, id, name, status: 'present', level: 12, archetype, cell, chemistry: null, strategy: { role, target: 'weakest' },
  ...(cls ? { class: cls, class_name: className } : {}),
});
const roster = [
  ['leader', 'Vesper', 'dollmaster', 'puppeteer', 'Puppeteer', 1, 0],
  ['companion:1', 'Cog', 'dollmaster', 'golemancer', 'Golemancer', 0, 0],
  ['companion:2', 'Wick', 'dollmaster', 'marionettist', 'Marionettist', 2, 0],
  ['companion:3', 'Tam', 'dollmaster', null, null, 0, 1],
];
const company = {
  leader: member('leader', 0, roster[0][1], roster[0][2], { row: 1, col: 0 }, 'fighter', roster[0][3], roster[0][4]),
  members: roster.slice(1).map((r, i) => member(r[0], i + 1, r[1], r[2], { row: r[5], col: r[6] }, 'fighter', r[3], r[4])),
  alive: 4, dead: 0, vitals: {},
};
const positions = { 'doll:1': { row: 1, col: 1 } };
roster.forEach(r => { positions[r[0]] = { row: r[5], col: r[6] }; company.vitals[r[0]] = { hp: 30, hp_max: 40 }; });
const battle = {
  group: 'a pack of timber wolves', narrow: false, positions,
  enemies: [{ id: 'm:1', label: 'the first wolf', cell: { row: 1, col: 1 }, health: 'unhurt', sprite: 'wolf-timber', reach: true, target: '' }],
  dolls: [{ key: 'doll:1', name: 'Cog the doll', hp: 20, hp_max: 30 }],
  company: [], focus: 'none', saved_focus: 'none', focus_ready: true,
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto(base + '/scripts/browser/dock-windows-harness.html');
await page.evaluate(() => localStorage.clear());
await page.reload();
const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
await page.evaluate(() => window.BattleScreen.setMotion('off'));
await gmcp('Room', { Info: { environment: 'forest', area: 'Frostfang' } });
await gmcp('Company', company);
await gmcp('Company.Battle', battle);
await page.waitForTimeout(500);
const s = await page.evaluate(() => window.BattleScreen.state());
const unit = id => s.units.find(u => u.id === id);

check(unit('leader').promoted === 'puppeteer' && unit('companion:3').promoted === '', 'promoted members draw their class; an unpromoted Doll Master draws its lineage');
check(unit('companion:3').sprite === 'dollmaster', 'the Doll Master lineage is its own sprite key');
check(unit('doll:1') && unit('doll:1').sprite === 'doll', 'the doll is a unit with the doll sprite');

const sheets = await page.evaluate(async ([base, keys]) => Promise.all(keys.map(k => new Promise(res => {
  const img = new Image();
  img.onload = () => res(k + ':' + img.naturalWidth);
  img.onerror = () => res(k + ':missing');
  img.src = base + '/_datafiles/html/public/static/sprites/battle/units/' + k + '/idle.png';
}))), [base, ['dollmaster', 'puppeteer', 'golemancer', 'marionettist', 'doll']]);
check(sheets.every(x => !x.endsWith(':missing') && !x.endsWith(':0')), 'every sheet loads: ' + sheets.join(' '));

for (const [key, want] of [['leader', 'Puppeteer'], ['companion:2', 'Marionettist'], ['doll:1', 'Doll']]) {
  const b = unit(key).at;
  const box = await page.locator('#battle-screen canvas').boundingBox();
  await page.mouse.move(box.x + b.x * box.width / 320, box.y + (b.y - 12) * box.height / 180);
  const cap = await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent);
  check(cap.includes(want), 'hovering names ' + want + ': ' + cap);
}
await page.mouse.move(0, 0);
if (shot) { await page.locator('#battle-screen').screenshot({ path: shot }); }
await browser.close();
server.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
