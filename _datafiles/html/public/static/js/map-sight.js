/**
 * map-sight.js
 *
 * Light and sight on the map (LM2, docs/designs/2026-10-10-living-map-design.md).
 * The server sends each player their sight in Gametime:
 * { level: 0-2, daylight, source, carried, fog, room } (internal/rooms Sight). These
 * pure rules turn it into what the map draws, so Node can test them
 * (scripts/js/map-sight.test.mjs, `make js-test`):
 *
 *   MapSight.visible(sight, here, joined) -> null when everything you have
 *                       seen is shown as today (no sight from the server, or
 *                       daylight at full sight), else a Set of the room ids in
 *                       sight: your room and the rooms joined to it by exits
 *                       (joined, an array of ids) at level 2, your room only
 *                       at level 1, nothing at level 0. Rooms outside it are
 *                       drawn as remembered: dark and still.
 *   MapSight.glow(sight) -> null in daylight, else { radius, color, mono, fog }:
 *                       the soft pool of light round you, radius in tiles
 *                       (1.5 at level 2, 0.6 at level 1, 0 when you can't
 *                       see your room); color 'warm' for a flame (lantern,
 *                       torch, another carried light, a hearth), 'cool' for a
 *                       party light, 'pale' for moonlight; mono true for
 *                       night vision, which has no glow but a cool
 *                       monochrome tint. fog is true outdoors by day when fog
 *                       shortens sight: the map draws a grey veil, not night,
 *                       and there is no moonlight.
 *   MapSight.marker(sight) -> the marker beside your figure: the light you
 *                       carry ('torch', 'lantern', 'lantern-out'), whichever
 *                       light wins, else 'orb' (your own party light, over
 *                       your head), or ''.
 *   MapSight.key(sight) -> a string that changes when the drawing would.
 */

/* globals module */

(function(root, factory) {
    'use strict';
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.MapSight = factory();
    }
}(typeof self !== 'undefined' ? self : this, function() {
    'use strict';

    function valid(s) { return !!s && typeof s.level === 'number'; }

    function visible(sight, here, joined) {
        if (!valid(sight) || (sight.daylight && sight.level >= 2)) { return null; }
        var set = new Set();
        if (sight.level >= 1 && here !== null && here !== undefined) { set.add(here); }
        if (sight.level >= 2) { (joined || []).forEach(function(id) { set.add(id); }); }
        return set;
    }

    var WARM = { lantern: true, torch: true, light: true, fixture: true };

    function glow(sight) {
        if (!valid(sight) || (sight.daylight && sight.level >= 2)) { return null; }
        var fog = !!sight.fog;
        if (sight.source === 'nightvision') { return { radius: 0, color: '', mono: true, fog: fog }; }
        var radius = sight.level >= 2 ? 1.5 : (sight.level === 1 ? 0.6 : 0);
        var color = WARM[sight.source] ? 'warm'
            : (sight.source === 'orb' || sight.source === 'party') ? 'cool'
                : (radius > 0 && !fog ? 'pale' : '');
        return { radius: radius, color: color, mono: false, fog: fog };
    }

    var CARRIED = { torch: true, lantern: true, 'lantern-out': true };

    function marker(sight) {
        if (!valid(sight)) { return ''; }
        if (CARRIED[sight.carried]) { return sight.carried; }
        if (sight.source === 'orb') { return 'orb'; }
        return '';
    }

    function key(sight) {
        if (!valid(sight)) { return ''; }
        return [sight.level, sight.daylight ? 1 : 0, sight.source || '', sight.carried || '', sight.fog ? 1 : 0].join('|');
    }

    return { visible: visible, glow: glow, marker: marker, key: key };
}));
