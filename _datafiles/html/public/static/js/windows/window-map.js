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
        style:           'tiles',    // Phase 40c: 'tiles' (terrain art, landmarks, walls, fog) | 'classic' (coloured squares and letters)
        dayNight:        true,       // Phase 40d: shade outdoor tiles by the game's time of day
    };

    // Phase 40a: room resources. The tiles style draws each one's S1 icon in
    // the tile's corner (Phase 40d); the classic style, a tile too small for
    // an icon and an icon still loading keep the small coloured dot. Tooltips
    // name them.
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

    // Phase 40a2: gathering resources picked clean for now ("depleted").
    function depletedFor(roomId) {
        var info = roomInfoStore.get(roomId);
        return (info && Array.isArray(info.depleted)) ? info.depleted : [];
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

    function showTooltip(mouseX, mouseY, info, canWalk) {
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
            var gone = Array.isArray(info.depleted) ? info.depleted : [];
            var resNames = info.resources.map(function (r) {
                var label = (RESOURCE_INFO[r] && RESOURCE_INFO[r].label) || r;
                return gone.indexOf(r) !== -1 ? label + ' (picked clean)' : label;
            });
            html += '<hr class="tt-divider"><div class="tt-row">' +
                    '<span class="tt-label">Here</span>' +
                    '<span class="tt-value">' + resNames.join(', ') + '</span></div>';
        }

        if (canWalk) {
            html += '<hr class="tt-divider"><div class="tt-row"><span class="tt-label">Click</span>' +
                    '<span class="tt-value">walk here</span></div>';
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
        '@media (hover: hover) and (pointer: fine) { .map-controls button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); } }',
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
        '@media (hover: hover) and (pointer: fine) { .msp-btngroup button:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); } }',
        '.msp-btngroup button.active { background: var(--t-map-controls-active); color: var(--t-text-white); border-color: var(--t-map-controls-active); }',
        '.msp-slider { flex: 1; min-width: 80px; cursor: pointer; accent-color: var(--t-map-controls-active); }',
        '.msp-color { width: 32px; height: 20px; padding: 0; border: 1px solid var(--t-map-controls-border); border-radius: 3px; cursor: pointer; background: none; }',
        '.msp-reset { margin-top: 2px; align-self: flex-end; font-size: 0.68em; padding: 1px 6px; background: none; color: var(--t-text-secondary); border: 1px solid var(--t-accent-dim); border-radius: 3px; cursor: pointer; line-height: 1.6; }',
        '@media (hover: hover) and (pointer: fine) { .msp-reset:hover { background: var(--t-map-controls-hover); color: var(--t-text-white); border-color: var(--t-map-controls-hover); } }',
    ].join('\n'));

    // =========================================================================
    // 2D view
    // =========================================================================

    var view2d = (function () {

        // -- Constants ---------------------------------------------------------
        var ROOM_GRID_STEP = 42;  // fallback only; runtime reads mapSettings.roomSpacing

        // Phase 40c: in the tiles style a room is one 32 px art tile and
        // neighbours touch, so the size and spacing settings do not apply.
        var TILE_PX = 32;
        var TILE_ZOOMS = [0.5, 0.75, 1, 1.5, 2, 3, 4]; // tile sizes 16, 24, 32, 48, 64, 96 and 128 px stay crisp
        var ANIM_TILE_MS = 250;
        function tilesOn()      { return mapSettings.style !== 'classic'; }
        function getRoomSize()  { return tilesOn() ? TILE_PX : Math.round(mapSettings.roomSize); }
        function getBaseStep()  { return tilesOn() ? TILE_PX : Math.round(mapSettings.roomSpacing); }
        function snapZoom(z) {
            var best = TILE_ZOOMS[0];
            TILE_ZOOMS.forEach(function (v) { if (Math.abs(v - z) < Math.abs(best - z)) { best = v; } });
            return best;
        }
        function reducedMotion() {
            try { return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches); } catch (e) { return false; }
        }

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
        // Phase 47: a phone starts one tile size closer (48 px tiles), so
        // the rooms around you read at arm's length; a default zoom in the
        // map settings or a pinch still wins.
        function startZoom() {
            try { return window.matchMedia && window.matchMedia('(max-width: 820px)').matches ? 1.5 : 1.0; } catch (e) { return 1.0; }
        }
        var zoomScale     = startZoom();
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
        var companions  = [];       // Phase 40c: present companions { key, lineage, classid }, drawn beside you
        // Where each companion stands around the player, in art pixels from
        // the player's feet (x across, y up); a fifth or later is not drawn
        // (the badge still counts everyone).
        var COMPANION_SLOTS = [{ x: -13, y: 2 }, { x: 13, y: 2 }, { x: -7, y: 7 }, { x: 7, y: 7 }];
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
        var drawn = { tiles: 0, fallbacks: 0, walls: 0, fog: 0, landmarks: 0, glyphs: 0, animated: 0, icons: 0, dots: 0, path: 0, shaded: 0 }; // what the last render drew (browser checks)
        var walkInfo = null;   // Phase 40d: Walkto { target, path: [room ids ahead] } while a walk is under way
        var timeInfo = null;   // Phase 40d: Gametime, for the day/night shading
        var nightQuant = -1;   // the shading level last drawn, in tenths

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

        // artSmoothing turns smoothing on only for high-density art, which
        // is drawn below its native size; 1x pixel art stays crisp.
        function artSmoothing(info) {
            var hi = (info.density || 1) > 1;
            ctx.imageSmoothingEnabled = hi;
            if (hi) { ctx.imageSmoothingQuality = 'high'; }
        }

        // A sheet with density N has frames N times the 1x size; it is
        // drawn at the same on-map size as 1x art (mult is per 1x pixel).
        function drawFrame(sheet, row, flip, cx, feetY, mult, now, startMs, alpha) {
            var info = sheet.info;
            var r = Math.max(0, (info.rows || []).indexOf(row));
            var f = Sprites.frame(info, r, now, startMs);
            var d = info.density || 1;
            var w = f.sw * mult / d, h = f.sh * mult / d;
            var x = Math.round(cx - w / 2);
            var y = Math.round(feetY - h * ((info.feet_baseline || f.sh) / f.sh));
            ctx.save();
            artSmoothing(info);
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
            var d = a.info.density || 1;
            var w = f.sw * mult / d, h = f.sh * mult / d;
            ctx.save();
            artSmoothing(a.info);
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

        // fireAndRest draws a camp's fire (bottom right of the tent, or at
        // <at>) and its resting mark (over the tent).
        function fireAndRest(cx, cy, mult, lit, resting, now, at, embers) {
            var fx = at ? at.px : cx + 8 * mult, fy = at ? at.py : cy + 8 * mult;
            if (lit) {
                drawIcon('map/camp/fire-lit.png', fx, fy, mult, now);
                drawIcon('map/camp/smoke.png', fx, fy - 12 * mult, mult, now);
            } else if (embers) {
                // Phase 40c: a fire burned down to embers glows low, no smoke.
                if (!drawIcon('map/camp/embers.png', fx, fy, mult, now)) {
                    drawIcon('map/camp/fire-unlit.png', fx, fy, mult, now);
                }
            } else {
                drawIcon('map/camp/fire-unlit.png', fx, fy, mult, now);
            }
            if (resting) { drawIcon('map/camp/resting.png', cx - 8 * mult, cy - 12 * mult, mult, now); }
        }

        // tent is false for a camp pitched without a tent (Phase 40a3): the
        // rough camp is drawn instead. A payload that does not say (older
        // server) is drawn with its tent.
        function drawCamp(roomId, ally, lit, resting, innRest, now, occupied, embers, tent) {
            var r = rooms.get(roomId);
            if (!r) { return; }
            var p = gridToCanvas(r.x, r.y);
            var mult = spriteMult(getRoomSize() * zoomScale);
            var fire = null;
            if (occupied) {
                // Your sprite stands on this tile and would hide the camp:
                // pitch the tent behind your left shoulder and the fire by
                // your right foot so both still show (40b review).
                fire = { px: p.px + 14 * mult, py: p.py + 6 * mult };
                p = { px: p.px - 12 * mult, py: p.py - 8 * mult };
            }
            if (innRest) {
                drawIcon('map/camp/inn-rest.png', p.px, p.py, mult, now);
                return;
            }
            var pitch = (tent === false)
                ? (ally ? 'map/camp/camp-rough-ally.png' : 'map/camp/camp-rough.png')
                : (ally ? 'map/camp/tent-ally.png' : 'map/camp/tent.png');
            if (!drawIcon(pitch, p.px, p.py, mult, now)) {
                // no art yet: a small tent triangle
                var q = getRoomSize() * zoomScale * 0.4;
                ctx.fillStyle = ally ? '#6a9ec9' : '#c9a15a';
                ctx.beginPath(); ctx.moveTo(p.px, p.py - q); ctx.lineTo(p.px + q, p.py + q); ctx.lineTo(p.px - q, p.py + q); ctx.closePath(); ctx.fill();
            }
            fireAndRest(p.px, p.py, mult, lit, resting, now, fire, embers);
        }

        // drawCamps draws your camp and your party's. spriteOn says your class
        // sprite stands on your tile, so a camp there is drawn beside it.
        function drawCamps(now, spriteOn) {
            if (!campInfo || mapSettings.showCamp === false) { return; }
            (campInfo.allied_camps || []).forEach(function (c) {
                drawCamp(c.room_id, true, !!c.fire_lit, !!c.resting, false, now, spriteOn && c.room_id === currentRoomId, !!c.embers, c.tent);
            });
            if (campInfo.has_camp && campInfo.room_id) {
                var inn = !!(campInfo.here && campInfo.inn && campInfo.resting);
                drawCamp(campInfo.room_id, false, !!campInfo.fire_lit, !!campInfo.resting, inn, now, spriteOn && campInfo.room_id === currentRoomId, !!campInfo.embers, campInfo.tent);
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
            // Companions stand in behind you and move with you.
            companions.slice(0, COMPANION_SLOTS.length).forEach(function (c, i) {
                var cres = resolveSheet(chainKeys(c.classid, c.lineage), pose.walking);
                if (!cres) { return; }
                var slot = COMPANION_SLOTS[i], cm = mult * ALLY_SCALE;
                drawFrame(cres.sheet, unit.face, unit.flip, p.px + slot.x * mult, p.py + (12 - slot.y) * mult, cm, now, pose.walking ? pose.start : 0, alpha);
            });
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
        // The canvas backing store is sized in device pixels so art stays
        // sharp on high-resolution screens; all drawing uses CSS pixels
        // (viewW x viewH) through the transform render() sets.
        var viewW = 1, viewH = 1, pixelRatio = 1;
        function resizeCanvas() {
            if (!canvas || !container) { return; }
            pixelRatio = window.devicePixelRatio || 1;
            viewW = container.clientWidth  || 1;
            viewH = container.clientHeight || 1;
            canvas.width  = Math.round(viewW * pixelRatio);
            canvas.height = Math.round(viewH * pixelRatio);
            canvas.style.width  = viewW + 'px';
            canvas.style.height = viewH + 'px';
        }

        function gridToCanvas(gx, gy) {
            var midX = Math.floor(viewW / 2);
            var midY = Math.floor(viewH / 2);
            var step = getBaseStep() * zoomScale;
            return {
                px: midX + (gx - cameraX - panOffsetX) * step,
                py: midY + (gy - cameraY - panOffsetY) * step,
            };
        }

        var lastWheelStep = 0;
        function stepTileZoom(dir) {
            var i = TILE_ZOOMS.indexOf(snapZoom(zoomScale)) + dir;
            return TILE_ZOOMS[Math.max(0, Math.min(TILE_ZOOMS.length - 1, i))];
        }

        function setCameraTarget(tx, ty) {
            panOffsetX = 0; panOffsetY = 0;
            var targetZoom = (mapSettings.defaultZoom !== null) ? mapSettings.defaultZoom : zoomScale;
            if (tilesOn()) { targetZoom = snapZoom(targetZoom); }
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

        // -- Phase 40c: terrain tiles, walls, landmarks --------------------------
        // drawTileArt draws one 32 px frame of <path> centred on (px, py) at
        // <size> px, smoothing off. <variant> picks a column of a variants
        // sheet; <animated> instead picks the frame for <now>. It reports
        // whether the image was ready.
        function drawTileArt(path, variant, px, py, size, now, animated) {
            var a = Sprites.art(path);
            if (!a) { return false; }
            var fw = a.info.frame[0], fh = a.info.frame[1];
            var f = animated ? Sprites.frame(a.info, 0, now, 0) : { sx: variant * fw, sy: 0, sw: fw, sh: fh };
            ctx.save();
            artSmoothing(a.info);
            ctx.drawImage(a.img, f.sx, f.sy, f.sw, f.sh, Math.round(px - size / 2), Math.round(py - size / 2), size, size);
            ctx.restore();
            return true;
        }

        // drawTerrain draws a room's biome tile (the variant is the room id
        // mod 3, the same for every viewer) and its animated overlay. It
        // returns null while the art is not ready, else { animated }.
        function drawTerrain(room, id, p, size, now) {
            var biome = room.env ? room.env : 'default';
            var path = 'map/terrain/' + biome + '.png';
            if (Sprites.status(path) === 'none') {
                // A biome with no art (or none yet, before the manifest loads).
                if (!Sprites.has('map/terrain/unknown.png')) { return null; }
                path = 'map/terrain/unknown.png';
            }
            var a = Sprites.art(path);
            if (!a) { return null; }
            var variants = Math.max(1, a.info.variants || 1);
            drawTileArt(path, ((id % variants) + variants) % variants, p.px, p.py, size, now, false);
            var animated = false;
            if (a.info.animated_overlay && !reducedMotion()) {
                animated = true;
                drawTileArt('map/terrain/' + a.info.animated_overlay, 0, p.px, p.py, size, now, true);
            }
            return { animated: animated };
        }

        // drawWalls puts a dark edge between two touching rooms with no exit
        // between them, so the map never suggests a way through a wall.
        function drawWalls(index, size) {
            var w = Math.max(2, Math.round(size / 16));
            ctx.save();
            ctx.strokeStyle = '#0e0a08';
            ctx.lineWidth = w;
            ctx.lineCap = 'butt';
            var count = 0;
            ctx.beginPath();
            rooms.forEach(function (room, id) {
                var p = gridToCanvas(room.x, room.y);
                var h = Math.round(size / 2);
                [[1, 0], [0, 1]].forEach(function (d) {
                    var other = index[(room.x + d[0]) + ',' + (room.y + d[1])];
                    if (other === undefined) { return; }
                    var key = id < other ? (id + '-' + other) : (other + '-' + id);
                    if (edges.has(key)) { return; }
                    count++;
                    if (d[0] === 1) {
                        var x = Math.round(p.px) + h;
                        ctx.moveTo(x, Math.round(p.py) - h); ctx.lineTo(x, Math.round(p.py) + h);
                    } else {
                        var y = Math.round(p.py) + h;
                        ctx.moveTo(Math.round(p.px) - h, y); ctx.lineTo(Math.round(p.px) + h, y);
                    }
                });
            });
            ctx.stroke();
            ctx.restore();
            return count;
        }

        // drawLandmark draws the landmark overlay a room's legend or symbol
        // maps to (map/landmarks.json). A symbol with no landmark keeps its
        // letter, outlined so it reads on any terrain; an intentional-glyph
        // legend (shore) and a plain room draw nothing over the tile.
        function drawLandmark(room, id, p, size, fontPx, now) {
            var info = roomInfoStore.get(id) || {};
            var table = Sprites.data('map/landmarks.json');
            var legend = String(info.maplegend || '').toLowerCase();
            var sym = info.mapsymbol || '';
            var lm = table ? ((table.legends && table.legends[legend]) || (table.symbols && table.symbols[sym])) : '';
            if (lm) {
                var path = 'map/landmarks/' + lm + '.png';
                var st = Sprites.status(path);
                if (st === 'loading') { return ''; }
                if (st === 'ready') { drawTileArt(path, 0, p.px, p.py, size, now, false); return 'landmark'; }
            }
            if (!sym) { return ''; }
            if (table && Array.isArray(table.glyphs) && table.glyphs.indexOf(legend) !== -1) { return ''; }
            ctx.save();
            ctx.font = 'bold ' + Math.max(8, Math.round(fontPx)) + 'px monospace';
            ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
            ctx.lineJoin = 'round';
            ctx.lineWidth = Math.max(2, Math.round(fontPx / 4));
            ctx.strokeStyle = '#000000';
            ctx.strokeText(sym, p.px, p.py);
            ctx.fillStyle = '#ffffff';
            ctx.fillText(sym, p.px, p.py);
            ctx.restore();
            return 'glyph';
        }

        // drawResources marks what a room offers in its corner: the S1 icon
        // for each (a depleted one gets the empty-basket overlay), at most
        // RESOURCE_ICON_MAX with a "+" for the rest. A classic map, a tile
        // under 24 px and an icon not loaded yet fall back to coloured dots,
        // hollow with a slash once picked clean.
        function drawResources(id, p, tilePx, scaledSize, half, useCircle, tiles, tiled, symColor, nowMs) {
            var resIds = resourcesFor(id);
            if (resIds.length === 0) { return; }
            var goneIds = depletedFor(id);
            var shown = resIds.slice(0, RESOURCE_ICON_MAX);
            var iconMult = Math.max(0.5, spriteMult(tilePx) / 2);   // 8 px icons on a 32 px tile, 16 px on a 64 px one
            var iconPx = 16 * iconMult;
            var useIcons = tiles && tiled && tilePx >= 24;
            var dot = Math.max(3, scaledSize * 0.16);
            var inset = (useCircle && !tiles) ? Math.max(2, half * 0.45) : Math.max(2, scaledSize * 0.1);

            shown.forEach(function (rid, i) {
                var picked = goneIds.indexOf(rid) !== -1;
                if (useIcons) {
                    var cx = p.px - tilePx / 2 + 1 + iconPx / 2 + i * (iconPx + 1);
                    var cy = p.py - tilePx / 2 + 1 + iconPx / 2;
                    var path = 'map/resources/' + (RESOURCE_INFO[rid] ? rid : 'unknown') + '.png';
                    if (drawIcon(path, cx, cy, iconMult, nowMs)) {
                        drawn.icons++;
                        if (picked) { drawIcon('map/resources/depleted.png', cx, cy, iconMult, nowMs); }
                        return;
                    }
                }
                var meta = RESOURCE_INFO[rid];
                var dx = p.px - half + inset + i * (dot * 2 + 1) + dot * 0.5;
                var dy = p.py - half + inset;
                if (useCircle && !tiles) {
                    dx = p.px - half * 0.5 + i * (dot * 2 + 1) - dot;
                    dy = p.py - half * 0.62;
                }
                drawn.dots++;
                ctx.fillStyle = (meta && meta.color) || '#aaaaaa';
                ctx.strokeStyle = '#000000';
                ctx.lineWidth = 1;
                ctx.beginPath();
                ctx.arc(dx, dy, dot, 0, Math.PI * 2);
                if (picked) {
                    // depleted: the dot is hollow with a slash through it
                    ctx.fillStyle = mapSettings.mapBackground;
                    ctx.fill();
                    ctx.strokeStyle = (meta && meta.color) || '#aaaaaa';
                    ctx.stroke();
                    ctx.beginPath();
                    ctx.moveTo(dx - dot, dy + dot);
                    ctx.lineTo(dx + dot, dy - dot);
                    ctx.stroke();
                } else {
                    ctx.fill();
                    ctx.stroke();
                }
            });
            if (resIds.length > RESOURCE_ICON_MAX) {
                var plusX = useIcons ? p.px - tilePx / 2 + 1 + RESOURCE_ICON_MAX * (iconPx + 1) + 2
                                     : p.px - half + inset + RESOURCE_ICON_MAX * (dot * 2 + 1);
                var plusY = useIcons ? p.py - tilePx / 2 + 1 + iconPx / 2 : p.py - half + inset;
                ctx.fillStyle = tiled ? '#ffffff' : symColor;
                ctx.font = 'bold ' + Math.max(6, (useIcons ? iconPx : dot * 2)) + 'px monospace';
                ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
                ctx.fillText('+', plusX, plusY);
            }
        }

        // nightLevel is how far into night the game's clock is, 0 (day) to 1
        // (night), with an hour of dusk before NightStart and of dawn after
        // DayStart. Without a clock it is 0.
        function nightLevel() {
            var g = timeInfo;
            if (!g || typeof g.hour24 !== 'number') { return 0; }
            var h = g.hour24 + ((g.minute || 0) / 60);
            var ds = g.day_start, ns = g.night_start;
            if (typeof ds !== 'number' || typeof ns !== 'number' || ds >= ns) { return g.night ? 1 : 0; }
            if (h >= ns) { return 1; }
            if (h >= ns - 1) { return h - (ns - 1); }
            if (h >= ds + 1) { return 0; }
            if (h >= ds) { return 1 - (h - ds); }
            return 1;
        }

        var NIGHT_STRENGTH = 0.55;   // how dark full night multiplies a tile

        // drawNightShade darkens and cools the tiles the sky reaches (not
        // indoor or dark biomes) by the time of day; labels stay readable
        // because only terrain, landmark and icon layers sit under it.
        function drawNightShade(drewTiles, tilePx) {
            var level = nightLevel();
            nightQuant = Math.round(level * 10);
            if (!mapSettings.dayNight || level <= 0.02) { return; }
            var t = level * NIGHT_STRENGTH;
            var r = Math.round(255 - t * (255 - 70)), g = Math.round(255 - t * (255 - 85)), b = Math.round(255 - t * (255 - 150));
            ctx.save();
            ctx.globalCompositeOperation = 'multiply';
            ctx.fillStyle = 'rgb(' + r + ',' + g + ',' + b + ')';
            rooms.forEach(function (room, id) {
                if (!drewTiles[id]) { return; }
                var biome = biomeTable[room.env];
                if (biome && (biome.indoor || biome.dark)) { return; }
                var p = gridToCanvas(room.x, room.y);
                ctx.fillRect(Math.round(p.px - tilePx / 2), Math.round(p.py - tilePx / 2), tilePx, tilePx);
                drawn.shaded++;
            });
            ctx.restore();
        }

        // drawWalkPath marks the planned walk: a breadcrumb on each room
        // ahead and the destination flag on the last (S1 markers). Rooms off
        // this map level or zone are skipped.
        function drawWalkPath(tilePx, nowMs) {
            if (!walkInfo || !Array.isArray(walkInfo.path)) { return false; }
            var mult = spriteMult(tilePx);
            var animated = false;
            walkInfo.path.forEach(function (rid) {
                var room = rooms.get(rid);
                if (!room) { return; }
                var p = gridToCanvas(room.x, room.y);
                var isTarget = rid === walkInfo.target;
                var drew;
                if (isTarget) {
                    drew = drawIcon('map/markers/walk-target.png', p.px, p.py - 4 * mult, mult, nowMs);
                    animated = animated || drew;
                } else {
                    drew = drawIcon('map/markers/walk-dot.png', p.px, p.py, mult, nowMs);
                }
                if (!drew) {
                    ctx.save();
                    ctx.fillStyle = isTarget ? '#f2c14e' : '#ffffff';
                    ctx.strokeStyle = '#000000';
                    ctx.lineWidth = 1;
                    ctx.beginPath();
                    ctx.arc(p.px, p.py, Math.max(2, tilePx * (isTarget ? 0.14 : 0.08)), 0, Math.PI * 2);
                    ctx.fill(); ctx.stroke();
                    ctx.restore();
                }
                drawn.path++;
            });
            return animated;
        }

        // drawChevrons marks a room with stairs up or down (S1 chevrons).
        // It returns false when the art is not ready, so the glyph stands.
        function drawChevrons(room, p, now) {
            var size = getRoomSize() * zoomScale;
            var mult = spriteMult(size);
            var ok = true;
            if (room.hasUp) {
                ok = drawIcon('map/markers/exit-up.png', p.px + size / 2 - 5 * mult, p.py - size / 2 + 5 * mult, mult, now) && ok;
            }
            if (room.hasDown) {
                ok = drawIcon('map/markers/exit-down.png', p.px - size / 2 + 5 * mult, p.py + size / 2 - 5 * mult, mult, now) && ok;
            }
            return ok;
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
            ctx.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);
            ctx.clearRect(0, 0, viewW, viewH);
            ctx.fillStyle = mapSettings.mapBackground;
            ctx.fillRect(0, 0, viewW, viewH);

            var ROOM_SIZE = getRoomSize();
            var BASE_STEP = getBaseStep();
            var useCircle = (mapSettings.roomShape === 'circle');

            var tiles = tilesOn();
            drawn = { tiles: 0, fallbacks: 0, walls: 0, fog: 0, landmarks: 0, glyphs: 0, animated: 0, icons: 0, dots: 0, path: 0, shaded: 0 };
            var nowMs        = performance.now();
            var scaledSize   = ROOM_SIZE        * zoomScale;
            var scaledBorder = ROOM_BORDER_WIDTH * zoomScale;
            var scaledFont   = SYMBOL_FONT_SIZE  * zoomScale;
            var half         = scaledSize / 2;
            var tilePx       = Math.max(1, Math.round(scaledSize));
            var drewTiles    = {};   // room id -> its terrain tile was drawn
            var animTiles    = false;

            ctx.strokeStyle = mapSettings.connectionColor;
            ctx.lineWidth   = CONNECTION_WIDTH * zoomScale;
            ctx.lineCap     = 'round';

            // drawConnections draws corridor lines and their lock badges. In
            // the tiles style touching rooms need no line (only the badge);
            // a longer or diagonal exit keeps its line, drawn over the tiles.
            function drawConnections() {
                ctx.strokeStyle = mapSettings.connectionColor;
                ctx.lineWidth   = CONNECTION_WIDTH * zoomScale;
                ctx.lineCap     = 'round';
                edges.forEach(function (flags, key) {
                    var parts = key.split('-');
                    var rA = rooms.get(parseInt(parts[0], 10));
                    var rB = rooms.get(parseInt(parts[1], 10));
                    if (!rA || !rB) { return; }
                    var pA = gridToCanvas(rA.x, rA.y), pB = gridToCanvas(rB.x, rB.y);
                    var touching = Math.abs(rA.x - rB.x) + Math.abs(rA.y - rB.y) === 1;
                    if (!(tiles && touching)) {
                        ctx.beginPath(); ctx.moveTo(pA.px, pA.py); ctx.lineTo(pB.px, pB.py); ctx.stroke();
                    }
                    if (flags.locked || flags.secret) {
                        drawLineBadge((pA.px + pB.px) / 2, (pA.py + pB.py) / 2,
                                      flags.secret ? 'secret' : 'key');
                    }
                });
            }

            // drawStubs draws exits that lead to unvisited rooms: a short
            // line in the classic style; in tiles a fog tile at the far end
            // (or the line, while the fog art is not ready).
            function drawStubs(fogOnly) {
                var stubLen = BASE_STEP * zoomScale * 0.55;
                var fogged = {};
                zoneExitStubs.forEach(function (stub) {
                    var r = rooms.get(stub.roomId);
                    if (!r) { return; }
                    var p = gridToCanvas(r.x, r.y);
                    var len = Math.sqrt(stub.dx * stub.dx + stub.dy * stub.dy);
                    if (len === 0) { return; }
                    var fogReady = tiles && Sprites.status('map/terrain/fog.png') === 'ready';
                    if (fogReady) {
                        var fx = r.x + stub.dx, fy = r.y + stub.dy;
                        var occupied = coordIndex[fx + ',' + fy] !== undefined;
                        if (fogOnly && !occupied && !fogged[fx + ',' + fy]) {
                            fogged[fx + ',' + fy] = true;
                            var fp = gridToCanvas(fx, fy);
                            drawTileArt('map/terrain/fog.png', 0, fp.px, fp.py, tilePx, nowMs, false);
                            drawn.fog++;
                        }
                        if (!fogOnly && (stub.locked || stub.secret)) {
                            var fq = gridToCanvas(fx, fy);
                            drawLineBadge((p.px + fq.px) / 2, (p.py + fq.py) / 2, stub.secret ? 'secret' : 'key');
                        }
                        return;
                    }
                    if (fogOnly) { return; }
                    ctx.strokeStyle = mapSettings.connectionColor;
                    ctx.lineWidth   = CONNECTION_WIDTH * zoomScale;
                    ctx.lineCap     = 'round';
                    var ex = p.px + (stub.dx / len) * stubLen;
                    var ey = p.py + (stub.dy / len) * stubLen;
                    ctx.beginPath(); ctx.moveTo(p.px, p.py); ctx.lineTo(ex, ey); ctx.stroke();
                    if (stub.locked || stub.secret) {
                        drawLineBadge((p.px + ex) / 2, (p.py + ey) / 2, stub.secret ? 'secret' : 'key');
                    }
                });
            }

            var coordIndex = {};
            rooms.forEach(function (room, id) { coordIndex[room.x + ',' + room.y] = id; });

            if (!tiles) {
                drawConnections();
                drawStubs(false);
            }

            // Phase 40b: with a class sprite the current room keeps its terrain
            // colour and the sprite stands above it; without one it stays the
            // classic red square.
            var spriteOn = spritesOn() && currentRoomId !== null && unit.x !== null &&
                resolveSheet(chainKeys(identity.classid, identity.lineage), false) !== null;

            // Pass 1: terrain. A room with no tile art yet (loading, failed or
            // missing) draws its classic colour square so the map never blanks.
            rooms.forEach(function (room, id) {
                var p         = gridToCanvas(room.x, room.y);
                var isCurrent = (id === currentRoomId) && !spriteOn;
                if (tiles) {
                    var t = drawTerrain(room, id, p, tilePx, nowMs);
                    if (t) {
                        drewTiles[id] = true;
                        drawn.tiles++;
                        if (t.animated) { animTiles = true; drawn.animated++; }
                        if (isCurrent) {
                            ctx.strokeStyle = CURRENT_ROOM_COLOR;
                            ctx.lineWidth = Math.max(2, Math.round(scaledSize * 0.08));
                            ctx.strokeRect(Math.round(p.px - tilePx / 2) + 1, Math.round(p.py - tilePx / 2) + 1, tilePx - 2, tilePx - 2);
                        }
                        return;
                    }
                }
                if (tiles) { drawn.fallbacks++; }
                var fill      = isCurrent ? CURRENT_ROOM_COLOR : colorForSymbol(room.symbol, room.env);
                var rx = p.px - half, ry = p.py - half;
                ctx.fillStyle   = fill;
                ctx.strokeStyle = ROOM_BORDER_COLOR;
                ctx.lineWidth   = scaledBorder;

                if (useCircle && !tiles) {
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
                ctx.fillText(room.symbol || '•', p.px, p.py);
            });

            if (tiles) {
                drawStubs(true);   // fog sits on the terrain, under walls and landmarks
                drawn.walls = drawWalls(coordIndex, tilePx);
                drawConnections();
                drawStubs(false);
            }

            // Pass 2: landmark, resources, up/down marks.
            rooms.forEach(function (room, id) {
                var p         = gridToCanvas(room.x, room.y);
                var isCurrent = (id === currentRoomId) && !spriteOn;
                var tiled     = !!drewTiles[id];
                var fill      = isCurrent ? CURRENT_ROOM_COLOR : colorForSymbol(room.symbol, room.env);
                var symColor  = isCurrent ? CURRENT_ROOM_TEXT_COLOR
                    : (fill !== '#3a3a4a' ? contrastColor(fill) : SYMBOL_TEXT_COLOR);
                if (tiled) {
                    var mark = drawLandmark(room, id, p, tilePx, scaledFont, nowMs);
                    if (mark === 'landmark') { drawn.landmarks++; } else if (mark === 'glyph') { drawn.glyphs++; }
                }
                if (mapSettings.showResources) {
                    drawResources(id, p, tilePx, scaledSize, half, useCircle, tiles, tiled, symColor, nowMs);
                }
                if (room.hasUp || room.hasDown) {
                    if (tiled && drawChevrons(room, p, nowMs)) { return; }
                    var arrowSize = Math.max(5, scaledSize * 0.28);
                    ctx.font      = 'bold ' + arrowSize + 'px monospace';
                    ctx.fillStyle = isCurrent ? CURRENT_ROOM_TEXT_COLOR : (tiled ? '#ffffff' : symColor);
                    // For circles the bounding-box corners sit outside the circle.
                    // Inset from centre by half/√2 so the arrows stay inside.
                    var arrowInset = (useCircle && !tiles)
                        ? Math.max(2, half * 0.707 - arrowSize * 0.5)
                        : Math.max(2, scaledSize * 0.1);
                    if (room.hasDown) {
                        ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
                        ctx.fillText('▾', p.px - arrowInset, p.py + arrowInset);
                    }
                    if (room.hasUp) {
                        ctx.textAlign = 'right'; ctx.textBaseline = 'top';
                        ctx.fillText('▴', p.px + arrowInset, p.py - arrowInset);
                    }
                }
            });

            // Phase 40d: the time of day shades the outdoor tiles, and a walk
            // under way draws its path over them.
            if (tiles) { drawNightShade(drewTiles, tilePx); }
            var walkAnim = drawWalkPath(tilePx, nowMs);

            // Animated tiles (water, shore, swamp, snow, desert) cycle on a
            // timer while the map is drawn; reduced motion keeps them still.
            if ((animTiles || walkAnim) && !reducedMotion() && !document.hidden) { scheduleAnim(false); }

            drawCamps(nowMs, spriteOn);

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
                canvas.style.cursor = (id !== null && id !== currentRoomId) ? 'pointer' : '';
                if (info) { clearTimeout(tooltipHideTimer); showTooltip(e.clientX, e.clientY, info, id !== currentRoomId); }
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
                var isAdmin = !!charInfo && charInfo.role === 'admin';
                var rect = canvas.getBoundingClientRect();
                var id   = roomAtPoint(e.clientX - rect.left, e.clientY - rect.top);
                if (id === null) { return; }
                e.stopPropagation();
                // Phase 40d: a click on a room you can reach offers a walk there
                // (one pick confirms it); the server only walks a visited room.
                var items = [];
                if (id !== currentRoomId) {
                    var info = roomInfoStore.get(id);
                    items.push({ label: 'Walk to ' + ((info && info.name) || ('room ' + id)), cmd: 'walkto ' + id });
                }
                if (walkInfo) { items.push({ label: 'Stop walking', cmd: 'walkto stop' }); }
                if (isAdmin) {
                    items.push({ label: 'teleport ' + id, cmd: 'teleport ' + id },
                               { label: 'room info ' + id, cmd: 'room info ' + id });
                }
                if (items.length === 0) { return; }
                hideTooltip();
                uiMenu(e, items);
            });
            // Phase 40i: one finger pans the map, two pinch to zoom; a tap is
            // left to the click handler above. (Mouse drags use the handlers
            // above; this is for touch and pen only.)
            var touches = new Map();
            var pinchStart = null;
            canvas.style.touchAction = 'none';
            function touchPair() {
                var pts = Array.from(touches.values());
                return { dist: Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y) || 1 };
            }
            canvas.addEventListener('pointerdown', function (e) {
                if (e.pointerType === 'mouse') { return; }
                canvas.setPointerCapture(e.pointerId);
                touches.set(e.pointerId, { x: e.clientX, y: e.clientY, sx: e.clientX, sy: e.clientY });
                if (touches.size === 2) { pinchStart = { dist: touchPair().dist, zoom: zoomScale }; }
            });
            canvas.addEventListener('pointermove', function (e) {
                var t = touches.get(e.pointerId);
                if (!t) { return; }
                if (touches.size === 1) {
                    var step = getBaseStep() * zoomScale;
                    var dx = e.clientX - t.x, dy = e.clientY - t.y;
                    if (Math.abs(e.clientX - t.sx) > 6 || Math.abs(e.clientY - t.sy) > 6) { canvas.dataset.suppressClick = '1'; }
                    panOffsetX -= dx / step; panOffsetY -= dy / step;
                    t.x = e.clientX; t.y = e.clientY;
                    render();
                } else if (touches.size === 2 && pinchStart) {
                    t.x = e.clientX; t.y = e.clientY;
                    canvas.dataset.suppressClick = '1';
                    zoomScale = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, pinchStart.zoom * touchPair().dist / pinchStart.dist));
                    render();
                }
            });
            function touchEnd(e) {
                if (!touches.delete(e.pointerId)) { return; }
                if (pinchStart && touches.size < 2) {
                    pinchStart = null;
                    if (tilesOn()) { zoomScale = snapZoom(zoomScale); render(); }
                }
                if (touches.size === 0) { setTimeout(function () { delete canvas.dataset.suppressClick; }, 0); }
            }
            canvas.addEventListener('pointerup', touchEnd);
            canvas.addEventListener('pointercancel', touchEnd);
            canvas.addEventListener('wheel', function (e) {
                e.preventDefault();
                var factor = Math.pow(ZOOM_STEP, e.deltaY * 0.002);
                var next = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoomScale / factor));
                if (tilesOn()) {
                    // A wheel turn moves one step at a time so tiles stay crisp.
                    var from = snapZoom(zoomScale);
                    var i = TILE_ZOOMS.indexOf(from) + (next > zoomScale ? 1 : (next < zoomScale ? -1 : 0));
                    next = TILE_ZOOMS[Math.max(0, Math.min(TILE_ZOOMS.length - 1, i))];
                    if (Date.now() - lastWheelStep < 80) { return; }
                    lastWheelStep = Date.now();
                }
                zoomScale = next;
                render();
            }, { passive: false });

            var controls = document.createElement('div');
            controls.className = 'map-controls';
            var btnOut = document.createElement('button');
            btnOut.textContent = '\u2212'; btnOut.title = 'Zoom out';
            btnOut.addEventListener('click', function () {
                zoomScale = tilesOn() ? stepTileZoom(-1) : Math.max(ZOOM_MIN, zoomScale / ZOOM_STEP); render();
            });
            var btnIn = document.createElement('button');
            btnIn.textContent = '+'; btnIn.title = 'Zoom in';
            btnIn.addEventListener('click', function () {
                zoomScale = tilesOn() ? stepTileZoom(1) : Math.min(ZOOM_MAX, zoomScale * ZOOM_STEP); render();
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

            // Phase 40c: tiles (terrain art, landmarks, walls, fog) or the classic
            // coloured squares. Shape, size and spacing only apply to classic.
            var classicRows = [];
            function applyStyleRows() {
                classicRows.forEach(function (r) { r.style.display = tilesOn() ? 'none' : ''; });
            }
            panel.appendChild(row('Style', btnGroup(
                [{ label: 'Tiles', value: 'tiles' }, { label: 'Classic', value: 'classic' }],
                function () { return tilesOn() ? 'tiles' : 'classic'; },
                function (v) {
                    mapSettings.style = v;
                    if (v === 'tiles') { zoomScale = snapZoom(zoomScale); }
                    applyStyleRows();
                }
            )));

            var shapeRow = row('Shape', btnGroup(
                [{ label: 'Squares', value: 'square' }, { label: 'Circles', value: 'circle' }],
                function () { return mapSettings.roomShape; },
                function (v) { mapSettings.roomShape = v; }
            ));
            panel.appendChild(shapeRow);
            classicRows.push(shapeRow);

            panel.appendChild(row('Resources', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.showResources !== false; },
                function (v) { mapSettings.showResources = v; }
            )));

            panel.appendChild(row('Day/night', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.dayNight !== false; },
                function (v) { mapSettings.dayNight = v; }
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
            var sizeRow = row('Size', slider);
            panel.appendChild(sizeRow);
            classicRows.push(sizeRow);

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
            var spacingRow = row('Spacing', spacingSlider);
            panel.appendChild(spacingRow);
            classicRows.push(spacingRow);
            applyStyleRows();

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
                zoomScale = startZoom();
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
            // Phase 40i: the phone layout announces a view change with a
            // window resize; measure then too, in case the observer missed
            // the panel coming back into sight.
            window.addEventListener('resize', function () {
                if (container && container.clientWidth && canvas && (viewW !== container.clientWidth || viewH !== container.clientHeight || pixelRatio !== (window.devicePixelRatio || 1))) { resizeCanvas(); render(); }
            });
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
            setCompanions: function (list) {
                var same = list.length === companions.length && list.every(function (c, i) {
                    return c.key === companions[i].key && c.classid === companions[i].classid && c.lineage === companions[i].lineage;
                });
                if (same) { return; }
                companions = list; render();
            },
            setCamp: function (camp) { campInfo = camp; render(); },
            // Phase 40d: the walk under way ({} or null when none) and the clock.
            setWalk: function (walk) {
                var next = (walk && Array.isArray(walk.path) && walk.path.length > 0) ? { target: walk.target, path: walk.path.slice() } : null;
                if (JSON.stringify(next) === JSON.stringify(walkInfo)) { return; }
                walkInfo = next; render();
            },
            setTime: function (time) {
                timeInfo = time || null;
                var q = Math.round(nightLevel() * 10);
                if (q !== nightQuant) { render(); }
            },
            redraw: function () { render(); },
            // Phase 40i: the named places the map knows (a landmark legend),
            // nearest first, for the phone's "Walk to" list. The server still
            // decides whether the player may walk there.
            places: function () {
                var here = rooms.get(currentRoomId);
                var out = [];
                rooms.forEach(function (r, id) {
                    var info = roomInfoStore.get(id);
                    if (id === currentRoomId || !info || !info.maplegend || !info.name) { return; }
                    out.push({ id: id, name: info.name, legend: info.maplegend,
                               d: here ? Math.abs(r.x - here.x) + Math.abs(r.y - here.y) : 0 });
                });
                out.sort(function (a, b) { return a.d - b.d || a.id - b.id; });
                return out.slice(0, 14);
            },
            // Phase 47: every visited room whose name contains q (all of them
            // for an empty q), nearest first, for the phone's room search.
            search: function (q) {
                var needle = (q || '').trim().toLowerCase();
                var here = rooms.get(currentRoomId);
                var out = [];
                rooms.forEach(function (r, id) {
                    var info = roomInfoStore.get(id);
                    if (id === currentRoomId || !info || !info.name) { return; }
                    if (needle && info.name.toLowerCase().indexOf(needle) < 0) { return; }
                    out.push({ id: id, name: info.name, legend: info.maplegend || '',
                               d: here ? Math.abs(r.x - here.x) + Math.abs(r.y - here.y) : 0 });
                });
                out.sort(function (a, b) { return a.d - b.d || a.id - b.id; });
                return out.slice(0, 40);
            },
            walking: function () { return !!walkInfo; },
            viewport: function () { return { pan: [panOffsetX, panOffsetY], zoom: zoomScale }; },
            // pointOf is a room's centre in client pixels, for the browser checks.
            pointOf: function (roomId) {
                var room = rooms.get(roomId);
                if (!room || !canvas) { return null; }
                var p = gridToCanvas(room.x, room.y), r = canvas.getBoundingClientRect();
                return { x: r.left + p.px, y: r.top + p.py };
            },
            // state is for the browser checks (scripts/browser/map-check.mjs).
            state: function () {
                var res = resolveSheet(chainKeys(identity.classid, identity.lineage), false);
                var pose = unit.x === null ? null : unitPose(performance.now());
                return {
                    spriteDrawn: spritesOn() && !!res, face: unit.face, flip: unit.flip, walking: !!(pose && pose.walking),
                    queued: unit.queue.length, companySize: companySize,
                    companions: companions.map(function (c) {
                        return { key: c.key, sprite: spritesOn() && resolveSheet(chainKeys(c.classid, c.lineage), false) !== null };
                    }),
                    style: tilesOn() ? 'tiles' : 'classic', drawn: drawn, zoom: zoomScale,
                    camp: campInfo, fading: unit.fadeStart >= 0,
                    walk: walkInfo, night: nightLevel(),
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
    Sprites.data('map/landmarks.json'); // start the landmark table loading
    window.MapView = { state: function () { return view2d.state(); }, pointOf: function (id) { return view2d.pointOf(id); } };

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

    // Phase 40c: a room's resources changed (picked clean, or regrown). The
    // map keeps the last info it was given for each room, so patch it and
    // redraw instead of waiting for the next visit.
    function updateWorldResources() {
        var w = Client.GMCPStructs.World;
        var body = w && w.Resources;
        var info = body && roomInfoStore.get(body.num);
        if (!info) { return; }
        info.resources = Array.isArray(body.resources) ? body.resources : [];
        info.depleted = Array.isArray(body.depleted) ? body.depleted : [];
        view2d.redraw();
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

    // Phase 40d: the planned walk (Walkto) and the game's time of day.
    function updateWalk() {
        view2d.setWalk(Client.GMCPStructs.Walkto || null);
    }

    function updateTime() {
        view2d.setTime(Client.GMCPStructs.Gametime || null);
    }

    function updateCompany() {
        var co = Client.GMCPStructs.Company;
        var size = 0;
        if (co && co.leader && Array.isArray(co.members)) {
            var withLeader = co.members.filter(function (m) { return m && m.status === 'present'; }).length;
            size = withLeader > 0 ? withLeader + 1 : 0;
        }
        view2d.setCompanySize(size);
        var present = [];
        if (co && Array.isArray(co.members)) {
            co.members.forEach(function (m) {
                if (m && m.status === 'present' && (m.class || m.lineage)) {
                    present.push({ key: m.key, lineage: m.lineage || '', classid: m.class || '' });
                }
            });
        }
        view2d.setCompanions(present);
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

    // Phase 40i: the phone's touch bar asks the map for places to walk to.
    window.MapPlaces = {
        list:    function () { return view2d.places(); },
        search:  function (q) { return view2d.search(q); },
        walking: function () { return view2d.walking(); },
        viewport: function () { return view2d.viewport(); },
    };

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Room', 'World', 'Party', 'Party.Vitals', 'Char', 'Company', 'Walkto', 'Gametime'],
        onGMCP: function (namespace) {
            if (namespace === 'Char.Info' || namespace === 'Char') {
                updateIdentity();
            } else if (namespace.indexOf('Company') === 0) {
                updateCompany();
            } else if (namespace === 'Walkto') {
                updateWalk();
            } else if (namespace === 'Gametime') {
                updateTime();
            } else if (namespace === 'World.Map') {
                updateWorldMap();
            } else if (namespace === 'World.Resources') {
                updateWorldResources();
            } else if (namespace === 'Room.Info' || namespace === 'Room') {
                updateMap();
            } else if (namespace === 'Party' || namespace === 'Party.Vitals') {
                updatePartyPositions();
            }
        },
    });

}());
