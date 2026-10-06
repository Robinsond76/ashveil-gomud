// Weather browser check: drives the real Time and Date window script
// (through weather-harness.html, with the real web client core) in Chromium
// with Playwright, feeding it Gametime payloads with and without weather and
// screenshotting the sky for each.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/weather-check.mjs [outdir]
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

const base = { calendar: 'gregorian/Ashveil', hour: 2, hour24: 14, minute: 0, ampm: 'PM', day: 3, month: 2, month_name: 'Frost', year: 1, zodiac: 'Heron', night: false, day_start: 6, night_start: 22, sun_count: 1, moon_count: 1 };
const night = { ...base, hour: 11, hour24: 23, ampm: 'PM', night: true };
const wx = (name, cloud_cover, visibility = 0, extra = {}) => ({ name, description: 'The sky: ' + name + '.', cloud_cover, visibility, indoor: false, ...extra });

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1000, height: 600 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'weather-harness.html'));

const label = () => page.evaluate(() => {
  const el = document.getElementById('gametime-weather');
  return el && el.style.display !== 'none' ? el.textContent : null;
});
const send = (data) => page.evaluate(d => window.gmcp('Gametime', d), data);

await send(base);
check(await label() === null, 'no weather in the payload shows no weather tag');

const cases = [
  ['clear', base, wx('clear', 0), 'Clear'],
  ['overcast', base, wx('overcast', 3), 'Overcast'],
  ['rain', base, wx('rain', 3), 'Rain'],
  ['storm', base, wx('storm', 3), 'Storm'],
  ['fog', base, wx('fog', 2, -1), 'Fog'],
  ['thick-fog', base, wx('thick-fog', 3, -2), 'Thick fog'],
  ['night-rain', night, wx('rain', 3), 'Rain'],
  ['night-clear', night, wx('clear', 0), 'Clear'],
];
let lastTime = base;
for (const [key, time, weather, text] of cases) {
  if (time !== lastTime) {
    // The panel learns the clock rate from consecutive packets, so a jump
    // in hours needs a fresh page.
    await page.reload();
    lastTime = time;
  }
  await send({ ...time, weather });
  await page.waitForTimeout(250);
  check(await label() === text, key + ': the tag reads "' + text + '"');
  if (outdir) {
    const panel = await page.$('#gametime-panel');
    await panel.screenshot({ path: path.join(outdir, 'weather-' + key + '.png') });
  }
}

await send({ ...base, weather: wx('rain', 3, 0, { indoor: true }) });
check(await label() === 'Rain (outside)', 'a view through an exit says so');
await send(base);
check(await label() === null, 'leaving the weather clears the tag');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('weather check passed');
