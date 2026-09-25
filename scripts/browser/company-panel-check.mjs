// Phase 26b browser check: drives the real window-party.js (through
// company-panel-harness.html) in Chromium with Playwright.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/company-panel-check.mjs [outdir]
//
// It checks the Company and Players sections, the formation table, a name
// holding markup rendered as text, a vitals-only update keeping the roster,
// a 360px-wide layout, keyboard focus, and the accessibility tree. With an
// outdir it saves desktop and narrow screenshots there. Exits non-zero on
// the first failure.
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const outdir = process.argv[2];

const snapshot = {
  leader: { key: 'leader', id: 0, name: 'Wren', status: 'present', level: 5, archetype: 'Ranger', cell: { row: 0, col: 0 }, chemistry: 'Trusted' },
  members: [
    { key: 'companion:1', id: 1, name: 'Bran', status: 'present', level: 3, archetype: 'Warrior', cell: { row: 0, col: 1 }, chemistry: 'Trusted' },
    { key: 'companion:2', id: 2, name: '<img src=x onerror="window.__xss=1">', status: 'awaiting', level: 2, archetype: null, cell: null, chemistry: null },
    { key: 'companion:3', id: 3, name: 'Ysolde', status: 'dead', level: 5, archetype: 'Cleric', cell: null, chemistry: null },
  ],
  alive: 3, dead: 1,
  load: { label: 'Burdened', total_g: 8000, capacity_g: 10000, cargo_g: 3000 },
  activity: 'Resting 12m',
  rest: { tier: 'Rested', seconds: 3600 },
  checkpoint: 'The Chapel of the Wayfarer',
  rescue: { 'companion:3': 5400 },
  vitals: {
    leader: { hp: 30, hp_max: 40, needs: { hunger: { value: 40, label: 'Hungry', warn: true }, thirst: { value: 90, label: 'Hydrated', warn: false }, fatigue: { value: 80, label: 'Rested', warn: false } }, warmth: 'Chilled' },
    'companion:1': { hp: 12, hp_max: 25, needs: { hunger: { value: 70, label: 'Sated', warn: false }, thirst: null, fatigue: null }, warmth: '' },
    'companion:2': { hp: null, hp_max: null, needs: null, warmth: null },
    'companion:3': { hp: null, hp_max: null, needs: null, warmth: null },
  },
};

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1024, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'company-panel-harness.html'));

const text = () => page.evaluate(() => document.getElementById('party-panel').textContent);

// Nothing yet.
await page.evaluate(() => window.gmcp('Party', {}));
check((await text()).includes('No company or party'), 'empty state before any company or party');

// No client request: the server sends the snapshot at login and copyover.
await page.evaluate(() => { window.gmcp('Char.Vitals', {}); });
check(JSON.stringify(await page.evaluate(() => window.requests)) === '[]', 'the panel sends no request of its own');

// The snapshot.
await page.evaluate(s => window.gmcp('Company', s), snapshot);
const t1 = await text();
check(/Company/.test(t1) && /Players/.test(t1), 'Company and Players sections both shown');
check(t1.includes('Not in a party'), 'a company shows with no human party');
check(t1.includes('3 alive, 1 fallen'), 'summary line');
check(t1.includes('Burdened (8.0/10.0 kg)') && t1.includes('Resting 12m') && t1.includes('Rested 1h 0m'), 'load, activity, rest');
check(t1.includes('Fallen: 1h 30m to raise'), 'a fallen member shows its time to raise');
check(t1.includes('Away: rejoins when you return'), 'an awaiting member says so');
check(t1.includes('Hunger: Hungry!') && t1.includes('Chilled'), 'the leader\'s warnings');
check(t1.includes('Chemistry: Trusted'), 'chemistry');
check(await page.evaluate(() => window.__xss === undefined), 'a name holding markup is not interpreted');
check(t1.includes('<img src=x onerror="window.__xss=1">'), 'and is shown as text');
const table = page.getByRole('table', { name: /Formation/ });
check(await table.count() === 1, 'formation table with its caption');
check((await table.textContent()).includes('Wren') && (await table.textContent()).includes('Bran'), 'formation cells hold names as text');
check(await page.getByRole('listitem', { name: /Bran, level 3, Warrior, Health 12 of 25/ }).count() === 1, 'member card has a spoken summary');
if (outdir) { await page.locator('#party-panel').screenshot({ path: path.join(outdir, 'company-desktop.png') }); }

// A Company.Vitals keeps the roster and brings the live values, and the
// focused card keeps focus across the rebuild.
await page.locator('.company-member[data-key="companion:1"]').focus();
const liveUpdate = JSON.parse(JSON.stringify({ vitals: snapshot.vitals, activity: 'Resting 11m', rest: snapshot.rest, rescue: { 'companion:3': 5340 } }));
liveUpdate.vitals['companion:1'] = { hp: 5, hp_max: 25, needs: null, warmth: null };
await page.evaluate(v => window.gmcp('Company.Vitals', v), liveUpdate);
const t2 = await text();
check(t2.includes('5/25'), 'vitals update applied');
check(t2.includes('Resting 11m') && t2.includes('Fallen: 1h 29m to raise'), 'countdowns come with the vitals');
check(await page.locator('.company-members > li').count() === 4, 'roster kept after a vitals update');
check(t2.includes('Hunger: Hungry!'), 'other members\' vitals kept');
check(await page.evaluate(() => document.activeElement && document.activeElement.getAttribute('data-key')) === 'companion:1', 'focus kept on the same card across an update');

// A human party beside the company, names safe there too.
await page.evaluate(() => window.gmcp('Party', { Leader: 'Wren', Members: [{ Name: 'Wren', Position: 'leader' }, { Name: '<b>Tamsin</b>', Position: 'member' }], Invited: [], Vitals: { Wren: { health: 80, level: 5, location: 'Dunmar' }, '<b>Tamsin</b>': { health: 40, level: 4, location: 'Dunmar' } } }));
const t3 = await text();
check(t3.includes('<b>Tamsin</b>') && await page.locator('#party-panel b').count() === 0, 'party names shown as text, not markup');
check(await page.locator('.company-section .company-member').count() === 4 && await page.locator('.players-section .party-member').count() === 2, 'company members never listed as party members');

// Keyboard: cards are in the tab order.
await page.evaluate(() => document.activeElement && document.activeElement.blur());
await page.keyboard.press('Tab');
check(await page.evaluate(() => document.activeElement && document.activeElement.classList.contains('company-member')), 'member cards are reachable by keyboard');

// A Party.Vitals-only payload (no member list) still renders the party.
await page.evaluate(() => window.gmcp('Party.Vitals', { Wren: { health: 70, level: 5, location: 'Dunmar' } }));
check(await page.locator('.players-section .party-member').count() >= 1, 'a vitals-only party payload still shows the party');

// Closed window: payloads still land in the store; reopening shows them.
await page.evaluate(() => window.windows[0].close());
const reopened = JSON.parse(JSON.stringify(snapshot));
reopened.members = reopened.members.slice(0, 1);
reopened.dead = 0; reopened.alive = 2; reopened.rescue = {};
await page.evaluate(s2 => window.gmcp('Company', s2), reopened);
await page.evaluate(() => window.windows[0].reopen());
await page.waitForTimeout(20);
check(await page.locator('.company-members > li').count() === 2, 'a reopened window shows what arrived while it was closed');
check(!(await text()).includes('fallen'), 'and nothing stale');

// Narrow: no sideways scrolling at 360px.
await page.setViewportSize({ width: 360, height: 900 });
const overflow = await page.evaluate(() => { const p = document.getElementById('party-panel'); return p.scrollWidth - p.clientWidth; });
check(overflow <= 0, 'no horizontal overflow at 360px (' + overflow + ')');
if (outdir) { await page.locator('#party-panel').screenshot({ path: path.join(outdir, 'company-narrow.png') }); }

// Accessibility tree: headings, table, and list are exposed with text.
const aria = await page.locator('#party-panel').ariaSnapshot();
check(/heading "Company"/.test(aria) && /heading "Players"/.test(aria), 'headings in the accessibility tree');
check(/table "Formation/.test(aria) && /cell "Bran"/.test(aria), 'formation table and cells in the accessibility tree');

// No company any more.
await page.evaluate(() => { window.gmcp('Company', {}); window.gmcp('Party', {}); });
check((await text()).includes('No company or party'), 'an empty snapshot clears the company');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all checks passed');
