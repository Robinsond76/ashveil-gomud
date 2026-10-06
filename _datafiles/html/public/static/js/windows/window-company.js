/**
 * window-company.js
 *
 * Virtual window: Company, the company dock's second tab (Phase 32g; it
 * was Phase 26b's Party window). Three sub-tabs:
 *
 *   Status    - the company (26b): a 3x3 formation table and a card per
 *               member (health, needs, chemistry, a fallen member's time
 *               left to raise), then GoMud's human party under
 *               "Travelling with" (level, rank, location, health).
 *   Inventory - `company inventory` for the web: the load and its split,
 *               every member's worn and carried items (the player first),
 *               the horses, and the cargo, with tooltips and click menus
 *               that send real commands; Meal, Eat, and Drink buttons.
 *   Camp      - the camp seen from here, the fire, a rest's progress, the
 *               rest tier, each member's needs; buttons shown only when
 *               they would work.
 *
 * Every name and label is set with textContent, never innerHTML. Menus
 * name items by the reference the server sends ("!<id>:<uuid>"), which
 * picks out exactly that item even when two share a name.
 *
 * Responds to GMCP namespaces:
 *   Company           - full company snapshot ({} when there is none)
 *   Company.Vitals    - health, needs, and countdowns (activity, rest, rescue
 *                       time); applied over the roster, never replacing it
 *   Company.Inventory - the Inventory sub-tab
 *   Company.Camp      - the Camp sub-tab
 *   Party             - full party update (roster + vitals)
 *   Party.Vitals      - lightweight vitals-only update
 *
 * The state is read from Client.GMCPStructs on every render, where the
 * client stores each payload even while this window is closed, so a
 * reopened window is current. The server sends the snapshot at login and
 * copyover.
 */

'use strict';

(function() {

    injectStyles(`
        /* Phase 32g: the Company tab and its sub-tabs */
        #company-window {
            color: var(--t-text);
            height: 100%;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
        }

        .cmp-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        .cmp-tab-btn {
            flex: 1;
            padding: 5px 4px;
            background: var(--t-bg-surface);
            border: none;
            border-right: 1px solid var(--t-border);
            cursor: pointer;
            font: inherit;
            font-size: 0.7em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .cmp-tab-btn:last-child { border-right: none; }
        .cmp-tab-btn:hover { background: var(--t-border); color: var(--t-text); }
        .cmp-tab-btn.active { background: var(--t-bg); color: var(--t-text); border-bottom: 2px solid var(--t-accent); }
        .cmp-tab-btn:focus-visible { outline: 2px solid var(--t-accent); outline-offset: -2px; }

        .cmp-panel { flex: 1; min-height: 0; overflow-y: auto; }
        .cmp-panel[hidden] { display: none !important; }

        .cmp-pad {
            padding: 4px 6px;
            display: flex;
            flex-direction: column;
            gap: 5px;
            font-size: 0.8em;
        }

        .cmp-note { color: var(--t-text-secondary); font-style: italic; }

        .cmp-line { color: var(--t-text-secondary); }

        .cmp-actions { display: flex; flex-wrap: wrap; gap: 4px; }

        .cmp-btn {
            font: inherit;
            font-size: 0.95em;
            padding: 3px 9px;
            border: 1px solid var(--t-btn-border, var(--t-accent-dim));
            border-radius: 3px;
            background: var(--t-bg-surface);
            color: var(--t-text);
            cursor: pointer;
        }

        .cmp-btn:hover { background: var(--t-accent-dim); color: var(--t-text-white); }
        .cmp-btn:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }

        .cmp-block {
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 4px 6px;
            background: var(--t-bg-surface-alt);
        }

        .cmp-block h4 {
            margin: 0 0 3px;
            font-size: 1em;
            color: var(--t-text);
            display: flex;
            justify-content: space-between;
            gap: 6px;
        }

        .cmp-block h4 .cmp-weight { color: var(--t-text-secondary); font-weight: normal; }
        .cmp-sub { color: var(--t-text-secondary); font-size: 0.9em; margin-top: 2px; }

        .cmp-items { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 1px; }

        .cmp-item {
            display: flex;
            justify-content: space-between;
            gap: 6px;
            width: 100%;
            text-align: left;
            font: inherit;
            color: var(--t-text);
            background: none;
            border: none;
            padding: 1px 2px;
            border-radius: 2px;
        }

        button.cmp-item { cursor: pointer; }
        button.cmp-item:hover { background: var(--t-accent-dim); color: var(--t-text-white); }
        button.cmp-item:focus-visible { outline: 2px solid var(--t-accent); }
        .cmp-item .cmp-meta { color: var(--t-text-secondary); white-space: nowrap; }

        .cmp-meter {
            height: 6px;
            border-radius: 3px;
            background: var(--t-bar-empty);
            overflow: hidden;
        }

        .cmp-meter > div { height: 100%; background: var(--t-accent); }
        .cmp-meter.full > div { background: var(--t-party-hp-low); }

        .cmp-needs { border-collapse: collapse; width: 100%; table-layout: fixed; font-size: 0.92em; }
        .cmp-needs th, .cmp-needs td { text-align: left; padding: 1px 3px; border-bottom: 1px solid var(--t-accent-dim); overflow-wrap: anywhere; }
        .cmp-needs caption { text-align: left; }
        .cmp-needs th { color: var(--t-text-secondary); font-weight: normal; }

        #party-panel {
            overflow-y: auto;
            padding: 4px 6px;
            background: var(--t-bg);
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        #party-panel::-webkit-scrollbar       { width: 4px; }
        #party-panel::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #party-panel::-webkit-scrollbar-thumb  { background: var(--t-scrollbar-thumb); border-radius: 2px; }

        .party-empty {
            color: var(--t-text-secondary);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .party-member {
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 5px 7px;
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .party-member.is-leader {
            border-color: var(--t-party-leader);
        }

        .party-member.is-invited {
            border-color: var(--t-party-invited-border);
            background: var(--t-party-invited-bg);
            opacity: 0.7;
        }

        .party-member-header {
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .party-member-name {
            flex: 1;
            font-size: 0.82em;
            color: var(--t-text);
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .party-member.is-leader .party-member-name {
            color: var(--t-party-leader);
        }

        .party-member.is-invited .party-member-name {
            color: var(--t-party-invited-text);
        }

        .party-member-level {
            font-size: 0.7em;
            color: var(--t-text-secondary);
            flex-shrink: 0;
        }

        .party-member-rank {
            font-size: 0.65em;
            color: var(--t-party-invited-border);
            flex-shrink: 0;
            text-transform: capitalize;
        }

        .party-member-location {
            font-size: 0.68em;
            color: var(--t-party-location);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .party-hp-track {
            width: 100%;
            height: 5px;
            background: var(--t-party-hp-bg);
            border-radius: 3px;
            overflow: hidden;
            border: 1px solid var(--t-party-hp-border);
        }

        .party-hp-fill {
            height: 100%;
            border-radius: 3px;
            transition: width 0.3s ease-out;
        }

        /* Colour shifts from green → yellow → red as HP drops */
        .party-hp-fill[data-pct="high"]   { background: var(--t-party-hp-high); }
        .party-hp-fill[data-pct="medium"] { background: var(--t-party-hp-mid); }
        .party-hp-fill[data-pct="low"]    { background: var(--t-party-hp-low); }

        .party-invited-label {
            font-size: 0.65em;
            color: var(--t-party-invited-border);
            font-style: italic;
        }

        /* Phase 26b: the Company and Players sections */
        .company-section, .players-section {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .panel-heading {
            margin: 2px 0 0;
            font-size: 0.72em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
            border-bottom: 1px solid var(--t-accent-dim);
        }

        .company-summary {
            font-size: 0.72em;
            color: var(--t-text-secondary);
        }

        /* Phase 32a: names at the member cards' size and colour, wrapped
           rather than cut off. */
        .company-formation {
            border-collapse: collapse;
            font-size: 0.82em;
            color: var(--t-text);
            table-layout: fixed;
            width: 100%;
        }

        .company-formation caption {
            text-align: left;
            font-size: 0.88em;
            color: var(--t-text-secondary);
            padding-bottom: 2px;
        }

        .company-formation th {
            width: 3.6em;
            font-size: 0.88em;
            font-weight: normal;
            color: var(--t-text-secondary);
            text-align: left;
        }

        .company-formation td {
            border: 1px solid var(--t-accent-dim);
            padding: 3px 4px;
            text-align: center;
            white-space: normal;
            overflow-wrap: anywhere;
            line-height: 1.25;
        }

        .company-formation td.empty { color: var(--t-text-secondary); }
        .company-formation td.is-leader { color: var(--t-party-leader); font-weight: bold; }

        .company-members {
            list-style: none;
            margin: 0;
            padding: 0;
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
        }

        .company-members > li {
            flex: 1 1 200px;
            min-width: 0;
        }

        .company-members > li:focus-visible {
            outline: 2px solid var(--t-party-leader);
            outline-offset: 1px;
        }

        .company-member.status-dead { opacity: 0.75; border-style: dashed; }

        .company-status, .company-hp-text, .company-needs, .company-warmth, .company-chemistry {
            font-size: 0.68em;
            color: var(--t-text-secondary);
        }

        .company-fallen, .need-warn { color: var(--t-party-hp-low); font-weight: bold; }

        .company-class { font-size: 0.68em; color: var(--t-text-secondary); }
        .company-class.is-elite { color: var(--t-party-leader); font-weight: bold; }
        .company-promote { font-size: 0.68em; color: var(--t-party-leader); font-weight: bold; }
        .company-promote.is-waiting { color: var(--t-text-secondary); font-weight: normal; }
    `);

    function hpClass(pct) {
        if (pct >= 60) { return 'high'; }
        if (pct >= 30) { return 'medium'; }
        return 'low';
    }

    // el builds an element with an optional class and text, safely.
    const el            = CompanyData.el;
    const formatSeconds = CompanyData.formatSeconds;

    function hpBar(hp, max, label) {
        const pct   = max > 0 ? Math.max(0, Math.min(100, Math.round(hp * 100 / max))) : 0;
        const track = el('div', 'party-hp-track');
        track.setAttribute('role', 'meter');
        track.setAttribute('aria-label', label);
        track.setAttribute('aria-valuemin', '0');
        track.setAttribute('aria-valuemax', String(max));
        track.setAttribute('aria-valuenow', String(hp));
        const fill = el('div', 'party-hp-fill');
        fill.setAttribute('data-pct', hpClass(pct));
        fill.style.width = pct + '%';
        track.appendChild(fill);
        return track;
    }

    const SUBTABS = [
        { id: 'party-panel',       label: 'Status' },
        { id: 'company-inventory', label: 'Inventory' },
        { id: 'company-camp',      label: 'Camp' },
    ];
    const SUBTAB_KEY = 'companySubTab';

    function showSubtab(root, id, remember) {
        if (!SUBTABS.some(t => t.id === id)) { id = SUBTABS[0].id; }
        root.querySelectorAll('.cmp-tab-btn').forEach(b => {
            const on = b.dataset.panel === id;
            b.classList.toggle('active', on);
            b.setAttribute('aria-selected', on ? 'true' : 'false');
        });
        root.querySelectorAll('.cmp-panel').forEach(p => { p.hidden = p.id !== id; });
        if (remember) {
            try { localStorage.setItem(SUBTAB_KEY, id); } catch (e) { /* unavailable */ }
        }
    }

    function createDOM() {
        const root = el('div');
        root.id = 'company-window';
        const bar = el('div', 'cmp-tab-bar');
        bar.setAttribute('role', 'tablist');
        bar.setAttribute('aria-label', 'Company');
        SUBTABS.forEach(t => {
            const btn = el('button', 'cmp-tab-btn', t.label);
            btn.type = 'button';
            btn.dataset.panel = t.id;
            btn.setAttribute('role', 'tab');
            btn.addEventListener('click', () => showSubtab(root, t.id, true));
            bar.appendChild(btn);
        });
        root.appendChild(bar);
        SUBTABS.forEach(t => {
            const panel = el('div', 'cmp-panel');
            panel.id = t.id;
            panel.setAttribute('role', 'tabpanel');
            panel.setAttribute('aria-label', t.label);
            root.appendChild(panel);
        });
        root.querySelector('#party-panel').appendChild(el('div', 'party-empty', 'You travel alone; help company to recruit.'));
        let saved = null;
        try { saved = localStorage.getItem(SUBTAB_KEY); } catch (e) { /* unavailable */ }
        showSubtab(root, saved || SUBTABS[0].id, false);
        document.body.appendChild(root);
        return root;
    }

    const win = new VirtualWindow('Company', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        factory() {
            const root = createDOM();
            // Opened or reopened: show what arrived while it was closed.
            setTimeout(update, 0);
            return {
                title:      'Company',
                mount:      root,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      280,
                height:     240,
                header:     20,
                bottom:     60,
            };
        },
    });

    // --- Company state (Phase 26b) ---
    // company is the last snapshot; live is the newest health, needs, and
    // countdowns: a Company.Vitals that arrived after the snapshot (the
    // client stores it at Company.Vitals, and a new snapshot replaces the
    // whole Company object), else the snapshot's own.
    let company = null;
    let live    = null;

    function readCompany() {
        const stored = Client.GMCPStructs.Company;
        company = (stored && stored.leader) ? stored : null;
        live    = null;
        if (!company) { return; }
        const newer = stored.Vitals;
        live = (newer && typeof newer === 'object' && newer.vitals) ? newer : company;
    }

    function liveVitals(key) {
        return (live && live.vitals && live.vitals[key]) || {};
    }

    function companyMembers() {
        if (!company) { return []; }
        return [company.leader].concat(Array.isArray(company.members) ? company.members : []);
    }

    function needsText(needs) {
        if (!needs) { return []; }
        const out = [];
        [['hunger', 'Hunger'], ['thirst', 'Thirst'], ['fatigue', 'Fatigue']].forEach(pair => {
            const n = needs[pair[0]];
            if (n && n.label) { out.push({ name: pair[1], label: n.label, warn: !!n.warn }); }
        });
        return out;
    }

    function summaryLine() {
        const parts = [];
        const alive = company.alive || 0;
        const dead  = company.dead || 0;
        parts.push(alive + ' alive' + (dead ? ', ' + dead + ' fallen' : ''));
        if (company.load && company.load.label) {
            parts.push(company.load.label + ' (' + (company.load.total_g / 1000).toFixed(1) + '/' + (company.load.capacity_g / 1000).toFixed(1) + ' kg)');
        }
        if (live.activity) { parts.push(live.activity); }
        if (live.rest && live.rest.tier && live.rest.tier !== 'none') {
            parts.push(live.rest.tier + ' ' + formatSeconds(live.rest.seconds));
        }
        return parts.join(' \u00b7 ');
    }

    function formationTable(members) {
        const grid = [[null, null, null], [null, null, null], [null, null, null]];
        members.forEach(m => {
            const c = m && m.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) { grid[c.row][c.col] = m; }
        });
        const table = el('table', 'company-formation');
        table.appendChild(el('caption', null, 'Formation (row 1 is the front)'));
        grid.forEach((row, r) => {
            const tr = el('tr');
            const th = el('th', null, 'Row ' + (r + 1));
            th.setAttribute('scope', 'row');
            tr.appendChild(th);
            row.forEach(m => {
                const td = el('td', m ? 'filled' + (m.key === 'leader' ? ' is-leader' : '') : 'empty', m ? m.name : '\u00b7');
                if (m) { td.title = m.name; } else { td.setAttribute('aria-label', 'empty'); }
                tr.appendChild(td);
            });
            table.appendChild(tr);
        });
        return table;
    }

    function memberCard(m) {
        const v    = liveVitals(m.key);
        const card = el('li', 'party-member company-member' + (m.key === 'leader' ? ' is-leader' : '') + ' status-' + m.status);
        card.tabIndex = 0;
        card.setAttribute('data-key', m.key);

        const header = el('div', 'party-member-header');
        header.appendChild(el('span', 'party-member-name', m.name + (m.key === 'leader' ? ' \u2605' : '')));
        if (m.level) { header.appendChild(el('span', 'party-member-level', 'Lv ' + m.level)); }
        // A promoted member shows its class (Phase 40s5), its lineage on hover.
        const rank = m.class_name || m.archetype;
        if (rank) {
            const node = el('span', 'party-member-rank', rank);
            if (m.class_name && m.archetype) { node.title = m.archetype + ' line'; }
            header.appendChild(node);
        }
        card.appendChild(header);

        const spoken = [m.name, m.level ? 'level ' + m.level : '', rank || ''];
        // Phase 38c1: under the class name in the header, its tier (an elite
        // badge) and rank, and whether a promotion is ready or waiting on
        // alignment.
        if (m.class_name && (m.tier || m.rank)) {
            const elite = m.tier === 'elite';
            const text  = (elite ? '\u2605 elite' : (m.tier || '')) + (m.rank ? (m.tier ? ', ' : '') + 'rank ' + m.rank : '');
            card.appendChild(el('div', 'company-class' + (elite ? ' is-elite' : ''), text));
            spoken.push((elite ? 'elite' : (m.tier || '')) + (m.rank ? ' rank ' + m.rank : ''));
        }
        if (m.promotion === 'ready') {
            card.appendChild(el('div', 'company-promote', 'Promotion ready'));
            spoken.push('promotion ready');
        } else if (m.promotion === 'waiting-gate') {
            card.appendChild(el('div', 'company-promote is-waiting', 'Promotion waiting on alignment'));
            spoken.push('promotion waiting on alignment');
        }
        if (m.status === 'dead') {
            const rescue = live.rescue && live.rescue[m.key];
            const left = typeof rescue === 'number' ? 'Fallen: ' + formatSeconds(rescue) + ' to raise' : 'Fallen';
            card.appendChild(el('div', 'company-status company-fallen', left));
            spoken.push(left);
        } else if (m.status === 'awaiting') {
            card.appendChild(el('div', 'company-status', 'Away: rejoins when you return'));
            spoken.push('away');
        } else if (m.status === 'separated') {
            card.appendChild(el('div', 'company-status', 'Separated: finding the way back'));
            spoken.push('separated');
        }
        if (m.status !== 'dead' && typeof v.hp === 'number' && typeof v.hp_max === 'number') {
            const label = 'Health ' + v.hp + ' of ' + v.hp_max;
            card.appendChild(hpBar(v.hp, v.hp_max, label));
            card.appendChild(el('div', 'company-hp-text', v.hp + '/' + v.hp_max));
            spoken.push(label);
        }
        const needs = needsText(v.needs);
        if (m.status !== 'dead' && needs.length) {
            const line = el('div', 'company-needs');
            needs.forEach((n, i) => {
                if (i > 0) { line.appendChild(document.createTextNode(' \u00b7 ')); }
                line.appendChild(el('span', n.warn ? 'need-warn' : 'need-ok', n.name + ': ' + n.label + (n.warn ? '!' : '')));
                spoken.push(n.name + ' ' + n.label);
            });
            card.appendChild(line);
        }
        if (m.status !== 'dead' && v.warmth) {
            card.appendChild(el('div', 'company-warmth need-warn', v.warmth));
            spoken.push(v.warmth);
        }
        if (m.chemistry) {
            card.appendChild(el('div', 'company-chemistry', 'Persistent bonus — Chemistry: ' + m.chemistry));
            spoken.push('chemistry ' + m.chemistry);
        }
        const conditions = Client.GMCPStructs.Company && Client.GMCPStructs.Company.Conditions;
        const state = conditions && (conditions[m.key] || { state: 'unknown' });
        if (state) {
            const states = { away: 'Away: live effects unknown; wounds last recorded',
                separated: 'Separated: live effects unknown; wounds last recorded',
                'away-live': 'Away from you: current effects and wounds', dead: 'Fallen: no active member effects',
                unavailable: 'Conditions unavailable', unknown: 'Conditions unknown' };
            if (states[state.state]) {
                card.appendChild(el('div', 'company-status', states[state.state]));
                spoken.push(states[state.state]);
            }
            [['effects', 'Active effects', 'effect'], ['wounds', 'Wounds', 'wound'], ['bonuses', 'Persistent bonuses', 'bonus']].forEach(group => {
                const entries = state[group[0]] || [];
                if (!entries.length) { return; }
                card.appendChild(el('h4', 'panel-heading', group[1]));
                entries.forEach(effect => {
                    card.appendChild(CompanyData.condition(effect, group[2]));
                    spoken.push(CompanyData.conditionLabel(effect, group[2]));
                });
            });
            if (state.state === 'live' && !(state.effects || []).length && !(state.wounds || []).length) {
                card.appendChild(el('div', 'company-status', 'No active effects or wounds'));
            }
        }
        card.setAttribute('aria-label', spoken.filter(Boolean).join(', '));
        return card;
    }

    function companySection() {
        const section = el('section', 'company-section');
        section.setAttribute('aria-label', 'Company');
        section.appendChild(el('h3', 'panel-heading', 'Company'));
        section.appendChild(el('div', 'company-summary', summaryLine()));
        const members = companyMembers();
        section.appendChild(formationTable(members));
        const list = el('ul', 'company-members');
        list.setAttribute('aria-label', 'Company members');
        members.forEach(m => { if (m && m.key) { list.appendChild(memberCard(m)); } });
        section.appendChild(list);
        return section;
    }

    function playersSection(partyData) {
        const vitals  = (partyData && partyData.Vitals)  || {};
        const members = (partyData && partyData.Members) || [];
        const invited = (partyData && partyData.Invited) || [];
        const leader  = (partyData && partyData.Leader)  || '';

        const section = el('section', 'players-section');
        section.setAttribute('aria-label', 'Travelling with');
        section.appendChild(el('h3', 'panel-heading', 'Travelling with'));
        section.appendChild(el('div', 'company-summary', 'Allied companies: party leadership coordinates players; each owner commands their own company.'));

        // When we only have vitals data, synthesise member entries from it.
        const allMembers = members.length > 0 ? members : Object.keys(vitals).map(name => ({ Name: name, Status: 'In Party', Position: '' }));

        allMembers.forEach(m => {
            const name     = m.Name || m.name || '';
            const rank     = m.Position || m.position || '';
            const isLeader = name === leader;
            const v        = vitals[name] || {};
            const hpPct    = Math.max(0, Math.min(100, v.health || 0));
            const level    = v.level || 0;
            const location = v.location || '';

            const div    = el('div', 'party-member' + (isLeader ? ' is-leader' : ''));
            const header = el('div', 'party-member-header');
            header.appendChild(el('span', 'party-member-name', name + (isLeader ? ' \u2605' : '')));
            if (level) { header.appendChild(el('span', 'party-member-level', 'Lv ' + level)); }
            if (rank)  { header.appendChild(el('span', 'party-member-rank', rank)); }
            div.appendChild(header);
            if (location) { div.appendChild(el('div', 'party-member-location', location)); }
            if (m.owner_user_id) {
                const consent = 'Follow ' + (m.follow ? 'on' : 'off') + ', support ' + (m.support ? 'on' : 'off') + ', autoattack ' + (m.autoattack ? 'on' : 'off');
                div.appendChild(el('div', 'party-member-location', consent + (m.online === false ? ' (offline)' : '')));
            }
            div.appendChild(hpBar(hpPct, 100, 'Health ' + hpPct + ' percent'));
            section.appendChild(div);
        });

        // Invited members (no vitals available)
        invited.forEach(m => {
            const div    = el('div', 'party-member is-invited');
            const header = el('div', 'party-member-header');
            header.appendChild(el('span', 'party-member-name', m.Name || m.name || ''));
            header.appendChild(el('span', 'party-invited-label', 'invited'));
            div.appendChild(header);
            section.appendChild(div);
        });
        return section;
    }

    // update redraws the tab; namespace is the payload that prompted it.
    function update(namespace) {
        win.open();
        if (!win.isOpen()) { return; }

        const panel = document.getElementById('party-panel');
        if (!panel) { return; }
        readCompany();
        const partyData = Client.GMCPStructs.Party;

        // Keep keyboard focus on the same card across the rebuild.
        const focused    = document.activeElement;
        const focusedKey = (focused && panel.contains(focused) && focused.getAttribute('data-key')) || null;
        const hasParty  = !!(partyData && ((partyData.Members && partyData.Members.length) || (partyData.Vitals && Object.keys(partyData.Vitals).length)));

        panel.textContent = '';
        if (company) {
            panel.appendChild(companySection());
        } else {
            panel.appendChild(el('div', 'party-empty', 'You travel alone; help company to recruit.'));
        }
        if (hasParty) { panel.appendChild(playersSection(partyData)); }

        if (focusedKey) {
            const again = panel.querySelector('[data-key="' + (window.CSS && CSS.escape ? CSS.escape(focusedKey) : focusedKey) + '"]');
            if (again) { again.focus(); }
        }
        // The other sub-tabs read the snapshot too: Camp its needs, and
        // Inventory the companions out (names only, so not on vitals).
        if (namespace !== 'Company.Vitals') { updateInventory(); }
        updateCamp();
    }

    // --- Inventory (Phase 32g) ---

    function send(cmd) {
        Client.SendInput(cmd);
    }

    // quoted is a name as one argument, for a command that takes the last
    // word as its target (give).
    function quoted(name) {
        return '"' + String(name).replace(/"/g, '') + '"';
    }

    function button(text, cmd, title) {
        const b = el('button', 'cmp-btn', text);
        b.type = 'button';
        b.setAttribute('data-focus', 'btn|' + text);
        if (title) { b.title = title; }
        b.addEventListener('click', () => send(cmd));
        return b;
    }

    function usesText(i) {
        return i.uses_max > 1 && i.uses > 0 ? i.uses + '/' + i.uses_max : '';
    }

    // label is how players see an item; name, the plain name a command
    // matches (32g review finding 3).
    function label(i) {
        return i.label || i.name;
    }

    function itemTip(i) {
        const parts = [label(i), CompanyData.kg(i.grams) + (i.count > 1 ? ' each' : '')];
        if (usesText(i)) { parts.push(i.uses + ' of ' + i.uses_max + ' uses left'); }
        return parts.join(' \u00b7 ');
    }

    // presentCompanions are the companions out with the player, who can be
    // handed things.
    function presentCompanions() {
        const data = CompanyData.read();
        return data.members.filter(m => m && m.key !== 'leader' && m.status === 'present');
    }

    function yourItemMenu(i, worn) {
        const items = [{ label: 'look ' + i.name, cmd: 'look ' + i.ref }];
        if (worn) {
            items.push({ label: 'remove ' + i.name, cmd: 'remove ' + i.ref });
            return items;
        }
        const type = (i.type || '').toLowerCase(), sub = (i.subtype || '').toLowerCase();
        if (type === 'weapon' || sub === 'wearable') { items.push({ label: 'equip ' + i.name, cmd: 'equip ' + i.ref }); }
        if (sub === 'edible')    { items.push({ label: 'eat ' + i.name, cmd: 'eat ' + i.ref }); }
        if (sub === 'drinkable') { items.push({ label: 'drink ' + i.name, cmd: 'drink ' + i.ref }); }
        items.push({ label: 'Put in cargo', cmd: 'cargo put ' + i.ref });
        presentCompanions().forEach(m => {
            items.push({ label: 'Give to ' + m.name, cmd: 'give ' + i.ref + ' ' + quoted(m.name) });
        });
        return items;
    }

    function memberSelector(key) {
        return key === 'leader' ? 'leader' : '#' + String(key).replace('companion:', '');
    }

    function sharedCargoMenu(i, inv) {
        const menu = [{ label: 'Look', cmd: 'look ' + i.ref }];
        const wearable = i.type === 'weapon' || i.subtype === 'wearable';
        if (wearable) {
            inv.members.filter(m => m.key === 'leader' || m.available).forEach(m => {
                const member = memberSelector(m.key);
                menu.push({ label: 'Compare for ' + m.name, cmd: 'company compare ' + member + ' ' + i.ref });
                menu.push({ label: 'Equip ' + m.name, cmd: 'company equip ' + member + ' ' + i.ref });
            });
        }
        if (i.subtype === 'edible') { menu.push({ label: 'Eat', cmd: 'eat ' + i.ref }); }
        if (i.subtype === 'drinkable') { menu.push({ label: 'Drink', cmd: 'drink ' + i.ref }); }
        if (i.subtype === 'usable') { menu.push({ label: 'Use', cmd: 'use ' + i.ref }); }
        return menu;
    }

    function itemRow(i, menu) {
        const li = el('li');
        const row = el(menu ? 'button' : 'div', 'cmp-item');
        if (menu) {
            row.type = 'button';
            row.setAttribute('aria-haspopup', 'menu');
            row.addEventListener('click', e => uiMenu(e, menu()));
        }
        row.appendChild(el('span', 'cmp-name', label(i) + (i.count > 1 ? ' x' + i.count : '')));
        if (menu) { row.setAttribute('data-focus', (i.ref || '') + '|' + label(i)); }
        row.appendChild(el('span', 'cmp-meta', [usesText(i), CompanyData.kg(i.grams * Math.max(1, i.count || 1))].filter(Boolean).join(' \u00b7 ')));
        row.title = itemTip(i);
        li.appendChild(row);
        return li;
    }

    function itemList(items, menuFor) {
        const ul = el('ul', 'cmp-items');
        items.forEach(i => ul.appendChild(itemRow(i, menuFor ? () => menuFor(i) : null)));
        return ul;
    }

    function memberBlock(m, isYou, shared) {
        const block = el('section', 'cmp-block');
        block.setAttribute('aria-label', isYou ? m.name + ' (you)' : m.name);
        const h = el('h4');
        h.appendChild(el('span', null, m.name + (isYou ? ' (you)' : '')));
        h.appendChild(el('span', 'cmp-weight', m.fallen ? 'fallen' : CompanyData.kg(m.grams)));
        block.appendChild(h);
        if (m.fallen) {
            block.appendChild(el('div', 'cmp-note', 'Fallen: their gear is with the body.'));
            return block;
        }
        if (m.unrecorded) {
            block.appendChild(el('div', 'cmp-note', 'Gear not yet recorded.'));
            return block;
        }
        if (!shared) { block.appendChild(el('div', 'cmp-sub', m.pack ? 'Pack: ' + m.pack + ' (+' + CompanyData.kg(m.pack_bonus_g) + ')' : 'No pack')); }
        block.appendChild(el('div', 'cmp-sub', 'Wearing'));
        block.appendChild(m.worn.length ? itemList(m.worn, shared && (isYou || m.available) ? (i => [{ label: 'Remove to cargo', cmd: 'company remove ' + memberSelector(m.key) + ' ' + i.slot }]) : (isYou ? (i => yourItemMenu(i, true)) : null)) : el('div', 'cmp-note', 'nothing'));
        if (!shared) { block.appendChild(el('div', 'cmp-sub', 'Carrying'));
        block.appendChild(m.carried.length ? itemList(m.carried, isYou ? (i => yourItemMenu(i, false)) : null) : el('div', 'cmp-note', 'nothing')); }
        return block;
    }

    function horseMenu(h, yourItems) {
        const items = [];
        yourItems.filter(i => /saddle/i.test(i.name)).forEach(i => {
            items.push({ label: 'Fit ' + label(i), cmd: 'mount saddle #' + h.id + ' ' + i.ref });
        });
        if (h.saddle) { items.push({ label: 'Unsaddle', cmd: 'mount unsaddle #' + h.id }); }
        // A released horse is gone for good: ask first.
        items.push({ label: 'Release\u2026', cmd: 'mount release #' + h.id,
            confirm: 'Release #' + h.id + ' ' + h.name + '? Its saddle comes back to you; the gold doesn\'t.' });
        return items;
    }

    function horseRow(h, yourItems) {
        const li = el('li');
        const row = el('button', 'cmp-item');
        row.type = 'button';
        row.setAttribute('aria-haspopup', 'menu');
        row.appendChild(el('span', 'cmp-name', '#' + h.id + ' ' + h.name));
        const what = [h.saddle || 'no saddle'];
        if (h.rides) { what.push('carries a rider'); } else if (h.capacity_g > 0) { what.push('+' + CompanyData.kg(h.capacity_g)); }
        row.appendChild(el('span', 'cmp-meta', what.join(', ')));
        row.title = '#' + h.id + ' ' + h.name + ': ' + what.join(', ');
        row.setAttribute('data-focus', 'horse|' + h.id);
        row.addEventListener('click', e => uiMenu(e, horseMenu(h, yourItems)));
        li.appendChild(row);
        return li;
    }

    // keepFocus rebuilds a panel, keeping keyboard focus on the same
    // control by its data-focus (32g review finding 8).
    function keepFocus(panel, rebuild) {
        const focused = document.activeElement;
        const key = focused && panel.contains(focused) ? focused.getAttribute('data-focus') : null;
        rebuild();
        if (!key) { return; }
        const again = [...panel.querySelectorAll('[data-focus]')].find(n => n.getAttribute('data-focus') === key);
        if (again) { again.focus(); }
    }

    function updateInventory() {
        const panel = document.getElementById('company-inventory');
        if (!panel) { return; }
        keepFocus(panel, () => buildInventory(panel));
    }

    function buildInventory(panel) {
        const inv = Client.GMCPStructs.Company && Client.GMCPStructs.Company.Inventory;
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        if (!inv || !Array.isArray(inv.members)) {
            pad.appendChild(el('div', 'cmp-note', 'Nothing yet.'));
            return;
        }
        if (inv.load) {
            const l = inv.load;
            const pct = l.capacity_g > 0 ? Math.round(l.total_g * 100 / l.capacity_g) : (l.total_g > 0 ? 100 : 0);
            pad.appendChild(el('div', null, 'Load ' + CompanyData.kg(l.total_g) + ' / ' + CompanyData.kg(l.capacity_g) + ' (' + pct + '%) — ' + (l.total_g > l.capacity_g ? 'Overloaded' : (l.label || 'Within capacity'))));
            const meter = el('div', 'cmp-meter' + (pct >= 100 ? ' full' : ''));
            meter.setAttribute('role', 'meter');
            meter.setAttribute('aria-label', 'Load ' + pct + ' percent of capacity');
            meter.setAttribute('aria-valuemin', '0');
            meter.setAttribute('aria-valuemax', '100');
            meter.setAttribute('aria-valuenow', String(Math.min(pct, 100)));
            const fill = el('div');
            fill.style.width = Math.min(pct, 100) + '%';
            meter.appendChild(fill);
            pad.appendChild(meter);
            pad.appendChild(el('div', 'cmp-line', 'Capacity: assigned packs ' + CompanyData.kg(l.member_capacity_g) + ', horses ' + CompanyData.kg(l.mount_capacity_g) + '. Cargo ' + CompanyData.kg(l.cargo_g) + '.'));
        }
        const actions = el('div', 'cmp-actions');
        actions.appendChild(button('Meal', 'company meal', 'Everyone with you eats and drinks (company meal)'));
        actions.appendChild(button('Eat', 'company eat', 'Everyone hungry eats (company eat)'));
        actions.appendChild(button('Drink', 'company drink', 'Everyone thirsty drinks (company drink)'));
        if (inv.shared) {
            pad.appendChild(el('div', 'cmp-line', 'Company treasury: ' + inv.treasury + ' gold'));
            actions.appendChild(button('Loot', 'loot', 'Collect eligible battle spoils'));
            actions.appendChild(button(inv.autoloot ? 'Autoloot off' : 'Autoloot on', inv.autoloot ? 'autoloot off' : 'autoloot on', 'Toggle automatic loot after battle'));
        }
        pad.appendChild(actions);

        if (inv.shared) {
            const containers = el('section', 'cmp-block');
            containers.setAttribute('aria-label', 'Assigned packs');
            containers.appendChild(el('h4', null, 'Containers'));
            (inv.containers || []).forEach(c => {
                const card = el('div', 'cmp-line', c.name + ' (' + c.carrier + ') — ' + CompanyData.kg(c.capacity_g) + (c.available ? ' available' : ' unavailable'));
                containers.appendChild(card);
            });
            if (!(inv.containers || []).length) { containers.appendChild(el('div', 'cmp-note', 'No assigned packs. Equip a pack from shared cargo (help pack).')); }
            pad.appendChild(containers);
        }
        const you = inv.members[0];
        if (!inv.shared) { inv.members.forEach((m, idx) => pad.appendChild(memberBlock(m, idx === 0, false))); }

        const horses = el('section', 'cmp-block');
        horses.setAttribute('aria-label', 'Horses');
        horses.appendChild(el('h4', null, 'Horses'));
        if (inv.horses && inv.horses.length) {
            const ul = el('ul', 'cmp-items');
            inv.horses.forEach(h => ul.appendChild(horseRow(h, (inv.shared ? inv.cargo : (you && you.carried)) || [])));
            horses.appendChild(ul);
        } else {
            horses.appendChild(el('div', 'cmp-note', 'none (help mount)'));
        }
        pad.appendChild(horses);

        const cargo = el('section', 'cmp-block');
        cargo.setAttribute('aria-label', 'Cargo');
        const ch = el('h4');
        ch.appendChild(el('span', null, 'Cargo'));
        if (inv.load) { ch.appendChild(el('span', 'cmp-weight', CompanyData.kg(inv.load.cargo_g))); }
        cargo.appendChild(ch);
        cargo.appendChild(inv.cargo && inv.cargo.length
            ? itemList(inv.cargo, i => inv.shared ? sharedCargoMenu(i, inv) : [{ label: 'Take one', cmd: 'cargo take ' + i.ref }])
            : el('div', 'cmp-note', 'empty'));
        pad.appendChild(cargo);
    }

    // --- Camp (Phase 32g) ---

    function updateCamp() {
        const panel = document.getElementById('company-camp');
        if (!panel) { return; }
        keepFocus(panel, () => buildCamp(panel));
    }

    function buildCamp(panel) {
        const camp = (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Camp) || {};
        const data = CompanyData.read();
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);

        let where = 'No camp.';
        if (camp.has_camp && camp.here) {
            where = 'Your camp is here' + (camp.fire_lit ? ', around a lit fire.' : (camp.embers ? ', around glowing embers.' : '; the fire is cold.'));
        } else if (camp.has_camp) {
            where = 'Your camp is at ' + (camp.room || 'another place') + '.';
        }
        pad.appendChild(el('div', null, where));
        if (camp.has_camp && camp.here && camp.tent) {
            pad.appendChild(el('div', 'cmp-line', 'An oiled canvas tent is pitched here: shelter, and no cold while you rest.'));
        }
        if (camp.rested && camp.here && !camp.resting) {
            pad.appendChild(el('div', 'cmp-line', camp.embers
                ? 'Your company has rested. The fire has burned to embers: feed it (Light fire) to rest again.'
                : 'Your company has rested at this camp.'));
        }
        if (camp.resting) {
            pad.appendChild(el('div', null, 'Resting: ' + CompanyData.formatSeconds(camp.rest_seconds) + ' left.'));
            const meter = el('div', 'cmp-meter');
            meter.setAttribute('role', 'progressbar');
            meter.setAttribute('aria-label', 'Rest');
            meter.setAttribute('aria-valuemin', '0');
            meter.setAttribute('aria-valuemax', '100');
            meter.setAttribute('aria-valuenow', String(camp.rest_percent || 0));
            const fill = el('div');
            fill.style.width = (camp.rest_percent || 0) + '%';
            meter.appendChild(fill);
            pad.appendChild(meter);
        }
        if (data.live && data.live.rest && data.live.rest.tier && data.live.rest.tier !== 'none') {
            pad.appendChild(el('div', 'cmp-line', data.live.rest.tier + ' for ' + CompanyData.formatSeconds(data.live.rest.seconds) + '.'));
        }

        // Each button only when it would work.
        const actions = el('div', 'cmp-actions');
        if (camp.can_camp) { actions.appendChild(button('Make camp', 'camp', 'Make camp here (camp)')); }
        if (camp.has_camp && camp.here && !camp.fire_lit) { actions.appendChild(button(camp.embers ? 'Feed fire' : 'Light fire', 'camp fire', camp.embers ? 'Feed the embers fuel (camp fire)' : 'Light the campfire (camp fire)')); }
        if (camp.has_camp && camp.here && camp.fire_lit && !camp.resting) { actions.appendChild(button('Rest', 'camp rest', 'Rest by the fire (camp rest)')); }
        if (camp.has_camp && camp.here && !camp.resting) { actions.appendChild(button('Break camp', 'camp break', 'Strike the camp (camp break)')); }
        actions.appendChild(button('Meal', 'company meal', 'Everyone with you eats and drinks (company meal)'));
        if (camp.inn) { actions.appendChild(button('Inn', 'inn', 'This inn\'s price and your stay (inn)')); }
        pad.appendChild(actions);

        const members = data.members.filter(m => m && m.status !== 'dead');
        if (members.length) {
            const table = el('table', 'cmp-needs');
            table.appendChild(el('caption', 'cmp-line', 'Needs'));
            const head = el('tr');
            ['', 'Hunger', 'Thirst', 'Fatigue'].forEach(t => { const th = el('th', null, t); th.setAttribute('scope', 'col'); head.appendChild(th); });
            table.appendChild(head);
            members.forEach(m => {
                const tr = el('tr');
                const th = el('th', null, m.key === 'leader' ? m.name + ' (you)' : m.name);
                th.setAttribute('scope', 'row');
                tr.appendChild(th);
                const needs = data.vitals(m.key).needs || {};
                ['hunger', 'thirst', 'fatigue'].forEach(k => {
                    const n = needs[k];
                    tr.appendChild(el('td', n && n.warn ? 'need-warn' : null, n ? n.label : '\u2014'));
                });
                table.appendChild(tr);
            });
            pad.appendChild(table);
        }
    }

    VirtualWindows.register({
        window:       win,
        // handleGMCP calls a handler once per matching level; registering
        // only the top names gives one call per payload.
        gmcpHandlers: ['Company', 'Party'],
        onGMCP(namespace) {
            if (namespace === 'Company.Inventory') {
                win.open();
                if (win.isOpen()) { updateInventory(); }
                return;
            }
            if (namespace === 'Company.Camp') {
                win.open();
                if (win.isOpen()) { updateCamp(); }
                return;
            }
            update(namespace);
        },
    });

})();
