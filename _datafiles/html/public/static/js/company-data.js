/* global Client */
/**
 * company-data.js (Phase 32g)
 *
 * Helpers the company dock's windows share: the vitals strip
 * (window-vitals.js), the Company tab (window-company.js), and the Combat
 * tab (window-combat.js). Loaded after webclient-core.js, before them.
 *
 *   CompanyData.read()    the Company snapshot with its newest live half
 *                         (a Company.Vitals that arrived after the snapshot,
 *                         else the snapshot's own): { company, live,
 *                         members, vitals(key) }, company null when none.
 *   CompanyData.el(tag, className, text)
 *                         an element whose text is set with textContent,
 *                         never markup: server strings are data.
 *   CompanyData.formatSeconds(s)   "45s", "12m", "1h 30m"
 *   CompanyData.kg(grams)          "12.4 kg"
 */

'use strict';

window.CompanyData = (function() {

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

    function kg(grams) {
        return ((grams || 0) / 1000).toFixed(1) + ' kg';
    }

    function read() {
        const stored  = Client.GMCPStructs.Company;
        const company = (stored && stored.leader) ? stored : null;
        let live = null;
        if (company) {
            const newer = stored.Vitals;
            live = (newer && typeof newer === 'object' && newer.vitals) ? newer : company;
        }
        const members = company
            ? [company.leader].concat(Array.isArray(company.members) ? company.members : [])
            : [];
        return {
            company,
            live,
            members,
            vitals(key) { return (live && live.vitals && live.vitals[key]) || {}; },
        };
    }

    return { el, formatSeconds, kg, read };
})();
