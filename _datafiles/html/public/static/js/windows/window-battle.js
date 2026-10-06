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

    const W = 320;
    const H = 180;
    const SETTING_KEY = 'ashveil-battle-screen';
    const OUTCOME_MS = 3000;
    const FOCI = ['none', 'leader', 'casters', 'nearest', 'weakest', 'strongest', 'wounded'];

    // Health words from the server, as the fraction of the bar they fill.
    const BANDS = { 'unhurt': 1, 'scratched': 0.8, 'wounded': 0.6, 'badly wounded': 0.4, 'near death': 0.2 };

    // Role letters drawn on a member's badge.
    const ROLE_GLYPH = { fighter: 'F', healer: 'H', caster: 'C', guardian: 'G', controller: 'K' };

    // Class hues for company figures: body, trim.
    const CLASS_HUES = {
        warrior: ['#8a8f99', '#c0504d'], cleric: ['#e8e4d0', '#d4a72c'], ranger: ['#4e7d3a', '#8a6a3b'],
        rogue: ['#444a56', '#9b59b6'], wizard: ['#4a5fc1', '#e0c040'], witch: ['#6b3f8c', '#3fb08a'],
    };
    const DEFAULT_HUES = ['#7a6a55', '#c0a060'];

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
    let raf = 0;

    function setting() {
        try { return localStorage.getItem(SETTING_KEY) === 'manual' ? 'manual' : 'auto'; } catch (err) { return 'auto'; }
    }

    function saveSetting(v) {
        try { localStorage.setItem(SETTING_KEY, v); } catch (err) { /* unavailable: the choice lasts this page */ }
        memorySetting = v;
    }
    let memorySetting = null;
    function currentSetting() { return memorySetting || setting(); }

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

    function unseen() {
        const u = unit('?');
        u.side = 'enemy';
        u.label = 'something unseen';
        u.sprite = 'unknown-shape';
        u.cell = { row: 0, col: 1 };
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
            u.cell = cell;
            u.leader = m.key === 'leader';
            u.role = (m.strategy && m.strategy.role) || '';
            u.fallen = m.status === 'dead';
            u.frac = (v.hp !== null && v.hp !== undefined && v.hp_max > 0) ? Math.max(0, Math.min(1, v.hp / v.hp_max)) : (u.fallen ? 0 : 1);
            u.band = '';
            seen.add(m.key);
        });

        if (battle) {
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
            (battle.fallen || []).forEach(f => { if (units.has(f.id)) { unit(f.id).fallen = true; seen.add(f.id); } });
            (battle.surrendered || []).forEach(f => { if (units.has(f.id)) { const u = unit(f.id); u.yielded = true; seen.add(f.id); } });
            if (battle.dark) { unseen(); seen.add('?'); }
        }
        // Anyone no longer in the fight leaves the picture; the unseen
        // presence stays while an event still names one.
        units.forEach((u, id) => {
            if (!seen.has(id) && !(id === '?' && u.keep)) { units.delete(id); }
        });
    }

    // ---------------------------------------------------------------------
    // Events (Company.Battle.Event)
    // ---------------------------------------------------------------------

    function flash(u, color, text) {
        if (!u) { return; }
        u.flash = Date.now() + 260;
        u.flashColor = color;
        if (text) {
            const p = u.cell ? slot(u.side, u.cell.row, u.cell.col) : { x: W / 2, y: 120 };
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

    function onEvents(body) {
        (body.events || []).forEach(applyEvent);
        draw();
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
            if (src) { src.casting = e.src === '?' ? 'a spell' : (e.spell || 'a spell'); wake(); }
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

    const HINT = 'Hover or tap a figure for its name, health, and whom it strikes.';

    const OUTCOMES = { victory: 'Victory', defeat: 'Defeat', 'broken-off': 'The company breaks off' };

    function beginOutcome(outcome) {
        if (!isShown()) { return; }
        outcomeText = OUTCOMES[outcome] || 'The battle is over';
        if (outcomeTimer) { clearTimeout(outcomeTimer); }
        outcomeTimer = setTimeout(() => { outcomeTimer = null; endBattleView(); }, OUTCOME_MS);
        paintChrome();
    }

    function clearOutcome() {
        if (outcomeTimer) { clearTimeout(outcomeTimer); outcomeTimer = null; }
        outcomeText = '';
    }

    function endBattleView() {
        outcomeText = '';
        units = new Map();
        floaters = [];
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

    let titleNode, bannerNode, captionNode, outcomeNode, retreatBtn, autoBox, focusBtns = [];

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

        captionNode = el('div', 'bs-caption');
        outcomeNode = el('div', 'bs-outcome');
        outcomeNode.setAttribute('role', 'status');
        overlay.appendChild(captionNode);
        overlay.appendChild(outcomeNode);

        const foot = el('div', 'bs-foot');
        retreatBtn = el('button', null, 'Retreat');
        retreatBtn.type = 'button';
        retreatBtn.title = 'Withdraw your company: one round to prepare, then the attempt (retreat)';
        retreatBtn.addEventListener('click', () => Client.SendInput('retreat'));
        foot.appendChild(retreatBtn);
        FOCI.forEach(rule => {
            const b = el('button', null, rule);
            b.type = 'button';
            b.dataset.focus = rule;
            b.title = 'Company focus for this battle only (company tactics focus ' + rule + ')';
            b.addEventListener('click', () => Client.SendInput('company tactics focus ' + rule));
            focusBtns.push(b);
            foot.appendChild(b);
        });
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
        help.addEventListener('click', () => Client.SendInput('help battlescreen'));
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
        if (room >= 1) {
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
        titleNode.textContent = battle ? 'Battle: ' + (battle.group || 'the enemy') : 'Battle';
        const banners = [];
        if (battle) {
            if (battle.dark) { banners.push('dark'); }
            if (battle.narrow) { banners.push('narrow'); }
            if (battle.retreat) { banners.push('withdrawing ' + battle.retreat.exit); }
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
        paintCaption();
    }

    function paintCaption() {
        if (!captionNode) { return; }
        const u = hover ? units.get(hover) : null;
        if (!u) { captionNode.textContent = battle && !outcomeText ? HINT : ''; return; }
        let text = u.label;
        if (u.side === 'enemy' && u.band) { text += ', ' + u.band; }
        if (u.side === 'company' && u.fallen) { text += ', fallen'; }
        if (u.yielded) { text += ', surrendered'; }
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
        let bestD = 18 * 18;
        units.forEach(u => {
            if (!u.cell) { return; }
            const p = slot(u.side, u.cell.row, u.cell.col);
            const d = (p.x - x) * (p.x - x) + (p.y - 12 - y) * (p.y - 12 - y);
            if (d < bestD) { bestD = d; best = u.id; }
        });
        hover = ev.type === 'click' && hover === best ? null : best;
        paintCaption();
        draw();
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
        let live = floaters.length > 0;
        units.forEach(u => { if (u.flash > now) { live = true; } });
        if (live) { wake(); }
    }

    function rect(x, y, w, h, c) {
        ctx.fillStyle = c;
        ctx.fillRect(Math.round(x), Math.round(y), w, h);
    }

    function scene() {
        return SCENES[biome] || SCENES.land;
    }

    function drawBackground() {
        const s = scene();
        const bg = art('battle/backgrounds/' + s.bg + '.png');
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
    function drawFigure(u, x, y, now) {
        const dir = u.side === 'company' ? 1 : -1;       // +1 faces right
        const dim = battle && battle.dark ? 0.5 : 1;
        const flashing = u.flash > now;
        // Shadow.
        ctx.fillStyle = 'rgba(0,0,0,0.3)';
        ctx.fillRect(Math.round(x - 7), y - 1, 14, 2);
        if (u.fallen) {
            // Lying down: a flat shape, the figure's body hue.
            const c = u.side === 'company' ? (CLASS_HUES[u.klass] || DEFAULT_HUES)[0] : '#6a4a4a';
            rect(x - 9, y - 5, 18, 4, shade(c, 0.6));
            rect(x - (9 * dir), y - 7, 4, 4, '#c8a888');
            return;
        }
        if (u.unseen) {
            rect(x - 6, y - 26, 12, 26, 'rgba(10,10,16,0.75)');
            rect(x - 4, y - 30, 8, 6, 'rgba(10,10,16,0.75)');
            return;
        }
        const sheet = art('battle/units/' + u.sprite + '/idle.png');
        if (sheet) {
            // Art: the idle loop, anchored bottom-centre, enemies mirrored.
            const fw = (sheet.info.frame || [64, 64])[0], fh = (sheet.info.frame || [64, 64])[1];
            const frames = sheet.info.frames || 1;
            const i = Math.floor(now / (sheet.info.frame_ms || 250)) % frames;
            ctx.save();
            ctx.translate(Math.round(x), y);
            if (dir < 0) { ctx.scale(-1, 1); }
            if (dim < 1) { ctx.filter = 'brightness(0.5)'; }
            ctx.drawImage(sheet.img, i * fw, 0, fw, fh, -fw / 2, -fh, fw, fh);
            ctx.restore();
        } else if (u.side === 'company') {
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
        if (flashing) {
            ctx.fillStyle = u.flashColor;
            ctx.globalAlpha = 0.45;
            ctx.fillRect(Math.round(x - 10), y - 32, 20, 33);
            ctx.globalAlpha = 1;
        }
    }

    // drawInfo draws the bar, role badge, statuses and chant mark.
    function drawInfo(u, x, y) {
        if (u.fallen || u.unseen) { return; }
        const w = 18;
        rect(x - w / 2, y + 3, w, 3, '#101014');
        if (u.side === 'company') {
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
            ctx.font = '7px monospace';
            ctx.fillStyle = '#e8e8f0';
            ctx.fillText(ROLE_GLYPH[u.role], Math.round(x - w / 2), y + 13);
        }
        if (u.yielded) {
            ctx.font = '7px monospace';
            ctx.fillStyle = '#e8e8f0';
            ctx.fillText('yields', Math.round(x - 10), y - 36);
        }
        let i = 0;
        u.statuses.forEach(s => {
            rect(x - 9 + i * 4, y - 36, 3, 3, 'hsl(' + hashHue(s) + ',70%,60%)');
            i++;
        });
        if (u.casting) { rect(x - 1, y - 40, 2, 2, '#9ab0ff'); rect(x - 3, y - 38, 6, 1, '#9ab0ff'); }
    }

    function draw() {
        if (!ctx || !isShown()) { return; }
        const now = Date.now();
        ctx.imageSmoothingEnabled = false;
        drawBackground();
        // Back to front by lane (then row), so nearer units overlap.
        const list = Array.from(units.values()).filter(u => u.cell);
        list.sort((a, b) => (a.cell.col - b.cell.col) || (b.cell.row - a.cell.row));
        list.forEach(u => {
            const p = slot(u.side, u.cell.row, u.cell.col);
            drawFigure(u, p.x, p.y, now);
            drawInfo(u, p.x, p.y);
        });
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
                const pa = slot(a.side, a.cell.row, a.cell.col), pb = slot(b.side, b.cell.row, b.cell.col);
                ctx.beginPath();
                ctx.moveTo(pa.x, pa.y - 12);
                ctx.lineTo(pb.x, pb.y - 12);
                ctx.stroke();
            });
        }
        floaters.forEach(f => {
            const age = (now - f.born) / 900;
            ctx.font = 'bold 9px monospace';
            ctx.fillStyle = f.color;
            ctx.globalAlpha = Math.max(0, 1 - age);
            ctx.fillText(f.text, Math.round(f.x - 4), Math.round(f.y - age * 14));
            ctx.globalAlpha = 1;
        });
        if (battle && battle.dark) {
            ctx.fillStyle = 'rgba(0,0,10,0.5)';
            ctx.fillRect(0, 0, W, H);
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
            minimised = false;
            userOpened = false;
        }
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
            if (!outcomeTimer) { beginOutcome(''); }
            if (!outcomeTimer) { endBattleView(); }
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
                if (info && typeof info.environment === 'string') { biome = info.environment; draw(); }
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
        slot,
        // state is what a test reads: the units and where they stand.
        state() {
            return {
                open: isShown(),
                minimised,
                outcome: outcomeText,
                biome,
                badge: !!badge && badge.classList.contains('show'),
                units: Array.from(units.values()).map(u => ({
                    id: u.id, side: u.side, label: u.label, sprite: u.sprite, cell: u.cell, frac: u.frac, band: u.band,
                    role: u.role, leader: u.leader, fallen: u.fallen, yielded: u.yielded, unseen: !!u.unseen,
                    statuses: Array.from(u.statuses), casting: u.casting,
                    at: u.cell ? slot(u.side, u.cell.row, u.cell.col) : null,
                })),
            };
        },
    };

})();
