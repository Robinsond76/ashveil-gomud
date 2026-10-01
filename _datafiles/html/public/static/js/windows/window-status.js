/**
 * window-status.js
 *
 * Worth - shown in the Character tab's Overview (Phase 32g; hosted
 * through window.CharacterTabs).
 *
 * Displays XP progress bar, gold (carried + bank), and how burdened the
 * player's own load leaves them in a fight (Phase 30g3, a word only).
 *
 * Responds to GMCP namespaces:
 *   Char.Worth  - XP, gold
 *   Char.Inventory.Backpack.Summary - burden
 *   Char        - full character update
 *
 * Reads:
 *   Client.GMCPStructs.Char.Worth
 *   Client.GMCPStructs.Char.Inventory.Backpack.Summary.burden
 */

'use strict';

(function() {

    injectStyles(`
        #status-window {
            flex: 0 0 auto;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
            padding: 8px 10px;
            gap: 8px;
            justify-content: flex-start;
            overflow-y: auto;
            box-sizing: border-box;
        }

        #status-window::-webkit-scrollbar       { width: 4px; }
        #status-window::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #status-window::-webkit-scrollbar-thumb  { background: var(--t-scrollbar-thumb); border-radius: 2px; }

        .sw-xp-section {
            display: flex;
            flex-direction: column;
            gap: 3px;
        }

        .sw-xp-label-row {
            display: flex;
            justify-content: space-between;
            font-size: 0.7em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .sw-xp-track {
            width: 100%;
            height: 10px;
            background: var(--t-bg-row);
            border-radius: 5px;
            overflow: hidden;
            border: 1px solid var(--t-border-faint);
        }

        .sw-xp-fill {
            height: 100%;
            border-radius: 5px;
            background: linear-gradient(to right, var(--t-progress-from), var(--t-progress-to));
            transition: width 0.4s ease-out;
        }

        .sw-worth-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 6px 10px;
        }

        .sw-worth-cell {
            display: flex;
            flex-direction: column;
            gap: 1px;
        }

        .sw-worth-cell-label {
            font-size: 0.66em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .sw-worth-cell-value {
            font-size: 0.85em;
            color: var(--t-text);
        }

        .sw-burden-link {
            cursor: pointer;
            text-decoration: underline dotted;
        }
    `);

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'status-window';
        el.innerHTML =
            '<div class="sw-xp-section">' +
                '<div class="sw-xp-label-row"><span>Experience</span><span id="sw-xp-text">\u2014 / \u2014</span></div>' +
                '<div class="sw-xp-track"><div class="sw-xp-fill" id="sw-xp-fill" style="width:0%"></div></div>' +
            '</div>' +
            '<div class="sw-worth-grid">' +
                '<div class="sw-worth-cell"><span class="sw-worth-cell-label">Gold (on hand)</span><span class="sw-worth-cell-value" id="sw-gold">\u2014</span></div>' +
                '<div class="sw-worth-cell"><span class="sw-worth-cell-label">Gold (bank)</span><span class="sw-worth-cell-value" id="sw-bank">\u2014</span></div>' +
                '<div class="sw-worth-cell"><span class="sw-worth-cell-label">Burden</span>' +
                    '<span class="sw-worth-cell-value sw-burden-link" id="sw-burden" role="button" tabindex="0" title="Your own worn and carried load in a fight. Type help burden.">\u2014</span></div>' +
            '</div>';

        const burden = el.querySelector('#sw-burden');
        const openHelp = () => Client.GMCPRequest('Help', 'burden');
        burden.addEventListener('click', openHelp);
        burden.addEventListener('keydown', e => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openHelp(); }
        });

        document.body.appendChild(el);
        return el;
    }


    // -----------------------------------------------------------------------
    // Worth update
    // -----------------------------------------------------------------------
    function fmt(n) {
        if (n === undefined || n === null) { return '\u2014'; }
        return Number(n).toLocaleString();
    }

    function updateWorth() {
        const worth = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Worth;
        if (!worth) { return; }

        const xp  = worth.xp  || 0;
        const tnl = worth.tnl || 0;
        const pct = tnl > 0 ? Math.min(100, Math.round((xp / tnl) * 100)) : 0;

        document.getElementById('sw-xp-fill').style.width = pct + '%';
        document.getElementById('sw-xp-text').textContent = fmt(xp) + ' / ' + fmt(tnl);
        document.getElementById('sw-gold').textContent    = fmt(worth.gold_carry);
        document.getElementById('sw-bank').textContent    = fmt(worth.gold_bank);
    }

    // updateBurden shows the burden word (Phase 30g3), capitalised.
    function updateBurden() {
        const inv     = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        const summary = inv && inv.Backpack && inv.Backpack.Summary;
        const el      = document.getElementById('sw-burden');
        if (!el || !summary || typeof summary.burden !== 'string' || summary.burden === '') { return; }
        el.textContent = summary.burden.charAt(0).toUpperCase() + summary.burden.slice(1);
    }

    // -----------------------------------------------------------------------
    // Update
    // -----------------------------------------------------------------------
    function update() {
        if (!document.getElementById('sw-xp-fill')) { return; }
        updateWorth();
        updateBurden();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    CharacterTabs.addToOverview(() => {
        const el = createDOM();
        setTimeout(update, 0);
        return el;
    });

    VirtualWindows.register({
        gmcpHandlers: ['Char'],
        onGMCP() { update(); },
    });

})();
