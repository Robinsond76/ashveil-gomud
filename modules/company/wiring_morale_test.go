package company

import (
	"errors"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/configs"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Start a group, then kill its original leader through the resolved round.
func yieldingBrawl(t *testing.T) (*brawl, []int) {
	b := newBrawl(t)
	b.toughen()
	forceBlows(t, false)
	for _, m := range b.livingBandits() {
		m.Temperament = "craven"
	}
	t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return 50 }))
	b.aimAt("bandit captain")
	b.fight()
	b.companion(1).Character.Aggro = nil
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	require.NotNil(t, captain)
	captain.Character.Health = 0
	b.fight()
	var ids []int
	for _, m := range b.livingBandits() {
		require.True(t, m.Character.CombatWithdrawn)
		ids = append(ids, m.InstanceId)
	}
	require.Len(t, ids, 4)
	_, busy := battle.Current(7)
	require.False(t, busy, "only yielded enemies settles victory")
	return b, ids
}
func TestMoraleRoundYieldProtectionAndSpare(t *testing.T) {
	b, ids := yieldingBrawl(t)
	for _, id := range ids {
		m := mobs.GetInstance(id)
		hp := m.Character.Health
		assert.Zero(t, m.Character.ApplyHealthChange(-999))
		assert.Zero(t, combat.AttackPlayerVsMob(b.aria, m).DamageToTarget)
		assert.Zero(t, combat.AttackMobVsMob(b.companion(1), m).DamageToTarget)
		assert.Zero(t, combat.AttackMobVsPlayer(m, b.aria).DamageToTarget)
		a := scripting.GetActor(0, id)
		a.SetHealth(0)
		a.AddHealth(-999)
		assert.Equal(t, hp, m.Character.Health)
		m.Character.SetAggro(7, 0, characters.DefaultAttack)
		assert.Nil(t, m.Character.Aggro)
		_, err := mobcommands.TryCommand("attack", "Aria", id)
		require.NoError(t, err)
		assert.Nil(t, m.Character.Aggro)
		_, err = mobcommands.Suicide("quiet", m, b.road)
		require.NoError(t, err)
		assert.NotNil(t, mobs.GetInstance(id))
		assert.Empty(t, status.Tick(&m.Character))
		_, inParty := enemyparty.PartyOf(b.road, id)
		assert.False(t, inParty)
	}
	hooks.MercyTick(events.NewTurn{})
	p := b.aria.GetPrompt()
	require.NotNil(t, p)
	assert.Contains(t, p.Questions[0].Question, "first cutthroat") // stable ordinal or bandit name below
	token := p.Rest
	before := b.aria.Character.Alignment
	id := ids[0]
	// Prompt selects sorted instance order, as the test's ids may be map order.
	p.Questions[0].Answer("yes")
	b.cmd("mercy", token)
	assert.Equal(t, int(before)+5, int(b.aria.Character.Alignment))
	assert.Nil(t, b.aria.GetPrompt())
	require.NoError(t, morale.AnswerMercy(7, token, "yes"))
	assert.Equal(t, int(before)+5, int(b.aria.Character.Alignment))
	removed := 0
	for _, mid := range ids {
		if mobs.GetInstance(mid) == nil {
			removed++
		}
	}
	assert.Equal(t, 1, removed)
	_ = id
	hooks.MercyLeave(events.PlayerDespawn{UserId: 7})
	for _, mid := range ids {
		assert.Nil(t, mobs.GetInstance(mid))
	}
}
func TestMercyExpiryExistingPromptAndSaveRetry(t *testing.T) {
	for _, mode := range []string{"expiry", "existing", "retry"} {
		t.Run(mode, func(t *testing.T) {
			b := newBrawl(t)
			now := time.Unix(1000, 0)
			fail := false
			calls := 0
			t.Cleanup(hooks.UseMoraleStateForTest(func() time.Time { return now }, func(u *users.UserRecord) error {
				calls++
				if fail {
					return errors.New("disk full")
				}
				return nil
			}))
			b.toughen()
			forceBlows(t, false)
			for _, m := range b.livingBandits() {
				m.Temperament = "craven"
			}
			t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return 50 }))
			b.aimAt("bandit captain")
			b.fight()
			mobs.GetInstance(b.bandits["bandit captain"][0]).Character.Health = 0
			b.fight()
			if mode == "existing" {
				p, _ := b.aria.StartPrompt("bank", "withdraw")
				p.Ask("Amount?", nil)
				hooks.MercyTick(events.NewTurn{})
				assert.Same(t, p, b.aria.GetPrompt())
				assert.Empty(t, b.livingBandits())
				return
			}
			hooks.MercyTick(events.NewTurn{})
			p := b.aria.GetPrompt()
			require.NotNil(t, p)
			before := b.aria.Character.Alignment
			if mode == "expiry" {
				now = now.Add(31 * time.Second)
				hooks.MercyTick(events.NewTurn{})
				assert.Nil(t, b.aria.GetPrompt())
				assert.Len(t, b.livingBandits(), 3)
				assert.Equal(t, before, b.aria.Character.Alignment)
				hooks.MercyLeave(events.RoomChange{UserId: 7})
				assert.Empty(t, b.livingBandits())
				return
			}
			fail = true
			require.Error(t, morale.AnswerMercy(7, p.Rest, "yes"))
			assert.Equal(t, before, b.aria.Character.Alignment)
			fail = false
			hooks.MercyTick(events.NewTurn{})
			assert.Equal(t, int(before)+5, int(b.aria.Character.Alignment))
			assert.Equal(t, 2, calls)
		})
	}
}
func TestCompanyFlightSaveRestoreAndFailure(t *testing.T) {
	b := newBrawl(t)
	c := b.companion(1)
	hp, mana := c.Character.Health, c.Character.Mana
	gear := c.Character.GetAllWornItems()
	require.NoError(t, module.BeginFlight(7, 1))
	_, out := module.instance(7, 1)
	assert.False(t, out)
	r, _ := module.registry.Get(7)
	member, _ := findCompanion(r, 1)
	require.True(t, member.PendingReturn)
	data, err := yaml.Marshal(module.registry)
	require.NoError(t, err)
	var loaded domain.Registry
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	module.registry = loaded
	require.NoError(t, module.ReturnFlight(7))
	restored := b.companion(1)
	assert.Equal(t, hp, restored.Character.Health)
	assert.Equal(t, mana, restored.Character.Mana)
	gotGear := restored.Character.GetAllWornItems()
	require.Len(t, gotGear, len(gear))
	for i, itm := range gear {
		assert.Equal(t, itm.ItemId, gotGear[i].ItemId)
		assert.Equal(t, itm.Uses, gotGear[i].Uses)
		assert.Equal(t, itm.SharpStrikes, gotGear[i].SharpStrikes)
	}
	r, _ = module.registry.Get(7)
	member, _ = findCompanion(r, 1)
	assert.False(t, member.PendingReturn)
	loyalty := member.Disposition.Loyalty
	require.NoError(t, module.ReturnFlight(7))
	r, _ = module.registry.Get(7)
	member, _ = findCompanion(r, 1)
	assert.Equal(t, loyalty, member.Disposition.Loyalty)
}
func TestMercyReactionRollback(t *testing.T) {
	w := newFakeWorld()
	w.templates[58] = 50
	r := *domain.NewRegistry()
	r.Put(domain.Record{LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 50, 70)}})
	m, _ := newAlignmentModule(r, w)
	store := m.store.(*fakeStore)
	store.saveErr = errors.New("full")
	_, err := m.MercyReaction(7, "choice", []int{1}, true)
	require.Error(t, err)
	saved, _ := m.registry.Get(7)
	assert.Equal(t, 70, saved.Companions[0].Disposition.Loyalty)
	store.saveErr = nil
	lines, err := m.MercyReaction(7, "choice", []int{1}, true)
	require.NoError(t, err)
	assert.Len(t, lines, 1)
	saved, _ = m.registry.Get(7)
	assert.Equal(t, 72, saved.Companions[0].Disposition.Loyalty)
}

func TestNerveThroughRoundHesitationAndFlight(t *testing.T) {
	for _, mode := range []string{"hesitate", "flight"} {
		t.Run(mode, func(t *testing.T) {
			b := newBrawl(t)
			forceBlows(t, false)
			b.toughen()
			b.aimAt("bandit captain")
			b.fight()
			r, _ := module.registry.Get(7)
			for i := range r.Companions {
				a := module.companionAlignment(r.Companions[i])
				r.Companions[i].Disposition = &domain.Disposition{Alignment: a, Loyalty: 20}
			}
			module.registry.Put(r)
			hardMaxTo(b.aria.Character, 100)
			b.aria.Character.Health = 20
			for i := 1; i <= 4; i++ {
				m := b.companion(i)
				hardMaxTo(&m.Character, 100)
				m.Character.Health = 20
			}
			roll := 0
			if mode == "flight" {
				roll = 22
			}
			t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return roll }))
			if mode == "hesitate" {
				b.mobCasts(b.companion(2), "heal aria")
			}
			out := b.fight()
			if mode == "hesitate" {
				assert.Contains(t, out, "hesitates")
				assert.NotEqual(t, characters.SpellCast, b.companion(2).Character.Aggro.Type)
				assert.Equal(t, 98, b.companion(2).Character.Mana)
				out = b.fight()
				assert.NotContains(t, out, "hesitates")
			} else {
				assert.Contains(t, out, "loses nerve")
				assert.Empty(t, module.instances[7])
				r, _ = module.registry.Get(7)
				for _, c := range r.Companions {
					assert.True(t, c.PendingReturn)
				}
				b.aria.Character.RoomId = 920102
				b.fight()
				assert.Len(t, module.instances[7], 4)
				r, _ = module.registry.Get(7)
				for _, c := range r.Companions {
					assert.False(t, c.PendingReturn)
					assert.Equal(t, 15, c.Disposition.Loyalty)
				}
			}
		})
	}
}
func TestMercyExecutionAwardsAndDoesNotDoubleAlignment(t *testing.T) {
	b, ids := yieldingBrawl(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	for _, id := range ids {
		m := mobs.GetInstance(id)
		m.Character.TrackPlayerDamage(7, 1)
		m.Character.Gold = 10
	}
	beforeXP := b.aria.Character.Experience
	beforeAlign := b.aria.Character.Alignment
	beforeGold := b.aria.Character.Gold
	hooks.MercyTick(events.NewTurn{})
	p := b.aria.GetPrompt()
	require.NotNil(t, p)
	token := p.Rest
	p.Questions[0].Answer("no")
	b.cmd("mercy", token)
	assert.Greater(t, b.aria.Character.Experience, beforeXP)
	assert.Equal(t, int(beforeAlign)-5, int(b.aria.Character.Alignment))
	b.cmd("loot", "")
	assert.Greater(t, b.aria.Character.Gold, beforeGold)
	xp := b.aria.Character.Experience
	gold := b.road.Gold
	require.NoError(t, morale.AnswerMercy(7, token, "no"))
	assert.Equal(t, xp, b.aria.Character.Experience)
	assert.Equal(t, gold, b.road.Gold)
}
func TestFlightFailedSaveStaysAndReturnRetryChargesOnce(t *testing.T) {
	b := newBrawl(t)
	oldStore := module.store
	store := &fakeStore{}
	module.store = store
	t.Cleanup(func() { module.store = oldStore })
	m := b.companion(1)
	id := m.InstanceId
	store.saveErr = errors.New("disk full")
	require.Error(t, module.BeginFlight(7, 1))
	assert.Same(t, m, mobs.GetInstance(id))
	r, _ := module.registry.Get(7)
	c, _ := findCompanion(r, 1)
	assert.False(t, c.PendingReturn)
	store.saveErr = nil
	require.NoError(t, module.BeginFlight(7, 1))
	r, _ = module.registry.Get(7)
	c, _ = findCompanion(r, 1)
	loyalty := c.Disposition.Loyalty
	store.saveErr = errors.New("disk full")
	require.Error(t, module.ReturnFlight(7))
	r, _ = module.registry.Get(7)
	c, _ = findCompanion(r, 1)
	assert.True(t, c.PendingReturn)
	assert.Equal(t, loyalty, c.Disposition.Loyalty)
	store.saveErr = nil
	require.NoError(t, module.ReturnFlight(7))
	require.NoError(t, module.ReturnFlight(7))
	r, _ = module.registry.Get(7)
	c, _ = findCompanion(r, 1)
	assert.Equal(t, loyalty-5, c.Disposition.Loyalty)
}

func TestFailedMercyReleasesOnLeaveAndRecoversEffects(t *testing.T) {
	for _, leave := range []string{"logout", "move", "purge"} {
		t.Run(leave, func(t *testing.T) {
			b, ids := yieldingBrawl(t)
			fail := true
			// Use a save function without replacing the established transient encounter.
			restore := hooks.SetMercySaveForTest(func(u *users.UserRecord) error {
				if fail {
					return errors.New("disk full")
				}
				return users.SaveUser(*u)
			})
			t.Cleanup(restore)
			hooks.MercyTick(events.NewTurn{})
			p := b.aria.GetPrompt()
			require.NotNil(t, p)
			before := b.aria.Character.Alignment
			require.Error(t, morale.AnswerMercy(7, p.Rest, "yes"))
			switch leave {
			case "logout":
				hooks.MercyLeave(events.PlayerDespawn{UserId: 7})
			case "move":
				hooks.MercyLeave(events.RoomChange{UserId: 7})
			case "purge":
				hooks.MercyLeave(events.UserPurged{UserId: 7})
				module.registry.Remove(7)
			}
			for _, id := range ids {
				assert.Nil(t, mobs.GetInstance(id), "no immortal prisoners after save failure")
			}
			fail = false
			hooks.MercyTick(events.NewTurn{})
			if leave == "purge" {
				assert.Equal(t, before, b.aria.Character.Alignment)
			} else {
				assert.Equal(t, int(before)+5, int(b.aria.Character.Alignment))
				hooks.MercyTick(events.NewTurn{})
				assert.Equal(t, int(before)+5, int(b.aria.Character.Alignment))
			}
		})
	}
}
func TestMercyPendingRecoveryDoesNotReplayRewards(t *testing.T) {
	b := newBrawl(t)
	r, _ := module.registry.Get(7)
	loyalty := r.Companions[0].Disposition.Loyalty
	_, err := module.MercyReaction(7, "saved-choice", []int{1}, true)
	require.NoError(t, err)
	data, err := yaml.Marshal(module.registry)
	require.NoError(t, err)
	var recovered domain.Registry
	require.NoError(t, decodeCompanies(data, &recovered))
	module.registry = recovered
	alignment, xp, gold := b.aria.Character.Alignment, b.aria.Character.Experience, b.road.Gold
	hooks.MercyTick(events.NewTurn{})
	assert.Equal(t, int(alignment)+5, int(b.aria.Character.Alignment))
	assert.Equal(t, xp, b.aria.Character.Experience)
	assert.Equal(t, gold, b.road.Gold)
	hooks.MercyTick(events.NewTurn{})
	r, _ = module.registry.Get(7)
	assert.Empty(t, r.MercyPending)
	assert.Equal(t, int(alignment)+5, int(b.aria.Character.Alignment))
	assert.GreaterOrEqual(t, r.Companions[0].Disposition.Loyalty, loyalty)
}
func TestMercyAtZeroLoyaltyDesertsAndRetryDoesNotRecover(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	w := newFakeWorld()
	r := *domain.NewRegistry()
	r.Put(domain.Record{LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 25, 1)}})
	m, _ := newAlignmentModule(r, w)
	_, err := m.MercyReaction(7, "execute", []int{1}, false)
	require.NoError(t, err)
	saved, _ := m.registry.Get(7)
	assert.Empty(t, saved.Companions)
}
func TestSharedMoraleOwnerDepartureAndSameRoundTie(t *testing.T) {
	b, other, _ := alliedBrawl(t)
	forceBlows(t, false)
	b.toughen()
	hardTo(other.Character, 10000)
	for _, m := range b.livingBandits() {
		m.Temperament = "craven"
	}
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	other.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
	b.aimAt("bandit captain")
	b.fight()
	first, ok := battle.Current(7)
	require.True(t, ok)
	second, ok := battle.Current(8)
	require.True(t, ok)
	assert.Less(t, first.FightID, second.FightID, "lower player ID registers first")
	b.aria.Character.RoomId = 920102
	b.fight()
	t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return 50 }))
	captain.Character.Health = 0
	b.fight()
	for _, m := range b.livingBandits() {
		assert.True(t, m.Character.CombatWithdrawn)
	}
	hooks.MercyTick(events.NewTurn{})
	assert.Nil(t, b.aria.GetPrompt())
	require.NotNil(t, other.GetPrompt())
	require.NoError(t, morale.AnswerMercy(8, other.GetPrompt().Rest, "yes"))
}
func TestYieldDoesNotHoldWaitingBattle(t *testing.T) {
	b := newBrawl(t)
	forceBlows(t, false)
	b.toughen()
	for _, m := range b.livingBandits() {
		m.Temperament = "craven"
	}
	t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return 50 }))
	b.aimAt("bandit captain")
	b.fight()
	next := mobs.NewMobById(9106, b.road.RoomId)
	require.NotNil(t, next)
	next.SpawnGroup = "next"
	b.road.AddMob(next.InstanceId)
	next.Character.SetAggro(7, 0, characters.DefaultAttack)
	mobs.GetInstance(b.bandits["bandit captain"][0]).Character.Health = 0
	b.fight()
	current, ok := battle.Current(7)
	require.True(t, ok)
	assert.True(t, current.Has(next.InstanceId))
	hooks.MercyTick(events.NewTurn{})
	assert.Nil(t, b.aria.GetPrompt())
	for _, ids := range b.bandits {
		for _, id := range ids {
			assert.Nil(t, mobs.GetInstance(id))
		}
	}
}

func TestMercyWaitsForPacedYieldAndSummary(t *testing.T) {
	b := newBrawl(t)
	b.toughen()
	forceBlows(t, false)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	t.Cleanup(hooks.UseMoraleStateForTest(func() time.Time { return now }, nil))
	t.Cleanup(hooks.UseMoraleRollForTest(func(int) int { return 50 }))
	var sent []string
	t.Cleanup(hooks.SetWriteTextForTest(func(uid int, text string) {
		if uid == 7 {
			sent = append(sent, text)
		}
	}))
	freshEvents(t)
	id := events.RegisterListener(events.Message{}, hooks.Message_SendMessage)
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	for _, m := range b.livingBandits() {
		m.Temperament = "craven"
	}
	b.aimAt("bandit captain")
	events.WithCause(100, func() { b.fight() })
	now = now.Add(10 * time.Second)
	hooks.ReleasePacedCombat(events.NewTurn{})
	events.ProcessEvents()
	sent = nil
	mobs.GetInstance(b.bandits["bandit captain"][0]).Character.Health = 0
	events.WithCause(101, func() { b.fight() })
	require.True(t, combatpace.Default().Busy(7))
	hooks.MercyTick(events.NewTurn{})
	require.Empty(t, b.aria.GetPrompt())
	for i := 0; i < 200; i++ {
		now = now.Add(50 * time.Millisecond)
		hooks.ReleasePacedCombat(events.NewTurn{})
		events.ProcessEvents()
	}
	require.False(t, combatpace.Default().Busy(7))
	require.NotEmpty(t, sent)
	hooks.MercyTick(events.NewTurn{})
	require.NotEmpty(t, b.aria.GetPrompt())
}

func TestOfflineMercyRecoveryPreservesLaterSession(t *testing.T) {
	b, ids := yieldingBrawl(t)
	fail := true
	t.Cleanup(hooks.SetMercySaveForTest(func(u *users.UserRecord) error {
		if fail {
			return errors.New("disk full")
		}
		return users.SaveUser(*u)
	}))
	hooks.MercyTick(events.NewTurn{})
	p := b.aria.GetPrompt()
	require.NotNil(t, p)
	before := b.aria.Character.Alignment
	require.Error(t, morale.AnswerMercy(7, p.Rest, "yes"))
	hooks.MercyLeave(events.PlayerDespawn{UserId: 7})
	for _, id := range ids {
		require.Nil(t, mobs.GetInstance(id))
	}
	require.NoError(t, users.SaveUser(*b.aria))
	users.ResetActiveUsers()
	later, err := users.LoadUserFile(7)
	require.NoError(t, err)
	require.NotSame(t, b.aria, later)
	users.SetTestUser(later)
	hooks.MercyTick(events.NewTurn{}) // retry still fails in this fresh session
	later.Character.Gold = 777
	later.Character.Experience = 12345
	require.NoError(t, users.SaveUser(*later))
	users.ResetActiveUsers()
	fail = false
	hooks.MercyTick(events.NewTurn{}) // offline recovery must load latest save
	saved, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, 777, saved.Character.Gold)
	assert.Equal(t, 12345, saved.Character.Experience)
	assert.Equal(t, int(before)+5, int(saved.Character.Alignment))
	hooks.MercyTick(events.NewTurn{})
	saved, err = users.LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, int(before)+5, int(saved.Character.Alignment))
}

// Phase 64 review: a mercy answer whose company save failed is retried after
// the leader walks away, when the queue no longer names the foe. The retry
// must not index the empty queue for the opinion's subject.
func TestFailedMercyRetryAfterLeavingDoesNotPanic(t *testing.T) {
	b, _ := yieldingBrawl(t)
	failing := &failingStore{Store: module.store, err: errors.New("disk full")}
	module.store = failing
	t.Cleanup(func() { module.store = failing.Store })
	hooks.MercyTick(events.NewTurn{})
	p := b.aria.GetPrompt()
	require.NotNil(t, p)
	require.Error(t, morale.AnswerMercy(7, p.Rest, "no"))
	hooks.MercyLeave(events.RoomChange{UserId: 7})
	failing.err = nil
	assert.NotPanics(t, func() { hooks.MercyTick(events.NewTurn{}) })
}
