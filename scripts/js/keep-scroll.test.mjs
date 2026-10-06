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
