/**
 * window-comm.js
 *
 * Virtual window: Communications (tabbed chat channels).
 *
 * Phase 32g: the company dock's Comm tab. While another dock tab is
 * showing, a message adds to a count on the Comm tab; showing it clears
 * the count.
 *
 * Responds to GMCP namespace:
 *   Comm  - incoming channel message
 *
 * Reads: Client.GMCPStructs.Comm.Channel
 *
 * Channels are defined in the CHANNELS constant below. Add entries there
 * to expose additional tabs without touching any other file.
 */

'use strict';

(function() {

    injectStyles(`
        #comm-output {
            width: 100%;
            display: flex;
            flex-direction: column;
            height: 100%;
            background: var(--t-bg);
        }

        #comm-output .tab-buttons {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        #comm-output .tab-button {
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

        #comm-output .tab-button:last-child {
            border-right: none;
        }

        @media (hover: hover) and (pointer: fine) {
            #comm-output .tab-button:hover {
                background: var(--t-border);
                color: var(--t-text);
            }
        }

        #comm-output .tab-button.active {
            background: var(--t-bg);
            color: var(--t-text);
            border-bottom: 2px solid var(--t-accent);
        }

        @keyframes comm-tab-glow {
            0%   { background: var(--t-bg-surface); color: var(--t-text-secondary); }
            50%  { background: var(--t-glow-bg); color: var(--t-accent); }
            100% { background: var(--t-bg-surface); color: var(--t-text-secondary); }
        }

        #comm-output .tab-button.pending {
            animation: comm-tab-glow 2s ease-in-out infinite;
        }

        #comm-output .tab-contents {
            flex: 1;
            overflow: hidden;
            background: var(--t-bg);
        }

        #comm-output .tab-content {
            display: none;
            height: 100%;
        }

        #comm-output .tab-content.active {
            display: block;
        }

        .chat-window {
            overflow: scroll;
            background-color: var(--t-bg);
            color: var(--t-text-white);
            font-size: 12px;
            padding: 2px;
        }

        .chat-window p {
            margin-bottom: 2px;
        }
        
        .chat-window.broadcast { color: var(--t-comm-broadcast); }
        .chat-window.whisper   { color: var(--t-comm-whisper); }

        .text-name.mob    { color: var(--t-comm-mob); }
        .text-name.player { color: var(--t-comm-player); }
    `);

    // -----------------------------------------------------------------------
    // Channel configuration
    // Add or remove entries here to change which tabs appear.
    // -----------------------------------------------------------------------
    const MAX_MESSAGES = 300;

    const CHANNELS = [
        { id: 'say',       label: 'Say',        cssClass: 'say',       active: true  },
        { id: 'whisper',   label: 'Whisper',     cssClass: 'whisper',   active: false },
        { id: 'party',     label: 'Party',       cssClass: 'party',     active: false },
        { id: 'broadcast', label: 'Broadcasts',  cssClass: 'broadcast', active: false },
    ];

    // -----------------------------------------------------------------------
    // DOM factory
    // Builds the full tabbed comm UI and appends it to document.body.
    // Returns the root element for WinBox to mount.
    // -----------------------------------------------------------------------
    function createDOM() {
        const root = document.createElement('div');
        root.id        = 'comm-output';
        root.style.height = '100%';

        // Tab button row
        const buttonRow = document.createElement('div');
        buttonRow.className = 'tab-buttons';

        // Tab panel container
        const panelContainer = document.createElement('div');
        panelContainer.className = 'tab-contents';

        CHANNELS.forEach(ch => {
            // Button
            const btn = document.createElement('button');
            btn.id               = 'comm-tab-' + ch.id;
            btn.className        = 'tab-button' + (ch.active ? ' active' : '');
            btn.dataset.tab      = 'comm-' + ch.id;
            btn.dataset.label    = ch.label;
            btn.dataset.unread   = '0';
            btn.textContent      = ch.label;
            buttonRow.appendChild(btn);

            // Panel
            const panel = document.createElement('div');
            panel.id        = 'comm-' + ch.id;
            panel.className = 'chat-window ' + ch.cssClass + ' tab-content' + (ch.active ? ' active' : '');
            panelContainer.appendChild(panel);
        });

        root.appendChild(buttonRow);
        root.appendChild(panelContainer);
        document.body.appendChild(root);

        // Wire up tab switching within this window's root element
        const buttons = buttonRow.querySelectorAll('.tab-button');
        const panels  = panelContainer.querySelectorAll('.tab-content');

        buttons.forEach(btn => {
            btn.addEventListener('click', () => {
                const target = btn.dataset.tab;

                buttons.forEach(b => b.classList.remove('active'));
                panels.forEach(p => p.classList.remove('active'));

                btn.classList.add('active');
                btn.classList.remove('pending');
                btn.dataset.unread = '0';
                btn.textContent    = btn.dataset.label;
                document.getElementById(target).classList.add('active');
            });
        });

        return root;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    // unseen counts messages that arrived while the Comm tab wasn't
    // showing (Phase 32g).
    let unseen = 0;

    const win = new VirtualWindow('Communications', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  290,
        tabGroup:      'dock',
        tabLabel:      'Comm',
        onTabShown() {
            unseen = 0;
            VirtualWindows.setTabBadge('Communications', 0);
        },
        factory() {
            const el = createDOM();
            return {
                title:      'Communications',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          'right',
                y:          450,
                width:      363,
                height:     20 + 290,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Message rendering
    // -----------------------------------------------------------------------
    function postMessage(channelName, fromName, fromSource, message) {
        const tab   = document.getElementById('comm-tab-' + channelName);
        const panel = document.getElementById('comm-' + channelName);

        if (!tab || !panel) {
            return;
        }

        // Update unread badge on inactive tabs
        if (tab.classList.contains('active')) {
            tab.dataset.unread = '0';
            tab.textContent    = tab.dataset.label;
        } else {
            tab.dataset.unread = String(parseInt(tab.dataset.unread) + 1);
            tab.textContent    = tab.dataset.label + '(' + tab.dataset.unread + ')';
            tab.classList.add('pending');
        }

        const p = document.createElement('p');
        p.innerHTML =
            '<span class="text-name ' + fromSource + '">' + fromName + '</span>: ' +
            '<span class="text-body ' + fromSource + '">' + message + '</span>';
        panel.appendChild(p);

        if (!VirtualWindows.isTabShowing('Communications')) {
            unseen++;
            VirtualWindows.setTabBadge('Communications', unseen);
        }

        // Docked, keep the newest messages only (Phase 32g).
        while (panel.childElementCount > MAX_MESSAGES) {
            panel.removeChild(panel.firstElementChild);
        }

        // Trim overflow: remove oldest messages when content exceeds window height
        const winBox = win.get();
        if (winBox) {
            const winContainer = winBox.window;
            while (panel.scrollHeight > winContainer.clientHeight - 58) {
                if (panel.childElementCount < 1) {
                    break;
                }
                panel.removeChild(panel.firstElementChild);
            }
        }
    }

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function updateComm() {
        const obj = Client.GMCPStructs.Comm;
        if (!obj || !obj.Channel) {
            return;
        }

        win.open();
        if (!win.isOpen()) {
            return;
        }

        const ch = obj.Channel;
        postMessage(ch.channel, ch.sender, ch.source, ch.text);
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Comm'],
        onGMCP(namespace, body) {
            updateComm();
        },
    });

})();
