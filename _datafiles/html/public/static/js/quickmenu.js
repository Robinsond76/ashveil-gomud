/* global Client, module */
/**
 * quickmenu.js: the quick command menu.
 *
 * Press Enter on an empty command box and a menu opens over the game with
 * what you can do in this room: look, attack, move, get, gather, your
 * company, and more. The arrow keys move through it, Enter chooses, an
 * entry marked with an arrow opens a smaller menu, and every menu ends with
 * Back (or Close at the first level). Escape and Backspace go back too, and
 * the number keys 1-9 choose an entry. Choosing a command sends it as if
 * you had typed it. The menu never does anything the commands cannot do
 * (help quickmenu).
 *
 * With the command box empty the arrow keys walk (up north, down south,
 * left west, right east, the same as the numpad); Alt+Up and Alt+Down
 * recall earlier commands. Typing anything gives the arrows back to the box.
 *
 * Two halves in one file. build(state) is pure: it turns the room, company,
 * camp and battle that GMCP sent into the menu tree, and it is tested under
 * Node (scripts/js/quickmenu.test.mjs). The rest draws the menu and routes
 * the keys.
 *
 * A menu entry is { label, hint?, cmd? | fn? | sub? }: cmd is sent, fn is
 * called, sub (a function returning entries, so it is built from the newest
 * state when opened) opens a submenu. Targets are named by the server's id
 * (look/attack/get <id>), never by display name alone.
 */

'use strict';

(function(root) {

    var FOCI = ['none', 'leader', 'casters', 'healers', 'nearest', 'weakest', 'strongest', 'wounded'];
    var GATHER = {
        herbs:    { label: 'Gather herbs',    cmd: 'gather herbs' },
        firewood: { label: 'Gather firewood', cmd: 'gather firewood' },
        fishing:  { label: 'Go fishing',      cmd: 'fish' },
        game:     { label: 'Hunt game',       cmd: 'hunt' },
        water:    { label: 'Drink from the water here', cmd: 'drink water' },
    };

    function list(v) { return Array.isArray(v) ? v : []; }
    function has(arr, v) { return list(arr).indexOf(v) !== -1; }

    // exitEntries lists the way out of the room, with locked and secret
    // marks the room window shows too.
    function exitEntries(room) {
        var exits = room.exits || {};
        var v2 = room.exitsv2 || {};
        return Object.keys(exits).map(function(dir) {
            var d = (v2[dir] && v2[dir].details) || [];
            var hint = has(d, 'locked') ? 'locked' : (has(d, 'secret') ? 'secret' : '');
            return { label: dir, hint: hint, cmd: dir };
        });
    }

    function targets(verb, mobs) {
        return mobs.map(function(c) { return { label: c.name, cmd: verb + ' ' + c.id }; });
    }

    // foes are the NPCs worth attacking: not your own company (charmed),
    // not a shopkeeper, and not one already down or giving up.
    var NOT_FOES = ['charmed', 'shop', 'downed', 'surrendered'];
    function foes(npcs) {
        return npcs.filter(function(c) {
            return !NOT_FOES.some(function(a) { return has(c.adjectives, a); });
        });
    }

    function battleOn(battle) {
        return !!(battle && Array.isArray(battle.enemies) && battle.enemies.length);
    }

    function battleEntry(battle) {
        return {
            label: 'Battle', sub: function() {
                var out = [{ label: 'Retreat', hint: 'pull the company out', cmd: 'retreat' }];
                if (battle.focus_ready !== false) {
                    FOCI.forEach(function(f) {
                        if (f === 'none') { return; }
                        out.push({ label: 'Focus: ' + f, cmd: 'company tactics focus ' + f });
                    });
                    out.push({ label: 'Focus: none', cmd: 'company tactics focus none' });
                }
                return out;
            },
        };
    }

    function campEntries(camp) {
        var out = [];
        if (camp.can_camp) { out.push({ label: 'Make camp', cmd: 'camp' }); }
        if (camp.has_camp && camp.here && !camp.fire_lit) {
            out.push({ label: camp.embers ? 'Feed the fire' : 'Light the fire', cmd: 'camp fire' });
        }
        if (camp.has_camp && camp.here && camp.fire_lit && !camp.resting) { out.push({ label: 'Rest', cmd: 'camp rest' }); }
        if (camp.has_camp && camp.here && !camp.resting) { out.push({ label: 'Break camp', cmd: 'camp break' }); }
        if (camp.inn) { out.push({ label: 'Inn: price and your stay', cmd: 'inn' }); }
        return out;
    }

    // companyEntry is the company's orders; during a battle only the views,
    // since the battle runs itself (retreat and focus are under Battle).
    function companyEntry(state, inBattle) {
        var camp = state.camp || {};
        return {
            label: 'Company', sub: function() {
                if (inBattle) {
                    return [
                        { label: 'Status', cmd: 'company status' },
                        { label: 'Inventory', cmd: 'company inventory' },
                    ];
                }
                var out = [
                    { label: 'Status', cmd: 'company status' },
                    { label: 'Inventory', cmd: 'company inventory' },
                    { label: 'Meal: everyone eats and drinks', cmd: 'company meal' },
                    { label: 'Patch the wounded', cmd: 'company patch' },
                    { label: 'Scout ahead', cmd: 'scout' },
                ];
                var c = campEntries(camp);
                if (c.length) { out.push({ label: 'Camp', sub: function() { return campEntries(camp); } }); }
                out.push({
                    label: 'Tactics', sub: function() {
                        return FOCI.map(function(f) { return { label: 'Focus: ' + f, cmd: 'company tactics focus ' + f }; });
                    },
                });
                out.push({ label: 'Cargo', cmd: 'cargo' });
                return out;
            },
        };
    }

    var ME = [
        ['Inventory', 'inventory'], ['Status', 'status'], ['Experience', 'experience'],
        ['Quests', 'quests'], ['Skills', 'skills'], ['Effects', 'conditions'], ['Who is online', 'who'],
    ];
    var HELP = [
        ['This menu', 'help quickmenu'], ['Combat', 'help combat'], ['Web client', 'help webclient'],
        ['Adventure: every part of the game', 'help adventure'], ['All help topics', 'help'],
    ];
    function plain(rows) { return rows.map(function(r) { return { label: r[0], cmd: r[1] }; }); }

    // build returns the first level of the menu from the newest state:
    //   { room, camp, battle, places, walking }.
    function build(state) {
        state = state || {};
        var room = state.room || {};
        var contents = room.Contents || {};
        var npcs = list(contents.Npcs);
        var items = list(contents.Items);
        var containers = list(contents.Containers);
        var players = list(contents.Players);
        var exits = exitEntries(room);
        var inBattle = battleOn(state.battle);
        var out = [];

        if (inBattle) { out.push(battleEntry(state.battle)); }

        out.push({
            label: 'Look', sub: function() {
                var l = [{ label: 'Look around', cmd: 'look' }];
                l = l.concat(targets('look', npcs), targets('look', players), items.map(function(i) {
                    return { label: i.label || i.name, cmd: 'look ' + i.id };
                }), containers.map(function(c) { return { label: c.name, cmd: 'look ' + c.name }; }));
                exits.forEach(function(e) { l.push({ label: 'Look ' + e.label, cmd: 'look ' + e.label }); });
                return l;
            },
        });

        var enemies = foes(npcs);
        if (!inBattle && enemies.length) {
            out.push({ label: 'Attack', sub: function() { return targets('attack', enemies); } });
        }

        // During a battle the server refuses walking, shopping, picking up
        // and gathering (only retreat and the focus are orders), so the menu
        // leaves them out.
        if (inBattle) {
            out.push(companyEntry(state, true));
            out.push({ label: 'Me', sub: function() { return plain(ME); } });
            out.push({ label: 'Help', sub: function() { return plain(HELP); } });
            return out;
        }

        if (exits.length || (state.places && state.places.length) || state.walking) {
            out.push({
                label: 'Move', sub: function() {
                    var m = exits.slice();
                    if (state.walking) { m.push({ label: 'Stop walking', cmd: 'walkto stop' }); }
                    var places = list(state.places);
                    if (places.length) {
                        m.push({
                            label: 'Walk to', sub: function() {
                                return places.map(function(p) {
                                    return { label: p.name, hint: p.legend || '', cmd: 'walkto ' + p.id };
                                });
                            },
                        });
                    }
                    m.push({ label: 'Scout ahead', cmd: 'scout' });
                    return m;
                },
            });
        }

        var shops = npcs.filter(function(c) { return has(c.adjectives, 'shop'); });
        var details = room.details || [];
        var services = [];
        shops.forEach(function(c) { services.push({ label: 'Shop: ' + c.name, cmd: 'list ' + c.id }); });
        if (has(details, 'trainer')) { services.push({ label: 'Train', cmd: 'train' }); }
        if (has(details, 'bank')) { services.push({ label: 'Bank', cmd: 'bank' }); }
        campEntries(state.camp || {}).filter(function(e) { return e.cmd === 'inn'; }).forEach(function(e) { services.push(e); });
        if (services.length) { out.push({ label: 'Services', sub: function() { return services; } }); }

        out.push({
            label: 'Get', sub: function() {
                var g = [];
                if (items.length) { g.push({ label: 'Get everything here', cmd: 'get all' }); }
                items.forEach(function(i) { g.push({ label: i.label || i.name, cmd: 'get ' + i.id }); });
                g.push({ label: 'Loot the fallen', hint: 'after a battle', cmd: 'loot' });
                return g;
            },
        });

        // A resource picked clean is left out; it regrows with time.
        var gone = room.depleted || [];
        var gather = [];
        list(room.resources).forEach(function(r) {
            if (GATHER[r] && !has(gone, r)) { gather.push({ label: GATHER[r].label, cmd: GATHER[r].cmd }); }
        });
        if (gather.length) { out.push({ label: 'Gather', sub: function() { return gather; } }); }

        out.push(companyEntry(state, false));
        out.push({ label: 'Me', sub: function() { return plain(ME); } });
        out.push({ label: 'Help', sub: function() { return plain(HELP); } });
        return out;
    }

    var api = { build: build, FOCI: FOCI };
    if (typeof module !== 'undefined' && module.exports) { module.exports = api; }
    if (!root.document) { return; }

    // -----------------------------------------------------------------------
    // The picture and the keys
    // -----------------------------------------------------------------------
    var panel = null;
    var stack = [];       // [{ title, entries, at }], the last is showing
    var rows = [];

    function state() {
        var g = Client.GMCPStructs || {};
        var places = (root.MapPlaces && root.MapPlaces.list && root.MapPlaces.list()) || [];
        return {
            room: g.Room && g.Room.Info,
            camp: g.Company && g.Company.Camp,
            battle: g.Company && g.Company.Battle,
            places: places,
            walking: !!(root.MapPlaces && root.MapPlaces.walking && root.MapPlaces.walking()),
        };
    }

    function isOpen() { return !!panel; }

    function ensureStyle() {
        if (document.getElementById('quickmenu-style')) { return; }
        var s = document.createElement('style');
        s.id = 'quickmenu-style';
        s.textContent =
            '#quickmenu { position: fixed; z-index: 2147483000; min-width: 220px; max-width: min(92vw, 420px); max-height: 70vh; overflow-y: auto;' +
            ' background: var(--t-bg-surface); color: var(--t-text); border: 1px solid var(--t-accent); border-radius: 6px;' +
            ' box-shadow: 0 6px 22px rgba(0,0,0,0.75); font-family: inherit; font-size: 0.9em; padding: 4px 0; }' +
            '#quickmenu .qm-title { padding: 4px 12px 6px; color: var(--t-text-secondary); font-size: 0.85em; letter-spacing: 0.05em; text-transform: uppercase; border-bottom: 1px solid var(--t-border); margin-bottom: 2px; }' +
            '#quickmenu .qm-row { display: flex; align-items: center; gap: 8px; width: 100%; padding: 6px 12px; min-height: 28px; box-sizing: border-box; background: none; border: none; text-align: left; font: inherit; color: inherit; cursor: pointer; }' +
            '#quickmenu .qm-row.sel { background: var(--t-bg-hover); box-shadow: inset 3px 0 0 var(--t-accent); }' +
            '#quickmenu .qm-num { width: 1.1em; color: var(--t-text-secondary); font-size: 0.85em; }' +
            '#quickmenu .qm-label { flex: 1; }' +
            '#quickmenu .qm-hint { color: var(--t-text-secondary); font-size: 0.85em; }' +
            '#quickmenu .qm-more { color: var(--t-accent); }' +
            '#quickmenu .qm-foot { padding: 4px 12px 2px; color: var(--t-text-secondary); font-size: 0.75em; border-top: 1px solid var(--t-border); margin-top: 2px; }' +
            'body.mobile #quickmenu .qm-row { min-height: 44px; }' +
            'body.mobile #quickmenu .qm-foot { display: none; }';
        document.head.appendChild(s);
    }

    function level() { return stack[stack.length - 1]; }

    // entriesOf is a level's entries with its Back or Close entry added.
    function withBack(entries, first) {
        return entries.concat([{ label: first ? 'Close' : 'Back', back: true }]);
    }

    function el(tag, cls, text) {
        var n = document.createElement(tag);
        if (cls) { n.className = cls; }
        if (text !== undefined) { n.textContent = text; }
        return n;
    }

    // place sits the menu over the game text, just above the command box
    // (over the box's own column when the text is out of sight).
    function place() {
        var input = document.getElementById('command-input');
        var ir = input ? input.getBoundingClientRect() : { left: 8, top: root.innerHeight };
        var term = document.getElementById('terminal');
        var tr = term ? term.getBoundingClientRect() : null;
        var left = (tr && tr.width > panel.offsetWidth + 16) ? tr.left + 12 : ir.left;
        panel.style.left = Math.max(8, Math.min(left, root.innerWidth - panel.offsetWidth - 8)) + 'px';
        // On a phone the touch bar sits above the box; stay clear of it.
        var bar = document.getElementById('touch-bar');
        var top = ir.top;
        if (bar && bar.offsetParent !== null) { top = Math.min(top, bar.getBoundingClientRect().top); }
        panel.style.bottom = Math.max(8, root.innerHeight - top + 6) + 'px';
        panel.style.top = '';
    }

    function render() {
        var lv = level();
        panel.textContent = '';
        panel.appendChild(el('div', 'qm-title', lv.title));
        rows = [];
        lv.entries.forEach(function(e, i) {
            var b = el('button', 'qm-row' + (i === lv.at ? ' sel' : ''));
            b.type = 'button';
            b.setAttribute('role', 'menuitem');
            b.tabIndex = -1;
            b.appendChild(el('span', 'qm-num', i < 9 ? String(i + 1) : ''));
            b.appendChild(el('span', 'qm-label', e.label));
            if (e.hint) { b.appendChild(el('span', 'qm-hint', e.hint)); }
            if (e.sub) { b.appendChild(el('span', 'qm-more', '▸')); }
            b.addEventListener('mouseenter', function() { lv.at = i; mark(); });
            b.addEventListener('mousedown', function(ev) { ev.preventDefault(); });
            b.addEventListener('click', function() { lv.at = i; choose(); });
            rows.push(b);
            panel.appendChild(b);
        });
        panel.appendChild(el('div', 'qm-foot', '↑↓ move · Enter choose · Esc back'));
        place();
        mark();
    }

    function mark() {
        var lv = level();
        rows.forEach(function(b, i) {
            b.classList.toggle('sel', i === lv.at);
            if (i === lv.at && b.scrollIntoView) { b.scrollIntoView({ block: 'nearest' }); }
        });
    }

    function close() {
        if (!panel) { return; }
        panel.remove();
        panel = null;
        stack = [];
        rows = [];
        var input = document.getElementById('command-input');
        if (input) { input.focus(); }
    }

    function push(title, entries) {
        stack.push({ title: title, entries: withBack(entries, stack.length === 0), at: 0 });
        render();
    }

    function back() {
        if (stack.length <= 1) { close(); return; }
        stack.pop();
        render();
    }

    function choose() {
        var lv = level();
        var e = lv.entries[lv.at];
        if (!e) { return; }
        if (e.back) { back(); return; }
        if (e.sub) { push(e.label, e.sub()); return; }
        close();
        if (typeof e.fn === 'function') { e.fn(); return; }
        if (e.cmd) { Client.SendInput(e.cmd); }
    }

    function open() {
        if (panel) { return; }
        ensureStyle();
        panel = el('div');
        panel.id = 'quickmenu';
        panel.setAttribute('role', 'menu');
        panel.setAttribute('aria-label', 'Quick commands');
        document.body.appendChild(panel);
        // Take the focus off the command box while the menu is up: Firefox
        // opens its saved-logins dropdown ("Manage passwords") on the arrow
        // keys in a focused text box, and the menu needs those keys.
        var box = document.getElementById('command-input');
        if (box && document.activeElement === box) { box.blur(); }
        stack = [];
        push('Quick commands', build(state()));
    }

    function move(by) {
        var lv = level();
        var n = lv.entries.length;
        lv.at = (lv.at + by + n) % n;
        mark();
    }

    // Keys while the menu is up are the menu's alone.
    function menuKey(e) {
        var k = e.key;
        if (k === 'ArrowDown') { move(1); }
        else if (k === 'ArrowUp') { move(-1); }
        else if (k === 'Home') { level().at = 0; mark(); }
        else if (k === 'End') { level().at = level().entries.length - 1; mark(); }
        else if (k === 'Enter' || k === 'ArrowRight') { choose(); }
        else if (k === 'ArrowLeft' || k === 'Backspace') { back(); }
        else if (k === 'Escape') { back(); }
        else if (k >= '1' && k <= '9' && level().entries[Number(k) - 1]) { level().at = Number(k) - 1; choose(); }
        else if (k === 'Tab' || k === 'Shift') { return false; }
        else { return false; }
        return true;
    }

    // typingReady: the command box has the focus, is empty and is not a
    // password prompt, and the player is in play (not logging in, and no
    // question such as a yes/no waits, where a blank Enter takes its
    // default), so Enter is free to open the menu.
    function typingReady(ev) {
        var t = ev.target;
        return t && t.id === 'command-input' && t.value === '' && t.type !== 'password' &&
            !!(Client.Playing && Client.Playing());
    }

    document.addEventListener('keydown', function(ev) {
        if (isOpen()) {
            if (menuKey(ev)) { ev.preventDefault(); ev.stopPropagation(); }
            return;
        }
        if (ev.key === 'Enter' && !ev.ctrlKey && !ev.altKey && !ev.metaKey && !ev.shiftKey && typingReady(ev)) {
            ev.preventDefault();
            ev.stopPropagation();
            open();
        }
    }, true);

    // A click or tap outside closes the menu.
    document.addEventListener('mousedown', function(ev) {
        if (panel && !panel.contains(ev.target)) { close(); }
    }, true);

    root.QuickMenu = { open: open, close: close, isOpen: isOpen, build: build };
}(typeof window !== 'undefined' ? window : globalThis));
