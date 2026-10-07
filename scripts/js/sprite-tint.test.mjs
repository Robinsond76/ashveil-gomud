// Phase 72a: skin and hair repaint on player sprites, under Node (make js-test).
import { createRequire } from 'node:module';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const T = require(path.join(here, '../../_datafiles/html/public/static/js/sprite-tint.js'));

const px = (...rgba) => Uint8ClampedArray.from(rgba.flat());

test('the base ramps match the generator palette', () => {
  const py = fs.readFileSync(path.join(here, '../sprites/palette.py'), 'utf8');
  for (const name of ['skin', 'hair']) {
    const m = py.match(new RegExp('"' + name + '": \\("([0-9a-f]{6})", "([0-9a-f]{6})", "([0-9a-f]{6})"\\)'));
    assert.ok(m, name + ' ramp is in palette.py');
    const want = [1, 2, 3].map(i => [0, 2, 4].map(o => parseInt(m[i].slice(o, o + 2), 16)));
    assert.deepEqual(T.BASE[name], want, name);
  }
});

test('look takes only plain #rrggbb colours', () => {
  assert.equal(T.look('', ''), null);
  assert.equal(T.look('red', 'url(x)'), null);
  assert.equal(T.look(undefined, null), null);
  assert.deepEqual(T.look('#C89A64', ''), { skin: '#c89a64', hair: '' });
  assert.deepEqual(T.look('nope', '#1c1612'), { skin: '', hair: '#1c1612' });
});

test('a ramp keeps the chosen colour as its mid step, darker below and lighter above', () => {
  const r = T.ramp('#c89a64');
  assert.deepEqual(r.m, [0xc8, 0x9a, 0x64]);
  for (let i = 0; i < 3; i++) {
    assert.ok(r.d[i] < r.m[i]);
    assert.ok(r.l[i] > r.m[i]);
  }
  // extremes stay inside 0..255
  for (const hex of ['#000000', '#ffffff', '#ecd0b4', '#52311d']) {
    const x = T.ramp(hex);
    for (const step of [x.d, x.m, x.l]) { for (const c of step) { assert.ok(c >= 0 && c <= 255); } }
  }
});

test('apply repaints only the skin and hair ramps', () => {
  const [sd, sm, sl] = T.BASE.skin, [hd, hm, hl] = T.BASE.hair;
  const iron = [0x55, 0x59, 0x5f];
  const data = px(
    [...sd, 255], [...sm, 255], [...sl, 255],
    [...hd, 255], [...hm, 255], [...hl, 255],
    [...iron, 255], [...sm, 0], [0x15, 0x12, 0x0f, 255],
  );
  const look = T.look('#7c4c2c', '#dcd8cc');
  assert.equal(T.apply(data, look), 6);
  const at = i => Array.from(data.slice(i * 4, i * 4 + 4));
  const skin = T.ramp('#7c4c2c'), hair = T.ramp('#dcd8cc');
  assert.deepEqual(at(0), [...skin.d, 255]);
  assert.deepEqual(at(1), [...skin.m, 255]);
  assert.deepEqual(at(2), [...skin.l, 255]);
  assert.deepEqual(at(3), [...hair.d, 255]);
  assert.deepEqual(at(4), [...hair.m, 255]);
  assert.deepEqual(at(5), [...hair.l, 255]);
  assert.deepEqual(at(6), [...iron, 255], 'iron is untouched');
  assert.deepEqual(at(7), [...sm, 0], 'transparent pixels are untouched');
  assert.deepEqual(at(8), [0x15, 0x12, 0x0f, 255], 'the outline is untouched');
});

test('a look with only skin leaves hair alone, and no look changes nothing', () => {
  const data = px([...T.BASE.hair[1], 255], [...T.BASE.skin[1], 255]);
  assert.equal(T.apply(data, T.look('#a8744a', '')), 1);
  assert.deepEqual(Array.from(data.slice(0, 3)), T.BASE.hair[1]);
  assert.equal(T.apply(data, null), 0);
});

test('canvasFor repaints once per image and look and caches the result', () => {
  let made = 0;
  const pixels = px([...T.BASE.skin[1], 255]);
  const make = (w, h) => {
    made++;
    return {
      getContext: () => ({
        drawImage() {}, putImageData() {}, getImageData: () => ({ data: pixels }),
        set imageSmoothingEnabled(v) {},
      }),
    };
  };
  const img = { width: 1, height: 1 };
  const cache = {};
  const look = T.look('#52311d', '');
  const a = T.canvasFor(img, look, make, cache, 'map/units/cleric/idle.png');
  const b = T.canvasFor(img, look, make, cache, 'map/units/cleric/idle.png');
  assert.equal(a, b);
  assert.equal(made, 1);
  assert.deepEqual(Array.from(pixels.slice(0, 3)), T.ramp('#52311d').m);
  assert.equal(T.canvasFor(img, null, make, cache, 'x'), img, 'no look: the image itself');
  const c = T.canvasFor(img, T.look('#ecd0b4', ''), make, cache, 'map/units/cleric/idle.png');
  assert.notEqual(c, a);
  assert.equal(made, 2);
});

test('canvasFor falls back to the stock image when the canvas cannot be read', () => {
  // A cross-origin sprite taints the canvas and getImageData throws.
  const make = () => ({
    getContext: () => ({
      drawImage() {}, putImageData() {},
      getImageData: () => { throw new Error('SecurityError'); },
      set imageSmoothingEnabled(v) {},
    }),
  });
  const img = { width: 1, height: 1 };
  const cache = {};
  assert.equal(T.canvasFor(img, T.look('#52311d', ''), make, cache, 'cdn'), img);
});
