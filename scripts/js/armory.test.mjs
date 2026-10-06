// The armory catalog's pure half, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const { filterCatalog } = require(path.join(here, '../../_datafiles/html/public/static/js/windows/window-armory.js'));

const items = [
  { id: 1, name: 'Iron Sword', type: 'weapon', subtype: 'sword', tier: 1 },
  { id: 2, name: 'Oak Staff', type: 'weapon', subtype: 'staff', family: 'quarterstaff', tier: 2 },
  { id: 3, name: 'Leather Cap', type: 'head', family: 'leather', tier: 1 },
];

test('no type and no words keeps everything', () => {
  assert.equal(filterCatalog(items, '', '').length, 3);
});
test('a type chip narrows to that type', () => {
  assert.deepEqual(filterCatalog(items, 'weapon', '').map(i => i.id), [1, 2]);
});
test('every word must match name, type, subtype or family', () => {
  assert.deepEqual(filterCatalog(items, '', 'staff weapon').map(i => i.id), [2]);
  assert.deepEqual(filterCatalog(items, '', 'LEATHER').map(i => i.id), [3]);
  assert.equal(filterCatalog(items, '', 'sword leather').length, 0);
});
test('type and words combine', () => {
  assert.equal(filterCatalog(items, 'head', 'sword').length, 0);
});
