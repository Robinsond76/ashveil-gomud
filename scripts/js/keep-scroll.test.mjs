// keepScroll puts scroll positions back after a panel rebuild (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const keepScroll = require(path.join(here, '../../_datafiles/html/public/static/js/keep-scroll.js'));

test('restores the node and its scrolled ancestors after a rebuild', async () => {
    const outer = { scrollTop: 40, parentElement: null };
    const panel = { scrollTop: 120, parentElement: outer };
    keepScroll(panel);
    panel.scrollTop = 0; // the rebuild empties the panel
    outer.scrollTop = 0;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 120);
    assert.equal(outer.scrollTop, 40);
});

test('does nothing for an unscrolled panel', async () => {
    const panel = { scrollTop: 0, parentElement: null };
    keepScroll(panel);
    panel.scrollTop = 5;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 5);
});

// A box whose content can be shorter than its scroll position, like a
// browser's scroll container: scrollTop is clamped to max.
function box(max, parentElement = null) {
    let top = 0;
    return {
        max, parentElement,
        get scrollTop() { return top; },
        set scrollTop(v) { top = Math.max(0, Math.min(v, this.max)); },
    };
}

test('a rebuild that is briefly too short does not lose the position', async () => {
    const panel = box(500);
    panel.scrollTop = 120;
    keepScroll(panel);
    panel.max = 0; // "Nothing yet" while the Inventory payload is on its way
    panel.scrollTop = 0;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 0, 'clamped by the short content');
    keepScroll(panel); // the full rebuild
    panel.max = 500;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 120);
});

test('a clamped position is forgotten once the player scrolls', async () => {
    const panel = box(500);
    panel.scrollTop = 120;
    keepScroll(panel);
    panel.max = 50;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 50);
    panel.scrollTop = 10; // the player scrolls up
    keepScroll(panel);
    panel.max = 500;
    await Promise.resolve();
    assert.equal(panel.scrollTop, 10);
});
