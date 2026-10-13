// The painted-orb map's layout helpers, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const O = require(path.join(here, '../../_datafiles/html/public/static/js/map-orbs.js'));

const PI = Math.PI;
const near = (a, b) => Math.abs(a - b) < 1e-9;

test('angleOf reads grid offsets with y growing south', () => {
  assert.ok(near(O.angleOf(1, 0), 0));
  assert.ok(near(O.angleOf(0, 1), PI / 2));
  assert.ok(near(O.angleOf(-1, 0), PI));
  assert.ok(near(O.angleOf(0, -1), -PI / 2));
});

test('gap is the short way round', () => {
  assert.ok(near(O.gap(0, PI / 2), PI / 2));
  assert.ok(near(O.gap(-PI + 0.1, PI - 0.1), 0.2));
  assert.ok(near(O.gap(0, 2 * PI), 0));
});

test('icons start at the bottom and spread both ways', () => {
  const s = O.ringSlots(3, [], 0.5);
  assert.equal(s.length, 3);
  assert.ok(near(s[0], PI / 2));
  assert.ok(near(s[1], PI / 2 + 0.5) && near(s[2], PI / 2 - 0.5));
  assert.deepEqual(O.ringSlots(0, [], 0.5), []);
});

test('icons keep clear of exit lines', () => {
  // a room with exits on all four sides
  const blocked = [0, PI / 2, PI, -PI / 2].map(a => ({ a, half: 0.15 }));
  const step = 0.6;
  const s = O.ringSlots(4, blocked, step);
  assert.equal(s.length, 4);
  for (const a of s) {
    for (const b of blocked) { assert.ok(O.gap(a, b.a) >= b.half + step / 2 - 1e-9, `slot ${a} sits on the line at ${b.a}`); }
  }
  for (let i = 0; i < s.length; i++) {
    for (let j = i + 1; j < s.length; j++) { assert.ok(O.gap(s[i], s[j]) >= step - 1e-9, 'two icons overlap'); }
  }
});

test('a crowded ring still places every icon', () => {
  const blocked = [];
  for (let k = 0; k < 8; k++) { blocked.push({ a: k * PI / 4, half: 0.3 }); }
  const s = O.ringSlots(7, blocked, 0.6);
  assert.equal(s.length, 7);
  assert.equal(new Set(s.map(a => a.toFixed(6))).size, 7, 'no two icons share a slot');
});

test('title rules match room names in order, ignoring case', () => {
  const rules = [['castle gates', 'gate'], ['castle', 'keep'], ['library', 'alts']];
  assert.equal(O.titleMatch(rules, 'The Castle Gates'), 'gate');
  assert.equal(O.titleMatch(rules, 'Frostfang Castle East Wing'), 'keep');
  assert.equal(O.titleMatch(rules, 'Frostfang Library'), 'alts');
  assert.equal(O.titleMatch(rules, 'Cobblestone Way'), '');
  assert.equal(O.titleMatch(null, 'Castle'), '');
  assert.equal(O.titleMatch(rules, undefined), '');
});

test('a road is painted over the ground it passes through', () => {
  assert.equal(O.roadGround(['forest', 'road', 'forest', 'land']), 'forest');
  assert.equal(O.roadGround(['road', 'water', 'water', 'shore']), 'shore');
  assert.equal(O.roadGround(['road', 'road']), 'land');
  assert.equal(O.roadGround([]), 'land');
});

test('seeded draws repeat for a seed and differ between seeds', () => {
  const a = O.seeded(2101), b = O.seeded(2101), c = O.seeded(2102);
  const xs = [a(), a(), a()], ys = [b(), b(), b()], zs = [c(), c(), c()];
  assert.deepEqual(xs, ys);
  assert.notDeepEqual(xs, zs);
  for (const x of xs.concat(zs)) { assert.ok(x >= 0 && x < 1); }
  assert.ok(O.seeded(0)() >= 0, 'a zero seed still works');
});
