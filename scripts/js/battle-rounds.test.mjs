// BattleRounds keeps the explained rounds of the latest fight (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const BattleRounds = require(path.join(here, '../../_datafiles/html/public/static/js/battle-rounds.js'));

const names = { leader: 'Aria', 'm:5': 'bandit' };
const nameOf = (id) => names[id] || '';
const attack = (extra) => Object.assign({ kind: 'attack', src: 'leader', tgt: 'm:5', outcome: 'hit', damage: 5, explain: ['To hit: 60 in 100, rolled 12 (it landed).'] }, extra);

test('headings say who, whom and what happened', () => {
    assert.equal(BattleRounds.heading(attack(), nameOf), 'Aria → bandit: hit for 5');
    assert.equal(BattleRounds.heading(attack({ crit: true, damage: 9 }), nameOf), 'Aria → bandit: critical hit for 9');
    assert.equal(BattleRounds.heading(attack({ outcome: 'miss', damage: 0 }), nameOf), 'Aria → bandit: missed');
    assert.equal(BattleRounds.heading(attack({ defenses: ['dodged'], outcome: 'miss', damage: 0 }), nameOf), 'Aria → bandit: turned aside (dodged)');
    assert.equal(BattleRounds.heading(attack({ damage: 0 }), nameOf), 'Aria → bandit: hit, no damage');
    assert.equal(BattleRounds.heading(attack({ src: 'x' }), nameOf), 'someone → bandit: hit for 5');
    // Review: a round where one strike was dodged and another landed hit.
    assert.equal(BattleRounds.heading(attack({ defenses: ['dodged'], damage: 5 }), nameOf), 'Aria → bandit: hit for 5');
});

test('keeps only explained attacks, newest first, and stops at the cap', () => {
    const log = BattleRounds.create(2);
    const changed = log.add({ fight: 1, fight_round: 1, events: [attack({ damage: 1 }), { kind: 'heal', src: 'leader' }, attack({ explain: [] })] }, nameOf);
    assert.equal(changed, true);
    assert.equal(log.list().length, 1);
    log.add({ fight: 1, fight_round: 2, events: [attack({ damage: 2 }), attack({ damage: 3 })] }, nameOf);
    const rounds = log.list();
    assert.equal(rounds.length, 2);
    assert.match(rounds[0].head, /hit for 3$/);
    assert.match(rounds[1].head, /hit for 2$/);
    assert.equal(rounds[0].round, 2);
    assert.deepEqual(rounds[0].lines, ['To hit: 60 in 100, rolled 12 (it landed).']);
});

test('a new fight starts the list afresh and a message with nothing to explain changes nothing', () => {
    const log = BattleRounds.create();
    log.add({ fight: 1, round: 7, events: [attack()] }, nameOf);
    assert.equal(log.add({ fight: 1, events: [{ kind: 'heal' }] }, nameOf), false);
    log.add({ fight: 2, round: 9, events: [attack({ damage: 8 })] }, nameOf);
    assert.equal(log.list().length, 1);
    assert.match(log.list()[0].head, /hit for 8$/);
    assert.equal(log.add(null, nameOf), false);
    log.clear();
    assert.equal(log.list().length, 0);
});
