// Phase 40f browser check: drives the battle screen (window-battle.js,
// through dock-windows-harness.html with the real web client core) in
// Chromium with Playwright: GMCP feeds and Company.Battle.Event messages
// in, positions, bars, statuses, the outcome hold and the commands out.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/battle-check.mjs [screenshot.png]
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const shot = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const member = (key, id, name, archetype, cell, role) => ({
  key, id, name, status: 'present', level: 5, archetype, cell, chemistry: null, strategy: { role, target: 'weakest' },
});
const company = {
  leader: member('leader', 0, 'Wren', 'Ranger', { row: 0, col: 1 }, 'fighter'),
  members: [
    member('companion:1', 1, 'Oswin', 'Cleric', { row: 1, col: 0 }, 'healer'),
    member('companion:2', 2, 'Brant', 'Warrior', { row: 0, col: 0 }, 'guardian'),
    member('companion:3', 3, 'Ysolde', 'Wizard', { row: 2, col: 2 }, 'caster'),
    member('companion:4', 4, 'Tamsin', 'Rogue', null, 'fighter'),
  ],
  alive: 5, dead: 0,
  vitals: {
    leader: { hp: 30, hp_max: 40 }, 'companion:1': { hp: 12, hp_max: 24 }, 'companion:2': { hp: 20, hp_max: 20 },
    'companion:3': { hp: 5, hp_max: 20 }, 'companion:4': { hp: null, hp_max: null },
  },
};
const positions = {
  leader: { row: 0, col: 1 }, 'companion:1': { row: 1, col: 0 }, 'companion:2': { row: 0, col: 0 }, 'companion:3': { row: 2, col: 2 },
};
const enemy = (n, label, row, col, health, sprite, target) => ({ id: 'm:' + n, label, cell: { row, col }, health, sprite, reach: true, target });
const battle = {
  group: 'a pack of timber wolves', narrow: false, positions,
  enemies: [
    enemy(1, 'the first wolf', 0, 0, 'wounded', 'wolf-timber', 'companion:2'),
    enemy(2, 'the second wolf', 0, 2, 'unhurt', 'wolf-timber', 'leader'),
    enemy(3, 'the third wolf', 1, 1, 'near death', 'wolf-timber', 'companion:1'),
    enemy(4, 'a hulking brute', 2, 1, 'scratched', 'unknown-large', ''),
  ],
  company: [{ key: 'leader', target: 'm:2' }, { key: 'companion:2', target: 'm:1' }],
  focus: 'none', saved_focus: 'none', focus_ready: true,
  outlook: { risk: 'fair', close: true, text: 'It could go either way.' },
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
page.on('console', m => { if (m.type() === 'error') { failures++; console.log('FAIL console error: ' + m.text()); } });
await page.goto(process.env.DOCK_HARNESS_URL || 'file://' + path.join(here, 'dock-windows-harness.html'));
await page.evaluate(() => localStorage.clear());
await page.reload();

const state = () => page.evaluate(() => window.BattleScreen.state());
const unitOf = (s, id) => s.units.find(u => u.id === id);
const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
const events = body => page.evaluate(b => Client.dispatchBattleEvents(b), body);

await gmcp('Room', { Info: { environment: 'forest' } });
await gmcp('Company', company);
check(!(await state()).open, 'closed with no battle');

// --- Opening, positions, bars ---
await gmcp('Company.Battle', battle);
let s = await state();
check(s.open, 'the screen opens by itself when a battle starts');
check(s.biome === 'forest', 'the biome comes from Room.Info.environment');
check(s.units.filter(u => u.side === 'company').length === 4, 'four placed members stand (the unplaced one does not)');
check(s.units.filter(u => u.side === 'enemy').length === 4, 'four enemies stand');
check(unitOf(s, 'leader').cell.row === 0 && unitOf(s, 'leader').cell.col === 1, 'the leader stands in their cell');
check(unitOf(s, 'leader').at.x === 160 - 46 && unitOf(s, 'leader').at.y === 140, 'row 0 col 1 sits left of the line, middle lane');
check(unitOf(s, 'm:1').at.x > 160 && unitOf(s, 'leader').at.x < 160, 'company left, enemy right');
check(unitOf(s, 'companion:3').at.x < unitOf(s, 'companion:1').at.x, 'back rows stand further from the line');
check(unitOf(s, 'leader').leader, 'the leader is marked');
check(unitOf(s, 'leader').frac === 0.75 && unitOf(s, 'companion:1').frac === 0.5, 'company bars match the vitals');
const bands = [1, 0.8, 0.6, 0.4, 0.2];
check(s.units.filter(u => u.side === 'enemy').every(u => bands.includes(u.frac)), 'enemy bars are banded');
check(unitOf(s, 'm:1').frac === 0.6 && unitOf(s, 'm:3').frac === 0.2, 'bands follow the health words');
check(unitOf(s, 'm:1').sprite === 'wolf-timber' && unitOf(s, 'm:4').sprite === 'unknown-large', 'enemies carry their sprite keys');
check(unitOf(s, 'companion:2').role === 'guardian', 'roles come from the company');
check(await page.evaluate(() => document.querySelector('#battle-screen canvas').getAttribute('aria-hidden') === 'true'), 'the canvas is decorative');
check((await page.evaluate(() => document.querySelector('#battle-screen .bs-title').textContent)).includes('a pack of timber wolves'), 'the title names the group');

// --- Screenshot ---
await events({ fight: 1, round: 1, events: [
  { seq: 1, kind: 'attack', src: 'leader', tgt: 'm:2', outcome: 'hit', damage: 6 },
  { seq: 2, kind: 'status-applied', tgt: 'm:1', status: 'bleeding' },
  { seq: 3, kind: 'cast-start', src: 'companion:3', spell: 'fire bolt' },
  { seq: 4, kind: 'heal', src: 'companion:1', tgt: 'companion:2', amount: 4 },
] });
if (shot) { await page.locator('#battle-screen').screenshot({ path: shot }); }

// --- Events ---
s = await state();
check(unitOf(s, 'm:1').statuses.includes('bleeding'), 'a status applied shows on its unit');
check(unitOf(s, 'companion:3').casting === 'fire bolt', 'a chanting caster is marked');
await events({ fight: 1, round: 2, events: [
  { seq: 5, kind: 'status-expired', tgt: 'm:1', status: 'bleeding' },
  { seq: 6, kind: 'cast-complete', src: 'companion:3', outcome: 'cast' },
  { seq: 7, kind: 'death', tgt: 'm:3' },
  { seq: 8, kind: 'yield', src: 'm:2' },
] });
s = await state();
check(unitOf(s, 'm:1').statuses.length === 0, 'an expired status leaves');
check(unitOf(s, 'companion:3').casting === '', 'the chant mark ends with the cast');
check(unitOf(s, 'm:3').fallen, 'a death lays its unit down');
check(unitOf(s, 'm:2').yielded, 'a yield is marked');

// --- Unseen presence ---
await events({ fight: 1, round: 3, events: [
  { seq: 9, kind: 'cast-start', src: '?', spell: 'secret rite' },
  { seq: 10, kind: 'attack', src: '?', tgt: 'leader', outcome: 'hit', damage: 3 },
  { seq: 11, kind: 'status-applied', tgt: '?', status: 'cursed' },
] });
s = await state();
const ghost = s.units.filter(u => u.unseen);
check(ghost.length === 1, 'every ? enemy is drawn as one unseen presence');
check(ghost[0].casting === 'a spell' && ghost[0].statuses.length === 0, 'an unseen caster names no spell and shows no status');

// --- A fight that grows mid-battle ---
await gmcp('Company.Battle', { ...battle, enemies: battle.enemies.concat([enemy(5, 'a late wolf', 1, 2, 'unhurt', 'wolf-timber', '')]) });
s = await state();
check(unitOf(s, 'm:5') && unitOf(s, 'm:5').cell.row === 1, 'a newcomer appears from the next snapshot');

// --- Buttons ---
await page.evaluate(() => { window.sent.length = 0; });
await page.click('#battle-screen button:text("Retreat")');
await page.click('#battle-screen button[data-focus="weakest"]');
const sent = await page.evaluate(() => window.sent);
check(sent[0] === 'retreat' && sent[1] === 'company tactics focus weakest', 'Retreat and focus send the dock\'s commands: ' + JSON.stringify(sent));
await gmcp('Company.Battle', { ...battle, focus_ready: false });
check(await page.evaluate(() => document.querySelector('#battle-screen button[data-focus="weakest"]').disabled), 'focus buttons wait while an order is pending');
await gmcp('Company.Battle', battle);

await page.evaluate(() => { window.sent.length = 0; });
await page.click('#battle-screen button:text("Help")');
check((await page.evaluate(() => window.sent))[0] === 'help battlescreen', 'Help opens the battle screen\'s page');
check((await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent)).includes('Hover or tap'), 'the caption says how to read a figure');

// --- A member who fled leaves the picture (review fix) ---
await gmcp('Company', { ...company, members: company.members.map(m => (m.key === 'companion:1' ? { ...m, status: 'fled' } : m)) });
check(!unitOf(await state(), 'companion:1'), 'a member who fled no longer stands');
await gmcp('Company', company);

// --- Minimise and the badge ---
await page.click('#battle-screen button:text("Minimise")');
s = await state();
check(!s.open && s.badge, 'minimising leaves a badge');
await gmcp('Company.Battle', battle);
check(!(await state()).open, 'a snapshot does not reopen a minimised screen');
await page.click('#battle-badge');
check((await state()).open, 'the badge reopens it');

// --- The outcome hold, then close ---
await events({ fight: 1, round: 9, events: [{ seq: 20, kind: 'fight-end', outcome: 'victory' }] });
await gmcp('Company.Battle', {});
s = await state();
check(s.open && s.outcome === 'Victory', 'the outcome shows after the fight');
await page.waitForTimeout(3300);
check(!(await state()).open, 'the screen closes about 3 seconds later');

// --- A late snapshot of the finished battle keeps the outcome (review fix) ---
await gmcp('Company.Battle', battle);
await events({ fight: 1, round: 10, events: [{ seq: 21, kind: 'fight-end', outcome: 'defeat' }] });
await gmcp('Company.Battle', battle);
s = await state();
check(s.open && s.outcome === 'Defeat', 'a snapshot after fight-end does not clear the outcome');
await page.waitForTimeout(3300);
await gmcp('Company.Battle', battle);
s = await state();
check(!s.open && !s.badge, 'nor reopen the screen once the hold has closed it');
await gmcp('Company.Battle', {});
check(!(await state()).open, 'the end of the battle does not show a second outcome');

// --- A new battle reopens it, in the dark ---
await gmcp('Company.Battle', { group: 'the enemy', dark: true, enemies: [], focus: 'none', saved_focus: 'none', focus_ready: true });
s = await state();
check(s.open && s.units.filter(u => u.unseen).length === 1 && s.units.filter(u => u.side === 'enemy').length === 1, 'a dark battle shows one unseen presence');
await gmcp('Company.Battle', {});
await page.waitForTimeout(3300);

// --- Manual mode ---
await page.evaluate(() => localStorage.setItem('ashveil-battle-screen', 'manual'));
await page.reload();
await gmcp('Company', company);
await gmcp('Company.Battle', battle);
s = await state();
check(!s.open && s.badge, 'manual mode does not open the screen; the badge offers it');
await page.evaluate(() => window.BattleScreen.open());
check((await state()).open, 'open() shows it by hand');

// --- Phone width ---
await page.setViewportSize({ width: 360, height: 700 });
await page.evaluate(() => window.dispatchEvent(new Event('resize')));
const box = await page.evaluate(() => { const r = document.querySelector('#battle-screen').getBoundingClientRect(); return { l: r.left, r: r.right, w: window.innerWidth }; });
check(box.l >= 0 && box.r <= box.w, 'the screen fits a phone width with nothing cropped');

await browser.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
