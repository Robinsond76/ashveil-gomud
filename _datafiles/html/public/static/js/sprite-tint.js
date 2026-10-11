/**
 * sprite-tint.js
 *
 * Skin tone and hair colour on a player's sprites (Phase 72a). The sprite
 * generator draws every figure's face and hands in the master palette's
 * "skin" ramp and, on a bare head, its hair in the "hair" ramp, and nothing
 * else uses either ramp. This repaints those six palette colours with ramps
 * built from the player's chosen colours, so a figure keeps its class, gear
 * and shading and only its skin and hair change. Everything else, including
 * outlines, is left alone.
 *
 * It is pure: it works on RGBA pixel data and a plain-object cache, and
 * knows nothing of the DOM except in canvasFor, which takes a canvas factory
 * so Node can test the rest (scripts/js/sprite-tint.test.mjs, `make js-test`).
 * The base colours below must match scripts/sprites/palette.py; the Node test
 * reads that file and checks.
 *
 *   SpriteTint.look(skin, hair)         -> a look, or null when neither is a #rrggbb
 *   SpriteTint.ramp(hex)                -> { d, m, l } three RGB triples around hex
 *   SpriteTint.apply(data, look)        -> repaints an RGBA array in place, returns
 *                                          how many pixels it changed
 *   SpriteTint.recolour(data, mask, look) -> repaints the pixels a look mask marks
 *                                          (E1b: red skin, green hair) in the chosen
 *                                          colours, keeping their light and shade;
 *                                          returns how many pixels it changed
 *   SpriteTint.applies(info)            -> whether a sheet (its manifest entry) can be
 *                                          repainted: 1x palette art by the swap, and
 *                                          high-density art (E1, E3) that has a
 *                                          look_mask by recolour
 *   SpriteTint.canvasFor(img, look, make, cache, key, maskImg)
 *                                       -> a canvas with img repainted (cached by key);
 *                                          with maskImg, by recolour
 */

/* globals module */

(function(root, factory) {
    'use strict';
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.SpriteTint = factory();
    }
}(typeof self !== 'undefined' ? self : this, function() {
    'use strict';

    // The palette's ramps, dark, mid, light (scripts/sprites/palette.py).
    const BASE = {
        skin: [[0x8a, 0x5a, 0x40], [0xb5, 0x7d, 0x5a], [0xd4, 0xa3, 0x7c]],
        hair: [[0x2f, 0x21, 0x18], [0x4d, 0x37, 0x27], [0x71, 0x56, 0x40]],
    };

    const HEX = /^#[0-9a-fA-F]{6}$/;

    function parse(hex) {
        return [parseInt(hex.slice(1, 3), 16), parseInt(hex.slice(3, 5), 16), parseInt(hex.slice(5, 7), 16)];
    }

    function clamp(v) { return Math.max(0, Math.min(255, Math.round(v))); }

    // ramp: the chosen colour is the mid step; the dark step is a quarter
    // darker and the light step a little brighter, kept inside the palette's
    // own contrast between steps.
    function ramp(hex) {
        const m = parse(hex);
        return {
            d: m.map(c => clamp(c * 0.74)),
            m: m.slice(),
            l: m.map(c => clamp(c * 1.18 + (255 - c) * 0.1)),
        };
    }

    function look(skin, hair) {
        const s = typeof skin === 'string' && HEX.test(skin) ? skin.toLowerCase() : '';
        const h = typeof hair === 'string' && HEX.test(hair) ? hair.toLowerCase() : '';
        if (!s && !h) { return null; }
        return { skin: s, hair: h };
    }

    function key(look) { return look ? (look.skin || '-') + '|' + (look.hair || '-') : ''; }

    // The swaps for a look: palette colour -> replacement, as 24-bit keys.
    function swaps(look) {
        const out = new Map();
        [['skin', look.skin], ['hair', look.hair]].forEach(function(pair) {
            if (!pair[1]) { return; }
            const r = ramp(pair[1]);
            [r.d, r.m, r.l].forEach(function(to, i) {
                const from = BASE[pair[0]][i];
                out.set((from[0] << 16) | (from[1] << 8) | from[2], to);
            });
        });
        return out;
    }

    function apply(data, look) {
        if (!look || !data) { return 0; }
        const map = swaps(look);
        let changed = 0;
        for (let i = 0; i + 3 < data.length; i += 4) {
            if (data[i + 3] === 0) { continue; }
            const to = map.get((data[i] << 16) | (data[i + 1] << 8) | data[i + 2]);
            if (to) {
                data[i] = to[0]; data[i + 1] = to[1]; data[i + 2] = to[2];
                changed++;
            }
        }
        return changed;
    }

    function luma(r, g, b) { return 0.299 * r + 0.587 * g + 0.114 * b; }

    // recolour (E1b) repaints high-density art, which has thousands of
    // colours, through its look mask (docs/art/A11-look-masks.md): a mask
    // pixel is red for skin, green for hair. Each material's masked pixels
    // take the chosen colour scaled by their own brightness over the
    // material's median brightness, so the median pixel becomes the chosen
    // colour, darker ones darken toward black and lighter ones lift toward
    // white (at most LIFT of the way): the art's light and shade survive.
    const LIFT = 0.6;
    function recolour(data, mask, look) {
        if (!look || !data || !mask || mask.length !== data.length) { return 0; }
        let changed = 0;
        [['skin', 0], ['hair', 1]].forEach(function(m) {
            const hex = look[m[0]];
            if (!hex) { return; }
            const c = parse(hex);
            const idx = [], lum = [];
            for (let i = 0; i + 3 < data.length; i += 4) {
                if (mask[i + 3] === 0 || mask[i + m[1]] < 128 || data[i + 3] === 0) { continue; }
                idx.push(i);
                lum.push(luma(data[i], data[i + 1], data[i + 2]));
            }
            if (!idx.length) { return; }
            const sorted = lum.slice().sort(function(a, b) { return a - b; });
            const ref = Math.max(1, sorted[Math.floor(sorted.length / 2)]);
            idx.forEach(function(i, n) {
                const k = lum[n] / ref;
                for (let ch = 0; ch < 3; ch++) {
                    data[i + ch] = k <= 1 ? clamp(c[ch] * k) : clamp(c[ch] + (255 - c[ch]) * Math.min(1, k - 1) * LIFT);
                }
                changed++;
            });
        });
        return changed;
    }

    // applies: the swap needs the palette's exact colours, which only the
    // generated 1x art uses. High-density imported art (density > 1) has its
    // own colours, so the swap would at best do nothing and at worst
    // recolour a stray pixel that happens to match; it is repainted through
    // its look mask instead (recolour, E1b) when it has one, and shown as
    // drawn when it has none (no skin or hair shows, or no masks yet).
    function applies(info) {
        return !!info && ((info.density || 1) <= 1 || !!info.look_mask);
    }

    // canvasFor draws img into a canvas and repaints it, once per (img, look):
    // the result is kept in cache under key + the look. make(w, h) returns a
    // canvas; it is the only part that touches the DOM.
    function canvasFor(img, lk, make, cache, cacheKey, maskImg) {
        if (!lk || !img) { return img; }
        const k = cacheKey + '#' + key(lk);
        if (cache[k]) { return cache[k]; }
        const w = img.naturalWidth || img.width, h = img.naturalHeight || img.height;
        const cv = make(w, h);
        const ctx = cv.getContext('2d', { willReadFrequently: true });
        ctx.imageSmoothingEnabled = false;
        ctx.drawImage(img, 0, 0);
        try {
            const px = ctx.getImageData(0, 0, w, h);
            if (maskImg) {
                const mc = make(w, h);
                const mctx = mc.getContext('2d', { willReadFrequently: true });
                mctx.imageSmoothingEnabled = false;
                mctx.drawImage(maskImg, 0, 0);
                recolour(px.data, mctx.getImageData(0, 0, w, h).data, lk);
            } else {
                apply(px.data, lk);
            }
            ctx.putImageData(px, 0, 0);
        } catch (e) {
            // A sprite served from another origin (a CDN) cannot be read
            // back; draw it in its stock colours rather than fail the frame.
            cache[k] = img;
            return img;
        }
        cache[k] = cv;
        return cv;
    }

    return { BASE: BASE, look: look, ramp: ramp, apply: apply, swaps: swaps, recolour: recolour, applies: applies, canvasFor: canvasFor };
}));
