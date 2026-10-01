/**
 * window-pet.js
 *
 * Pet - the Character tab's Pet sub-tab, shown only while the player has
 * a pet (Phase 32g; hosted through window.CharacterTabs), tabbed.
 *
 * Tabs:
 *   Info  - pet identity, level bar, hunger, combat, stats, buffs
 *   Items - carried items with context menus
 *
 * Responds to GMCP namespaces:
 *   Char.Pets - pet data
 *   Char      - full character update
 *
 * Reads:
 *   Client.GMCPStructs.Char.Pets
 */

'use strict';

(function() {

    var LEVEL_SEGMENTS = 10;

    injectStyles(`
        /* ---- shell ---- */
        #pet-window {
            height: 100%;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
        }

        /* ---- tab chrome ---- */
        #pet-window .pw-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        #pet-window .pw-tab-btn {
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

        #pet-window .pw-tab-btn:last-child { border-right: none; }

        #pet-window .pw-tab-btn:hover {
            background: var(--t-border);
            color: var(--t-text);
        }

        #pet-window .pw-tab-btn.active {
            background: var(--t-bg);
            color: var(--t-text);
            border-bottom: 2px solid var(--t-accent);
        }

        #pet-window .pw-tab-panel {
            display: none;
            flex: 1;
            overflow-y: auto;
        }

        #pet-window .pw-tab-panel::-webkit-scrollbar       { width: 4px; }
        #pet-window .pw-tab-panel::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #pet-window .pw-tab-panel::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 2px; }

        #pet-window .pw-tab-panel.active {
            display: flex;
            flex-direction: column;
        }

        /* ---- Info tab ---- */
        #pw-info {
            padding: 8px 10px;
            gap: 6px;
        }

        .pw-no-pet {
            color: var(--t-text-dim);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 16px 0;
        }

        /* identity header */
        .pw-identity {
            display: flex;
            flex-direction: column;
            gap: 1px;
        }

        .pw-pet-name {
            font-size: 0.88em;
            color: var(--t-text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .pw-pet-type {
            font-size: 0.7em;
            color: var(--t-text-muted);
            text-transform: capitalize;
        }

        /* level bar */
        .pw-level-section {
            display: flex;
            flex-direction: column;
            gap: 3px;
        }

        .pw-level-label-row {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            padding: 0 1px;
        }

        .pw-level-label {
            font-family: monospace;
            font-size: 0.7em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
        }

        .pw-level-value {
            font-family: monospace;
            font-size: 0.72em;
            color: var(--t-text-muted);
            letter-spacing: 0.04em;
        }

        .pw-level-track {
            display: flex;
            gap: 2px;
            height: 10px;
            align-items: stretch;
        }

        .pw-level-seg {
            flex: 1;
            border-radius: 2px;
            transition: background 0.25s ease, box-shadow 0.25s ease;
        }

        .pw-level-seg.filled {
            background: var(--t-accent);
            box-shadow: 0 0 3px color-mix(in srgb, var(--t-accent) 40%, transparent);
        }

        .pw-level-seg.empty {
            background: var(--t-bar-empty);
            opacity: 0.55;
        }

        .pw-level-seg.empty::after {
            content: '';
            display: block;
            height: 100%;
            border-radius: 2px;
            background: linear-gradient(to bottom, rgba(0,0,0,0.3) 0%, transparent 60%);
        }

        /* badge row */
        .pw-badge-row {
            display: flex;
            gap: 6px;
        }

        .pw-badge {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 2px 6px;
            background: var(--t-bg-surface);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            gap: 4px;
            min-height: 18px;
        }

        .pw-badge-label {
            font-size: 0.62em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            white-space: nowrap;
        }

        .pw-badge-value {
            font-size: 0.78em;
            color: var(--t-text);
            font-weight: bold;
            white-space: nowrap;
        }

        .pw-hunger-starving .pw-badge-value { color: var(--t-cursed-text); }
        .pw-hunger-starving { border-color: var(--t-cursed-text); }
        .pw-hunger-hungry .pw-badge-value { color: var(--t-quest-text); }
        .pw-hunger-hungry { border-color: var(--t-quest-text); }
        .pw-hunger-full .pw-badge-value { color: var(--t-accent); }
        .pw-hunger-full { border-color: var(--t-accent); }

        /* combat card */
        .pw-combat-card {
            display: flex;
            gap: 6px;
            padding: 4px 0 2px;
            border-top: 1px solid var(--t-border);
            border-bottom: 1px solid var(--t-border);
        }

        .pw-combat-cell {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 1px;
        }

        .pw-combat-cell-val {
            font-size: 0.82em;
            color: var(--t-text);
            font-weight: bold;
            font-family: monospace;
        }

        .pw-combat-cell-label {
            font-size: 0.58em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        /* stat grid */
        .pw-section-label {
            font-size: 0.64em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            padding: 2px 0 2px;
        }

        .pw-stat-grid {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 2px 6px;
        }

        .pw-stat-cell {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 3px;
        }

        .pw-stat-cell:hover .pw-stat-abbr,
        .pw-stat-cell:hover .pw-stat-num {
            color: var(--t-accent);
        }

        .pw-stat-abbr {
            font-size: 0.64em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            flex-shrink: 0;
        }

        .pw-stat-num {
            font-size: 0.78em;
            color: var(--t-text);
            font-weight: bold;
        }

        /* buff chips */
        .pw-buff-list {
            display: flex;
            flex-wrap: wrap;
            gap: 3px;
            padding: 2px 0;
        }

        .pw-buff-tag {
            font-size: 0.64em;
            padding: 2px 6px;
            border-radius: 3px;
            background: var(--t-bg-surface);
            color: var(--t-accent);
            border: 1px solid var(--t-accent-dim);
        }

        /* ---- Items tab ---- */
        #pw-items {
            padding: 4px 6px;
            gap: 3px;
        }

        #pw-items-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 3px 2px 5px;
            border-bottom: 1px solid var(--t-border);
            margin-bottom: 2px;
            flex-shrink: 0;
        }

        #pw-items-title {
            font-size: 0.68em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        #pw-items-count {
            font-size: 0.68em;
            color: var(--t-text-muted);
        }

        #pw-items-count .pw-count-num { color: var(--t-text); }
        #pw-items-count .pw-count-num.full { color: var(--t-cursed-text); }

        #pw-items-list {
            display: flex;
            flex-direction: column;
            gap: 2px;
            flex: 1;
        }

        .pw-no-items {
            color: var(--t-text-dim);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .pw-item-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 18px;
            border-bottom: 1px solid var(--t-border-faint);
            padding-bottom: 2px;
            cursor: pointer;
            flex-shrink: 0;
        }

        .pw-item-row:last-child { border-bottom: none; }
        .pw-item-row:hover { background: var(--t-bg-surface-alt); }

        .pw-item-type {
            width: 54px;
            font-size: 0.66em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.03em;
            flex-shrink: 0;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .pw-item-name {
            flex: 1;
            font-size: 0.76em;
            color: var(--t-text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .pw-item-badge {
            font-size: 0.58em;
            padding: 1px 3px;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .pw-item-badge.cursed { background:var(--t-cursed-badge-bg); color:var(--t-cursed-text); border:1px solid var(--t-cursed-badge-border); }
        .pw-item-badge.quest  { background:var(--t-quest-badge-bg); color:var(--t-quest-text); border:1px solid var(--t-quest-badge-border); }
        .pw-item-badge.uses   { background:var(--t-uses-badge-bg); color:var(--t-uses-badge-text); border:1px solid var(--t-uses-badge-border); }
    `);

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    function makeTabSwitcher(root) {
        var btns   = root.querySelectorAll('.pw-tab-btn');
        var panels = root.querySelectorAll('.pw-tab-panel');
        btns.forEach(function(btn) {
            btn.addEventListener('click', function() {
                btns.forEach(function(b)   { b.classList.remove('active'); });
                panels.forEach(function(p) { p.classList.remove('active'); });
                btn.classList.add('active');
                root.querySelector('#' + btn.dataset.panel).classList.add('active');
            });
        });
    }

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        var el = document.createElement('div');
        el.id = 'pet-window';
        el.innerHTML =
            '<div class="pw-tab-bar">' +
                '<button class="pw-tab-btn active" data-panel="pw-info">Info</button>' +
                '<button class="pw-tab-btn"        data-panel="pw-items">Items</button>' +
            '</div>' +

            '<div class="pw-tab-panel active" id="pw-info">' +
                '<div class="pw-no-pet">No pet</div>' +
            '</div>' +

            '<div class="pw-tab-panel" id="pw-items">' +
                '<div id="pw-items-header">' +
                    '<span id="pw-items-title">Carried Items</span>' +
                    '<span id="pw-items-count"><span class="pw-count-num" id="pw-item-num">0</span> / <span id="pw-item-max">—</span></span>' +
                '</div>' +
                '<div id="pw-items-list"><div class="pw-no-items">No items</div></div>' +
            '</div>';

        document.body.appendChild(el);
        makeTabSwitcher(el);
        return el;
    }


    // -----------------------------------------------------------------------
    // Helpers
    // -----------------------------------------------------------------------
    function getPet() {
        var pets = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Pets;
        if (!pets || pets.length === 0) { return null; }
        return pets[0];
    }

    function esc(s) {
        return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // -----------------------------------------------------------------------
    // Level bar builder
    // -----------------------------------------------------------------------
    function buildLevelBar(level) {
        var html = '<div class="pw-level-section">';
        html += '<div class="pw-level-label-row">' +
            '<span class="pw-level-label">Level</span>' +
            '<span class="pw-level-value">' + level + ' / ' + LEVEL_SEGMENTS + '</span>' +
        '</div>';
        html += '<div class="pw-level-track">';
        for (var i = 0; i < LEVEL_SEGMENTS; i++) {
            html += '<div class="pw-level-seg ' + (i < level ? 'filled' : 'empty') + '"></div>';
        }
        html += '</div></div>';
        return html;
    }

    // -----------------------------------------------------------------------
    // Stat abbreviations
    // -----------------------------------------------------------------------
    var STAT_ABBR = {
        strength: 'STR', speed: 'SPD', smarts: 'SMT', vitality: 'VIT',
        mysticism: 'MYS', perception: 'PER',
        healthmax: 'HP+', manamax: 'MP+', healthrecovery: 'HPR', manarecovery: 'MPR',
        attacks: 'ATK', damage: 'DMG', casting: 'CST', xpscale: 'XP%',
        picklock: 'PLK',
    };

    // -----------------------------------------------------------------------
    // Update: Info tab
    // -----------------------------------------------------------------------
    function updateInfo() {
        var panel = document.getElementById('pw-info');
        if (!panel) { return; }

        var pet = getPet();
        if (!pet) {
            panel.innerHTML = '<div class="pw-no-pet">No pet</div>';
            return;
        }

        var html = '';

        // Identity header
        html += '<div class="pw-identity">' +
            '<span class="pw-pet-name">' + esc(pet.name || pet.type || '—') + '</span>' +
            '<span class="pw-pet-type">' + esc(pet.type || '') + '</span>' +
        '</div>';

        // Level bar
        html += buildLevelBar(pet.level || 0);

        // Hunger + Capacity badges
        var hungerCls = 'pw-badge pw-hunger-' + (pet.hunger || 'full').toLowerCase();
        html += '<div class="pw-badge-row">' +
            '<div class="' + hungerCls + '">' +
                '<span class="pw-badge-label">Hunger</span>' +
                '<span class="pw-badge-value">' + esc(pet.hunger || '—') + '</span>' +
            '</div>' +
            '<div class="pw-badge">' +
                '<span class="pw-badge-label">Capacity</span>' +
                '<span class="pw-badge-value">' + (pet.capacity || 0) + '</span>' +
            '</div>' +
        '</div>';

        // Combat card
        if (pet.ability && (pet.ability.combat_chance || pet.ability.dice_roll)) {
            html += '<div class="pw-combat-card">';
            html += '<div class="pw-combat-cell">' +
                '<span class="pw-combat-cell-val">' + (pet.ability.combat_chance || 0) + '%</span>' +
                '<span class="pw-combat-cell-label">Combat</span>' +
            '</div>';
            html += '<div class="pw-combat-cell">' +
                '<span class="pw-combat-cell-val">' + esc(pet.ability.dice_roll || '—') + '</span>' +
                '<span class="pw-combat-cell-label">Damage</span>' +
            '</div>';
            html += '</div>';
        }

        // Stat mods grid
        if (pet.ability && pet.ability.stat_mods && Object.keys(pet.ability.stat_mods).length > 0) {
            html += '<div class="pw-section-label">Stat Mods</div>';
            html += '<div class="pw-stat-grid">';
            var stats = Object.entries(pet.ability.stat_mods).sort(function(a, b) {
                return a[0].localeCompare(b[0]);
            });
            for (var i = 0; i < stats.length; i++) {
                var key = stats[i][0];
                var val = stats[i][1];
                var sign = val > 0 ? '+' : '';
                var abbr = STAT_ABBR[key] || key.substring(0, 3).toUpperCase();
                html += '<div class="pw-stat-cell">' +
                    '<span class="pw-stat-abbr">' + esc(abbr) + '</span>' +
                    '<span class="pw-stat-num">' + sign + val + '</span>' +
                '</div>';
            }
            html += '</div>';
        }

        // Buffs
        if (pet.buffs && pet.buffs.length > 0) {
            html += '<div class="pw-section-label">Buffs</div>';
            html += '<div class="pw-buff-list">';
            for (var b = 0; b < pet.buffs.length; b++) {
                html += '<span class="pw-buff-tag">' + esc(pet.buffs[b]) + '</span>';
            }
            html += '</div>';
        }

        panel.innerHTML = html;
    }

    // -----------------------------------------------------------------------
    // Update: Items tab
    // -----------------------------------------------------------------------
    function updateItems() {
        var pet = getPet();
        if (!pet) { return; }

        var items    = pet.items    || [];
        var capacity = pet.capacity || 0;

        var numEl = document.getElementById('pw-item-num');
        var maxEl = document.getElementById('pw-item-max');
        if (numEl) {
            numEl.textContent = items.length;
            numEl.classList.toggle('full', capacity > 0 && items.length >= capacity);
        }
        if (maxEl) { maxEl.textContent = capacity || '—'; }

        var list = document.getElementById('pw-items-list');
        if (!list) { return; }
        list.innerHTML = '';

        if (items.length === 0) {
            list.innerHTML = '<div class="pw-no-items">No items</div>';
            return;
        }

        items.forEach(function(item) {
            var isCursed = item.details && item.details.includes('cursed');
            var isQuest  = item.details && item.details.includes('quest');

            var row = document.createElement('div');
            row.className = 'pw-item-row';

            var typeEl = document.createElement('span');
            typeEl.className   = 'pw-item-type';
            typeEl.textContent = item.type || '';

            var nameEl = document.createElement('span');
            nameEl.className   = 'pw-item-name' + (isCursed ? ' cursed' : isQuest ? ' quest' : '');
            nameEl.textContent = item.name || '';

            var badgeEl = document.createElement('span');
            badgeEl.className = 'pw-item-badge';
            if (isCursed) {
                badgeEl.textContent = 'cursed'; badgeEl.classList.add('cursed');
            } else if (isQuest) {
                badgeEl.textContent = 'quest';  badgeEl.classList.add('quest');
            } else if (item.uses > 0) {
                badgeEl.textContent = item.uses + 'x'; badgeEl.classList.add('uses');
            } else {
                badgeEl.style.display = 'none';
            }

            row.appendChild(typeEl);
            row.appendChild(nameEl);
            row.appendChild(badgeEl);
            list.appendChild(row);

            row.addEventListener('click', function(e) {
                if (!item.name) { return; }
                uiMenu(e, [
                    { label: 'look ' + item.name,              cmd: 'look ' + item.name },
                    { label: 'get ' + item.name + ' from pet', cmd: 'get ' + item.name + ' from pet' },
                ]);
            });
        });
    }

    // -----------------------------------------------------------------------
    // Main update
    // -----------------------------------------------------------------------
    function update() {
        var pet = getPet();
        CharacterTabs.setVisible('pet', !!pet);
        if (!document.getElementById('pw-info')) { return; }
        updateInfo();
        if (pet) { updateItems(); }
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    CharacterTabs.add({
        id:      'pet',
        label:   'Pet',
        order:   5,
        visible: false,
        build: function() {
            var el = createDOM();
            setTimeout(update, 0);
            return el;
        },
    });

    VirtualWindows.register({
        gmcpHandlers: ['Char'],
        onGMCP: function(namespace) {
            if (namespace === 'Char' || namespace === 'Char.Pets') { update(); }
        },
    });

})();
