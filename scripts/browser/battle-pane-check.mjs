// Phase 82a browser check: the battle screen docks at the top of the middle
// column (dock-windows-harness.html with the real web client core), above
// the terminal, which keeps at least 12 rows and steps its font down a size
// while the pane is open. Checked at 1280x800 with both docks, with one
// dock, and at 360x740; minimise and restore; the "Smaller text" setting.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/battle-pane-check.mjs [outdir]
//
// BROWSER=firefox runs it in Firefox (Robinson's browser) when Playwright
// has it installed; the default is Chromium.
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const pw = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const outdir = process.argv[2];
const which = process.env.BROWSER === 'firefox' ? 'firefox' : 'chromium';

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
  ],
  alive: 3, dead: 0,
  vitals: { leader: { hp: 30, hp_max: 40 }, 'companion:1': { hp: 12, hp_max: 24 }, 'companion:2': { hp: 20, hp_max: 20 } },
};
const battle = {
  group: 'a pack of timber wolves', narrow: false,
  positions: { leader: { row: 0, col: 1 }, 'companion:1': { row: 1, col: 0 }, 'companion:2': { row: 0, col: 0 } },
  enemies: [
    { id: 'm:1', label: 'the first wolf', cell: { row: 0, col: 0 }, health: 'wounded', sprite: 'wolf-timber', reach: true, target: 'companion:2' },
    { id: 'm:2', label: 'the second wolf', cell: { row: 0, col: 2 }, health: 'unhurt', sprite: 'wolf-timber', reach: true, target: 'leader' },
  ],
  company: [{ key: 'leader', target: 'm:2' }],
  focus: 'none', saved_focus: 'none', focus_ready: true,
};

const launch = { executablePath: which === 'chromium' ? (process.env.CHROMIUM_EXECUTABLE_PATH || undefined) : undefined };
const browser = await pw[which].launch(launch);
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
page.on('console', m => { if (m.type() === 'error') { failures++; console.log('FAIL console error: ' + m.text()); } });
await page.goto(process.env.DOCK_HARNESS_URL || 'file://' + path.join(here, 'dock-windows-harness.html'));
await page.evaluate(() => localStorage.clear());
await page.reload();
await page.evaluate(() => window.BattleScreen.setMotion('off'));
// The harness does not connect, so mount the terminal as Client.init would.
await page.evaluate(() => {
  Client.term.open(document.getElementById('terminal'));
  window.addEventListener('resize', Client.resizeTerminal);
  Client.resizeTerminal();
});

const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
const state = () => page.evaluate(() => window.BattleScreen.state());
const layout = () => page.evaluate(() => {
  const r = s => { const e = document.querySelector(s); return e ? e.getBoundingClientRect() : null; };
  const pane = r('#battle-pane'), term = r('#terminal'), canvas = r('#battle-screen canvas'), center = r('#center');
  const t = Client.term;
  return {
    paneOpen: document.getElementById('battle-pane').classList.contains('open'),
    pane: pane && { top: pane.top, bottom: pane.bottom, left: pane.left, right: pane.right, h: pane.height },
    term: term && { top: term.top, bottom: term.bottom, h: term.height },
    center: center && { top: center.top, h: center.height, w: center.width },
    canvas: canvas && { w: canvas.width, h: canvas.height, left: canvas.left, right: canvas.right },
    rows: t.rows, font: t.options.fontSize,
    docks: [document.getElementById('dock-left').classList.contains('has-panels'), document.getElementById('dock-right').classList.contains('has-panels')],
    wide: document.documentElement.scrollWidth <= window.innerWidth,
  };
});
const settle = () => page.waitForTimeout(250);
const shot = async name => { if (outdir) { await page.screenshot({ path: path.join(outdir, '82a-' + name + '-' + which + '.png') }); } };

// --- Desktop, one dock (the harness opens its windows into the right dock) ---
await gmcp('Room', { Info: { environment: 'forest', area: 'Frostfang' } });
await gmcp('Company', company);
await settle();
let before = await layout();
console.log('before: docks ' + before.docks + ', font ' + before.font + ', rows ' + before.rows);
await gmcp('Company.Battle', battle);
await settle();
let l = await layout();
check((await state()).open && l.paneOpen, 'a battle opens the pane');
check(l.pane.top === l.center.top && l.pane.bottom <= l.term.top + 1, 'the pane sits at the top of the middle column, the terminal below it');
check(l.pane.left >= l.center.top - 1 && l.canvas.left >= l.pane.left && l.canvas.right <= l.pane.right, 'the picture is inside the column');
check(l.canvas.w % 320 === 0 && l.canvas.w >= 320 && l.canvas.h <= l.center.h / 2 + 1, 'the picture is a whole-number scale (' + (l.canvas.w / 320) + 'x) at most half the column high');
check(l.rows >= 12, 'the terminal keeps at least 12 rows: ' + l.rows);
check(l.font < before.font, 'the terminal font stepped down: ' + before.font + ' -> ' + l.font);
check(l.wide, 'nothing scrolls sideways');
await shot('desktop-one-dock');

// --- Minimise and restore ---
await page.click('#battle-screen .bs-head button:has-text("Minimise")');
await settle();
l = await layout();
check(!l.paneOpen && (await state()).badge, 'minimising closes the pane and leaves the badge');
check(l.term.top === l.center.top && l.font === before.font && l.rows >= before.rows - 1, 'the terminal takes the column again at its old font (' + l.font + ', ' + l.rows + ' rows)');
await page.click('#battle-badge');
await settle();
l = await layout();
check(l.paneOpen && l.font < before.font, 'the badge restores the pane and the smaller font');

// --- The "Smaller text" setting ---
await page.click('#battle-screen .bs-foot label:has-text("Smaller text") input');
await settle();
l = await layout();
check((await state()).text === 'same' && l.font === before.font && l.paneOpen, 'unticking Smaller text keeps the terminal font with the pane open');
await page.click('#battle-screen .bs-foot label:has-text("Smaller text") input');
await settle();
l = await layout();
check((await state()).text === 'smaller' && l.font < before.font, 'ticking it steps the font down again');
await shot('desktop-restored');

// --- One dock ---
// --- No dock (the harness opens its windows into the right dock only) ---
await page.evaluate(() => { const d = document.getElementById('dock-right'); d.classList.remove('has-panels'); d.style.display = 'none'; window.dispatchEvent(new Event('resize')); });
await settle();
l = await layout();
check(l.paneOpen && l.pane.bottom <= l.term.top + 1 && l.rows >= 12 && l.canvas.w % 320 === 0 && l.font === 15, 'with no dock the pane still sits above a terminal of ' + l.rows + ' rows, picture ' + (l.canvas.w / 320) + 'x, font ' + l.font);
await shot('desktop-no-dock');
await page.evaluate(() => { const d = document.getElementById('dock-right'); d.classList.add('has-panels'); d.style.display = ''; window.dispatchEvent(new Event('resize')); });

// --- The fight ends: the pane closes after the outcome ---
await page.evaluate(() => Client.dispatchBattleEvents({ fight: 1, round: 3, fight_round: 3, pace: 'off', events: [{ seq: 1, kind: 'fight-end', outcome: 'victory' }] }));
await page.waitForTimeout(3500);
await gmcp('Company.Battle', {});
l = await layout();
check(!l.paneOpen && l.font === before.font && l.term.top === l.center.top, 'the pane closes after the outcome and the terminal is whole again');

// --- Phone: 360x740 ---
await page.setViewportSize({ width: 360, height: 740 });
await page.evaluate(() => { document.body.classList.add('mobile'); document.body.dataset.mview = 'game'; window.dispatchEvent(new Event('resize')); });
await settle();
before = await layout();
await gmcp('Company.Battle', Object.assign({}, battle, { group: 'a pack of timber wolves' }));
await settle();
l = await layout();
check(l.paneOpen && l.pane.top === l.center.top && l.pane.bottom <= l.term.top + 1, 'on a phone the pane sits above the terminal in the Game view');
check(l.canvas.w === 320 && l.canvas.right <= 360, 'the picture is 1x and fits 360 px');
check(l.pane.h <= l.center.h * 0.55 + 1 && l.rows >= 8, 'the pane takes at most 55% of the column; the terminal keeps ' + l.rows + ' rows');
check(l.font < before.font, 'the phone font stepped down: ' + before.font + ' -> ' + l.font);
check(l.wide, 'nothing scrolls sideways on a phone');
await shot('phone');

await browser.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
