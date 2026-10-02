/* global Client, VirtualWindows, injectStyles */
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
 *   CompanyData.sentence(text)     text ending in one full stop ('' if blank)
 *   CompanyData.condition(c, kind) a Company.Conditions entry as an element:
 *                                  name, Harmful/Helpful when known, a live countdown and
 *                                  duration meter, description and modifiers.
 *                                  The server resends a condition only when it
 *                                  starts, is refreshed or ends (Phase 34
 *                                  review); its seconds_left counts down here,
 *                                  from the moment the message arrived.
 *   CompanyData.conditionLabel(c, kind)  the same as one line, for a label.
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

    function sentence(text) {
        const t = String(text || '').trim();
        if (!t) { return ''; }
        return /[.!?]$/.test(t) ? t : t + '.';
    }

    injectStyles(`
        .cmp-condition {
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 4px;
            padding: 4px 6px;
            margin: 2px 0;
            font-size: 0.8em;
            display: flex;
            flex-direction: column;
            gap: 2px;
            min-width: 0;
            overflow-wrap: anywhere;
        }
        /* Harm shows in the border and the word Harmful; all text keeps the
           theme's primary colour, the only one readable on this surface in
           every theme (the debuff and secondary tints are not). */
        .cmp-condition.harmful { border-color: var(--t-debuff-border); border-left-width: 4px; }
        .cmp-condition-head { display: flex; justify-content: space-between; gap: 6px; }
        .cmp-condition-name { font-weight: bold; }
        .cmp-condition-tag, .cmp-condition-duration { font-size: 0.92em; }
        .cmp-condition.harmful .cmp-condition-tag { font-weight: bold; }
        .cmp-condition-meter { height: 4px; background: var(--t-bg-row); border-radius: 2px; overflow: hidden; }
        .cmp-condition-fill { height: 100%; background: var(--t-accent-dim); transition: width 1s linear; }
        .cmp-condition.harmful .cmp-condition-fill { background: var(--t-debuff-border); }
    `);

    // When the current Company.Conditions arrived: its countdowns run from here.
    let conditionsAt = Date.now();
    VirtualWindows.register({
        gmcpHandlers: ['Company.Conditions'],
        onGMCP() { conditionsAt = Date.now(); },
    });

    function remainingText(seconds) {
        return seconds > 0 ? formatSeconds(seconds) + ' remaining' : 'Ending';
    }

    function modsText(mods) {
        return Object.keys(mods || {}).sort()
            .map(k => k + ' ' + (mods[k] > 0 ? '+' : '') + mods[k]).join(', ');
    }

    // paint shows a countdown node (its text, or a meter) as of now.
    function paint(node, now) {
        const deadline = Number(node.dataset.cdDeadline);
        const total = Number(node.dataset.cdTotal) || 0;
        const left = Math.max(0, Math.ceil((deadline - now) / 1000));
        if (node.dataset.cdRole === 'meter') {
            const pct = total > 0 ? Math.min(100, Math.round(left * 100 / total)) : 0;
            node.setAttribute('aria-valuenow', String(left));
            node.setAttribute('aria-valuetext', remainingText(left));
            node.firstChild.style.width = pct + '%';
        } else {
            node.textContent = remainingText(left);
        }
    }

    // One ticker updates every live countdown on the page.
    setInterval(() => {
        const now = Date.now();
        document.querySelectorAll('[data-cd-deadline]').forEach(node => paint(node, now));
    }, 1000);

    // effectTag names an effect known to harm or help; others get none.
    function effectTag(c, kind) {
        if (kind !== 'effect') { return ''; }
        return c.harmful ? 'Harmful' : (c.helpful ? 'Helpful' : '');
    }

    // condition renders one entry; kind is 'effect', 'wound' or 'bonus'.
    function condition(c, kind) {
        const harmful = kind === 'wound' || !!c.harmful;
        const card = el('div', 'cmp-condition' + (harmful ? ' harmful' : ''));
        const head = el('div', 'cmp-condition-head');
        head.appendChild(el('span', 'cmp-condition-name', (c.name || '') + (c.stacks > 1 ? ' (×' + c.stacks + ')' : '')));
        const tag = effectTag(c, kind);
        if (tag) { head.appendChild(el('span', 'cmp-condition-tag', tag)); }
        card.appendChild(head);
        const timed = c.seconds_left > 0;
        const deadline = conditionsAt + (c.seconds_left || 0) * 1000;
        const now = Date.now();
        const duration = el('div', 'cmp-condition-duration', String(c.duration || ''));
        if (timed) {
            duration.dataset.cdDeadline = String(deadline);
            paint(duration, now);
        }
        card.appendChild(duration);
        if (timed && c.seconds_total > 0) {
            const meter = el('div', 'cmp-condition-meter');
            meter.setAttribute('role', 'meter');
            meter.setAttribute('aria-label', (c.name || 'Effect') + ' time left');
            meter.setAttribute('aria-valuemin', '0');
            meter.setAttribute('aria-valuemax', String(c.seconds_total));
            meter.dataset.cdDeadline = String(deadline);
            meter.dataset.cdTotal = String(c.seconds_total);
            meter.dataset.cdRole = 'meter';
            meter.appendChild(el('div', 'cmp-condition-fill'));
            paint(meter, now);
            card.appendChild(meter);
        }
        const text = [sentence(c.description), modsText(c.mods)].filter(Boolean).join(' ');
        if (text) { card.appendChild(el('div', 'cmp-condition-text', text)); }
        return card;
    }

    // conditionLabel is the condition as one line, for accessible names.
    function conditionLabel(c, kind) {
        const tag = effectTag(c, kind);
        // A countdown would go stale in a label: the card's meter carries it.
        return [
            (c.name || '') + (c.stacks > 1 ? ' (×' + c.stacks + ')' : '') + (tag ? ', ' + tag.toLowerCase() : ''),
            c.seconds_left > 0 ? 'timed' : String(c.duration || ''),
            sentence(c.description).replace(/\.$/, ''),
            modsText(c.mods),
        ].filter(Boolean).join(', ');
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

    return { el, formatSeconds, kg, read, sentence, condition, conditionLabel };
})();
