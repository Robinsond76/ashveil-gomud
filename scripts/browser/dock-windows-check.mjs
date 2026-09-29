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

// --- Task 10: the Company tab ---
await page.evaluate(c => window.gmcp('Company', c), company);
await page.getByRole('tab', { name: 'Company' }).first().click();
const csubs = await page.evaluate(() => [...document.querySelectorAll('#company-window .cmp-tab-btn')].map(b => b.textContent));
check(JSON.stringify(csubs) === '["Status","Inventory","Camp"]', 'Company sub-tabs: Status, Inventory, Camp');
const status = () => page.evaluate(() => document.getElementById('party-panel').textContent);
check((await status()).includes('4 alive, 1 fallen') && (await status()).includes('Fallen: 1h 30m to raise'), 'Status: the 26b summary and cards');
check(await page.getByRole('table', { name: /Formation/ }).count() === 1, 'Status: the formation table');
check(await page.getByRole('listitem', { name: /Oswin, level 3, Cleric, Health 12 of 25/ }).count() === 1, 'a member card has a spoken summary');
check(!(await status()).includes('Travelling with'), 'no human party: no Travelling with');
await page.evaluate(() => window.gmcp('Party', { Leader: 'Wren', Members: [{ Name: 'Wren', Position: 'leader' }, { Name: '<b>Tamsin</b>', Position: 'member' }], Invited: [], Vitals: { Wren: { health: 80, level: 5, location: 'Dunmar' }, '<b>Tamsin</b>': { health: 40, level: 4, location: 'Dunmar' } } }));
check((await status()).includes('Travelling with') && (await status()).includes('<b>Tamsin</b>') && await page.locator('#party-panel b').count() === 0, 'a human party under Travelling with, names as text');
await page.locator('.company-member[data-key="companion:1"]').focus();
await page.evaluate(() => window.gmcp('Company.Vitals', { vitals: { 'companion:1': { hp: 5, hp_max: 25, needs: null, warmth: null } }, rescue: { 'companion:4': 5340 } }));
check((await status()).includes('5/25') && await page.evaluate(() => document.activeElement && document.activeElement.getAttribute('data-key')) === 'companion:1', 'a vitals update applies, keeping focus on the card');
await page.evaluate(c => window.gmcp('Company', c), company);

const inventory = {
  load: { total_g: 46000, capacity_g: 150000, member_capacity_g: 50000, mount_capacity_g: 100000, cargo_g: 3000 },
  companions_known: true,
  members: [
    { key: 'leader', name: 'Wren', fallen: false, unrecorded: false, grams: 4500, pack: 'satchel', pack_bonus_g: 5000,
      worn: [{ ref: '!1:sword', name: 'iron sword', grams: 1500, count: 1, uses: 0, uses_max: 0, type: 'weapon', subtype: 'slashing', slot: 'weapon' }],
      carried: [
        { ref: '!2:a', name: 'waterskin', grams: 1000, count: 1, uses: 3, uses_max: 5, type: 'object', subtype: 'drinkable' },
        { ref: '!2:b', name: 'waterskin', grams: 1000, count: 1, uses: 5, uses_max: 5, type: 'object', subtype: 'drinkable' },
        { ref: '!7:saddle', name: 'pack saddle', grams: 1000, count: 1, uses: 0, uses_max: 0, type: 'object', subtype: '' },
      ] },
    { key: 'companion:1', name: 'Brother Oswin', fallen: false, unrecorded: false, grams: 2000, pack: '', pack_bonus_g: 0, worn: [],
      carried: [{ ref: '!3:meat', name: 'seared meat', grams: 300, count: 1, uses: 0, uses_max: 0, type: 'object', subtype: 'edible' }] },
    { key: 'companion:4', name: 'Ysolde', fallen: true, unrecorded: false, grams: 0, pack: '', pack_bonus_g: 0, worn: [], carried: [] },
  ],
  horses: [{ id: 1, name: 'pack horse', kind: 'pack', saddle: '', capacity_g: 40000, rides: false }],
  cargo: [{ ref: '!3', name: 'seared meat', grams: 300, count: 6, uses: 0, uses_max: 0, type: 'object', subtype: 'edible' }],
};
// Oswin is out with the player (the snapshot's companion:1).
await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);
await page.getByRole('tab', { name: 'Inventory' }).click();
const invText = () => page.evaluate(() => document.getElementById('company-inventory').textContent);
check((await invText()).includes('Load 46.0 kg / 150.0 kg (31%)') && (await invText()).includes('horses 100.0 kg'), 'Inventory: the load and its split');
check((await invText()).includes('Wren (you)') && (await invText()).includes('Pack: satchel (+5.0 kg)') && (await invText()).includes('Fallen: their gear is with the body.'), 'every member: you first, your pack, the fallen');
check((await invText()).includes('3/5') && (await invText()).includes('seared meat x6'), 'uses left and cargo counts');
const sentNow = async fn => { await page.evaluate(() => { window.sent = []; }); await fn(); return page.evaluate(() => window.sent); };
const first = page.locator('#company-inventory button.cmp-item', { hasText: 'waterskin' }).first();
check((await first.getAttribute('title')).includes('3 of 5 uses left'), 'an item\'s tooltip: weight and uses');
let got = await sentNow(async () => { await first.click(); await page.getByText('Put in cargo').click(); });
check(JSON.stringify(got) === '["cargo put !2:a"]', 'Put in cargo names exactly that waterskin');
await first.click();
const menuLabels = await page.evaluate(() => { const m = [...document.querySelectorAll('body > div')].pop(); return [...m.children].map(c => c.textContent); });
check(menuLabels.includes('Give to Oswin') && !menuLabels.some(l => /Tamsin|Ysolde/.test(l)), 'Give only to companions out with you (not away, not fallen)');
await page.mouse.click(5, 5);
await page.evaluate(() => { const c = JSON.parse(JSON.stringify(Client.GMCPStructs.Company)); c.members[0].name = 'Brother Oswin'; window.gmcp('Company', c); });
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'waterskin' }).nth(1).click(); await page.getByText('Give to Brother Oswin').click(); });
check(JSON.stringify(got) === '["give !2:b \\"Brother Oswin\\""]', 'Give to a companion out, by reference, the name quoted');
check(await page.locator('#company-inventory [aria-label="Brother Oswin"] button').count() === 0, 'a companion\'s items have no menu (tooltip only)');
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'seared meat x6' }).click(); await page.getByText('Take one').click(); });
check(JSON.stringify(got) === '["cargo take !3"]', 'a cargo stack: Take one');
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'pack horse' }).click(); await page.getByText('Fit pack saddle').click(); });
check(JSON.stringify(got) === '["mount saddle #1 !7:saddle"]', 'a horse: fit a saddle from your pack');
page.once('dialog', d => d.dismiss());
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'pack horse' }).click(); await page.getByText('Release').click(); });
check(got.length === 0, 'Release asks first; declining sends nothing');
page.once('dialog', d => d.accept());
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'pack horse' }).click(); await page.getByText('Release').click(); });
check(JSON.stringify(got) === '["mount release #1"]', 'accepting sends mount release');
got = await sentNow(async () => { await page.locator('#company-inventory').getByRole('button', { name: 'Meal' }).click(); });
check(JSON.stringify(got) === '["company meal"]', 'the Meal button');
await page.evaluate(i => { const x = JSON.parse(JSON.stringify(i)); x.members[1].name = '<img src=x onerror="window.__xss=1">'; window.gmcp('Company.Inventory', x); }, inventory);
check(await page.evaluate(() => window.__xss === undefined) && (await invText()).includes('<img'), 'markup in a name renders as text');
await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);

// Review finding 3: an item's label is shown, its plain name used in commands.
await page.evaluate(i => { const x = JSON.parse(JSON.stringify(i)); x.members[0].worn[0].label = 'iron sword (sharp: 3)'; window.gmcp('Company.Inventory', x); }, inventory);
check((await invText()).includes('iron sword (sharp: 3)'), 'the label, with its edge, as text');
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'sharp' }).click(); await page.getByText('remove iron sword', { exact: true }).click(); });
check(JSON.stringify(got) === '["remove iron sword"]', 'commands use the plain name');
// Review findings 4 and 8: a menu from the keyboard; focus kept across updates.
await page.locator('#company-inventory button.cmp-item', { hasText: 'seared meat x6' }).focus();
await page.evaluate(() => { window.gmcp('Company.Vitals', { vitals: {}, rescue: {} }); });
await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);
check(await page.evaluate(() => document.activeElement && document.activeElement.textContent.startsWith('seared meat x6')), 'focus stays on the same item across updates');
got = await sentNow(async () => { await page.keyboard.press('Enter'); await page.keyboard.press('Enter'); });
check(JSON.stringify(got) === '["cargo take !3"]', 'an item\'s menu works from the keyboard');

// Camp: each button only when it would work.
await page.getByRole('tab', { name: 'Camp' }).click();
const campButtons = () => page.evaluate(() => [...document.querySelectorAll('#company-camp .cmp-btn')].map(b => b.textContent));
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: false, here: false, room: '', fire_lit: false, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: true, inn: false }));
check(JSON.stringify(await campButtons()) === '["Make camp","Meal"]', 'no camp here but allowed: Make camp');
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: false, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Light fire","Break camp","Meal"]', 'a cold camp here: Light fire, Break camp');
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Rest","Break camp","Meal"]', 'a lit fire: Rest');
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, resting: true, rest_percent: 25, rest_seconds: 45, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Meal"]' && await page.getByRole('progressbar', { name: 'Rest' }).count() === 1, 'resting: the progress bar, no Rest or Break');
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Resting: 45s left.'), 'the time left');
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, resting: false, rested: true, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Break camp","Meal"]' && (await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('has rested at this camp'), 'after a rest: no Rest (review finding 7)');
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: false, room: 'A Clearing', fire_lit: true, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: true }));
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Your camp is at A Clearing.') && JSON.stringify(await campButtons()) === '["Meal","Inn"]', 'a camp elsewhere; an inn here');
got = await sentNow(async () => { await page.locator('#company-camp').getByRole('button', { name: 'Inn' }).click(); });
check(JSON.stringify(got) === '["inn"]', 'the Inn button');
check(await page.getByRole('table', { name: 'Needs' }).count() === 1, 'each member\'s needs in a table');

// No company: the note, and the player's own inventory.
await page.evaluate(() => { window.gmcp('Company', {}); window.gmcp('Party', {}); });
await page.getByRole('tab', { name: 'Status' }).click();
check((await status()).includes('You travel alone'), 'no company: the note');

// Narrow: the dock at 280px in a 360px window, no sideways scrolling.
await page.evaluate(c => window.gmcp('Company', c), company);
await page.setViewportSize({ width: 360, height: 800 });
await page.evaluate(() => { const d = document.getElementById('dock-right'); d.style.width = '280px'; });
for (const tab of ['Status', 'Inventory', 'Camp']) {
  await page.getByRole('tab', { name: tab }).click();
  const over = await page.evaluate(() => { const p = document.querySelector('#company-window .cmp-panel:not([hidden])'); return p.scrollWidth - p.clientWidth; });
  check(over <= 0, tab + ': no horizontal overflow at 280px (' + over + ')');
}
check(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), 'no horizontal page scroll at 360px');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'company-narrow.png') }); }
await page.setViewportSize({ width: 1280, height: 900 });
await page.evaluate(() => { document.getElementById('dock-right').style.width = ''; });

// --- Task 11: the Combat tab (Setup) ---
await page.evaluate(c => window.gmcp('Company', c), company);
await page.getByRole('tab', { name: 'Combat' }).click();
const combat = () => page.evaluate(() => document.getElementById('combat-window').textContent);
check(await page.getByRole('table', { name: /Formation/ }).filter({ has: page.locator('.cbt-role') }).count() === 1, 'Combat: the formation grid, with roles');
check(await page.getByRole('button', { name: 'Oswin: healer, weakest' }).count() === 1, 'a row per member: role and target rule');
check(await page.getByRole('button', { name: 'Scout' }).count() === 0, 'no Scout with no enemies here');
got = await sentNow(async () => { await page.getByRole('button', { name: 'Oswin: healer, weakest' }).click(); await page.getByText('Role: caster').click(); });
check(JSON.stringify(got) === '["strategy #1 caster"]', 'set a companion\'s role');
got = await sentNow(async () => { await page.getByRole('button', { name: 'Oswin: healer, weakest' }).click(); await page.getByText('Target: defend').click(); });
check(JSON.stringify(got) === '["strategy #1 target defend"]', 'set a companion\'s target rule');
await page.getByRole('button', { name: /^Wren \(you\)/ }).click();
const youMenu = await page.evaluate(() => [...[...document.querySelectorAll('body > div')].pop().children].map(c => c.textContent));
check(!youMenu.includes('Target: assist') && youMenu.includes('Swap with Oswin') && youMenu.includes('Move to row 3, column 3'), 'your menu: no assist; swap and move by the grid');
got = await sentNow(async () => { await page.getByText('Move to row 3, column 3').click(); });
check(JSON.stringify(got) === '["formation move me 3 3"]', 'move yourself');
got = await sentNow(async () => { await page.getByRole('button', { name: 'Oswin: healer, weakest' }).click(); await page.getByText('Swap with Wren').click(); });
check(JSON.stringify(got) === '["formation swap #1 me"]', 'swap two members');
got = await sentNow(async () => { await page.getByRole('button', { name: 'Oswin: healer, weakest' }).click(); await page.getByText('Back to the default').click(); });
check(JSON.stringify(got) === '["strategy #1 default"]', 'back to the default');
await page.evaluate(() => window.gmcp('Room.Info', { Contents: { Npcs: [{ id: '#9', name: 'ruffian', aggro: false, group: 'a band of ruffians' }], Players: [], Items: [], Containers: [] } }));
got = await sentNow(async () => { await page.getByRole('button', { name: 'Scout' }).click(); });
check(JSON.stringify(got) === '["scout"]', 'Scout appears with an enemy group here, and sends scout');
await page.evaluate(() => { const c = JSON.parse(JSON.stringify(Client.GMCPStructs.Company)); c.members[1].name = '<img src=x onerror="window.__xss2=1">'; window.gmcp('Company', c); });
check(await page.evaluate(() => window.__xss2 === undefined) && (await combat()).includes('<img'), 'markup in a name renders as text');

// --- Task 12: the Comm and Who tabs ---
await page.getByRole('tab', { name: 'Combat' }).click();
await page.evaluate(() => { window.gmcp('Comm.Channel', { channel: 'say', sender: 'Oswin', source: 'mob', text: 'Hello' }); window.gmcp('Comm.Channel', { channel: 'say', sender: 'Oswin', source: 'mob', text: 'Again' }); });
check(await page.getByRole('tab', { name: 'Comm, 2 new' }).count() === 1, 'messages while another tab shows: a count on Comm');
await page.getByRole('tab', { name: /^Comm/ }).click();
check(await page.getByRole('tab', { name: 'Comm' }).count() === 1 && await page.evaluate(() => document.querySelector('.dock-tabgroup-badge:not([hidden])') === null), 'opening Comm clears it');
await page.evaluate(() => window.gmcp('Comm.Channel', { channel: 'say', sender: 'Oswin', source: 'mob', text: 'Seen' }));
check(await page.evaluate(() => document.querySelector('.dock-tabgroup-badge:not([hidden])') === null), 'no count while Comm shows');
const dockTabs = () => page.evaluate(() => [...document.querySelectorAll('#dock-right [role=tab].dock-tabgroup-tab')].map(t => t.textContent));
check(JSON.stringify(await dockTabs()) === '["Character","Company","Combat","Comm"]', 'the dock\'s tabs: Character, Company, Combat, Comm');
await page.evaluate(() => VirtualWindows.getWindows().find(w => w._id === 'Online').reopen());
check(JSON.stringify(await dockTabs()) === '["Character","Company","Combat","Comm","Who"]', 'Who appears when Online is enabled');
await page.evaluate(() => VirtualWindows.getWindows().find(w => w._id === 'KillStats').reopen());
check((await dockTabs()).includes('Kills'), 'Kills appears when Kill Stats is enabled');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'dock-full.png') }); }

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all dock window checks passed');
