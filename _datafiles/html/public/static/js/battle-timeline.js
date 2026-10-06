/**
 * battle-timeline.js
 *
 * The battle screen's animation planner (Phase 40g). It is pure: it turns a
 * batch of Company.Battle.Event entries (Phase 40e) into scheduled steps, and
 * knows nothing of the canvas, the DOM or the clock, so Node can test it
 * (scripts/js/battle-timeline.test.mjs, `make js-test`). window-battle.js
 * plays the steps.
 *
 * Nothing here is sent to the server, and nothing changes or delays the
 * battle: the events have already happened, and this only decides how the
 * screen shows them.
 *
 * Vocabulary
 *   happening   one event and the steps it makes
 *   step        { unit, anim, ... } one thing one unit (or the screen) does
 *               for a span of time: a pose, a lunge, an effect, a digit
 *   op          a change of the picture's state (a status added, a unit
 *               fallen); an op belongs to a step and fires when the step
 *               starts (when: 'start') or ends (when: 'end')
 *
 * plan(events, opts) lists the happenings of a batch with their steps; a
 * Scheduler places them in time (queueing per unit) and, when it has fallen
 * a round behind, collapses the older work to its end state.
 */

/* globals module */

(function(root, factory) {
    'use strict';
    const api = factory();
    if (typeof module === 'object' && module.exports) { module.exports = api; }
    else { root.BattleTimeline = api; }
}(typeof window !== 'undefined' ? window : globalThis, function() {
    'use strict';

    // A combat round is 8 seconds (help combatpace).
    const ROUND_MS = 8000;

    // Animation budgets by the player's combat pace: an action (an attack, a
    // cast) and a reaction (a flinch, a pop of digits). Slow is the normal
    // budget lengthened by half; "off" has no pacing, so effects only.
    const BUDGETS = {
        fast: { action: 600, reaction: 300 },
        normal: { action: 1200, reaction: 600 },
        slow: { action: 1800, reaction: 900 },
        off: { action: 300, reaction: 300 },
    };

    // Hit effect by the weapon's subtype (Event.weapon). A creature without a
    // weapon rakes or bites; a bare-handed company member strikes blunt.
    const HIT_BY_WEAPON = {
        slashing: 'slash', stabbing: 'stab', bludgeoning: 'blunt', cleaving: 'cleave', claws: 'claw',
        shooting: 'arrow-hit', throwable: 'arrow-hit', whipping: 'slash',
    };
    const RANGED = { shooting: true, throwable: true };

    const DEFENSE_ANIM = { blocked: 'block', parried: 'parry', dodged: 'dodge' };

    const COLORS = { hurt: '#e04b3a', crit: '#ffd23f', magic: '#7aa8ff', heal: '#5fd08a', tick: '#c06a3a', miss: '#9aa0aa', held: '#8fb89a' };

    // How long a hit, glow or other effect shows when it names no span of its own.
    const FX_MS = 360;

    function budgetFor(pace) { return BUDGETS[pace] || BUDGETS.normal; }

    // isCompany is true for refs that name the player's own company.
    function isCompany(ref) {
        return ref === 'me' || ref === 'leader' || /^companion:/.test(ref || '');
    }

    function hitEffect(weapon, srcRef) {
        if (HIT_BY_WEAPON[weapon]) { return HIT_BY_WEAPON[weapon]; }
        return isCompany(srcRef) ? 'blunt' : 'claw';
    }

    // resolveAnim picks the first animation a unit has, from a preferred one
    // down its fallbacks (the S4 sheets are drawn class by class, so a unit
    // often lacks some). When none exists the step is a nudge: the idle
    // figure moves a little, and the effect still shows. has(unit, anim)
    // answers whether the art exists; without it every animation is assumed.
    function resolveAnim(unit, anims, has) {
        if (typeof has !== 'function') { return { anim: anims[0], nudge: false }; }
        for (let i = 0; i < anims.length; i++) {
            if (has(unit, anims[i])) { return { anim: anims[i], nudge: false }; }
        }
        return { anim: '', nudge: true };
    }

    // Fallback chains (design): attack to idle plus a nudge, cast to idle
    // plus a glow, a defense to hurt, a missing down to a fade.
    const FALLBACKS = {
        attack: ['attack'],
        shoot: ['shoot', 'attack'],
        cast: ['cast'],
        'cast-release': ['cast'],
        block: ['block', 'hurt'],
        parry: ['parry', 'hurt'],
        dodge: ['dodge', 'hurt'],
        hurt: ['hurt'],
        down: ['down'],
        windup: ['windup'],
        'shield-bash': ['hurt'],
        'guard-step': ['guard-step', 'walk'],
        prone: ['prone', 'down'],
        yield: ['yield'],
        victory: ['victory'],
        walk: ['walk'],
    };

    // plan turns events into happenings: [{ seq, kind, steps: [...] }].
    // opts: pace, motion ('full' or 'reduced'), has(unit, anim).
    function plan(events, opts) {
        opts = opts || {};
        const b = budgetFor(opts.pace);
        const reduced = opts.motion === 'reduced';
        const out = [];

        // mk builds a step. Durations in ms; delay is from the happening's start.
        function mk(unit, anims, props) {
            const chain = FALLBACKS[anims] || [anims];
            const r = anims ? resolveAnim(unit, chain, opts.has) : { anim: '', nudge: false };
            return Object.assign({ unit: unit, anim: r.anim, nudge: r.nudge, delay: 0, dur: b.reaction, ops: [] }, props);
        }
        const act = (unit, anim, props) => mk(unit, anim, Object.assign({ role: 'action', dur: b.action }, props));
        const react = (unit, anim, props) => mk(unit, anim, Object.assign({ role: 'reaction', dur: b.reaction }, props));
        const op = (name, unit, extra, when) => Object.assign({ op: name, unit: unit, when: when || 'start' }, extra || {});

        function digit(step, text, color, extra) {
            step.digits = (step.digits || []).concat([Object.assign({ text: String(text), color: color }, extra || {})]);
        }

        events.forEach(e => {
            const steps = [];
            const src = e.src || '';
            const tgt = e.tgt || '';
            const strikeAt = Math.round(b.action * 0.5);
            const ranged = !!RANGED[e.weapon];

            switch (e.kind) {
            case 'attack': {
                const a = act(src, ranged ? 'shoot' : 'attack', { strikeAt: strikeAt });
                if (!reduced && !ranged && tgt && src !== tgt) { a.lunge = tgt; }
                steps.push(a);
                let landAt = strikeAt;
                if (ranged && !reduced) {
                    landAt = strikeAt + Math.round(b.action * 0.25);
                    a.fx = [{ kind: 'projectile', style: 'arrow', from: src, to: tgt, at: strikeAt, dur: landAt - strikeAt }];
                }
                if (!tgt) { break; }
                const defended = (e.defenses || []).length > 0;
                const missed = e.outcome === 'miss';
                if (missed || (defended && !e.damage)) {
                    // The swing found nothing: the defense it met, or the miss.
                    const d = defended ? e.defenses[0] : '';
                    const r = react(tgt, DEFENSE_ANIM[d] || 'dodge', { delay: landAt });
                    r.icon = defended ? d : 'miss';
                    if (!defended) { digit(r, 'miss', COLORS.miss, { quiet: true }); }
                    steps.push(r);
                } else {
                    const r = react(tgt, 'hurt', { delay: landAt, tint: reduced ? '' : (e.crit || e.outcome === 'crit' ? COLORS.crit : COLORS.hurt) });
                    r.fx = [{ kind: 'hit', id: hitEffect(e.weapon, src), at: landAt, crit: !!(e.crit || e.outcome === 'crit') }];
                    if (defended) { r.icon = e.defenses[0]; }
                    if (e.damage > 0) { digit(r, e.damage, e.crit ? COLORS.crit : COLORS.hurt, { big: !!e.crit }); }
                    steps.push(r);
                }
                break;
            }
            case 'cast-start':
            case 'cast-progress': {
                const c = act(src, 'cast', { loop: true, dur: e.kind === 'cast-start' ? b.action : b.reaction });
                c.fx = [{ kind: 'glow', spell: e.spell || '', at: 0 }];
                if (e.kind === 'cast-start') { c.ops.push(op('casting', src, { spell: e.src === '?' ? 'a spell' : (e.spell || 'a spell') })); }
                steps.push(c);
                break;
            }
            case 'cast-complete': {
                const done = op('casting', src, { spell: '' }, 'start');
                if (e.outcome === 'interrupted') {
                    const c = react(src, '', { fx: [{ kind: 'hit', id: 'chant-broken', at: 0 }] });
                    c.ops.push(done);
                    steps.push(c);
                } else if (e.outcome === 'cast') {
                    const c = act(src, 'cast-release', { fx: [{ kind: 'glow', spell: e.spell || '', at: 0, burst: true }] });
                    c.ops.push(done);
                    steps.push(c);
                } else {
                    // Fizzled, held or wasted: the glow fades.
                    const c = react(src, '', { fx: [{ kind: 'fade-glow', at: 0 }] });
                    c.ops.push(done);
                    steps.push(c);
                }
                break;
            }
            case 'spell-hit': {
                if (!tgt) { break; }
                const travel = src && !reduced ? Math.round(b.reaction * 0.5) : 0;
                const r = react(tgt, 'hurt', { delay: travel, tint: reduced ? '' : COLORS.magic });
                r.fx = [{ kind: 'hit', id: 'magic-hit', at: travel, spell: e.spell || '' }];
                if (travel) { r.fx.unshift({ kind: 'projectile', style: 'bolt', from: src, to: tgt, at: 0, dur: travel, spell: e.spell || '' }); }
                if (e.damage > 0) { digit(r, e.damage, COLORS.magic); }
                steps.push(r);
                break;
            }
            case 'heal': {
                if (!tgt) { break; }
                const r = react(tgt, '', { fx: [{ kind: 'heal', at: 0 }] });
                if (e.amount > 0) { digit(r, '+' + e.amount, COLORS.heal); }
                if (e.held_back > 0) { digit(r, '(+' + e.held_back + ')', COLORS.held, { dim: true }); }
                steps.push(r);
                break;
            }
            case 'windup-start': {
                const w = act(src, 'windup', { hold: 'windup' });
                w.ops.push(op('status+', src, { status: 'winding-up' }));
                steps.push(w);
                break;
            }
            case 'windup-land': {
                const w = act(src, 'attack', { strikeAt: strikeAt });
                if (!reduced && tgt && e.outcome !== 'wasted') { w.lunge = tgt; }
                w.ops.push(op('status-', src, { status: 'winding-up' }));
                w.release = 'windup';
                steps.push(w);
                break;
            }
            case 'interrupt': {
                if (!tgt) { break; }
                steps.push(react(tgt, 'shield-bash', { fx: [{ kind: 'hit', id: 'shield-bash', at: 0 }], ops: [op('casting', tgt, { spell: '' })] }));
                break;
            }
            case 'status-applied': {
                if (!tgt) { break; }
                const s = react(tgt, '', { ops: [op('status+', tgt, { status: e.status || '' })] });
                if (/armou?r.?(broken|break|shatter)|sunder/i.test(e.status || '')) { s.fx = [{ kind: 'armor-shatter', at: 0 }]; }
                steps.push(s);
                break;
            }
            case 'status-expired': {
                if (!tgt) { break; }
                steps.push(react(tgt, '', { ops: [op('status-', tgt, { status: e.status || '' })] }));
                break;
            }
            case 'status-tick': {
                if (!tgt) { break; }
                const t = react(tgt, '', { tint: reduced ? '' : COLORS.tick, fx: [{ kind: 'tick', at: 0 }] });
                if (e.outcome === 'lost-action') { t.icon = 'skip'; }
                else if (e.damage > 0) { digit(t, e.damage, COLORS.tick); }
                steps.push(t);
                break;
            }
            case 'guard-used': {
                const g = act(src, 'guard-step', { lunge: reduced ? '' : tgt });
                g.fx = [{ kind: 'hit', id: 'guard-intercept', at: strikeAt, on: tgt || src }];
                steps.push(g);
                break;
            }
            case 'ability': {
                if (/tackle/i.test(e.status || '')) {
                    const a = act(src, 'attack', { strikeAt: strikeAt, lunge: reduced ? '' : tgt });
                    steps.push(a);
                    if (tgt && e.outcome !== 'failed') { steps.push(react(tgt, 'prone', { delay: strikeAt, hold: 'prone' })); }
                } else {
                    // Opening Strike, Aimed Shot and the like: a brief highlight before the blow.
                    steps.push(act(src, '', { fx: [{ kind: 'highlight', at: 0 }] }));
                }
                break;
            }
            case 'yield': {
                steps.push(act(src, 'yield', { ops: [op('yielded', src, { value: true })], hold: 'yield' }));
                break;
            }
            case 'flee': {
                const f = act(src, 'walk', { exit: true, dur: Math.round(b.action * 1.5), ops: [op('remove', src, {}, 'end')] });
                steps.push(f);
                break;
            }
            case 'death': {
                if (!tgt) { break; }
                steps.push(act(tgt, 'down', { ops: [op('fallen', tgt, { value: true }, 'end')], hold: 'down' }));
                break;
            }
            case 'mercy': {
                if (!tgt) { break; }
                if (e.outcome === 'executed') {
                    const a = act(src, 'attack', { strikeAt: strikeAt });
                    steps.push(a);
                    steps.push(act(tgt, 'down', { delay: strikeAt, ops: [op('fallen', tgt, { value: true }, 'end')], hold: 'down' }));
                } else {
                    steps.push(react(tgt, '', { fade: true, ops: [op('remove', tgt, {}, 'end')] }));
                }
                break;
            }
            case 'fight-end': {
                if (e.outcome === 'victory') { steps.push(act('*company*', 'victory', { hold: 'victory' })); }
                else { steps.push(act('*screen*', '', { fadeScreen: true, dur: b.action })); }
                break;
            }
            default:
                break;
            }
            if (steps.length) { out.push({ seq: e.seq, kind: e.kind, steps: steps }); }
        });
        return out;
    }

    // Scheduler places happenings in time. Steps queue per unit: a unit's new
    // step waits for its previous one, while different units overlap. A
    // happening's own steps keep their delays from its start, which is when
    // its acting unit (the first step's) is free.
    function Scheduler() {
        this.free = new Map();     // unit -> time it is next free
        this.pending = [];         // placed steps not yet finished
        this.held = new Map();     // hold key + unit -> step held at its last frame
    }

    Scheduler.prototype.backlog = function(now) {
        let end = now;
        this.free.forEach(t => { if (t > end) { end = t; } });
        return end - now;
    };

    // push places happenings from `now`. When the queue already holds more
    // than a round of work it first collapses it: the returned ops are the
    // end states of everything dropped, in order, for the caller to apply at
    // once, and the queue starts clean.
    Scheduler.prototype.push = function(happenings, now) {
        let collapsed = [];
        if (this.backlog(now) > ROUND_MS) { collapsed = this.collapse(); }
        happenings.forEach(h => {
            const base = Math.max(now, this.free.get(h.steps[0].unit) || 0);
            h.steps.forEach(s => {
                const placed = Object.assign({}, s);
                const start = Math.max(base + (s.delay || 0), now, this.free.get(s.unit) || 0);
                placed.start = start;
                placed.end = start + Math.max(0, s.dur || 0);
                // An effect's `at` counts from the happening's own start, which
                // queueing may have moved: that is this step's start less its delay.
                placed.fx = (s.fx || []).map(f => Object.assign({}, f, { start: start - (s.delay || 0) + (f.at || 0) }));
                placed.fired = { start: false, end: false };
                this.free.set(s.unit, placed.end);
                this.pending.push(placed);
            });
        });
        return collapsed;
    };

    // collapse drops every pending step and returns the ops they carried.
    Scheduler.prototype.collapse = function() {
        const ops = [];
        this.pending.forEach(s => {
            s.ops.forEach(o => { if (!s.fired[o.when]) { ops.push(o); } });
        });
        this.pending = [];
        this.free = new Map();
        this.held = new Map();
        return ops;
    };

    // due returns the ops whose time has come (start ops when a step has
    // started, end ops when it has ended), and drops the finished steps.
    Scheduler.prototype.due = function(now) {
        const ops = [];
        this.pending.forEach(s => {
            if (!s.fired.start && now >= s.start) {
                s.fired.start = true;
                // What the step shows on its unit as it starts: digits, a
                // feedback icon, a tint. The picture applies it like any op.
                if (s.digits || s.icon || s.tint) {
                    ops.push({ op: 'react', when: 'start', unit: s.unit, digits: s.digits || [], icon: s.icon || '', tint: s.tint || '', bigHit: (s.digits || []).some(d => d.big) });
                }
                s.ops.forEach(o => { if (o.when === 'start') { ops.push(o); } });
            }
            if (!s.fired.end && now >= s.end) {
                s.fired.end = true;
                s.ops.forEach(o => { if (o.when === 'end') { ops.push(o); } });
            }
        });
        this.pending = this.pending.filter(s => !(s.fired.end && now > s.end + 1500));
        return ops;
    };

    // active lists the steps under way at `now`.
    Scheduler.prototype.active = function(now) {
        return this.pending.filter(s => now >= s.start && now < s.end);
    };

    // activeFx lists the effects showing at `now`, each with its progress
    // (0 to 1) over its life (its own dur, or FX_MS).
    Scheduler.prototype.activeFx = function(now) {
        const out = [];
        this.pending.forEach(s => {
            s.fx.forEach(f => {
                const life = f.dur || FX_MS;
                if (now >= f.start && now < f.start + life) { out.push(Object.assign({}, f, { unit: s.unit, p: (now - f.start) / life })); }
            });
        });
        return out;
    };

    // idle is true when nothing is pending to play or to fire.
    Scheduler.prototype.idle = function(now) {
        return this.pending.every(s => s.fired.end && now > s.end);
    };

    // inferPace guesses the player's combat pace from how event batches
    // arrived (the feed does not carry the setting): history is a list of
    // { at, n } for each batch, oldest first. A batch of many events means
    // pacing is off (a round arrives whole); otherwise the gap between
    // batches within a round says fast, normal or slow.
    function inferPace(history) {
        if (!history || history.length < 2) { return 'normal'; }
        const recent = history.slice(-8);
        const avg = recent.reduce((a, h) => a + h.n, 0) / recent.length;
        if (avg >= 5) { return 'off'; }
        const gaps = [];
        for (let i = 1; i < recent.length; i++) { gaps.push(recent[i].at - recent[i - 1].at); }
        gaps.sort((a, b) => a - b);
        const med = gaps[Math.floor(gaps.length / 2)];
        if (med < 900) { return 'fast'; }
        if (med > 2500) { return 'slow'; }
        return 'normal';
    }

    return { ROUND_MS: ROUND_MS, FX_MS: FX_MS, BUDGETS: BUDGETS, budgetFor: budgetFor, plan: plan, Scheduler: Scheduler, inferPace: inferPace, resolveAnim: resolveAnim, hitEffect: hitEffect };
}));
