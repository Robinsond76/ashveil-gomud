/**
 * window-battle.js
 *
 * The battle screen (Phase 40f): an Ogre Battle style picture of the
 * player's battle. The company stands on the left and the enemy on the
 * right, each in its 3x3 formation, side-on, over a background chosen by
 * the biome. A battle plays out on its own, so the screen takes no orders
 * beyond the dock's own Retreat and focus buttons (help strategy).
 *
 * It opens by itself when a battle starts (a setting turns that off; the
 * corner badge and the Combat tab's "Battle screen" button open it), can be
 * minimised to the badge, and shows the outcome for three seconds when the
 * fight ends. The picture is decorative: the Combat tab and the narration
 * stay the accessible path, so the canvas is aria-hidden. Every server
 * string is set with textContent, never innerHTML.
 *
 * Figures are drawn in code as small pixel shapes (a figure of the class
 * or race, in the 64-colour palette's hues), keyed the way the sprite
 * manifest keys them, so art sheets can replace them later without a
 * change to the layout or the data (Phase 40s3 draws the real battle set).
 *
 * Feeds:
 *   Company.Battle        cells, labels, banded health, targets, focus
 *   Company, .Vitals      members, cells, roles, exact health
 *   Room                  the biome, for the background
 *   Company.Battle.Event  (Client.onBattleEvents) hits, heals, casts,
 *                         statuses, falls, yields, the fight's end
 *
 * Phase 40g animates it. Each Company.Battle.Event batch is planned by
 * battle-timeline.js (pure, tested under Node) into steps that this file
 * plays: a fighter steps in and strikes, a shot flies, a caster chants and
 * releases, the struck flinch, block, parry or dodge, statuses appear as
 * their events arrive, the fallen collapse, the beaten yield and the routed
 * run, with hit effects and damage digits. Animation is client-side only:
 * it never sends anything and never changes or delays the battle. The
 * setting `battleAnimations` is full, reduced (no projectile travel, flashes
 * or shake) or off (the 40f static screen); the default follows the
 * system's reduced-motion request. A unit without S4 art for a pose
 * (battle/units/<key>/<pose>.png) nudges its idle figure instead, and
 * effects without art are drawn in code.
 *
 * Units known by a "?" ref (an enemy the player can't make out) are drawn
 * as one unseen presence. Enemy health is shown in five bands, never as
 * numbers. A spell's results arrive on the line after its cast line, so a
 * cast shows as a chant mark until the cast ends and the results flash as
 * they come.
 *
 * Window.BattleScreen exposes open(), close(), state() for tests.
 */

'use strict';

(function() {

    const el = CompanyData.el;
    const TL = window.BattleTimeline;

    const W = 320;
    const H = 180;
    const SETTING_KEY = 'ashveil-battle-screen';
    const ANIM_KEY = 'ashveil-battle-animations';
    const OUTCOME_MS = 3000;
    const FOCI = ['none', 'leader', 'casters', 'healers', 'nearest', 'weakest', 'strongest', 'wounded'];

    // Health words from the server, as the fraction of the bar they fill.
    const BANDS = { 'unhurt': 1, 'scratched': 0.8, 'wounded': 0.6, 'badly wounded': 0.4, 'near death': 0.2 };

    // Role letters drawn on a member's badge.
    const ROLE_GLYPH = { fighter: 'F', healer: 'H', caster: 'C', guardian: 'G', controller: 'K' };

    // Class hues for company figures: body, trim.
    const CLASS_HUES = {
        warrior: ['#8a8f99', '#c0504d'], cleric: ['#e8e4d0', '#d4a72c'], ranger: ['#4e7d3a', '#8a6a3b'],
        rogue: ['#444a56', '#9b59b6'], wizard: ['#4a5fc1', '#e0c040'], witch: ['#6b3f8c', '#3fb08a'], halberdier: ['#7d8590', '#b87333'],
        samurai: ['#2f3a4a', '#d9a441'], shaman: ['#3b6a7a', '#9fd0e0'], 'gryphon-rider': ['#8a6d3b', '#e8e2c9'], alchemist: ['#8a7a4e', '#c4561c'], arbalist: ['#5a6a7a', '#c0a060'],
        dollmaster: ['#6a3d4a', '#d8b878'], doll: ['#b08850', '#6a4a2a'], // Phase 39d: the doll is painted wood
        beasttamer: ['#6b5a3a', '#c9a24a'], wolf: ['#7d7d85', '#c9c9d0'], warhound: ['#5a4632', '#b08850'], // Phase 39e: the bonded beasts
        bear: ['#5b4030', '#8a6a4a'], drake: ['#3f7a4a', '#d9622b'],
    };
    const DEFAULT_HUES = ['#7a6a55', '#c0a060'];
    // Phase 39e: a bonded beast is drawn as the battle unit nearest its kind.
    const BEAST_SPRITES = { wolf: 'wolf-timber', warhound: 'dog-junkyard', bear: 'war-bear', drake: 'drake-hatchling' };

    // Fallback silhouettes: width, height in virtual pixels.
    const SIZES = { 'unknown-humanoid': [10, 24], 'unknown-beast': [20, 14], 'unknown-large': [18, 34] };

    // Biome (from Room.Info.environment) to a backdrop.
    const SCENES = {
        snow:      { bg: 'snowfield', sky: ['#8fa8c8', '#d8e4f0'], ground: '#e8eef4', far: '#aebfd4', style: 'hills' },
        mountains: { bg: 'highlands', sky: ['#6f86a8', '#b9c6d8'], ground: '#7a7468', far: '#5a5f6e', style: 'peaks' },
        forest:    { bg: 'forest', sky: ['#6fa0a0', '#a8c8b0'], ground: '#4d6b3a', far: '#2f4a2c', style: 'trees' },
        desert:    { bg: 'desert', sky: ['#d8a860', '#f0d8a0'], ground: '#d8b878', far: '#b88a50', style: 'hills' },
        city:      { bg: 'city', sky: ['#7a8aa8', '#b8c0d0'], ground: '#6a6a70', far: '#4a4a56', style: 'wall' },
        fort:      { bg: 'city', sky: ['#6a7a98', '#a8b0c0'], ground: '#5e5e66', far: '#3e3e4a', style: 'wall' },
        house:     { bg: 'interior', sky: ['#3a2f28', '#5a4636'], ground: '#6e5238', far: '#4a382a', style: 'room' },
        spiderweb: { bg: 'deep-web', sky: ['#1c1c24', '#2c2c38'], ground: '#34303a', far: '#222028', style: 'cave' },
        cave:      { bg: 'cave', sky: ['#1c1c24', '#2c2c38'], ground: '#3a3640', far: '#22202a', style: 'cave' },
        road:      { bg: 'road', sky: ['#7a9ac0', '#c0d4e0'], ground: '#8a7a58', far: '#58704a', style: 'hills' },
        land:      { bg: 'plains', sky: ['#7aa4c8', '#bcd8e4'], ground: '#5c8040', far: '#3e6038', style: 'hills' },
    };

    injectStyles(`
        #battle-screen {
            position: fixed;
            top: 8px;
            left: 50%;
            transform: translateX(-50%);
            z-index: 9000;
            max-width: calc(100vw - 8px);
            box-sizing: border-box;
            padding: 4px 6px 6px;
            background: var(--t-bg-panel);
            border: 1px solid var(--t-border-accent);
            border-radius: 6px;
            box-shadow: 0 6px 24px rgba(0, 0, 0, 0.8);
            color: var(--t-text);
            font-size: 0.8em;
            display: none;
        }
        #battle-screen.open { display: block; }
        #battle-screen .bs-head { display: flex; align-items: baseline; gap: 8px; justify-content: space-between; min-height: 1.6em; }
        #battle-screen .bs-title { font-weight: bold; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        #battle-screen .bs-banners { color: var(--t-text-secondary); }
        #battle-screen canvas { display: block; margin: 4px auto; image-rendering: pixelated; image-rendering: crisp-edges; background: #000; }
        #battle-screen .bs-legend { min-height: 1.3em; display: flex; flex-wrap: wrap; gap: 2px 10px; justify-content: center; color: var(--t-text-secondary); font-size: 0.9em; }
        #battle-screen .bs-legend .bs-dot { display: inline-block; width: 8px; height: 8px; margin-right: 4px; border-radius: 1px; vertical-align: baseline; }
        #battle-screen .bs-last { min-height: 1.3em; text-align: center; font-style: italic; }
        #battle-screen .bs-caption { min-height: 1.3em; color: var(--t-text-secondary); text-align: center; }
        #battle-screen .bs-outcome { text-align: center; font-weight: bold; min-height: 1.3em; }
        #battle-screen .bs-foot { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; justify-content: center; }
        #battle-screen button, #battle-badge {
            font: inherit; color: var(--t-text); background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim); border-radius: 3px; padding: 1px 6px; cursor: pointer;
        }
        #battle-screen button[aria-pressed="true"] { border-color: var(--t-border-accent); font-weight: bold; }
        #battle-screen button:disabled { opacity: 0.5; cursor: default; }
        #battle-screen label { color: var(--t-text-secondary); }
        #battle-badge { position: fixed; right: 10px; bottom: 10px; z-index: 9000; display: none; padding: 4px 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.7); }
        #battle-badge.show { display: block; }
    `);

    // ---------------------------------------------------------------------
    // Art: the sprite manifest (Phase 40s1) and the S3 battle set
    // ---------------------------------------------------------------------

    // Sprites live beside the scripts: <base>/static/sprites/manifest.json.
    // Only files the manifest lists are ever requested, so a set that has
    // not been drawn yet costs no failed fetches: the unit or backdrop is
    // drawn in code instead, and the art replaces it when it appears,
    // with no change here. Layout (sprite specification, S3):
    //   battle/units/<sprite key>/idle.png        4 frames, facing right
    //   battle/backgrounds/<background id>.png    320x180, opaque
    const here = document.currentScript && document.currentScript.src ? document.currentScript.src : '';
    const spriteBase = here.replace(/js\/windows\/[^/]*$/, 'sprites/');
    let manifest = null;
    const images = new Map();   // path -> HTMLImageElement | null (failed)

    function loadManifest() {
        if (!spriteBase || !window.fetch || /^file:/.test(spriteBase)) { return; }
        fetch(spriteBase + 'manifest.json').then(r => (r.ok ? r.json() : null)).then(m => {
            if (m && m.files) { manifest = m; draw(); }
        }).catch(() => { /* no art yet: the code-drawn figures stand */ });
    }

    // art returns a loaded image and its manifest entry, or null.
    function art(path) {
        if (!manifest || !manifest.files[path]) { return null; }
        if (!images.has(path)) {
            const img = new Image();
            images.set(path, null);
            img.onload = () => { images.set(path, img); draw(); };
            img.onerror = () => { /* stays null */ };
            img.src = spriteBase + path;
        }
        const img = images.get(path);
        return img ? { img, info: manifest.files[path] } : null;
    }

    // tintedArt (Phase 72a) repaints a player's figure in their chosen skin
    // tone and hair colour (sprite-tint.js); the repainted canvas is made
    // once per sheet and look. Companions and creatures have no look.
    const tintCache = {};
    function tintedArt(a, path, look) {
        if (!a || !look || !window.SpriteTint) { return a; }
        const img = window.SpriteTint.canvasFor(a.img, look, (w, h) => {
            const c = document.createElement('canvas');
            c.width = w; c.height = h;
            return c;
        }, tintCache, path);
        return { img, info: a.info };
    }

    // ---------------------------------------------------------------------
    // State
    // ---------------------------------------------------------------------

    let battle = null;          // the last Company.Battle, null when none
    let biome = '';
    let units = new Map();      // id -> unit
    let floaters = [];          // transient floating numbers
    let overlay = null;
    let canvas = null;
    let ctx = null;
    let badge = null;
    let minimised = false;
    let userOpened = false;     // opened by hand in manual mode
    let outcomeText = '';
    let outcomeTimer = null;
    let ended = false;          // fight-end came: this battle's last snapshots change nothing
    let hover = null;           // the unit id under the pointer
    let watching = null;        // Phase 45: the allied company index shown full size, or null
    let raf = 0;
    let zone = '';              // Room.Info.area, for a zone's own backdrop
    let roundNo = 0;            // the round of the latest events
    let lastBlow = '';          // the latest happening, in words
    let sched = new TL.Scheduler();
    let paceHistory = [];       // how event batches arrived, to infer the pace of a server that sends none
    let feedPace = '';          // the player's combat pace, as the event feed sends it (Phase 40g2)
    let holding = new Set();    // units whose fall or exit is still to play
    let outcomeQueued = false;  // a fight-end is scheduled: its outcome shows when it plays
    let shakeUntil = 0;
    let endExtra = 0;

    function setting() {
        try { return localStorage.getItem(SETTING_KEY) === 'manual' ? 'manual' : 'auto'; } catch (err) { return 'auto'; }
    }

    function saveSetting(v) {
        try { localStorage.setItem(SETTING_KEY, v); } catch (err) { /* unavailable: the choice lasts this page */ }
        memorySetting = v;
    }
    let memorySetting = null;
    function currentSetting() { return memorySetting || setting(); }

    // battleAnimations: full, reduced or off. The default is reduced when
    // the system asks for reduced motion.
    const MOTIONS = ['full', 'reduced', 'off'];
    let memoryMotion = null;
    function systemReducedMotion() {
        try { return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches); } catch (err) { return false; }
    }
    function motion() {
        if (memoryMotion) { return memoryMotion; }
        try {
            const v = localStorage.getItem(ANIM_KEY);
            if (MOTIONS.indexOf(v) >= 0) { return v; }
        } catch (err) { /* unavailable: the default */ }
        return systemReducedMotion() ? 'reduced' : 'full';
    }
    function saveMotion(v) {
        if (MOTIONS.indexOf(v) < 0) { return; }
        memoryMotion = v;
        try { localStorage.setItem(ANIM_KEY, v); } catch (err) { /* the choice lasts this page */ }
        if (v === 'off') { collapseNow(); }
    }

    // ---------------------------------------------------------------------
    // Layout: the 3x3 formations on the 320x180 canvas
    // ---------------------------------------------------------------------

    // slot is where a cell's feet stand. Row 0 (the front) is nearest the
    // centre line; each row steps 32 px outward. Column 0 is the upper,
    // further lane; column 2 the lower, nearer one, with a small x skew.
    function slot(side, row, col) {
        const dx = 40 + row * 32 + col * 6;
        return { x: side === 'company' ? W / 2 - dx : W / 2 + dx, y: 120 + col * 20 };
    }

    // Allied companies (Phase 40g2) stand behind and above the player's, in
    // their own half-scale 3x3, at most two drawn; they face the enemy as the
    // company does. Each anchor is the right edge of a formation's front row.
    const ALLY_SCALE = 0.5;
    const ALLY_ANCHORS = [150, 94];
    const MAX_ALLIES = ALLY_ANCHORS.length;

    function allySlot(index, row, col) {
        return { x: ALLY_ANCHORS[index] - row * 16 - col * 3, y: 58 + col * 9 };
    }

    // Watching an allied company (Phase 45) swaps places: the watched ally
    // stands in the company's own formation at full size, and the player's
    // company shrinks into the ally's place. View only: no input changes.
    function isWatched(u) { return watching !== null && u.side === 'ally' && u.ally === watching; }
    function isShrunk(u) {
        if (u.side === 'ally') { return !isWatched(u); }
        return u.side === 'company' && watching !== null;
    }

    // slotOf is where a unit's feet stand.
    function slotOf(u) {
        if (isWatched(u)) { return slot('company', u.cell.row, u.cell.col); }
        if (u.side === 'company' && watching !== null) { return allySlot(watching, u.cell.row, u.cell.col); }
        if (u.side === 'ally') { return allySlot(u.ally || 0, u.cell.row, u.cell.col); }
        return slot(u.side, u.cell.row, u.cell.col);
    }

    // ---------------------------------------------------------------------
    // Pixel text: 3x5 glyphs drawn as whole pixels, so letters on the
    // canvas stay crisp at every scale (canvas text is anti-aliased)
    // ---------------------------------------------------------------------

    const GLYPHS = {
        A: '010101111101101', B: '110101110101110', C: '011100100100011', D: '110101101101110', E: '111100110100111',
        F: '111100110100100', G: '011100101101011', H: '101101111101101', I: '111010010010111', J: '001001001101010',
        K: '101101110101101', L: '100100100100111', M: '101111111101101', N: '110101101101101', O: '010101101101010',
        P: '110101110100100', Q: '010101101110011', R: '110101110101101', S: '011100010001110', T: '111010010010010',
        U: '101101101101111', V: '101101101101010', W: '101101111111101', X: '101101010101101', Y: '101101010010010',
        Z: '111001010100111', '0': '111101101101111', '1': '010110010010111', '2': '110001010100111', '3': '110001010001110',
        '4': '101101111001001', '5': '111100110001110', '6': '011100111101111', '7': '111001010010010', '8': '111101111101111',
        '9': '111101111001110', '+': '000010111010000', '?': '110001010000010', '-': '000000111000000', '.': '000000000000010',
    };

    // pixText draws text in the 3x5 pixel font with its top-left at (x, y);
    // lower case is drawn as capitals, anything unknown as a gap. It returns
    // the width it took.
    function pixText(text, x, y, color) {
        ctx.fillStyle = color;
        let cx = Math.round(x);
        const top = Math.round(y);
        String(text).toUpperCase().split('').forEach(ch => {
            const g = GLYPHS[ch];
            if (g) {
                for (let i = 0; i < 15; i++) {
                    if (g[i] === '1') { ctx.fillRect(cx + (i % 3), top + Math.floor(i / 3), 1, 1); }
                }
            }
            cx += 4;
        });
        return cx - Math.round(x);
    }

    // ---------------------------------------------------------------------
    // Units
    // ---------------------------------------------------------------------

    function unit(id) {
        let u = units.get(id);
        if (!u) {
            u = { id, side: 'enemy', label: '', sprite: 'unknown-humanoid', cell: null, frac: 1, band: '', role: '',
                  klass: '', leader: false, fallen: false, yielded: false, statuses: new Set(), casting: '', flash: 0, flashColor: '#fff' };
            units.set(id, u);
        }
        return u;
    }

    function cellOf(c) {
        return c && Number.isFinite(c.row) && Number.isFinite(c.col) ? { row: c.row, col: c.col } : null;
    }

    // unseenCell is where the unseen presence stands: the first cell of the
    // enemy formation, centre first, that no visible foe holds, so its
    // silhouette never overlaps one the player can see.
    const UNSEEN_CELLS = [[0, 1], [1, 1], [0, 0], [0, 2], [1, 0], [1, 2], [2, 1], [2, 0], [2, 2]];
    function unseenCell() {
        const held = new Set();
        units.forEach(o => { if (o.side === 'enemy' && !o.unseen && o.cell && !o.fallen) { held.add(o.cell.row + ',' + o.cell.col); } });
        const free = UNSEEN_CELLS.find(c => !held.has(c[0] + ',' + c[1])) || UNSEEN_CELLS[0];
        return { row: free[0], col: free[1] };
    }

    function unseen() {
        const u = unit('?');
        u.side = 'enemy';
        u.label = 'something unseen';
        u.sprite = 'unknown-shape';
        u.cell = unseenCell();
        u.unseen = true;
        u.fallen = false;
        return u;
    }

    // rebuild reads the snapshot feeds into units. Called on every
    // Company, Company.Vitals and Company.Battle message, so a fight that
    // grows mid-battle (no event announces newcomers) shows its newcomers.
    function rebuild() {
        const data = CompanyData.read();
        const seen = new Set();

        data.members.forEach(m => {
            if (!m || !m.key) { return; }
            const placed = battle && battle.positions ? battle.positions[m.key] : null;
            const cell = cellOf(placed) || cellOf(m.cell);
            // As the Combat tab: only members still in the fight stand.
            if (!cell || m.status === 'awaiting' || m.status === 'fled' || m.status === 'separated') { return; }
            const u = unit(m.key);
            const v = data.vitals(m.key);
            u.side = 'company';
            u.label = m.name || m.key;
            u.klass = String(m.archetype || '').toLowerCase();
            u.sprite = u.klass || 'adventurer';
            u.promoted = String(m.class || '').toLowerCase();   // Phase 40s5: an advanced or elite class has its own art
            u.skin = m.skin || '';                               // Phase 72a: the leader's chosen colours
            u.hair = m.hair || '';
            u.className = m.class_name || '';
            // Phase 39g: an Alchemist's flasks left, named when its figure is tapped.
            u.flasks = (v.flasks_max > 0) ? v.flasks + ' of ' + v.flasks_max + ' flasks' : '';
            u.cell = cell;
            u.leader = m.key === 'leader';
            u.role = (m.strategy && m.strategy.role) || '';
            u.fallen = m.status === 'dead' && !holding.has(m.key);
            u.frac = (v.hp !== null && v.hp !== undefined && v.hp_max > 0) ? Math.max(0, Math.min(1, v.hp / v.hp_max)) : (u.fallen ? 0 : 1);
            u.band = '';
            seen.add(m.key);
        });

        if (battle) {
            // Phase 39d: a Doll Master's dolls, in the cells they stand in.
            (battle.dolls || []).forEach(d => {
                if (!d || !d.key) { return; }
                const cell = cellOf(battle.positions ? battle.positions[d.key] : null);
                if (!cell) { return; }
                const u = unit(d.key);
                u.side = 'company';
                u.label = d.name || d.key;
                u.klass = d.kind || 'doll'; // Phase 39e: a beast's kind, else the doll
                u.sprite = BEAST_SPRITES[d.kind] || d.kind || 'doll';
                u.promoted = '';
                u.className = d.kind ? ({ bear: 'War bear', drake: 'Drake hatchling' }[d.kind] || (d.kind.charAt(0).toUpperCase() + d.kind.slice(1))) : 'Doll';
                u.cell = cell;
                u.leader = false;
                u.role = '';
                u.fallen = false;
                u.frac = d.hp_max > 0 ? Math.max(0, Math.min(1, d.hp / d.hp_max)) : 1;
                u.band = '';
                seen.add(d.key);
            });
            (battle.enemies || []).forEach(e => {
                const u = unit(e.id);
                u.side = 'enemy';
                u.label = e.label || e.id;
                u.sprite = e.sprite || 'unknown-humanoid';
                u.cell = cellOf(e.cell);
                u.band = e.health || '';
                u.frac = BANDS[e.health] !== undefined ? BANDS[e.health] : 1;
                u.fallen = false;
                u.yielded = false;
                seen.add(e.id);
            });
            (battle.fallen || []).forEach(f => { if (units.has(f.id)) { unit(f.id).fallen = !holding.has(f.id); seen.add(f.id); } });
            (battle.surrendered || []).forEach(f => { if (units.has(f.id)) { const u = unit(f.id); u.yielded = true; seen.add(f.id); } });
            if (battle.dark) { unseen(); seen.add('?'); }
            // Allied companies: the first two are drawn, half scale.
            (battle.allies || []).slice(0, MAX_ALLIES).forEach((al, index) => {
                (al.members || []).forEach(m => {
                    if (!m || !m.id) { return; }
                    const cell = cellOf(m.cell);
                    if (!cell) { return; }
                    const u = unit(m.id);
                    u.side = 'ally';
                    u.ally = index;
                    u.allyName = al.name || '';
                    u.label = m.name || m.id;
                    u.klass = String(m.class || '').toLowerCase();
                    u.sprite = u.klass || 'adventurer';
                    u.promoted = String(m.promoted || '').toLowerCase();
                    u.skin = m.skin || '';
                    u.hair = m.hair || '';
                    u.cell = cell;
                    u.band = m.health || '';
                    u.frac = BANDS[m.health] !== undefined ? BANDS[m.health] : 1;
                    u.fallen = !!m.down && !holding.has(m.id);
                    seen.add(m.id);
                });
            });
        }
        // Anyone no longer in the fight leaves the picture; the unseen
        // presence stays while an event still names one.
        units.forEach((u, id) => {
            if (!seen.has(id) && !(id === '?' && u.keep) && !holding.has(id)) { units.delete(id); }
        });
        // The unseen presence steps aside for a foe that came into view.
        if (units.has('?')) { units.get('?').cell = unseenCell(); }
    }

    // ---------------------------------------------------------------------
    // Events (Company.Battle.Event)
    // ---------------------------------------------------------------------

    function flash(u, color, text) {
        if (!u) { return; }
        u.flash = Date.now() + 260;
        u.flashColor = color;
        if (text) {
            const p = u.cell ? slotOf(u) : { x: W / 2, y: 120 };
            floaters.push({ x: p.x, y: p.y - 34, text: String(text), color, born: Date.now() });
        }
        wake();
    }

    function refUnit(ref) {
        if (!ref) { return null; }
        if (ref === '?') { const u = unseen(); u.keep = true; return u; }
        if (ref === 'me') { return units.get('leader') || null; }
        return units.get(ref) || null;
    }

    // narrate is a happening in a few plain words, for the last-blow line.
    // Damage in parentheses, as the narration has it; no exclamations.
    function nameOf(ref) {
        const u = refUnit(ref);
        return u ? (u.id === 'leader' ? (u.label || 'You') : u.label) : '';
    }

    function narrate(e) {
        const a = nameOf(e.src), t = nameOf(e.tgt);
        const dmg = e.damage ? ' (' + e.damage + ')' : '';
        switch (e.kind) {
        case 'attack':
            if (!a || !t) { return ''; }
            if (e.outcome === 'miss') { return a + ' misses ' + t; }
            if ((e.defenses || []).length && !e.damage) { return t + ' ' + e.defenses[0] + ' ' + a; }
            return a + (e.crit ? ' strikes ' + t + ' hard' : ' hits ' + t) + dmg;
        case 'spell-hit': {
            // An unseen caster's spell goes unnamed, as its chant does.
            const sp = e.src === '?' ? 'a spell' : (e.spell_name || e.spell || 'a spell');
            if (!t) { return ''; }
            const who = a && e.src !== '?' ? a + '\'s ' + sp : sp.charAt(0).toUpperCase() + sp.slice(1);
            return who + ' strikes ' + t + dmg;
        }
        case 'heal': return t ? (a ? a + ' mends ' : 'Mended: ') + t + (e.amount ? ' (+' + e.amount + ')' : '') : '';
        case 'cast-start': return a ? a + ' begins a chant' : '';
        case 'cast-complete': return a && e.outcome === 'interrupted' ? a + '\'s chant is broken' : '';
        case 'status-tick':
            if (t && e.outcome === 'lost-action') {
                return t + (e.status === 'hesitation' ? ' hesitates as the company falters' : ' loses the action');
            }
            return t && e.damage ? t + ' suffers ' + (e.status || 'a wound') + ' (' + e.damage + ')' : '';
        case 'death': return t ? t + ' falls' : '';
        case 'yield': return a ? a + ' yields' : '';
        case 'flee': return a ? a + ' flees' : '';
        case 'guard-used': return a && t ? a + ' guards ' + t : '';
        // An ability names itself (a Sentinel's Overwatch, a Nightblade's Death Mark...).
        case 'ability': return a && e.status ? a + ': ' + e.status + (t && e.outcome !== 'failed' ? ' on ' + t : (e.outcome === 'failed' ? ' (failed)' : '')) : '';
        default: return '';
        }
    }

    // normRef gives events the ids the units are kept by.
    function normRef(ref) { return ref === 'me' ? 'leader' : (ref || ''); }

    // landsOnNothing is true for a happening that involves an allied unit
    // the screen does not draw (a third company, or a foe the player is not
    // fighting): there is no figure to act or to be struck, so it is dropped
    // from the picture (Phase 45). Allies' blows on foes in the fight stay.
    function isAllyRef(ref) { return /^(a|u):/.test(ref || ''); }
    function landsOnNothing(e) {
        const a = isAllyRef(e.src), t = isAllyRef(e.tgt);
        if (a && !units.get(e.src)) { return true; }
        if (t && !units.get(e.tgt)) { return true; }
        if (a && !t && e.tgt && e.tgt !== '?' && !units.get(e.tgt)) { return true; }
        if (t && !a && e.src && e.src !== '?' && !units.get(e.src)) { return true; }
        return false;
    }

    function onEvents(body) {
        const evs = (body.events || []).map(e => Object.assign({}, e, { src: normRef(e.src), tgt: normRef(e.tgt) })).filter(e => !landsOnNothing(e));
        // fight_round counts this fight's rounds; round is the server's counter.
        if (body.fight_round) { roundNo = body.fight_round; paintChrome(); }
        // The feed carries the player's pace; inference from how batches
        // arrive is only the fallback for a server that does not send it.
        if (body.pace && TL.BUDGETS[body.pace]) { feedPace = body.pace; }
        paceHistory.push({ at: Date.now(), n: evs.length });
        if (paceHistory.length > 8) { paceHistory.shift(); }
        if (motion() === 'off') {
            evs.forEach(e => { const line = narrate(e); if (line) { lastBlow = line; } applyEvent(e); });
            paintCaption();
            draw();
            return;
        }
        playEvents(evs);
        draw();
    }

    // hasArt answers the planner: does this unit have art for that pose?
    function hasArt(id, anim) {
        const u = id === '*company*' ? null : units.get(id);
        return !!(u && art('battle/units/' + u.sprite + '/' + anim + '.png'));
    }

    // playEvents plans a batch and schedules it. Hidden, or when the screen
    // cannot play (no scheduler time to give), the happenings collapse to


    function playEvents(evs) {
        const now = Date.now();
        evs.forEach(e => {
            // Refs seen for the first time make their units (an unseen '?').
            if (e.src === '?' || e.tgt === '?') { refUnit('?'); }
        });
        // Start and end of the fight are not animated: they take effect now
        // (a fight-start before this batch's happenings are held).
        evs.forEach(e => {
            if (e.kind === 'fight-start') { sched.collapse(); holding = new Set(); outcomeQueued = false; applyEvent(e); }
            else if (e.kind === 'fight-end') { ended = true; paintBadge(); }
        });
        const happenings = TL.plan(evs, {
            pace: feedPace || TL.inferPace(paceHistory), motion: motion(), has: hasArt,
        });
        const bySeq = new Map(evs.map(e => [e.seq, e]));
        happenings.forEach(h => {
            const e = bySeq.get(h.seq);
            if (e) {
                const line = narrate(e);
                if (line) { h.steps[0].ops.push({ op: 'log', when: 'start', text: line }); }
            }
            // The outcome shows when the fight's end plays, after the last
            // blow; the screen is ended (no more snapshots change it) now.
            if (h.kind === 'fight-end') {
                const fe = bySeq.get(h.seq);
                // Why it ended is read now, while the last snapshot stands.
                const outcome = fe ? fe.outcome : '';
                h.steps[0].ops.push({ op: 'outcome', when: 'start', outcome: outcome, why: outcomeReason(outcome) });
                outcomeQueued = true;
            }
            h.steps.forEach(st => st.ops.forEach(o => {
                if (o.op === 'fallen' || o.op === 'remove') { holding.add(o.unit); }
            }));
        });
        const collapsed = sched.push(happenings, now);
        collapsed.forEach(applyOp);
        if (!isShown()) {
            sched.collapse().forEach(applyOp);
        }
        wake();
    }

    // collapseNow ends all animation at once, to the picture's final state.
    function collapseNow() {
        sched.collapse().forEach(applyOp);
        holding = new Set();
        floaters = [];
        draw();
    }

    // applyOp changes the picture's state: what a step carries that is not
    // a pose or an effect.
    function applyOp(o) {
        const u = o.unit ? (o.unit === '?' ? units.get('?') : units.get(o.unit)) : null;
        switch (o.op) {
        case 'casting': if (u) { u.casting = o.spell || ''; } break;
        case 'status+': if (u && !u.unseen && o.status) { u.statuses.add(o.status); } break;
        case 'status-': if (u && !u.unseen && o.status) { u.statuses.delete(o.status); } break;
        case 'yielded': if (u) { u.yielded = true; } break;
        case 'fallen':
            holding.delete(o.unit);
            if (u) { u.fallen = true; u.casting = ''; }
            break;
        case 'remove':
            holding.delete(o.unit);
            units.delete(o.unit);
            break;
        case 'log': lastBlow = o.text; paintCaption(); break;
        case 'outcome': outcomeQueued = false; beginOutcome(o.outcome, o.why); break;
        case 'react': react(o); break;
        default: break;
        }
    }

    // react shows what a step's start brings: a tint, digits, a feedback icon.
    function react(o) {
        const u = units.get(o.unit);
        if (!u) { return; }
        const now = Date.now();
        const p = u.cell ? slotOf(u) : { x: W / 2, y: 120 };
        if (o.tint && motion() === 'full') { u.flash = now + 260; u.flashColor = o.tint; }
        (o.digits || []).forEach((d, i) => {
            floaters.push({ x: p.x, y: p.y - 34 - i * 8, text: d.text, color: d.color, born: now, big: !!d.big, dim: !!d.dim });
        });
        if (o.icon) { floaters.push({ x: p.x + 8, y: p.y - 24, icon: o.icon, born: now }); }
        if (o.bigHit && motion() === 'full') { shakeUntil = now + 160; }
    }

    function applyEvent(e) {
        const src = refUnit(e.src);
        const tgt = refUnit(e.tgt);
        switch (e.kind) {
        case 'fight-start':
            if (!battle) { return; }
            clearOutcome();
            floaters = [];
            if (ended) { ended = false; showLive(); }
            break;
        case 'attack':
            if (e.outcome === 'miss') { flash(tgt, '#9aa0aa', 'miss'); }
            else { flash(tgt, e.crit ? '#ffd23f' : '#e04b3a', e.damage ? (e.crit ? e.damage + '!' : e.damage) : ''); }
            break;
        case 'spell-hit':
            flash(tgt, '#7aa8ff', e.damage || '');
            break;
        case 'heal':
            flash(tgt, '#5fd08a', e.amount ? '+' + e.amount : '');
            break;
        case 'status-tick':
            // Bleeding and the like: a small hurt, no name for an unseen holder.
            if (e.damage) { flash(tgt, '#c06a3a', e.damage); }
            break;
        case 'cast-start':
            // A spell cast by one the player can't make out stays unnamed.
            if (src) { src.casting = e.src === '?' ? 'a spell' : (e.spell_name || e.spell || 'a spell'); wake(); }
            break;
        case 'cast-complete':
        case 'interrupt':
            if (src) { src.casting = ''; }
            break;
        case 'status-applied':
            if (tgt && !tgt.unseen && e.status) { tgt.statuses.add(e.status); }
            break;
        case 'status-expired':
            if (tgt && !tgt.unseen && e.status) { tgt.statuses.delete(e.status); }
            break;
        case 'death':
            if (tgt) { tgt.fallen = true; tgt.casting = ''; flash(tgt, '#222', ''); }
            break;
        case 'yield':
            if (src) { src.yielded = true; }
            break;
        case 'flee':
            if (src) { units.delete(src.id); }
            break;
        case 'fight-end':
            ended = true;
            beginOutcome(e.outcome);
            paintBadge();
            break;
        default:
            break;
        }
    }

    // ---------------------------------------------------------------------
    // Outcome and open/close
    // ---------------------------------------------------------------------

    const HINT = 'Hover or tap a figure for its name, health, statuses, and whom it strikes.';

    const OUTCOMES = { victory: 'Victory', defeat: 'Defeat', 'broken-off': 'The company breaks off' };

    // outcomeReason says why the fight ended, from the last snapshot.
    function outcomeReason(outcome) {
        if (outcome === 'victory') { return 'no foe is left standing'; }
        if (outcome === 'defeat') { return 'the company has fallen'; }
        if (outcome === 'broken-off') { return battle && battle.retreat ? 'the company withdrew' : 'the battle was broken off'; }
        return '';
    }

    function beginOutcome(outcome, reason) {
        if (!isShown()) { return; }
        const head = OUTCOMES[outcome] || 'The battle is over';
        const why = reason === undefined ? outcomeReason(outcome) : reason;
        outcomeText = why ? head + ': ' + why : head;
        if (outcomeTimer) { clearTimeout(outcomeTimer); }
        endExtra = 0;
        outcomeTimer = setTimeout(holdOver, OUTCOME_MS);
        paintChrome();
    }

    // holdOver closes the screen after the hold, unless the animations are
    // still playing out the fight's last moments (a little longer at most).
    function holdOver() {
        outcomeTimer = null;
        if (!sched.idle(Date.now()) && endExtra < 6000 && motion() !== 'off') {
            endExtra += 500;
            outcomeTimer = setTimeout(holdOver, 500);
            return;
        }
        endBattleView();
    }

    function clearOutcome() {
        if (outcomeTimer) { clearTimeout(outcomeTimer); outcomeTimer = null; }
        outcomeText = '';
    }

    function endBattleView() {
        outcomeText = '';
        units = new Map();
        floaters = [];
        sched = new TL.Scheduler();
        holding = new Set();
        outcomeQueued = false;
        lastBlow = '';
        roundNo = 0;
        userOpened = false;
        minimised = false;
        hide();
    }

    function isShown() { return !!overlay && overlay.classList.contains('open'); }

    function hide() {
        if (overlay) { overlay.classList.remove('open'); }
        paintBadge();
    }

    function open() {
        if ((!battle || ended) && !outcomeTimer) { return; }
        build();
        minimised = false;
        userOpened = true;
        overlay.classList.add('open');
        fit();
        paintChrome();
        draw();
        paintBadge();
    }

    function close() {
        minimised = true;
        hide();
    }

    function paintBadge() {
        if (!badge) { return; }
        const live = !!battle && !ended && !isShown();
        badge.classList.toggle('show', live);
    }

    // ---------------------------------------------------------------------
    // DOM
    // ---------------------------------------------------------------------

    let titleNode, bannerNode, lastNode, captionNode, outcomeNode, legendNode, retreatBtn, autoBox, animSelect, focusBtns = [];

    function build() {
        if (overlay) { return; }
        overlay = el('div');
        overlay.id = 'battle-screen';
        overlay.setAttribute('role', 'region');
        overlay.setAttribute('aria-label', 'Battle screen: a picture of the battle. The Combat tab lists the same information.');

        const head = el('div', 'bs-head');
        titleNode = el('span', 'bs-title', 'Battle');
        bannerNode = el('span', 'bs-banners');
        const min = el('button', null, 'Minimise');
        min.type = 'button';
        min.title = 'Shrink the picture to a badge; the battle goes on';
        min.addEventListener('click', close);
        head.appendChild(titleNode);
        head.appendChild(bannerNode);
        head.appendChild(min);
        overlay.appendChild(head);

        canvas = document.createElement('canvas');
        canvas.width = W;
        canvas.height = H;
        canvas.setAttribute('aria-hidden', 'true');
        canvas.addEventListener('pointermove', onPointer);
        canvas.addEventListener('pointerleave', () => { hover = null; paintCaption(); draw(); });
        canvas.addEventListener('click', onPointer);
        ctx = canvas.getContext('2d');
        overlay.appendChild(canvas);

        // Phase 47: names the coloured dots over the figures; the screen
        // reader gets the same words from each figure's description.
        legendNode = el('div', 'bs-legend');
        legendNode.setAttribute('aria-hidden', 'true');
        overlay.appendChild(legendNode);
        lastNode = el('div', 'bs-last');
        lastNode.setAttribute('aria-live', 'off');
        captionNode = el('div', 'bs-caption');
        outcomeNode = el('div', 'bs-outcome');
        outcomeNode.setAttribute('role', 'status');
        overlay.appendChild(lastNode);
        overlay.appendChild(captionNode);
        overlay.appendChild(outcomeNode);

        const foot = el('div', 'bs-foot');
        retreatBtn = el('button', null, 'Retreat');
        retreatBtn.type = 'button';
        retreatBtn.title = 'Withdraw your company: one round to prepare, then the attempt (retreat)';
        retreatBtn.addEventListener('click', () => Client.SendInput('retreat'));
        foot.appendChild(retreatBtn);
        // A finger can't hover for the buttons' titles, so name the row.
        foot.appendChild(el('span', 'bs-focus-label', 'Focus:'));
        FOCI.forEach(rule => {
            const b = el('button', null, rule);
            b.type = 'button';
            b.dataset.focus = rule;
            b.title = 'Company focus for this battle only (company tactics focus ' + rule + ')';
            b.addEventListener('click', () => Client.SendInput('company tactics focus ' + rule));
            focusBtns.push(b);
            foot.appendChild(b);
        });
        const animLabel = el('label');
        animSelect = document.createElement('select');
        animSelect.title = 'Battle animations: full, reduced (no flashes, shake or travelling shots), or off (a still picture)';
        MOTIONS.forEach(m => { const o = el('option', null, m); o.value = m; animSelect.appendChild(o); });
        animSelect.value = motion();
        animSelect.addEventListener('change', () => saveMotion(animSelect.value));
        animLabel.appendChild(document.createTextNode('Animation '));
        animLabel.appendChild(animSelect);
        foot.appendChild(animLabel);
        const label = el('label');
        autoBox = document.createElement('input');
        autoBox.type = 'checkbox';
        autoBox.checked = currentSetting() === 'auto';
        autoBox.addEventListener('change', () => saveSetting(autoBox.checked ? 'auto' : 'manual'));
        label.appendChild(autoBox);
        label.appendChild(document.createTextNode(' Open automatically'));
        foot.appendChild(label);
        const help = el('button', null, 'Help');
        help.type = 'button';
        help.title = 'How to read the battle screen (help battlescreen)';
        help.addEventListener('click', () => {
            Client.SendInput('help battlescreen');
            // Phase 40i: on a phone the help text lands in the Game view behind
            // this screen, so step aside to it; the badge brings the battle back.
            if (window.Mobile && window.Mobile.active()) { close(); window.Mobile.show('game'); }
        });
        foot.appendChild(help);
        overlay.appendChild(foot);
        document.body.appendChild(overlay);

        badge = el('button', null, '⚔ Battle');
        badge.id = 'battle-badge';
        badge.type = 'button';
        badge.title = 'Show the battle screen';
        badge.addEventListener('click', open);
        document.body.appendChild(badge);

        window.addEventListener('resize', fit);
    }

    // fit scales the canvas by a whole number to the room there is, down
    // to the window's width on a phone.
    function fit() {
        if (!canvas) { return; }
        const room = Math.min((window.innerWidth - 28) / W, (window.innerHeight - 150) / H);
        // Phase 40i: on a phone the picture takes the screen's whole width, at
        // any scale (the pixel art stays crisp); the buttons below need the rest.
        if (document.body.classList.contains('mobile')) {
            const w = Math.max(160, Math.min(Math.floor(window.innerWidth - 12), Math.floor((window.innerHeight - 260) * W / H)));
            canvas.style.width = w + 'px';
            canvas.style.height = Math.round(w * H / W) + 'px';
        } else if (room >= 1) {
            const s = Math.min(4, Math.floor(room));
            canvas.style.width = (W * s) + 'px';
            canvas.style.height = (H * s) + 'px';
        } else {
            const w = Math.max(160, Math.floor(window.innerWidth - 28));
            canvas.style.width = w + 'px';
            canvas.style.height = Math.round(w * H / W) + 'px';
        }
    }

    function paintChrome() {
        if (!overlay) { return; }
        titleNode.textContent = battle ? 'Battle: ' + (battle.group || 'the enemy') + (roundNo ? ', round ' + roundNo : '') : 'Battle';
        const banners = [];
        if (battle) {
            if (battle.dark) { banners.push('dark'); }
            if (battle.narrow) { banners.push('narrow'); }
            if (battle.retreat) { banners.push('withdrawing ' + battle.retreat.exit); }
            if (battle.nerve === 'faltering') { banners.push('company faltering'); }
            // Phase 39c: a Shaman's weather over the battle.
            if (battle.weather && battle.weather.name) {
                banners.push(battle.weather.name + ' (' + (battle.weather.endless ? 'the whole battle' : battle.weather.rounds + (battle.weather.rounds === 1 ? ' round' : ' rounds')) + ': ' + battle.weather.effect + ')');
            }
            // Phase 54: the sigil the company stands in.
            if (battle.sigil && battle.sigil.name) {
                banners.push(battle.sigil.name + ' (' + battle.sigil.effect + ')');
            }
            // Phase 50: members that went in hungry, parched or tired, or on a meal buff.
            const fareNames = Object.keys(battle.fare || {}).map(k => { const u = units.get(k); return u ? u.label : ''; }).filter(Boolean);
            if (fareNames.length) { banners.push('condition: ' + fareNames.join(', ')); }
            if ((battle.allies || []).length) { banners.push('allies: ' + battle.allies.map(a => a.name).join(', ')); }
            if (battle.waiting && battle.waiting.length) { banners.push('waiting: ' + battle.waiting.join(', ')); }
        }
        bannerNode.textContent = banners.join(' · ');
        outcomeNode.textContent = outcomeText;
        const ready = !!battle && battle.focus_ready !== false && !outcomeText;
        focusBtns.forEach(b => {
            b.setAttribute('aria-pressed', battle && battle.focus === b.dataset.focus ? 'true' : 'false');
            b.disabled = !ready || typeof (battle && battle.focus) !== 'string';
        });
        retreatBtn.disabled = !battle || !!outcomeText;
        autoBox.checked = currentSetting() === 'auto';
        animSelect.value = motion();
        paintCaption();
    }

    function paintCaption() {
        if (!captionNode) { return; }
        lastNode.textContent = battle && !outcomeText ? lastBlow : '';
        const u = hover ? units.get(hover) : null;
        if (!u) {
            const al = watching !== null && battle && battle.allies ? battle.allies[watching] : null;
            captionNode.textContent = battle && !outcomeText ? (al ? 'Watching ' + (al.name || 'an ally') + '\'s company. Tap your band to return.' : HINT) : '';
            return;
        }
        let text = u.label;
        if (u.side === 'company' && u.className) { text += ', ' + u.className; }
        if (u.side === 'company' && u.flasks && !u.fallen) { text += ', ' + u.flasks; }
        if (u.side === 'ally' && u.allyName) { text += ' of ' + u.allyName + '\'s company'; }
        if (isShrunk(u) && u.side === 'ally') { text += ' (tap to watch)'; }
        if (u.side !== 'company' && u.band) { text += ', ' + u.band; }
        if (u.side !== 'enemy' && u.fallen) { text += ', fallen'; }
        if (u.side === 'company' && !u.fallen && battle && battle.nerve === 'faltering') { text += ', shaken'; }
        if (u.yielded) { text += ', surrendered'; }
        // Phase 40h: the coloured marks over a figure are named here, in the
        // words the narration uses (a status the screen draws is a status
        // the server has told in text).
        if (u.side !== 'ally' && !u.fallen && u.statuses.size) {
            text += ', ' + Array.from(u.statuses).map(s => s.replace(/-/g, ' ')).join(', ');
        }
        // Phase 50: the battle condition the member went in with.
        if (u.side === 'company' && !u.fallen && battle && battle.fare && battle.fare[u.id]) { text += ', ' + battle.fare[u.id]; }
        const t = targetOf(u.id);
        if (t && units.get(t)) { text += ', striking ' + units.get(t).label; }
        captionNode.textContent = text;
    }

    // targetOf is whom a fighter strikes, from the Company.Battle feed.
    function targetOf(id) {
        if (!battle) { return ''; }
        const aim = (battle.company || []).find(a => a.key === id);
        if (aim) { return aim.target; }
        const en = (battle.enemies || []).find(e => e.id === id);
        return en && en.target ? en.target : '';
    }

    function onPointer(ev) {
        const r = canvas.getBoundingClientRect();
        const x = (ev.clientX - r.left) * W / r.width;
        const y = (ev.clientY - r.top) * H / r.height;
        let best = null;
        // A fingertip is blunter than a pointer: reach further on a touch screen.
        const reach = window.matchMedia && window.matchMedia('(pointer: coarse)').matches ? 28 : 18;
        let bestD = reach * reach;
        const compact = compactAllies();
        units.forEach(u => {
            if (!u.cell || (compact && isShrunk(u))) { return; }
            const p = slotOf(u);
            const d = (p.x - x) * (p.x - x) + (p.y - 12 - y) * (p.y - 12 - y);
            if (d < bestD) { bestD = d; best = u.id; }
        });
        if (ev.type === 'click') {
            // Phase 45: tap an allied formation (or its pennant) to watch it
            // full size; tap the shrunken company to come back.
            const target = best && units.get(best);
            const pennant = bannerAt(x, y);
            if (target && target.side === 'ally' && !isWatched(target)) { watch(target.ally); return; }
            if (target && target.side === 'company' && watching !== null) { watch(null); return; }
            if (pennant !== null) { watch(pennant === watching ? null : pennant); return; }
        }
        hover = ev.type === 'click' && hover === best ? null : best;
        paintCaption();
        draw();
    }

    // watch shows an allied company full size (index), or the player's own
    // company again (null). It changes only the view.
    function watch(index) {
        const count = Math.min(MAX_ALLIES, ((battle && battle.allies) || []).length);
        watching = index !== null && index >= 0 && index < count ? index : null;
        hover = null;
        paintCaption();
        draw();
    }

    // bannerAt is the allied pennant under a canvas point, or null.
    function bannerAt(x, y) {
        const g = allyGroups();
        // A fingertip needs a little more pennant than a pointer does.
        const pad = window.matchMedia && window.matchMedia('(pointer: coarse)').matches ? 8 : 0;
        for (const a of g.list) {
            const fx = ALLY_ANCHORS[a.index] - 38;
            if (x >= fx - 2 - pad && x <= fx + 46 + pad && y >= 20 - pad && y <= 42 + pad) { return a.index; }
        }
        return null;
    }

    // ---------------------------------------------------------------------
    // Drawing
    // ---------------------------------------------------------------------

    function wake() {
        if (raf || !isShown()) { return; }
        raf = requestAnimationFrame(tick);
    }

    function tick() {
        raf = 0;
        draw();
        const now = Date.now();
        floaters = floaters.filter(f => now - f.born < 900);
        let live = floaters.length > 0 || !sched.idle(now) || shakeUntil > now;
        units.forEach(u => { if (u.flash > now) { live = true; } });
        if (live) { wake(); }
    }

    // ---------------------------------------------------------------------
    // Poses: what a unit is doing now, from the steps under way
    // ---------------------------------------------------------------------

    // tri rises from 0 to 1 at the middle of a step and falls back.
    function tri(p) { return p < 0.5 ? p * 2 : (1 - p) * 2; }

    // poseOf reads a unit's active steps into a pose: an offset from its
    // slot, a vertical squash, and the art animation to draw (with its
    // progress) when the art has it. Without art the idle figure nudges.
    function poseOf(u, p0, now) {
        const dir = u.side !== 'enemy' ? 1 : -1;
        const pose = { dx: 0, dy: 0, squash: 1, anim: '', p: 0, loop: false, alpha: 1, scale: isShrunk(u) ? ALLY_SCALE : 1 };
        const full = motion() === 'full';
        sched.active(now).forEach(st => {
            if (st.unit !== u.id && !(st.unit === '*company*' && u.side === 'company')) { return; }
            const p = Math.max(0, Math.min(1, (now - st.start) / Math.max(1, st.end - st.start)));
            let anim = st.anim;
            // An allied unit, small and far, strikes from where it stands.
            if (st.lunge && !isShrunk(u) && units.get(st.lunge) && units.get(st.lunge).cell) {
                // Step in toward the target, strike at the middle, step back.
                const t = units.get(st.lunge);
                const tp = slotOf(t);
                const e = tri(p);
                pose.dx += (tp.x - dir * 16 - p0.x) * e;
                pose.dy += (tp.y - p0.y) * e;
                // Art: walk in, the blow in the middle, walk back.
                if (p < 0.3 || p > 0.7) { if (hasArt(u.id, 'walk')) { anim = 'walk'; } }
            }
            pose.anim = anim || pose.anim;
            pose.p = p;
            pose.loop = !!st.loop;
            if (st.exit) {
                pose.dx += dir * -1 * p * (W / 2 - 10);
                pose.alpha = 1 - p * 0.6;
            } else if (st.fade) {
                pose.alpha = 1 - p;
            }
            if (st.nudge || !anim) {
                // No art for this pose: the idle figure moves a little.
                switch (st.anim === '' ? st.role : (FALLBACK_NAME[st.anim] || st.anim)) {
                case 'attack': case 'shoot': if (!st.lunge) { pose.dx += dir * 5 * tri(p); } break;
                case 'hurt': case 'block': case 'parry': pose.dx -= dir * 3 * tri(p); break;
                case 'dodge': pose.dx -= dir * 6 * tri(p); pose.dy -= 2 * tri(p); break;
                case 'windup': pose.dx -= dir * 3; break;
                case 'cast': pose.dy -= Math.abs(Math.sin(p * Math.PI * 4)); break;
                case 'prone': pose.squash = p < 0.6 ? 0.45 : 0.45 + (p - 0.6) * 1.4; break;
                case 'down': pose.squash = 1 - 0.7 * p; break;
                case 'victory': pose.dy -= 2 * Math.abs(Math.sin(p * Math.PI * 2)); break;
                default: break;
                }
            }
        });
        if (!full) { pose.dx = Math.round(pose.dx); pose.dy = Math.round(pose.dy); }
        // A winding-up unit holds its pose until the blow lands.
        if (u.statuses.has('winding-up') && !pose.anim && pose.dx === 0) { pose.dx = -dir * 3; pose.anim = 'windup'; pose.p = 1; }
        return pose;
    }

    const FALLBACK_NAME = { 'cast-release': 'cast', 'shield-bash': 'hurt', 'guard-step': 'walk' };

    function rect(x, y, w, h, c) {
        ctx.fillStyle = c;
        ctx.fillRect(Math.round(x), Math.round(y), w, h);
    }

    function zoneSlug() { return zone.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, ''); }

    function scene() {
        return SCENES[biome] || SCENES.land;
    }

    function drawBackground() {
        const s = scene();
        // A zone's own backdrop, when the art set has one, comes first.
        const bg = (zoneSlug() && art('battle/backgrounds/zone-' + zoneSlug() + '.png')) || art('battle/backgrounds/' + s.bg + '.png');
        if (bg) { ctx.drawImage(bg.img, 0, 0, W, H); return; }
        const g = ctx.createLinearGradient(0, 0, 0, 100);
        g.addColorStop(0, s.sky[0]);
        g.addColorStop(1, s.sky[1]);
        ctx.fillStyle = g;
        ctx.fillRect(0, 0, W, 100);
        // The far line.
        ctx.fillStyle = s.far;
        for (let x = 0; x < W; x += 8) {
            let h = 14;
            if (s.style === 'peaks') { h = 18 + ((x * 7) % 31); }
            else if (s.style === 'trees') { h = 20 + ((x * 5) % 13); }
            else if (s.style === 'wall') { h = ((x / 8) % 3 === 0) ? 24 : 30; }
            else if (s.style === 'room' || s.style === 'cave') { h = 26 + ((x * 3) % 9); }
            else { h = 12 + ((x * 3) % 10); }
            ctx.fillRect(x, 100 - h, 8, h);
        }
        rect(0, 100, W, H - 100, s.ground);
        // Ground lanes, so the three depth lanes read.
        ctx.fillStyle = 'rgba(0,0,0,0.12)';
        [120, 140, 160].forEach(y => ctx.fillRect(0, y + 1, W, 1));
        // The centre line between the two sides.
        ctx.fillStyle = 'rgba(255,255,255,0.12)';
        ctx.fillRect(W / 2 - 1, 104, 2, H - 104);
    }

    function shade(hex, f) {
        const n = parseInt(hex.slice(1), 16);
        const c = k => Math.max(0, Math.min(255, Math.round(((n >> k) & 255) * f)));
        return 'rgb(' + c(16) + ',' + c(8) + ',' + c(0) + ')';
    }

    function hashHue(key) {
        let h = 0;
        for (let i = 0; i < key.length; i++) { h = (h * 31 + key.charCodeAt(i)) >>> 0; }
        return h % 360;
    }

    // drawFigure draws one unit standing with its feet at (x, y), facing
    // the centre line.
    function drawFigure(u, x, y, now, pose) {
        pose = pose || { dx: 0, dy: 0, squash: 1, anim: '', p: 0, loop: false, alpha: 1, scale: 1 };
        const sc = pose.scale || 1;
        const dir = u.side !== 'enemy' ? 1 : -1;         // +1 faces right
        const dim = battle && battle.dark ? 0.5 : 1;
        const flashing = u.flash > now;
        // Shadow.
        ctx.fillStyle = 'rgba(0,0,0,0.3)';
        ctx.fillRect(Math.round(x - 7 * sc), y - 1, Math.round(14 * sc), Math.max(1, Math.round(2 * sc)));
        // The pose moves the figure; its shadow stays on the ground.
        ctx.save();
        ctx.globalAlpha = pose.alpha;
        if (pose.dx || pose.dy || pose.squash !== 1 || sc !== 1) {
            ctx.translate(Math.round(x + pose.dx), Math.round(y + pose.dy));
            ctx.scale(sc, sc * pose.squash);
            ctx.translate(-Math.round(x), -y);
        }
        drawBody(u, x, y, now, pose, dir, dim);
        ctx.restore();
        if (flashing && !u.fallen && !u.tinted) {
            ctx.fillStyle = u.flashColor;
            ctx.globalAlpha = 0.45;
            ctx.fillRect(Math.round(x + pose.dx - 10 * sc), Math.round(y + pose.dy - 32 * sc), Math.round(20 * sc), Math.round(33 * sc));
            ctx.globalAlpha = 1;
        }
    }

    // tintLayer is a scratch canvas the size of a frame, for tinting art.
    let tintCanvas = null;
    function tintLayer(w, h) {
        if (!tintCanvas) { tintCanvas = document.createElement('canvas'); }
        if (tintCanvas.width < w || tintCanvas.height < h) { tintCanvas.width = Math.max(w, tintCanvas.width); tintCanvas.height = Math.max(h, tintCanvas.height); }
        return tintCanvas.getContext('2d');
    }

    function drawBody(u, x, y, now, pose, dir, dim) {
        u.tinted = false;
        if (u.fallen) {
            // Lying down: a flat shape, the figure's body hue.
            const c = u.side !== 'enemy' ? (CLASS_HUES[u.klass] || DEFAULT_HUES)[0] : '#6a4a4a';
            rect(x - 9, y - 5, 18, 4, shade(c, 0.6));
            rect(x - (9 * dir), y - 7, 4, 4, '#c8a888');
            return;
        }
        if (u.unseen) {
            rect(x - 6, y - 26, 12, 26, 'rgba(10,10,16,0.75)');
            rect(x - 4, y - 30, 8, 6, 'rgba(10,10,16,0.75)');
            return;
        }
        // Art: the pose's own sheet when there is one (S4), else the idle
        // loop, anchored bottom-centre, enemies mirrored. A promoted member
        // (40s5) keeps to its class's sheets once its idle exists, so it never
        // flickers into the base class's poses.
        const key = u.promoted && art('battle/units/' + u.promoted + '/idle.png') ? u.promoted : u.sprite;
        const look = window.SpriteTint ? window.SpriteTint.look(u.skin, u.hair) : null;
        const posedPath = pose.anim ? 'battle/units/' + key + '/' + pose.anim + '.png' : '';
        const posedArt = posedPath ? art(posedPath) : null;
        const sheetPath = posedArt ? posedPath : 'battle/units/' + key + '/idle.png';
        const posed = posedArt ? tintedArt(posedArt, posedPath, look) : null;
        const sheet = posed || tintedArt(art(sheetPath), sheetPath, look);
        if (sheet) {
            const fw = (sheet.info.frame || [64, 64])[0], fh = (sheet.info.frame || [64, 64])[1];
            const frames = sheet.info.frames || 1;
            const i = posed && !pose.loop ? Math.min(frames - 1, Math.floor(pose.p * frames))
                : Math.floor(now / (sheet.info.frame_ms || 250)) % frames;
            ctx.save();
            ctx.translate(Math.round(x), y);
            if (dir < 0) { ctx.scale(-1, 1); }
            if (dim < 1) { ctx.filter = 'brightness(0.5)'; }
            ctx.drawImage(sheet.img, i * fw, 0, fw, fh, -fw / 2, -fh, fw, fh);
            if (u.flash > now) {
                // A hit tints the figure's own shape, not a box around it.
                const t = tintLayer(fw, fh);
                t.clearRect(0, 0, fw, fh);
                t.globalCompositeOperation = 'source-over';
                t.drawImage(sheet.img, i * fw, 0, fw, fh, 0, 0, fw, fh);
                t.globalCompositeOperation = 'source-atop';
                t.globalAlpha = 0.55;
                t.fillStyle = u.flashColor;
                t.fillRect(0, 0, fw, fh);
                t.globalAlpha = 1;
                ctx.drawImage(tintCanvas, 0, 0, fw, fh, -fw / 2, -fh, fw, fh);
            }
            ctx.restore();
            u.tinted = u.flash > now;
        } else if (u.side !== 'enemy') {
            const hues = CLASS_HUES[u.klass] || DEFAULT_HUES;
            const body = shade(hues[0], dim), trim = shade(hues[1], dim);
            rect(x - 3, y - 8, 2, 8, '#2a2a30');                 // legs
            rect(x + 1, y - 8, 2, 8, '#2a2a30');
            rect(x - 4, y - 20, 8, 12, body);                    // torso
            rect(x - 4, y - 14, 8, 2, trim);                     // belt
            rect(x - 3, y - 26, 6, 6, '#e0b890');                // head
            rect(x - 3, y - 27, 6, 2, shade(hues[1], 0.7 * dim)); // hair or hood
            const wx = dir > 0 ? x + 4 : x - 6;                  // weapon in the leading hand
            if (u.klass === 'wizard' || u.klass === 'witch' || u.klass === 'cleric') {
                rect(wx + (dir > 0 ? 1 : 3), y - 30, 1, 22, '#8a6a3b');
                rect(wx + (dir > 0 ? 0 : 2), y - 32, 3, 3, trim);
            } else if (u.klass === 'ranger') {
                rect(wx + (dir > 0 ? 2 : 0), y - 22, 1, 14, '#8a6a3b');
            } else {
                rect(wx + (dir > 0 ? 1 : 1), y - 24, 1, 12, '#c8ccd4');
            }
            if (u.leader) { rect(x - 2, y - 30, 4, 2, '#ffd23f'); } // the crown pip
        } else {
            const key = u.sprite;
            const size = SIZES[key] || SIZES['unknown-humanoid'];
            const hue = hashHue(key);
            const body = 'hsl(' + hue + ',35%,' + Math.round(38 * dim) + '%)';
            const dark = 'hsl(' + hue + ',30%,' + Math.round(22 * dim) + '%)';
            if (key === 'unknown-beast' || /wolf|boar|rat|spider|beast/.test(key)) {
                rect(x - 9, y - 12, 18, 7, body);                 // body
                rect(x - 9 - (dir < 0 ? 0 : 0), y - 5, 3, 5, dark); // legs
                rect(x + 6, y - 5, 3, 5, dark);
                rect(dir > 0 ? x + 7 : x - 13, y - 15, 6, 6, body);  // head toward the centre
                rect(dir > 0 ? x + 11 : x - 12, y - 13, 1, 1, '#ff5a3c');
            } else {
                const w = size[0], h = size[1];
                rect(x - w / 2, y - h * 0.3, w * 0.4, h * 0.3, dark);          // legs
                rect(x + w * 0.1, y - h * 0.3, w * 0.4, h * 0.3, dark);
                rect(x - w / 2, y - h * 0.75, w, h * 0.5, body);               // torso
                rect(x - w * 0.3, y - h, w * 0.6, h * 0.25, body);             // head
                rect(dir > 0 ? x + w * 0.1 : x - w * 0.3, y - h * 0.9, 2, 2, '#ff5a3c'); // eyes, toward the centre
                rect(dir > 0 ? x + w / 2 + 2 : x - w / 2 - 3, y - h * 0.7, 1, h * 0.45, '#b0b4bc'); // a weapon in the leading hand
            }
        }
    }

    // drawInfo draws the bar, role badge, statuses and chant mark.
    function drawInfo(u, x, y) {
        if (u.fallen || u.unseen) { return; }
        if (isShrunk(u)) { drawAllyInfo(u, x, y); return; }
        const w = 18;
        rect(x - w / 2, y + 3, w, 3, '#101014');
        if (u.side === 'company' && !isShrunk(u)) {
            const f = Math.max(0, Math.min(1, u.frac));
            rect(x - w / 2 + 1, y + 4, Math.round((w - 2) * f), 1, f > 0.5 ? '#4fc16a' : (f > 0.25 ? '#e0b030' : '#d84a3a'));
        } else {
            // Banded: five segments, never a number.
            const segs = Math.max(0, Math.round(u.frac * 5));
            for (let i = 0; i < 5; i++) {
                rect(x - w / 2 + 1 + i * 3 + (i > 0 ? 0 : 0), y + 4, 2, 1, i < segs ? '#d8c24a' : '#3a3a42');
            }
        }
        if (u.role && ROLE_GLYPH[u.role]) {
            // A pixel-font letter on a dark chip: crisp at every scale.
            rect(Math.round(x - w / 2), y + 7, 5, 7, '#101014');
            pixText(ROLE_GLYPH[u.role], x - w / 2 + 1, y + 8, '#e8e8f0');
        }
        if (u.side === 'company' && battle && battle.nerve === 'faltering') {
            // The company's nerve is tested: a drop of sweat beside the bar.
            rect(Math.round(x + w / 2 - 1), y + 8, 1, 2, '#e8a838');
            rect(Math.round(x + w / 2 - 2), y + 10, 3, 2, '#e8a838');
        }
        if (u.yielded) {
            pixText('yields', x - 11, y - 44, '#e8e8f0');
        }
        let i = 0;
        u.statuses.forEach(s => {
            rect(x - 9 + i * 4, y - 36, 3, 3, 'hsl(' + hashHue(s) + ',70%,60%)');
            i++;
        });
        if (u.casting) { rect(x - 1, y - 40, 2, 2, '#9ab0ff'); rect(x - 3, y - 38, 6, 1, '#9ab0ff'); }
    }

    // drawAllyInfo draws an allied unit's small band bar (words, never a
    // number) and its chant mark.
    function drawAllyInfo(u, x, y) {
        const segs = Math.max(0, Math.round(u.frac * 5));
        rect(x - 6, y + 2, 12, 3, '#101014');
        for (let i = 0; i < 5; i++) { rect(x - 5 + i * 2, y + 3, 1, 1, i < segs ? '#d8c24a' : '#3a3a42'); }
        if (u.casting) { rect(x, y - 22, 1, 1, '#9ab0ff'); rect(x - 1, y - 21, 3, 1, '#9ab0ff'); }
    }

    // compactAllies is true when the canvas shows too small for half-scale
    // figures (a phone): each allied company is a pennant instead.
    function compactAllies() {
        if (!canvas) { return false; }
        const r = canvas.getBoundingClientRect();
        return r.width > 0 && r.width < 420;
    }

    // allyGroups are the allied companies drawn, each with its members that
    // stand, and how many more companies there are than are drawn.
    function allyGroups() {
        const list = ((battle && battle.allies) || []).slice(0, MAX_ALLIES).map((al, index) => ({
            index, name: al.name || '', up: units.size ? Array.from(units.values()).filter(u => u.side === 'ally' && u.ally === index && !u.fallen).length : 0,
        }));
        return { list, more: Math.max(0, ((battle && battle.allies) || []).length - MAX_ALLIES) };
    }

    // drawAllyBanners labels each allied formation with a pennant and its
    // leader's name; on a phone the pennant (with a count of those standing)
    // stands for the formation. Companies beyond the two drawn collapse to a
    // "+N" pennant.
    function drawAllyBanners() {
        const g = allyGroups();
        const compact = compactAllies();
        g.list.forEach(a => {
            const ax = ALLY_ANCHORS[a.index];
            // While one is watched, its place holds the player's company.
            const mine = watching === a.index;
            const label = mine ? 'your band' : a.name;
            const hue = hashHue(label || 'ally');
            const fx = ax - 38;
            rect(fx, 22, 1, 12, '#c8ccd4');
            ctx.fillStyle = 'hsl(' + hue + ',55%,45%)';
            ctx.fillRect(fx + 1, 22, 6, 4);
            if (compact) {
                pixText(String(a.up), fx + 9, 22, '#ffffff');
                pixText(label.slice(0, 6), fx + 2, 36, '#e8e8f0');
            } else {
                pixText(label.slice(0, 9), fx + 9, 22, '#e8e8f0');
            }
        });
        if (g.more > 0) {
            pixText('+' + g.more + ' more', 4, 4, '#e8e8f0');
        }
    }

    // legendOf lists the statuses whose dots are on the picture now, in the
    // order first seen, so the legend names exactly the colours on screen.
    function legendOf(list) {
        const seen = [];
        list.forEach(u => {
            if (u.fallen || u.unseen || isShrunk(u)) { return; }
            u.statuses.forEach(s => { if (seen.indexOf(s) < 0) { seen.push(s); } });
        });
        return seen;
    }

    let legendKey = '';
    function paintLegend(list) {
        if (!legendNode) { return; }
        const seen = legendOf(list);
        const key = seen.join('|');
        if (key === legendKey) { return; }
        legendKey = key;
        legendNode.textContent = '';
        seen.forEach(s => {
            const item = el('span', 'bs-legend-item');
            const dot = el('span', 'bs-dot');
            dot.style.background = 'hsl(' + hashHue(s) + ',70%,60%)';
            item.appendChild(dot);
            item.appendChild(document.createTextNode(s.replace(/-/g, ' ')));
            legendNode.appendChild(item);
        });
    }

    function draw() {
        if (!ctx || !isShown()) { return; }
        const now = Date.now();
        sched.due(now).forEach(applyOp);
        ctx.save();
        if (shakeUntil > now) { ctx.translate(Math.round((Math.random() - 0.5) * 3), 0); }
        ctx.imageSmoothingEnabled = false;
        drawBackground();
        // Back to front by lane (then row), so nearer units overlap.
        const compact = compactAllies();
        const list = Array.from(units.values()).filter(u => u.cell && !(compact && isShrunk(u)));
        // Small formations stand behind: they go first, then the lanes.
        list.sort((a, b) => ((isShrunk(a) ? 0 : 1) - (isShrunk(b) ? 0 : 1)) || (a.cell.col - b.cell.col) || (b.cell.row - a.cell.row));
        drawAllyBanners();
        list.forEach(u => {
            const p = slotOf(u);
            drawFigure(u, p.x, p.y, now, poseOf(u, p, now));
        });
        // Bars, roles and statuses go over every figure, so a large unit in
        // front never hides the health of those behind it.
        list.forEach(u => {
            const p = slotOf(u);
            drawInfo(u, p.x, p.y);
        });
        paintLegend(list);
        drawEffects(now);
        // Target lines for the hovered unit, and anyone striking it.
        if (hover && units.get(hover)) {
            ctx.strokeStyle = 'rgba(255,255,255,0.7)';
            ctx.lineWidth = 1;
            const pairs = [];
            const t = targetOf(hover);
            if (t) { pairs.push([hover, t]); }
            units.forEach((_, id) => { if (id !== hover && targetOf(id) === hover) { pairs.push([id, hover]); } });
            pairs.forEach(pr => {
                const a = units.get(pr[0]), b = units.get(pr[1]);
                if (!a || !b || !a.cell || !b.cell) { return; }
                const pa = slotOf(a), pb = slotOf(b);
                ctx.beginPath();
                ctx.moveTo(pa.x, pa.y - 12);
                ctx.lineTo(pb.x, pb.y - 12);
                ctx.stroke();
            });
        }
        floaters.forEach(f => {
            const age = (now - f.born) / 900;
            ctx.globalAlpha = Math.max(0, 1 - age);
            if (f.icon) {
                drawFeedback(f.icon, Math.round(f.x), Math.round(f.y - age * 8), age);
            } else {
                ctx.font = (f.big ? 'bold 13px' : (f.dim ? '8px' : 'bold 9px')) + ' monospace';
                ctx.fillStyle = f.color;
                ctx.fillText(f.text, Math.round(f.x - 4), Math.round(f.y - age * 14));
            }
            ctx.globalAlpha = 1;
        });
        if (battle && battle.dark) {
            ctx.fillStyle = 'rgba(0,0,10,0.5)';
            ctx.fillRect(0, 0, W, H);
        }
        // A fight that ends in defeat or a breaking off fades the screen.
        sched.active(now).forEach(st => {
            if (st.fadeScreen) {
                ctx.fillStyle = 'rgba(0,0,0,' + (0.55 * (now - st.start) / Math.max(1, st.end - st.start)) + ')';
                ctx.fillRect(0, 0, W, H);
            }
        });
        ctx.restore();
    }

    // ---------------------------------------------------------------------
    // Effects: hits, projectiles, glows (S4 art when listed, else code)
    // ---------------------------------------------------------------------

    function centreOf(id) {
        const u = units.get(id);
        if (!u || !u.cell) { return null; }
        const p = slotOf(u);
        return { x: p.x, y: p.y - 12, u: u };
    }

    function spellHue(spell) { return spell ? hashHue(spell) : 220; }

    function drawEffects(now) {
        sched.activeFx(now).forEach(f => {
            if (f.kind === 'projectile') {
                const a = centreOf(f.from), b = centreOf(f.to);
                if (!a || !b) { return; }
                const x = a.x + (b.x - a.x) * f.p, y = a.y + (b.y - a.y) * f.p - Math.sin(f.p * Math.PI) * 6;
                if (f.style === 'arrow') {
                    const dx = Math.sign(b.x - a.x) * 4;
                    ctx.fillStyle = '#d8d0b0';
                    ctx.fillRect(Math.round(Math.min(x, x - dx)), Math.round(y), 5, 1);
                    ctx.fillStyle = '#b0b4bc';
                    ctx.fillRect(Math.round(x + (dx > 0 ? 1 : -2)), Math.round(y) - 1, 2, 3);
                } else {
                    ctx.fillStyle = 'hsl(' + spellHue(f.spell) + ',80%,70%)';
                    ctx.globalAlpha = 0.5;
                    ctx.fillRect(Math.round(x - 3), Math.round(y - 3), 6, 6);
                    ctx.globalAlpha = 1;
                    ctx.fillRect(Math.round(x - 1), Math.round(y - 1), 3, 3);
                }
                return;
            }
            const c = centreOf(f.on || f.unit);
            if (!c) { return; }
            if (f.kind === 'hit') { drawHit(f, c); }
            else if (f.kind === 'glow') { drawGlow(f, c); }
            else if (f.kind === 'fade-glow') { drawGlow(f, c, 1 - f.p); }
            else if (f.kind === 'heal') {
                ctx.fillStyle = '#5fd08a';
                for (let i = 0; i < 3; i++) {
                    const px = Math.round(c.x - 6 + i * 6), py = Math.round(c.y + 6 - f.p * 18 - i * 3);
                    ctx.globalAlpha = 1 - f.p;
                    ctx.fillRect(px, py - 2, 1, 5);
                    ctx.fillRect(px - 2, py, 5, 1);
                }
                ctx.globalAlpha = 1;
            } else if (f.kind === 'tick') {
                ctx.fillStyle = '#c06a3a';
                ctx.globalAlpha = 0.5 * (1 - f.p);
                ctx.fillRect(Math.round(c.x - 6), Math.round(c.y - 8), 12, 18);
                ctx.globalAlpha = 1;
            } else if (f.kind === 'armor-shatter') {
                ctx.fillStyle = '#b8bcc4';
                for (let i = 0; i < 6; i++) {
                    const a = i * 1.05;
                    ctx.fillRect(Math.round(c.x + Math.cos(a) * f.p * 12), Math.round(c.y + Math.sin(a) * f.p * 12 + f.p * f.p * 8), 2, 2);
                }
            } else if (f.kind === 'highlight') {
                ctx.strokeStyle = 'rgba(255,255,255,' + (0.8 * (1 - f.p)) + ')';
                ctx.strokeRect(Math.round(c.x - 7) + 0.5, Math.round(c.y - 16) + 0.5, 14, 28);
            }
        });
    }

    function drawGlow(f, c, fade) {
        const hue = spellHue(f.spell);
        const r = (f.burst ? 8 + f.p * 8 : 5 + Math.sin(f.p * Math.PI * 4) * 2);
        ctx.globalAlpha = (fade === undefined ? (f.burst ? 1 - f.p : 0.6) : 0.6 * fade);
        ctx.strokeStyle = 'hsl(' + hue + ',80%,70%)';
        ctx.beginPath();
        ctx.arc(Math.round(c.x), Math.round(c.y - 14), r, 0, Math.PI * 2);
        ctx.stroke();
        ctx.globalAlpha = 1;
    }

    // drawHit draws a hit effect: the S4 sheet (battle/effects/hits/<id>.png,
    // 32x32, 4 frames) when listed, else a few strokes in code. A critical
    // hit layers a starburst over the blow.
    function drawHit(f, c) {
        const sheet = art('battle/effects/hits/' + f.id + '.png');
        if (sheet) {
            const frames = sheet.info.frames || 4;
            const i = Math.min(frames - 1, Math.floor(f.p * frames));
            ctx.drawImage(sheet.img, i * 32, 0, 32, 32, Math.round(c.x - 16), Math.round(c.y - 16), 32, 32);
        } else {
            drawHitCode(f.id, c.x, c.y, f.p, f.spell);
        }
        if (f.crit) { drawHitCode('crit', c.x, c.y, f.p); }
    }

    function drawHitCode(id, x, y, p, spell) {
        x = Math.round(x); y = Math.round(y);
        ctx.save();
        ctx.globalAlpha = Math.max(0, 1 - p * p);
        ctx.lineWidth = 1;
        switch (id) {
        case 'slash': case 'cleave':
            ctx.strokeStyle = id === 'cleave' ? '#f0d0a0' : '#f0e8e8';
            ctx.lineWidth = id === 'cleave' ? 2 : 1;
            ctx.beginPath(); ctx.arc(x, y + 6, 12, -2.2 + p * 0.8, -0.9 + p * 0.8); ctx.stroke();
            break;
        case 'stab':
            ctx.fillStyle = '#f0e8e8';
            ctx.fillRect(x - 8 + Math.round(p * 6), y, 10, 1);
            ctx.fillStyle = '#ffd23f'; ctx.fillRect(x + Math.round(p * 6), y - 1, 2, 3);
            break;
        case 'blunt':
            ctx.strokeStyle = '#e8d8b8';
            ctx.beginPath(); ctx.arc(x, y, 3 + p * 9, 0, Math.PI * 2); ctx.stroke();
            ctx.fillStyle = '#a89878'; ctx.fillRect(x - 6, y + 8, 3, 2); ctx.fillRect(x + 4, y + 8, 3, 2);
            break;
        case 'claw':
            ctx.strokeStyle = '#f0e0d0';
            for (let i = -1; i <= 1; i++) { ctx.beginPath(); ctx.moveTo(x - 6 + i * 4, y - 8); ctx.lineTo(x + 2 + i * 4, y + 8 * (0.4 + p)); ctx.stroke(); }
            break;
        case 'bite':
            ctx.strokeStyle = '#f0e0d0';
            ctx.beginPath(); ctx.moveTo(x - 7, y - 4 - p * 3); ctx.lineTo(x, y + 2); ctx.lineTo(x + 7, y - 4 - p * 3); ctx.stroke();
            ctx.beginPath(); ctx.moveTo(x - 7, y + 8 + p * 3); ctx.lineTo(x, y + 2); ctx.lineTo(x + 7, y + 8 + p * 3); ctx.stroke();
            break;
        case 'arrow-hit':
            ctx.fillStyle = '#c8a878';
            for (let i = 0; i < 5; i++) { ctx.fillRect(x + Math.round(Math.cos(i * 1.3) * p * 9), y + Math.round(Math.sin(i * 1.3) * p * 9), 1, 1); }
            break;
        case 'magic-hit':
            ctx.strokeStyle = 'hsl(' + spellHue(spell) + ',85%,70%)';
            ctx.beginPath(); ctx.arc(x, y, 2 + p * 11, 0, Math.PI * 2); ctx.stroke();
            break;
        case 'crit':
            ctx.strokeStyle = '#ffd23f';
            for (let i = 0; i < 8; i++) {
                const a = i * Math.PI / 4;
                ctx.beginPath(); ctx.moveTo(x + Math.cos(a) * 4, y + Math.sin(a) * 4); ctx.lineTo(x + Math.cos(a) * (8 + p * 8), y + Math.sin(a) * (8 + p * 8)); ctx.stroke();
            }
            break;
        case 'shield-bash':
            ctx.fillStyle = '#ffd23f';
            for (let i = 0; i < 3; i++) { ctx.fillRect(x - 8 + i * 8, y - 18 - Math.round(Math.sin(p * 6 + i) * 2), 2, 2); }
            ctx.strokeStyle = '#c8ccd4'; ctx.strokeRect(x - 5.5, y - 5.5, 11, 11);
            break;
        case 'chant-broken':
            ctx.fillStyle = '#9ab0ff';
            for (let i = 0; i < 6; i++) { ctx.fillRect(x + Math.round(Math.cos(i * 1.05) * p * 14), y - 14 + Math.round(Math.sin(i * 1.05) * p * 10), 2, 2); }
            break;
        case 'guard-intercept':
            ctx.strokeStyle = '#d8e0f0'; ctx.lineWidth = 2;
            ctx.beginPath(); ctx.arc(x, y, 7 + p * 3, -1.2, 1.2); ctx.stroke();
            break;
        default: break;
        }
        ctx.restore();
    }

    // drawFeedback draws a defense icon (S4: 16x16 art when listed): a
    // whoosh for a miss, a shield, crossed blades, an afterimage, a grey
    // pulse for a lost action.
    function drawFeedback(id, x, y, age) {
        const sheet = art('battle/effects/feedback/' + id + '.png');
        if (sheet) {
            const frames = sheet.info.frames || 3;
            ctx.drawImage(sheet.img, Math.min(frames - 1, Math.floor(age * frames)) * 16, 0, 16, 16, x - 8, y - 8, 16, 16);
            return;
        }
        ctx.strokeStyle = '#d8dce4';
        ctx.fillStyle = '#d8dce4';
        switch (id) {
        case 'miss': ctx.fillRect(x - 6, y, 10, 1); ctx.fillRect(x - 3, y + 2, 8, 1); break;
        case 'blocked': ctx.strokeRect(x - 3.5, y - 4.5, 7, 8); ctx.fillRect(x - 1, y - 2, 2, 2); break;
        case 'parried':
            ctx.beginPath(); ctx.moveTo(x - 5, y - 5); ctx.lineTo(x + 5, y + 5); ctx.moveTo(x + 5, y - 5); ctx.lineTo(x - 5, y + 5); ctx.stroke();
            break;
        case 'dodged': ctx.globalAlpha *= 0.5; ctx.fillRect(x - 4, y - 5, 4, 10); ctx.fillRect(x + 1, y - 5, 4, 10); break;
        case 'skip': ctx.fillStyle = '#8a8f99'; ctx.beginPath(); ctx.arc(x, y, 3 + age * 4, 0, Math.PI * 2); ctx.fill(); break;
        default: break;
        }
    }

    // ---------------------------------------------------------------------
    // GMCP wiring
    // ---------------------------------------------------------------------

    function onBattle(body) {
        const next = (body && Array.isArray(body.enemies)) ? body : null;
        const was = battle;
        battle = next;
        if (next && !was) {
            // A new battle: a fresh picture, and the screen opens itself.
            clearOutcome();
            ended = false;
            units = new Map();
            floaters = [];
            sched = new TL.Scheduler();
            holding = new Set();
            outcomeQueued = false;
            paceHistory = [];
            feedPace = '';
            lastBlow = '';
            roundNo = 0;
            minimised = false;
            userOpened = false;
            watching = null;
        }
        // The watched company left the fight: back to the player's own view.
        if (watching !== null && (!next || watching >= Math.min(MAX_ALLIES, (next.allies || []).length))) { watching = null; }
        if (next && ended) {
            // The fight-end event came first (it rides with the narration):
            // a snapshot of the finished battle neither clears the outcome
            // nor reopens a screen the hold has closed.
            paintChrome();
        } else if (next) {
            showLive();
        } else if (was) {
            // The fight is over. The outcome event normally started the
            // three-second hold; if none came, show a plain ending.
            // While the fight's end is still to play, its outcome follows.
            if (!outcomeTimer && !outcomeQueued) { beginOutcome(''); }
            if (!outcomeTimer && !outcomeQueued) { endBattleView(); }
            paintChrome();
        }
        paintBadge();
    }

    // showLive refreshes the picture of a battle in progress and opens it
    // unless it was minimised or the player keeps it shut.
    function showLive() {
        rebuild();
        build();
        if (!isShown() && !minimised && (currentSetting() === 'auto' || userOpened)) {
            overlay.classList.add('open');
            fit();
        }
        paintChrome();
        draw();
        paintBadge();
    }

    VirtualWindows.register({
        gmcpHandlers: ['Company', 'Room'],
        onGMCP(namespace, body) {
            if (namespace === 'Company.Battle') {
                onBattle(body);
                return;
            }
            if (namespace === 'Room' || namespace === 'Room.Info') {
                const info = body && (body.Info || body);
                if (info && typeof info.environment === 'string') { biome = info.environment; }
                if (info && typeof info.area === 'string') { zone = info.area; }
                if (info) { draw(); }
                return;
            }
            if ((namespace === 'Company' || namespace === 'Company.Vitals') && battle) {
                rebuild();
                paintChrome();
                draw();
            }
        },
    });

    Client.onBattleEvents(onEvents);
    loadManifest();

    window.BattleScreen = {
        open,
        close,
        setMotion: saveMotion,
        watch,
        slot,
        // state is what a test reads: the units and where they stand.
        state() {
            return {
                open: isShown(),
                minimised,
                outcome: outcomeText,
                biome,
                zone,
                round: roundNo,
                lastBlow,
                motion: motion(),
                pace: feedPace,
                nerve: battle && battle.nerve ? battle.nerve : '',
                weather: battle && battle.weather ? battle.weather.kind : '',
                sigil: battle && battle.sigil ? battle.sigil.kind : '',
                watching,
                allies: allyGroups(),
                compact: compactAllies(),
                backlog: sched.backlog(Date.now()),
                fx: sched.activeFx(Date.now()).map(f => f.kind + (f.id ? ':' + f.id : '')),
                digits: floaters.filter(f => f.text).map(f => f.text),
                icons: floaters.filter(f => f.icon).map(f => f.icon),
                shaking: shakeUntil > Date.now(),
                legend: legendOf(Array.from(units.values()).filter(u => u.cell)),
                badge: !!badge && badge.classList.contains('show'),
                units: Array.from(units.values()).map(u => ({
                    id: u.id, side: u.side, ally: u.ally, label: u.label, sprite: u.sprite, promoted: u.promoted || "", skin: u.skin || '', hair: u.hair || '', cell: u.cell, frac: u.frac, band: u.band,
                    role: u.role, leader: u.leader, fallen: u.fallen, yielded: u.yielded, unseen: !!u.unseen,
                    statuses: Array.from(u.statuses), casting: u.casting, flashing: u.flash > Date.now(),
                    pose: u.cell ? poseOf(u, slotOf(u), Date.now()) : null,
                    at: u.cell ? slotOf(u) : null,
                })),
            };
        },
    };

})();
