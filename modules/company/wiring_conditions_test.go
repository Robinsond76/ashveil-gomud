package company

import (
	"encoding/json"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOwnedConditionsThroughRefreshExpiryAndReconnect(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())
	mob := b.companion(1)
	count := 0
	var received map[string]struct {
		State   string                                         `json:"state"`
		Effects []struct{ Name, Description, Duration string } `json:"effects"`
		Wounds  []struct{ Name, Description, Duration string } `json:"wounds"`
	}
	listener := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module == "Company.Conditions" {
			assert.Equal(t, 7, out.UserId)
			require.NoError(t, json.Unmarshal(out.Payload.([]byte), &received))
			count++
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, listener) })
	refresh := func() { companyview.RefreshUser(7); events.ProcessEvents() }
	mob.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "ribs", Points: 2, Light: true}}
	require.NoError(t, mob.Character.AddBuff(status.Stunned, false))
	refresh()
	require.Len(t, received["companion:1"].Effects, 1)
	assert.Contains(t, received["companion:1"].Effects[0].Duration, "combat rounds remaining")
	require.Len(t, received["companion:1"].Wounds, 2)
	previous := count
	refresh()
	assert.Equal(t, previous, count, "unchanged conditions suppressed")
	mob.Character.RemoveBuff(status.Stunned)
	refresh()
	assert.Empty(t, received["companion:1"].Effects)
	require.NoError(t, mob.Character.AddBuff(status.Stunned, false))
	refresh()
	for i := 0; i < 3; i++ {
		status.Tick(&mob.Character)
	}
	refresh()
	assert.Empty(t, received["companion:1"].Effects, "expired before ordinary pruning is omitted")
	mob.Character.Wounds = wounds.CloseLight(mob.Character.Wounds)
	refresh()
	require.Len(t, received["companion:1"].Wounds, 1)
	originalRoom := mob.Character.RoomId
	t.Cleanup(func() { mob.Character.RoomId = originalRoom })
	mob.Character.RoomId = b.aria.Character.RoomId + 1
	refresh()
	assert.Equal(t, "away-live", received["companion:1"].State)
	mob.Character.Charm(999, characters.CharmPermanent, "")
	refresh()
	assert.Equal(t, "unavailable", received["companion:1"].State)
	assert.Empty(t, received["companion:1"].Wounds, "another owner's private state is never read")
	assert.Empty(t, received["companion:1"].Effects)
	foreign, known := module.CompanyConditions(999)
	require.True(t, known)
	assert.Empty(t, foreign, "a foreign requester cannot read this company's roster")
	previous = count
	events.AddToQueue(gmcp.GMCPCompanyRequest{UserId: 7})
	events.ProcessEvents()
	assert.Greater(t, count, previous)
}

func TestConditionsSavedWoundsAndFallenMembers(t *testing.T) {
	saved := &domain.MemberState{Wounds: []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "side", Points: 2, Light: true}}}
	m := newTestModule(domain.Registry{Companies: map[int]domain.Record{7: {LeaderUserID: 7, Companions: []domain.Companion{
		{ID: 1, MobTemplateID: 58, State: saved},
		{ID: 2, MobTemplateID: 58, State: saved, Death: &domain.CompanionDeath{Remaining: 30}},
	}}}}, &fakeRuntime{})
	views, ok := m.CompanyConditions(7)
	require.True(t, ok)
	require.Len(t, views, 2)
	assert.Equal(t, "away", views[0].State)
	require.Len(t, views[0].Wounds, 1)
	assert.False(t, views[0].Wounds[0].Light)
	assert.Equal(t, "dead", views[1].State)
	assert.Empty(t, views[1].Wounds)
	assert.Empty(t, views[1].Buffs)
	m.loadErr = assert.AnError
	_, ok = m.CompanyConditions(7)
	assert.False(t, ok)
}

// Integration with shipped 33h3: a passage persists wounds when separating
// the absent, and rejoin returns live owned state through the same GMCP seam.
func TestSeparatedConditionsFollowRealRelocationAndRejoin(t *testing.T) {
	b := relocationBrawl(t)
	loadStatusBuffs(t)
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())
	mob := b.companion(4)
	mob.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "side", Points: 2, Light: true}}
	require.NoError(t, mob.Character.AddBuff(status.Stunned, false))
	require.True(t, nativeRuntime{}.Relocate(mob.InstanceId, 920103))
	var received map[string]struct {
		State   string
		Wounds  []struct{ Name, Duration string }
		Effects []struct{ Name string }
	}
	listener := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module == "Company.Conditions" {
			require.NoError(t, json.Unmarshal(out.Payload.([]byte), &received))
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, listener) })
	b.pull()
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, "separated", received["companion:4"].State)
	require.Len(t, received["companion:4"].Wounds, 1, "only lasting recorded wounds are shown")
	assert.Empty(t, received["companion:4"].Effects, "no invented saved temporary buffs")
	assert.Equal(t, "Until treated or rested away", received["companion:4"].Wounds[0].Duration)
	module.load()
	require.NoError(t, module.loadErr)
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, "separated", received["companion:4"].State, "durable separated record survives reload")
	b.rounds(domain.DefaultSeparationRounds)
	companyview.RefreshUser(7)
	events.ProcessEvents()
	assert.Equal(t, "live", received["companion:4"].State)
	require.Len(t, received["companion:4"].Wounds, 1)
	assert.Equal(t, b.aria.Character.RoomId, b.companion(4).Character.RoomId)
	assert.Contains(t, rooms.LoadRoom(b.aria.Character.RoomId).GetMobs(rooms.FindCharmed), b.companion(4).InstanceId)
}
