/**
 * window-character.js
 *
 * Virtual window: Character - the company dock's first tab (Phase 32g),
 * with sub-tabs:
 *   Overview - name, race/class, level, alignment, stats grid, point
 *              badges, then Worth (window-status.js: XP, gold)
 *   Gear     - worn and carried items (window-gear.js)
 *   Skills   - trained ranks with what each skill does, automatic and
 *              field/camp capabilities, the companions' optional skills and
 *              training points (Phase 35c, read-only). The stock "jobs"
 *              (profession titles derived from skills) are not shown: a
 *              class names a character in Ashveil (Phase 57)
 *   Quests   - in-progress quest log, click to expand
 *   Effects  - active effects, wounds and persistent bonuses with durations
 *   Pet      - only while the player has a pet (window-pet.js)
 *
 * Other windows' scripts host their content here through
 * window.CharacterTabs (add, addToOverview, setVisible); they keep their
 * own GMCP handling. The chosen sub-tab is remembered per browser.
 *
 * Responds to GMCP namespaces:
 *   Char         - full character update
 *   Char.Info    - name, race, class, level, alignment, skill/training points,
 *                  route/tier/rank/promotion (the promoted class, Phase 38c1)
 *   Char.Stats   - six core stats
 *   Char.Quests  - quest progress
 *   Char.Skills  - skill names, titles, descriptions, levels, max flag
 *   Char.Affects - active buffs/debuffs
 *   Company      - members' optional skills and training points (35c)
 *
 * Reads:
 *   Client.GMCPStructs.Char.Info
 *   Client.GMCPStructs.Char.Stats
 *   Client.GMCPStructs.Char.Quests
 *   Client.GMCPStructs.Char.Skills
 *   Client.GMCPStructs.Char.Affects
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shared tab chrome ---- */
        #character-window {
            color: var(--t-text);
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

        @media (hover: hover) and (pointer: fine) {
            #character-window .cw-tab-btn:hover {
                background: var(--t-border);
                color: var(--t-text);
            }
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
            font-size: 0.8em;
        }

        #cw-char-name {
            display: flex;
            flex-direction: column;
            gap: 1px;
            min-width: 0;
        }

        #cw-char-name .cw-id-name {
            font-size: 1.2em;
            font-weight: bold;
            color: var(--t-text);
            overflow-wrap: anywhere;
        }

        #cw-char-name .cw-id-sub,
        #cw-char-name .cw-id-promo {
            color: var(--t-text-secondary);
            overflow-wrap: anywhere;
        }

        #cw-char-name .cw-id-promo { color: var(--t-accent); }

        #cw-char-name .cw-char-race {
            cursor: help;
            text-decoration: underline dotted;
            text-underline-offset: 2px;
        }

        #cw-char-name .cw-char-race:focus-visible { outline: 2px solid var(--t-accent); }

        #cw-id-row {
            display: flex;
            align-items: baseline;
            gap: 6px;
            color: var(--t-text-muted);
            padding-bottom: 2px;
        }

        #cw-char-alignment { font-style: italic; text-transform: capitalize; }
        #cw-char-alignment:empty { display: none; }
        #cw-char-alignment:not(:empty)::before { content: '\\00b7'; margin-right: 6px; color: var(--t-text-muted); font-style: normal; }

        /* The Worth block (window-status.js) sits flush with the rest */
        #cw-overview #status-window { padding: 4px 0 0; gap: 6px; }

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

        .cw-stat-cell:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }

        @media (hover: hover) and (pointer: fine) {
            .cw-stat-cell:hover .cw-stat-abbr,
            .cw-stat-cell:hover .cw-stat-num {
                color: var(--t-accent);
            }
        }

        .cw-stat-abbr {
            font-size: 0.8em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .cw-stat-num {
            font-size: 1em;
            color: var(--t-text);
            font-weight: bold;
            text-align: right;
        }

        .cw-stat-mod {
            font-size: 0.85em;
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

        @media (hover: hover) and (pointer: fine) {
            .cw-point-badge:hover {
                background: var(--t-border);
            }
        }

        .cw-point-badge-label {
            font-size: 0.8em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            white-space: nowrap;
        }

        .cw-point-badge-value {
            font-size: 1em;
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

        /* ---- Quests tab ---- */
        #cw-quests {
            padding: 4px 6px;
            gap: 5px;
        }

        #cw-quests .cq-empty {
            color: var(--t-text-secondary);
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

        #cw-quests .cq-item.expanded {
            background: var(--t-bg-surface);
        }

        @media (hover: hover) and (pointer: fine) {
            #cw-quests .cq-item:hover { background: var(--t-bg-surface); }
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

        #cw-quests .cq-item.complete.expanded {
            opacity: 1;
            background: var(--t-bg-surface-alt);
        }

        @media (hover: hover) and (pointer: fine) {
            #cw-quests .cq-item.complete:hover { opacity: 1; background: var(--t-bg-surface-alt); }
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
            flex-shrink: 0;
        }

        #character-window .cw-capability { font-size: 0.8em; line-height: 1.35; padding: 2px 0; overflow-wrap: anywhere; }

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

        /* ---- Skills tab (Phase 57): compact cards in Company's type scale ---- */
        #cw-skills-tab { padding-bottom: 6px; font-size: 0.8em; }
        #cw-skills-tab .cw-subhead { font-size: 0.82em; margin: 8px 6px 3px; }
        #cw-skills-tab .cw-subhead:first-child { margin-top: 6px; }

        #cw-skills, #cw-capabilities, #cw-company-skills { gap: 4px; padding: 0 6px; }

        #cw-skills .csk-empty, .csk-none {
            color: var(--t-text-secondary);
            font-style: italic;
            padding: 2px 0;
        }

        .csk-card {
            display: flex;
            flex-direction: column;
            gap: 2px;
            flex-shrink: 0;
            width: 100%;
            box-sizing: border-box;
            text-align: left;
            font: inherit;
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 4px 6px;
            overflow-wrap: anywhere;
        }

        button.csk-card { cursor: pointer; }
        button.csk-card:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }

        /* A long status badge wraps under the name instead of squeezing it to one
           letter a line (Phase 57 review: "Missing recipe ingredients..."). */
        .csk-head { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: space-between; gap: 2px 6px; }
        .csk-name { font-weight: bold; color: var(--t-text); }
        .csk-side { display: flex; flex-wrap: wrap; align-items: center; gap: 3px 5px; min-width: 0; max-width: 100%; }
        .csk-rank { color: var(--t-text-secondary); font-size: 0.9em; white-space: nowrap; }
        .csk-desc { color: var(--t-text-secondary); font-size: 0.92em; line-height: 1.35; }
        .csk-meta { color: var(--t-text-secondary); opacity: 0.85; font-size: 0.88em; line-height: 1.3; }

        .csk-pips { display: flex; gap: 2px; flex-shrink: 0; }

        .csk-pip {
            width: 7px;
            height: 7px;
            border-radius: 2px;
            border: 1px solid var(--t-accent-dim);
            background: var(--t-bg-surface-alt);
        }

        .csk-pip.filled { background: var(--t-accent); border-color: var(--t-accent); }
        .csk-pip.filled.max { background: var(--t-warning); border-color: var(--t-warning); }

        .csk-badge {
            font-size: 0.85em;
            padding: 0 4px;
            border-radius: 3px;
            flex-shrink: 1;
            min-width: 0;
            background: var(--t-bg-surface);
            color: var(--t-text-secondary);
            border: 1px solid var(--t-accent-dim);
        }

        .csk-badge.ready { color: var(--t-accent); border-color: var(--t-accent); }
        .csk-badge.max   { color: var(--t-warning); border-color: var(--t-quest-badge-border); background: var(--t-quest-badge-bg); }

        .csk-note { color: var(--t-text-secondary); font-style: italic; line-height: 1.35; padding: 2px 6px 0; }

        @media (hover: hover) and (pointer: fine) {
            button.csk-card:hover { background: var(--t-bg-surface); border-color: var(--t-accent); }
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
            color: var(--t-text-secondary);
            font-size: 0.76em;
            font-style: italic;
            text-align: center;
            padding: 14px 0;
        }

        #cw-effects .cw-subhead, #cw-effects .cw-capability { grid-column: 1 / -1; }
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
        { key: 'strength',   abbr: 'STR', name: 'Strength' },
        { key: 'speed',      abbr: 'SPD', name: 'Speed' },
        { key: 'smarts',     abbr: 'SMT', name: 'Smarts' },
        { key: 'vitality',   abbr: 'VIT', name: 'Vitality' },
        { key: 'mysticism',  abbr: 'MYS', name: 'Mysticism' },
        { key: 'perception', abbr: 'PER', name: 'Perception' },
    ];

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function buildStatsGrid() {
        const cells = STAT_DEFS.map(d =>
            '<div class="cw-stat-cell" role="button" tabindex="0" title="' + d.name + ': open its help page">' +
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
                '<div id="cw-id-row"><span id="cw-char-level">Level \u2014</span><span id="cw-char-alignment"></span></div>' +
                buildStatsGrid() +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-quests">' +
                '<div class="cq-empty">No active quests</div>' +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-skills-tab">' +
                '<h4 class="cw-subhead">Trained ranks</h4>' +
                '<div class="cw-sub" id="cw-skills">' +
                    '<div class="csk-empty">No skills learned</div>' +
                '</div>' +
                '<div class="cw-sub" id="cw-capabilities"></div>' +
                '<h4 class="cw-subhead">Company training</h4>' +
                '<div class="cw-sub" id="cw-company-skills"></div>' +
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
                const open = () => Client.GMCPRequest('Help', d.key);
                cell.addEventListener('click', open);
                cell.addEventListener('keydown', e => {
                    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(); }
                });
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

        // The name on its own line, then class, lineage and race beneath it.
        // Phase 38c1: the promoted class, its tier and the rank reached.
        const route = info.route
            ? info.route + (info.tier === 'elite' ? ' (elite)' : '') + (info.rank ? ', rank ' + info.rank : '')
            : '';
        const promo = info.promotion === 'ready' ? 'Promotion ready'
            : info.promotion === 'waiting-gate' ? 'Promotion waits on alignment' : '';
        if (info.name) { node('div', 'cw-id-name', info.name, nameEl); }
        const subParts = [info.class, route].filter(Boolean);
        if (subParts.length || info.race) {
            const sub = node('div', 'cw-id-sub', subParts.join(' \u00b7 '), nameEl);
            if (info.race) {
                if (subParts.length) { sub.appendChild(document.createTextNode(' \u00b7 ')); }
                const raceSpan = node('span', 'cw-char-race', info.race, sub);
                raceSpan.tabIndex = 0;
                raceSpan.setAttribute('role', 'button');
                raceSpan.title = 'Open the help page for ' + info.race;
                const openRace = () => Client.GMCPRequest('Help', 'race ' + info.race.toLowerCase());
                raceSpan.addEventListener('click', openRace);
                raceSpan.addEventListener('keydown', e => {
                    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openRace(); }
                });
            }
        }
        if (promo) { node('div', 'cw-id-promo', promo, nameEl); }

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

    // A small element builder; every server string goes in as text.
    function node(tag, cls, text, parent) {
        const n = document.createElement(tag);
        if (cls) { n.className = cls; }
        if (text !== undefined && text !== '') { n.textContent = text; }
        if (parent) { parent.appendChild(n); }
        return n;
    }

    // "dual-wield" -> "Dual Wield" when the server sent no display title.
    function skillTitle(skill) {
        if (skill.title) { return skill.title; }
        return String(skill.name || '').replace(/[-_]+/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
    }

    function updateSkills() {
        const skillList = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Skills;
        const panel = document.getElementById('cw-skills');
        if (!panel) { return; }

        if (!Array.isArray(skillList) || skillList.length === 0) {
            panel.innerHTML = '<div class="csk-empty">No skills learned</div>';
            return;
        }

        const focusedSkill = panel.contains(document.activeElement) ? document.activeElement.dataset.skill : null;
        const sorted = [...skillList].sort((a, b) => skillTitle(a).localeCompare(skillTitle(b)));
        keepScroll(panel);
        panel.innerHTML = '';

        sorted.forEach(function(skill) {
            const level   = skill.level   || 0;
            const isMax   = skill.maximum || false;
            const MAX_LVL = Math.max(1, Math.min(20, skill.max_level || 4));
            const title   = skillTitle(skill);

            const row = node('button', 'csk-card csk-row');
            row.type = 'button';
            row.dataset.skill = skill.name || '';
            row.setAttribute('aria-label', title + ', rank ' + level + ' of ' + MAX_LVL + ', help');
            row.title = 'Open the help page for ' + title;

            const head = node('span', 'csk-head', '', row);
            node('span', 'csk-name', title, head);
            const side = node('span', 'csk-side', '', head);
            if (isMax) { node('span', 'csk-badge max', 'MAX', side); }
            node('span', 'csk-rank', level + '/' + MAX_LVL, side);
            const pipsEl = node('span', 'csk-pips', '', side);
            for (let i = 1; i <= MAX_LVL; i++) {
                node('span', 'csk-pip' + (i <= level ? ' filled' + (isMax ? ' max' : '') : ''), '', pipsEl);
            }
            if (skill.description) { node('span', 'csk-desc', skill.description, row); }

            row.addEventListener('click', function() {
                Client.GMCPRequest('Help', (skill.name || '').toLowerCase().replace(/\s+/g, '-'));
            });
            panel.appendChild(row);
            if (focusedSkill === skill.name) { row.focus(); }
        });
    }

    function capabilityText(panel, text, heading) {
        const n = node(heading ? 'h4' : 'div', heading ? 'cw-subhead' : 'cw-capability', text, panel);
        n.style.overflowWrap = 'anywhere';
        return n;
    }

    // One capability as a card: name and status chip, what it does, and its
    // terms (skill, trigger, cooldown) in a quieter line.
    function capabilityCard(panel, c, o) {
        const card = node('div', 'csk-card csk-capability', '', panel);
        const head = node('div', 'csk-head', '', card);
        node('span', 'csk-name', c.name, head);
        const side = node('span', 'csk-side', '', head);
        if (o.manual) { node('span', 'csk-badge', 'Manual', side); }
        node('span', 'csk-badge' + (c.enabled ? ' ready' : ''), o.status, side);
        const desc = CompanyData.sentence(c.description).replace(/^./, ch => ch.toUpperCase());
        if (desc) { node('div', 'csk-desc', desc, card); }
        if (o.meta.length) { node('div', 'csk-meta', o.meta.join(' \u00b7 '), card); }
    }

    function updateCapabilities() {
        const panel = document.getElementById('cw-capabilities');
        if (!panel) { return; }
        keepScroll(panel);
        panel.textContent = '';
        const caps = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Capabilities;
        if (!caps) { node('div', 'csk-none', 'Current capabilities unavailable', panel); return; }
        capabilityText(panel, 'Automatic combat abilities', true).style.margin = '8px 0 0';
        (caps.automatic || []).forEach(c => capabilityCard(panel, c, {
            status: c.enabled ? 'Ready' : (c.reason || 'Disabled'),
            meta: ['Uses ' + c.skill, c.when ? 'When ' + c.when : '', c.cooldown > 0 ? 'Cooldown ' + c.cooldown + ' rounds' : ''].filter(Boolean),
        }));
        if (!(caps.automatic || []).length) { node('div', 'csk-none', 'No trained automatic combat abilities', panel); }
        ['field', 'camp'].forEach(group => {
            capabilityText(panel, group === 'field' ? 'Field capabilities' : 'Camp capabilities', true).style.margin = '8px 0 0';
            const list = (caps.utility || []).filter(c => c.group === group);
            list.forEach(c => capabilityCard(panel, c, {
                manual: c.mode === 'manual',
                status: c.enabled ? 'Eligible' : (c.reason || 'Unavailable'),
                meta: [c.skill.replace(/^./, ch => ch.toUpperCase()) + ' rank ' + c.rank, c.mode === 'manual' ? '' : 'automatic'].filter(Boolean),
            }));
            if (!list.length) { node('div', 'csk-none', 'No current capabilities', panel); }
        });
        node('div', 'csk-note', 'The best eligible company specialist does automatic field and camp work when its conditions hold. Camp Cooking is manual: camp cook. See help specialists and help cooking.', panel);
    }

    // Phase 35c: each companion's optional skills and training points, read
    // only (train with "company train").
    function updateCompanySkills() {
        const panel = document.getElementById('cw-company-skills');
        if (!panel) { return; }
        keepScroll(panel);
        panel.textContent = '';
        const company = Client.GMCPStructs.Company;
        const members = (company && Array.isArray(company.members)) ? company.members : [];
        if (!members.length) { node('div', 'csk-none', 'No companions', panel); return; }
        members.forEach(m => {
            const skills = m.skills || {};
            const ranks = Object.keys(skills).sort().map(id => id.charAt(0).toUpperCase() + id.slice(1) + ' rank ' + skills[id]);
            const points = typeof m.training_points === 'number' ? m.training_points : 0;
            const card = node('div', 'csk-card', '', panel);
            const head = node('div', 'csk-head', '', card);
            node('span', 'csk-name', m.name || m.key, head);
            node('span', 'csk-badge' + (points > 0 ? ' ready' : ''), points + ' training point' + (points === 1 ? '' : 's'), head);
            node('div', 'csk-desc', ranks.length ? ranks.join(', ') : 'No optional skills', card);
        });
        node('div', 'csk-note', 'Companions learn optional skills with company train. See help company-train.', panel);
    }

    function updateQuests() {
        const quests = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Quests;
        const panel  = document.getElementById('cw-quests');
        if (!panel) { return; }

        const expanded = new Set();
        panel.querySelectorAll('.cq-item.expanded').forEach(el => {
            expanded.add(el.dataset.questName);
        });

        keepScroll(panel);

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

    function _formatMods(mods) {
        if (!mods || Object.keys(mods).length === 0) { return ''; }
        return Object.entries(mods)
            .map(([k, v]) => (v >= 0 ? '+' : '') + v + ' ' + k)
            .join('  ');
    }

    function updateEffects() {
        const panel = document.getElementById('cw-effects');
        if (!panel) { return; }
        const all = Client.GMCPStructs.Company && Client.GMCPStructs.Company.Conditions;
        const state = all && all.leader;
        keepScroll(panel);
        panel.textContent = '';
        if (state) {
            [['effects', 'Active effects', 'effect'], ['wounds', 'Wounds', 'wound'], ['bonuses', 'Persistent bonuses', 'bonus']].forEach(group => {
                capabilityText(panel, group[1], true);
                const entries = state[group[0]] || [];
                entries.forEach(c => panel.appendChild(CompanyData.condition(c, group[2])));
                if (!entries.length) { capabilityText(panel, 'None'); }
            });
            return;
        }
        // Legacy Char.Affects remains usable without a Company provider.
        const affects = (Client.GMCPStructs.Char && Client.GMCPStructs.Char.Affects) || {};
        Object.keys(affects).sort().forEach(key => {
            const c = affects[key];
            capabilityText(panel, [(c.name || key) + ' — ' + (c.duration_max === -1 ? 'Persistent' : c.duration_left + ' seconds remaining') + '.',
                CompanyData.sentence(c.description), _formatMods(c.affects || {})].filter(Boolean).join(' '));
        });
        if (!Object.keys(affects).length) { capabilityText(panel, 'No active effects'); }
    }

    function update() {
        win.open();
        if (!win.isOpen()) { return; }
        updateOverview();
        updateStats();
        updateQuests();
        updateSkills();
        updateCapabilities();
        updateCompanySkills();
        updateEffects();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char', 'Company'],
        onGMCP(namespace) {
            // A Company snapshot only changes the company training list;
            // of its extras, only Conditions (Effects) is shown here.
            if (namespace === 'Company') {
                if (win.isOpen()) { updateCompanySkills(); }
                return;
            }
            if (namespace.startsWith('Company.') && namespace !== 'Company.Conditions') { return; }
            update();
        },
    });

})();
