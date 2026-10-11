// LM2: light and sight on the map, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const S = require(path.join(here, '../../_datafiles/html/public/static/js/map-sight.js'));

const ids = set => (set ? Array.from(set).sort((a, b) => a - b) : set);

test('daylight at full sight, or no sight at all, shows the map as today', () => {
  assert.equal(S.visible({ level: 2, daylight: true }, 1, [2, 3]), null);
  assert.equal(S.visible(null, 1, [2]), null);
  assert.equal(S.visible({}, 1, [2]), null, 'an old server sends no level');
  assert.equal(S.glow({ level: 2, daylight: true }), null);
});

test('in the dark, sight reaches your room and the rooms its exits join, or less', () => {
  assert.deepEqual(ids(S.visible({ level: 2, daylight: false, source: 'torch' }, 5, [6, 7])), [5, 6, 7]);
  assert.deepEqual(ids(S.visible({ level: 1, daylight: false }, 5, [6, 7])), [5], 'level 1: your room only');
  assert.deepEqual(ids(S.visible({ level: 0, daylight: false }, 5, [6, 7])), [], 'level 0: not even your room');
  assert.deepEqual(ids(S.visible({ level: 1, daylight: true }, 5, [6])), [5], 'thick fog by day still limits sight');
  assert.deepEqual(ids(S.visible({ level: 2, daylight: false }, null, [])), [], 'no room yet');
});

test('the glow is warm for a flame, cool for a party light, pale for the moon, none for night vision', () => {
  assert.deepEqual(S.glow({ level: 2, source: 'lantern' }), { radius: 1.5, color: 'warm', mono: false });
  assert.deepEqual(S.glow({ level: 1, source: 'torch' }), { radius: 0.6, color: 'warm', mono: false });
  assert.deepEqual(S.glow({ level: 1, source: 'fixture' }), { radius: 0.6, color: 'warm', mono: false });
  assert.deepEqual(S.glow({ level: 1, source: 'party' }), { radius: 0.6, color: 'cool', mono: false });
  assert.deepEqual(S.glow({ level: 2, source: 'orb' }), { radius: 1.5, color: 'cool', mono: false });
  assert.deepEqual(S.glow({ level: 1, source: '' }), { radius: 0.6, color: 'pale', mono: false }, 'moonlight');
  assert.deepEqual(S.glow({ level: 0, source: '' }), { radius: 0, color: '', mono: false });
  assert.deepEqual(S.glow({ level: 2, source: 'nightvision' }), { radius: 0, color: '', mono: true });
});

test('the marker shows the light you really carry', () => {
  assert.equal(S.marker({ level: 1, source: 'torch' }), 'torch');
  assert.equal(S.marker({ level: 1, source: 'lantern' }), 'lantern');
  assert.equal(S.marker({ level: 2, source: 'orb' }), 'orb');
  assert.equal(S.marker({ level: 1, source: 'party' }), '', 'an ally\'s light is over the ally');
  assert.equal(S.marker({ level: 0, carried: 'lantern-out' }), 'lantern-out');
  assert.equal(S.marker({ level: 2, daylight: true, source: 'lantern' }), 'lantern', 'a lit lantern shows by day too');
  assert.equal(S.marker(null), '');
});

test('the key changes when the drawing would', () => {
  assert.notEqual(S.key({ level: 1, source: 'torch' }), S.key({ level: 1, source: 'lantern' }));
  assert.notEqual(S.key({ level: 1 }), S.key({ level: 2 }));
  assert.equal(S.key({ level: 1, source: 'torch' }), S.key({ level: 1, source: 'torch' }));
  assert.equal(S.key(null), '');
});
