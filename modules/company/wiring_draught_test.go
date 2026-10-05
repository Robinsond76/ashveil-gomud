package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b: a mana draught restores its share of the drinker's mana pool
// at once, through the real drink command and buff script; in a battle it
// is refused.
func TestManaDraughtsRestoreTheirShare(t *testing.T) {
	b := newBrawl(t)
	copyShipped(t, configs.GetFilePathsConfig().DataFiles.String(), "buffs")
	buffs.LoadDataFiles()
	// The brawl stands in for the world loop, so it applies queued buffs
	// as the server's own listener does.
	lid := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, lid) })
	for _, c := range []struct {
		item, pct int
		name      string
	}{{30014, 25, "minor mana draught"}, {30022, 40, "lesser mana draught"}, {30023, 60, "greater mana draught"}} {
		spec := items.GetItemSpec(c.item)
		require.NotNil(t, spec, c.name)
		assert.Equal(t, c.name, spec.Name)
		b.aria.Character.Validate()
		b.aria.Character.Mana = 0
		require.True(t, b.aria.Character.StoreItem(items.New(c.item)))
		out := b.cmd("drink", c.name)
		top := b.aria.Character.ManaMax.Value
		require.Positive(t, top)
		assert.Equal(t, top*c.pct/100, b.aria.Character.Mana, "%s: %s", c.name, out)
	}

	b.aimAt("bandit captain")
	b.fight()
	require.True(t, b.aria.Character.StoreItem(items.New(30014)))
	b.aria.Character.Mana = 10
	assert.Contains(t, b.cmd("drink", "minor mana draught"), "The battle is under way")
	assert.Equal(t, 10, b.aria.Character.Mana)
}
