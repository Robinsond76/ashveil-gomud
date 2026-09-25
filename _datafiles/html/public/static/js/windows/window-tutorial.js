/**
 * window-tutorial.js
 *
 * Virtual window: Tutorial (Phase 27d).
 * The player's place in the tutorial course: "Stage N of M: Title", the
 * goal, a checklist, and the hints. The same stage, goal, and checklist the
 * terminal "tutorial" command shows.
 *
 * Every string is set with textContent, never innerHTML. Checklist items
 * say "done" or "to do" in words, so the state never rests on the mark or
 * its colour alone.
 *
 * Responds to GMCP namespaces:
 *   Tutorial - the whole view ({} when not in the course)
 *
 * Reads: Client.GMCPStructs.Tutorial, where the client stores each payload
 * even while this window is closed, so a reopened window is current. The
 * server sends it at login and copyover and whenever it changes.
 */

'use strict';

(function() {

    injectStyles(`
        #tutorial-panel {
            height: 100%;
            overflow-y: auto;
            padding: 6px 8px;
            background: var(--t-bg);
            color: var(--t-text);
            font-size: 0.85em;
            line-height: 1.35;
            box-sizing: border-box;
            overflow-wrap: anywhere;
        }

        #tutorial-panel .tutorial-heading {
            margin: 0 0 4px 0;
            font-size: 1em;
            color: var(--t-accent);
        }

        #tutorial-panel .tutorial-progress {
            font-size: 0.85em;
            color: var(--t-text-secondary);
            margin-bottom: 4px;
        }

        #tutorial-panel .tutorial-goal {
            margin: 0 0 6px 0;
        }

        #tutorial-panel .tutorial-goal-label {
            font-weight: bold;
        }

        #tutorial-panel .tutorial-subheading {
            margin: 6px 0 2px 0;
            font-size: 0.9em;
            color: var(--t-text-secondary);
        }

        #tutorial-panel ul {
            margin: 0;
            padding-left: 0;
            list-style: none;
        }

        #tutorial-panel .tutorial-check {
            display: flex;
            gap: 6px;
            align-items: baseline;
        }

        #tutorial-panel .tutorial-check-mark {
            flex: 0 0 auto;
            font-family: monospace;
        }

        #tutorial-panel .tutorial-check.is-done .tutorial-check-label {
            color: var(--t-text-secondary);
        }

        #tutorial-panel .tutorial-check-state {
            flex: 0 0 auto;
            font-size: 0.8em;
            color: var(--t-text-secondary);
        }

        #tutorial-panel .tutorial-hints li {
            margin: 2px 0;
            padding-left: 10px;
            text-indent: -10px;
        }

        #tutorial-panel .tutorial-empty {
            color: var(--t-text-secondary);
            font-style: italic;
        }
    `);

    function el(tag, className, text) {
        const node = document.createElement(tag);
        if (className) { node.className = className; }
        if (text !== undefined && text !== null) { node.textContent = String(text); }
        return node;
    }

    function createDOM() {
        const root = el('div');
        root.id = 'tutorial-panel';
        root.setAttribute('role', 'region');
        root.setAttribute('aria-label', 'Tutorial');
        root.setAttribute('aria-live', 'polite');
        return root;
    }

    function asString(v) { return typeof v === 'string' ? v : ''; }

    function render(panel, data) {
        panel.textContent = '';
        if (!data || data.active !== true) {
            panel.appendChild(el('div', 'tutorial-empty', 'Not in the tutorial.'));
            return;
        }
        const stage  = Number.isFinite(data.stage) ? data.stage : 0;
        const stages = Number.isFinite(data.stages) ? data.stages : 0;

        panel.appendChild(el('h3', 'tutorial-heading', asString(data.title)));
        if (stage && stages) {
            panel.appendChild(el('div', 'tutorial-progress', 'Stage ' + stage + ' of ' + stages));
        }

        const goal = el('p', 'tutorial-goal');
        goal.appendChild(el('span', 'tutorial-goal-label', 'Goal: '));
        goal.appendChild(document.createTextNode(asString(data.goal)));
        panel.appendChild(goal);

        const checks = Array.isArray(data.checklist) ? data.checklist : [];
        if (checks.length) {
            panel.appendChild(el('h4', 'tutorial-subheading', 'Checklist'));
            const list = el('ul', 'tutorial-checklist');
            list.setAttribute('aria-label', 'Checklist');
            checks.forEach(c => {
                if (!c) { return; }
                const done = c.done === true;
                const item = el('li', 'tutorial-check' + (done ? ' is-done' : ''));
                const mark = el('span', 'tutorial-check-mark', done ? '[x]' : '[ ]');
                mark.setAttribute('aria-hidden', 'true');
                item.appendChild(mark);
                item.appendChild(el('span', 'tutorial-check-label', asString(c.label)));
                item.appendChild(el('span', 'tutorial-check-state', done ? '(done)' : '(to do)'));
                list.appendChild(item);
            });
            panel.appendChild(list);
        }

        const hints = Array.isArray(data.hints) ? data.hints : [];
        if (hints.length) {
            panel.appendChild(el('h4', 'tutorial-subheading', 'Hints'));
            const list = el('ul', 'tutorial-hints');
            list.setAttribute('aria-label', 'Hints');
            hints.forEach(h => { list.appendChild(el('li', null, '• ' + asString(h))); });
            panel.appendChild(list);
        }
    }

    function update() {
        if (!win.isOpen()) { return; }
        const panel = document.getElementById('tutorial-panel');
        if (!panel) { return; }
        render(panel, Client.GMCPStructs.Tutorial);
    }

    const win = new VirtualWindow('Tutorial', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  220,
        factory() {
            const root = createDOM();
            // Opened or reopened: show what arrived while it was closed.
            setTimeout(update, 0);
            return {
                title:      'Tutorial',
                mount:      root,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     260,
                header:     20,
                bottom:     60,
            };
        },
    });

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Tutorial'],
        onGMCP() { update(); },
    });

})();
