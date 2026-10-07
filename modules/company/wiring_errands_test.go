package company

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/errands"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 70 wiring: the brawl world with the road made an inn in a 7-9 zone,
// and the real commands, round tick, registry, store and runtime.
func errandBrawl(t *testing.T) (*brawl, *time.Time) {
	t.Helper()
	b := newBrawl(t)
	b.road.Tags = append(b.road.Tags, errandRoomTag)
	t.Cleanup(func() { b.road.Tags = nil })
	zone := rooms.GetZoneConfig("brawl")
	require.NotNil(t, zone)
	oldBand := zone.Encounters.Band
	zone.Encounters.Band.Low, zone.Encounters.Band.High = 7, 9
	t.Cleanup(func() { zone.Encounters.Band = oldBand })
	oldSave := module.saveUser
	module.saveUser = func(*users.UserRecord) error { return nil }
	t.Cleanup(func() { module.saveUser = oldSave })
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	now := time.Unix(2_000_000, 0)
	module.clock = func() time.Time { return now }
	t.Cleanup(func() { module.clock = nil })
	return b, &now
}

func TestErrandThroughTheRealCommandsAndRoundTick(t *testing.T) {
	b, now := errandBrawl(t)
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	ysolde := b.companion(4)

	// the help in the command path
	assert.Contains(t, b.cmd("errands", ""), "free to send")

	out := b.cmd("errand", "send #4 hunt medium")
	assert.Contains(t, out, "sets out hunting for brawl")
	assert.NotContains(t, b.road.GetMobs(rooms.FindCharmed), ysolde.InstanceId, "off the map")
	_, tracked := module.instance(7, 4)
	assert.False(t, tracked)
	e := errandOf(t, module, 4)
	require.NotNil(t, e)
	assert.Equal(t, now.Unix()+2*3600, e.ReturnsAt)
	assert.Equal(t, 7, e.BandLow)

	// absent where it should be
	assert.Contains(t, b.cmd("company", "status"), "away hunting; back in about 2 hours")
	assert.Contains(t, b.cmd("errands", ""), "away on a hunt in brawl")
	_, err := usercommands.TryCommand("formation", "move #4 2 2", b.aria.UserId, events.CmdSkipScripts)
	assert.ErrorIs(t, err, domain.ErrMemberAway, "a companion who is away cannot be placed")
	assert.Contains(t, b.cmd("errand", "send #4 scout short"), "already on an errand")
	assert.NotContains(t, b.cmd("company", "inventory"), "Ysolde:\n", "the load leaves with it")
	assert.NotContains(t, module.CompanionsWithLeader(7), 4, "it is not with the leader")

	// durable: a restart (the file is read again) and a relog bring nobody back
	module.load()
	require.NoError(t, module.loadErr)
	require.NotNil(t, errandOf(t, module, 4), "the errand survives a restart")
	require.NoError(t, module.restoreForLeader(7, b.road.RoomId))
	_, tracked = module.instance(7, 4)
	assert.False(t, tracked, "a login never brings it back early")

	// not due: rounds change nothing
	b.rounds(3)
	require.NotNil(t, errandOf(t, module, 4))

	// real time passes while the leader is in a fight: it waits
	*now = now.Add(2*time.Hour + time.Minute)
	bandit := b.bandits["bandit cutthroat"][0]
	b.aria.Character.Aggro = &charactersAggro{MobInstanceId: bandit}
	b.rounds(1)
	require.NotNil(t, errandOf(t, module, 4), "due, but the leader is fighting")
	b.aria.Character.Aggro = nil

	// the roll is fixed at the send; force a gold one so the payout is exact
	forceOutcome(t, 4, errands.Gold)
	e = errandOf(t, module, 4)
	gold := b.aria.Character.Gold
	told := b.rounds(1)

	assert.Nil(t, errandOf(t, module, 4), "home")
	back := b.companion(4)
	assert.Equal(t, b.road.RoomId, back.Character.RoomId, "beside the leader")
	assert.Contains(t, told, "is back from a hunt in brawl with")
	assert.Equal(t, gold+errands.Pay(*e), b.aria.Character.Gold)
	deeds := chronicle.Query(7, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Errand}})
	require.Len(t, deeds, 1)
	assert.Contains(t, b.cmd("errands", ""), "free to send")
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

// charactersAggro keeps the import list short.
type charactersAggro = characters.Aggro

func forceOutcome(t *testing.T, id int, want errands.OutcomeKind) {
	t.Helper()
	record, _ := module.registry.Get(7)
	for i := range record.Companions {
		if record.Companions[i].ID == id && record.Companions[i].Errand != nil {
			e := *record.Companions[i].Errand
			e.Seed = seedFor(t, e, want, false)
			record.Companions[i].Errand = &e
		}
	}
	module.registry.Put(record)
}

func TestErrandsAreRefusedAwayFromAnInnOrInAFight(t *testing.T) {
	b, _ := errandBrawl(t)

	b.road.Tags = nil
	assert.Contains(t, b.cmd("errand", "send #2 escort"), "from an inn")
	assert.Nil(t, errandOf(t, module, 2))
	assert.Contains(t, b.cmd("errands", ""), "from an inn")

	b.road.Tags = []string{errandRoomTag}
	bandit := b.bandits["bandit cutthroat"][0]
	b.aria.Character.Aggro = &charactersAggro{MobInstanceId: bandit}
	assert.Contains(t, b.cmd("errand", "send #2 escort"), "Not while you are fighting")
	b.aria.Character.Aggro = nil
	assert.Nil(t, errandOf(t, module, 2))

	// a companion standing in the formation can go, and leaves it
	b.cmd("formation", "move #2 3 3")
	r, _ := module.registry.Get(7)
	require.Equal(t, domain.CompanionMemberKey(2), r.Formation.At(2, 2))
	assert.Contains(t, b.cmd("errand", "send #2 escort"), "sets out escorting")
	assert.Contains(t, b.cmd("formation", ""), "away on an errand")
}

func TestRecallAndDismissWhileAway(t *testing.T) {
	b, _ := errandBrawl(t)
	require.Contains(t, b.cmd("errand", "send #3 scout long"), "sets out scouting")
	require.Contains(t, b.cmd("errand", "send #4 escort short"), "sets out escorting")

	out := b.cmd("errand", "recall #3")
	assert.Contains(t, out, "returns with nothing")
	assert.Nil(t, errandOf(t, module, 3))
	assert.Equal(t, b.road.RoomId, b.companion(3).Character.RoomId)
	assert.Contains(t, b.cmd("errand", "recall #3"), "not away on an errand")

	assert.Contains(t, b.cmd("company", "dismiss #4"), "Companion dismissed")
	record, _ := module.registry.Get(7)
	_, found := findCompanion(record, 4)
	assert.False(t, found, "dismissed while away: the errand goes with the record")
	events.ProcessEvents()
}
