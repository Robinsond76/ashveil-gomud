// The quick command menu's pure half, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const QM = require(path.join(here, '../../_datafiles/html/public/static/js/quickmenu.js'));

const room = {
  exits: { north: 1, east: 2 },
  exitsv2: { east: { details: ['locked'] } },
  resources: ['herbs', 'fishing', 'water'],
  depleted: ['herbs'],
  details: ['trainer'],
  Contents: {
    Npcs: [{ id: 11, name: 'Old Fisher', adjectives: ['shop'] }, { id: 12, name: 'Rat' }],
    Items: [{ id: 'a:1', name: 'dagger', label: 'a dagger' }],
    Containers: [{ name: 'chest' }],
  },
};
const top = (s) => QM.build(s).map(e => e.label);
const find = (entries, label) => entries.find(e => e.label === label);

test('the first level follows the room', () => {
  const labels = top({ room });
  for (const want of ['Look', 'Attack', 'Move', 'Services', 'Get', 'Gather', 'Company', 'Me', 'Help']) {
    assert.ok(labels.includes(want), want);
  }
  assert.ok(!labels.includes('Battle'));
});

test('an empty room offers no attack, no gather, no exits', () => {
  const labels = top({ room: {} });
  for (const gone of ['Attack', 'Move', 'Services', 'Gather', 'Battle']) { assert.ok(!labels.includes(gone), gone); }
  assert.ok(labels.includes('Look'));
});

test('targets are named by id', () => {
  const attack = find(QM.build({ room }), 'Attack').sub();
  assert.deepEqual(attack.map(e => e.cmd), ['attack 11', 'attack 12']);
  const get = find(QM.build({ room }), 'Get').sub();
  assert.ok(get.some(e => e.cmd === 'get all'));
  assert.ok(get.some(e => e.cmd === 'get a:1'));
  const look = find(QM.build({ room }), 'Look').sub();
  assert.equal(look[0].cmd, 'look');
  assert.ok(look.some(e => e.cmd === 'look 12'));
  assert.ok(look.some(e => e.cmd === 'look north'));
});

test('exits show locked, and a walk list appears with places', () => {
  const move = find(QM.build({ room, places: [{ id: 5, name: 'Inn', legend: 'inn' }], walking: true }), 'Move').sub();
  assert.equal(find(move, 'east').hint, 'locked');
  assert.ok(move.some(e => e.cmd === 'walkto stop'));
  assert.deepEqual(find(move, 'Walk to').sub().map(e => e.cmd), ['walkto 5']);
});

test('gathering leaves out what is picked clean', () => {
  const cmds = find(QM.build({ room }), 'Gather').sub().map(e => e.cmd);
  assert.deepEqual(cmds, ['fish', 'drink water']);
});

test('services list shops and the trainer', () => {
  const cmds = find(QM.build({ room }), 'Services').sub().map(e => e.cmd);
  assert.deepEqual(cmds, ['list 11', 'train']);
});

test('a battle puts its orders first and hides attack and gathering', () => {
  const s = { room, battle: { enemies: [{}], focus_ready: true } };
  const labels = top(s);
  assert.equal(labels[0], 'Battle');
  assert.ok(!labels.includes('Attack') && !labels.includes('Gather'));
  const cmds = QM.build(s)[0].sub().map(e => e.cmd);
  assert.ok(cmds.includes('retreat'));
  assert.ok(cmds.includes('company tactics focus weakest'));
  const waiting = QM.build({ room, battle: { enemies: [{}], focus_ready: false } })[0].sub().map(e => e.cmd);
  assert.deepEqual(waiting, ['retreat']);
});

test('camp commands appear only when they would work', () => {
  const camp = (c) => find(find(QM.build({ room, camp: c }), 'Company').sub(), 'Camp');
  assert.equal(camp({}), undefined);
  assert.deepEqual(camp({ can_camp: true }).sub().map(e => e.cmd), ['camp']);
  const cold = camp({ has_camp: true, here: true }).sub().map(e => e.cmd);
  assert.deepEqual(cold, ['camp fire', 'camp break']);
  const lit = camp({ has_camp: true, here: true, fire_lit: true }).sub().map(e => e.cmd);
  assert.deepEqual(lit, ['camp rest', 'camp break']);
});

test('every entry sends, runs or opens something', () => {
  const walk = (entries) => entries.forEach(e => {
    assert.ok(e.cmd || e.fn || e.sub, e.label);
    if (e.sub) { walk(e.sub()); }
  });
  walk(QM.build({ room, camp: { can_camp: true, inn: true }, battle: { enemies: [{}] }, places: [{ id: 1, name: 'x' }] }));
});
