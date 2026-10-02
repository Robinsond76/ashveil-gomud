package company

import (
	"os"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/require"
)

// TestCompanyRefreshLoad measures what one player's company feed costs the
// game loop each round (Phase 34 review): the summary, the Company snapshot
// and every extra (Inventory, Equipment, Conditions, Capabilities, Camp,
// Battle), for a leader with companions, 30 pieces of gear in cargo, and a
// timed and a permanent effect ticking. The loop refreshes every online
// player once a round, so the per-player figure times the player count is
// that part of each round. Combat, mob AI and movement are not included.
//
// Run: ASHVEIL_LOAD_BENCH=1 go test ./modules/company -run TestCompanyRefreshLoad -v
func TestCompanyRefreshLoad(t *testing.T) {
	if os.Getenv("ASHVEIL_LOAD_BENCH") == "" {
		t.Skip("set ASHVEIL_LOAD_BENCH=1 to measure the per-player refresh")
	}
	b := equipmentBrawl(t)
	c := b.aria.Character
	timed := buffs.BuffSpec{BuffId: 989906, Name: "Well fed", Description: "Fed.", RoundInterval: 5, TriggerCount: 100000, StatMods: map[string]int{"strength": 1}}
	perma := buffs.BuffSpec{BuffId: 989907, Name: "Night Vision", Description: "Sees.", RoundInterval: 30, TriggerCount: 1}
	for _, spec := range []*buffs.BuffSpec{&timed, &perma} {
		buffs.SetTestBuffSpec(spec)
		id := spec.BuffId
		t.Cleanup(func() { buffs.RemoveTestBuffSpec(id) })
	}
	// A running server has its buff flags loaded; without them every flag
	// check logs a warning, which would dominate the measurement.
	for _, flag := range []string{"accuracy", "armor-broken", "bleeding", "blink", "burning", "cancel-on-action", "cancel-on-combat",
		"cancel-on-water", "combat-status", "drunk", "exposed", "hidden", "hydrated", "lightsource", "lose-actions", "lose-first-action",
		"nightvision", "no-block", "no-combat", "no-dodge", "no-flee", "no-go", "perma-gear", "poison", "remove-curse", "revive-on-death",
		"see-hidden", "see-nouns", "superhearing", "thirsty", "tripping", "warmed", "partylight", "rested", "well-rested"} {
		if !buffs.IsValidFlag(flag) {
			buffs.SetTestFlag(flag)
			f := flag
			t.Cleanup(func() { buffs.RemoveTestFlag(f) })
		}
	}
	round := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(round) })
	gear := []int{10001, 10002, 10004, 10005, 10007, 10010, 20001, 20003, 20006, 20007, 20008, 20009, 20010, 20012, 20014, 20016,
		20018, 20020, 20022, 20028, 20029, 20030, 20034, 20040, 20042, 10003, 10006, 10011, 20013, 20031}
	c.Items = nil
	for _, id := range gear {
		c.Items = append(c.Items, items.New(id))
	}
	for i := 0; i < 6; i++ {
		c.Items = append(c.Items, items.New(30004+i%3))
	}
	c.Buffs.List = []*buffs.Buff{{BuffId: timed.BuffId, TriggersLeft: 100000, TriggersInitial: 100000},
		{BuffId: perma.BuffId, PermaBuff: true, TriggersLeft: buffs.TriggersLeftUnlimited}}
	require.NoError(t, c.Validate(true))
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())

	tick := func() {
		util.SetRoundCount(util.GetRoundCount() + 1)
		c.Buffs.Trigger()
		c.Health = max(1, c.Health-1)
		companyview.Refresh(b.aria)
		events.ProcessEvents()
	}
	measure := func(name string, each func()) time.Duration {
		tick() // settle: first sends
		r := testing.Benchmark(func(bb *testing.B) {
			for i := 0; i < bb.N; i++ {
				each()
				tick()
			}
		})
		per := time.Duration(r.NsPerOp())
		t.Logf("%-38s %8.2f ms/player/round   100 players: %6.0f ms   200: %6.0f ms   500: %6.0f ms",
			name, per.Seconds()*1000, per.Seconds()*1000*100, per.Seconds()*1000*200, per.Seconds()*1000*500)
		return per
	}

	events.AddToQueue(gmcp.GMCPGearWatch{UserId: 7, Open: false})
	events.ProcessEvents()
	closed := measure("Gear tab closed (most players)", func() {})
	events.AddToQueue(gmcp.GMCPGearWatch{UserId: 7, Open: true, Slot: "body"})
	events.ProcessEvents()
	open := measure("Gear tab open, nothing changed", func() {})
	extra := items.New(20044)
	changing := measure("Gear tab open, cargo changes each round", func() {
		if len(c.Items) > len(gear)+6 {
			c.Items = c.Items[:len(c.Items)-1]
		} else {
			c.Items = append(c.Items, extra)
		}
	})
	t.Logf("a round is %d ms; these costs are per round, on the game loop", 4000)
	require.Less(t, closed, 5*time.Millisecond, "a closed Gear tab must keep the refresh cheap")
	_ = open
	_ = changing
}
