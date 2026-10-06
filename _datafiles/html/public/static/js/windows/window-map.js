/**
 * window-map.js
 *
 * Virtual window: Map (flat grid).
 *
 * Responds to GMCP namespaces:
 *   Room      - incremental update as the player moves room-to-room
 *   World.Map - bulk snapshot of all visited rooms, requested once on connect
 */

'use strict';

(function () {

    // =========================================================================
    // Shared constants
    // =========================================================================

    var ZOOM_STEP = 1.25;  // How much each zoom button click scales the view; higher = bigger jumps per click
    var ZOOM_MIN  = 0.25;  // Furthest out the user can zoom; lower = more of the map visible but smaller rooms
    var ZOOM_MAX  = 4.0;   // Closest in the user can zoom; higher = larger rooms but less of the map visible

    var ROOM_SIZE_MIN    = 10;  // Smallest room size the slider can reach
    var ROOM_SIZE_MAX    = 48;  // Largest room size the slider can reach
    var ROOM_SPACING_MIN = 4;   // Minimum center-to-center grid spacing (pixels)
    var ROOM_SPACING_MAX = 80;  // Maximum center-to-center grid spacing (pixels)

    // =========================================================================
    // Map settings (persisted to localStorage)
    // =========================================================================

    var MAP_SETTINGS_KEY = 'gomud_map_settings';

    var MAP_SETTINGS_DEFAULTS = {
        roomShape:       'square',   // 'square' | 'circle'
        roomSize:        28,         // pixels; clamped to [ROOM_SIZE_MIN, ROOM_SIZE_MAX]
        roomSpacing:     42,         // center-to-center grid distance; clamped to [ROOM_SPACING_MIN, ROOM_SPACING_MAX]
        connectionColor: '#7a4a1a',  // color of corridor lines between rooms
        mapBackground:   '#111111',  // canvas background color
        defaultZoom:     null,       // null = no default; number = zoom level to ease to on room change
        showResources:   true,       // Phase 40a: draw room-resource icons in tile corners
        sprites:         true,       // Phase 40b: your class sprite, company badge and ally sprites; off = the classic red square and hearts
        showCamp:        true,       // Phase 40b: your camp and your party's camps
    };

    // Phase 40a: room resources. Until the S1 icon sprites exist each one
    // is a small coloured dot; tooltips name them.
    var RESOURCE_INFO = {
        water:    { label: 'Fresh water', color: '#4aa3ff' },
        forage:   { label: 'Forage',      color: '#7ccf3c' },
        shelter:  { label: 'Shelter',     color: '#d1a15f' },
        herbs:    { label: 'Herbs',       color: '#3fbf8f' },
        firewood: { label: 'Firewood',    color: '#c8742c' },
        fishing:  { label: 'Fishing',     color: '#5fd0d0' },
        game:     { label: 'Game',        color: '#c85a5a' },
    };
    var RESOURCE_ICON_MAX = 3;

    function resourcesFor(roomId) {
        var info = roomInfoStore.get(roomId);
        return (info && Array.isArray(info.resources)) ? info.resources : [];
    }

    var mapSettings = (function () {
        try {
            var stored = JSON.parse(localStorage.getItem(MAP_SETTINGS_KEY) || 'null');
            if (stored && typeof stored === 'object') {
                return Object.assign({}, MAP_SETTINGS_DEFAULTS, stored);
            }
        } catch (e) { /* ignore */ }
        return Object.assign({}, MAP_SETTINGS_DEFAULTS);
    }());

    function saveMapSettings() {
        var delta = {};
        var hasAny = false;
        Object.keys(MAP_SETTINGS_DEFAULTS).forEach(function (k) {
            // null is the default for defaultZoom; only save when non-null or when different from default
            if (mapSettings[k] !== MAP_SETTINGS_DEFAULTS[k]) {
                delta[k] = mapSettings[k];
                hasAny = true;
            }
        });
        try {
            if (hasAny) {
                localStorage.setItem(MAP_SETTINGS_KEY, JSON.stringify(delta));
            } else {
                localStorage.removeItem(MAP_SETTINGS_KEY);
            }
        } catch (e) { /* ignore */ }
    }

    var CENTER_EASE_DURATION = 0.2; // How long (seconds) the camera takes to pan to a new room; 0 = instant snap, higher = slower slide

    var CURRENT_ROOM_COLOR      = '#c20000'; // Fill color of the room the player is currently in; change to adjust how much it stands out
    var CURRENT_ROOM_TEXT_COLOR = '#ffffff'; // Symbol character color inside the current room; should contrast with CURRENT_ROOM_COLOR
    var SYMBOL_TEXT_COLOR       = '#e0e0e0'; // Symbol character color inside all non-current rooms; lower contrast = subtler symbols


    // =========================================================================
    // Shared helpers
    // =========================================================================

    function symbolForRoom(info) {
        if (info.mapsymbol) { return info.mapsymbol; }
        return '\u2022';
    }

    /**
     * Returns the fill color for a room square.
     * Cascade (mirrors admin mapper):
     *   1. per-symbol bg override for this biome
     *   2. per-symbol fg override for this biome
     *   3. biome bg color
     *   4. biome fg color
     *   5. default room color
     */
    function colorForSymbol(sym, biomeId) {
        var b = biomeTable[biomeId];
        if (!b) { return '#3a3a4a'; }
        if (sym && b.overrides && b.overrides[sym]) {
            if (b.overrides[sym].bg) { return b.overrides[sym].bg; }
            if (b.overrides[sym].fg) { return b.overrides[sym].fg; }
        }
        if (b.color) {
            if (b.color.bg) { return b.color.bg; }
            if (b.color.fg) { return b.color.fg; }
        }
        return '#3a3a4a';
    }

    /**
     * Returns '#ffffff' or '#000000', whichever contrasts better against
     * the given CSS hex fill color.
     */
    function contrastColor(hex) {
        var r = parseInt(hex.slice(1, 3), 16);
        var g = parseInt(hex.slice(3, 5), 16);
        var b = parseInt(hex.slice(5, 7), 16);
        return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.45 ? '#000000' : '#ffffff';
    }

    function smoothstep(t) {
        return t * t * (3 - 2 * t);
    }

    // =========================================================================
    // Shared data pipeline
    // =========================================================================

    /** Full GMCP info objects keyed by roomId - used by the view for tooltips. */
    var roomInfoStore = new Map();

    /**
     * partyMemberPositions: keyed by member name -> { x, y, z }
     * Updated whenever Party or Party.Vitals GMCP arrives.
     */
    var partyMemberPositions = {};

    /**
     * biomeTable: populated from World.Map payload.
     * keyed by biomeId -> { name, symbol, color: {fg, bg}, overrides: {sym: {fg, bg}} }
     */
    var biomeTable = {};

    /**
     * roomCache: keyed by roomId.
     * { RoomId, zoneName, x, y, z, symbol, env, exits, stubs, hasUp, hasDown }
     */
    var roomCache = {};

    var worldMapRequested = false;

    function upsertRoomCache(id, zoneName, gx, gy, gz, sym, env, exitsv2) {
        var exitIds   = [];
        var exitStubs = [];
        var hasUp     = false;
        var hasDown   = false;

        if (exitsv2) {
            for (var dir in exitsv2) {
                var exitInfo = exitsv2[dir];

                if (exitInfo.dz > 0) { hasUp   = true; }
                if (exitInfo.dz < 0) { hasDown = true; }

                if (exitInfo.dx === 0 && exitInfo.dy === 0 && exitInfo.dz === 0) { continue; }

                var isSecret    = Array.isArray(exitInfo.details) && exitInfo.details.indexOf('secret') !== -1;
                var isLocked    = Array.isArray(exitInfo.details) && exitInfo.details.indexOf('locked') !== -1;
                var destVisited = roomInfoStore.has(exitInfo.num);

                if (isSecret && !destVisited) { continue; }

                if (destVisited) {
                    exitIds.push({ num: exitInfo.num, locked: isLocked, secret: isSecret, dz: exitInfo.dz });
                } else {
                    exitStubs.push({ dx: exitInfo.dx, dy: exitInfo.dy, dz: exitInfo.dz, locked: isLocked, secret: isSecret });
                }
            }
        }

        roomCache[id] = {
            RoomId:   id,
            zoneName: zoneName,
            x: gx, y: gy, z: gz,
            symbol:   sym,
            env:      env,
            exits:    exitIds,
            stubs:    exitStubs,
            hasUp:    hasUp,
            hasDown:  hasDown,
        };
    }

    function ingestWorldMap(payload) {
        // Support both the legacy bare-array shape and the new {rooms, biomes} shape.
        var entries = Array.isArray(payload) ? payload : (payload && payload.rooms ? payload.rooms : []);
        var newBiomes = (!Array.isArray(payload) && payload && payload.biomes) ? payload.biomes : {};

        Object.keys(newBiomes).forEach(function (id) { biomeTable[id] = newBiomes[id]; });

        if (!Array.isArray(entries) || entries.length === 0) { return; }

        entries.forEach(function (info) {
            if (info.num) { roomInfoStore.set(info.num, info); }
        });

        entries.forEach(function (info) {
            var id = info.num;
            if (!id) { return; }
            var coords = info.coords ? info.coords.split(',').map(function (s) { return s.trim(); }) : null;
            if (!coords || coords.length < 4) { return; }
            var zoneName = coords[0];
            var gx = parseInt(coords[1], 10);
            var gy = parseInt(coords[2], 10);
            var gz = parseInt(coords[3], 10);
            var isZoneRoot = Array.isArray(info.details) && info.details.indexOf('root') !== -1;
            if (gx === 0 && gy === 0 && !isZoneRoot) { return; }
            upsertRoomCache(id, zoneName, gx, gy, gz, symbolForRoom(info), info.environment || '', info.exitsv2);
        });

        view2d.onWorldMap();
    }

    // =========================================================================
    // Shared tooltip
    // =========================================================================

    var tooltip          = null;
    var tooltipHideTimer = null;

    function ensureTooltip() {
        if (tooltip) { return; }
        tooltip = document.createElement('div');
        tooltip.id = 'map-tooltip';
        document.body.appendChild(tooltip);
    }

    function showTooltip(mouseX, mouseY, info) {
        ensureTooltip();
        clearTimeout(tooltipHideTimer);

        var html = '<div class="tt-name">' + (info.name || 'Unknown') + '</div>';
        var rows = [];
        var envDisplay = info.environment
            ? ((biomeTable[info.environment] && biomeTable[info.environment].name) || info.environment)
            : null;
        if (envDisplay)        { rows.push({ label: 'Env',    value: envDisplay      }); }
        if (info.maplegend)    { rows.push({ label: 'Type',   value: info.maplegend  }); }
        if (info.mapsymbol)    { rows.push({ label: 'Symbol', value: info.mapsymbol  }); }
        if (info.area)         { rows.push({ label: 'Area',   value: info.area       }); }
        if (rows.length > 0) {
            html += '<hr class="tt-divider">';
            rows.forEach(function (r) {
                html += '<div class="tt-row"><span class="tt-label">' + r.label +
                        '</span><span class="tt-value">' + r.value + '</span></div>';
            });
        }

        if (Array.isArray(info.resources) && info.resources.length > 0) {
            var resNames = info.resources.map(function (r) {
                return (RESOURCE_INFO[r] && RESOURCE_INFO[r].label) || r;
            });
            html += '<hr class="tt-divider"><div class="tt-row">' +
                    '<span class="tt-label">Here</span>' +
                    '<span class="tt-value">' + resNames.join(', ') + '</span></div>';
        }

        var details    = info.details || [];
        var badgeOrder = ['pvp', 'bank', 'trainer', 'storage', 'character', 'ephemeral'];
        var badges     = badgeOrder.filter(function (d) { return details.indexOf(d) !== -1; });
        if (badges.length > 0) {
            html += '<hr class="tt-divider"><div class="tt-badges">';
            badges.forEach(function (b) { html += '<span class="tt-badge ' + b + '">' + b + '</span>'; });
            html += '</div>';
        }

        if (info.exitsv2) {
            var exitNames = Object.keys(info.exitsv2).filter(function (dir) {
                var e = info.exitsv2[dir];
                return !(Array.isArray(e.details) && e.details.indexOf('secret') !== -1) ||
                       roomInfoStore.has(e.num);
            }).sort();
            if (exitNames.length > 0) {
                html += '<hr class="tt-divider"><div class="tt-row">' +
                        '<span class="tt-label">Exits</span>' +
                        '<span class="tt-value">' + exitNames.join(', ') + '</span></div>';
            }
        }

        var partyHere = [];
        var hoveredRc = roomCache[info.num];
        if (hoveredRc) {
            Object.keys(partyMemberPositions).forEach(function (name) {
                var pos = partyMemberPositions[name];
                if (pos.x === hoveredRc.x && pos.y === hoveredRc.y && pos.z === hoveredRc.z) {
                    partyHere.push(name);
                }
            });
        }
        if (partyHere.length > 0) {
            html += '<hr class="tt-divider"><div class="tt-row">' +
                    '<span class="tt-label">Party</span>' +
                    '<span class="tt-value" style="color:#ff6666">\u2665 ' + partyHere.join(', ') + '</span></div>';
        }

        tooltip.innerHTML     = html;
        tooltip.style.display = 'block';
        positionTooltip(mouseX, mouseY);
    }

    function positionTooltip(mouseX, mouseY) {
        if (!tooltip) { return; }
        var ttW  = tooltip.offsetWidth;
        var ttH  = tooltip.offsetHeight;
        var vw   = window.innerWidth;
        var vh   = window.innerHeight;
        var left = mouseX + 14;
        if (left + ttW > vw - 8) { left = mouseX - ttW - 14; }
        left = Math.max(8, left);
        var top = mouseY - Math.floor(ttH / 2);
        if (top + ttH > vh - 8) { top = vh - ttH - 8; }
        top = Math.max(8, top);
        tooltip.style.left = left + 'px';
        tooltip.style.top  = top  + 'px';
    }

    function hideTooltip() {
        tooltipHideTimer = setTimeout(function () {
            if (tooltip) { tooltip.style.display = 'none'; }
        }, 80);
    }

    // =========================================================================
    // Styles
    // =========================================================================

    injectStyles([
        '#map-window {',
        '    display: flex;',
        '    flex-direction: column;',
        '    width: 100%;',
        '    height: 100%;',
        '    background: var(--t-bg-panel);',
        '}',
        '#map-panels {',
        '    flex: 1;',
        '    position: relative;',
        '    overflow: hidden;',
        '}',
        '.map-canvas-wrap {',
        '    width: 100%;',
        '    height: 100%;',
        '    position: relative;',
        '    overflow: hidden;',
        '}',
        '.map-canvas-wrap canvas {',
        '    display: block;',
        '    position: absolute;',
        '    top: 0; left: 0;',
        '    cursor: grab;',
        '}',
        '#map-tooltip {',
        '    position: fixed;',
        '    z-index: 99999;',
        '    pointer-events: none;',
        '    background: var(--t-bg-surface);',
        '    border: 1px solid var(--t-accent-dim);',
        '    border-radius: 6px;',
        '    box-shadow: 0 4px 16px rgba(0,0,0,0.7);',
        '    padding: 8px 10px;',
        '    min-width: 140px;',
        '    max-width: 240px;',
        '    display: none;',
        '    font-family: monospace;',
        '}',
        '#map-tooltip .tt-name { font-size:0.85em; font-weight:bold; color:var(--t-text); margin-bottom:4px; line-height:1.3; }',
        '#map-tooltip .tt-divider { border:none; border-top:1px solid var(--t-accent-dim); margin:5px 0; }',
        '#map-tooltip .tt-row { display:flex; justify-content:space-between; align-items:baseline; gap:8px; font-size:0.75em; line-height:1.6; }',
        '#map-tooltip .tt-label { color:var(--t-text-secondary); text-transform:uppercase; letter-spacing:0.04em; font-size:0.88em; flex-shrink:0; }',
        '#map-tooltip .tt-value { color:var(--t-text); text-align:right; }',
        '#map-tooltip .tt-badges { display:flex; flex-wrap:wrap; gap:3px; margin-top:4px; }',
        '#map-tooltip .tt-badge { font-size:0.62em; padding:1px 4px; border-radius:3px; background:var(--t-map-badge-bg); color:var(--t-text-secondary); border:1px solid var(--t-accent-dim); }',
        '#map-tooltip .tt-badge.pvp     { background:var(--t-badge-pvp-bg); color:var(--t-badge-pvp-text); border-color:var(--t-badge-pvp-border); }',
        '#map-tooltip .tt-badge.bank    { background:var(--t-badge-bank-bg); color:var(--t-badge-bank-text); border-color:var(--t-badge-bank-border); }',
        '#map-tooltip .tt-badge.trainer { background:var(--t-badge-trainer-bg); color:var(--t-badge-trainer-text); border-color:var(--t-badge-trainer-border); }',
        '#map-tooltip .tt-badge.storage { background:var(--t-badge-storage-bg); color:var(--t-badge-storage-text); border-color:var(--t-badge-storage-border); }',
        '.map-controls {',
        '    position: absolute;',
        '    top: 6px;',
        '    right: 6px;',
        '    display: flex;',
        '    align-items: center;',
        '    gap: 2px;',
        '    z-index: 10;',
        '}',
        '.map-controls button {',
        '    width: 22px; height: 22px;',
        '    padding: 0;',
        '    font-size: 14px;',
        '    line-height: 1;',
        '    background: var(--t-map-controls-bg);',
        '    color: var(--t-map-controls-text);',
        '    border: 1px solid var(--t-map-controls-border);',
        '    border-radius: 3px;',
        '    cursor: pointer;',
        '}',
        '.map-controls button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); }',
        '.map-controls button.active { background: var(--t-map-controls-active); color: var(--t-text-white); }',
        '.map-settings-panel {',
        '    position: absolute;',
        '    top: 32px;',
        '    right: 6px;',
        '    z-index: 20;',
        '    background: var(--t-bg-surface);',
        '    border: 1px solid var(--t-accent-dim);',
        '    border-radius: 5px;',
        '    box-shadow: 0 4px 16px rgba(0,0,0,0.65);',
        '    padding: 8px 10px;',
        '    min-width: 160px;',
        '    display: flex;',
        '    flex-direction: column;',
        '    gap: 6px;',
        '}',
        '.msp-row {',
        '    display: flex;',
        '    align-items: center;',
        '    justify-content: space-between;',
        '    gap: 8px;',
        '}',
        '.msp-label {',
        '    font-size: 0.75em;',
        '    color: var(--t-text-secondary);',
        '    text-transform: uppercase;',
        '    letter-spacing: 0.04em;',
        '    flex-shrink: 0;',
        '}',
        '.msp-btngroup { display:flex; gap:2px; }',
        '.msp-btngroup button { padding:2px 7px; font-size:0.72em; line-height:1.5; background:var(--t-map-controls-bg); color:var(--t-map-controls-text); border:1px solid var(--t-map-controls-border); border-radius:3px; cursor:pointer; white-space:nowrap; }',
        '.msp-btngroup button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); }',
        '.msp-btngroup button.active { background: var(--t-map-controls-active); color: var(--t-text-white); border-color: var(--t-map-controls-active); }',
        '.msp-slider { flex: 1; min-width: 80px; cursor: pointer; accent-color: var(--t-map-controls-active); }',
        '.msp-color { width: 32px; height: 20px; padding: 0; border: 1px solid var(--t-map-controls-border); border-radius: 3px; cursor: pointer; background: none; }',
        '.msp-reset { margin-top: 2px; align-self: flex-end; font-size: 0.68em; padding: 1px 6px; background: none; color: var(--t-text-secondary); border: 1px solid var(--t-accent-dim); border-radius: 3px; cursor: pointer; line-height: 1.6; }',
        '.msp-reset:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); border-color: var(--t-map-controls-hover); }',
    ].join('\n'));

    // =========================================================================
    // 2D view
    // =========================================================================

    var view2d = (function () {

        // -- Constants ---------------------------------------------------------
        var ROOM_GRID_STEP = 42;  // fallback only; runtime reads mapSettings.roomSpacing

        function getRoomSize()  { return Math.round(mapSettings.roomSize); }
        function getBaseStep()  { return Math.round(mapSettings.roomSpacing); }

        var CONNECTION_WIDTH = 4;    // Stroke width of corridor lines; thicker = more visible connections, can obscure small rooms
        var ROOM_BORDER_WIDTH = 1.5; // Stroke width of the outline drawn around each room square; higher = bolder room edges
        var SYMBOL_FONT_SIZE  = 14;  // Font size of the symbol character drawn inside each room; larger = more readable but may overflow small rooms
        var MAP_BACKGROUND    = '#111111'; // fallback only; runtime reads mapSettings.mapBackground
        var ROOM_BORDER_COLOR = '#000000'; // Outline color drawn around each room square; darker = crisper separation between rooms

        // -- State -------------------------------------------------------------
        var canvas        = null;
        var ctx           = null;
        var container     = null;
        var rooms         = new Map();
        var edges         = new Map();
        var zoneExitStubs = [];
        var currentRoomId = null;
        var cameraX = 0, cameraY = 0;
        var easeStartX = 0, easeStartY = 0;
        var easeTargetX = 0, easeTargetY = 0;
        var easeStartTime = null, easeRafId = null;
        var panOffsetX = 0, panOffsetY = 0;
        var dragActive = false;
        var dragStartPxX = 0, dragStartPxY = 0;
        var dragStartPanX = 0, dragStartPanY = 0;
        var zoomScale     = 1.0;
        var currentZoneKey = '';
        var partyPositions = {}; // name -> { x, y, z }

        // Per-member heart ease state.
        // keyed by member name -> { fromGx, fromGy, toGx, toGy, startTime }
        // fromGx/fromGy are the grid coords the heart is easing FROM.
        // When a member has no previous position, fromGx/fromGy equal toGx/toGy (no animation).
        var HEART_EASE_DURATION = 0.5; // seconds
        var partyHeartEase = {}; // name -> { fromGx, fromGy, toGx, toGy, toZ, startTime }
        var heartRafId = null;


        // -- Phase 40b: sprites, badge, camps ------------------------------------
        var WALK_STEP_MS    = 200;  // time the sprite takes per tile; matches CENTER_EASE_DURATION
        var WALK_QUEUE_MAX  = 2;    // steps the sprite may trail the player before it snaps
        var FADE_MS         = 250;  // fade-in after a level (z) or zone change
        var ALLY_SCALE      = 0.75; // allied class sprites at 75% of the player's
        var identity    = { classid: '', lineage: '' }; // Char.Info
        var companySize = 0;        // members with the leader, 0 when alone or unknown
        var campInfo    = null;     // Company.Camp
        var unit = {
            x: null, y: null,       // grid position drawn (eased)
            targetX: null, targetY: null,
            zoneKey: '',
            face: 'down', flip: false,
            step: null,             // { fromX, fromY, toX, toY, face, flip, start }
            queue: [],
            fadeStart: -1,
        };
        var animTimer = null;

        function faceOf(dx, dy, prev) {
            if (dx === 0 && dy === 0) { return prev || { face: 'down', flip: false }; }
            if (dx === 0) { return { face: dy < 0 ? 'up' : 'down', flip: false }; }
            return { face: 'side', flip: dx < 0 }; // sprites face right; mirror for west; diagonals use side
        }

        // chainKeys is the sprite folder order for a character: current
        // class, lineage, then the adventurer.
        function chainKeys(classid, lineage) {
            var keys = [];
            [classid, lineage, 'adventurer'].forEach(function (k) {
                if (k && keys.indexOf(k) === -1) { keys.push(k); }
            });
            return keys;
        }

        // resolveSheet walks the fallback chain. It returns null while the
        // wanted image is still loading (nothing flashes in a worse sprite)
        // and when every image is missing or failed (the caller then draws
        // the classic marker).
        function resolveSheet(keys, walking) {
            for (var i = 0; i < keys.length; i++) {
                var idle = 'map/units/' + keys[i] + '/idle.png';
                var walk = 'map/units/' + keys[i] + '/walk.png';
                var st = Sprites.status(idle);
                if (st === 'ready') {
                    if (Sprites.status(walk) === 'ready' && walking) { return { sheet: Sprites.art(walk), walk: true }; }
                    return { sheet: Sprites.art(idle), walk: false };
                }
                if (st === 'loading') { return null; }
            }
            return null;
        }

        // spriteMult is the whole-number multiple of 32 px nearest the tile
        // size, so pixels stay square; zoomed far out it halves (16 px).
        function spriteMult(tilePx) {
            return tilePx < 16 ? 0.5 : Math.max(1, Math.round(tilePx / 32));
        }

        function drawFrame(sheet, row, flip, cx, feetY, mult, now, startMs, alpha) {
            var info = sheet.info;
            var r = Math.max(0, (info.rows || []).indexOf(row));
            var f = Sprites.frame(info, r, now, startMs);
            var w = f.sw * mult, h = f.sh * mult;
            var x = Math.round(cx - w / 2);
            var y = Math.round(feetY - h * ((info.feet_baseline || f.sh) / f.sh));
            ctx.save();
            ctx.imageSmoothingEnabled = false;
            ctx.globalAlpha = alpha;
            if (flip) {
                ctx.translate(x + w, 0);
                ctx.scale(-1, 1);
                ctx.drawImage(sheet.img, f.sx, f.sy, f.sw, f.sh, 0, y, w, h);
            } else {
                ctx.drawImage(sheet.img, f.sx, f.sy, f.sw, f.sh, x, y, w, h);
            }
            ctx.restore();
        }

        // drawIcon draws a centered, anchor-less marker (ring, badge, camp
        // pieces) at <mult>x. It reports whether the image was ready.
        function drawIcon(path, cx, cy, mult, now) {
            var a = Sprites.art(path);
            if (!a) { return false; }
            var f = Sprites.frame(a.info, 0, now, 0);
            var w = f.sw * mult, h = f.sh * mult;
            ctx.save();
            ctx.imageSmoothingEnabled = false;
            ctx.drawImage(a.img, f.sx, f.sy, f.sw, f.sh, Math.round(cx - w / 2), Math.round(cy - h / 2), w, h);
            ctx.restore();
            return true;
        }

        function spritesOn() { return mapSettings.sprites !== false; }

        // unitMoveTo records the player's move: a one-tile step walks (facing
        // the way it went) and queues at most WALK_QUEUE_MAX behind; a jump
        // of more than a tile (recall, teleport) and a faster pile-up snap;
        // a level or zone change snaps and fades in, keeping the facing.
        function unitMoveTo(gx, gy, zoneKey, now) {
            if (unit.x === null) {
                unit.x = unit.targetX = gx; unit.y = unit.targetY = gy; unit.zoneKey = zoneKey;
                return;
            }
            if (zoneKey !== unit.zoneKey) {
                unit.zoneKey = zoneKey;
                unitSnap(gx, gy);
                unit.fadeStart = now;
                return;
            }
            var dx = gx - unit.targetX, dy = gy - unit.targetY;
            if (dx === 0 && dy === 0) { return; }
            if (Math.max(Math.abs(dx), Math.abs(dy)) > 1) { unitSnap(gx, gy); return; }
            var f = faceOf(dx, dy);
            unit.queue.push({ x: gx, y: gy, face: f.face, flip: f.flip });
            unit.targetX = gx; unit.targetY = gy;
            if (unit.queue.length > WALK_QUEUE_MAX) { unitSnap(gx, gy); }
        }

        function unitSnap(gx, gy) {
            unit.x = unit.targetX = gx; unit.y = unit.targetY = gy;
            unit.queue = [];
            unit.step = null;
        }

        // unitPose advances the walk to <now> and returns where the sprite
        // is and whether it is walking.
        function unitPose(now) {
            var chainStart = null;
            while (true) {
                if (!unit.step && unit.queue.length) {
                    var n = unit.queue.shift();
                    unit.step = { fromX: unit.x, fromY: unit.y, toX: n.x, toY: n.y,
                                  start: chainStart !== null ? chainStart : now };
                    unit.face = n.face; unit.flip = n.flip;
                }
                if (!unit.step) { break; }
                var t = (now - unit.step.start) / WALK_STEP_MS;
                if (t >= 1) {
                    unit.x = unit.step.toX; unit.y = unit.step.toY;
                    chainStart = unit.step.start + WALK_STEP_MS;
                    unit.step = null;
                    continue;
                }
                return {
                    x: unit.step.fromX + (unit.step.toX - unit.step.fromX) * t,
                    y: unit.step.fromY + (unit.step.toY - unit.step.fromY) * t,
                    walking: true, start: unit.step.start,
                };
            }
            return { x: unit.x, y: unit.y, walking: false, start: 0 };
        }

        function fireAndRest(cx, cy, mult, lit, resting, now) {
            var fx = cx + 8 * mult, fy = cy + 8 * mult;
            if (lit) {
                drawIcon('map/camp/fire-lit.png', fx, fy, mult, now);
                drawIcon('map/camp/smoke.png', fx, fy - 12 * mult, mult, now);
            } else {
                drawIcon('map/camp/fire-unlit.png', fx, fy, mult, now);
            }
            if (resting) { drawIcon('map/camp/resting.png', cx - 8 * mult, cy - 12 * mult, mult, now); }
        }

        function drawCamp(roomId, ally, lit, resting, innRest, now) {
            var r = rooms.get(roomId);
            if (!r) { return; }
            var p = gridToCanvas(r.x, r.y);
            var mult = spriteMult(getRoomSize() * zoomScale);
            if (innRest) {
                drawIcon('map/camp/inn-rest.png', p.px, p.py, mult, now);
                return;
            }
            var tent = ally ? 'map/camp/tent-ally.png' : 'map/camp/tent.png';
            if (!drawIcon(tent, p.px, p.py, mult, now)) {
                // no art yet: a small tent triangle
                var q = getRoomSize() * zoomScale * 0.4;
                ctx.fillStyle = ally ? '#6a9ec9' : '#c9a15a';
                ctx.beginPath(); ctx.moveTo(p.px, p.py - q); ctx.lineTo(p.px + q, p.py + q); ctx.lineTo(p.px - q, p.py + q); ctx.closePath(); ctx.fill();
            }
            fireAndRest(p.px, p.py, mult, lit, resting, now);
        }

        function drawCamps(now) {
            if (!campInfo || mapSettings.showCamp === false) { return; }
            (campInfo.allied_camps || []).forEach(function (c) {
                drawCamp(c.room_id, true, !!c.fire_lit, !!c.resting, false, now);
            });
            if (campInfo.has_camp && campInfo.room_id) {
                var inn = !!(campInfo.here && campInfo.inn && campInfo.resting);
                drawCamp(campInfo.room_id, false, !!campInfo.fire_lit, !!campInfo.resting, inn, now);
            }
        }

        // drawUnit draws the player's marker above the terrain: the here-ring,
        // the class sprite, then the company badge. It returns false when
        // there is no sprite to draw (off, still loading, or no art), so the
        // caller shows the classic red square.
        function drawUnit(now) {
            if (!spritesOn() || currentRoomId === null || unit.x === null) { return false; }
            var pose = unitPose(now);
            var res = resolveSheet(chainKeys(identity.classid, identity.lineage), pose.walking);
            if (!res) { return false; }
            var tile = getRoomSize() * zoomScale;
            var mult = spriteMult(tile);
            var p = gridToCanvas(pose.x, pose.y);
            var alpha = 1;
            if (unit.fadeStart >= 0) {
                alpha = Math.min(1, (now - unit.fadeStart) / FADE_MS);
                if (alpha >= 1) { unit.fadeStart = -1; }
            }
            ctx.save();
            ctx.globalAlpha = alpha;
            if (!drawIcon('map/markers/here-ring.png', p.px, p.py + 8 * mult, mult, now)) {
                ctx.strokeStyle = '#ffd24a'; ctx.lineWidth = Math.max(1, mult * 2);
                ctx.strokeRect(p.px - tile / 2, p.py - tile / 2, tile, tile);
            }
            ctx.restore();
            drawFrame(res.sheet, unit.face, unit.flip, p.px, p.py + 12 * mult, mult, now, pose.walking ? pose.start : 0, alpha);
            if (companySize > 1) {
                var bx = p.px + 9 * mult, by = p.py - 12 * mult;
                var bw = 12 * mult;
                if (!drawIcon('map/markers/company-badge.png', bx, by, mult, now)) {
                    ctx.fillStyle = '#8a2a2a'; ctx.beginPath(); ctx.arc(bx, by, bw / 2, 0, Math.PI * 2); ctx.fill();
                }
                ctx.fillStyle = '#ffffff';
                ctx.font = 'bold ' + Math.round(8 * mult) + 'px monospace';
                ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText(String(companySize), bx, by);
            }
            return { walking: pose.walking };
        }

        // drawAllySprite draws a party member as a class sprite at 75% with
        // an ally pennant. It returns false when there is no art to draw, so
        // the caller keeps the heart.
        function drawAllySprite(ease, gx, gy, moving, now) {
            if (!ease.classid && !ease.lineage) { return false; }
            var res = resolveSheet(chainKeys(ease.classid, ease.lineage), moving);
            if (!res) { return false; }
            var p = gridToCanvas(gx, gy);
            var mult = spriteMult(getRoomSize() * zoomScale) * ALLY_SCALE;
            drawFrame(res.sheet, ease.face || 'down', !!ease.flip, p.px, p.py + 12 * mult, mult, now, ease.startTime, 1);
            drawIcon('map/markers/ally-banner.png', p.px + 9 * mult, p.py - 10 * mult, mult, now);
            return true;
        }

        function scheduleAnim(walking) {
            if (!container || !container.isConnected || animTimer !== null) { return; }
            if (walking) {
                animTimer = requestAnimationFrame(function () { animTimer = null; render(); });
            } else {
                animTimer = setTimeout(function () { animTimer = null; render(); }, 120);
            }
        }

        // -- Helpers -----------------------------------------------------------
        function resizeCanvas() {
            if (!canvas || !container) { return; }
            canvas.width  = container.clientWidth  || 1;
            canvas.height = container.clientHeight || 1;
        }

        function gridToCanvas(gx, gy) {
            var midX = Math.floor(canvas.width  / 2);
            var midY = Math.floor(canvas.height / 2);
            var step = getBaseStep() * zoomScale;
            return {
                px: midX + (gx - cameraX - panOffsetX) * step,
                py: midY + (gy - cameraY - panOffsetY) * step,
            };
        }

        function setCameraTarget(tx, ty) {
            panOffsetX = 0; panOffsetY = 0;
            var targetZoom = (mapSettings.defaultZoom !== null) ? mapSettings.defaultZoom : zoomScale;
            if (CENTER_EASE_DURATION <= 0) {
                cameraX = tx; cameraY = ty;
                zoomScale = targetZoom;
                render(); return;
            }
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
            easeStartX = cameraX; easeStartY = cameraY;
            easeTargetX = tx; easeTargetY = ty;
            var easeStartZoom = zoomScale;
            easeStartTime = null;
            function step(ts) {
                if (easeStartTime === null) { easeStartTime = ts; }
                var t = Math.min((ts - easeStartTime) / 1000 / CENTER_EASE_DURATION, 1);
                var s = smoothstep(t);
                cameraX = easeStartX + (easeTargetX - easeStartX) * s;
                cameraY = easeStartY + (easeTargetY - easeStartY) * s;
                zoomScale = easeStartZoom + (targetZoom - easeStartZoom) * s;
                render();
                easeRafId = t < 1 ? requestAnimationFrame(step) : null;
            }
            easeRafId = requestAnimationFrame(step);
        }

        function addOrUpdateRoom(id, gx, gy, symbol, env) {
            var rc = roomCache[id];
            rooms.set(id, { x: gx, y: gy, symbol: symbol || '\u2022', env: env || '',
                            hasUp: rc ? rc.hasUp : false, hasDown: rc ? rc.hasDown : false });
        }

        function addEdge(idA, idB, locked, secret) {
            var key = idA < idB ? (idA + '-' + idB) : (idB + '-' + idA);
            if (!edges.has(key)) {
                edges.set(key, { locked: !!locked, secret: !!secret });
            } else {
                var ex = edges.get(key);
                ex.locked = ex.locked || !!locked;
                ex.secret = ex.secret || !!secret;
            }
        }

        function resetMap() {
            rooms.clear(); edges.clear(); zoneExitStubs = [];
            currentRoomId = null;
            cameraX = 0; cameraY = 0; panOffsetX = 0; panOffsetY = 0;
            dragActive = false;
            if (easeRafId !== null) { cancelAnimationFrame(easeRafId); easeRafId = null; }
        }

        function replayZone(zoneKey) {
            resetMap();
            var zMatch = zoneKey.match(/\/z:(-?\d+)$/);
            if (!zMatch) { return; }
            var targetZ = parseInt(zMatch[1], 10);
            var visited = {}, queue = [];
            for (var rid in roomCache) {
                var rc = roomCache[rid];
                if (rc.z === targetZ && rc.zoneName + '/z:' + rc.z === zoneKey) {
                    queue.push(parseInt(rid, 10));
                }
            }
            while (queue.length > 0) {
                var id = queue.shift();
                if (visited[id]) { continue; }
                visited[id] = true;
                var r = roomCache[id];
                if (!r) { continue; }
                addOrUpdateRoom(r.RoomId, r.x, r.y, r.symbol, r.env);
                if (Array.isArray(r.exits)) {
                    r.exits.forEach(function (exit) {
                        if (exit.dz === 0 && !visited[exit.num] && roomCache[exit.num]) {
                            queue.push(exit.num);
                        }
                    });
                }
            }
            rooms.forEach(function (room, id) {
                var r = roomCache[id];
                if (!r) { return; }
                if (Array.isArray(r.exits)) {
                    r.exits.forEach(function (exit) {
                        if (exit.dz === 0 && rooms.has(exit.num)) {
                            addEdge(id, exit.num, exit.locked, exit.secret);
                        }
                    });
                }
                if (Array.isArray(r.stubs)) {
                    r.stubs.forEach(function (stub) {
                        if (stub.dz === 0) {
                            zoneExitStubs.push({ roomId: id, dx: stub.dx, dy: stub.dy,
                                                 locked: stub.locked, secret: stub.secret });
                        }
                    });
                }
            });
        }

        // -- Rendering ---------------------------------------------------------
        function drawLineBadge(mx, my, type) {
            var sz = Math.max(7, Math.round(CONNECTION_WIDTH * zoomScale * 2.5));
            var half = sz / 2;
            ctx.save();
            ctx.fillStyle = mapSettings.mapBackground;
            ctx.fillRect(mx - half, my - half, sz, sz);
            if (type === 'secret') {
                ctx.fillStyle = '#d4a843';
                ctx.font = 'bold ' + Math.round(sz * 0.85) + 'px monospace';
                ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText('?', mx, my);
            } else {
                var kc = '#9ab0d4', lw = Math.max(1, sz * 0.14);
                ctx.strokeStyle = kc; ctx.fillStyle = kc;
                ctx.lineWidth = lw; ctx.lineCap = 'round';
                var bowR = sz * 0.22, bowCx = mx - sz * 0.14, bowCy = my;
                ctx.beginPath(); ctx.arc(bowCx, bowCy, bowR, 0, Math.PI * 2); ctx.stroke();
                var shaftX1 = bowCx + bowR, shaftX2 = mx + half * 0.82;
                ctx.beginPath(); ctx.moveTo(shaftX1, bowCy); ctx.lineTo(shaftX2, bowCy); ctx.stroke();
                var toothH = sz * 0.18;
                var t1x = shaftX1 + (shaftX2 - shaftX1) * 0.45;
                var t2x = shaftX1 + (shaftX2 - shaftX1) * 0.72;
                ctx.beginPath();
                ctx.moveTo(t1x, bowCy); ctx.lineTo(t1x, bowCy + toothH);
                ctx.moveTo(t2x, bowCy); ctx.lineTo(t2x, bowCy + toothH);
                ctx.stroke();
            }
            ctx.restore();
        }

        function render() {
            if (!ctx || !canvas) { return; }
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = mapSettings.mapBackground;
            ctx.fillRect(0, 0, canvas.width, canvas.height);

            var ROOM_SIZE = getRoomSize();
            var BASE_STEP = getBaseStep();
            var useCircle = (mapSettings.roomShape === 'circle');

            ctx.strokeStyle = mapSettings.connectionColor;
            ctx.lineWidth   = CONNECTION_WIDTH * zoomScale;
            ctx.lineCap     = 'round';

            edges.forEach(function (flags, key) {
                var parts = key.split('-');
                var rA = rooms.get(parseInt(parts[0], 10));
                var rB = rooms.get(parseInt(parts[1], 10));
                if (!rA || !rB) { return; }
                var pA = gridToCanvas(rA.x, rA.y), pB = gridToCanvas(rB.x, rB.y);
                ctx.beginPath(); ctx.moveTo(pA.px, pA.py); ctx.lineTo(pB.px, pB.py); ctx.stroke();
                if (flags.locked || flags.secret) {
                    drawLineBadge((pA.px + pB.px) / 2, (pA.py + pB.py) / 2,
                                  flags.secret ? 'secret' : 'key');
                }
            });

            var stubLen = BASE_STEP * zoomScale * 0.55;
            zoneExitStubs.forEach(function (stub) {
                var r = rooms.get(stub.roomId);
                if (!r) { return; }
                var p = gridToCanvas(r.x, r.y);
                var len = Math.sqrt(stub.dx * stub.dx + stub.dy * stub.dy);
                if (len === 0) { return; }
                var ex = p.px + (stub.dx / len) * stubLen;
                var ey = p.py + (stub.dy / len) * stubLen;
                ctx.beginPath(); ctx.moveTo(p.px, p.py); ctx.lineTo(ex, ey); ctx.stroke();
                if (stub.locked || stub.secret) {
                    drawLineBadge((p.px + ex) / 2, (p.py + ey) / 2, stub.secret ? 'secret' : 'key');
                }
            });

            var nowMs        = performance.now();
            var scaledSize   = ROOM_SIZE        * zoomScale;
            var scaledBorder = ROOM_BORDER_WIDTH * zoomScale;
            var scaledFont   = SYMBOL_FONT_SIZE  * zoomScale;
            var half         = scaledSize / 2;

            // Phase 40b: with a class sprite the current room keeps its terrain
            // colour and the sprite stands above it; without one it stays the
            // classic red square.
            var spriteOn = spritesOn() && currentRoomId !== null && unit.x !== null &&
                resolveSheet(chainKeys(identity.classid, identity.lineage), false) !== null;

            rooms.forEach(function (room, id) {
                var p         = gridToCanvas(room.x, room.y);
                var isCurrent = (id === currentRoomId) && !spriteOn;
                var fill      = isCurrent ? CURRENT_ROOM_COLOR : colorForSymbol(room.symbol, room.env);
                var rx = p.px - half, ry = p.py - half;
                ctx.fillStyle   = fill;
                ctx.strokeStyle = ROOM_BORDER_COLOR;
                ctx.lineWidth   = scaledBorder;

                if (useCircle) {
                    ctx.beginPath();
                    ctx.arc(p.px, p.py, half, 0, Math.PI * 2);
                    ctx.fill();
                    ctx.stroke();
                } else {
                    ctx.fillRect(rx, ry, scaledSize, scaledSize);
                    ctx.strokeRect(rx, ry, scaledSize, scaledSize);
                }

                var symColor = isCurrent ? CURRENT_ROOM_TEXT_COLOR
                    : (fill !== '#3a3a4a' ? contrastColor(fill) : SYMBOL_TEXT_COLOR);
                ctx.fillStyle    = symColor;
                ctx.font         = 'bold ' + scaledFont + 'px monospace';
                ctx.textAlign    = 'center'; ctx.textBaseline = 'middle';
                ctx.fillText(room.symbol || '\u2022', p.px, p.py);
                if (mapSettings.showResources) {
                    var resIds = resourcesFor(id);
                    if (resIds.length > 0) {
                        var dot   = Math.max(3, scaledSize * 0.16);
                        var shown = resIds.slice(0, RESOURCE_ICON_MAX);
                        var inset = useCircle ? Math.max(2, half * 0.45) : Math.max(2, scaledSize * 0.1);
                        shown.forEach(function (rid, i) {
                            var meta = RESOURCE_INFO[rid];
                            ctx.fillStyle = (meta && meta.color) || '#aaaaaa';
                            ctx.strokeStyle = '#000000';
                            ctx.lineWidth = 1;
                            var dx = p.px - half + inset + i * (dot * 2 + 1) + dot * 0.5;
                            var dy = p.py - half + inset;
                            if (useCircle) {
                                dx = p.px - half * 0.5 + i * (dot * 2 + 1) - dot;
                                dy = p.py - half * 0.62;
                            }
                            ctx.beginPath();
                            ctx.arc(dx, dy, dot, 0, Math.PI * 2);
                            ctx.fill();
                            ctx.stroke();
                        });
                        if (resIds.length > RESOURCE_ICON_MAX) {
                            ctx.fillStyle = symColor;
                            ctx.font = 'bold ' + Math.max(6, dot * 2) + 'px monospace';
                            ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
                            ctx.fillText('+', p.px - half + inset + RESOURCE_ICON_MAX * (dot * 2 + 1), p.py - half + inset);
                        }
                    }
                }
                if (room.hasUp || room.hasDown) {
                    var arrowSize = Math.max(5, scaledSize * 0.28);
                    ctx.font      = 'bold ' + arrowSize + 'px monospace';
                    ctx.fillStyle = isCurrent ? CURRENT_ROOM_TEXT_COLOR : symColor;
                    // For circles the bounding-box corners sit outside the circle.
                    // Inset from centre by half/√2 so the arrows stay inside.
                    var arrowInset = useCircle
                        ? Math.max(2, half * 0.707 - arrowSize * 0.5)
                        : Math.max(2, scaledSize * 0.1);
                    if (room.hasDown) {
                        ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
                        ctx.fillText('\u25be', p.px - arrowInset, p.py + arrowInset);
                    }
                    if (room.hasUp) {
                        ctx.textAlign = 'right'; ctx.textBaseline = 'top';
                        ctx.fillText('\u25b4', p.px + arrowInset, p.py - arrowInset);
                    }
                }
            });

            drawCamps(nowMs);

            // Draw party member hearts over rooms (skip the player's current room).
            // Each heart eases from its previous grid position to the new one over HEART_EASE_DURATION.
            // Hearts on a different z-plane than the current room are not drawn.
            var currentRc = (currentRoomId !== null) ? roomCache[currentRoomId] : null;
            var currentRoomZ = currentRc ? currentRc.z : null;
            ctx.font = 'bold ' + Math.round(scaledSize) + 'px serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            var now = performance.now();
            var anyEasing = false;
            Object.keys(partyHeartEase).forEach(function (name) {
                var ease = partyHeartEase[name];
                if (currentRoomZ === null || ease.toZ !== currentRoomZ) { return; }
                if (currentRc && ease.toGx === currentRc.x && ease.toGy === currentRc.y) { return; }
                var t = Math.min((now - ease.startTime) / 1000 / HEART_EASE_DURATION, 1);
                var s = smoothstep(t);
                var gx = ease.fromGx + (ease.toGx - ease.fromGx) * s;
                var gy = ease.fromGy + (ease.toGy - ease.fromGy) * s;
                var p = gridToCanvas(gx, gy);
                // Phase 40b: a member whose class is known shows as that class's
                // sprite with an ally pennant; otherwise (or with sprites off)
                // the heart stays.
                if (!(spritesOn() && drawAllySprite(ease, gx, gy, t < 1, nowMs))) {
                    ctx.fillStyle = ease.aggro ? '#ff3333' : '#00cfcf';
                    ctx.font = 'bold ' + Math.round(scaledSize) + 'px serif';
                    ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
                    ctx.fillText('\u2665', p.px, p.py);
                }
                if (t < 1) { anyEasing = true; }
            });
            var drewUnit = drawUnit(nowMs);
            if (anyEasing && heartRafId === null) {
                heartRafId = requestAnimationFrame(function () {
                    heartRafId = null;
                    render();
                });
            }
            // The sprites and camp fires animate: keep redrawing while any is
            // on screen (a walk redraws every frame, idling a few times a second).
            if (drewUnit || (spritesOn() && campInfo && mapSettings.showCamp !== false &&
                (campInfo.has_camp || (campInfo.allied_camps || []).length))) {
                scheduleAnim(!!(drewUnit && drewUnit.walking));
            }
        }

        function roomAtPoint(cx, cy) {
            var half = (getRoomSize() * zoomScale) / 2;
            for (var [id, room] of rooms) {
                var p = gridToCanvas(room.x, room.y);
                if (cx >= p.px - half && cx <= p.px + half &&
                    cy >= p.py - half && cy <= p.py + half) { return id; }
            }
            return null;
        }

        // -- DOM ---------------------------------------------------------------
        function createPanel() {
            var wrap = document.createElement('div');
            wrap.className = 'map-canvas-wrap';

            canvas = document.createElement('canvas');
            canvas.id = 'map-2d-canvas';
            wrap.appendChild(canvas);
            ctx = canvas.getContext('2d');

            canvas.addEventListener('mouseleave', function () {
                hideTooltip();
                if (dragActive) { dragActive = false; canvas.style.cursor = ''; }
            });
            canvas.addEventListener('mousedown', function (e) {
                if (e.button !== 0) { return; }
                dragActive = true;
                dragStartPxX = e.clientX; dragStartPxY = e.clientY;
                dragStartPanX = panOffsetX; dragStartPanY = panOffsetY;
                canvas.style.cursor = 'grabbing'; e.preventDefault();
            });
            canvas.addEventListener('mousemove', function (e) {
                var rect = canvas.getBoundingClientRect();
                if (dragActive) {
                    var step = getBaseStep() * zoomScale;
                    panOffsetX = dragStartPanX - (e.clientX - dragStartPxX) / step;
                    panOffsetY = dragStartPanY - (e.clientY - dragStartPxY) / step;
                    render(); return;
                }
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                var info = id !== null ? roomInfoStore.get(id) : null;
                if (info) { clearTimeout(tooltipHideTimer); showTooltip(e.clientX, e.clientY, info); }
                else      { hideTooltip(); }
            });
            canvas.addEventListener('mouseup', function (e) {
                if (!dragActive) { return; }
                var dx = e.clientX - dragStartPxX, dy = e.clientY - dragStartPxY;
                dragActive = false; canvas.style.cursor = '';
                if (Math.abs(dx) > 4 || Math.abs(dy) > 4) { canvas.dataset.suppressClick = '1'; }
            });
            canvas.addEventListener('click', function (e) {
                if (canvas.dataset.suppressClick) { delete canvas.dataset.suppressClick; return; }
                var charInfo = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info;
                if (!charInfo || charInfo.role !== 'admin') { return; }
                var rect = canvas.getBoundingClientRect();
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                if (id === null) { return; }
                e.stopPropagation();
                uiMenu(e, [{ label: 'teleport ' + id, cmd: 'teleport ' + id },
                            { label: 'room info ' + id, cmd: 'room info ' + id }]);
            });
            canvas.addEventListener('wheel', function (e) {
                e.preventDefault();
                var factor = Math.pow(ZOOM_STEP, e.deltaY * 0.002);
                zoomScale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoomScale / factor));
                render();
            }, { passive: false });

            var controls = document.createElement('div');
            controls.className = 'map-controls';
            var btnOut = document.createElement('button');
            btnOut.textContent = '\u2212'; btnOut.title = 'Zoom out';
            btnOut.addEventListener('click', function () {
                zoomScale = Math.max(ZOOM_MIN, zoomScale / ZOOM_STEP); render();
            });
            var btnIn = document.createElement('button');
            btnIn.textContent = '+'; btnIn.title = 'Zoom in';
            btnIn.addEventListener('click', function () {
                zoomScale = Math.min(ZOOM_MAX, zoomScale * ZOOM_STEP); render();
            });
            var btnSettings = document.createElement('button');
            btnSettings.innerHTML = '&#9881;';
            btnSettings.title = 'Map settings';
            btnSettings.addEventListener('click', function (e) {
                e.stopPropagation();
                toggleSettingsPanel(wrap);
            });

            controls.appendChild(btnOut);
            controls.appendChild(btnIn);
            controls.appendChild(btnSettings);
            wrap.appendChild(controls);

            container = wrap;
            return wrap;
        }

        function onSettingsChanged() {
            render();
        }

        function onActivate() {
            resizeCanvas();
            render();
        }

        function onWorldMap() {
            resizeCanvas();
            if (currentZoneKey) {
                var savedId = currentRoomId;
                replayZone(currentZoneKey);
                currentRoomId = savedId;
            }
            render();
        }

        function onRoomUpdate(info, gx, gy, gz, sym, env) {
            resizeCanvas();
            var zoneKey = info.coords.split(',').map(function (s) { return s.trim(); })[0] + '/z:' + gz;
            if (currentZoneKey !== zoneKey) {
                currentZoneKey = zoneKey;
                replayZone(zoneKey);
            } else {
                addOrUpdateRoom(info.num, gx, gy, sym, env);
                var rc = roomCache[info.num];
                if (rc) {
                    if (Array.isArray(rc.exits)) {
                        rc.exits.forEach(function (exit) {
                            if (exit.dz !== 0) { return; }
                            var destRc = roomCache[exit.num];
                            if (destRc) {
                                if (!rooms.has(exit.num)) {
                                    addOrUpdateRoom(exit.num, destRc.x, destRc.y, destRc.symbol, destRc.env);
                                }
                                addEdge(info.num, exit.num, exit.locked, exit.secret);
                            }
                        });
                    }
                    if (Array.isArray(rc.stubs)) {
                        rc.stubs.forEach(function (stub) {
                            if (stub.dz === 0) {
                                zoneExitStubs.push({ roomId: info.num, dx: stub.dx, dy: stub.dy,
                                                     locked: stub.locked, secret: stub.secret });
                            }
                        });
                    }
                }
            }
            currentRoomId = info.num;
            unitMoveTo(gx, gy, zoneKey, performance.now());
            setCameraTarget(gx, gy);
        }

        // -- Settings panel ----------------------------------------------------
        function toggleSettingsPanel(wrap) {
            var existing = wrap.querySelector('.map-settings-panel');
            if (existing) { existing.remove(); return; }

            var panel = document.createElement('div');
            panel.className = 'map-settings-panel';

            function row(labelText, content) {
                var r = document.createElement('div');
                r.className = 'msp-row';
                var lbl = document.createElement('span');
                lbl.className = 'msp-label';
                lbl.textContent = labelText;
                r.appendChild(lbl);
                r.appendChild(content);
                return r;
            }

            function btnGroup(options, getValue, setValue) {
                var grp = document.createElement('div');
                grp.className = 'msp-btngroup';
                options.forEach(function (opt) {
                    var b = document.createElement('button');
                    b.textContent = opt.label;
                    b.dataset.val = opt.value;
                    if (getValue() === opt.value) { b.classList.add('active'); }
                    b.addEventListener('click', function () {
                        grp.querySelectorAll('button').forEach(function (x) { x.classList.remove('active'); });
                        b.classList.add('active');
                        setValue(opt.value);
                        saveMapSettings();
                        render();
                    });
                    grp.appendChild(b);
                });
                return grp;
            }

            var shapeRow = row('Shape', btnGroup(
                [{ label: 'Squares', value: 'square' }, { label: 'Circles', value: 'circle' }],
                function () { return mapSettings.roomShape; },
                function (v) { mapSettings.roomShape = v; }
            ));
            panel.appendChild(shapeRow);

            panel.appendChild(row('Resources', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.showResources !== false; },
                function (v) { mapSettings.showResources = v; }
            )));

            panel.appendChild(row('Sprites', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.sprites !== false; },
                function (v) { mapSettings.sprites = v; }
            )));

            panel.appendChild(row('Camps', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.showCamp !== false; },
                function (v) { mapSettings.showCamp = v; }
            )));

            var slider = document.createElement('input');
            slider.type  = 'range';
            slider.min   = String(ROOM_SIZE_MIN);
            slider.max   = String(ROOM_SIZE_MAX);
            slider.value = String(mapSettings.roomSize);
            slider.className = 'msp-slider';
            slider.addEventListener('input', function () {
                mapSettings.roomSize = parseInt(slider.value, 10);
                saveMapSettings();
                render();
            });
            panel.appendChild(row('Size', slider));

            var spacingSlider = document.createElement('input');
            spacingSlider.type  = 'range';
            spacingSlider.min   = String(ROOM_SPACING_MIN);
            spacingSlider.max   = String(ROOM_SPACING_MAX);
            spacingSlider.value = String(mapSettings.roomSpacing);
            spacingSlider.className = 'msp-slider';
            spacingSlider.addEventListener('input', function () {
                mapSettings.roomSpacing = parseInt(spacingSlider.value, 10);
                saveMapSettings();
                render();
            });
            panel.appendChild(row('Spacing', spacingSlider));

            function colorPicker(settingKey) {
                var input = document.createElement('input');
                input.type  = 'color';
                input.value = mapSettings[settingKey];
                input.className = 'msp-color';
                input.addEventListener('input', function () {
                    mapSettings[settingKey] = input.value;
                    saveMapSettings();
                    render();
                });
                return input;
            }

            panel.appendChild(row('Connections', colorPicker('connectionColor')));
            panel.appendChild(row('Background', colorPicker('mapBackground')));

            // -- Default zoom button --
            var btnDefaultZoom = document.createElement('button');
            btnDefaultZoom.className = 'msp-reset';
            btnDefaultZoom.style.alignSelf = 'stretch';
            btnDefaultZoom.style.marginTop = '2px';
            (function updateDefaultZoomLabel() {
                btnDefaultZoom.textContent = mapSettings.defaultZoom !== null
                    ? 'Default zoom: ' + mapSettings.defaultZoom.toFixed(2) + ' (click to update)'
                    : 'Set default zoom';
            }());
            btnDefaultZoom.addEventListener('click', function (e) {
                e.stopPropagation();
                mapSettings.defaultZoom = Math.round(zoomScale * 100) / 100;
                saveMapSettings();
                btnDefaultZoom.textContent = 'Default zoom: ' + mapSettings.defaultZoom.toFixed(2) + ' (click to update)';
            });
            panel.appendChild(btnDefaultZoom);

            // -- Import / Export JSON --
            // (handled by the main webclient settings Export JSON / Import JSON buttons)

            // -- Reset --
            var btnReset = document.createElement('button');
            btnReset.textContent = 'Reset to defaults';
            btnReset.className = 'msp-reset';
            btnReset.addEventListener('click', function (e) {
                e.stopPropagation();
                Object.assign(mapSettings, MAP_SETTINGS_DEFAULTS);
                zoomScale = 1.0;
                saveMapSettings();
                panel.remove();
                document.removeEventListener('click', onOutsideClick, true);
                toggleSettingsPanel(wrap);
                render();
            });
            panel.appendChild(btnReset);

            wrap.appendChild(panel);

            function onOutsideClick(e) {
                if (!panel.contains(e.target) && !e.target.closest('.map-controls')) {
                    panel.remove();
                    document.removeEventListener('click', onOutsideClick, true);
                }
            }
            setTimeout(function () {
                document.addEventListener('click', onOutsideClick, true);
            }, 0);
        }

        function setupResizeObserver(win) {
            if (typeof ResizeObserver === 'undefined') { return; }
            var ro = new ResizeObserver(function () { resizeCanvas(); render(); });
            var orig = win.open.bind(win);
            win.open = function () { orig(); if (container) { ro.observe(container); } };
        }

        return {
            createPanel:         createPanel,
            onActivate:          onActivate,
            onWorldMap:          onWorldMap,
            onRoomUpdate:        onRoomUpdate,
            setupResizeObserver: setupResizeObserver,
            getCurrentRoomId:    function () { return currentRoomId; },
            setIdentity: function (classid, lineage) {
                if (identity.classid === classid && identity.lineage === lineage) { return; }
                identity.classid = classid; identity.lineage = lineage;
                render();
            },
            setCompanySize: function (n) { if (n !== companySize) { companySize = n; render(); } },
            setCamp: function (camp) { campInfo = camp; render(); },
            redraw: function () { render(); },
            // state is for the browser checks (scripts/browser/map-check.mjs).
            state: function () {
                var res = resolveSheet(chainKeys(identity.classid, identity.lineage), false);
                var pose = unit.x === null ? null : unitPose(performance.now());
                return {
                    spriteDrawn: spritesOn() && !!res, face: unit.face, flip: unit.flip, walking: !!(pose && pose.walking),
                    queued: unit.queue.length, companySize: companySize,
                    camp: campInfo, fading: unit.fadeStart >= 0,
                    unit: pose ? { x: pose.x, y: pose.y } : null,
                    keys: chainKeys(identity.classid, identity.lineage),
                    allies: Object.keys(partyHeartEase).map(function (n) {
                        var e = partyHeartEase[n];
                        return { name: n, classid: e.classid, lineage: e.lineage, face: e.face,
                                 sprite: spritesOn() && !!(e.classid || e.lineage) && resolveSheet(chainKeys(e.classid, e.lineage), false) !== null };
                    }),
                };
            },
            setPartyPositions: function (positions) {
                var newPositions = positions || {};
                var now = performance.now();

                // Update ease entries for each member in the new state.
                Object.keys(newPositions).forEach(function (name) {
                    var pos = newPositions[name];
                    if (!pos.hasCoordinates) { return; }

                    var existing = partyHeartEase[name];
                    var fromGx, fromGy;

                    if (existing) {
                        // Interpolate current visual position as the new start.
                        var t = Math.min((now - existing.startTime) / 1000 / HEART_EASE_DURATION, 1);
                        var s = smoothstep(t);
                        fromGx = existing.fromGx + (existing.toGx - existing.fromGx) * s;
                        fromGy = existing.fromGy + (existing.toGy - existing.fromGy) * s;
                    } else if (partyPositions[name] !== undefined) {
                        var oldPos = partyPositions[name];
                        fromGx = oldPos.x;
                        fromGy = oldPos.y;
                    } else {
                        // First time we see this member: start at destination (no animation).
                        fromGx = pos.x;
                        fromGy = pos.y;
                    }

                    partyHeartEase[name] = {
                        fromGx:    fromGx,
                        fromGy:    fromGy,
                        toGx:      pos.x,
                        toGy:      pos.y,
                        toZ:       pos.z,
                        aggro:     pos.aggro,
                        lineage:   pos.lineage,
                        classid:   pos.classid,
                        startTime: now,
                    };
                    var f = faceOf(pos.x - fromGx, pos.y - fromGy, existing ? { face: existing.face, flip: existing.flip } : null);
                    partyHeartEase[name].face = f.face;
                    partyHeartEase[name].flip = f.flip;
                });

                // Remove ease entries for members no longer in the party.
                Object.keys(partyHeartEase).forEach(function (name) {
                    if (!newPositions[name]) { delete partyHeartEase[name]; }
                });

                partyPositions = newPositions;
                render();
            },
        };

    }());

    // =========================================================================
    // Window DOM
    // =========================================================================

    function createDOM() {
        var root = document.createElement('div');
        root.id = 'map-window';

        var panels = document.createElement('div');
        panels.id = 'map-panels';

        var panel2d = document.createElement('div');
        panel2d.className = 'map-panel active';
        panel2d.style.inset = '0';
        panel2d.style.position = 'absolute';
        panel2d.style.display = 'block';
        panel2d.appendChild(view2d.createPanel());

        panels.appendChild(panel2d);
        root.appendChild(panels);

        document.body.appendChild(root);

        return root;
    }

    // =========================================================================
    // VirtualWindow
    // =========================================================================

    var win = new VirtualWindow('Map', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  363,
        factory: function () {
            var el = createDOM();
            return {
                title:      'Map',
                mount:      el,
                background: 'var(--t-bg-panel)',
                border:     1,
                x:          'right',
                y:          66,
                width:      363,
                height:     20 + 363,
                header:     20,
                bottom:     60,
            };
        },
    });

    view2d.setupResizeObserver(win);
    Sprites.onChange(function () { view2d.redraw(); });
    window.MapView = { state: function () { return view2d.state(); } };

    // =========================================================================
    // GMCP update logic
    // =========================================================================

    function updateWorldMap() {
        var worldData = Client.GMCPStructs.World;
        if (!worldData || !worldData.Map) { return; }
        win.open();
        if (!win.isOpen()) { return; }
        ingestWorldMap(worldData.Map);
    }

    function updatePartyPositions() {
        var partyData = Client.GMCPStructs.Party;
        if (!partyData || !partyData.Vitals) { partyMemberPositions = {}; view2d.setPartyPositions(partyMemberPositions); return; }
        var vitals = partyData.Vitals;
        var myName = (Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info && Client.GMCPStructs.Char.Info.name) || '';
        partyMemberPositions = {};
        Object.keys(vitals).forEach(function (name) {
            if (name === myName) { return; }
            var v = vitals[name];
            if (!v.hascoordinates) { return; }
            partyMemberPositions[name] = { x: v.mapx, y: v.mapy, z: v.mapz, hasCoordinates: true, aggro: !!v.aggro,
                                           lineage: v.lineage || '', classid: v.classid || '' };
        });
        view2d.setPartyPositions(partyMemberPositions);
    }

    // Phase 40b: who the player is (Char.Info), how many travel with them and
    // their camps (Company), all read from the stored GMCP state.
    function updateIdentity() {
        var c = Client.GMCPStructs.Char;
        var info = c && c.Info;
        view2d.setIdentity((info && info.classid) || '', (info && info.lineage) || '');
    }

    function updateCompany() {
        var co = Client.GMCPStructs.Company;
        var size = 0;
        if (co && co.leader && Array.isArray(co.members)) {
            var withLeader = co.members.filter(function (m) { return m && m.status === 'present'; }).length;
            size = withLeader > 0 ? withLeader + 1 : 0;
        }
        view2d.setCompanySize(size);
        view2d.setCamp((co && co.Camp) || null);
    }

    function updateMap() {
        var obj = Client.GMCPStructs.Room;
        if (!obj || !obj.Info) { return; }
        win.open();
        if (!win.isOpen()) { return; }

        updateIdentity();
        updateCompany();

        if (!worldMapRequested) {
            worldMapRequested = true;
            Client.GMCPRequest('World.Map');
        }

        var info     = obj.Info;
        var coords   = info.coords.split(',').map(function (s) { return s.trim(); });
        var gx       = parseInt(coords[1], 10);
        var gy       = parseInt(coords[2], 10);
        var gz       = parseInt(coords[3], 10);
        var sym      = symbolForRoom(info);
        var env      = info.environment || '';

        roomInfoStore.set(info.num, info);
        upsertRoomCache(info.num, coords[0], gx, gy, gz, sym, env, info.exitsv2);

        var winBox = win.get();
        if (winBox) { winBox.setTitle('map (' + info.area + ')'); }

        view2d.onRoomUpdate(info, gx, gy, gz, sym, env);
    }

    // =========================================================================
    // Registration
    // =========================================================================

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Room', 'World', 'Party', 'Party.Vitals', 'Char', 'Company'],
        onGMCP: function (namespace) {
            if (namespace === 'Char.Info' || namespace === 'Char') {
                updateIdentity();
            } else if (namespace.indexOf('Company') === 0) {
                updateCompany();
            } else if (namespace === 'World.Map') {
                updateWorldMap();
            } else if (namespace === 'Room.Info' || namespace === 'Room') {
                updateMap();
            } else if (namespace === 'Party' || namespace === 'Party.Vitals') {
                updatePartyPositions();
            }
        },
    });

}());
