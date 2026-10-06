// Elite and route art check: fifteen elite and Gryphon Rider route classes on
// the battle screen, in three companies of five. Each draws its own class key
// (not its lineage), its sheet loads, and hovering names its class.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/elite-art-check.mjs [screenshot-prefix]
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

const member = (key, id, name, archetype, cell, cls, className) => ({
  key, id, name, status: 'present', level: 40, archetype, cell, chemistry: null, strategy: { role: 'fighter', target: 'weakest' },
  class: cls, class_name: className,
});
const groups = [
  [['pathfinder', 'Pathfinder', 'rogue'], ['swordmaster', 'Swordmaster', 'rogue'], ['nightblade', 'Nightblade', 'rogue'],
   ['sentinel', 'Sentinel', 'ranger'], ['marksman', 'Marksman', 'ranger']],
  [['ravager', 'Ravager', 'ranger'], ['archon', 'Archon', 'wizard'], ['archmage', 'Archmage', 'wizard'],
   ['necromancer', 'Necromancer', 'wizard'], ['wise-one', 'Wise One', 'witch']],
  [['coven-mother', 'Coven Mother', 'witch'], ['crone-of-ash', 'Crone of Ash', 'witch'], ['gryphon-knight', 'Gryphon Knight', 'gryphon-rider'],
   ['skyscout', 'Skyscout', 'gryphon-rider'], ['wyvern-rider', 'Wyvern Rider', 'gryphon-rider']],
];
const cells = [{ row: 0, col: 0 }, { row: 1, col: 0 }, { row: 2, col: 0 }, { row: 0, col: 1 }, { row: 2, col: 1 }];

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto(base + '/scripts/browser/dock-windows-harness.html');
await page.evaluate(() => localStorage.clear());
await page.reload();
const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
await page.evaluate(() => window.BattleScreen.setMotion('off'));
await gmcp('Room', { Info: { environment: 'forest', area: 'Frostfang' } });

const all = groups.flat().map(g => g[0]);
const sheets = await page.evaluate(async ([base, keys]) => Promise.all(keys.map(k => Promise.all(['battle/units/' + k + '/idle.png', 'map/units/' + k + '/idle.png', 'map/units/' + k + '/walk.png'].map(f => new Promise(res => {
  const img = new Image();
  img.onload = () => res(img.naturalWidth > 0);
  img.onerror = () => res(false);
  img.src = base + '/_datafiles/html/public/static/sprites/' + f;
}))).then(r => k + ':' + (r.every(Boolean) ? 'ok' : 'missing')))), [base, all]);
check(sheets.every(x => x.endsWith(':ok')), 'every battle and map sheet loads: ' + sheets.join(' '));

for (const [gi, g] of groups.entries()) {
  const m = g.map((c, i) => member(i === 0 ? 'leader' : 'companion:' + i, i, c[1] + 'son', c[2], cells[i], c[0], c[1]));
  const company = { leader: m[0], members: m.slice(1), alive: 5, dead: 0, vitals: {} };
  const positions = {};
  m.forEach(x => { positions[x.key] = x.cell; company.vitals[x.key] = { hp: 30, hp_max: 40 }; });
  await gmcp('Company', company);
  await gmcp('Company.Battle', {
    group: 'a pack of timber wolves', narrow: false, positions,
    enemies: [{ id: 'm:1', label: 'the first wolf', cell: { row: 1, col: 1 }, health: 'unhurt', sprite: 'wolf-timber', reach: true, target: '' }],
    dolls: [], company: [], focus: 'none', saved_focus: 'none', focus_ready: true,
  });
  await page.waitForTimeout(500);
  const s = await page.evaluate(() => window.BattleScreen.state());
  for (const [i, c] of g.entries()) {
    const u = s.units.find(x => x.id === m[i].key);
    check(u && u.promoted === c[0], c[1] + ' draws its own class key, not ' + c[2]);
  }
  for (const i of [0, 4]) {
    const b = s.units.find(x => x.id === m[i].key).at;
    const box = await page.locator('#battle-screen canvas').boundingBox();
    await page.mouse.move(box.x + b.x * box.width / 320, box.y + (b.y - 12) * box.height / 180);
    const cap = await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent);
    check(cap.includes(g[i][1]), 'hovering names ' + g[i][1] + ': ' + cap);
  }
  await page.mouse.move(0, 0);
  if (shot) { await page.locator('#battle-screen').screenshot({ path: shot.replace(/\.png$/, '') + '-' + (gi + 1) + '.png' }); }
}
await browser.close();
server.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
