/**
 * window-event.js
 *
 * Story events (Phase 60): a scene is a page of text with numbered choices,
 * drawn in GameModal. A choice a member's skill decides names the member who
 * would take it; a choice no one qualifies for is shown closed, with what it
 * needs. The server holds the page: closing the screen leaves it waiting, and
 * `event` in the input shows it again.
 *
 * Responds to GMCP namespace:
 *   Event - { active, id, title, picture, text, result: [line],
 *             choices: [{n, label, open, who, needs, risk}] }
 *   active false closes the screen; with `result` lines it first shows how
 *   the scene ended, with a Continue button.
 *
 * Every server string is set with textContent. A choice sends `choose <n>`;
 * the number keys 1 to 6 do the same while the screen is open. A picture is
 * `static/images/events/<picture>.png`; a missing file is simply not shown.
 */

/* global Client, VirtualWindows, GameModal, module */

'use strict';

(function() {

    // noteOf is the small grey note after a choice: who takes it and how
    // risky it is, or what a closed choice needs.
    function noteOf(choice) {
        if (!choice.open) {
            return choice.needs ? 'closed: needs ' + choice.needs : 'closed';
        }
        const parts = [];
        if (choice.who) { parts.push(choice.who); }
        if (choice.risk) { parts.push(choice.risk); }
        return parts.join(', ');
    }

    // viewOf turns a payload into what the screen draws: null when nothing
    // is to be shown.
    function viewOf(payload) {
        if (!payload) { return null; }
        const result = (payload.result || []).filter(Boolean);
        if (!payload.active) {
            if (result.length === 0) { return null; }
            return { title: payload.title || '', picture: '', result: result, paragraphs: [], choices: [], ended: true };
        }
        return {
            title: payload.title || '',
            picture: /^[a-z0-9-]+$/.test(payload.picture || '') ? payload.picture : '',
            result: result,
            paragraphs: String(payload.text || '').split(/\n\s*\n/).map(function(p) { return p.trim(); }).filter(Boolean),
            choices: (payload.choices || []).map(function(c) {
                return { n: c.n, label: c.label, open: !!c.open, note: noteOf(c) };
            }),
            ended: false,
        };
    }

    // choiceForKey is the choice a number key answers, or null.
    function choiceForKey(view, key) {
        if (!view || view.ended || !/^[1-9]$/.test(key)) { return null; }
        const n = Number(key);
        const hit = view.choices.filter(function(c) { return c.n === n && c.open; });
        return hit.length ? hit[0] : null;
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = { viewOf: viewOf, noteOf: noteOf, choiceForKey: choiceForKey };
    }
    if (typeof document === 'undefined') { return; }

    let current = null; // the view on screen, or null
    let keyHandler = null;

    function el(tag, cls, text) {
        const e = document.createElement(tag);
        if (cls) { e.className = cls; }
        if (text !== undefined) { e.textContent = text; }
        return e;
    }

    function style() {
        if (document.getElementById('story-event-style')) { return; }
        const s = document.createElement('style');
        s.id = 'story-event-style';
        s.textContent = [
            '#story-event .ev-picture { display: block; max-width: 100%; max-height: 28vh; margin: 0 auto 10px; border-radius: 4px; }',
            '#story-event .ev-result { color: var(--t-text-secondary); font-style: italic; margin: 0 0 10px; }',
            '#story-event .ev-text { margin: 0 0 10px; }',
            '#story-event .ev-choices { display: flex; flex-direction: column; gap: 6px; margin-top: 12px; }',
            '#story-event .ev-choice { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 10px; text-align: left; min-height: 40px; padding: 6px 10px;',
            '  background: var(--t-bg-surface); color: var(--t-text); border: 1px solid var(--t-border-accent); border-radius: 4px; cursor: pointer; font: inherit; }',
            '#story-event .ev-choice[aria-disabled="true"] { color: var(--t-text-secondary); border-color: var(--t-border); cursor: not-allowed; }',
            '#story-event .ev-n { color: var(--t-accent); min-width: 1.2em; }',
            '#story-event .ev-note { color: var(--t-text-secondary); font-size: 0.85em; }',
            'body.mobile #story-event .ev-choice { min-height: 48px; }',
        ].join('\n');
        document.head.appendChild(s);
    }

    function send(n) { Client.SendInput('choose ' + n); }

    function draw(view) {
        style();
        GameModal.open({ title: view.title || 'A scene', body: '', format: 'html' });
        const host = document.getElementById('game-modal-html-container');
        if (!host) { return; }
        host.textContent = '';
        const root = el('div');
        root.id = 'story-event';
        if (view.picture) {
            const img = el('img', 'ev-picture');
            img.alt = '';
            img.src = 'static/images/events/' + view.picture + '.png';
            img.addEventListener('error', function() { img.remove(); });
            root.appendChild(img);
        }
        view.result.forEach(function(line) { root.appendChild(el('p', 'ev-result', line)); });
        view.paragraphs.forEach(function(p) { root.appendChild(el('p', 'ev-text', p)); });
        const list = el('div', 'ev-choices');
        view.choices.forEach(function(c) {
            const b = el('button', 'ev-choice');
            b.type = 'button';
            b.appendChild(el('span', 'ev-n', c.n + '.'));
            b.appendChild(el('span', 'ev-label', c.label));
            if (c.note) { b.appendChild(el('span', 'ev-note', c.note)); }
            if (c.open) {
                b.addEventListener('click', function() { send(c.n); });
            } else {
                b.setAttribute('aria-disabled', 'true');
            }
            list.appendChild(b);
        });
        if (view.ended) {
            const b = el('button', 'ev-choice');
            b.type = 'button';
            b.appendChild(el('span', 'ev-label', 'Continue'));
            b.addEventListener('click', function() { GameModal.close(); current = null; });
            list.appendChild(b);
        }
        root.appendChild(list);
        host.appendChild(root);
    }

    function show(payload) {
        const view = viewOf(payload);
        if (!view) {
            if (current) { GameModal.close(); }
            current = null;
            return;
        }
        current = view;
        draw(view);
    }

    if (!keyHandler) {
        keyHandler = function(ev) {
            if (!current || ev.ctrlKey || ev.metaKey || ev.altKey) { return; }
            const tag = ev.target && ev.target.tagName;
            if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') { return; }
            const hit = choiceForKey(current, ev.key);
            if (hit) {
                ev.preventDefault();
                send(hit.n);
            }
        };
        document.addEventListener('keydown', keyHandler);
    }

    window.StoryEvent = { show: show, viewOf: viewOf };

    if (typeof VirtualWindows !== 'undefined') {
        document.addEventListener('DOMContentLoaded', function() {
            VirtualWindows.register({
                window: null,
                gmcpHandlers: ['Event'],
                onGMCP: function(namespace, payload) { show(payload); },
            });
        });
    }
})();
