// Phase 40i browser check: the phone layout of the real web client
// (webclient-pure.html with its template fields filled in, served over HTTP
// with the real scripts and sprites) in Chromium at a phone-sized touch
// viewport. The views (Game, Map, Here, Company), the touch bar and its
// "Walk to..." list, tap-to-walk on the tiled map, touch pan and pinch, the
// Camp tab and Room Info gather progress, the battle screen (with tapping an
// allied formation to watch it) and the web app manifest.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/mobile-check.mjs [outdir]
import { createRequire } from 'node:module';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const pub = path.join(root, '_datafiles/html/public');
const outdir = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json', '.webmanifest': 'application/manifest+json', '.mp3': 'audio/mpeg' };
const server = http.createServer((req, res) => {
  const url = decodeURIComponent(req.url.split('?')[0]);
  if (url === '/favicon.ico') { res.writeHead(204); res.end(); return; }
  if (url === '/webclient-pure.html') {
    const html = fs.readFileSync(path.join(pub, 'webclient-pure.html'), 'utf8')
      .replaceAll('{{ .ASSET_BASE_URL }}', '').replaceAll('{{ .CONFIG.Server.MudName }}', 'Ashveil');
    res.writeHead(200, { 'content-type': 'text/html' });
    res.end(html);
    return;
  }
  const file = path.join(pub, url);
  if (!file.startsWith(pub) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const base = 'http://127.0.0.1:' + server.address().port;

// A 6 x 4 field of rooms; two of them are named landmarks.
const W = 6, H = 4;
const id = (x, y) => 1 + x + W * y;
const names = { [id(5, 0)]: ['The Old Bridge', 'bridge'], [id(0, 3)]: ['Hermit\'s Hut', 'hermit'], [id(3, 3)]: ['The Obelisk', 'obelisk'] };
function roomInfo(x, y) {
  const exitsv2 = {};
  const add = (name, dx, dy) => {
    const nx = x + dx, ny = y + dy;
    if (nx >= 0 && nx < W && ny >= 0 && ny < H) { exitsv2[name] = { num: id(nx, ny), dx, dy, dz: 0 }; }
  };
  add('north', 0, -1); add('south', 0, 1); add('east', 1, 0); add('west', -1, 0);
  const lm = names[id(x, y)];
  return { num: id(x, y), name: lm ? lm[0] : 'Meadow ' + x + ',' + y, area: 'Alderbrook', coords: ['Alderbrook', x, y, 0].join(','),
    environment: 'forest', exitsv2, details: [], ...(lm ? { maplegend: lm[1], mapsymbol: lm[1][0].toUpperCase() } : {}) };
}
const world = { biomes: { forest: { color: { bg: '#2f4f2f' } } }, rooms: [] };
for (let y = 0; y < H; y++) { for (let x = 0; x < W; x++) { world.rooms.push(roomInfo(x, y)); } }

const member = (key, id, name, archetype, cell, role) => ({
  key, id, name, status: 'present', level: 5, archetype, cell, chemistry: null, strategy: { role, target: 'weakest' },
});
const company = {
  leader: member('leader', 0, 'Wren', 'Ranger', { row: 0, col: 1 }, 'fighter'),
  members: [
    member('companion:1', 1, 'Oswin', 'Cleric', { row: 1, col: 0 }, 'healer'),
    { ...member('companion:2', 2, 'Brant', 'Warrior', { row: 0, col: 0 }, 'guardian'), class: 'knight', class_name: 'Knight' },
    member('companion:3', 3, 'Ysolde', 'Wizard', { row: 2, col: 2 }, 'caster'),
  ],
  alive: 4, dead: 0,
  vitals: {
    leader: { hp: 30, hp_max: 40 }, 'companion:1': { hp: 12, hp_max: 24 }, 'companion:2': { hp: 20, hp_max: 20 }, 'companion:3': { hp: 5, hp_max: 20 },
  },
};
const enemy = (n, label, row, col, health, sprite, target) => ({ id: 'm:' + n, label, cell: { row, col }, health, sprite, reach: true, target });
const battle = {
  group: 'a pack of timber wolves', narrow: false,
  positions: { leader: { row: 0, col: 1 }, 'companion:1': { row: 1, col: 0 }, 'companion:2': { row: 0, col: 0 }, 'companion:3': { row: 2, col: 2 } },
  enemies: [
    enemy(1, 'the first wolf', 0, 0, 'wounded', 'wolf-timber', 'companion:2'),
    enemy(2, 'the second wolf', 0, 2, 'unhurt', 'wolf-timber', 'leader'),
    enemy(3, 'the third wolf', 1, 1, 'near death', 'wolf-timber', 'companion:1'),
  ],
  company: [{ key: 'leader', target: 'm:2' }],
  focus: 'none', saved_focus: 'none', focus_ready: true,
  allies: [{ id: 'a:9', name: 'Maren', members: [
    { id: 'a:9:leader', name: 'Maren', class: 'wizard', cell: { row: 0, col: 0 }, health: 'scratched' },
    { id: 'a:9:companion:1', name: 'Pell', class: 'rogue', cell: { row: 2, col: 1 }, health: 'unhurt' },
  ] }],
  outlook: { risk: 'fair', close: true, text: 'It could go either way.' },
};

const VW = Number(process.env.MOBILE_WIDTH || 390);
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const ctx = await browser.newContext({ viewport: { width: VW, height: 780 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
const page = await ctx.newPage();
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
page.on('console', m => {
  if (m.type() === 'error' && !/Failed to load resource/.test(m.text())) { failures++; console.log('FAIL console error: ' + m.text()); }
});
await page.goto(base + '/webclient-pure.html');
await page.evaluate(() => localStorage.clear());
await page.reload();

await page.evaluate(() => {
  window.sent = [];
  Client.SendInput = cmd => { window.sent.push(cmd); };
  Client.GMCPRequest = () => {};
  window.gmcp = (namespace, body) => {
    const parts = namespace.split('.');
    const last = parts.pop();
    let cursor = Client.GMCPStructs;
    parts.forEach(seg => { cursor = cursor[seg] = cursor[seg] || {}; });
    cursor[last] = body;
    VirtualWindows.handleGMCP(namespace, body);
  };
});
await page.evaluate(() => { VirtualWindows.setConnected(true); document.getElementById('connect-button').style.display = 'none'; });
const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
const sent = () => page.evaluate(() => window.sent);
const clearSent = () => page.evaluate(() => { window.sent = []; });
const shot = async name => { if (outdir) { await page.screenshot({ path: path.join(outdir, '40i-' + name + '.png') }); } };
const box = sel => page.locator(sel).first().boundingBox();
const overflow = () => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
const visibleWins = sel => page.evaluate(s => [...document.querySelectorAll(s + ' .dock-panel')].filter(p => getComputedStyle(p).display !== 'none').map(p => p.dataset.win), sel);

// --- the layout and its controls ---
check(await page.evaluate(() => Mobile.active() && document.body.classList.contains('mobile')), 'a phone-width screen gets the phone layout');
check(await page.evaluate(() => Mobile.view()) === 'game', 'it starts on the Game view');
check(await page.evaluate(() => document.querySelector('meta[name=viewport]').content.includes('width=device-width')), 'the page declares a device-width viewport');
const navLabels = await page.evaluate(() => [...document.querySelectorAll('#mobile-nav .mn-btn')].map(b => b.textContent));
check(navLabels.join() === 'Game,Map,Here,Company,⚙', 'the bottom bar: ' + navLabels.join());
const small = await page.evaluate(() => [...document.querySelectorAll('#mobile-nav button, #touch-bar button, #command-input')]
  .filter(b => b.offsetParent && (b.getBoundingClientRect().height < 43 || (b.className.includes('mb-btn') && !b.className.includes('mb-fold') && b.getBoundingClientRect().width < 43)))
  .map(b => b.textContent || b.id));
check(small.length === 0, 'every touch control is at least 44 px: ' + small.join());
check(await overflow() <= 0, 'the page never scrolls sideways');
const hidden = await page.evaluate(() => [...document.querySelectorAll('#touch-bar button')]
  .filter(b => !b.hidden).filter(b => { const r = b.getBoundingClientRect(); return r.left < 0 || r.right > window.innerWidth; })
  .map(b => b.textContent));
check(hidden.length === 0, 'every touch bar button is in sight without scrolling the bar: ' + hidden.join());
const barBox = await box('#touch-bar'), navBox = await box('#mobile-nav'), inBox = await box('#input-area');
check(inBox.y + inBox.height <= barBox.y + barBox.height + 80 && navBox.y + navBox.height <= 780 + 1, 'the command box, touch bar and view bar sit at the bottom of the screen');
const termBox = await box('#terminal');
check(termBox.height > 300 && termBox.width >= VW - 10, 'the game text still has room: ' + Math.round(termBox.width) + 'x' + Math.round(termBox.height));
await shot('game');

// --- the touch bar's commands ---
await page.locator('#touch-bar button[aria-label="Go north"]').tap();
await page.locator('#touch-bar button[aria-label="Go up"]').tap();
await page.locator('#touch-bar button', { hasText: 'Look' }).tap();
await page.locator('#touch-bar button', { hasText: 'Camp' }).tap();
check((await sent()).join() === 'north,up,look,camp', 'the compass and quick commands send ordinary commands: ' + (await sent()).join());
await page.evaluate(() => Mobile.show('company'));
await page.locator('#touch-bar button', { hasText: 'Inventory' }).tap();
check(await page.evaluate(() => Mobile.view()) === 'game', 'a command that answers in text brings the Game view to the front');
await page.locator('#touch-bar .mb-fold').tap();
const foldedShown = await page.evaluate(() => [...document.querySelectorAll('#touch-bar button')].filter(b => b.offsetParent).map(b => b.title));
check(foldedShown.join() === 'Show the touch bar', 'folding the touch bar leaves only its unfold button: ' + foldedShown.join());
await page.locator('#touch-bar .mb-fold').tap();
check(await page.evaluate(() => document.querySelector('#touch-bar button[aria-label="Go north"]').offsetParent !== null), 'and unfolding brings the buttons back');
await clearSent();
await clearSent();

// --- the Map view, tap-to-walk, touch pan and pinch ---
await gmcp('World.Map', world);
await gmcp('Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
await gmcp('Company', company);
await gmcp('Room', { Info: roomInfo(2, 1) });
await page.locator('#mobile-nav button', { hasText: 'Map' }).tap();
check(await page.evaluate(() => Mobile.view()) === 'map', 'the Map button shows the map');
check((await visibleWins('#dock-left')).join() === 'Map', 'only the map shows in the Map view: ' + (await visibleWins('#dock-left')).join());
const mapBox = await box('#map-2d-canvas');
check(mapBox && mapBox.width >= VW - 10 && mapBox.height >= 250, 'the map fills the screen: ' + Math.round(mapBox.width) + 'x' + Math.round(mapBox.height));
check((await box('#terminal')).height > 0 && await page.evaluate(() => getComputedStyle(document.getElementById('terminal')).visibility) === 'hidden', 'the terminal is out of sight behind it');
await page.waitForTimeout(500);
await shot('map');
const pt = await page.evaluate(([r]) => window.MapView ? window.MapView.pointOf(r) : null, [id(3, 1)]).catch(() => null);
// The map module keeps its view private; find a room's pixel by scanning the zoom step.
const centre = { x: mapBox.x + mapBox.width / 2, y: mapBox.y + mapBox.height / 2 };
const stepPx = await page.evaluate(() => {
  const c = document.getElementById('map-2d-canvas');
  return c.getBoundingClientRect().width;
});
void pt; void stepPx;
// A tap beside the player's room (east) offers a walk there.
let offered = null;
for (const dx of [32, 40, 48, 56, 64]) {
  await page.touchscreen.tap(centre.x + dx, centre.y);
  offered = await page.evaluate(() => [...document.querySelectorAll('.ui-menu-item')].map(b => b.textContent));
  if (offered.length) { break; }
}
check(offered && offered.length > 0 && offered[0].startsWith('Walk to '), 'tapping a room offers a walk there: ' + (offered || []).join('|'));
const itemH = await page.evaluate(() => Math.min(...[...document.querySelectorAll('.ui-menu-item')].map(b => b.getBoundingClientRect().height)));
check(itemH >= 44, 'the menu rows are finger-sized: ' + Math.round(itemH));
await shot('map-tap-menu');
await page.locator('.ui-menu-item').first().tap();
const walkSent = (await sent())[0] || '';
check(/^walkto \d+$/.test(walkSent), 'picking it sends walkto: ' + walkSent);
await clearSent();

// One finger pans, two pinch to zoom.
const cdp = await ctx.newCDPSession(page);
const touch = (type, pts) => cdp.send('Input.dispatchTouchEvent', { type, touchPoints: pts.map((p, i) => ({ x: p[0], y: p[1], id: i })) });
const zoomOf = () => page.evaluate(() => { const t = [...document.querySelectorAll('canvas#map-2d-canvas')][0]; return t ? t.width : 0; });
void zoomOf;
const vp = () => page.evaluate(() => MapPlaces.viewport());
const before = await vp();
await touch('touchStart', [[centre.x, centre.y]]);
for (let i = 1; i <= 6; i++) { await touch('touchMove', [[centre.x - i * 14, centre.y - i * 6]]); }
await touch('touchEnd', []);
await page.waitForTimeout(200);
const afterPan = await vp();
check(Math.abs(afterPan.pan[0] - before.pan[0]) > 1 && afterPan.zoom === before.zoom, 'dragging with a finger pans the map: ' + JSON.stringify([before.pan, afterPan.pan]));
check(!(await page.evaluate(() => document.querySelector('.ui-menu'))), 'and a drag opens no walk menu');
const sizeBefore = await vp();
await touch('touchStart', [[centre.x - 30, centre.y], [centre.x + 30, centre.y]]);
for (let i = 1; i <= 6; i++) { await touch('touchMove', [[centre.x - 30 - i * 12, centre.y], [centre.x + 30 + i * 12, centre.y]]); }
await touch('touchEnd', []);
await page.waitForTimeout(200);
const afterPinch = await vp();
check(afterPinch.zoom > sizeBefore.zoom, 'a two-finger pinch zooms the map: ' + sizeBefore.zoom + ' to ' + afterPinch.zoom);
check(!(await page.evaluate(() => document.querySelector('.ui-menu'))), 'and a pinch opens no walk menu');

// --- the Walk to... list ---
await page.locator('#mobile-nav button', { hasText: 'Game' }).tap();
await page.locator('#touch-bar .mb-walk').tap();
const places = await page.evaluate(() => [...document.querySelectorAll('.ui-menu-item')].map(b => b.textContent));
check(places.length === 3 && places[0].startsWith('The Obelisk (obelisk)'), 'Walk to... lists the named places, nearest first: ' + places.join('|'));
check(await page.evaluate(() => { const m = document.querySelector('.ui-menu').getBoundingClientRect(); return m.left >= 0 && m.right <= window.innerWidth && m.top >= 0 && m.bottom <= window.innerHeight; }), 'the list stays on screen');
await shot('walk-to-list');
await page.locator('.ui-menu-item', { hasText: 'Bridge' }).tap();
check((await sent()).join() === 'walkto ' + id(5, 0), 'picking a place sends walkto with its room: ' + (await sent()).join());
await clearSent();
check(await page.evaluate(() => document.querySelector('#touch-bar .mb-stop').hidden), 'no Stop button when no walk is under way');
await gmcp('Walkto', { target: id(5, 0), path: [id(3, 1), id(4, 1)] });
check(!(await page.evaluate(() => document.querySelector('#touch-bar .mb-stop').hidden)), 'a Stop button shows while walking');
await page.locator('#touch-bar .mb-stop').tap();
check((await sent()).join() === 'walkto stop', 'Stop sends walkto stop');
await clearSent();
await page.locator('#touch-bar .mb-walk').tap();
check((await page.evaluate(() => document.querySelector('.ui-menu-item').textContent)) === 'Stop walking', 'the list leads with Stop walking while a walk is under way');
await page.keyboard.press('Escape');
await gmcp('Walkto', {});
check(await page.evaluate(() => document.querySelector('#touch-bar .mb-stop').hidden), 'and the Stop button goes when the walk ends');

// --- Here: Room Info with the gather progress ---
await gmcp('Room.Info', { ...roomInfo(1, 1), name: 'A Mossy Yard', exits: {}, Contents: {} });
await gmcp('Room.Gather', { phase: 'start', kind: 'herbs', label: 'gathering herbs', seconds: 20 });
await page.locator('#mobile-nav button', { hasText: 'Here' }).tap();
const hereWins = await visibleWins('#dock-left');
check(hereWins.includes('RoomInfo') && !hereWins.includes('Map'), 'the Here view shows the room, not the map: ' + hereWins.join());
check(await page.evaluate(() => getComputedStyle(document.getElementById('rw-gather')).display !== 'none'), 'the gather progress strip shows');
const gb = await box('#rw-gather');
check(gb && gb.x >= 0 && gb.x + gb.width <= VW && gb.y + gb.height <= 780, 'and fits the screen: ' + JSON.stringify(gb && { x: Math.round(gb.x), w: Math.round(gb.width) }));
check(await overflow() <= 0, 'Here never scrolls sideways');
await shot('here-gather');
await gmcp('Room.Gather', { phase: 'done', kind: 'herbs', label: 'gathering herbs', lines: ['Your company gathers 3 wild thyme.'] });

// --- Company: tabs, the Camp tab ---
await page.locator('#mobile-nav button', { hasText: 'Company' }).tap();
check((await visibleWins('#dock-right')).join() === 'group:dock', 'the Company view shows the dock');
const tabH = await page.evaluate(() => Math.min(...[...document.querySelectorAll('#dock-right .dock-tabgroup-tab')].map(t => t.getBoundingClientRect().height)));
check(tabH >= 40, 'the dock tabs are tall enough to tap: ' + Math.round(tabH));
await gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, embers: false, tent: true, gear: ['Bedrolls 2/3', 'Tent', 'Bells and trip lines'], resting: false, rested: true, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false });
await page.getByRole('tab', { name: 'Company' }).tap();
await page.getByRole('tab', { name: 'Camp' }).tap();
await page.waitForTimeout(150);
const campText = await page.evaluate(() => (document.getElementById('company-camp') || {}).textContent || '');
check(campText.includes('Camp gear: Bedrolls 2/3, Tent, Bells and trip lines.'), 'the Camp tab shows its gear line');
const camp = await box('#company-camp');
check(camp && camp.x >= 0 && camp.x + camp.width <= VW + 1, 'the Camp tab fits the screen width: ' + (camp && Math.round(camp.width)));
check(await overflow() <= 0, 'Company never scrolls sideways');
await shot('company-camp');
// Phase 48: every member's gear slots in Company > Inventory, tap-to-pick on a phone.
const sword = (ref, label, slot, type) => ({ ref, name: label, label, grams: 1500, count: 1, uses: 0, uses_max: 0, type, subtype: 'wearable', slot });
await gmcp('Company.Inventory', {
  shared: true, treasury: 40, autoloot: false, companions_known: true, containers: [], horses: [],
  load: { total_g: 9000, capacity_g: 60000, member_capacity_g: 60000, mount_capacity_g: 0, cargo_g: 3000 },
  slots: [{ slot: 'weapon', label: 'Weapon' }, { slot: 'offhand', label: 'Offhand' }, { slot: 'head', label: 'Head' }, { slot: 'body', label: 'Body' }, { slot: 'pack', label: 'Pack' }],
  members: [
    { key: 'leader', name: 'Wren', available: true, fallen: false, unrecorded: false, grams: 4000, pack: '', pack_bonus_g: 0, worn: [sword('!1:sword', 'iron sword', 'weapon', 'weapon')], carried: [] },
    { key: 'companion:1', name: 'Oswin', available: true, fallen: false, unrecorded: false, grams: 3000, pack: '', pack_bonus_g: 0, worn: [sword('!5:mace', 'oak mace', 'weapon', 'weapon'), sword('!6:cap', 'leather cap', 'head', 'head')], carried: [] },
    { key: 'companion:2', name: 'Brant', available: true, fallen: false, unrecorded: false, grams: 2000, pack: '', pack_bonus_g: 0, worn: [], carried: [] },
  ],
  cargo: [sword('!9:coat', 'quilted coat', 'body', 'body'), sword('!10:buckler', 'oak buckler', 'offhand', 'offhand')],
});
await page.getByRole('tab', { name: 'Company' }).tap();
await page.getByRole('tab', { name: 'Inventory' }).tap();
await page.waitForTimeout(150);
const gearBox = page.locator('#company-inventory [aria-label="Oswin equipment"]');
check((await gearBox.textContent()).includes('Body') && (await gearBox.textContent()).includes('empty') && (await gearBox.textContent()).includes('leather cap'), 'the phone Company view lists a companion\'s slots, empty ones too');
const slotHeights = await page.evaluate(() => [...document.querySelectorAll('#company-inventory .cmp-slot')].map(n => n.getBoundingClientRect().height));
check(slotHeights.length > 0 && slotHeights.every(h => h >= 43.5), 'every slot row is finger-sized: ' + Math.min(...slotHeights));
check(await overflow() <= 0, 'the equipment list never scrolls sideways');
await clearSent();
await gearBox.locator('button', { hasText: 'Body' }).tap();
const picks = await page.evaluate(() => [...document.querySelectorAll('.ui-menu-item')].map(b => b.textContent));
check(picks.join('|') === 'Equip quilted coat', 'tapping an empty slot offers only the cargo that fits: ' + picks.join('|'));
await shot('equipment-pick');
await page.locator('.ui-menu-item', { hasText: 'Equip quilted coat' }).tap();
check((await sent()).join() === 'company equip #1 !9:coat body', 'picking it equips that companion: ' + (await sent()).join());
await clearSent();
await gearBox.locator('button', { hasText: 'oak mace' }).tap();
check((await page.evaluate(() => [...document.querySelectorAll('.ui-menu-item')].map(b => b.textContent))).includes('Remove to cargo'), 'tapping worn gear offers to remove it');
await page.mouse.click(5, 5);
await shot('equipment');
await page.getByRole('tab', { name: 'Combat' }).tap();
check(await overflow() <= 0, 'the Combat tab never scrolls sideways');

// --- Battle screen, with tapping an allied formation ---
await gmcp('Room', { Info: roomInfo(2, 1) });
await gmcp('Company.Battle', battle);
await page.waitForTimeout(400);
check(await page.evaluate(() => BattleScreen.state().open), 'the battle screen opens by itself on a phone');
const bs = await box('#battle-screen');
check(bs && bs.x === 0 && bs.width === VW && bs.height >= 700, 'it takes the whole screen: ' + JSON.stringify(bs && { w: bs.width, h: Math.round(bs.height) }));
const cv = await box('#battle-screen canvas');
check(cv && cv.width <= VW && cv.width >= VW - 30, 'the picture fits the width: ' + Math.round(cv.width));
const foot = await page.evaluate(() => [...document.querySelectorAll('#battle-screen .bs-foot button')].map(b => {
  const r = b.getBoundingClientRect();
  return { t: b.textContent, h: r.height, inside: r.left >= 0 && r.right <= window.innerWidth && r.bottom <= window.innerHeight };
}));
check(foot.length >= 5 && foot.every(b => b.h >= 44 && b.inside), 'Retreat, the focus buttons and Help are finger-sized and on screen: ' + foot.map(b => b.t + ':' + Math.round(b.h)).join(' '));
await shot('battle');
// On a phone an allied company is a pennant at the top of the picture; a tap on it watches that company.
await page.evaluate(() => { document.getElementById('connect-button').style.display = 'none'; });
check(await page.evaluate(() => BattleScreen.state().allies.list.length) === 1, 'the allied company is a pennant on a phone');
const cb = await box('#battle-screen canvas');
await page.touchscreen.tap(cb.x + 134 * cb.width / 320, cb.y + 31 * cb.height / 180);
await page.waitForTimeout(250);
check(await page.evaluate(() => BattleScreen.state().watching) === 0, 'tapping the allied pennant watches that company');
await page.waitForTimeout(300);
await shot('battle-watching');
check(await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent.includes('Watching')), 'the caption says whom you are watching');
await page.locator('#battle-screen .bs-foot button', { hasText: 'Retreat' }).tap();
check((await sent()).includes('retreat'), 'Retreat is one tap, and still works while watching');
await page.locator('#battle-screen .bs-foot button', { hasText: 'weakest' }).first().tap().catch(() => {});
await clearSent();
await page.locator('#battle-screen .bs-head button', { hasText: 'Minimise' }).tap();
const badge = await box('#battle-badge');
const barTop = (await box('#touch-bar')).y;
check(badge && badge.y + badge.height <= barTop && badge.y > 0, 'minimised, the badge sits clear of the bottom bars: ' + Math.round(badge.y + badge.height) + ' <= ' + Math.round(barTop));
await shot('battle-badge');
await page.locator('#battle-badge').tap();
await page.evaluate(() => Mobile.show('map'));
await page.locator('#battle-screen .bs-foot button', { hasText: 'Help' }).tap();
check((await sent()).includes('help battlescreen') && await page.evaluate(() => Mobile.view() === 'game' && getComputedStyle(document.getElementById('battle-screen')).display === 'none'),
  'Help steps aside to the Game view, where its text lands');
check(await page.evaluate(() => document.querySelector('#battle-screen .bs-focus-label').textContent) === 'Focus:', 'the focus buttons are named on a phone, where there is no hover');
await gmcp('Company.Battle', {});

// --- the manifest and icons ---
const man = await page.evaluate(async () => {
  const href = document.querySelector('link[rel=manifest]').href;
  const m = await (await fetch(href)).json();
  const icons = await Promise.all(m.icons.map(async i => ({ src: i.src, purpose: i.purpose, status: (await fetch(new URL(i.src, href))).status })));
  return { name: m.name, display: m.display, start: m.start_url, icons };
});
check(man.name === 'Ashveil' && man.display === 'standalone' && man.start === '/webclient-pure.html', 'the manifest names the app, standalone, starting at the client: ' + JSON.stringify([man.name, man.display, man.start]));
check(man.icons.length === 3 && man.icons.every(i => i.status === 200) && man.icons.some(i => i.purpose === 'maskable'), 'its 192, 512 and maskable icons load');

// --- back to a wide screen ---
await page.setViewportSize({ width: 1280, height: 800 });
await page.waitForTimeout(300);
check(await page.evaluate(() => !Mobile.active() && !document.body.classList.contains('mobile')), 'a wide screen gets the desktop layout back');
check(await page.evaluate(() => getComputedStyle(document.getElementById('mobile-nav')).display === 'none' && getComputedStyle(document.getElementById('touch-bar')).display === 'none'), 'the bottom bar and touch bar are gone');
const docks = await page.evaluate(() => ['dock-left', 'dock-right'].map(i => { const e = document.getElementById(i); const r = e.getBoundingClientRect(); return { w: Math.round(r.width), pos: getComputedStyle(e).position }; }));
check(docks.every(d => d.w > 100 && d.w < 600 && d.pos !== 'absolute'), 'the docks sit beside the terminal again: ' + JSON.stringify(docks));
check(await page.evaluate(() => getComputedStyle(document.getElementById('terminal')).visibility) === 'visible', 'and the terminal shows');

await browser.close();
server.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all mobile checks passed');
