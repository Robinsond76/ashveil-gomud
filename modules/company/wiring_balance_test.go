package company

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBalanceSpreadHasDistinctOpeningAims(t *testing.T) {
	f := newBalanceFight(t, 10, companySpread, enemySpread)
	seen := map[int]bool{}
	for _, c := range f.members() {
		require.NotNil(t, c.Aggro)
		assert.False(t, seen[c.Aggro.MobInstanceId])
		seen[c.Aggro.MobInstanceId] = true
	}
	assert.Len(t, seen, 5)
	// Real upkeep/attack resolution must preserve the controlled spread.
	f.toughen()
	for _, id := range f.enemies {
		m := mobs.GetInstance(id)
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	first := map[string]string{}
	combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Kind == combatstream.Attack && e.WeaponType != shieldBash {
			if _, ok := first[e.Source.Key()]; !ok {
				first[e.Source.Key()] = e.Target.Key()
			}
		}
	})
	// Shooting waits, automatic tackles and sub-1 tempo can delay a first blow.
	for i := 0; i < 6 && len(first) < 10; i++ {
		f.step()
	}
	seenTargets := map[string]bool{}
	for _, target := range first {
		seenTargets[target] = true
	}
	assert.Len(t, first, 10)
	assert.Len(t, seenTargets, 10)
}

func (f *balanceFight) members() []*characters.Character {
	out := []*characters.Character{f.aria.Character}
	for id := 1; id <= 4; id++ {
		out = append(out, &f.companion(id).Character)
	}
	return out
}

func TestBalanceNoNaturalRegenBetweenCombatBlows(t *testing.T) {
	f := newBalanceFight(t, 10, companySpread, enemySpread)
	f.step() // real battle exists, including its membership
	members := f.members()
	for _, id := range f.enemies {
		members = append(members, &mobs.GetInstance(id).Character)
	}
	for _, c := range members {
		c.Health = max(1, c.HealthMax.Value/2)
		c.Mana = 0
		c.EndAggro()
	}
	// No current aim is not permission to regenerate while in a battle.
	before := make([]int, len(members))
	for i, c := range members {
		before[i] = c.Health
	}
	for game := uint64(3); game <= 12; game++ {
		hooks.AutoHeal(events.NewRound{RoundNumber: game})
		events.ProcessEvents()
	}
	for i, c := range members {
		assert.Equal(t, before[i], c.Health, c.Name)
		assert.Zero(t, c.Mana, c.Name)
	}
}

func TestBalanceBuffCadenceThroughRoundHooks(t *testing.T) {
	f := newBalanceFight(t, 10, companySpread, enemySpread)
	f.step()
	spec := buffs.GetBuffSpec(status.Exposed)
	require.NotNil(t, spec)
	old := spec.RoundInterval
	spec.RoundInterval = 1
	t.Cleanup(func() { spec.RoundInterval = old })
	members := []*characters.Character{f.aria.Character, &f.companion(1).Character, &mobs.GetInstance(f.enemies[0]).Character}
	for _, c := range members {
		require.True(t, c.Buffs.AddBuff(status.Exposed, false, 3))
	}
	for game := uint64(3); game <= 4; game++ {
		hooks.UserRoundTick(events.NewRound{RoundNumber: game})
		hooks.MobRoundTick(events.NewRound{RoundNumber: game})
		events.ProcessEvents()
	}
	for _, c := range members {
		assert.Equal(t, 3, c.Buffs.TriggersLeft(status.Exposed), c.Name)
	}
	f.step()
	for _, c := range members {
		assert.Equal(t, 2, c.Buffs.TriggersLeft(status.Exposed), c.Name)
	}
}

func TestBalanceFocusOpensOnActualWeakest(t *testing.T) {
	for _, level := range []int{1, 5, 10, 30, 60, 100} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			f := newBalanceFight(t, level, companyFocus, enemySpread)
			target := f.aria.Character.Aggro.MobInstanceId
			hp := mobs.GetInstance(target).Character.Health
			for _, id := range f.enemies {
				assert.LessOrEqual(t, hp, mobs.GetInstance(id).Character.Health)
			}
			for _, c := range f.members() {
				require.NotNil(t, c.Aggro)
				assert.Equal(t, target, c.Aggro.MobInstanceId)
				c.HealthMax.Value, c.Health = 1000, 1000
			}
			for _, id := range f.enemies {
				m := mobs.GetInstance(id)
				m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
			}
			first := map[string]int{}
			combatstream.Default().Subscribe(func(e combatstream.Event) {
				if sideOf(e.Source) == sideCompany && (e.Kind == combatstream.Attack && e.WeaponType != shieldBash || e.Kind == combatstream.Ability) {
					if _, ok := first[e.Source.Key()]; !ok {
						first[e.Source.Key()] = e.Target.MobInstanceId
					}
				}
			})
			for i := 0; i < 3; i++ {
				f.step()
			}
			require.Len(t, first, 5)
			for _, id := range first {
				assert.Equal(t, target, id)
			}
		})
	}
}

func TestBalanceEarnedTurnsSurviveAnEarlierKill(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	for _, who := range []string{"me", "tamsin", "oswin", "garrick", "ysolde"} {
		require.Contains(t, b.cmd("strategy", who+" abilities off"), "class abilities")
	}
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	t.Cleanup(hooks.UseTempoForTest(func(*characters.Character) float64 { return 2 }))
	b.saveTactics(strategy.Tactics{Focus: strategy.Weakest})
	b.toughen()
	b.hardenBandits()
	opening := b.bandits["bandit cutthroat"][0]
	b.cmd("attack", fmt.Sprintf("#%d", opening))
	for _, m := range b.livingBandits() {
		m.Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
	}
	b.fight() // the opening grants one turn even at a rate of two
	b.toughen()
	b.hardenBandits()
	mobs.GetInstance(opening).Character.Health = 1
	seen := b.listen()
	b.fight()
	aria := swingsBy(*seen, "Aria")
	require.Len(t, aria, 2, "the player uses both earned turns")
	assert.Equal(t, opening, aria[0].Target.MobInstanceId)
	assert.NotEqual(t, opening, aria[1].Target.MobInstanceId)
	for _, who := range []string{"Tamsin Reed", "Brother Oswin", "Garrick Vane"} {
		blows := swingsBy(*seen, who)
		require.Len(t, blows, 2, who)
		for _, e := range blows {
			assert.NotEqual(t, opening, e.Target.MobInstanceId)
		}
	}
}

func TestBalanceStrengthDamageThroughDoCombat(t *testing.T) {
	f := newBalanceFight(t, 10, companySpread, enemySpread)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	// Pin the damage knobs: equal Strength 12 gives 2 + 21, capped at 20.
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax, cfg.Combat.DamagePerStrength = 2, 20, 1.75
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	c := f.aria.Character
	c.Equipment.Weapon = items.New(10002)
	c.Stats.Strength.ValueAdj = 12
	target := mobs.GetInstance(c.Aggro.MobInstanceId)
	target.Character.Equipment = characters.Worn{}
	target.Character.Stats.Strength.ValueAdj = 12
	target.Character.HealthMax.Value, target.Character.Health = 1000, 1000
	var blows []combatstream.Event
	combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			blows = append(blows, e)
		}
	})
	f.step()
	require.NotEmpty(t, blows)
	for _, e := range blows {
		assert.GreaterOrEqual(t, e.Damage, 21)
		assert.LessOrEqual(t, e.Damage, 26, fmt.Sprint(e))
	}
}

// Phase 35a2 acceptance 2 (small HP): every member's configured maximum at
// level 60 is at most 1.6x its level-1 maximum, with the same stat
// investment the harness gives each level. (30g6's 2-3x L10-to-L60
// envelope is retired: a level makes a character better, not thicker.)
func TestBalanceHPEnvelope(t *testing.T) {
	// The design measures the envelope before Vitality (its "HP before
	// Vitality" table): stat growth is a player's choice, so the class
	// curve is what must stay flat. Live HP is logged beside it.
	type pair struct{ before, after, live [2]int }
	values := map[string]pair{}
	for at, level := range []int{1, 60} {
		t.Run(fmt.Sprintf("L%d", level), func(t *testing.T) {
			f := newBalanceFight(t, level, companySpread, enemySpread)
			// 30g4's shipped formula: 5 base, 6/5/4/3 per class level to 20,
			// then a flat 1, and 1 per Vitality.
			oldHP := func(level, vitality int, rate float64) int {
				full := min(level, 20)
				return 5 + int(float64(full)*rate) + (level - full) + vitality
			}
			for id, c := range f.members() {
				rate := 5.0
				if id > 0 {
					rate = c.HealthGainPerLevel() * 2
				}
				v := values[c.Name]
				v.before[at] = oldHP(level, c.Stats.Vitality.ValueAdj, rate) + c.StatMod("healthmax")
				v.after[at] = configs.GetProgressionConfig().HealthAtLevel(level, 0, c.HealthGainPerLevel(), c.HPStart())
				v.live[at] = c.HealthMax.Value
				values[c.Name] = v
			}
		})
	}
	for name, v := range values {
		require.Positive(t, v.after[0], name)
		ratio := float64(v.after[1]) / float64(v.after[0])
		assert.GreaterOrEqual(t, ratio, 1.0, name)
		assert.LessOrEqual(t, ratio, 1.6, name)
		t.Logf("%s: 30g4 HP L1/L60 %d/%d; 35a2 before Vitality %d/%d (%.2fx), live %d/%d", name, v.before[0], v.before[1], v.after[0], v.after[1], ratio, v.live[0], v.live[1])
	}
}

// 30g6 amendment E, as 35a2 reshapes it: class HP still differs at level
// 60 (warrior, ranger, rogue, cleric, wizard, the cleric a caster now), and
// enemies with no class take the ranger's middle rate.
func TestBalanceClassHPShape(t *testing.T) {
	f := newBalanceFight(t, 60, companySpread, enemySpread)
	cfg := configs.GetProgressionConfig()
	warrior, ok := archetypes.HealthPerLevel("warrior")
	require.True(t, ok)
	wizard, ok := archetypes.HealthPerLevel("wizard")
	require.True(t, ok)
	cleric, _ := archetypes.HealthPerLevel("cleric")
	ranger, _ := archetypes.HealthPerLevel("ranger")
	rogue, _ := archetypes.HealthPerLevel("rogue")
	assert.True(t, warrior > ranger && ranger > rogue && rogue > cleric && cleric > wizard, "warrior %g, ranger %g, rogue %g, cleric %g, wizard %g", warrior, ranger, rogue, cleric, wizard)
	assert.Equal(t, ranger, float64(cfg.DefaultHPPerLevel), "unknown archetypes take the middle rate")
	require.NotNil(t, f)
	vitality := 0 // the design compares classes before Vitality
	warriorProfile, _ := archetypes.CombatProfile("warrior")
	wizardProfile, _ := archetypes.CombatProfile("wizard")
	strong, weak := cfg.HealthAtLevel(60, vitality, warrior, warriorProfile.HPStart), cfg.HealthAtLevel(60, vitality, wizard, wizardProfile.HPStart)
	assert.GreaterOrEqual(t, float64(strong), 1.25*float64(weak), "warrior %d against wizard %d at level 60", strong, weak)
	assert.GreaterOrEqual(t, int(cfg.HPFullLevels), 10, "class rates carry the early levels")
}

func retargetBrawl(t *testing.T) *brawl {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	for _, who := range []string{"me", "tamsin", "oswin", "garrick", "ysolde"} {
		b.cmd("strategy", who+" abilities off")
	}
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	t.Cleanup(hooks.UseTempoForTest(func(*characters.Character) float64 { return 2 }))
	b.toughen()
	b.hardenBandits()
	return b
}
func TestBalanceRetargetRespectsCasters(t *testing.T) {
	for _, chanting := range []bool{false, true} {
		t.Run(fmt.Sprint(chanting), func(t *testing.T) {
			b := retargetBrawl(t)
			b.saveTactics(strategy.Tactics{Focus: strategy.Casters})
			opening := b.bandits["bandit cutthroat"][0]
			wanted := b.bandits["bandit slinger"][0]
			mobs.GetInstance(wanted).Character.SpellBook["mm"] = 250
			if chanting {
				mobs.GetInstance(b.bandits["bandit cutthroat"][1]).Character.SpellBook["mm"] = 250
				mobs.GetInstance(wanted).Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
			}
			b.cmd("attack", fmt.Sprintf("#%d", opening))
			b.aria.Character.SetAggro(0, opening, characters.DefaultAttack)
			mobs.GetInstance(opening).Character.Health = 1
			seen := b.listen()
			b.fight()
			replacements := 0
			for _, e := range *seen {
				if e.Kind == combatstream.TargetChange && e.Source.UserId == 7 {
					replacements++
					assert.Equal(t, wanted, e.Target.MobInstanceId)
				}
			}
			require.Positive(t, replacements, "replacement passed through the actual round")
		})
	}
}

func TestBalanceReadyArcherRetainsEarnedTurn(t *testing.T) {
	for _, player := range []bool{false, true} {
		t.Run(fmt.Sprintf("player-%t", player), func(t *testing.T) {
			b := retargetBrawl(t)
			b.saveTactics(strategy.Tactics{Focus: strategy.Weakest})
			opening := b.bandits["bandit cutthroat"][0]
			c := &b.companion(4).Character
			name := "Ysolde"
			if player {
				c = b.aria.Character
				name = "Aria"
				c.Equipment.Weapon = items.New(10014)
			}
			if player {
				require.Contains(t, b.cmd("strategy", "tamsin abilities on"), "will use Tackle")
			}
			b.cmd("attack", fmt.Sprintf("#%d", opening))
			for _, m := range b.livingBandits() {
				m.Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
			}
			b.fight() // ready after the opening sling wait
			require.NotNil(t, c.Aggro)
			require.Zero(t, c.Aggro.RoundsWaiting)
			c.Aggro.ColdDelayed, c.Aggro.ColdNotice = true, true
			b.toughen()
			b.hardenBandits()
			mobs.GetInstance(opening).Character.Health = 1
			if player {
				// A successful ability happens after upkeep and before the player pass.
				// Its event injects an earlier fall without relying on player map order.
				hooks.ResetAbilitiesForTest()
				t.Cleanup(hooks.UseAbilityRollForTest(func(int) int { return 0 }))
				t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
					if e.Kind == combatstream.Ability && e.Source.Name == "Tamsin Reed" {
						mobs.GetInstance(opening).Character.Health = 0
					}
				}))
			}
			checked := false
			t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
				if e.Kind == combatstream.Attack && e.Source.Name == name {
					checked = true
					assert.True(t, c.Aggro.ColdDelayed)
					assert.True(t, c.Aggro.ColdNotice)
				}
			}))
			seen := b.listen()
			b.fight()
			blows := swingsBy(*seen, name)
			require.Len(t, blows, 1, "ready archer attacks after an earlier kill")
			assert.NotEqual(t, opening, blows[0].Target.MobInstanceId)
			require.True(t, checked)
			require.NotNil(t, c.Aggro)
			assert.Equal(t, c.Equipment.Weapon.GetSpec().WaitRounds, c.Aggro.RoundsWaiting, "firing starts the ordinary reload")
			n := len(*seen)
			b.fight()
			assert.Empty(t, swingsBy(since(*seen, n), name), "reload still costs the next round")
		})
	}
}

// Review (30g6a): enemies keep an earned turn too. A bandit whose company
// target fell earlier in the round re-aims at a living member and swings,
// as company members do, so the mirror stays symmetric.
func TestBalanceEnemyEarnedTurnSurvivesAnEarlierKill(t *testing.T) {
	b := retargetBrawl(t)
	opening := b.bandits["bandit cutthroat"][0]
	b.cmd("attack", fmt.Sprintf("#%d", opening))
	b.fight()
	b.toughen()
	b.hardenBandits()
	fallen := b.companion(2)
	var attacker *mobs.Mob
	for _, m := range b.livingBandits() {
		if m.Character.Aggro != nil && m.Character.Aggro.Type == characters.DefaultAttack {
			attacker = m
			break
		}
	}
	require.NotNil(t, attacker, "a bandit fighting")
	attacker.Character.SetAggro(0, fallen.InstanceId, characters.DefaultAttack)
	seen := b.listen()
	// The member falls to an earlier blow this round: after upkeep has
	// kept the bandit's aim, during the player's pass (players act first).
	felled := false
	t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
		if !felled && e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			felled = true
			require.Equal(t, fallen.InstanceId, attacker.Character.Aggro.MobInstanceId, "upkeep kept the aim")
			fallen.Character.Health = 0
		}
	}))
	b.fight()
	require.True(t, felled)
	var blows []combatstream.Event
	for _, e := range *seen {
		if e.Kind == combatstream.Attack && e.Source.MobInstanceId == attacker.InstanceId {
			blows = append(blows, e)
		}
	}
	require.NotEmpty(t, blows, "the bandit still swings")
	for _, e := range blows {
		assert.NotEqual(t, fallen.InstanceId, e.Target.MobInstanceId, "not at the fallen member")
	}
}

// Review (30g6a): the mirror's cleric casts nothing, yet caster-targeting
// enemies still find him: the casters cell measures caster targeting.
func TestBalanceMirrorClericIsACasterWhoCastsNothing(t *testing.T) {
	t.Cleanup(hooks.UseAimRollForTest(func(n int) int { return n - 1 })) // never the noise
	f := newBalanceFight(t, 10, companySpread, enemyCasters)
	casts := 0
	t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Kind == combatstream.CastStart && sideOf(e.Source) == sideCompany {
			casts++
		}
	}))
	f.toughen()
	for _, id := range f.enemies {
		m := mobs.GetInstance(id)
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	f.step() // upkeep chooses the opening aims
	oswin := f.companion(2).InstanceId
	aimed := 0
	for _, id := range f.enemies {
		if a := mobs.GetInstance(id).Character.Aggro; a != nil && a.MobInstanceId == oswin {
			aimed++
		}
	}
	assert.GreaterOrEqual(t, aimed, 2, "caster targeting finds the cleric (%d of 5; a fighter draws none)", aimed)
	for i := 0; i < 3; i++ {
		f.step()
	}
	assert.Zero(t, casts, "the mirror's cleric casts nothing")
}
