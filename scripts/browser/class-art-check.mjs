// Phase 40h browser check: every class has art on the battle screen. A
// company of neutral and elite classes is fed in; each member draws its own
// sheet (not the silhouette), hovering names its class, and the sheets load.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/class-art-check.mjs [screenshot.png]
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
  ['leader', 'Hale', 'Halberdier', 'valkyrie', 'Valkyrie', 0, 1],
  ['companion:1', 'Isamu', 'Samurai', 'kensai', 'Kensai', 1, 0],
  ['companion:2', 'Noa', 'Shaman', 'stormcaller', 'Stormcaller', 2, 1],
  ['companion:3', 'Gorm', 'Samurai', 'ronin', 'Ronin', 0, 0],
  ['companion:4', 'Hild', 'Halberdier', 'vanguard', 'Vanguard', 1, 2],
  ['companion:5', 'Brun', 'Shaman', null, null, 2, 2],
  ['companion:6', 'Tor', 'Warrior', 'warlord', 'Warlord', 2, 0],
];
const company = {
  leader: member('leader', 0, roster[0][1], roster[0][2], { row: 0, col: 1 }, 'fighter', roster[0][3], roster[0][4]),
  members: roster.slice(1).map((r, i) => member(r[0], i + 1, r[1], r[2], { row: r[5], col: r[6] }, 'fighter', r[3], r[4])),
  alive: 7, dead: 0, vitals: {},
};
const positions = {};
roster.forEach(r => { positions[r[0]] = { row: r[5], col: r[6] }; });
roster.forEach(r => { company.vitals[r[0]] = { hp: 30, hp_max: 40 }; });
const battle = {
  group: 'a pack of timber wolves', narrow: false, positions,
  enemies: [{ id: 'm:1', label: 'the first wolf', cell: { row: 1, col: 1 }, health: 'unhurt', sprite: 'wolf-timber', reach: true, target: '' }],
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

check(unit('leader').promoted === 'valkyrie' && unit('companion:5').promoted === '', 'promoted members draw their class; an unpromoted Shaman draws its lineage');
check(unit('companion:5').sprite === 'shaman' && unit('companion:4').sprite === 'halberdier', 'neutral lineages are their own sprite keys, not the adventurer');

const sheets = await page.evaluate(async ([base, keys]) => Promise.all(keys.map(k => new Promise(res => {
  const img = new Image();
  img.onload = () => res(k + ':' + img.naturalWidth);
  img.onerror = () => res(k + ':missing');
  img.src = base + '/_datafiles/html/public/static/sprites/battle/units/' + k + '/idle.png';
}))), [base, ['halberdier', 'samurai', 'shaman', 'valkyrie', 'kensai', 'stormcaller', 'ronin', 'vanguard', 'warlord']]);
check(sheets.every(x => !x.endsWith(':missing') && !x.endsWith(':0')), 'every sheet loads: ' + sheets.join(' '));

for (const [key, want] of [['leader', 'Valkyrie'], ['companion:2', 'Stormcaller'], ['companion:6', 'Warlord']]) {
  const b = unit(key).at;
  const box = await page.locator('#battle-screen canvas').boundingBox();
  await page.mouse.move(box.x + b.x * box.width / 320, box.y + (b.y - 12) * box.height / 180);
  const cap = await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent);
  check(cap.includes(want), 'hovering names the class ' + want + ': ' + cap);
}
await page.mouse.move(0, 0);
if (shot) { await page.locator('#battle-screen').screenshot({ path: shot }); }
await browser.close();
server.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
