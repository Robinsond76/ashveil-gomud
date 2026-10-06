// Browser check: arrow-key walking and the Enter quick menu in the real web
// client (webclient-pure.html with its template fields filled in, served
// over HTTP). A fake WebSocket records what the client sends.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/quickmenu-check.mjs [outdir]
import { createRequire } from 'node:module';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));
const pub = path.join(path.resolve(here, '../..'), '_datafiles/html/public');
const outdir = process.argv[2];

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json', '.webmanifest': 'application/manifest+json', '.mp3': 'audio/mpeg' };
const server = http.createServer((req, res) => {
  const url = decodeURIComponent(req.url.split('?')[0]);
  if (url === '/favicon.ico') { res.writeHead(204); res.end(); return; }
  if (url === '/webclient-pure.html') {
    const html = fs.readFileSync(path.join(pub, 'webclient-pure.html'), 'utf8')
      .replaceAll('{{ .ASSET_BASE_URL }}', '').replaceAll('{{ .CONFIG.Server.MudName }}', 'Ashveil');
    res.writeHead(200, { 'content-type': 'text/html' });
    res.end(html);
    return;
  }
  const file = path.join(pub, url);
  if (!file.startsWith(pub) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const base = 'http://127.0.0.1:' + server.address().port;

const room = {
  num: 7, name: 'The Trappers\' Post', area: 'Frost Vale', coords: 'Frost Vale,0,0,0', environment: 'land',
  exits: { north: 8, east: 9 }, exitsv2: { east: { num: 9, details: ['locked'] } },
  resources: ['herbs', 'fishing'], details: ['trainer'],
  Contents: {
    Npcs: [{ id: 11, name: 'Old Trapper', adjectives: ['shop'] }, { id: 12, name: 'giant rat', aggro: true }],
    Items: [{ id: 'i:1', name: 'dagger', label: 'a rusty dagger' }],
  },
};

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined });
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.addInitScript(() => {
  window.wsSent = [];
  window.WebSocket = class {
    constructor() { this.readyState = 1; setTimeout(() => this.onopen && this.onopen({}), 0); }
    send(d) { window.wsSent.push(d); }
    close() { this.readyState = 3; }
    addEventListener() {}
  };
  window.WebSocket.OPEN = 1;
});
await page.goto(base + '/webclient-pure.html');
await page.evaluate(() => localStorage.clear());
await page.reload();
await page.click('#connect-button');
await page.waitForTimeout(200);
const gmcp = (ns, body) => page.evaluate(([n, b]) => {
  const parts = n.split('.'); const last = parts.pop();
  let c = Client.GMCPStructs; parts.forEach(s => { c = c[s] = c[s] || {}; }); c[last] = b;
  VirtualWindows.handleGMCP(n, b);
}, [ns, body]);
const sent = () => page.evaluate(() => window.wsSent.filter(s => !s.startsWith('!!GMCP')));
const clear = () => page.evaluate(() => { window.wsSent = []; });
const menuOpen = () => page.evaluate(() => !!document.getElementById('quickmenu'));
const rows = () => page.evaluate(() => [...document.querySelectorAll('#quickmenu .qm-row .qm-label')].map(n => n.textContent));
const selected = () => page.evaluate(() => (document.querySelector('#quickmenu .qm-row.sel .qm-label') || {}).textContent);
const shot = async name => { if (outdir) { await page.screenshot({ path: path.join(outdir, 'quickmenu-' + name + '.png') }); } };

await gmcp('Room.Info', room);
await gmcp('Company', { leader: { key: 'leader', name: 'Wren' }, members: [], alive: 1, dead: 0 });
await page.focus('#command-input');
await clear();

// --- the arrows walk while the box is empty ---
await page.keyboard.press('ArrowUp');
await page.keyboard.press('ArrowRight');
await page.keyboard.press('ArrowDown');
await page.keyboard.press('ArrowLeft');
check(JSON.stringify(await sent()) === JSON.stringify(['north', 'east', 'south', 'west']), 'empty box: arrows walk north, east, south, west: ' + JSON.stringify(await sent()));

// --- typing keeps the arrows for the box ---
await clear();
await page.keyboard.type('say hi');
await page.keyboard.press('ArrowLeft');
await page.keyboard.press('ArrowRight');
check((await sent()).length === 0, 'with text typed, left and right edit the text, they do not walk');
check(await page.inputValue('#command-input') !== '', 'and the typed text stays');
await page.fill('#command-input', '');

// --- history still reachable with Alt ---
await page.keyboard.type('look');
await page.keyboard.press('Enter');
await clear();
await page.keyboard.press('Alt+ArrowUp');
check(await page.inputValue('#command-input') === 'look', 'Alt+Up recalls the last command');
await page.fill('#command-input', '');

// --- Enter on an empty box opens the menu and sends nothing ---
await clear();
await page.keyboard.press('Enter');
check(await menuOpen(), 'Enter on an empty box opens the quick menu');
check((await sent()).length === 0, 'and sends nothing');
const top = await rows();
check(['Look', 'Attack', 'Move', 'Services', 'Get', 'Gather', 'Company', 'Me', 'Help', 'Close'].every(l => top.includes(l)), 'the room\'s options are listed: ' + top.join(', '));
check(await selected() === 'Look', 'the first entry is selected');
await shot('root');

// --- arrows move the highlight, they do not walk ---
await page.keyboard.press('ArrowDown');
check(await selected() === 'Attack', 'Down moves to the next entry');
check((await sent()).length === 0, 'and does not walk');
await page.keyboard.press('ArrowUp');
await page.keyboard.press('ArrowUp');
check(await selected() === 'Close', 'Up from the top wraps to Close');
await page.keyboard.press('ArrowDown');

// --- attack opens a submenu of this room's foes; the pick sends by id ---
await page.keyboard.press('ArrowDown');
await page.keyboard.press('Enter');
check(JSON.stringify(await rows()) === JSON.stringify(['Old Trapper', 'giant rat', 'Back']), 'Attack lists the room\'s foes and Back: ' + JSON.stringify(await rows()));
await shot('attack');
await page.keyboard.press('Escape');
check((await rows()).includes('Attack') && await selected() === 'Attack', 'Esc goes back to the first menu, on the entry it came from');
await page.keyboard.press('Enter');
await page.keyboard.press('ArrowDown');
await page.keyboard.press('Enter');
check(!(await menuOpen()), 'choosing a command closes the menu');
check(JSON.stringify(await sent()) === JSON.stringify(['attack 12']), 'and sends attack by id: ' + JSON.stringify(await sent()));

// --- Back entry, number keys, Esc to close ---
await clear();
await page.keyboard.press('Enter');
await page.keyboard.press('3');
check((await rows()).includes('north') && (await rows()).includes('Back'), 'the 3 key opens Move');
const east = (await rows()).indexOf('east');
check(east >= 0, 'exits are listed');
await page.keyboard.press('ArrowDown'); await page.keyboard.press('ArrowDown');
await page.keyboard.press('ArrowDown'); await page.keyboard.press('ArrowDown');
// last entry is Back
await page.keyboard.press('End');
check(await selected() === 'Back', 'End reaches Back');
await page.keyboard.press('Enter');
check(await menuOpen() && (await rows()).includes('Look'), 'Back returns to the first menu with only the arrows and Enter');
await page.keyboard.press('Escape');
check(!(await menuOpen()), 'Esc at the first level closes it');
check((await sent()).length === 0, 'nothing was sent');
check(await page.evaluate(() => document.activeElement.id === 'command-input'), 'focus returns to the command box');

// --- the menu follows the room; a battle puts its orders first ---
await gmcp('Company.Battle', { group: 'wolves', enemies: [{ id: 'm:1', label: 'wolf' }], focus_ready: true, focus: 'none' });
await page.keyboard.press('Enter');
const inFight = await rows();
check(inFight[0] === 'Battle' && !inFight.includes('Attack'), 'in a battle the menu leads with Battle and drops Attack: ' + inFight.join(', '));
await page.keyboard.press('Enter');
check((await rows())[0] === 'Retreat', 'Battle lists Retreat first');
await shot('battle');
await page.keyboard.press('Enter');
check(JSON.stringify(await sent()) === JSON.stringify(['retreat']), 'and Retreat sends retreat');
await gmcp('Company.Battle', {});
await gmcp('Room.Info', { num: 20, name: 'A Bare Ledge', area: 'Frost Vale', coords: 'Frost Vale,1,0,0', exits: {}, Contents: {} });
await page.keyboard.press('Enter');
check(!(await rows()).includes('Move') && !(await rows()).includes('Attack'), 'a bare room has no Move or Attack');
await page.keyboard.press('Escape');

// --- Enter still sends a typed line ---
await clear();
await page.keyboard.type('say hello');
await page.keyboard.press('Enter');
check(JSON.stringify(await sent()) === JSON.stringify(['say hello']) && !(await menuOpen()), 'Enter with text sends the line and opens no menu');

// --- a mouse can drive it too ---
await gmcp('Room.Info', room);
await page.focus('#command-input');
await page.keyboard.press('Enter');
await clear();
await page.locator('#quickmenu .qm-row', { hasText: 'Get' }).click();
await page.locator('#quickmenu .qm-row', { hasText: 'Get everything here' }).click();
check(JSON.stringify(await sent()) === JSON.stringify(['get all']), 'clicking an entry chooses it: ' + JSON.stringify(await sent()));
await page.keyboard.press('Enter');
await page.mouse.click(5, 5);
check(!(await menuOpen()), 'clicking outside closes the menu');

await browser.close();
server.close();
console.log(failures ? failures + ' failure(s)' : 'all passed');
process.exit(failures ? 1 : 0);
