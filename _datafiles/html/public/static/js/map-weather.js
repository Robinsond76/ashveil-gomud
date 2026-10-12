/**
 * map-weather.js
 *
 * Weather on the map (LM4, docs/designs/2026-10-10-living-map-design.md).
 * The server sends the weather a player can see in Gametime:
 * { name, cloud_cover 0-3, visibility, indoor }. These pure rules turn it
 * into what the map draws over its outdoor tiles, so Node can test them
 * (scripts/js/map-weather.test.mjs, `make js-test`):
 *
 *   MapWeather.plan(weather, opts) -> null when there is nothing to draw,
 *                       else { name, layers, splash, lightning, dim }:
 *     layers     the textures to slide over the tiles, back to front:
 *                { tex, size, vx, vy, alpha }. tex names an A14 texture
 *                (map/weather/<tex>.png); size is how many tiles one copy
 *                spans; vx, vy its speed in tiles per second (rain slants
 *                down to the left); alpha its opacity.
 *     splash     raindrops per tile per second (0 when it isn't raining).
 *     lightning  true in a storm.
 *     dim        how much the sky darkens the tiles (0-1).
 *   opts.reduced (reduced motion) stops every layer and the splashes;
 *   opts.fogVeil says LM2 already draws a fog veil, so fog thins to a third.
 *
 *   MapWeather.nextStrike(now, rand) -> when the next lightning strike
 *                       comes: 4 to 9 seconds after now (rand is 0-1).
 *   MapWeather.strike(start, now) -> null once a strike is over, else
 *                       { flash 0-1, frame 0-2 }: the flash fades within
 *                       FLASH_MS; the bolt's three frames last BOLT_MS.
 *   MapWeather.key(weather) -> a string that changes when the drawing would.
 */

/* globals module */

(function(root, factory) {
    'use strict';
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.MapWeather = factory();
    }
}(typeof self !== 'undefined' ? self : this, function() {
    'use strict';

    // Rain falls from the top right to the bottom left, about 20 degrees
    // off vertical (A14).
    var SLANT_X = -0.34, SLANT_Y = 0.94;
    var FLASH_MS = 140, BOLT_MS = 420;
    var STRIKE_MIN_MS = 4000, STRIKE_SPREAD_MS = 5000;

    // A rain texture spans two tiles, so its streaks stay long enough to
    // read at the map's usual zoom.
    function rain(tex, speed, alpha) {
        return { tex: tex, size: 2, vx: SLANT_X * speed, vy: SLANT_Y * speed, alpha: alpha };
    }

    function plan(weather, opts) {
        if (!weather || !weather.name) { return null; }
        opts = opts || {};
        var name = String(weather.name).toLowerCase();
        var layers = [], splash = 0, lightning = false, dim = 0;
        var clouds = (weather.cloud_cover || 0) >= 2 || name === 'overcast' || name === 'rain' || name === 'storm';
        if (clouds) {
            layers.push({ tex: 'clouds', size: 2, vx: 0.05, vy: 0.02, alpha: 0.18 });
            dim = name === 'storm' ? 0.18 : 0.08;
        }
        // LM2 already veils the fog when it cuts sight: the drifting banks
        // thin to a third so the map stays readable.
        var fog = opts.fogVeil ? 0.35 : 1;
        switch (name) {
        case 'rain':
            layers.push(rain('rain', 3, 0.65));
            splash = 0.6;
            break;
        case 'storm':
            layers.push(rain('rain-heavy', 5, 0.6));
            splash = 1.2;
            lightning = true;
            break;
        case 'mist':
            layers.push({ tex: 'fog', size: 2, vx: 0.08, vy: 0, alpha: 0.25 * fog });
            break;
        case 'fog':
            layers.push({ tex: 'fog', size: 2, vx: 0.06, vy: 0, alpha: 0.4 * fog });
            break;
        case 'thick-fog':
            layers.push({ tex: 'fog', size: 2, vx: 0.05, vy: 0, alpha: 0.55 * fog });
            break;
        case 'snow':
            layers.push({ tex: 'snow-far', size: 1, vx: -0.06, vy: 0.3, alpha: 0.45 });
            layers.push({ tex: 'snow', size: 1, vx: -0.12, vy: 0.55, alpha: 0.7 });
            break;
        case 'blizzard':
            layers.push({ tex: 'snow-far', size: 1, vx: -0.6, vy: 0.9, alpha: 0.6 });
            layers.push({ tex: 'snow', size: 1, vx: -1.0, vy: 1.5, alpha: 0.8 });
            dim = 0.12;
            break;
        }
        if (!layers.length) { return null; }
        if (opts.reduced) {
            layers.forEach(function(l) { l.vx = 0; l.vy = 0; });
            splash = 0;
        }
        return { name: name, layers: layers, splash: splash, lightning: lightning, dim: dim };
    }

    function nextStrike(now, rand) {
        return now + STRIKE_MIN_MS + Math.max(0, Math.min(1, rand)) * STRIKE_SPREAD_MS;
    }

    function strike(start, now) {
        var t = now - start;
        if (t < 0 || t >= BOLT_MS) { return null; }
        return {
            flash: t < FLASH_MS ? 1 - t / FLASH_MS : 0,
            frame: Math.min(2, Math.floor(t / (BOLT_MS / 3))),
        };
    }

    function key(weather) {
        if (!weather || !weather.name) { return ''; }
        return [weather.name, weather.cloud_cover || 0, weather.indoor ? 1 : 0].join('|');
    }

    return {
        plan: plan, nextStrike: nextStrike, strike: strike, key: key,
        FLASH_MS: FLASH_MS, BOLT_MS: BOLT_MS, STRIKE_MIN_MS: STRIKE_MIN_MS,
    };
}));
