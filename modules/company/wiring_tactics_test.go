package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30c1 wiring: company tactics through the real strategy module's
// store, attack, and combat round (shipped config, DoCombat).

// saveTactics stores Aria's tactics in the real strategy module.
func (b *brawl) saveTactics(t strategy.Tactics) {
	b.t.Helper()
	require.NoError(b.t, strategy.SaveTactics(7, t))
}

// hold keeps every bandit standing at the given health (of 1000), so the
// rules have fixed answers through a round.
func (b *brawl) hold(hp map[int]int) {
	for _, m := range b.livingBandits() {
		hardTo(&m.Character, 1000)
		m.Character.Health = 800
		if v, ok := hp[m.InstanceId]; ok {
			m.Character.Health = v
		}
	}
}

func TestTacticsFocusOverridesStrategies(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, slinger, _, _ := b.shapeBandits()
	b.cmd("strategy", "tamsin strongest")
	b.cmd("strategy", "ysolde wounded")
	// Garrick stands at the right of the front row: the captain (front
	// left) is out of his reach.
	require.Contains(t, b.cmd("formation", "move garrick 1 3"), "Placed Garrick Vane")
	b.saveTactics(strategy.Tactics{Focus: strategy.Leader})

	b.cmd("attack", fmt.Sprintf("#%d", slinger))
	assert.Equal(t, captain, aimOf(b.aria.Character), "Aria: the focus, their leader")
	assert.Equal(t, captain, aimOf(&b.companion(1).Character), "Tamsin: the focus over her strongest")
	assert.Equal(t, captain, aimOf(&b.companion(2).Character), "Oswin: the focus")
	assert.Equal(t, captain, aimOf(&b.companion(4).Character), "Ysolde: the focus over her wounded")
	garrick := aimOf(&b.companion(3).Character)
	assert.NotEqual(t, captain, garrick, "reach still binds")
	assert.Contains(t, []int{bruiser, slinger}, garrick, "the nearest he can reach")

	// Roles stay: Oswin, a healer, still heals under a focus.
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 300
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "the healer heals")
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds)
}

func TestTacticsHealingThreshold(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("attack", fmt.Sprintf("#%d", captain))

	// Aria at 60%: above the default half, Oswin swings.
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 600
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type, "60%: no heal at the default threshold")

	// Raised to 70: he heals her.
	b.saveTactics(strategy.Tactics{Healing: 70})
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 600
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "60%: healed under a 70 threshold")
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds)
}

func TestTacticsFocusMidBattleTurnsEveryoneOnce(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, _, cutA, _ := b.shapeBandits()
	stream := b.listen()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	require.Equal(t, cutA, aimOf(&b.companion(1).Character), "by her default, the weakest")
	b.hold(map[int]int{cutA: 700})
	b.toughen()
	b.fight() // the battle begins
	require.Equal(t, cutA, aimOf(&b.companion(1).Character))

	// The order: this battle's focus is the strongest.
	require.NoError(t, battle.SetFocus(7, string(strategy.Strongest)))
	b.hold(map[int]int{bruiser: 990, captain: 900})
	b.toughen()
	out := b.fight()
	assert.Equal(t, bruiser, aimOf(b.aria.Character), "Aria turns at once")
	for id := 1; id <= 4; id++ {
		c := &b.companion(id).Character
		if c.Aggro != nil && c.Aggro.Type == characters.SpellCast {
			continue
		}
		assert.Equal(t, bruiser, aimOf(c), "%s turns at once, from a foe still standing", c.Name)
	}
	assert.Contains(t, out, "You turn toward the bandit bruiser.")
	focus := 0
	for _, e := range *stream {
		if e.Kind == combatstream.FocusChange {
			focus++
			cur, _ := battle.Current(7)
			assert.NotZero(t, e.FightID, "review: the event is placed in the battle's fight")
			assert.Equal(t, cur.FightID, e.FightID)
			assert.Equal(t, "strongest", e.Rule)
			assert.Equal(t, 7, e.Source.UserId)
		}
	}
	assert.Equal(t, 1, focus, "one focus-change event")
	assert.True(t, battle.FocusReady(7), "the order is applied: another may be given")

	// The round after, aims stick: the captain is now the strongest, but
	// nobody turns.
	b.hold(map[int]int{bruiser: 900, captain: 990})
	b.toughen()
	b.fight()
	assert.Equal(t, bruiser, aimOf(b.aria.Character), "a kept aim is sticky after the turn")
	assert.Equal(t, bruiser, aimOf(&b.companion(1).Character))
	// The saved tactics never changed.
	assert.Equal(t, strategy.NoFocus, strategy.TacticsFor(7).Focus)
}

func TestTacticsFocusTurnsAPlayerAlone(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	b.into(920103)
	hardTo(&pair[0].Character, 1000)
	pair[0].Character.Health = 900
	hardTo(&pair[1].Character, 1000)
	pair[1].Character.Health = 500

	b.cmd("attack", "ruffians")
	require.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character), "the weakest, by default")
	hardTo(b.aria.Character, 1000)
	b.fight() // the battle begins
	require.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character))
	pair[0].Character.Health, pair[1].Character.Health = 900, 500
	require.NoError(t, battle.SetFocus(7, string(strategy.Strongest)))
	hardTo(b.aria.Character, 1000)
	b.fight()
	assert.Equal(t, pair[0].InstanceId, aimOf(b.aria.Character), "alone, she turns by the focus at once")
	assert.NotNil(t, mobs.GetInstance(pair[1].InstanceId), "from a foe still standing")
}

// Enemy personalities (Phase 30c): a bandit that re-aims picks by its
// personality among the members it can reach.
func TestEnemyPersonalities(t *testing.T) {
	cases := []struct {
		rule  string
		noise int
		roll  func(int) int
		want  string // who it turns on
	}{
		{"", 0, nil, "Garrick Vane"},                          // the human race's weakest: the least health
		{"wounded", 0, nil, "Tamsin Reed"},                    // the lowest fraction
		{"casters", 0, nil, "Brother Oswin"},                  // a healer
		{"strongest", 0, nil, "Aria"},                         // the most health left (first of equals)
		{"weakest", 50, func(n int) int { return 0 }, "Aria"}, // noise: a random member (the first)
	}
	for _, c := range cases {
		t.Run(c.rule+fmt.Sprint(c.noise), func(t *testing.T) {
			b := newBrawl(t)
			b.withArchetypes("")
			b.unplaced()
			b.cmd("strategy", "tamsin fighter") // Phase 35d: no default guard to step in
			captain, _, _, _, _ := b.shapeBandits()
			b.cmd("attack", fmt.Sprintf("#%d", captain))
			b.hold(nil)
			b.toughen()
			b.fight() // the battle begins; the bandits are hostile to Aria

			if c.roll != nil {
				t.Cleanup(hooks.UseAimRollForTest(c.roll))
			}
			b.hold(nil)
			b.toughen()
			tamsin, oswin, garrick := b.companion(1), b.companion(2), b.companion(3)
			hardTo(&garrick.Character, 200) // 60%, 120
			garrick.Character.Health = 120
			tamsin.Character.Health = 300 // 30%, 300
			hardTo(&oswin.Character, 200) // 75%, 150
			oswin.Character.Health = 150
			bandit := mobs.GetInstance(captain)
			bandit.Targeting, bandit.TargetingNoise = c.rule, c.noise
			bandit.Character.Aggro = nil // it re-aims at the upkeep
			b.fight()
			require.NotNil(t, bandit.Character.Aggro, "it rejoins the fight")
			got := "Aria"
			if bandit.Character.Aggro.UserId == 0 {
				got = mobs.GetInstance(bandit.Character.Aggro.MobInstanceId).Character.Name
			}
			assert.Equal(t, c.want, got)
		})
	}
}

func TestTacticsCommand(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	out := b.cmd("company", "tactics")
	assert.Contains(t, out, "Focus:   none (each member goes for the foe its own strategy picks)")
	assert.Contains(t, out, "below 50% of their health")
	assert.Regexp(t, `Brother Oswin\s+healer, weakest`, out)
	assert.Equal(t, out, b.cmd("tactics", ""), "the shorthand")

	assert.Contains(t, b.cmd("company", "tactics focus leader"), "Your company's focus is now leader: everyone goes for their leader")
	assert.Equal(t, strategy.Leader, strategy.TacticsFor(7).Focus)
	assert.Contains(t, b.cmd("company", "tactics"), "the focus overrides whom they go for; roles stay")
	assert.Contains(t, b.cmd("company", "tactics focus assist"), `"assist" is no focus`)
	assert.Contains(t, b.cmd("company", "tactics focus sideways"), "Choose one of: none, leader, casters")
	assert.Contains(t, b.cmd("company", "tactics healing 70"), "below 70% of their health")
	assert.Contains(t, b.cmd("company", "tactics healing 45"), "from 10 to 90, in tens")
	assert.Equal(t, strategy.Tactics{Focus: strategy.Leader, Healing: 70, Patch: 80}, strategy.TacticsFor(7))
	assert.Contains(t, b.cmd("company", "tactics focus none"), "focus is now none")
	assert.Contains(t, b.cmd("company", "tactics focus default"), "back to the default for your level")
	assert.Contains(t, b.cmd("company", "tactics default"), "back to the defaults")
	assert.Equal(t, strategy.Tactics{Focus: strategy.NoFocus, Healing: 50, Patch: 80}, strategy.TacticsFor(7))
	assert.Contains(t, b.cmd("company", "tactics sideways"), "Usage: company tactics")
}

func TestTacticsCommandMidBattle(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	// Review: before the battle has begun, the order is refused as such.
	assert.Contains(t, b.cmd("company", "tactics focus strongest"), "The battle hasn't begun yet", "the battle begins at the next round")
	b.hold(nil)
	b.toughen()
	b.fight()

	b.hold(map[int]int{bruiser: 990})
	order := b.cmd("company", "tactics focus strongest")
	assert.Contains(t, order, "You call the company onto the bandit bruiser.")
	assert.Regexp(t, `Aria calls \w+ company onto the bandit bruiser\.`, order, "the room hears the order")
	assert.Contains(t, b.cmd("company", "tactics focus leader"), "Your company is still turning; try again next round.")
	assert.Contains(t, b.cmd("company", "tactics"), "In this battle: focus strongest, until it ends (turning next round)")
	for _, change := range []string{"tactics healing 70", "tactics default"} {
		assert.Contains(t, b.cmd("company", change), "only call a new focus", change)
	}
	assert.Contains(t, b.cmd("strategy", "me weakest"), "The battle is under way", "setup stays locked")
	assert.Contains(t, b.cmd("formation", "move me 1 1"), "The battle is under way")

	b.toughen()
	b.fight()
	assert.Equal(t, bruiser, aimOf(b.aria.Character), "the order turned her at the upkeep")
	for id := 1; id <= 4; id++ {
		c := &b.companion(id).Character
		if c.Aggro != nil && c.Aggro.Type == characters.SpellCast {
			continue // a chant is kept
		}
		assert.Equal(t, bruiser, aimOf(c), "%s turned by the command's order", c.Name)
	}
	assert.Contains(t, b.cmd("company", "tactics"), "(ready for an order)")
	assert.Equal(t, strategy.NoFocus, strategy.TacticsFor(7).Focus, "the saved focus is untouched")

	back := b.cmd("company", "tactics focus default")
	assert.Contains(t, back, "You let each of your company choose their own foe.")
	assert.Regexp(t, `Aria lets each of \w+ company choose their own foe\.`, back, "the room hears it")
	_, set := battle.Focus(7)
	assert.False(t, set, "back to the saved focus")
}

// TestBattleViewCarriesTheFocus (30c): Company.Battle, through the real
// feed, carries the focus, the saved one, and whether an order may be
// given.
func TestBattleViewCarriesTheFocus(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	views := battleViews(t)
	b.saveTactics(strategy.Tactics{Focus: strategy.Wounded, Healing: 60})
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hold(nil)
	b.toughen()
	b.fight()
	b.refresh(7)
	view := lastView(views, 7)
	require.NotEmpty(t, view)
	assert.Equal(t, "wounded", view["focus"], "the saved focus holds")
	assert.Equal(t, "wounded", view["saved_focus"])
	assert.Equal(t, true, view["focus_ready"])

	b.cmd("company", "tactics focus leader")
	b.refresh(7)
	view = lastView(views, 7)
	assert.Equal(t, "leader", view["focus"])
	assert.Equal(t, "wounded", view["saved_focus"])
	assert.Equal(t, false, view["focus_ready"], "the order waits for the next round")
}

// Review: an order holds for its battle only; when the battle ends the
// company aims by the saved focus again (through the real rounds).
func TestTacticsFocusRevertsWhenTheBattleEnds(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	b.saveTactics(strategy.Tactics{Focus: strategy.Wounded})
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hold(nil)
	b.toughen()
	b.fight()
	assert.Contains(t, b.cmd("company", "tactics focus leader"), "You call the company onto")
	b.hold(nil)
	b.toughen()
	b.fight()
	rule, _ := enemyparty.Focus(7)
	require.Equal(t, strategy.Leader, rule, "the order holds in its battle")

	for _, m := range b.livingBandits() { // the battle is won
		m.Character.Health = 0
		b.road.RemoveMob(m.InstanceId)
		mobs.DestroyInstance(m.InstanceId)
	}
	b.toughen()
	b.fight()
	_, inBattle := battle.Current(7)
	require.False(t, inBattle, "the battle is over")
	rule, _ = enemyparty.Focus(7)
	assert.Equal(t, strategy.Wounded, rule, "the next battle starts from the saved focus")
	assert.Equal(t, strategy.Wounded, enemyparty.PlayerAttacker(b.aria).Rule)
}

// Review: tactics, like company, can't be given while downed.
func TestTacticsRefusedWhileDowned(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.Health = -2
	b.cmd("tactics", "focus leader")
	b.cmd("company", "tactics focus leader")
	assert.Equal(t, strategy.NoFocus, strategy.TacticsFor(7).Focus)
}
