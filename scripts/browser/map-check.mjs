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

// Phase 72a: the player's skin and hair colours (Char.Info) repaint 1x
// palette art. Every class now draws imported high-density art (E1), which
// the palette swap cannot repaint, so the old 1x warrior sheets
// (fixtures/warrior-1x-*.png, which carry the skin and hair ramps) are served
// under a test-only class id to test the repaint on a page of their own.
{
  const fixturePage = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  fixturePage.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  const fixMeta = (frames, ms) => ({ frames, frame_ms: ms, kind: 'map-unit', set: 'S1', frame: [32, 32], rows: ['down', 'up', 'side'],
    anchor: 'bottom-center', feet_baseline: 30, size: [32 * frames, 96] });
  await fixturePage.route('**/sprites/manifest.json', async r => {
    const m = JSON.parse(fs.readFileSync(path.join(root, '_datafiles/html/public/static/sprites/manifest.json'), 'utf8'));
    m.files['map/units/tint-fixture/idle.png'] = fixMeta(2, 500);
    m.files['map/units/tint-fixture/walk.png'] = fixMeta(4, 120);
    await r.fulfill({ contentType: 'application/json', body: JSON.stringify(m) });
  });
  for (const f of ['idle', 'walk']) {
    await fixturePage.route('**/sprites/map/units/tint-fixture/' + f + '.png', r =>
      r.fulfill({ contentType: 'image/png', body: fs.readFileSync(path.join(here, 'fixtures', 'warrior-1x-' + f + '.png')) }));
  }
  await fixturePage.goto(url);
  await fixturePage.evaluate(() => localStorage.clear());
  await fixturePage.reload();
  await gmcp(fixturePage, 'World.Map', world);
  await gmcp(fixturePage, 'Char.Info', { name: 'Wren', classid: 'tint-fixture', lineage: '' });
  await moveTo(fixturePage, 2, 1);
  await settle(fixturePage);
  // The repaint is checked on the sheet itself (the drawn frame changes with
  // the idle animation, so canvas snapshots are not compared).
  const skinPixels = look => fixturePage.evaluate(lk => {
    const res = window.Sprites.tinted('map/units/tint-fixture/idle.png', lk ? window.SpriteTint.look(lk[0], lk[1]) : null);
    const c = document.createElement('canvas'); c.width = res.img.width; c.height = res.img.height;
    const x = c.getContext('2d'); x.drawImage(res.img, 0, 0);
    const d = x.getImageData(0, 0, c.width, c.height).data;
    let n = 0;
    for (let i = 0; i < d.length; i += 4) { if (d[i + 3] && d[i] === 0xb5 && d[i + 1] === 0x7d && d[i + 2] === 0x5a) { n++; } }
    return n;
  }, look);
  s = await state(fixturePage);
  check(s.art && s.art.density === 1 && !s.art.tinted, 'the 1x fixture draws untinted without a look');
  check(await skinPixels(null) > 0, 'the 1x fixture carries the palette skin colour');
  await gmcp(fixturePage, 'Char.Info', { name: 'Wren', classid: 'tint-fixture', lineage: '', skin: '#5a3a28', hair: '#d8b868' });
  await fixturePage.waitForTimeout(150);
  s = await state(fixturePage);
  check(s.look && s.look.skin === '#5a3a28' && s.look.hair === '#d8b868', 'Char.Info skin and hair become the figure\'s look');
  check(s.art.tinted && await skinPixels(['#5a3a28', '#d8b868']) === 0, 'the look repaints the 1x figure\'s skin');
  await gmcp(fixturePage, 'Char.Info', { name: 'Wren', classid: 'tint-fixture', lineage: '', skin: 'red', hair: '' });
  check((await state(fixturePage)).look === null, 'a colour that is not #rrggbb is ignored');
  await fixturePage.close();
}
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
await settle(page); await page.waitForTimeout(150);
s = await state(page);
check(s.art && s.art.density === 4 && s.art.idleFrames === 2 && s.art.walkFrames === 6, 'the warrior draws its imported density-4 sheets (2 idle, 6 walk frames)');
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior', skin: '#5a3a28', hair: '#d8b868' });
await page.waitForTimeout(150);
s = await state(page);
check(s.look && s.art && s.art.density === 4 && !s.art.tinted, 'high-density art is drawn untinted (skin and hair masks come later)');
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });

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

// --- E2: road and coast pieces, replace-style animation ---
// The mixed harness: the road at (1,1) has no road neighbour (road-none), the
// shore at (2,0) has water to the west (shore-w), and the water draws its
// 4-frame replace sheet.
page = await gotoMixed(null);
s = await state(page);
check(s.drawn.pieces[mid(1, 1)] === 'road-none' && s.drawn.pieces[mid(2, 0)] === 'shore-w',
  'the road and shore draw the pieces their neighbours pick: ' + JSON.stringify(s.drawn.pieces));
check(Object.keys(s.drawn.pieces).length === 2, 'only road and shore rooms draw pieces');
await page.close();

// A road network: (1,1) joins the road north of it; a wall (no exit) to the
// east and a found secret passage to the west don't join; (2,1) and (1,0)
// run on into the fog of unvisited exits; the shore at (1,2) has water on
// three sides and the road to its north.
const roads = { biomes: {}, rooms: [] };
const rid = (x, y) => 300 + x + 4 * y;
const rlayout = { '1,0': 'road', '0,1': 'road', '1,1': 'road', '2,1': 'road', '0,2': 'water', '1,2': 'shore', '2,2': 'water', '1,3': 'water' };
const rexits = [['1,1', '1,0', ''], ['1,1', '0,1', 'secret'], ['1,1', '1,2', ''], ['0,2', '1,2', ''], ['1,2', '2,2', ''], ['1,2', '1,3', '']];
const names = { '0,-1': 'north', '1,0': 'east', '0,1': 'south', '-1,0': 'west' };
for (const [key, env] of Object.entries(rlayout)) {
  const [x, y] = key.split(',').map(Number);
  const exitsv2 = {};
  rexits.forEach(([a, b, flag]) => {
    const other = a === key ? b : (b === key ? a : null);
    if (!other) { return; }
    const [ox, oy] = other.split(',').map(Number);
    exitsv2[names[(ox - x) + ',' + (oy - y)]] = { num: rid(ox, oy), dx: ox - x, dy: oy - y, dz: 0, details: flag ? [flag] : [] };
  });
  if (key === '2,1') { exitsv2.east = { num: 997, dx: 1, dy: 0, dz: 0 }; }
  if (key === '1,0') { exitsv2.north = { num: 998, dx: 0, dy: -1, dz: 0 }; }
  roads.rooms.push({ num: rid(x, y), area: 'Roads', coords: ['Roads', x, y, 0].join(','), environment: env, exitsv2, details: [] });
}
page = await open({ width: 1280, height: 900 });
await gmcp(page, 'World.Map', roads);
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior' });
await gmcp(page, 'Room', { Info: roads.rooms.find(r => r.num === rid(1, 1)) });
await page.waitForFunction(() => Object.keys(window.MapView.state().drawn.pieces).length === 5, null, { timeout: 5000 }).catch(() => {});
s = await state(page);
const want = { [rid(1, 1)]: 'road-n', [rid(0, 1)]: 'road-none', [rid(2, 1)]: 'road-e', [rid(1, 0)]: 'road-ns', [rid(1, 2)]: 'shore-esw' };
check(JSON.stringify(s.drawn.pieces) === JSON.stringify(want),
  'roads join through exits only (not a wall or a secret passage) and run on into the fog; the coast faces its water: ' + JSON.stringify(s.drawn.pieces));
for (let i = 0; i < 3; i++) { await page.locator('.map-controls button[title="Zoom in"]').click(); }
await tick(page, 400);
await page.locator('#map-window').screenshot({ path: shot ? shot.replace(/\.png$/, '-roads.png') : '/tmp/e2-roads.png' });
await page.close();

// A missing piece falls back to the biome's own tile.
page = await open({ width: 1280, height: 900 }, null, '**/map/terrain/road-none.png');
await gmcp(page, 'World.Map', mixed);
await gmcp(page, 'Room', { Info: mixed.rooms[5] });
await tick(page, 500);
s = await state(page);
check(s.drawn.tiles === 7 && s.drawn.fallbacks === 0 && !(mid(1, 1) in s.drawn.pieces),
  'a missing road piece draws the road biome tile instead: ' + JSON.stringify(s.drawn));
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
check(s.drawn.animated === 1, 'the water animates (the shore, now a coast piece, does not): ' + s.drawn.animated);
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


// --- Phase 40d: resource icons, the walk path, click-to-walk, day and night ---
const withRes = (resByRoom, biomes) => ({ biomes: biomes || {}, rooms: mixed.rooms.map(r => resByRoom[r.num] ? Object.assign({}, r, resByRoom[r.num]) : r) });

// Resource icons replace the dots in the tiles style, and keep them in the classic one.
page = await gotoMixed(null);
await gmcp(page, 'World.Map', withRes({ [mid(1, 1)]: { resources: ['water', 'forage', 'herbs', 'game'] }, [mid(0, 1)]: { resources: ['fishing'], depleted: ['fishing'] } }));
await tick(page, 400);
s = await state(page);
check(s.drawn.icons === 4 && s.drawn.dots === 0, 'the tiles style draws resource icons (3 in a corner plus the lone one), no dots: ' + JSON.stringify(s.drawn));
const iconsOn = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await page.close();
page = await gotoMixed({ style: 'classic' });
await gmcp(page, 'World.Map', withRes({ [mid(1, 1)]: { resources: ['water', 'forage'] } }));
await tick(page, 300);
s = await state(page);
check(s.drawn.icons === 0 && s.drawn.dots === 2, 'the classic style keeps the coloured dots: ' + JSON.stringify(s.drawn));
await page.close();
page = await open({ width: 1280, height: 900 }, null, '**/map/resources/*.png');
await gmcp(page, 'Room', { Info: mixed.rooms[5] });
await gmcp(page, 'World.Map', withRes({ [mid(0, 1)]: { resources: ['water'] } }));
await tick(page, 500);
s = await state(page);
check(s.drawn.icons === 0 && s.drawn.dots === 1, 'an icon that fails to load falls back to its dot: ' + JSON.stringify(s.drawn));
await page.close();

// The walk path: breadcrumbs ahead, the flag on the target, cleared by {}.
page = await gotoMixed(null);
await gmcp(page, 'Walkto', { target: mid(3, 0), path: [mid(2, 1), mid(3, 1), mid(3, 0)] });
await tick(page, 400);
s = await state(page);
check(s.drawn.path === 3 && s.walk && s.walk.target === mid(3, 0), 'a walk draws a marker on each room ahead: ' + JSON.stringify(s.drawn));
const withPath = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await gmcp(page, 'Walkto', { target: mid(3, 0), path: [mid(3, 0)] });
await tick(page, 300);
s = await state(page);
check(s.drawn.path === 1, 'the path shortens as the walker advances: ' + s.drawn.path);
await gmcp(page, 'Walkto', {});
await tick(page, 300);
s = await state(page);
check(s.drawn.path === 0 && s.walk === null, 'an empty Walkto clears the path');
check(withPath !== await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL()), 'clearing the path repaints the map');
await page.close();

// Click-to-walk: a click on another room offers the walk, one pick sends it.
page = await gotoMixed(null);
await page.evaluate(() => { window.sent.length = 0; });
s = await state(page);
const at = (id) => page.evaluate(i => window.MapView.pointOf(i), id);
await page.mouse.click((await at(mid(2, 1))).x, (await at(mid(2, 1))).y);   // the room east of you
let items = await page.locator('[role=menuitem]').allTextContents();
check(items.length === 1 && /^Walk to /.test(items[0]), 'a click on another room offers one pick, to walk there: ' + JSON.stringify(items));
await page.locator('[role=menuitem]').first().click();
check((await page.evaluate(() => window.sent)).join('|') === 'walkto ' + mid(2, 1), 'the pick sends walkto <room>: ' + (await page.evaluate(() => window.sent)).join('|'));
await page.mouse.click((await at(mid(1, 1))).x, (await at(mid(1, 1))).y);   // your own room
check(await page.locator('[role=menuitem]').count() === 0, 'a click on your own room offers nothing');
await gmcp(page, 'Walkto', { target: mid(3, 1), path: [mid(2, 1), mid(3, 1)] });
await page.mouse.click((await at(mid(2, 1))).x, (await at(mid(2, 1))).y);
items = await page.locator('[role=menuitem]').allTextContents();
check(items.length === 2 && items[1] === 'Stop walking', 'while walking the menu also offers Stop walking: ' + JSON.stringify(items));
await page.keyboard.press('Escape');
await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'warrior', lineage: 'warrior', role: 'admin' });
await page.mouse.click((await at(mid(2, 1))).x, (await at(mid(2, 1))).y);
items = await page.locator('[role=menuitem]').allTextContents();
check(items.some(t => /^teleport /.test(t)) && items.some(t => /^Walk to /.test(t)), 'an admin keeps the teleport menu beside the walk: ' + JSON.stringify(items));
await page.keyboard.press('Escape');
await page.close();

// Day and night: outdoor tiles are shaded by the game's clock; indoor ones are not.
page = await gotoMixed(null);
const clock = (hour24, minute = 0) => ({ hour24, minute, day_start: 6, night_start: 20, night: hour24 >= 20 || hour24 < 6 });
await gmcp(page, 'Gametime', clock(12));
await tick(page, 300);
s = await state(page);
check(s.night === 0 && s.drawn.shaded === 0, 'noon draws no shading: ' + JSON.stringify([s.night, s.drawn.shaded]));
const noon = await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL());
await gmcp(page, 'Gametime', clock(19, 30));
await tick(page, 200);
s = await state(page);
check(s.night > 0.4 && s.night < 0.6 && s.drawn.shaded > 0, 'half an hour before nightfall is half dark: ' + s.night);
await gmcp(page, 'Gametime', clock(23));
await tick(page, 200);
s = await state(page);
const allShaded = s.drawn.shaded;
check(s.night === 1 && allShaded >= 7, 'midnight is full night on every outdoor tile: ' + JSON.stringify([s.night, allShaded]));
check(noon !== await page.evaluate(() => document.getElementById('map-2d-canvas').toDataURL()), 'night changes the picture');
await gmcp(page, 'Gametime', clock(6, 30));
await tick(page, 200);
s = await state(page);
check(s.night > 0.4 && s.night < 0.6, 'half an hour after dawn is half dark: ' + s.night);
await gmcp(page, 'Gametime', clock(23));
await gmcp(page, 'World.Map', { biomes: { moonbog: { name: 'Moonbog', indoor: true }, forest: { dark: true } }, rooms: mixed.rooms });
await tick(page, 300);
s = await state(page);
check(s.drawn.shaded < allShaded, 'indoor and dark biomes are left unshaded: ' + JSON.stringify([allShaded, s.drawn.shaded]));
await page.locator('#map-window').screenshot({ path: shot ? shot.replace(/\.png$/, '-night.png') : '/tmp/40d-night.png' });
await page.close();
page = await gotoMixed({ dayNight: false });
await gmcp(page, 'Gametime', clock(23));
await tick(page, 300);
s = await state(page);
check(s.drawn.shaded === 0, 'with day/night off nothing is shaded');
await page.close();

// --- E1: high-density figures at whole and fractional pixel ratios ---
for (const dpr of [1, 1.5, 2]) {
  const ctx = await browser.newContext({ viewport: { width: 900, height: 700 }, deviceScaleFactor: dpr });
  page = await ctx.newPage();
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  await page.goto(url);
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await start(page);
  await gmcp(page, 'Char.Info', { name: 'Wren', classid: 'paladin', lineage: 'warrior' });
  await settle(page); await page.waitForTimeout(200);
  s = await state(page);
  check(s.art && s.art.density === 4 && s.keys[0] === 'paladin', 'ratio ' + dpr + ': an elite draws its own density-4 art');
  const canvasPx = await page.evaluate(() => { const c = document.getElementById('map-2d-canvas'); return [c.width, c.clientWidth]; });
  check(Math.abs(canvasPx[0] - canvasPx[1] * dpr) <= 1, 'ratio ' + dpr + ': the canvas renders at device pixels ' + JSON.stringify(canvasPx));
  // walk east across two rooms and catch a frame mid-step
  await moveTo(page, 3, 1); await page.waitForTimeout(90);
  check((await state(page)).walking, 'ratio ' + dpr + ': the figure walks');
  await page.locator('#map-window').screenshot({ path: (shot ? shot.replace(/\.png$/, '') : '/tmp/e1-map') + '-dpr' + dpr + '-walk.png' });
  await settle(page);
  await page.locator('#map-window').screenshot({ path: (shot ? shot.replace(/\.png$/, '') : '/tmp/e1-map') + '-dpr' + dpr + '.png' });
  await ctx.close();
}

await browser.close();
server.close();
console.log(failures ? failures + ' check(s) failed' : 'all checks passed');
process.exit(failures ? 1 : 0);
