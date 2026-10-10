// E3 browser check: the battle screen draws the commissioned high-density
// art at device pixels. A company of base and promoted classes meets a
// wolf, a rat and a forest ogre (M, S and L art) at pixel ratios 1, 1.5
// and 2: the canvas's backing store is the shown size times the ratio,
// every figure and the backdrop draw their imported art, and the
// Houndmaster's warhound draws its own A7 sheet.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/battle-art-check.mjs [screenshot.png]
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
  key, id, name, status: 'present', level: 12, archetype, cell, chemistry: null, strategy: { role: 'fighter', target: 'weakest' },
  ...(cls ? { class: cls, class_name: className } : {}),
});
const company = {
  leader: member('leader', 0, 'Wren', 'Warrior', { row: 0, col: 1 }, 'paladin', 'Paladin'),
  members: [
    member('companion:1', 1, 'Oswin', 'Cleric', { row: 1, col: 0 }),
    member('companion:2', 2, 'Ysolde', 'Wizard', { row: 2, col: 2 }, 'archmage', 'Archmage'),
  ],
  alive: 3, dead: 0,
  vitals: { leader: { hp: 30, hp_max: 40 }, 'companion:1': { hp: 20, hp_max: 20 }, 'companion:2': { hp: 9, hp_max: 20 } },
};
const positions = { leader: { row: 0, col: 1 }, 'companion:1': { row: 1, col: 0 }, 'companion:2': { row: 2, col: 2 } };
const enemy = (n, label, row, col, sprite) => ({ id: 'm:' + n, label, cell: { row, col }, health: 'unhurt', sprite, reach: true, target: '' });
const battle = {
  group: 'a mixed rabble', narrow: false, positions,
  enemies: [enemy(1, 'a timber wolf', 0, 0, 'wolf-timber'), enemy(2, 'a rat', 0, 2, 'rat'), enemy(3, 'a forest ogre', 1, 1, 'ogre-forest')],
  company: [], focus: 'none', saved_focus: 'none', focus_ready: true,
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
for (const dpr of [1, 1.5, 2]) {
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 }, deviceScaleFactor: dpr });
  const page = await ctx.newPage();
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  await page.goto(base + '/scripts/browser/dock-windows-harness.html');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
  await page.evaluate(() => window.BattleScreen.setMotion('off'));
  await gmcp('Room', { Info: { environment: 'forest', area: 'Frostfang' } });
  await gmcp('Company', company);
  await gmcp('Company.Battle', battle);
  await page.waitForFunction(() => {
    const st = window.BattleScreen.state();
    return st.backdrop && st.units.filter(u => u.cell).every(u => u.art && u.art.density > 1);
  }, null, { timeout: 8000 }).catch(() => {});
  const s = await page.evaluate(() => window.BattleScreen.state());
  const box = await page.locator('#battle-screen canvas').boundingBox();
  const shown = Math.round(box.width / 320);
  check(s.pxScale === shown * dpr && s.backing[0] === Math.round(320 * s.pxScale) && s.backing[1] === Math.round(180 * s.pxScale),
    'ratio ' + dpr + ': the canvas draws at device pixels (' + shown + 'x shown, backing ' + JSON.stringify(s.backing) + ')');
  check(s.backdrop && s.backdrop.density === 2, 'ratio ' + dpr + ': the forest backdrop is the imported art: ' + JSON.stringify(s.backdrop));
  const art = id => (s.units.find(u => u.id === id) || {}).art || {};
  check(art('leader').path === 'battle/units/paladin/idle.png' && art('leader').density === 2, 'ratio ' + dpr + ': a promoted member draws its class\'s imported sheet: ' + JSON.stringify(art('leader')));
  check(art('companion:1').path === 'battle/units/cleric/idle.png' && art('companion:1').density === 2, 'ratio ' + dpr + ': a base member draws its lineage\'s imported sheet');
  check(art('m:1').density === 2 && Math.abs(art('m:2').density - 8 / 3) < 1e-9 && art('m:3').density === 2,
    'ratio ' + dpr + ': M, S and L creatures draw their imported art: ' + JSON.stringify([art('m:1'), art('m:2'), art('m:3')]));
  if (shot) { await page.locator('#battle-screen').screenshot({ path: shot.replace(/\.png$/, '-dpr' + dpr + '.png') }); }
  if (dpr === 1) {
    // The Houndmaster's warhound draws the A7 warhound, not the junkyard dog.
    // A Beast Tamer's beast stands among the battle's dolls (39e).
    await gmcp('Company.Battle', { ...battle, positions: { ...positions, 'beast:1': { row: 1, col: 2 } },
      dolls: [{ key: 'beast:1', kind: 'warhound', name: 'Fang' }] });
    await page.waitForFunction(() => {
      const u = window.BattleScreen.state().units.find(o => o.id === 'beast:1');
      return u && u.art;
    }, null, { timeout: 5000 }).catch(() => {});
    const s2 = await page.evaluate(() => window.BattleScreen.state());
    const hound = s2.units.find(u => u.id === 'beast:1') || {};
    check(hound.sprite === 'warhound' && hound.art && hound.art.path === 'battle/units/warhound/idle.png' && hound.art.density === 2,
      'a warhound draws its own A7 sheet: ' + JSON.stringify([hound.sprite, hound.art]));
  }
  await ctx.close();
}

// A pose sheet for a base class (warrior/hurt.png, served here from the
// warrior's idle) must not freeze a promoted member: the Paladin draws its
// own class's sheets, which have no hurt pose, so it still flinches (the
// nudge) instead of standing still.
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  const page = await ctx.newPage();
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  const spritesDir = path.join(root, '_datafiles/html/public/static/sprites');
  await page.route('**/sprites/manifest.json', async r => {
    const m = JSON.parse(fs.readFileSync(path.join(spritesDir, 'manifest.json'), 'utf8'));
    m.files['battle/units/warrior/hurt.png'] = Object.assign({}, m.files['battle/units/warrior/idle.png']);
    await r.fulfill({ contentType: 'application/json', body: JSON.stringify(m) });
  });
  await page.route('**/battle/units/warrior/hurt.png', r => r.fulfill({ contentType: 'image/png', body: fs.readFileSync(path.join(spritesDir, 'battle/units/warrior/idle.png')) }));
  await page.goto(base + '/scripts/browser/dock-windows-harness.html');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
  await page.evaluate(() => window.BattleScreen.setMotion('full'));
  await gmcp('Room', { Info: { environment: 'forest', area: 'Frostfang' } });
  await gmcp('Company', company);
  await gmcp('Company.Battle', battle);
  await page.waitForFunction(() => { const u = window.BattleScreen.state().units.find(o => o.id === 'leader'); return u && u.art; }, null, { timeout: 8000 }).catch(() => {});
  // The first blow asks for the pose sheets, which load lazily; the second
  // is planned with them loaded.
  await page.evaluate(b => Client.dispatchBattleEvents(b), { fight: 1, round: 5001, fight_round: 1,
    events: [{ seq: 1, kind: 'attack', src: 'm:1', tgt: 'leader', outcome: 'hit', damage: 3 }] });
  await page.waitForFunction(() => window.BattleScreen.state().backlog === 0, null, { timeout: 8000 }).catch(() => {});
  await page.waitForTimeout(300);
  await page.evaluate(b => Client.dispatchBattleEvents(b), { fight: 1, round: 5002, fight_round: 2,
    events: [{ seq: 2, kind: 'attack', src: 'm:1', tgt: 'leader', outcome: 'hit', damage: 3 }] });
  let moved = false, anims = new Set();
  for (let i = 0; i < 40 && !moved; i++) {
    const u = await page.evaluate(() => window.BattleScreen.state().units.find(o => o.id === 'leader'));
    if (u.pose) { anims.add(u.pose.anim); if (u.pose.dx !== 0) { moved = true; } }
    await page.waitForTimeout(40);
  }
  check(moved, 'a promoted member flinches when struck though its base class has a hurt sheet: ' + JSON.stringify(Array.from(anims)));
  await ctx.close();
}
await browser.close();
server.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
