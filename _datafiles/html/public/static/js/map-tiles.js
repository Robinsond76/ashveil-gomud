/**
 * map-tiles.js
 *
 * Road and coast auto-tiling for the map's terrain (E2). The A2 art draws
 * roads and coasts as 16 pieces each, named by the edges they join, in
 * north, east, south, west order (`road-ne`, `shore-nesw`, `road-none`):
 *
 *   - a road joins each neighbour that is also road and has an exit to it
 *     (a found secret passage doesn't count), and each exit that leads into
 *     the fog (an unvisited room), where the road runs on out of sight;
 *   - a coast faces each neighbour whose ground is water, exit or not.
 *
 * It is pure, so Node can test it (scripts/js/map-tiles.test.mjs, `make
 * js-test`).
 *
 *   MapTiles.sides(joins)       -> the edges joins(dx, dy) is true for, in
 *                                  n, e, s, w order, or 'none'
 *   MapTiles.piece(env, look)   -> the piece's file name ('road-ew',
 *                                  'shore-s'), or '' for other ground; look
 *                                  has envAt(dx, dy) (the ground of the room
 *                                  at that offset, '' when none), linked(dx,
 *                                  dy) (an exit joins it, not a secret one)
 *                                  and fogAt(dx, dy) (an exit leads into fog)
 */

/* globals module */

(function(root, factory) {
    'use strict';
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.MapTiles = factory();
    }
}(typeof self !== 'undefined' ? self : this, function() {
    'use strict';

    // Grid y grows southward, as on the map.
    var DIRS = [['n', 0, -1], ['e', 1, 0], ['s', 0, 1], ['w', -1, 0]];

    function sides(joins) {
        var out = '';
        DIRS.forEach(function(d) { if (joins(d[1], d[2])) { out += d[0]; } });
        return out || 'none';
    }

    function piece(env, look) {
        if (env === 'road') {
            return 'road-' + sides(function(dx, dy) {
                return (look.envAt(dx, dy) === 'road' && look.linked(dx, dy)) || look.fogAt(dx, dy);
            });
        }
        if (env === 'shore') {
            return 'shore-' + sides(function(dx, dy) { return look.envAt(dx, dy) === 'water'; });
        }
        return '';
    }

    return { sides: sides, piece: piece };
}));
