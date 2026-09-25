// Phase 27d browser check: drives the real window-tutorial.js (through
// tutorial-panel-harness.html) in Chromium with Playwright.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/tutorial-panel-check.mjs [outdir]
//
// It checks the stage, goal, checklist, and hints; done and to-do in
// words; markup in a string rendered as text; the empty state; a reopened
// window being current; a 360px-wide layout; the accessibility tree; and
// that only a new stage or a newly done item is announced.
//
// The harness stubs the client's window and GMCP plumbing (mirroring
// webclient-core.js) rather than loading the whole client, and measures the
// panel in a host at most 420px wide.
// With an outdir it saves desktop and narrow screenshots there. Exits
// non-zero on any failure.
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const outdir = process.argv[2];

const view = {
  active: true, stage: 3, stages: 8, id: 'formation', title: 'Formation',
  goal: 'Put one companion in the front row and another behind it.',
  checklist: [
    { label: 'a companion in the front row', done: true },
    { label: 'another behind it', done: false },
  ],
  hints: [
    'formation shows the grid, row by row.',
    'formation move <name> <row> <col> places someone, e.g. formation move tamsin 1 2.',
    '<img src=x onerror="window.__xss=1"> is not markup',
  ],
};

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM || undefined });
const page = await browser.newPage({ viewport: { width: 1024, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'tutorial-panel-harness.html'));
await page.waitForTimeout(20);

const text = () => page.evaluate(() => document.querySelector('#tutorial-panel .tutorial-content').textContent);
const status = () => page.evaluate(() => document.querySelector('#tutorial-panel .tutorial-status').textContent);
const clearStatus = () => page.evaluate(() => { document.querySelector('#tutorial-panel .tutorial-status').textContent = ''; });

check((await text()).includes('Not in the tutorial.'), 'empty state before any payload');
check(JSON.stringify(await page.evaluate(() => window.requests)) === '[]', 'the panel sends no request of its own');

await page.evaluate(v => window.gmcp('Tutorial', v), view);
const t1 = await text();
check(t1.includes('Stage 3 of 8: Formation'), 'title and stage');
check(await status() === 'Stage 3 of 8: Formation', 'a new stage is announced');
check(t1.includes('Goal: Put one companion in the front row and another behind it.'), 'goal');
check(await page.locator('.tutorial-check').count() === 2, 'two checklist items');
check(t1.includes('a companion in the front row(done)') && t1.includes('another behind it(to do)'), 'done and to do in words');
check(t1.includes('formation move <name> <row> <col>'), 'angle brackets in a hint shown as text');
check(await page.evaluate(() => window.__xss === undefined) && t1.includes('<img src=x onerror="window.__xss=1">'), 'markup in a string is not interpreted');
check(await page.locator('#tutorial-panel img').count() === 0, 'no element made from a string');
if (outdir) { await page.locator('#tutorial-panel').screenshot({ path: path.join(outdir, 'tutorial-desktop.png') }); }

// Accessibility tree: a region with a heading and the two lists; the
// [x] marks are hidden, the words spoken.
const aria = await page.locator('#tutorial-panel').ariaSnapshot();
check(/heading "Stage 3 of 8: Formation"/.test(aria), 'heading in the accessibility tree');
check(!/bullet|\u2022/.test(aria), 'hints are list items without bullet characters');
check(await page.evaluate(() => document.getElementById('tutorial-panel').getAttribute('aria-live')) === null, 'the rebuilt panel is not itself a live region');

// Announcements: a newly done item, and nothing for an unchanged payload.
await clearStatus();
await page.evaluate(v => window.gmcp('Tutorial', v), view);
check(await status() === '', 'an unchanged payload announces nothing');
await page.evaluate(v => window.gmcp('Tutorial', v), { ...view, checklist: [{ label: 'a companion in the front row', done: true }, { label: 'another behind it', done: true }] });
check(await status() === 'Done: another behind it', 'a newly done item is announced');
check(/list "Checklist"/.test(aria) && /list "Hints"/.test(aria), 'checklist and hints lists');
check(/another behind it/.test(aria) && /\(to do\)/.test(aria) && !/\[x\]/.test(aria), 'state spoken in words, marks hidden');
check(await page.getByRole('region', { name: 'Tutorial' }).count() === 1, 'a named region');

// The next stage replaces the last.
await page.evaluate(v => window.gmcp('Tutorial', v), { ...view, stage: 4, id: 'survival', title: 'Survival', checklist: [{ label: 'eat something', done: false }], hints: [] });
const t2 = await text();
check(t2.includes('Stage 4 of 8') && !t2.includes('Formation'), 'a new stage replaces the old');
check(await page.locator('.tutorial-hints').count() === 0, 'no empty hints list');

// Closed window: payloads still land in the store; reopening shows them.
await page.evaluate(() => window.windows[0].close());
await page.evaluate(v => window.gmcp('Tutorial', v), { ...view, stage: 5, id: 'camp', title: 'Camp', checklist: [{ label: 'Rested', done: true }] });
await page.evaluate(() => window.windows[0].reopen());
await page.waitForTimeout(20);
check((await text()).includes('Camp') && (await text()).includes('Rested(done)'), 'a reopened window shows what arrived while it was closed');

// Narrow: no sideways scrolling at 360px, even with a long unbroken word.
await page.evaluate(v => window.gmcp('Tutorial', v), { ...view, hints: ['x'.repeat(200)] });
await page.setViewportSize({ width: 360, height: 900 });
const overflow = await page.evaluate(() => { const p = document.getElementById('tutorial-panel'); return p.scrollWidth - p.clientWidth; });
check(overflow <= 0, 'no horizontal overflow at 360px (' + overflow + ')');
await page.evaluate(v => window.gmcp('Tutorial', v), view);
if (outdir) { await page.locator('#tutorial-panel').screenshot({ path: path.join(outdir, 'tutorial-narrow.png') }); }

// Left the course: {} clears it; a malformed payload doesn't throw.
await clearStatus();
await page.evaluate(() => window.gmcp('Tutorial', {}));
check((await text()).includes('Not in the tutorial.'), '{} shows the empty state');
check(await status() === '', 'and announces nothing (every login gets one)');
await page.evaluate(() => window.gmcp('Tutorial', { active: true, title: 5, checklist: 'nope', hints: null }));
check((await text()).includes('Goal:'), 'a malformed payload renders what it can');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all checks passed');
