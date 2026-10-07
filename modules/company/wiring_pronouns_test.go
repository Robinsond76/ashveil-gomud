package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPronounsAndOrdinalsThroughRealRound establishes the labels through the
// real company round, rather than assigning a battle directly in the test.
func TestPronounsAndOrdinalsThroughRealRound(t *testing.T) {
	b := newBrawl(t)
	events := b.listen()
	first, second := b.bandits["bandit cutthroat"][0], b.bandits["bandit cutthroat"][1]
	firstMob, secondMob := mobs.GetInstance(first), mobs.GetInstance(second)
	require.NotNil(t, firstMob)
	require.NotNil(t, secondMob)
	b.aimAt("bandit cutthroat")
	// Keep the other foes alive until the first has really been removed;
	// a miss on its first turn must not make this test flaky.
	var transcript string
	for round := 0; round < 30 && mobs.GetInstance(first) != nil; round++ {
		b.toughen()
		for _, foe := range b.livingBandits() {
			foe.Character.HealthMax.Value = 1000
			foe.Character.Health = 1000
		}
		firstMob.Character.Health = 1
		b.aria.Character.SetAggro(0, first, characters.DefaultAttack)
		transcript += b.fight() + "\n"
	}
	require.Nil(t, mobs.GetInstance(first), "the selected foe actually falls")

	fight, ok := battle.Current(b.aria.UserId)
	require.True(t, ok)
	require.Equal(t, "first cutthroat", fight.EnemyNames[first].DisplayName)
	require.Equal(t, "second cutthroat", fight.EnemyNames[second].DisplayName)
	info, ok := combatstream.Default().Fight(fight.FightID)
	require.True(t, ok)
	assert.Contains(t, info.Enemies, combatstream.Ref{MobInstanceId: first, MobId: int(firstMob.MobId), Name: "first cutthroat"})
	assert.Contains(t, info.Enemies, combatstream.Ref{MobInstanceId: second, MobId: int(secondMob.MobId), Name: "second cutthroat"})
	assert.Equal(t, "bandit cutthroat", firstMob.Character.Name)
	assert.Equal(t, "bandit cutthroat", secondMob.Character.Name)

	// The second cutthroat must swing after the first has fallen; a round
	// of company crits could otherwise drop it before its turn. Keep it
	// standing until it has.
	swungSinceFirstFell := func() bool {
		fell := false
		for _, e := range *events {
			fell = fell || (e.Kind == combatstream.Death && e.Target.MobInstanceId == first)
			if fell && e.Kind == combatstream.Attack && e.Source.MobInstanceId == second {
				return true
			}
		}
		return false
	}
	for round := 0; round < 30 && !swungSinceFirstFell(); round++ {
		b.toughen()
		secondMob.Character.HealthMax.Value = 1000
		secondMob.Character.Health = 1000
		transcript += b.fight() + "\n"
	}
	require.True(t, swungSinceFirstFell(), "the second cutthroat fights on after the first falls")
	for _, foe := range b.livingBandits() {
		foe.Character.Health = min(foe.Character.Health, 30)
	}
	transcript += b.fightItOut(200)
	firstDeathAt, laterAttack, laterTarget, secondDeathAt := -1, -1, -1, -1
	roles := map[string]int{}
	deaths := map[int]int{}
	for i, e := range *events {
		for _, ref := range []combatstream.Ref{e.Source, e.Target, e.Previous} {
			if ref.MobInstanceId == first {
				assert.Equal(t, "first cutthroat", ref.Name)
			}
			if ref.MobInstanceId == second {
				assert.Equal(t, "second cutthroat", ref.Name)
			}
		}
		if e.Kind == combatstream.Death {
			deaths[e.Target.MobInstanceId]++
			if e.Target.MobInstanceId == first {
				firstDeathAt = i
			}
			if e.Target.MobInstanceId == second {
				secondDeathAt = i
			}
		}
		if e.Kind == combatstream.Attack {
			switch {
			case e.Source.UserId == 7:
				roles["player"]++
			case e.Target.UserId == 7:
				roles["enemy"]++
			case e.Source.LeaderUserId == 7 && e.Source.MemberKey != "":
				roles["companion"]++
			}
			if firstDeathAt >= 0 && e.Source.MobInstanceId == second {
				laterAttack = i
			}
		}
		if e.Kind == combatstream.TargetChange && e.Previous.MobInstanceId == first && e.Target.MobInstanceId == second {
			laterTarget = i
		}
		if e.Kind == combatstream.FightEnd {
			require.NotNil(t, e.Summary)
			endingNames := map[int]string{}
			for _, enemy := range e.Summary.Enemies {
				endingNames[enemy.Ref.MobInstanceId] = enemy.Ref.Name
			}
			assert.Equal(t, "first cutthroat", endingNames[first])
			assert.Equal(t, "second cutthroat", endingNames[second])
		}
	}
	for _, role := range []string{"player", "enemy", "companion"} {
		require.Positive(t, roles[role], role)
	}
	require.Equal(t, 1, deaths[first])
	require.Equal(t, 1, deaths[second])
	require.Greater(t, laterAttack, firstDeathAt)
	// Replacement now happens on the kill round, before queued death resolution.
	require.GreaterOrEqual(t, laterTarget, 0, "a replacement from first to second kept both ordinals")
	require.Greater(t, secondDeathAt, laterAttack)
	assert.Contains(t, transcript, "first cutthroat")
	assert.Contains(t, transcript, "second cutthroat")
	assert.Contains(t, transcript, summaryHeading)
	assert.True(t, strings.Index(transcript, "first cutthroat") < strings.LastIndex(transcript, "second cutthroat"))
}

func TestBattleNameGrowthReachesStream(t *testing.T) {
	b := newBrawl(t)
	b.aimAt("bandit cutthroat")
	b.toughen()
	b.fight()
	first, second := b.bandits["bandit cutthroat"][0], b.bandits["bandit cutthroat"][1]
	fight, ok := battle.Current(7)
	require.True(t, ok)
	b.road.RemoveMob(first)
	mobs.DestroyInstance(first)
	newcomer := mobs.NewMobByIdNoElite(9101, b.road.RoomId, 1)
	require.NotNil(t, newcomer)
	newcomer.SpawnGroup = brawlBandits
	b.road.AddMob(newcomer.InstanceId)
	brom := users.NewUserRecord(8, 1)
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	brom.Character.HealthMax.Value = 1000
	brom.Character.Health = 1000
	users.SetTestUser(brom)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	brom.Character.SetAggro(0, second, characters.DefaultAttack)
	b.toughen()
	b.fight()
	for _, uid := range []int{7, 8} {
		current, exists := battle.Current(uid)
		require.True(t, exists)
		assert.Equal(t, "first cutthroat", current.EnemyNames[first].DisplayName)
		assert.Equal(t, "second cutthroat", current.EnemyNames[second].DisplayName)
		assert.Equal(t, "third cutthroat", current.EnemyNames[newcomer.InstanceId].DisplayName)
		info, exists := combatstream.Default().Fight(current.FightID)
		require.True(t, exists)
		names := map[int]string{}
		for _, ref := range info.Enemies {
			names[ref.MobInstanceId] = ref.Name
		}
		assert.Equal(t, "second cutthroat", names[second])
		assert.Equal(t, "third cutthroat", names[newcomer.InstanceId])
	}
	info, ok := combatstream.Default().Fight(fight.FightID)
	require.True(t, ok)
	assert.Contains(t, info.Enemies, combatstream.Ref{MobInstanceId: first, MobId: 9101, Name: "first cutthroat"})
	battle.End(8)
	assert.Equal(t, "second cutthroat", battle.EnemyDisplayName(second, "lost"))
}

// TestMinorHealCombatPronouns casts Minor Heal through real rounds. Every
// cast can fizzle (a roll of 100 fails even a 100% chance), so a fizzled
// cast is cast again.
func TestMinorHealCombatPronouns(t *testing.T) {
	b := newBrawl(t)
	for _, tc := range []struct {
		id               int
		name, possessive string
	}{{1, "Tamsin Reed", "her"}, {2, "Brother Oswin", "his"}} {
		m := b.companion(tc.id)
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 900
		m.Character.SpellBook["heal"] = 5000
		var landed string
		for attempt := 0; attempt < 10 && (attempt == 0 || strings.Contains(landed, "the spell fizzles")); attempt++ {
			m.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "heal", TargetMobInstanceIds: []int{m.InstanceId}})
			m.Character.Aggro.RoundsWaiting = 1
			waiting := b.fight()
			assert.Contains(t, waiting, tc.name+" keeps praying, and a soft glow grows in "+tc.possessive+" hands.")
			landed = b.fight()
		}
		assert.Contains(t, landed, tc.name+" presses glowing hands to "+tc.possessive+" own wounds.")
		assert.Positive(t, m.Character.Health)
	}
	b.aria.Character.Pronouns = "she"
	b.aria.Character.HealthMax.Value = 1000
	b.aria.Character.Health = 900
	b.aria.Character.SpellBook["heal"] = 5000
	var landed string
	for attempt := 0; attempt < 10 && (attempt == 0 || strings.Contains(landed, "the spell fizzles")); attempt++ {
		b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
		b.aria.Character.Aggro.RoundsWaiting = 1
		waiting := b.fight()
		assert.Contains(t, waiting, "Your hands begin to glow")
		assert.Contains(t, waiting, "their hands")
		landed = b.fight()
	}
	assert.Contains(t, landed, "your own wounds")
	assert.Contains(t, landed, "their own wounds")
	assert.Positive(t, b.aria.Character.Health)
}

func TestTrackedSpellName(t *testing.T) {
	b := newBrawl(t)
	b.aimAt("bandit cutthroat")
	b.toughen()
	b.fight()
	id := b.bandits["bandit cutthroat"][1]
	actor := scripting.GetMob(id)
	assert.Contains(t, actor.GetCombatName(false), "second cutthroat")
	assert.Contains(t, actor.GetCharacterName(false), "bandit cutthroat")
}

func TestEnemyLabelsOnSecondarySurfaces(t *testing.T) {
	t.Run("wait and distant shot", func(t *testing.T) {
		b := startNarrationBrawl(t)
		first := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
		require.NotNil(t, first)
		b.aria.Character.SetAggro(0, first.InstanceId, characters.DefaultAttack, 1)
		first.Character.SetAggro(7, 0, characters.DefaultAttack, 1)
		b.companion(1).Character.SetAggro(0, first.InstanceId, characters.DefaultAttack, 1)
		mobs.GetInstance(b.bandits["bandit cutthroat"][1]).Character.SetAggro(0, b.companion(1).InstanceId, characters.DefaultAttack, 1)
		waiting := b.fight()
		assert.Contains(t, waiting, "first cutthroat")
		assert.NotContains(t, waiting, "bandit cutthroat")
		b.road.RemoveMob(first.InstanceId)
		verge := rooms.LoadRoom(920102)
		verge.AddMob(first.InstanceId)
		first.Character.RoomId = verge.RoomId
		b.aria.Character.Equipment.Weapon = items.New(10014)
		b.aria.Character.SetAggro(0, first.InstanceId, characters.Shooting, 0)
		b.aria.Character.Aggro.ExitName = "east"
		seen := b.listen()
		shot := b.fight()
		assert.Contains(t, shot, "first cutthroat")
		found := false
		for _, e := range *seen {
			if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
				found = true
				assert.Equal(t, first.InstanceId, e.Target.MobInstanceId)
				assert.Equal(t, "first cutthroat", e.Target.Name)
			}
		}
		require.True(t, found, "a real shot through the exit resolves")
	})
	t.Run("blocked flee", func(t *testing.T) {
		b := startNarrationBrawl(t)
		first := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
		require.NotNil(t, first)
		for _, m := range b.livingBandits() {
			m.Character.Aggro = nil
			m.Character.Stats.Speed.ValueAdj = 0
			m.Character.HealthMax.Value = 1000
			m.Character.Health = 1000
		}
		first.Character.Stats.Speed.ValueAdj = 50 // the fastest pursuer
		b.aria.Character.Stats.Speed.ValueAdj = 1
		// Phase 33c: flee is the retreat order; the fastest pursuer is the
		// one named when it cuts the withdrawal off.
		t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 99 }))
		first.Character.SetAggro(7, 0, characters.DefaultAttack)
		b.aria.Character.SetAggro(0, first.InstanceId, characters.DefaultAttack)
		b.unpin()
		require.Contains(t, b.cmd("flee", ""), "begins an ordered retreat")
		b.toughen()
		b.fight()
		b.toughen()
		b.unpin()
		var blocked string
		if got := b.fight(); strings.Contains(got, "cuts off") {
			blocked = got
		}
		require.NotEmpty(t, blocked)
		assert.Contains(t, blocked, "first cutthroat")
	})
	t.Run("fizzle", func(t *testing.T) {
		b := startNarrationBrawl(t)
		first := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
		require.NotNil(t, first)
		first.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "missing-pronoun-test"})
		first.Character.Aggro.RoundsWaiting = 0
		forceBlows(t, false) // Phase 30d1: no blow breaks the chant first
		seen := b.listen()
		got := b.fight()
		assert.Contains(t, got, "The first cutthroat falters")
		found := false
		for _, e := range *seen {
			if e.Kind == combatstream.CastComplete && e.Outcome == combatstream.OutcomeFizzled {
				found = true
				assert.Equal(t, "first cutthroat", e.Source.Name)
			}
		}
		require.True(t, found)
	})
	t.Run("shield and interception", func(t *testing.T) {
		b := startNarrationBrawl(t)
		first := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
		require.NotNil(t, first)
		for _, m := range b.livingBandits() {
			m.Character.HealthMax.Value = 1000
			m.Character.Health = 1000
		}
		shield := items.New(20004)
		require.NotZero(t, shield.ItemId)
		spec := shield.GetSpec()
		spec.BreakChance = 100
		shield.Spec = &spec
		first.Character.Health = 500
		first.Character.Equipment.Offhand = shield
		var transcript string
		for i := 0; i < 30 && first.Character.Equipment.Offhand.ItemId > 0; i++ {
			b.toughen()
			b.companion(1).Character.SetAggro(0, first.InstanceId, characters.DefaultAttack, 0)
			transcript += b.fight() + "\n"
		}
		require.Zero(t, first.Character.Equipment.Offhand.ItemId)
		assert.Contains(t, transcript, "first cutthroat's")
		assert.Contains(t, transcript, "cracks apart")
		// Put Tamsin in front of the leader; an enemy aiming at Aria hits her.
		record, ok := module.registry.Get(7)
		require.True(t, ok)
		record.Formation = domain.Formation{}
		require.NoError(t, record.Formation.Place(domain.CompanionMemberKey(1), 0, 0))
		require.NoError(t, record.Formation.Place(domain.LeaderMemberKey, 1, 0))
		module.registry.Put(record)
		first.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
		seen := b.listen()
		b.toughen()
		b.fight()
		intercepted := false
		for _, e := range *seen {
			if e.Kind == combatstream.Attack && e.Source.MobInstanceId == first.InstanceId {
				intercepted = true
				assert.Equal(t, b.companion(1).InstanceId, e.Target.MobInstanceId)
				assert.Equal(t, "first cutthroat", e.Source.Name)
			}
		}
		require.True(t, intercepted)
	})
	t.Run("beaten and fallback death", func(t *testing.T) {
		b := startNarrationBrawl(t)
		first := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
		second := mobs.GetInstance(b.bandits["bandit cutthroat"][1])
		require.NotNil(t, first)
		require.NotNil(t, second)
		first.Practice = true
		var transcript string
		for i := 0; i < 30 && mobs.GetInstance(first.InstanceId) != nil; i++ {
			b.toughen()
			first.Character.Health = 1
			b.aria.Character.SetAggro(0, first.InstanceId, characters.DefaultAttack)
			transcript += b.fight() + "\n"
		}
		require.Nil(t, mobs.GetInstance(first.InstanceId))
		assert.Equal(t, 1, strings.Count(transcript, "first cutthroat is beaten"))
		*b.messages = nil
		_, err := mobcommands.Suicide("", second, b.road)
		require.NoError(t, err)
		events.ProcessEvents()
		fallback := companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
		assert.Contains(t, fallback, "second cutthroat")
		// Phase 66's bestiary line names the kind on purpose; the death
		// narration itself must still use the battle label.
		var narration []string
		for _, line := range strings.Split(fallback, "\n") {
			if !strings.HasPrefix(line, "Bestiary:") {
				narration = append(narration, line)
			}
		}
		assert.NotContains(t, strings.Join(narration, "\n"), "bandit cutthroat")
	})
}

func startNarrationBrawl(t *testing.T) *brawl {
	b := newBrawl(t)
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}
	b.aimAt("bandit cutthroat")
	b.toughen()
	b.fight()
	return b
}

func TestSparksCombatPronouns(t *testing.T) {
	for _, tc := range []struct {
		id               int
		name, possessive string
	}{{1, "Tamsin Reed", "her"}, {2, "Brother Oswin", "his"}, {0, "Aria", "their"}} {
		t.Run(tc.name, func(t *testing.T) {
			b := startNarrationBrawl(t)
			c := b.aria.Character
			if tc.id > 0 {
				c = &b.companion(tc.id).Character
			}
			c.Pronouns = "she" // the player must still render their
			if tc.id == 2 {
				c.Pronouns = "he"
			}
			c.SpellBook["sparks"] = 5000
			c.Stats.Mysticism.ValueAdj = 1000
			var transcript string
			for attempt := 0; attempt < 30 && !strings.Contains(transcript, "flings open"); attempt++ {
				b.toughen()
				c.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{b.bandits["bandit cutthroat"][0]}})
				transcript += b.fight() + "\n"
			}
			require.Contains(t, transcript, tc.name+" flings open "+tc.possessive+" hands")
		})
	}
}
