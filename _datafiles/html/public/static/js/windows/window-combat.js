/**
 * window-combat.js
 *
 * Virtual window: Combat, the company dock's third tab (Phase 32g). Its
 * Setup view: how the company stands and how each member fights, set
 * before a battle (a battle plays out on its own; help strategy).
 *
 *   - The formation grid, row 1 the front, each member with their role.
 *   - A row per member: role and target rule. Clicking a member opens a
 *     menu that sends real commands: strategy <who> <role>,
 *     strategy <who> target <rule>, strategy <who> default, and
 *     formation move / swap / clear. <who> is "me" for the player and
 *     "#<id>" for a companion, which both commands accept.
 *   - Scout (sends "scout") when the room holds an enemy group or a
 *     hostile creature.
 *
 * Phase 32g2, the Battle view, shown instead while the player's battle
 * runs (Company.Battle), with Setup folded under it:
 *
 *   - The enemy group's formation above the company's, both front rows
 *     toward the middle (the dock is one narrow column, so the two sides
 *     face each other up and down). Each enemy shows its battle label and
 *     how hurt it looks (scout's words, never numbers), and a dot when
 *     the player can reach it from their place.
 *   - Target lines (SVG, no server strings) from each fighter to whom it
 *     strikes; someone outside the company an enemy strikes is a chip
 *     beside the company. Hovering, focusing, or tapping a fighter lights
 *     its target and everyone striking it.
 *   - The fallen and the groups waiting their turn, a plain "who strikes
 *     whom" list for screen readers and narrow screens, a polite live
 *     region (a new foe on the player, a fall), and Retreat (sends "retreat",
 *     the one way out of a battle; Phase 33c).
 *   - A company member's click opens Setup's menu. The Combat tab shows a
 *     marker while a battle runs and another tab is showing.
 *
 * Phase 62, battle lines that explain themselves:
 *
 *   - Under either view, "Last rounds" lists the latest fight's weapon
 *     rounds of the company's own (newest first, kept after the fight ends,
 *     12 at most), each a one-line heading that opens into the engine's own
 *     roll in plain lines (battle-rounds.js; Company.Battle.Event's
 *     `explain`): what the hit needed and rolled, the defence it met, armor
 *     and named modifiers. `why` says the same in the log.
 *
 * Phase 30c, company tactics:
 *
 *   - The Battle view's focus buttons (none and the seven focus rules), the
 *     current one pressed, all disabled while an order waits for the next
 *     round; a click sends company tactics focus <rule>, for that battle
 *     only. "Saved" (company tactics focus default) returns to the saved
 *     focus. The live region says when the focus changes.
 *   - Setup's tactics row: the saved focus and healing threshold, with a
 *     menu sending company tactics focus / healing / default.
 *
 * Phase 30c2, guardians:
 *
 *   - guardian is a role in the member menu; a guardian's menu adds
 *     "Guard: <member>" (strategy <who> guard <other>) and "Guard: the
 *     most hurt" (strategy <who> guard). Its row says whom it guards,
 *     "(out of reach)" when the server marks the ward too far away.
 *   - In the Battle view a guardian's button adds whom it guards and its
 *     guards left (Company.Battle's guards).
 *
 * Phase 33i1, the company's outlook:
 *
 *   - Below the focus buttons, scout's assessment headline for the battle's
 *     group (Company.Battle's outlook: risk and closeness in words, never a
 *     number). None in the dark.
 *   - Phase 33i2: under it, how the group fights together and the roles it
 *     shows (the outlook's coordination), as scout says it.
 *
 * Phase 61, battle orders:
 *
 *   - Under each member's row in Setup, their battle orders in the words the
 *     `orders` command uses (Company's members[].orders), read in order
 *     each round before the member's strategy, and an "Orders" button whose
 *     menu sends orders <who> add <condition> then <action>, remove <n>,
 *     up <n>, preset and clear. Orders are set between battles: in a battle
 *     they are listed, with no button.
 *
 * Phase 69, weapon stances:
 *
 *   - Under each member's orders, their weapon stance (Company's
 *     members[].stance: name, whether the member holds what it needs, and
 *     the trade in the tooltip) and a "Stance" button whose menu sends
 *     stance <who> <key> or off. Stances are set between battles: in a
 *     battle a set stance is listed, with no button.
 *
 * Every name is set with textContent, never innerHTML.
 *
 * Responds to GMCP namespaces:
 *   Company         - the snapshot: members, cells, strategies
 *   Company.Vitals  - members' health, for the Battle view
 *   Company.Battle  - the player's battle; {} when there is none
 *   Room            - Room.Info.Contents.Npcs, for Scout
 */

'use strict';

(function() {

    const el = CompanyData.el;

    const ROLES = ['fighter', 'healer', 'caster', 'guardian'];
    // Target rules, in the order `help strategy` lists them; assist is for
    // companions only.
    const RULES = ['weakest', 'strongest', 'wounded', 'nearest', 'furthest', 'leader', 'casters', 'healers', 'assist', 'defend'];
    // Phase 30c: the company focus values, in the order `help tactics`
    // lists them, and the healing thresholds.
    const FOCI = ['none', 'leader', 'casters', 'healers', 'nearest', 'weakest', 'strongest', 'wounded'];
    const HEALING = [10, 20, 30, 40, 50, 60, 70, 80, 90];

    injectStyles(`
        #combat-window {
            height: 100%;
            overflow-y: auto;
            background: var(--t-bg);
            padding: 4px 6px;
            box-sizing: border-box;
            display: flex;
            flex-direction: column;
            gap: 6px;
            font-size: 0.8em;
        }

        .cbt-note { color: var(--t-text-secondary); font-style: italic; }
        .cbt-outlook { font-style: normal; margin: 4px 0; }
        .cbt-outlook.cbt-risk-easy, .cbt-outlook.cbt-risk-fair { color: var(--t-success); }
        .cbt-outlook.cbt-risk-grave, .cbt-outlook.cbt-risk-hopeless { color: var(--t-error); }

        .cbt-grid {
            border-collapse: collapse;
            table-layout: fixed;
            width: 100%;
        }

        .cbt-grid caption { text-align: left; color: var(--t-text-secondary); padding-bottom: 2px; }
        .cbt-grid th { width: 3.4em; font-weight: normal; color: var(--t-text-secondary); text-align: left; }

        .cbt-grid td {
            border: 1px solid var(--t-accent-dim);
            height: 2.8em;
            padding: 2px;
            text-align: center;
            vertical-align: middle;
            overflow-wrap: anywhere;
            line-height: 1.2;
        }

        .cbt-grid td.empty { color: var(--t-text-dim); }
        .cbt-grid td.is-leader { color: var(--t-party-leader); font-weight: bold; }
        .cbt-grid .cbt-role { display: block; font-weight: normal; font-size: 0.85em; color: var(--t-text-secondary); }

        .cbt-members { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }

        .cbt-member {
            display: grid;
            grid-template-columns: minmax(0, 1fr) auto;
            gap: 6px;
            width: 100%;
            text-align: left;
            font: inherit;
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            padding: 3px 5px;
            cursor: pointer;
        }

        @media (hover: hover) and (pointer: fine) {
            .cbt-member:hover { background: var(--t-bg-hover); color: var(--t-text); box-shadow: inset 0 0 0 1px var(--t-accent); }
            .cbt-member:hover .cbt-how { color: inherit; }
        }
        .cbt-member:focus-visible { outline: 2px solid var(--t-accent); }
        .cbt-member .cbt-how { color: var(--t-text-secondary); white-space: nowrap; }
        .cbt-member.is-fallen { opacity: 0.6; border-style: dashed; }

        .cbt-orders { margin: 1px 0 3px 10px; font-size: 0.92em; color: var(--t-text-secondary); display: flex; flex-direction: column; gap: 2px; }
        .cbt-orders ol { margin: 0; padding-left: 1.4em; }
        .cbt-orders .cbt-btn { align-self: flex-start; }
        .cbt-stance { margin: 1px 0 3px 10px; font-size: 0.92em; color: var(--t-text-secondary); display: flex; flex-direction: column; gap: 2px; }
        .cbt-stance .cbt-btn { align-self: flex-start; }

        .cbt-actions { display: flex; flex-wrap: wrap; gap: 4px; }

        .cbt-btn {
            font: inherit;
            padding: 3px 9px;
            border: 1px solid var(--t-btn-border, var(--t-accent-dim));
            border-radius: 3px;
            background: var(--t-bg-surface);
            color: var(--t-text);
            cursor: pointer;
        }

        @media (hover: hover) and (pointer: fine) {
            .cbt-btn:hover { background: var(--t-bg-hover); color: var(--t-text); box-shadow: inset 0 0 0 1px var(--t-accent); }
        }

        /* Phase 32g2: the Battle view. */
        .cbt-live {
            position: absolute; width: 1px; height: 1px; overflow: hidden;
            clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap;
        }
        .cbt-body { display: flex; flex-direction: column; gap: 6px; }
        .cbt-battle-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
        .cbt-battle-head h3 { margin: 0; font-size: 1em; color: var(--t-text-heading); overflow-wrap: anywhere; }
        .cbt-arena { position: relative; display: flex; flex-direction: column; gap: 22px; }
        .cbt-side-label { color: var(--t-text-secondary); margin-bottom: 2px; }
        .cbt-field {
            display: grid;
            grid-template-columns: repeat(3, minmax(0, 1fr));
            gap: 3px;
        }
        .cbt-spot { min-height: 2.9em; border: 1px dashed var(--t-border-faint, var(--t-accent-dim)); border-radius: 3px; }
        .cbt-fighter {
            position: relative;
            z-index: 1;
            min-height: 2.9em;
            width: 100%;
            font: inherit;
            line-height: 1.2;
            padding: 2px 3px;
            text-align: center;
            overflow-wrap: anywhere;
            color: var(--t-text);
            background: var(--t-bg-surface-alt);
            border: 1px solid var(--t-accent-dim);
            border-radius: 3px;
            cursor: pointer;
        }
        .cbt-fighter:focus-visible { outline: 2px solid var(--t-accent); }
        .cbt-fighter.is-enemy { border-color: var(--t-aggro-text, var(--t-error)); }
        .cbt-fighter.is-you { color: var(--t-party-leader); font-weight: bold; }
        .cbt-fighter.is-doll { font-style: italic; } /* Phase 39d: a Doll Master's doll */
        .cbt-fighter.is-beast { border-left: 3px solid #9a7b4f; } /* Phase 39e: a Beast Tamer's bonded beast */
        .cbt-fighter.is-fallen { opacity: 0.55; border-style: dashed; }
        .cbt-fighter.is-hl { background: var(--t-accent-dim); color: var(--t-text-white); }
        .cbt-fighter .cbt-sub { display: block; font-weight: normal; font-size: 0.85em; color: var(--t-text-secondary); }
        .cbt-fighter.is-hl .cbt-sub { color: var(--t-text-white); }
        .cbt-fighter .cbt-reach { color: var(--t-success); }
        .cbt-h-scratched, .cbt-h-wounded { color: var(--t-hp-mid); }
        .cbt-h-badly-wounded, .cbt-h-near-death { color: var(--t-hp-low); }
        .cbt-chips { display: flex; flex-wrap: wrap; gap: 3px; align-items: center; }
        .cbt-chips .cbt-fighter { width: auto; min-height: 0; }
        .cbt-lines { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; overflow: visible; z-index: 0; }
        .cbt-lines line { stroke-width: 1.5; opacity: 0.7; }
        .cbt-lines line.from-us { stroke: var(--t-accent); }
        .cbt-lines line.from-them { stroke: var(--t-error); stroke-dasharray: 4 3; }
        .cbt-lines.has-hl line { opacity: 0.12; }
        .cbt-lines.has-hl line.is-hl { opacity: 1; stroke-width: 2.5; }
        .cbt-aside { color: var(--t-text-secondary); overflow-wrap: anywhere; }
        .cbt-strikes { margin: 0; padding-left: 1.2em; color: var(--t-text-secondary); }
        .cbt-focus { display: flex; flex-wrap: wrap; gap: 3px; align-items: center; }
        .cbt-focus .cbt-btn { padding: 2px 6px; }
        .cbt-focus .cbt-btn[aria-pressed="true"] { background: var(--t-accent-dim); color: var(--t-text-white); font-weight: bold; }
        .cbt-focus .cbt-btn:disabled { opacity: 0.55; cursor: default; }
        .cbt-rounds { margin-top: 6px; }
        .cbt-rounds > summary { cursor: pointer; color: var(--t-text-secondary); }
        .cbt-rounds ul { list-style: none; margin: 4px 0 0; padding: 0; }
        .cbt-rounds li { margin: 0 0 2px; overflow-wrap: anywhere; }
        .cbt-rounds li summary { cursor: pointer; }
        .cbt-rounds .cbt-why { margin: 2px 0 4px 1em; color: var(--t-text-secondary); font-size: 0.9em; }
        .cbt-setup > summary { cursor: pointer; color: var(--t-text-secondary); }
        .cbt-setup[open] > summary { margin-bottom: 4px; }
    `);

    function createDOM() {
        const root = el('div');
        root.id = 'combat-window';
        // The live region outlives each rebuild of the body, so screen
        // readers hear what it says.
        const live = el('div', 'cbt-live');
        live.id = 'combat-live';
        live.setAttribute('aria-live', 'polite');
        live.setAttribute('role', 'status');
        root.appendChild(live);
        const body = el('div', 'cbt-body');
        body.id = 'combat-body';
        body.appendChild(el('div', 'cbt-note', 'No company yet.'));
        root.appendChild(body);
        document.body.appendChild(root);
        return root;
    }

    // Phase 62: the explained rounds of the latest fight, kept after it ends.
    const rounds = window.BattleRounds ? window.BattleRounds.create(12) : null;
    let roundsOpen = false;
    const roundsOpenIds = new Set();

    // rememberRounds reads which rounds are open before a rebuild drops them.
    function rememberRounds(root) {
        const d = root.querySelector('details.cbt-rounds');
        if (!d) { return; }
        roundsOpen = d.open;
        roundsOpenIds.clear();
        d.querySelectorAll('li details[open]').forEach(n => roundsOpenIds.add(Number(n.getAttribute('data-round'))));
    }

    function renderRounds(root) {
        const list = rounds ? rounds.list() : [];
        if (!list.length) { return; }
        const d = el('details', 'cbt-rounds');
        d.open = roundsOpen;
        d.appendChild(el('summary', null, 'Last rounds: why each blow went as it did (' + list.length + ')'));
        const ul = el('ul');
        list.forEach(r => {
            const li = el('li');
            const inner = el('details');
            inner.open = roundsOpenIds.has(r.id);
            inner.setAttribute('data-round', r.id);
            inner.appendChild(el('summary', null, (r.round ? 'Round ' + r.round + ': ' : '') + r.head));
            r.lines.forEach(line => inner.appendChild(el('div', 'cbt-why', line)));
            li.appendChild(inner);
            ul.appendChild(li);
        });
        d.appendChild(ul);
        root.appendChild(d);
    }

    const win = new VirtualWindow('Combat', {
        dock:          'right',
        defaultDocked: true,
        tabGroup:      'dock',
        onTabShown() {
            VirtualWindows.setTabBadge('Combat', 0);
            requestAnimationFrame(drawLines); // lines need a laid-out arena
        },
        factory() {
            const root = createDOM();
            setTimeout(update, 0);
            return {
                title:      'Combat',
                mount:      root,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     320,
                header:     20,
                bottom:     60,
            };
        },
    });

    // who names a member to `strategy` and `formation`.
    function who(m) {
        return m.key === 'leader' ? 'me' : '#' + m.id;
    }

    // wardName names a guardian's ward among the members: "you" for the
    // player; none (or one gone) is the most hurt (Phase 30c2).
    function wardName(key, members) {
        const w = key ? members.find(x => x.key === key) : null;
        if (!w) { return 'the most hurt'; }
        return w.key === 'leader' ? 'you' : w.name;
    }

    function guardText(m, members) {
        const s = m.strategy;
        if (!s || s.role !== 'guardian') { return ''; }
        return 'guards ' + wardName(s.ward, members) + (s.ward_reach === false ? ' (out of reach)' : '');
    }

    // abilityText is a member's class abilities, or that they are off, and
    // its mana reserve (Phase 33e).
    function abilityText(s) {
        const parts = [];
        if (s.abilities_off) {
            parts.push('abilities off');
        } else if (Array.isArray(s.abilities) && s.abilities.length) {
            parts.push(s.abilities.join(', '));
        }
        if (s.reserve) { parts.push('keeps ' + s.reserve + '% mana'); }
        return parts.join(', ');
    }

    // MAX_ORDERS is the most orders a member carries; ORDER_ADDS are the
    // orders the menu offers, as `orders <who> add` reads them, each with
    // its short label (the command checks every one).
    const MAX_ORDERS = 3;
    const ORDER_ADDS = [
        ['ally 25 then heal', 'Ally below 25%: heal them'],
        ['ally 50 then heal', 'Ally below 50%: heal them'],
        ['ally 75 then heal', 'Ally below 75%: heal them'],
        ['self 50 then heal', 'Me below 50%: heal myself'],
        ['ally 25 then guard', 'Ally below 25%: guard them'],
        ['ally 50 then guard', 'Ally below 50%: guard them'],
        ['ally 75 then guard', 'Ally below 75%: guard them'],
        ['chanting then break', 'A foe chants: break it'],
        ['foe healer then break', 'A healer stands: turn on it'],
        ['foe caster then break', 'A caster stands: turn on it'],
        ['boss then break', 'A boss stands: turn on it'],
        ['boss then strongest', 'A boss stands: cast my strongest'],
        ['first then strongest', 'Battle opens: cast my strongest'],
        ['first then hold', 'Battle opens: hold my mana'],
    ];

    // ordersMenu is a member's orders menu: take one off, read one sooner,
    // add one of the offered, the class's starting set, or clear them all.
    function ordersMenu(m, list) {
        const w = who(m);
        const items = [];
        list.forEach((line, i) => {
            items.push({ label: 'Remove order ' + (i + 1), cmd: 'orders ' + w + ' remove ' + (i + 1) });
            if (i > 0) { items.push({ label: 'Read order ' + (i + 1) + ' sooner', cmd: 'orders ' + w + ' up ' + (i + 1) }); }
        });
        if (list.length < MAX_ORDERS) {
            ORDER_ADDS.forEach(a => items.push({ label: 'Add: ' + a[1], cmd: 'orders ' + w + ' add ' + a[0] }));
        }
        items.push({ label: 'Starting set for the class', cmd: 'orders ' + w + ' preset' });
        if (list.length) { items.push({ label: 'Clear all orders', cmd: 'orders ' + w + ' clear' }); }
        return items;
    }

    // ordersBlock is the orders under a member's row: the list, and the
    // button that edits it between battles.
    function ordersBlock(m, inBattle) {
        const list = Array.isArray(m.orders) ? m.orders : [];
        if (m.status === 'dead' || (inBattle && !list.length)) { return null; }
        const box = el('div', 'cbt-orders');
        box.setAttribute('data-key', m.key);
        if (list.length) {
            const ol = el('ol');
            ol.setAttribute('aria-label', 'Orders for ' + m.name + ', read in this order each round');
            list.forEach(line => ol.appendChild(el('li', null, line)));
            box.appendChild(ol);
        }
        if (!inBattle) {
            const b = el('button', 'cbt-btn cbt-orders-btn', list.length ? 'Orders (' + list.length + ')' : 'Orders: none');
            b.type = 'button';
            b.setAttribute('aria-haspopup', 'menu');
            b.title = 'Battle orders for ' + m.name + ': when this happens, do that (help orders)';
            b.addEventListener('click', e => uiMenu(e, ordersMenu(m, list)));
            box.appendChild(b);
        }
        return box;
    }

    // STANCES are the weapon stances the menu offers, as `stance <who> <key>`
    // reads them: key, name, the family it is for.
    const STANCES = [
        ['heavy', 'Heavy blows', 'great weapon'],
        ['wall', 'Shield wall', 'shield'],
        ['quick', 'Quick draw', 'bow'],
        ['keen', 'Keen edge', 'dagger'],
    ];

    // stanceMenu is a member's stance menu: pick one, or none.
    function stanceMenu(m) {
        const w = who(m);
        const cur = m.stance && m.stance.key;
        const items = [];
        STANCES.filter(st => st[0] !== cur).forEach(st => {
            items.push({ label: st[1] + ' (' + st[2] + ')', cmd: 'stance ' + w + ' ' + st[0] });
        });
        if (cur) { items.push({ label: 'No stance', cmd: 'stance ' + w + ' off' }); }
        return items;
    }

    // stanceBlock is the weapon stance under a member's row (Phase 69): what
    // it is, whether the member holds what it needs, and the button that
    // changes it between battles.
    function stanceBlock(m, inBattle) {
        const st = m.stance;
        if (m.status === 'dead' || (inBattle && !st)) { return null; }
        const box = el('div', 'cbt-stance');
        box.setAttribute('data-key', m.key);
        if (st) {
            let text = 'Stance: ' + st.name;
            if (st.ready === true) { text += ' (ready)'; }
            if (st.ready === false) { text += ' (idle: needs ' + st.needs + ')'; }
            const line = el('div', 'cbt-stance-line', text);
            line.title = st.name + ': ' + st.gain + ', but ' + st.cost + '.';
            box.appendChild(line);
        }
        if (!inBattle) {
            const b = el('button', 'cbt-btn cbt-stance-btn', st ? 'Stance: ' + st.name : 'Stance: none');
            b.type = 'button';
            b.setAttribute('aria-haspopup', 'menu');
            b.title = 'Weapon stance for ' + m.name + ': trade one strength for another (help stances)';
            b.addEventListener('click', e => uiMenu(e, stanceMenu(m)));
            box.appendChild(b);
        }
        return box;
    }

    function howText(m, members) {
        const s = m.strategy;
        if (!s) { return ''; }
        const g = guardText(m, members || []);
        const a = abilityText(s);
        return s.role + ', ' + s.target + (g ? ', ' + g : '') + (a ? ', ' + a : '');
    }

    function memberMenu(m, members) {
        const items = [];
        const s = m.strategy || {};
        ROLES.filter(r => r !== s.role).forEach(r => {
            items.push({ label: 'Role: ' + r, cmd: 'strategy ' + who(m) + ' ' + r });
        });
        if (s.role === 'guardian') {
            members.filter(x => x !== m && x.key !== s.ward && x.status !== 'dead').forEach(x => {
                items.push({ label: 'Guard: ' + wardName(x.key, members), cmd: 'strategy ' + who(m) + ' guard ' + who(x) });
            });
            if (s.ward) { items.push({ label: 'Guard: the most hurt', cmd: 'strategy ' + who(m) + ' guard' }); }
        }
        RULES.filter(r => r !== s.target && !(r === 'assist' && m.key === 'leader')).forEach(r => {
            items.push({ label: 'Target: ' + r, cmd: 'strategy ' + who(m) + ' target ' + r });
        });
        if (s.abilities_off) {
            items.push({ label: 'Abilities: on', cmd: 'strategy ' + who(m) + ' abilities on' });
        } else if (Array.isArray(s.abilities) && s.abilities.length) {
            items.push({ label: 'Abilities: off', cmd: 'strategy ' + who(m) + ' abilities off' });
        }
        items.push({ label: 'Back to the default', cmd: 'strategy ' + who(m) + ' default' });
        const taken = {};
        members.forEach(x => { if (x.cell) { taken[x.cell.row + ',' + x.cell.col] = x; } });
        for (let r = 0; r < 3; r++) {
            for (let c = 0; c < 3; c++) {
                const here = taken[r + ',' + c];
                if (here === m) { continue; }
                if (here) {
                    items.push({ label: 'Swap with ' + here.name, cmd: 'formation swap ' + who(m) + ' ' + who(here) });
                } else {
                    items.push({ label: 'Move to row ' + (r + 1) + ', column ' + (c + 1), cmd: 'formation move ' + who(m) + ' ' + (r + 1) + ' ' + (c + 1) });
                }
            }
        }
        if (m.cell) { items.push({ label: 'Take out of the formation', cmd: 'formation clear ' + who(m) }); }
        return items;
    }

    function grid(members) {
        const cells = [[null, null, null], [null, null, null], [null, null, null]];
        members.forEach(m => {
            const c = m.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) { cells[c.row][c.col] = m; }
        });
        const table = el('table', 'cbt-grid');
        table.appendChild(el('caption', null, 'Formation (row 1 is the front)'));
        cells.forEach((row, r) => {
            const tr = el('tr');
            const th = el('th', null, 'Row ' + (r + 1));
            th.setAttribute('scope', 'row');
            tr.appendChild(th);
            row.forEach(m => {
                const td = el('td', m ? (m.key === 'leader' ? 'is-leader' : '') : 'empty', m ? m.name : '·');
                if (m) {
                    td.title = m.name + (howText(m, members) ? ': ' + howText(m, members) : '');
                    if (m.strategy) { td.appendChild(el('span', 'cbt-role', m.strategy.role)); }
                } else {
                    td.setAttribute('aria-label', 'empty');
                }
                tr.appendChild(td);
            });
            table.appendChild(tr);
        });
        return table;
    }

    function hostileHere() {
        const room = Client.GMCPStructs.Room && Client.GMCPStructs.Room.Info;
        const npcs = room && room.Contents && room.Contents.Npcs;
        return Array.isArray(npcs) && npcs.some(n => n && (n.group || n.aggro));
    }

    // Setup: the formation grid and a row per member, each opening the
    // member menu.
    function renderSetup(root, data, inBattle) {
        if (hostileHere()) {
            const actions = el('div', 'cbt-actions');
            const scout = el('button', 'cbt-btn', 'Scout');
            scout.type = 'button';
            scout.title = 'See how the enemy groups here stand (scout)';
            scout.addEventListener('click', () => Client.SendInput('scout'));
            actions.appendChild(scout);
            root.appendChild(actions);
        }
        if (!data.company) {
            root.appendChild(el('div', 'cbt-note', 'No company yet. Your strategy: help strategy.'));
            return;
        }
        const members = data.members.filter(m => m && m.key);
        root.appendChild(grid(members));
        root.appendChild(el('div', 'cbt-note', 'A battle plays out on its own, by how you set it up here. Click a member to change it; orders say what each does when something happens.'));
        const list = el('ul', 'cbt-members');
        list.setAttribute('aria-label', 'How your company fights');
        members.forEach(m => {
            const li = el('li');
            const b = el('button', 'cbt-member' + (m.status === 'dead' ? ' is-fallen' : ''));
            b.type = 'button';
            b.setAttribute('data-key', m.key);
            b.setAttribute('aria-haspopup', 'menu');
            b.appendChild(el('span', null, m.name + (m.key === 'leader' ? ' (you)' : '')));
            const how = howText(m, members);
            b.appendChild(el('span', 'cbt-how', how || '—'));
            b.setAttribute('aria-label', m.name + (m.key === 'leader' ? ' (you)' : '') + (how ? ': ' + how : ''));
            b.addEventListener('click', e => uiMenu(e, memberMenu(m, members)));
            li.appendChild(b);
            const orders = ordersBlock(m, inBattle);
            if (orders) { li.appendChild(orders); }
            const stance = stanceBlock(m, inBattle);
            if (stance) { li.appendChild(stance); }
            list.appendChild(li);
        });
        root.appendChild(list);
        const t = data.company.tactics;
        if (t && t.focus) {
            const how = 'focus ' + t.focus + (t.healers_first ? ' (healers first when an enemy has one)' : '') + ', heal below ' + t.healing + '%';
            if (inBattle) {
                // The saved tactics are set between battles; in one, the
                // Focus buttons above call this battle's focus.
                root.appendChild(el('div', 'cbt-note', 'Company tactics: ' + how + ' (saved; set between battles, or use Focus above)'));
                return;
            }
            const tb = el('button', 'cbt-member cbt-tactics');
            tb.type = 'button';
            tb.setAttribute('data-key', 'tactics');
            tb.setAttribute('aria-haspopup', 'menu');
            tb.appendChild(el('span', null, 'Company tactics'));
            tb.appendChild(el('span', 'cbt-how', how));
            tb.setAttribute('aria-label', 'Company tactics: ' + how);
            tb.addEventListener('click', e => uiMenu(e, tacticsMenu(t)));
            root.appendChild(tb);
        }
    }

    // tacticsMenu sets the saved company tactics (Phase 30c).
    function tacticsMenu(t) {
        const items = [];
        FOCI.filter(f => f !== t.focus).forEach(f => {
            items.push({ label: 'Focus: ' + f, cmd: 'company tactics focus ' + f });
        });
        HEALING.filter(h => h !== t.healing).forEach(h => {
            items.push({ label: 'Heal below ' + h + '%', cmd: 'company tactics healing ' + h });
        });
        items.push({ label: 'Back to the defaults', cmd: 'company tactics default' });
        return items;
    }

    // focusBar is the Battle view's focus buttons (Phase 30c): one order a
    // round, for this battle only.
    function focusBar(battle) {
        const bar = el('div', 'cbt-focus');
        bar.setAttribute('role', 'group');
        bar.setAttribute('aria-label', 'Company focus' + (battle.focus_ready ? '' : ', turning next round'));
        bar.appendChild(el('span', 'cbt-aside', 'Focus:'));
        const add = (label, value, pressed, title) => {
            const b = el('button', 'cbt-btn', label);
            b.type = 'button';
            b.setAttribute('data-focus', value);
            b.setAttribute('aria-pressed', pressed ? 'true' : 'false');
            b.title = title;
            b.disabled = !battle.focus_ready;
            b.addEventListener('click', () => Client.SendInput('company tactics focus ' + value));
            bar.appendChild(b);
        };
        FOCI.forEach(f => add(f, f, battle.focus === f,
            f === 'none' ? 'Each fights by their own strategy, this battle only' : 'Call the company onto their ' + f + ', this battle only'));
        if (battle.saved_focus && battle.saved_focus !== battle.focus) {
            add('saved (' + battle.saved_focus + ')', 'default', false, 'Back to your saved focus');
        }
        if (!battle.focus_ready) { bar.appendChild(el('span', 'cbt-aside', 'turning next round')); }
        // Phase 35e: the default focus goes for the enemy's healer first.
        if (battle.healers_first) {
            bar.appendChild(el('span', 'cbt-aside', 'Default: healers first (an enemy healer stands)'));
        }
        return bar;
    }

    // ---------------------------------------------------------------------
    // Phase 32g2: the Battle view.
    // ---------------------------------------------------------------------

    function currentBattle() {
        const stored = Client.GMCPStructs.Company;
        const b = stored && stored.Battle;
        return (b && Array.isArray(b.enemies)) ? b : null;
    }

    // lines holds this render's target lines, { from, to, us }, by fighter
    // id: an enemy's "m:<id>", a member's key, an outsider's "u:<id>".
    let lines = [];
    let pinned = null;   // a tapped fighter whose lines stay lit
    let arenaObserver = null;
    // The newest Company.Battle. A Company snapshot replaces everything
    // stored under Company, the battle too, and the server sends the battle
    // again right after it; until then the view keeps this one rather than
    // flicker to Setup and back (32g2 review finding 1).
    let lastBattle = null;

    // The company's fighters: every member from the snapshot (the player
    // alone, as "You", without a company).
    function ourFighters(data) {
        if (!data.company) { return [{ key: 'leader', name: 'You', cell: null, status: 'present' }]; }
        return data.members.filter(m => m && m.key && m.status !== 'awaiting' && m.status !== 'fled' && m.status !== 'separated' && m.status !== 'errand');
    }

    function fighterButton(id, cls, name, sub, spoken) {
        const b = el('button', 'cbt-fighter ' + cls);
        b.type = 'button';
        b.setAttribute('data-fid', id);
        b.appendChild(el('span', null, name));
        if (sub) { b.appendChild(sub); }
        b.setAttribute('aria-label', spoken);
        b.addEventListener('mouseenter', () => light(id));
        b.addEventListener('mouseleave', () => light(pinned));
        b.addEventListener('focus', () => light(id));
        b.addEventListener('blur', () => light(pinned));
        return b;
    }

    // field lays fighters out on the formation's grid, three columns wide,
    // from the front row to the deepest row anyone stands in (empty rows
    // behind them would only take room). The enemy's is drawn back row
    // first, so both front rows face the middle.
    function field(fighters, flip, label, narrow) {
        const g = el('div', 'cbt-field');
        g.setAttribute('role', 'group');
        g.setAttribute('aria-label', label);
        if (narrow) { g.style.gridTemplateColumns = 'repeat(2, minmax(0, 1fr))'; }
        const at = {};
        let deepest = 0;
        fighters.forEach(f => {
            const c = f.cell;
            if (c && c.row >= 0 && c.row < 3 && c.col >= 0 && c.col < 3) {
                at[c.row + ',' + c.col] = f;
                deepest = Math.max(deepest, c.row);
            }
        });
        for (let i = 0; i <= deepest; i++) {
            const r = flip ? deepest - i : i;
            for (let c = 0; c < (narrow ? 2 : 3); c++) {
                const f = at[r + ',' + c];
                g.appendChild(f ? f.node : el('div', 'cbt-spot'));
            }
        }
        if (narrow) {
            fighters.filter(f => f.cell && f.cell.col === 2).forEach(f => {
                const reserve = el('div', 'cbt-reserve');
                reserve.style.gridColumn = '1 / -1';
                reserve.appendChild(el('span', 'cbt-note', 'Reserve: '));
                reserve.appendChild(f.node);
                g.appendChild(reserve);
            });
        }
        return g;
    }

    function nameOf(id, battle, data) {
        if (id === 'leader') { return data.company ? data.company.leader.name : 'you'; }
        const e = battle.enemies.find(x => x.id === id);
        if (e) { return e.label; }
        const o = (battle.others || []).find(x => x.id === id);
        if (o) { return o.name; }
        const m = data.members.find(x => x.key === id);
        if (m) { return m.name; }
        const d = (battle.dolls || []).find(x => x.key === id); // Phase 39d
        return d ? d.name : '';
    }

    // dollOwner names a doll's Master: "your" or a companion's name with 's.
    function dollOwner(d, data) {
        if (d.master === 'leader') { return 'your'; }
        const m = data.members.find(x => x.key === d.master);
        return m ? m.name + '\'s' : 'a';
    }

    function renderBattle(root, battle, data) {
        const head = el('div', 'cbt-battle-head');
        head.appendChild(el('h3', null, 'Battle: ' + battle.group));
        if (battle.retreat) {
            const left = battle.retreat.rounds;
            head.appendChild(el('div', null, 'Withdrawing ' + battle.retreat.exit + ' (' + left + (left === 1 ? ' round' : ' rounds') + ' remaining)'));
        }
        const retreat = el('button', 'cbt-btn', 'Retreat');
        retreat.type = 'button';
        retreat.title = 'Withdraw your company: one round to prepare, then the attempt (retreat)';
        retreat.addEventListener('click', () => Client.SendInput('retreat'));
        head.appendChild(retreat);
        // Phase 40f: the picture of this battle (it opens itself unless turned off).
        if (window.BattleScreen) {
            const picture = el('button', 'cbt-btn', 'Battle screen');
            picture.type = 'button';
            picture.title = 'Show the battle as a picture (help battlescreen)';
            picture.addEventListener('click', () => window.BattleScreen.open());
            head.appendChild(picture);
        }
        root.appendChild(head);
        if (typeof battle.focus === 'string') { root.appendChild(focusBar(battle)); }
        // Phase 39c: a Shaman's weather over the battle (help shaman).
        if (battle.weather && battle.weather.name) {
            const w = battle.weather;
            const note = el('div', 'cbt-note cbt-weather',
                'Weather: ' + w.name + ', ' + (w.endless ? 'the whole battle' : w.rounds + (w.rounds === 1 ? ' round' : ' rounds')) + ' (' + w.effect + ')');
            note.title = 'A Shaman\'s weather, this battle only (help shaman)';
            root.appendChild(note);
        }
        // Phase 54: the sigil the company stands in (help sigils).
        if (battle.sigil && battle.sigil.name) {
            const sg = battle.sigil;
            const note = el('div', 'cbt-note cbt-sigil', 'Sigil: ' + sg.name + ' (' + sg.effect + ')');
            note.title = 'Laid before the fight with cast sigil of [kind] (help sigils)';
            root.appendChild(note);
        }
        // Phase 50: each member's battle condition as the battle began (help survival).
        const fare = battle.fare || {};
        Object.keys(fare).forEach(key => {
            const note = el('div', 'cbt-note cbt-fare', 'Condition: ' + (nameOf(key, battle, data) || key) + ', ' + fare[key]);
            note.title = 'Set as the battle began from needs and a meal buff (help survival, help cooking)';
            root.appendChild(note);
        });
        if (battle.outlook && battle.outlook.text) {
            const outlook = el('div', 'cbt-note cbt-outlook cbt-risk-' + String(battle.outlook.risk || '').replace(/[^a-z]/g, ''),
                'Outlook: ' + battle.outlook.text);
            outlook.title = 'Your company\'s assessment, as scout gives it (help assessment)';
            root.appendChild(outlook);
            if (battle.outlook.coordination) {
                const coord = el('div', 'cbt-note cbt-coordination', battle.outlook.coordination);
                coord.title = 'How the enemy fights together (help coordination)';
                root.appendChild(coord);
            }
        }

        // Phase 66: what the bestiary knows of the foes' habits (help bestiary).
        const knownFoes = battle.dark ? [] : battle.enemies.filter(e => Array.isArray(e.known) && e.known.length);
        if (knownFoes.length) {
            const note = el('div', 'cbt-note cbt-bestiary',
                'Bestiary: ' + knownFoes.map(e => e.label + ' ' + e.known.join(', ')).join('; '));
            note.title = 'Habits you have learned by beating this kind (help bestiary)';
            root.appendChild(note);
        }

        lines = [];
        if (battle.dark) {
            // As scout: in the dark, nothing to see (32g2 review finding 3).
            root.appendChild(el('div', 'cbt-note', "It's too dark to make them out."));
            const setup = el('details', 'cbt-setup');
            setup.appendChild(el('summary', null, 'Setup: formation and strategies'));
            renderSetup(setup, data, true);
            root.appendChild(setup);
            return;
        }
        const aimsAt = {};
        (battle.company || []).forEach(a => { aimsAt[a.key] = a.target; lines.push({ from: a.key, to: a.target, us: true }); });

        const arena = el('div', 'cbt-arena');
        const enemies = battle.enemies.map(e => {
            const sub = el('span', 'cbt-sub');
            sub.appendChild(el('span', 'cbt-h-' + String(e.health).replace(/ /g, '-'), e.health));
            if (e.reach) { sub.appendChild(el('span', 'cbt-reach', ' •')); }
            const spoken = e.label + ', ' + e.health + (e.reach ? ', within your reach' : '') +
                (e.target ? ', striking ' + (e.target === 'leader' ? 'you' : nameOf(e.target, battle, data)) : '') +
                (e.known && e.known.length ? ', known: ' + e.known.join(', ') : '');
            if (e.target) { lines.push({ from: e.id, to: e.target, us: false }); }
            return { cell: e.cell, node: fighterButton(e.id, 'is-enemy', e.label, sub, spoken) };
        });
        enemies.forEach(f => f.node.addEventListener('click', () => pin(f.node.getAttribute('data-fid'))));
        const them = el('div');
        them.appendChild(el('div', 'cbt-side-label', battle.group + ' (front row nearest you)'));
        them.appendChild(field(enemies, true, battle.group, battle.narrow));
        arena.appendChild(them);

        const members = data.members.filter(m => m && m.key);
        const ours = ourFighters(data).map(m => {
            const v = data.company ? data.vitals(m.key) : {};
            const fallen = m.status === 'dead';
            let subText = '';
            if (fallen) { subText = 'fallen'; } else if (v.hp !== null && v.hp !== undefined) { subText = v.hp + ' / ' + v.hp_max; }
            // Phase 39g: an Alchemist's flasks left.
            const flaskNote = !fallen && v.flasks_max > 0 ? v.flasks + ' of ' + v.flasks_max + ' flasks' : '';
            // Phase 30c2: a guardian's ward and guards left.
            const g = fallen ? null : (battle.guards || []).find(x => x.key === m.key);
            const guardNote = g ? 'guards ' + wardName(g.ward, members) + ', ' +
                (g.left > 0 ? g.left + (g.left === 1 ? ' guard' : ' guards') + ' left' : 'no guards left') : '';
            const subLine = [subText, flaskNote, guardNote].filter(Boolean).join(' · ');
            const sub = subLine ? el('span', 'cbt-sub', subLine) : null;
            const you = m.key === 'leader';
            const target = aimsAt[m.key];
            const spoken = (you && data.company ? m.name + ' (you)' : m.name) + (subText ? ', ' + (fallen ? 'fallen' : 'health ' + subText.replace(' / ', ' of ')) : '') + (flaskNote ? ', ' + flaskNote : '') +
                (target ? ', striking ' + nameOf(target, battle, data) : '') + (guardNote ? ', ' + guardNote : '');
            const node = fighterButton(m.key, (you ? 'is-you' : '') + (fallen ? ' is-fallen' : ''), m.name, sub, spoken);
            if (data.company) {
                node.setAttribute('aria-haspopup', 'menu');
                node.addEventListener('click', e => { pin(m.key); uiMenu(e, memberMenu(m, members)); });
            } else {
                node.addEventListener('click', () => pin(m.key));
            }
            return { cell: battle.positions ? battle.positions[m.key] || null : m.cell, node };
        });
        // Phase 39d: a Doll Master's dolls stand in cells of their own.
        (battle.dolls || []).forEach(d => {
            const hp = d.hp + ' / ' + d.hp_max;
            const owner = dollOwner(d, data);
            const beast = !!d.kind; // Phase 39e: a Beast Tamer's bonded beast
            const noun = beast ? ({ bear: 'war bear', drake: 'drake hatchling' }[d.kind] || d.kind) : 'doll';
            const sub = el('span', 'cbt-sub', hp + ' · ' + owner + ' ' + noun);
            const node = fighterButton(d.key, beast ? 'is-beast' : 'is-doll', d.name, sub, d.name + ', ' + owner + ' ' + noun + ', health ' + d.hp + ' of ' + d.hp_max);
            node.title = beast ? 'A Beast Tamer\'s bonded beast: it takes its own turn (help beast)' : 'A Doll Master\'s doll: it strikes on its Master\'s turn (help doll)';
            node.addEventListener('click', () => pin(d.key));
            ours.push({ cell: battle.positions ? battle.positions[d.key] || null : null, node });
        });
        const us = el('div');
        const placed = ours.filter(f => f.cell);
        const loose = ours.filter(f => !f.cell);
        us.appendChild(field(placed, false, data.company ? 'Your company' : 'You', battle.narrow));
        const others = (battle.others || []).map(o =>
            fighterButton(o.id, '', o.name, el('span', 'cbt-sub', 'not in your company'), o.name + ', not in your company'));
        others.forEach(n => n.addEventListener('click', () => pin(n.getAttribute('data-fid'))));
        if (loose.length || others.length) {
            const chips = el('div', 'cbt-chips');
            if (loose.length) {
                if (data.company) { chips.appendChild(el('span', 'cbt-aside', 'Not placed:')); }
                loose.forEach(f => chips.appendChild(f.node));
            }
            if (others.length) {
                chips.appendChild(el('span', 'cbt-aside', 'Also fighting:'));
                others.forEach(n => chips.appendChild(n));
            }
            us.appendChild(chips);
        }
        us.appendChild(el('div', 'cbt-side-label', (data.company ? 'Your company' : 'You') + ' (front row nearest them)'));
        arena.appendChild(us);

        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('class', 'cbt-lines');
        svg.setAttribute('aria-hidden', 'true');
        arena.appendChild(svg);
        root.appendChild(arena);
        // Lines follow the fighters when the dock is resized, which fires
        // no window resize (32g2 review finding 6).
        if (window.ResizeObserver) {
            if (arenaObserver) { arenaObserver.disconnect(); }
            arenaObserver = new ResizeObserver(() => requestAnimationFrame(drawLines));
            arenaObserver.observe(arena);
        }

        if (battle.surrendered && battle.surrendered.length) {
 root.appendChild(el('div', 'cbt-aside', 'Surrendered: ' + battle.surrendered.map(f => f.label).join(', ')));
 }
 if (battle.fallen && battle.fallen.length) {
            root.appendChild(el('div', 'cbt-aside', 'Fallen: ' + battle.fallen.map(f => f.label).join(', ')));
        }
        if (battle.waiting && battle.waiting.length) {
            root.appendChild(el('div', 'cbt-aside', 'Waiting their turn: ' + battle.waiting.join(', ')));
        }

        const list = el('ul', 'cbt-strikes');
        list.setAttribute('aria-label', 'Who strikes whom');
        lines.forEach(l => {
            const you = data.company ? nameOf('leader', battle, data) + ' (you)' : 'you';
            const who = l.from === 'leader' ? (data.company ? you : 'You') : nameOf(l.from, battle, data);
            const whom = l.to === 'leader' ? you : nameOf(l.to, battle, data);
            list.appendChild(el('li', null, who + ' → ' + whom));
        });
        if (!lines.length) { list.appendChild(el('li', null, 'No one is striking anyone this moment.')); }
        root.appendChild(list);

        renderRounds(root);
        const setup = el('details', 'cbt-setup');
        setup.appendChild(el('summary', null, 'Setup: formation and strategies'));
        renderSetup(setup, data, true);
        root.appendChild(setup);

        requestAnimationFrame(drawLines);
    }

    // drawLines draws each target line between its two fighters' centres.
    function drawLines() {
        const svg = document.querySelector('#combat-window .cbt-lines');
        if (!svg) { return; }
        const arena = svg.parentNode;
        const box = arena.getBoundingClientRect();
        while (svg.firstChild) { svg.removeChild(svg.firstChild); }
        if (!box.width) { return; }
        const centre = id => {
            const n = arena.querySelector('[data-fid="' + (window.CSS && CSS.escape ? CSS.escape(id) : id) + '"]');
            if (!n) { return null; }
            const r = n.getBoundingClientRect();
            return { x: r.left - box.left + r.width / 2, y: r.top - box.top + r.height / 2 };
        };
        lines.forEach(l => {
            const a = centre(l.from);
            const b = centre(l.to);
            if (!a || !b) { return; }
            const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
            line.setAttribute('x1', a.x);
            line.setAttribute('y1', a.y);
            line.setAttribute('x2', b.x);
            line.setAttribute('y2', b.y);
            line.setAttribute('class', l.us ? 'from-us' : 'from-them');
            line.setAttribute('data-from', l.from);
            line.setAttribute('data-to', l.to);
            svg.appendChild(line);
        });
        light(pinned);
    }

    // light lights a fighter, its target, those striking it, and the
    // lines between them; null lights nothing.
    function light(id) {
        const root = document.getElementById('combat-window');
        if (!root) { return; }
        root.querySelectorAll('.cbt-fighter.is-hl').forEach(n => n.classList.remove('is-hl'));
        const svg = root.querySelector('.cbt-lines');
        if (svg) { svg.classList.toggle('has-hl', !!id); }
        if (!id) {
            if (svg) { svg.querySelectorAll('line.is-hl').forEach(n => n.classList.remove('is-hl')); }
            return;
        }
        const lit = new Set([id]);
        lines.forEach(l => {
            if (l.from === id) { lit.add(l.to); }
            if (l.to === id) { lit.add(l.from); }
        });
        root.querySelectorAll('.cbt-fighter[data-fid]').forEach(n => {
            if (lit.has(n.getAttribute('data-fid'))) { n.classList.add('is-hl'); }
        });
        if (svg) {
            svg.querySelectorAll('line').forEach(n => {
                n.classList.toggle('is-hl', n.getAttribute('data-from') === id || n.getAttribute('data-to') === id);
            });
        }
    }

    // pin keeps a tapped fighter lit (a touch screen has no hover); a
    // second tap lets go.
    function pin(id) {
        pinned = pinned === id ? null : id;
        light(pinned || id);
    }

    // What the live region last knew: who struck the player, who had fallen.
    let heard = null;

    function announce(battle) {
        const live = document.getElementById('combat-live');
        if (!live) { return; }
        const said = [];
        if (!battle) {
            if (heard) { said.push('The battle is over.'); }
            heard = null;
        } else {
            const onYou = new Set(battle.enemies.filter(e => e.target === 'leader').map(e => e.id));
            const fallen = new Set((battle.fallen || []).map(f => f.id));
            if (!heard) {
                said.push('Battle: ' + battle.group + '.');
            } else {
                (battle.fallen || []).forEach(f => { if (!heard.fallen.has(f.id)) { said.push(f.label + ' falls.'); } });
                battle.enemies.forEach(e => { if (onYou.has(e.id) && !heard.onYou.has(e.id)) { said.push(e.label + ' turns on you.'); } });
                if (battle.focus && heard.focus && battle.focus !== heard.focus) { said.push('Focus: ' + battle.focus + '.'); }
            }
            heard = { onYou, fallen, focus: battle.focus };
        }
        if (said.length) { live.textContent = said.join(' '); }
    }

    function update() {
        const battle = currentBattle();
        announce(battle);
        if (battle && !VirtualWindows.isTabShowing('Combat')) {
            VirtualWindows.setTabBadge('Combat', '⚔', 'a battle is under way');
        } else {
            VirtualWindows.setTabBadge('Combat', 0);
        }
        win.open();
        if (!win.isOpen()) { return; }
        const root = document.getElementById('combat-body');
        if (!root) { return; }
        const data = CompanyData.read();

        // Keep keyboard focus on the same member or fighter across the
        // rebuild.
        const focused = document.activeElement;
        const inside = focused && root.contains(focused);
        const focusedKey = (inside && focused.getAttribute('data-key')) || null;
        const focusedFid = (inside && focused.getAttribute('data-fid')) || null;
        const setupOpen = !!root.querySelector('details.cbt-setup[open]');

        keepScroll(root);
        rememberRounds(root);

        root.textContent = '';
        if (battle) {
            renderBattle(root, battle, data);
            if (setupOpen) { root.querySelector('details.cbt-setup').open = true; }
        } else {
            lines = [];
            pinned = null;
            if (arenaObserver) { arenaObserver.disconnect(); arenaObserver = null; }
            renderSetup(root, data);
            renderRounds(root);
        }

        const esc = v => (window.CSS && CSS.escape ? CSS.escape(v) : v);
        const again = focusedFid ? root.querySelector('[data-fid="' + esc(focusedFid) + '"]')
            : focusedKey ? root.querySelector('[data-key="' + esc(focusedKey) + '"]') : null;
        if (again) { again.focus(); }
    }

    window.addEventListener('resize', () => requestAnimationFrame(drawLines));

    // Phase 62: names are read as each round arrives, while the battle that
    // names them is still the current one.
    // Names seen while the battle was live are kept, so the last blows,
    // paced behind the narration past the battle's end, keep theirs; a new
    // fight starts them afresh (Phase 62 review).
    const roundNames = new Map();
    let roundNamesFight = null;
    let roundsPending = false;
    Client.onBattleEvents(msg => {
        if (!rounds) { return; }
        if (msg && msg.fight !== roundNamesFight) { roundNames.clear(); roundNamesFight = msg.fight; }
        const battle = currentBattle();
        const data = CompanyData.read();
        const name = id => {
            if (id === 'me') { return 'you'; }
            const n = battle ? nameOf(id, battle, data) : '';
            if (n) { roundNames.set(id, n); return n; }
            return roundNames.get(id) || '';
        };
        // One redraw a frame, however many rounds arrive in it.
        if (rounds.add(msg, name) && !roundsPending) {
            roundsPending = true;
            requestAnimationFrame(() => { roundsPending = false; update(); });
        }
    });

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Company', 'Room'],
        onGMCP(namespace, body) {
            if (namespace === 'Company.Battle') {
                lastBattle = (body && Array.isArray(body.enemies)) ? body : null;
            } else if (namespace === 'Company' && lastBattle) {
                const stored = Client.GMCPStructs.Company;
                if (stored && typeof stored === 'object' && !stored.Battle) { stored.Battle = lastBattle; }
            }
            if (namespace === 'Company.Inventory' || namespace === 'Company.Camp') { return; }
            if (namespace === 'Company.Vitals' && !currentBattle()) { return; }
            update();
        },
    });

})();
