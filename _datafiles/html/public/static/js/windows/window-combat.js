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
 * Phase 32g2, the Battle view, shown instead while the player's battle
 * runs (Company.Battle), with Setup folded under it:
 *
 *   - The enemy group's formation above the company's, both front rows
 *     toward the middle (the dock is one narrow column, so the two sides
 *     face each other up and down). Each enemy shows its battle label and
 *     how hurt it looks (scout's words, never numbers), and a dot when
 *     the player can reach it from their place.
 *   - Target lines (SVG, no server strings) from each fighter to whom it
 *     strikes; someone outside the company an enemy strikes is a chip
 *     beside the company. Hovering, focusing, or tapping a fighter lights
 *     its target and everyone striking it.
 *   - The fallen and the groups waiting their turn, a plain "who strikes
 *     whom" list for screen readers and narrow screens, a polite live
 *     region (a new foe on the player, a fall), and Flee (sends "flee").
 *   - A company member's click opens Setup's menu. The Combat tab shows a
 *     marker while a battle runs and another tab is showing.
 *
 * Phase 30c, company tactics:
 *
 *   - The Battle view's focus buttons (none and the six focus rules), the
 *     current one pressed, all disabled while an order waits for the next
 *     round; a click sends company tactics focus <rule>, for that battle
 *     only. "Saved" (company tactics focus default) returns to the saved
 *     focus. The live region says when the focus changes.
 *   - Setup's tactics row: the saved focus and healing threshold, with a
 *     menu sending company tactics focus / healing / default.
 *
 * Phase 30c2, guardians:
 *
 *   - guardian is a role in the member menu; a guardian's menu adds
 *     "Guard: <member>" (strategy <who> guard <other>) and "Guard: the
 *     most hurt" (strategy <who> guard). Its row says whom it guards,
 *     "(out of reach)" when the server marks the ward too far away.
 *   - In the Battle view a guardian's button adds whom it guards and its
 *     guards left (Company.Battle's guards).
 *
 * Every name is set with textContent, never innerHTML.
 *
 * Responds to GMCP namespaces:
 *   Company         - the snapshot: members, cells, strategies
 *   Company.Vitals  - members' health, for the Battle view
 *   Company.Battle  - the player's battle; {} when there is none
 *   Room            - Room.Info.Contents.Npcs, for Scout
 */

'use strict';

(function() {

    const el = CompanyData.el;

    const ROLES = ['fighter', 'healer', 'caster', 'guardian'];
    // Target rules, in the order `help strategy` lists them; assist is for
    // companions only.
    const RULES = ['weakest', 'strongest', 'wounded', 'nearest', 'furthest', 'leader', 'casters', 'assist', 'defend'];
    // Phase 30c: the company focus values, in the order `help tactics`
    // lists them, and the healing thresholds.
    const FOCI = ['none', 'leader', 'casters', 'nearest', 'weakest', 'strongest', 'wounded'];
    const HEALING = [10, 20, 30, 40, 50, 60, 70, 80, 90];

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

        /* Phase 32g2: the Battle view. */
        .cbt-live {
            position: absolute; width: 1px; height: 1px; overflow: hidden;
            clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap;
        }
        .cbt-body { display: flex; flex-direction: column; gap: 6px; }
        .cbt-battle-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
        .cbt-battle-head h3 { margin: 0; font-size: 1em; color: var(--t-text-heading); overflow-wrap: anywhere; }
        .cbt-arena { position: relative; display: flex; flex-direction: column; gap: 22px; }
        .cbt-side-label { color: var(--t-text-secondary); margin-bottom: 2px; }
        .cbt-field {
            display: grid;
            grid-template-columns: repeat(3, minmax(0, 1fr));
            gap: 3px;
        }
        .cbt-spot { min-height: 2.9em; border: 1px dashed var(--t-border-faint, var(--t-accent-dim)); border-radius: 3px; }
        .cbt-fighter {
            position: relative;
            z-index: 1;
            min-height: 2.9em;
            width: 100%;
            font: inherit;
            line-height: 1.2;
            padding: 2px 3px;
            text-align: center;
            overflow-wrap: anywhere;
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            cursor: pointer;
        }
        .cbt-fighter:focus-visible { outline: 2px solid var(--t-accent); }
        .cbt-fighter.is-enemy { border-color: var(--t-aggro-text, var(--t-error)); }
        .cbt-fighter.is-you { color: var(--t-party-leader); font-weight: bold; }
        .cbt-fighter.is-fallen { opacity: 0.55; border-style: dashed; }
        .cbt-fighter.is-hl { background: var(--t-accent-dim); color: var(--t-text-white); }
        .cbt-fighter .cbt-sub { display: block; font-weight: normal; font-size: 0.85em; color: var(--t-text-secondary); }
        .cbt-fighter.is-hl .cbt-sub { color: var(--t-text-white); }
        .cbt-fighter .cbt-reach { color: var(--t-success); }
        .cbt-h-scratched, .cbt-h-wounded { color: var(--t-hp-mid); }
        .cbt-h-badly-wounded, .cbt-h-near-death { color: var(--t-hp-low); }
        .cbt-chips { display: flex; flex-wrap: wrap; gap: 3px; align-items: center; }
        .cbt-chips .cbt-fighter { width: auto; min-height: 0; }
        .cbt-lines { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; overflow: visible; z-index: 0; }
        .cbt-lines line { stroke-width: 1.5; opacity: 0.7; }
        .cbt-lines line.from-us { stroke: var(--t-accent); }
        .cbt-lines line.from-them { stroke: var(--t-error); stroke-dasharray: 4 3; }
        .cbt-lines.has-hl line { opacity: 0.12; }
        .cbt-lines.has-hl line.is-hl { opacity: 1; stroke-width: 2.5; }
        .cbt-aside { color: var(--t-text-secondary); overflow-wrap: anywhere; }
        .cbt-strikes { margin: 0; padding-left: 1.2em; color: var(--t-text-secondary); }
        .cbt-focus { display: flex; flex-wrap: wrap; gap: 3px; align-items: center; }
        .cbt-focus .cbt-btn { padding: 2px 6px; }
        .cbt-focus .cbt-btn[aria-pressed="true"] { background: var(--t-accent-dim); color: var(--t-text-white); font-weight: bold; }
        .cbt-focus .cbt-btn:disabled { opacity: 0.55; cursor: default; }
        .cbt-setup > summary { cursor: pointer; color: var(--t-text-secondary); }
        .cbt-setup[open] > summary { margin-bottom: 4px; }
    `);

    function createDOM() {
        const root = el('div');
        root.id = 'combat-window';
        // The live region outlives each rebuild of the body, so screen
        // readers hear what it says.
        const live = el('div', 'cbt-live');
        live.id = 'combat-live';
        live.setAttribute('aria-live', 'polite');
        live.setAttribute('role', 'status');
        root.appendChild(live);
        const body = el('div', 'cbt-body');
        body.id = 'combat-body';
        body.appendChild(el('div', 'cbt-note', 'No company yet.'));
        root.appendChild(body);
        document.body.appendChild(root);
        return root;
    }

    const win = new VirtualWindow('Combat', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        onTabShown() {
            VirtualWindows.setTabBadge('Combat', 0);
            requestAnimationFrame(drawLines); // lines need a laid-out arena
        },
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

    // wardName names a guardian's ward among the members: "you" for the
    // player; none (or one gone) is the most hurt (Phase 30c2).
    function wardName(key, members) {
        const w = key ? members.find(x => x.key === key) : null;
        if (!w) { return 'the most hurt'; }
        return w.key === 'leader' ? 'you' : w.name;
    }

    function guardText(m, members) {
        const s = m.strategy;
        if (!s || s.role !== 'guardian') { return ''; }
        return 'guards ' + wardName(s.ward, members) + (s.ward_reach === false ? ' (out of reach)' : '');
    }

    function howText(m, members) {
        const s = m.strategy;
        if (!s) { return ''; }
        const g = guardText(m, members || []);
        return s.role + ', ' + s.target + (g ? ', ' + g : '');
    }

    function memberMenu(m, members) {
        const items = [];
        const s = m.strategy || {};
        ROLES.filter(r => r !== s.role).forEach(r => {
            items.push({ label: 'Role: ' + r, cmd: 'strategy ' + who(m) + ' ' + r });
        });
        if (s.role === 'guardian') {
            members.filter(x => x !== m && x.key !== s.ward && x.status !== 'dead').forEach(x => {
                items.push({ label: 'Guard: ' + wardName(x.key, members), cmd: 'strategy ' + who(m) + ' guard ' + who(x) });
            });
            if (s.ward) { items.push({ label: 'Guard: the most hurt', cmd: 'strategy ' + who(m) + ' guard' }); }
        }
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
                    td.title = m.name + (howText(m, members) ? ': ' + howText(m, members) : '');
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

    // Setup: the formation grid and a row per member, each opening the
    // member menu.
    function renderSetup(root, data, inBattle) {
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
            const how = howText(m, members);
            b.appendChild(el('span', 'cbt-how', how || '—'));
            b.setAttribute('aria-label', m.name + (m.key === 'leader' ? ' (you)' : '') + (how ? ': ' + how : ''));
            b.addEventListener('click', e => uiMenu(e, memberMenu(m, members)));
            li.appendChild(b);
            list.appendChild(li);
        });
        root.appendChild(list);
        const t = data.company.tactics;
        if (t && t.focus) {
            const how = 'focus ' + t.focus + ', heal below ' + t.healing + '%';
            if (inBattle) {
                // The saved tactics are set between battles; in one, the
                // Focus buttons above call this battle's focus.
                root.appendChild(el('div', 'cbt-note', 'Company tactics: ' + how + ' (saved; set between battles, or use Focus above)'));
                return;
            }
            const tb = el('button', 'cbt-member cbt-tactics');
            tb.type = 'button';
            tb.setAttribute('data-key', 'tactics');
            tb.setAttribute('aria-haspopup', 'menu');
            tb.appendChild(el('span', null, 'Company tactics'));
            tb.appendChild(el('span', 'cbt-how', how));
            tb.setAttribute('aria-label', 'Company tactics: ' + how);
            tb.addEventListener('click', e => uiMenu(e, tacticsMenu(t)));
            root.appendChild(tb);
        }
    }

    // tacticsMenu sets the saved company tactics (Phase 30c).
    function tacticsMenu(t) {
        const items = [];
        FOCI.filter(f => f !== t.focus).forEach(f => {
            items.push({ label: 'Focus: ' + f, cmd: 'company tactics focus ' + f });
        });
        HEALING.filter(h => h !== t.healing).forEach(h => {
            items.push({ label: 'Heal below ' + h + '%', cmd: 'company tactics healing ' + h });
        });
        items.push({ label: 'Back to the defaults', cmd: 'company tactics default' });
        return items;
    }

    // focusBar is the Battle view's focus buttons (Phase 30c): one order a
    // round, for this battle only.
    function focusBar(battle) {
        const bar = el('div', 'cbt-focus');
        bar.setAttribute('role', 'group');
        bar.setAttribute('aria-label', 'Company focus' + (battle.focus_ready ? '' : ', turning next round'));
        bar.appendChild(el('span', 'cbt-aside', 'Focus:'));
        const add = (label, value, pressed, title) => {
            const b = el('button', 'cbt-btn', label);
            b.type = 'button';
            b.setAttribute('data-focus', value);
            b.setAttribute('aria-pressed', pressed ? 'true' : 'false');
            b.title = title;
            b.disabled = !battle.focus_ready;
            b.addEventListener('click', () => Client.SendInput('company tactics focus ' + value));
            bar.appendChild(b);
        };
        FOCI.forEach(f => add(f, f, battle.focus === f,
            f === 'none' ? 'Each fights by their own strategy, this battle only' : 'Call the company onto their ' + f + ', this battle only'));
        if (battle.saved_focus && battle.saved_focus !== battle.focus) {
            add('saved (' + battle.saved_focus + ')', 'default', false, 'Back to your saved focus');
        }
        if (!battle.focus_ready) { bar.appendChild(el('span', 'cbt-aside', 'turning next round')); }
        return bar;
    }

    // ---------------------------------------------------------------------
    // Phase 32g2: the Battle view.
    // ---------------------------------------------------------------------

    function currentBattle() {
        const stored = Client.GMCPStructs.Company;
        const b = stored && stored.Battle;
        return (b && Array.isArray(b.enemies)) ? b : null;
    }

    // lines holds this render's target lines, { from, to, us }, by fighter
    // id: an enemy's "m:<id>", a member's key, an outsider's "u:<id>".
    let lines = [];
    let pinned = null;   // a tapped fighter whose lines stay lit
    let arenaObserver = null;
    // The newest Company.Battle. A Company snapshot replaces everything
    // stored under Company, the battle too, and the server sends the battle
    // again right after it; until then the view keeps this one rather than
    // flicker to Setup and back (32g2 review finding 1).
    let lastBattle = null;

    // The company's fighters: every member from the snapshot (the player
    // alone, as "You", without a company).
    function ourFighters(data) {
        if (!data.company) { return [{ key: 'leader', name: 'You', cell: null, status: 'present' }]; }
        return data.members.filter(m => m && m.key && m.status !== 'awaiting');
    }

    function fighterButton(id, cls, name, sub, spoken) {
        const b = el('button', 'cbt-fighter ' + cls);
        b.type = 'button';
        b.setAttribute('data-fid', id);
        b.appendChild(el('span', null, name));
        if (sub) { b.appendChild(sub); }
        b.setAttribute('aria-label', spoken);
        b.addEventListener('mouseenter', () => light(id));
        b.addEventListener('mouseleave', () => light(pinned));
        b.addEventListener('focus', () => light(id));
        b.addEventListener('blur', () => light(pinned));
        return b;
    }

    // field lays fighters out on the formation's grid, three columns wide,
    // from the front row to the deepest row anyone stands in (empty rows
    // behind them would only take room). The enemy's is drawn back row
    // first, so both front rows face the middle.
    function field(fighters, flip, label) {
        const g = el('div', 'cbt-field');
        g.setAttribute('role', 'group');
        g.setAttribute('aria-label', label);
        const at = {};
        let deepest = 0;
        fighters.forEach(f => {
            const c = f.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) {
                at[c.row + ',' + c.col] = f;
                deepest = Math.max(deepest, c.row);
            }
        });
        for (let i = 0; i <= deepest; i++) {
            const r = flip ? deepest - i : i;
            for (let c = 0; c < 3; c++) {
                const f = at[r + ',' + c];
                g.appendChild(f ? f.node : el('div', 'cbt-spot'));
            }
        }
        return g;
    }

    function nameOf(id, battle, data) {
        if (id === 'leader') { return data.company ? data.company.leader.name : 'you'; }
        const e = battle.enemies.find(x => x.id === id);
        if (e) { return e.label; }
        const o = (battle.others || []).find(x => x.id === id);
        if (o) { return o.name; }
        const m = data.members.find(x => x.key === id);
        return m ? m.name : '';
    }

    function renderBattle(root, battle, data) {
        const head = el('div', 'cbt-battle-head');
        head.appendChild(el('h3', null, 'Battle: ' + battle.group));
        const flee = el('button', 'cbt-btn', 'Flee');
        flee.type = 'button';
        flee.title = 'Try to get away (flee)';
        flee.addEventListener('click', () => Client.SendInput('flee'));
        head.appendChild(flee);
        root.appendChild(head);
        if (typeof battle.focus === 'string') { root.appendChild(focusBar(battle)); }

        lines = [];
        if (battle.dark) {
            // As scout: in the dark, nothing to see (32g2 review finding 3).
            root.appendChild(el('div', 'cbt-note', "It's too dark to make them out."));
            const setup = el('details', 'cbt-setup');
            setup.appendChild(el('summary', null, 'Setup: formation and strategies'));
            renderSetup(setup, data, true);
            root.appendChild(setup);
            return;
        }
        const aimsAt = {};
        (battle.company || []).forEach(a => { aimsAt[a.key] = a.target; lines.push({ from: a.key, to: a.target, us: true }); });

        const arena = el('div', 'cbt-arena');
        const enemies = battle.enemies.map(e => {
            const sub = el('span', 'cbt-sub');
            sub.appendChild(el('span', 'cbt-h-' + String(e.health).replace(/ /g, '-'), e.health));
            if (e.reach) { sub.appendChild(el('span', 'cbt-reach', ' •')); }
            const spoken = e.label + ', ' + e.health + (e.reach ? ', within your reach' : '') +
                (e.target ? ', striking ' + (e.target === 'leader' ? 'you' : nameOf(e.target, battle, data)) : '');
            if (e.target) { lines.push({ from: e.id, to: e.target, us: false }); }
            return { cell: e.cell, node: fighterButton(e.id, 'is-enemy', e.label, sub, spoken) };
        });
        enemies.forEach(f => f.node.addEventListener('click', () => pin(f.node.getAttribute('data-fid'))));
        const them = el('div');
        them.appendChild(el('div', 'cbt-side-label', battle.group + ' (front row nearest you)'));
        them.appendChild(field(enemies, true, battle.group));
        arena.appendChild(them);

        const members = data.members.filter(m => m && m.key);
        const ours = ourFighters(data).map(m => {
            const v = data.company ? data.vitals(m.key) : {};
            const fallen = m.status === 'dead';
            let subText = '';
            if (fallen) { subText = 'fallen'; } else if (v.hp !== null && v.hp !== undefined) { subText = v.hp + ' / ' + v.hp_max; }
            // Phase 30c2: a guardian's ward and guards left.
            const g = fallen ? null : (battle.guards || []).find(x => x.key === m.key);
            const guardNote = g ? 'guards ' + wardName(g.ward, members) + ', ' +
                (g.left > 0 ? g.left + (g.left === 1 ? ' guard' : ' guards') + ' left' : 'no guards left') : '';
            const subLine = [subText, guardNote].filter(Boolean).join(' · ');
            const sub = subLine ? el('span', 'cbt-sub', subLine) : null;
            const you = m.key === 'leader';
            const target = aimsAt[m.key];
            const spoken = (you && data.company ? m.name + ' (you)' : m.name) + (subText ? ', ' + (fallen ? 'fallen' : 'health ' + subText.replace(' / ', ' of ')) : '') +
                (target ? ', striking ' + nameOf(target, battle, data) : '') + (guardNote ? ', ' + guardNote : '');
            const node = fighterButton(m.key, (you ? 'is-you' : '') + (fallen ? ' is-fallen' : ''), m.name, sub, spoken);
            if (data.company) {
                node.setAttribute('aria-haspopup', 'menu');
                node.addEventListener('click', e => { pin(m.key); uiMenu(e, memberMenu(m, members)); });
            } else {
                node.addEventListener('click', () => pin(m.key));
            }
            return { cell: m.cell, node };
        });
        const us = el('div');
        const placed = ours.filter(f => f.cell);
        const loose = ours.filter(f => !f.cell);
        us.appendChild(field(placed, false, data.company ? 'Your company' : 'You'));
        const others = (battle.others || []).map(o =>
            fighterButton(o.id, '', o.name, el('span', 'cbt-sub', 'not in your company'), o.name + ', not in your company'));
        others.forEach(n => n.addEventListener('click', () => pin(n.getAttribute('data-fid'))));
        if (loose.length || others.length) {
            const chips = el('div', 'cbt-chips');
            if (loose.length) {
                if (data.company) { chips.appendChild(el('span', 'cbt-aside', 'Not placed:')); }
                loose.forEach(f => chips.appendChild(f.node));
            }
            if (others.length) {
                chips.appendChild(el('span', 'cbt-aside', 'Also fighting:'));
                others.forEach(n => chips.appendChild(n));
            }
            us.appendChild(chips);
        }
        us.appendChild(el('div', 'cbt-side-label', (data.company ? 'Your company' : 'You') + ' (front row nearest them)'));
        arena.appendChild(us);

        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('class', 'cbt-lines');
        svg.setAttribute('aria-hidden', 'true');
        arena.appendChild(svg);
        root.appendChild(arena);
        // Lines follow the fighters when the dock is resized, which fires
        // no window resize (32g2 review finding 6).
        if (window.ResizeObserver) {
            if (arenaObserver) { arenaObserver.disconnect(); }
            arenaObserver = new ResizeObserver(() => requestAnimationFrame(drawLines));
            arenaObserver.observe(arena);
        }

        if (battle.fallen && battle.fallen.length) {
            root.appendChild(el('div', 'cbt-aside', 'Fallen: ' + battle.fallen.map(f => f.label).join(', ')));
        }
        if (battle.waiting && battle.waiting.length) {
            root.appendChild(el('div', 'cbt-aside', 'Waiting their turn: ' + battle.waiting.join(', ')));
        }

        const list = el('ul', 'cbt-strikes');
        list.setAttribute('aria-label', 'Who strikes whom');
        lines.forEach(l => {
            const you = data.company ? nameOf('leader', battle, data) + ' (you)' : 'you';
            const who = l.from === 'leader' ? (data.company ? you : 'You') : nameOf(l.from, battle, data);
            const whom = l.to === 'leader' ? you : nameOf(l.to, battle, data);
            list.appendChild(el('li', null, who + ' → ' + whom));
        });
        if (!lines.length) { list.appendChild(el('li', null, 'No one is striking anyone this moment.')); }
        root.appendChild(list);

        const setup = el('details', 'cbt-setup');
        setup.appendChild(el('summary', null, 'Setup: formation and strategies'));
        renderSetup(setup, data, true);
        root.appendChild(setup);

        requestAnimationFrame(drawLines);
    }

    // drawLines draws each target line between its two fighters' centres.
    function drawLines() {
        const svg = document.querySelector('#combat-window .cbt-lines');
        if (!svg) { return; }
        const arena = svg.parentNode;
        const box = arena.getBoundingClientRect();
        while (svg.firstChild) { svg.removeChild(svg.firstChild); }
        if (!box.width) { return; }
        const centre = id => {
            const n = arena.querySelector('[data-fid="' + (window.CSS && CSS.escape ? CSS.escape(id) : id) + '"]');
            if (!n) { return null; }
            const r = n.getBoundingClientRect();
            return { x: r.left - box.left + r.width / 2, y: r.top - box.top + r.height / 2 };
        };
        lines.forEach(l => {
            const a = centre(l.from);
            const b = centre(l.to);
            if (!a || !b) { return; }
            const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
            line.setAttribute('x1', a.x);
            line.setAttribute('y1', a.y);
            line.setAttribute('x2', b.x);
            line.setAttribute('y2', b.y);
            line.setAttribute('class', l.us ? 'from-us' : 'from-them');
            line.setAttribute('data-from', l.from);
            line.setAttribute('data-to', l.to);
            svg.appendChild(line);
        });
        light(pinned);
    }

    // light lights a fighter, its target, those striking it, and the
    // lines between them; null lights nothing.
    function light(id) {
        const root = document.getElementById('combat-window');
        if (!root) { return; }
        root.querySelectorAll('.cbt-fighter.is-hl').forEach(n => n.classList.remove('is-hl'));
        const svg = root.querySelector('.cbt-lines');
        if (svg) { svg.classList.toggle('has-hl', !!id); }
        if (!id) {
            if (svg) { svg.querySelectorAll('line.is-hl').forEach(n => n.classList.remove('is-hl')); }
            return;
        }
        const lit = new Set([id]);
        lines.forEach(l => {
            if (l.from === id) { lit.add(l.to); }
            if (l.to === id) { lit.add(l.from); }
        });
        root.querySelectorAll('.cbt-fighter[data-fid]').forEach(n => {
            if (lit.has(n.getAttribute('data-fid'))) { n.classList.add('is-hl'); }
        });
        if (svg) {
            svg.querySelectorAll('line').forEach(n => {
                n.classList.toggle('is-hl', n.getAttribute('data-from') === id || n.getAttribute('data-to') === id);
            });
        }
    }

    // pin keeps a tapped fighter lit (a touch screen has no hover); a
    // second tap lets go.
    function pin(id) {
        pinned = pinned === id ? null : id;
        light(pinned || id);
    }

    // What the live region last knew: who struck the player, who had fallen.
    let heard = null;

    function announce(battle) {
        const live = document.getElementById('combat-live');
        if (!live) { return; }
        const said = [];
        if (!battle) {
            if (heard) { said.push('The battle is over.'); }
            heard = null;
        } else {
            const onYou = new Set(battle.enemies.filter(e => e.target === 'leader').map(e => e.id));
            const fallen = new Set((battle.fallen || []).map(f => f.id));
            if (!heard) {
                said.push('Battle: ' + battle.group + '.');
            } else {
                (battle.fallen || []).forEach(f => { if (!heard.fallen.has(f.id)) { said.push(f.label + ' falls.'); } });
                battle.enemies.forEach(e => { if (onYou.has(e.id) && !heard.onYou.has(e.id)) { said.push(e.label + ' turns on you.'); } });
                if (battle.focus && heard.focus && battle.focus !== heard.focus) { said.push('Focus: ' + battle.focus + '.'); }
            }
            heard = { onYou, fallen, focus: battle.focus };
        }
        if (said.length) { live.textContent = said.join(' '); }
    }

    function update() {
        const battle = currentBattle();
        announce(battle);
        if (battle && !VirtualWindows.isTabShowing('Combat')) {
            VirtualWindows.setTabBadge('Combat', '⚔', 'a battle is under way');
        } else {
            VirtualWindows.setTabBadge('Combat', 0);
        }
        win.open();
        if (!win.isOpen()) { return; }
        const root = document.getElementById('combat-body');
        if (!root) { return; }
        const data = CompanyData.read();

        // Keep keyboard focus on the same member or fighter across the
        // rebuild.
        const focused = document.activeElement;
        const inside = focused && root.contains(focused);
        const focusedKey = (inside && focused.getAttribute('data-key')) || null;
        const focusedFid = (inside && focused.getAttribute('data-fid')) || null;
        const setupOpen = !!root.querySelector('details.cbt-setup[open]');

        root.textContent = '';
        if (battle) {
            renderBattle(root, battle, data);
            if (setupOpen) { root.querySelector('details.cbt-setup').open = true; }
        } else {
            lines = [];
            pinned = null;
            if (arenaObserver) { arenaObserver.disconnect(); arenaObserver = null; }
            renderSetup(root, data);
        }

        const esc = v => (window.CSS && CSS.escape ? CSS.escape(v) : v);
        const again = focusedFid ? root.querySelector('[data-fid="' + esc(focusedFid) + '"]')
            : focusedKey ? root.querySelector('[data-key="' + esc(focusedKey) + '"]') : null;
        if (again) { again.focus(); }
    }

    window.addEventListener('resize', () => requestAnimationFrame(drawLines));

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Company', 'Room'],
        onGMCP(namespace, body) {
            if (namespace === 'Company.Battle') {
                lastBattle = (body && Array.isArray(body.enemies)) ? body : null;
            } else if (namespace === 'Company' && lastBattle) {
                const stored = Client.GMCPStructs.Company;
                if (stored && typeof stored === 'object' && !stored.Battle) { stored.Battle = lastBattle; }
            }
            if (namespace === 'Company.Inventory' || namespace === 'Company.Camp') { return; }
            if (namespace === 'Company.Vitals' && !currentBattle()) { return; }
            update();
        },
    });

})();
