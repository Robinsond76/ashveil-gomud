/**
 * keep-scroll.js
 *
 * Panels here rebuild themselves from every GMCP update by emptying and
 * refilling their content, which snaps any scrolled box back to the top.
 * keepScroll(node) is called just before such a rebuild: it remembers the
 * scroll position of the node and of each ancestor, and puts them back once
 * the rebuild has finished (a microtask, so before the next paint).
 */
(function (root, factory) {
    if (typeof module === 'object' && module.exports) { module.exports = factory(); }
    else { root.keepScroll = factory(); }
}(typeof window !== 'undefined' ? window : globalThis, function () {
    function keepScroll(node) {
        var saved = [];
        for (var n = node; n; n = n.parentElement) {
            if (n.scrollTop > 0) { saved.push([n, n.scrollTop]); }
        }
        if (!saved.length) { return; }
        queueMicrotask(function () {
            saved.forEach(function (s) { s[0].scrollTop = s[1]; });
        });
    }
    return keepScroll;
}));
