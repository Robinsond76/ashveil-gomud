// Phase 32g browser check: drives the company dock's real window scripts
// (through dock-windows-harness.html, with the real web client core) in
// Chromium with Playwright, feeding them GMCP payloads and recording the
// commands they send.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/dock-windows-check.mjs [outdir]
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const outdir = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const xss = '<img src=x onerror="window.__xss=1">';
const need = (value, label, warn) => ({ value, label, warn });
const company = {
  leader: { key: 'leader', id: 0, name: 'Wren', status: 'present', level: 5, archetype: 'Ranger', cell: { row: 0, col: 0 }, chemistry: null, strategy: { role: 'fighter', target: 'weakest' } },
  members: [
    { key: 'companion:1', id: 1, name: 'Oswin', status: 'present', level: 3, archetype: 'Cleric', cell: { row: 1, col: 1 }, chemistry: 'Trusted', strategy: { role: 'healer', target: 'weakest' } },
    { key: 'companion:2', id: 2, name: xss, status: 'present', level: 2, archetype: 'Warrior', cell: { row: 0, col: 1 }, chemistry: null, strategy: { role: 'fighter', target: 'leader' } },
    { key: 'companion:3', id: 3, name: 'Tamsin', status: 'awaiting', level: 2, archetype: 'Ranger', cell: null, chemistry: null, strategy: { role: 'fighter', target: 'weakest' } },
    { key: 'companion:4', id: 4, name: 'Ysolde', status: 'dead', level: 5, archetype: 'Wizard', cell: { row: 2, col: 1 }, chemistry: null, strategy: { role: 'caster', target: 'weakest' } },
  ],
  alive: 4, dead: 1,
  load: { label: 'Heavily burdened', total_g: 46000, capacity_g: 50000, cargo_g: 3000, companion_g: 20000 },
  activity: 'Camped', rest: { tier: 'Rested', seconds: 600 }, checkpoint: 'The Chapel',
  rescue: { 'companion:4': 5400 },
  vitals: {
    leader: { hp: 30, hp_max: 40, mp: 6, mp_max: 14, needs: { hunger: need(40, 'Hungry', true), thirst: need(90, 'Hydrated', false), fatigue: need(80, 'Rested', false) }, warmth: '' },
    'companion:1': { hp: 12, hp_max: 25, mp: 8, mp_max: 20, needs: { hunger: need(70, 'Sated', false), thirst: need(20, 'Parched', true), fatigue: null }, warmth: 'Chilled' },
    'companion:2': { hp: 20, hp_max: 20, needs: null, warmth: '' },
    'companion:3': { hp: null, hp_max: null, needs: null, warmth: null },
    'companion:4': { hp: null, hp_max: null, needs: null, warmth: null },
  },
};

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'dock-windows-harness.html'));
await page.evaluate(() => localStorage.clear());
await page.reload();

// --- Task 8: the vitals strip ---
await page.evaluate(() => window.gmcp('Char.Vitals', { hp: 30, hp_max: 40, sp: 6, sp_max: 14 }));
await page.evaluate(c => window.gmcp('Company', c), company);
const strip = () => page.evaluate(() => document.getElementById('vitals-bars').textContent);
check(await page.evaluate(() => document.querySelector('.dock-tabgroup-header #vitals-bars') !== null), 'the strip sits in the dock\'s header');
check((await strip()).includes('30 / 40') && (await strip()).includes('6 / 14'), 'the player\'s HP and MP');
const rows = await page.evaluate(() => [...document.querySelectorAll('.vitals-member')].map(r => ({
  key: r.dataset.key, text: r.textContent, label: r.getAttribute('aria-label'), meters: r.querySelectorAll('[role=meter]').length,
  away: r.classList.contains('is-away'), fallen: r.classList.contains('is-fallen'),
})));
check(rows.length === 4, 'a row per companion');
check(rows[0].meters === 2 && rows[0].label === 'Oswin, health 12 of 25, mana 8 of 20', 'a caster shows health and mana, spoken with numbers');
check(rows[1].meters === 1, 'one with no mana shows health only');
check(rows[2].away && rows[2].text.includes('not with you') && rows[2].meters === 0, 'one not with you is dimmed, with no bars');
check(rows[3].fallen && rows[3].text.includes('fallen, 1h 30m to raise'), 'a fallen one shows its time to raise');
check(await page.evaluate(() => window.__xss === undefined) && rows[1].text.includes('<img'), 'a name holding markup renders as text');
const warn = await page.evaluate(() => { const w = document.getElementById('vitals-warn'); return { hidden: w.hidden, text: w.textContent }; });
check(!warn.hidden && warn.text.includes('You: Hungry') && warn.text.includes('Oswin: Parched') && warn.text.includes('Oswin: Chilled') && warn.text.includes('Load 92%'), 'the warnings line: needs, warmth, load');
await page.evaluate(() => window.gmcp('Company.Vitals', { vitals: {
  leader: { hp: 30, hp_max: 40, mp: 6, mp_max: 14, needs: null, warmth: '' },
  'companion:1': { hp: 5, hp_max: 25, mp: 2, mp_max: 20, needs: null, warmth: '' },
}, rescue: { 'companion:4': 60 } }));
const after = await page.evaluate(() => document.querySelector('.vitals-member[data-key="companion:1"]').getAttribute('aria-label'));
check(after === 'Oswin, health 5 of 25, mana 2 of 20', 'a Company.Vitals updates the rows');
check(await page.evaluate(() => document.getElementById('vitals-warn').textContent.includes('Load 92%') && !document.getElementById('vitals-warn').textContent.includes('Hungry')), 'the warnings follow the newest vitals');
await page.evaluate(() => window.gmcp('Company', {}));
check(await page.evaluate(() => document.querySelectorAll('.vitals-member').length === 0 && document.getElementById('vitals-warn').hidden), 'no company: no rows and no warnings');
await page.evaluate(c => window.gmcp('Company', c), company);
if (outdir) { await page.locator('#vitals-bars').screenshot({ path: path.join(outdir, 'vitals-strip.png') }); }

// --- Task 9: the Character tab ---
await page.evaluate(() => {
  window.gmcp('Char.Info', { name: 'Wren', class: 'ranger', race: 'Human', level: 5 });
  window.gmcp('Char.Worth', { xp: 40, tnl: 100, gold_carry: 12, gold_bank: 300 });
  window.gmcp('Char.Inventory', {
    Worn: { weapon: { id: '!1:sword', name: 'iron sword', type: 'weapon', subtype: 'slashing', details: [] } },
    Backpack: { items: [{ id: '!2:water', name: 'waterskin', type: 'object', subtype: 'drinkable', uses: 3, details: [] }],
      Summary: { count: 1, weight_g: 2500, load_g: 46000, capacity_g: 50000 } },
  });
  window.gmcp('Char.Jobs', []);
});
const subtabs = () => page.evaluate(() => [...document.querySelectorAll('#character-window .cw-tab-btn')]
  .filter(b => !b.hidden).map(b => b.textContent));
check(JSON.stringify(await subtabs()) === JSON.stringify(['Overview', 'Gear', 'Skills', 'Quests', 'Effects']), 'Character sub-tabs in order; no Pet without a pet');
check(await page.evaluate(() => ['Worth', 'Gear', 'Pet', 'Party'].every(id => !VirtualWindows.getWindows().some(w => w._id === id))), 'Worth, Gear, Pet, and Party are no longer windows of their own');
const overview = await page.evaluate(() => document.getElementById('cw-overview').textContent);
check(overview.includes('Wren') && overview.includes('40 / 100') && overview.includes('300'), 'Overview carries Worth (XP, gold, bank)');
await page.getByRole('tab', { name: 'Gear' }).click();
check(await page.evaluate(() => document.getElementById('gw-bp-count').textContent) === 'You 2.5 kg \u00b7 company 46.0 / 50.0 kg', 'Gear header: your weight and the company\'s load against capacity');
await page.evaluate(() => window.gmcp('Company.Inventory', { load: { total_g: 50500, capacity_g: 50000, member_capacity_g: 50000, mount_capacity_g: 0, cargo_g: 0 },
  members: [{ key: 'leader', name: 'Wren', grams: 2500, worn: [{ ref: '!1:sword', name: 'iron sword', grams: 1500 }], carried: [{ ref: '!2:water', name: 'waterskin', grams: 1000, uses: 3, uses_max: 5 }] }],
  companions_known: true, horses: [], cargo: [] }));
check(await page.evaluate(() => document.getElementById('gw-bp-count').textContent.includes('company 50.5 / 50.0 kg') && document.getElementById('gw-bp-count').classList.contains('full')), 'Company.Inventory updates the header, marked full');
await page.evaluate(() => { document.querySelector('.gw-tab-btn[data-panel=gw-backpack]').click(); });
await page.hover('.gw-bp-row');
await page.waitForTimeout(50);
check(await page.evaluate(() => document.getElementById('gw-item-tooltip').textContent.includes('Weight1.0 kg')), 'an item\'s tooltip carries its weight');
await page.click('.gw-bp-row');
await page.getByText('drink waterskin').click();
check((await page.evaluate(() => window.sent)).includes('drink waterskin'), 'the gear menus still send their commands');
await page.getByRole('tab', { name: 'Skills' }).click();
check(await page.evaluate(() => document.getElementById('cw-skills-tab').textContent.includes('Jobs')), 'Jobs sit under Skills');
await page.evaluate(() => window.gmcp('Char.Pets', [{ name: 'Rex', type: 'dog', level: 2, hunger: 'full', items: [], buffs: [] }]));
check((await subtabs()).includes('Pet'), 'Pet appears with a pet');
await page.getByRole('tab', { name: 'Pet' }).click();
check(await page.evaluate(() => document.getElementById('pw-info').textContent.includes('Rex')), 'and shows it');
await page.evaluate(() => window.gmcp('Char.Pets', []));
check(!(await subtabs()).includes('Pet') && await page.evaluate(() => document.querySelector('#character-window .cw-tab-btn.active').textContent) === 'Overview', 'and goes with it, back to Overview');
await page.getByRole('tab', { name: 'Gear' }).click();
await page.reload();
check(await page.evaluate(() => document.querySelector('#character-window .cw-tab-btn.active').textContent) === 'Gear', 'the sub-tab survives a reload');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all dock window checks passed');
