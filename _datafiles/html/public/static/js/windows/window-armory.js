/**
 * window-armory.js
 *
 * The admin test area's armory catalog: a screen listing every item of the
 * world with a search box, type filters and a Take button, so an admin picks
 * what to carry instead of taking a whole kit. Opened by `testarea catalog`.
 *
 * Responds to GMCP namespace:
 *   Armory - { filter, items: [{id, name, type, subtype, family, tier}] }
 *
 * Shown inside GameModal (window-modal.js). Every server string is set with
 * textContent. Take sends `testarea give <id> [count]`.
 */

/* global Client, VirtualWindows, GameModal, injectStyles, module */

'use strict';

(function() {

    if (typeof injectStyles === 'function') { injectStyles(`
                #armory .arm-bar { display: flex; flex-wrap: wrap; gap: 6px; padding-bottom: 8px; }
        #armory .arm-search { flex: 1 1 160px; min-width: 0; min-height: 36px; box-sizing: border-box;
            background: var(--t-bg); color: var(--t-text); border: 1px solid var(--t-border); border-radius: 4px; padding: 4px 8px; font: inherit; }
        #armory .arm-count { min-height: 36px; background: var(--t-bg); color: var(--t-text); border: 1px solid var(--t-border); border-radius: 4px; font: inherit; }
        #armory .arm-types { display: flex; flex-wrap: wrap; gap: 4px; padding-bottom: 8px; }
        #armory .arm-chip { min-height: 32px; padding: 2px 10px; background: var(--t-bg-surface); color: var(--t-text-secondary);
            border: 1px solid var(--t-border); border-radius: 14px; cursor: pointer; font: inherit; font-size: 0.9em; }
        #armory .arm-chip.active { background: var(--t-bg); color: var(--t-text); border-color: var(--t-accent); }
        #armory .arm-list { max-height: 52vh; overflow-y: auto; border-top: 1px solid var(--t-border); }
        #armory .arm-row { display: flex; align-items: center; gap: 8px; padding: 4px 2px; border-bottom: 1px solid var(--t-border); }
        #armory .arm-info { flex: 1; min-width: 0; display: flex; flex-wrap: wrap; align-items: baseline; column-gap: 8px; }
        #armory .arm-name { flex: 1 1 10em; min-width: 0; overflow-wrap: anywhere; }
        #armory .arm-meta { color: var(--t-text-secondary); font-size: 0.85em; }
        #armory .arm-take { min-height: 36px; min-width: 56px; background: var(--t-bg-surface); color: var(--t-text);
            border: 1px solid var(--t-border-accent); border-radius: 4px; cursor: pointer; font: inherit; }
        #armory .arm-status { padding-top: 6px; color: var(--t-text-secondary); font-size: 0.9em; min-height: 1.4em; }
        body.mobile #armory .arm-take, body.mobile #armory .arm-chip { min-height: 44px; }
        @media (max-width: 600px) {
            #armory .arm-types { flex-wrap: nowrap; overflow-x: auto; -webkit-overflow-scrolling: touch; }
            #armory .arm-chip { flex: 0 0 auto; }
            #armory .arm-name { flex-basis: 100%; }
        }
    `); }

    const LIMIT = 150; // rows drawn at once; the filters narrow the rest

    // filterCatalog is the screen's logic: the type chip and the search
    // words (all must match the name, type, subtype or family).
    function filterCatalog(items, type, text) {
        const words = String(text || '').toLowerCase().split(/\s+/).filter(Boolean);
        return items.filter(function(it) {
            if (type && it.type !== type) { return false; }
            const hay = (it.name + ' ' + it.type + ' ' + (it.subtype || '') + ' ' + (it.family || '')).toLowerCase();
            return words.every(function(w) { return hay.indexOf(w) !== -1; });
        });
    }

    function el(tag, cls, text) {
        const e = document.createElement(tag);
        if (cls) { e.className = cls; }
        if (text !== undefined) { e.textContent = text; }
        return e;
    }

    function openArmory(payload) {
        const items = (payload && payload.items) || [];
        const types = Array.from(new Set(items.map(function(i) { return i.type; }))).sort();
        let type = '';

        GameModal.open({ title: 'Armory catalog', body: '', format: 'html' });
        const host = document.getElementById('game-modal-html-container');
        if (!host) { return; }
        host.textContent = '';
        const root = el('div');
        root.id = 'armory';

        const bar = el('div', 'arm-bar');
        const search = el('input', 'arm-search');
        search.type = 'search';
        search.name = 'armory-filter';
        search.placeholder = 'Search ' + items.length + ' items';
        search.autocomplete = 'off';
        search.setAttribute('aria-label', 'Search the catalog');
        search.value = (payload && payload.filter) || '';
        const count = el('select', 'arm-count');
        count.setAttribute('aria-label', 'How many to take');
        [1, 5, 10, 20].forEach(function(n) {
            const o = el('option', null, 'x' + n);
            o.value = String(n);
            count.appendChild(o);
        });
        bar.appendChild(search);
        bar.appendChild(count);

        const chips = el('div', 'arm-types');
        const list = el('div', 'arm-list');
        const status = el('div', 'arm-status');
        status.setAttribute('aria-live', 'polite');

        // A phone keeps its keyboard down until the search box is tapped.
        function focusSearch() {
            const touch = document.body.classList.contains('mobile') ||
                (window.matchMedia && window.matchMedia('(pointer: coarse)').matches);
            if (!touch) { search.focus(); }
        }

        function chip(label, value) {
            const b = el('button', 'arm-chip', label);
            b.type = 'button';
            b.dataset.type = value;
            b.addEventListener('click', function() { type = value; draw(); focusSearch(); });
            return b;
        }

        function draw() {
            chips.querySelectorAll('.arm-chip').forEach(function(c) {
                c.classList.toggle('active', c.dataset.type === type);
            });
            const shown = filterCatalog(items, type, search.value);
            list.textContent = '';
            shown.slice(0, LIMIT).forEach(function(it) {
                const row = el('div', 'arm-row');
                const info = el('div', 'arm-info');
                info.appendChild(el('span', 'arm-name', it.name));
                const meta = it.type + (it.family ? ', ' + it.family : '') + (it.tier ? ', tier ' + it.tier : '') + ' #' + it.id;
                info.appendChild(el('span', 'arm-meta', meta));
                row.appendChild(info);
                const take = el('button', 'arm-take', 'Take');
                take.type = 'button';
                take.setAttribute('aria-label', 'Take ' + it.name);
                take.addEventListener('click', function() {
                    const n = count.value;
                    Client.SendInput('testarea give ' + it.id + ' ' + n);
                    status.textContent = 'Took ' + n + ' x ' + it.name + '.';
                });
                row.appendChild(take);
                list.appendChild(row);
            });
            if (shown.length === 0) {
                list.appendChild(el('div', 'arm-status', 'Nothing matches.'));
            } else if (shown.length > LIMIT) {
                list.appendChild(el('div', 'arm-status', 'Showing ' + LIMIT + ' of ' + shown.length + '; search or pick a type to narrow.'));
            }
        }

        chips.appendChild(chip('All', ''));
        types.forEach(function(t) { chips.appendChild(chip(t, t)); });
        search.addEventListener('input', draw);
        root.appendChild(bar);
        root.appendChild(chips);
        root.appendChild(list);
        root.appendChild(status);
        host.appendChild(root);
        draw();
        focusSearch();
    }

    if (typeof window !== 'undefined') { window.Armory = { filterCatalog: filterCatalog, open: openArmory }; }
    if (typeof module !== 'undefined' && module.exports) { module.exports = { filterCatalog: filterCatalog }; }

    if (typeof VirtualWindows !== 'undefined' && typeof document !== 'undefined') {
        document.addEventListener('DOMContentLoaded', function() {
            VirtualWindows.register({
                window: null,
                gmcpHandlers: ['Armory'],
                onGMCP: function(namespace, payload) {
                    if (payload) { openArmory(payload); }
                },
            });
        });
    }
})();
