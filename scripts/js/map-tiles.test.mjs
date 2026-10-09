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
// south). links lists the offsets with an exit ('0,-1'), fog the offsets
// whose exit leads into fog.
function look(rows, links = [], fog = []) {
  return {
    envAt: (dx, dy) => (rows[1 + dy] || [])[1 + dx] || '',
    linked: (dx, dy) => links.includes(dx + ',' + dy),
    fogAt: (dx, dy) => fog.includes(dx + ',' + dy),
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

test('a road runs on into the fog of an unvisited exit', () => {
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
