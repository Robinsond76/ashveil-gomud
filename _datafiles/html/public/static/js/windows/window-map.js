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
        weather:         true,       // LM4: rain, storms, fog, clouds and snow over outdoor tiles
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
                    exitIds.push({ num: exitInfo.num, locked: isLocked, secret: isSecret, dx: exitInfo.dx, dy: exitInfo.dy, dz: exitInfo.dz });
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

    var tooltip = Client.tooltip('map-tooltip');

    function showTooltip(mouseX, mouseY, info, canWalk) {
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

        tooltip.show(html, { x: mouseX, y: mouseY });
    }

    function positionTooltip(mouseX, mouseY) { tooltip.position({ x: mouseX, y: mouseY }); }

    function hideTooltip() { tooltip.hide(); }

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
        // LM3: the place title shown when you cross into a zone or level.
        '.map-crossing-title { position:absolute; left:0; right:0; top:28%; text-align:center; pointer-events:none; z-index:9; opacity:0; transition:opacity 0.4s ease; color:#f2e6c8; text-shadow:0 1px 3px #000, 0 0 8px #000; }',
        '.map-crossing-title.show { opacity:1; }',
        '.map-crossing-title .mct-name { font-size:1.35em; font-weight:bold; letter-spacing:0.04em; }',
        '.map-crossing-title .mct-level { font-size:0.85em; margin-top:2px; color:#d8ccb0; }',
        '@media (prefers-reduced-motion: reduce) { .map-crossing-title { transition:none; } }',
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
        var lakeCells     = [];   // E2: open water a ring of shore encloses ({ x, y }), from refreshTiling
        var onwardCache   = {};   // E2: exits that leave the drawn map ('id,dx,dy'), from refreshTiling
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
        // LM3: crossing into a zone or level fades the old map to black, then
        // the new one in, under a title naming the place for about 2 s.
        var CROSS_OUT_MS    = 300;
        var CROSS_IN_MS     = 450;
        var CROSS_TITLE_MS  = 2000;
        var crossing = null;        // { start, old (canvas snapshot), name, level }
        var crossTitle = null;      // the title element
        var crossTitleTimer = null;
        var lastCrossing = null;    // the last crossing's title (browser checks)
        var ALLY_SCALE      = 0.75; // allied class sprites at 75% of the player's
        var identity    = { classid: '', lineage: '', look: null }; // Char.Info (look: Phase 72a skin and hair colours)
        var companySize = 0;        // members with the leader, 0 when alone or unknown
        var campInfo    = null;     // Company.Camp
        var lastCampAsYou = false;  // LM1: the last render drew your camp as your marker (browser checks)
        var lastFigures = { you: false, companions: 0 };  // what drawUnit last drew (browser checks)
        var lastFire = '';          // the last camp fire drawn: lit, bonfire (your camp as you), embers or unlit (browser checks)
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
        var drawn = { tiles: 0, fallbacks: 0, walls: 0, fog: 0, landmarks: 0, glyphs: 0, animated: 0, icons: 0, dots: 0, path: 0, shaded: 0, pieces: {}, corners: {}, replaced: 0, lakes: 0, remembered: 0 }; // what the last render drew (browser checks)
        var walkInfo = null;   // Phase 40d: Walkto { target, path: [room ids ahead] } while a walk is under way
        var timeInfo = null;   // Phase 40d: Gametime, for the day/night shading
        var sightKey = '';     // LM2: MapSight.key of the last sight drawn
        var weatherKey = '';   // LM4: MapWeather.key of the last weather drawn
        var weatherTex = {};   // LM4: code-drawn stand-ins for the A14 textures, by name
        var patternCache = new Map(); // LM4: a canvas pattern per texture image
        var storm = { next: 0, start: -1, tile: null, strikes: 0 }; // LM4: the lightning under way
        var lastWeather = null; // LM4: what the last render drew of the weather (browser checks)
        var lastSight = null;  // LM2: what the last render drew of sight (browser checks)
        var heldSight = null;  // LM2: the sight last drawn for the room you are in
        var figureDim = 1;     // LM2: your figure is faint when you can't see your own room
        var darkCanvas = null; // LM2: the darkness layer, cut by your glow
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
        // look (Phase 72a) repaints the figure's skin and hair in a player's
        // chosen colours; companions and creatures keep their art. The
        // imported high-density art is recoloured through its A11 look mask
        // (E1b, Sprites.tinted); a class with no mask is drawn as is.
        function resolveSheet(keys, walking, look) {
            for (var i = 0; i < keys.length; i++) {
                var idle = 'map/units/' + keys[i] + '/idle.png';
                var walk = 'map/units/' + keys[i] + '/walk.png';
                var st = Sprites.status(idle);
                if (st === 'ready') {
                    if (Sprites.status(walk) === 'ready' && walking) { return { sheet: Sprites.tinted(walk, look), walk: true }; }
                    return { sheet: Sprites.tinted(idle, look), walk: false };
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

        // snap (E1) rounds a CSS coordinate to a whole device pixel, so a
        // figure's edges fall on device pixels at any pixel ratio. Sizes are
        // left alone: rounding them would resize 1x markers and companions
        // against the figures they sit beside.
        function snap(v) { return Math.round(v * pixelRatio) / pixelRatio; }

        // A sheet with density N has frames N times the 1x size; it is
        // drawn at the same on-map size as 1x art (mult is per 1x pixel).
        function drawFrame(sheet, row, flip, cx, feetY, mult, now, startMs, alpha) {
            var info = sheet.info;
            var r = Math.max(0, (info.rows || []).indexOf(row));
            var f = Sprites.frame(info, r, now, startMs);
            var d = info.density || 1;
            var w = f.sw * mult / d, h = f.sh * mult / d;
            var x = snap(cx - w / 2);
            var y = snap(feetY - h * ((info.feet_baseline || f.sh) / f.sh));
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
            ctx.drawImage(a.img, f.sx, f.sy, f.sw, f.sh, snap(cx - w / 2), snap(cy - h / 2), w, h);
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
        function fireAndRest(cx, cy, mult, lit, resting, now, at, embers, big) {
            var fx = at ? at.px : cx + 8 * mult, fy = at ? at.py : cy + 8 * mult;
            lastFire = lit ? 'lit' : (embers ? 'embers' : 'unlit');
            if (lit && big && drawIcon('map/camp/bonfire.png', fx, fy - 3 * mult, mult * 0.75, now)) {
                // A13: your camp as your marker burns a bigger fire.
                lastFire = 'bonfire';
                drawIcon('map/camp/smoke.png', fx, fy - 18 * mult, mult, now);
            } else if (lit) {
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
        // asYou (LM1) draws your camp as your marker: the tent a quarter
        // larger, standing a little back, with the fire in front of it.
        function drawCamp(roomId, ally, lit, resting, innRest, now, occupied, embers, tent, asYou) {
            var r = rooms.get(roomId);
            if (!r) { return; }
            var p = gridToCanvas(r.x, r.y);
            var mult = spriteMult(getRoomSize() * zoomScale);
            var fire = null;
            var tentMult = mult;
            if (asYou && !innRest) {
                tentMult = mult * 1.25;
                fire = { px: p.px + 5 * mult, py: p.py + 10 * mult };
                p = { px: p.px, py: p.py - 3 * mult };
            }
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
            if (!drawIcon(pitch, p.px, p.py, tentMult, now)) {
                // no art yet: a small tent triangle
                var q = getRoomSize() * zoomScale * 0.4;
                ctx.fillStyle = ally ? '#6a9ec9' : '#c9a15a';
                ctx.beginPath(); ctx.moveTo(p.px, p.py - q); ctx.lineTo(p.px + q, p.py + q); ctx.lineTo(p.px - q, p.py + q); ctx.closePath(); ctx.fill();
            }
            fireAndRest(p.px, p.py, mult, lit, resting, now, fire, embers, asYou);
        }

        // campAsYou (living map LM1) says your company is camped in the room
        // you stand in, and you have stopped walking: the camp is then your
        // marker and your figure and companions are not drawn (they are
        // round the fire). An inn rest and a hidden camp layer keep you.
        // Standing in the camp room is read from Room.Info (room_id is the
        // room you stand in), not Company.Camp's here, which can lag a move.
        function campAsYou(pose) {
            return !!(campInfo && mapSettings.showCamp !== false && campInfo.has_camp &&
                !campInfo.inn && currentRoomId !== null && campInfo.room_id === currentRoomId && pose && !pose.walking);
        }

        // drawCamps draws your camp and your party's. spriteOn says your class
        // sprite stands on your tile, so a camp there is drawn beside it;
        // asYou (LM1) says your camp is your marker, drawn on the tile itself.
        // Your own camp in the room you stand in is never pushed aside: you
        // are walking into it or already are it (LM1).
        function drawCamps(now, spriteOn, asYou, alpha, outOfSight) {
            if (!campInfo || mapSettings.showCamp === false) { return; }
            var ownHere = campInfo.has_camp && !campInfo.inn && campInfo.room_id === currentRoomId;
            (campInfo.allied_camps || []).forEach(function (c) {
                // LM2: an ally's camp out of sight isn't drawn (yours is: you know where it is).
                if (outOfSight && outOfSight(c.room_id)) { return; }
                drawCamp(c.room_id, true, !!c.fire_lit, !!c.resting, false, now, !asYou && spriteOn && c.room_id === currentRoomId, !!c.embers, c.tent);
            });
            if (campInfo.has_camp && campInfo.room_id) {
                var inn = !!(campInfo.here && campInfo.inn && campInfo.resting);
                ctx.save();
                if (asYou) { ctx.globalAlpha = alpha; }
                drawCamp(campInfo.room_id, false, !!campInfo.fire_lit, !!campInfo.resting, inn, now, !ownHere && spriteOn && campInfo.room_id === currentRoomId, !!campInfo.embers, campInfo.tent, asYou);
                ctx.restore();
            }
        }

        // drawUnit draws the player's marker above the terrain: the here-ring,
        // the class sprite, then the company badge. It returns false when
        // there is no sprite to draw (off, still loading, or no art), so the
        // caller shows the classic red square.
        function drawUnit(now, asYou) {
            lastFigures = { you: false, companions: 0 };
            if (!spritesOn() || currentRoomId === null || unit.x === null) { return false; }
            var pose = unitPose(now);
            var res = resolveSheet(chainKeys(identity.classid, identity.lineage), pose.walking, identity.look);
            if (!res) { return false; }
            var tile = getRoomSize() * zoomScale;
            var mult = spriteMult(tile);
            var p = gridToCanvas(pose.x, pose.y);
            var alpha = unitAlpha(now);
            if (alpha >= 1) { unit.fadeStart = -1; }
            alpha *= figureDim;
            ctx.save();
            ctx.globalAlpha = alpha;
            if (!drawIcon('map/markers/here-ring.png', p.px, p.py + 8 * mult, mult, now)) {
                ctx.strokeStyle = '#ffd24a'; ctx.lineWidth = Math.max(1, mult * 2);
                ctx.strokeRect(p.px - tile / 2, p.py - tile / 2, tile, tile);
            }
            ctx.restore();
            if (asYou) {
                // LM1: you are the camp. The ring marks your tile; the badge
                // sits on the tent.
                ctx.save(); ctx.globalAlpha = alpha;
                drawBadge(p, mult, now);
                ctx.restore();
                lastFigures = { you: false, companions: 0 };
                return { walking: false, camp: true };
            }
            var drawnCompanions = 0;
            // Companions stand in behind you and move with you.
            companions.slice(0, COMPANION_SLOTS.length).forEach(function (c, i) {
                var cres = resolveSheet(chainKeys(c.classid, c.lineage), pose.walking);
                if (!cres) { return; }
                var slot = COMPANION_SLOTS[i], cm = mult * ALLY_SCALE;
                drawFrame(cres.sheet, unit.face, unit.flip, p.px + slot.x * mult, p.py + (12 - slot.y) * mult, cm, now, pose.walking ? pose.start : 0, alpha);
                drawnCompanions++;
            });
            drawFrame(res.sheet, unit.face, unit.flip, p.px, p.py + 12 * mult, mult, now, pose.walking ? pose.start : 0, alpha);
            ctx.save(); ctx.globalAlpha = alpha;
            drawBadge(p, mult, now);
            ctx.restore();
            lastFigures = { you: true, companions: drawnCompanions };
            return { walking: pose.walking };
        }

        // unitAlpha is how far your marker has faded in after a level or
        // zone change (1 when it hasn't been faded).
        function unitAlpha(now) {
            return unit.fadeStart >= 0 ? Math.min(1, (now - unit.fadeStart) / FADE_MS) : 1;
        }

        // drawBadge draws the company badge (how many are with you) over
        // your marker; it is hidden when you travel alone.
        function drawBadge(p, mult, now) {
            if (companySize <= 1) { return; }
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

        // drawAllySprite draws a party member as a class sprite at 75% with
        // an ally pennant. It returns false when there is no art to draw, so
        // the caller keeps the heart.
        function drawAllySprite(ease, gx, gy, moving, now) {
            if (!ease.classid && !ease.lineage) { return false; }
            var res = resolveSheet(chainKeys(ease.classid, ease.lineage), moving, ease.look);
            if (!res) { return false; }
            var p = gridToCanvas(gx, gy);
            var mult = spriteMult(getRoomSize() * zoomScale) * ALLY_SCALE;
            drawFrame(res.sheet, ease.face || 'down', !!ease.flip, p.px, p.py + 12 * mult, mult, now, ease.startTime, 1);
            drawIcon('map/markers/ally-banner.png', p.px + 9 * mult, p.py - 10 * mult, mult, now);
            return true;
        }

        // animKind is what animTimer holds: 'frame' (a requestAnimationFrame)
        // or 'timer' (a timeout). A request for every frame replaces a
        // pending timeout (LM4 review: rain must not drop to the slow timer).
        var animKind = null;
        var lastAnim = '';     // the last kind of redraw scheduled (browser checks)
        function scheduleAnim(walking, delayMs) {
            if (!container || !container.isConnected) { return; }
            if (animTimer !== null) {
                if (!(walking && animKind === 'timer')) { return; }
                clearTimeout(animTimer);
                animTimer = null;
            }
            // A hidden map (another phone view, a closed tab) checks back
            // once a second instead of drawing every frame.
            if (container.offsetParent === null) { walking = false; delayMs = 1000; }
            if (walking) {
                animKind = 'frame';
                animTimer = requestAnimationFrame(function () { animTimer = null; animKind = null; render(); });
            } else {
                animKind = 'timer';
                animTimer = setTimeout(function () { animTimer = null; animKind = null; render(); }, delayMs || 120);
            }
            lastAnim = animKind;
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
            // Setting a canvas's size clears it, even to the same size: only
            // set it when it changes, so the map last drawn survives (LM3
            // fades it out on a crossing).
            var w = Math.round(viewW * pixelRatio), h = Math.round(viewH * pixelRatio);
            if (canvas.width !== w)  { canvas.width = w; }
            if (canvas.height !== h) { canvas.height = h; }
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
            rooms.clear(); edges.clear(); zoneExitStubs = []; lakeCells = []; onwardCache = {};
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
            refreshTiling(zoneKey);
        }

        // refreshTiling (E2) recomputes what road and coast pieces and lakes
        // depend on, whenever the drawn rooms change (a zone replay or a new
        // room in the same zone).
        function refreshTiling(zoneKey) {
            onwardCache = onwardExits();
            lakeCells = findLakes(zoneKey);
        }

        // findLakes (E2) finds the open water a ring of shore rooms encloses
        // (MapTiles.lakes), from this zone's rooms only (a visited room of
        // another zone an exit reaches is drawn at its own zone's grid
        // place). A cell any exit leads into holds a room not seen yet, so
        // it is never a lake.
        function findLakes(zoneKey) {
            if (!window.MapTiles) { return []; }
            var cells = [], blocked = {};
            rooms.forEach(function (room, id) {
                var rc = roomCache[id];
                if (!rc || rc.zoneName + '/z:' + rc.z !== zoneKey) { return; }
                cells.push({ x: room.x, y: room.y, env: room.env });
                (rc.stubs || []).concat(rc.exits || []).forEach(function (e) {
                    if (e.dz === 0 && typeof e.dx === 'number' && typeof e.dy === 'number') {
                        blocked[(room.x + e.dx) + ',' + (room.y + e.dy)] = true;
                    }
                });
            });
            return window.MapTiles.lakes(cells, blocked).map(function (k) {
                var xy = k.split(',');
                return { x: parseInt(xy[0], 10), y: parseInt(xy[1], 10) };
            });
        }

        // -- Phase 40c: terrain tiles, walls, landmarks --------------------------
        // drawTileArt draws one frame of <path> (32 px at 1x, density times
        // that for imported art) centred on (px, py) at <size> px, smoothed
        // only for high-density art. <variant> picks a column of a variants
        // sheet; <animated> instead picks the frame for <now>, its loop
        // starting at <startMs>. It reports whether the image was ready.
        function drawTileArt(path, variant, px, py, size, now, animated, startMs) {
            var a = Sprites.art(path);
            if (!a) { return false; }
            var fw = a.info.frame[0], fh = a.info.frame[1];
            var f = animated ? Sprites.frame(a.info, 0, now, startMs || 0) : { sx: variant * fw, sy: 0, sw: fw, sh: fh };
            ctx.save();
            artSmoothing(a.info);
            ctx.drawImage(a.img, f.sx, f.sy, f.sw, f.sh, Math.round(px - size / 2), Math.round(py - size / 2), size, size);
            ctx.restore();
            return true;
        }

        // onwardExits lists, as 'id,dx,dy', every exit on the drawn level:
        // into the fog of an unvisited room, or into a visited room. Where the
        // neighbouring cell on that side has no room drawn (tileLook), the
        // exit leaves the map there (a room of another zone sits at its own
        // zone's grid place), and a road runs on through it (MapTiles); a
        // secret passage doesn't count.
        function onwardExits() {
            var out = {};
            rooms.forEach(function (room, id) {
                var rc = roomCache[id];
                if (!rc) { return; }
                (rc.stubs || []).forEach(function (s) {
                    if (s.dz === 0 && !s.secret) { out[id + ',' + s.dx + ',' + s.dy] = true; }
                });
                (rc.exits || []).forEach(function (e) {
                    if (e.dz === 0 && !e.secret && typeof e.dx === 'number') {
                        out[id + ',' + e.dx + ',' + e.dy] = true;
                    }
                });
            });
            return out;
        }

        // tileLook describes a road or coast room's four neighbours for
        // MapTiles (E2): their ground (a lake cell is water), whether an exit
        // joins them (a secret passage doesn't), and whether an exit leaves
        // the drawn map there.
        function tileLook(room, id, coordIndex, onward, lakeIndex) {
            function at(dx, dy) { return coordIndex[(room.x + dx) + ',' + (room.y + dy)]; }
            return {
                envAt: function (dx, dy) {
                    var o = at(dx, dy);
                    if (o === undefined) { return lakeIndex[(room.x + dx) + ',' + (room.y + dy)] ? 'water' : ''; }
                    var r = rooms.get(o);
                    return r ? (r.env || '') : '';
                },
                linked: function (dx, dy) {
                    var o = at(dx, dy);
                    if (o === undefined) { return false; }
                    var e = edges.get(id < o ? (id + '-' + o) : (o + '-' + id));
                    return !!e && !e.secret;
                },
                onward: function (dx, dy) {
                    return at(dx, dy) === undefined && !!onward[id + ',' + dx + ',' + dy];
                }
            };
        }

        // lakeId is a lake cell's stand-in room id: a hash of its place, so
        // its variant and phase look random rather than striped.
        function lakeId(x, y) { return ((x * 73856093) ^ (y * 19349663)) >>> 0; }

        // drawTerrain draws a room's biome tile (the variant is the room id
        // mod 3, the same for every viewer) and its animation. A road or
        // coast room draws the piece its neighbours pick (MapTiles) when that
        // art is listed, else its biome's variants. An animation whose sheet
        // says replace (E2) holds whole tiles drawn from variant 1, so only
        // rooms showing variant 1 play it, each at its own phase, and the
        // others keep their still variants; an older animation is an overlay
        // on every room. It returns null while the art is not ready, else
        // { animated }.
        function drawTerrain(room, id, p, size, now, coordIndex, onward, lakeIndex, still) {
            var biome = room.env ? room.env : 'default';
            var mt = window.MapTiles;
            var look = (mt && mt.tiled(biome)) ? tileLook(room, id, coordIndex, onward, lakeIndex) : null;
            var name = look ? mt.piece(biome, look) : '';
            if (name) {
                var pp = 'map/terrain/' + name + '.png';
                var ps = Sprites.status(pp);
                if (ps === 'loading') { return null; }
                if (ps === 'ready') {
                    drawTileArt(pp, 0, p.px, p.py, size, now, false);
                    drawn.pieces[id] = name;
                    // E2b: a coast bending round a lake's corner gets the
                    // inside-corner water over its piece (an overlay that
                    // hasn't loaded yet is simply left off this frame).
                    (mt.corners ? mt.corners(biome, look) : []).forEach(function (c) {
                        var cp = 'map/terrain/shore-corner-' + c + '.png';
                        if (Sprites.status(cp) !== 'ready') { return; }
                        drawTileArt(cp, 0, p.px, p.py, size, now, false);
                        (drawn.corners[id] = drawn.corners[id] || []).push(c);
                    });
                    return { animated: false };
                }
            }
            var path = 'map/terrain/' + biome + '.png';
            if (Sprites.status(path) === 'none') {
                // A biome with no art (or none yet, before the manifest loads).
                if (!Sprites.has('map/terrain/unknown.png')) { return null; }
                path = 'map/terrain/unknown.png';
            }
            var a = Sprites.art(path);
            if (!a) { return null; }
            var variants = Math.max(1, a.info.variants || 1);
            var variant = ((id % variants) + variants) % variants;
            // A remembered room (LM2: out of sight) is drawn still.
            var anim = (a.info.animated_overlay && !reducedMotion() && !still) ? 'map/terrain/' + a.info.animated_overlay : '';
            var an = anim ? Sprites.art(anim) : null;
            if (an && an.info.replace) {
                if (variant === 0) {
                    var n = Math.max(1, an.info.frames || 1);
                    drawTileArt(anim, 0, p.px, p.py, size, now, true, -(((id % n) + n) % n) * (an.info.frame_ms || 0));
                    drawn.replaced++;
                    return { animated: true };
                }
                drawTileArt(path, variant, p.px, p.py, size, now, false);
                return { animated: false };
            }
            drawTileArt(path, variant, p.px, p.py, size, now, false);
            if (anim) { drawTileArt(anim, 0, p.px, p.py, size, now, true, 0); }
            return { animated: !!anim };
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
        function drawNightShade(drewTiles, tilePx, drewLakes) {
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
            var water = biomeTable.water;
            if (!(water && (water.indoor || water.dark))) {
                (drewLakes || []).forEach(function (c) {
                    var p = gridToCanvas(c.x, c.y);
                    ctx.fillRect(Math.round(p.px - tilePx / 2), Math.round(p.py - tilePx / 2), tilePx, tilePx);
                    drawn.shaded++;
                });
            }
            ctx.restore();
        }

        // joinedTo lists the drawn rooms an exit joins to room id (LM2: at full
        // sight you see them from where you stand).
        function joinedTo(id) {
            if (id === null || id === undefined) { return []; }
            var out = [];
            edges.forEach(function (flags, key) {
                var parts = key.split('-');
                var a = parseInt(parts[0], 10), b = parseInt(parts[1], 10);
                var other = a === id ? b : (b === id ? a : null);
                if (other !== null && rooms.has(other)) { out.push(other); }
            });
            return out;
        }

        // LM2 darkness: how dark the darkness layer makes a room in sight
        // (outside the glow), a remembered room, and your own room when you
        // can't see it.
        var SEEN_DARK = 0.5, REMEMBERED_DARK = 0.78, BLIND_DARK = 0.9;
        var FOG_VEIL = 0.5;   // fog by day veils lighter than night darkens
        var LIGHT_TINT = { warm: 'rgba(255,150,50,0.32)', cool: 'rgba(150,190,255,0.22)', pale: 'rgba(190,200,230,0.12)' };

        // drawSight (LM2) replaces the clock's night shade when you can't see
        // the whole map: remembered rooms lose their colour and sink into the
        // dark; the rooms in sight are dimmed but for a soft pool of light round
        // you (warm for a flame, cool for a party light, pale for the moon),
        // and a lit campfire is a pool of firelight wherever it is. Night
        // vision has no glow: what you see turns a cool monochrome.
        function drawSight(inSight, glow, tilePx, lakes, nowMs) {
            var cells = [];
            rooms.forEach(function (room, id) {
                cells.push({ p: gridToCanvas(room.x, room.y), seen: inSight.has(id), here: id === currentRoomId });
            });
            lakes.forEach(function (c) { cells.push({ p: gridToCanvas(c.x, c.y), seen: c.seen, here: false }); });
            function cell(c2, c) {
                c2.fillRect(Math.round(c.p.px - tilePx / 2), Math.round(c.p.py - tilePx / 2), tilePx, tilePx);
            }
            ctx.save();
            ctx.globalCompositeOperation = 'saturation';
            ctx.fillStyle = '#808080';
            cells.forEach(function (c) { if (!c.seen || glow.mono) { cell(ctx, c); } });
            if (glow.mono) {
                ctx.globalCompositeOperation = 'multiply';
                ctx.fillStyle = 'rgb(170,215,190)';
                cells.forEach(function (c) { if (c.seen) { cell(ctx, c); } });
            }
            ctx.restore();

            if (!darkCanvas) { darkCanvas = document.createElement('canvas'); }
            if (darkCanvas.width !== canvas.width || darkCanvas.height !== canvas.height) {
                darkCanvas.width = canvas.width; darkCanvas.height = canvas.height;
            }
            var d = darkCanvas.getContext('2d');
            d.setTransform(1, 0, 0, 1, 0, 0);
            d.globalCompositeOperation = 'source-over';
            d.clearRect(0, 0, darkCanvas.width, darkCanvas.height);
            d.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);
            // Fog by day is a pale veil, not night.
            var veil = glow.fog ? 'rgba(168,174,182,' : 'rgba(5,7,14,';
            cells.forEach(function (c) {
                var a = c.seen ? (glow.mono ? 0.2 : SEEN_DARK) : (c.here ? BLIND_DARK : REMEMBERED_DARK);
                d.fillStyle = veil + (glow.fog ? a * FOG_VEIL : a) + ')';
                cell(d, c);
                drawn.shaded++;
            });

            var lights = [];
            var centre = glowCentre(nowMs);
            if (glow.radius > 0 && centre) { lights.push({ x: centre.x, y: centre.y, r: glow.radius * tilePx, color: glow.color }); }
            if (campInfo && campInfo.has_camp && campInfo.fire_lit && !campInfo.inn && rooms.has(campInfo.room_id)) {
                var cr = rooms.get(campInfo.room_id), cp = gridToCanvas(cr.x, cr.y);
                lights.push({ x: cp.px, y: cp.py, r: 0.9 * tilePx, color: 'warm' });
            }
            // An ally's lit campfire in sight lights its tile too.
            if (campInfo && mapSettings.showCamp !== false) {
                (campInfo.allied_camps || []).forEach(function (ac) {
                    if (!ac.fire_lit || !rooms.has(ac.room_id) || !inSight.has(ac.room_id)) { return; }
                    var ar = rooms.get(ac.room_id), ap = gridToCanvas(ar.x, ar.y);
                    lights.push({ x: ap.px, y: ap.py, r: 0.9 * tilePx, color: 'warm' });
                });
            }
            var flicker = reducedMotion() ? 1 : 1 + 0.03 * Math.sin(nowMs / 140) + 0.02 * Math.sin(nowMs / 57);
            d.globalCompositeOperation = 'destination-out';
            lights.forEach(function (l) {
                var r = l.r * (l.color === 'warm' ? flicker : 1);
                var g = d.createRadialGradient(l.x, l.y, 0, l.x, l.y, r);
                g.addColorStop(0, 'rgba(0,0,0,1)');
                g.addColorStop(0.55, 'rgba(0,0,0,0.75)');
                g.addColorStop(1, 'rgba(0,0,0,0)');
                d.fillStyle = g;
                d.fillRect(l.x - r, l.y - r, 2 * r, 2 * r);
            });
            d.globalCompositeOperation = 'source-over';
            ctx.save();
            ctx.setTransform(1, 0, 0, 1, 0, 0);
            ctx.drawImage(darkCanvas, 0, 0);
            ctx.restore();
            ctx.save();
            lights.forEach(function (l) {
                if (!LIGHT_TINT[l.color]) { return; }
                var g = ctx.createRadialGradient(l.x, l.y, 0, l.x, l.y, l.r);
                g.addColorStop(0, LIGHT_TINT[l.color]);
                g.addColorStop(1, 'rgba(0,0,0,0)');
                ctx.fillStyle = g;
                ctx.fillRect(l.x - l.r, l.y - l.r, 2 * l.r, 2 * l.r);
            });
            ctx.restore();
            drawn.lights = lights.length;
        }

        // glowCentre is where your light is: your figure (as it walks), else
        // your room.
        function glowCentre(nowMs) {
            if (unit.x !== null) {
                var pose = unitPose(nowMs);
                var p = gridToCanvas(pose.x, pose.y);
                return { x: p.px, y: p.py };
            }
            var r = rooms.get(currentRoomId);
            if (!r) { return null; }
            var q = gridToCanvas(r.x, r.y);
            return { x: q.px, y: q.py };
        }

        // -- LM4: weather ------------------------------------------------------
        // weatherArt is an A14 texture once imported, else a code-drawn
        // stand-in (a small canvas, made once), so the weather shows before
        // the art comes.
        function weatherArt(tex) {
            var a = Sprites.art('map/weather/' + tex + '.png');
            if (a) { return { img: a.img, art: true }; }
            if (!weatherTex[tex]) { weatherTex[tex] = codeTexture(tex); }
            return { img: weatherTex[tex], art: false };
        }

        // codeTexture draws a seamless stand-in texture: 64 px for a one-tile
        // texture, 128 for a two-tile one, on a 2 px grain like the art.
        function codeTexture(tex) {
            var big = tex === 'fog' || tex === 'clouds';
            var n = big ? 128 : 64;
            var c = document.createElement('canvas');
            c.width = n; c.height = n;
            var g = c.getContext('2d');
            var seed = 7;
            function rnd() { seed = (seed * 16807) % 2147483647; return seed / 2147483647; }
            function px(x, y, w, h) {
                // Wrap round the edges so the texture tiles seamlessly.
                for (var dx = -n; dx <= n; dx += n) {
                    for (var dy = -n; dy <= n; dy += n) { g.fillRect(x + dx, y + dy, w, h); }
                }
            }
            var i, x, y;
            if (tex === 'rain' || tex === 'rain-heavy') {
                g.fillStyle = 'rgb(176,196,214)';
                var count = tex === 'rain' ? 22 : 40, len = tex === 'rain' ? 4 : 7;
                for (i = 0; i < count; i++) {
                    x = Math.floor(rnd() * n / 2) * 2; y = Math.floor(rnd() * n / 2) * 2;
                    // A streak slanting down to the left, about 20 degrees.
                    for (var k = 0; k < len; k++) { px(x - Math.round(k * 0.7), y + k * 2, 2, 2); }
                }
            } else if (tex === 'snow' || tex === 'snow-far') {
                var far = tex === 'snow-far';
                for (i = 0; i < (far ? 30 : 16); i++) {
                    x = Math.floor(rnd() * n / 2) * 2; y = Math.floor(rnd() * n / 2) * 2;
                    g.fillStyle = rnd() < 0.3 ? 'rgb(214,224,238)' : 'rgb(236,238,242)';
                    px(x, y, far ? 2 : (rnd() < 0.5 ? 2 : 4), far ? 2 : (rnd() < 0.5 ? 2 : 4));
                }
            } else if (tex === 'fog') {
                // Long banks that taper at both ends, their edges dithered.
                for (i = 0; i < 16; i++) {
                    x = Math.floor(rnd() * n / 2) * 2; y = Math.floor(rnd() * n / 2) * 2;
                    g.fillStyle = ['rgb(214,218,222)', 'rgb(196,202,208)', 'rgb(230,232,234)'][i % 3];
                    var len2 = 24 + Math.floor(rnd() * 24) * 2, thick = 2 + Math.floor(rnd() * 3) * 2;
                    for (var sx = 0; sx < len2; sx += 2) {
                        var edge = Math.min(sx, len2 - sx) / (len2 / 2);
                        var hgt = Math.max(2, Math.round(thick * Math.sqrt(edge) / 2) * 2);
                        if (edge < 0.35 && (sx / 2) % 2) { continue; }
                        px(x + sx, y - hgt / 2, 2, hgt);
                        if ((sx / 2) % 2 === 0) { px(x + sx, y - hgt / 2 - 2, 2, 2); }
                    }
                }
            } else if (tex === 'clouds') {
                g.fillStyle = 'rgb(28,34,46)';
                for (i = 0; i < 4; i++) {
                    x = Math.floor(rnd() * n / 2) * 2; y = Math.floor(rnd() * n / 2) * 2;
                    px(x, y, 30, 16); px(x + 6, y - 6, 18, 28);
                }
            }
            return c;
        }

        // weatherCells are the tiles the weather falls on: outdoor tiles drawn
        // (as the night shade picks them), the lakes, only those in sight
        // when LM2 limits sight, and not your own room when you only see the
        // weather through an exit.
        // In fog the fog is what limits sight, so it covers every outdoor tile
        // (allSky), remembered ones too (LM4 review).
        function weatherCells(drewTiles, drewLakes, inSight, indoorView, allSky, lakeSeen) {
            var cells = [];
            if (allSky) { inSight = null; }
            rooms.forEach(function (room, id) {
                if (!drewTiles[id]) { return; }
                var biome = biomeTable[room.env];
                if (biome && (biome.indoor || biome.dark)) { return; }
                if (inSight !== null && !inSight.has(id)) { return; }
                if (indoorView && id === currentRoomId) { return; }
                cells.push({ x: room.x, y: room.y, id: id });
            });
            var water = biomeTable.water;
            if (!(water && (water.indoor || water.dark))) {
                (drewLakes || []).forEach(function (c) {
                    if (inSight !== null && !(lakeSeen && lakeSeen(c))) { return; }
                    cells.push({ x: c.x, y: c.y, id: null });
                });
            }
            return cells;
        }

        // drawWeather (LM4) slides each texture of the plan over the weather
        // cells, anchored to the map so it pans with it, then scatters the
        // raindrops' splashes. It returns true while anything moves.
        function drawWeather(plan, cells, tilePx, nowMs) {
            if (!cells.length) { return false; }
            var t = nowMs / 1000;
            var origin = gridToCanvas(0, 0);
            ctx.save();
            ctx.beginPath();
            cells.forEach(function (c) {
                var p = gridToCanvas(c.x, c.y);
                ctx.rect(Math.round(p.px - tilePx / 2), Math.round(p.py - tilePx / 2), tilePx, tilePx);
            });
            ctx.clip();
            if (plan.dim > 0) {
                ctx.fillStyle = 'rgba(14,18,28,' + plan.dim + ')';
                ctx.fillRect(0, 0, viewW, viewH);
            }
            var art = {}, moving = false, fast = false;
            plan.layers.forEach(function (l) {
                var tx = weatherArt(l.tex);
                art[l.tex] = tx.art ? 'art' : 'code';
                var span = tilePx * l.size;
                var scale = span / tx.img.width;
                var ox = (origin.px - tilePx / 2 + l.vx * t * tilePx) % span;
                var oy = (origin.py - tilePx / 2 + l.vy * t * tilePx) % span;
                if (l.vx || l.vy) { moving = true; }
                if (Math.abs(l.vx) + Math.abs(l.vy) > 0.5) { fast = true; }
                var pat = patternCache.get(tx.img);
                if (!pat) { pat = ctx.createPattern(tx.img, 'repeat'); if (pat) { patternCache.set(tx.img, pat); } }
                if (!pat) { return; }
                if (pat.setTransform && window.DOMMatrix) {
                    pat.setTransform(new window.DOMMatrix().translateSelf(ox, oy).scaleSelf(scale, scale));
                }
                ctx.globalAlpha = l.alpha;
                ctx.imageSmoothingEnabled = false;
                ctx.fillStyle = pat;
                ctx.fillRect(0, 0, viewW, viewH);
            });
            ctx.globalAlpha = 1;
            var splashes = plan.splash > 0 ? drawSplashes(plan.splash, cells, tilePx, nowMs) : 0;
            ctx.restore();
            lastWeather = { name: plan.name, layers: plan.layers.map(function (l) { return l.tex; }), art: art,
                cells: cells.length, splashes: splashes, strikes: storm.strikes, flash: false, dim: plan.dim, moving: moving };
            // Rain, snow and splashes need every frame; drifting clouds and
            // fog the slow timer.
            return (fast || splashes > 0) ? 'fast' : (moving ? 'slow' : '');
        }

        // drawSplashes scatters raindrops: each tile gets a splash at a time
        // and place picked from its position and the second, so they don't
        // jump about between frames. At most 24 tiles splash.
        var SPLASH_MS = 400;
        function drawSplashes(rate, cells, tilePx, nowMs) {
            var a = Sprites.art('map/weather/splash.png');
            var mult = spriteMult(tilePx);
            var drawnN = 0;
            var period = 1000 / rate;
            // Only tiles in view splash, up to 24 spread across them.
            var inView = cells.filter(function (c) {
                var q = gridToCanvas(c.x, c.y);
                return q.px > -tilePx && q.py > -tilePx && q.px < viewW + tilePx && q.py < viewH + tilePx;
            });
            var step = Math.max(1, Math.ceil(inView.length / 24));
            inView.filter(function (c, i) { return i % step === 0; }).forEach(function (c) {
                var h = ((c.x * 73856093) ^ (c.y * 19349663)) >>> 0;
                var phase = (nowMs + (h % 1000)) % period;
                if (phase >= SPLASH_MS) { return; }
                var slot = Math.floor((nowMs + (h % 1000)) / period);
                var hh = (h ^ (slot * 83492791)) >>> 0;
                var p = gridToCanvas(c.x, c.y);
                var sx = p.px + ((hh % 100) / 100 - 0.5) * tilePx * 0.7;
                var sy = p.py + (((hh >>> 8) % 100) / 100 - 0.5) * tilePx * 0.7;
                var k = phase / SPLASH_MS;
                if (a) {
                    var fw = a.info.frame[0], fh = a.info.frame[1], d = a.info.density || 1;
                    var fi = Math.min(Math.floor(k * 4), 3), w = fw * mult / d, hgt = fh * mult / d;
                    ctx.globalAlpha = 0.7;
                    ctx.drawImage(a.img, fi * fw, 0, fw, fh, Math.round(sx - w / 2), Math.round(sy - hgt / 2), w, hgt);
                } else {
                    ctx.globalAlpha = 0.5 * (1 - k);
                    ctx.strokeStyle = 'rgb(190,208,224)';
                    ctx.lineWidth = Math.max(1, mult * 0.6);
                    ctx.beginPath(); ctx.ellipse(sx, sy, (1 + 3 * k) * mult, (0.5 + 1.5 * k) * mult, 0, 0, Math.PI * 2); ctx.stroke();
                }
                drawnN++;
            });
            ctx.globalAlpha = 1;
            return drawnN;
        }

        // drawLightning (LM4) runs a storm's strikes: every 4 to 9 seconds the
        // map flashes briefly and a bolt strikes a tile in view. Reduced
        // motion shows the bolt without the flash. It returns how soon to
        // redraw (ms): every frame during a strike, else at the next one; 0
        // when there is no storm.
        function drawLightning(plan, cells, tilePx, nowMs) {
            if (!plan || !plan.lightning || !cells.length) { storm.start = -1; storm.next = 0; return 0; }
            if (!storm.next) { storm.next = window.MapWeather.nextStrike(nowMs, Math.random()); }
            if (nowMs >= storm.next) {
                storm.start = nowMs;
                storm.tile = cells[Math.floor(Math.random() * cells.length)];
                storm.next = window.MapWeather.nextStrike(nowMs, Math.random());
                storm.strikes++;
            }
            // A strike whose tile has left the view (a pan, a new zone) ends.
            if (storm.start >= 0 && !cells.some(function (c) { return c.x === storm.tile.x && c.y === storm.tile.y; })) { storm.start = -1; }
            var st = storm.start >= 0 ? window.MapWeather.strike(storm.start, nowMs) : null;
            if (!st) { return Math.max(16, storm.next - nowMs); }
            var p = gridToCanvas(storm.tile.x, storm.tile.y);
            var mult = spriteMult(tilePx);
            var a = Sprites.art('map/weather/lightning.png');
            ctx.save();
            if (a) {
                var fw = a.info.frame[0], fh = a.info.frame[1], d = a.info.density || 1;
                var w = fw * mult / d, h = fh * mult / d;
                artSmoothing(a.info);
                ctx.drawImage(a.img, st.frame * fw, 0, fw, fh, Math.round(p.px - w / 2), Math.round(p.py - h / 2), w, h);
            } else {
                // A forked bolt down onto the tile, thinner each frame.
                ctx.strokeStyle = 'rgba(236,230,255,' + (1 - st.frame * 0.3) + ')';
                ctx.lineWidth = Math.max(1, mult * (1.6 - st.frame * 0.5));
                ctx.beginPath();
                ctx.moveTo(p.px + 4 * mult, p.py - 16 * mult);
                ctx.lineTo(p.px - 1 * mult, p.py - 7 * mult);
                ctx.lineTo(p.px + 3 * mult, p.py - 5 * mult);
                ctx.lineTo(p.px, p.py + 2 * mult);
                ctx.moveTo(p.px - 1 * mult, p.py - 7 * mult);
                ctx.lineTo(p.px - 5 * mult, p.py - 2 * mult);
                ctx.stroke();
            }
            if (st.flash > 0 && !reducedMotion()) {
                ctx.setTransform(1, 0, 0, 1, 0, 0);
                ctx.fillStyle = 'rgba(214,220,255,' + (0.28 * st.flash) + ')';
                ctx.fillRect(0, 0, canvas.width, canvas.height);
                if (lastWeather) { lastWeather.flash = true; }
            }
            ctx.restore();
            if (lastWeather) { lastWeather.strikes = storm.strikes; lastWeather.bolt = a ? 'art' : 'code'; }
            return 16;
        }

        // levelName (LM3) names a level that isn't the surface: z below 0 is
        // "level 2 below", above 0 "level 1 above".
        function levelName(z) {
            // A room the mapper couldn't place has absurd coordinates.
            if (!z || !isFinite(z) || Math.abs(z) > 50) { return ''; }
            return 'level ' + Math.abs(z) + (z < 0 ? ' below' : ' above');
        }

        // meanLight is a canvas's average brightness (0-765, red+green+blue),
        // read from a 16x16 copy (browser checks: the kept map isn't blank).
        function meanLight(c) {
            try {
                var t = document.createElement('canvas');
                t.width = 16; t.height = 16;
                var tc = t.getContext('2d');
                tc.drawImage(c, 0, 0, 16, 16);
                var d = tc.getImageData(0, 0, 16, 16).data, sum = 0;
                for (var i = 0; i < d.length; i += 4) { sum += (d[i] + d[i + 1] + d[i + 2]) * d[i + 3] / 255; }
                return sum / 256;
            } catch (e) { return -1; }
        }

        // startCrossing (LM3) begins the fade and title for a zone or level
        // change: the canvas still shows the old map, so it is kept to fade
        // out. Reduced motion shows the title without the fade.
        function startCrossing(name, z) {
            var old = null;
            if (!reducedMotion() && canvas && canvas.width > 0 && canvas.height > 0) {
                old = document.createElement('canvas');
                old.width = canvas.width; old.height = canvas.height;
                try { old.getContext('2d').drawImage(canvas, 0, 0); } catch (e) { old = null; }
            }
            lastCrossing = { name: name, level: levelName(z), faded: !!old, oldLight: old ? meanLight(old) : 0 };
            crossing = old ? { start: performance.now(), old: old } : null;
            lastCrossing.start = performance.now();
            if (crossTitle) {
                crossTitle.textContent = '';
                var n = document.createElement('div');
                n.className = 'mct-name'; n.textContent = name;
                crossTitle.appendChild(n);
                if (lastCrossing.level) {
                    var l = document.createElement('div');
                    l.className = 'mct-level'; l.textContent = lastCrossing.level;
                    crossTitle.appendChild(l);
                }
                crossTitle.classList.add('show');
                if (crossTitleTimer) { clearTimeout(crossTitleTimer); }
                crossTitleTimer = setTimeout(function () {
                    crossTitleTimer = null;
                    crossTitle.classList.remove('show');
                    crossTitle.textContent = '';   // no stale name for screen readers
                }, CROSS_TITLE_MS);
            }
        }

        // drawCrossing (LM3) draws the fade over the new map: the old map
        // darkening to black, then black lifting off the new one. It returns
        // true while the fade runs.
        function drawCrossing(nowMs) {
            if (!crossing) { return false; }
            var t = nowMs - crossing.start;
            if (t >= CROSS_OUT_MS + CROSS_IN_MS) { crossing = null; return false; }
            ctx.save();
            ctx.setTransform(1, 0, 0, 1, 0, 0);
            if (t < CROSS_OUT_MS) {
                ctx.drawImage(crossing.old, 0, 0, canvas.width, canvas.height);
                ctx.globalAlpha = t / CROSS_OUT_MS;
            } else {
                ctx.globalAlpha = 1 - (t - CROSS_OUT_MS) / CROSS_IN_MS;
            }
            ctx.fillStyle = '#000';
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            ctx.restore();
            return true;
        }

        // drawLightMarker (LM2) shows the light you carry beside your figure:
        // a torch or lantern at your hand (a dark lantern when it is out), or
        // your party light floating over your head. It uses the A13 marker art
        // when it has loaded (its hand point sits 2 px below its centre), and
        // draws a small light in code until then.
        function drawLightMarker(kind, now) {
            if (!kind || unit.x === null) { return; }
            var pose = unitPose(now);
            var mult = spriteMult(getRoomSize() * zoomScale);
            var p = gridToCanvas(pose.x, pose.y);
            var orb = kind === 'orb';
            var hx = orb ? p.px : p.px + (unit.flip ? -7 : 7) * mult;
            var hy = orb ? p.py - 22 * mult : p.py + 2 * mult;
            ctx.save();
            ctx.globalAlpha = figureDim * unitAlpha(now);
            var art = drawIcon('map/markers/' + (orb ? 'light-orb' : kind) + '.png', hx, orb ? hy : hy - 2 * mult, mult, now);
            if (!art) {
                var pulse = reducedMotion() ? 0 : Math.sin(now / 120);
                if (kind === 'torch') {
                    ctx.fillStyle = '#5a3a22';
                    ctx.fillRect(Math.round(hx - 0.6 * mult), Math.round(hy - 3 * mult), Math.max(1, Math.round(1.2 * mult)), Math.round(6 * mult));
                    ctx.fillStyle = '#f0a040';
                    ctx.beginPath(); ctx.arc(hx, hy - (4.5 + 0.3 * pulse) * mult, (2 + 0.25 * pulse) * mult, 0, Math.PI * 2); ctx.fill();
                    ctx.fillStyle = '#ffe08a';
                    ctx.beginPath(); ctx.arc(hx, hy - 4.2 * mult, 0.9 * mult, 0, Math.PI * 2); ctx.fill();
                } else if (kind === 'lantern' || kind === 'lantern-out') {
                    ctx.fillStyle = '#2e2a26';
                    ctx.fillRect(Math.round(hx - 1.8 * mult), Math.round(hy - 1 * mult), Math.round(3.6 * mult), Math.round(4.4 * mult));
                    ctx.fillStyle = kind === 'lantern' ? '#ffd27a' : '#4a4a52';
                    ctx.fillRect(Math.round(hx - 1.1 * mult), Math.round(hy - 0.3 * mult), Math.max(1, Math.round(2.2 * mult)), Math.max(1, Math.round(3 * mult)));
                } else {
                    ctx.globalAlpha *= 0.85 + 0.15 * pulse;
                    ctx.fillStyle = '#dfe8ff';
                    ctx.beginPath(); ctx.arc(hx, hy, 2.2 * mult, 0, Math.PI * 2); ctx.fill();
                }
            }
            ctx.restore();
            if (lastSight) { lastSight.markerArt = art ? 'art' : 'code'; }
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
            drawn = { tiles: 0, fallbacks: 0, walls: 0, fog: 0, landmarks: 0, glyphs: 0, animated: 0, icons: 0, dots: 0, path: 0, shaded: 0, pieces: {}, corners: {}, replaced: 0, lakes: 0, remembered: 0 };
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
            var onward = onwardCache;
            var lakeIndex = {};
            if (tiles) { lakeCells.forEach(function (c) { lakeIndex[c.x + ',' + c.y] = true; }); }

            // LM2: what you can see where you stand. inSight is null when the
            // whole map shows as today (daylight, no sight from the server, or
            // the day/night setting off); else the rooms in sight, and the
            // others are drawn as remembered: dark and still, with no figures,
            // landmarks or resource icons.
            var sight = (window.MapSight && mapSettings.dayNight !== false && timeInfo) ? timeInfo.sight : null;
            // After a move the new room's sight comes before the room itself:
            // keep drawing the sight you had until Room.Info catches up.
            if (sight && sight.room && currentRoomId !== null && sight.room !== currentRoomId) {
                sight = heldSight;
            } else {
                heldSight = sight;
            }
            var inSight = window.MapSight ? window.MapSight.visible(sight, currentRoomId, joinedTo(currentRoomId)) : null;
            var glow = window.MapSight ? window.MapSight.glow(sight) : null;
            function remembered(id) { return inSight !== null && !inSight.has(id); }
            function lakeSeen(c) {
                if (inSight === null) { return true; }
                // Open water is never your room: it is in sight only when
                // sight reaches past your room (level 2) to a shore beside it.
                if (!sight || sight.level < 2) { return false; }
                return [[1, 0], [-1, 0], [0, 1], [0, -1]].some(function (d) {
                    var n = coordIndex[(c.x + d[0]) + ',' + (c.y + d[1])];
                    return n !== undefined && inSight.has(n);
                });
            }

            if (!tiles) {
                drawConnections();
                drawStubs(false);
            }

            // Phase 40b: with a class sprite the current room keeps its terrain
            // colour and the sprite stands above it; without one it stays the
            // classic red square.
            var spriteOn = spritesOn() && currentRoomId !== null && unit.x !== null &&
                resolveSheet(chainKeys(identity.classid, identity.lineage), false) !== null;

            // Lakes (E2): open water a ring of shore encloses, drawn as water
            // tiles though no room is there. Their stand-in id, a hash of the
            // cell, picks a variant and phase.
            var drewLakes = [];
            if (tiles) {
                lakeCells.forEach(function (c) {
                    var lp = gridToCanvas(c.x, c.y);
                    var lt = drawTerrain({ x: c.x, y: c.y, env: 'water' }, lakeId(c.x, c.y), lp, tilePx, nowMs, coordIndex, onward, lakeIndex, !lakeSeen(c));
                    if (!lt) { return; }
                    drewLakes.push(c);
                    drawn.lakes++;
                    if (lt.animated) { animTiles = true; }
                });
            }

            // Pass 1: terrain. A room with no tile art yet (loading, failed or
            // missing) draws its classic colour square so the map never blanks.
            rooms.forEach(function (room, id) {
                var p         = gridToCanvas(room.x, room.y);
                var isCurrent = (id === currentRoomId) && !spriteOn;
                if (tiles) {
                    var t = drawTerrain(room, id, p, tilePx, nowMs, coordIndex, onward, lakeIndex, remembered(id));
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
                var mem = remembered(id);
                if (mem) { drawn.remembered++; }
                if (tiled && !mem) {
                    var mark = drawLandmark(room, id, p, tilePx, scaledFont, nowMs);
                    if (mark === 'landmark') { drawn.landmarks++; } else if (mark === 'glyph') { drawn.glyphs++; }
                }
                if (mapSettings.showResources && !mem) {
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

            // LM4: the weather falls on the outdoor tiles in sight, under the
            // night shade and darkness so they dim it too.
            var weatherInfo = (window.MapWeather && mapSettings.weather !== false && tiles && timeInfo) ? timeInfo.weather : null;
            var weatherPlan = weatherInfo ? window.MapWeather.plan(weatherInfo, { reduced: reducedMotion(), fogVeil: !!(glow && glow.fog) }) : null;
            var weatherAt = weatherPlan ? weatherCells(drewTiles, drewLakes, inSight, !!weatherInfo.indoor, !!(glow && glow.fog), lakeSeen) : [];
            lastWeather = null;
            var weatherMoving = weatherPlan ? drawWeather(weatherPlan, weatherAt, tilePx, nowMs) : false;
            if (weatherPlan && !lastWeather) { lastWeather = { name: weatherPlan.name, layers: [], art: {}, cells: 0, splashes: 0, strikes: storm.strikes, flash: false, dim: 0 }; }

            // Phase 40d: the time of day shades the outdoor tiles, and a walk
            // under way draws its path over them.
            if (glow) {
                nightQuant = Math.round(nightLevel() * 10);
                drawSight(inSight, glow, tilePx, drewLakes.map(function (c) { return { x: c.x, y: c.y, seen: lakeSeen(c) }; }), nowMs);
            } else if (tiles) {
                drawNightShade(drewTiles, tilePx, drewLakes);
            }
            figureDim = (sight && inSight !== null && sight.level === 0) ? 0.45 : 1;
            lastSight = sight ? {
                level: sight.level, daylight: !!sight.daylight, source: sight.source || '',
                carried: sight.carried || '', fog: !!sight.fog,
                visible: inSight === null ? 'all' : Array.from(inSight).sort(function (a, b) { return a - b; }),
                glow: glow, marker: window.MapSight.marker(sight), dim: figureDim
            } : null;
            var walkAnim = drawWalkPath(tilePx, nowMs);

            // Animated tiles (water, shore, swamp, snow, desert) cycle on a
            // timer while the map is drawn; reduced motion keeps them still.
            if ((animTiles || walkAnim) && !reducedMotion() && !document.hidden) { scheduleAnim(false); }

            // LM1: decided before the camps are drawn, so your camp sits on
            // your tile rather than beside a figure that isn't drawn.
            var asYou = spriteOn && unit.x !== null && campAsYou(unitPose(nowMs));
            lastCampAsYou = asYou;
            drawCamps(nowMs, spriteOn, asYou, unitAlpha(nowMs) * figureDim, remembered);

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
                // LM2: an ally out of sight isn't drawn.
                var allyRoom = coordIndex[ease.toGx + ',' + ease.toGy];
                if (inSight !== null && (allyRoom === undefined || remembered(allyRoom))) { return; }
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
            var drewUnit = drawUnit(nowMs, asYou);
            if (drewUnit && lastSight) { drawLightMarker(lastSight.marker, nowMs); }
            // LM2 review: with no figure (sprites off or Classic) your room's
            // red square sits under the darkness; outline it again over the
            // darkness so you still know where you stand.
            if (!drewUnit && inSight !== null && rooms.has(currentRoomId)) {
                var hr = rooms.get(currentRoomId), hp = gridToCanvas(hr.x, hr.y);
                ctx.save();
                ctx.globalAlpha = sight.level === 0 ? 0.55 : 0.9;
                ctx.strokeStyle = CURRENT_ROOM_COLOR;
                ctx.lineWidth = Math.max(2, Math.round(scaledSize * 0.08));
                ctx.strokeRect(Math.round(hp.px - tilePx / 2) + 1, Math.round(hp.py - tilePx / 2) + 1, tilePx - 2, tilePx - 2);
                ctx.restore();
                drawn.hereOutline = true;
            }
            if (anyEasing && heartRafId === null) {
                heartRafId = requestAnimationFrame(function () {
                    heartRafId = null;
                    render();
                });
            }
            // LM4: lightning over everything but the crossing fade.
            var stormIn = drawLightning(weatherPlan, weatherAt, tilePx, nowMs);
            if (!document.hidden) {
                if (weatherMoving === 'fast' || (stormIn && stormIn <= 16)) { scheduleAnim(true); }
                else if (weatherMoving === 'slow') { scheduleAnim(false); }
                if (stormIn > 16) { scheduleAnim(false, stormIn); }
            }
            if (drawCrossing(nowMs)) { scheduleAnim(true); }
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
            crossTitle = document.createElement('div');
            crossTitle.className = 'map-crossing-title';
            crossTitle.setAttribute('role', 'status');
            crossTitle.setAttribute('aria-live', 'polite');
            wrap.appendChild(crossTitle);

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
                if (info) { showTooltip(e.clientX, e.clientY, info, id !== currentRoomId); }
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
                    items.push({ label: 'Walk to ' + ((info && info.name) || ('room ' + id)), cmd: 'walkto ' + id, echo: info && info.name ? 'walkto ' + info.name : '' });
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
                if (currentZoneKey) { startCrossing(info.area || info.coords.split(',')[0].trim(), gz); }
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
                refreshTiling(zoneKey);
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

            panel.appendChild(row('Weather', btnGroup(
                [{ label: 'On', value: true }, { label: 'Off', value: false }],
                function () { return mapSettings.weather !== false; },
                function (v) { mapSettings.weather = v; }
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
            setIdentity: function (classid, lineage, skin, hair) {
                var look = (window.SpriteTint && window.SpriteTint.look(skin, hair)) || null;
                var lookKey = look ? look.skin + '|' + look.hair : '';
                var oldKey = identity.look ? identity.look.skin + '|' + identity.look.hair : '';
                if (identity.classid === classid && identity.lineage === lineage && lookKey === oldKey) { return; }
                identity.classid = classid; identity.lineage = lineage; identity.look = look;
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
                var sight = timeInfo && timeInfo.sight;
                var k = window.MapSight ? window.MapSight.key(sight) : '';
                // A sight for a room you aren't drawn in yet waits for
                // Room.Info, whose redraw uses it (render holds the old one).
                if (sight && sight.room && currentRoomId !== null && sight.room !== currentRoomId) { return; }
                var wk = window.MapWeather ? window.MapWeather.key(timeInfo && timeInfo.weather) : '';
                if (q !== nightQuant || k !== sightKey || wk !== weatherKey) { sightKey = k; weatherKey = wk; render(); }
            },
            redraw: function () { render(); },
            // LM4 (browser checks): bring the next lightning strike forward.
            strikeNow: function () { storm.next = performance.now(); render(); },
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
                var walkRes = resolveSheet(chainKeys(identity.classid, identity.lineage), true);
                var lookRes = identity.look ? resolveSheet(chainKeys(identity.classid, identity.lineage), false, identity.look) : null;
                var pose = unit.x === null ? null : unitPose(performance.now());
                return {
                    spriteDrawn: spritesOn() && !!res, face: unit.face, flip: unit.flip, walking: !!(pose && pose.walking),
                    queued: unit.queue.length, companySize: companySize,
                    companions: companions.map(function (c) {
                        return { key: c.key, sprite: spritesOn() && resolveSheet(chainKeys(c.classid, c.lineage), false) !== null };
                    }),
                    style: tilesOn() ? 'tiles' : 'classic', drawn: drawn, zoom: zoomScale,
                    camp: campInfo, campAsYou: lastCampAsYou, figures: lastFigures, fire: lastFire, fading: unit.fadeStart >= 0,
                    walk: walkInfo, night: nightLevel(),
                    unit: pose ? { x: pose.x, y: pose.y } : null,
                    keys: chainKeys(identity.classid, identity.lineage),
                    look: identity.look,
                    sight: lastSight,
                    weather: lastWeather, anim: lastAnim,
                    crossing: lastCrossing ? {
                        name: lastCrossing.name, level: lastCrossing.level, faded: lastCrossing.faded, oldLight: lastCrossing.oldLight,
                        fading: !!crossing && performance.now() - crossing.start < CROSS_OUT_MS + CROSS_IN_MS,
                        title: !!(crossTitle && crossTitle.classList.contains('show')),
                    } : null,
                    // E1: the drawn sheet's density and frame counts
                    art: res ? { density: res.sheet.info.density || 1, idleFrames: res.sheet.info.frames,
                                 walkFrames: walkRes && walkRes.walk ? walkRes.sheet.info.frames : 0,
                                 // tinted: the figure is drawn repainted in the player's look (a
                                 // high-density sheet once its look mask has loaded, E1b).
                                 tinted: !!(lookRes && lookRes.sheet.img !== res.sheet.img),
                                 lookMask: res.sheet.info.look_mask || '' } : null,
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
                        look:      pos.look || null,
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
    window.MapView = { state: function () { return view2d.state(); }, pointOf: function (id) { return view2d.pointOf(id); },
                       strikeNow: function () { view2d.strikeNow(); } };

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
                                           lineage: v.lineage || '', classid: v.classid || '',
                                           look: (window.SpriteTint && window.SpriteTint.look(v.skin, v.hair)) || null };
        });
        view2d.setPartyPositions(partyMemberPositions);
    }

    // Phase 40b: who the player is (Char.Info), how many travel with them and
    // their camps (Company), all read from the stored GMCP state.
    function updateIdentity() {
        var c = Client.GMCPStructs.Char;
        var info = c && c.Info;
        view2d.setIdentity((info && info.classid) || '', (info && info.lineage) || '', info && info.skin, info && info.hair);
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
        gmcpHandlers: ['Room', 'World', 'Party', 'Char', 'Company', 'Walkto', 'Gametime'],
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
