/**
 * window-character.js
 *
 * Virtual window: Character - the company dock's first tab (Phase 32g),
 * with sub-tabs:
 *   Overview - name, race/class, level, alignment, stats grid, point
 *              badges, then Worth (window-status.js: XP, gold)
 *   Gear     - worn and carried items (window-gear.js)
 *   Skills   - learned skills with levels and max indicator, then jobs
 *              (profession completion and proficiency)
 *   Quests   - in-progress quest log, click to expand
 *   Effects  - active buffs/debuffs with duration bars
 *   Pet      - only while the player has a pet (window-pet.js)
 *
 * Other windows' scripts host their content here through
 * window.CharacterTabs (add, addToOverview, setVisible); they keep their
 * own GMCP handling. The chosen sub-tab is remembered per browser.
 *
 * Responds to GMCP namespaces:
 *   Char         - full character update
 *   Char.Info    - name, race, class, level, alignment, skill/training points
 *   Char.Stats   - six core stats
 *   Char.Quests  - quest progress
 *   Char.Skills  - skill names, levels, max flag
 *   Char.Jobs    - profession completion and proficiency
 *   Char.Affects - active buffs/debuffs
 *
 * Reads:
 *   Client.GMCPStructs.Char.Info
 *   Client.GMCPStructs.Char.Stats
 *   Client.GMCPStructs.Char.Quests
 *   Client.GMCPStructs.Char.Skills
 *   Client.GMCPStructs.Char.Jobs
 *   Client.GMCPStructs.Char.Affects
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shared tab chrome ---- */
        #character-window {
            height: 100%;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
        }

        #character-window .cw-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        #character-window .cw-tab-btn {
            flex: 1;
            padding: 5px 4px;
            background: var(--t-bg-surface);
            border: none;
            cursor: pointer;
            font: inherit;
            font-size: 0.7em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            transition: background 0.15s, color 0.15s;
            border-right: 1px solid var(--t-border);
        }

        #character-window .cw-tab-btn:last-child {
            border-right: none;
        }

        #character-window .cw-tab-btn:hover {
            background: var(--t-border);
            color: var(--t-text);
        }

        #character-window .cw-tab-btn.active {
            background: var(--t-bg);
            color: var(--t-text);
            border-bottom: 2px solid var(--t-accent);
        }

        #character-window .cw-tab-panel {
            display: none;
            flex: 1;
            overflow-y: auto;
        }

        #character-window .cw-tab-panel::-webkit-scrollbar       { width: 4px; }
        #character-window .cw-tab-panel::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #character-window .cw-tab-panel::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 2px; }

        #character-window .cw-tab-panel.active {
            display: flex;
            flex-direction: column;
        }

        /* ---- Overview tab ---- */
        #cw-overview {
            padding: 8px 10px;
            gap: 5px;
        }

        #cw-char-name {
            font-size: 0.88em;
            color: var(--t-text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        #cw-char-name .cw-char-race {
            cursor: help;
        }

        #cw-char-name .cw-char-race:hover {
            color: var(--t-accent);
        }

        #cw-char-level {
            font-size: 0.74em;
            color: var(--t-text-muted);
        }

        #cw-char-alignment {
            font-size: 0.7em;
            font-style: italic;
            margin-bottom: 2px;
        }

        .cw-align-good    { color: var(--t-good-align); }
        .cw-align-neutral { color: var(--t-neutral-align);    }
        .cw-align-evil    { color: var(--t-evil-align); }

        /* ---- Stats grid (inside Overview) ---- */
        #cw-stats-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 3px 6px;
            padding: 4px 0 2px;
            border-top: 1px solid var(--t-border);
            border-bottom: 1px solid var(--t-border);
        }

        .cw-stat-cell {
            display: grid;
            grid-template-columns: auto 1fr auto;
            align-items: baseline;
            gap: 3px;
            cursor: help;
        }

        .cw-stat-cell:hover .cw-stat-abbr,
        .cw-stat-cell:hover .cw-stat-num {
            color: var(--t-accent);
        }

        .cw-stat-abbr {
            font-size: 0.64em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .cw-stat-num {
            font-size: 0.78em;
            color: var(--t-text);
            font-weight: bold;
            text-align: right;
        }

        .cw-stat-mod {
            font-size: 0.68em;
            color: var(--t-text-secondary);
            font-weight: normal;
            cursor: help;
            visibility: hidden;
        }

        .cw-stat-mod.visible {
            visibility: visible;
        }
        #cw-points-row {
            display: flex;
            gap: 6px;
            padding: 4px 0 2px;
            border-bottom: 1px solid var(--t-border);
        }

        .cw-point-badge {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 2px 6px;
            background: var(--t-bg-surface);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            cursor: help;
            gap: 4px;
        }

        .cw-point-badge:hover {
            background: var(--t-border);
        }

        .cw-point-badge-label {
            font-size: 0.62em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            white-space: nowrap;
        }

        .cw-point-badge-value {
            font-size: 0.8em;
            color: var(--t-text);
            font-weight: bold;
        }

        .cw-point-badge.has-points {
            border-color: var(--t-accent);
            background: var(--t-bg-hover);
        }

        .cw-point-badge.has-points .cw-point-badge-value {
            color: var(--t-accent);
        }

        #cw-stat-tooltip {
            position: fixed;
            z-index: 99999;
            pointer-events: none;
            background: var(--t-bg-surface);
            border: 1px solid var(--t-border-accent);
            border-radius: 6px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.7);
            padding: 8px 10px;
            min-width: 140px;
            max-width: 240px;
            font-size: 0.75em;
            color: var(--t-text-secondary);
            display: none;
        }

        .cw-tt-name {
            font-size: 0.85em;
            font-weight: bold;
            color: var(--t-text);
            margin-bottom: 4px;
            line-height: 1.3;
        }

        .cw-tt-divider {
            border: none;
            border-top: 1px solid var(--t-border-accent);
            margin: 5px 0;
        }

        .cw-tt-row {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 8px;
            line-height: 1.6;
        }

        .cw-tt-row-label {
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            font-size: 0.88em;
            flex-shrink: 0;
        }

        .cw-tt-row-value {
            color: var(--t-text);
            text-align: right;
        }

        .cw-affect-item { cursor: default; }

        /* ---- Quests tab ---- */
        #cw-quests {
            padding: 4px 6px;
            gap: 5px;
        }

        #cw-quests .cq-empty {
            color: var(--t-text-dim);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        #cw-quests .cq-item {
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 5px 7px;
            display: flex;
            flex-direction: column;
            gap: 4px;
            cursor: pointer;
            transition: background 0.15s;
            flex-shrink: 0;
        }

        #cw-quests .cq-item:hover,
        #cw-quests .cq-item.expanded {
            background: var(--t-bg-surface);
        }

        #cw-quests .cq-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 6px;
        }

        #cw-quests .cq-name {
            font-size: 0.82em;
            color: var(--t-text);
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        #cw-quests .cq-pct {
            font-size: 0.7em;
            color: var(--t-text-secondary);
            flex-shrink: 0;
        }

        #cw-quests .cq-bar-track {
            width: 100%;
            height: 5px;
            background: var(--t-bg-row);
            border-radius: 3px;
            overflow: hidden;
            border: 1px solid var(--t-party-hp-border);
        }

        #cw-quests .cq-bar-fill {
            height: 100%;
            border-radius: 3px;
            background: linear-gradient(to right, var(--t-progress-from), var(--t-progress-to));
            transition: width 0.4s ease-out;
        }

        #cw-quests .cq-item.complete {
            background: var(--t-bg-deep);
            border-color: var(--t-border-accent);
            opacity: 0.6;
        }

        #cw-quests .cq-item.complete:hover,
        #cw-quests .cq-item.complete.expanded {
            opacity: 1;
            background: var(--t-bg-surface-alt);
        }

        #cw-quests .cq-item.complete .cq-name {
            color: var(--t-text-secondary);
            text-decoration: line-through;
        }

        #cw-quests .cq-item.complete .cq-pct {
            color: var(--t-accent);
            font-weight: bold;
        }

        #cw-quests .cq-bar-fill.complete {
            background: var(--t-accent);
        }

        #cw-quests .cq-desc {
            font-size: 0.73em;
            color: var(--t-text-secondary);
            line-height: 1.4;
            display: none;
            padding-top: 2px;
            border-top: 1px solid var(--t-border);
        }

        #cw-quests .cq-item.expanded .cq-desc {
            display: block;
        }

        #character-window .cw-sub {
            display: flex;
            flex-direction: column;
        }

        #character-window .cw-subhead {
            margin: 6px 6px 0;
            font-size: 0.7em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
            border-bottom: 1px solid var(--t-accent-dim);
        }

        #character-window .cw-tab-btn[hidden],
        #character-window .cw-tab-panel[hidden] {
            display: none !important;
        }

        /* ---- Skills tab ---- */
        #cw-skills {
            padding: 4px 6px;
            gap: 3px;
        }

        #cw-skills .csk-empty {
            color: var(--t-text-dim);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .csk-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 20px;
            border-bottom: 1px solid var(--t-border-faint);
            padding: 3px 2px;
            flex-shrink: 0;
        }

        .csk-row:last-child { border-bottom: none; }

        .csk-name {
            flex: 1;
            font-size: 0.78em;
            color: var(--t-text);
            text-transform: capitalize;
        }

        .csk-pips {
            display: flex;
            gap: 3px;
            flex-shrink: 0;
        }

        .csk-pip {
            width: 9px;
            height: 9px;
            border-radius: 2px;
            border: 1px solid var(--t-accent-dim);
            background: var(--t-bg-surface-alt);
        }

        .csk-pip.filled {
            background: var(--t-accent);
            border-color: var(--t-accent);
        }

        .csk-pip.filled.max {
            background: var(--t-warning);
            border-color: var(--t-warning);
        }

        .csk-badge {
            font-size: 0.58em;
            padding: 1px 4px;
            border-radius: 3px;
            flex-shrink: 0;
            background: var(--t-quest-badge-bg);
            color: var(--t-warning);
            border: 1px solid var(--t-quest-badge-border);
        }

        /* ---- Jobs tab ---- */
        #cw-jobs {
            padding: 4px 6px;
            gap: 5px;
        }

        #cw-jobs .cjb-empty {
            color: var(--t-text-dim);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .cjb-item {
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 5px 7px;
            display: flex;
            flex-direction: column;
            gap: 4px;
            flex-shrink: 0;
        }

        .cjb-header {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 6px;
        }

        .cjb-name {
            font-size: 0.82em;
            color: var(--t-text);
            font-weight: bold;
            text-transform: capitalize;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cjb-meta {
            display: flex;
            align-items: center;
            gap: 5px;
            flex-shrink: 0;
        }

        .cjb-proficiency {
            font-size: 0.68em;
            color: var(--t-text-secondary);
            text-transform: capitalize;
        }

        .cjb-pct {
            font-size: 0.7em;
            color: var(--t-text-secondary);
        }

        .cjb-bar-track {
            width: 100%;
            height: 5px;
            background: var(--t-bg-row);
            border-radius: 3px;
            overflow: hidden;
            border: 1px solid var(--t-party-hp-border);
        }

        .cjb-bar-fill {
            height: 100%;
            border-radius: 3px;
            background: linear-gradient(to right, var(--t-progress-from), var(--t-progress-to));
            transition: width 0.4s ease-out;
        }

        .cjb-item.complete .cjb-name {
            color: var(--t-warning);
        }

        .cjb-item.complete .cjb-pct {
            color: var(--t-warning);
            font-weight: bold;
        }

        .cjb-bar-fill.complete {
            background: var(--t-warning);
        }

        /* ---- Effects tab ---- */
        #cw-effects {
            padding: 4px 6px;
            gap: 4px;
            display: grid;
            grid-template-columns: 1fr 1fr;
            align-content: flex-start;
        }

        .cw-affect-empty {
            grid-column: 1 / -1;
            color: var(--t-text-dim);
            font-size: 0.76em;
            font-style: italic;
            text-align: center;
            padding: 14px 0;
        }

        .cw-affect-item {
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 4px 6px;
            display: flex;
            flex-direction: column;
            gap: 3px;
            min-width: 0;
            box-sizing: border-box;
        }

        .cw-affect-item.debuff {
            border-color: var(--t-debuff-border);
            background: var(--t-debuff-bg);
        }

        .cw-affect-header {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 4px;
        }

        .cw-affect-name {
            font-size: 0.5em;
            color: var(--t-text);
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cw-affect-item.debuff .cw-affect-name { color: var(--t-debuff-text); }

        .cw-affect-source {
            font-size: 0.63em;
            color: var(--t-text-secondary);
            white-space: nowrap;
            flex-shrink: 0;
        }

        .cw-affect-item.debuff .cw-affect-source { color: var(--t-debuff-secondary); }

        .cw-affect-mods {
            font-size: 0.66em;
            color: var(--t-text-secondary);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cw-affect-item.debuff .cw-affect-mods { color: var(--t-debuff-secondary); }

        .cw-affect-dur-track {
            width: 100%;
            height: 4px;
            background: var(--t-bg-row);
            border-radius: 2px;
            overflow: hidden;
        }

        .cw-affect-dur-fill {
            height: 100%;
            border-radius: 2px;
            background: var(--t-accent-dim);
            transition: width 1s linear;
        }

        .cw-affect-item.debuff .cw-affect-dur-fill { background: var(--t-debuff-border); }

        .cw-affect-dur-fill.permanent {
            background: var(--t-accent);
            width: 100% !important;
        }

        .cw-affect-item.debuff .cw-affect-dur-fill.permanent { background: var(--t-debuff-perm); }
    `);

    // -----------------------------------------------------------------------
    // Stat tooltip
    // -----------------------------------------------------------------------
    let statTooltip   = null;
    let statHideTimer = null;

    function ensureStatTooltip() {
        if (statTooltip) { return; }
        statTooltip = document.createElement('div');
        statTooltip.id = 'cw-stat-tooltip';
        document.body.appendChild(statTooltip);
    }

    function showStatTooltip(el, html) {
        ensureStatTooltip();
        clearTimeout(statHideTimer);
        statTooltip.innerHTML = html;
        statTooltip.style.display = 'block';
        _positionStatTooltip(el);
    }

    function _positionStatTooltip(el) {
        if (!statTooltip) { return; }
        const rect = el.getBoundingClientRect();
        const ttW  = statTooltip.offsetWidth;
        const ttH  = statTooltip.offsetHeight;
        const vw   = window.innerWidth;
        const vh   = window.innerHeight;
        let left = rect.right + 8;
        if (left + ttW > vw - 8) { left = rect.left - ttW - 8; }
        left = Math.max(8, left);
        let top = rect.top;
        if (top + ttH > vh - 8) { top = vh - ttH - 8; }
        statTooltip.style.left = left + 'px';
        statTooltip.style.top  = Math.max(8, top) + 'px';
    }

    function hideStatTooltip() {
        if (!statTooltip) { return; }
        statHideTimer = setTimeout(() => { statTooltip.style.display = 'none'; }, 80);
    }

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    const SUBTAB_KEY = 'characterSubTab';

    function showPanel(root, panelId, remember) {
        const btn = root.querySelector('.cw-tab-btn[data-panel="' + panelId + '"]');
        if (!btn || btn.hidden) { panelId = 'cw-overview'; }
        root.querySelectorAll('.cw-tab-btn').forEach(b => {
            const on = b.dataset.panel === panelId;
            b.classList.toggle('active', on);
            b.setAttribute('aria-selected', on ? 'true' : 'false');
        });
        root.querySelectorAll('.cw-tab-panel').forEach(p => p.classList.toggle('active', p.id === panelId));
        if (remember) {
            try { localStorage.setItem(SUBTAB_KEY, panelId); } catch (e) { /* unavailable */ }
        }
    }

    function wireTab(root, btn) {
        btn.setAttribute('role', 'tab');
        btn.addEventListener('click', () => showPanel(root, btn.dataset.panel, true));
    }

    function makeTabSwitcher(root) {
        root.querySelector('.cw-tab-bar').setAttribute('role', 'tablist');
        root.querySelector('.cw-tab-bar').setAttribute('aria-label', 'Character');
        root.querySelectorAll('.cw-tab-btn').forEach(btn => wireTab(root, btn));
    }

    // -----------------------------------------------------------------------
    // Hosted sub-tabs (Phase 32g): other windows' content, by order among
    // the built-in tabs (Overview 0, Skills 2, Quests 3, Effects 4).
    // -----------------------------------------------------------------------
    const hosted   = [];   // { id, label, order, build, visible }
    const overview = [];   // build functions appended to Overview
    let root = null;

    function attachHosted(spec) {
        const panelId = 'cw-hosted-' + spec.id;
        const btn = document.createElement('button');
        btn.className = 'cw-tab-btn';
        btn.dataset.panel = panelId;
        btn.dataset.order = String(spec.order);
        btn.textContent = spec.label;
        btn.hidden = spec.visible === false;
        const bar = root.querySelector('.cw-tab-bar');
        const after = [...bar.children].find(b => Number(b.dataset.order) > spec.order);
        bar.insertBefore(btn, after || null);
        wireTab(root, btn);
        const panel = document.createElement('div');
        panel.className = 'cw-tab-panel';
        panel.id = panelId;
        panel.setAttribute('role', 'tabpanel');
        panel.hidden = btn.hidden;
        panel.appendChild(spec.build());
        root.appendChild(panel);
    }

    window.CharacterTabs = {
        // spec: { id, label, order, build() -> element, visible (default true) }
        add(spec) {
            hosted.push(spec);
            if (root) { attachHosted(spec); }
        },
        // build() -> element, appended to the Overview tab.
        addToOverview(build) {
            overview.push(build);
            if (root) { root.querySelector('#cw-overview').appendChild(build()); }
        },
        setVisible(id, visible) {
            const spec = hosted.find(h => h.id === id);
            if (spec) { spec.visible = visible; }
            if (!root) { return; }
            const btn = root.querySelector('.cw-tab-btn[data-panel="cw-hosted-' + id + '"]');
            const panel = root.querySelector('#cw-hosted-' + id);
            if (!btn || !panel) { return; }
            btn.hidden = !visible;
            panel.hidden = !visible;
            if (!visible && btn.classList.contains('active')) { showPanel(root, 'cw-overview', false); }
        },
    };

    // -----------------------------------------------------------------------
    // Data definitions
    // -----------------------------------------------------------------------
    const STAT_DEFS = [
        { key: 'strength',   abbr: 'STR' },
        { key: 'speed',      abbr: 'SPD' },
        { key: 'smarts',     abbr: 'SMT' },
        { key: 'vitality',   abbr: 'VIT' },
        { key: 'mysticism',  abbr: 'MYS' },
        { key: 'perception', abbr: 'PER' },
    ];

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function buildStatsGrid() {
        const cells = STAT_DEFS.map(d =>
            '<div class="cw-stat-cell">' +
                '<span class="cw-stat-abbr">' + d.abbr + '</span>' +
                '<span class="cw-stat-num" id="cw-stat-' + d.key + '">\u2014</span>' +
                '<span class="cw-stat-mod" id="cw-stat-mod-' + d.key + '"></span>' +
            '</div>'
        ).join('');
        const pointsRow =
            '<div id="cw-points-row">' +
                '<div class="cw-point-badge" id="cw-badge-sp">' +
                    '<span class="cw-point-badge-label">Skill Pts</span>' +
                    '<span class="cw-point-badge-value" id="cw-sp">\u2014</span>' +
                '</div>' +
                '<div class="cw-point-badge" id="cw-badge-tp">' +
                    '<span class="cw-point-badge-label">Train Pts</span>' +
                    '<span class="cw-point-badge-value" id="cw-tp">\u2014</span>' +
                '</div>' +
            '</div>';
        return '<div id="cw-stats-grid">' + cells + '</div>' + pointsRow;
    }

    function createDOM() {
        const el = document.createElement('div');
        el.id = 'character-window';
        el.innerHTML =
            '<div class="cw-tab-bar">' +
                '<button class="cw-tab-btn active" data-panel="cw-overview" data-order="0">Overview</button>' +
                '<button class="cw-tab-btn"        data-panel="cw-skills-tab" data-order="2">Skills</button>' +
                '<button class="cw-tab-btn"        data-panel="cw-quests" data-order="3">Quests</button>' +
                '<button class="cw-tab-btn"        data-panel="cw-effects" data-order="4">Effects</button>' +
            '</div>' +

            '<div class="cw-tab-panel active" id="cw-overview">' +
                '<div id="cw-char-name">\u2014</div>' +
                '<div id="cw-char-level">Level \u2014</div>' +
                '<div id="cw-char-alignment"></div>' +
                buildStatsGrid() +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-quests">' +
                '<div class="cq-empty">No active quests</div>' +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-skills-tab">' +
                '<div class="cw-sub" id="cw-skills">' +
                    '<div class="csk-empty">No skills learned</div>' +
                '</div>' +
                '<h4 class="cw-subhead">Jobs</h4>' +
                '<div class="cw-sub" id="cw-jobs">' +
                    '<div class="cjb-empty">No job progress</div>' +
                '</div>' +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-effects">' +
                '<div class="cw-affect-empty">No active effects</div>' +
            '</div>';

        document.body.appendChild(el);
        root = el;
        makeTabSwitcher(el);
        el.querySelectorAll('.cw-tab-panel').forEach(p => p.setAttribute('role', 'tabpanel'));
        overview.forEach(build => el.querySelector('#cw-overview').appendChild(build()));
        hosted.forEach(attachHosted);
        let saved = null;
        try { saved = localStorage.getItem(SUBTAB_KEY); } catch (e) { /* unavailable */ }
        showPanel(el, saved || 'cw-overview', false);

        STAT_DEFS.forEach(d => {
            const cell  = el.querySelector('.cw-stat-cell:has(#cw-stat-' + d.key + ')');
            if (cell) {
                cell.addEventListener('click', () => Client.GMCPRequest('Help', d.key));
            }
            const modEl = el.querySelector('#cw-stat-mod-' + d.key);
            if (modEl) {
                modEl.addEventListener('mouseenter', () => showStatTooltip(modEl, 'How much of this stat is due to equipment, buffs and pets.'));
                modEl.addEventListener('mouseleave', hideStatTooltip);
            }
        });

        const spBadge = el.querySelector('#cw-badge-sp');
        if (spBadge) { spBadge.addEventListener('click', () => Client.GMCPRequest('Help', 'stat-train')); }
        const tpBadge = el.querySelector('#cw-badge-tp');
        if (tpBadge) { tpBadge.addEventListener('click', () => Client.GMCPRequest('Help', 'train')); }

        return el;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Character', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        factory() {
            const el = createDOM();
            return {
                title:      'Character',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     180,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Update functions
    // -----------------------------------------------------------------------
    function updateOverview() {
        const info = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info;
        if (!info) { return; }

        const nameEl = document.getElementById('cw-char-name');
        nameEl.innerHTML = '';

        const parts = [info.name, info.class].filter(Boolean);
        if (parts.length) {
            nameEl.appendChild(document.createTextNode(parts.join(' \u00b7 ')));
        }

        if (info.race) {
            if (parts.length) {
                nameEl.appendChild(document.createTextNode(' \u00b7 '));
            }
            const raceSpan = document.createElement('span');
            raceSpan.className   = 'cw-char-race';
            raceSpan.textContent = info.race;
            raceSpan.addEventListener('click', () => {
                Client.GMCPRequest('Help', 'race ' + info.race.toLowerCase());
            });
            nameEl.appendChild(raceSpan);
        }

        if (!nameEl.textContent) {
            nameEl.textContent = '\u2014';
        }

        document.getElementById('cw-char-level').textContent = info.level ? 'Level ' + info.level : 'Level \u2014';

        const alignEl = document.getElementById('cw-char-alignment');
        alignEl.textContent = info.alignment || '';
        const a = (info.alignment || '').toLowerCase();
        alignEl.className = 'cw-char-alignment ' +
            (a.includes('good') ? 'cw-align-good' : a.includes('evil') ? 'cw-align-evil' : 'cw-align-neutral');

        const sp      = info.skillpoints    || 0;
        const tp      = info.trainingpoints || 0;
        const spEl    = document.getElementById('cw-sp');
        const tpEl    = document.getElementById('cw-tp');
        const spBadge = document.getElementById('cw-badge-sp');
        const tpBadge = document.getElementById('cw-badge-tp');
        if (spEl)    { spEl.textContent = sp; }
        if (tpEl)    { tpEl.textContent = tp; }
        if (spBadge) { spBadge.classList.toggle('has-points', sp > 0); }
        if (tpBadge) { tpBadge.classList.toggle('has-points', tp > 0); }
    }

    function updateStats() {
        const stats = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Stats;
        if (!stats) { return; }

        STAT_DEFS.forEach(def => {
            const el = document.getElementById('cw-stat-' + def.key);
            if (el) { el.textContent = stats[def.key] || '\u2014'; }

            const mod   = stats[def.key + 'mod'];
            const modEl = document.getElementById('cw-stat-mod-' + def.key);
            if (modEl) {
                if (mod) {
                    modEl.textContent = '(' + mod + ')';
                    modEl.classList.add('visible');
                } else {
                    modEl.textContent = '';
                    modEl.classList.remove('visible');
                }
            }
        });
    }

    function updateSkills() {
        const skillList = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Skills;
        const panel = document.getElementById('cw-skills');
        if (!panel) { return; }

        if (!Array.isArray(skillList) || skillList.length === 0) {
            panel.innerHTML = '<div class="csk-empty">No skills learned</div>';
            return;
        }

        const sorted = [...skillList].sort((a, b) => (a.name || '').localeCompare(b.name || ''));
        panel.innerHTML = '';

        sorted.forEach(function(skill) {
            const level   = skill.level   || 0;
            const isMax   = skill.maximum || false;
            const MAX_LVL = 4;

            const row = document.createElement('div');
            row.className = 'csk-row';
            row.style.cursor = 'help';

            const nameEl = document.createElement('span');
            nameEl.className   = 'csk-name';
            nameEl.textContent = skill.name || '';

            const pipsEl = document.createElement('span');
            pipsEl.className = 'csk-pips';
            for (var i = 1; i <= MAX_LVL; i++) {
                const pip = document.createElement('span');
                pip.className = 'csk-pip' + (i <= level ? ' filled' + (isMax ? ' max' : '') : '');
                pipsEl.appendChild(pip);
            }

            row.appendChild(nameEl);
            row.appendChild(pipsEl);

            if (isMax) {
                const badge = document.createElement('span');
                badge.className   = 'csk-badge';
                badge.textContent = 'MAX';
                row.appendChild(badge);
            }

            row.addEventListener('click', function() {
                Client.GMCPRequest('Help', (skill.name || '').toLowerCase().replace(/\s+/g, '-'));
            });
            panel.appendChild(row);
        });
    }

    function updateJobs() {
        const jobs  = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Jobs;
        const panel = document.getElementById('cw-jobs');
        if (!panel) { return; }

        if (!Array.isArray(jobs) || jobs.length === 0) {
            panel.innerHTML = '<div class="cjb-empty">No job progress</div>';
            return;
        }

        const sorted = [...jobs].sort(function(a, b) {
            if (b.completion !== a.completion) { return b.completion - a.completion; }
            return (a.name || '').localeCompare(b.name || '');
        });

        panel.innerHTML = '';

        sorted.forEach(function(job) {
            const pct      = Math.max(0, Math.min(100, job.completion || 0));
            const complete = pct >= 100;

            const item = document.createElement('div');
            item.className    = 'cjb-item' + (complete ? ' complete' : '');
            item.style.cursor = 'help';
            item.innerHTML =
                '<div class="cjb-header">' +
                    '<span class="cjb-name">' + (job.name || '') + '</span>' +
                    '<div class="cjb-meta">' +
                        '<span class="cjb-proficiency">' + (job.proficiency || '') + '</span>' +
                        '<span class="cjb-pct">' + pct + '%</span>' +
                    '</div>' +
                '</div>' +
                '<div class="cjb-bar-track">' +
                    '<div class="cjb-bar-fill' + (complete ? ' complete' : '') + '" style="width:' + pct + '%"></div>' +
                '</div>';

            item.addEventListener('click', function() {
                Client.GMCPRequest('Help', (job.name || '').toLowerCase().replace(/\s+/g, '-'));
            });
            panel.appendChild(item);
        });
    }

    function updateQuests() {
        const quests = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Quests;
        const panel  = document.getElementById('cw-quests');
        if (!panel) { return; }

        const expanded = new Set();
        panel.querySelectorAll('.cq-item.expanded').forEach(el => {
            expanded.add(el.dataset.questName);
        });

        panel.innerHTML = '';

        if (!Array.isArray(quests) || quests.length === 0) {
            panel.innerHTML = '<div class="cq-empty">No active quests</div>';
            return;
        }

        const sorted = [...quests].sort((a, b) => {
            const ac = (a.completion || 0) >= 100;
            const bc = (b.completion || 0) >= 100;
            if (ac !== bc) { return ac ? 1 : -1; }
            if (a.completion !== b.completion) { return a.completion - b.completion; }
            return (a.name || '').localeCompare(b.name || '');
        });

        sorted.forEach(q => {
            const pct        = Math.max(0, Math.min(100, q.completion || 0));
            const complete   = pct >= 100;
            const isExpanded = expanded.has(q.name);

            const item = document.createElement('div');
            item.className       = 'cq-item' + (complete ? ' complete' : '') + (isExpanded ? ' expanded' : '');
            item.dataset.questName = q.name || '';
            item.innerHTML =
                '<div class="cq-header">' +
                    '<span class="cq-name">' + (q.name || 'Unknown Quest') + '</span>' +
                    '<span class="cq-pct">' + (complete ? 'Complete' : pct + '%') + '</span>' +
                '</div>' +
                '<div class="cq-bar-track">' +
                    '<div class="cq-bar-fill' + (complete ? ' complete' : '') + '" style="width:' + pct + '%"></div>' +
                '</div>' +
                '<div class="cq-desc">' + (q.description || '') + '</div>';

            item.addEventListener('click', () => item.classList.toggle('expanded'));
            panel.appendChild(item);
        });
    }

    function _isDebuff(mods) {
        if (!mods) { return false; }
        return Object.values(mods).some(v => v < 0);
    }

    function _formatMods(mods) {
        if (!mods || Object.keys(mods).length === 0) { return ''; }
        return Object.entries(mods)
            .map(([k, v]) => (v >= 0 ? '+' : '') + v + ' ' + k)
            .join('  ');
    }

    function updateEffects() {
        const affects = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Affects;
        const panel   = document.getElementById('cw-effects');
        if (!panel || !affects) { return; }

        panel.innerHTML = '';

        const keys = Object.keys(affects);
        if (keys.length === 0) {
            panel.innerHTML = '<div class="cw-affect-empty">No active effects</div>';
            return;
        }

        keys.sort((a, b) => {
            const da = _isDebuff(affects[a].affects);
            const db = _isDebuff(affects[b].affects);
            if (da !== db) { return da ? 1 : -1; }
            const pa = affects[a].duration_max === -1;
            const pb = affects[b].duration_max === -1;
            if (pa !== pb) { return pa ? 1 : -1; }
            return a.localeCompare(b);
        });

        keys.forEach(key => {
            const aff     = affects[key];
            const debuff  = _isDebuff(aff.affects);
            const perma   = aff.duration_max === -1;
            const modText = _formatMods(aff.affects);

            let durPct = 100;
            if (!perma && aff.duration_max > 0) {
                durPct = Math.max(0, Math.min(100, Math.round((aff.duration_left / aff.duration_max) * 100)));
            }

            let timeLabel;
            if (perma) {
                timeLabel = 'Unlimited';
            } else {
                const secs = aff.duration_left;
                if (secs <= 0) {
                    timeLabel = 'Expiring';
                } else if (secs < 60) {
                    timeLabel = secs + 's remaining';
                } else if (secs < 3600) {
                    timeLabel = Math.ceil(secs / 60) + 'm remaining';
                } else {
                    timeLabel = Math.ceil(secs / 3600) + 'h remaining';
                }
            }

            const tooltipHtml = (() => {
                let h = '<div class="cw-tt-name">' + (aff.name || key) + '</div>';
                if (aff.description) {
                    h += '<hr class="cw-tt-divider">' +
                         '<div>' + aff.description + '</div>';
                }
                h += '<hr class="cw-tt-divider">' +
                     '<div class="cw-tt-row">' +
                         '<span class="cw-tt-row-label">Time left</span>' +
                         '<span class="cw-tt-row-value">' + timeLabel + '</span>' +
                     '</div>';
                return h;
            })();

            const item = document.createElement('div');
            item.className = 'cw-affect-item' + (debuff ? ' debuff' : '');
            item.innerHTML =
                '<div class="cw-affect-header">' +
                    '<span class="cw-affect-name">' + (aff.name || key) + '</span>' +
                    '<span class="cw-affect-source">' + (aff.type || '') + '</span>' +
                '</div>' +
                (modText ? '<div class="cw-affect-mods">' + modText + '</div>' : '') +
                '<div class="cw-affect-dur-track">' +
                    '<div class="cw-affect-dur-fill' + (perma ? ' permanent' : '') + '" style="width:' + durPct + '%"></div>' +
                '</div>';

            item.addEventListener('mouseenter', () => showStatTooltip(item, tooltipHtml));
            item.addEventListener('mouseleave', hideStatTooltip);
            item.addEventListener('mousemove', () => { if (statTooltip && statTooltip.style.display === 'block') { _positionStatTooltip(item); } });

            panel.appendChild(item);
        });
    }

    function update() {
        win.open();
        if (!win.isOpen()) { return; }
        updateOverview();
        updateStats();
        updateQuests();
        updateSkills();
        updateJobs();
        updateEffects();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Info', 'Char.Stats', 'Char.Quests', 'Char.Skills', 'Char.Jobs', 'Char.Affects', 'Char'],
        onGMCP() { update(); },
    });

})();
