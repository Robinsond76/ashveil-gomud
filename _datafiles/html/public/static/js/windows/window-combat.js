/**
 * window-combat.js
 *
 * Virtual window: Combat, the company dock's third tab (Phase 32g). Its
 * Setup view: how the company stands and how each member fights, set
 * before a battle (a battle plays out on its own; help strategy).
 *
 *   - The formation grid, row 1 the front, each member with their role.
 *   - A row per member: role and target rule. Clicking a member opens a
 *     menu that sends real commands: strategy <who> <role>,
 *     strategy <who> target <rule>, strategy <who> default, and
 *     formation move / swap / clear. <who> is "me" for the player and
 *     "#<id>" for a companion, which both commands accept.
 *   - Scout (sends "scout") when the room holds an enemy group or a
 *     hostile creature.
 *
 * The live battle view (both sides' formations, targets) is Phase 32g2.
 * Every name is set with textContent, never innerHTML.
 *
 * Responds to GMCP namespaces:
 *   Company   - the snapshot: members, cells, strategies
 *   Room      - Room.Info.Contents.Npcs, for Scout
 */

'use strict';

(function() {

    const el = CompanyData.el;

    const ROLES = ['fighter', 'healer', 'caster'];
    // Target rules, in the order `help strategy` lists them; assist is for
    // companions only.
    const RULES = ['weakest', 'strongest', 'wounded', 'nearest', 'furthest', 'leader', 'assist', 'defend'];

    injectStyles(`
        #combat-window {
            height: 100%;
            overflow-y: auto;
            background: var(--t-bg);
            padding: 4px 6px;
            box-sizing: border-box;
            display: flex;
            flex-direction: column;
            gap: 6px;
            font-size: 0.8em;
        }

        .cbt-note { color: var(--t-text-secondary); font-style: italic; }

        .cbt-grid {
            border-collapse: collapse;
            table-layout: fixed;
            width: 100%;
        }

        .cbt-grid caption { text-align: left; color: var(--t-text-secondary); padding-bottom: 2px; }
        .cbt-grid th { width: 3.4em; font-weight: normal; color: var(--t-text-secondary); text-align: left; }

        .cbt-grid td {
            border: 1px solid var(--t-accent-dim);
            height: 2.8em;
            padding: 2px;
            text-align: center;
            vertical-align: middle;
            overflow-wrap: anywhere;
            line-height: 1.2;
        }

        .cbt-grid td.empty { color: var(--t-text-dim); }
        .cbt-grid td.is-leader { color: var(--t-party-leader); font-weight: bold; }
        .cbt-grid .cbt-role { display: block; font-weight: normal; font-size: 0.85em; color: var(--t-text-secondary); }

        .cbt-members { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }

        .cbt-member {
            display: grid;
            grid-template-columns: minmax(0, 1fr) auto;
            gap: 6px;
            width: 100%;
            text-align: left;
            font: inherit;
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            padding: 3px 5px;
            cursor: pointer;
        }

        .cbt-member:hover { background: var(--t-accent-dim); color: var(--t-text-white); }
        .cbt-member:focus-visible { outline: 2px solid var(--t-accent); }
        .cbt-member .cbt-how { color: var(--t-text-secondary); white-space: nowrap; }
        .cbt-member.is-fallen { opacity: 0.6; border-style: dashed; }

        .cbt-actions { display: flex; flex-wrap: wrap; gap: 4px; }

        .cbt-btn {
            font: inherit;
            padding: 3px 9px;
            border: 1px solid var(--t-btn-border, var(--t-accent-dim));
            border-radius: 3px;
            background: var(--t-bg-surface);
            color: var(--t-text);
            cursor: pointer;
        }

        .cbt-btn:hover { background: var(--t-accent-dim); color: var(--t-text-white); }
    `);

    function createDOM() {
        const root = el('div');
        root.id = 'combat-window';
        root.appendChild(el('div', 'cbt-note', 'No company yet.'));
        document.body.appendChild(root);
        return root;
    }

    const win = new VirtualWindow('Combat', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        factory() {
            const root = createDOM();
            setTimeout(update, 0);
            return {
                title:      'Combat',
                mount:      root,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     320,
                header:     20,
                bottom:     60,
            };
        },
    });

    // who names a member to `strategy` and `formation`.
    function who(m) {
        return m.key === 'leader' ? 'me' : '#' + m.id;
    }

    function howText(m) {
        const s = m.strategy;
        return s ? s.role + ', ' + s.target : '';
    }

    function memberMenu(m, members) {
        const items = [];
        const s = m.strategy || {};
        ROLES.filter(r => r !== s.role).forEach(r => {
            items.push({ label: 'Role: ' + r, cmd: 'strategy ' + who(m) + ' ' + r });
        });
        RULES.filter(r => r !== s.target && !(r === 'assist' && m.key === 'leader')).forEach(r => {
            items.push({ label: 'Target: ' + r, cmd: 'strategy ' + who(m) + ' target ' + r });
        });
        items.push({ label: 'Back to the default', cmd: 'strategy ' + who(m) + ' default' });
        const taken = {};
        members.forEach(x => { if (x.cell) { taken[x.cell.row + ',' + x.cell.col] = x; } });
        for (let r = 0; r < 3; r++) {
            for (let c = 0; c < 3; c++) {
                const here = taken[r + ',' + c];
                if (here === m) { continue; }
                if (here) {
                    items.push({ label: 'Swap with ' + here.name, cmd: 'formation swap ' + who(m) + ' ' + who(here) });
                } else {
                    items.push({ label: 'Move to row ' + (r + 1) + ', column ' + (c + 1), cmd: 'formation move ' + who(m) + ' ' + (r + 1) + ' ' + (c + 1) });
                }
            }
        }
        if (m.cell) { items.push({ label: 'Take out of the formation', cmd: 'formation clear ' + who(m) }); }
        return items;
    }

    function grid(members) {
        const cells = [[null, null, null], [null, null, null], [null, null, null]];
        members.forEach(m => {
            const c = m.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) { cells[c.row][c.col] = m; }
        });
        const table = el('table', 'cbt-grid');
        table.appendChild(el('caption', null, 'Formation (row 1 is the front)'));
        cells.forEach((row, r) => {
            const tr = el('tr');
            const th = el('th', null, 'Row ' + (r + 1));
            th.setAttribute('scope', 'row');
            tr.appendChild(th);
            row.forEach(m => {
                const td = el('td', m ? (m.key === 'leader' ? 'is-leader' : '') : 'empty', m ? m.name : '·');
                if (m) {
                    td.title = m.name + (howText(m) ? ': ' + howText(m) : '');
                    if (m.strategy) { td.appendChild(el('span', 'cbt-role', m.strategy.role)); }
                } else {
                    td.setAttribute('aria-label', 'empty');
                }
                tr.appendChild(td);
            });
            table.appendChild(tr);
        });
        return table;
    }

    function hostileHere() {
        const room = Client.GMCPStructs.Room && Client.GMCPStructs.Room.Info;
        const npcs = room && room.Contents && room.Contents.Npcs;
        return Array.isArray(npcs) && npcs.some(n => n && (n.group || n.aggro));
    }

    function update() {
        win.open();
        if (!win.isOpen()) { return; }
        const root = document.getElementById('combat-window');
        if (!root) { return; }
        const data = CompanyData.read();

        // Keep keyboard focus on the same member across the rebuild.
        const focused = document.activeElement;
        const focusedKey = (focused && root.contains(focused) && focused.getAttribute('data-key')) || null;

        root.textContent = '';
        if (hostileHere()) {
            const actions = el('div', 'cbt-actions');
            const scout = el('button', 'cbt-btn', 'Scout');
            scout.type = 'button';
            scout.title = 'See how the enemy groups here stand (scout)';
            scout.addEventListener('click', () => Client.SendInput('scout'));
            actions.appendChild(scout);
            root.appendChild(actions);
        }
        if (!data.company) {
            root.appendChild(el('div', 'cbt-note', 'No company yet. Your strategy: help strategy.'));
            return;
        }
        const members = data.members.filter(m => m && m.key);
        root.appendChild(grid(members));
        root.appendChild(el('div', 'cbt-note', 'A battle plays out on its own, by how you set it up here. Click a member to change it.'));
        const list = el('ul', 'cbt-members');
        list.setAttribute('aria-label', 'How your company fights');
        members.forEach(m => {
            const li = el('li');
            const b = el('button', 'cbt-member' + (m.status === 'dead' ? ' is-fallen' : ''));
            b.type = 'button';
            b.setAttribute('data-key', m.key);
            b.setAttribute('aria-haspopup', 'menu');
            b.appendChild(el('span', null, m.name + (m.key === 'leader' ? ' (you)' : '')));
            b.appendChild(el('span', 'cbt-how', howText(m) || '—'));
            b.setAttribute('aria-label', m.name + (m.key === 'leader' ? ' (you)' : '') + (howText(m) ? ': ' + howText(m) : ''));
            b.addEventListener('click', e => uiMenu(e, memberMenu(m, members)));
            li.appendChild(b);
            list.appendChild(li);
        });
        root.appendChild(list);

        if (focusedKey) {
            const again = root.querySelector('[data-key="' + (window.CSS && CSS.escape ? CSS.escape(focusedKey) : focusedKey) + '"]');
            if (again) { again.focus(); }
        }
    }

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Company', 'Room'],
        onGMCP(namespace) {
            if (namespace === 'Company.Inventory' || namespace === 'Company.Camp' || namespace === 'Company.Vitals') { return; }
            update();
        },
    });

})();
