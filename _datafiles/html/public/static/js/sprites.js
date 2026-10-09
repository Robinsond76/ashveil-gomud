/*
 * Sprite loader (Phase 40b): the shared client for the pixel-art sprites in
 * static/sprites/, read through its manifest.json. The map (window-map.js)
 * uses it; 40c and later phases reuse it.
 *
 *   Sprites.art(path)       -> { img, info } once <path> is listed in the
 *                              manifest and loaded, else null. A path the
 *                              manifest does not list is never requested,
 *                              so art that has not been drawn costs no
 *                              failed fetches; a missing or broken image
 *                              stays null and the caller falls back.
 *   Sprites.onChange(fn)    -> fn() runs when the manifest or an image
 *                              arrives, so a window can redraw.
 *   Sprites.has(path)       -> true when the manifest lists <path>.
 *   Sprites.status(path)    -> 'none' (not listed, or no manifest yet),
 *                              'loading', 'ready' or 'failed'; asking
 *                              starts the load. A caller with a fallback
 *                              chain waits on 'loading' and moves down the
 *                              chain on 'none' or 'failed'.
 *   Sprites.data(path)      -> the parsed JSON file at <path> under sprites/
 *                              (Phase 40c: map/landmarks.json) once it has
 *                              loaded, else null; asking starts the load
 *                              and a failure stays null.
 *   Sprites.tinted(path, look) -> Sprites.art(path) with the figure's skin and
 *                              hair repainted in look's colours (Phase 72a;
 *                              sprite-tint.js builds a look). With no look, or
 *                              before sprite-tint.js has loaded, it is art(path),
 *                              and so is high-density art, which the palette
 *                              swap cannot repaint (SpriteTint.applies).
 *                              The repainted canvas is made once per path and
 *                              look.
 *   Sprites.frame(info, rowIndex, now, startMs)
 *                           -> the { sx, sy, sw, sh } source rectangle of
 *                              the frame an animation shows at time <now>
 *                              (ms; startMs when it began, for one-shots).
 *
 * Frames run left to right and rows top to bottom (the manifest's `rows`).
 */
window.Sprites = (function () {
    var here = document.currentScript && document.currentScript.src ? document.currentScript.src : '';
    var base = here.replace(/js\/sprites\.js(\?.*)?$/, 'sprites/');
    var manifest = null;
    var images = {};      // path -> HTMLImageElement | false (loading) | 'failed'
    var listeners = [];

    function changed() {
        listeners.slice().forEach(function (fn) {
            try { fn(); } catch (e) { /* a listener's fault never stops the others */ }
        });
    }

    function load() {
        if (!base || !window.fetch || /^file:/.test(base)) { return; }
        fetch(base + 'manifest.json').then(function (r) { return r.ok ? r.json() : null; }).then(function (m) {
            if (m && m.files) { manifest = m; changed(); }
        }).catch(function () { /* no art: the code-drawn markers stand */ });
    }

    function has(path) {
        return !!(manifest && manifest.files[path]);
    }

    function art(path) {
        if (!has(path)) { return null; }
        if (!(path in images)) {
            var img = new Image();
            images[path] = false;
            img.onload = function () { images[path] = img; changed(); };
            img.onerror = function () { images[path] = 'failed'; changed(); };
            img.src = base + path;
        }
        return (images[path] && images[path] !== 'failed') ? { img: images[path], info: manifest.files[path] } : null;
    }

    var tintCache = {};

    function tinted(path, look) {
        var a = art(path);
        if (!a || !look || !window.SpriteTint || !window.SpriteTint.applies(a.info)) { return a; }
        var canvas = window.SpriteTint.canvasFor(a.img, look, function (w, h) {
            var c = document.createElement('canvas');
            c.width = w; c.height = h;
            return c;
        }, tintCache, path);
        return { img: canvas, info: a.info };
    }

    function status(path) {
        if (!has(path)) { return 'none'; }
        if (art(path)) { return 'ready'; }
        return images[path] === 'failed' ? 'failed' : 'loading';
    }

    var datas = {};       // path -> parsed JSON | false (loading or failed)

    function data(path) {
        if (!(path in datas)) {
            datas[path] = false;
            if (!base || !window.fetch || /^file:/.test(base)) { return null; }
            fetch(base + path).then(function (r) { return r.ok ? r.json() : null; }).then(function (d) {
                if (d) { datas[path] = d; changed(); }
            }).catch(function () { /* the glyphs stand */ });
        }
        return datas[path] || null;
    }

    function frame(info, rowIndex, now, startMs) {
        var fw = info.frame[0], fh = info.frame[1];
        var n = Math.max(1, info.frames || 1);
        var ms = info.frame_ms || 0;
        var i = (ms > 0 && n > 1) ? Math.floor(Math.max(0, now - (startMs || 0)) / ms) % n : 0;
        return { sx: i * fw, sy: (rowIndex || 0) * fh, sw: fw, sh: fh };
    }

    function onChange(fn) { listeners.push(fn); }

    load();
    return { art: art, tinted: tinted, has: has, status: status, data: data, frame: frame, onChange: onChange };
}());
