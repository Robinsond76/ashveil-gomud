package gmcp

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPacedCombatHoldsTheWebViews (Phase 29f): while a player's combat
// round is being paced out, a refresh sends no Company payload (the battle
// view included) and a vitals change sends no Char.Vitals; when the lines
// drain, both catch up through the real listeners.
func TestPacedCombatHoldsTheWebViews(t *testing.T) {
	facts := sampleBattle()
	f, out := testFeed()
	f.extras = []companyExtra{battleExtra(func(*users.UserRecord) battleFacts { return facts })}
	prev := companyFeeds
	companyFeeds = f
	t.Cleanup(func() { companyFeeds = prev })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(7, 1)
	users.SetTestUser(u)

	pacer := combatpace.New()
	t.Cleanup(combatpace.UseForTest(pacer))

	var vitals int
	id := events.RegisterListener(GMCPCharUpdate{}, func(e events.Event) events.ListenerReturn {
		if up := e.(GMCPCharUpdate); up.UserId == 7 && up.Identifier == `Char.Vitals` {
			vitals++
		}
		return events.Cancel // no connection to send it to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPCharUpdate{}, id) })
	events.ProcessEvents()

	pacer.StartRound(7) // a combat round opens for Aria
	companyview.Refresh(u)
	events.AddToQueue(events.CharacterVitalsChanged{UserId: 7})
	events.ProcessEvents()
	assert.Empty(t, *out, "no Company payload during the paced round")
	assert.Zero(t, vitals, "no Char.Vitals during the paced round")

	_, drained := pacer.Due(time.Now())
	require.Equal(t, []int{7}, drained)
	events.AddToQueue(events.CombatPaceDrained{UserId: 7})
	events.ProcessEvents()
	modules := map[string]bool{}
	for _, s := range *out {
		modules[s.module] = true
	}
	assert.True(t, modules["Company"], "Company caught up: %v", *out)
	assert.True(t, modules["Company.Battle"], "the battle view caught up: %v", *out)
	assert.Equal(t, 1, vitals, "Char.Vitals caught up once")

	// Outside a paced round, both go out as before.
	*out = nil
	vitals = 0
	facts.Enemies[0].Health = 1
	companyview.Refresh(u)
	events.AddToQueue(events.CharacterVitalsChanged{UserId: 7})
	events.ProcessEvents()
	assert.NotEmpty(t, *out)
	assert.Equal(t, 1, vitals)

	// A drain with nothing held sends no vitals.
	vitals = 0
	events.AddToQueue(events.CombatPaceDrained{UserId: 7})
	events.ProcessEvents()
	assert.Zero(t, vitals)
}
