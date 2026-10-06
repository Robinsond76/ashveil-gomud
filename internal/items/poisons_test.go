package items

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoatingLifecycle(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	var sword Item
	assert.False(t, sword.Coated(now))
	assert.True(t, sword.Coat("bitterleaf", CoatExpiry(now), CoatContacts, now))
	assert.True(t, sword.Coated(now))
	assert.Equal(t, now.Add(10*time.Minute).Unix(), sword.CoatExpires)

	// Never replaced or refreshed while live.
	assert.False(t, sword.Coat("leadroot", CoatExpiry(now.Add(time.Minute)), 3, now.Add(time.Minute)))
	assert.Equal(t, "bitterleaf", sword.CoatKind)
	assert.Equal(t, CoatContacts, sword.CoatContacts)

	// Real time passes whether or not anyone is logged in.
	assert.True(t, sword.Coated(now.Add(9*time.Minute)))
	assert.False(t, sword.Coated(now.Add(10*time.Minute)), "expired at the deadline")
	assert.Empty(t, sword.CoatLabel(now.Add(11*time.Minute)))
	assert.True(t, sword.Coat("leadroot", CoatExpiry(now.Add(11*time.Minute)), 3, now.Add(11*time.Minute)), "a lapsed coating can be replaced")
	assert.Equal(t, "leadroot", sword.CoatKind)
}

func TestCoatingContactsSpendAndEnd(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	var sword Item
	sword.Coat("mirethorn", CoatExpiry(now), 3, now)
	sword.SpendCoat(2)
	assert.Equal(t, 1, sword.CoatContacts)
	assert.Contains(t, sword.CoatLabel(now), "Mirethorn: 10 min, 1 hits")
	assert.Equal(t, "Mirethorn, 10 min, 1 hits", sword.CoatSummary(now))
	sword.SpendCoat(1)
	assert.Empty(t, sword.CoatKind, "no contacts left, no coating")
	assert.False(t, sword.Coated(now))
	assert.False(t, sword.ClearCoat())
}

func TestCoatRejectsNonsense(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	var sword Item
	assert.False(t, sword.Coat("", CoatExpiry(now), 8, now))
	assert.False(t, sword.Coat("bitterleaf", CoatExpiry(now), 0, now))
	assert.False(t, sword.Coat("bitterleaf", now, 8, now), "already expired")
}

func TestPoisonCatalogue(t *testing.T) {
	assert.Len(t, Poisons, 4)
	for _, p := range Poisons {
		got, ok := PoisonByID(p.ID)
		assert.True(t, ok)
		assert.Equal(t, p, got)
		byVial, ok := PoisonByVial(p.VialID)
		assert.True(t, ok)
		assert.Equal(t, p.ID, byVial.ID)
	}
	_, ok := PoisonByID("nightshade")
	assert.False(t, ok)
}

func TestEdgeLabelShowsSharpAndCoating(t *testing.T) {
	var sword Item
	sword.Sharpen(1, 5)
	assert.Contains(t, sword.EdgeLabel(), "sharp: 5")
	now := time.Now()
	sword.Coat("leechbane", CoatExpiry(now), 8, now)
	label := sword.EdgeLabel()
	assert.Contains(t, label, "sharp: 5")
	assert.Contains(t, label, "Leechbane")
}
