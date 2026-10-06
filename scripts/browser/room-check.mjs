// Phase 45 browser check: the Character window names a promoted class and
// the Room Info window shows a gather in progress (a bar over the work's
// length, then its result), through room-harness.html with the real web
// client core.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/room-check.mjs [outdir]
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

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'room-harness.html'));
await page.evaluate(() => localStorage.clear());
await page.reload();

// --- the Character window names the promoted class (38c1's route line) ---
await page.evaluate(() => window.gmcp('Char.Info', { name: 'Wren', class: 'Cleric', route: 'Priest', tier: 'advanced', rank: 6, race: 'Human', level: 12 }));
const heading = await page.evaluate(() => document.getElementById('cw-char-name').textContent);
check(heading.includes('Wren') && heading.includes('Cleric') && heading.includes('Priest, rank 6'), 'the Character window names lineage and class: ' + heading);
check(heading.split('Priest').length === 2, 'the class is named once: ' + heading);
if (outdir) { await page.locator('#cw-char-name').screenshot({ path: path.join(outdir, 'character-class.png') }); }

// --- the Room Info window shows gathering progress ---
await page.evaluate(() => window.gmcp('Room.Info', { name: 'A Mossy Yard', area: 'Frost Vale', environment: 'land', exits: {}, Contents: {} }));
const visible = () => page.evaluate(() => getComputedStyle(document.getElementById('rw-gather')).display !== 'none');
check(!(await visible()), 'no strip before any work');
await page.evaluate(() => window.gmcp('Room.Gather', { phase: 'start', kind: 'herbs', label: 'gathering herbs', seconds: 20 }));
check(await visible(), 'the strip shows when work starts');
const what = await page.evaluate(() => document.getElementById('rw-gather-what').textContent);
check(what.startsWith('Gathering herbs'), 'it names the work: ' + what);
const left = await page.evaluate(() => document.getElementById('rw-gather-left').textContent);
check(/^(19|20)s left$/.test(left), 'and the time left: ' + left);
await page.waitForTimeout(1200);
const width = await page.evaluate(() => document.getElementById('rw-gather-fill').getBoundingClientRect().width / document.getElementById('rw-gather-track').getBoundingClientRect().width);
check(width > 0.02 && width < 0.5, 'the bar fills over the work\'s length: ' + width.toFixed(2));
if (outdir) { await page.locator('#room-window').screenshot({ path: path.join(outdir, 'room-gather-progress.png') }); }
await page.evaluate(() => window.gmcp('Room.Gather', { phase: 'done', kind: 'herbs', label: 'gathering herbs', lines: ['Your company gathers 3 wild thyme.', 'The herbs here are picked clean for now.'] }));
const result = await page.evaluate(() => document.getElementById('rw-gather-result').textContent);
check(result.includes('3 wild thyme') && result.includes('picked clean'), 'the result shows when it finishes: ' + result);
check(await page.evaluate(() => getComputedStyle(document.getElementById('rw-gather-track')).display === 'none'), 'and the bar goes away');
if (outdir) { await page.locator('#room-window').screenshot({ path: path.join(outdir, 'room-gather-result.png') }); }
await page.evaluate(() => window.gmcp('Room.Gather', { phase: 'start', kind: 'fishing', label: 'fishing', seconds: 30 }));
await page.evaluate(() => window.gmcp('Room.Gather', { phase: 'stopped', kind: 'fishing', label: 'fishing', lines: ['You stop what you were doing; the work is abandoned.'] }));
const stopped = await page.evaluate(() => document.getElementById('rw-gather-what').textContent + ' / ' + document.getElementById('rw-gather-result').textContent);
check(stopped.includes('Work stopped') && stopped.includes('abandoned'), 'a stopped job says so: ' + stopped);
// Phase 46: a client that reconnected mid-work asks for Room.Gather on its
// first room and resumes the bar where the work stands.
await page.evaluate(() => { window.requests = []; Client.GMCPRequest = function(id) { window.requests.push(id); }; });
await page.reload();
await page.evaluate(() => { window.requests = []; Client.GMCPRequest = function(id) { window.requests.push(id); }; });
await page.evaluate(() => window.gmcp('Room.Info', { name: 'A Mossy Yard', area: 'Frost Vale', environment: 'land', exits: {}, Contents: {} }));
await page.evaluate(() => window.gmcp('Room.Info', { name: 'A Mossy Yard', area: 'Frost Vale', environment: 'land', exits: {}, Contents: {} }));
const asked = await page.evaluate(() => window.requests.filter(r => r === 'Room.Gather').length);
check(asked === 1, 'the Room window asks for the work in progress once, on its first room: ' + asked);
await page.evaluate(() => window.gmcp('Room.Gather', { phase: 'start', kind: 'herbs', label: 'gathering herbs', seconds: 20, elapsed: 15 }));
const resumedLeft = await page.evaluate(() => document.getElementById('rw-gather-left').textContent);
check(/^(5|6)s left$/.test(resumedLeft), 'a resumed bar shows only the time left: ' + resumedLeft);
const resumedWidth = await page.evaluate(() => document.getElementById('rw-gather-fill').getBoundingClientRect().width / document.getElementById('rw-gather-track').getBoundingClientRect().width);
check(resumedWidth > 0.7, 'and starts most of the way along: ' + resumedWidth.toFixed(2));
// Room.Info still works beside it.
await page.evaluate(() => window.gmcp('Room.Info', { name: 'The Road', area: 'Frost Vale', environment: 'land', exits: {}, Contents: {} }));
check(await page.evaluate(() => document.getElementById('rw-room-name').textContent.includes('The Road')), 'Room.Info still updates the window');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all room checks passed');
