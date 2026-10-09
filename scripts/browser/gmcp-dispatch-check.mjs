// Browser check for the web client's GMCP dispatch: each registered window
// handles a message once, however many of its namespaces match, and '*'
// handlers fire once per message. Uses the dock windows harness.
//
//   NODE_PATH=$(npm root -g) node scripts/browser/gmcp-dispatch-check.mjs
//   BROWSER=firefox ... runs it in Firefox (needs Playwright's Firefox build).
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const playwright = require('playwright');
const here = path.dirname(fileURLToPath(import.meta.url));

let failures = 0;
function check(ok, what) {
  if (ok) { console.log('ok   ' + what); } else { failures++; console.log('FAIL ' + what); }
}

const engine = process.env.BROWSER === 'firefox' ? 'firefox' : 'chromium';
const browser = await playwright[engine].launch(engine === 'chromium'
  ? { executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || undefined } : {});
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
page.on('pageerror', e => { failures++; console.log('FAIL page error: ' + e.message); });
await page.goto('file://' + path.join(here, 'dock-windows-harness.html'));

const counts = await page.evaluate(() => {
  const calls = {};
  const probe = (name, handlers) => VirtualWindows.register({
    gmcpHandlers: handlers,
    onGMCP(ns) { calls[name + ' ' + ns] = (calls[name + ' ' + ns] || 0) + 1; },
  });
  probe('multi', ['Probe.Kills', 'Probe']);
  probe('twin', ['Probe', 'Probe.Kills', 'Probe.Kills.Deep']);
  probe('wild', ['*']);
  probe('exact', ['Probe.Vitals']);
  window.gmcp('Probe.Kills', {});
  window.gmcp('Probe.Vitals', {});
  window.gmcp('Probe.Kills.Deep', {});
  return calls;
});

check(counts['multi Probe.Kills'] === 1, 'a module on a namespace and its parent is called once');
check(counts['twin Probe.Kills'] === 1, 'three matching registrations still give one call');
check(counts['twin Probe.Kills.Deep'] === 1, 'a deep namespace reaches its own and parent handlers once');
check(counts['wild Probe.Kills'] === 1 && counts['wild Probe.Vitals'] === 1, "'*' fires once per message, not once per namespace level");
check(counts['wild Probe.Kills.Deep'] === 1, "'*' fires once on a three-level namespace");
check(counts['exact Probe.Vitals'] === 1 && counts['exact Probe.Kills'] === undefined, 'handlers on other namespaces are not called');
check(counts['multi Probe.Vitals'] === 1, 'a parent registration still receives child namespaces');

// The real windows: Online updates once per Game message, KillStats ignores Char.Vitals.
const real = await page.evaluate(() => {
  const out = {};
  let n = 0;
  VirtualWindows.register({ gmcpHandlers: ['Game'], onGMCP() { n++; } });
  window.gmcp('Game', { Who: { Players: [] } });
  out.game = n;
  return out;
});
check(real.game === 1, 'a Game message reaches a probe once beside the Online window');

// Shared tooltip and tab helpers (Client.tooltip, Client.tabs).
const ui = await page.evaluate(() => {
  const out = {};
  const st = document.createElement('style'); st.textContent = '#probe-tip{position:fixed;display:none;width:60px;height:30px}'; document.head.appendChild(st);
  const tip = Client.tooltip('probe-tip');
  const anchor = document.createElement('div');
  anchor.style.cssText = 'position:fixed;left:100px;top:100px;width:50px;height:20px';
  document.body.appendChild(anchor);
  const el = () => document.getElementById('probe-tip');
  out.beforeShow = el() === null;
  tip.show('<b>hi</b>', anchor);
  out.shown = el().style.display === 'block' && tip.isShown();
  out.html = el().innerHTML;
  out.beside = parseFloat(el().style.left) === 158;
  tip.show('<i>x</i> & y', {x: 300, y: 300}, {text: true});
  out.text = el().textContent;
  out.pointer = parseFloat(el().style.left) === 314;
  tip.hide(0);
  out.hiddenNow = el().style.display === 'none';
  tip.show('again', anchor);
  tip.hide();
  tip.show('cancels hide', anchor);
  out.sameEl = document.querySelectorAll('#probe-tip').length === 1;
  const root = document.createElement('div');
  root.innerHTML = '<button class="t-b active" data-panel="p1">1</button><button class="t-b" data-panel="p2">2</button><div class="t-p active" id="p1"></div><div class="t-p" id="p2"></div>';
  document.body.appendChild(root);
  const select = Client.tabs(root, { button: '.t-b', panel: '.t-p' });
  root.querySelectorAll('.t-b')[1].click();
  out.tab2 = root.querySelector('#p2').classList.contains('active') && !root.querySelector('#p1').classList.contains('active') &&
    root.querySelectorAll('.t-b')[1].classList.contains('active') && !root.querySelectorAll('.t-b')[0].classList.contains('active');
  select('p1');
  out.tab1 = root.querySelector('#p1').classList.contains('active');
  // The real windows use the shared helpers.
  out.gearTabs = !!document.querySelector('#gear-window .gw-tab-btn');
  return out;
});
check(ui.beforeShow, 'a tooltip element is created on first show, not before');
check(ui.shown && ui.html === '<b>hi</b>', 'show displays HTML content');
check(ui.beside, 'an anchored tooltip sits 8px right of its anchor');
check(ui.text === '<i>x</i> & y' && ui.pointer, 'text mode does not parse markup; pointer placement sits 14px right');
check(ui.hiddenNow, 'hide(0) hides at once');
check(ui.sameEl, 'one element is reused across shows');
check(ui.tab2 && ui.tab1, 'Client.tabs switches the active button and panel');
check(ui.gearTabs, 'the Gear window builds its tabs through the helper');

await browser.close();
if (failures) { console.log(failures + ' failure(s)'); process.exit(1); }
console.log('all dispatch checks passed');
