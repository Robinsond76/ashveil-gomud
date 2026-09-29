// Phase 32g browser check: drives the real dock core (webclient-core.js,
// through dock-harness.html) in Chromium with Playwright: the right column
// is one tab group with the vitals strip above its tabs, the left column
// stacks as before, the active tab survives a reload, a tab pops out and
// docks back into its place, a badge shows and is spoken, arrow keys move
// between tabs, a layout saved before 32g is replaced once with a notice,
// and the group's panel goes when its last member leaves.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/dock-check.mjs [outdir]
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

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
const url = 'file://' + path.join(here, 'dock-harness.html');
await page.goto(url);
await page.evaluate(() => localStorage.clear());
await page.reload();

const tabs = () => page.evaluate(() => [...document.querySelectorAll('#dock-right [role=tab]')].map(t => ({
  label: t.textContent, selected: t.getAttribute('aria-selected') === 'true',
})));
const visible = () => page.evaluate(() => [...document.querySelectorAll('#dock-right .dock-tabgroup-pane')]
  .filter(p => !p.hidden).map(p => p.textContent));

// The default layout.
check(await page.evaluate(() => document.querySelectorAll('#dock-right .dock-panel').length) === 1, 'the right column is one panel');
check(await page.evaluate(() => document.querySelector('#dock-left [data-stub=Map]') !== null), 'a window without a group stacks in its column as before');
check(JSON.stringify((await tabs()).map(t => t.label)) === JSON.stringify(['Character', 'Company', 'Combat', 'Comm']), 'tabs in order, with a tab label');
check(await page.evaluate(() => document.querySelector('.dock-tabgroup-header [data-stub=Vitals]') !== null), 'the vitals strip sits above the tabs');
check((await tabs())[0].selected && JSON.stringify(await visible()) === '["Character content"]', 'the first tab shows, the others are hidden');
check(await page.evaluate(() => window.notice === false), 'no notice on a fresh browser');
check(await page.getByRole('tablist').count() === 1 && await page.getByRole('tabpanel').count() === 1, 'the accessibility tree names the tablist and the one visible panel');
if (outdir) { await page.screenshot({ path: path.join(outdir, 'dock-default.png') }); }

// Choosing a tab; it survives a reload.
await page.getByRole('tab', { name: 'Company' }).click();
check(JSON.stringify(await visible()) === '["Company content"]', 'a click shows its tab');
check(await page.evaluate(() => document.querySelector('#dock-right .dock-panel-title').textContent) === 'Company', 'the panel title follows the tab');
await page.reload();
check((await tabs()).find(t => t.selected).label === 'Company', 'the active tab survives a reload');

// Keyboard: arrows move between tabs.
await page.getByRole('tab', { name: 'Company' }).focus();
await page.keyboard.press('ArrowRight');
check((await tabs()).find(t => t.selected).label === 'Combat', 'ArrowRight moves to the next tab');
await page.keyboard.press('Home');
check((await tabs()).find(t => t.selected).label === 'Character', 'Home moves to the first tab');

// A badge, spoken, and cleared.
await page.evaluate(() => VirtualWindows.setTabBadge('Communications', 3));
check(await page.getByRole('tab', { name: 'Comm, 3 new' }).count() === 1, 'a badge shows and is part of the tab\'s name');
check(await page.evaluate(() => VirtualWindows.isTabShowing('Communications')) === false, 'the tab is known not to be showing');
const before = await page.evaluate(() => window.shown || 0);
await page.getByRole('tab', { name: /Comm/ }).click();
check(await page.evaluate(() => window.shown || 0) === before + 1, 'the window is told its tab is shown');
await page.evaluate(() => VirtualWindows.setTabBadge('Communications', 0));
check(await page.evaluate(() => document.querySelector('#dock-right .dock-tabgroup-badge:not([hidden])') === null), 'a cleared badge hides');

// Pop out the active tab, then dock it back.
await page.getByRole('tab', { name: 'Company' }).click();
await page.click('#dock-right .dock-panel-popout');
check(await page.evaluate(() => document.querySelector('.vwin [data-stub=Company]') !== null), 'the active tab pops out into a floating window');
check(!(await tabs()).some(t => t.label === 'Company'), 'and leaves the tab row');
check((await tabs()).find(t => t.selected).label === 'Character', 'another tab shows in its place');
await page.click('.vwin .vw-dock-btn');
check((await tabs()).map(t => t.label).join(',') === 'Character,Company,Combat,Comm', 'docking it returns it to its place');

// Leaving: the panel goes with its last member.
await page.evaluate(() => Object.values(window.wins).filter(w => w._id !== 'Map').forEach(w => { w._removeDocked(); w._win = false; }));
check(await page.evaluate(() => !document.getElementById('dock-right').classList.contains('has-panels')), 'the group panel goes when its last member leaves');

// A layout saved before 32g is replaced once, with a notice.
await page.evaluate(() => localStorage.setItem('windowLayout', JSON.stringify({ windows: { Map: { enabled: true, docked: true, dockSide: 'right' } }, dockOrder: { left: [], right: ['Map'] } })));
await page.reload();
check(await page.evaluate(() => window.notice === true), 'an old layout gives the notice');
check(await page.evaluate(() => document.querySelector('#dock-left [data-stub=Map]') !== null), 'and the new defaults apply (the map on the left)');
await page.reload();
check(await page.evaluate(() => window.notice === false), 'the notice shows only once');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all dock checks passed');
