/**
 * window-bestiary.js
 *
 * Virtual window: Bestiary (Phase 66) - the company dock's Bestiary tab.
 * It lists every kind of creature the leader has beaten, by zone; each
 * entry opens to the lines its tier has earned: lore at the first kill,
 * defences at the third, habits and weaknesses at the sixth (a boss
 * teaches faster). It is the same text as the `bestiary` command.
 *
 * Responds to GMCP namespace:
 *   Char.Bestiary - the entries (asked for when the tab opens, refreshed
 *                   by the server when a battle ends)
 *
 * Reads: Client.GMCPStructs.Char.Bestiary
 */

'use strict';

(function() {

    injectStyles(`
        #bs-window {
            height: 100%;
            overflow-y: auto;
            background: var(--t-bg);
            color: var(--t-text);
            font-size: 0.78em;
        }
        #bs-window .bs-zone {
            padding: 3px 8px;
            background: var(--t-bg-col-header);
            border-bottom: 1px solid var(--t-border);
            color: var(--t-text-heading);
            font-size: 0.8em;
            text-transform: uppercase;
            letter-spacing: 0.07em;
        }
        #bs-window details { border-bottom: 1px solid var(--t-border-faint); }
        #bs-window summary {
            display: flex;
            align-items: baseline;
            gap: 6px;
            padding: 5px 8px;
            cursor: pointer;
        }
        #bs-window summary .bs-name { flex: 1; min-width: 0; overflow-wrap: anywhere; }
        #bs-window summary .bs-meta { color: var(--t-text-secondary); font-size: 0.9em; white-space: nowrap; }
        #bs-window .bs-tier { color: var(--t-accent); font-size: 0.9em; white-space: nowrap; }
        #bs-window .bs-body { padding: 2px 10px 8px 14px; }
        #bs-window .bs-head {
            margin: 6px 0 2px;
            color: var(--t-text-heading);
            font-size: 0.85em;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        #bs-window .bs-line { margin: 2px 0; line-height: 1.35; overflow-wrap: anywhere; }
        #bs-window .bs-progress { color: var(--t-text-secondary); font-style: italic; margin-top: 6px; }
        #bs-window .bs-empty {
            padding: 16px 10px;
            color: var(--t-text-dim);
            font-style: italic;
            text-align: center;
        }
    `);

    function createDOM() {
        const el = document.createElement('div');
        el.id = 'bs-window';
        document.body.appendChild(el);
        return el;
    }

    function node(tag, cls, text) {
        const n = document.createElement(tag);
        if (cls) { n.className = cls; }
        if (text !== undefined) { n.textContent = text; }
        return n;
    }

    // Which entries are open survives a refresh (a battle ending rebuilds
    // the list).
    const opened = new Set();

    function section(body, title, lines) {
        if (!lines || !lines.length) { return; }
        body.appendChild(node('div', 'bs-head', title));
        lines.forEach(l => body.appendChild(node('div', 'bs-line', l)));
    }

    function entryNode(e) {
        const d = document.createElement('details');
        if (opened.has(e.id)) { d.open = true; }
        d.addEventListener('toggle', () => { if (d.open) { opened.add(e.id); } else { opened.delete(e.id); } });
        const s = document.createElement('summary');
        s.appendChild(node('span', 'bs-name', e.name + (e.boss ? ' (boss)' : '')));
        s.appendChild(node('span', 'bs-meta', 'L' + e.level + ' · ' + e.kills + (e.kills === 1 ? ' kill' : ' kills')));
        s.appendChild(node('span', 'bs-tier', e.tier_name));
        d.appendChild(s);
        const body = node('div', 'bs-body');
        section(body, 'Lore', e.lore);
        section(body, 'Defences', e.defences);
        section(body, 'Habits and weaknesses', e.habits);
        body.appendChild(node('div', 'bs-progress', e.progress.charAt(0).toUpperCase() + e.progress.slice(1) + '.'));
        d.appendChild(body);
        return d;
    }

    function update() {
        const root = document.getElementById('bs-window');
        const data = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Bestiary;
        if (!root || !data) { return; }
        keepScroll(root);
        root.innerHTML = '';
        const entries = data.entries || [];
        if (!entries.length) {
            root.appendChild(node('div', 'bs-empty',
                'Your bestiary is empty. Beat a creature and it is entered: lore at the first kill, defences at the third, habits at the sixth.'));
            return;
        }
        let zone = null;
        entries.forEach(e => {
            const z = e.zone || 'Elsewhere';
            if (z !== zone) {
                zone = z;
                root.appendChild(node('div', 'bs-zone', z));
            }
            root.appendChild(entryNode(e));
        });
    }

    const win = new VirtualWindow('Bestiary', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  290,
        tabGroup:      'dock',
        tabLabel:      'Bestiary',
        onTabShown() { Client.GMCPRequest('Char.Bestiary'); },
        factory() {
            const el = createDOM();
            Client.GMCPRequest('Char.Bestiary');
            requestAnimationFrame(function() { update(); });
            return {
                title:      'Bestiary',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          'right',
                y:          450,
                width:      363,
                height:     20 + 290,
                header:     20,
                bottom:     60,
            };
        },
    });

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Bestiary'],
        onGMCP(namespace) {
            if (namespace.indexOf('Char.Bestiary') === 0) { update(); }
        },
    });

})();
