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
 *   Chronicle - the company's deeds in prose, newest first (Phase 63), with
 *               a filter by kind; `chronicle` reads the same in text. Above
 *               them, what the towns have said of the company (Phase 68;
 *               `townsfolk` reads the same).
 *   Opinions  - what each companion likes, dislikes and has lately said
 *               about the leader's choices (Phase 64); `opinions` reads the
 *               same in text.
 *   Bonds     - how each pair of companions feels about the other, and what
 *               that does in a battle (Phase 65); `bonds` reads the same in
 *               text.
 *   Errands   - who is away on an errand and when they are due, and a card
 *               to send each free companion (Phase 70); `errands` reads the
 *               same in text.
 *   Bounties  - the bounty board the leader stands at and the bounties the
 *               company holds, with progress (Phase 76); `bounty` reads the
 *               same in text.
 *   Rites     - the companions the company has lost and not yet mourned,
 *               with Hold and Let pass buttons at a camp or an inn
 *               (Phase 74); `rites` reads the same in text.
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
 *   Company.Chronicle - the Chronicle sub-tab: { total, tally: {kind: n},
 *                       entries: [{seq, at, ago, kind, label, text}] }
 *   Company.Townsfolk - the Chronicle sub-tab's town memory: { total, told:
 *                       [{ago, text}], fresh: [{ago, kind, text}], marks: [] }
 *   Company.Opinions  - the Opinions sub-tab: { spared, executed, members:
 *                       [{key, id, name, personality, loyalty, mood, likes,
 *                       dislikes, notes: [{label, verdict, ago, subject}],
 *                       deeds}] }
 *   Company.Errands   - the Errands sub-tab: { now, here, where, zone, band,
 *                       rows: [{id, name, level, state: away|ready|busy, why,
 *                       kind_label, zone, returns_at, remaining, due, waiting}],
 *                       options: [{kind, label, blurb, lengths: [{length,
 *                       label}]}], recent: [string] }
 *   Company.Rites     - the Rites sub-tab: { here, where, rows: [{id, name,
 *                       level, cause, close: [name], offered}] }
 *   Company.Bounties  - the Bounties sub-tab: { now, at_board, board, band,
 *                       rotates, max, postings: [{n, kind, name, zone, band,
 *                       rating, count, reward, taken, done}], held: [{n, kind,
 *                       name, zone, have, count, reward, left, ready}] }
 *   Company.Bonds     - the Bonds sub-tab: { pairs: [{a, b, a_name, b_name,
 *                       value, tier, phrase, effect, warned}], members: [{id,
 *                       name, feelings: [{id, name, words, tier}]}] }
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
            /* Phase 74: eight tabs outgrow a phone, so the bar scrolls. */
            overflow-x: auto;
            scrollbar-width: none;
        }
        .cmp-tab-bar::-webkit-scrollbar { display: none; }

        .cmp-tab-btn {
            flex: 1 0 auto;
            padding: 5px 8px;
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
        @media (hover: hover) and (pointer: fine) {
            .cmp-tab-btn:hover { background: var(--t-border); color: var(--t-text); }
        }
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

        .cmp-held { color: var(--t-text); border-left: 3px solid var(--t-accent); padding-left: 6px; }

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

        @media (hover: hover) and (pointer: fine) {
            .cmp-btn:hover { background: var(--t-bg-hover); color: var(--t-text); box-shadow: inset 0 0 0 1px var(--t-accent); }
        }
        .cmp-btn:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }
        .cmp-btn[aria-pressed="true"] { box-shadow: inset 0 0 0 2px var(--t-accent); font-weight: bold; }
        .cmp-btn[disabled] { opacity: 0.6; cursor: default; }
        .cmp-duty { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; margin-top: 6px; }
        .cmp-duty-name { flex-basis: 100%; }

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

        .cmp-chron-filter { display: flex; flex-wrap: wrap; gap: 4px; }
        .cmp-chron-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
        .cmp-chron-item { display: flex; flex-direction: column; gap: 1px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-chron-meta { color: var(--t-text-secondary); font-size: 0.9em; }
        .cmp-chron-text { color: var(--t-text); }
        .cmp-town { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
        .cmp-town-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
        .cmp-town-item { display: flex; flex-direction: column; gap: 1px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-town-meta { color: var(--t-text-secondary); font-size: 0.9em; }
        .cmp-town-text { color: var(--t-text); font-style: italic; }
        .cmp-opn-card { display: flex; flex-direction: column; gap: 2px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-opn-head { font-weight: bold; }
        .cmp-opn-mood { color: var(--t-text-secondary); font-weight: normal; }
        .cmp-opn-yes { color: var(--t-text); }
        .cmp-opn-no { color: var(--t-text-secondary); }
        .cmp-bond-card { display: flex; flex-direction: column; gap: 3px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-bond-card[data-feel="rival"] { border-left-color: #b5584f; }
        .cmp-bond-head { font-weight: bold; }
        .cmp-bond-num { color: var(--t-text-secondary); font-weight: normal; }
        .cmp-bond-track { position: relative; height: 6px; background: rgba(128, 128, 128, 0.25); border-radius: 3px; }
        .cmp-bond-mid { position: absolute; left: 50%; top: -1px; bottom: -1px; width: 1px; background: var(--t-text-secondary); }
        .cmp-bond-fill { position: absolute; top: 0; bottom: 0; border-radius: 3px; background: var(--t-accent); }
        .cmp-bond-card[data-feel="rival"] .cmp-bond-fill { background: #b5584f; }
        .cmp-err-card { display: flex; flex-direction: column; gap: 3px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-err-card[data-state="away"] { border-left-color: var(--t-accent); }
        .cmp-err-head { font-weight: bold; }
        .cmp-rite-card { display: flex; flex-direction: column; gap: 3px; border-left: 3px solid var(--t-accent-dim); padding-left: 6px; overflow-wrap: anywhere; }
        .cmp-rite-card[data-offered="1"] { border-left-color: var(--t-accent); }
        .cmp-rite-btns { display: flex; flex-wrap: wrap; gap: 6px; }
        .cmp-err-sub { color: var(--t-text-secondary); font-weight: normal; }
        .cmp-err-pick { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
        .cmp-err-pick select { max-width: 100%; }

        .cmp-recipes summary { cursor: pointer; font-weight: bold; }
        .cmp-recipes ul { margin: 4px 0; padding-left: 18px; }
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
        /* Phase 48: a member's equipment slots; drop targets for cargo */
        .cmp-equip-member { margin-bottom: 6px; }
        .cmp-equip-member h5 { margin: 4px 0 2px; font-size: 1em; display: flex; justify-content: space-between; gap: 6px; }
        .cmp-slot .cmp-name { overflow-wrap: anywhere; flex: 1; text-align: left; }
        .cmp-slot .cmp-slotname { color: var(--t-text-secondary); min-width: 4.5em; }
        .cmp-slot.empty .cmp-name { color: var(--t-text-secondary); font-style: italic; }
        .cmp-drop { outline: 2px dashed var(--t-accent); outline-offset: -2px; }
        .cmp-away { opacity: 0.6; }
        @media (hover: hover) and (pointer: fine) {
            button.cmp-item:hover { background: var(--t-bg-hover); color: var(--t-text); box-shadow: inset 0 0 0 1px var(--t-accent); }
            /* Phase 57: every part of a highlighted row stays readable */
            button.cmp-item:hover .cmp-meta, button.cmp-item:hover .cmp-slotname { color: inherit; }
        }
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
        .company-formation td.fallen { color: var(--t-text-secondary); font-style: italic; }

        /* Click a member, then a place: the formation's cells are buttons. */
        .company-formation td { padding: 0; }
        .fm-cell {
            display: block;
            width: 100%;
            min-height: 2.2em;
            padding: 3px 4px;
            font: inherit;
            color: inherit;
            background: transparent;
            border: 0;
            cursor: pointer;
            overflow-wrap: anywhere;
            line-height: 1.25;
        }
        @media (hover: hover) { .fm-cell:hover { background: var(--t-bg-surface-alt); } }
        .fm-cell:focus-visible { outline: 2px solid var(--t-accent); outline-offset: -2px; }
        .company-formation td.is-moving { outline: 2px solid var(--t-accent); outline-offset: -2px; background: var(--t-bg-surface-alt); }
        .company-formation.is-picking td.empty .fm-cell,
        .company-formation.is-picking td.filled:not(.is-moving) .fm-cell { border: 1px dashed var(--t-accent); }
        .fm-help { font-size: 0.72em; color: var(--t-text-secondary); margin: 3px 0 0; min-height: 1.2em; }
        .fm-help.is-warn { color: var(--t-warn, var(--t-text)); font-weight: bold; }
        .fm-unplaced { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 3px; font-size: 0.78em; align-items: center; }
        .fm-unplaced .fm-cell { display: inline-block; width: auto; min-height: 0; border: 1px solid var(--t-accent-dim); }
        .fm-unplaced .fm-cell.is-moving { outline: 2px solid var(--t-accent); }
        /* Phone: 44px targets. */
        body.mobile .company-formation .fm-cell, body.mobile .fm-unplaced .fm-cell { min-height: 44px; }
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
        { id: 'company-chronicle', label: 'Chronicle' },
        { id: 'company-opinions',  label: 'Opinions' },
        { id: 'company-bonds',     label: 'Bonds' },
        { id: 'company-errands',   label: 'Errands' },
        { id: 'company-rites',     label: 'Rites' },
        { id: 'company-bounties',  label: 'Bounties' },
    ];
    const SUBTAB_KEY = 'companySubTab';

    function showSubtab(root, id, remember) {
        if (!SUBTABS.some(t => t.id === id)) { id = SUBTABS[0].id; }
        root.querySelectorAll('.cmp-tab-btn').forEach(b => {
            const on = b.dataset.panel === id;
            b.classList.toggle('active', on);
            b.setAttribute('aria-selected', on ? 'true' : 'false');
            if (on && b.scrollIntoView) { b.scrollIntoView({ block: 'nearest', inline: 'nearest' }); }
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
        // The bar's scrollbar is hidden, so a mouse wheel scrolls it sideways
        // when the tabs overflow a narrow dock (Phase 74 review).
        bar.addEventListener('wheel', e => {
            if (bar.scrollWidth <= bar.clientWidth || Math.abs(e.deltaY) <= Math.abs(e.deltaX)) { return; }
            bar.scrollLeft += e.deltaY;
            e.preventDefault();
        }, { passive: false });
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

    // Phase 58: click (or tap) a member in the drawing, then an empty cell
    // to move them there, or another member to swap places. The commands
    // are `formation move` / `formation swap`, so the server's rules stand
    // (no changes in a battle, no moving the fallen, a solo leader stays at
    // the centre) and its answer shows in the game text.
    let movingKey = null;      // the member picked up, by key
    let formationNote = '';    // the line under the drawing
    let formationWarn = false;

    function whoArg(m) {
        return m.key === 'leader' ? 'me' : '#' + m.id;
    }

    function inBattleNow() {
        const stored = Client.GMCPStructs.Company;
        const b = stored && stored.Battle;
        return !!(b && Array.isArray(b.enemies));
    }

    function setFormationNote(text, warn) {
        formationNote = text || '';
        formationWarn = !!warn;
        update('local');
    }

    function pickMember(m, members) {
        if (movingKey === m.key) {
            movingKey = null;
            setFormationNote('');
            return;
        }
        if (inBattleNow()) {
            setFormationNote('The battle is under way: it plays out as you set it up. Move people before it starts.', true);
            return;
        }
        if (m.status === 'dead') {
            setFormationNote(m.name + ' has fallen and can\'t be moved until raised.', true);
            return;
        }
        if (members.length < 2) {
            setFormationNote('You stand at the centre alone. Recruit a companion first (help company).', true);
            return;
        }
        movingKey = m.key;
        setFormationNote('');
    }

    function placeMover(target, members) {
        const mover = members.find(x => x.key === movingKey);
        if (!mover) { movingKey = null; setFormationNote(''); return; }
        if (inBattleNow()) {
            movingKey = null;
            setFormationNote('The battle is under way: it plays out as you set it up. Move people before it starts.', true);
            return;
        }
        let cmd;
        let said;
        if (target.member) {
            if (target.member.status === 'dead') {
                setFormationNote(target.member.name + ' has fallen: pick another place.', true);
                return;
            }
            // `formation swap` needs both members placed: an unplaced
            // member is picked up instead, and one being placed needs an
            // empty cell.
            if (!target.member.cell) {
                pickMember(target.member, members);
                return;
            }
            if (!mover.cell) {
                setFormationNote('Pick an empty cell to place ' + mover.name + '.', true);
                return;
            }
            cmd = 'formation swap ' + whoArg(mover) + ' ' + whoArg(target.member);
            said = 'Swapping ' + mover.name + ' with ' + target.member.name + '.';
        } else {
            cmd = 'formation move ' + whoArg(mover) + ' ' + (target.row + 1) + ' ' + (target.col + 1);
            said = 'Moving ' + mover.name + ' to row ' + (target.row + 1) + ', column ' + (target.col + 1) + '.';
        }
        movingKey = null;
        formationNote = said;
        formationWarn = false;
        send(cmd);
        update('local');
    }

    function formationTable(members) {
        const grid = [[null, null, null], [null, null, null], [null, null, null]];
        members.forEach(m => {
            const c = m && m.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) { grid[c.row][c.col] = m; }
        });
        const mover = movingKey && members.find(x => x.key === movingKey);
        if (movingKey && !mover) { movingKey = null; }

        const wrap = el('div', 'company-formation-wrap');
        const table = el('table', 'company-formation' + (mover ? ' is-picking' : ''));
        table.appendChild(el('caption', null, 'Formation (row 1 is the front)'));
        wrap.addEventListener('keydown', e => {
            if (e.key === 'Escape' && movingKey) {
                e.preventDefault();
                movingKey = null;
                setFormationNote('Move cancelled.');
            }
        });
        grid.forEach((row, r) => {
            const tr = el('tr');
            const th = el('th', null, 'Row ' + (r + 1));
            th.setAttribute('scope', 'row');
            tr.appendChild(th);
            row.forEach((m, c) => {
                const cls = m ? 'filled' + (m.key === 'leader' ? ' is-leader' : '') + (m.status === 'dead' ? ' fallen' : '') + (mover && m === mover ? ' is-moving' : '') : 'empty';
                const td = el('td', cls);
                const b = el('button', 'fm-cell', m ? m.name : '\u00b7');
                b.type = 'button';
                b.setAttribute('data-cell', r + ',' + c);
                if (m) {
                    b.title = m.name;
                    if (mover && m === mover) {
                        b.setAttribute('aria-pressed', 'true');
                        b.setAttribute('aria-label', m.name + ', picked up. Choose a place, or press again to cancel.');
                        b.addEventListener('click', () => pickMember(m, members));
                    } else if (mover) {
                        b.setAttribute('aria-label', 'Swap ' + mover.name + ' with ' + m.name);
                        b.addEventListener('click', () => placeMover({ member: m }, members));
                    } else {
                        b.setAttribute('aria-pressed', 'false');
                        b.setAttribute('aria-label', 'Move ' + m.name + ' (row ' + (r + 1) + ', column ' + (c + 1) + ')');
                        b.title = 'Click to move ' + m.name;
                        b.addEventListener('click', () => pickMember(m, members));
                    }
                } else if (mover) {
                    b.setAttribute('aria-label', 'Move ' + mover.name + ' to row ' + (r + 1) + ', column ' + (c + 1));
                    b.addEventListener('click', () => placeMover({ row: r, col: c }, members));
                } else {
                    b.setAttribute('aria-label', 'Empty, row ' + (r + 1) + ', column ' + (c + 1));
                    b.disabled = true;
                    b.style.cursor = 'default';
                }
                td.appendChild(b);
                tr.appendChild(td);
            });
            table.appendChild(tr);
        });
        wrap.appendChild(table);

        // Members the drawing doesn't hold (not placed yet) can be set down too.
        const unplaced = members.filter(m => m && m.key && !m.cell && m.status !== 'dead');
        if (unplaced.length) {
            const row = el('div', 'fm-unplaced');
            row.appendChild(el('span', null, 'Not placed:'));
            unplaced.forEach(m => {
                const b = el('button', 'fm-cell', m.name);
                b.type = 'button';
                b.setAttribute('data-cell', 'u:' + m.key);
                b.setAttribute('aria-pressed', movingKey === m.key ? 'true' : 'false');
                b.setAttribute('aria-label', 'Place ' + m.name + ' in the formation');
                if (movingKey === m.key) { b.classList.add('is-moving'); }
                b.addEventListener('click', () => (mover && mover !== m) ? placeMover({ member: m }, members) : pickMember(m, members));
                row.appendChild(b);
            });
            wrap.appendChild(row);
        }

        let help = formationNote;
        let warn = formationWarn;
        if (!help) {
            help = mover ? 'Moving ' + mover.name + ': pick an empty cell to move there, or a member to swap places. Pick ' + mover.name + ' again (or press Esc) to cancel.'
                : 'Click a member, then a place, to rearrange (help formation).';
        }
        const line = el('div', 'fm-help' + (warn ? ' is-warn' : ''), help);
        line.setAttribute('role', 'status');
        wrap.appendChild(line);
        return wrap;
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
        } else if (m.status === 'fled') {
            card.appendChild(el('div', 'company-status', 'Fled: returns after the battle'));
            spoken.push('fled');
        } else if (m.status === 'separated') {
            card.appendChild(el('div', 'company-status', 'Separated: finding the way back'));
            spoken.push('separated');
        } else if (m.status === 'errand') {
            card.appendChild(el('div', 'company-status', 'Away on an errand (see Errands)'));
            spoken.push('away on an errand');
        }
        if (m.status !== 'dead' && typeof v.hp === 'number' && typeof v.hp_max === 'number') {
            const label = 'Health ' + v.hp + ' of ' + v.hp_max;
            card.appendChild(hpBar(v.hp, v.hp_max, label));
            card.appendChild(el('div', 'company-hp-text', v.hp + '/' + v.hp_max));
            spoken.push(label);
        }
        // Phase 39g: an Alchemist's flask satchel.
        if (m.status !== 'dead' && typeof v.flasks === 'number' && typeof v.flasks_max === 'number') {
            const flasksLabel = 'Flasks ' + v.flasks + ' of ' + v.flasks_max;
            card.appendChild(el('div', 'company-needs ' + (v.flasks === 0 ? 'need-warn' : 'need-ok'), flasksLabel + (v.flasks === 0 ? '!' : '')));
            spoken.push(flasksLabel);
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
        // Phase 50: what those needs and a meal buff do in the next battle (help survival).
        if (m.status !== 'dead' && v.fare) {
            const fare = el('div', 'company-needs company-fare', 'In battle: ' + v.fare);
            fare.title = 'Set as each battle begins: hunger cuts damage, thirst raises damage taken, fatigue cuts hit chance; a cooked meal adds its buff (help survival, help cooking)';
            card.appendChild(fare);
            spoken.push('in battle ' + v.fare);
        }
        if (m.status !== 'dead' && v.warmth) {
            card.appendChild(el('div', 'company-warmth need-warn', v.warmth));
            spoken.push(v.warmth);
        }
        // Camp music: the member's family and level, and practice to the next.
        if (m.status !== 'dead' && m.music) {
            const music = el('div', 'company-chemistry company-music', 'Music: ' + m.music);
            music.title = 'Plays at camp for the rest\'s buffs (help music)';
            card.appendChild(music);
            spoken.push('music ' + m.music);
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
                errand: 'On an errand: effects unknown until they return',
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
        const focusedCell = (focused && panel.contains(focused) && focused.getAttribute('data-cell')) || null;
        const hasParty  = !!(partyData && ((partyData.Members && partyData.Members.length) || (partyData.Vitals && Object.keys(partyData.Vitals).length)));

        keepScroll(panel);

        panel.textContent = '';
        if (company) {
            panel.appendChild(companySection());
        } else {
            panel.appendChild(el('div', 'party-empty', 'You travel alone; help company to recruit.'));
        }
        if (hasParty) { panel.appendChild(playersSection(partyData)); }

        if (focusedCell) {
            const cell = panel.querySelector('[data-cell="' + focusedCell + '"]');
            if (cell && !cell.disabled) { cell.focus(); }
        } else if (focusedKey) {
            const again = panel.querySelector('[data-key="' + (window.CSS && CSS.escape ? CSS.escape(focusedKey) : focusedKey) + '"]');
            if (again) { again.focus(); }
        }
        // The other sub-tabs read the snapshot too: Camp its needs, and
        // Inventory the companions out (names only, so not on vitals). The
        // Chronicle keeps its own payload; it is redrawn here so a window
        // closed and opened again shows it at once.
        if (namespace !== 'Company.Vitals') { updateInventory(); updateChronicle(); updateOpinions(); updateBonds(); updateErrands(); updateRites(); updateBounties(); }
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
        // Phase 36d review: a relic says what it does (help relics).
        const relic = Array.isArray(i.relic) && i.relic.length ? '\n' + i.relic.join('\n') : '';
        return parts.join(' \u00b7 ') + relic;
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

    function itemRow(i, menu, drag) {
        const li = el('li');
        const row = el(menu ? 'button' : 'div', 'cmp-item');
        if (drag) { draggable(row, drag(i)); }
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

    function itemList(items, menuFor, drag) {
        const ul = el('ul', 'cmp-items');
        items.forEach(i => ul.appendChild(itemRow(i, menuFor ? () => menuFor(i) : null, drag)));
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

    // --- Equipment slots (Phase 48) ---

    // fitsSlot: whether a cargo item can go in a slot (the server decides
    // for real; this only offers sensible picks).
    function fitsSlot(i, slot) {
        const type = (i.type || '').toLowerCase();
        return type === slot || (slot === 'offhand' && type === 'weapon');
    }

    // fitsMember: a creature (Phase 38e) wears only gear cut for its
    // species, and nobody else wears that gear.
    function fitsMember(i, m) {
        const cut = Array.isArray(i.worn_by) ? i.worn_by : [];
        return m.species ? cut.indexOf(m.species) >= 0 : cut.length === 0;
    }

    function equipCommand(sel, i, slot) {
        // A slot can be named only with an exact (instance) reference.
        return 'company equip ' + sel + ' ' + i.ref + (i.ref && i.ref.charAt(0) === '!' && i.ref.indexOf(':') > 0 ? ' ' + slot : '');
    }

    function dragRef(e) {
        try { return JSON.parse(e.dataTransfer.getData('application/x-ashveil-gear') || 'null'); } catch (err) { return null; }
    }

    // dropTarget makes node accept a dragged gear item when ok(drag) says so,
    // calling onDrop(drag).
    function dropTarget(node, ok, onDrop) {
        node.addEventListener('dragover', e => {
            const types = Array.from(e.dataTransfer && e.dataTransfer.types || []);
            if (!types.includes('application/x-ashveil-gear')) { return; }
            e.preventDefault();
            node.classList.add('cmp-drop');
        });
        node.addEventListener('dragleave', () => node.classList.remove('cmp-drop'));
        node.addEventListener('drop', e => {
            node.classList.remove('cmp-drop');
            const drag = dragRef(e);
            if (!drag) { return; }
            e.preventDefault();
            // A drop on the wrong spot does nothing; it never falls through
            // to the member or cargo behind it.
            e.stopPropagation();
            if (ok(drag)) { onDrop(drag); }
        });
    }

    function draggable(node, payload) {
        node.draggable = true;
        node.addEventListener('dragstart', e => {
            e.dataTransfer.setData('application/x-ashveil-gear', JSON.stringify(payload));
            e.dataTransfer.effectAllowed = 'move';
        });
    }

    function equipmentSection(inv) {
        const section = el('section', 'cmp-block');
        section.setAttribute('aria-label', 'Equipment');
        section.appendChild(el('h4', null, 'Equipment'));
        section.appendChild(el('div', 'cmp-note', 'Tap a slot to pick from cargo, or drag a cargo item onto a member. Changes are refused in battle.'));
        const slots = Array.isArray(inv.slots) ? inv.slots : [];
        inv.members.forEach((m, idx) => {
            const isYou = idx === 0;
            const sel = memberSelector(m.key);
            const ready = isYou || m.available;
            const box = el('div', 'cmp-equip-member' + (ready ? '' : ' cmp-away'));
            box.setAttribute('role', 'group');
            box.setAttribute('aria-label', m.name + (isYou ? ' (you)' : '') + ' equipment');
            const h = el('h5');
            h.appendChild(el('span', null, m.name + (isYou ? ' (you)' : '')));
            h.appendChild(el('span', 'cmp-weight', m.fallen ? 'fallen' : CompanyData.kg(m.grams)));
            box.appendChild(h);
            if (m.fallen) {
                box.appendChild(el('div', 'cmp-note', 'Fallen: their gear is with the body.'));
                section.appendChild(box);
                return;
            }
            if (m.unrecorded) {
                box.appendChild(el('div', 'cmp-note', 'Gear not yet recorded.'));
                section.appendChild(box);
                return;
            }
            if (!ready) { box.appendChild(el('div', 'cmp-note', 'Away from you: gear can change when they rejoin.')); }
            const ul = el('ul', 'cmp-items');
            const closed = Array.isArray(m.closed) ? m.closed : [];
            slots.filter(slot => closed.indexOf(slot.slot) < 0).forEach(slot => {
                const worn = (m.worn || []).find(w => w.slot === slot.slot);
                const picks = ready ? (inv.cargo || []).filter(i => fitsSlot(i, slot.slot) && fitsMember(i, m)) : [];
                const menu = () => {
                    const out = [];
                    if (worn) {
                        out.push({ label: 'Look', cmd: 'look ' + worn.ref });
                        out.push({ label: 'Remove to cargo', cmd: 'company remove ' + sel + ' ' + slot.slot });
                    }
                    picks.forEach(i => out.push({ label: (worn ? 'Swap in ' : 'Equip ') + label(i), cmd: equipCommand(sel, i, slot.slot) }));
                    if (worn && window.GearEditor) { out.push({ label: 'Compare in Gear', fn: () => window.GearEditor.show(sel, slot.slot) }); }
                    return out;
                };
                const interactive = ready && (worn || picks.length);
                const li = el('li');
                const row = el(interactive ? 'button' : 'div', 'cmp-item cmp-slot' + (worn ? '' : ' empty'));
                if (interactive) {
                    row.type = 'button';
                    row.setAttribute('aria-haspopup', 'menu');
                    row.addEventListener('click', e => uiMenu(e, menu()));
                    row.setAttribute('data-focus', 'slot|' + sel + '|' + slot.slot);
                }
                row.appendChild(el('span', 'cmp-slotname', slot.label));
                row.appendChild(el('span', 'cmp-name', worn ? label(worn) : 'empty'));
                row.appendChild(el('span', 'cmp-meta', worn ? CompanyData.kg(worn.grams) : ''));
                if (worn) { row.title = itemTip(worn); }
                if (ready && worn && worn.slot !== 'pack') { draggable(row, { from: 'worn', member: sel, slot: slot.slot, ref: worn.ref }); }
                if (ready) {
                    dropTarget(row, d => d.from === 'cargo' && fitsSlot(d, slot.slot) && fitsMember(d, m), d => send(equipCommand(sel, d, slot.slot)));
                }
                li.appendChild(row);
                ul.appendChild(li);
            });
            box.appendChild(ul);
            if (ready) { dropTarget(box, d => d.from === 'cargo' && (d.type || '') !== '' && fitsMember(d, m), d => send('company equip ' + sel + ' ' + d.ref)); }
            section.appendChild(box);
        });
        return section;
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
        keepScroll(panel);
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
        if (inv.seized) {
            // Phase 53: a capture holds the leader's pack and gold in a chest.
            const held = inv.seized;
            const parts = [];
            if (held.items > 0) { parts.push(held.items + (held.items === 1 ? ' item' : ' items')); }
            if (held.gold > 0) { parts.push(held.gold + ' gold'); }
            const what = parts.join(' and ');
            const line = el('div', 'cmp-line cmp-held', 'Held by your captors: ' + what + (held.where ? ' in ' + held.where : '') + '. ' +
                (held.here ? 'Defeat the guards, then reclaim it.' : 'Go back there, defeat the guards, and reclaim it.'));
            line.setAttribute('role', 'status');
            pad.appendChild(line);
            if (held.here) {
                const seizedActions = el('div', 'cmp-actions');
                seizedActions.appendChild(button('Reclaim', 'reclaim', 'Open the chest and take back your pack (reclaim)'));
                pad.appendChild(seizedActions);
            }
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
        if (inv.shared) { pad.appendChild(equipmentSection(inv)); }
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
            ? itemList(inv.cargo, i => inv.shared ? sharedCargoMenu(i, inv) : [{ label: 'Take one', cmd: 'cargo take ' + i.ref }],
                inv.shared ? (i => ({ from: 'cargo', ref: i.ref, type: (i.type || '').toLowerCase(), worn_by: i.worn_by || [] })) : null)
            : el('div', 'cmp-note', 'empty'));
        if (inv.shared) {
            // Dropping worn gear on the cargo takes it off, back to cargo.
            dropTarget(cargo, d => d.from === 'worn', d => send('company remove ' + d.member + ' ' + d.slot));
        }
        pad.appendChild(cargo);
    }

    // --- Camp (Phase 32g) ---

    function updateCamp() {
        const panel = document.getElementById('company-camp');
        if (!panel) { return; }
        keepFocus(panel, () => buildCamp(panel));
    }

    // dutiesBlock is the rest duty picker (Phase 51): one row per member
    // at the camp, a button for each duty they can take. The running rest's
    // duties are fixed, so the buttons are disabled then.
    function dutiesBlock(camp) {
        const block = el('section', 'cmp-block');
        block.setAttribute('aria-label', 'Rest duties');
        block.appendChild(el('h4', null, camp.duties_locked ? 'Rest duties (fixed for this rest)' : 'Rest duties'));
        camp.duties.forEach(row => {
            const line = el('div', 'cmp-duty');
            line.setAttribute('role', 'group');
            line.setAttribute('aria-label', row.name + ' duty');
            line.appendChild(el('span', 'cmp-duty-name', row.key === 'leader' ? row.name + ' (you)' : row.name));
            (row.options || []).forEach(duty => {
                const b = el('button', 'cmp-btn', duty.charAt(0).toUpperCase() + duty.slice(1));
                b.type = 'button';
                b.setAttribute('data-focus', 'duty|' + row.key + '|' + duty);
                b.setAttribute('aria-pressed', row.duty === duty ? 'true' : 'false');
                b.title = row.name + ': ' + duty + ' (camp duties ' + row.command + ' ' + duty + ')';
                if (camp.duties_locked) { b.disabled = true; }
                b.addEventListener('click', () => send('camp duties ' + row.command + ' ' + duty));
                line.appendChild(b);
            });
            block.appendChild(line);
        });
        block.appendChild(el('div', 'cmp-note', camp.duties_locked
            ? 'Duties are settled when the rest ends.'
            : 'Anyone on a duty misses the Rested buff; a watcher also ends no better than Ready (help camp duties).'));
        return block;
    }

    // tentsBlock is the tent picker (Phase 52): one button per tent carried,
    // shown when there is a choice. The pitched tent is fixed for a running
    // rest, so the buttons are disabled then.
    function tentsBlock(camp) {
        const block = el('section', 'cmp-block');
        block.setAttribute('aria-label', 'Tent');
        block.appendChild(el('h4', null, camp.resting ? 'Tent (fixed for this rest)' : 'Tent'));
        const line = el('div', 'cmp-duty');
        line.setAttribute('role', 'group');
        line.setAttribute('aria-label', 'Tent choice');
        camp.tents.forEach(t => {
            const b = el('button', 'cmp-btn', t.name.charAt(0).toUpperCase() + t.name.slice(1));
            b.type = 'button';
            b.setAttribute('data-focus', 'tent|' + t.kind);
            b.setAttribute('aria-pressed', t.pitched ? 'true' : 'false');
            b.title = t.name + ': ' + t.effect + ' (' + t.command + ')';
            if (camp.resting) { b.disabled = true; }
            b.addEventListener('click', () => send(t.command));
            line.appendChild(b);
        });
        block.appendChild(line);
        block.appendChild(el('div', 'cmp-note', 'Each tent trades something for something (help camp gear).'));
        return block;
    }

    // musicBlock is the Camp tab's Music row (camp music): each member's
    // family and level, the families covered, the effect of the next song,
    // and the on/off switch.
    function musicBlock(camp) {
        const music = camp.music;
        const block = el('section', 'cmp-block cmp-music');
        block.setAttribute('aria-label', 'Camp music');
        block.appendChild(el('h4', null, 'Music'));
        (music.players || []).forEach(p => {
            const text = p.label
                ? p.label + (p.family === 'voice' ? ', sings' : (p.instrument ? ', on the ' + p.instrument : ', no instrument carried'))
                : 'no music yet';
            block.appendChild(el('div', 'cmp-line', (p.key === 'leader' ? p.name + ' (you)' : p.name) + ': ' + text));
        });
        if (music.off) {
            block.appendChild(el('div', 'cmp-line', 'The camp song is off.'));
        } else if (music.effects && music.effects.length) {
            block.appendChild(el('div', 'cmp-line', 'Families covered: ' + music.covered + '. The next rest\'s song:'));
            const list = el('ul');
            music.effects.forEach(e => list.appendChild(el('li', 'cmp-line', e)));
            block.appendChild(list);
            if (music.cost) {
                block.appendChild(el('div', 'cmp-note', music.cost));
            }
        } else {
            block.appendChild(el('div', 'cmp-note', 'Nobody can play yet: learn a family from a music teacher and carry its instrument (help music).'));
        }
        if (music.teacher) {
            block.appendChild(el('div', 'cmp-line', 'A music teacher is here: music learn [family] [member] costs ' + music.teach_price + ' gold (strings, winds, drums or voice).'));
        }
        const row = el('div', 'cmp-actions');
        row.appendChild(button(music.off ? 'Song on' : 'Song off', music.off ? 'camp music on' : 'camp music off',
            music.off ? 'Play at camp again (camp music on)' : 'Rest in silence (camp music off)'));
        block.appendChild(row);
        return block;
    }

    // gigBlock is the inn's gig notice: the window, the company's
    // eligibility and the button to play (inn gig).
    function gigBlock(camp) {
        const gig = camp.gig;
        const block = el('section', 'cmp-block cmp-gig');
        block.setAttribute('aria-label', 'Inn gig');
        block.appendChild(el('h4', null, 'Gig notice'));
        block.appendChild(el('div', 'cmp-line', 'Musicians wanted here, ' + gig.window + ' each evening.'));
        if (gig.ready) {
            block.appendChild(el('div', 'cmp-line', 'Your company can play now: ' + gig.families + ' families, about ' + gig.pay + ' gold.'));
            const row = el('div', 'cmp-actions');
            row.appendChild(button('Play a gig', 'inn gig', 'Play for the room until the song ends (inn gig)'));
            block.appendChild(row);
        } else {
            block.appendChild(el('div', 'cmp-line', gig.reason || 'Your company cannot play now.'));
            if (gig.families >= 2) {
                block.appendChild(el('div', 'cmp-note', 'It would earn about ' + gig.pay + ' gold (help gigs).'));
            }
        }
        return block;
    }

    // recipesOpen remembers whether the Camp tab's recipe book is unfolded.
    let recipesOpen = false;

    function buildCamp(panel) {
        const camp = (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Camp) || {};
        const data = CompanyData.read();
        keepScroll(panel);
        const oldBook = panel.querySelector('details.cmp-recipes');
        if (oldBook) { recipesOpen = oldBook.open; }
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
            const tentName = camp.tent_name || 'oiled canvas tent';
            const article = /^[aeiou]/i.test(tentName) ? 'An ' : 'A ';
            pad.appendChild(el('div', 'cmp-line', article + tentName + ' is pitched here: ' + (camp.tent_note || 'shelter, and no cold while you rest') + '.'));
        }
        if (camp.has_camp && Array.isArray(camp.gear) && camp.gear.length) {
            pad.appendChild(el('div', 'cmp-line', 'Camp gear: ' + camp.gear.join(', ') + '.'));
        } else if (camp.has_camp) {
            pad.appendChild(el('div', 'cmp-line', 'No camp gear carried (help camp gear).'));
        }
        if (camp.has_camp && Array.isArray(camp.supplies) && camp.supplies.length) {
            pad.appendChild(el('div', 'cmp-line', 'Supplies: ' + camp.supplies.join(', ') + ' (camp prepare).'));
        }
        if (camp.has_camp && camp.here) {
            const ailing = [];
            data.members.forEach(m => {
                if (!m || m.status === 'dead') { return; }
                const list = data.vitals(m.key).ailments;
                if (Array.isArray(list) && list.length) {
                    ailing.push((m.key === 'leader' ? 'You' : m.name) + ': ' + list.join(', '));
                }
            });
            if (ailing.length) {
                pad.appendChild(el('div', 'cmp-line', 'Ailing: ' + ailing.join('; ') + '. Make the remedy from gathered herbs with camp prepare remedy (help ailments).'));
                if (!camp.resting) {
                    // 55 review: the cure is one press, like the camp's other actions.
                    const b = el('button', 'cmp-btn', 'Make remedies');
                    b.type = 'button';
                    b.title = 'camp prepare remedy all: uses gathered herbs (help ailments)';
                    b.addEventListener('click', () => send('camp prepare remedy all'));
                    const row = el('div');
                    row.appendChild(b);
                    pad.appendChild(row);
                }
            }
        }
        if (camp.has_camp && Array.isArray(camp.prepared) && camp.prepared.length) {
            pad.appendChild(el('div', 'cmp-line', 'Set by for the next rest: ' + camp.prepared.join(', ') + '.'));
        }
        if (camp.has_camp && camp.theft_risk) {
            pad.appendChild(el('div', 'cmp-line', 'Thieves work this road: without bells and trip lines, a rest here may be robbed.'));
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

        // Phase 49: the company's latest talk, at camp or after a battle.
        if (Array.isArray(camp.banter) && camp.banter.length) {
            const talk = el('div', 'cmp-banter');
            talk.setAttribute('role', 'log');
            talk.setAttribute('aria-label', 'Company talk');
            talk.appendChild(el('div', 'cmp-note', 'Around the fire'));
            camp.banter.forEach(line => {
                const row = el('div', 'cmp-line');
                row.appendChild(el('b', null, line.name));
                row.appendChild(document.createTextNode(' ' + (line.verb || 'says') + ', \u201c' + line.text + '\u201d'));
                talk.appendChild(row);
            });
            pad.appendChild(talk);
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

        // Phase 75: the inn's rooms, cheapest first, each with its price for
        // the whole company and the Well Rested it leaves.
        if (camp.inn && Array.isArray(camp.inn_rooms) && camp.inn_rooms.length && !camp.resting) {
            const rooms = el('div', 'cmp-actions');
            rooms.setAttribute('role', 'group');
            rooms.setAttribute('aria-label', 'Inn rooms');
            camp.inn_rooms.forEach(r => {
                const length = r.minutes >= 60 && r.minutes % 60 === 0 ? (r.minutes / 60) + ' h' : r.minutes + ' min';
                rooms.appendChild(button(r.name + ' room, ' + r.price + ' gold', r.command,
                    'Pay ' + r.price + ' gold for the company: Well Rested for ' + length + ' (' + r.command + ')'));
            });
            pad.appendChild(rooms);
        }

        // Phase 51: who does what during the next rest, under the camp's
        // own buttons (51 review: above them it pushed Rest off a phone).
        if (camp.has_camp && camp.here && Array.isArray(camp.duties) && camp.duties.length) {
            pad.appendChild(dutiesBlock(camp));
        }

        if (camp.has_camp && camp.here && Array.isArray(camp.tents) && camp.tents.length > 1) {
            pad.appendChild(tentsBlock(camp));
        }

        // Camp music: who plays and what the song gives, and an inn's gig
        // notice.
        if (camp.music && (camp.has_camp && camp.here || camp.inn || camp.music.teacher)) {
            pad.appendChild(musicBlock(camp));
        }
        if (camp.gig) {
            pad.appendChild(gigBlock(camp));
        }

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

        // Phase 56: the recipe book, folded below everything the camp's
        // buttons do (56 review: open above them it pushed Rest off a
        // phone). Whether it is open survives the tab's rebuilds.
        if (Array.isArray(camp.recipes) && camp.recipes.length) {
            const book = el('details', 'cmp-block cmp-recipes');
            book.open = recipesOpen;
            book.addEventListener('toggle', () => { recipesOpen = book.open; });
            book.appendChild(el('summary', null, 'Recipe book (' + camp.recipes.length + ')'));
            const list = el('ul');
            camp.recipes.forEach(r => list.appendChild(el('li', 'cmp-line', r)));
            book.appendChild(list);
            book.appendChild(el('div', 'cmp-note', 'Type cook with a new mix of ingredients to find another dish (help recipes).'));
            pad.appendChild(book);
        }
    }

    // --- Chronicle (Phase 63) ---
    // chronicle is the newest Company.Chronicle payload, kept here as well
    // as in GMCPStructs because a later full Company snapshot replaces the
    // namespace's children there. chronicleKind is the filter: a kind, or
    // '' for every deed.
    let chronicle = null;
    let chronicleKind = '';
    // townsfolk is the newest Company.Townsfolk payload (Phase 68), kept as
    // the chronicle's is.
    let townsfolk = null;

    // buildTownsfolk draws what the towns say of the company: what talkers
    // have told, the deeds they may yet speak of, and the marks left.
    function buildTownsfolk(pad) {
        const data = townsfolk || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Townsfolk) || null;
        const told = data && Array.isArray(data.told) ? data.told : [];
        const fresh = data && Array.isArray(data.fresh) ? data.fresh : [];
        const marks = data && Array.isArray(data.marks) ? data.marks : [];
        const box = el('div', 'cmp-town');
        box.appendChild(el('div', 'cmp-line', 'What the towns say of you'));
        if (!told.length && !fresh.length) {
            box.appendChild(el('div', 'cmp-note', 'Nobody has spoken of the company yet. Town talkers mention its deeds once each, when you pass them idle in a settlement (help townsfolk).'));
            pad.appendChild(box);
            return;
        }
        const list = el('ul', 'cmp-town-list');
        told.forEach(t => {
            const li = el('li', 'cmp-town-item');
            li.appendChild(el('span', 'cmp-town-meta', (t.ago || '') + ' \u00b7 said'));
            li.appendChild(el('span', 'cmp-town-text', '\u201c' + (t.text || '') + '\u201d'));
            list.appendChild(li);
        });
        fresh.forEach(f => {
            const li = el('li', 'cmp-town-item');
            li.appendChild(el('span', 'cmp-town-meta', (f.ago || '') + ' \u00b7 not yet spoken of'));
            li.appendChild(el('span', 'cmp-town-text', f.text || ''));
            list.appendChild(li);
        });
        box.appendChild(list);
        if (marks.length) {
            box.appendChild(el('div', 'cmp-note', 'Marks the towns carry of you: ' + marks.join(', ') + '.'));
        }
        pad.appendChild(box);
    }

    // chronicleKinds are the kinds worth a filter button: those with deeds
    // in the list, in the order they first appear.
    function chronicleKinds(entries) {
        const seen = [];
        entries.forEach(e => {
            if (e.kind && !seen.some(k => k.kind === e.kind)) { seen.push({ kind: e.kind, label: e.label || e.kind }); }
        });
        return seen;
    }

    function buildChronicle(panel) {
        const data = chronicle || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Chronicle) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        const entries = data && Array.isArray(data.entries) ? data.entries : [];
        if (!entries.length) {
            pad.appendChild(el('div', 'cmp-note', 'Nothing is written yet. The company\'s deeds are recorded here as they happen: recruits, the fallen, bosses slain, relics found (help chronicle).'));
            return;
        }
        const kinds = chronicleKinds(entries);
        if (!kinds.some(k => k.kind === chronicleKind)) { chronicleKind = ''; }
        const total = data.total || entries.length;
        buildTownsfolk(pad);
        pad.appendChild(el('div', 'cmp-line', total + ' deed' + (total === 1 ? '' : 's') + ' recorded, newest first.'));
        if (kinds.length > 1) {
            const bar = el('div', 'cmp-chron-filter');
            bar.setAttribute('role', 'group');
            bar.setAttribute('aria-label', 'Filter the chronicle');
            [{ kind: '', label: 'All' }].concat(kinds).forEach(k => {
                const b = el('button', 'cmp-btn', k.label);
                b.type = 'button';
                b.setAttribute('data-focus', 'chron|' + k.kind);
                b.setAttribute('aria-pressed', chronicleKind === k.kind ? 'true' : 'false');
                b.addEventListener('click', () => { chronicleKind = k.kind; updateChronicle(); });
                bar.appendChild(b);
            });
            pad.appendChild(bar);
        }
        const list = el('ul', 'cmp-chron-list');
        entries.filter(e => !chronicleKind || e.kind === chronicleKind).forEach(e => {
            const li = el('li', 'cmp-chron-item');
            li.appendChild(el('span', 'cmp-chron-meta', (e.ago || '') + (e.label ? ' \u00b7 ' + e.label : '')));
            li.appendChild(el('span', 'cmp-chron-text', e.text || ''));
            list.appendChild(li);
        });
        pad.appendChild(list);
        if (total > entries.length) {
            pad.appendChild(el('div', 'cmp-note', 'The newest ' + entries.length + ' are shown; type chronicle all to read every deed kept.'));
        }
    }

    function updateChronicle() {
        const panel = document.getElementById('company-chronicle');
        if (!panel) { return; }
        keepFocus(panel, () => buildChronicle(panel));
    }

    // --- Opinions (Phase 64) ---
    // opinions is the newest Company.Opinions payload, kept as the
    // Chronicle's is, since a full Company snapshot replaces the
    // namespace's children in GMCPStructs.
    let opinions = null;

    function buildOpinions(panel) {
        const data = opinions || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Opinions) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        const members = data && Array.isArray(data.members) ? data.members : [];
        if (!members.length) {
            pad.appendChild(el('div', 'cmp-note', 'No companions hold opinions yet. Recruit some, and they will have views on mercy, camps, relics and the choices you make (help opinions).'));
            return;
        }
        pad.appendChild(el('div', 'cmp-line', 'What your companions think of your choices.'));
        members.forEach(m => {
            const card = el('div', 'cmp-opn-card');
            const head = el('div', 'cmp-opn-head', m.name || 'Companion');
            head.appendChild(el('span', 'cmp-opn-mood', ' \u00b7 ' + (m.personality || '') + ' \u00b7 ' + (m.mood || '') + ' (loyalty ' + (m.loyalty == null ? '?' : m.loyalty) + ')'));
            card.appendChild(head);
            card.appendChild(el('div', 'cmp-opn-yes', 'Likes: ' + (Array.isArray(m.likes) && m.likes.length ? m.likes.join(', ') : 'nothing in particular') + '.'));
            card.appendChild(el('div', 'cmp-opn-no', 'Dislikes: ' + (Array.isArray(m.dislikes) && m.dislikes.length ? m.dislikes.join(', ') : 'nothing in particular') + '.'));
            (Array.isArray(m.notes) ? m.notes : []).forEach(n => {
                const text = (n.verdict < 0 ? 'Disliked ' : 'Approved of ') + (n.label || '') + ', ' + (n.ago || '') + (n.subject ? ' (' + n.subject + ')' : '') + '.';
                card.appendChild(el('div', n.verdict < 0 ? 'cmp-opn-no' : 'cmp-opn-yes', text));
            });
            (Array.isArray(m.deeds) ? m.deeds : []).forEach(d => card.appendChild(el('div', 'cmp-note', 'Remembers: ' + d)));
            pad.appendChild(card);
        });
        pad.appendChild(el('div', 'cmp-note', 'The company has seen you spare ' + (data.spared || 0) + ' and execute ' + (data.executed || 0) + '. A companion speaks once per choice, and again on the same kind only after a while; approval lifts loyalty no higher than 80, disapproval never below 30.'));
    }

    function updateOpinions() {
        const panel = document.getElementById('company-opinions');
        if (!panel) { return; }
        keepFocus(panel, () => buildOpinions(panel));
    }

    // --- Bonds (Phase 65) ---
    // bondData is the newest Company.Bonds payload, kept as the Opinions'
    // is.
    let bondData = null;

    function buildBonds(panel) {
        const data = bondData || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Bonds) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        const pairs = data && Array.isArray(data.pairs) ? data.pairs : [];
        if (!pairs.length) {
            pad.appendChild(el('div', 'cmp-note', 'Bonds form between companions. Recruit at least two, camp together and fight side by side, and they will come to trust each other, or not (help bonds).'));
            return;
        }
        pad.appendChild(el('div', 'cmp-line', 'How your companions feel about each other.'));
        pairs.forEach(p => {
            const value = Number(p.value) || 0;
            const card = el('div', 'cmp-bond-card');
            card.setAttribute('data-feel', value <= -50 ? 'rival' : (value >= 25 ? 'friend' : 'plain'));
            const head = el('div', 'cmp-bond-head', p.phrase || ((p.a_name || '?') + ' and ' + (p.b_name || '?')));
            head.appendChild(el('span', 'cmp-bond-num', ' (bond ' + (value > 0 ? '+' : '') + value + ')'));
            card.appendChild(head);
            const track = el('div', 'cmp-bond-track');
            track.appendChild(el('div', 'cmp-bond-mid'));
            const fill = el('div', 'cmp-bond-fill');
            const half = Math.min(Math.abs(value), 100) / 2;
            fill.style.left = (value >= 0 ? 50 : 50 - half) + '%';
            fill.style.width = half + '%';
            track.appendChild(fill);
            card.appendChild(track);
            if (p.effect) { card.appendChild(el('div', 'cmp-note', p.effect)); }
            pad.appendChild(card);
        });
        const members = data && Array.isArray(data.members) ? data.members : [];
        members.forEach(m => {
            const feelings = Array.isArray(m.feelings) ? m.feelings : [];
            if (!feelings.length) { return; }
            pad.appendChild(el('div', 'cmp-line', (m.name || 'Companion') + ' ' + feelings.map(f => f.words).join('; ') + '.'));
        });
        pad.appendChild(el('div', 'cmp-note', 'Time together raises a bond to 50 at most and lowers it to wary at most; stepping in for each other takes it higher, and only splitting over your choices or a refused guard makes rivals. Friends step in once a battle for a friend at 40% health or less (kin twice); rivals will not guard each other.'));
    }

    function updateBonds() {
        const panel = document.getElementById('company-bonds');
        if (!panel) { return; }
        keepFocus(panel, () => buildBonds(panel));
    }

    // --- Errands (Phase 70) ---
    // errandData is the newest Company.Errands payload; errandPick is what
    // each free companion's job and length selectors show, kept across the
    // rebuilds a new payload causes. errandClock offsets the server's real
    // time against this browser's, so a countdown is right whatever the
    // browser's clock says; the panel redraws each half minute while a
    // companion is away.
    let errandData = null;
    let errandClock = 0;
    let errandTimer = null;
    const errandPick = {};

    function errandLeft(seconds) {
        if (seconds <= 0) { return 'due now'; }
        if (seconds < 90) { return 'a minute'; }
        if (seconds < 90 * 60) { return Math.round(seconds / 60) + ' minutes'; }
        return Math.round(seconds / 3600) + ' hours';
    }

    function errandSelect(label, options, current, onPick) {
        const sel = el('select', 'cmp-select');
        sel.setAttribute('aria-label', label);
        options.forEach(o => {
            const opt = el('option', null, o.label);
            opt.value = o.value;
            if (o.value === current) { opt.selected = true; }
            sel.appendChild(opt);
        });
        sel.addEventListener('change', () => onPick(sel.value));
        return sel;
    }

    function buildErrands(panel) {
        const data = errandData || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Errands) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        const rows = data && Array.isArray(data.rows) ? data.rows : [];
        if (!rows.length) {
            pad.appendChild(el('div', 'cmp-note', 'Companions you can spare can be sent on errands from an inn. Recruit one first (help errands).'));
            return;
        }
        const options = Array.isArray(data.options) ? data.options : [];
        const here = !!data.here;
        pad.appendChild(el('div', 'cmp-line', data.where || ''));
        if (here && data.zone) {
            pad.appendChild(el('div', 'cmp-note', data.zone + (data.band ? ', a zone for levels ' + data.band + ': the pay follows the band, and a companion under it risks a wound.' : ': it names no level band, so the pay follows the companion.')));
        }
        // The server's clock, as of when this payload arrived.
        const nowSec = () => Math.floor(Date.now() / 1000) + errandClock;
        let anyAway = false;
        rows.forEach(r => {
            const card = el('div', 'cmp-err-card');
            card.setAttribute('data-state', r.state || 'busy');
            const head = el('div', 'cmp-err-head', r.name || 'Companion');
            head.appendChild(el('span', 'cmp-err-sub', ' (level ' + (Number(r.level) || '?') + ')'));
            card.appendChild(head);
            if (r.state === 'away') {
                anyAway = true;
                const left = (Number(r.returns_at) || 0) - nowSec();
                const line = (r.due || left <= 0)
                    ? 'Away on ' + (r.kind_label || 'an errand') + ': due back, ' + (r.waiting || 'as soon as you are free') + '.'
                    : 'Away on ' + (r.kind_label || 'an errand') + (r.zone ? ' in ' + r.zone : '') + '; back in ' + errandLeft(left) + '.';
                card.appendChild(el('div', 'cmp-line', line));
                card.appendChild(button('Call back', 'errand recall #' + r.id, 'Bring them home now, with nothing to show for it'));
            } else if (r.state === 'ready') {
                if (here && options.length) {
                    const pick = errandPick[r.id] || (errandPick[r.id] = { kind: options[0].kind, length: options[0].lengths[0].length });
                    const row = el('div', 'cmp-err-pick');
                    row.appendChild(errandSelect('Errand for ' + r.name, options.map(o => ({ value: o.kind, label: o.label })), pick.kind, v => { pick.kind = v; updateErrands(); }));
                    const chosen = options.find(o => o.kind === pick.kind) || options[0];
                    row.appendChild(errandSelect('Length for ' + r.name, chosen.lengths.map(l => ({ value: l.length, label: l.label })), pick.length, v => { pick.length = v; }));
                    // The command is built when clicked, so a length picked
                    // after the card was drawn is the one sent.
                    const go = el('button', 'cmp-btn', 'Send');
                    go.type = 'button';
                    go.setAttribute('data-focus', 'btn|Send|' + r.id);
                    go.title = 'Send ' + r.name + ' away from here';
                    go.addEventListener('click', () => send('errand send #' + r.id + ' ' + pick.kind + ' ' + pick.length));
                    row.appendChild(go);
                    card.appendChild(row);
                    if (chosen.blurb) { card.appendChild(el('div', 'cmp-note', 'Goes ' + chosen.blurb + '.')); }
                } else {
                    card.appendChild(el('div', 'cmp-note', 'Free to send, from an inn.'));
                }
            } else {
                card.appendChild(el('div', 'cmp-note', r.why || 'Not free right now.'));
            }
            pad.appendChild(card);
        });
        const recent = Array.isArray(data.recent) ? data.recent : [];
        if (recent.length) {
            pad.appendChild(el('div', 'cmp-line', 'Lately:'));
            recent.forEach(t => pad.appendChild(el('div', 'cmp-note', t)));
        }
        pad.appendChild(el('div', 'cmp-note', 'An errand runs in real time, so it carries on while you are away and ends when its time is up: the companion comes home with modest gold, a small find, word of a lair or a wound. A companion who goes leaves the formation and takes its place again on return (help errands).'));
        if (anyAway && !errandTimer) {
            errandTimer = setInterval(updateErrands, 30000);
        } else if (!anyAway && errandTimer) {
            clearInterval(errandTimer);
            errandTimer = null;
        }
    }

    function updateErrands() {
        const panel = document.getElementById('company-errands');
        if (!panel) { return; }
        keepFocus(panel, () => buildErrands(panel));
    }

    // --- Rites (Phase 74) ---
    // riteData is the newest Company.Rites payload.
    let riteData = null;

    function riteButton(label, cmd, title, id) {
        const b = el('button', 'cmp-btn', label);
        b.type = 'button';
        b.setAttribute('data-focus', 'btn|' + label + '|' + id);
        b.title = title;
        b.addEventListener('click', () => send(cmd));
        return b;
    }

    function buildRites(panel) {
        const data = riteData || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Rites) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        const rows = data && Array.isArray(data.rows) ? data.rows : [];
        if (!rows.length) {
            pad.appendChild(el('div', 'cmp-note', 'No one is waiting to be mourned. When a companion is lost for good, or leaves after long service, you can hold rites for them at your next camp or inn (help rites).'));
            return;
        }
        const here = !!data.here;
        pad.appendChild(el('div', 'cmp-line', data.where || ''));
        rows.forEach(r => {
            const card = el('div', 'cmp-rite-card');
            card.setAttribute('data-offered', r.offered ? '1' : '0');
            const head = el('div', 'cmp-err-head', r.name || 'Companion');
            head.appendChild(el('span', 'cmp-err-sub', ' (level ' + (Number(r.level) || '?') + ', ' + (r.cause || 'gone') + ')'));
            card.appendChild(head);
            const close = Array.isArray(r.close) ? r.close : [];
            if (close.length) {
                card.appendChild(el('div', 'cmp-note', close.join(', ') + ' trusted' + ' them.'));
            }
            if (r.offered) {
                card.appendChild(el('div', 'cmp-note', 'If you say nothing, this passes at your next camp or inn, and the company will feel it.'));
            }
            if (here) {
                const btns = el('div', 'cmp-rite-btns');
                btns.appendChild(riteButton('Hold rites', 'rite hold #' + r.id, 'Gather the company to mourn ' + r.name, r.id));
                btns.appendChild(riteButton('Let pass', 'rite skip #' + r.id, 'Say nothing; the company loses a little loyalty', r.id));
                card.appendChild(btns);
            }
            pad.appendChild(card);
        });
        pad.appendChild(el('div', 'cmp-note', 'Holding rites gives the company a line in its chronicle, steadies the companions who trusted the one gone (loyalty up to 3) and draws them together. Letting it pass costs loyalty (up to 5, never below 30). Rites cost no gold and give none (help rites).'));
    }

    function updateRites() {
        const panel = document.getElementById('company-rites');
        if (!panel) { return; }
        keepFocus(panel, () => buildRites(panel));
    }

    // --- Bounties (Phase 76) ---
    // bountyData is the newest Company.Bounties payload; bountyAt is when it
    // arrived, so "time left" counts down between pushes.
    let bountyData = null;
    let bountyAt = 0;

    function bountyButton(label, cmd, title, id, disabled) {
        const b = el('button', 'cmp-btn', label);
        b.type = 'button';
        b.setAttribute('data-focus', 'btn|' + label + '|' + id);
        b.title = title;
        if (disabled) { b.disabled = true; }
        b.addEventListener('click', () => send(cmd));
        return b;
    }

    function bountyWhat(t) {
        const count = Number(t.count) || 1;
        return (t.kind === 'boss' ? 'Slay the master of the lair in ' : 'Break ' + count + ' groups in ') + (t.zone || 'the wilds');
    }

    function buildBounties(panel) {
        const data = bountyData || (Client.GMCPStructs.Company && Client.GMCPStructs.Company.Bounties) || null;
        keepScroll(panel);
        panel.textContent = '';
        const pad = el('div', 'cmp-pad');
        panel.appendChild(pad);
        if (!data) {
            pad.appendChild(el('div', 'cmp-note', 'Bounty boards in towns post lair masters and bands of foes wanted nearby (help bounties).'));
            return;
        }
        const waited = bountyAt ? Math.max(0, Math.floor((Date.now() - bountyAt) / 1000)) : 0;
        const held = Array.isArray(data.held) ? data.held : [];
        const postings = Array.isArray(data.postings) ? data.postings : [];
        const max = Number(data.max) || 3;

        pad.appendChild(el('div', 'cmp-line', 'Held bounties (' + held.length + ' of ' + max + ')'));
        if (!held.length) {
            pad.appendChild(el('div', 'cmp-note', 'Your company holds none. Take one at a bounty board in a town.'));
        }
        held.forEach(h => {
            const card = el('div', 'cmp-rite-card');
            card.setAttribute('data-offered', h.ready ? '1' : '0');
            const head = el('div', 'cmp-err-head', h.name || 'A mark');
            head.appendChild(el('span', 'cmp-err-sub', ' ' + (Number(h.reward) || 0) + ' gold'));
            card.appendChild(head);
            card.appendChild(el('div', 'cmp-note', bountyWhat(h)));
            const left = Math.max(0, (Number(h.left) || 0) - waited);
            card.appendChild(el('div', 'cmp-note', h.ready ? 'Proof found: ready to claim.' : (Number(h.have) || 0) + ' of ' + (Number(h.count) || 1) + ' done; ' + errandLeft(left) + ' left.'));
            const btns = el('div', 'cmp-rite-btns');
            if (data.at_board) {
                btns.appendChild(bountyButton('Claim', 'bounty claim ' + h.n, 'Collect the gold for ' + h.name, 'held' + h.n, !h.ready));
            }
            btns.appendChild(bountyButton('Drop', 'bounty drop ' + h.n, 'Give up this bounty', 'held' + h.n, false));
            card.appendChild(btns);
            pad.appendChild(card);
        });

        if (!data.at_board) {
            pad.appendChild(el('div', 'cmp-note', 'No bounty board here. Boards hang in towns; take and claim bounties at one.'));
            return;
        }
        const rotates = Math.max(0, (Number(data.rotates) || 0) - waited);
        pad.appendChild(el('div', 'cmp-line', 'The board at ' + (data.board || 'this town') + (data.band ? ' (level ' + data.band + ')' : '') + '; a new list in ' + errandLeft(rotates)));
        if (!postings.length) {
            pad.appendChild(el('div', 'cmp-note', 'Nothing is posted. The board lists lairs and bands from the zones near here, and none are known.'));
        }
        const full = held.length >= max;
        postings.forEach(p => {
            const card = el('div', 'cmp-rite-card');
            card.setAttribute('data-offered', '0');
            const head = el('div', 'cmp-err-head', p.name || 'A mark');
            head.appendChild(el('span', 'cmp-err-sub', ' ' + (Number(p.reward) || 0) + ' gold'));
            card.appendChild(head);
            card.appendChild(el('div', 'cmp-note', bountyWhat(p) + ' (zone level ' + (p.band || '?') + (p.rating ? ', ' + p.rating + ' for you' : '') + ')'));
            const btns = el('div', 'cmp-rite-btns');
            const label = p.taken ? 'Taken' : (p.done ? 'Settled' : 'Take');
            btns.appendChild(bountyButton(label, 'bounty take ' + p.n, 'Take this bounty', 'post' + p.n, p.taken || p.done || full));
            card.appendChild(btns);
            pad.appendChild(card);
        });
        pad.appendChild(el('div', 'cmp-note', 'Gold only, by the zone\'s level band. A bounty never changes a foe. The chronicle is the proof: only kills after you take it count (help bounties).'));
    }

    function updateBounties() {
        const panel = document.getElementById('company-bounties');
        if (!panel) { return; }
        keepFocus(panel, () => buildBounties(panel));
    }

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Company', 'Party'],
        onGMCP(namespace, body) {
            if (namespace === 'Company.Chronicle') {
                chronicle = body && typeof body === 'object' ? body : null;
                win.open();
                if (win.isOpen()) { updateChronicle(); }
                return;
            }
            if (namespace === 'Company.Townsfolk') {
                townsfolk = body && typeof body === 'object' ? body : null;
                win.open();
                if (win.isOpen()) { updateChronicle(); }
                return;
            }
            if (namespace === 'Company.Errands') {
                errandData = body && typeof body === 'object' ? body : null;
                errandClock = errandData && Number(errandData.now) ? Number(errandData.now) - Math.floor(Date.now() / 1000) : 0;
                win.open();
                if (win.isOpen()) { updateErrands(); }
                return;
            }
            if (namespace === 'Company.Bounties') {
                bountyData = body && typeof body === 'object' ? body : null;
                bountyAt = Date.now();
                win.open();
                if (win.isOpen()) { updateBounties(); }
                return;
            }
            if (namespace === 'Company.Rites') {
                riteData = body && typeof body === 'object' ? body : null;
                win.open();
                if (win.isOpen()) { updateRites(); }
                return;
            }
            if (namespace === 'Company.Bonds') {
                bondData = body && typeof body === 'object' ? body : null;
                win.open();
                if (win.isOpen()) { updateBonds(); }
                return;
            }
            if (namespace === 'Company.Opinions') {
                opinions = body && typeof body === 'object' ? body : null;
                win.open();
                if (win.isOpen()) { updateOpinions(); }
                return;
            }
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
