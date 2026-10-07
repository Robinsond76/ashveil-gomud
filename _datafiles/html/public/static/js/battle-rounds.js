/**
 * battle-rounds.js
 *
 * Phase 62, battle lines that explain themselves. The server sends each of
 * the player's own company's weapon rounds with `explain`, the plain lines of
 * the engine's own roll (what the hit needed and rolled, the defence it
 * met, armor, named modifiers) in Company.Battle.Event. This keeps the last
 * rounds of the latest fight for the Combat tab, newest first, each a
 * one-line heading the player can open for its breakdown. It is pure (no
 * DOM, no Client) so it runs under Node (make js-test); the Combat tab draws
 * it with textContent only.
 *
 *   BattleRounds.create(cap)  -> { add(message, nameOf), list(), clear() }
 *   BattleRounds.heading(ev, nameOf) -> "Aria → bandit: hit for 5"
 *
 * nameOf(ref) names a ref ('' when unknown, shown as "someone").
 */
(function (root, factory) {
    if (typeof module === 'object' && module.exports) { module.exports = factory(); }
    else { root.BattleRounds = factory(); }
}(typeof window !== 'undefined' ? window : globalThis, function () {
    var DEFAULT_CAP = 12;

    function result(ev) {
        if (ev.defenses && ev.defenses.length) { return 'turned aside (' + ev.defenses.join(', ') + ')'; }
        if (ev.outcome === 'miss') { return 'missed'; }
        if (ev.damage > 0) { return (ev.crit ? 'critical hit for ' : 'hit for ') + ev.damage; }
        return 'hit, no damage';
    }

    function heading(ev, nameOf) {
        var who = (nameOf && nameOf(ev.src)) || 'someone';
        var whom = (nameOf && nameOf(ev.tgt)) || 'someone';
        return who + ' → ' + whom + ': ' + result(ev);
    }

    function create(cap) {
        var max = cap > 0 ? cap : DEFAULT_CAP;
        var rounds = [];
        var fight = null;
        var n = 0;
        return {
            // add keeps the explained attacks of one message; a new fight
            // starts the list afresh. It returns true when the list changed.
            add: function (message, nameOf) {
                if (!message || !Array.isArray(message.events)) { return false; }
                var changed = false;
                if (fight !== null && message.fight !== fight && rounds.length) { rounds = []; changed = true; }
                fight = message.fight;
                message.events.forEach(function (ev) {
                    if (ev.kind !== 'attack' || !Array.isArray(ev.explain) || !ev.explain.length) { return; }
                    n += 1;
                    rounds.unshift({
                        id: n,
                        round: message.fight_round || message.round || 0,
                        head: heading(ev, nameOf),
                        lines: ev.explain.slice(),
                    });
                    changed = true;
                });
                if (rounds.length > max) { rounds.length = max; }
                return changed;
            },
            list: function () { return rounds.slice(); },
            clear: function () { rounds = []; fight = null; },
        };
    }

    return { create: create, heading: heading };
}));
