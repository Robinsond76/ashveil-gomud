// Phase 40g: the battle timeline planner, under Node (make js-test).
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const TL = require(path.join(here, '../../_datafiles/html/public/static/js/battle-timeline.js'));

const plan = (events, opts) => TL.plan(events, opts);
const first = (events, opts) => plan(events, opts)[0].steps;
const stepFor = (steps, unit) => steps.find(s => s.unit === unit);

test('a melee hit: the attacker lunges and strikes, the target flinches with the weapon effect and digits', () => {
  const steps = first([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'hit', damage: 6, weapon: 'slashing' }]);
  const a = stepFor(steps, 'leader');
  const t = stepFor(steps, 'm:1');
  assert.equal(a.anim, 'attack');
  assert.equal(a.lunge, 'm:1');
  assert.equal(t.anim, 'hurt');
  assert.equal(t.delay, a.strikeAt, 'the blow lands at the strike');
  assert.deepEqual(t.fx.map(f => f.id), ['slash']);
  assert.equal(t.digits[0].text, '6');
});

test('weapon types pick their hit effect; no weapon is blunt for the company and a claw for creatures', () => {
  const eff = (weapon, src) => first([{ seq: 1, kind: 'attack', src, tgt: 'x', outcome: 'hit', damage: 1, weapon }]).find(s => s.unit === 'x').fx[0].id;
  assert.equal(eff('stabbing', 'leader'), 'stab');
  assert.equal(eff('bludgeoning', 'leader'), 'blunt');
  assert.equal(eff('cleaving', 'leader'), 'cleave');
  assert.equal(eff('claws', 'm:1'), 'claw');
  assert.equal(eff('', 'm:1'), 'claw');
  assert.equal(eff('', 'leader'), 'blunt');
});

test('a critical hit adds the crit burst and large digits', () => {
  const t = stepFor(first([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'crit', crit: true, damage: 12, weapon: 'slashing' }]), 'm:1');
  assert.equal(t.fx[0].crit, true);
  assert.equal(t.digits[0].big, true);
  assert.equal(t.digits[0].text, '12');
});

test('a miss gives the miss feedback; defenses give block, parry or dodge with their icon', () => {
  const miss = stepFor(first([{ seq: 1, kind: 'attack', src: 'm:1', tgt: 'leader', outcome: 'miss' }]), 'leader');
  assert.equal(miss.icon, 'miss');
  for (const [defense, anim] of [['blocked', 'block'], ['parried', 'parry'], ['dodged', 'dodge']]) {
    const t = stepFor(first([{ seq: 1, kind: 'attack', src: 'm:1', tgt: 'leader', outcome: 'miss', defenses: [defense] }]), 'leader');
    assert.equal(t.anim, anim, defense);
    assert.equal(t.icon, defense);
  }
});

test('a missing animation uses its fallback, else a nudge', () => {
  const ev = [{ seq: 1, kind: 'attack', src: 'm:1', tgt: 'leader', outcome: 'miss', defenses: ['parried'] }];
  assert.equal(stepFor(first(ev, { has: (u, a) => a === 'hurt' }), 'leader').anim, 'hurt', 'parry falls back to hurt');
  const none = stepFor(first(ev, { has: () => false }), 'leader');
  assert.equal(none.anim, '');
  assert.equal(none.nudge, true);
  assert.equal(stepFor(first(ev, { has: () => true }), 'leader').anim, 'parry');
  assert.equal(stepFor(first([{ seq: 1, kind: 'attack', src: 'm:2', tgt: 'leader', outcome: 'hit', damage: 1 }], { has: () => false }), 'm:2').nudge, true, 'attack without art nudges');
});

test('ranged weapons shoot and loose a projectile that lands the blow later', () => {
  const steps = first([{ seq: 1, kind: 'attack', src: 'companion:1', tgt: 'm:1', outcome: 'hit', damage: 4, weapon: 'shooting' }]);
  const a = stepFor(steps, 'companion:1');
  assert.equal(a.anim, 'shoot');
  assert.equal(a.lunge, undefined);
  assert.equal(a.fx[0].kind, 'projectile');
  const t = stepFor(steps, 'm:1');
  assert.ok(t.delay >= a.fx[0].at + a.fx[0].dur, 'the hit waits for the arrow');
  assert.equal(t.fx[0].id, 'arrow-hit');
});

test('casting: chant loops, release bursts, interrupted breaks, fizzled fades', () => {
  const start = stepFor(first([{ seq: 1, kind: 'cast-start', src: 'companion:3', spell: 'fire bolt' }]), 'companion:3');
  assert.equal(start.anim, 'cast');
  assert.equal(start.loop, true);
  assert.deepEqual(start.ops.map(o => o.op), ['casting']);
  assert.equal(start.ops[0].spell, 'fire bolt');
  const done = ['cast', 'interrupted', 'fizzled', 'wasted'].map(outcome => first([{ seq: 1, kind: 'cast-complete', src: 'companion:3', outcome }])[0]);
  assert.equal(done[0].fx[0].burst, true);
  assert.equal(done[1].fx[0].id, 'chant-broken');
  assert.equal(done[2].fx[0].kind, 'fade-glow');
  assert.equal(done[3].fx[0].kind, 'fade-glow');
  done.forEach(d => assert.equal(d.ops[0].spell, '', 'every end clears the chant mark'));
});

test('a spell hit travels to its target as a bolt; a heal shows green digits and a dim held-back number', () => {
  const steps = first([{ seq: 1, kind: 'spell-hit', src: 'companion:3', tgt: 'm:1', spell: 'fire bolt', damage: 8 }]);
  const t = stepFor(steps, 'm:1');
  assert.equal(t.fx[0].kind, 'projectile');
  assert.equal(t.fx[1].id, 'magic-hit');
  assert.equal(t.fx[1].at, t.delay);
  const heal = stepFor(first([{ seq: 2, kind: 'heal', src: 'companion:1', tgt: 'leader', amount: 4, held_back: 2 }]), 'leader');
  assert.equal(heal.digits[0].text, '+4');
  assert.equal(heal.digits[0].color, '#5fd08a');
  assert.equal(heal.digits[1].text, '(+2)');
  assert.equal(heal.digits[1].dim, true);
});

test('wind-ups hold with a status and release on the blow; wasted lands on nothing', () => {
  const w = first([{ seq: 1, kind: 'windup-start', src: 'm:4' }])[0];
  assert.equal(w.anim, 'windup');
  assert.deepEqual(w.ops, [{ op: 'status+', unit: 'm:4', when: 'start', status: 'winding-up' }]);
  const land = first([{ seq: 2, kind: 'windup-land', src: 'm:4', tgt: 'leader', outcome: 'landed' }])[0];
  assert.equal(land.anim, 'attack');
  assert.equal(land.lunge, 'leader');
  assert.equal(land.ops[0].op, 'status-');
  assert.equal(first([{ seq: 3, kind: 'windup-land', src: 'm:4', tgt: 'leader', outcome: 'wasted' }])[0].lunge, undefined, 'a wasted swing goes nowhere');
});

test('statuses add and remove their icon; armor broken shatters; a tick shows digits or a skip', () => {
  assert.deepEqual(first([{ seq: 1, kind: 'status-applied', tgt: 'm:1', status: 'bleeding' }])[0].ops.map(o => o.op), ['status+']);
  assert.deepEqual(first([{ seq: 1, kind: 'status-expired', tgt: 'm:1', status: 'bleeding' }])[0].ops.map(o => o.op), ['status-']);
  assert.equal(first([{ seq: 1, kind: 'status-applied', tgt: 'm:1', status: 'armor broken' }])[0].fx[0].kind, 'armor-shatter');
  assert.equal(first([{ seq: 1, kind: 'status-tick', tgt: 'm:1', status: 'bleeding', damage: 2 }])[0].digits[0].text, '2');
  assert.equal(first([{ seq: 1, kind: 'status-tick', tgt: 'm:1', status: 'stunned', outcome: 'lost-action' }])[0].icon, 'skip');
});

test('guard, abilities, yield, flee, death, mercy and the end of the fight', () => {
  const guard = first([{ seq: 1, kind: 'guard-used', src: 'companion:2', tgt: 'companion:1' }])[0];
  assert.equal(guard.anim, 'guard-step');
  assert.equal(guard.fx[0].id, 'guard-intercept');
  const tackle = first([{ seq: 2, kind: 'ability', src: 'leader', tgt: 'm:1', status: 'Tackle', outcome: 'succeeded' }]);
  assert.equal(stepFor(tackle, 'm:1').anim, 'prone');
  assert.equal(stepFor(first([{ seq: 2, kind: 'ability', src: 'leader', tgt: 'm:1', status: 'Tackle', outcome: 'failed' }]), 'm:1'), undefined);
  assert.equal(first([{ seq: 3, kind: 'ability', src: 'leader', tgt: 'm:1', status: 'Opening Strike' }])[0].fx[0].kind, 'highlight');
  // Phase 61: a battle order carried out gives its member a brief highlight.
  assert.equal(first([{ seq: 7, kind: 'order-fired', src: 'companion:1', tgt: 'leader', status: 'heal' }])[0].fx[0].kind, 'highlight');
  assert.equal(first([{ seq: 4, kind: 'yield', src: 'm:2' }])[0].ops[0].op, 'yielded');
  const flee = first([{ seq: 5, kind: 'flee', src: 'm:2' }])[0];
  assert.equal(flee.exit, true);
  assert.equal(flee.ops[0].op, 'remove');
  assert.equal(flee.ops[0].when, 'end', 'a fleeing unit leaves when its run ends');
  const death = first([{ seq: 6, kind: 'death', tgt: 'm:3' }])[0];
  assert.equal(death.anim, 'down');
  assert.equal(death.ops[0].when, 'end', 'a unit is fallen once its collapse has played');
  assert.equal(first([{ seq: 7, kind: 'mercy', src: 'leader', tgt: 'm:3', outcome: 'spared' }])[0].fade, true);
  const executed = first([{ seq: 8, kind: 'mercy', src: 'leader', tgt: 'm:3', outcome: 'executed' }]);
  assert.equal(executed.length, 2);
  assert.equal(stepFor(executed, 'm:3').anim, 'down');
  assert.equal(first([{ seq: 9, kind: 'fight-end', outcome: 'victory' }])[0].unit, '*company*');
  assert.equal(first([{ seq: 10, kind: 'fight-end', outcome: 'defeat' }])[0].fadeScreen, true);
});

test('pace budgets: fast halves, slow lengthens, off is effects only', () => {
  const dur = pace => first([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'hit', damage: 1 }], { pace })[0].dur;
  assert.equal(dur('normal'), 1200);
  assert.equal(dur('fast'), 600);
  assert.ok(dur('slow') > 1200);
  assert.equal(dur('off'), 300);
  assert.equal(TL.budgetFor('normal').reaction, 600);
  assert.equal(TL.budgetFor('bogus').action, 1200, 'an unknown pace is normal');
});

test('reduced motion: no lunge, no projectile travel, no tint', () => {
  const steps = first([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'hit', damage: 6, weapon: 'slashing' }], { motion: 'reduced' });
  assert.equal(stepFor(steps, 'leader').lunge, undefined);
  assert.equal(stepFor(steps, 'm:1').tint, '');
  const ranged = first([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'hit', damage: 6, weapon: 'shooting' }], { motion: 'reduced' });
  assert.equal(stepFor(ranged, 'leader').fx, undefined);
  const bolt = stepFor(first([{ seq: 1, kind: 'spell-hit', src: 'companion:3', tgt: 'm:1', spell: 'fire bolt', damage: 8 }], { motion: 'reduced' }), 'm:1');
  assert.ok(bolt.fx.every(f => f.kind !== 'projectile'));
});

test('events that name no one, or an unknown kind, plan nothing and never throw', () => {
  assert.deepEqual(plan([{ seq: 1, kind: 'fight-start' }, { seq: 2, kind: 'target-change', src: 'leader' }, { seq: 3, kind: 'death' }, { seq: 4, kind: 'heal' }]), []);
});

test('the scheduler queues per unit and overlaps different units', () => {
  const s = new TL.Scheduler();
  const hit = (src, tgt, seq) => ({ seq, kind: 'attack', src, tgt, outcome: 'hit', damage: 1 });
  s.push(plan([hit('leader', 'm:1', 1), hit('leader', 'm:2', 2), hit('companion:2', 'm:3', 3)]), 1000);
  const placed = s.pending.filter(p => p.role === 'action');
  assert.equal(placed[0].start, 1000);
  assert.equal(placed[1].start, placed[0].end, 'the same unit waits for its previous action');
  assert.equal(placed[2].start, 1000, 'a different unit overlaps');
});

test('a happening starts its reaction at the blow, in time order, and fires ops at their times', () => {
  const s = new TL.Scheduler();
  s.push(plan([{ seq: 1, kind: 'death', tgt: 'm:3' }, { seq: 2, kind: 'status-applied', tgt: 'm:1', status: 'bleeding' }]), 0);
  assert.deepEqual(s.due(10).map(o => o.op), ['status+']);
  assert.deepEqual(s.due(1000).map(o => o.op), []);
  assert.deepEqual(s.due(1300).map(o => o.op), ['fallen']);
  assert.equal(s.due(1400).length, 0, 'ops fire once');
});

test('digits, icons and tints arrive as a react op when the step starts', () => {
  const s = new TL.Scheduler();
  s.push(plan([{ seq: 1, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'crit', crit: true, damage: 12, weapon: 'slashing' }]), 0);
  assert.equal(s.due(100).length, 0);
  const ops = s.due(700);
  assert.equal(ops[0].op, 'react');
  assert.equal(ops[0].unit, 'm:1');
  assert.equal(ops[0].digits[0].text, '12');
  assert.equal(ops[0].bigHit, true);
  assert.equal(s.activeFx(700).length > 0, true, 'the hit effect is showing');
});

test('never fall behind: over a round of backlog collapses to the end state', () => {
  const s = new TL.Scheduler();
  const flurry = [];
  for (let i = 0; i < 12; i++) { flurry.push({ seq: i, kind: 'attack', src: 'leader', tgt: 'm:1', outcome: 'hit', damage: 1 }); }
  flurry.push({ seq: 20, kind: 'death', tgt: 'm:1' }, { seq: 21, kind: 'status-applied', tgt: 'leader', status: 'cursed' });
  assert.deepEqual(s.push(plan(flurry), 0), [], 'nothing is collapsed on the first push');
  assert.ok(s.backlog(0) > TL.ROUND_MS);
  const collapsed = s.push(plan([{ seq: 30, kind: 'yield', src: 'm:2' }]), 100);
  assert.deepEqual(collapsed.map(o => o.op + ':' + o.unit), ['fallen:m:1', 'status+:leader'], 'the older end states apply at once, in order');
  assert.ok(s.backlog(100) < 2000, 'the new round plays from now');
  assert.equal(s.pending.length, 1);
});

test('a collapse leaves ops already fired alone', () => {
  const s = new TL.Scheduler();
  s.push(plan([{ seq: 1, kind: 'status-applied', tgt: 'm:1', status: 'bleeding' }]), 0);
  s.due(10);
  assert.deepEqual(s.collapse(), []);
});

test('the pace is inferred from how batches arrive', () => {
  const at = (...gaps) => { let t = 0; return [{ at: 0, n: 2 }].concat(gaps.map(g => ({ at: (t += g), n: 2 }))); };
  assert.equal(TL.inferPace([]), 'normal');
  assert.equal(TL.inferPace(at(500, 400, 600)), 'fast');
  assert.equal(TL.inferPace(at(1500, 1400, 1800)), 'normal');
  assert.equal(TL.inferPace(at(3000, 3200, 2900)), 'slow');
  assert.equal(TL.inferPace([{ at: 0, n: 9 }, { at: 8000, n: 11 }]), 'off', 'whole rounds in a batch mean pacing is off');
});

test('the fight\'s end waits for every unit: the victory comes after the last blow', () => {
  const sc = new TL.Scheduler();
  sc.push(plan([{ seq: 1, kind: 'death', tgt: 'm:1' }]), 0);
  sc.push(plan([{ seq: 2, kind: 'fight-end', outcome: 'victory' }]), 100);
  const end = sc.pending.find(s => s.unit === '*company*');
  const fall = sc.pending.find(s => s.unit === 'm:1');
  assert.ok(end.start >= fall.end, 'victory at ' + end.start + ', fall ends at ' + fall.end);
});

test('an unseen caster\'s spell is never named nor coloured; a seen one uses the spell\'s name', () => {
  const hidden = first([{ seq: 1, kind: 'cast-start', src: '?', spell: 'mm', spell_name: 'Magic Missile' }])[0];
  assert.equal(hidden.ops[0].spell, 'a spell');
  assert.equal(hidden.fx[0].spell, '');
  const bolt = stepFor(first([{ seq: 2, kind: 'spell-hit', src: '?', tgt: 'leader', spell: 'mm', damage: 5 }]), 'leader');
  assert.ok(bolt.fx.every(f => f.spell === ''));
  const seen = first([{ seq: 3, kind: 'cast-start', src: 'companion:3', spell: 'mm', spell_name: 'Magic Missile' }])[0];
  assert.equal(seen.ops[0].spell, 'Magic Missile');
});

test('an allied company\'s unit counts as company for the hit effect, and allied refs plan like any other', () => {
  const hit = first([{ seq: 1, kind: 'attack', src: 'a:8:leader', tgt: 'm:1', outcome: 'hit', damage: 3 }]);
  assert.equal(hit.find(s => s.unit === 'm:1').fx[0].id, 'blunt', 'a bare-handed ally strikes blunt, not with claws');
  const down = first([{ seq: 2, kind: 'death', src: 'm:1', tgt: 'a:8:companion:2' }]);
  assert.ok(down.some(s => s.unit === 'a:8:companion:2'), 'an allied unit falls like any other');
});
