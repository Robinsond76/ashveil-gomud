package company

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func battlefieldBrawl(t *testing.T) *brawl {
	b := newBrawl(t)
	roomTags := append([]string(nil), b.road.Tags...)
	t.Cleanup(func() { b.road.Tags = roomTags })
	rand.Seed(30)
	b.withArchetypes("")
	b.toughen()
	b.hold(nil)
	rec, _ := module.registry.Get(7)
	rec.Formation = domain.Formation{{domain.CompanionMemberKey(3), domain.CompanionMemberKey(1), ""}, {"", "", domain.CompanionMemberKey(4)}, {domain.CompanionMemberKey(2), domain.LeaderMemberKey, ""}}
	module.registry.Put(rec)
	return b
}

func TestBattlefieldOpeningLastsExactlyOneRealRound(t *testing.T) {
	for _, advantage := range []int{-1, 0, 1} {
		t.Run(string(rune('b'+advantage)), func(t *testing.T) {
			b := battlefieldBrawl(t)
			stream := b.listen()
			for _, m := range b.livingBandits() {
				m.AmbushOwner, m.AmbushAdvantage, m.AmbushObserver = 7, advantage, "Ysolde"
			}
			b.aimAt("bandit captain")
			turn, round := util.GetTurnCount(), util.GetRoundCount()
			text := b.fight()
			bt, ok := battle.Current(7)
			require.True(t, ok)
			assert.Equal(t, advantage, bt.Opening)
			company, enemy := 0, 0
			for _, e := range *stream {
				if e.Kind != combatstream.Attack {
					continue
				}
				if e.Source.UserId == 7 || b.companyInstances()[e.Source.MobInstanceId] {
					company++
				} else {
					enemy++
				}
			}
			if advantage < 0 {
				assert.Zero(t, company)
				assert.Contains(t, text, "enemy has the opening round")
			}
			if advantage > 0 {
				assert.Zero(t, enemy)
				assert.Contains(t, text, "company has the opening round")
			}
			for _, m := range b.livingBandits() {
				assert.Zero(t, m.AmbushOwner, "consumed, no replay")
			}
			before := len(*stream)
			b.toughen()
			b.hold(nil)
			b.fight()
			company, enemy = 0, 0
			for _, e := range (*stream)[before:] {
				if e.Kind != combatstream.Attack {
					continue
				}
				if e.Source.UserId == 7 || b.companyInstances()[e.Source.MobInstanceId] {
					company++
				} else {
					enemy++
				}
			}
			assert.Positive(t, company)
			assert.Positive(t, enemy)
			assert.Equal(t, turn, util.GetTurnCount())
			assert.Equal(t, round, util.GetRoundCount())
		})
	}
}

func TestBattlefieldSurprisePausesChantsAndAutomaticAbilities(t *testing.T) {
	b := battlefieldBrawl(t)
	stream := b.listen()
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.AmbushOwner, m.AmbushAdvantage = 7, -1
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 10)
	}
	b.aria.Character.SetCast(3, characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{b.bandit("bandit captain").InstanceId}})
	b.fight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, 3, b.aria.Character.Aggro.RoundsWaiting)
	for _, e := range *stream {
		if e.Kind == combatstream.Ability || e.Kind == combatstream.CastStart || e.Kind == combatstream.CastProgress {
			assert.NotEqual(t, 7, e.Source.UserId)
			assert.False(t, b.companyInstances()[e.Source.MobInstanceId])
		}
	}
	b.toughen()
	b.hold(nil)
	b.fight()
	assert.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting)
}

func TestBattlefieldNarrowGroundPreservesSavedFormation(t *testing.T) {
	b := battlefieldBrawl(t)
	saved, ok := domain.FormationFor(7)
	require.True(t, ok)
	b.road.Tags = append(b.road.Tags, "narrow")
	effective, ok := enemyparty.CompanyFormation(7)
	require.True(t, ok)
	assert.NotEqual(t, saved, effective)
	again, _ := domain.FormationFor(7)
	assert.Equal(t, saved, again)
	for _, row := range effective {
		assert.Empty(t, row[2], "five members fit")
	}
	for _, p := range enemyparty.Parties(b.road) {
		for _, row := range p.Formation {
			assert.Empty(t, row[2], "five foes fit")
		}
	}
	b.aimAt("bandit captain")
	assert.Contains(t, b.fight(), "two columns")
	b.road.Tags = nil
	restored, _ := enemyparty.CompanyFormation(7)
	assert.Equal(t, saved, restored)
}

func TestBattlefieldSparksHitsOnlyCurrentOrthogonalCells(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.fight()
	p, ok := enemyparty.PartyOf(b.road, b.bandit("bandit captain").InstanceId)
	require.True(t, ok)
	primary := p.Formation.At(0, 0)
	ids := p.Members
	center, ok := mobparty.InstanceIdFromMemberKey(primary)
	require.True(t, ok)
	require.NotZero(t, center)
	b.aria.Character.SetAggro(0, center, characters.DefaultAttack, 0)
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: ids})
	stream := b.listen()
	b.aria.Character.SpellBook["sparks"] = 5000
	b.aria.Character.Stats.Mysticism.ValueAdj = 1000
	b.fight()
	allowed := formationcombat.Cluster(p.Formation, primary, enemyparty.Alive(p), false)
	hit := map[int]bool{}
	for _, e := range *stream {
		if e.Kind == combatstream.SpellHit && e.SpellId == "sparks" && e.Source.UserId == 7 {
			hit[e.Target.MobInstanceId] = true
		}
	}
	require.NotEmpty(t, hit, "the seeded cast landed")
	for _, id := range ids {
		assert.Equal(t, containsMember(allowed, mobparty.MemberKeyFor(id)), hit[id], "target %d", id)
	}

}

func containsMember(keys []domain.MemberKey, key domain.MemberKey) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func TestBattlefieldColdChantDelaysOnceWithNarration(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aimAt("bandit captain")
	b.aria.Character.Buffs.List = append(b.aria.Character.Buffs.List, &buffs.Buff{BuffId: 1011, TriggersLeft: 10})
	b.aria.Character.Buffs.Validate(true)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.aria.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{b.bandit("bandit captain").InstanceId}})
	require.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting)
	text := b.fight()
	assert.Contains(t, text, "numb fingers slow the chant")
	assert.Equal(t, 1, b.aria.Character.Aggro.RoundsWaiting)
	text = b.fight()
	assert.NotContains(t, text, "one extra round")
	assert.Equal(t, 0, b.aria.Character.Aggro.RoundsWaiting)
	assert.False(t, strings.Contains(text, "fumble the sling"))
}

func TestBattlefieldLeaperBypassesDownedButNotStandingFront(t *testing.T) {
	for _, down := range []bool{false, true} {
		t.Run(map[bool]string{false: "standing", true: "downed"}[down], func(t *testing.T) {
			b := battlefieldBrawl(t)
			loadStatusBuffs(t)
			rec, _ := module.registry.Get(7)
			rec.Formation = domain.Formation{{domain.CompanionMemberKey(3), domain.CompanionMemberKey(1), domain.CompanionMemberKey(4)}, {"", domain.LeaderMemberKey, ""}, {"", domain.CompanionMemberKey(2), ""}}
			module.registry.Put(rec)
			captain := b.bandit("bandit captain")
			captain.Leap = true
			b.aimAt("bandit captain")
			for _, m := range b.livingBandits() {
				m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
			}
			captain.Character.SetAggro(0, b.companion(2).InstanceId, characters.DefaultAttack, 0)
			if down {
				require.NoError(t, b.companion(1).Character.AddBuff(status.KnockedDown, false))
			}
			stream := b.listen()
			text := b.fight()
			rearHit := false
			for _, e := range *stream {
				if e.Kind == combatstream.Attack && e.Source.MobInstanceId == captain.InstanceId && e.Target.MobInstanceId == b.companion(2).InstanceId {
					rearHit = true
				}
			}
			assert.Equal(t, down, rearHit)
			if down {
				assert.Contains(t, text, "leaps past")
			}
		})
	}
}

func TestBattlefieldSweepStrikesFrontOnceAndHasCooldown(t *testing.T) {
	b := battlefieldBrawl(t)
	rec, _ := module.registry.Get(7)
	rec.Formation = domain.Formation{{domain.LeaderMemberKey, domain.CompanionMemberKey(1), ""}, {"", domain.CompanionMemberKey(3), ""}, {domain.CompanionMemberKey(2), domain.CompanionMemberKey(4), ""}}
	module.registry.Put(rec)
	captain := b.bandit("bandit captain")
	captain.Sweep = true
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	captain.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	stream := b.listen()
	assert.Contains(t, b.fight(), "sweeps a heavy blow")
	hits := map[string]int{}
	for _, e := range *stream {
		if e.Kind == combatstream.Attack && e.Source.MobInstanceId == captain.InstanceId {
			hits[e.Target.Key()]++
		}
	}
	assert.Equal(t, map[string]int{"u:7": 1, "m:" + strconv.Itoa(b.companion(1).InstanceId): 1}, hits)
	before := len(*stream)
	b.toughen()
	b.hold(nil)
	captain.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	b.fight()
	for _, e := range (*stream)[before:] {
		assert.False(t, e.Kind == combatstream.Ability && e.Source.MobInstanceId == captain.InstanceId && e.Status == "sweep", "cooldown prevents next-round sweep")
	}
}

func TestBattlefieldRealSpawnAndCampWatchConsumeDetectionOnce(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aria.Character.Stats.Perception.ValueAdj = 0
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Stats.Perception.ValueAdj = 0
	}
	t.Cleanup(enemyparty.UseAmbushRollForTest(func(int) int { return 99 }))
	first, err := enemyparty.SpawnAmbush(b.road.RoomId, 9106, 7)
	require.NoError(t, err)
	p, ok := enemyparty.PartyOf(b.road, first)
	require.True(t, ok)
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		assert.Equal(t, -1, m.AmbushAdvantage)
		assert.Equal(t, 7, m.AmbushOwner)
		m.Character.Health, m.Character.HealthMax.Value = 10000, 10000
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	enemyparty.WatchAmbush(b.road.RoomId, first, 7)
	for _, id := range p.Members {
		assert.Zero(t, mobs.GetInstance(id).AmbushAdvantage, "successful watch guarantees detection")
	}
	b.aria.Character.SetAggro(0, first, characters.DefaultAttack, 0)
	b.fight()
	bt, ok := battle.Current(7)
	require.True(t, ok)
	assert.Zero(t, bt.Opening)
	for _, id := range p.Members {
		assert.Zero(t, mobs.GetInstance(id).AmbushOwner)
	}
}

func TestBattlefieldEffectiveCellsReachPrivateGMCP(t *testing.T) {
	b := battlefieldBrawl(t)
	saved, _ := domain.FormationFor(7)
	b.road.Tags = append(b.road.Tags, "narrow")
	views := battleViews(t)
	b.aimAt("bandit captain")
	b.fight()
	b.refresh(7)
	view := lastView(views, 7)
	require.NotEmpty(t, view)
	assert.Equal(t, true, view["narrow"])
	positions, ok := view["positions"].(map[string]any)
	require.True(t, ok)
	require.Len(t, positions, 5)
	effective, _ := enemyparty.CompanyFormation(7)
	for key, cell := range positions {
		r, c, ok := effective.Find(domain.MemberKey(key))
		require.True(t, ok)
		assert.Equal(t, map[string]any{"row": float64(r), "col": float64(c)}, cell)
	}
	stored, _ := domain.FormationFor(7)
	assert.Equal(t, saved, stored)
	brom := users.NewUserRecord(8, 2)
	brom.Character.RoomId = b.road.RoomId
	users.SetTestUser(brom)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8); users.RemoveTestUser(8) })
	b.refresh(8)
	assert.Empty(t, lastView(views, 8), "a spectator receives no battle positions")
}

func TestBattlefieldSparksDoesNotRetargetHiddenAnchor(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.fight()
	captain := b.bandit("bandit captain")
	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93306, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93306) })
	b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack, 0)
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{captain.InstanceId, b.bandit("bandit bruiser").InstanceId}})
	require.NoError(t, captain.Character.AddBuff(93306, true))
	stream := b.listen()
	b.fight()
	for _, e := range *stream {
		assert.False(t, e.Kind == combatstream.SpellHit && e.SpellId == "sparks" && e.Source.UserId == 7, "hidden primary ends the spell")
	}
}

func TestBattlefieldEnemySparksHitsOnlyCompanyCluster(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.fight()
	captain := b.bandit("bandit captain")
	captain.Character.Stats.Mysticism.ValueAdj = 1000
	captain.Character.SpellBook = map[string]int{"sparks": 5000}
	captain.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetUserIds: []int{7}, TargetMobInstanceIds: []int{b.companion(1).InstanceId, b.companion(2).InstanceId, b.companion(3).InstanceId, b.companion(4).InstanceId}})
	stream := b.listen()
	b.fight()
	f, _ := enemyparty.CompanyFormation(7)
	allowed := formationcombat.Cluster(f, domain.LeaderMemberKey, enemyparty.CompanyAlive(7, f), false)
	hit := map[string]bool{}
	for _, e := range *stream {
		if e.Kind == combatstream.SpellHit && e.Source.MobInstanceId == captain.InstanceId && e.SpellId == "sparks" {
			hit[e.Target.Key()] = true
		}
	}
	require.NotEmpty(t, hit)
	for id := 1; id <= 4; id++ {
		assert.Equal(t, containsMember(allowed, domain.CompanionMemberKey(id)), hit["m:"+strconv.Itoa(b.companion(id).InstanceId)], "companion %d", id)
	}
	assert.True(t, hit["u:7"])
}

func TestBattlefieldOpeningDoesNotSuppressAnotherPlayersBattle(t *testing.T) {
	b := battlefieldBrawl(t)
	brom := users.NewUserRecord(8, 2)
	brom.Character.Name, brom.Character.RaceId, brom.Character.RoomId = "Brom", 1, b.road.RoomId
	brom.Character.Validate()
	brom.Character.HealthMax.Value, brom.Character.Health = 10000, 10000
	users.SetTestUser(brom)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8); users.RemoveTestUser(8) })
	first, err := enemyparty.SpawnAmbush(b.road.RoomId, 9106, 8)
	require.NoError(t, err)
	p, ok := enemyparty.PartyOf(b.road, first)
	require.True(t, ok)
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		m.Character.HealthMax.Value, m.Character.Health = 10000, 10000
		m.AmbushAdvantage = 0
		m.Character.SetAggro(8, 0, characters.DefaultAttack, 0)
	}
	brom.Character.SetAggro(0, first, characters.DefaultAttack, 0)
	for _, m := range b.livingBandits() {
		m.AmbushOwner, m.AmbushAdvantage = 7, -1
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.aimAt("bandit captain")
	stream := b.listen()
	b.fight()
	a, ok := battle.Current(7)
	require.True(t, ok)
	other, ok := battle.Current(8)
	require.True(t, ok)
	assert.Equal(t, -1, a.Opening)
	assert.Zero(t, other.Opening)
	attacked := false
	for _, e := range *stream {
		if e.Kind == combatstream.Attack {
			assert.NotEqual(t, 7, e.Source.UserId)
			if e.Source.UserId == 8 {
				attacked = true
			}
			if e.Source.MobInstanceId > 0 {
				assert.False(t, b.companyInstances()[e.Source.MobInstanceId])
			}
		}
	}
	assert.True(t, attacked, "the other player's round proceeds")
}

func TestBattlefieldSweepRespectsGuardian(t *testing.T) {
	b := battlefieldBrawl(t)
	rec, _ := module.registry.Get(7)
	rec.Formation = domain.Formation{{domain.LeaderMemberKey, domain.CompanionMemberKey(2), ""}, {domain.CompanionMemberKey(1), "", ""}, {domain.CompanionMemberKey(3), domain.CompanionMemberKey(4), ""}}
	module.registry.Put(rec)
	b.cmd("strategy", "tamsin guard me")
	captain := b.bandit("bandit captain")
	captain.Sweep = true
	b.aimAt("bandit captain")
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	captain.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	stream := b.listen()
	b.fight()
	guarded := false
	for _, e := range *stream {
		if e.Kind == combatstream.GuardUsed {
			guarded = true
		}
		if e.Kind == combatstream.Attack && e.Source.MobInstanceId == captain.InstanceId {
			assert.NotEqual(t, 7, e.Target.UserId, "guardian intercepts the leader's sweep strike")
		}
	}
	assert.True(t, guarded)
}

func TestBattlefieldColdSlingWaitsOncePerShot(t *testing.T) {
	b := battlefieldBrawl(t)
	b.aria.Character.Equipment.Weapon = items.New(10014)
	b.aria.Character.Buffs.List = append(b.aria.Character.Buffs.List, &buffs.Buff{BuffId: 1011, TriggersLeft: 100})
	b.aria.Character.Buffs.Validate(true)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(7, 0, characters.DefaultAttack, 20)
	}
	b.aria.Character.SetAggro(0, b.bandit("bandit captain").InstanceId, characters.DefaultAttack)
	require.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting)
	stream := b.listen()
	text := b.fight()
	assert.Contains(t, text, "numb fingers fumble the sling stone")
	assert.Equal(t, 1, b.aria.Character.Aggro.RoundsWaiting)
	text = b.fight()
	assert.NotContains(t, text, "one extra round")
	assert.Equal(t, 0, b.aria.Character.Aggro.RoundsWaiting)
	for _, e := range *stream {
		assert.False(t, e.Kind == combatstream.Attack && e.Source.UserId == 7)
	}
	b.fight()
	assert.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting, "next shot samples cold again")
	attacks := 0
	for _, e := range *stream {
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			attacks++
		}
	}
	assert.Equal(t, 1, attacks)
	b.aria.Character.Buffs.List = nil
	b.aria.Character.Buffs.Validate(true)
	assert.Equal(t, 2, b.aria.Character.Aggro.RoundsWaiting, "warming does not change a started load")
	b.fight()
	b.fight()
	b.fight()
	assert.Equal(t, 1, b.aria.Character.Aggro.RoundsWaiting, "next shot has normal load time")
}

func TestBattlefieldDisabledProtectorLeavesColumnOpen(t *testing.T) {
	b := battlefieldBrawl(t)
	buffs.SetTestFlag("no-combat")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93307, Name: "disabled protector", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"no-combat"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93307) })
	require.NoError(t, b.companion(1).Character.AddBuff(93307, true))
	f, _ := enemyparty.CompanyFormation(7)
	alive := enemyparty.CompanyAlive(7, f)
	standing := enemyparty.Standing(f, alive, 7)
	assert.False(t, standing[domain.CompanionMemberKey(1)])
	assert.True(t, standing[domain.CompanionMemberKey(3)])
	require.NoError(t, b.aria.Character.AddBuff(93307, true))
	standing = enemyparty.Standing(f, alive, 7)
	assert.False(t, standing[domain.LeaderMemberKey])
}
