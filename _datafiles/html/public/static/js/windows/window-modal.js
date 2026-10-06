/**
 * window-modal.js
 *
 * Generic scrollable modal overlay.
 *
 * Public API (global):
 *   GameModal.open({ title, body, format })
 *     title  - string shown in the header bar
 *     body   - content string
 *     format - "terminal" (default) renders ANSI/MUD output via xterm.js
 *              "html"     renders raw HTML inside a styled container
 *
 *   GameModal.close()
 *
 * GMCP:
 *   Responds to the "Help" namespace. Payload: { title, body, format }
 *
 * Invokable from any other JS:
 *   GameModal.open({ title: 'Help: cast', body: '...', format: 'terminal' });
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- Backdrop ---- */
        #game-modal-backdrop {
            display: none;
            position: fixed;
            inset: 0;
            z-index: 10000;
            background: rgba(0, 0, 0, 0.72);
            align-items: center;
            justify-content: center;
        }

        #game-modal-backdrop.open {
            display: flex;
        }

        /* While the screen is open nothing behind it takes the pointer, so the
           control that opened it drops its hover highlight at once (Phase 57). */
        body.game-modal-open #main-container,
        body.game-modal-open .vw-window { pointer-events: none; }

        /* ---- Panel ---- */
        #game-modal-panel {
            position: relative;
            display: flex;
            flex-direction: column;
            width: min(780px, 92vw);
            max-height: 82vh;
            background: var(--t-bg-panel);
            border: 1px solid var(--t-border-accent);
            border-radius: 6px;
            box-shadow: 0 8px 40px rgba(0, 0, 0, 0.9);
            overflow: hidden;
        }

        /* ---- Header ---- */
        #game-modal-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 7px 12px 7px 14px;
            background: var(--t-bg-surface);
            border-bottom: 1px solid var(--t-border-accent);
            flex-shrink: 0;
        }

        #game-modal-title {
            font-size: 0.88em;
            color: var(--t-accent);
            text-transform: uppercase;
            letter-spacing: 0.06em;
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        #game-modal-close {
            background: none;
            border: none;
            color: var(--t-text-secondary);
            font-size: 1.1em;
            cursor: pointer;
            padding: 0 2px;
            line-height: 1;
            flex-shrink: 0;
            transition: color 0.15s;
        }

        @media (hover: hover) and (pointer: fine) {
            #game-modal-close:hover { color: var(--t-text); }
        }

        /* ---- Body ---- */
        #game-modal-body {
            flex: 1;
            overflow: hidden;
            display: flex;
            flex-direction: column;
            min-height: 0;
        }

        /* Terminal mode: xterm.js fills the body */
        #game-modal-term-container {
            flex: 1;
            overflow: hidden;
            padding: 8px 6px 6px;
            box-sizing: border-box;
        }

        #game-modal-term-container .xterm-viewport::-webkit-scrollbar       { width: 6px; }
        #game-modal-term-container .xterm-viewport::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        #game-modal-term-container .xterm-viewport::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 3px; }
        #game-modal-term-container .xterm-viewport::-webkit-scrollbar-thumb:hover { background: var(--t-accent); }
        #game-modal-term-container .xterm-viewport { scrollbar-width: thin; scrollbar-color: var(--t-accent-dim) var(--t-scrollbar-track); }

        /* HTML mode: scrollable prose container */
        #game-modal-html-container {
            flex: 1;
            overflow-y: auto;
            padding: 12px 16px;
            color: var(--t-text);
            font-size: 0.84em;
            line-height: 1.6;
        }

        #game-modal-html-container::-webkit-scrollbar       { width: 5px; }
        #game-modal-html-container::-webkit-scrollbar-track  { background: var(--t-bg-panel); }
        #game-modal-html-container::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 3px; }
        #game-modal-html-container::-webkit-scrollbar-thumb:hover { background: var(--t-accent); }

        /* ---- Footer hint ---- */
        #game-modal-footer {
            flex-shrink: 0;
            padding: 4px 14px 5px;
            border-top: 1px solid var(--t-border);
            font-size: 0.66em;
            color: var(--t-text-heading);
            text-align: right;
        }
    `);

    // -----------------------------------------------------------------------
    // DOM - built once on DOMContentLoaded, hidden until opened
    // -----------------------------------------------------------------------
    let backdrop, panel, titleEl, closeBtn, termContainer, htmlContainer;
    let opener        = null;   // what had focus when the screen opened
    let openerByKeys  = false;  // whether that focus came from the keyboard
    let modalTerm     = null;
    let modalFitAddon = null;

    function _buildDOM() {
        backdrop = document.createElement('div');
        backdrop.id = 'game-modal-backdrop';

        panel = document.createElement('div');
        panel.id = 'game-modal-panel';

        const header = document.createElement('div');
        header.id = 'game-modal-header';

        titleEl = document.createElement('span');
        titleEl.id = 'game-modal-title';
        titleEl.textContent = '';

        closeBtn = document.createElement('button');
        closeBtn.id          = 'game-modal-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.setAttribute('aria-label', 'Close');

        header.appendChild(titleEl);
        header.appendChild(closeBtn);

        const body = document.createElement('div');
        body.id = 'game-modal-body';

        termContainer = document.createElement('div');
        termContainer.id = 'game-modal-term-container';

        htmlContainer = document.createElement('div');
        htmlContainer.id = 'game-modal-html-container';

        body.appendChild(termContainer);
        body.appendChild(htmlContainer);

        const footer = document.createElement('div');
        footer.id          = 'game-modal-footer';
        footer.textContent = 'Press Esc or click outside to close';

        panel.appendChild(header);
        panel.appendChild(body);
        panel.appendChild(footer);
        backdrop.appendChild(panel);
        document.body.appendChild(backdrop);

        closeBtn.addEventListener('click', close);

        backdrop.addEventListener('click', function(e) {
            if (e.target === backdrop) { close(); }
        });
    }

    function ensureTerminal() {
        if (modalTerm) { return; }
        var cs = getComputedStyle(document.documentElement);
        modalTerm = new Terminal({
            cols:            80,
            rows:            24,
            cursorBlink:     false,
            disableStdin:    true,
            scrollback:      2000,
            fontSize:        14,
            theme: {
                background:  cs.getPropertyValue('--t-modal-term-bg').trim(),
                foreground:  cs.getPropertyValue('--t-modal-term-fg').trim(),
                cursor:      cs.getPropertyValue('--t-modal-term-cursor').trim(),
                black:       '#1e1e1e',
                red:         '#e06060',
                green:       '#3ad4b8',
                yellow:      '#d4a843',
                blue:        '#7ab8a0',
                magenta:     '#c06090',
                cyan:        '#3ad4b8',
                white:       '#dffbd1',
                brightBlack: '#555',
                brightWhite: '#ffffff',
            },
        });
        modalFitAddon = new FitAddon.FitAddon();
        modalTerm.loadAddon(modalFitAddon);
        modalTerm.open(termContainer);
    }

    // -----------------------------------------------------------------------
    // Open / close
    // -----------------------------------------------------------------------
    function open(opts) {
        opts = opts || {};
        const title   = opts.title  || '';
        const content = opts.body   || '';
        const format  = (opts.format || 'terminal').toLowerCase();

        titleEl.textContent = title;

        // Hand focus to the screen: a control left focused behind it would
        // keep its highlight (Phase 57). Close returns focus to it only when
        // the keyboard opened the screen; returning it after a mouse click
        // would light the control's focus ring when Esc closes the screen,
        // and the ring would stay until the next click (Phase 57 review).
        if (!backdrop.classList.contains('open')) {
            const active = document.activeElement;
            opener = (active && active !== document.body && active.blur) ? active : null;
            openerByKeys = !!opener && _matches(opener, ':focus-visible');
            if (opener) { opener.blur(); }
        }

        if (format === 'html') {
            termContainer.style.display = 'none';
            htmlContainer.style.display = '';
            htmlContainer.innerHTML     = content;
            htmlContainer.scrollTop     = 0;
        } else {
            htmlContainer.style.display = 'none';
            termContainer.style.display = '';
        }

        // Show the backdrop first so the terminal container has layout dimensions
        backdrop.classList.add('open');
        document.body.classList.add('game-modal-open');

        if (format !== 'html') {
            requestAnimationFrame(function() {
                ensureTerminal();
                modalTerm.reset();
                modalFitAddon.fit();

                const lines = content.split('\n');
                lines.forEach(function(line) {
                    modalTerm.writeln(line.replace(/\r$/, ''));
                });

                // Double rAF: let xterm finish its own post-write scroll before
                // we force back to the top.
                requestAnimationFrame(function() {
                    modalTerm.scrollToTop();
                });
            });
        }
    }

    function close() {
        backdrop.classList.remove('open');
        document.body.classList.remove('game-modal-open');
        if (openerByKeys && opener && opener.isConnected && typeof opener.focus === 'function') {
            opener.focus({ preventScroll: true });
        } else if (opener && opener.id === 'command-input') {
            opener.focus({ preventScroll: true });
        } else if (window.matchMedia && window.matchMedia('(hover: hover) and (pointer: fine)').matches) {
            // Opened with the mouse: go back to typing, as a click on the
            // terminal does. Touch screens skip this so no keyboard pops up.
            const input = document.getElementById('command-input');
            if (input) { input.focus({ preventScroll: true }); }
        }
        opener = null;
        openerByKeys = false;
    }

    function _matches(el, selector) {
        try { return el.matches(selector); } catch (e) { return false; }
    }

    // -----------------------------------------------------------------------
    // Public API - exposed before DOMContentLoaded so callers can queue calls
    // -----------------------------------------------------------------------
    window.GameModal = { open: open, close: close };

    // -----------------------------------------------------------------------
    // Init on DOMContentLoaded - document.body is guaranteed to exist here
    // -----------------------------------------------------------------------
    document.addEventListener('DOMContentLoaded', function() {
        _buildDOM();

        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && backdrop.classList.contains('open')) {
                close();
            }
        });

        VirtualWindows.register({
            window:       null,
            gmcpHandlers: ['Help'],
            onGMCP: function(namespace, payload) {
                if (!payload) { return; }
                open({
                    title:  payload.title  || 'Help',
                    body:   payload.body   || '',
                    format: payload.format || 'terminal',
                });
            },
        });
    });

})();
