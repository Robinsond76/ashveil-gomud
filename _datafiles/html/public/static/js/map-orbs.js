/**
 * map-orbs.js
 *
 * The painted-orb map style (owner request 2026-10-13): every room is a
 * round, shaded picture of what it is (its terrain, and its landmark when it
 * has one) in a bronze rim; exits are lines between the orbs; the room's
 * resource icons sit round the orb, clear of its exit lines. Roads are
 * smaller orbs painted as a track that runs out the way the road's exits go,
 * over the ground the road passes through, so no two stretches look alike.
 *
 * The layout helpers are pure, so Node can test them
 * (scripts/js/map-orbs.test.mjs, `make js-test`); paint() and paintRoad()
 * need a browser canvas. Painting only composites images and gradients: it
 * never reads pixels back, so art served from another origin (a CDN) still
 * draws.
 *
 *   MapOrbs.angleOf(dx, dy)          -> the screen angle of a grid offset
 *                                       (y grows south, 0 is east)
 *   MapOrbs.ringSlots(n, blocked, step)
 *                                    -> n angles round an orb for resource
 *                                       icons, starting at the bottom and
 *                                       spreading out both ways, skipping
 *                                       the blocked angles ([{ a, half }])
 *                                       while the ring has room
 *   MapOrbs.titleMatch(rules, name)  -> the value of the first [text, value]
 *                                       rule whose text the room name holds
 *                                       (case-insensitive), or ''
 *   MapOrbs.roadGround(envs)         -> the ground a road crosses: the
 *                                       commonest neighbouring ground that
 *                                       is not road or water, else 'land'
 *   MapOrbs.seeded(n)                -> a repeatable random function
 *   MapOrbs.paint(spec)              -> a canvas holding the orb (browser)
 */

/* globals module */

(function(root, factory) {
    'use strict';
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.MapOrbs = factory();
    }
}(typeof self !== 'undefined' ? self : this, function() {
    'use strict';

    var TAU = Math.PI * 2;

    function angleOf(dx, dy) { return Math.atan2(dy, dx); }

    // gap is the smallest distance between two angles, in [0, PI].
    function gap(a, b) {
        var d = Math.abs(a - b) % TAU;
        return d > Math.PI ? TAU - d : d;
    }

    function ringSlots(n, blocked, step) {
        var out = [];
        if (n <= 0) { return out; }
        blocked = blocked || [];
        var cands = [];
        var maxK = Math.floor(Math.PI / step);
        for (var k = 0; k <= maxK; k++) {
            cands.push(Math.PI / 2 + k * step);
            if (k > 0 && k * step < Math.PI - step / 2) { cands.push(Math.PI / 2 - k * step); }
        }
        function free(a) {
            return blocked.every(function (b) { return gap(a, b.a) >= b.half + step / 2; });
        }
        cands.forEach(function (a) { if (out.length < n && free(a)) { out.push(a); } });
        // A full ring: the rest go where they must, over a line if need be.
        cands.forEach(function (a) { if (out.length < n && out.indexOf(a) === -1) { out.push(a); } });
        while (out.length < n) { out.push(Math.PI / 2 + out.length * step); }
        return out;
    }

    function titleMatch(rules, name) {
        if (!Array.isArray(rules) || !name) { return ''; }
        var lower = String(name).toLowerCase();
        for (var i = 0; i < rules.length; i++) {
            var r = rules[i];
            if (Array.isArray(r) && r.length >= 2 && r[0] && lower.indexOf(String(r[0]).toLowerCase()) !== -1) {
                return String(r[1]);
            }
        }
        return '';
    }

    function roadGround(envs) {
        var count = {}, best = '', most = 0;
        (envs || []).forEach(function (e) {
            if (!e || e === 'road' || e === 'water') { return; }
            count[e] = (count[e] || 0) + 1;
            if (count[e] > most) { most = count[e]; best = e; }
        });
        return best || 'land';
    }

    function seeded(n) {
        var s = (Math.abs(Math.floor(n)) % 2147483646) + 1;
        return function () {
            s = (s * 16807) % 2147483647;
            return (s - 1) / 2147483646;
        };
    }

    // -- Painting (browser) --------------------------------------------------

    // ROAD_TONES are the track's colours: packed earth, pale gravel, dark
    // loam and worn flagstones, one per road picked by its room id.
    var ROAD_TONES = [
        { edge: '#5a4529', fill: '#a3845a', rut: 'rgba(72,52,30,0.55)' },
        { edge: '#5e5444', fill: '#b5a483', rut: 'rgba(90,78,60,0.5)' },
        { edge: '#3f2f1d', fill: '#87683f', rut: 'rgba(52,36,20,0.6)' },
        { edge: '#4b4a46', fill: '#9a958a', rut: 'rgba(60,58,54,0.5)' },
    ];

    // paintRoad draws the track over the ground: a band from the middle out
    // each way an exit leaves, with an edge, two cart ruts, loose stones and
    // grass tufts, and now and then a milestone, all picked by the seed.
    function paintRoad(g, cx, cy, r, road) {
        var rnd = seeded(road.seed || 1);
        var tone = ROAD_TONES[Math.floor(rnd() * ROAD_TONES.length)];
        var band = r * (0.40 + rnd() * 0.12);
        var dirs = (road.dirs && road.dirs.length) ? road.dirs : [];
        function each(fn) {
            dirs.forEach(function (d) {
                var l = Math.sqrt(d[0] * d[0] + d[1] * d[1]) || 1;
                fn(d[0] / l, d[1] / l);
            });
        }
        // a slight bend: the middle drifts off centre a little
        var mx = cx + (rnd() - 0.5) * r * 0.22, my = cy + (rnd() - 0.5) * r * 0.22;
        function stroke(width, style) {
            g.strokeStyle = style; g.lineWidth = width; g.lineCap = 'round'; g.lineJoin = 'round';
            if (!dirs.length) {
                g.beginPath(); g.arc(mx, my, width * 0.5, 0, TAU); g.fillStyle = style; g.fill();
                return;
            }
            each(function (ux, uy) {
                g.beginPath();
                g.moveTo(mx, my);
                g.quadraticCurveTo(cx + ux * r * 0.5, cy + uy * r * 0.5, cx + ux * r * 1.3, cy + uy * r * 1.3);
                g.stroke();
            });
        }
        stroke(band * 1.3, tone.edge);
        stroke(band, tone.fill);
        // ruts
        g.save();
        g.lineWidth = Math.max(0.8, band * 0.08);
        g.strokeStyle = tone.rut;
        each(function (ux, uy) {
            [-0.24, 0.24].forEach(function (o) {
                var ox = -uy * band * o, oy = ux * band * o;
                g.beginPath();
                g.moveTo(mx + ox, my + oy);
                g.quadraticCurveTo(cx + ux * r * 0.5 + ox, cy + uy * r * 0.5 + oy, cx + ux * r * 1.3 + ox, cy + uy * r * 1.3 + oy);
                g.stroke();
            });
        });
        g.restore();
        // loose stones on and beside the track
        var stones = 5 + Math.floor(rnd() * 7);
        for (var i = 0; i < stones; i++) {
            var a = rnd() * TAU, d = rnd() * r * 0.9;
            var sr = Math.max(0.6, r * (0.025 + rnd() * 0.035));
            var shade = 120 + Math.floor(rnd() * 70);
            g.fillStyle = 'rgba(' + shade + ',' + (shade - 6) + ',' + (shade - 16) + ',0.9)';
            g.beginPath(); g.arc(cx + Math.cos(a) * d, cy + Math.sin(a) * d, sr, 0, TAU); g.fill();
        }
        // grass tufts at the verge
        var tufts = 3 + Math.floor(rnd() * 6);
        g.strokeStyle = 'rgba(70,110,40,0.85)';
        g.lineWidth = Math.max(0.6, r * 0.025);
        for (var t = 0; t < tufts; t++) {
            var ta = rnd() * TAU, td = r * (0.45 + rnd() * 0.45);
            var tx = cx + Math.cos(ta) * td, ty = cy + Math.sin(ta) * td, th = r * (0.06 + rnd() * 0.05);
            g.beginPath();
            g.moveTo(tx - th * 0.5, ty); g.lineTo(tx - th * 0.7, ty - th);
            g.moveTo(tx, ty); g.lineTo(tx, ty - th * 1.2);
            g.moveTo(tx + th * 0.5, ty); g.lineTo(tx + th * 0.8, ty - th);
            g.stroke();
        }
        // a milestone on one road in five
        if (rnd() < 0.2) {
            var ma = rnd() * TAU, md = r * 0.6;
            var sx = cx + Math.cos(ma) * md, sy = cy + Math.sin(ma) * md, sw = r * 0.12, sh = r * 0.2;
            g.fillStyle = 'rgba(0,0,0,0.35)';
            g.beginPath(); g.ellipse(sx + sw * 0.3, sy + sh * 0.5, sw * 0.8, sw * 0.35, 0, 0, TAU); g.fill();
            g.fillStyle = '#9c968a';
            g.beginPath();
            g.moveTo(sx - sw / 2, sy + sh * 0.5); g.lineTo(sx - sw / 2, sy - sh * 0.3);
            g.quadraticCurveTo(sx, sy - sh * 0.75, sx + sw / 2, sy - sh * 0.3);
            g.lineTo(sx + sw / 2, sy + sh * 0.5); g.closePath(); g.fill();
            g.fillStyle = 'rgba(255,255,255,0.25)';
            g.fillRect(sx - sw / 2, sy - sh * 0.2, sw * 0.3, sh * 0.6);
        }
    }

    function drawFrame(g, f, x, y, w, h) {
        g.drawImage(f.img, f.sx, f.sy, f.sw, f.sh, x, y, w, h);
    }

    // paint builds one orb at device pixels. spec:
    //   r        radius, device pixels
    //   ground   { img, sx, sy, sw, sh } the terrain frame, or null
    //   color    the flat colour used while there is no ground art
    //   landmark { img, sx, sy, sw, sh } or null
    //   road     { dirs: [[dx, dy], ...], seed } for a road orb, or null
    //   rim      'bronze' (default) or 'gold' (your room with the halo off)
    // The canvas is square, with the orb in the middle; pad is the room
    // left round it for the rim.
    function paint(spec) {
        var r = Math.max(2, spec.r);
        var bw = Math.max(1.5, r * (spec.road ? 0.065 : 0.085));
        var pad = Math.ceil(bw * 2 + 2);
        var size = Math.ceil(2 * (r + pad));
        var c = document.createElement('canvas');
        c.width = c.height = size;
        var g = c.getContext('2d');
        var cx = size / 2, cy = size / 2;

        g.save();
        g.beginPath(); g.arc(cx, cy, r, 0, TAU); g.clip();
        g.fillStyle = spec.color || '#4a4a52';
        g.fillRect(0, 0, size, size);
        g.imageSmoothingEnabled = true;
        if (g.imageSmoothingQuality !== undefined) { g.imageSmoothingQuality = 'high'; }
        if (spec.ground) {
            // a little larger than the orb, so the rim never shows an edge
            drawFrame(g, spec.ground, cx - r * 1.04, cy - r * 1.04, r * 2.08, r * 2.08);
        }
        if (spec.road) { paintRoad(g, cx, cy, r, spec.road); }
        if (spec.landmark) {
            var sh = g.createRadialGradient(cx, cy + r * 0.38, 1, cx, cy + r * 0.38, r * 0.8);
            sh.addColorStop(0, 'rgba(0,0,0,0.45)'); sh.addColorStop(1, 'rgba(0,0,0,0)');
            g.fillStyle = sh; g.fillRect(0, 0, size, size);
            var lw = r * 1.56;
            drawFrame(g, spec.landmark, cx - lw / 2, cy - lw / 2 - r * 0.09, lw, lw);
        }
        // light from the upper left, the far side in shadow
        var lit = g.createRadialGradient(cx - r * 0.35, cy - r * 0.45, 0, cx - r * 0.35, cy - r * 0.45, r * 1.75);
        lit.addColorStop(0, 'rgba(255,244,220,0.18)');
        lit.addColorStop(0.45, 'rgba(0,0,0,0)');
        lit.addColorStop(1, 'rgba(0,0,0,0.62)');
        g.fillStyle = lit; g.fillRect(0, 0, size, size);
        var edge = g.createRadialGradient(cx, cy, r * 0.62, cx, cy, r);
        edge.addColorStop(0, 'rgba(0,0,0,0)');
        edge.addColorStop(1, 'rgba(0,0,0,0.38)');
        g.fillStyle = edge; g.fillRect(0, 0, size, size);
        // a soft varnish highlight
        g.globalCompositeOperation = 'lighter';
        var hx = cx - r * 0.4, hy = cy - r * 0.45;
        var spec1 = g.createRadialGradient(hx, hy, 0, hx, hy, r * 0.42);
        spec1.addColorStop(0, 'rgba(255,255,255,0.22)');
        spec1.addColorStop(1, 'rgba(255,255,255,0)');
        g.fillStyle = spec1; g.fillRect(0, 0, size, size);
        var spec2 = g.createRadialGradient(hx, hy, 0, hx, hy, r * 0.12);
        spec2.addColorStop(0, 'rgba(255,255,255,0.28)');
        spec2.addColorStop(1, 'rgba(255,255,255,0)');
        g.fillStyle = spec2; g.fillRect(0, 0, size, size);
        g.restore();

        // the rim
        var gold = spec.rim === 'gold';
        g.strokeStyle = 'rgba(0,0,0,0.6)';
        g.lineWidth = bw + 2;
        g.beginPath(); g.arc(cx, cy, r + bw * 0.4, 0, TAU); g.stroke();
        var rim = g.createLinearGradient(cx - r, cy - r, cx + r, cy + r);
        if (gold) {
            rim.addColorStop(0, '#fff2b8'); rim.addColorStop(0.35, '#f0c048');
            rim.addColorStop(0.7, '#9a6a12'); rim.addColorStop(1, '#ffd870');
        } else {
            rim.addColorStop(0, '#f0d39a'); rim.addColorStop(0.35, '#a87a3c');
            rim.addColorStop(0.7, '#5a3c1a'); rim.addColorStop(1, '#c79a5a');
        }
        g.strokeStyle = rim;
        g.lineWidth = bw;
        g.beginPath(); g.arc(cx, cy, r + bw * 0.4, 0, TAU); g.stroke();
        return c;
    }

    return { angleOf: angleOf, ringSlots: ringSlots, titleMatch: titleMatch, roadGround: roadGround,
             seeded: seeded, paint: paint, gap: gap };
}));
