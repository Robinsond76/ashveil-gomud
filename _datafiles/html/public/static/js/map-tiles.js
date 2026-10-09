/**
 * map-tiles.js
 *
 * Road and coast auto-tiling for the map's terrain (E2). The A2 art draws
 * roads and coasts as 16 pieces each, named by the edges they join, in
 * north, east, south, west order (`road-ne`, `shore-nesw`, `road-none`):
 *
 *   - a road joins each neighbour that is also road and has an exit to it
 *     (a found secret passage doesn't count), and each exit that leaves the
 *     drawn map (into the fog of an unvisited room, or into another zone),
 *     where the road runs on out of sight;
 *   - a coast faces each neighbour whose ground is water, exit or not.
 *
 * The world has no water rooms: a lake is the empty middle of a ring of
 * shore rooms (Frost Lake, Alderbrook's pond). So lakes() finds the empty
 * cells a map's rooms fully enclose where every room around them is shore,
 * and the map draws those as open water, which the coast then faces (owner
 * decision, 2026-10-09: "lakes from shape").
 *
 * It is pure, so Node can test it (scripts/js/map-tiles.test.mjs, `make
 * js-test`).
 *
 *   MapTiles.sides(joins)       -> the edges joins(dx, dy) is true for, in
 *                                  n, e, s, w order, or 'none'
 *   MapTiles.tiled(env)         -> whether env's ground is drawn as pieces
 *   MapTiles.piece(env, look)   -> the piece's file name ('road-ew',
 *                                  'shore-s'), or '' for other ground; look
 *                                  has envAt(dx, dy) (the ground of the room
 *                                  at that offset, '' when none), linked(dx,
 *                                  dy) (an exit joins it, not a secret one)
 *                                  and onward(dx, dy) (an exit leaves the
 *                                  drawn map there)
 *   MapTiles.lakes(cells, blocked) -> the 'x,y' keys of lake cells; cells
 *                                  are the drawn rooms ({ x, y, env }),
 *                                  blocked the 'x,y' keys an exit leads into
 *                                  (a room not yet seen, so not a lake)
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
                return (look.envAt(dx, dy) === 'road' && look.linked(dx, dy)) || look.onward(dx, dy);
            });
        }
        if (env === 'shore') {
            return 'shore-' + sides(function(dx, dy) { return look.envAt(dx, dy) === 'water'; });
        }
        return '';
    }

    var LAKE_MAX_AREA = 40000;   // a map larger than this (with its margin) finds no lakes

    function lakes(cells, blocked) {
        if (!cells.length) { return []; }
        var env = {}, minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
        cells.forEach(function(c) {
            env[c.x + ',' + c.y] = c.env || '';
            minX = Math.min(minX, c.x); maxX = Math.max(maxX, c.x);
            minY = Math.min(minY, c.y); maxY = Math.max(maxY, c.y);
        });
        minX--; minY--; maxX++; maxY++;
        if ((maxX - minX + 1) * (maxY - minY + 1) > LAKE_MAX_AREA) { return []; }
        var seen = {};
        function fill(x0, y0) {
            // The empty cells 4-connected to (x0, y0) inside the box.
            var out = [], q = [[x0, y0]];
            seen[x0 + ',' + y0] = true;
            while (q.length) {
                var c = q.pop();
                out.push(c);
                DIRS.forEach(function(d) {
                    var x = c[0] + d[1], y = c[1] + d[2], k = x + ',' + y;
                    if (x < minX || x > maxX || y < minY || y > maxY || seen[k] || k in env) { return; }
                    seen[k] = true;
                    q.push([x, y]);
                });
            }
            return out;
        }
        // Everything reachable from the box's margin is open country, not a lake.
        fill(minX, minY);
        var found = [];
        for (var y = minY; y <= maxY; y++) {
            for (var x = minX; x <= maxX; x++) {
                var k = x + ',' + y;
                if (seen[k] || k in env) { continue; }
                var region = fill(x, y);
                var ok = region.every(function(c) {
                    if (blocked && blocked[c[0] + ',' + c[1]]) { return false; }
                    return DIRS.every(function(d) {
                        var n = (c[0] + d[1]) + ',' + (c[1] + d[2]);
                        return !(n in env) || env[n] === 'shore';
                    });
                });
                if (ok) { region.forEach(function(c) { found.push(c[0] + ',' + c[1]); }); }
            }
        }
        return found;
    }

    function tiled(env) { return env === 'road' || env === 'shore'; }

    return { sides: sides, piece: piece, tiled: tiled, lakes: lakes };
}));
