/**
 * keep-scroll.js
 *
 * Panels here rebuild themselves from every GMCP update by emptying and
 * refilling their content, which snaps any scrolled box back to the top.
 * keepScroll(node) is called just before such a rebuild: it remembers the
 * scroll position of the node and of each ancestor, and puts them back once
 * the rebuild has finished (a microtask, so before the next paint).
 *
 * A rebuild can briefly be shorter than the place the player had scrolled to
 * (the Company payload lands before its Company.Inventory, so the Inventory
 * tab shows "Nothing yet" for a moment). The browser then clamps the restore;
 * the wanted position is kept and used again at the next rebuild, as long as
 * the player has not scrolled since.
 */
(function (root, factory) {
    if (typeof module === 'object' && module.exports) { module.exports = factory(); }
    else { root.keepScroll = factory(); }
}(typeof window !== 'undefined' ? window : globalThis, function () {
    // node -> { want, got }: a restore the browser clamped to got.
    var clamped = new WeakMap();

    function keepScroll(node) {
        var saved = [];
        for (var n = node; n; n = n.parentElement) {
            var top = n.scrollTop;
            var c = clamped.get(n);
            if (c && c.got === top) { top = c.want; }
            if (top > 0) { saved.push([n, top]); }
        }
        if (!saved.length) { return; }
        queueMicrotask(function () {
            saved.forEach(function (s) {
                s[0].scrollTop = s[1];
                var got = s[0].scrollTop;
                if (got < s[1]) { clamped.set(s[0], { want: s[1], got: got }); }
                else { clamped.delete(s[0]); }
            });
        });
    }
    return keepScroll;
}));
