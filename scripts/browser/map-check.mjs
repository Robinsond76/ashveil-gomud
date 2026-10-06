// Phase 40b and 40c browser check: drives the map window (window-map.js and the
// shared sprite loader, through map-harness.html with the real web client
// core) in Chromium with Playwright: the class sprite, its facing and walk
// queue, the company badge, camp markers and the settings (40b); terrain
// tiles, walls, fog, landmarks, companions and the classic style (40c).
//
//   NODE_PATH=$(npm root -g) node scripts/browser/map-check.mjs [screenshot.png]
import { createRequire } from 'node:module';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const shot = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

// The sprite manifest is fetched, so serve the repository over HTTP.
const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json' };
const server = http.createServer((req, res) => {
  if (req.url === '/favicon.ico') { res.writeHead(204); res.end(); return; }
  const file = path.join(root, decodeURIComponent(req.url.split('?')[0]));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const url = 'http://127.0.0.1:' + server.address().port + '/scripts/browser/map-harness.html';

// A 5 x 3 field of rooms, ids 1 + x + 5*y at grid (x, y), z 0, in one zone.
const W = 5, H = 3;
const id = (x, y) => 1 + x + W * y;
function roomInfo(x, y, z = 0, zone = 'Test') {
  const exitsv2 = {};
  const add = (name, dx, dy) => {
    const nx = x + dx, ny = y + dy;
    if (z === 0 && nx >= 0 && nx < W && ny >= 0 && ny < H) { exitsv2[name] = { num: id(nx, ny), dx, dy, dz: 0 }; }
  };
  add('north', 0, -1); add('south', 0, 1); add('east', 1, 0); add('west', -1, 0);
  return { num: z === 0 ? id(x, y) : 100 + id(x, y), area: zone, coords: [zone, x, y, z].join(','), environment: 'forest', exitsv2, details: [] };
}
const world = { biomes: { forest: { color: { bg: '#2f4f2f' } } }, rooms: [] };
for (let y = 0; y < H; y++) { for (let x = 0; x < W; x++) { world.rooms.push(roomInfo(x, y)); } }

const member = (key, name, status) => ({ key, id: 0, name, status, level: 5, archetype: 'Warrior', cell: null, chemistry: null, strategy: null });
const company = {
  leader: member('leader', 'Wren', 'present'),
  members: [member('companion:1', 'Oswin', 'present'), member('companion:2', 'Brant', 'present'),
            member('companion:3', 'Ysolde', 'dead'), member('companion:4', 'Tamsin', 'separated')],
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
async function open(viewport, settings, abort) {
  const page = await browser.newPage({ viewport });
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  page.on('console', m => {
    if (m.type() !== 'error') { return; }
    if (abort && /Failed to load resource/.test(m.text())) { return; }
    failures++; console.log('FAIL console error: ' + m.text());
  });
  if (abort) { await page.route(abort, r => r.abort()); }
  await page.goto(url);
  await page.evaluate(s => { localStorage.clear(); if (s) { localStorage.setItem('gomud_map_settings', JSON.stringify(s)); } }, settings || null);
  await page.reload();
  return page;
}
const gmcp = (page, ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
const state = page => page.evaluate(() => window.MapView.state());
const settle = page => page.waitForFunction(() => window.MapView.state().spriteDrawn && !window.MapView.state().walking, null, { timeout: 5000 });
const moveTo = (page, x, y) => gmcp(page, 'Room', { Info: roomInfo(x, y) });
const tick = (page, ms = 300) => page.waitForTimeout(ms);

async function start(page) {
  await gmcp(page, 'World.Map', world);
  await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
  await moveTo(page, 2, 1);
  await settle(page);
}
// The canvas colour at a room's centre pixel, to see what a layer drew.
const pixelAt = (page, gx, gy) => page.evaluate(([x, y]) => {
  const c = document.getElementById('map-2d-canvas');
  const ctx = c.getContext('2d');
  // the room centre is the canvas centre once the camera has eased to the player
  return Array.from(ctx.getImageData(Math.round(c.width / 2 + x), Math.round(c.height / 2 + y), 1, 1).data);
}, [gx, gy]);

// --- Sprite, fallback chain, facing ---
let page = await open({ width: 1280, height: 900 });
await gmcp(page, 'World.Map', world);
await moveTo(page, 2, 1);
await settle(page);
let s = await state(page);
check(s.spriteDrawn && s.keys.join() === 'adventurer', 'before a class is known the adventurer sprite stands in');
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'knight', lineage: 'warrior' });
await settle(page);
s = await state(page);
check(s.keys.join() === 'knight,warrior,adventurer' && s.spriteDrawn, 'class, then lineage, then adventurer; an undrawn class falls to its lineage');
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
check((await state(page)).keys.join() === 'warrior,adventurer', 'an unpromoted class is its lineage');

await moveTo(page, 2, 0); await tick(page, 60);
s = await state(page);
check(s.face === 'up' && s.walking, 'walking north faces up and walks');
await settle(page);
await moveTo(page, 3, 0); await tick(page, 60);
s = await state(page);
check(s.face === 'side' && !s.flip, 'walking east faces side, unmirrored');
await settle(page);
await moveTo(page, 2, 0); await tick(page, 60);
s = await state(page);
check(s.face === 'side' && s.flip, 'walking west faces side, mirrored');
await settle(page);
await moveTo(page, 2, 1); await tick(page, 60);
check((await state(page)).face === 'down', 'walking south faces down');
await settle(page);
await moveTo(page, 3, 2); await tick(page, 60);
check((await state(page)).face === 'side', 'a diagonal uses the side view');
await settle(page);

// queue: a few quick moves play in order, never more than 2 behind
await moveTo(page, 2, 2); await moveTo(page, 1, 2); await moveTo(page, 0, 2);
s = await state(page);
check(s.queued <= 2 && s.queued >= 1, 'quick moves queue (at most 2 steps behind): ' + s.queued);
for (const [x, y] of [[0, 1], [0, 0], [1, 0], [2, 0], [3, 0], [4, 0]]) {
  await moveTo(page, x, y);
  check((await state(page)).queued <= 2, 'never more than 2 steps behind (' + x + ',' + y + ')');
}
await settle(page);
s = await state(page);
check(s.unit.x === 4 && s.unit.y === 0, 'a pile-up ends at the newest room');
await moveTo(page, 4, 2);
s = await state(page);
check(s.queued === 0 && s.unit.x === 4 && !s.walking, 'a jump of more than a tile (recall, teleport) does not walk');

// z change: snap and fade, facing kept
await gmcp(page, 'Room', { Info: roomInfo(4, 2, 1) });
s = await state(page);
check(s.fading && s.unit.x === 4, 'changing level snaps and fades the sprite in');
await page.close();

// --- Badge ---
page = await open({ width: 1280, height: 900 });
await start(page);
check((await state(page)).companySize === 0, 'no company: no badge');
await gmcp(page, 'Company', company);
s = await state(page);
check(s.companySize === 3, 'badge counts the members with the leader (not dead, not separated): ' + s.companySize);
await gmcp(page, 'Company', { leader: company.leader, members: [member('companion:3', 'Ysolde', 'dead')] });
check((await state(page)).companySize === 0, 'a leader travelling alone: the badge is hidden');
await gmcp(page, 'Company', company);

// --- Camps ---
const tile = await pixelAt(page, 0, 0);
await gmcp(page, 'Company.Camp', { has_camp: true, here: false, room: 'x', room_id: id(3, 1), fire_lit: false, resting: false,
                                   can_camp: false, inn: false, allied_camps: [] });
await tick(page, 400);
s = await state(page);
check(s.camp && s.camp.room_id === id(3, 1), 'the camp payload is kept');
const before = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await gmcp(page, 'Company.Camp', { has_camp: true, here: false, room: 'x', room_id: id(3, 1), fire_lit: true, resting: true,
                                   can_camp: false, inn: false, allied_camps: [{ room_id: id(0, 0), leader: 'Ally', fire_lit: true, resting: false }] });
await tick(page, 400);
const lit = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
check(before !== lit, 'lighting the fire (and an allied camp) changes what is drawn');
const f1 = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await tick(page, 400);
const f2 = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
check(f1 !== f2, 'the lit fire and smoke animate');
await gmcp(page, 'Company.Camp', { has_camp: false, here: false, room: '', room_id: 0, fire_lit: false, resting: false,
                                   can_camp: true, inn: false, allied_camps: [] });
await tick(page, 400);
const gone = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
check(gone === before ? true : gone !== lit, 'breaking the camp removes the marker');
const noCamp = await page.evaluate(() => { window.MapView.state(); return document.getElementById('map-2d-canvas').toDataURL(); });
await gmcp(page, 'Company.Camp', { has_camp: true, here: false, room: 'x', room_id: id(3, 1), fire_lit: false, allied_camps: [] });
await tick(page, 300);
check(noCamp !== await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL()), 'a camp draws its tent');
// A camp on your own tile is pitched beside your sprite, not hidden under it.
const leftOfYou = () => page.evaluate(() => {
  const c = document.getElementById('map-2d-canvas');
  return Array.from(c.getContext('2d').getImageData(Math.round(c.width / 2) - 40, Math.round(c.height / 2) - 40, 24, 40).data).join();
});
const yourTile = (await state(page)).unit;
await gmcp(page, 'Company.Camp', { has_camp: false, here: false, room: '', room_id: 0, fire_lit: false, allied_camps: [] });
await tick(page, 300);
const bare = await leftOfYou();
await gmcp(page, 'Company.Camp', { has_camp: true, here: true, room: 'x', room_id: id(yourTile.x, yourTile.y), fire_lit: false, allied_camps: [] });
await tick(page, 300);
check(bare !== await leftOfYou(), 'a camp on your own tile shows beside your sprite');

// --- Allies ---
await gmcp(page, 'Party.Vitals', {
  Cleric: { mapx: 1, mapy: 1, mapz: 0, hascoordinates: true, aggro: false, lineage: 'cleric', classid: 'cleric' },
  Plain: { mapx: 3, mapy: 1, mapz: 0, hascoordinates: true, aggro: false },
});
await page.waitForFunction(() => window.MapView.state().allies.some(a => a.name === 'Cleric' && a.sprite), null, { timeout: 5000 }).catch(() => {});
s = await state(page);
const cleric = s.allies.find(a => a.name === 'Cleric'), plain = s.allies.find(a => a.name === 'Plain');
check(cleric && cleric.sprite, 'a party member with a class shows as that class sprite');
check(plain && !plain.sprite, 'a member with no class stays a heart');
await gmcp(page, 'Company.Camp', { has_camp: true, here: false, room: 'x', room_id: id(3, 1), fire_lit: true, resting: true, allied_camps: [{ room_id: id(1, 2), leader: 'Ally', fire_lit: true, resting: false }] });
await tick(page, 400);
if (shot) { await page.locator('#map-window').screenshot({ path: shot }); }
await page.close();

// --- Sprites off: the classic square ---
page = await open({ width: 1280, height: 900 }, { sprites: false, style: 'classic' });
await start(page).catch(() => {});
await moveTo(page, 2, 1); await tick(page, 400);
s = await state(page);
check(!s.spriteDrawn, 'with sprites off nothing is drawn as a sprite');
const px = await pixelAt(page, 0, 0);
check(px[0] > 150 && px[1] < 40 && px[2] < 40, 'with sprites off the current room is the classic red square: ' + px);
await page.close();

// --- A missing image falls back down the chain, without errors ---
page = await open({ width: 1280, height: 900 }, null, '**/map/units/warrior/*.png');
await gmcp(page, 'World.Map', world);
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
await moveTo(page, 2, 1);
await page.waitForFunction(() => window.MapView.state().spriteDrawn, null, { timeout: 5000 });
check((await state(page)).spriteDrawn, 'a missing warrior image falls back to the adventurer sprite');
await page.close();


// --- Phase 40c: terrain tiles, walls, fog, landmarks, companions ---
// (0,0) is the grid origin the map treats as no room, so it is left out: 7 rooms.
// Row 0: forest - water - shore - city(Inn, east exit to an unvisited room);
// row 1: forest - road - city(Bank) - city (an unmarked room under the shore).
// (2,0) and (2,1) touch with no exit between them: a wall.
const mixed = { biomes: {}, rooms: [] };
const mid = (x, y) => 200 + x + 4 * y;
const layout = [
  ['forest', 'water', 'shore', 'city'],
  ['forest', 'road', 'city', 'city'],
];
const noExit = new Set([mid(2, 0) + '-' + mid(2, 1), mid(2, 1) + '-' + mid(2, 0)]);
for (let y = 0; y < 2; y++) {
  for (let x = 0; x < 4; x++) {
    const exitsv2 = {};
    [['east', 1, 0], ['west', -1, 0], ['south', 0, 1], ['north', 0, -1]].forEach(([n, dx, dy]) => {
      const nx = x + dx, ny = y + dy;
      if (nx < 0 || nx > 3 || ny < 0 || ny > 1 || noExit.has(mid(x, y) + '-' + mid(nx, ny))) { return; }
      exitsv2[n] = { num: mid(nx, ny), dx, dy, dz: 0 };
    });
    if (x === 3 && y === 0) { exitsv2.east = { num: 999, dx: 1, dy: 0, dz: 0 }; }
    const r = { num: mid(x, y), area: 'Mixed', coords: ['Mixed', x, y, 0].join(','), environment: layout[y][x], exitsv2, details: [] };
    if (x === 3 && y === 0) { r.maplegend = 'Inn'; r.mapsymbol = 'I'; }
    if (x === 2 && y === 1) { r.maplegend = 'Bank'; r.mapsymbol = '$'; }
    if (x === 2 && y === 0) { r.maplegend = 'Shore'; r.mapsymbol = '~'; }
    if (x === 0 && y === 1) { r.mapsymbol = '%'; }   // no landmark: the letter stays
    if (x === 3 && y === 1) { r.environment = 'moonbog'; } // no art for this biome: the unknown tile
    mixed.rooms.push(r);
  }
}
const gotoMixed = async (settings, abort) => {
  const pg = await open({ width: 1280, height: 900 }, settings, abort);
  await gmcp(pg, 'World.Map', mixed);
  await gmcp(pg, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
  await gmcp(pg, 'Room', { Info: mixed.rooms[5] });
  await settle(pg);
  await pg.waitForFunction(() => { const d = window.MapView.state().drawn; return d.tiles === 7 && d.landmarks >= 2; }, null, { timeout: 5000 }).catch(() => {});
  return pg;
};

page = await gotoMixed(null);
s = await state(page);
check(s.style === 'tiles', 'the tiles style is the default');
check(s.drawn.tiles === 7 && s.drawn.fallbacks === 0, 'every room draws its biome tile (the unknown biome too): ' + JSON.stringify(s.drawn));
check(s.drawn.walls === 1, 'two touching rooms with no exit between them show one wall edge: ' + s.drawn.walls);
check(s.drawn.fog === 1, 'the unvisited exit ends in a fog tile: ' + s.drawn.fog);
check(s.drawn.landmarks === 2 && s.drawn.glyphs === 1, 'Inn and Bank draw landmarks, an unmapped symbol keeps its letter, the shore draws none: ' + JSON.stringify(s.drawn));
check(s.zoom === 1, 'the tile map starts at a crisp 1x zoom');
await page.locator('#map-window').screenshot({ path: shot ? shot.replace(/\.png$/, '-tiles.png') : '/tmp/40c-tiles.png' });
// zoom stays on crisp steps
await tick(page, 400); // the camera ease from the last move has finished
const box = await page.locator('#map-2d-canvas').boundingBox();
await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
await page.mouse.wheel(0, -120);
await tick(page, 200);
const z1 = (await state(page)).zoom;
check([0.5, 0.75, 1, 1.5, 2, 3, 4].includes(z1) && z1 > 1, 'a wheel turn moves one crisp zoom step: ' + z1);
// the classic style is today's look
await page.evaluate(() => { const c = JSON.parse(localStorage.getItem('gomud_map_settings') || '{}'); c.style = 'classic'; localStorage.setItem('gomud_map_settings', JSON.stringify(c)); });
await page.reload();
await gmcp(page, 'World.Map', mixed);
await gmcp(page, 'Room', { Info: mixed.rooms[5] });
await tick(page, 400);
s = await state(page);
check(s.style === 'classic' && s.drawn.tiles === 0 && s.drawn.walls === 0 && s.drawn.fog === 0, 'classic draws no tiles, walls or fog: ' + JSON.stringify(s.drawn));
await page.close();

// A missing terrain image falls back to the colour square, without a page error.
page = await open({ width: 1280, height: 900 }, null, '**/map/terrain/forest.png');
await gmcp(page, 'World.Map', mixed);
await gmcp(page, 'Room', { Info: mixed.rooms[5] });
await tick(page, 500);
s = await state(page);
check(s.drawn.fallbacks === 1 && s.drawn.tiles === 6, 'a missing forest tile falls back to the colour square on that room only: ' + JSON.stringify(s.drawn));
await page.close();

// Reduced motion keeps the water still.
page = await open({ width: 1280, height: 900 });
await page.emulateMedia({ reducedMotion: 'reduce' });
await gmcp(page, 'World.Map', mixed);
await gmcp(page, 'Room', { Info: mixed.rooms[5] });
await tick(page, 500);
s = await state(page);
check(s.drawn.tiles === 7 && s.drawn.animated === 0, 'with reduced motion no tile animates: ' + JSON.stringify(s.drawn));
await page.close();

// Animated water cycles its frames.
page = await gotoMixed(null);
s = await state(page);
check(s.drawn.animated >= 1, 'water and shore animate: ' + s.drawn.animated);
const w1 = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await tick(page, 600);
const w2 = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
check(w1 !== w2, 'the animated tiles change over time');

// Companions stand beside you, by class; the badge still counts everyone.
const cmember = (key, status, lineage, classid) => ({ key, id: 1, name: key, status, level: 5, archetype: 'x', lineage, class: classid, cell: null, chemistry: null, strategy: null });
await gmcp(page, 'Company', { leader: member('leader', 'Wren', 'present'), members: [
  cmember('companion:1', 'present', 'cleric', 'cleric'), cmember('companion:2', 'present', 'ranger', ''),
  cmember('companion:3', 'dead', 'rogue', ''), cmember('companion:4', 'present', '', '') ] });
await tick(page, 300);
s = await state(page);
check(s.companions.length === 2 && s.companions.every(c => c.sprite), 'present companions with a class draw as sprites (not the dead, not the unknown): ' + JSON.stringify(s.companions));
check(s.companySize === 4, 'the badge counts all present members and the leader: ' + s.companySize);
for (let i = 0; i < 2; i++) { await page.locator('.map-controls button[title="Zoom in"]').click(); }
await tick(page, 300);
await page.locator('#map-window').screenshot({ path: shot ? shot.replace(/\.png$/, '-companions.png') : '/tmp/40c-companions.png' });

// Regrowth: a World.Resources update repaints a room you are not in.
await gmcp(page, 'World.Map', { biomes: {}, rooms: mixed.rooms.map(r => r.num === mid(1, 1) ? Object.assign({}, r, { resources: ['herbs'], depleted: ['herbs'] }) : r) });
await tick(page, 200);
const picked = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await gmcp(page, 'World.Resources', { num: mid(1, 1), resources: ['herbs'], depleted: [] });
await tick(page, 200);
const grown = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
check(picked !== grown, 'a regrown resource redraws a room you are not standing in');
await page.close();


// Phase 40c: a fire burned to embers glows low (no smoke), a camp without a tent shows the rough camp.
page = await gotoMixed(null);
const campView = () => page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
const camp = (o) => Object.assign({ has_camp: true, here: false, room: 'x', room_id: mid(3, 1), fire_lit: false, resting: false, allied_camps: [] }, o);
await gmcp(page, 'Company.Camp', camp({ tent: true, embers: false }));
await tick(page, 300);
const cold = await campView();
await gmcp(page, 'Company.Camp', camp({ tent: true, embers: true }));
await tick(page, 300);
const glowing = await campView();
check(cold !== glowing, 'a camp burned down to embers draws differently from a cold fire pit');
await gmcp(page, 'Company.Camp', camp({ tent: false, embers: true }));
await tick(page, 300);
check(glowing !== await campView(), 'a camp pitched without a tent draws the rough camp, not the tent');
await page.close();

await browser.close();
server.close();
console.log(failures ? failures + ' check(s) failed' : 'all checks passed');
process.exit(failures ? 1 : 0);
