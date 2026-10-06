// Phase 50 browser check: a member's battle condition (hunger, thirst,
// fatigue and a meal buff) in the real web client windows, through
// dock-windows-harness.html: the Company panel's "In battle:" line, the
// Combat tab's "Condition:" notes, and the battle screen's banner and hover
// caption. Screenshots go to [prefix]-*.png when given.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/fare-check.mjs [screenshot-prefix]
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const shot = process.argv[2];

let failures = 0;
const check = (ok, what) => { if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); } };

const need = (value, label, warn) => ({ value, label, warn });
const member = (key, id, name, archetype, cell, role) => ({ key, id, name, status: 'present', level: 5, archetype, cell, chemistry: null, strategy: { role, target: 'weakest' } });
const company = {
  leader: member('leader', 0, 'Wren', 'Ranger', { row: 0, col: 1 }, 'fighter'),
  members: [
    member('companion:1', 1, 'Oswin', 'Cleric', { row: 1, col: 0 }, 'healer'),
    member('companion:2', 2, 'Brant', 'Warrior', { row: 0, col: 0 }, 'guardian'),
  ],
  alive: 3, dead: 0,
  vitals: {
    leader: { hp: 30, hp_max: 40, needs: { hunger: need(10, 'Starving', true), thirst: need(40, 'Thirsty', true), fatigue: need(80, 'Rested', false) }, warmth: '',
      fare: 'Starving, Thirsty: -10% damage, +5% damage taken' },
    'companion:1': { hp: 12, hp_max: 24, needs: { hunger: need(90, 'Well fed', false), thirst: need(90, 'Hydrated', false), fatigue: need(90, 'Rested', false) }, warmth: '',
      fare: 'Hearty: 10% less damage taken (3 battles)' },
    'companion:2': { hp: 20, hp_max: 20, needs: { hunger: need(90, 'Well fed', false), thirst: need(90, 'Hydrated', false), fatigue: need(90, 'Rested', false) }, warmth: '' },
  },
};
const fare = { leader: company.vitals.leader.fare, 'companion:1': company.vitals['companion:1'].fare };
const battle = {
  group: 'a band of ruffians', narrow: false,
  positions: { leader: { row: 0, col: 1 }, 'companion:1': { row: 1, col: 0 }, 'companion:2': { row: 0, col: 0 } },
  enemies: [{ id: 'm:1', label: 'the first ruffian', cell: { row: 0, col: 0 }, health: 'unhurt', sprite: 'unknown', reach: true, target: 'leader' },
    { id: 'm:2', label: 'the second ruffian', cell: { row: 0, col: 2 }, health: 'unhurt', sprite: 'unknown', reach: true, target: 'companion:2' }],
  company: [{ key: 'leader', target: 'm:1' }],
  focus: 'none', saved_focus: 'none', focus_ready: true, fare,
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
page.on('console', m => { if (m.type() === 'error') { failures++; console.log('FAIL console error: ' + m.text()); } });
await page.goto(process.env.DOCK_HARNESS_URL || 'file://' + path.join(here, 'dock-windows-harness.html'));
await page.evaluate(() => { localStorage.clear(); localStorage.setItem('ashveil-battle-screen', 'manual'); });
await page.reload();
const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);

// The Company panel: an "In battle:" line on each member it changes.
await gmcp('Company', company);
await page.getByRole('tab', { name: 'Company' }).first().click();
const card = key => page.evaluate(k => (document.querySelector(`#party-panel [data-key="${k}"] .company-fare`) || {}).textContent || '', key);
check((await card('leader')) === 'In battle: Starving, Thirsty: -10% damage, +5% damage taken', 'the leader card names the battle condition');
check((await card('companion:1')).includes('Hearty'), 'a fed member names its meal buff');
check((await card('companion:2')) === '', 'a well-kept member has no line');
if (shot) { await page.locator('#company-window').screenshot({ path: shot + '-company.png' }); }

// The Combat tab during a battle: one note per member.
await gmcp('Company.Battle', battle);
await page.getByRole('tab', { name: 'Combat' }).click();
const notes = await page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-fare')].map(n => n.textContent));
check(notes.length === 2 && notes[0].startsWith('Condition: ') && notes.some(n => n.includes('Oswin, Hearty')), 'the Combat tab notes each member\'s condition: ' + JSON.stringify(notes));
if (shot) { await page.locator('#combat-window').screenshot({ path: shot + '-combat.png' }); }

// The battle screen: the banner names who went in so, hovering says how.
await page.evaluate(() => window.BattleScreen.open());
await page.evaluate(() => window.BattleScreen.setMotion('off'));
await page.waitForTimeout(200);
const banner = await page.evaluate(() => document.querySelector('#battle-screen .bs-banners').textContent);
check(banner.includes('condition: Wren, Oswin'), 'the battle banner names the members: ' + banner);
const s = await page.evaluate(() => window.BattleScreen.state());
const at = s.units.find(u => u.id === 'leader').at;
const box = await page.locator('#battle-screen canvas').boundingBox();
await page.mouse.move(box.x + at.x * box.width / 320, box.y + (at.y - 12) * box.height / 180);
const cap = await page.evaluate(() => document.querySelector('#battle-screen .bs-caption').textContent);
check(cap.includes('Starving, Thirsty: -10% damage'), 'hovering a member shows its condition: ' + cap);
if (shot) { await page.locator('#battle-screen').screenshot({ path: shot + '-battle.png' }); }

await browser.close();
console.log(failures ? failures + ' check(s) failed' : 'all checks passed');
process.exit(failures ? 1 : 0);
