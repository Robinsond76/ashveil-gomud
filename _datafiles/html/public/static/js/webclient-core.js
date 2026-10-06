/* global Triggers */

/**
 * webclient-core.js
 *
 * Core infrastructure for the GoMud web client. Provides:
 *   - Client namespace (shared state accessible by window modules)
 *   - VirtualWindow class (lifecycle management for VWin panels)
 *   - VirtualWindows registry (GMCP handler dispatch)
 *   - WebSocket connection management
 *   - Terminal (xterm.js) setup
 *   - MSP audio (music + sound)
 *   - Volume slider UI
 *
 * Window modules call VirtualWindows.register(...) to add themselves.
 * The HTML file calls Client.init() on page load.
 */

'use strict';

// ---------------------------------------------------------------------------
// injectStyles
//
// Appends a <style> block to <head>. Called by window modules at load time
// so each module owns and ships its own CSS alongside its JS.
// ---------------------------------------------------------------------------
function injectStyles(css) {
    const style = document.createElement('style');
    style.textContent = css;
    document.head.appendChild(style);
}

// ---------------------------------------------------------------------------
// uiMenu
//
// Spawns a small context menu anchored near a click event.
// Dismisses on any outside click or when a command is chosen.
//
// Usage:
//   uiMenu(event, [
//       { label: 'look item',   cmd: 'look longsword'   },
//       { label: 'remove item', cmd: 'remove longsword' },
//   ]);
//
// An item with fn: function runs it instead of sending a command (Phase 48).
// An item with confirm: '<question>' asks first and sends nothing unless
// the player agrees (Phase 32g: for what can't be undone).
// ---------------------------------------------------------------------------
(function() {
    let menuEl   = null;
    let offClick = null;
    let opener   = null;

    // dismiss closes the menu; refocus returns focus to what opened it
    // (Escape), so a keyboard user isn't left nowhere.
    function dismiss(refocus) {
        if (menuEl) {
            menuEl.remove();
            menuEl = null;
        }
        if (offClick) {
            document.removeEventListener('mousedown', offClick, true);
            offClick = null;
        }
        if (refocus && opener && typeof opener.focus === 'function') { opener.focus(); }
        opener = null;
    }

    // Phase 32g: entries are buttons in a role="menu" list, so the keyboard
    // reaches them: the first is focused on open, arrows, Home, and End
    // move, Enter or Space chooses, Escape closes.
    function ensureMenuStyle() {
        if (document.getElementById('ui-menu-style')) { return; }
        const style = document.createElement('style');
        style.id = 'ui-menu-style';
        style.textContent =
            '.ui-menu-item:focus { outline: none; background: var(--t-bg-hover) !important; color: var(--t-text) !important; box-shadow: inset 2px 0 0 var(--t-accent); }' +
            '.ui-menu-item:focus-visible { outline: 1px solid var(--t-accent); outline-offset: -1px; }';
        document.head.appendChild(style);
    }

    window.uiMenu = function uiMenu(event, items) {
        ensureMenuStyle();
        dismiss(false);
        opener = (event && event.currentTarget instanceof Element) ? event.currentTarget
               : (event && event.target instanceof Element ? event.target : null);

        menuEl = document.createElement('div');
        menuEl.setAttribute('role', 'menu');
        menuEl.className = 'ui-menu';
        menuEl.style.cssText = [
            'position:fixed',
            'z-index:2147483647',
            'background:var(--t-bg-surface)',
            'border:1px solid var(--t-btn-border)',
            'border-radius:4px',
            'box-shadow:0 4px 14px rgba(0,0,0,0.7)',
            'padding:3px 0',
            'min-width:120px',
            'font-family:inherit',
            'font-size:0.75em',
            'display:flex',
            'flex-direction:column',
            'max-height:70vh',
            'overflow-y:auto',
        ].join(';');

        const entries = [];
        items.forEach(function(item) {
            const entry = document.createElement('button');
            entry.type = 'button';
            entry.setAttribute('role', 'menuitem');
            entry.tabIndex = -1;
            entry.className = 'ui-menu-item';
            entry.textContent = item.label;
            entry.style.cssText = [
                'padding:5px 12px',
                'color:var(--t-text)',
                'cursor:pointer',
                'white-space:nowrap',
                'letter-spacing:0.03em',
                'background:none',
                'border:none',
                'text-align:left',
                'font:inherit',
            ].join(';');
            // One highlighted entry at a time (Phase 57): the pointer moves
            // focus to the entry under it, and the highlight is the focus
            // style in the stylesheet below, so there is no inline state to
            // leave behind when a menu closes or the pointer leaves.
            entry.addEventListener('mouseenter', function() { entry.focus({ preventScroll: true }); });
            entry.addEventListener('click', function(e) {
                e.stopPropagation();
                dismiss(false);
                if (item.confirm && !window.confirm(item.confirm)) { return; }
                if (typeof item.fn === 'function') { item.fn(e); return; }
                Client.SendInput(item.cmd);
            });
            entries.push(entry);
            menuEl.appendChild(entry);
        });

        menuEl.addEventListener('keydown', function(e) {
            const at = entries.indexOf(document.activeElement);
            let next = -1;
            if (e.key === 'ArrowDown') { next = (at + 1) % entries.length; }
            if (e.key === 'ArrowUp')   { next = (at - 1 + entries.length) % entries.length; }
            if (e.key === 'Home')      { next = 0; }
            if (e.key === 'End')       { next = entries.length - 1; }
            if (e.key === 'Escape' || e.key === 'Tab') {
                e.preventDefault();
                dismiss(true);
                return;
            }
            if (next !== -1) {
                e.preventDefault();
                entries[next].focus();
            }
        });

        // Position: prefer below-right of the click, flip if it would
        // overflow. A menu opened from the keyboard (no pointer position)
        // opens by the control that opened it.
        const vw = window.innerWidth;
        const vh = window.innerHeight;
        menuEl.style.left = '-9999px';
        menuEl.style.top  = '-9999px';
        document.body.appendChild(menuEl);

        let cx = event ? event.clientX : 0;
        let cy = event ? event.clientY : 0;
        if (!cx && !cy && opener) {
            const r = opener.getBoundingClientRect();
            cx = r.left;
            cy = r.bottom;
        }
        const mw = menuEl.offsetWidth;
        const mh = menuEl.offsetHeight;
        let x = cx;
        let y = cy + 4;
        if (x + mw > vw - 8) { x = vw - mw - 8; }
        if (y + mh > vh - 8) { y = cy - mh - 4; }
        menuEl.style.left = Math.max(8, x) + 'px';
        menuEl.style.top  = Math.max(8, y) + 'px';

        offClick = function(e) {
            if (menuEl && !menuEl.contains(e.target)) { dismiss(false); }
        };
        document.addEventListener('mousedown', offClick, true);
        if (entries.length) { entries[0].focus(); }
    };
}());

// ---------------------------------------------------------------------------
// DockSlot
//
// Manages one side's dock column (#dock-left or #dock-right).
// Handles:
//   - adding / removing panels
//   - showing / hiding the slot (zero-width when empty)
//   - the slot-width drag handle
//   - the per-panel vertical resize handles
// ---------------------------------------------------------------------------
class DockSlot {
    constructor(side) {
        this.side    = side;
        this.el      = document.getElementById('dock-' + side);
        this._panels = [];
        if (!this.el) {
            console.error('DockSlot: #dock-' + side + ' not found. Check that webclient-pure.html contains <div id="dock-' + side + '"> inside #main-container.');
            return;
        }
        this._initSlotResize();
    }

    // Add a content element as a new panel with the given title.
    // height (optional) sets the preferred panel height in px.
    // onClose (optional) called when the panel's X button is clicked.
    // onMoveTo (optional) called with (newSide, dropIdx) when dragged to the opposite slot.
    // insertAt (optional) index at which to insert; appends if omitted or out of range.
    // Returns the panel wrapper element.
    addPanel(contentEl, title, onPopout, height, onClose, onMoveTo, insertAt) {
        if (!this.el) { return null; }
        const panel = document.createElement('div');
        panel.className = 'dock-panel';

        // Apply preferred height as a fixed flex-basis so the panel does not
        // grow to fill the slot. The user can still drag the resize handle to
        // redistribute space between panels.
        if (height) {
            panel.style.flex      = '0 0 ' + height + 'px';
            panel.style.flexBasis = height + 'px';
        }

        const titlebar = document.createElement('div');
        titlebar.className = 'dock-panel-titlebar';

        const titleSpan = document.createElement('span');
        titleSpan.className   = 'dock-panel-title';
        titleSpan.textContent = title;

        const popoutBtn = document.createElement('span');
        popoutBtn.className   = 'dock-panel-popout';
        popoutBtn.title       = 'Pop out';
        popoutBtn.textContent = '⧉';
        popoutBtn.addEventListener('click', onPopout);

        titlebar.appendChild(titleSpan);
        titlebar.appendChild(popoutBtn);

        const content = document.createElement('div');
        content.className = 'dock-panel-content';
        content.appendChild(contentEl);

        panel.appendChild(titlebar);
        panel.appendChild(content);

        // Wire up drag-to-reorder on the titlebar, with cross-slot transfer support
        this._initPanelDrag(titlebar, panel, (newSide, dropIdx) => {
            if (typeof onMoveTo === 'function') { onMoveTo(newSide, dropIdx); }
        });

        // Insert a vertical resize handle and the panel at the correct position.
        // If insertAt is a valid index within the current panels, insert before
        // that panel; otherwise append at the end.
        const useInsert = (typeof insertAt === 'number' && insertAt >= 0 && insertAt < this._panels.length);
        let resizeHandle = null;

        if (useInsert) {
            const refEntry = this._panels[insertAt];
            // A resize handle goes between panels, so insert one before the new panel
            // (which sits before refEntry).
            resizeHandle = document.createElement('div');
            resizeHandle.className = 'dock-panel-resize';
            this.el.insertBefore(resizeHandle, refEntry.panel);
            this._initPanelResize(resizeHandle);
            this.el.insertBefore(panel, refEntry.panel);
            this._panels.splice(insertAt, 0, { panel, contentEl, resizeHandle });
        } else {
            // Append at the end - only add a resize handle if there are existing panels.
            if (this._panels.length > 0) {
                resizeHandle = document.createElement('div');
                resizeHandle.className = 'dock-panel-resize';
                this.el.appendChild(resizeHandle);
                this._initPanelResize(resizeHandle);
            }
            this.el.appendChild(panel);
            this._panels.push({ panel, contentEl, resizeHandle });
        }
        this._setActive(true);
        return panel;
    }

    // Remove a panel by its content element. Returns the content element.
    removePanel(contentEl) {
        if (!this.el) { return contentEl; }
        const idx = this._panels.findIndex(p => p.contentEl === contentEl);
        if (idx === -1) { return contentEl; }

        const { panel, resizeHandle } = this._panels[idx];

        // Remove the resize handle that was inserted before this panel,
        // or the one after it if this is the first panel.
        if (resizeHandle) {
            resizeHandle.remove();
        } else if (this._panels.length > 1) {
            // This was the first panel; remove the handle that was after it
            const next = this._panels[1];
            if (next.resizeHandle) {
                next.resizeHandle.remove();
                next.resizeHandle = null;
            }
        }

        // Move content back out before removing the panel
        document.body.appendChild(contentEl);
        panel.remove();
        this._panels.splice(idx, 1);

        if (this._panels.length === 0) {
            this._setActive(false);
        }
        return contentEl;
    }

    hasPanel(contentEl) {
        return this._panels.some(p => p.contentEl === contentEl);
    }

    _setActive(active) {
        if (active) {
            this.el.classList.add('has-panels');
        } else {
            this.el.classList.remove('has-panels');
        }
        // Defer until after the browser has completed its layout pass so
        // fitAddon.fit() measures the terminal at its new settled dimensions.
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    }

    // Slot-width drag handle - inserted as a sibling of the slot in
    // #main-container so it is never clipped by the slot's overflow:hidden.
    // Hidden when the slot is empty, shown when it has panels.
    _initSlotResize() {
        if (!this.el) { return; }
        const handle = document.createElement('div');
        handle.className = 'dock-slot-resize dock-slot-resize-' + this.side;
        // Insert adjacent to the slot inside #main-container
        const container = this.el.parentNode;
        if (this.side === 'right') {
            container.insertBefore(handle, this.el);
        } else {
            this.el.insertAdjacentElement('afterend', handle);
        }

        // Keep visibility in sync with the slot's active state
        const observer = new MutationObserver(() => {
            handle.style.display = this.el.classList.contains('has-panels') ? '' : 'none';
        });
        observer.observe(this.el, { attributes: true, attributeFilter: ['class'] });
        handle.style.display = 'none';  // hidden until first panel is added

        let startX, startWidth, _rafPending = false;
        const onMove = (e) => {
            const dx    = (e.clientX || e.touches[0].clientX) - startX;
            const width = Math.max(80, startWidth + (this.side === 'right' ? -dx : dx));
            this.el.style.setProperty('--dock-' + this.side + '-width', width + 'px');
            this.el.style.width = width + 'px';
            if (!_rafPending) {
                _rafPending = true;
                requestAnimationFrame(() => {
                    _rafPending = false;
                    window.dispatchEvent(new Event('resize'));
                });
            }
        };
        const onUp = () => {
            handle.classList.remove('dragging');
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup',   onUp);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend',  onUp);
            LayoutStore.saveDockWidths();
        };
        handle.addEventListener('mousedown', (e) => {
            e.preventDefault();
            handle.classList.add('dragging');
            startX     = e.clientX;
            startWidth = this.el.offsetWidth;
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
        handle.addEventListener('touchstart', (e) => {
            startX     = e.touches[0].clientX;
            startWidth = this.el.offsetWidth;
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend',  onUp);
        }, { passive: true });
    }

    // Vertical resize handle between two stacked panels
    _initPanelResize(handle) {
        let startY, prevHeight, nextHeight, prevPanel, nextPanel;

        const onMove = (e) => {
            const dy   = (e.clientY || e.touches[0].clientY) - startY;
            const newPrev = Math.max(40, prevHeight + dy);
            const newNext = Math.max(40, nextHeight - dy);
            prevPanel.style.flexBasis = newPrev + 'px';
            prevPanel.style.flex      = '0 0 ' + newPrev + 'px';
            nextPanel.style.flexBasis = newNext + 'px';
            nextPanel.style.flex      = '0 0 ' + newNext + 'px';
        };
        const onUp = () => {
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup',   onUp);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend',  onUp);
            // Save the docked height for both panels that were resized
            [prevPanel, nextPanel].forEach(panelEl => {
                if (!panelEl) { return; }
                const entry = this._panels.find(p => p.panel === panelEl);
                if (!entry) { return; }
                const win = VirtualWindows.getWindows().find(w => w._contentEl === entry.contentEl);
                if (win) { LayoutStore.saveWindow(win); }
            });
        };
        handle.addEventListener('mousedown', (e) => {
            e.preventDefault();
            startY      = e.clientY;
            prevPanel   = handle.previousElementSibling;
            nextPanel   = handle.nextElementSibling;
            prevHeight  = prevPanel.offsetHeight;
            nextHeight  = nextPanel.offsetHeight;
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
        handle.addEventListener('touchstart', (e) => {
            startY      = e.touches[0].clientY;
            prevPanel   = handle.previousElementSibling;
            nextPanel   = handle.nextElementSibling;
            prevHeight  = prevPanel.offsetHeight;
            nextHeight  = nextPanel.offsetHeight;
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend',  onUp);
        }, { passive: true });
    }

    // Drag-to-reorder on a panel's titlebar.
    // Shows a ghost label following the cursor and a drop indicator line
    // between panels. On drop, reorders the panel in the DOM and _panels array,
    // or calls onMoveTo(newSide, dropIdx) if dropped into the opposite slot.
    _initPanelDrag(titlebar, panel, onMoveTo) {
        titlebar.addEventListener('mousedown', (e) => {
            // Ignore clicks on the action buttons
            if (e.target.classList.contains('dock-panel-popout') ||
                e.target.classList.contains('dock-panel-close')) {
                return;
            }
            e.preventDefault();

            const srcIdx = this._panels.findIndex(p => p.panel === panel);
            if (srcIdx === -1) { return; }

            const oppSide = this.side === 'left' ? 'right' : 'left';
            const oppSlot = DockSlots[oppSide];

            // Ghost label that follows the cursor
            const ghost = document.createElement('div');
            ghost.className = 'dock-drag-ghost';
            ghost.textContent = titlebar.querySelector('.dock-panel-title').textContent;
            document.body.appendChild(ghost);

            // Two drop indicators - one per slot
            const ownIndicator = document.createElement('div');
            ownIndicator.className = 'dock-drop-indicator';
            ownIndicator.style.display = 'none';
            this.el.appendChild(ownIndicator);

            let oppIndicator = null;
            if (oppSlot && oppSlot.el) {
                oppIndicator = document.createElement('div');
                oppIndicator.className = 'dock-drop-indicator';
                oppIndicator.style.display = 'none';
                oppSlot.el.appendChild(oppIndicator);
            }

            panel.classList.add('dock-dragging');

            let dropSide = this.side;
            let dropIdx  = srcIdx;

            const _calcDropIdx = (slot, clientY) => {
                const panels = slot._panels;
                let idx = panels.length;
                for (let i = 0; i < panels.length; i++) {
                    if (panels[i].panel === panel) { continue; }
                    const r = panels[i].panel.getBoundingClientRect();
                    if (clientY < r.top + r.height / 2) { idx = i; break; }
                }
                return idx;
            };

            const _showIndicator = (indicator, slot, idx) => {
                if (!indicator || !slot.el) { return; }
                const panels    = slot._panels;
                const slotRect  = slot.el.getBoundingClientRect();
                indicator.style.display = 'block';
                if (panels.length === 0 || idx >= panels.length) {
                    const last = panels.length > 0 ? panels[panels.length - 1].panel : null;
                    indicator.style.top = last
                        ? (last.getBoundingClientRect().bottom - slotRect.top + 2) + 'px'
                        : '4px';
                } else {
                    const r = panels[idx].panel.getBoundingClientRect();
                    indicator.style.top = (r.top - slotRect.top - 2) + 'px';
                }
            };

            const onMove = (e) => {
                ghost.style.top = e.clientY + 'px';

                const ownRect = this.el.getBoundingClientRect();
                const oppRect = oppSlot && oppSlot.el ? oppSlot.el.getBoundingClientRect() : null;

                // Determine which slot the cursor is over
                const overOpp = oppRect &&
                    e.clientX >= oppRect.left && e.clientX <= oppRect.right &&
                    oppRect.width > 0;

                if (overOpp) {
                    dropSide = oppSide;
                    dropIdx  = _calcDropIdx(oppSlot, e.clientY);
                    ghost.style.left  = oppRect.left + 'px';
                    ghost.style.width = oppRect.width + 'px';
                    ownIndicator.style.display = 'none';
                    _showIndicator(oppIndicator, oppSlot, dropIdx);
                } else {
                    dropSide = this.side;
                    dropIdx  = _calcDropIdx(this, e.clientY);
                    ghost.style.left  = ownRect.left + 'px';
                    ghost.style.width = ownRect.width + 'px';
                    if (oppIndicator) { oppIndicator.style.display = 'none'; }
                    // Hide own indicator when drop would not change order
                    if (dropIdx === srcIdx || dropIdx === srcIdx + 1) {
                        ownIndicator.style.display = 'none';
                    } else {
                        _showIndicator(ownIndicator, this, dropIdx);
                    }
                }
            };

            // Set initial ghost position
            const initRect = this.el.getBoundingClientRect();
            ghost.style.left  = initRect.left + 'px';
            ghost.style.width = initRect.width + 'px';
            ghost.style.top   = e.clientY + 'px';

            const onUp = () => {
                document.removeEventListener('mousemove', onMove);
                document.removeEventListener('mouseup',   onUp);

                ghost.remove();
                ownIndicator.remove();
                if (oppIndicator) { oppIndicator.remove(); }
                panel.classList.remove('dock-dragging');

                if (dropSide !== this.side) {
                    // Dropped into the opposite slot
                    if (typeof onMoveTo === 'function') {
                        onMoveTo(dropSide, dropIdx);
                    }
                } else if (dropIdx !== srcIdx && dropIdx !== srcIdx + 1) {
                    // Reorder within own slot
                    this._movePanel(srcIdx, dropIdx);
                }
            };

            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
    }

    // Reorder a panel from fromIdx to toIdx (insert-before semantics).
    // Rebuilds the DOM order and the resize handles between panels.
    _movePanel(fromIdx, toIdx) {
        if (fromIdx === toIdx) { return; }

        // Remove all resize handles from the DOM - we'll rebuild them
        this._panels.forEach(p => {
            if (p.resizeHandle) {
                p.resizeHandle.remove();
                p.resizeHandle = null;
            }
        });

        // Reorder the _panels array
        const moved = this._panels.splice(fromIdx, 1)[0];
        const insertAt = toIdx > fromIdx ? toIdx - 1 : toIdx;
        this._panels.splice(insertAt, 0, moved);

        // Re-append panels to the slot in the new order
        this._panels.forEach(p => this.el.appendChild(p.panel));

        // Rebuild resize handles between adjacent panels
        for (let i = 1; i < this._panels.length; i++) {
            const handle = document.createElement('div');
            handle.className = 'dock-panel-resize';
            // Insert before the panel at index i
            this.el.insertBefore(handle, this._panels[i].panel);
            this._panels[i].resizeHandle = handle;
            this._initPanelResize(handle);
        }

        // Notify the registry so the canonical order is updated
        VirtualWindows.notifyReorder(this.side, this._panels.map(p => p.contentEl));
    }
}

// Singleton slot instances, populated by Client.init() once the DOM is ready.
const DockSlots = {};

// ---------------------------------------------------------------------------
// DockTabGroup (Phase 32g)
//
// One dock panel holding several windows as tabs. A VirtualWindow with
// tabGroup: '<name>' joins its group's panel when docked instead of getting
// a panel of its own; with groupHeader: true it is shown above the tabs
// (the vitals strip) instead of as a tab. Each window keeps its own content
// element and GMCP handling; the group only shows one tab's content at a
// time. The panel's pop-out button floats the active tab, and a floating
// member's dock button returns it to its tab. The group's panel fills the
// column; dragging it to the other column moves every member.
//
// Tab order is WINDOW_DOCK_DEFAULTS order. The active tab is remembered in
// LayoutStore. A member can show a count on its tab (setBadge) and be told
// when its tab is shown (the window's onTabShown option).
// ---------------------------------------------------------------------------
class DockTabGroup {
    constructor(name) {
        this.name     = name;
        this.side     = null;
        this.root     = null;   // the content element handed to the DockSlot
        this._header  = null;
        this._tablist = null;
        this._panes   = null;
        this._members = [];     // { win, tab, pane, badge }
        this._active  = null;   // window id
    }

    _order(win) {
        const i = WINDOW_DOCK_DEFAULTS.findIndex(d => d.id === win._id);
        return i === -1 ? WINDOW_DOCK_DEFAULTS.length : i;
    }

    // The window whose id stands for the group in the dock order: its
    // first member in WINDOW_DOCK_DEFAULTS.
    anchorId() {
        const first = WINDOW_DOCK_DEFAULTS.find(d => d.group === this.name);
        return first ? first.id : this.name;
    }

    _build() {
        const root = document.createElement('div');
        root.className = 'dock-tabgroup';
        root.dataset.group = this.name;
        this._header = document.createElement('div');
        this._header.className = 'dock-tabgroup-header';
        this._tablist = document.createElement('div');
        this._tablist.className = 'dock-tabgroup-tabs';
        this._tablist.setAttribute('role', 'tablist');
        this._tablist.addEventListener('keydown', (e) => {
            if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight' && e.key !== 'Home' && e.key !== 'End') { return; }
            const tabs = this._members.filter(m => m.tab).map(m => m.tab);
            const at = tabs.indexOf(document.activeElement);
            if (at === -1) { return; }
            e.preventDefault();
            let next = at;
            if (e.key === 'ArrowLeft')  { next = (at - 1 + tabs.length) % tabs.length; }
            if (e.key === 'ArrowRight') { next = (at + 1) % tabs.length; }
            if (e.key === 'Home')       { next = 0; }
            if (e.key === 'End')        { next = tabs.length - 1; }
            tabs[next].focus();
            tabs[next].click();
        });
        this._panes = document.createElement('div');
        this._panes.className = 'dock-tabgroup-panes';
        root.appendChild(this._header);
        root.appendChild(this._tablist);
        root.appendChild(this._panes);
        this.root = root;
    }

    _title() {
        const m = this._members.find(x => x.win._id === this._active);
        return m ? m.label : this.name;
    }

    _ensurePanel(side) {
        if (!this.root) { this._build(); }
        const slot = DockSlots[side];
        if (!slot) { return; }
        if (this.side === side && slot.hasPanel(this.root)) { return; }
        this.side = side;
        const panel = slot.addPanel(
            this.root,
            this._title(),
            () => this.popOutActive(),
            null,
            null,
            (newSide) => this.moveTo(newSide),
            VirtualWindows.getDockInsertIndexFor(this.anchorId(), side)
        );
        if (panel) { panel.classList.add('dock-panel-fill'); panel.dataset.win = 'group:' + this.name; }
    }

    _syncTitle() {
        const slot = DockSlots[this.side];
        if (!slot || !this.root) { return; }
        const entry = slot._panels.find(p => p.contentEl === this.root);
        const title = entry && entry.panel.querySelector('.dock-panel-title');
        if (title) { title.textContent = this._title(); }
    }

    has(win) {
        return this._members.some(m => m.win === win);
    }

    add(win, side) {
        this._ensurePanel(side);
        if (this.has(win)) { return; }
        const label = win._tabLabel || (win._vwinOpts && win._vwinOpts.title) || win._id;
        const member = { win, label, tab: null, pane: null, badge: null };
        if (win._groupHeader) {
            this._header.appendChild(win._contentEl);
        } else {
            const key = win._id.replace(/[^A-Za-z0-9]/g, '');
            const tab = document.createElement('button');
            tab.type = 'button';
            tab.className = 'dock-tabgroup-tab';
            tab.id = 'dock-tab-' + key;
            tab.setAttribute('role', 'tab');
            tab.setAttribute('aria-controls', 'dock-pane-' + key);
            tab.dataset.win = win._id;
            const text = document.createElement('span');
            text.textContent = label;
            const badge = document.createElement('span');
            badge.className = 'dock-tabgroup-badge';
            badge.hidden = true;
            tab.appendChild(text);
            tab.appendChild(badge);
            tab.addEventListener('click', () => this.activate(win._id));
            const pane = document.createElement('div');
            pane.className = 'dock-tabgroup-pane';
            pane.id = 'dock-pane-' + key;
            pane.setAttribute('role', 'tabpanel');
            pane.setAttribute('aria-labelledby', tab.id);
            pane.appendChild(win._contentEl);
            member.tab = tab;
            member.pane = pane;
            member.badge = badge;
        }
        // Keep WINDOW_DOCK_DEFAULTS order.
        const order = this._order(win);
        let at = this._members.findIndex(m => this._order(m.win) > order);
        if (at === -1) { at = this._members.length; }
        this._members.splice(at, 0, member);
        if (member.tab) {
            const after = this._members.slice(at + 1).find(m => m.tab);
            this._tablist.insertBefore(member.tab, after ? after.tab : null);
            this._panes.appendChild(member.pane);
        }
        const saved = LayoutStore.getActiveTab(this.name);
        const current = this._members.find(m => m.win._id === this._active && m.tab);
        if (!current || (saved === win._id && member.tab)) {
            this.activate(saved && this._members.some(m => m.win._id === saved && m.tab) ? saved : this._firstTabId(), true);
        } else {
            this._showActive();
        }
    }

    _firstTabId() {
        const m = this._members.find(x => x.tab);
        return m ? m.win._id : null;
    }

    remove(win) {
        const i = this._members.findIndex(m => m.win === win);
        if (i === -1) { return; }
        const member = this._members[i];
        this._members.splice(i, 1);
        if (member.tab) { member.tab.remove(); }
        if (member.pane) { member.pane.remove(); }
        if (win._contentEl && win._contentEl.parentNode) {
            document.body.appendChild(win._contentEl);
        }
        if (this._members.length === 0) {
            const slot = DockSlots[this.side];
            if (slot) { slot.removePanel(this.root); }
            if (this.root.parentNode) { this.root.parentNode.removeChild(this.root); }
            this.side = null;
            return;
        }
        if (this._active === win._id) {
            this.activate(this._firstTabId(), true);
        }
    }

    // Show one tab. quiet: don't remember it as the player's choice.
    activate(id, quiet) {
        if (!id) { this._active = null; this._showActive(); return; }
        this._active = id;
        if (!quiet) { LayoutStore.saveActiveTab(this.name, id); }
        this._showActive();
        const m = this._members.find(x => x.win._id === id);
        if (m && typeof m.win._onTabShown === 'function') { m.win._onTabShown(); }
    }

    _showActive() {
        this._members.forEach(m => {
            if (!m.tab) { return; }
            const on = m.win._id === this._active;
            m.tab.classList.toggle('active', on);
            m.tab.setAttribute('aria-selected', on ? 'true' : 'false');
            m.tab.tabIndex = on ? 0 : -1;
            m.pane.hidden = !on;
        });
        this._syncTitle();
    }

    isActive(id) {
        return this._active === id;
    }

    // spoken (32g2) replaces "<text> new" in the tab's accessible name.
    setBadge(id, text, spoken) {
        const m = this._members.find(x => x.win._id === id);
        if (!m || !m.badge) { return; }
        m.badge.textContent = text ? String(text) : '';
        m.badge.hidden = !text;
        if (text) {
            m.tab.setAttribute('aria-label', m.label + ', ' + (spoken || text + ' new'));
        } else {
            m.tab.removeAttribute('aria-label');
        }
    }

    popOutActive() {
        const m = this._members.find(x => x.win._id === this._active);
        if (m) { m.win.undock(); }
    }

    // Move the whole group to the other column.
    moveTo(newSide) {
        const oldSide = this.side;
        if (!oldSide || newSide === oldSide) { return; }
        const slot = DockSlots[oldSide];
        if (slot) { slot.removePanel(this.root); }
        // Every window of the group moves, docked or not: one popped out, or
        // one not yet enabled, docks back here (32g review finding 2).
        const all = VirtualWindows.getWindows().filter(w => w._tabGroup === this.name);
        all.forEach(w => {
            VirtualWindows.notifySlotChange(w._id, oldSide, newSide);
            w._dockSide = newSide;
        });
        this.side = null;
        this._ensurePanel(newSide);
        all.forEach(w => LayoutStore.saveWindow(w));
    }

    // The group panel's rectangle, for a popped-out tab to open over.
    panelRect() {
        const slot = DockSlots[this.side];
        const entry = slot && slot._panels.find(p => p.contentEl === this.root);
        return entry ? entry.panel.getBoundingClientRect() : null;
    }
}

const DockTabGroups = (() => {
    const groups = {};
    return {
        get(name) {
            if (!groups[name]) { groups[name] = new DockTabGroup(name); }
            return groups[name];
        },
        // The group holding a window id, or null.
        of(id) {
            return Object.values(groups).find(g => g._members.some(m => m.win._id === id)) || null;
        },
        // The group whose panel is this content element, or null.
        byRoot(contentEl) {
            return Object.values(groups).find(g => g.root === contentEl) || null;
        },
    };
})();

// ---------------------------------------------------------------------------
// LayoutStore
//
// Persists window layout to localStorage under the key 'windowLayout'.
// Saved state per window:
//   enabled       bool    - whether the window is open
//   docked        bool    - whether it is in a dock slot
//   dockSide      string  - 'left' | 'right' (only when docked)
//   dockedHeight  number  - panel height in px (only when docked)
//   floatX        number  - VWin x position (only when floating)
//   floatY        number  - VWin y position (only when floating)
//   floatWidth    number  - VWin width (only when floating)
//   floatHeight   number  - VWin height (only when floating)
// Plus top-level keys:
//   dockWidths    object  - { left: number, right: number }
// ---------------------------------------------------------------------------
const LayoutStore = (() => {
    const KEY = 'windowLayout';
    // VERSION 2 (Phase 32g): the company dock. A layout saved before it
    // names windows that no longer exist and puts the map on the right, so
    // it is discarded once and the player told (takeResetNotice).
    const VERSION = 2;
    let resetNotice = false;

    function load() {
        try {
            const raw = localStorage.getItem(KEY);
            const data = raw ? JSON.parse(raw) : {};
            if (data && Object.keys(data).length > 0 && data.version !== VERSION) {
                localStorage.removeItem(KEY);
                resetNotice = true;
                return {};
            }
            return data || {};
        } catch (e) {
            return {};
        }
    }

    function save(data) {
        try {
            data.version = VERSION;
            localStorage.setItem(KEY, JSON.stringify(data));
        } catch (e) {
            // localStorage unavailable - silently ignore
        }
    }

    // True once after an old layout was discarded.
    function takeResetNotice() {
        load();
        const was = resetNotice;
        resetNotice = false;
        return was;
    }

    function saveActiveTab(group, id) {
        patch(data => {
            if (!data.activeTabs) { data.activeTabs = {}; }
            data.activeTabs[group] = id;
        });
    }

    function getActiveTab(group) {
        const data = load();
        return (data.activeTabs && data.activeTabs[group]) || null;
    }

    // Merge a partial update into the stored layout and persist.
    function patch(updater) {
        const data = load();
        updater(data);
        save(data);
    }

    // Save the current state of a single VirtualWindow.
    function saveWindow(win) {
        patch(data => {
            if (!data.windows) { data.windows = {}; }
            const entry = data.windows[win._id] || {};

            entry.enabled = win.isOpen() || win._win === false ? win.isOpen() : true;

            if (win._win === 'docked') {
                entry.docked   = true;
                entry.dockSide = win._dockSide;
                // Read current rendered panel height
                const slot = DockSlots[win._dockSide];
                if (slot) {
                    const pe = slot._panels.find(p => p.contentEl === win._contentEl);
                    if (pe) { entry.dockedHeight = Math.round(pe.panel.offsetHeight); }
                }
            } else if (win._win && win._win !== false) {
                entry.docked     = false;
                entry.dockSide   = win._dockSide;
                entry.floatX     = Math.round(win._win.x);
                entry.floatY     = Math.round(win._win.y);
                entry.floatWidth  = Math.round(win._win.width);
                entry.floatHeight = Math.round(win._win.height);
            } else {
                // closed - preserve last known docked state
                if (entry.docked === undefined) {
                    entry.docked   = win._defaultDocked;
                    entry.dockSide = win._dockSide;
                }
            }

            data.windows[win._id] = entry;
        });
    }

    // Save dock slot widths.
    function saveDockWidths() {
        patch(data => {
            data.dockWidths = {};
            ['left', 'right'].forEach(side => {
                const slot = DockSlots[side];
                if (slot && slot.el && slot.el.classList.contains('has-panels')) {
                    data.dockWidths[side] = slot.el.offsetWidth;
                }
            });
        });
    }

    // Return saved state for a single window, or null.
    function getWindow(id) {
        const data = load();
        return (data.windows && data.windows[id]) ? data.windows[id] : null;
    }

    function getDockWidths() {
        const data = load();
        return data.dockWidths || {};
    }

    function reset() {
        try { localStorage.removeItem(KEY); } catch (e) { /* unavailable */ }
    }

    function clearWindow(id) {
        patch(data => {
            if (data.windows) { delete data.windows[id]; }
        });
    }

    function saveDockOrder(order) {
        patch(data => {
            if (order) {
                data.dockOrder = order;
            } else {
                delete data.dockOrder;
            }
        });
    }

    function getDockOrder() {
        const data = load();
        return data.dockOrder || null;
    }

    return { saveWindow, saveDockWidths, getWindow, getDockWidths, reset, clearWindow, saveDockOrder, getDockOrder,
        saveActiveTab, getActiveTab, takeResetNotice };
})();

// ---------------------------------------------------------------------------
// VirtualWindow
//
// Wraps a VWin instance with a well-defined lifecycle and optional docking.
//
// States:
//   undefined  -> never opened
//   'docked'   -> content is in a dock slot panel; no VWin exists
//   VWin obj   -> floating
//   false      -> user closed it from floating state; will not reopen
//
// Constructor options (passed as the second argument to VirtualWindow):
//   factory()         required  Returns VWin opts object. Must append
//                               the content element to document.body.
//   dock              optional  'left' | 'right'  - which slot to use.
//                               If omitted the window is float-only.
//   defaultDocked     optional  boolean - start docked instead of floating.
//   dockedHeight      optional  number (px) - preferred panel height when docked.
//                               Defaults to the height from the factory opts.
//
// Usage:
//   const win = new VirtualWindow('id', {
//       factory() { ... return { title, mount: el, ... }; },
//       dock: 'right',
//       defaultDocked: true,
//   });
//   win.open();       creates on first call, no-op if closed by user
//   win.isOpen()      true when floating or docked
//   win.get()         returns VWin instance, or null when docked/closed
//   win.dock()        move from floating -> docked
//   win.undock()      move from docked   -> floating
// ---------------------------------------------------------------------------
class VirtualWindow {
    constructor(id, options) {
        this._id              = id;
        this._factory         = options.factory;
        this._dockSide        = options.dock || null;
        this._defaultDocked   = options.defaultDocked || false;
        this._dockedHeight    = options.dockedHeight || null;
        this._origDockSide    = options.dock || null;
        this._origDockedHeight = options.dockedHeight || null;
        this._win             = options.offOnLoad ? false : undefined;
        this._contentEl       = null;
        this._vwinOpts        = null;
        // Phase 32g: tab groups (DockTabGroup).
        this._tabGroup        = options.tabGroup || null;
        this._tabLabel        = options.tabLabel || null;
        this._groupHeader     = !!options.groupHeader;
        this._onTabShown      = options.onTabShown || null;
    }

    // Take a docked window out of its dock panel or tab group (Phase 32g).
    _removeDocked() {
        if (this._win !== 'docked') { return; }
        const group = this._tabGroup ? DockTabGroups.of(this._id) : null;
        if (group) {
            group.remove(this);
            return;
        }
        const slot = DockSlots[this._dockSide];
        if (slot) { slot.removePanel(this._contentEl); }
    }

    // Open the window. On first call, honours defaultDocked and saved layout.
    // Subsequent calls are no-ops unless the window is not yet open.
    open() {
        if (this._win !== undefined && this._win !== false) { return; }  // already open (float or docked)

        // Check saved layout for this window
        const saved = LayoutStore.getWindow(this._id);

        // If saved as disabled, mark closed and stop.
        if (saved && saved.enabled === false) {
            this._win = false;
            return;
        }

        // offOnLoad windows stay closed unless the user has explicitly enabled them.
        if (this._win === false && !(saved && saved.enabled === true)) {
            return;
        }

        // Reset to undefined so the rest of open() treats this as a first open.
        this._win = undefined;

        // First open: run the factory to get opts + content element
        const opts = this._factory();
        if (!opts) { return; }
        this._vwinOpts   = opts;
        this._contentEl  = opts.mount;

        // Apply saved float geometry if present
        if (saved && saved.docked === false && saved.floatWidth) {
            this._vwinOpts.x      = saved.floatX;
            this._vwinOpts.y      = saved.floatY;
            this._vwinOpts.width  = saved.floatWidth;
            this._vwinOpts.height = saved.floatHeight;
        }

        // Apply saved docked height
        if (saved && saved.docked !== false && saved.dockedHeight) {
            this._dockedHeight = saved.dockedHeight;
        }

        // Determine whether to dock or float
        const shouldDock = saved
            ? (saved.docked !== false && !!this._dockSide)
            : (this._dockSide && this._defaultDocked);

        // If saved on a different dock side, update
        if (saved && saved.dockSide && saved.docked !== false) {
            this._dockSide = saved.dockSide;
        }

        if (shouldDock) {
            this._dockNow();
        } else {
            this._floatNow();
        }
    }

    isOpen() {
        return this._win === 'docked' || (!!this._win && this._win !== false);
    }

    // Re-open a window that was previously closed by the user.
    // Resets the closed state and opens as if for the first time.
    reopen() {
        if (this._win !== false) { return; }
        this._win = undefined;
        // Reset content so factory runs again on open()
        this._contentEl  = null;
        this._vwinOpts   = null;
        // Clear any saved enabled:false so open() does not immediately re-close.
        LayoutStore.clearWindow(this._id);
        this.open();
        LayoutStore.saveWindow(this);
    }

    // Returns the VWin instance when floating, null when docked or closed.
    get() {
        return (this._win && this._win !== false && this._win !== 'docked')
            ? this._win : null;
    }

    // Move from floating to docked. Safe to call when already docked.
    dock() {
        if (!this._dockSide)          { return; }
        if (this._win === 'docked')   { return; }
        if (!this._win || this._win === false) { return; }

        // Destroy the VWin without triggering the user-close state
        const wb = this._win;
        wb.onclose = null;
        wb.close();
        this._win = undefined;

        this._dockNow();
        LayoutStore.saveWindow(this);
    }

    // Move from docked to floating. Safe to call when already floating.
    undock() {
        if (this._win !== 'docked') { return; }

        const group = this._tabGroup ? DockTabGroups.of(this._id) : null;
        if (group) {
            // A tab pops out over its group's panel.
            const rect = group.panelRect();
            group.remove(this);
            this._win = undefined;
            let spawnY = null, spawnSize = null;
            if (rect) {
                const margin = 10;
                spawnY    = Math.min(Math.max(margin, Math.round(rect.top)), window.innerHeight - Math.round(rect.height) - margin);
                spawnSize = { width: Math.round(rect.width), height: Math.round(rect.height) - 24 };
            }
            this._floatNow(spawnY, spawnSize);
            LayoutStore.saveWindow(this);
            return;
        }

        const slot = DockSlots[this._dockSide];

        // Measure the panel's current rendered dimensions before removing it,
        // so the floating window matches what the user saw while docked.
        // The titlebar height (~24px) is subtracted from the panel height to
        // get the content-only height that VWin will use for its body.
        let spawnY    = null;
        let spawnSize = null;
        const panelEntry = slot._panels.find(p => p.contentEl === this._contentEl);
        if (panelEntry) {
            const rect      = panelEntry.panel.getBoundingClientRect();
            const titlebar  = panelEntry.panel.querySelector('.dock-panel-titlebar');
            const titleH    = titlebar ? titlebar.offsetHeight : 24;
            const margin    = 10;
            const spawnH    = Math.round(rect.height);
            const maxY      = window.innerHeight - spawnH - margin;
            spawnY    = Math.min(Math.max(margin, Math.round(rect.top)), maxY);
            spawnSize = { width: Math.round(rect.width), height: spawnH - titleH };
        }

        slot.removePanel(this._contentEl);
        this._win = undefined;

        this._floatNow(spawnY, spawnSize);
        LayoutStore.saveWindow(this);
    }

    // -----------------------------------------------------------------------
    // Private
    // -----------------------------------------------------------------------

    _floatNow(spawnY, spawnSize) {
        const opts = Object.assign({}, this._vwinOpts);

        // If spawning from a docked position, use the panel's measured dimensions
        // and place the window inset from the dock edge.
        if (spawnY !== undefined && spawnY !== null) {
            opts.y = spawnY;
            opts.x = this._dockSide === 'right'
                ? window.innerWidth  - (spawnSize ? spawnSize.width : (opts.width  || 363)) - 50
                : 50;
        }
        if (spawnSize) {
            opts.width  = spawnSize.width;
            opts.height = spawnSize.height;
        }

        // Re-attach content to body if it was moved by the dock slot
        if (this._contentEl && !document.body.contains(this._contentEl)) {
            document.body.appendChild(this._contentEl);
        }
        opts.mount = this._contentEl;

        // Inject close handler - sets state to false and removes the content
        // element from the DOM so VWin's unmount() doesn't leave it visible
        // as a bare element on document.body.
        const userOnClose = opts.onclose;
        opts.onclose = (force) => {
            this._win = false;
            if (this._contentEl && this._contentEl.parentNode) {
                this._contentEl.parentNode.removeChild(this._contentEl);
            }
            LayoutStore.saveWindow(this);
            if (typeof userOnClose === 'function') { return userOnClose(force); }
            return false;
        };

        // Save position/size when the user moves or resizes the floating window.
        const self = this;
        const _saveThrottled = (() => {
            let t = null;
            return () => {
                clearTimeout(t);
                t = setTimeout(() => LayoutStore.saveWindow(self), 300);
            };
        })();
        const existingOnMove   = opts.onmove;
        const existingOnResize = opts.onresize;
        opts.onmove = function(x, y) {
            if (existingOnMove) { existingOnMove.call(this, x, y); }
            _saveThrottled();
        };
        opts.onresize = function(w, h) {
            if (existingOnResize) { existingOnResize.call(this, w, h); }
            _saveThrottled();
        };

        // Wrap oncreate to add the dock button.
        // IMPORTANT: this._win must be set before oncreate fires because VWin
        // calls oncreate synchronously inside its constructor, before the
        // assignment `this._win = new VWin(opts)` completes. We use a
        // placeholder object so addControl can be called safely, then replace
        // it with the real VWin instance immediately after construction.
        const existingOncreate = opts.oncreate;
        if (this._dockSide) {
            const self = this;
            opts.oncreate = function(o) {
                if (existingOncreate) { existingOncreate.call(this, o); }
                this.addControl({
                    index: 0,
                    class: 'vw-dock-btn',
                    click: () => self.dock(),
                });
            };
        } else if (existingOncreate) {
            opts.oncreate = existingOncreate;
        }

        this._win = new VWin(opts);
    }

    _dockNow() {
        if (this._tabGroup) {
            // A tab docks into its group wherever the group now is, even if
            // the group moved while this tab was out (32g review finding 2).
            const group = DockTabGroups.get(this._tabGroup);
            if (group.side) { this._dockSide = group.side; }
            group.add(this, this._dockSide);
            this._win = 'docked';
            LayoutStore.saveWindow(this);
            return;
        }
        const slot      = DockSlots[this._dockSide];
        const height    = this._dockedHeight || (this._vwinOpts && this._vwinOpts.height) || null;
        const insertAt  = VirtualWindows.getDockInsertIndex(this);
        const docked = slot.addPanel(
            this._contentEl,
            this._vwinOpts.title,
            () => this.undock(),
            height,
            () => {
                // User clicked X on the docked panel - same semantics as
                // closing a floating window: remove content and deregister.
                if (this._contentEl && this._contentEl.parentNode) {
                    this._contentEl.parentNode.removeChild(this._contentEl);
                }
                this._win = false;
            },
            (newSide) => {
                // User dragged the panel to the opposite slot.
                // Update the canonical order tables: remove from old side,
                // append to new side (position will settle via notifyReorder
                // once the panel is dropped and _movePanel fires, but we need
                // the ID present in the new side's list for getDockInsertIndex).
                VirtualWindows.notifySlotChange(this._id, this._dockSide, newSide);
                slot.removePanel(this._contentEl);
                this._dockSide = newSide;
                this._win = undefined;
                this._dockNow();
            },
            insertAt
        );
        // Phase 40i: the phone layout shows one panel at a time, by window id.
        if (docked) { docked.dataset.win = this._id; }
        this._win = 'docked';
        LayoutStore.saveWindow(this);
    }
}

// ---------------------------------------------------------------------------
// Default dock layout configuration.
//
// Defines which side each window docks to by default and the order windows
// appear within each dock (top-to-bottom).  Edit this array to change the
// out-of-the-box layout.  Modules not listed here fall back to their own
// constructor-declared dock side and are appended after configured windows.
// ---------------------------------------------------------------------------
// Phase 32g: the left column is the world (time, map, room, the tutorial
// while in the course); the right column is the company dock, one tab
// group ('dock'): the vitals strip above Character, Company, Combat, Comm,
// and, when enabled in Settings, Who (Online) and Kills (KillStats). A
// group's order here is its tab order.
const WINDOW_DOCK_DEFAULTS = [
    { id: 'Time & Date',    side: 'left' },
    { id: 'Map',            side: 'left' },
    { id: 'RoomInfo',       side: 'left' },
    { id: 'Tutorial',       side: 'left' },
    { id: 'Vitals',         side: 'right', group: 'dock' },
    { id: 'Character',      side: 'right', group: 'dock' },
    { id: 'Company',        side: 'right', group: 'dock' },
    { id: 'Combat',         side: 'right', group: 'dock' },
    { id: 'Communications', side: 'right', group: 'dock' },
    { id: 'Online',         side: 'right', group: 'dock' },
    { id: 'KillStats',      side: 'right', group: 'dock' },
];

// ---------------------------------------------------------------------------
// VirtualWindows registry
//
// Window modules call VirtualWindows.register(descriptor) where descriptor is:
//   {
//       window:       VirtualWindow instance (required for openAll)
//       gmcpHandlers: ['Char.Vitals', 'Char'],   // GMCP namespaces this handles
//       onGMCP(namespace, data) { ... }           // called when any listed namespace updates
//   }
//
// Multiple modules may register for the same namespace - all handlers are called.
// handleGMCP(namespace, body) walks from the most-specific to least-specific
// namespace segment and calls every handler registered at the first level that
// has any handlers.
// openAll() opens every registered window immediately - called by Client.init().
// ---------------------------------------------------------------------------
const VirtualWindows = (() => {
    // Map<gmcpNamespace, Array<handler function>>
    const _handlers = {};
    // Ordered list of all registered VirtualWindow instances
    const _windows  = [];

    // Per-dock-side ordered list of window IDs, representing the canonical
    // slot order. Populated on register() and updated on drag reorder.
    // Map<side, string[]>
    const _dockOrder         = { left: [], right: [] };
    const _dockOrderOriginal = { left: [], right: [] };

    // Precompute per-side default order from WINDOW_DOCK_DEFAULTS
    const _defaultSideOrder = { left: [], right: [] };
    WINDOW_DOCK_DEFAULTS.forEach(d => {
        if (_defaultSideOrder[d.side]) { _defaultSideOrder[d.side].push(d.id); }
    });

    function register(descriptor) {
        if (!descriptor || !Array.isArray(descriptor.gmcpHandlers)) {
            console.error('VirtualWindows.register: descriptor must have gmcpHandlers array');
            return;
        }
        if (typeof descriptor.onGMCP !== 'function') {
            console.error('VirtualWindows.register: descriptor must have onGMCP function');
            return;
        }
        const win = (descriptor.window instanceof VirtualWindow) ? descriptor.window : null;
        descriptor.gmcpHandlers.forEach(ns => {
            if (!_handlers[ns]) {
                _handlers[ns] = [];
            }
            // Store the handler alongside its window so dispatch can skip
            // handlers whose window has been closed by the user.
            _handlers[ns].push({ fn: descriptor.onGMCP.bind(descriptor), win });
        });
        if (win) {
            _windows.push(win);

            // Apply configured default dock side from WINDOW_DOCK_DEFAULTS
            const defaultEntry = WINDOW_DOCK_DEFAULTS.find(d => d.id === win._id);
            if (defaultEntry) {
                win._dockSide     = defaultEntry.side;
                win._origDockSide = defaultEntry.side;
                if (defaultEntry.group && !win._tabGroup) { win._tabGroup = defaultEntry.group; }
            }

            // Insert into _dockOrderOriginal (and _dockOrder) at the position
            // dictated by WINDOW_DOCK_DEFAULTS so the default layout matches
            // the configured order regardless of file-load order.
            const side = win._dockSide;
            if (side && _dockOrderOriginal[side]) {
                const sideOrder = _defaultSideOrder[side];
                const configIdx = sideOrder ? sideOrder.indexOf(win._id) : -1;

                if (configIdx === -1) {
                    _dockOrderOriginal[side].push(win._id);
                    _dockOrder[side].push(win._id);
                } else {
                    let insertAt = _dockOrderOriginal[side].length;
                    for (let i = 0; i < _dockOrderOriginal[side].length; i++) {
                        const existingIdx = sideOrder.indexOf(_dockOrderOriginal[side][i]);
                        if (existingIdx !== -1 && existingIdx > configIdx) {
                            insertAt = i;
                            break;
                        }
                    }
                    _dockOrderOriginal[side].splice(insertAt, 0, win._id);
                    _dockOrder[side].splice(insertAt, 0, win._id);
                }
            }
        }
    }

    // Returns the index at which a window should be inserted into its dock slot,
    // based on the canonical order relative to currently docked windows.
    function getDockInsertIndex(win) {
        return getDockInsertIndexFor(win._id, win._dockSide);
    }

    // The window id a dock panel stands for: its window's, or for a tab
    // group's panel the group's anchor (Phase 32g).
    function _idForContent(contentEl) {
        const w = _windows.find(x => x._contentEl === contentEl);
        if (w) { return w._id; }
        const group = DockTabGroups.byRoot(contentEl);
        return group ? group.anchorId() : null;
    }

    function getDockInsertIndexFor(id, side) {
        if (!side || !_dockOrder[side]) { return undefined; }

        const order     = _dockOrder[side];
        const winPos    = order.indexOf(id);
        if (winPos === -1) { return undefined; }

        const slot      = DockSlots[side];
        if (!slot)       { return undefined; }

        // Count how many currently-docked panels belong to windows that
        // appear before this window in the canonical order.
        let insertIdx = 0;
        for (const entry of slot._panels) {
            const entryId = _idForContent(entry.contentEl);
            if (entryId === null) { continue; }
            const entryPos = order.indexOf(entryId);
            if (entryPos !== -1 && entryPos < winPos) {
                insertIdx++;
            }
        }
        return insertIdx;
    }

    function _saveDockOrder() {
        LayoutStore.saveDockOrder({ left: [..._dockOrder.left], right: [..._dockOrder.right] });
    }

    // Called by DockSlot._movePanel after a drag reorder completes.
    // newOrder is the array of contentEl references in their new slot order.
    function notifyReorder(side, newOrder) {
        if (!_dockOrder[side]) { return; }
        // Rebuild the canonical order for this side by mapping contentEls back
        // to window IDs, preserving the positions of any IDs not currently docked.
        const currentIds = newOrder
            .map(contentEl => _idForContent(contentEl))
            .filter(id => id !== null);

        // Merge: replace positions of currently-docked windows with the new
        // order, leaving undocked/closed windows at their last known positions.
        const undocked = _dockOrder[side].filter(id => !currentIds.includes(id));
        // Interleave undocked IDs back into the new order at their nearest position
        const merged = [...currentIds];
        for (const id of undocked) {
            const oldIdx = _dockOrder[side].indexOf(id);
            // Find the insertion point: after the last merged entry whose old
            // index was less than oldIdx.
            let insertAt = 0;
            for (let i = 0; i < merged.length; i++) {
                const mergedOld = _dockOrder[side].indexOf(merged[i]);
                if (mergedOld !== -1 && mergedOld < oldIdx) {
                    insertAt = i + 1;
                }
            }
            merged.splice(insertAt, 0, id);
        }
        _dockOrder[side] = merged;
        _saveDockOrder();
    }

    // Called when a window is dragged from one slot to the other.
    // Removes the window ID from oldSide's order and appends it to newSide's.
    function notifySlotChange(winId, oldSide, newSide) {
        if (_dockOrder[oldSide]) {
            const idx = _dockOrder[oldSide].indexOf(winId);
            if (idx !== -1) { _dockOrder[oldSide].splice(idx, 1); }
        }
        if (_dockOrder[newSide] && !_dockOrder[newSide].includes(winId)) {
            _dockOrder[newSide].push(winId);
        }
        _saveDockOrder();
    }

    function resetOrder() {
        _dockOrder.left  = [..._dockOrderOriginal.left];
        _dockOrder.right = [..._dockOrderOriginal.right];
        LayoutStore.saveDockOrder(null);
    }

    function handleGMCP(namespace, body) {
        // Walk from most-specific to least-specific namespace segment.
        // Call all handlers registered at the first matching level,
        // skipping any whose associated window has been closed.
        var handled = false;
        const parts = namespace.split('.');
        for (let i = parts.length; i >= 1; i--) {
            const path = parts.slice(0, i).join('.');
            if (_handlers['*'] && _handlers['*'].length > 0) {
                _handlers['*'].forEach(entry => {
                    if (entry.win && !entry.win.isOpen()) { return; }
                    entry.fn(namespace, body);
                    handled = true;
                });
            }
            if (_handlers[path] && _handlers[path].length > 0) {
                _handlers[path].forEach(entry => {
                    if (entry.win && !entry.win.isOpen()) { return; }
                    entry.fn(namespace, body);
                    handled = true;
                });
            }
        }
        if (!handled) {
            console.log('GMCP (unhandled):', namespace, body);
        }
    }

    function openAll() {
        // Restore saved dock order, or fall back to defaults.
        const savedOrder = LayoutStore.getDockOrder();
        if (savedOrder) {
            const allRegistered = new Set(_windows.map(w => w._id));
            const inSaved = new Set([
                ...(savedOrder.left  || []),
                ...(savedOrder.right || []),
            ]);
            ['left', 'right'].forEach(side => {
                if (!savedOrder[side]) {
                    _dockOrder[side] = [..._dockOrderOriginal[side]];
                    return;
                }
                const restored = savedOrder[side].filter(id => allRegistered.has(id));
                // Append any newly-registered windows not present in the saved order
                _dockOrderOriginal[side].forEach(id => {
                    if (!inSaved.has(id) && !restored.includes(id)) {
                        const sideOrder = _defaultSideOrder[side];
                        const configIdx = sideOrder ? sideOrder.indexOf(id) : -1;
                        let insertAt = restored.length;
                        if (configIdx !== -1) {
                            for (let i = 0; i < restored.length; i++) {
                                const ei = sideOrder.indexOf(restored[i]);
                                if (ei !== -1 && ei > configIdx) { insertAt = i; break; }
                            }
                        }
                        restored.splice(insertAt, 0, id);
                    }
                });
                _dockOrder[side] = restored;
            });
            // Sync each window's dock side with the loaded order so that
            // cross-dock moves from a prior session are honoured.
            _windows.forEach(win => {
                if (_dockOrder.left.includes(win._id) && win._dockSide !== 'left') {
                    win._dockSide = 'left';
                } else if (_dockOrder.right.includes(win._id) && win._dockSide !== 'right') {
                    win._dockSide = 'right';
                }
            });
        } else {
            _dockOrder.left  = [..._dockOrderOriginal.left];
            _dockOrder.right = [..._dockOrderOriginal.right];
        }

        _windows.forEach(win => win.open());

        // Restore saved dock slot widths
        const widths = LayoutStore.getDockWidths();
        ['left', 'right'].forEach(side => {
            if (!widths[side]) { return; }
            const slot = DockSlots[side];
            if (!slot || !slot.el || !slot.el.classList.contains('has-panels')) { return; }
            slot.el.style.setProperty('--dock-' + side + '-width', widths[side] + 'px');
            slot.el.style.width = widths[side] + 'px';
        });
        // Trigger a terminal resize after layout is settled
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    }

    function getWindows() {
        return _windows.slice();
    }

    function setConnected(connected) {
        if (connected) {
            document.body.classList.remove('windows-disconnected');
            // Phase 47: windows that asked for state once per page (the Room
            // window's gather in progress) ask again after a reconnect.
            window.dispatchEvent(new Event('vwin:connected'));
        } else {
            document.body.classList.add('windows-disconnected');
        }
    }

    // Phase 32g: a tab's count, and whether its tab is showing.
    function setTabBadge(id, text, spoken) {
        const group = DockTabGroups.of(id);
        if (group) { group.setBadge(id, text, spoken); }
    }

    function isTabShowing(id) {
        const group = DockTabGroups.of(id);
        return group ? group.isActive(id) : true;
    }

    return { register, handleGMCP, openAll, getWindows, setConnected, getDockInsertIndex, getDockInsertIndexFor,
        notifyReorder, notifySlotChange, resetOrder, setTabBadge, isTabShowing };
})();

// ---------------------------------------------------------------------------
// Client namespace
//
// Shared state and services that window modules may read or call.
// Nothing here is truly private - window modules are trusted collaborators.
// ---------------------------------------------------------------------------
const Client = (() => {

    // -----------------------------------------------------------------------
    // Audio
    // -----------------------------------------------------------------------
    let baseMp3Url = '';
    const MusicPlayer = new MP3Player(false);
    const SoundPlayer = new MP3Player(true);

    // -----------------------------------------------------------------------
    // Terminal
    // -----------------------------------------------------------------------
    const term = new window.Terminal({
        cols:        80,
        rows:        60,
        cursorBlink: true,
        fontSize:    20,
    });
    const fitAddon = new window.FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    function resizeTerminal() {
        const hasLeft  = DockSlots.left  && DockSlots.left.el  && DockSlots.left.el.classList.contains('has-panels');
        const hasRight = DockSlots.right && DockSlots.right.el && DockSlots.right.el.classList.contains('has-panels');
        // Phase 40i: the phone layout (mobile.js) shows one view at a time.
        const phone = document.body.classList.contains('mobile');
        const fontSize = phone ? 13 : (hasLeft && hasRight) ? 16 : (hasLeft || hasRight) ? 18 : 20;
        if (term.options.fontSize !== fontSize) {
            term.options.fontSize = fontSize;
        }
        fitAddon.fit();
    }

    // -----------------------------------------------------------------------
    // Networking stats
    // -----------------------------------------------------------------------
    let totalBytesReceived = 0;
    let totalBytesSent     = 0;
    const gmcpInBytes      = {};  // namespace -> bytes received
    const gmcpInCount      = {};  // namespace -> number of payloads received
    const gmcpOutBytes     = {};  // identifier -> bytes sent
    const gmcpOutCount     = {};  // identifier -> number of payloads sent
    const gmcpOutLast      = {};  // identifier -> last sent payload string
    let connectTime        = null; // Date of last successful connection

    // -----------------------------------------------------------------------
    // Command history
    // -----------------------------------------------------------------------
    let commandHistory          = [];
    let historyPosition         = 0;
    const commandHistoryMaxLength = 30;

    // -----------------------------------------------------------------------
    // GMCP state store
    //
    // GMCPStructs holds the most-recently-received value for every namespace.
    // Window modules read from it inside their onGMCP callbacks.
    // -----------------------------------------------------------------------
    const GMCPStructs = {};

    function _applyGMCPPayload(namespace, body) {
        const parts        = namespace.split('.');
        const lastProperty = parts.pop();
        let cursor         = GMCPStructs;
        for (const seg of parts) {
            if (!cursor[seg]) {
                cursor[seg] = {};
            }
            cursor = cursor[seg];
        }
        cursor[lastProperty] = body;
    }

    // -----------------------------------------------------------------------
    // Battle events (Phase 40e)
    //
    // Company.Battle.Event is a stream of combat happenings, released in step
    // with the narration: { fight, round, events: [{ seq, kind, src, tgt, ... }] }.
    // It is not state, so it is never stored in GMCPStructs and never reaches
    // the windows' onGMCP handlers; a listener added with
    // Client.onBattleEvents(fn) gets each message (the battle screen,
    // Phases 40f and 40g). Refs match Company.Battle: a member key (the
    // player is "leader", their cell's key), "me" for a player leading no
    // company, "m:<instance>", "u:<id>", or "?" for an enemy that can't be
    // made out.
    // -----------------------------------------------------------------------
    const _battleEventListeners = [];

    function onBattleEvents(fn) {
        if (typeof fn !== 'function') { return function() {}; }
        _battleEventListeners.push(fn);
        return function() {
            const i = _battleEventListeners.indexOf(fn);
            if (i >= 0) { _battleEventListeners.splice(i, 1); }
        };
    }

    function _dispatchBattleEvents(body) {
        if (!body || !Array.isArray(body.events)) { return; }
        debugLog('Company.Battle.Event ' + JSON.stringify(body));
        _battleEventListeners.slice().forEach(function(fn) {
            try { fn(body); } catch (err) { console.error('battle event listener failed', err); }
        });
    }

    // -----------------------------------------------------------------------
    // WebSocket
    // -----------------------------------------------------------------------
    let socket               = null;
    let pendingReconnectToken = null;
    let debugOutput           = false;  // set Client.debug = true from the console to enable

    function debugLog(msg) {
        if (debugOutput) {
            console.log(msg);
        }
    }

    function SendInput(str) {
        sendData(str);
    }

    //
    // Request that the server send a GMCP payload.
    // Examples: Party, Room, Char
    // If additional is provided it is appended after a space: GMCPRequest('Help', 'train') -> !!GMCP(Help train)
    //
    function GMCPRequest(identifier, additional) {
        const payload = additional !== undefined ? identifier + ' ' + additional : identifier;
        const msg = `!!GMCP(${payload})`;
        gmcpOutBytes[identifier] = (gmcpOutBytes[identifier] || 0) + msg.length;
        gmcpOutCount[identifier] = (gmcpOutCount[identifier] || 0) + 1;
        gmcpOutLast[identifier]  = payload;
        sendData(msg);
    }

    function sendData(dataToSend) {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return false;
        }
        totalBytesSent += dataToSend.length;
        socket.send(dataToSend);
        return true;
    }

    function _parseMSPProps(parts, startIndex) {
        const props = {};
        for (let i = startIndex; i < parts.length; i++) {
            const eq = parts[i].indexOf('=');
            if (eq !== -1) {
                props[parts[i].slice(0, eq)] = parts[i].slice(eq + 1);
            }
        }
        return props;
    }

    function _handleMusicCommand(raw) {
        const inner  = raw.slice(8, raw.length - 1);
        const parts  = inner.split(' ');
        const fileName = parts[0];
        const obj    = _parseMSPProps(parts, 1);

        if (fileName === 'Off') {
            if (obj.U) {
                baseMp3Url = obj.U;
                if (baseMp3Url[baseMp3Url.length - 1] !== '/') {
                    baseMp3Url += '/';
                }
            } else {
                MusicPlayer.stop();
            }
            return;
        }

        let loopMusic  = true;
        let soundLevel = 1.0;
        if (obj.L && obj.L !== '-1') { loopMusic  = false; }
        if (obj.V)                    { soundLevel = Number(obj.V) / 100; }

        if (!MusicPlayer.isPlaying(baseMp3Url + fileName)) {
            const mult = _recordSound('music', baseMp3Url + fileName);
            MusicPlayer.play(baseMp3Url + fileName, loopMusic, soundLevel * (sliderValues['music'] / 100) * mult);
        }
    }

    function _handleSoundCommand(raw) {
        const inner    = raw.slice(8, raw.length - 1);
        const parts    = inner.split(' ');
        const fileName = parts[0];
        const obj      = _parseMSPProps(parts, 1);

        if (fileName === 'Off') {
            if (obj.U) {
                baseMp3Url = obj.U;
                if (baseMp3Url[baseMp3Url.length - 1] !== '/') {
                    baseMp3Url += '/';
                }
            } else {
                SoundPlayer.stop();
            }
            return;
        }

        let soundLevel = 1.0;
        let loopSound  = true;
        if (obj.L && obj.L !== '-1') { loopSound  = false; }
        if (obj.V)                    { soundLevel = Number(obj.V) / 100; }

        const typeKey = ((obj.T || 'other').toLowerCase()) + ' sounds';
        const mult = _recordSound(typeKey, baseMp3Url + fileName);
        SoundPlayer.play(baseMp3Url + fileName, false, soundLevel * (sliderValues[typeKey] / 100) * mult);
    }

    function _handleWebclientCommand(data) {
        if (data.startsWith('TEXTMASK:')) {
            debugLog(data);
            textInput.type = data.substring(9) === 'true' ? 'password' : 'text';
            return true;
        }
        if (data.startsWith('RELOGTKN:')) {
            pendingReconnectToken = data.substring(9);
            return true;
        }
        return false;
    }

    // A pending prompt question (yes/no, a name, a choice) replaces the game
    // prompt with a line that starts ".:" and waits on the same line, where
    // a blank Enter takes its default. While one waits, Enter and the arrows
    // answer it as typed text, not the quick menu or walking.
    let questionPending = false;
    function _trackQuestion(data) {
        const plain = data.replace(/\x1b\[[0-9;?]*[A-Za-z]/g, '');
        const tail  = plain.slice(Math.max(plain.lastIndexOf('\n'), plain.lastIndexOf('\r')) + 1);
        if (tail.trim() === '') { return; }
        questionPending = tail.trimStart().startsWith('.:');
    }

    // playing: logged in and in a room, not at the login prompts or a
    // pending question, so the empty box's keys can walk and open menus.
    function playing() {
        return !questionPending && !!(GMCPStructs.Room && GMCPStructs.Room.Info);
    }

    function _onMessage(event) {
        totalBytesReceived += event.data.length;

        // Webclient protocol commands (TEXTMASK:, RELOGTKN:)
        if (_handleWebclientCommand(event.data)) {
            return;
        }

        // MSP / GMCP commands (all start with "!!")
        if (event.data.length > 2 && event.data.slice(0, 2) === '!!') {

            if (event.data.slice(0, 7) === '!!GMCP(') {
                const gmcpPayload = event.data.trim().slice(7, event.data.length - 1).trim();
                const lastChar    = gmcpPayload[gmcpPayload.length - 1];
                const jsonIndex   = (lastChar === '}') ? gmcpPayload.indexOf('{') : gmcpPayload.indexOf('[');
                if (jsonIndex === -1) {
                    return;
                }
                const gmcpNamespace = gmcpPayload.slice(0, jsonIndex).trim();
                const gmcpBody      = JSON.parse(gmcpPayload.slice(jsonIndex).trim());
                gmcpInBytes[gmcpNamespace] = (gmcpInBytes[gmcpNamespace] || 0) + event.data.length;
                gmcpInCount[gmcpNamespace] = (gmcpInCount[gmcpNamespace] || 0) + 1;
                if (gmcpNamespace === 'Company.Battle.Event') {
                    _dispatchBattleEvents(gmcpBody);
                    return;
                }
                _applyGMCPPayload(gmcpNamespace, gmcpBody);
                VirtualWindows.handleGMCP(gmcpNamespace, gmcpBody);
                return;
            }

            if (event.data.slice(0, 8) === '!!MUSIC(') {
                _handleMusicCommand(event.data);
                return;
            }

            if (event.data.slice(0, 8) === '!!SOUND(') {
                _handleSoundCommand(event.data);
                return;
            }
        }

        term.write(event.data);
        _trackQuestion(event.data);
        Triggers.Try(event.data);
    }

    function attachSocketHandlers(openMessage, clearOnOpen) {
        socket.onopen = function() {
            if (clearOnOpen) { term.clear(); }
            term.writeln(openMessage);
            connectButton.style.display = 'none';
            connectButton.disabled = true;
            textInput.focus();
            // Reset all network stats on each new connection
            totalBytesReceived = 0;
            totalBytesSent     = 0;
            Object.keys(gmcpInBytes).forEach(k => delete gmcpInBytes[k]);
            Object.keys(gmcpInCount).forEach(k => delete gmcpInCount[k]);
            Object.keys(gmcpOutBytes).forEach(k => delete gmcpOutBytes[k]);
            Object.keys(gmcpOutCount).forEach(k => delete gmcpOutCount[k]);
            Object.keys(gmcpOutLast).forEach(k => delete gmcpOutLast[k]);
            connectTime = Date.now();
            VirtualWindows.setConnected(true);
        };

        socket.onmessage = _onMessage;

        socket.onerror = function(error) {
            term.writeln('Error: ' + (error.message || 'unknown'));
        };

        socket.onclose = function(event) {
            VirtualWindows.setConnected(false);
            if (event.wasClean) {
                term.writeln('Connection closed cleanly, code=' + event.code + ', reason=' + event.reason);
            } else {
                term.writeln('Connection died');
            }
            connectButton.style.display = 'block';
            connectButton.disabled = false;

            if (textInput.type === 'password') {
                textInput.value = '';
                textInput.type  = 'text';
            }

            if (pendingReconnectToken) {
                const token = pendingReconnectToken;
                pendingReconnectToken = null;
                setTimeout(() => reconnectWithToken(token), 500);
            }
        };
    }

    function reconnectWithToken(token) {
        debugLog('Reconnecting with copyover token');
        const wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws';
        socket = new WebSocket(wsUrl);
        attachSocketHandlers('Reconnected after server reboot.', false);
        const origOnOpen = socket.onopen;
        socket.onopen = function() {
            origOnOpen();
            sendData(token);
        };
    }

    // -----------------------------------------------------------------------
    // Volume sliders
    // -----------------------------------------------------------------------
    const defaultSliders = {
        'music':               75,
        'combat sounds':       75,
        'movement sounds':     75,
        'environment sounds':  75,
        'other sounds':        75,
    };

    let sliderValues        = { ...defaultSliders };
    let unmutedSliderValues = null;

    // Per-sound volume multipliers: { [categoryKey]: { [soundUrl]: 0-100 } }
    // A value of 100 means full category volume; lower values reduce further.
    let soundVolumeOverrides = {};

    // Sounds that have been played, grouped by category key.
    // { [categoryKey]: string[] }  - ordered by first-play time, deduplicated.
    let soundHistory = {};

    // Which category rows are expanded in the slider UI.
    const _categoryExpanded = {};

    function _loadSoundStorage() {
        try {
            const h = localStorage.getItem('soundHistory');
            if (h) { soundHistory = JSON.parse(h) || {}; }
        } catch (e) { soundHistory = {}; }
        try {
            const v = localStorage.getItem('soundVolumeOverrides');
            if (v) { soundVolumeOverrides = JSON.parse(v) || {}; }
        } catch (e) { soundVolumeOverrides = {}; }
    }

    function _saveSoundStorage() {
        try {
            localStorage.setItem('soundHistory',        JSON.stringify(soundHistory));
            localStorage.setItem('soundVolumeOverrides', JSON.stringify(soundVolumeOverrides));
        } catch (e) { /* ignore */ }
    }

    // Record that a sound URL was played under a given category key.
    // Returns the effective per-sound volume multiplier (0-1).
    function _recordSound(categoryKey, url) {
        if (!soundHistory[categoryKey]) { soundHistory[categoryKey] = []; }
        if (!soundHistory[categoryKey].includes(url)) {
            soundHistory[categoryKey].push(url);
            _saveSoundStorage();
        }
        const cat = soundVolumeOverrides[categoryKey];
        if (cat && cat[url] !== undefined) {
            return cat[url] / 100;
        }
        return 1.0;
    }

    // Apply a per-sound override value and immediately update the cached Audio element.
    function _setSoundOverride(categoryKey, url, value) {
        if (!soundVolumeOverrides[categoryKey]) { soundVolumeOverrides[categoryKey] = {}; }
        soundVolumeOverrides[categoryKey][url] = value;
        _saveSoundStorage();
        // Update the live Audio element so the change is heard immediately.
        const player = (categoryKey === 'music') ? MusicPlayer : SoundPlayer;
        const finalVol = (sliderValues[categoryKey] / 100) * (value / 100);
        player.setVolume(url, Math.min(1, Math.max(0, finalVol)));
    }

    // Returns a fingerprint of the current soundHistory (total count across all categories).
    // Used by the Volume tab polling to detect new sounds without full rebuilds.
    function _soundHistoryFingerprint() {
        var total = 0;
        Object.keys(soundHistory).forEach(function(k) { total += soundHistory[k].length; });
        return total;
    }

    function resetVolumeControls() {
        sliderValues        = { ...defaultSliders };
        soundVolumeOverrides = {};
        unmutedSliderValues  = null;
        localStorage.setItem('sliderValues',         JSON.stringify(sliderValues));
        localStorage.setItem('soundVolumeOverrides', JSON.stringify(soundVolumeOverrides));
        localStorage.removeItem('unmutedSliderValues');
        localStorage.setItem('muteAllSound', JSON.stringify(false));
        const muteCheckbox = document.getElementById('mute-checkbox');
        const muteIcon     = document.getElementById('mute-icon');
        if (muteCheckbox) { muteCheckbox.checked = false; }
        if (muteIcon)     { muteIcon.textContent = '🔊'; }
        MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);
        buildSliders();
    }

    function getSpeakerIcon(value) {
        value = Number(value);
        if (value === 0)       { return '🔇'; }
        if (value < 33)        { return '🔈'; }
        if (value < 66)        { return '🔉'; }
        return '🔊';
    }

    // Return a short display name for a sound URL (strip path prefix, extension).
    function _soundDisplayName(url) {
        const parts = url.split('/');
        const file  = parts[parts.length - 1] || url;
        return file.replace(/\.mp3$/i, '').replace(/[-_]/g, ' ');
    }

    function buildSliders() {
        const container = document.getElementById('sliders-container');
        container.innerHTML = '';

        Object.keys(sliderValues).forEach(key => {
            // --- Category row (header + category slider) ---
            const categoryBlock = document.createElement('div');
            categoryBlock.className = 'sound-category-block';

            const headerRow = document.createElement('div');
            headerRow.className = 'sound-category-header';

            // Expand/collapse arrow
            const sounds = soundHistory[key] || [];
            const isExpanded = !!_categoryExpanded[key];

            const arrow = document.createElement('span');
            arrow.className   = 'sound-category-arrow';
            arrow.textContent = isExpanded ? '\u25bc' : '\u25b6';

            const label = document.createElement('label');
            label.className   = 'sound-category-label';
            label.textContent = key.toLowerCase().split(' ').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');

            const slider = document.createElement('input');
            slider.type  = 'range';
            slider.min   = 0;
            slider.max   = 100;
            slider.value = sliderValues[key];

            const iconSpan = document.createElement('span');
            iconSpan.className   = 'slider-icon';
            iconSpan.textContent = getSpeakerIcon(sliderValues[key]);

            slider.addEventListener('input', e => {
                const val = Number(e.target.value);
                sliderValues[key] = val;
                iconSpan.textContent = getSpeakerIcon(val);
                localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
                if (key === 'music') {
                    MusicPlayer.setGlobalVolume(val / 100);
                }
                const muteCheckbox = document.getElementById('mute-checkbox');
                if (muteCheckbox.checked && val > 0) {
                    muteCheckbox.checked = false;
                    localStorage.setItem('muteAllSound', JSON.stringify(false));
                    document.getElementById('mute-icon').textContent = '🔊';
                }
            });

            // Clicking the arrow or label toggles expansion
            function toggleExpand() {
                _categoryExpanded[key] = !_categoryExpanded[key];
                buildSliders();
            }
            arrow.addEventListener('click', toggleExpand);
            label.addEventListener('click', toggleExpand);
            headerRow.style.cursor = 'default';

            headerRow.appendChild(arrow);
            headerRow.appendChild(label);
            headerRow.appendChild(slider);
            headerRow.appendChild(iconSpan);
            categoryBlock.appendChild(headerRow);

            // --- Per-sound sub-rows (only when expanded and sounds exist) ---
            if (isExpanded && sounds.length > 0) {
                const soundList = document.createElement('div');
                soundList.className = 'sound-list';

                sounds.forEach(url => {
                    const override = soundVolumeOverrides[key] && soundVolumeOverrides[key][url] !== undefined
                        ? soundVolumeOverrides[key][url]
                        : 100;

                    const row = document.createElement('div');
                    row.className = 'slider-container sound-sub-row';

                    const sLabel = document.createElement('label');
                    sLabel.className   = 'sound-sub-label';
                    sLabel.textContent = _soundDisplayName(url);
                    sLabel.title       = url;

                    const sSlider = document.createElement('input');
                    sSlider.type  = 'range';
                    sSlider.min   = 0;
                    sSlider.max   = 100;
                    sSlider.value = override;

                    sSlider.addEventListener('input', e => {
                        const val = Number(e.target.value);
                        _setSoundOverride(key, url, val);
                    });

                    row.appendChild(sLabel);
                    row.appendChild(sSlider);
                    soundList.appendChild(row);
                });

                categoryBlock.appendChild(soundList);
            } else if (isExpanded && sounds.length === 0) {
                const empty = document.createElement('div');
                empty.className   = 'sound-list-empty';
                empty.textContent = 'No sounds played yet in this session.';
                categoryBlock.appendChild(empty);
            }

            container.appendChild(categoryBlock);
        });
    }

    function toggleMuteAll() {
        const muteCheckbox = document.getElementById('mute-checkbox');
        const muteIcon     = document.getElementById('mute-icon');
        const isChecked    = muteCheckbox.checked;

        if (isChecked) {
            unmutedSliderValues = { ...sliderValues };
            localStorage.setItem('unmutedSliderValues', JSON.stringify(unmutedSliderValues));
            Object.keys(sliderValues).forEach(k => { sliderValues[k] = 0; });
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            buildSliders();
            muteIcon.textContent = '🔇';
            MusicPlayer.setGlobalVolume(0);
            localStorage.setItem('muteAllSound', JSON.stringify(true));
        } else {
            const savedUnmuted = localStorage.getItem('unmutedSliderValues');
            if (savedUnmuted) {
                let loaded = JSON.parse(savedUnmuted) || {};
                loaded = { ...defaultSliders, ...loaded };
                unmutedSliderValues = { ...loaded };
                sliderValues = { ...unmutedSliderValues };
                localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            }
            buildSliders();
            muteIcon.textContent = '🔊';
            MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);
            localStorage.setItem('muteAllSound', JSON.stringify(false));
        }
    }

    function buildWindowToggles() {
        const container = document.getElementById('windows-container');
        if (!container) { return; }
        container.innerHTML = '';

        VirtualWindows.getWindows().forEach(win => {
            const row = document.createElement('div');
            row.className = 'win-toggle-row';

            const label = document.createElement('span');
            label.textContent = win._id;

            const switchEl = document.createElement('label');
            switchEl.className = 'toggle-switch';

            const input = document.createElement('input');
            input.type    = 'checkbox';
            input.checked = win.isOpen();
            input.addEventListener('change', () => {
                if (input.checked) {
                    win.reopen();
                } else {
                    // Close: mimic user closing the window
                    if (win._win === 'docked') {
                        win._removeDocked();
                        if (win._contentEl && win._contentEl.parentNode) {
                            win._contentEl.parentNode.removeChild(win._contentEl);
                        }
                        win._win = false;
                    } else if (win._win && win._win !== false) {
                        const wb = win._win;
                        win._win = false;
                        wb.onclose = null;
                        wb.close();
                        if (win._contentEl && win._contentEl.parentNode) {
                            win._contentEl.parentNode.removeChild(win._contentEl);
                        }
                    }
                    LayoutStore.saveWindow(win);
                }
            });

            const track = document.createElement('span');
            track.className = 'toggle-track';

            switchEl.appendChild(input);
            switchEl.appendChild(track);
            row.appendChild(label);
            row.appendChild(switchEl);
            container.appendChild(row);
        });
    }

    function toggleDockAutoHide(side, enabled) {
        const container = document.getElementById('main-container');
        const cls = 'dock-autohide-' + side;
        if (enabled) {
            container.classList.add(cls);
        } else {
            container.classList.remove(cls);
        }
        var parsed;
        try { parsed = JSON.parse(localStorage.getItem('dockAutoHide')); } catch (e) { /* ignore */ }
        var state = (parsed && typeof parsed === 'object') ? parsed : {};
        state[side] = enabled;
        localStorage.setItem('dockAutoHide', JSON.stringify(state));
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    }

    function toggleMenu() {
        const backdrop = document.getElementById('settings-backdrop');
        const isOpen   = backdrop.classList.contains('open');
        if (!isOpen) {
            buildWindowToggles();
            const mc = document.getElementById('main-container');
            const ahL = document.getElementById('dock-autohide-left-checkbox');
            const ahR = document.getElementById('dock-autohide-right-checkbox');
            if (ahL) { ahL.checked = mc.classList.contains('dock-autohide-left'); }
            if (ahR) { ahR.checked = mc.classList.contains('dock-autohide-right'); }
            backdrop.classList.add('open');
        } else {
            backdrop.classList.remove('open');
        }
    }

    function resetLayout() {
        LayoutStore.reset();
        toggleDockAutoHide('left', false);
        toggleDockAutoHide('right', false);

        // Close all windows first, then reopen each one in its default state.
        VirtualWindows.getWindows().forEach(win => {
            // Tear down whatever state the window is currently in
            if (win._win === 'docked') {
                win._removeDocked();
                if (win._contentEl && win._contentEl.parentNode) {
                    win._contentEl.parentNode.removeChild(win._contentEl);
                }
            } else if (win._win && win._win !== false) {
                const wb = win._win;
                wb.onclose = null;
                wb.close();
                if (win._contentEl && win._contentEl.parentNode) {
                    win._contentEl.parentNode.removeChild(win._contentEl);
                }
            } else if (win._win === false && win._contentEl && win._contentEl.parentNode) {
                win._contentEl.parentNode.removeChild(win._contentEl);
            }

            // Reset all window state back to construction defaults
            win._win         = undefined;
            win._contentEl   = null;
            win._vwinOpts    = null;
            win._dockSide    = win._origDockSide;
            win._dockedHeight = win._origDockedHeight;
        });

        // Restore dock slot widths to default (remove inline styles)
        ['left', 'right'].forEach(side => {
            const slot = DockSlots[side];
            if (!slot || !slot.el) { return; }
            slot.el.style.removeProperty('width');
            slot.el.style.removeProperty('--dock-' + side + '-width');
        });

        // Reopen all windows in default positions
        VirtualWindows.openAll();
    }


    // -----------------------------------------------------------------------
    // Net stats - readable by the settings Stats tab
    // -----------------------------------------------------------------------
    function getNetStats() {
        return {
            totalBytesSent,
            totalBytesReceived,
            gmcpInBytes:   Object.assign({}, gmcpInBytes),
            gmcpInCount:   Object.assign({}, gmcpInCount),
            gmcpOutBytes:  Object.assign({}, gmcpOutBytes),
            gmcpOutCount:  Object.assign({}, gmcpOutCount),
            gmcpOutLast:   Object.assign({}, gmcpOutLast),
            connectTime,
        };
    }

    function resetNetStats() {
        totalBytesReceived = 0;
        totalBytesSent     = 0;
        Object.keys(gmcpInBytes).forEach(k  => delete gmcpInBytes[k]);
        Object.keys(gmcpInCount).forEach(k  => delete gmcpInCount[k]);
        Object.keys(gmcpOutBytes).forEach(k => delete gmcpOutBytes[k]);
        Object.keys(gmcpOutCount).forEach(k => delete gmcpOutCount[k]);
        Object.keys(gmcpOutLast).forEach(k  => delete gmcpOutLast[k]);
    }

    // -----------------------------------------------------------------------
    // Keyboard shortcuts
    //
    // Window modules may call Client.registerShortcut(code, command) to add
    // their own bindings, e.g. Client.registerShortcut('KeyM', 'map').
    // -----------------------------------------------------------------------
    const codeShortcuts = {
        Numpad1: 'southwest', Numpad2: 'south',  Numpad3: 'southeast',
        Numpad4: 'west',      Numpad5: 'default', Numpad6: 'east',
        Numpad7: 'northwest', Numpad8: 'north',   Numpad9: 'northeast',
        F1: '=1', F2: '=2', F3: '=3',  F4: '=4',  F5: '=5',
        F6: '=6', F7: '=7', F8: '=8',  F9: '=9',  F10: '=10',
        ArrowUp: 'north', ArrowDown: 'south', ArrowLeft: 'west', ArrowRight: 'east',
    };

    function registerShortcut(code, command) {
        codeShortcuts[code] = command;
    }

    // -----------------------------------------------------------------------
    // Tab-completion (web client autocomplete)
    // -----------------------------------------------------------------------
    const _tabSug = {
        input:       '',   // the original typed text when tab was first pressed
        suggestions: [],   // full completed strings returned by the server
        index:       -1,   // which suggestion is currently displayed (-1 = none)
    };

    function _tabSugReset() {
        _tabSug.input       = '';
        _tabSug.suggestions = [];
        _tabSug.index       = -1;
    }

    // Apply suggestion at _tabSug.index to the input field, selecting the
    // appended portion so the user can backspace it away quickly.
    function _tabSugApply() {
        if (_tabSug.suggestions.length === 0) { return; }
        const sug    = _tabSug.suggestions[_tabSug.index];
        const prefix = _tabSug.input;
        textInput.value = sug;
        // Select only the suggested suffix so the user can see what was added
        // and delete it with a single backspace.
        if (sug.length > prefix.length) {
            textInput.setSelectionRange(prefix.length, sug.length);
        }
    }

    // Called from VirtualWindows.handleGMCP when the server sends Suggestion data.
    function _onSuggestionGMCP(namespace, body) {
        if (!body || !Array.isArray(body.suggestions)) { return; }
        // Only act on the response if the input still matches what we sent.
        if (body.input !== _tabSug.input) { return; }
        if (body.suggestions.length === 0) { return; }
        _tabSug.suggestions = body.suggestions;
        _tabSug.index       = 0;
        _tabSugApply();
    }

    // -----------------------------------------------------------------------
    // DOM references (resolved at init time)
    // -----------------------------------------------------------------------
    let connectButton, textOutput, textInput;

    // -----------------------------------------------------------------------
    // init()
    // -----------------------------------------------------------------------
    function init() {
        // Initialise dock slots first - VirtualWindows.openAll() depends on them.
        DockSlots.left  = new DockSlot('left');
        DockSlots.right = new DockSlot('right');

        try {
            var ah = JSON.parse(localStorage.getItem('dockAutoHide'));
            if (ah) {
                var mc = document.getElementById('main-container');
                if (typeof ah !== 'object') {
                    // Migrate old boolean format → enable both sides
                    mc.classList.add('dock-autohide-left', 'dock-autohide-right');
                    localStorage.setItem('dockAutoHide', JSON.stringify({ left: true, right: true }));
                } else {
                    if (ah.left)  { mc.classList.add('dock-autohide-left'); }
                    if (ah.right) { mc.classList.add('dock-autohide-right'); }
                }
            }
        } catch (e) {}

        connectButton = document.getElementById('connect-button');
        textOutput    = document.getElementById('terminal');
        textInput     = document.getElementById('command-input');

        // Mount terminal
        term.open(textOutput);
        window.addEventListener('resize', resizeTerminal);
        resizeTerminal();

        // Keep focus on input: click (not drag/select) in the terminal focuses the input box.
        // xterm.js manages its own selection on a canvas, so we check term.hasSelection()
        // rather than window.getSelection().
        let isDragging = false;
        textOutput.addEventListener('mousedown', () => { isDragging = false; });
        textOutput.addEventListener('mousemove', () => { isDragging = true; });
        textOutput.addEventListener('mouseup', () => {
            if (!isDragging && !term.hasSelection()) { textInput.focus(); }
            isDragging = false;
        });

        // Connect button
        connectButton.addEventListener('click', () => {
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.close();
                return;
            }
            const wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws';
            debugLog('Connecting to: ' + wsUrl);
            socket = new WebSocket(wsUrl);
            attachSocketHandlers('Connected to the server!', true);
        });

        // Input keydown
        textInput.addEventListener('keydown', function(event) {
        // Space while a suggestion is active: accept the suggestion and append
        // a space so the user can keep typing (mirrors telnet behaviour).
            if (event.key === ' ' && _tabSug.suggestions.length > 0) {
                event.preventDefault();
                const accepted = _tabSug.suggestions[_tabSug.index];
                _tabSugReset();
                textInput.value = accepted + ' ';
                textInput.setSelectionRange(textInput.value.length, textInput.value.length);
                return false;
            }

        // Tab: request or cycle autocomplete suggestions
            if (event.key === 'Tab') {
                event.preventDefault();
                const currentText = textInput.value;
                const selStart    = textInput.selectionStart;
                // The confirmed typed prefix is everything before the selection.
                const typedPrefix = currentText.substring(0, selStart);
                // If we already have suggestions for this typed prefix, cycle them.
                if (_tabSug.suggestions.length > 0 && _tabSug.input === typedPrefix) {
                    _tabSug.index = (_tabSug.index + 1) % _tabSug.suggestions.length;
                    _tabSugApply();
                } else {
                    // New request: use the typed prefix (excludes any selected suffix).
                    // Fall back to the full value if there is no selection (cursor at end).
                    const typed = typedPrefix || currentText;
                    _tabSugReset();
                    _tabSug.input = typed;
                    GMCPRequest('Suggestion', typed);
                }
                return false;
            }

            // F-key macros
            if (event.key.substring(0, 1) === 'F' && event.key.length === 2) {
                sendData('=' + event.key.substring(1));
                if (event.preventDefault) { event.preventDefault(); }
                return false;
            }

            // Command history. With the box empty the arrows walk instead
            // (the numpad shortcuts below), so Alt+Up and Alt+Down start
            // the history; once it is showing, plain arrows keep going.
            const arrowsWalk = textInput.value.length === 0 && historyPosition === 0 &&
                !event.altKey && textInput.type !== 'password' && playing();
            if ((event.key === 'ArrowUp' || event.key === 'ArrowDown') && !arrowsWalk) {
                event.preventDefault();
                historyPosition += (event.key === 'ArrowUp') ? 1 : -1;
                if (historyPosition < 0) { historyPosition = 0; }
                if (historyPosition > commandHistory.length) { historyPosition = commandHistory.length; }
                event.target.value = historyPosition === 0 ? '' : commandHistory[commandHistory.length - historyPosition];
                _tabSugReset();
                return;
            }

            // Numpad / arrow shortcuts when input is empty (arrows only in
            // play, so they never answer a login or a pending question)
            if (textInput.value.length === 0 && codeShortcuts[event.code] &&
                (event.code.indexOf('Arrow') !== 0 || playing())) {
                sendData(codeShortcuts[event.code]);
                if (event.preventDefault) { event.preventDefault(); }
                return false;
            }

            // Enter
            if (event.key === 'Enter') {
                // Accept any active suggestion before submitting
                if (_tabSug.suggestions.length > 0) {
                    _tabSugReset();
                }
                if (event.target.value !== '' && textInput.type !== 'password') {
                    commandHistory.push(event.target.value);
                    historyPosition = 0;
                    if (commandHistory.length > commandHistoryMaxLength) {
                        commandHistory = commandHistory.slice(commandHistory.length - commandHistoryMaxLength);
                    }
                }

                if (sendData(event.target.value)) {
                    event.target.value = '';
                } else {
                    term.writeln('Not connected to the server. Did you click the Connect button?');
                }
            }
        });

        // Clear tab-completion state whenever the input value changes by means
        // other than the tab handler (typing, paste, cut, etc.).
        textInput.addEventListener('input', function() {
            _tabSugReset();
        });

        // Volume sliders: load from localStorage
        _loadSoundStorage();
        const savedValues = localStorage.getItem('sliderValues');
        if (savedValues) {
            try {
                sliderValues = { ...defaultSliders, ...JSON.parse(savedValues) };
            } catch (e) {
                console.warn('Could not parse saved sliderValues, using defaults.');
            }
        } else {
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
        }

        const savedMute = localStorage.getItem('muteAllSound');
        if (savedMute) {
            try {
                document.getElementById('mute-checkbox').checked = JSON.parse(savedMute);
            } catch (e) {
                console.warn('Could not parse muteAllSound, ignoring.');
            }
        }

        buildSliders();

        const muteCheckbox = document.getElementById('mute-checkbox');
        const muteIcon     = document.getElementById('mute-icon');

        if (muteCheckbox.checked) {
            const savedUnmuted = localStorage.getItem('unmutedSliderValues');
            if (savedUnmuted) {
                try {
                    unmutedSliderValues = { ...defaultSliders, ...JSON.parse(savedUnmuted) };
                } catch (e) {
                    console.warn('Could not parse unmutedSliderValues.');
                }
            }
            Object.keys(sliderValues).forEach(k => { sliderValues[k] = 0; });
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            buildSliders();
            muteIcon.textContent = '🔇';
            MusicPlayer.setGlobalVolume(0);
        } else {
            MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);
            muteIcon.textContent = '🔊';
        }

        // Register GMCP handler for tab-completion responses
        VirtualWindows.register({
            gmcpHandlers: ['Suggestion'],
            onGMCP: function(namespace, body) {
                _onSuggestionGMCP(namespace, body);
            },
        });

        // Open all registered virtual windows immediately so they are present
        // on page load rather than waiting for the first GMCP payload.
        // Windows start in the disconnected (grayed-out) state until the
        // WebSocket connects.
        VirtualWindows.setConnected(false);
        VirtualWindows.openAll();
        if (LayoutStore.takeResetNotice()) {
            term.writeln('The web client\'s layout has changed; Settings > Reset Layout restores it at any time.');
        }
    }

    function getByPath(obj, path) {
         return path.split('.').reduce((acc, key) => {
        return acc && acc[key];
        }, obj);
    }

    function GetGMCP(path) {
        return getByPath(Client.GMCPStructs, path);
    }

    // -----------------------------------------------------------------------
    // Public surface
    // -----------------------------------------------------------------------
    return {
        // Services
        get term()         { return term; },
        get MusicPlayer()  { return MusicPlayer; },
        get SoundPlayer()  { return SoundPlayer; },

        // Shared state (read by window modules)
        get GMCPStructs()  { return GMCPStructs; },
        Playing:           playing,
        // sliderValues is a `let` that gets reassigned on mute/unmute, so the
        // getter captures the variable binding, not a snapshot of the object.
        get sliderValues() { return sliderValues; },

        // Debug toggle: set Client.debug = true from the browser console
        get debug()        { return debugOutput; },
        set debug(v)       { debugOutput = !!v; },

        // Extension points for window modules
        registerShortcut,
        onBattleEvents,
        dispatchBattleEvents: _dispatchBattleEvents, // for browser checks that feed the screen

        // Functions called from HTML event handlers
        init,
        toggleMenu,
        toggleMuteAll,
        toggleDockAutoHide,
        resetLayout,
        resetVolumeControls,
        buildSliders,
        soundHistoryFingerprint: _soundHistoryFingerprint,

        // Utility
        sendData,
        debugLog,
        SendInput,
        GMCPRequest,
        GetGMCP,
        getNetStats,
        resetNetStats,
    };
})();
