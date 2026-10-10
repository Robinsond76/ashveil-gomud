/**
 * window-creation.js
 *
 * The creation panel (Phase 72a): the looks and life story steps of
 * character creation as buttons, with a live preview of the description.
 * It is shown while a step is pending and hides itself when the steps end.
 *
 * The server stays the one source of truth. It sends each step as
 * Char.Creation ({ active, mode, step, key, kind, title, options, number,
 * total, picks, preview, backstory, canback, canskip }), and the panel
 * answers with the same input a telnet player types: the option's number,
 * "back", "skip", or a typed line. Nothing is decided here.
 *
 * It also draws the player's class figure in the skin tone and hair colour
 * chosen so far (sprite-tint.js). Every string is set with textContent, never innerHTML. The panel is a
 * dialog; the current question is its label, a hidden status line announces
 * each new step, and every choice is a real button. It never captures the
 * game input line: "Type instead" tucks the panel away until the next step.
 *
 * Responds to GMCP namespaces:
 *   Char.Creation - the step now being asked ({"active":false} when over)
 */

'use strict';

(function() {

    injectStyles(`
        #creation-backdrop {
            display: none;
            position: fixed;
            inset: 0;
            z-index: 9000;
            background: rgba(0, 0, 0, 0.72);
            align-items: center;
            justify-content: center;
        }
        #creation-backdrop.open { display: flex; }
        body.creation-open #main-container,
        body.creation-open .vw-window { pointer-events: none; }

        #creation-panel {
            position: relative;
            display: flex;
            flex-direction: column;
            width: min(760px, 96vw);
            max-height: 92vh;
            box-sizing: border-box;
            background: var(--t-bg-panel);
            border: 1px solid var(--t-border-accent);
            border-radius: 6px;
            box-shadow: 0 8px 40px rgba(0, 0, 0, 0.9);
            color: var(--t-text);
            font-size: 0.9em;
            overflow: hidden;
        }

        #creation-panel .creation-head {
            display: flex;
            align-items: baseline;
            justify-content: space-between;
            gap: 12px;
            padding: 8px 14px;
            background: var(--t-bg-surface);
            border-bottom: 1px solid var(--t-border-accent);
        }
        #creation-panel .creation-heading {
            margin: 0;
            font-size: 0.9em;
            color: var(--t-accent);
            text-transform: uppercase;
            letter-spacing: 0.06em;
        }
        #creation-panel .creation-count { color: var(--t-text-secondary); font-size: 0.85em; white-space: nowrap; }

        #creation-panel .creation-body {
            flex: 1;
            overflow-y: auto;
            min-height: 0;
            padding: 10px 14px;
        }
        #creation-panel .creation-question { margin: 0 0 8px 0; font-size: 1.1em; color: var(--t-text); }

        #creation-panel .creation-options {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
            gap: 6px;
            margin: 0 0 10px 0;
            padding: 0;
            list-style: none;
        }
        #creation-panel .creation-options.wide { grid-template-columns: 1fr; }
        #creation-panel .creation-options button { width: 100%; height: 100%; }

        #creation-panel button {
            font: inherit;
            color: var(--t-text);
            background: var(--t-bg-surface);
            border: 1px solid var(--t-border);
            border-radius: 4px;
            padding: 6px 10px;
            text-align: left;
            cursor: pointer;
        }
        #creation-panel button:focus-visible { outline: 2px solid var(--t-accent); outline-offset: 1px; }
        #creation-panel button .creation-option-name { display: flex; align-items: center; gap: 8px; font-weight: bold; }
        #creation-panel button .creation-option-text { display: block; margin-top: 3px; color: var(--t-text-secondary); font-size: 0.9em; }
        #creation-panel button .creation-option-stats { display: block; margin-top: 3px; color: var(--t-accent); font-size: 0.85em; }
        #creation-panel .creation-swatch {
            display: inline-block;
            width: 16px;
            height: 16px;
            border: 1px solid var(--t-border-accent);
            border-radius: 3px;
            flex-shrink: 0;
        }
        @media (hover: hover) and (pointer: fine) {
            #creation-panel button:hover { border-color: var(--t-accent); }
        }

        #creation-panel .creation-line { display: flex; gap: 6px; flex-wrap: wrap; margin: 0 0 10px 0; }
        #creation-panel .creation-line input {
            flex: 1 1 220px;
            font: inherit;
            padding: 6px 8px;
            color: var(--t-text);
            background: var(--t-bg);
            border: 1px solid var(--t-border);
            border-radius: 4px;
        }

        #creation-panel .creation-preview {
            margin: 4px 0 0 0;
            padding: 8px 10px;
            background: var(--t-bg);
            border: 1px solid var(--t-border);
            border-radius: 4px;
            line-height: 1.45;
        }
        #creation-panel .creation-preview h3 {
            margin: 0 0 4px 0;
            font-size: 0.8em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        #creation-panel .creation-preview p { margin: 0 0 6px 0; }
        #creation-panel .creation-preview-art { display: block; margin: 0 0 6px 0; image-rendering: pixelated; background: var(--t-bg-surface); border: 1px solid var(--t-border); border-radius: 4px; }

        #creation-panel .creation-foot {
            display: flex;
            gap: 6px;
            flex-wrap: wrap;
            padding: 8px 14px;
            border-top: 1px solid var(--t-border);
            background: var(--t-bg-surface);
        }
        #creation-panel .creation-foot .creation-spacer { flex: 1; }

        #creation-tab {
            display: none;
            position: fixed;
            right: 12px;
            bottom: 12px;
            z-index: 9000;
            font: inherit;
            padding: 6px 12px;
            color: var(--t-text);
            background: var(--t-bg-surface);
            border: 1px solid var(--t-border-accent);
            border-radius: 4px;
            cursor: pointer;
        }
        #creation-tab.open { display: block; }

        .creation-sr-only {
            position: absolute;
            width: 1px;
            height: 1px;
            margin: -1px;
            padding: 0;
            overflow: hidden;
            clip: rect(0, 0, 0, 0);
            white-space: nowrap;
            border: 0;
        }

        @media (max-width: 520px) {
            #creation-panel { max-height: 100vh; width: 100vw; border-radius: 0; }
            #creation-panel .creation-options { grid-template-columns: 1fr 1fr; }
            #creation-panel .creation-options.wide { grid-template-columns: 1fr; }
        }
    `);

    const MODE_HEADINGS = {
        new:    'Create your character',
        legacy: 'Describe your character',
        edit:   'Change your looks',
        story:  'Write your life story',
    };

    let backdrop, panel, tab, statusEl;
    let state       = null;   // the last active payload
    let hidden      = false;  // tucked away by "Type instead"
    let lastStepKey = '';
    let opener      = null;

    function asString(v) { return typeof v === 'string' ? v : ''; }

    function el(tag, cls, text) {
        const node = document.createElement(tag);
        if (cls) { node.className = cls; }
        if (text !== undefined && text !== null) { node.textContent = text; }
        return node;
    }

    // A colour from the server is used only when it is a plain #rrggbb.
    function safeColor(c) { return /^#[0-9a-fA-F]{6}$/.test(asString(c)) ? c : ''; }

    function send(text) {
        if (typeof Client !== 'undefined' && typeof Client.SendInput === 'function') {
            Client.SendInput(text);
        }
    }

    function build() {
        backdrop = el('div');
        backdrop.id = 'creation-backdrop';
        panel = el('div');
        panel.id = 'creation-panel';
        panel.setAttribute('role', 'dialog');
        panel.setAttribute('aria-modal', 'true');
        backdrop.appendChild(panel);
        document.body.appendChild(backdrop);

        tab = el('button', null, 'Reopen the creation panel');
        tab.id = 'creation-tab';
        tab.type = 'button';
        tab.addEventListener('click', function() { hidden = false; render(); });
        document.body.appendChild(tab);

        statusEl = el('div', 'creation-sr-only');
        statusEl.setAttribute('role', 'status');
        statusEl.setAttribute('aria-live', 'polite');
        document.body.appendChild(statusEl);
    }

    function optionButton(opt, index, wide) {
        const li = el('li');
        const b = el('button');
        b.type = 'button';
        const name = el('span', 'creation-option-name');
        const color = safeColor(opt.color);
        if (color) {
            const sw = el('span', 'creation-swatch');
            sw.style.background = color;
            sw.setAttribute('aria-hidden', 'true');
            name.appendChild(sw);
        }
        name.appendChild(el('span', null, asString(opt.name)));
        b.appendChild(name);
        if (opt.text) { b.appendChild(el('span', 'creation-option-text', asString(opt.text))); }
        if (Array.isArray(opt.stats) && opt.stats.length) {
            b.appendChild(el('span', 'creation-option-stats', '+1 ' + opt.stats.map(asString).join(' or ')));
        }
        b.addEventListener('click', function() { send(String(index + 1)); });
        li.appendChild(b);
        return li;
    }

    // figure draws the player's class sprite, in the skin tone and hair
    // colour chosen so far. It is null until the sprite sheet has loaded
    // (Sprites.onChange then redraws the panel) or when there is no art.
    // Since E3 the battle idles are high-density art the palette swap can't
    // repaint, so the panel draws the class's 1x look sheet
    // (battle/units/<class>/look.png) when there is one; a high-density
    // sheet is drawn smoothed at its 1x size.
    function figure(d) {
        if (!window.Sprites || !d.lineage) { return null; }
        const look = window.SpriteTint ? window.SpriteTint.look(d.skin, d.hair) : null;
        const lookPath = 'battle/units/' + d.lineage + '/look.png';
        const path = Sprites.has(lookPath) ? lookPath : 'battle/units/' + d.lineage + '/idle.png';
        const sheet = Sprites.tinted(path, look);
        if (!sheet || !sheet.info || !sheet.info.frame) { return null; }
        const fw = sheet.info.frame[0], fh = sheet.info.frame[1];
        const density = sheet.info.density || 1;
        const scale = 2;
        const cv = el('canvas', 'creation-preview-art');
        cv.width = fw / density * scale;
        cv.height = fh / density * scale;
        cv.setAttribute('role', 'img');
        cv.setAttribute('aria-label', 'Your figure, with the skin tone and hair colour chosen so far');
        const ctx = cv.getContext('2d');
        ctx.imageSmoothingEnabled = density > 1;
        ctx.drawImage(sheet.img, 0, 0, fw, fh, 0, 0, cv.width, cv.height);
        return cv;
    }

    function previewBlock(d) {
        const hasPreview = !!d.preview;
        const hasStory = !!d.backstory;
        if (!hasPreview && !hasStory && !d.lineage) { return null; }
        const box = el('div', 'creation-preview');
        const art = figure(d);
        if (art) { box.appendChild(art); }
        if (hasPreview) {
            box.appendChild(el('h3', null, 'You look like this'));
            box.appendChild(el('p', 'creation-preview-text', d.preview));
        }
        if (hasStory) {
            box.appendChild(el('h3', null, 'Your story'));
            box.appendChild(el('p', 'creation-backstory', d.backstory));
        }
        return box;
    }

    function textEntry(d) {
        const form = el('form', 'creation-line');
        const input = el('input');
        input.type = 'text';
        input.maxLength = 200;
        input.setAttribute('aria-label', 'A line of your own');
        input.autocomplete = 'off';
        const use = el('button', null, 'Use this line');
        use.type = 'submit';
        const none = el('button', null, 'No line');
        none.type = 'button';
        form.appendChild(input);
        form.appendChild(use);
        form.appendChild(none);
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            const v = input.value.trim();
            send(v === '' ? 'none' : v);
        });
        none.addEventListener('click', function() { send('none'); });
        return form;
    }

    function render() {
        if (!panel) { return; }
        const d = state;
        const showing = !!(d && d.active) && !hidden;
        backdrop.classList.toggle('open', showing);
        document.body.classList.toggle('creation-open', showing);
        tab.classList.toggle('open', !!(d && d.active) && hidden);
        if (!d || !d.active) { panel.textContent = ''; return; }

        const heading = el('h2', 'creation-heading', MODE_HEADINGS[d.mode] || 'Create your character');
        heading.id = 'creation-heading';
        panel.setAttribute('aria-labelledby', 'creation-heading');

        const head = el('div', 'creation-head');
        head.appendChild(heading);
        if (d.number > 0 && d.total > 0) {
            head.appendChild(el('span', 'creation-count', 'Question ' + d.number + ' of ' + d.total));
        }

        const body = el('div', 'creation-body');
        const q = el('p', 'creation-question', asString(d.title));
        q.id = 'creation-question';
        body.appendChild(q);

        const options = Array.isArray(d.options) ? d.options : [];
        if (d.kind === 'text') {
            body.appendChild(textEntry(d));
        } else if (options.length) {
            const wide = d.kind === 'confirm' || options.some(o => o && o.text);
            const list = el('ul', 'creation-options' + (wide ? ' wide' : ''));
            list.setAttribute('aria-labelledby', 'creation-question');
            options.forEach((o, i) => { list.appendChild(optionButton(o || {}, i, wide)); });
            body.appendChild(list);
        }
        const pv = previewBlock(d);
        if (pv) { body.appendChild(pv); }

        const foot = el('div', 'creation-foot');
        if (d.canback) {
            const back = el('button', null, 'Back: start this part over');
            back.type = 'button';
            back.addEventListener('click', function() { send('back'); });
            foot.appendChild(back);
        }
        foot.appendChild(el('span', 'creation-spacer'));
        if (d.canskip) {
            // An existing character puts the steps off; appearance edit and
            // lifestory choose are cancelled with nothing changed.
            const legacy = d.mode === 'legacy';
            const skip = el('button', null, legacy ? 'Skip for now' : 'Cancel');
            skip.type = 'button';
            skip.addEventListener('click', function() { send(legacy ? 'skip' : 'cancel'); });
            foot.appendChild(skip);
        }
        const typeInstead = el('button', null, 'Type instead');
        typeInstead.type = 'button';
        typeInstead.addEventListener('click', function() { hidden = true; render(); });
        foot.appendChild(typeInstead);

        // A rebuild (a sprite finishing loading, say) must not drop focus.
        let focusIndex = -1;
        const focused = document.activeElement;
        if (focused && panel.contains(focused)) {
            focusIndex = Array.prototype.indexOf.call(panel.querySelectorAll('button, input'), focused);
        }

        panel.textContent = '';
        panel.appendChild(head);
        panel.appendChild(body);
        panel.appendChild(foot);

        // A new question: announce it and put focus on its first answer.
        const stepKey = [d.step, d.key, d.number, d.title].join('|');
        if (showing && stepKey !== lastStepKey) {
            lastStepKey = stepKey;
            statusEl.textContent = (d.number > 0 ? 'Question ' + d.number + ' of ' + d.total + ': ' : '') + asString(d.title);
            const first = panel.querySelector('.creation-options button, .creation-line input');
            if (first && first.focus) { first.focus({ preventScroll: true }); }
        } else if (focusIndex >= 0) {
            const again = panel.querySelectorAll('button, input')[focusIndex];
            if (again && again.focus) { again.focus({ preventScroll: true }); }
        }
    }

    function onCreation(namespace, payload) {
        if (!payload || typeof payload !== 'object') { return; }
        if (!payload.active) {
            state = null;
            hidden = false;
            lastStepKey = '';
            render();
            if (opener && opener.isConnected && opener.focus) { opener.focus({ preventScroll: true }); }
            opener = null;
            return;
        }
        if (!state) {
            const active = document.activeElement;
            opener = (active && active !== document.body) ? active : null;
            if (opener && opener.blur) { opener.blur(); }
        }
        // A fresh question brings the panel back after "Type instead".
        const stepKey = [payload.step, payload.key, payload.number, payload.title].join('|');
        if (!state || stepKey !== [state.step, state.key, state.number, state.title].join('|')) { hidden = false; }
        state = payload;
        render();
    }

    document.addEventListener('DOMContentLoaded', function() {
        build();
        if (window.Sprites && typeof Sprites.onChange === 'function') {
            Sprites.onChange(function() { if (state && state.active) { render(); } });
        }
        VirtualWindows.register({
            window:       null,
            gmcpHandlers: ['Char.Creation'],
            onGMCP:       onCreation,
        });
        // Ask once the connection is up, in case the first step was sent
        // before GMCP was accepted (a relog mid-creation).
        setTimeout(function() {
            if (typeof Client !== 'undefined' && typeof Client.GMCPRequest === 'function') { Client.GMCPRequest('Char.Creation'); }
        }, 2500);
    });

    window.CreationPanel = { onGMCP: onCreation, render: render };

})();
