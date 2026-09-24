package death

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryRejectsBadEntries(t *testing.T) {
	r, errs := NewRegistry([]Settlement{
		{Zone: "Frostfang", Kind: City, ServiceRoomID: 18},
		{Zone: "", Kind: City, ServiceRoomID: 5},
		{Zone: "Nowhere", Kind: "hamlet", ServiceRoomID: 6},
		{Zone: "Zero", Kind: Village, ServiceRoomID: 0},
		{Zone: "Frostfang", Kind: City, ServiceRoomID: 19},
		{Zone: " Dunmar ", Kind: " CITY ", ServiceRoomID: 2007},
	})

	require.Len(t, errs, 4)
	assert.True(t, errors.Is(errs[0], ErrInvalidSettlement))
	assert.True(t, errors.Is(errs[3], ErrDuplicateZone))
	assert.Equal(t, []Settlement{
		{Zone: "Frostfang", Kind: City, ServiceRoomID: 18},
		{Zone: "Dunmar", Kind: City, ServiceRoomID: 2007},
	}, r.Settlements(), "the first entry for a zone wins; names and kinds are trimmed")
}

func TestChurchForCityOnly(t *testing.T) {
	r, errs := NewRegistry([]Settlement{
		{Zone: "Dunmar", Kind: City, ServiceRoomID: 2007},
		{Zone: "Old Kings Road", Kind: Village, ServiceRoomID: 2005},
	})
	require.Empty(t, errs)

	church, ok := r.ChurchFor("Dunmar")
	assert.True(t, ok)
	assert.Equal(t, 2007, church)
	_, ok = r.ChurchFor("Old Kings Road")
	assert.False(t, ok, "a village is never a checkpoint")
	_, ok = r.ChurchFor("Frostfang")
	assert.False(t, ok, "unregistered")
	assert.Equal(t, ChurchTag, r.Settlements()[0].ServiceTag())
	assert.Equal(t, ShamanTag, r.Settlements()[1].ServiceTag())
}

func TestIsChurch(t *testing.T) {
	r, _ := NewRegistry([]Settlement{
		{Zone: "Dunmar", Kind: City, ServiceRoomID: 2007},
		{Zone: "Old Kings Road", Kind: Village, ServiceRoomID: 2005},
	})
	assert.True(t, r.IsChurch(2007))
	assert.False(t, r.IsChurch(2005), "a shaman isn't a church")
	assert.False(t, r.IsChurch(0))
	assert.False(t, r.IsChurch(1))
}

func TestDestinationPrefersValidCheckpoint(t *testing.T) {
	valid := func(id int) bool { return id == 2007 }
	loads := func(int) bool { return true }

	room, ok := Destination(2007, 18, valid, loads)

	assert.True(t, ok)
	assert.Equal(t, 2007, room)
}

func TestDestinationFallsBack(t *testing.T) {
	valid := func(int) bool { return false }
	loads := func(id int) bool { return id == 18 }

	for _, checkpoint := range []int{0, 2007, -3} {
		room, ok := Destination(checkpoint, 18, valid, loads)
		assert.True(t, ok)
		assert.Equal(t, 18, room)
	}
}

func TestDestinationNone(t *testing.T) {
	none := func(int) bool { return false }

	_, ok := Destination(2007, 18, none, none)
	assert.False(t, ok)
	_, ok = Destination(0, 0, none, func(int) bool { return true })
	assert.False(t, ok, "no fallback configured")
}

type fakeProvider struct{}

func (fakeProvider) Pending(int) bool      { return false }
func (fakeProvider) JustReturned(int) bool { return false }
func (fakeProvider) Respawn(int, bool)     {}

func TestProviderNoneRegistered(t *testing.T) {
	SetProvider(nil)
	_, ok := Active()
	assert.False(t, ok)

	SetProvider(fakeProvider{})
	t.Cleanup(func() { SetProvider(nil) })
	p, ok := Active()
	assert.True(t, ok)
	assert.NotNil(t, p)
}

// TestServiceAt (Phase 25b): the settlement whose service room this is, a
// city's church or a village's shaman, with its keeper.
func TestServiceAt(t *testing.T) {
	r, errs := NewRegistry([]Settlement{
		{Zone: "Dunmar", Kind: City, ServiceRoomID: 2007, ServiceMobID: 65},
		{Zone: "Fernhollow", Kind: Village, ServiceRoomID: 2009, ServiceMobID: 66},
		{Zone: "Bleakmoor", Kind: City, ServiceRoomID: 3002, ServiceMobID: -1},
	})
	assert.Len(t, errs, 1, "a keeper below 0 is rejected")
	s, ok := r.ServiceAt(2009)
	assert.True(t, ok)
	assert.Equal(t, Village, s.Kind)
	assert.Equal(t, 66, s.ServiceMobID)
	s, ok = r.ServiceAt(2007)
	assert.True(t, ok)
	assert.Equal(t, 65, s.ServiceMobID)
	_, ok = r.ServiceAt(2004)
	assert.False(t, ok)
	_, ok = r.ServiceAt(0)
	assert.False(t, ok)
}
