// Phase 36d review browser check: the gear tooltip shows a relic's rarity
// colour, signature or set bonuses, and lore; the Company window's gear
// rows carry the same words. Desktop and phone sizes.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/relic-check.mjs [outdir]
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

const reaper = { id: '!50001:a', name: 'Ashen Reaper', label: 'Ashen Reaper', rarity: 'legendary', type: 'weapon', subtype: 'slashing', details: [],
  relic: ['Reaping (while worn): blows deal 20% more to a foe at or below half health; blows deal 20% more to undead and demons.',
    // Phase 67: awakenings, one woken and one asleep.
    'Awakened, Bone-Breaker: blows deal 5% more to a foe at or below half health.',
    'Sleeping, Lich-Bane (0 of 1): defeat the lich while it is worn, and it wakes with +3 Attack.'],
  relic_lore: 'The lich carried it out of a dead kingdom. It is said to reap best what is already falling.' };
const helm = { id: '!50011:b', name: "Ogre-hunter's Helm", label: "Ogre-hunter's Helm", rarity: 'set', type: 'head', subtype: 'wearable', details: [],
  relic: ["Piece of the Ogre-hunter's Kit set (3 pieces).", '  2 worn: +3% damage reduction on top of worn armor.', '  3 worn: +4 Attack; blows deal 15% more to a foe at or below half health.'],
  relic_lore: 'Those who hunt the forest ogre learn to wear what it cannot easily crush.' };
// Phase 71 review: a plain piece with a trophy enchant (no rarity, no lore).
const vest = { id: '!20001:c', name: 'leather vest', label: 'leather vest (enchanted: chitin plate)', type: 'body', subtype: 'wearable', details: [],
  relic: ['Enchanted with chitin plate (while worn): +2 Evasion.'] };

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
for (const [size, viewport] of [['desk', { width: 1280, height: 900 }], ['phone', { width: 390, height: 844 }]]) {
  const page = await browser.newPage({ viewport });
  page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
  await page.goto('file://' + path.join(here, 'dock-windows-harness.html'));
  await page.evaluate(() => { localStorage.clear(); localStorage.setItem('ashveil-battle-screen', 'manual'); });
  await page.reload();
  await page.evaluate(([reaper, helm, vest]) => {
    window.gmcp('Char.Info', { name: 'Wren', class: 'ranger', race: 'Human', level: 40 });
    window.gmcp('Char.Inventory', {
      Worn: { weapon: reaper, head: helm, body: vest },
      Backpack: { items: [], Summary: { count: 0, weight_g: 0, load_g: 0, capacity_g: 50000 } },
    });
  }, [reaper, helm, vest]);
  await page.getByRole('tab', { name: 'Gear' }).click();
  for (const [slot, item] of [['weapon', reaper], ['head', helm]]) {
    await page.hover('#gw-eqrow-' + slot);
    await page.waitForTimeout(100);
    const tt = await page.evaluate(() => {
      const t = document.getElementById('gw-item-tooltip');
      const name = t.querySelector('.gw-tt-name');
      const r = t.getBoundingClientRect();
      return { shown: t.style.display === 'block', text: t.textContent, colour: getComputedStyle(name).color, left: r.left, right: r.right, vw: window.innerWidth };
    });
    check(tt.shown && tt.text.includes(item.relic[0]) && tt.text.includes(item.relic_lore), size + ': ' + slot + ' tooltip shows the relic text and lore');
    check(tt.colour === (item.rarity === 'set' ? 'rgb(0, 175, 175)' : 'rgb(255, 135, 0)'), size + ': ' + slot + ' name in its rarity colour (' + tt.colour + ')');
    check(tt.left >= 0 && tt.right <= tt.vw, size + ': ' + slot + ' tooltip fits the screen (' + Math.round(tt.left) + '-' + Math.round(tt.right) + ' of ' + tt.vw + ')');
    if (slot === 'weapon') {
      const lines = await page.evaluate(() => {
        const t = document.getElementById('gw-item-tooltip');
        const awake = t.querySelector('.gw-tt-awake'), sleep = t.querySelector('.gw-tt-sleep');
        return { awake: awake && getComputedStyle(awake).color, sleep: sleep && getComputedStyle(sleep).color, sig: getComputedStyle(t.querySelector('.gw-tt-relic div')).color };
      });
      check(lines.awake === 'rgb(255, 215, 95)', size + ': a woken awakening is gold (' + lines.awake + ')');
      check(lines.sleep && lines.sleep !== lines.awake && lines.sleep !== lines.sig, size + ': a sleeping awakening is dim (' + lines.sleep + ')');
    }
    if (outdir) { await page.screenshot({ path: path.join(outdir, '67-tooltip-' + slot + '-' + size + '.png') }); }
  }
  // Phase 71 review: the enchanted plain piece shows its enchant line in purple.
  await page.hover('#gw-eqrow-body');
  await page.waitForTimeout(100);
  const ench = await page.evaluate(() => {
    const t = document.getElementById('gw-item-tooltip');
    const line = t.querySelector('.gw-tt-ench');
    const r = t.getBoundingClientRect();
    return { shown: t.style.display === 'block', text: line && line.textContent, colour: line && getComputedStyle(line).color, left: r.left, right: r.right, vw: window.innerWidth };
  });
  check(ench.shown && ench.text === vest.relic[0], size + ': an enchanted plain piece shows its enchant (' + ench.text + ')');
  check(ench.colour === 'rgb(215, 135, 255)', size + ': the enchant line is purple (' + ench.colour + ')');
  check(ench.left >= 0 && ench.right <= ench.vw, size + ': the enchant tooltip fits the screen');
  if (outdir) { await page.screenshot({ path: path.join(outdir, '71-tooltip-enchant-' + size + '.png') }); }
  // The Company window's gear row carries the relic words in its tooltip.
  await page.evaluate(([reaper]) => {
    window.gmcp('Company.Inventory', { shared: true, slots: [{ slot: 'weapon', label: 'Weapon' }], load: { total_g: 1000, capacity_g: 50000, member_capacity_g: 50000, mount_capacity_g: 0, cargo_g: 0 },
      members: [{ key: 'leader', name: 'Wren', grams: 3600, worn: [{ ref: '!50001:a', name: 'Ashen Reaper', label: 'Ashen Reaper', grams: 3600, slot: 'weapon', type: 'weapon', relic: reaper.relic }], carried: [] }],
      companions_known: true, horses: [], cargo: [] });
  }, [reaper]);
  const title = await page.evaluate(() => { const r = [...document.querySelectorAll('.cmp-slot')].find(e => e.textContent.includes('Ashen Reaper')); return r ? r.title : null; });
  check(title && title.includes('Reaping (while worn)'), size + ': Company gear row tooltip names the signature');
  await page.close();
}
await browser.close();
console.log(failures ? failures + ' failed' : 'all passed');
process.exit(failures ? 1 : 0);
