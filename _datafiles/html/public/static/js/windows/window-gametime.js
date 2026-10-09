/**
 * window-gametime.js
 *
 * Virtual window: Time and Date.
 * Displays the current in-game time and date, with an animated sky showing
 * suns or moons moving across the sky based on the time of day.
 *
 * Responds to GMCP namespaces:
 *   Gametime  - full gametime update
 *
 * Reads: Client.GMCPStructs.Gametime (including its optional weather
 * object, drawn as cloud, rain, storm or fog over the sky)
 */

'use strict';

(function() {

    injectStyles(`
        #gametime-panel {
            height: 100%;
            display: flex;
            flex-direction: column;
            overflow: hidden;
            background: var(--t-bg);
            user-select: none;
        }

        #gametime-sky {
            position: relative;
            flex: 1;
            overflow: hidden;
            min-height: 0;
        }

        #gametime-sky canvas {
            display: block;
            width: 100%;
            height: 100%;
        }

        #gametime-weather {
            position: absolute;
            top: 4px;
            left: 6px;
            padding: 1px 7px;
            border-radius: 9px;
            background: rgba(0, 0, 0, 0.45);
            color: #e8eef5;
            font-family: monospace;
            font-size: 0.72em;
            letter-spacing: 0.03em;
            pointer-events: auto;
            display: none;
        }

        #gametime-labels {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 2px 8px;
            background: var(--t-gametime-label-bg);
            flex-shrink: 0;
        }

        #gametime-time {
            font-family: monospace;
            font-size: 0.85em;
            color: var(--t-gametime-time);
            letter-spacing: 0.04em;
        }

        #gametime-date {
            font-family: monospace;
            font-size: 0.72em;
            color: var(--t-gametime-date);
        }

        #gametime-tooltip {
            position: fixed;
            z-index: 99999;
            pointer-events: none;
            background: var(--t-bg-surface);
            border: 1px solid var(--t-accent-dim);
            border-radius: 6px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.7);
            padding: 8px 10px;
            min-width: 140px;
            display: none;
            font-family: monospace;
            font-size: 0.78em;
            color: var(--t-text);
            white-space: nowrap;
        }
    `);

    // -----------------------------------------------------------------------
    // Celestial body appearance configuration
    // -----------------------------------------------------------------------

    // Size variation: base radius is computed from canvas height, then scaled
    // by a per-body multiplier drawn from this range.
    const BODY_SIZE_MIN = 0.75;
    const BODY_SIZE_MAX = 1.40;

    // Tint palettes defined as hex colour strings for easy editing.
    // Converted to [r, g, b] arrays once at load time by hexTintsToRgb().
    const SUN_TINT_HEX = [
        '#fff5a0',   
        '#ffdc64',   
        '#ffc850',   
        '#fff0c8',   
        '#ffb43c',   
        '#f0ffb4', 
        '#f56816', 
    ];

    const MOON_TINT_HEX = [
        '#d8dde8',   
        '#c8d7c8',   
        '#e6d2c8',   
        '#b4beDC',   
        '#dcc8e6',   
        '#d2e6d2',   
        '#ffdbcb',   
    ];

    // Convert a hex string (#rrggbb) to an [r, g, b] array.
    function hexToRgb(hex) {
        const v = parseInt(hex.replace('#', ''), 16);
        return [(v >>> 16) & 0xff, (v >>> 8) & 0xff, v & 0xff];
    }

    // Convert an array of hex strings to an array of [r, g, b] arrays.
    function hexTintsToRgb(hexArr) {
        return hexArr.map(hexToRgb);
    }

    const SUN_TINTS  = hexTintsToRgb(SUN_TINT_HEX);
    const MOON_TINTS = hexTintsToRgb(MOON_TINT_HEX);

    // -----------------------------------------------------------------------
    // Deterministic seeded PRNG (mulberry32)
    // Returns a function that produces floats in [0, 1).
    // -----------------------------------------------------------------------
    function seededRNG(seed) {
        let s = seed >>> 0;
        return function() {
            s += 0x6D2B79F5;
            let t = s;
            t = Math.imul(t ^ (t >>> 15), t | 1);
            t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
            return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
        };
    }

    // Hash a string to a uint32.
    function hashString(str) {
        let h = 0x811C9DC5;
        for (let i = 0; i < str.length; i++) {
            h ^= str.charCodeAt(i);
            h = (Math.imul(h, 0x01000193)) >>> 0;
        }
        return h;
    }

    // Build per-calendar appearance data for `count` bodies using the given
    // tint palette. Returns an array of { scale, tint, rawOffset } objects.
    // `rawOffset` is a seeded-random value in [-0.5, +0.5] used as a placement
    // hint; the final pixel positions are resolved in spreadPositions() where
    // the minimum-gap constraint can be applied.
    function buildBodyAppearances(calendar, count, tintPalette) {
        const seed = hashString(calendar + (tintPalette === SUN_TINTS ? ':sun' : ':moon'));
        const rng  = seededRNG(seed);
        const appearances = [];
        for (let i = 0; i < count; i++) {
            // The first body always uses BODY_SIZE_MAX so there is always one
            // prominent body; additional bodies use seeded random sizes.
            const scale     = i === 0 ? BODY_SIZE_MAX : BODY_SIZE_MIN + rng() * (BODY_SIZE_MAX - BODY_SIZE_MIN);
            const tint      = tintPalette[Math.floor(rng() * tintPalette.length)];
            const rawOffset = i === 0 ? 0 : rng() - 0.5;
            appearances.push({ scale, tint, rawOffset });
        }
        return appearances;
    }

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const panel = document.createElement('div');
        panel.id = 'gametime-panel';

        const sky = document.createElement('div');
        sky.id = 'gametime-sky';

        const canvas = document.createElement('canvas');
        canvas.id = 'gametime-canvas';
        sky.appendChild(canvas);

        const wx = document.createElement('div');
        wx.id = 'gametime-weather';
        sky.appendChild(wx);

        const labels = document.createElement('div');
        labels.id = 'gametime-labels';
        labels.innerHTML =
            '<span id="gametime-time">--:-- --</span>' +
            '<span id="gametime-date">Day 1, Year 1</span>';

        panel.appendChild(sky);
        panel.appendChild(labels);

        document.body.appendChild(panel);
        return panel;
    }

    // -----------------------------------------------------------------------
    // Tooltip
    // -----------------------------------------------------------------------
    const tooltip = Client.tooltip('gametime-tooltip');
    let tooltipAnchor = null;

    function showTooltip(anchorEl) {
        const data = Client.GMCPStructs.Gametime;
        if (!data) { return; }

        const hour24     = data.hour24     || 0;
        const minute     = data.minute     || 0;
        const dayStart   = data.day_start  || 6;
        const nightStart = data.night_start || 22;
        const night      = data.night;

        // Calculate hours remaining until the next transition.
        // All arithmetic is in fractional hours.
        const timeNow = hour24 + minute / 60;
        let hoursLeft;
        if (night) {
            // Time until dayStart (may wrap midnight)
            hoursLeft = dayStart - timeNow;
            if (hoursLeft <= 0) { hoursLeft += 24; }
        } else {
            // Time until nightStart
            hoursLeft = nightStart - timeNow;
            if (hoursLeft <= 0) { hoursLeft += 24; }
        }

        // Round to whole minutes first so 1.999h reads "2h 0m", not "1h 60m".
        const totalMinutes = Math.round(hoursLeft * 60);
        const h   = Math.floor(totalMinutes / 60);
        const m   = totalMinutes % 60;
        const hStr = h > 0 ? h + 'h ' : '';
        const mStr = m + 'm';
        const label = night
            ? 'Sunrise in ' + hStr + mStr
            : 'Sunset in '  + hStr + mStr;

        tooltip.show(label, anchorEl, { text: true, topOffset: 4 });
    }

    function hideTooltip() {
        tooltip.hide(0);
        tooltipAnchor = null;
    }

    function attachTooltip(el) {
        el.addEventListener('mouseenter', function() {
            tooltipAnchor = el;
            showTooltip(el);
        });
        el.addEventListener('mouseleave', hideTooltip);
    }

    // -----------------------------------------------------------------------
    // Debug: fast day/night cycle preview.
    // Set window.gametimeDebugCycle = true in the browser console to enable.
    // One full 24-hour day completes every 10 real seconds.
    // Uses seeded appearance data from the live GMCP payload; ignores game time.
    // -----------------------------------------------------------------------
    let _debugRafId       = null;
    let _debugStartTime   = null;
    let _debugWasRunning  = false;

    function _debugTick(now) {
        if (!window.gametimeDebugCycle) {
            _debugRafId      = null;
            _debugStartTime  = null;
            _debugWasRunning = false;
            update();
            return;
        }

        if (_debugStartTime === null) { _debugStartTime = now; }
        const elapsed   = (now - _debugStartTime) / 1000; // seconds
        const dayFrac   = (elapsed / 10) % 1;             // 0..1 over 10s
        const hour24    = dayFrac * 24;
        const minute    = (hour24 % 1) * 60;

        const base = Client.GMCPStructs.Gametime;
        if (base && win.isOpen()) {
            const dayStart   = base.day_start   || 6;
            const nightStart = base.night_start || 22;
            const isNight    = hour24 >= nightStart || hour24 < dayStart;

            const fakeData = Object.assign({}, base, {
                hour24:  Math.floor(hour24),
                minute:  Math.floor(minute),
                hour:    Math.floor(hour24) % 12 || 12,
                ampm:    hour24 < 12 ? 'AM' : 'PM',
                night:   isNight,
            });

            const timeEl = document.getElementById('gametime-time');
            const dateEl = document.getElementById('gametime-date');
            if (timeEl) {
                const min = String(fakeData.minute).padStart(2, '0');
                timeEl.textContent = fakeData.hour + ':' + min + ' ' + fakeData.ampm;
            }
            if (dateEl) {
                const monthName = fakeData.month_name || ('Month ' + fakeData.month);
                const zodiac    = fakeData.zodiac     || '';
                dateEl.textContent = 'Day ' + fakeData.day + ' of ' + monthName +
                    ', year ' + fakeData.year + ' (the ' + zodiac + ').';
            }

            drawSky(fakeData, true);
        }

        _debugRafId = requestAnimationFrame(_debugTick);
    }

    function _debugMaybeStart() {
        if (window.gametimeDebugCycle && !_debugRafId) {
            _debugStartTime = null;
            _debugRafId = requestAnimationFrame(_debugTick);
        }
    }

    // Poll for the flag being toggled on from the console.
    setInterval(_debugMaybeStart, 500);

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Time & Date', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  100,
        factory() {
            const el = createDOM();
            attachTooltip(el);
            return {
                title:      'Time & Date',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     100,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Sky rendering
    // -----------------------------------------------------------------------

    // Returns a value in [0, 1] representing how far through the current
    // period (day or night) the time is.
    //   0 = just started (risen from left horizon)
    //   0.5 = midpoint (highest point, centered horizontally)
    //   1 = just ended (set at right horizon)
    function celestialPosition(hour24, minute, dayStart, nightStart) {
        const timeNow = hour24 + minute / 60;

        const dayDuration = nightStart > dayStart
            ? nightStart - dayStart
            : (24 - dayStart) + nightStart;
        const nightDuration = 24 - dayDuration;

        const isDay = !(timeNow >= nightStart || timeNow < dayStart);

        if (isDay) {
            let elapsed = timeNow - dayStart;
            if (elapsed < 0) { elapsed += 24; }
            return elapsed / dayDuration;
        } else {
            let elapsed = timeNow - nightStart;
            if (elapsed < 0) { elapsed += 24; }
            return elapsed / nightDuration;
        }
    }

    // Given a rawPos in [0, 1], return the pixel Y coordinate on the parabolic
    // arc. Peaks (highest point) at rawPos=0.5.
    function rawPosToY(rawPos, yBottom, yTop) {
        return yBottom - (yBottom - yTop) * 4 * rawPos * (1 - rawPos);
    }

    // Spread bodies as pixel offsets from the group centre.
    // Returns an array of pixel offsets, one per body, relative to the group
    // centre X. The group centre follows the rawPos arc; each body is drawn at
    // groupCenterX + pixelOffset[i].
    function spreadPixelOffsets(appearances, baseRadius) {
        if (appearances.length <= 1) {
            return [0];
        }

        const maxR   = Math.max.apply(null, appearances.map(function(a) { return baseRadius * a.scale; }));
        const spread = maxR * 8;

        const offsets = appearances.map(function(a) {
            const x         = a.rawOffset;           // [-0.5, +0.5]
            const sign      = x < 0 ? -1 : 1;
            const stretched = sign * Math.sqrt(Math.abs(x) * 2); // [-1, +1]
            return stretched * spread;
        });

        // Enforce minimum separation in pixels.
        const radii = appearances.map(function(a) { return baseRadius * a.scale; });
        const MAX_PASSES = 20;
        for (let pass = 0; pass < MAX_PASSES; pass++) {
            let moved = false;
            for (let i = 0; i < offsets.length - 1; i++) {
                for (let j = i + 1; j < offsets.length; j++) {
                    const minDist = (radii[i] + radii[j]) * 0.5;
                    const delta   = offsets[j] - offsets[i];
                    const dist    = Math.abs(delta);
                    if (dist < minDist) {
                        const push = (minDist - dist) / 2;
                        const dir  = delta >= 0 ? 1 : -1;
                        offsets[i] -= push * dir;
                        offsets[j] += push * dir;
                        moved = true;
                    }
                }
            }
            if (!moved) { break; }
        }

        return offsets;
    }

    // Compute the bounding radius of a group: the furthest any body's outer edge
    // (including glow for suns) extends from the group centre.
    function groupBoundingRadius(appearances, baseRadius, isSun) {
        let maxEdge = 0;
        const offsets = spreadPixelOffsets(appearances, baseRadius);
        for (let i = 0; i < appearances.length; i++) {
            const r    = baseRadius * appearances[i].scale;
            const edge = Math.abs(offsets[i]) + (isSun ? r * 2.2 : r);
            if (edge > maxEdge) { maxEdge = edge; }
        }
        return maxEdge;
    }

    // Convert [r,g,b] to a CSS rgba string.
    function rgbToCss(rgb, alpha) {
        return 'rgba(' + rgb[0] + ',' + rgb[1] + ',' + rgb[2] + ',' + alpha + ')';
    }


    // -----------------------------------------------------------------------
    // Weather layer
    //
    // Drawn over the sky from the room's weather in the Gametime payload
    // (data.weather: name, cloud_cover 0-3, visibility 0..-2). Cloud, rain,
    // storm and fog are all animated from the wall clock, so they keep
    // moving between GMCP ticks.
    // -----------------------------------------------------------------------
    const WEATHER_LABELS = {
        'clear': 'Clear',
        'overcast': 'Overcast',
        'rain': 'Rain',
        'storm': 'Storm',
        'fog': 'Fog',
        'thick-fog': 'Thick fog',
        'mist': 'Mist',
    };

    function weatherLabel(name) {
        if (WEATHER_LABELS[name]) { return WEATHER_LABELS[name]; }
        const s = String(name || '').replace(/-/g, ' ');
        return s.charAt(0).toUpperCase() + s.slice(1);
    }

    function setWeatherLabel(wx) {
        const el = document.getElementById('gametime-weather');
        if (!el) { return; }
        if (!wx) {
            el.style.display = 'none';
            return;
        }
        el.style.display = 'block';
        el.textContent = weatherLabel(wx.name) + (wx.indoor ? ' (outside)' : '');
        el.title = wx.description || '';
    }

    // Soft cloud: a few overlapping translucent ellipses.
    function drawCloud(ctx, x, y, size, rgb, alpha) {
        ctx.fillStyle = 'rgba(' + rgb + ',' + alpha + ')';
        const puffs = [[0, 0, 1, 0.55], [-0.7, 0.15, 0.7, 0.4], [0.75, 0.12, 0.8, 0.45], [0.2, -0.25, 0.7, 0.4]];
        puffs.forEach(function(p) {
            ctx.beginPath();
            ctx.ellipse(x + p[0] * size, y + p[1] * size, p[2] * size, p[3] * size, 0, 0, Math.PI * 2);
            ctx.fill();
        });
    }

    function drawWeather(ctx, w, h, wx, night) {
        if (!wx) { return; }
        const t     = performance.now() / 1000;
        const cover = wx.cloud_cover || 0;
        const name  = wx.name || '';
        const rainy = name === 'rain' || name === 'storm';
        const foggy = (wx.visibility || 0) < 0 || name === 'mist';
        const cloudRgb = rainy ? (night ? '40,46,58' : '96,106,120')
                               : (night ? '58,66,82' : '226,232,240');

        // Overcast greys the whole sky.
        if (cover > 0) {
            ctx.fillStyle = 'rgba(' + (night ? '10,14,24' : '120,130,142') + ',' + (0.16 * cover + (rainy ? 0.12 : 0)) + ')';
            ctx.fillRect(0, 0, w, h);
        }

        // Drifting clouds: more, bigger and darker with cover.
        if (cover > 0) {
            const count = 2 + cover * 2;
            const rng = seededRNG(0xC10D5 + cover);
            for (let i = 0; i < count; i++) {
                const speed = 4 + rng() * 6;
                const size  = h * (0.28 + rng() * 0.22) * (0.8 + cover * 0.2);
                const span  = w + size * 4;
                const x     = ((rng() * span + t * speed) % span) - size * 2;
                const y     = h * (0.12 + rng() * 0.38);
                drawCloud(ctx, x, y, size, cloudRgb, 0.28 + cover * 0.14);
            }
        }

        // Fog and mist: slow pale bands low in the sky.
        if (foggy) {
            const dense = (wx.visibility || 0) <= -2 ? 0.55 : (wx.visibility < 0 ? 0.38 : 0.22);
            const fogRgb = night ? '120,130,150' : '235,240,245';
            const g = ctx.createLinearGradient(0, 0, 0, h);
            g.addColorStop(0, 'rgba(' + fogRgb + ',' + (dense * 0.35) + ')');
            g.addColorStop(1, 'rgba(' + fogRgb + ',' + dense + ')');
            ctx.fillStyle = g;
            ctx.fillRect(0, 0, w, h);
            for (let i = 0; i < 3; i++) {
                const bx = ((t * (5 + i * 3) + i * w * 0.4) % (w * 1.6)) - w * 0.3;
                ctx.fillStyle = 'rgba(' + fogRgb + ',' + (dense * 0.6) + ')';
                ctx.beginPath();
                ctx.ellipse(bx, h * (0.55 + i * 0.15), w * 0.35, h * 0.14, 0, 0, Math.PI * 2);
                ctx.fill();
            }
        }

        // Rain and storm: slanted streaks falling past the clouds.
        if (rainy) {
            const storm  = name === 'storm';
            const drops  = Math.round(Math.min(90, w / 4) * (storm ? 1.6 : 1));
            const slant  = storm ? 0.45 : 0.18;
            const len    = h * (storm ? 0.3 : 0.22);
            const fall   = storm ? 1.8 : 1.1;
            const rng    = seededRNG(0x4A17);
            ctx.strokeStyle = night ? 'rgba(170,190,220,0.55)' : 'rgba(205,220,240,0.7)';
            ctx.lineWidth   = 1;
            ctx.beginPath();
            for (let i = 0; i < drops; i++) {
                const fx = rng(), fy = rng(), sp = 0.7 + rng() * 0.6;
                const y  = (((fy + t * fall * sp) % 1) * (h + len)) - len;
                const x  = ((fx * (w + h) - y * slant) % (w + h) + (w + h)) % (w + h) - h * 0.5;
                ctx.moveTo(x, y);
                ctx.lineTo(x - len * slant, y + len);
            }
            ctx.stroke();

            // Storm: an occasional flash lights the sky.
            if (storm) {
                const cycle = (t % 7);
                if (cycle < 0.25) {
                    ctx.fillStyle = 'rgba(255,255,240,' + (0.35 * (1 - cycle / 0.25)) + ')';
                    ctx.fillRect(0, 0, w, h);
                }
            }
        }
    }

    function drawSky(data, debug) {
        const canvas = document.getElementById('gametime-canvas');
        if (!canvas) { return; }

        const parent = canvas.parentElement;
        const w = parent.clientWidth  || 300;
        const h = parent.clientHeight || 60;

        if (canvas.width !== w || canvas.height !== h) {
            canvas.width  = w;
            canvas.height = h;
        }

        const ctx        = canvas.getContext('2d', { willReadFrequently: true });
        const night      = data.night;
        const hour24     = data.hour24;
        const minute     = data.minute;
        const dayStart   = data.day_start   || 6;
        const nightStart = data.night_start || 22;
        const calendar   = data.calendar    || 'default';
        const sunCount   = Math.max(1, data.sun_count  || 1);
        const moonCount  = Math.max(0, data.moon_count || 1);

        // --- background ---
        // During the last 2 game-hours of day, blend from day sky toward dusk/night.
        const dayDuration   = (nightStart > dayStart)
            ? nightStart - dayStart
            : (24 - dayStart) + nightStart;
        const duskHours     = Math.min(2, dayDuration * 0.25); // dusk window, max 2h
        const duskRawStart  = (dayDuration - duskHours) / dayDuration; // rawPos where dusk begins

        // duskT: 0 = full day, 1 = full dusk (at nightStart)
        let duskT = 0;
        if (!night) {
            const rawPos0 = celestialPosition(hour24, minute, dayStart, nightStart);
            duskT = Math.max(0, Math.min(1, (rawPos0 - duskRawStart) / (1 - duskRawStart)));
        }

        // Interpolate between day colours and dusk/night colours.
        function lerpHex(a, b, t) {
            const ar = (a >> 16) & 0xff, ag = (a >> 8) & 0xff, ab = a & 0xff;
            const br = (b >> 16) & 0xff, bg = (b >> 8) & 0xff, bb = b & 0xff;
            const r = Math.round(ar + (br - ar) * t);
            const g = Math.round(ag + (bg - ag) * t);
            const bl2 = Math.round(ab + (bb - ab) * t);
            return 'rgb(' + r + ',' + g + ',' + bl2 + ')';
        }

        const skyTop    = lerpHex(0x1a6fbf, 0x020510, duskT);
        // The dark-sky colour that creeps down from the top during dusk.
        const skyMid    = lerpHex(0x1a6fbf, 0x020510, duskT);
        // The warm horizon glow: starts blended into the full gradient and
        // compresses toward the bottom as duskT increases.
        // glowStop is the gradient position where the dark sky ends and the
        // warm glow begins: 0 at duskT=0 (glow fills everything), 1 at duskT=1
        // (glow is just a sliver at the very bottom).
        const glowStop  = duskT;
        const glowColor = lerpHex(0x87ceeb, 0xd4703a, duskT);

        const grad = ctx.createLinearGradient(0, 0, 0, h);
        if (night) {
            grad.addColorStop(0, '#020510');
            grad.addColorStop(1, '#0a0e20');
        } else if (duskT <= 0) {
            grad.addColorStop(0, '#1a6fbf');
            grad.addColorStop(1, '#87ceeb');
        } else {
            grad.addColorStop(0, skyTop);
            grad.addColorStop(Math.min(glowStop, 0.999), skyMid);
            grad.addColorStop(1, glowColor);
        }
        ctx.fillStyle = grad;
        ctx.fillRect(0, 0, w, h);

        // --- star field (night only) ---
        if (night) {
            ctx.fillStyle = 'rgba(255,255,255,0.7)';
            const starData = [
                [0.05,0.15],[0.12,0.55],[0.20,0.10],[0.28,0.70],[0.35,0.25],
                [0.42,0.80],[0.50,0.05],[0.57,0.60],[0.65,0.30],[0.73,0.85],
                [0.80,0.18],[0.88,0.65],[0.95,0.40],[0.08,0.90],[0.60,0.50],
                [0.33,0.45],[0.75,0.10],[0.18,0.38],[0.90,0.75],[0.48,0.92],
            ];
            starData.forEach(function(s) {
                // Shift the star left by _starOffset and wrap around using modulo
                // so stars that drift off the left edge reappear on the right.
                const sx = ((s[0] * w - _starOffset) % w + w) % w;
                const sy = s[1] * h * 0.85;
                ctx.beginPath();
                ctx.arc(sx, sy, 0.8, 0, Math.PI * 2);
                ctx.fill();
            });
        }

        // --- celestial bodies ---
        // The group of bodies moves as a unit. The group centre follows a single
        // rawPos arc; each body is offset from that centre in pixels. The pad is
        // the group's bounding radius so the entire group is off-screen at
        // rawPos=0 (left) and rawPos=1 (right).
        const baseRadius  = Math.max(8, Math.min(14, h * 0.28));
        const rawPos      = celestialPosition(hour24, minute, dayStart, nightStart);

        const isSun       = !night;
        const count       = night ? moonCount : sunCount;
        const tintPalette = night ? MOON_TINTS : SUN_TINTS;
        const appearances = buildBodyAppearances(calendar, count, tintPalette);
        const pixOffsets  = spreadPixelOffsets(appearances, baseRadius);
        const pad         = Math.ceil(groupBoundingRadius(appearances, baseRadius, isSun));
        const wWide       = w + 2 * pad;

        const yTop     = h * 0.18;
        // Compute yBottom so the arc touches y=h exactly at the visible corners.
        // The visible corners correspond to rawPos = pad/wWide (left) and
        // (pad+w)/wWide (right) on the wide canvas. Solving
        //   rawPosToY(pad/wWide, yBottom, yTop) = h  for yBottom:
        const cornerRawPos  = pad / wWide;
        const cornerFactor  = 4 * cornerRawPos * (1 - cornerRawPos);
        const yBottom       = cornerFactor > 0
            ? (h - yTop * cornerFactor) / (1 - cornerFactor)
            : h;

        // Group centre in wide-canvas coordinates.
        // rawPos=0 -> centre at 0 (left edge of wide canvas, pad pixels left of visible)
        // rawPos=0.5 -> centre at wWide/2 (visible centre)
        // rawPos=1 -> centre at wWide (right edge of wide canvas, pad pixels right of visible)
        function groupCentreX(rp) {
            return rp * wWide;
        }

        const offscreen = document.createElement('canvas');
        offscreen.width  = wWide;
        offscreen.height = h;
        const octx = offscreen.getContext('2d');

        const cx = groupCentreX(rawPos);

        for (let i = 0; i < count; i++) {
            const app    = appearances[i];
            const r      = baseRadius * app.scale;
            const bodyX  = cx + pixOffsets[i];
            // Each body follows the arc at its own X position. Invert groupCentreX
            // to get the effective rawPos for this body's wide-canvas X, then use
            // that for Y so each body sits on the arc.
            const bodyRawPos   = bodyX / wWide;
            const bodyY  = rawPosToY(bodyRawPos, yBottom, yTop);
            const tint   = app.tint;

            if (night) {
                octx.save();
                octx.beginPath();
                octx.arc(bodyX, bodyY, r, 0, Math.PI * 2);
                octx.fillStyle   = rgbToCss(tint, 1.0);
                octx.shadowColor = rgbToCss(tint, 0.7);
                octx.shadowBlur  = 10;
                octx.fill();
                octx.restore();
            } else {
                // Soft glow
                const glow = octx.createRadialGradient(bodyX, bodyY, r * 0.5, bodyX, bodyY, r * 2.2);
                glow.addColorStop(0, rgbToCss(tint, 0.35));
                glow.addColorStop(1, rgbToCss(tint, 0.0));
                octx.beginPath();
                octx.arc(bodyX, bodyY, r * 2.2, 0, Math.PI * 2);
                octx.fillStyle = glow;
                octx.fill();

                // Disc
                octx.save();
                octx.beginPath();
                octx.arc(bodyX, bodyY, r, 0, Math.PI * 2);
                octx.fillStyle   = rgbToCss(tint, 1.0);
                octx.shadowColor = rgbToCss(tint, 1.0);
                octx.shadowBlur  = 12;
                octx.fill();
                octx.restore();
            }
        }

        // Blit the centre slice of the wide canvas onto the visible canvas.
        // Cloud dims the sun and, when overcast, hides the moon (as the
        // server's room text does).
        const wx    = data.weather || null;
        const cover = wx ? (wx.cloud_cover || 0) : 0;
        ctx.save();
        ctx.globalAlpha = night ? (cover >= 3 ? 0 : 1 - 0.3 * cover) : 1 - 0.22 * cover;
        ctx.drawImage(offscreen, pad, 0, w, h, 0, 0, w, h);
        ctx.restore();

        drawWeather(ctx, w, h, wx, night);

        // --- debug overlays ---
        if (debug) {
            // Arc path: trace the group centre across rawPos 0..1, mapped to
            // visible canvas coordinates (subtract pad).
            ctx.save();
            ctx.strokeStyle = 'rgba(255,255,0,0.7)';
            ctx.lineWidth   = 1.5;
            ctx.setLineDash([4, 3]);
            ctx.beginPath();
            const steps = 120;
            for (let s = 0; s <= steps; s++) {
                const t  = s / steps;
                const ax = groupCentreX(t) - pad;
                const ay = rawPosToY(t, yBottom, yTop);
                if (s === 0) { ctx.moveTo(ax, ay); } else { ctx.lineTo(ax, ay); }
            }
            ctx.stroke();

            // Center vertical line at w/2.
            ctx.strokeStyle = 'rgba(255,80,80,0.8)';
            ctx.lineWidth   = 1;
            ctx.setLineDash([3, 3]);
            ctx.beginPath();
            ctx.moveTo(w / 2, 0);
            ctx.lineTo(w / 2, h);
            ctx.stroke();
            ctx.restore();
        }
    }

    // -----------------------------------------------------------------------
    // Star field scroll state.
    // Incremented by 1 pixel each time drawSky is called during night.
    // -----------------------------------------------------------------------
    let _starOffset = 0;

    // -----------------------------------------------------------------------
    // Continuous animation: interpolates game time between GMCP updates.
    // -----------------------------------------------------------------------

    // State updated on each GMCP packet.
    let _animData        = null;   // last received data object
    let _animRealMs      = null;   // real timestamp (ms) when last packet arrived
    let _animGameMinutes = null;   // game time in fractional minutes at last packet
    let _animRateMinPerMs = null;  // game minutes per real ms, learned from two packets
    let _animPrevRealMs  = null;   // real timestamp of the packet before last
    let _animPrevGameMin = null;   // game minutes of the packet before last
    let _animRafId       = null;

    // Convert a data object's hour24+minute into a single fractional-minute value
    // that increases monotonically within a day (0..1440).
    function _gameMinutes(data) {
        return (data.hour24 || 0) * 60 + (data.minute || 0);
    }

    // Build a synthetic data object with overridden hour24/minute/hour/ampm/night.
    function _syntheticData(base, fracMinutes) {
        const dayStart   = base.day_start   || 6;
        const nightStart = base.night_start || 22;
        const totalMins  = ((fracMinutes % 1440) + 1440) % 1440;
        const hour24     = Math.floor(totalMins / 60);
        const minute     = Math.floor(totalMins % 60);
        const isNight    = hour24 >= nightStart || hour24 < dayStart;
        return Object.assign({}, base, {
            hour24,
            minute,
            hour:  hour24 % 12 || 12,
            ampm:  hour24 < 12 ? 'AM' : 'PM',
            night: isNight,
        });
    }

    function _animTick(now) {
        if (window.gametimeDebugCycle) {
            _animRafId = null;
            return;
        }
        if (!_animData || !win.isOpen()) {
            _animRafId = requestAnimationFrame(_animTick);
            return;
        }

        // Extrapolate game time from last known position.
        let currentGameMin = _animGameMinutes;
        if (_animRateMinPerMs !== null && _animRealMs !== null) {
            const elapsed = now - _animRealMs;
            currentGameMin = _animGameMinutes + elapsed * _animRateMinPerMs;
        }

        const synth = _syntheticData(_animData, currentGameMin);
        drawSky(synth);

        if (tooltipAnchor && tooltip.isShown()) {
            showTooltip(tooltipAnchor);
        }

        _animRafId = requestAnimationFrame(_animTick);
    }

    function _animStart() {
        if (!_animRafId) {
            _animRafId = requestAnimationFrame(_animTick);
        }
    }

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function update() {
        // Suppress normal updates while the debug cycle is running.
        if (window.gametimeDebugCycle) { return; }
        const data = Client.GMCPStructs.Gametime;
        if (!data) { return; }

        win.open();
        if (!win.isOpen()) { return; }

        const nowMs      = performance.now();
        const gameMins   = _gameMinutes(data);

        // Learn the rate from consecutive packets.
        if (_animRealMs !== null && _animGameMinutes !== null) {
            const realDelta = nowMs - _animRealMs;
            let   gameDelta = gameMins - _animGameMinutes;
            // Handle midnight wrap (game minutes reset from ~1439 back to 0).
            if (gameDelta < -720) { gameDelta += 1440; }
            if (realDelta > 0 && gameDelta > 0) {
                _animRateMinPerMs = gameDelta / realDelta;
            }
        }

        _animPrevRealMs  = _animRealMs;
        _animPrevGameMin = _animGameMinutes;
        _animRealMs      = nowMs;
        _animGameMinutes = gameMins;
        _animData        = data;

        // Advance star scroll offset by one pixel on each nighttime tick.
        if (data.night) {
            const canvas = document.getElementById('gametime-canvas');
            const w = canvas ? (canvas.parentElement.clientWidth || 300) : 300;
            _starOffset = (_starOffset + 2) % w;
        }

        setWeatherLabel(data.weather || null);

        // Update labels on real GMCP ticks only.
        const timeEl = document.getElementById('gametime-time');
        const dateEl = document.getElementById('gametime-date');
        if (timeEl) {
            const min = String(data.minute).padStart(2, '0');
            timeEl.textContent = data.hour + ':' + min + ' ' + data.ampm;
        }
        if (dateEl) {
            const monthName = data.month_name || ('Month ' + data.month);
            const zodiac    = data.zodiac     || '';
            dateEl.textContent = 'Day ' + data.day + ' of ' + monthName +
                ', year ' + data.year + ' (the ' + zodiac + ').';
        }

        _animStart();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Gametime'],
        onGMCP() { update(); },
    });

})();
