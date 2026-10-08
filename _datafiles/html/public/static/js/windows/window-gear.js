/**
 * window-gear.js
 *
 * Gear - the Character tab's Gear sub-tab (Phase 32g; hosted through
 * window.CharacterTabs), tabbed.
 *
 * Tabs:
 *   Worn     - equipped items by slot, hover tooltips, click menu
 *   Backpack - carried items, hover tooltips, click menu; the header shows
 *              the player's own gear weight against the company's load
 *              and capacity (weight is the only limit, Phase 32f)
 *
 * Responds to GMCP namespaces:
 *   Char.Inventory          - worn equipment + backpack
 *   Char.Inventory.Backpack - backpack items only
 *   Char                    - full character update
 *   Company.Inventory       - the company's load, and each item's weight
 *
 * Reads:
 *   Client.GMCPStructs.Char.Inventory.Worn
 *   Client.GMCPStructs.Char.Inventory.Backpack
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shell ---- */
        #gear-window {
            color: var(--t-text);
            height: 100%;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
        }

        /* ---- Equipment editor (Phase 57: Company's type scale and cards) ---- */
        #gw-worn.gw-editing { padding: 6px 8px 8px; gap: 4px; font-size: 0.8em; flex-shrink: 0; }
        .gw-editor-slots, .gw-editor-choices { display: flex; flex-wrap: wrap; gap: 4px; }
        .gw-editor-members { padding-bottom: 4px; border-bottom: 1px solid var(--t-border-faint); }
        #gw-worn.gw-editing button {
            font: inherit;
            font-size: 0.95em;
            color: var(--t-text);
            background: var(--t-bg-surface);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            padding: 2px 8px;
            text-align: left;
            overflow-wrap: anywhere;
            cursor: pointer;
        }
        #gw-worn.gw-editing button[aria-pressed="true"] { border-color: var(--t-accent); background: var(--t-bg-hover); color: var(--t-text); box-shadow: inset 0 0 0 1px var(--t-accent); font-weight: bold; }
        #gw-worn.gw-editing button:disabled { opacity: 0.5; cursor: default; }
        #gw-worn.gw-editing button:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }
        @media (hover: hover) and (pointer: fine) {
            #gw-worn.gw-editing button:not(:disabled):hover { border-color: var(--t-accent); }
        }
        #gw-worn.gw-editing .gw-apply { align-self: flex-start; margin-top: 4px; border-color: var(--t-accent); color: var(--t-accent); }
        #gw-worn.gw-editing p { margin: 2px 0; line-height: 1.35; color: var(--t-text-secondary); overflow-wrap: anywhere; }
        #gw-worn.gw-editing p.gw-editor-note { font-style: italic; }
        #gw-worn.gw-editing p.gw-editor-help { font-size: 0.9em; font-style: italic; margin-top: 4px; }
        #gw-worn.gw-editing h4 {
            margin: 6px 0 2px;
            font-size: 0.82em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
            border-bottom: 1px solid var(--t-accent-dim);
            overflow-wrap: anywhere;
        }
        .gw-editor-stats { width: 100%; font-size: 0.95em; border-collapse: collapse; }
        .gw-editor-stats th, .gw-editor-stats td { padding: 1px 3px; border-bottom: 1px solid var(--t-border-faint); }
        .gw-editor-stats th { text-align: left; font-weight: normal; color: var(--t-text-secondary); }
        .gw-editor-stats thead th { font-size: 0.9em; text-transform: uppercase; letter-spacing: 0.04em; border-bottom-color: var(--t-accent-dim); }
        .gw-editor-stats thead th:not(:first-child) { text-align: right; }
        .gw-editor-stats td { text-align: right; color: var(--t-text); }
        .gw-editor-stats td + td { color: var(--t-text-secondary); }
        .gw-editor-stats td.changed { color: var(--t-accent); font-weight: bold; }
        #gear-window.gw-editor .gw-tab-bar { display: none; }
        /* ---- tab chrome ---- */
        #gear-window .gw-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        #gear-window .gw-tab-btn {
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

        #gear-window .gw-tab-btn:last-child { border-right: none; }

        @media (hover: hover) and (pointer: fine) {
            #gear-window .gw-tab-btn:hover {
                background: var(--t-border);
                color: var(--t-text);
            }
        }

        #gear-window .gw-tab-btn.active {
            background: var(--t-bg);
            color: var(--t-text);
            border-bottom: 2px solid var(--t-accent);
        }

        #gear-window .gw-tab-panel {
            display: none;
            flex: 1;
            overflow-y: auto;
        }

        #gear-window .gw-tab-panel::-webkit-scrollbar       { width: 4px; }
        #gear-window .gw-tab-panel::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #gear-window .gw-tab-panel::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 2px; }

        #gear-window .gw-tab-panel.active {
            display: flex;
            flex-direction: column;
        }

        /* ---- Worn tab ---- */
        #gw-worn {
            padding: 4px 6px;
            gap: 2px;
        }

        .gw-equip-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 18px;
            border-bottom: 1px solid var(--t-border-faint);
            padding-bottom: 2px;
            cursor: pointer;
        }

        .gw-equip-row:last-child { border-bottom: none; }

        @media (hover: hover) and (pointer: fine) {
            .gw-equip-row:hover { background: var(--t-bg-surface-alt); }
            }

        .gw-equip-slot {
            width: 54px;
            font-size: 0.66em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.03em;
            flex-shrink: 0;
        }

        .gw-equip-name {
            flex: 1;
            font-size: 0.76em;
            color: var(--t-text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .gw-equip-name.empty     { color: var(--t-text-secondary); font-style: italic; }
        .gw-equip-row.empty       { cursor: default; }
        .gw-equip-name.cursed { color: var(--t-cursed-text); }
        .gw-equip-name.quest  { color: var(--t-quest-text); }

        .gw-equip-badge {
            font-size: 0.58em;
            padding: 1px 3px;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .gw-equip-badge.cursed { background:var(--t-cursed-badge-bg); color:var(--t-cursed-text); border:1px solid var(--t-cursed-badge-border); }
        .gw-equip-badge.quest  { background:var(--t-quest-badge-bg); color:var(--t-quest-text); border:1px solid var(--t-quest-badge-border); }
        .gw-equip-badge.uses   { background:var(--t-uses-badge-bg); color:var(--t-uses-badge-text); border:1px solid var(--t-uses-badge-border); }

        /* ---- Backpack tab ---- */
        #gw-backpack {
            padding: 4px 6px;
            gap: 3px;
        }

        #gw-bp-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 3px 2px 5px;
            border-bottom: 1px solid var(--t-border);
            margin-bottom: 2px;
            flex-shrink: 0;
        }

        #gw-bp-title {
            font-size: 0.68em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        #gw-bp-count {
            font-size: 0.68em;
            color: var(--t-text-muted);
        }

        #gw-bp-count.full {
            color: var(--t-cursed-text);
        }

        #gw-bp-list {
            display: flex;
            flex-direction: column;
            gap: 2px;
            flex: 1;
        }

        .gw-bp-empty {
            color: var(--t-text-secondary);
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .gw-bp-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 18px;
            border-bottom: 1px solid var(--t-border-faint);
            padding-bottom: 2px;
            cursor: pointer;
            flex-shrink: 0;
        }

        .gw-bp-row:last-child { border-bottom: none; }

        @media (hover: hover) and (pointer: fine) {
            .gw-bp-row:hover { background: var(--t-bg-surface-alt); }
        }

        .gw-bp-type {
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

        .gw-bp-name {
            flex: 1;
            font-size: 0.76em;
            color: var(--t-text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .gw-bp-name.cursed { color: var(--t-cursed-text); }
        .gw-bp-name.quest  { color: var(--t-quest-text); }

        .gw-bp-badge {
            font-size: 0.58em;
            padding: 1px 3px;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .gw-bp-badge.cursed { background:var(--t-cursed-badge-bg); color:var(--t-cursed-text); border:1px solid var(--t-cursed-badge-border); }
        .gw-bp-badge.quest  { background:var(--t-quest-badge-bg); color:var(--t-quest-text); border:1px solid var(--t-quest-badge-border); }
        .gw-bp-badge.uses   { background:var(--t-uses-badge-bg); color:var(--t-uses-badge-text); border:1px solid var(--t-uses-badge-border); }

        /* ---- Tooltip ---- */
        #gw-item-tooltip {
            position: fixed;
            z-index: 99999;
            pointer-events: none;
            background: var(--t-bg-surface);
            border: 1px solid var(--t-border-accent);
            border-radius: 6px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.7);
            padding: 8px 10px;
            min-width: 160px;
            max-width: 260px;
            display: none;
        }

        .gw-tt-name {
            font-size: 0.85em;
            font-weight: bold;
            color: var(--t-text);
            margin-bottom: 4px;
            line-height: 1.3;
        }

        .gw-tt-name.r-uncommon { color: #5fd75f; }
        .gw-tt-name.r-rare     { color: #5f87ff; }
        .gw-tt-name.r-epic     { color: #af5fff; }
        .gw-tt-name.r-legendary { color: #ff8700; }
        .gw-tt-name.r-set      { color: #00afaf; }

        .gw-tt-relic {
            font-size: 0.75em;
            line-height: 1.5;
            color: var(--t-text);
        }

        /* Phase 67: a relic's awakenings, woken and asleep. */
        .gw-tt-relic .gw-tt-awake { color: #ffd75f; }
        .gw-tt-relic .gw-tt-sleep { color: var(--t-text-secondary); }
        /* Phase 71: a trophy enchant, on a relic or a plain item. */
        .gw-tt-relic .gw-tt-ench { color: #d787ff; }

        .gw-tt-relic .gw-tt-relic-lore {
            color: var(--t-text-secondary);
            font-style: italic;
        }

        .gw-tt-details {
            font-weight: normal;
            font-style: italic;
            color: var(--t-text-secondary);
        }

        .gw-tt-details.cursed { color: var(--t-cursed-text); }
        .gw-tt-details.quest  { color: var(--t-quest-text); }

        .gw-tt-divider {
            border: none;
            border-top: 1px solid var(--t-border-accent);
            margin: 5px 0;
        }

        .gw-tt-row {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 8px;
            font-size: 0.75em;
            line-height: 1.6;
        }

        .gw-tt-row-label {
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            font-size: 0.88em;
            flex-shrink: 0;
        }

        .gw-tt-row-value {
            color: var(--t-text);
            text-align: right;
        }

        .gw-tt-hint {
            font-size: 0.73em;
            color: var(--t-text-secondary);
            line-height: 1.4;
            font-style: italic;
        }

        .gw-tt-hint .gw-tt-cmd {
            font-style: normal;
            color: var(--t-accent);
            font-weight: bold;
        }
    `);

    // -----------------------------------------------------------------------
    // Tooltip
    // -----------------------------------------------------------------------
    const tooltip = Client.tooltip('gw-item-tooltip');
    const rowItemData = new Map();

    function _itemHint(item) {
        const type    = (item.type    || '').toLowerCase();
        const subtype = (item.subtype || '').toLowerCase();
        const details = item.details || [];

        function cmd(name) {
            return '<span class="gw-tt-cmd">' + name + '</span>';
        }

        if (details.includes('quest'))    { return 'This is a quest item.'; }
        if (type === 'readable')          { return 'You should probably ' + cmd('read') + ' this.'; }
        if (subtype === 'drinkable')      { return 'You could probably ' + cmd('drink') + ' this.'; }
        if (subtype === 'edible')         { return 'You could probably ' + cmd('eat') + ' this.'; }
        if (type === 'lockpicks')         { return 'These are used with the ' + cmd('picklock') + ' command.'; }
        if (type === 'key')               { return 'When you find the right door, keys are added to your ' + cmd('keyring') + ' automatically.'; }
        if (subtype === 'wearable')       { return 'It looks like wearable ' + type + ' equipment.'; }
        if (type === 'weapon') {
            const handsDetail = details.find(d => d.endsWith('-handed'));
            const handsText   = handsDetail || '1-handed';
            if (subtype === 'shooting') { return 'A ' + handsText + ' ranged weapon. Can be fired into adjacent areas. (' + cmd('help shoot') + ')'; }
            if (subtype === 'claws')    { return 'A ' + handsText + ' claws weapon. Can be dual wielded without training.'; }
            return 'A ' + handsText + ' weapon.';
        }
        if (subtype === 'usable') { return 'You could probably ' + cmd('use') + ' this.'; }
        return null;
    }

    // weightOf is an item's weight from Company.Inventory (Phase 32g), by
    // the reference both payloads carry; null when unknown.
    function weightOf(ref) {
        const inv = Client.GMCPStructs.Company && Client.GMCPStructs.Company.Inventory;
        const you = inv && Array.isArray(inv.members) && inv.members[0];
        if (!ref || !you) { return null; }
        const found = [].concat(you.worn || [], you.carried || []).find(i => i.ref === ref);
        return found ? found.grams : null;
    }

    function showTooltip(rowEl, item) {
        const details     = (item.details && item.details.length > 0) ? item.details.join(', ') : null;
        const detailClass = item.details && item.details.includes('cursed') ? 'cursed'
                          : item.details && item.details.includes('quest')  ? 'quest' : '';

        const rarityClass = item.rarity ? ' r-' + String(item.rarity).replace(/[^a-z]/g, '') : '';
        let html = '<div class="gw-tt-name' + rarityClass + '">' + (item.label || item.name);
        if (details) {
            html += ' <span class="gw-tt-details ' + detailClass + '">(' + details + ')</span>';
        }
        html += '</div>';

        const rows = [];
        if (item.type)     { rows.push({ label: 'Type',    value: item.type    }); }
        if (item.subtype)  { rows.push({ label: 'Subtype', value: item.subtype }); }
        if (item.uses > 0) { rows.push({ label: 'Uses',    value: item.uses    }); }
        const grams = weightOf(item.id);
        if (grams !== null) { rows.push({ label: 'Weight', value: (grams / 1000).toFixed(1) + ' kg' }); }

        if (rows.length > 0) {
            html += '<hr class="gw-tt-divider">';
            rows.forEach(r => {
                html += '<div class="gw-tt-row">' +
                    '<span class="gw-tt-row-label">' + r.label + '</span>' +
                    '<span class="gw-tt-row-value">' + r.value + '</span>' +
                '</div>';
            });
        }

        // Phase 36d: a Legendary's signature, or a set piece's set and its
        // bonuses, as the server words them.
        if (item.relic && item.relic.length > 0) {
            const esc = function (v) {
                return String(v).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
            };
            html += '<hr class="gw-tt-divider"><div class="gw-tt-relic">' +
                item.relic.map(function (line) {
                    const cls = line.indexOf('Awakened, ') === 0 ? ' class="gw-tt-awake"'
                        : line.indexOf('Sleeping, ') === 0 ? ' class="gw-tt-sleep"'
                        : line.indexOf('Enchanted with ') === 0 ? ' class="gw-tt-ench"' : '';
                    return '<div' + cls + '>' + esc(line) + '</div>';
                }).join('');
            if (item.relic_lore) {
                html += '<div class="gw-tt-relic-lore">' + esc(item.relic_lore) + '</div>';
            }
            html += '</div>';
        }

        const hint = _itemHint(item);
        if (hint) {
            html += '<hr class="gw-tt-divider"><div class="gw-tt-hint">' + hint + '</div>';
        }

        tooltip.show(html, rowEl);
    }

    function attachTooltip(rowEl) {
        rowEl.addEventListener('mouseenter', () => {
            const item = rowItemData.get(rowEl);
            if (item) { showTooltip(rowEl, item); }
        });
        rowEl.addEventListener('mouseleave', () => tooltip.hide());
        rowEl.addEventListener('mousemove', () => {
            if (tooltip.isShown()) { tooltip.position(rowEl); }
        });
    }

    // -----------------------------------------------------------------------
    // Context menu helpers
    // -----------------------------------------------------------------------
    function _equipMenuItems(item) {
        if (!item || !item.name) { return null; }
        return [
            { label: 'look '   + item.name, cmd: 'look '   + item.id },
            { label: 'remove ' + item.name, cmd: 'remove ' + item.id },
        ];
    }

    function _backpackMenuItems(item) {
        if (!item || !item.name) { return null; }
        const type    = (item.type    || '').toLowerCase();
        const subtype = (item.subtype || '').toLowerCase();
        const cmds = [{ label: 'look ' + item.name, cmd: 'look ' + item.id }];
        if (type === 'weapon' || subtype === 'wearable') {
            cmds.push({ label: 'equip ' + item.name, cmd: 'equip ' + item.id });
        } else if (subtype === 'edible') {
            cmds.push({ label: 'eat '   + item.name, cmd: 'eat '   + item.id });
        } else if (subtype === 'drinkable') {
            cmds.push({ label: 'drink ' + item.name, cmd: 'drink ' + item.id });
        } else if (subtype === 'usable') {
            cmds.push({ label: 'use '   + item.name, cmd: 'use '   + item.id });
        } else if (subtype === 'throwable') {
            cmds.push({ label: 'throw ' + item.name, cmd: 'throw ' + item.id });
        } else if (type === 'readable') {
            cmds.push({ label: 'read '  + item.name, cmd: 'read '  + item.id });
        }
        return cmds;
    }

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    function makeTabSwitcher(root) {
        Client.tabs(root, { button: '.gw-tab-btn', panel: '.gw-tab-panel' });
    }

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'gear-window';
        el.innerHTML =
            '<div class="gw-tab-bar">' +
                '<button class="gw-tab-btn active" data-panel="gw-worn">Worn</button>' +
                '<button class="gw-tab-btn"        data-panel="gw-backpack">Backpack</button>' +
            '</div>' +
            '<div class="gw-tab-panel active" id="gw-worn"></div>' +
            '<div class="gw-tab-panel" id="gw-backpack">' +
                '<div id="gw-bp-header">' +
                    '<span id="gw-bp-title">Carried Items</span>' +
                    '<span id="gw-bp-count"></span>' +
                '</div>' +
                '<div id="gw-bp-list"><div class="gw-bp-empty">Empty</div></div>' +
            '</div>';

        document.body.appendChild(el);
        makeTabSwitcher(el);
        return el;
    }


    // -----------------------------------------------------------------------
    // Update functions
    // -----------------------------------------------------------------------
    function _makeEquipRow(key) {
        const label  = key.charAt(0).toUpperCase() + key.slice(1);
        const rowEl  = document.createElement('div');
        rowEl.className = 'gw-equip-row';
        rowEl.id        = 'gw-eqrow-' + key;

        const slotEl  = document.createElement('span');
        slotEl.className   = 'gw-equip-slot';
        slotEl.textContent = label;

        const nameEl  = document.createElement('span');
        nameEl.className   = 'gw-equip-name empty';
        nameEl.id          = 'gw-eq-' + key;
        nameEl.textContent = 'empty';

        const badgeEl = document.createElement('span');
        badgeEl.className    = 'gw-equip-badge';
        badgeEl.id           = 'gw-eqb-' + key;
        badgeEl.style.display = 'none';

        rowEl.appendChild(slotEl);
        rowEl.appendChild(nameEl);
        rowEl.appendChild(badgeEl);

        attachTooltip(rowEl);
        rowEl.addEventListener('click', function(e) {
            const menuItems = _equipMenuItems(rowItemData.get(rowEl));
            if (menuItems) { uiMenu(e, menuItems); }
        });

        return rowEl;
    }

    function updateWorn() {
        const inv = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        if (!inv || !inv.Worn) { return; }

        const worn    = inv.Worn;
        const wornEl  = document.getElementById('gw-worn');
        if (!wornEl) { return; }

        // Build or reorder rows to match the current payload keys.
        const keys = Object.keys(worn);

        // Remove rows for slots no longer present in the payload.
        const existing = wornEl.querySelectorAll('.gw-equip-row');
        existing.forEach(function(row) {
            const key = row.id.replace('gw-eqrow-', '');
            if (!worn.hasOwnProperty(key)) {
                rowItemData.delete(row);
                row.remove();
            }
        });

        // Insert/reorder rows to match sorted key order.
        keys.forEach(function(key, idx) {
            let rowEl = document.getElementById('gw-eqrow-' + key);
            if (!rowEl) {
                rowEl = _makeEquipRow(key);
            }
            // Move into correct position if needed.
            const current = wornEl.children[idx];
            if (current !== rowEl) {
                wornEl.insertBefore(rowEl, current || null);
            }
        });

        // Update content of every row.
        keys.forEach(function(key) {
            const item    = worn[key];
            const rowEl   = document.getElementById('gw-eqrow-' + key);
            const nameEl  = document.getElementById('gw-eq-'    + key);
            const badgeEl = document.getElementById('gw-eqb-'   + key);
            if (!rowEl || !nameEl || !badgeEl) { return; }

            if (!item || !item.name || item.name === '-nothing-') {
                nameEl.textContent = 'empty';
                nameEl.className   = 'gw-equip-name empty';
                badgeEl.style.display = 'none';
                rowEl.classList.remove('disabled');
                rowEl.classList.add('empty');
                rowItemData.delete(rowEl);
                return;
            }

            rowEl.classList.remove('empty');
            rowItemData.set(rowEl, item);

            const isCursed = item.details && item.details.includes('cursed');
            const isQuest  = item.details && item.details.includes('quest');

            nameEl.textContent = item.label || item.name;
            nameEl.className   = 'gw-equip-name' + (isCursed ? ' cursed' : isQuest ? ' quest' : '');

            if (isCursed) {
                badgeEl.textContent = 'cursed'; badgeEl.className = 'gw-equip-badge cursed'; badgeEl.style.display = '';
            } else if (isQuest) {
                badgeEl.textContent = 'quest';  badgeEl.className = 'gw-equip-badge quest';  badgeEl.style.display = '';
            } else if (item.uses > 0) {
                badgeEl.textContent = item.uses + 'x'; badgeEl.className = 'gw-equip-badge uses'; badgeEl.style.display = '';
            } else {
                badgeEl.style.display = 'none';
            }
        });
    }

    function updateBackpack() {
        const inv = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        if (!inv || !inv.Backpack) { return; }

        const bp      = inv.Backpack;
        const items   = bp.items   || [];
        updateWeights();

        const list = document.getElementById('gw-bp-list');
        if (!list) { return; }

        list.querySelectorAll('.gw-bp-row').forEach(r => rowItemData.delete(r));
        list.innerHTML = '';

        if (items.length === 0) {
            list.innerHTML = '<div class="gw-bp-empty">Empty</div>';
            return;
        }

        const sorted = [...items].sort((a, b) => {
            const aq = a.details && a.details.includes('quest');
            const bq = b.details && b.details.includes('quest');
            if (aq !== bq) { return aq ? -1 : 1; }
            const ac = a.details && a.details.includes('cursed');
            const bc = b.details && b.details.includes('cursed');
            if (ac !== bc) { return ac ? -1 : 1; }
            return (a.name || '').localeCompare(b.name || '');
        });

        sorted.forEach(item => {
            const isCursed = item.details && item.details.includes('cursed');
            const isQuest  = item.details && item.details.includes('quest');

            const row = document.createElement('div');
            row.className = 'gw-bp-row';

            const typeEl = document.createElement('span');
            typeEl.className   = 'gw-bp-type';
            typeEl.textContent = item.type || '';

            const nameEl = document.createElement('span');
            nameEl.className   = 'gw-bp-name' + (isCursed ? ' cursed' : isQuest ? ' quest' : '');
            nameEl.textContent = item.label || item.name || '';

            const badgeEl = document.createElement('span');
            badgeEl.className = 'gw-bp-badge';
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

            rowItemData.set(row, item);
            attachTooltip(row);
            row.addEventListener('click', function(e) {
                const menuItems = _backpackMenuItems(rowItemData.get(row));
                if (menuItems) { uiMenu(e, menuItems); }
            });
        });
    }

    // updateWeights: "You 12.4 kg · company 46.0 / 50.0 kg". The company
    // figures come from Company.Inventory, which is sent whenever they
    // change, else from the backpack summary (Phase 32g).
    function updateWeights() {
        const out = document.getElementById('gw-bp-count');
        if (!out) { return; }
        const inv     = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        const summary = (inv && inv.Backpack && inv.Backpack.Summary) || {};
        const company = Client.GMCPStructs.Company && Client.GMCPStructs.Company.Inventory;
        const cargoTab = document.querySelector('[data-panel="gw-backpack"]');
        if (cargoTab) { cargoTab.textContent = (summary.shared || (company && company.shared)) ? 'Cargo' : 'Backpack'; }
        const you     = company && Array.isArray(company.members) && company.members[0];
        const load    = company && company.load;
        const mine    = you ? you.grams : summary.weight_g;
        const total   = load ? load.total_g : summary.load_g;
        const cap     = load ? load.capacity_g : summary.capacity_g;
        const parts = [];
        if (typeof mine === 'number') { parts.push('You ' + (mine / 1000).toFixed(1) + ' kg'); }
        if (cap > 0) { parts.push('company ' + (total / 1000).toFixed(1) + ' / ' + (cap / 1000).toFixed(1) + ' kg'); }
        out.textContent = parts.join(' \u00b7 ');
        out.classList.toggle('full', cap > 0 && total >= cap);
    }

    // The server supplies every compatible exact item and before/after value.
    // This view never calculates combat or capacity rules.
    let editorSlot = 'weapon';
    let editorChoice = '';
    // Phase 48: whose gear the editor shows: 'me' or a companion '#N'.
    let editorMember = 'me';
    function editorNode(tag, text, parent) {
        const node = document.createElement(tag);
        if (text !== undefined) { node.textContent = text; }
        if (parent) { parent.appendChild(node); }
        return node;
    }
    function updateEditor(view) {
        const panel = document.getElementById('gw-worn');
        if (!panel) { return; }
        const active = document.activeElement;
        const focusKey = active && panel.contains(active) && active.dataset.gearFocus;
        keepScroll(panel);
        panel.replaceChildren();
        const tabs = document.querySelectorAll('#gear-window .gw-tab-btn');
        tabs[0].textContent = 'Equipment';
        tabs[1].hidden = true;
        document.getElementById('gw-backpack').classList.remove('active');
        panel.classList.add('active', 'gw-editing');
        document.getElementById('gear-window').classList.add('gw-editor');
        tabs[0].classList.add('active');
        const members = view.members || [];
        if (editorMember !== 'me' && members.length && !members.some(mm => mm.ref === editorMember)) {
            editorMember = 'me';
            editorChoice = '';
            announceGear();
        }
        if (members.length > 1) {
            const bar = editorNode('div', undefined, panel);
            bar.className = 'gw-editor-slots gw-editor-members';
            bar.setAttribute('role', 'group');
            bar.setAttribute('aria-label', 'Whose gear');
            members.forEach(mm => {
                const b = editorNode('button', mm.ref === 'me' ? mm.name + ' (you)' : mm.name, bar);
                b.type = 'button';
                b.dataset.gearFocus = 'member:' + mm.ref;
                b.setAttribute('aria-pressed', String(mm.ref === editorMember));
                b.addEventListener('click', () => { editorMember = mm.ref; editorChoice = ''; announceGear(); updateEditor(view); });
            });
        }
        if (view.member && view.member !== editorMember) {
            // The server has not caught up with the member just chosen.
            editorNode('p', 'Loading gear…', panel).setAttribute('role', 'status');
            return;
        }
        const note = editorNode('p', view.available ? 'Select a slot, then an exact cargo item to preview.' : view.reason, panel);
        note.className = 'gw-editor-note';
        const slots = editorNode('div', undefined, panel);
        slots.className = 'gw-editor-slots';
        const list = view.slots || [];
        if (!list.some(s => s.slot === editorSlot) && list.length) { editorSlot = list[0].slot; editorChoice = ''; announceGear(); }
        list.forEach(slot => {
            const button = editorNode('button', slot.label + ': ' + (slot.equipped ? slot.equipped.label : 'empty'), slots);
            button.type = 'button';
            button.dataset.gearFocus = 'slot:' + slot.slot;
            button.setAttribute('aria-pressed', String(slot.slot === editorSlot));
            button.addEventListener('click', () => { editorSlot = slot.slot; editorChoice = ''; announceGear(); updateEditor(view); });
        });
        const slot = list.find(s => s.slot === editorSlot);
        if (!slot) { return; }
        editorNode('h4', slot.label, panel);
        editorNode('p', slot.equipped ? 'Equipped: ' + slot.equipped.label : 'This slot is empty.', panel);
        if (slot.pending) {
            // The server previews only the selected slot; it is on its way.
            editorNode('p', 'Loading choices…', panel).setAttribute('role', 'status');
            return;
        }
        const candidates = (slot.choices || []).map(c => ({ key: c.ref, choice: c }));
        if (slot.remove) { candidates.unshift({ key: 'remove:' + slot.slot + ':' + slot.remove.ref, choice: slot.remove }); }
        let selected = candidates.find(c => c.key === editorChoice);
        if (editorChoice && !selected) {
            const stale = editorNode('p', 'The selected item is no longer available. Choose again.', panel);
            stale.setAttribute('role', 'status');
            editorChoice = '';
        }
        if (!(slot.choices || []).length) { editorNode('p', 'No compatible items in shared cargo.', panel); }
        const choices = editorNode('div', undefined, panel);
        choices.className = 'gw-editor-choices';
        candidates.forEach(entry => {
            const c = entry.choice;
            const removal = entry.key.startsWith('remove:');
            const button = editorNode('button', removal ? 'Preview removal' : c.label, choices);
            button.type = 'button';
            button.dataset.gearFocus = 'choice:' + entry.key;
            button.setAttribute('aria-pressed', String(entry.key === editorChoice));
            button.addEventListener('click', () => { editorChoice = entry.key; updateEditor(view); });
            if (!c.allowed) { editorNode('p', c.reason || 'Unavailable', choices); }
        });
        selected = candidates.find(c => c.key === editorChoice);
        const candidate = selected && selected.choice;
        const after = candidate && candidate.after;
        editorNode('h4', after ? 'Current → After' : 'Current stats', panel);
        const table = editorNode('table', undefined, panel);
        table.className = 'gw-editor-stats';
        table.setAttribute('aria-label', after ? 'Current and proposed equipment stats' : 'Current equipment stats');
        const headings = editorNode('tr', undefined, editorNode('thead', undefined, table));
        ['Stat', 'Current'].concat(after ? ['After'] : []).forEach(label => {
            editorNode('th', label, headings).scope = 'col';
        });
        const body = editorNode('tbody', undefined, table);
        function valueText(value) {
            if (value === undefined || value === '') { return '—'; }
            if (typeof value === 'boolean') { return value ? 'Yes' : 'No'; }
            return String(value);
        }
        const rows = [
            ['Weapon damage', 'damage'], ['Offhand damage', 'offhand_damage'], ['Weapon edge bonus', 'edge_bonus'], ['Weapon edge strikes', 'edge_strikes'], ['Offhand edge bonus', 'offhand_edge_bonus'], ['Offhand edge strikes', 'offhand_edge_strikes'], ['Weapon poison', 'weapon_coat'], ['Offhand poison', 'offhand_coat'], ['Hands', 'hands'], ['Reach', 'reach'], ['Shield', 'shield'], ['Protection (%)', 'defense'], ['Maximum health', 'health_max'], ['Maximum mana', 'mana_max'],
            ['Worn weight (g)', 'worn_g'], ['Burden', 'burden'], ['Dodge retained (%)', 'dodge_pct'],
            ['Pack capacity (g)', 'pack_capacity_g'], ['Company capacity (g)', 'capacity_g'], ['Cargo weight (g)', 'cargo_g']
        ];
        rows.forEach(([label, key]) => {
            const row = editorNode('tr', undefined, body);
            editorNode('th', label, row).scope = 'row';
            editorNode('td', valueText(view.current[key]), row);
            if (after) { editorNode('td', valueText(after[key]), row).className = valueText(after[key]) !== valueText(view.current[key]) ? 'changed' : ''; }
        });
        Object.keys(view.current.stats || {}).forEach(key => {
            const row = editorNode('tr', undefined, body);
            editorNode('th', key, row).scope = 'row';
            editorNode('td', String(view.current.stats[key]), row);
            if (after) { editorNode('td', String(after.stats[key]), row).className = String(after.stats[key]) !== String(view.current.stats[key]) ? 'changed' : ''; }
        });
        editorNode('p', 'Burden reduces dodge. Weapon damage is its dice roll; an active edge adds damage on successful strikes until its strikes run out. A poison coating (time and hits left) may leave a poison on a foe your blade wounds. The foe and combat conditions affect actual damage.', panel).className = 'gw-editor-help';
        if (candidate) {
            if (candidate.returned && candidate.returned.length) { editorNode('p', 'Returns to cargo: ' + candidate.returned.join(', '), panel); }
            if (!candidate.allowed) { const reason = editorNode('p', candidate.reason, panel); reason.setAttribute('role', 'status'); }
            const apply = editorNode('button', selected.key.startsWith('remove:') ? 'Remove equipment' : 'Equip item', panel);
            apply.type = 'button';
            apply.className = 'gw-apply';
            apply.disabled = !candidate.allowed;
            apply.dataset.gearFocus = 'apply';
            // The command names the item by raw id; the terminal shows its name.
            const whose = editorMember === 'me' ? '' : ((members.find(mm => mm.ref === editorMember) || {}).name || '');
            const echo = (selected.key.startsWith('remove:') ? 'remove ' : 'equip ') + (whose ? whose + ' ' : '') + candidate.label;
            apply.addEventListener('click', () => Client.SendInput(candidate.command, 'company ' + echo));
        }
        if (focusKey) {
            const next = [...panel.querySelectorAll('[data-gear-focus]')].find(n => n.dataset.gearFocus === focusKey);
            const fallback = [...panel.querySelectorAll('[data-gear-focus]')].find(n => n.dataset.gearFocus === 'slot:' + editorSlot);
            if (next || fallback) { (next || fallback).focus(); }
        }
    }

    function update() {
        if (!document.getElementById('gear-window')) { return; }
        const company = Client.GMCPStructs.Company;
        if (company && company.Equipment) { updateEditor(company.Equipment); return; }
        updateWorn();
        updateBackpack();
    }

    // -----------------------------------------------------------------------
    // Registration: a sub-tab of the Character tab (Phase 32g)
    // -----------------------------------------------------------------------
    CharacterTabs.add({
        id:    'gear',
        label: 'Gear',
        order: 1,
        build() {
            const el = createDOM();
            setTimeout(update, 0);
            return el;
        },
    });

    // The server builds the Gear editor's previews only while this view is
    // shown (Phase 34 review): say "open" or "closed" when that changes. A
    // Company snapshot follows a login or reconnect, when the server has
    // forgotten, so it is said again then.
    let gearSaid = null;
    function announceGear() {
        const el = document.getElementById('gear-window');
        const shown = !!el && el.getClientRects().length > 0 && document.visibilityState === 'visible';
        // The open message names the selected slot: only its choices are
        // previewed, so a newly selected slot is announced too.
        const say = shown ? 'open ' + editorSlot + ' ' + editorMember : 'closed';
        if (say === gearSaid) { return; }
        gearSaid = say;
        Client.GMCPRequest('Company.Equipment', say);
    }
    // GearEditor.show lets the Company panel open the editor on a member's
    // slot (Phase 48): it selects them and brings the Gear tab forward.
    window.GearEditor = {
        show(member, slot) {
            editorMember = member || 'me';
            if (slot) { editorSlot = slot; }
            editorChoice = '';
            gearSaid = null;
            const tab = document.querySelector('.cw-tab-btn[data-panel="cw-hosted-gear"]');
            if (tab) { tab.click(); }
            announceGear();
        },
    };
    setInterval(announceGear, 1000);
    document.addEventListener('visibilitychange', announceGear);
    document.addEventListener('click', () => setTimeout(announceGear, 0));

    VirtualWindows.register({
        gmcpHandlers: ['Char', 'Company'],
        onGMCP(namespace) {
            if (namespace === 'Company') { gearSaid = null; announceGear(); }
            if (namespace === 'Company.Equipment' || namespace === 'Company') { update(); return; }
            if (namespace === 'Company.Inventory') {
                updateWeights();
                return;
            }
            if (namespace.indexOf('Char') === 0) { update(); }
        },
    });

})();
