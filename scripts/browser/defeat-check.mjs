// Phase 53 browser check: a capture's held goods in the Company panel's
// Inventory tab, through dock-windows-harness.html: the "Held by your
// captors" notice, the Reclaim button only in the capture room, and the
// notice gone once nothing is held. Desktop and phone widths. Screenshots go
// to [prefix]-*.png when given.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/defeat-check.mjs [screenshot-prefix]
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const shot = process.argv[2];

let failures = 0;
const check = (ok, what) => { if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); } };

const member = (key, id, name, cell) => ({ key, id, name, status: 'present', level: 5, archetype: 'Warrior', cell, chemistry: null, strategy: { role: 'fighter', target: 'weakest' } });
const company = { leader: member('leader', 0, 'Wren', { row: 0, col: 1 }), members: [member('companion:1', 1, 'Brant', { row: 0, col: 0 })], alive: 2, dead: 0 };
const inventory = (seized) => ({
  load: { total_g: 1500, capacity_g: 100000, member_capacity_g: 100000, mount_capacity_g: 0, cargo_g: 0 },
  companions_known: true, shared: true, treasury: 0, autoloot: true, containers: [],
  members: [{ key: 'leader', name: 'Wren', fallen: false, unrecorded: false, grams: 1500, pack: '', pack_bonus_g: 0,
    worn: [{ ref: '!1:sword', name: 'iron sword', grams: 1500, count: 1, uses: 0, uses_max: 0, type: 'weapon', subtype: 'slashing', slot: 'weapon' }], carried: [] }],
  horses: [], cargo: [], seized,
});

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
for (const [label, viewport] of [['desk', { width: 1280, height: 900 }], ['m360', { width: 360, height: 760 }]]) {
  const page = await browser.newPage({ viewport });
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  page.on('console', m => { if (m.type() === 'error') { failures++; console.log('FAIL console error: ' + m.text()); } });
  await page.goto(process.env.DOCK_HARNESS_URL || 'file://' + path.join(here, 'dock-windows-harness.html'));
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  const gmcp = (ns, body) => page.evaluate(([n, b]) => window.gmcp(n, b), [ns, body]);
  await gmcp('Company', company);
  await gmcp('Company.Inventory', inventory({ items: 6, gold: 150, where: "Brigands' Tent", here: true }));
  await page.getByRole('tab', { name: 'Company' }).first().click();
  await page.getByRole('tab', { name: 'Inventory' }).click();
  const text = () => page.evaluate(() => document.getElementById('company-inventory').textContent);
  const held = page.locator('#company-inventory .cmp-held');
  check(await held.isVisible() && (await text()).includes("Held by your captors: 6 items and 150 gold in Brigands' Tent. Defeat the guards, then reclaim it."), label + ': the notice names what is held and where');
  const reclaim = page.locator('#company-inventory button', { hasText: 'Reclaim' });
  check(await reclaim.isVisible(), label + ': Reclaim shows in the capture room');
  const box = await reclaim.boundingBox();
  check(box && box.x >= 0 && box.x + box.width <= viewport.width, label + ': Reclaim fits the width');
  check(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), label + ': no sideways scroll');
  await page.evaluate(() => { window.sent = []; });
  await reclaim.click();
  check(JSON.stringify(await page.evaluate(() => window.sent)) === '["reclaim"]', label + ': Reclaim sends reclaim');
  if (shot) { await page.screenshot({ path: `${shot}-${label}-here.png` }); }

  await gmcp('Company.Inventory', inventory({ items: 1, gold: 0, where: "Brigands' Tent", here: false }));
  check((await text()).includes("Held by your captors: 1 item in Brigands' Tent. Go back there, defeat the guards, and reclaim it.")
    && await page.locator('#company-inventory button', { hasText: 'Reclaim' }).count() === 0, label + ': away from it, the way back and no button');
  if (shot) { await page.screenshot({ path: `${shot}-${label}-away.png` }); }

  await gmcp('Company.Inventory', inventory(undefined));
  check(await page.locator('#company-inventory .cmp-held').count() === 0, label + ': gone once nothing is held');
  await page.close();
}
await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all checks passed');
