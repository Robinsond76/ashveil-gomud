/*
 * mobile.js (Phase 40i)
 *
 * The phone layout of the web client. On a narrow screen (820 px or less)
 * the page shows one view at a time: the game text, the map, the room, or the
 * company dock, chosen from a bottom bar, with the command box and a touch
 * bar (compass, common commands, "Walk to...") always in reach. Nothing here
 * changes what the server sends: every control sends an ordinary command, and
 * the views are the same windows the desktop dock shows (the layout is CSS in
 * mobile.css; a panel's data-win attribute says which view owns it).
 *
 * Mobile.view() / Mobile.show(id) / Mobile.active() are for the browser
 * checks (scripts/browser/mobile-check.mjs).
 */
(function () {
    'use strict';

    var QUERY = '(max-width: 820px)';
    var VIEWS = [
        { id: 'game',    label: 'Game' },
        { id: 'map',     label: 'Map' },
        { id: 'here',    label: 'Here' },
        { id: 'company', label: 'Company' },
    ];
    var COMPASS = [
        { label: 'N', cmd: 'north', title: 'Go north' },
        { label: 'S', cmd: 'south', title: 'Go south' },
        { label: 'E', cmd: 'east',  title: 'Go east' },
        { label: 'W', cmd: 'west',  title: 'Go west' },
        { label: 'U', cmd: 'up',    title: 'Go up' },
        { label: 'D', cmd: 'down',  title: 'Go down' },
    ];
    var QUICK = [
        { label: 'Look',      cmd: 'look',      title: 'Look around' },
        { label: 'Inventory', cmd: 'inventory', title: 'What you carry' },
        { label: 'Camp',      cmd: 'camp',      title: 'The camp: supplies, gear and rest' },
    ];
    var BAR_KEY = 'ashveil-touch-bar';

    var mq = window.matchMedia(QUERY);
    var built = false;
    var current = 'game';
    var nav, bar, stopBtn, walkBtn, foldBtn, navBtns = {};

    function store(key, value) {
        try { if (value === undefined) { return localStorage.getItem(key); } localStorage.setItem(key, value); } catch (e) { /* private mode */ }
        return null;
    }

    function el(tag, cls, text) {
        var n = document.createElement(tag);
        if (cls) { n.className = cls; }
        if (text !== undefined) { n.textContent = text; }
        return n;
    }

    function button(label, title, onClick, cls) {
        var b = el('button', cls || 'mb-btn', label);
        b.type = 'button';
        if (title) { b.title = title; b.setAttribute('aria-label', title); }
        b.addEventListener('click', onClick);
        return b;
    }

    function send(cmd, echo) { if (Client.SendInput) { Client.SendInput(cmd, echo); } }

    // "Walk to..." lists the named places the map shows, nearest first, and
    // sends one ordinary walkto for the pick (Phase 40d's click-to-walk).
    function walkMenu(ev) {
        var places = (window.MapPlaces && MapPlaces.list()) || [];
        var items = [];
        if (window.MapPlaces && MapPlaces.walking()) { items.push({ label: 'Stop walking', cmd: 'walkto stop' }); }
        places.forEach(function (p) {
            items.push({ label: p.name + ' (' + p.legend + ')', cmd: 'walkto ' + p.id, echo: 'walkto ' + p.name });
        });
        if (places.length === 0) {
            items.push({ label: 'No named places on this map yet. Tap a room on the Map instead (help walkto)', cmd: 'help walkto' });
        }
        // Phase 47: "Find a visited room..." types part of a name and lists
        // the rooms the map has seen that match, nearest first.
        if (window.MapPlaces && MapPlaces.search) {
            items.push({ label: 'Find a visited room\u2026', fn: searchRooms });
        }
        window.uiMenu(ev, items);
    }

    // A sheet with a search box over every room the map has seen (named
    // places and plain rooms alike), nearest first; a pick sends walkto.
    function searchRooms() {
        var old = document.getElementById('walk-search');
        if (old) { old.remove(); }
        var sheet = el('div');
        sheet.id = 'walk-search';
        sheet.setAttribute('role', 'dialog');
        sheet.setAttribute('aria-label', 'Find a visited room');
        var input = el('input');
        input.type = 'search';
        input.placeholder = 'Search visited rooms';
        input.setAttribute('aria-label', 'Search visited rooms');
        var list = el('div', 'ws-list');
        var close = button('Close', 'Close the search', function () { sheet.remove(); }, 'mb-btn ws-close');
        function paint() {
            list.textContent = '';
            var found = MapPlaces.search(input.value);
            found.forEach(function (p) {
                var b = button(p.name + (p.legend ? ' (' + p.legend + ')' : ''), 'Walk to ' + p.name, function () {
                    sheet.remove();
                    send('walkto ' + p.id, 'walkto ' + p.name);
                }, 'ws-item');
                list.appendChild(b);
            });
            if (found.length === 0) { list.appendChild(el('div', 'ws-empty', 'No visited room matches.')); }
        }
        input.addEventListener('input', paint);
        var head = el('div', 'ws-head');
        head.appendChild(input);
        head.appendChild(close);
        sheet.appendChild(head);
        sheet.appendChild(list);
        document.body.appendChild(sheet);
        paint();
        input.focus();
    }

    function build() {
        if (built) { return; }
        built = true;

        nav = el('nav');
        nav.id = 'mobile-nav';
        nav.setAttribute('aria-label', 'Views');
        VIEWS.forEach(function (v) {
            var b = button(v.label, '', function () { show(v.id); }, 'mn-btn');
            b.dataset.view = v.id;
            navBtns[v.id] = b;
            nav.appendChild(b);
        });
        nav.appendChild(button('⚙', 'Settings', function () { Client.toggleMenu(); }, 'mn-btn mn-settings'));

        bar = el('div');
        bar.id = 'touch-bar';
        bar.setAttribute('role', 'toolbar');
        bar.setAttribute('aria-label', 'Touch commands');
        foldBtn = button('▾', 'Hide the touch bar', function () { fold(!bar.classList.contains('folded')); }, 'mb-btn mb-fold');
        // Two rows, so every control is in sight on a 360 px screen: the
        // compass, then the walk list and the commands that answer in text.
        var rows = el('div', 'tb-rows');
        var row = el('div', 'tb-row');
        row.appendChild(foldBtn);
        COMPASS.forEach(function (c) {
            row.appendChild(button(c.label, c.title, function () { send(c.cmd); }, 'mb-btn mb-dir'));
        });
        rows.appendChild(row);
        row = el('div', 'tb-row');
        walkBtn = button('Walk to…', 'Walk to a named place (walkto)', walkMenu, 'mb-btn mb-walk');
        row.appendChild(walkBtn);
        stopBtn = button('Stop', 'Stop walking (walkto stop)', function () { send('walkto stop'); }, 'mb-btn mb-stop');
        stopBtn.hidden = true;
        row.appendChild(stopBtn);
        row.appendChild(button('Menu', 'Quick commands for this room (help quickmenu)', function () {
            if (window.QuickMenu) { QuickMenu.open(); }
        }, 'mb-btn mb-menu'));
        QUICK.forEach(function (c) {
            // These answer in the game text, so bring it to the front.
            row.appendChild(button(c.label, c.title, function () { send(c.cmd); show('game'); }));
        });
        rows.appendChild(row);
        bar.appendChild(rows);

        var input = document.getElementById('input-area');
        input.parentNode.insertBefore(bar, input);
        input.parentNode.insertBefore(nav, input.nextSibling);

        if (store(BAR_KEY) === 'folded') { fold(true); }
        window.addEventListener('resize', measureBars);
        // The walk in progress (Walkto) shows a Stop button beside the list.
        VirtualWindows.register({
            gmcpHandlers: ['Walkto'],
            onGMCP: function () { paintWalk(); },
        });
        paintNav();
        watchDocks();
    }

    // The height of the bars at the bottom (touch bar, command box, view
    // bar), so the battle badge can sit clear of them (mobile.css).
    function measureBars() {
        if (!built) { return; }
        var top = Math.min(bar.getBoundingClientRect().top, document.getElementById('input-area').getBoundingClientRect().top);
        document.body.style.setProperty('--mobile-bars', Math.max(0, Math.round(window.innerHeight - top)) + 'px');
    }

    function fold(folded) {
        bar.classList.toggle('folded', folded);
        foldBtn.textContent = folded ? '▴' : '▾';
        foldBtn.title = folded ? 'Show the touch bar' : 'Hide the touch bar';
        foldBtn.setAttribute('aria-label', foldBtn.title);
        store(BAR_KEY, folded ? 'folded' : 'open');
        measureBars();
        window.dispatchEvent(new Event('resize'));
    }

    function paintWalk() {
        if (!stopBtn) { return; }
        var w = Client.GMCPStructs && Client.GMCPStructs.Walkto;
        stopBtn.hidden = !(w && Array.isArray(w.path) && w.path.length > 0);
    }

    function paintNav() {
        VIEWS.forEach(function (v) {
            var on = v.id === current;
            navBtns[v.id].classList.toggle('active', on);
            navBtns[v.id].setAttribute('aria-pressed', on ? 'true' : 'false');
        });
    }

    // Phase 47: a view owns panels by window id, wherever the player docked
    // them on desktop. Map and Here claim theirs; the Company view takes the
    // rest (the tab groups and any panel dragged out of them). A dock shows
    // when it holds a panel of the front view; two docks share the screen.
    var OWNS = { map: ['Map'], here: ['RoomInfo', 'Time & Date', 'Tutorial'] };
    function viewOf(win) {
        if (OWNS.map.indexOf(win) >= 0) { return 'map'; }
        if (OWNS.here.indexOf(win) >= 0) { return 'here'; }
        return 'company';
    }

    function layoutDocks() {
        var shown = [];
        var docks = [document.getElementById('dock-left'), document.getElementById('dock-right')].filter(Boolean);
        docks.forEach(function (d) {
            var has = false;
            d.querySelectorAll('.dock-panel').forEach(function (p) {
                var mine = viewOf(p.dataset.win || '') === current;
                p.classList.toggle('m-off', !mine);
                if (mine) { has = true; }
            });
            has = has && d.classList.contains('has-panels');
            d.dataset.mshow = has ? '1' : '0';
            if (has) { shown.push(d); }
        });
        docks.forEach(function (d) { d.dataset.msplit = shown.length > 1 ? '1' : '0'; });
    }

    function watchDocks() {
        if (!window.MutationObserver) { return; }
        var observer = new MutationObserver(function () { if (mq.matches) { layoutDocks(); } });
        ['dock-left', 'dock-right'].forEach(function (id) {
            var d = document.getElementById(id);
            if (d) { observer.observe(d, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-win', 'class'] }); }
        });
    }

    function show(id) {
        if (!VIEWS.some(function (v) { return v.id === id; })) { return; }
        current = id;
        document.body.dataset.mview = id;
        if (built) { paintNav(); layoutDocks(); }
        // The map and the terminal measure themselves when they become visible.
        window.dispatchEvent(new Event('resize'));
        if (id === 'game') {
            var term = document.getElementById('terminal');
            if (term) { term.scrollTop = 0; }
        }
    }

    function apply() {
        var on = mq.matches;
        document.body.classList.toggle('mobile', on);
        if (on) {
            build();
            show(current);
        } else {
            delete document.body.dataset.mview;
        }
        window.dispatchEvent(new Event('resize'));
    }

    function start() {
        apply();
        if (mq.addEventListener) { mq.addEventListener('change', apply); } else if (mq.addListener) { mq.addListener(apply); }
    }

    window.Mobile = {
        active: function () { return document.body.classList.contains('mobile'); },
        view:   function () { return current; },
        show:   show,
        layout: layoutDocks,
    };

    // Client.init() mounts the terminal and docks on body load; the layout
    // needs them in place, so start after it.
    window.addEventListener('load', function () { setTimeout(start, 0); });
}());
