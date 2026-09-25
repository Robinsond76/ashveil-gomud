/**
 * window-party.js
 *
 * Virtual window: Party.
 * Two sections, each with its own heading so they are never confused:
 *
 *   Company  - the player's Ashveil company (Phase 26b): a 3x3 formation
 *              table and a card per member (health, needs, chemistry, and a
 *              fallen member's time left to raise).
 *   Players  - GoMud's human party: members with level, rank, location, and
 *              a health bar.
 *
 * Every name and label is set with textContent, never innerHTML.
 *
 * Responds to GMCP namespaces:
 *   Company         - full company snapshot ({} when there is none)
 *   Company.Vitals  - members' health and needs only; merged, never replacing
 *   Party           - full party update (roster + vitals)
 *   Party.Vitals    - lightweight vitals-only update
 *   Char            - used once, to ask for the company after login
 */

'use strict';

(function() {

    injectStyles(`
        #party-panel {
            height: 100%;
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
            color: var(--t-text-dim);
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

        .company-formation {
            border-collapse: collapse;
            font-size: 0.68em;
            table-layout: fixed;
            width: 100%;
        }

        .company-formation caption {
            text-align: left;
            color: var(--t-text-dim);
            padding-bottom: 2px;
        }

        .company-formation th {
            width: 3.2em;
            font-weight: normal;
            color: var(--t-text-dim);
            text-align: left;
        }

        .company-formation td {
            border: 1px solid var(--t-accent-dim);
            padding: 2px 3px;
            text-align: center;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .company-formation td.empty { color: var(--t-text-dim); }

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
    `);

    function hpClass(pct) {
        if (pct >= 60) { return 'high'; }
        if (pct >= 30) { return 'medium'; }
        return 'low';
    }

    // el builds an element with an optional class and text, safely.
    function el(tag, className, text) {
        const node = document.createElement(tag);
        if (className) { node.className = className; }
        if (text !== undefined && text !== null) { node.textContent = String(text); }
        return node;
    }

    function formatSeconds(seconds) {
        seconds = Math.max(0, Math.floor(seconds || 0));
        if (seconds < 60) { return seconds + 's'; }
        if (seconds < 3600) { return Math.floor(seconds / 60) + 'm'; }
        return Math.floor(seconds / 3600) + 'h ' + Math.floor((seconds % 3600) / 60) + 'm';
    }

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

    function createDOM() {
        const root = el('div');
        root.id = 'party-panel';
        root.appendChild(el('div', 'party-empty', 'No company or party'));
        document.body.appendChild(root);
        return root;
    }

    const win = new VirtualWindow('Party', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  200,
        factory() {
            const root = createDOM();
            return {
                title:      'Party',
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
    let company          = null; // the last Company snapshot, or null
    let companyVitals    = {};   // member key -> vitals, merged from Company.Vitals
    let companyRequested = false;

    function applyCompany(namespace, body) {
        if (namespace === 'Company.Vitals') {
            if (!company || !body || typeof body.vitals !== 'object' || body.vitals === null) { return; }
            Object.keys(body.vitals).forEach(key => { companyVitals[key] = body.vitals[key]; });
            return;
        }
        if (!body || !body.leader) {
            company       = null;
            companyVitals = {};
            return;
        }
        company       = body;
        companyVitals = Object.assign({}, body.vitals || {});
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
        if (company.activity) { parts.push(company.activity); }
        if (company.rest && company.rest.tier && company.rest.tier !== 'none') {
            parts.push(company.rest.tier + ' ' + formatSeconds(company.rest.seconds));
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
                const td = el('td', m ? 'filled' : 'empty', m ? m.name : '\u00b7');
                if (m) { td.title = m.name; } else { td.setAttribute('aria-label', 'empty'); }
                tr.appendChild(td);
            });
            table.appendChild(tr);
        });
        return table;
    }

    function memberCard(m) {
        const v    = companyVitals[m.key] || {};
        const card = el('li', 'party-member company-member' + (m.key === 'leader' ? ' is-leader' : '') + ' status-' + m.status);
        card.tabIndex = 0;

        const header = el('div', 'party-member-header');
        header.appendChild(el('span', 'party-member-name', m.name + (m.key === 'leader' ? ' \u2605' : '')));
        if (m.level) { header.appendChild(el('span', 'party-member-level', 'Lv ' + m.level)); }
        if (m.archetype) { header.appendChild(el('span', 'party-member-rank', m.archetype)); }
        card.appendChild(header);

        const spoken = [m.name, m.level ? 'level ' + m.level : '', m.archetype || ''];
        if (m.status === 'dead') {
            const left = 'Fallen: ' + formatSeconds(m.rescue_seconds) + ' to raise';
            card.appendChild(el('div', 'company-status company-fallen', left));
            spoken.push(left);
        } else if (m.status === 'awaiting') {
            card.appendChild(el('div', 'company-status', 'Away: rejoins when you return'));
            spoken.push('away');
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
        if (m.chemistry && m.chemistry !== 'none') {
            card.appendChild(el('div', 'company-chemistry', 'Chemistry: ' + m.chemistry));
            spoken.push('chemistry ' + m.chemistry);
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
        section.setAttribute('aria-label', 'Players');
        section.appendChild(el('h3', 'panel-heading', 'Players'));

        const hasMembers = members.length > 0 || Object.keys(vitals).length > 0;
        if (!hasMembers) {
            section.appendChild(el('div', 'party-empty', 'Not in a party'));
            return section;
        }

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

    function update() {
        win.open();
        if (!win.isOpen()) { return; }

        const panel = document.getElementById('party-panel');
        if (!panel) { return; }
        const partyData = Client.GMCPStructs.Party;
        const hasParty  = !!(partyData && ((partyData.Members && partyData.Members.length) || (partyData.Vitals && Object.keys(partyData.Vitals).length)));

        panel.textContent = '';
        if (!company && !hasParty) {
            panel.appendChild(el('div', 'party-empty', 'No company or party'));
            return;
        }
        if (company) { panel.appendChild(companySection()); }
        panel.appendChild(playersSection(partyData));
    }

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Company', 'Party', 'Char'],
        onGMCP(namespace, body) {
            // handleGMCP calls this once per matching level, so each
            // namespace is acted on only for its own full name.
            if (namespace === 'Company' || namespace === 'Company.Vitals') {
                applyCompany(namespace, body);
                update();
                return;
            }
            if (namespace.indexOf('Char') === 0) {
                // Logged in: ask once for the company, in case a login
                // snapshot was missed (a reloaded tab, a reconnect).
                if (!companyRequested && Client.GMCPRequest) {
                    companyRequested = true;
                    Client.GMCPRequest('Company');
                }
                return;
            }
            update();
        },
    });

})();
