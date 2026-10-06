// Phase 32g browser check: drives the company dock's real window scripts
// (through dock-windows-harness.html, with the real web client core) in
// Chromium with Playwright, feeding them GMCP payloads and recording the
// commands they send.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/dock-windows-check.mjs [outdir]
import { createRequire } from 'node:module';
import path from 'node:path';
import { readdirSync } from 'node:fs';
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
  leader: { key: 'leader', id: 0, name: 'Wren', status: 'present', level: 5, archetype: 'Ranger', class: 'warden', class_name: 'Warden', cell: { row: 0, col: 0 }, chemistry: null, strategy: { role: 'fighter', target: 'weakest' } },
  members: [
    { key: 'companion:1', id: 1, name: 'Oswin', status: 'present', level: 3, archetype: 'Cleric', cell: { row: 1, col: 1 }, chemistry: 'Trusted', strategy: { role: 'healer', target: 'weakest' } },
    { key: 'companion:2', id: 2, name: xss, status: 'present', level: 2, archetype: 'Warrior', cell: { row: 0, col: 1 }, chemistry: null, strategy: { role: 'fighter', target: 'leader' } },
    { key: 'companion:3', id: 3, name: 'Tamsin', status: 'awaiting', level: 2, archetype: 'Ranger', cell: null, chemistry: null, strategy: { role: 'fighter', target: 'weakest' } },
    { key: 'companion:4', id: 4, name: 'Ysolde', status: 'dead', level: 5, archetype: 'Wizard', cell: { row: 2, col: 1 }, chemistry: null, strategy: { role: 'caster', target: 'weakest' } },
  ],
  alive: 4, dead: 1,
  load: { label: 'Heavily burdened', total_g: 46000, capacity_g: 50000, cargo_g: 3000, companion_g: 20000 },
  activity: 'Camped', rest: { tier: 'Rested', seconds: 600 }, checkpoint: 'The Chapel',
  tactics: { focus: 'none', healing: 50 },
  rescue: { 'companion:4': 5400 },
  vitals: {
    leader: { hp: 30, hp_max: 40, mp: 6, mp_max: 14, needs: { hunger: need(40, 'Hungry', true), thirst: need(90, 'Hydrated', false), fatigue: need(80, 'Rested', false) }, warmth: '' },
    'companion:1': { hp: 12, hp_max: 25, mp: 8, mp_max: 20, needs: { hunger: need(70, 'Sated', false), thirst: need(20, 'Parched', true), fatigue: null }, warmth: 'Chilled' },
    'companion:2': { hp: 20, hp_max: 20, needs: null, warmth: '' },
    'companion:3': { hp: null, hp_max: null, needs: null, warmth: null },
    'companion:4': { hp: null, hp_max: null, needs: null, warmth: null },
  },
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto(process.env.DOCK_HARNESS_URL || 'file://' + path.join(here, 'dock-windows-harness.html'));
await page.evaluate(() => localStorage.clear());
// The battle screen (40f) opens over the dock on every battle and would
// cover the Combat tab checked here; battle-check.mjs covers the screen.
await page.evaluate(() => localStorage.setItem('ashveil-battle-screen', 'manual'));
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
// Phase 30b: a wounded member's limit, shown, spoken, and warned of.
await page.evaluate(() => window.gmcp('Company.Vitals', { vitals: {
  leader: { hp: 30, hp_max: 40, mp: 6, mp_max: 14, needs: null, warmth: '' },
  'companion:1': { hp: 5, hp_max: 25, hp_limit: 19, mp: 2, mp_max: 20, needs: null, warmth: '' },
}, rescue: { 'companion:4': 60 } }));
const wounded = await page.evaluate(() => { const r = document.querySelector('.vitals-member[data-key="companion:1"]'); return { label: r.getAttribute('aria-label'), text: r.textContent }; });
check(wounded.label === 'Oswin, health 5 of 25, wound limit 19, mana 2 of 20' && wounded.text.includes('limit 19'), 'a wounded member shows and speaks the wound limit');
check(await page.evaluate(() => document.getElementById('vitals-warn').textContent.includes('Oswin: wounded, limit 19')), 'and the warnings name it');
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
// Phase 38c1: the character window names the promoted class, tier and rank.
await page.evaluate(() => window.gmcp('Char.Info', { name: 'Wren', class: 'ranger', race: 'Human', level: 52, route: 'Marksman', tier: 'elite', rank: 50, promotion: '' }));
check((await page.evaluate(() => document.getElementById('cw-char-name').textContent)).includes('Marksman (elite), rank 50'), 'the character window shows the class, tier and rank');
await page.evaluate(() => window.gmcp('Char.Info', { name: 'Wren', class: 'ranger', race: 'Human', level: 30, route: 'Warden', tier: 'advanced', rank: 25, promotion: 'waiting-gate' }));
check((await page.evaluate(() => document.getElementById('cw-char-name').textContent)).includes('Promotion waits on alignment'), 'and a promotion waiting on alignment');
await page.evaluate(() => window.gmcp('Char.Info', { name: 'Wren', class: 'ranger', race: 'Human', level: 5 }));
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
check((await page.evaluate(() => window.sent)).includes('drink !2:water'), 'the gear menus use the exact item reference');
for (const [verb, type, subtype] of [['eat', 'food', 'edible'], ['use', 'object', 'usable'], ['read', 'readable', '']]) {
  await page.evaluate(({type, subtype}) => {
    window.sent = [];
    window.gmcp('Char.Inventory', { Worn: {}, Backpack: { items: [
      {id: '!3:first', name: 'same supply', type, subtype, details: []},
      {id: '!3:second', name: 'same supply', type, subtype, details: []}
    ], Summary: {count: 2, weight_g: 500} } });
  }, {type, subtype});
  await page.locator('.gw-bp-row').nth(1).click();
  await page.getByText(verb + ' same supply', {exact: true}).click();
  check(JSON.stringify(await page.evaluate(() => window.sent)) === JSON.stringify([verb + ' !3:second']), verb + ' selects the clicked duplicate instance');
}
await page.getByRole('tab', { name: 'Skills' }).click();
check(await page.evaluate(() => !document.getElementById('cw-skills-tab').textContent.includes('Jobs') && !document.getElementById('cw-jobs')), 'the stock Jobs section is gone from Skills (Phase 57)');
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
check(await page.getByRole('listitem', { name: /Wren, level 5, Warden/ }).count() === 1
  && await page.evaluate(() => document.querySelector('#party-panel [data-key=leader] .party-member-rank').title) === 'Ranger line', 'a promoted member card names its class, its lineage on hover (40s5)');
check(!(await status()).includes('Travelling with'), 'no human party: no Travelling with');
await page.evaluate(() => window.gmcp('Party', { Leader: 'Wren', Members: [{ Name: 'Wren', Position: 'leader', owner_user_id: 7, online: true, follow: false, support: true, autoattack: false }, { Name: '<b>Tamsin</b>', Position: 'member', owner_user_id: 8, online: false, follow: false, support: false, autoattack: false }], Invited: [], Vitals: { Wren: { health: 80, level: 5, location: 'Dunmar' }, '<b>Tamsin</b>': { health: 40, level: 4, location: 'Dunmar' } } }));
check((await status()).includes('Travelling with') && (await status()).includes('<b>Tamsin</b>') && await page.locator('#party-panel b').count() === 0, 'a human party under Travelling with, names as text');
check((await status()).includes('each owner commands their own company') && (await status()).includes('(offline)'), 'alliance authority and offline status appear');
await page.locator('.company-member[data-key="companion:1"]').focus();
await page.evaluate(() => window.gmcp('Company.Vitals', { vitals: { 'companion:1': { hp: 5, hp_max: 25, needs: null, warmth: null } }, rescue: { 'companion:4': 5340 } }));
check((await status()).includes('5/25') && await page.evaluate(() => document.activeElement && document.activeElement.getAttribute('data-key')) === 'companion:1', 'a vitals update applies, keeping focus on the card');
// Phase 38c1: a promoted member shows its class with an elite badge and rank,
// and a promotion that is ready or waiting on a gate.
await page.evaluate(c => { const g = JSON.parse(JSON.stringify(c)); g.members[0].class = 'paladin'; g.members[0].class_name = 'Paladin'; g.members[0].tier = 'elite'; g.members[0].rank = 45; g.members[1].class = 'knight'; g.members[1].class_name = 'Knight'; g.members[1].tier = 'advanced'; g.members[1].rank = 25; g.members[1].promotion = 'waiting-gate'; g.leader.promotion = 'ready'; window.gmcp('Company', g); }, company);
check((await status()).includes('\u2605 elite, rank 45') && (await status()).includes('advanced, rank 25'), 'a card shows the class, the elite badge and the rank');
check((await status()).includes('Promotion ready') && (await status()).includes('Promotion waiting on alignment'), 'a card marks a promotion ready or waiting on alignment');
check(await page.getByRole('listitem', { name: /Oswin, level 3, Paladin, elite rank 45, Health 12 of 25/ }).count() === 1, 'the spoken summary names the class');
await page.evaluate(c => window.gmcp('Company', c), company);
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
check(JSON.stringify(got) === '["remove !1:sword"]', 'commands use the exact item reference');
// Review findings 4 and 8: a menu from the keyboard; focus kept across updates.
await page.locator('#company-inventory button.cmp-item', { hasText: 'seared meat x6' }).focus();
await page.evaluate(() => { window.gmcp('Company.Vitals', { vitals: {}, rescue: {} }); });
await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);
check(await page.evaluate(() => document.activeElement && document.activeElement.textContent.startsWith('seared meat x6')), 'focus stays on the same item across updates');
got = await sentNow(async () => { await page.keyboard.press('Enter'); await page.keyboard.press('Enter'); });
check(JSON.stringify(got) === '["cargo take !3"]', 'an item\'s menu works from the keyboard');

// 33g: shared cargo, exact assignments, comparison and treasury.
const sharedInventory = JSON.parse(JSON.stringify(inventory));
sharedInventory.shared = true;
sharedInventory.treasury = 123;
sharedInventory.autoloot = false;
sharedInventory.containers = [
 { key: 'leader', name: 'cloth knapsack', carrier: 'Dain', ref: '!38:pack', capacity_g: 10000, available: true },
 { key: 'companion:2', name: '<img src=x onerror=alert(1)> frame pack', carrier: 'Away companion', capacity_g: 15000, available: false }
];
sharedInventory.members[1].available = true;
sharedInventory.cargo = [...sharedInventory.members[0].carried, ...sharedInventory.cargo,
  { ref: '!8:glaive', name: 'steel glaive', grams: 2000, count: 1, type: 'weapon', subtype: 'slashing' }];
sharedInventory.members[0].carried = [];
await page.evaluate(i => window.gmcp('Company.Inventory', i), sharedInventory);
check((await invText()).includes('Company treasury: 123 gold'), 'shared treasury is visible');
check((await invText()).includes('cloth knapsack (Dain)') && (await invText()).includes('15.0 kg unavailable'), 'assigned container cards show capacity and unavailable carriers');
check(await page.locator('#company-inventory img').count() === 0, 'container names render as text');
check(!/Wearing|Carrying|No pack/.test(await invText()), 'shared inventory contains no personal or worn blocks');
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'steel glaive' }).click(); await page.getByText('Compare for Brother Oswin', { exact: true }).click(); });
check(JSON.stringify(got) === '["company compare #1 !8:glaive"]', 'comparison names the stable member and exact cargo instance');
got = await sentNow(async () => { await page.locator('#company-inventory button.cmp-item', { hasText: 'steel glaive' }).click(); await page.getByText('Equip Brother Oswin', { exact: true }).click(); });
check(JSON.stringify(got) === '["company equip #1 !8:glaive"]', 'equipment assignment uses the same backend command');
got = await sentNow(async () => { await page.locator('#company-inventory').getByRole('button', { name: 'Autoloot on', exact: true }).click(); });
check(JSON.stringify(got) === '["autoloot on"]', 'autoloot is an explicit opt-in');

// Phase 48: every member's equipment slots, tap-to-pick and drag-and-drop.
sharedInventory.slots = [{ slot: 'weapon', label: 'Weapon' }, { slot: 'offhand', label: 'Offhand' }, { slot: 'body', label: 'Body' }, { slot: 'pack', label: 'Pack' }];
sharedInventory.cargo.push({ ref: '!9:coat', name: 'quilted coat', grams: 900, count: 1, type: 'body', subtype: 'wearable' });
sharedInventory.members[1].worn = [{ ref: '!5:club', name: 'oak club', label: 'oak club', grams: 1200, count: 1, uses: 0, uses_max: 0, type: 'weapon', subtype: 'blunt', slot: 'weapon' }];
await page.evaluate(i => window.gmcp('Company.Inventory', i), sharedInventory);
const equipBox = page.locator('#company-inventory [aria-label="Brother Oswin equipment"]');
check((await equipBox.textContent()).includes('Weapon') && (await equipBox.textContent()).includes('oak club') && (await equipBox.textContent()).includes('Body') && (await equipBox.textContent()).includes('empty'), 'shared inventory lists a companion\'s slots, filled and empty');
check((await page.locator('#company-inventory [aria-label="Wren (you) equipment"]').textContent()).includes('iron sword'), 'your own slots are listed too');
got = await sentNow(async () => { await equipBox.locator('button', { hasText: 'Body' }).click(); await page.getByText('Equip quilted coat', { exact: true }).click(); });
check(JSON.stringify(got) === '["company equip #1 !9:coat body"]', 'tap an empty slot, pick a cargo item: equips the companion with an exact reference and slot');
got = await sentNow(async () => { await equipBox.locator('button', { hasText: 'oak club' }).click(); await page.getByText('Remove to cargo', { exact: true }).click(); });
check(JSON.stringify(got) === '["company remove #1 weapon"]', 'tap a worn slot: remove to cargo');
// Drag and drop: a cargo item onto a member's slot, and worn gear onto the cargo.
const dropData = async (from, to, payload) => page.evaluate(([f, t, p]) => {
  const src = document.querySelector(f), dst = document.querySelector(t);
  const dt = new DataTransfer();
  dt.setData('application/x-ashveil-gear', JSON.stringify(p));
  window.sent = [];
  dst.dispatchEvent(new DragEvent('dragover', { dataTransfer: dt, bubbles: true, cancelable: true }));
  dst.dispatchEvent(new DragEvent('drop', { dataTransfer: dt, bubbles: true, cancelable: true }));
  return window.sent;
}, [from, to, payload]);
const slotSel = '#company-inventory [aria-label="Brother Oswin equipment"] .cmp-slot:nth-child(1)';
check(JSON.stringify(await dropData('x', '#company-inventory [aria-label="Brother Oswin equipment"] li:nth-child(3) .cmp-slot', { from: 'cargo', ref: '!9:coat', type: 'body' })) === '["company equip #1 !9:coat body"]', 'dropping a cargo item on a fitting slot equips it');
check((await dropData('x', '#company-inventory [aria-label="Brother Oswin equipment"] li:nth-child(1) .cmp-slot', { from: 'cargo', ref: '!9:coat', type: 'body' })).length === 0, 'dropping a cargo item on a slot it does not fit does nothing');
check(JSON.stringify(await dropData('x', '#company-inventory [aria-label="Brother Oswin equipment"]', { from: 'cargo', ref: '!9:coat', type: 'body' })) === '["company equip #1 !9:coat"]', 'dropping a cargo item on a member equips it where it fits');
check(JSON.stringify(await dropData('x', '#company-inventory section[aria-label="Cargo"]', { from: 'worn', member: '#1', slot: 'weapon', ref: '!5:club' })) === '["company remove #1 weapon"]', 'dropping worn gear on the cargo takes it off');
check(await page.locator('#company-inventory [aria-label="Brother Oswin equipment"] .cmp-slot[draggable=true]').count() === 1 && await page.locator('#company-inventory section[aria-label="Cargo"] .cmp-item[draggable=true]').count() > 0, 'worn gear and cargo items are draggable');
if (outdir) { await page.locator('#company-inventory').screenshot({ path: path.join(outdir, 'companion-gear-company-panel.png') }); }
// 38e review: a creature shows only the slots its body has, and only gear cut
// for its species is offered to it (and to no one else).
{
  const withHound = JSON.parse(JSON.stringify(sharedInventory));
  withHound.members.push({ key: 'companion:5', name: 'Brindle', available: true, grams: 0, worn: [], carried: [], closed: ['weapon', 'offhand'], species: 'hound' });
  withHound.cargo.push({ ref: '!20500:harness', name: 'hound harness', grams: 900, count: 1, type: 'body', subtype: 'wearable', worn_by: ['hound'] });
  await page.evaluate(i => window.gmcp('Company.Inventory', i), withHound);
  const houndBox = page.locator('#company-inventory [aria-label="Brindle equipment"]');
  const houndText = await houndBox.textContent();
  check(!houndText.includes('Weapon') && !houndText.includes('Offhand') && houndText.includes('Body'), 'a hound lists only the slots its body has');
  await houndBox.locator('button', { hasText: 'Body' }).click();
  const houndMenu = await page.locator('body').textContent();
  check(houndMenu.includes('Equip hound harness') && !houndMenu.includes('Equip quilted coat'), 'a hound is offered only gear cut for it');
  await page.keyboard.press('Escape');
  await equipBox.locator('button', { hasText: 'Body' }).click();
  check(!(await page.locator('body').textContent()).includes('Equip hound harness'), 'a person is not offered gear cut for a creature');
  await page.keyboard.press('Escape');
  check((await dropData('x', '#company-inventory [aria-label="Brindle equipment"]', { from: 'cargo', ref: '!9:coat', type: 'body', worn_by: [] })).length === 0, 'dropping ordinary gear on a hound does nothing');
}
await page.evaluate(i => window.gmcp('Company.Inventory', i), sharedInventory);

// Phase 34a: inherited black text is visible even under a dark theme.
// Check primary load text and secondary labels in every shipped theme.
for (const theme of readdirSync(path.join(here, '../../_datafiles/html/public/static/css')).filter(n => /^theme-.*\.css$/.test(n))) {
    await page.evaluate(theme => new Promise(resolve => {
        const link = document.getElementById('theme-css');
        link.onload = resolve;
        link.href = '../../_datafiles/html/public/static/css/' + theme;
    }), theme);
    const ratios = await page.evaluate(() => {
        function luminance(color) {
            const v = color.match(/[\d.]+/g).slice(0, 3).map(n => Number(n) / 255).map(n => n <= 0.04045 ? n / 12.92 : Math.pow((n + 0.055) / 1.055, 2.4));
            return v[0] * 0.2126 + v[1] * 0.7152 + v[2] * 0.0722;
        }
        function ratio(fg, bg) {
            const a = luminance(fg), b = luminance(bg);
            return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
        }
        // Simulate the actual window parent, which need not inherit body color.
        document.getElementById('company-window').parentElement.style.color = 'black';
        const bg = getComputedStyle(document.getElementById('company-window')).backgroundColor;
        return [...document.querySelectorAll('#company-inventory .cmp-pad > div:not(.cmp-meter):not(.cmp-actions)')].map(e => ratio(getComputedStyle(e).color, bg));
    });
    check(ratios.length > 0 && ratios.every(n => n >= 4.5), theme + ': load and capacity text contrast >= 4.5');
}
await page.evaluate(() => new Promise(resolve => { const link = document.getElementById('theme-css'); link.onload = resolve; link.href = '../../_datafiles/html/public/static/css/theme-brooding.css'; }));

await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);

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
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: false, embers: true, tent: true, gear: ['Bedrolls 2/3', 'Tent', 'Bells and trip lines'], resting: false, rested: true, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Feed fire","Break camp","Meal"]' && (await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('burned to embers'), 'after a rest: embers, Feed fire, no Rest until fed (40a3)');
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('oiled canvas tent'), 'a pitched tent shows (40a3)');
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Camp gear: Bedrolls 2/3, Tent, Bells and trip lines.'), 'the camp gear line (40a4)');
check(!(await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Thieves work'), 'no thieves warning without theft_risk (40a4)');
if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '40a4-camp-gear.png') }); }
// Phase 43a: camp supplies carried, and what is set by for the next rest.
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, embers: false, tent: true, gear: ['Tent'], supplies: ['fortifying broth x2', 'warming draught x1', 'watch incense x1'], prepared: ['fortifying broth for Oswin', 'watch incense'], resting: false, rested: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Supplies: fortifying broth x2, warming draught x1, watch incense x1 (camp prepare).'), 'the camp supplies line (43a)');
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Set by for the next rest: fortifying broth for Oswin, watch incense.'), 'what is set by for the rest (43a)');
if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '43a-camp-supplies.png') }); }
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, embers: false, tent: false, gear: [], resting: false, rested: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(!(await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Set by for the next rest') && !(await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Supplies:'), 'no supplies lines with none carried or queued (43a)');
// Phase 49: the company's latest talk shows under the camp, and only when there is some.
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false, banter: [{ name: 'Hild Marrow', verb: 'mutters', text: 'Keep the fire low.' }, { name: 'Brann', verb: 'remarks', text: 'Cheerful as ever, Hild.' }] }));
{
  const camp = await page.evaluate(() => document.getElementById('company-camp').textContent);
  check(camp.includes('Around the fire') && camp.includes('Hild Marrow mutters, \u201cKeep the fire low.\u201d') && camp.includes('Brann remarks'), 'the banter lines on the Camp tab (49)');
}
if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '49-camp-banter.png') }); }
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(!(await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Around the fire'), 'no banter block before any talk (49)');
// Phase 51: the rest duty picker, a row per member at the camp.
{
  const opts = ['sleep', 'watch', 'tend', 'forage', 'cook'];
  const duties = [
    { key: 'leader', name: 'Wren', command: 'me', duty: 'sleep', options: opts },
    { key: 'companion:1', name: 'Brother Oswin', command: 'Brother Oswin', duty: 'watch', options: opts },
    { key: 'companion:5', name: 'Mira', command: 'Mira', duty: 'brew', options: [...opts, 'brew'] },
  ];
  const campWith = (locked) => ({ has_camp: true, here: true, room: '', fire_lit: true, resting: locked, rest_percent: 0, rest_seconds: 30, can_camp: false, inn: false, duties, duties_locked: locked });
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith(false));
  const rows = await page.evaluate(() => [...document.querySelectorAll('#company-camp .cmp-duty')].map(r => ({
    name: r.querySelector('.cmp-duty-name').textContent,
    pressed: [...r.querySelectorAll('button[aria-pressed="true"]')].map(b => b.textContent),
    count: r.querySelectorAll('button').length,
  })));
  check(rows.length === 3 && rows[0].name === 'Wren (you)' && rows[1].pressed.join() === 'Watch' && rows[2].pressed.join() === 'Brew' && rows[2].count === 6 && rows[0].count === 5, 'duty rows: one per member, the duty pressed, Brew only for an Alchemist (51)');
  got = await sentNow(async () => { await page.getByRole('group', { name: 'Brother Oswin duty' }).getByRole('button', { name: 'Cook' }).click(); });
  check(JSON.stringify(got) === '["camp duties Brother Oswin cook"]', 'a duty button sends camp duties (51)');
  if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '51-camp-duties.png') }); }
  await page.setViewportSize({ width: 360, height: 800 });
  check(await page.evaluate(() => { const p = document.getElementById('company-camp'); return p.scrollWidth <= p.clientWidth + 1; }), 'the duty picker fits a phone (51)');
  if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '51-camp-duties-phone.png') }); }
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith(true));
  check(await page.evaluate(() => [...document.querySelectorAll('#company-camp .cmp-duty button')].every(b => b.disabled)) && (await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('fixed for this rest'), 'duties lock while resting (51)');
}
// Phase 52: the pitched tent, and a picker when more than one tent is carried.
{
  const tents = [
    { kind: 'canvas', name: 'oiled canvas tent', effect: 'shelter', pitched: false, command: 'camp tent canvas' },
    { kind: 'large', name: 'large pavilion tent', effect: 'everyone wakes Well Rested', pitched: true, command: 'camp tent large' },
  ];
  const campWith = (extra) => ({ has_camp: true, here: true, room: '', fire_lit: true, resting: false, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false, tent: true, tent_kind: 'large', tent_name: 'large pavilion tent', tent_note: 'everyone wakes Well Rested', tents, gear: ['Large tent'], ...extra });
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith({}));
  check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('A large pavilion tent is pitched here: everyone wakes Well Rested.'), 'the pitched tent is named with its effect (52)');
  const picks = await page.evaluate(() => [...document.querySelectorAll('#company-camp [aria-label="Tent choice"] button')].map(b => ({ text: b.textContent, pressed: b.getAttribute('aria-pressed') })));
  check(picks.length === 2 && picks[1].text === 'Large pavilion tent' && picks[1].pressed === 'true' && picks[0].pressed === 'false', 'a button per tent carried, the pitched one pressed (52)');
  got = await sentNow(async () => { await page.getByRole('group', { name: 'Tent choice' }).getByRole('button', { name: 'Oiled canvas tent' }).click(); });
  check(JSON.stringify(got) === '["camp tent canvas"]', 'a tent button sends camp tent (52)');
  if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '52-camp-tents.png') }); }
  await page.setViewportSize({ width: 360, height: 800 });
  check(await page.evaluate(() => { const p = document.getElementById('company-camp'); return p.scrollWidth <= p.clientWidth + 1; }), 'the tent picker fits a phone (52)');
  if (outdir) { await page.locator('#company-camp').screenshot({ path: path.join(outdir, '52-camp-tents-phone.png') }); }
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith({ resting: true, rest_seconds: 20 }));
  check(await page.evaluate(() => [...document.querySelectorAll('#company-camp [aria-label="Tent choice"] button')].every(b => b.disabled)), 'the tent is fixed while resting (52)');
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith({ tents: [tents[1]] }));
  check(await page.evaluate(() => document.querySelectorAll('#company-camp [aria-label="Tent choice"]').length) === 0, 'no picker with one tent carried (52)');
  await page.evaluate(c => window.gmcp('Company.Camp', c), campWith({ tent_kind: 'canvas', tent_name: 'oiled canvas tent', tent_note: 'shelter' }));
  check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('An oiled canvas tent is pitched here'), 'a tent name starting with a vowel takes "An" (52 review)');
}
await page.evaluate(() => window.gmcp('Company.Camp', { has_camp: true, here: true, room: '', fire_lit: true, embers: false, tent: false, gear: [], theft_risk: true, resting: false, rested: true, rest_percent: 0, rest_seconds: 0, can_camp: false, inn: false }));
check(JSON.stringify(await campButtons()) === '["Rest","Break camp","Meal"]', 'a refed fire: Rest again (40a3)');
check((await page.evaluate(() => document.getElementById('company-camp').textContent)).includes('Thieves work this road'), 'no bells on a thieves road: the warning (40a4)');
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
// Phase 30c2: guardians.
await page.getByRole('button', { name: /^Wren \(you\)/ }).click();
const roleMenu = await page.evaluate(() => [...[...document.querySelectorAll('body > div')].pop().children].map(c => c.textContent));
check(roleMenu.includes('Role: guardian') && !roleMenu.some(l => l.startsWith('Guard: ')), 'guardian is a role; no ward items for a non-guardian');
await page.keyboard.press('Escape');
await page.mouse.click(5, 5);
await page.evaluate(c => { const g = JSON.parse(JSON.stringify(c)); g.members[0].strategy = { role: 'guardian', target: 'weakest', ward: 'leader' }; window.gmcp('Company', g); }, company);
check(await page.getByRole('button', { name: 'Oswin: guardian, weakest, guards you' }).count() === 1, 'a guardian\'s row says whom it guards');
await page.getByRole('button', { name: 'Oswin: guardian, weakest, guards you' }).click();
const gMenu = await page.evaluate(() => [...[...document.querySelectorAll('body > div')].pop().children].map(c => c.textContent));
check(gMenu.includes('Guard: the most hurt') && !gMenu.includes('Guard: you') && !gMenu.includes('Guard: Ysolde') && !gMenu.includes('Guard: Oswin') && !gMenu.includes('Role: guardian'), 'a guardian\'s menu: other wards and the most hurt, not the fallen or itself (' + gMenu.filter(l => l.startsWith('Guard')).join(', ') + ')');
got = await sentNow(async () => { await page.getByText('Guard: the most hurt').click(); });
check(JSON.stringify(got) === '["strategy #1 guard"]', 'guard the most hurt');
got = await sentNow(async () => { await page.getByRole('button', { name: /^Oswin: guardian/ }).click(); await page.getByText('Guard: Tamsin').click(); });
check(JSON.stringify(got) === '["strategy #1 guard #3"]', 'guard another member');
await page.evaluate(c => { const g = JSON.parse(JSON.stringify(c)); g.members[0].strategy = { role: 'guardian', target: 'weakest', ward: 'leader', ward_reach: false }; window.gmcp('Company', g); }, company);
check(await page.getByRole('button', { name: 'Oswin: guardian, weakest, guards you (out of reach)' }).count() === 1, 'a ward out of reach is marked');
await page.evaluate(c => window.gmcp('Company', c), company);
// Phase 33e: class abilities, abilities off, and a mana reserve.
await page.evaluate(c => { const g = JSON.parse(JSON.stringify(c)); g.members[0].strategy = { role: 'fighter', target: 'weakest', abilities: ['Tackle'] }; g.members[1].strategy = { role: 'caster', target: 'weakest', abilities_off: true, reserve: 30 }; window.gmcp('Company', g); }, company);
check(await page.getByRole('button', { name: 'Oswin: fighter, weakest, Tackle' }).count() === 1, 'a member\'s row names its abilities');
check(await page.getByRole('button', { name: /: caster, weakest, abilities off, keeps 30% mana$/ }).count() === 1, 'abilities off and a reserve are shown');
got = await sentNow(async () => { await page.getByRole('button', { name: 'Oswin: fighter, weakest, Tackle' }).click(); await page.getByText('Abilities: off').click(); });
check(JSON.stringify(got) === '["strategy #1 abilities off"]', 'the member menu turns abilities off');
got = await sentNow(async () => { await page.getByRole('button', { name: /abilities off, keeps 30% mana$/ }).click(); await page.getByText('Abilities: on').click(); });
check(JSON.stringify(got) === '["strategy #2 abilities on"]', 'and back on');
await page.evaluate(c => window.gmcp('Company', c), company);
// Phase 30c: the tactics row.
check(await page.getByRole('button', { name: 'Company tactics: focus none, heal below 50%' }).count() === 1, 'Setup shows the saved tactics');
got = await sentNow(async () => { await page.getByRole('button', { name: /^Company tactics/ }).click(); await page.getByText('Heal below 70%').click(); });
check(JSON.stringify(got) === '["company tactics healing 70"]', 'the tactics menu sets the healing threshold');
got = await sentNow(async () => { await page.getByRole('button', { name: /^Company tactics/ }).click(); await page.getByText('Focus: casters').click(); });
check(JSON.stringify(got) === '["company tactics focus casters"]', 'the tactics menu sets the focus');
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

// --- Phase 32g2: the Battle view ---
const battleFix = {
  group: 'a band of cutthroats',
  retreat: { exit: 'east', rounds: 2 },
  enemies: [
    { id: 'm:412', label: 'the cutthroat captain', cell: { row: 0, col: 0 }, health: 'wounded', reach: true, target: 'leader' },
    { id: 'm:413', label: 'the first cutthroat', cell: { row: 0, col: 1 }, health: 'unhurt', reach: true, target: 'companion:1' },
    { id: 'm:414', label: 'the slinger', cell: { row: 1, col: 1 }, health: 'near death', reach: false, target: 'u:9' },
    { id: 'm:416', label: '<img src=x onerror="window.__xss3=1">', cell: { row: 0, col: 2 }, health: 'unhurt', reach: true },
  ],
  fallen: [{ id: 'm:415', label: 'the bruiser' }],
  company: [{ key: 'leader', target: 'm:412' }, { key: 'companion:1', target: 'm:412' }, { key: 'companion:2', target: 'm:413' }],
  others: [{ id: 'u:9', name: 'Brannoc' }],
  waiting: ['a pack of grey wolves'],
};
await page.evaluate(c => window.gmcp('Company', c), company);
await page.getByRole('tab', { name: 'Character' }).click();
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);
// Phase 30f: effective combat cells and reserve presentation are separate
// from the saved Setup formation.
await page.evaluate(b => {
  const x = JSON.parse(JSON.stringify(b));
  x.narrow = true;
  x.positions = { leader: {row: 2, col: 1}, 'companion:1': {row: 1, col: 0}, 'companion:2': {row: 0, col: 1} };
  x.enemies[3].cell = {row: 1, col: 2};
  window.gmcp('Company.Battle', x);
}, battleFix);
check(await page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-field')].every(g => g.style.gridTemplateColumns === 'repeat(2, minmax(0px, 1fr))' || g.style.gridTemplateColumns === 'repeat(2, minmax(0, 1fr))')), 'narrow ground displays two active columns');
check(await page.locator('#combat-window .cbt-reserve').count() === 1, 'enemy overflow is visibly marked Reserve');
check(await page.evaluate(() => {
  const g = [...document.querySelectorAll('#combat-window .cbt-field')].find(g => g.getAttribute('aria-label') === 'Your company');
  return g && [...g.children].findIndex(n => n.dataset && n.dataset.fid === 'leader') === 5;
}), 'Battle uses the effective leader cell, while Setup stays saved');
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);

check(await page.getByRole('tab', { name: 'Combat, a battle is under way' }).count() === 1, 'a battle while another tab shows: a marker on Combat');
await page.getByRole('tab', { name: /^Combat/ }).click();
check(await page.getByRole('tab', { name: 'Combat' }).count() === 1, 'opening Combat clears the marker');
await page.waitForFunction(() => document.querySelectorAll('#combat-window .cbt-lines line').length > 0);
await page.evaluate(b => {
  const x = JSON.parse(JSON.stringify(b));
  x.narrow = true;
  x.positions = { leader: {row: 2, col: 1}, 'companion:1': {row: 1, col: 0}, 'companion:2': {row: 0, col: 1} };
  x.enemies[3].cell = {row: 1, col: 2};
  window.gmcp('Company.Battle', x);
}, battleFix);
check(await page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-field')].every(g => getComputedStyle(g).gridTemplateColumns.split(' ').length === 2)), 'visible narrow battlefield has two rendered columns');
if (outdir) { await page.locator('#combat-window').screenshot({ path: path.join(outdir, 'battlefield-two-columns.png') }); }
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);

const cbt = () => page.evaluate(() => document.getElementById('combat-body').textContent);
check(await page.getByRole('heading', { name: 'Battle: a band of cutthroats' }).count() === 1, 'the Battle view replaces Setup');
check((await cbt()).includes('Withdrawing east (2 rounds remaining)'), 'ordered withdrawal appears in the battle view');
const them = page.getByRole('group', { name: 'a band of cutthroats' });
const usField = page.getByRole('group', { name: 'Your company' });
check(await them.locator('.cbt-fighter').count() === 4, 'the enemy grid: each enemy');
const order = await page.evaluate(() => [...document.querySelectorAll('[aria-label="a band of cutthroats"] .cbt-fighter')].map(n => n.getAttribute('data-fid')));
check(order.indexOf('m:414') < order.indexOf('m:412'), 'the enemy\'s back row is drawn first: its front faces the middle');
const tb = await them.boundingBox();
const ub = await usField.boundingBox();
check(tb.y + tb.height <= ub.y, 'the enemy above, the company below');
check(await page.getByRole('button', { name: 'the cutthroat captain, wounded, within your reach, striking you' }).count() === 1, 'an enemy: its label, word, reach, and target');
check(await page.getByRole('button', { name: 'the slinger, near death, striking Brannoc' }).count() === 1, 'an enemy out of reach, striking someone else');
check(await page.getByRole('button', { name: 'Wren (you), health 30 of 40, striking the cutthroat captain' }).count() === 1, 'the player: health and target');
check(await page.getByRole('button', { name: /^Ysolde, fallen/ }).count() === 1, 'a fallen companion shows fallen');
check(await page.getByRole('button', { name: /^Tamsin/ }).count() === 0, 'a companion not with you is not in the battle');
const lineInfo = await page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-lines line')].map(l => l.getAttribute('data-from') + '>' + l.getAttribute('data-to')));
check(lineInfo.length === 6 && lineInfo.includes('leader>m:412') && lineInfo.includes('m:414>u:9') && lineInfo.includes('m:412>leader'), 'a line per target, both sides (' + lineInfo.join(' ') + ')');
const toChip = await page.evaluate(() => {
  const l = document.querySelector('#combat-window line[data-to="u:9"]');
  const chip = document.querySelector('#combat-window [data-fid="u:9"]').getBoundingClientRect();
  const arena = document.querySelector('#combat-window .cbt-arena').getBoundingClientRect();
  const x = arena.left + Number(l.getAttribute('x2'));
  const y = arena.top + Number(l.getAttribute('y2'));
  return x >= chip.left && x <= chip.right && y >= chip.top && y <= chip.bottom;
});
check(toChip, 'a line to someone outside the company ends at their chip');
const body = await cbt();
check(body.includes('Wren (you) → the cutthroat captain') && body.includes('the slinger → Brannoc') && body.includes('Oswin → the cutthroat captain'), 'the text list matches the lines');
check(body.includes('Fallen: the bruiser') && body.includes('Waiting their turn: a pack of grey wolves'), 'the fallen and waiting lines');
check(await page.evaluate(() => window.__xss3 === undefined) && body.includes('<img'), 'markup in a label renders as text');
check((await page.evaluate(() => document.getElementById('combat-live').textContent)) === 'Battle: a band of cutthroats.', 'the live region announces the battle');
const lit = () => page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-fighter.is-hl')].map(n => n.getAttribute('data-fid')).sort().join(' '));
await page.getByRole('button', { name: /^the cutthroat captain/ }).hover();
check((await lit()) === 'companion:1 leader m:412', 'hover: the fighter, its target, and those striking it (' + (await lit()) + ')');
check(await page.evaluate(() => document.querySelectorAll('#combat-window line.is-hl').length === 3), 'hover lights its lines');
await page.mouse.move(0, 0);
await page.getByRole('button', { name: /^the slinger/ }).focus();
check((await lit()) === 'm:414 u:9', 'keyboard focus lights a fighter too');
got = await sentNow(async () => { await page.getByRole('button', { name: /^Oswin, health/ }).click(); await page.getByText('Role: caster').click(); });
check(JSON.stringify(got) === '["strategy #1 caster"]', 'a member\'s click opens the Setup menu');
// Phase 30c2: a guardian's ward and guards left in the battle view.
await page.evaluate(b => { const x = JSON.parse(JSON.stringify(b)); x.guards = [{ key: 'companion:1', left: 1, ward: 'leader' }]; window.gmcp('Company.Battle', x); }, battleFix);
check(await page.getByRole('button', { name: /^Oswin, health .*, guards you, 1 guard left$/ }).count() === 1, 'a guardian: whom it guards and its guards left');
check((await cbt()).includes('guards you, 1 guard left'), 'shown under its name');
await page.evaluate(b => { const x = JSON.parse(JSON.stringify(b)); x.guards = [{ key: 'companion:1', left: 0, ward: '' }]; window.gmcp('Company.Battle', x); }, battleFix);
check(await page.getByRole('button', { name: /guards the most hurt, no guards left$/ }).count() === 1, 'none left, guarding the most hurt');
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);
got = await sentNow(async () => { await page.getByRole('button', { name: 'Retreat' }).click(); });
check(JSON.stringify(got) === '["retreat"]', 'Retreat sends retreat');
check(await page.getByRole('button', { name: 'Flee' }).count() === 0, 'one way out: no separate Flee button');
await page.evaluate(b => { const x = JSON.parse(JSON.stringify(b)); x.retreat = { exit: 'east', rounds: 1 }; window.gmcp('Company.Battle', x); }, battleFix);
check((await cbt()).includes('Withdrawing east (1 round remaining)'), 'the withdrawal countdown, singular');
await page.evaluate(b => { const x = JSON.parse(JSON.stringify(b)); x.retreat = { exit: 'east', rounds: 2 }; window.gmcp('Company.Battle', x); }, battleFix);
check((await cbt()).includes('Withdrawing east (2 rounds remaining)'), 'the withdrawal countdown, plural');
// Phase 33i1: the company's outlook, in words.
check(!(await cbt()).includes('Outlook:'), 'no outlook line without one');
await page.evaluate(b => { const x = JSON.parse(JSON.stringify(b)); x.outlook = { risk: 'hard', close: true, text: 'A hard fight for your company; it could go either way.' }; window.gmcp('Company.Battle', x); }, battleFix);
check((await cbt()).includes('Outlook: A hard fight for your company; it could go either way.'), 'the outlook headline is shown');
check(await page.locator('#combat-window .cbt-outlook.cbt-risk-hard').count() === 1, 'marked with its risk');
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);
check(await page.locator('details.cbt-setup summary').count() === 1, 'Setup folds under the view');

// An update: the slinger falls, the first cutthroat turns on the player.
const next = JSON.parse(JSON.stringify(battleFix));
next.enemies = next.enemies.filter(e => e.id !== 'm:414');
next.enemies[1].target = 'leader';
next.fallen.push({ id: 'm:414', label: 'the slinger' });
next.others = [];
await page.evaluate(b => window.gmcp('Company.Battle', b), next);
check((await page.evaluate(() => document.getElementById('combat-live').textContent)) === 'the slinger falls. the first cutthroat turns on you.', 'the live region: a fall and a new foe on the player, nothing else');
check((await cbt()).includes('Fallen: the bruiser, the slinger'), 'the fallen line grows');

// Review finding 1: a Company snapshot mid-battle (a companion falls)
// keeps the view, its focus, and Setup's fold; nothing is announced.
await page.locator('details.cbt-setup summary').click();
await page.getByRole('button', { name: /^the cutthroat captain/ }).focus();
await page.evaluate(() => { const live = document.getElementById('combat-live'); live.textContent = ''; });
await page.evaluate(c => { const x = JSON.parse(JSON.stringify(c)); x.members[0].status = 'dead'; window.gmcp('Company', x); }, company);
const kept = await page.evaluate(() => ({
  heading: !!document.querySelector('#combat-body h3'),
  said: document.getElementById('combat-live').textContent,
  focus: document.activeElement && document.activeElement.getAttribute('data-fid'),
  open: !!document.querySelector('details.cbt-setup[open]'),
}));
check(kept.heading && kept.said === '' && kept.focus === 'm:412' && kept.open, 'a Company snapshot mid-battle keeps the view, focus, and Setup open, and says nothing (' + JSON.stringify(kept) + ')');
check(await page.getByRole('button', { name: /^Oswin, fallen/ }).count() === 1, 'and shows the companion fallen');
await page.evaluate(b => window.gmcp('Company.Battle', b), next);
check((await page.evaluate(() => document.getElementById('combat-live').textContent)) === '', 'the battle sent again after it: nothing announced');
await page.evaluate(c => window.gmcp('Company', c), company);
await page.evaluate(b => window.gmcp('Company.Battle', b), next);

// Review finding 6: the lines follow a resized dock.
await page.evaluate(() => { document.getElementById('dock-right').style.width = '420px'; });
await page.waitForTimeout(100);
const follows = await page.evaluate(() => {
  const l = document.querySelector('#combat-window line[data-from="m:413"]');
  const n = document.querySelector('#combat-window [data-fid="m:413"]').getBoundingClientRect();
  const arena = document.querySelector('#combat-window .cbt-arena').getBoundingClientRect();
  const x = arena.left + Number(l.getAttribute('x1'));
  return Math.abs(x - (n.left + n.width / 2)) < 2;
});
check(follows, 'target lines follow a resized dock');
await page.evaluate(() => { document.getElementById('dock-right').style.width = ''; });

// Review finding 3: in the dark, nothing but that.
await page.evaluate(() => window.gmcp('Company.Battle', { group: 'the enemy', dark: true, enemies: [] }));
check((await cbt()).includes("It's too dark to make them out.") && await page.locator('#combat-window .cbt-field').count() === 0, 'in the dark: no grids, as scout');
await page.evaluate(b => window.gmcp('Company.Battle', b), next);

// Phase 30c: the focus buttons.
const focused = JSON.parse(JSON.stringify(next));
Object.assign(focused, { focus: 'none', saved_focus: 'none', focus_ready: true });
await page.evaluate(b => window.gmcp('Company.Battle', b), focused);
const bar = page.getByRole('group', { name: 'Company focus' });
check(await bar.getByRole('button').count() === 8, 'eight focus buttons');
check((await page.locator('.cbt-focus [aria-pressed="true"]').allTextContents()).join() === 'none', 'the current focus is pressed');
got = await sentNow(async () => { await page.locator('.cbt-focus [data-focus="leader"]').click(); });
check(JSON.stringify(got) === '["company tactics focus leader"]', 'a focus button sends the order');
await page.evaluate(() => { document.getElementById('combat-live').textContent = ''; });
Object.assign(focused, { focus: 'leader', focus_ready: false });
await page.evaluate(b => window.gmcp('Company.Battle', b), focused);
check(await page.getByRole('group', { name: 'Company focus, turning next round' }).count() === 1, 'while the order waits, the bar says so');
check(await page.evaluate(() => [...document.querySelectorAll('.cbt-focus button')].every(b => b.disabled)), 'every focus button is disabled while an order waits');
check((await page.locator('.cbt-focus [aria-pressed="true"]').allTextContents()).join() === 'leader', 'the new focus is pressed');
check((await page.evaluate(() => document.getElementById('combat-live').textContent)) === 'Focus: leader.', 'the live region: the focus changed');
focused.focus_ready = true;
await page.evaluate(b => window.gmcp('Company.Battle', b), focused);
got = await sentNow(async () => { await page.getByRole('button', { name: 'saved (none)' }).click(); });
check(JSON.stringify(got) === '["company tactics focus default"]', 'saved returns to the saved focus');
check(await page.getByRole('button', { name: /^Company tactics/ }).count() === 0 && (await cbt()).includes('Company tactics: focus none, heal below 50% (saved'), 'in a battle, Setup shows the saved tactics without a menu');

// Narrow: the dock at 280px in a 360px window.
await page.setViewportSize({ width: 360, height: 800 });
await page.evaluate(() => { document.getElementById('dock-right').style.width = '280px'; });
await page.waitForTimeout(50);
const bover = await page.evaluate(() => { const p = document.getElementById('combat-window'); return p.scrollWidth - p.clientWidth; });
check(bover <= 0, 'Battle: no horizontal overflow at 280px (' + bover + ')');
check(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), 'Battle: no horizontal page scroll at 360px');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'battle-narrow.png') }); }
await page.setViewportSize({ width: 1280, height: 900 });
await page.evaluate(() => { document.getElementById('dock-right').style.width = ''; });
if (outdir) { await page.locator('#combat-window').screenshot({ path: path.join(outdir, 'battle.png') }); }

// The end: back to Setup.
await page.evaluate(() => window.gmcp('Company.Battle', {}));
check(await page.getByRole('table', { name: /Formation/ }).count() === 1 && await page.getByRole('heading', { name: /^Battle/ }).count() === 0, '{} returns to Setup');
check((await page.evaluate(() => document.getElementById('combat-live').textContent)) === 'The battle is over.', 'the live region: the battle is over');

// Alone: the player is "You", with their lines.
await page.evaluate(() => window.gmcp('Company', {}));
await page.evaluate(b => window.gmcp('Company.Battle', b), battleFix);
await page.waitForFunction(() => document.querySelectorAll('#combat-window .cbt-lines line').length > 0);
check(await page.getByRole('button', { name: 'You, striking the cutthroat captain' }).count() === 1, 'alone: the player is You');
check(await page.evaluate(() => [...document.querySelectorAll('#combat-window .cbt-lines line')].some(l => l.getAttribute('data-to') === 'leader')), 'alone: the enemy\'s line reaches You');
await page.evaluate(() => window.gmcp('Company.Battle', {}));

// Phase 30e: surrender is separate from the grid and exposes no target button.
await page.evaluate(b => { const x=JSON.parse(JSON.stringify(b)); x.surrendered=[{id:'m:999',label:'the yielded goblin'}]; window.gmcp('Company.Battle',x); }, battleFix);
check((await cbt()).includes('Surrendered: the yielded goblin'), 'surrendered foes are labelled off the grid');
check(await page.getByRole('button', { name: /yielded goblin/ }).count() === 0, 'surrendered foe has no target button');

// Phase 34 review: the server builds the editor only while it is shown, so
// the client says when it opens and closes, and again after a snapshot.
await page.evaluate(() => { window.gearRequests = []; Client.GMCPRequest = (...args) => { if (args[0] === 'Company.Equipment') { window.gearRequests.push(args[1]); } }; });
const gearSaid = () => page.evaluate(() => window.gearRequests.slice(-1)[0]);
await page.getByRole('tab', { name: 'Character', exact: true }).click();
await page.getByRole('tab', { name: 'Skills', exact: true }).click();
await page.waitForTimeout(1200);
check(!String(await gearSaid() || '').startsWith('open'), 'a hidden Gear editor is not announced as open');
await page.getByRole('tab', { name: 'Gear', exact: true }).click();
await page.waitForTimeout(300);
check(await gearSaid() === 'open weapon me', 'opening Gear asks the server for the editor and its selected slot');
await page.getByRole('tab', { name: 'Skills', exact: true }).click();
await page.waitForTimeout(300);
check(await gearSaid() === 'closed', 'leaving Gear tells the server to stop building it');
await page.getByRole('tab', { name: 'Gear', exact: true }).click();
await page.waitForTimeout(300);
const saidBefore = await page.evaluate(() => window.gearRequests.length);
await page.evaluate(c => window.gmcp('Company', c), company);
check(await page.evaluate(n => window.gearRequests.length > n && window.gearRequests.slice(-1)[0] === 'open weapon me', saidBefore), 'a Company snapshot (login, reconnect) re-announces the open editor');
const pendingView = { available: true, current: {}, slots: [
  { slot: 'weapon', label: 'Weapon', choices: [] },
  { slot: 'body', label: 'Body', choices: [], pending: true, equipped: { ref: '!3:coat', label: 'quilted coat' } }] };
await page.evaluate(view => window.gmcp('Company.Equipment', view), pendingView);
await page.locator('#gw-worn').getByRole('button', { name: /^Body:/ }).click();
await page.waitForTimeout(100);
check(await gearSaid() === 'open body me' && (await page.locator('#gw-worn').textContent()).includes('Loading choices'), 'selecting a slot asks for its choices and says they are loading');
await page.locator('#gw-worn').getByRole('button', { name: /^Weapon:/ }).click();
await page.evaluate(() => { Client.GMCPRequest = function() {}; });

// Phase 34c: authoritative, exact-instance slot editor.
await page.getByRole('tab', { name: 'Character', exact: true }).click();
await page.getByRole('tab', { name: 'Gear', exact: true }).click();
const editorStats = { damage: '1d6+1', offhand_damage: '', hands: 1, reach: false, shield: true,
  edge_bonus: 0, edge_strikes: 0, offhand_edge_bonus: 0, offhand_edge_strikes: 0, defense: 13, worn_g: 1234, burden: 'unburdened', dodge_pct: 100,
  health_max: 44, mana_max: 20, pack_capacity_g: 10000, capacity_g: 50000, cargo_g: 2000,
  stats: { strength: 15, speed: 16 } };
const afterStats = { ...editorStats, damage: '2d9+3', defense: 19, worn_g: 11100,
  edge_bonus: 2, edge_strikes: 9, burden: 'burdened', dodge_pct: 73, cargo_g: 750, stats: { strength: 21, speed: 11 } };
const gearView = { available: true, current: editorStats, slots: [
  { slot: 'weapon', label: 'Weapon', equipped: { ref: '!1:current', label: 'iron sword' },
    remove: { ref: '!1:current', label: 'iron sword', allowed: true, after: editorStats,
      command: 'company remove me weapon !1:current' },
    choices: [
      { ref: '!8:first', label: 'same blade', allowed: true, after: afterStats, returned: ['iron sword'], command: 'company equip me !8:first weapon' },
      { ref: '!8:second', label: 'same blade', allowed: true, after: afterStats, returned: ['iron sword'], command: 'company equip me !8:second weapon' },
      { ref: '!8:bound', label: xss, allowed: false, reason: 'Your weapon is bound.', command: 'company equip me !8:bound weapon' }
    ] },
  { slot: 'body', label: 'Body', choices: [] },
  { slot: 'pack', label: 'Pack', equipped: { ref: '!38:oldpack', label: 'cloth knapsack' }, choices: [
    { ref: '!33:frame', label: 'frame pack', allowed: true, after: { ...editorStats, pack_capacity_g: 15000, capacity_g: 55000, cargo_g: 1400 }, command: 'company equip me !33:frame pack' }
  ] }
] };
await page.evaluate(view => window.gmcp('Company.Equipment', view), gearView);
const gearText = () => page.locator('#gw-worn').textContent();
check((await gearText()).includes('Body: empty') && (await gearText()).includes('Pack: cloth knapsack'), 'Gear shows equipped and empty slots including Pack');
check(await page.locator('#gear-window .gw-tab-btn[data-panel=gw-backpack]').isHidden(), 'shared Gear uses slot-specific choices, leaving the cargo list in Company');
await page.locator('#gw-worn').getByRole('button', { name: 'same blade', exact: true }).nth(1).click();
check((await gearText()).includes('Current → After') && (await gearText()).includes('Returns to cargo: iron sword'), 'Gear previews displaced items before assignment');
check(await page.locator('.gw-editor-stats thead').textContent() === 'StatCurrentAfter', 'comparison columns have accessible Current and After headings');
check(await page.locator('.gw-editor-stats tr', { hasText: 'Dodge retained (%)' }).textContent() === 'Dodge retained (%)10073', 'Gear displays server dodge values without recalculating');
check(await page.locator('.gw-editor-stats tr', { hasText: 'Weapon damage' }).textContent() === 'Weapon damage1d6+12d9+3', 'Gear displays authoritative damage dice before and after');
check(await page.locator('.gw-editor-stats tr', { hasText: 'Weapon edge bonus' }).textContent() === 'Weapon edge bonus02' && await page.locator('.gw-editor-stats tr', { hasText: 'Weapon edge strikes' }).textContent() === 'Weapon edge strikes09', 'Gear includes the active edge damage and remaining strikes');
await page.keyboard.press('Tab');
await page.locator('#gw-worn').getByRole('button', { name: 'Equip item', exact: true }).focus();
await page.evaluate(view => window.gmcp('Company.Equipment', view), gearView);
check(await page.evaluate(() => document.activeElement.dataset.gearFocus === 'apply'), 'Gear preserves keyboard focus across live updates');
let editorSent = await sentNow(async () => { await page.keyboard.press('Enter'); });
check(JSON.stringify(editorSent) === '["company equip me !8:second weapon"]', 'Gear executes the selected duplicate and explicit slot');
const staleView = JSON.parse(JSON.stringify(gearView));
staleView.slots[0].choices = staleView.slots[0].choices.filter(c => c.ref !== '!8:second');
await page.evaluate(view => window.gmcp('Company.Equipment', view), staleView);
check((await gearText()).includes('no longer available') && await page.locator('#gw-worn').getByRole('button', { name: 'Equip item', exact: true }).count() === 0, 'stale cargo selection clears the action and explains why');
check(await page.evaluate(() => document.activeElement.dataset.gearFocus === 'slot:weapon'), 'stale selection returns keyboard focus to the selected slot');
await page.locator('#gw-worn').getByRole('button', { name: 'Preview removal', exact: true }).click();
editorSent = await sentNow(async () => { await page.locator('#gw-worn').getByRole('button', { name: 'Remove equipment', exact: true }).click(); });
check(JSON.stringify(editorSent) === '["company remove me weapon !1:current"]', 'Gear removal names the exact worn instance');
const replacedView = JSON.parse(JSON.stringify(gearView));
replacedView.slots[0].remove.ref = '!1:replacement';
replacedView.slots[0].remove.command = 'company remove me weapon !1:replacement';
await page.evaluate(view => window.gmcp('Company.Equipment', view), replacedView);
check((await gearText()).includes('no longer available') && await page.locator('#gw-worn').getByRole('button', { name: 'Remove equipment', exact: true }).count() === 0, 'a changed worn instance also invalidates removal selection');
await page.locator('#gw-worn').getByRole('button', { name: xss, exact: true }).click();
check(await page.locator('#gw-worn').getByRole('button', { name: 'Equip item', exact: true }).isDisabled() && (await gearText()).includes('weapon is bound'), 'unavailable equipment shows its reason and disables assignment');
check(await page.locator('#gw-worn img').count() === 0, 'Gear labels render safely as text');
await page.locator('#gw-worn').getByRole('button', { name: 'Body: empty', exact: true }).click();
check((await gearText()).includes('No compatible items in shared cargo'), 'an empty compatible list has a useful explanation');
await page.locator('#gw-worn').getByRole('button', { name: 'Pack: cloth knapsack', exact: true }).click();
await page.locator('#gw-worn').getByRole('button', { name: 'frame pack', exact: true }).click();
check(await page.locator('.gw-editor-stats tr', { hasText: 'Company capacity (g)' }).textContent() === 'Company capacity (g)5000055000', 'Pack preview shows the final server capacity');
// Phase 48: the editor shows any member's gear.
const memberView = JSON.parse(JSON.stringify(gearView));
memberView.members = [{ ref: 'me', name: 'Wren', ready: true }, { ref: '#1', name: 'Brother Oswin', ready: true }];
memberView.member = 'me';
await page.evaluate(view => window.gmcp('Company.Equipment', view), memberView);
check(await page.locator('#gw-worn').getByRole('button', { name: 'Brother Oswin', exact: true }).count() === 1, 'Gear lists the company\'s members to pick from');
await page.evaluate(() => { window.gearRequests = []; Client.GMCPRequest = (...args) => { if (args[0] === 'Company.Equipment') { window.gearRequests.push(args[1]); } }; });
await page.locator('#gw-worn').getByRole('button', { name: 'Brother Oswin', exact: true }).click();
await page.waitForTimeout(100);
check(await page.evaluate(() => window.gearRequests.slice(-1)[0]) === 'open pack #1' && (await gearText()).includes('Loading gear'), 'choosing a member asks the server for that member\'s gear');
const oswinView = JSON.parse(JSON.stringify(memberView));
oswinView.member = '#1';
oswinView.slots[0].choices[0].command = 'company equip #1 !8:first weapon';
await page.evaluate(view => window.gmcp('Company.Equipment', view), oswinView);
await page.locator('#gw-worn').getByRole('button', { name: /^Weapon:/ }).click();
await page.locator('#gw-worn').getByRole('button', { name: 'same blade', exact: true }).first().click();
if (outdir) { await page.locator('#gear-window').screenshot({ path: path.join(outdir, 'companion-gear-editor.png') }); }
editorSent = await sentNow(async () => { await page.locator('#gw-worn').getByRole('button', { name: 'same blade', exact: true }).first().click(); await page.locator('#gw-worn').getByRole('button', { name: 'Equip item', exact: true }).click(); });
check(JSON.stringify(editorSent) === '["company equip #1 !8:first weapon"]', 'Gear equips the chosen companion');
await page.evaluate(() => window.GearEditor.show('me', 'weapon'));
await page.setViewportSize({ width: 360, height: 760 });
check(await page.evaluate(() => { const p = document.getElementById('gw-worn'); return p.scrollWidth <= p.clientWidth + 1; }), 'Gear editor fits a narrow viewport');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'gear-editor-narrow.png') }); }
await page.setViewportSize({ width: 1280, height: 900 });

// Phase 34d: current effects/wounds and capabilities, safe text and focus.
await page.evaluate(c => window.gmcp('Company', c), company);
await page.getByRole('tab', { name: 'Company' }).first().click();
await page.getByRole('tab', { name: 'Status', exact: true }).click();
const conditions = {
  leader: { state: 'live', effects: [{ name: xss, description: 'Loses its next action', duration: '2 combat rounds remaining' }],
    wounds: [{ name: 'a cut to the arm', description: 'Holds back 3 health', duration: 'Until treated or rested away' }],
    bonuses: [{ name: 'Vigor', description: 'Strength +2', duration: 'Persistent' }] },
  'companion:1': { state: 'live', effects: [], wounds: [{ name: 'a deep bruise', description: 'Holds back 2 health', duration: 'Until fight ends' }], bonuses: [] },
  'companion:3': { state: 'separated', effects: [], wounds: [], bonuses: [] },
  'companion:4': { state: 'dead', effects: [], wounds: [], bonuses: [] }
};
await page.evaluate(c => window.gmcp('Company.Conditions', c), conditions);
check((await status()).includes('2 combat rounds remaining') && (await status()).includes('Until fight ends') && (await status()).includes('Persistent bonuses'), 'Status separates effect duration, wounds and persistent bonuses');
check((await status()).includes('Separated: live effects unknown') && (await status()).includes('Fallen: no active member effects'), 'Status states away and dead conditions honestly');
check(await page.locator('#party-panel img').count() === 0, 'effect labels render as safe text');
await page.locator('#party-panel [data-key=leader]').focus();
conditions.leader.effects = [];
await page.evaluate(c => window.gmcp('Company.Conditions', c), conditions);
check(!(await status()).includes('2 combat rounds remaining') && await page.evaluate(() => document.activeElement.dataset.key === 'leader'), 'removed effect disappears while the member card keeps keyboard focus');
await page.setViewportSize({ width: 360, height: 760 });
check(await page.evaluate(() => { const p = document.getElementById('party-panel'); return p.scrollWidth <= p.clientWidth + 1; }), 'conditions fit a narrow Company Status');
await page.getByRole('tab', { name: 'Character', exact: true }).first().click();
await page.getByRole('tab', { name: 'Skills', exact: true }).click();
await page.evaluate(() => window.gmcp('Char.Skills', [{ name: 'protection', level: 3, max_level: 3, maximum: true }]));
const capabilities = { automatic: [{ name: xss, skill: 'brawling', description: 'Knocks down its foe', when: 'within reach', cooldown: 4, enabled: false, reason: 'Automatic abilities disabled by strategy' }],
  utility: [{ name: 'Camp Watch', group: 'camp', skill: 'brawling', rank: 2, description: 'spots raiders', enabled: false, reason: 'Autoskill off' },
    { name: 'Camp Cooking', group: 'camp', mode: 'manual', skill: 'cooking', rank: 2, description: 'seared game meat requires cooking rank 1', enabled: true }] };
await page.evaluate(c => window.gmcp('Char.Capabilities', c), capabilities);
const skillsText = () => page.locator('#cw-skills-tab').textContent();
check((await skillsText()).includes('Trained ranks') && (await skillsText()).includes('Automatic combat abilities') && (await skillsText()).includes('Field capabilities') && (await skillsText()).includes('Camp capabilities'), 'Skills groups ranks, combat, field and camp capabilities');
check((await skillsText()).includes('Camp CookingManual') && (await skillsText()).includes('eared game meat requires cooking rank 1'), 'Skills shows manual camp cooking independently of class specialists');
check((await skillsText()).includes('disabled by strategy') && (await skillsText()).includes('Autoskill off'), 'Skills explains disabled strategies and autoskills');
check(await page.locator('#cw-capabilities img').count() === 0 && await page.locator('#cw-skills .csk-pip').count() === 3, 'safe capability text and actual trained-rank maximum');
// Phase 57 review: a long status badge wraps instead of squeezing the name to one letter a line.
await page.evaluate(() => window.gmcp('Char.Capabilities', { automatic: [], utility: [
  { name: 'Camp Cooking', group: 'camp', mode: 'manual', skill: 'cooking', rank: 0, description: 'Manual camp cook at your own lit campfire', enabled: false, reason: 'Missing recipe ingredients or trained ranks' }] }));
const squeezed = await page.evaluate(() => {
  const name = [...document.querySelectorAll('#cw-capabilities .csk-name')].find(e => e.textContent === 'Camp Cooking');
  const lh = parseFloat(getComputedStyle(name).lineHeight) || parseFloat(getComputedStyle(name).fontSize) * 1.4;
  return { found: !!name, h: name && name.getBoundingClientRect().height, lh, badge: name && name.parentElement.querySelector('.csk-side').getBoundingClientRect().right <= name.closest('.csk-card').getBoundingClientRect().right + 0.5 };
});
check(squeezed.found && squeezed.h <= squeezed.lh * 1.5 && squeezed.badge, 'a long capability status wraps below the name and stays inside the card (' + JSON.stringify(squeezed) + ')');
await page.getByRole('button', { name: 'protection, rank 3 of 3, help' }).focus();
await page.evaluate(c => window.gmcp('Char.Capabilities', c), capabilities);
check(await page.evaluate(() => document.activeElement.dataset.skill === 'protection'), 'Skills keeps focus through capability refresh');
await page.evaluate(() => { window.helpRequests = []; Client.GMCPRequest = (...args) => window.helpRequests.push(args); });
await page.keyboard.press('Enter');
check(JSON.stringify(await page.evaluate(() => window.helpRequests)) === '[[' + '"Help","protection"' + ']]', 'trained-rank help works with the keyboard');
check(await page.evaluate(() => { const p = document.getElementById('cw-skills-tab'); return p.scrollWidth <= p.clientWidth + 1; }), 'Skills fits a narrow viewport');
// Phase 35c: the companions' optional skills and training points, read only.
await page.evaluate(c => { const g = JSON.parse(JSON.stringify(c)); g.members[0].skills = { cooking: 2 }; g.members[0].training_points = 3; g.members[1].training_points = 1; g.members[1].name = '<img src=x onerror="window.__xss35c=1">'; window.gmcp('Company', g); }, company);
check((await skillsText()).includes('Company training') && (await skillsText()).includes('Cooking rank 2') && (await skillsText()).includes('3 training points') && (await skillsText()).includes('No optional skills') && (await skillsText()).includes('1 training point'), 'Skills lists the companions\' optional skills and training points (35c)');
check(await page.locator('#cw-company-skills img').count() === 0 && await page.locator('#cw-company-skills button').count() === 0 && !(await page.evaluate(() => window.__xss35c)), 'company training text is plain and has no train button');
check(await page.evaluate(() => { const p = document.getElementById('cw-skills-tab'); return p.scrollWidth <= p.clientWidth + 1; }), 'company training fits a narrow viewport');
await page.getByRole('tab', { name: 'Effects', exact: true }).click();
check((await page.locator('#cw-effects').textContent()).includes('Wounds') && (await page.locator('#cw-effects').textContent()).includes('Holds back 3 health'), 'Character Effects includes wound mechanics apart from persistent bonuses');
await page.evaluate(markup => window.gmcp('Char.Affects', {test: {name: markup, description: markup, duration_max: 30, duration_left: 20}}), xss);
check(await page.locator('#cw-effects img').count() === 0, 'Character Effects never interprets server labels as HTML');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'phase34d-effects-narrow.png') }); }
await page.setViewportSize({ width: 1280, height: 900 });

// Phase 34 review: the server sends a timed effect once; the client counts it
// down with a meter, labels harm in words, and never doubles a full stop.
conditions.leader.effects = [{ name: 'Slowed', description: 'Moves slowly.', duration: '30 seconds remaining', seconds_left: 30, seconds_total: 60, harmful: true, mods: { speed: -2 } },
  { name: 'Blessed', description: 'Guided by light.', duration: 'Until removed', helpful: true, mods: { perception: 1 } },
  { name: 'Sleepy', description: 'Drowsing.', duration: 'Until removed' }];
conditions['companion:1'].state = 'away-live';
await page.evaluate(c => window.gmcp('Company.Conditions', c), conditions);
const effectsText = () => page.locator('#cw-effects').textContent();
check((await effectsText()).includes('Harmful') && (await effectsText()).includes('Helpful'), 'effects say Harmful or Helpful in words, not colour alone');
check(await page.locator('#cw-effects .cmp-condition', { hasText: 'Sleepy' }).locator('.cmp-condition-tag').count() === 0, 'an effect neither known to harm nor help has no tag');
check(!/\.\./.test(await effectsText()), 'effect text never doubles a full stop');
const meter = page.locator('#cw-effects [role=meter]');
check(await meter.count() === 1 && await meter.getAttribute('aria-valuemax') === '60', 'a timed effect has a duration meter');
const leftBefore = await meter.getAttribute('aria-valuenow');
await page.waitForTimeout(3200);
const leftAfter = await meter.getAttribute('aria-valuenow');
check(Number(leftBefore) - Number(leftAfter) >= 2 && (await effectsText()).includes(leftAfter + 's remaining'), 'the countdown runs without a new message');
check(!/\.\./.test(await page.locator('#cw-skills-tab').textContent()), 'capability text never doubles a full stop');
await page.getByRole('tab', { name: 'Company' }).first().click();
await page.getByRole('tab', { name: 'Status', exact: true }).click();
check((await status()).includes('Away from you: current effects and wounds') && (await status()).includes('Harmful'), 'Status shows an away member and the shared effect cards');
for (const theme of readdirSync(path.join(here, '../../_datafiles/html/public/static/css')).filter(n => /^theme-.*\.css$/.test(n))) {
    await page.evaluate(theme => new Promise(resolve => {
        const link = document.getElementById('theme-css');
        link.onload = resolve;
        link.href = '../../_datafiles/html/public/static/css/' + theme;
    }), theme);
    const ratios = await page.evaluate(() => {
        function luminance(color) {
            const v = color.match(/[\d.]+/g).slice(0, 3).map(n => Number(n) / 255).map(n => n <= 0.04045 ? n / 12.92 : Math.pow((n + 0.055) / 1.055, 2.4));
            return v[0] * 0.2126 + v[1] * 0.7152 + v[2] * 0.0722;
        }
        function ratio(fg, bg) {
            const a = luminance(fg), b = luminance(bg);
            return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
        }
        return [...document.querySelectorAll('#party-panel .cmp-condition')].flatMap(card => {
            const bg = getComputedStyle(card).backgroundColor;
            return [...card.querySelectorAll('span, .cmp-condition-duration, .cmp-condition-text')].map(e => ratio(getComputedStyle(e).color, bg));
        });
    });
    check(ratios.length > 0 && ratios.every(n => n >= 4.5), theme + ': effect card text contrast >= 4.5 (' + Math.min(...ratios).toFixed(2) + ')');
}

// Phase 57: the Character panel's Skills, Gear and Overview in the Company
// type scale, no stock Jobs, and hover highlights that read well and never stick.
await page.setViewportSize({ width: 1280, height: 900 });
await page.evaluate(() => { window.helpRequests = []; Client.GMCPRequest = (...args) => { window.helpRequests.push(args); }; });
await page.getByRole('tab', { name: 'Character', exact: true }).first().click();
await page.getByRole('tab', { name: 'Skills', exact: true }).click();
await page.evaluate(() => window.gmcp('Char.Skills', [
  { name: 'dual-wield', title: 'Dual Wield', description: 'Wield weapons in both hands that normally wouldn\'t allow it.', level: 4, max_level: 4, maximum: true },
  { name: 'track', title: 'Track', description: 'Read the trail for enemy groups beyond the exits.', level: 1, max_level: 4 }]));
check((await skillsText()).includes('Dual Wield') && (await skillsText()).includes('Wield weapons in both hands') && (await skillsText()).includes('4/4') && (await skillsText()).includes('MAX'),
  'Skills names each skill, says what it does and its rank (57)');
check(await page.getByRole('button', { name: 'Dual Wield, rank 4 of 4, help' }).count() === 1, 'a skill\'s accessible name uses its title');
check(await page.evaluate(() => {
  const size = e => parseFloat(getComputedStyle(e).fontSize);
  const cap = document.querySelector('#cw-capabilities .csk-card');
  const base = parseFloat(getComputedStyle(document.querySelector('.cmp-tab-btn') || document.body).fontSize) / 0.7;
  return !cap || (size(cap) <= base * 0.85 && size(document.querySelector('#cw-skills .csk-desc')) <= base * 0.8);
}), 'Skills text is in the compact Company scale, not the dock\'s full size');
await page.getByRole('button', { name: 'Track, rank 1 of 4, help' }).click();
check(JSON.stringify(await page.evaluate(() => window.helpRequests)) === '[["Help","track"]]', 'clicking a skill asks for its help page');
// The Help screen: it takes focus from the control that opened it, nothing behind it keeps a hover, and Escape returns focus.
await page.evaluate(() => { document.dispatchEvent(new Event('DOMContentLoaded')); });
await page.getByRole('button', { name: 'Track, rank 1 of 4, help' }).focus();
await page.keyboard.press('Tab'); await page.keyboard.press('Shift+Tab');  // reach it by keyboard
await page.evaluate(() => GameModal.open({ title: 'track', body: 'Track help', format: 'html' }));
check(await page.evaluate(() => document.body.classList.contains('game-modal-open') && !document.activeElement.classList.contains('csk-card')), 'opening a help screen takes focus from the control behind it');
check(await page.evaluate(() => getComputedStyle(document.getElementById('main-container')).pointerEvents === 'none'), 'nothing behind an open help screen takes the pointer');
await page.keyboard.press('Escape');
check(await page.evaluate(() => !document.body.classList.contains('game-modal-open') && document.activeElement && document.activeElement.classList.contains('csk-card')), 'closing a screen the keyboard opened returns focus to the control and restores the pointer');
// Phase 57 review: a screen opened with the mouse and closed with Esc must
// not light a focus ring on the control behind it; typing resumes instead.
await page.evaluate(() => document.activeElement.blur());
await page.getByRole('button', { name: 'Track, rank 1 of 4, help' }).click();
await page.evaluate(() => GameModal.open({ title: 'track', body: 'Track help', format: 'html' }));
await page.keyboard.press('Escape');
check(await page.evaluate(() => !document.body.classList.contains('game-modal-open') && document.querySelectorAll(':focus-visible:not(input)').length === 0 && document.activeElement && document.activeElement.id === 'command-input'), 'closing a screen the mouse opened leaves no focus ring behind and returns to the command line');
// Hover highlights only for a real pointer, so a tap on a touch screen leaves none stuck.
const stuck = await page.evaluate(() => {
  const bad = [];
  for (const sheet of document.styleSheets) {
    let rules; try { rules = sheet.cssRules; } catch (e) { continue; }
    for (const r of rules) {
      if (r.type === 1 && /:hover/.test(r.selectorText) && !/scrollbar/.test(r.selectorText) && (r.style.background || r.style.backgroundColor || r.style.color)) { bad.push(r.selectorText); }
    }
  }
  return bad;
});
check(stuck.length === 0, 'every hover highlight is limited to hover-capable pointers: ' + stuck.join(' | '));
// A menu has one highlighted entry: the pointer moves focus, no inline highlight is left behind.
await page.getByRole('tab', { name: 'Company', exact: true }).first().click();
await page.getByRole('tab', { name: 'Inventory', exact: true }).click();
await page.evaluate(i => window.gmcp('Company.Inventory', i), inventory);
await page.locator('button.cmp-item').first().click();
const entries = page.locator('.ui-menu-item');
const litCount = () => page.evaluate(() => [...document.querySelectorAll(".ui-menu-item")].filter(e => getComputedStyle(e).backgroundColor !== 'rgba(0, 0, 0, 0)').length);
check(await litCount() === 1, 'a fresh menu highlights only its first entry');
await entries.nth(1).hover();
check(await litCount() === 1 && await page.evaluate(() => document.activeElement === document.querySelectorAll('.ui-menu-item')[1]), 'hovering moves the one highlight, never adds a second');
await page.mouse.move(5, 5);
await page.mouse.down();
await page.mouse.up();
check(await page.locator('.ui-menu').count() === 0, 'clicking away closes the menu and its highlight');
// A highlighted row stays readable: the row and its quieter parts against the highlight, in every theme.
for (const theme of readdirSync(path.join(here, '../../_datafiles/html/public/static/css')).filter(n => /^theme-.*\.css$/.test(n))) {
  await page.evaluate(theme => new Promise(resolve => {
    const link = document.getElementById('theme-css');
    link.onload = resolve;
    link.href = '../../_datafiles/html/public/static/css/' + theme;
  }), theme);
  await page.locator('button.cmp-item').first().hover();
  const worst = await page.evaluate(() => {
    function lum(color) {
      const v = color.match(/[\d.]+/g).slice(0, 3).map(n => Number(n) / 255).map(n => n <= 0.04045 ? n / 12.92 : Math.pow((n + 0.055) / 1.055, 2.4));
      return v[0] * 0.2126 + v[1] * 0.7152 + v[2] * 0.0722;
    }
    const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);
    const row = document.querySelector('button.cmp-item:hover');
    if (!row) { return 0; }
    const bg = getComputedStyle(row).backgroundColor;
    return Math.min(...[row, ...row.querySelectorAll('span')].map(e => ratio(getComputedStyle(e).color, bg)));
  });
  check(worst >= 3, theme + ': a hovered inventory row reads (' + worst.toFixed(2) + ')');
}
// 360px: the Skills, Gear and Overview fit without sideways scrolling.
await page.setViewportSize({ width: 360, height: 760 });
await page.getByRole('tab', { name: 'Character', exact: true }).first().click();
for (const name of ['Overview', 'Gear', 'Skills']) {
  await page.getByRole('tab', { name, exact: true }).click();
  check(await page.evaluate(() => [...document.querySelectorAll('#character-window .cw-tab-panel.active')].every(p => p.scrollWidth <= p.clientWidth + 1)), name + ' fits a 360px viewport (57)');
}
await page.setViewportSize({ width: 1280, height: 900 });

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all dock window checks passed');
