package company

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/rites"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 74 wiring: the brawl world's real registry, store, round tick and
// commands. A companion's allowance runs out in the real round tick, the
// camp's own entry point offers the rites, and `rite hold` is typed through
// the real command table.
func TestRitesThroughTheRealRoundTickAndCommands(t *testing.T) {
	b := newBrawl(t)
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	now := time.Unix(3_000_000, 0)
	module.clock = func() time.Time { return now }
	t.Cleanup(func() { module.clock = nil; module.riteSeam = nil })

	// Garrick (#3) trusted Ysolde (#4); Ysolde dies and her time runs out.
	setBond(t, 3, 4, 40)
	require.NoError(t, module.registry.MarkDead(7, 4, domain.CompanionDeath{OpID: "rite-test", Remaining: 2, Allowance: 60}))
	module.startAnchor(7)
	now = now.Add(5 * time.Second)
	told := b.rounds(1)
	assert.Contains(t, told, "is lost to you")
	assert.Contains(t, told, "You can hold rites for them at your next camp or inn (rites)")
	record, _ := module.registry.Get(7)
	require.Len(t, record.Rites, 1)
	assert.Contains(t, record.Rites[0].Close, 3, "the companion who trusted her is close")
	assert.Contains(t, b.cmd("rites", ""), "Waiting to be mourned")

	// Not at a camp or an inn: the real check refuses, nothing changes.
	assert.Contains(t, b.cmd("rite", "hold"), "Rites are held at a camp or an inn")
	record, _ = module.registry.Get(7)
	assert.Len(t, record.Rites, 1)

	// A camp's start announces the rites through the provider seam camping
	// calls; at the fire the command holds them.
	assert.Contains(t, domain.OfferRites(7), "has not yet mourned")
	module.riteSeam = func(*users.UserRecord) string { return "" }
	before := companion(t, module, 3).Disposition.Loyalty
	out := b.cmd("rite", "hold")
	assert.Contains(t, out, "hold rites for")
	assert.Equal(t, min(rites.Ceiling, before+rites.CloseHold), companion(t, module, 3).Disposition.Loyalty)
	deeds := chronicle.Query(7, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Rites}})
	require.Len(t, deeds, 1)
	assert.Equal(t, "rite:held", deeds[0].Ref)
	assert.Contains(t, b.cmd("rites", ""), "No one is waiting to be mourned")
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Rites)
}
