package company

import (
	"fmt"
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
	f.step()
	f.step() // shooting waits and automatic tackles consume opening actions
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

func TestBalanceStrengthDamageThroughDoCombat(t *testing.T) {
	f := newBalanceFight(t, 10, companySpread, enemySpread)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
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

// Compare actual configured maxima at the two envelope levels, keeping the
// same stat investment for the pre-tuning and current formulas.
func TestBalanceHPEnvelope(t *testing.T) {
	type pair struct{ before, after [2]int }
	values := map[string]pair{}
	for at, level := range []int{10, 60} {
		t.Run(fmt.Sprintf("L%d", level), func(t *testing.T) {
			f := newBalanceFight(t, level, companySpread, enemySpread)
			old := configs.GetProgressionConfig()
			old.HPBase, old.HPPerVitality, old.HPFullLevels, old.HPAfterFull = 5, 1, 20, 1
			for id, c := range f.members() {
				rate := 5.0
				if id > 0 {
					rate = c.HealthGainPerLevel() * 2
				}
				v := values[c.Name]
				v.before[at] = old.HealthAtLevel(level, c.Stats.Vitality.ValueAdj, rate) + c.StatMod("healthmax")
				v.after[at] = c.HealthMax.Value
				values[c.Name] = v
			}
		})
	}
	for name, v := range values {
		ratio := float64(v.after[1]) / float64(v.after[0])
		assert.GreaterOrEqual(t, ratio, 2.0, name)
		assert.LessOrEqual(t, ratio, 3.0, name)
		t.Logf("%s: old HP L10/L60 %d/%d; tuned %d/%d (%.2fx)", name, v.before[0], v.before[1], v.after[0], v.after[1], ratio)
	}
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
