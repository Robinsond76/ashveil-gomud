package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40g2: allied companies in the battle feed and the Company.Battle
// payload, and the combat pace on each batch.

var allyLeader = combatstream.Ref{UserId: 8, Name: "Brannoc", LeaderUserId: 8, MemberKey: "leader"}
var allyCompanion = combatstream.Ref{MobInstanceId: 601, Name: "Cael", LeaderUserId: 8, MemberKey: "companion:2"}

// alliedRig adds an allied leader (8) in the rig's room, consenting
// with the rig's player (7), each in a battle of their own against the
// same bandit (88). It returns the ally's fight id.
type alliedRig struct {
	*eventRig
	ally    *users.UserRecord
	ownID   uint64
	allyID  uint64
	partyOf *parties.Party
}

func newAlliedRig(t *testing.T) *alliedRig {
	r := newEventRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	t.Cleanup(parties.UseMemoryForTest())
	battle.Reset()
	t.Cleanup(battle.Reset)

	ally := users.NewUserRecord(8, 81)
	ally.Character.Name = "Brannoc"
	users.SetTestUser(ally)
	r.user.Character.RoomId = 100
	ally.Character.RoomId = 100

	p := parties.New(8)
	require.NotNil(t, p)
	require.True(t, p.InvitePlayer(7))
	require.True(t, p.AcceptInvite(7))
	p.SetSupport(7, true)
	p.SetSupport(8, true)

	a := &alliedRig{eventRig: r, ally: ally, partyOf: p}
	a.ownID = r.stream.Open(341, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	a.allyID = r.stream.Open(341, 100, "party-a", allyLeader, []combatstream.Ref{allyCompanion}, []combatstream.Ref{rigEnemy(88)})
	battle.Begin(7, 100, 341, "party-a", []int{88})
	battle.SetFight(7, a.ownID)
	battle.Begin(8, 100, 341, "party-a", []int{88})
	battle.SetFight(8, a.allyID)
	events.ProcessEvents()
	r.got = nil
	return a
}

// TestAlliedHappeningsReachTheirAllyWithAllyIDs: what an allied company
// does in its own fight is sent to its consenting ally fighting the same
// enemy, under "a:<leader>:<key>" ids, without the numbers of what befell
// its members or the status a blow left, and without the ally's fight
// opening or ending.
func TestAlliedHappeningsReachTheirAllyWithAllyIDs(t *testing.T) {
	a := newAlliedRig(t)
	events.WithCause(341, func() {
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: a.allyID, Source: allyLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 4, WeaponType: "slashing"})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: a.allyID, Source: allyLeader, Target: rigEnemy(88), Status: "Bleeding"})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: a.allyID, Source: rigEnemy(88), Target: allyCompanion, Outcome: combatstream.OutcomeHit, Damage: 7, Status: "Poisoned"})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: a.allyID, Source: rigEnemy(88), Target: allyCompanion, Status: "Poisoned"})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Heal, FightID: a.allyID, Source: allyLeader, Target: allyCompanion, SpellId: "mend", Amount: 5, HeldBack: 1})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.WoundChange, FightID: a.allyID, Target: allyCompanion})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Death, FightID: a.allyID, Target: allyCompanion, Outcome: combatstream.OutcomeIncapacitated})
	})
	events.ProcessEvents()
	a.stream.EndFight(a.allyID, 342, combatstream.OutcomeDefeat, combatstream.Final{})
	events.ProcessEvents()

	assert.Equal(t, []string{
		"attack:a:8:leader>m:88",
		"status-applied:a:8:leader>m:88",
		"attack:m:88>a:8:companion:2",
		"heal:a:8:leader>a:8:companion:2",
		"death:m:88>a:8:companion:2",
	}, a.kinds(), "no status on an ally, no wound change, no fight start or end")

	ps := a.payloads()
	require.NotEmpty(t, ps)
	var hit, blow, heal battleEvent
	for _, p := range ps {
		assert.Equal(t, a.ownID, p.Fight, "the receiver's own fight")
		assert.Equal(t, string(combatpace.Off), p.Pace)
		for _, e := range p.Events {
			switch {
			case e.Kind == "attack" && e.Tgt == "m:88":
				hit = e
			case e.Kind == "attack":
				blow = e
			case e.Kind == "heal":
				heal = e
			}
		}
	}
	assert.Equal(t, 4, hit.Damage, "an ally's blow on the shared enemy is shown")
	assert.Equal(t, "slashing", hit.Weapon)
	assert.Zero(t, blow.Damage, "no number for what befell an ally")
	assert.Empty(t, blow.Status)
	assert.Zero(t, heal.Amount)
	assert.Zero(t, heal.HeldBack)
	raw, err := json.Marshal(ps)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "Poisoned")
}

// TestAlliedHappeningsNeedConsentSharedEnemyAndRoom: an unconsenting
// party member, an ally in another room, and one against other mobs see
// nothing of the ally's fight.
func TestAlliedHappeningsNeedConsentSharedEnemyAndRoom(t *testing.T) {
	strike := func(a *alliedRig) {
		events.WithCause(341, func() {
			a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: a.allyID, Source: allyLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 4})
		})
		events.ProcessEvents()
	}

	t.Run("no consent from the receiver", func(t *testing.T) {
		a := newAlliedRig(t)
		a.partyOf.SetSupport(7, false)
		strike(a)
		assert.Empty(t, a.payloads())
	})
	t.Run("a receiver elsewhere", func(t *testing.T) {
		a := newAlliedRig(t)
		battle.Begin(7, 101, 341, "party-a", []int{88})
		battle.SetFight(7, a.ownID)
		strike(a)
		assert.Empty(t, a.payloads())
	})
	t.Run("a different enemy", func(t *testing.T) {
		a := newAlliedRig(t)
		battle.Begin(7, 100, 341, "party-a", []int{99})
		battle.SetFight(7, a.ownID)
		strike(a)
		assert.Empty(t, a.payloads())
	})
	t.Run("a receiver in no battle", func(t *testing.T) {
		a := newAlliedRig(t)
		battle.End(7)
		strike(a)
		assert.Empty(t, a.payloads())
	})
	t.Run("with all of it, the blow is sent", func(t *testing.T) {
		a := newAlliedRig(t)
		strike(a)
		assert.Len(t, a.payloads(), 1)
	})
}

// TestAlliedHappeningsMaskTheUnseen: an enemy the receiver can't make out
// is still "?" in an ally's happenings.
func TestAlliedHappeningsMaskTheUnseen(t *testing.T) {
	a := newAlliedRig(t)
	battleSeen.note(7, battleSeenKey{start: 341, fight: a.ownID, party: "party-a"}, 88, true)
	t.Cleanup(func() { battleSeen.forget(7) })
	events.WithCause(341, func() {
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: a.allyID, Source: allyLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 4})
		a.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: a.allyID, Source: allyLeader, Target: rigEnemy(88), Status: "Bleeding"})
	})
	events.ProcessEvents()
	assert.Equal(t, []string{"attack:a:8:leader>?"}, a.kinds())
}

// TestBattlePayloadCarriesAlliesAndNerve: allied companies list with their
// members' ids, cells and health in words, and the company's faltering
// nerve is named only while it holds.
func TestBattlePayloadCarriesAlliesAndNerve(t *testing.T) {
	f := sampleBattle()
	f.Faltering = true
	f.Allies = []allyFact{{Leader: 8, Name: "Brannoc", Members: []allyMemberFact{
		{Key: "leader", Name: "Brannoc", Class: "warrior", Row: 0, Col: 1, Health: 20, HealthMax: 20},
		{Key: "companion:2", Name: "Cael", Class: "cleric", Row: 1, Col: 2, Health: 4, HealthMax: 20},
		{Key: "companion:3", Name: "Dun", Row: 2, Col: 0, Health: 0, HealthMax: 20, Down: true},
	}}}
	raw, err := json.Marshal(buildBattle(f))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "faltering", got["nerve"])
	allies := got["allies"].([]any)
	require.Len(t, allies, 1)
	al := allies[0].(map[string]any)
	assert.Equal(t, "a:8", al["id"])
	assert.Equal(t, "Brannoc", al["name"])
	members := al["members"].([]any)
	require.Len(t, members, 3)
	assert.Equal(t, map[string]any{"id": "a:8:leader", "name": "Brannoc", "class": "warrior", "cell": map[string]any{"row": 0.0, "col": 1.0}, "health": "unhurt"}, members[0])
	assert.Equal(t, "near death", members[1].(map[string]any)["health"])
	assert.Equal(t, true, members[2].(map[string]any)["down"])
	assert.NotContains(t, string(raw), "health_max")

	f.Allies, f.Faltering = nil, false
	raw, err = json.Marshal(buildBattle(f))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "allies")
	assert.NotContains(t, string(raw), "nerve")
}

// TestAlliesOfListsOnlyConsentingAlliesInTheSameBattle covers the gather's
// choice of allies against real parties and battles.
func TestAlliesOfListsOnlyConsentingAlliesInTheSameBattle(t *testing.T) {
	a := newAlliedRig(t)
	b, _ := battle.Current(7)
	got := alliesOf(a.user, b)
	require.Len(t, got, 1)
	assert.Equal(t, 8, got[0].UserId)

	a.partyOf.SetSupport(8, false)
	assert.Empty(t, alliesOf(a.user, b), "the ally withdrew consent")
	a.partyOf.SetSupport(8, true)
	battle.End(8)
	assert.Empty(t, alliesOf(a.user, b), "the ally's battle is over")
}

// TestCompanyFalteringFollowsTheNerveRule: the company is faltering when
// half of it is down or a quarter of its health is left, as the nerve check
// reads it.
func TestCompanyFalteringFollowsTheNerveRule(t *testing.T) {
	a := newAlliedRig(t)
	a.user.Character.HealthMax.Value = 100
	a.user.Character.Health = 100
	b, _ := battle.Current(7)
	assert.False(t, companyFaltering(b))
	a.user.Character.Health = 24
	assert.True(t, companyFaltering(b), "a quarter of health left")
	a.user.Character.Health = 60
	assert.False(t, companyFaltering(b))
	a.user.Character.Health = 0
	assert.True(t, companyFaltering(b), "the whole company is down")
	assert.False(t, companyFaltering(battle.Battle{}), "no fight")
}

// TestBattleEventPayloadCarriesThePace: the receiver's pace rides every
// batch, so the screen's animation follows the setting.
func TestBattleEventPayloadCarriesThePace(t *testing.T) {
	r := newEventRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	id := r.stream.Open(1, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	r.stream.Emit(combatstream.Event{Round: 1, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 3})
	events.ProcessEvents()
	require.NotEmpty(t, r.payloads())
	assert.Equal(t, "off", r.payloads()[0].Pace)

	// A paced player's batches carry their pace when released.
	r2 := newEventRig(t)
	r2.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Slow))
	id = r2.stream.Open(1, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	events.ProcessEvents()
	r2.got = nil
	events.WithCause(2, func() {
		r2.user.SendText("Aria swings.")
		r2.stream.Emit(combatstream.Event{Round: 2, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 3})
		r2.user.SendText("The bandit is cut. (3)")
	})
	events.ProcessEvents()
	r2.turns(400)
	require.NotEmpty(t, r2.payloads())
	assert.Equal(t, "slow", r2.payloads()[0].Pace)
}
