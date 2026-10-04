package company

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func tempoBrawl(t *testing.T, speed int) (*brawl, *[]combatstream.Event) {
	b := newBrawl(t)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.TempoSpeedRef, cfg.Combat.TempoSpeedSpan = 10, 40 // fixed rates for these meter integration fixtures
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	t.Cleanup(hooks.UseTempoForTest(nil))
	b.toughen()
	b.cmd("company", "dismiss all")
	b.hardenBandits()
	b.aria.Character.Stats.Speed.ValueAdj = speed
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
	}
	return b, b.listen()
}

func TestTempoRealRoundFastAndSlow(t *testing.T) {
	for _, tc := range []struct {
		name  string
		speed int
		want  []int
	}{{"fast", 30, []int{1, 1, 2, 1, 2, 1}}, {"slow", -100, []int{1, 0, 1, 0, 1, 1}}} {
		t.Run(tc.name, func(t *testing.T) {
			b, seen := tempoBrawl(t, tc.speed)
			for _, want := range tc.want {
				n := len(*seen)
				b.fight()
				assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), want)
			}
		})
	}
}

func TestTempoRealRoundChantAndWaitingOnce(t *testing.T) {
	b, seen := tempoBrawl(t, 30)
	b.fight()
	b.fight() // the next round earns two turns
	b.aria.Character.SetCast(3, characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{b.bandits["bandit captain"][0]}})
	n := len(*seen)
	b.fight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting)
	assert.Empty(t, swingsBy(since(*seen, n), b.aria.Character.Name))
	b.aria.Character.SetAggro(0, b.bandits["bandit captain"][0], characters.DefaultAttack, 1)
	n = len(*seen)
	b.fight()
	assert.Zero(t, b.aria.Character.Aggro.RoundsWaiting)
	assert.Empty(t, swingsBy(since(*seen, n), b.aria.Character.Name), "wait reaching zero still costs this round")
}

func TestTempoRealRoundStunLosesBothTurnsAndDoesNotBank(t *testing.T) {
	b, seen := tempoBrawl(t, 30)
	loadStatusBuffs(t)
	b.fight()
	b.fight()
	b.aria.Character.AddBuff(status.Stunned, false)
	n := len(*seen)
	b.fight()
	assert.Empty(t, swingsBy(since(*seen, n), b.aria.Character.Name))
	status.Clear(b.aria.Character)
	n = len(*seen)
	b.fight()
	assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), 1, "stunned turns weren't banked")
}

func TestTempoRealRoundNewBattleStartsFresh(t *testing.T) {
	b, seen := tempoBrawl(t, 30)
	b.fight()
	b.fight() // 50 carried; would earn two next
	battle.End(7)
	n := len(*seen)
	b.fight()
	assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), 1)
}

func TestTempoRealRoundMobTurns(t *testing.T) {
	b, seen := tempoBrawl(t, 10)
	enemy := mobs.GetInstance(b.bandits["bandit captain"][0])
	enemy.Character.Stats.Speed.ValueAdj = 30
	enemy.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	for _, want := range []int{1, 1, 2, 1, 2} {
		n := len(*seen)
		b.fight()
		got := 0
		for _, e := range since(*seen, n) {
			if e.Kind == combatstream.Attack && e.Source.MobInstanceId == enemy.InstanceId && e.WeaponType != "shield-bash" {
				got++
			}
		}
		assert.Equal(t, want, got)
	}
}

func TestTempoRealRoundPvPAndCompanionTurns(t *testing.T) {
	t.Run("PvP", func(t *testing.T) {
		b, seen := tempoBrawl(t, 30)
		for _, ids := range b.bandits {
			for _, id := range ids {
				b.road.RemoveMob(id)
				mobs.DestroyInstance(id)
			}
		}
		foe := users.NewUserRecord(8, 2)
		foe.Character.Name = "Brom"
		foe.Character.RaceId = 1
		foe.Character.Level = 1
		foe.Character.RoomId = b.road.RoomId
		foe.Character.Validate()
		foe.Character.HealthMax.Value = 1000
		foe.Character.Health = 1000
		users.SetTestUser(foe)
		b.road.AddPlayer(8)
		t.Cleanup(func() { b.road.RemovePlayer(8) })
		b.aria.Character.SetAggro(8, 0, characters.DefaultAttack, 0)
		clock := util.GetRoundCount()
		for _, want := range []int{1, 1, 2} {
			n := len(*seen)
			b.fight()
			assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), want)
		}
		assert.Equal(t, clock, util.GetRoundCount(), "combat never advances global rounds")
	})
	t.Run("companion against mob", func(t *testing.T) {
		b := newBrawl(t)
		t.Cleanup(hooks.UseTempoForTest(nil))
		b.toughen()
		b.hardenBandits()
		b.unplaced()
		b.aimAt("bandit captain")
		c := b.companion(1)
		c.Character.Stats.Speed.ValueAdj = 30
		for _, m := range b.livingBandits() {
			m.Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
		}
		seen := b.listen()
		for _, want := range []int{1, 1, 2} {
			n := len(*seen)
			b.fight()
			assert.Len(t, swingsBy(since(*seen, n), c.Character.Name), want)
		}
	})
}

func TestTempoRealRoundTackleCostsOneTurn(t *testing.T) {
	b, seen := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	forceBlows(t, false)
	t.Cleanup(hooks.UseTempoForTest(nil))
	b.start()
	fighter := b.companion(1)
	fighter.Character.Stats.Speed.ValueAdj = 30
	b.fight()
	b.fight() // next meter fill earns two
	hooks.ResetAbilitiesForTest()
	for _, m := range b.livingBandits() {
		status.Clear(&m.Character)
	}
	n := len(*seen)
	b.fight()
	assert.Len(t, abilityEvents(since(*seen, n), fighter.Character.Name), 1)
	assert.Len(t, swingsBy(since(*seen, n), fighter.Character.Name), 1, "tackle uses one turn, leaving one earned attack")
}

func TestTempoRealRoundWindUpCannotBurst(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, false)
	t.Cleanup(hooks.UseTempoForTest(nil))
	t.Cleanup(hooks.UseWindUpRollForTest(func(int) int { return 0 }))
	ogre.Character.Stats.Speed.ValueAdj = 30
	ogre.Character.Aggro.RoundsWaiting = 0
	seen := b.listen()
	n := len(*seen)
	b.fight()
	assert.Empty(t, swingsBy(since(*seen, n), ogre.Character.Name), "preparation occupies whole round")
	n = len(*seen)
	b.fight()
	assert.Len(t, swingsBy(since(*seen, n), ogre.Character.Name), 1, "release round has no second swing")
}

func TestTempoSlowWaitingStillTicksOnZeroTurnRound(t *testing.T) {
	for _, direction := range []string{"player-mob", "mob-player", "mob-mob", "player-player"} {
		t.Run(direction, func(t *testing.T) {
			b, seen := tempoBrawl(t, -100)
			enemy := mobs.GetInstance(b.bandits["bandit captain"][0])
			var actor *characters.Character
			switch direction {
			case "player-mob":
				actor = b.aria.Character
			case "mob-player":
				actor = &enemy.Character
				actor.SetAggro(7, 0, characters.DefaultAttack, 0)
			case "mob-mob":
				actor = &enemy.Character
				var target *mobs.Mob
				for _, m := range b.livingBandits() {
					if m.InstanceId != enemy.InstanceId {
						target = m
						break
					}
				}
				require.NotNil(t, target)
				actor.SetAggro(0, target.InstanceId, characters.DefaultAttack, 0)
			case "player-player":
				foe := users.NewUserRecord(8, 2)
				foe.Character.Name = "Brom"
				foe.Character.RoomId = b.road.RoomId
				foe.Character.HealthMax.Value = 10000
				foe.Character.Health = 10000
				users.SetTestUser(foe)
				b.road.AddPlayer(8)
				t.Cleanup(func() { b.road.RemovePlayer(8) })
				actor = b.aria.Character
				actor.SetAggro(8, 0, characters.DefaultAttack, 0)
			}
			actor.Stats.Speed.ValueAdj = -100
			b.fight() // opening turn; next allocation is zero
			require.NotNil(t, actor.Aggro)
			actor.Aggro.RoundsWaiting = 1
			n := len(*seen)
			b.fight()
			require.NotNil(t, actor.Aggro)
			assert.Zero(t, actor.Aggro.RoundsWaiting)
			assert.Empty(t, swingsBy(since(*seen, n), actor.Name))
		})
	}
}

func TestTempoSeparatedCompanionKeepsFreshOpening(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(hooks.UseTempoForTest(nil))
	b.toughen()
	b.hardenBandits()
	b.aimAt("bandit captain")
	c := b.companion(1)
	c.Character.RoomId = b.road.RoomId + 1000
	c.Character.EndAggro()
	c.Character.Stats.Speed.ValueAdj = -100
	b.fight()
	b.fight()
	c.Character.RoomId = b.road.RoomId
	seen := b.listen()
	b.fight()
	assert.Len(t, swingsBy(*seen, c.Character.Name), 1, "absent companion never spent its opening turn")
}

func TestTempoGuardBudgetSharedAcrossTwoTurns(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	t.Cleanup(hooks.UseTempoForTest(nil))
	hooks.ResetTempoForTest()
	enemy := mobs.GetInstance(b.bandits["bandit captain"][0])
	enemy.Character.Stats.Speed.ValueAdj = 30
	for i := 0; i < 2; i++ {
		b.strike(4, false)
		b.fight()
		b.toughen()
	}
	seen := b.listen()
	b.strike(0, false)
	b.fight()
	assert.Len(t, guardEvents(*seen, combatstream.GuardUsed), 2)
	assert.Zero(t, battle.GuardsLeft(7, "companion:1"))
}

func TestTempoCounterOnlyOnceAcrossTwoTurns(t *testing.T) {
	b := guardBrawl(t)
	t.Cleanup(hooks.UseTempoForTest(nil))
	hooks.ResetTempoForTest()
	enemy := b.captain()
	enemy.Character.Stats.Speed.ValueAdj = 30
	for i := 0; i < 2; i++ {
		b.strike(4, false)
		b.fight()
		b.toughen()
	}
	forceBlocks(t)
	counterDice(t, 0, 3, 99)
	seen := b.listen()
	b.strike(1, false)
	b.fight()
	attacks, bashes := 0, 0
	for _, e := range *seen {
		if e.Kind != combatstream.Attack {
			continue
		}
		if e.Source.MobInstanceId == enemy.InstanceId {
			attacks++
		}
		if e.WeaponType == "shield-bash" && e.Source.MobInstanceId == b.companion(1).InstanceId {
			bashes++
		}
	}
	assert.Equal(t, 2, attacks)
	assert.Equal(t, 1, bashes)
}

func TestTempoOpeningStrikeEnhancesOnlyFirstTurn(t *testing.T) {
	b, seen := abilityBrawl(t, map[int]string{1: "rogue", 2: "cleric", 3: "warrior", 4: "ranger"})
	alwaysLand(t)
	t.Cleanup(hooks.UseTempoForTest(nil))
	b.start()
	rogue := b.companion(1)
	rogue.Character.Stats.Speed.ValueAdj = 1000
	rogue.Character.Equipment.Weapon = items.New(10004)
	rogue.Character.Equipment.Offhand = items.Item{}
	b.fight()
	rogue.Character.Stats.Speed.ValueAdj = 1000
	b.fight()
	rogue.Character.Stats.Speed.ValueAdj = 1000
	foe := mobs.GetInstance(aimOf(&rogue.Character))
	require.NotNil(t, foe)
	foe.Character.AddBuff(status.KnockedDown, false)
	b.hardenBandits()
	hooks.ResetAbilitiesForTest()
	n := len(*seen)
	b.fight()
	swings := swingsBy(since(*seen, n), rogue.Character.Name)
	require.Len(t, swings, 2)
	assert.True(t, swings[0].Crit)
	assert.False(t, swings[1].Crit)
	assert.Len(t, abilityEvents(since(*seen, n), rogue.Character.Name), 1)
}

func TestTempoDeathBetweenPassesCancelsSecondTurn(t *testing.T) {
	b, seen := tempoBrawl(t, 30)
	b.fight()
	b.fight()
	t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			b.aria.Character.Health = 0
		}
	}))
	n := len(*seen)
	b.fight()
	assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), 1)
}

func TestTempoNewStatusBetweenPassesStartsAfterFirstTick(t *testing.T) {
	b, seen := tempoBrawl(t, 30)
	loadStatusBuffs(t)
	b.fight()
	b.fight()
	applied := false
	t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
		if !applied && e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			b.aria.Character.AddBuff(status.Stunned, false)
			applied = true
		}
	}))
	n := len(*seen)
	b.fight()
	assert.Len(t, swingsBy(since(*seen, n), b.aria.Character.Name), 2, "a new status waits for its first tick")
	n = len(*seen)
	b.fight()
	assert.Empty(t, swingsBy(since(*seen, n), b.aria.Character.Name), "first tick now loses the round")
}

func TestTempoAimedShotRestoresOrdinarySecondTurn(t *testing.T) {
	b, seen := abilityBrawl(t, map[int]string{1: "cleric", 2: "cleric", 3: "cleric", 4: "ranger"})
	alwaysLand(t)
	t.Cleanup(hooks.UseTempoForTest(nil))
	b.start()
	ranger := b.companion(4)
	// A zero-wait ranged weapon isolates turn replacement from sling waiting.
	weapon := ranger.Character.Equipment.Weapon.GetSpec()
	weapon.ItemId = 998899
	weapon.WaitRounds = 0
	items.SetTestItemSpec(&weapon)
	t.Cleanup(func() { items.RemoveTestItemSpec(weapon.ItemId) })
	ranger.Character.Equipment.Weapon = items.New(weapon.ItemId)
	ranger.Character.Equipment.Offhand = items.Item{}
	for i := 0; i < 2; i++ {
		ranger.Character.Stats.Speed.ValueAdj = 1000
		ranger.Character.Aggro.RoundsWaiting = 0
		b.fight()
	}
	ranger.Character.Stats.Speed.ValueAdj = 1000
	ranger.Character.Aggro.RoundsWaiting = 0
	for _, m := range b.livingBandits() {
		status.Clear(&m.Character)
	}
	b.hardenBandits()
	hooks.ResetAbilitiesForTest()
	n := len(*seen)
	b.fight()
	swings := swingsBy(since(*seen, n), ranger.Character.Name)
	require.Len(t, swings, 2)
	assert.True(t, swings[0].Crit)
	assert.False(t, swings[1].Crit)
	assert.Len(t, abilityEvents(since(*seen, n), ranger.Character.Name), 1)
}

// A foe felled in the company's pass makes no blow in the enemies' pass.
func TestTempoFoeFelledByThePlayerPassDoesNotStrikeBack(t *testing.T) {
	b, seen := tempoBrawl(t, 10)
	var captain *mobs.Mob
	for _, m := range b.livingBandits() {
		if m.Character.Name == "bandit captain" {
			captain = m
		}
	}
	require.NotNil(t, captain)
	captain.Character.Aggro = nil
	captain.Character.SetAggro(7, 0, characters.DefaultAttack)
	b.fight()
	require.NotEmpty(t, swingsBy(*seen, captain.Character.Name), "the captain swings while standing")
	t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			captain.Character.Health = 0
		}
	}))
	n := len(*seen)
	b.fight()
	require.NotEmpty(t, swingsBy(since(*seen, n), b.aria.Character.Name))
	assert.Empty(t, swingsBy(since(*seen, n), captain.Character.Name))
}
