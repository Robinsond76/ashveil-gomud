// E2: road and coast auto-tiling, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const M = require(path.join(here, '../../_datafiles/html/public/static/js/map-tiles.js'));

// look builds the neighbourhood of the room at the centre of rows (y grows
// south). links lists the offsets with an exit ('0,-1'), onward the
// offsets whose exit leaves the drawn map (fog, another zone).
function look(rows, links = [], onward = []) {
  return {
    envAt: (dx, dy) => (rows[1 + dy] || [])[1 + dx] || '',
    linked: (dx, dy) => links.includes(dx + ',' + dy),
    onward: (dx, dy) => onward.includes(dx + ',' + dy),
  };
}

test('sides lists edges in n, e, s, w order, or none', () => {
  assert.equal(M.sides(() => false), 'none');
  assert.equal(M.sides(() => true), 'nesw');
  assert.equal(M.sides((dx, dy) => dx === -1 || dy === -1), 'nw');
  assert.equal(M.sides((dx, dy) => dy === 1 || dx === 1), 'es');
});

test('a road joins road neighbours it has an exit to', () => {
  const rows = [
    ['', 'road', ''],
    ['road', 'road', 'road'],
    ['', 'road', ''],
  ];
  assert.equal(M.piece('road', look(rows, ['0,-1', '1,0', '0,1', '-1,0'])), 'road-nesw');
  assert.equal(M.piece('road', look(rows, ['1,0', '-1,0'])), 'road-ew', 'no exit, no join: a wall between two roads');
  assert.equal(M.piece('road', look(rows, ['0,1'])), 'road-s', 'a dead end');
  assert.equal(M.piece('road', look(rows, [])), 'road-none');
});

test('a road ignores other ground, even with an exit', () => {
  const rows = [
    ['', 'forest', ''],
    ['city', 'road', 'water'],
    ['', 'road', ''],
  ];
  assert.equal(M.piece('road', look(rows, ['0,-1', '1,0', '0,1', '-1,0'])), 'road-s');
});

test('a road runs on through an exit that leaves the map (fog, another zone)', () => {
  const rows = [['', '', ''], ['road', 'road', ''], ['', '', '']];
  assert.equal(M.piece('road', look(rows, ['-1,0'], ['1,0'])), 'road-ew');
});

test('a coast faces water neighbours, exits or not', () => {
  const rows = [
    ['', 'water', ''],
    ['land', 'shore', 'water'],
    ['', 'shore', ''],
  ];
  assert.equal(M.piece('shore', look(rows)), 'shore-ne');
  assert.equal(M.piece('shore', look([['', 'water', ''], ['water', 'shore', 'water'], ['', 'water', '']])), 'shore-nesw');
  assert.equal(M.piece('shore', look([['', '', ''], ['', 'shore', ''], ['', '', '']])), 'shore-none');
});

test('other ground has no piece', () => {
  assert.equal(M.piece('forest', look([['', 'road', ''], ['road', 'forest', 'road'], ['', '', '']], ['0,-1'])), '');
  assert.equal(M.piece('', look([])), '');
});

test('only road and shore are tiled', () => {
  assert.ok(M.tiled('road') && M.tiled('shore'));
  assert.ok(!M.tiled('water') && !M.tiled('forest') && !M.tiled(''));
});

// grid parses rows of single letters into rooms: s shore, l land, r road,
// . nothing.
const ENV = { s: 'shore', l: 'land', r: 'road' };
function grid(rows) {
  const cells = [];
  rows.forEach((row, y) => [...row].forEach((ch, x) => { if (ENV[ch]) { cells.push({ x, y, env: ENV[ch] }); } }));
  return cells;
}
const sorted = a => [...a].sort();

test('a ring of shore encloses a lake', () => {
  const cells = grid([
    '.sss.',
    's...s',
    's...s',
    '.sss.',
  ]);
  assert.deepEqual(sorted(M.lakes(cells, {})), sorted(['1,1', '2,1', '3,1', '1,2', '2,2', '3,2']));
});

test('a diagonal step in the ring still closes it', () => {
  const cells = grid([
    '..s..',
    '.s.s.',
    '..s..',
  ]);
  assert.deepEqual(M.lakes(cells, {}), ['2,1']);
});

test('an open ring, a non-shore border, or an exit into the middle is no lake', () => {
  assert.deepEqual(M.lakes(grid(['sss', 's..', 'sss']), {}), [], 'open to the east');
  assert.deepEqual(M.lakes(grid(['sss', 's.l', 'sss']), {}), [], 'land on its edge');
  assert.deepEqual(M.lakes(grid(['sss', 's.s', 'sss']), { '1,1': true }), [], 'an exit leads into it: an unseen room');
  assert.deepEqual(M.lakes(grid(['sss', 's.s', 'sss']), {}), ['1,1']);
  assert.deepEqual(M.lakes([], {}), []);
});

test('a coast faces a lake cell as water', () => {
  const lake = new Set(M.lakes(grid(['.s.', 's.s', '.s.']), {}));
  const at = (x, y) => (lake.has(x + ',' + y) ? 'water' : '');
  // The shore at (1,0) has the lake to its south.
  const piece = M.piece('shore', { envAt: (dx, dy) => at(1 + dx, 0 + dy), linked: () => false, onward: () => false });
  assert.equal(piece, 'shore-s');
});
