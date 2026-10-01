package company

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/withdrawal"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func retreatBrawl(t *testing.T) *brawl {
	b := newBrawl(t)
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	b.hardenBandits()
	b.aria.Character.Stats.Speed.ValueAdj = 20
	for i := 1; i <= 4; i++ {
		b.companion(i).Character.Stats.Speed.ValueAdj = 20
	}
	b.fight()
	_, active := battle.Current(b.aria.UserId)
	require.True(t, active)
	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 0 }))
	return b
}

func TestOrderedRetreatTakesTwoRoundsAndMovesOnlyOwnCompany(t *testing.T) {
	b := retreatBrawl(t)
	original := b.road.RoomId
	ids := []int{}
	for i := 1; i <= 4; i++ {
		ids = append(ids, b.companion(i).InstanceId)
	}
	assert.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	require.Equal(t, characters.Retreat, b.aria.Character.Aggro.Type)
	assert.Contains(t, b.cmd("retreat", "west"), "already withdrawing")
	b.fight()
	assert.Equal(t, original, b.aria.Character.RoomId)
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Zero(t, b.aria.Character.Aggro.RoundsWaiting)
	out := b.fight()
	assert.Contains(t, out, "withdraws together east")
	assert.Equal(t, 920102, b.aria.Character.RoomId)
	_, active := battle.Current(b.aria.UserId)
	assert.False(t, active)
	for i, id := range ids {
		m := b.companion(i + 1)
		assert.Equal(t, id, m.InstanceId, "same live instance, no clone")
		assert.Equal(t, 920102, m.Character.RoomId)
		assert.Nil(t, m.Character.Aggro)
	}
	for _, m := range b.livingBandits() {
		assert.Equal(t, original, m.Character.RoomId)
	}
}

func TestOrderedRetreatRevalidatesRestrictionsAndOwnership(t *testing.T) {
	for _, change := range []string{"no-go", "no-flee", "transfer", "source-move", "locked", "no-exits", "death"} {
		t.Run(change, func(t *testing.T) {
			b := retreatBrawl(t)
			require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
			b.fight()
			m := b.companion(1)
			switch change {
			case "no-go":
				buffs.SetTestFlag("no-go")
				buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 99999, Name: "blocked", Flags: []string{"no-go"}, TriggerCount: 10})
				t.Cleanup(func() { buffs.RemoveTestBuffSpec(99999) })
				require.NoError(t, m.Character.AddBuff(99999, false))
			case "no-flee":
				buffs.SetTestFlag("no-flee")
				buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 99999, Name: "blocked", Flags: []string{"no-flee"}, TriggerCount: 10})
				t.Cleanup(func() { buffs.RemoveTestBuffSpec(99999) })
				require.NoError(t, m.Character.AddBuff(99999, false))
			case "transfer":
				m.Character.Charm(8, -1, "")
			case "source-move":
				require.NoError(t, rooms.MoveToRoom(7, 920102))
			case "locked":
				ex := b.road.Exits["east"]
				original := ex
				t.Cleanup(func() { b.road.Exits["east"] = original })
				ex.Lock.Difficulty = 1
				b.road.Exits["east"] = ex
			case "no-exits":
				original := b.road.Exits["east"]
				t.Cleanup(func() { b.road.Exits["east"] = original })
				delete(b.road.Exits, "east")
			case "death":
				m.Character.Health = 0
			}
			b.fight()
			if change == "death" {
				assert.Equal(t, 920102, b.aria.Character.RoomId)
				assert.NotEqual(t, 920102, m.Character.RoomId)
			} else if change != "source-move" {
				assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
				assert.Equal(t, b.road.RoomId, b.companion(2).Character.RoomId)
			}
		})
	}
}

func TestRetreatFailureResumesBattleAndMobilityUsesBurdenAndWounds(t *testing.T) {
	b := retreatBrawl(t)
	m := b.companion(1)
	before := withdrawal.Mobility(&m.Character)
	m.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Points: m.Character.HealthMax.Value / 4}}
	assert.Less(t, withdrawal.Mobility(&m.Character), before)
	observed := 0
	t.Cleanup(hooks.UseRetreatRollForTest(func(n int) int { observed = n; return 99 }))
	require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	b.fight()
	out := b.fight()
	assert.Contains(t, out, "retreat fails")
	assert.Equal(t, 100, observed)
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
	_, active := battle.Current(7)
	require.True(t, active)
	require.NotNil(t, b.aria.Character.Aggro)
	assert.NotEqual(t, characters.Retreat, b.aria.Character.Aggro.Type)
}

func TestGuardianCoversRetreatWithPaidTurn(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 0 }))
	guardian := b.companion(1)
	stream := b.listen()
	before := battle.GuardsLeft(7, "companion:1")
	require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	assert.Contains(t, b.fight(), "covers the withdrawal")
	assert.Equal(t, before-1, battle.GuardsLeft(7, "companion:1"))
	for _, event := range *stream {
		if event.Source.MobInstanceId == guardian.InstanceId {
			assert.NotEqual(t, combatstream.Attack, event.Kind, "cover costs the preparation attack")
		}
	}
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, 15, b.aria.Character.Aggro.RetreatInfo.Cover)
	b.fight()
	assert.Equal(t, 920102, guardian.Character.RoomId)
}

func TestMoraleSeparationDuringOrderedRetreatUsesSavedReturn(t *testing.T) {
	b := retreatBrawl(t)
	original := b.companion(1).InstanceId
	b.cmd("retreat", "east")
	b.fight()
	require.NoError(t, module.BeginFlight(7, 1))
	record, _ := module.registry.Get(7)
	member, _ := findCompanion(record, 1)
	require.True(t, member.PendingReturn)
	loyalty := member.Disposition.Loyalty
	b.fight()
	assert.Equal(t, 920102, b.aria.Character.RoomId)
	returned := b.companion(1)
	assert.NotEqual(t, original, returned.InstanceId)
	assert.Equal(t, 920102, returned.Character.RoomId)
	record, _ = module.registry.Get(7)
	member, _ = findCompanion(record, 1)
	assert.False(t, member.PendingReturn)
	assert.Equal(t, loyalty-5, member.Disposition.Loyalty)
	require.NoError(t, module.ReturnFlight(7))
	record, _ = module.registry.Get(7)
	member, _ = findCompanion(record, 1)
	assert.Equal(t, loyalty-5, member.Disposition.Loyalty, "return debt paid once")
}

func TestRetreatRechecksRoomScriptsDuringWithdrawal(t *testing.T) {
	b := retreatBrawl(t)
	b.cmd("retreat", "east")
	b.fight()
	path := b.road.GetScriptPath()
	require.NoError(t, os.WriteFile(path, []byte("function onTryExit(actor,exitName) { return false; }"), 0600))
	// Room try-exit returns the script's allow result inverted.
	scripting.InvalidateRoomVM(b.road.RoomId)
	t.Cleanup(scripting.ClearRoomVMs)
	out := b.fight()
	assert.Contains(t, out, "no legal escape route")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
}

// Phase 33c owner review: flee is the retreat order. Every active foe of
// the battle pursues, even with all of them striking a companion, so a
// back-row leader no longer slips away for certain.
func TestFleeIsTheRetreatOrderAndEveryBattleFoePursues(t *testing.T) {
	b := retreatBrawl(t)
	for _, foe := range b.livingBandits() {
		foe.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
	}
	observed := 0
	t.Cleanup(hooks.UseRetreatRollForTest(func(n int) int { observed = n; return 99 }))
	assert.Contains(t, b.cmd("flee", ""), "begins an ordered retreat")
	require.Equal(t, characters.Retreat, b.aria.Character.Aggro.Type)
	assert.Contains(t, b.fight(), "gathers at the")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId, "one round to prepare")
	out := b.fight()
	assert.Equal(t, 100, observed, "the escape was contested")
	assert.Contains(t, out, "cuts off your withdrawal")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
	_, active := battle.Current(7)
	assert.True(t, active)
}

// Phase 33c owner review: nobody is separated. A pinned companion holds
// the whole company, flee included, and nobody loses loyalty or vanishes.
func TestFleeWithPinnedMemberHoldsTheCompany(t *testing.T) {
	b := retreatBrawl(t)
	loadStatusBuffs(t)
	m := b.companion(1)
	record, _ := module.registry.Get(7)
	member, _ := findCompanion(record, 1)
	loyalty := member.Disposition.Loyalty
	require.NoError(t, m.Character.AddBuff(status.Hobbled, false))
	assert.Contains(t, b.cmd("flee", ""), "is pinned and cannot withdraw")
	assert.NotEqual(t, characters.Retreat, b.aria.Character.Aggro.Type)

	// Pinned after the order: the order lapses at the next round.
	m.Character.RemoveBuff(status.Hobbled)
	require.Contains(t, b.cmd("flee", ""), "begins an ordered retreat")
	require.NoError(t, m.Character.AddBuff(status.Hobbled, false))
	out := b.fight()
	b.fight()
	assert.Contains(t, out, "is pinned and cannot withdraw")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
	assert.Same(t, m, mobs.GetInstance(m.InstanceId), "the same companion, never separated")
	assert.Equal(t, b.road.RoomId, m.Character.RoomId)
	record, _ = module.registry.Get(7)
	member, _ = findCompanion(record, 1)
	assert.False(t, member.PendingReturn)
	assert.Equal(t, loyalty, member.Disposition.Loyalty)
}

func TestRetreatRuntimeOrderNeverSurvivesSave(t *testing.T) {
	b := retreatBrawl(t)
	before := util.GetRoundCount()
	require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	raw, err := yaml.Marshal(b.aria.Character)
	require.NoError(t, err)
	var loaded characters.Character
	require.NoError(t, yaml.Unmarshal(raw, &loaded))
	assert.Nil(t, loaded.Aggro, "restart/copyover cannot replay runtime members or withdrawal")
	assert.Equal(t, b.aria.Character.RoomId, loaded.RoomId)
	assert.Equal(t, before, util.GetRoundCount(), "a local order doesn't advance shared world time")
}

func TestRetreatLeavesOutsidersAndTemporaryFollowersInPlace(t *testing.T) {
	b := retreatBrawl(t)
	other := users.NewUserRecord(8, 1)
	other.Character.RoomId = b.road.RoomId
	other.Character.Health = 100
	users.SetTestUser(other)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8); users.RemoveTestUser(8) })
	follower := mobs.NewMobById(9101, b.road.RoomId)
	follower.Character.Charm(7, -1, "")
	b.aria.Character.TrackCharmed(follower.InstanceId, true)
	b.road.AddMob(follower.InstanceId)
	require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	b.fight()
	b.fight()
	assert.Equal(t, 920102, b.aria.Character.RoomId)
	assert.Equal(t, b.road.RoomId, other.Character.RoomId)
	assert.Equal(t, b.road.RoomId, follower.Character.RoomId)
}

func TestWaitingGroupsDoNotAddRetreatPressure(t *testing.T) {
	b := newBrawl(t)
	b.looseBandits()
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.toughen()
	current, ok := battle.Current(7)
	require.True(t, ok)
	waiting := 0
	for _, m := range b.livingBandits() {
		if current.Has(m.InstanceId) {
			m.Character.Stats.Speed.ValueAdj = 0
			continue
		}
		m.Character.Stats.Speed.ValueAdj = 100000
		m.Character.SetAggro(7, 0, characters.DefaultAttack)
		waiting++
	}
	require.Positive(t, waiting)
	b.aria.Character.Stats.Speed.ValueAdj = 100
	for i := 1; i <= 4; i++ {
		b.companion(i).Character.Stats.Speed.ValueAdj = 100
	}
	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 50 }))
	require.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	b.fight()
	b.fight()
	assert.Equal(t, 920102, b.aria.Character.RoomId, "waiting fast pursuers must not hold an otherwise successful escape")
}

func TestAnotherActiveCompanyBattleContinuesAfterRetreat(t *testing.T) {
	b := retreatBrawl(t)
	other := users.NewUserRecord(8, 2)
	other.Character.Name = "Brom"
	other.Character.RaceId = 1
	other.Character.Level = 3
	other.Character.RoomId = b.road.RoomId
	other.Character.Validate()
	other.Character.HealthMax.Value = 1000
	other.Character.Health = 1000
	users.SetTestUser(other)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8); users.RemoveTestUser(8); battle.End(8) })
	foe := mobs.NewMobById(9108, b.road.RoomId)
	b.road.AddMob(foe.InstanceId)
	foe.Character.HealthMax.Value = 1000
	foe.Character.Health = 1000
	other.Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
	foe.Character.SetAggro(8, 0, characters.DefaultAttack)
	b.fight()
	second, ok := battle.Current(8)
	require.True(t, ok)
	b.cmd("retreat", "east")
	b.fight()
	b.fight()
	remaining, ok := battle.Current(8)
	require.True(t, ok)
	assert.Equal(t, second.FightID, remaining.FightID, "same ongoing encounter")
	assert.Equal(t, b.road.RoomId, other.Character.RoomId)
	require.NotNil(t, other.Character.Aggro)
	assert.Equal(t, foe.InstanceId, other.Character.Aggro.MobInstanceId)
}

func TestLateAttachedMemberDoesNotJoinCapturedRetreat(t *testing.T) {
	b := retreatBrawl(t)
	b.cmd("retreat", "east")
	old := b.companion(1).InstanceId
	require.NoError(t, module.BeginFlight(7, 1))
	require.NoError(t, module.ReturnFlight(7))
	late := b.companion(1)
	require.NotEqual(t, old, late.InstanceId)
	late.Character.HealthMax.Value = 1000
	late.Character.Health = 1000
	b.fight()
	b.fight()
	assert.Equal(t, 920102, b.aria.Character.RoomId)
	assert.Equal(t, b.road.RoomId, late.Character.RoomId, "new live instance cannot inherit a queued relocation")
}

func TestRetreatPrunesDepartedSpellPatients(t *testing.T) {
	b := retreatBrawl(t)
	forceBlows(t, false) // A random hit must not interrupt the test chant before pruning.
	caster := b.livingBandits()[0]
	// A long chant preserves captured targets until after the withdrawal.
	info := characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}, TargetMobInstanceIds: []int{b.companion(1).InstanceId}}
	caster.Character.SetCast(10, info)
	b.cmd("retreat", "east")
	b.fight()
	b.fight()
	require.NotNil(t, caster.Character.Aggro)
	assert.Empty(t, caster.Character.Aggro.SpellInfo.TargetUserIds)
	assert.Empty(t, caster.Character.Aggro.SpellInfo.TargetMobInstanceIds)
}

type failedWithdrawalRuntime struct {
	Runtime
	failID int
}

func (r failedWithdrawalRuntime) Relocate(id, destination int) bool {
	if id == r.failID && destination == 920102 {
		return false
	}
	return r.Runtime.Relocate(id, destination)
}
func TestRetreatRelocationFailureRollsBackEveryMovedActor(t *testing.T) {
	b := retreatBrawl(t)
	previous := module.runtime
	module.runtime = failedWithdrawalRuntime{Runtime: previous, failID: b.companion(2).InstanceId}
	t.Cleanup(func() { module.runtime = previous })
	b.cmd("retreat", "east")
	b.fight()
	out := b.fight()
	assert.Contains(t, out, "could not withdraw together")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
	for i := 1; i <= 4; i++ {
		assert.Equal(t, b.road.RoomId, b.companion(i).Character.RoomId)
	}
	_, active := battle.Current(7)
	assert.True(t, active)
}
