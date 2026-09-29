/**
 * window-vitals.js
 *
 * Virtual window: Vitals, the strip pinned above the company dock's tabs
 * (Phase 32g; groupHeader of the 'dock' tab group): the player's HP and MP
 * bars, a compact row per companion (an HP bar, and an MP bar for one with
 * mana; one not with you dimmed; a fallen one with its time to raise), and
 * one line of warnings (a member's need or warmth that warns, a heavy
 * load), absent when nothing warns. Companion names are set with
 * textContent, never markup.
 *
 * Responds to GMCP namespaces:
 *   Char.Vitals  - direct vitals update
 *   Char         - top-level Char update; delegates to Char.Vitals if present
 *   Company      - the company snapshot and its Company.Vitals
 *
 * Reads: Client.GMCPStructs.Char.Vitals, Client.GMCPStructs.Company
 */

'use strict';

(function() {

    const SEGMENT_COUNT = 20;

    injectStyles(`
        #vitals-bars {
            display: flex;
            flex-direction: column;
            justify-content: center;
            gap: 6px;
            padding: 4px 6px;
            box-sizing: border-box;
            background: var(--t-bg);
        }

        /* Phase 32g: a compact row per companion */
        .vitals-company {
            list-style: none;
            margin: 0;
            padding: 0;
            display: flex;
            flex-direction: column;
            gap: 3px;
        }

        .vitals-company:empty { display: none; }

        .vitals-member {
            display: grid;
            grid-template-columns: minmax(0, 7em) 1fr;
            align-items: center;
            gap: 6px;
            font-size: 0.72em;
            color: var(--t-text);
        }

        .vitals-member-name {
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .vitals-member.is-away { opacity: 0.55; }
        .vitals-member-note { color: var(--t-text-secondary); font-style: italic; }
        .vitals-member.is-fallen .vitals-member-note { color: var(--t-party-hp-low); font-style: normal; font-weight: bold; }

        .vitals-mini {
            display: flex;
            flex-direction: column;
            gap: 2px;
        }

        .vitals-mini-track {
            height: 5px;
            border-radius: 2px;
            background: var(--t-bar-empty);
            overflow: hidden;
        }

        .vitals-mini-fill { height: 100%; }
        .vitals-mini-fill.hp-filled-high { background: var(--t-hp-high); }
        .vitals-mini-fill.hp-filled-mid  { background: var(--t-hp-mid); }
        .vitals-mini-fill.hp-filled-low  { background: var(--t-hp-low); }
        .vitals-mini-fill.mp-filled      { background: linear-gradient(to right, var(--t-mana-from), var(--t-mana-to)); }

        .vitals-warn {
            font-size: 0.72em;
            color: var(--t-party-hp-low);
            font-weight: bold;
        }

        .vitals-warn[hidden] { display: none; }

        .vitals-row {
            display: flex;
            flex-direction: column;
            gap: 2px;
        }

        .vitals-label-row {
            display: flex;
            align-items: baseline;
            justify-content: space-between;
            padding: 0 1px;
        }

        .vitals-label {
            font-family: monospace;
            font-size: 0.7em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
        }

        .vitals-value {
            font-family: monospace;
            font-size: 0.72em;
            color: var(--t-text-muted);
            letter-spacing: 0.04em;
        }

        .vitals-track {
            display: flex;
            gap: 2px;
            height: 14px;
            align-items: stretch;
        }

        .vitals-segment {
            flex: 1;
            border-radius: 2px;
            transition: background 0.25s ease, box-shadow 0.25s ease, opacity 0.25s ease;
            position: relative;
        }

        /* HP segments - filled color driven by fill level */
        .vitals-segment.hp-filled-high {
            background: var(--t-hp-high);
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-hp-high) 50%, transparent);
        }

        .vitals-segment.hp-filled-mid {
            background: var(--t-hp-mid);
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-hp-mid) 40%, transparent);
        }

        .vitals-segment.hp-filled-low {
            background: var(--t-hp-low);
            box-shadow: 0 0 5px color-mix(in srgb, var(--t-hp-low) 55%, transparent);
        }

        /* Mana segments */
        .vitals-segment.mp-filled {
            background: linear-gradient(to bottom, var(--t-mana-to), var(--t-mana-from));
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-mana-to) 45%, transparent);
        }

        /* Empty segments */
        .vitals-segment.seg-empty {
            background: var(--t-bar-empty);
            box-shadow: none;
            opacity: 0.55;
        }

        /* Inset groove effect on empty */
        .vitals-segment.seg-empty::after {
            content: '';
            display: block;
            height: 100%;
            border-radius: 2px;
            background: linear-gradient(to bottom, rgba(0,0,0,0.3) 0%, transparent 60%);
        }

        /* Pulse animation on low HP */
        @keyframes vitals-pulse-low {
            0%   { opacity: 1; }
            50%  { opacity: 0.6; }
            100% { opacity: 1; }
        }

        .vitals-track.hp-critical .vitals-segment.hp-filled-low {
            animation: vitals-pulse-low 1.1s ease-in-out infinite;
        }
    `);

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function makeSegments(count, trackClass) {
        const track = document.createElement('div');
        track.className = 'vitals-track ' + trackClass;
        for (let i = 0; i < count; i++) {
            const seg = document.createElement('div');
            seg.className = 'vitals-segment seg-empty';
            track.appendChild(seg);
        }
        return track;
    }

    function createDOM() {
        const container = document.createElement('div');
        container.id = 'vitals-bars';

        // HP row
        const hpRow = document.createElement('div');
        hpRow.className = 'vitals-row';

        const hpLabelRow = document.createElement('div');
        hpLabelRow.className = 'vitals-label-row';
        hpLabelRow.innerHTML =
            '<span class="vitals-label">HP</span>' +
            '<span class="vitals-value" id="vitals-hp-value">-- / --</span>';

        const hpTrack = makeSegments(SEGMENT_COUNT, 'hp-track');
        hpTrack.id = 'vitals-hp-track';

        hpRow.appendChild(hpLabelRow);
        hpRow.appendChild(hpTrack);

        // MP row
        const mpRow = document.createElement('div');
        mpRow.className = 'vitals-row';

        const mpLabelRow = document.createElement('div');
        mpLabelRow.className = 'vitals-label-row';
        mpLabelRow.innerHTML =
            '<span class="vitals-label">MP</span>' +
            '<span class="vitals-value" id="vitals-mp-value">-- / --</span>';

        const mpTrack = makeSegments(SEGMENT_COUNT, 'mp-track');
        mpTrack.id = 'vitals-mp-track';

        mpRow.appendChild(mpLabelRow);
        mpRow.appendChild(mpTrack);

        container.appendChild(hpRow);
        container.appendChild(mpRow);

        // Phase 32g: the companions, then the warnings.
        const company = document.createElement('ul');
        company.className = 'vitals-company';
        company.id = 'vitals-company';
        company.setAttribute('aria-label', 'Companions');
        container.appendChild(company);

        const warn = document.createElement('div');
        warn.className = 'vitals-warn';
        warn.id = 'vitals-warn';
        warn.hidden = true;
        container.appendChild(warn);

        document.body.appendChild(container);
        return container;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Vitals', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        groupHeader:   true,
        factory() {
            const el = createDOM();
            // Opened or reopened: show what arrived while it was closed.
            setTimeout(() => { updateBars(); updateCompany(); }, 0);
            return {
                title:      'Vitals',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     20 + 100,
                header:     20,
                bottom:     72,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Segment update helpers
    // -----------------------------------------------------------------------
    function hpClass(pct) {
        if (pct > 60) return 'hp-filled-high';
        if (pct > 25) return 'hp-filled-mid';
        return 'hp-filled-low';
    }

    function updateTrack(trackEl, filledCount, filledClass, isCritical) {
        const segs = trackEl.children;
        for (let i = 0; i < segs.length; i++) {
            const seg = segs[i];
            if (i < filledCount) {
                seg.className = 'vitals-segment ' + filledClass;
            } else {
                seg.className = 'vitals-segment seg-empty';
            }
        }
        if (isCritical !== undefined) {
            trackEl.classList.toggle('hp-critical', isCritical);
        }
    }

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function updateBars() {
        const vitals = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Vitals;
        if (!vitals) return;

        win.open();
        if (!win.isOpen()) return;

        const hpPct = Math.max(0, Math.min(100, vitals.hp_max > 0 ? Math.floor(vitals.hp / vitals.hp_max * 100) : 0));
        const mpPct = Math.max(0, Math.min(100, vitals.sp_max > 0 ? Math.floor(vitals.sp / vitals.sp_max * 100) : 0));

        const hpFilled = Math.round(hpPct / 100 * SEGMENT_COUNT);
        const mpFilled = Math.round(mpPct / 100 * SEGMENT_COUNT);

        const hpTrack = document.getElementById('vitals-hp-track');
        const mpTrack = document.getElementById('vitals-mp-track');
        const hpVal   = document.getElementById('vitals-hp-value');
        const mpVal   = document.getElementById('vitals-mp-value');

        if (hpTrack) updateTrack(hpTrack, hpFilled, hpClass(hpPct), hpPct <= 25);
        if (mpTrack) updateTrack(mpTrack, mpFilled, 'mp-filled');
        if (hpVal)  hpVal.textContent  = vitals.hp + ' / ' + vitals.hp_max;
        if (mpVal)  mpVal.textContent  = vitals.sp + ' / ' + vitals.sp_max;
    }

    // -----------------------------------------------------------------------
    // Phase 32g: the companions and the warnings
    // -----------------------------------------------------------------------
    function miniBar(value, max, fillClass, label) {
        const pct   = max > 0 ? Math.max(0, Math.min(100, Math.round(value * 100 / max))) : 0;
        const track = CompanyData.el('div', 'vitals-mini-track');
        track.setAttribute('role', 'meter');
        track.setAttribute('aria-label', label);
        track.setAttribute('aria-valuemin', '0');
        track.setAttribute('aria-valuemax', String(max));
        track.setAttribute('aria-valuenow', String(value));
        const fill = CompanyData.el('div', 'vitals-mini-fill ' + (fillClass === 'hp' ? hpClass(pct) : 'mp-filled'));
        fill.style.width = pct + '%';
        track.appendChild(fill);
        return track;
    }

    function companionRow(m, data) {
        const v   = data.vitals(m.key);
        const row = CompanyData.el('li', 'vitals-member');
        row.setAttribute('data-key', m.key);
        row.appendChild(CompanyData.el('span', 'vitals-member-name', m.name));
        const spoken = [m.name];
        const tip = [m.name];
        if (m.status === 'dead') {
            row.classList.add('is-fallen');
            const rescue = data.live && data.live.rescue && data.live.rescue[m.key];
            const note = typeof rescue === 'number' ? 'fallen, ' + CompanyData.formatSeconds(rescue) + ' to raise' : 'fallen';
            row.appendChild(CompanyData.el('span', 'vitals-member-note', note));
            spoken.push(note);
        } else if (m.status === 'awaiting' || typeof v.hp !== 'number') {
            row.classList.add('is-away');
            row.appendChild(CompanyData.el('span', 'vitals-member-note', 'not with you'));
            spoken.push('not with you');
        } else {
            const bars = CompanyData.el('div', 'vitals-mini');
            bars.appendChild(miniBar(v.hp, v.hp_max, 'hp', m.name + ' health ' + v.hp + ' of ' + v.hp_max));
            spoken.push('health ' + v.hp + ' of ' + v.hp_max);
            tip.push('HP ' + v.hp + '/' + v.hp_max);
            if (typeof v.mp === 'number' && v.mp_max > 0) {
                bars.appendChild(miniBar(v.mp, v.mp_max, 'mp', m.name + ' mana ' + v.mp + ' of ' + v.mp_max));
                spoken.push('mana ' + v.mp + ' of ' + v.mp_max);
                tip.push('MP ' + v.mp + '/' + v.mp_max);
            }
            row.appendChild(bars);
        }
        row.title = tip.join(' \u00b7 ');
        row.setAttribute('aria-label', spoken.join(', '));
        return row;
    }

    function warnings(data) {
        const out = [];
        data.members.forEach(m => {
            if (!m || m.status === 'dead') { return; }
            const v = data.vitals(m.key);
            const who = m.key === 'leader' ? 'You' : m.name;
            const needs = v.needs || {};
            ['hunger', 'thirst', 'fatigue'].forEach(k => {
                const n = needs[k];
                if (n && n.warn && n.label) { out.push(who + ': ' + n.label); }
            });
            if (v.warmth) { out.push(who + ': ' + v.warmth); }
        });
        const load = data.company && data.company.load;
        if (load && load.capacity_g > 0 && load.total_g / load.capacity_g >= 0.9) {
            out.push('Load ' + Math.round(load.total_g * 100 / load.capacity_g) + '%');
        }
        return out;
    }

    function updateCompany() {
        const list = document.getElementById('vitals-company');
        const warn = document.getElementById('vitals-warn');
        if (!list || !warn) { return; }
        const data = CompanyData.read();
        list.textContent = '';
        data.members.forEach(m => {
            if (m && m.key && m.key !== 'leader') { list.appendChild(companionRow(m, data)); }
        });
        const words = warnings(data);
        warn.textContent = words.length ? '\u26a0 ' + words.join(' \u00b7 ') : '';
        warn.hidden = words.length === 0;
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        // handleGMCP calls a handler once per matching level; the top names
        // give one call per payload.
        gmcpHandlers: ['Char', 'Company'],
        onGMCP(namespace) {
            if (namespace.indexOf('Company') === 0) {
                win.open();
                if (win.isOpen()) { updateCompany(); }
                return;
            }
            updateBars();
        },
    });

})();
