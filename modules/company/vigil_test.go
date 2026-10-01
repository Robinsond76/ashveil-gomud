package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestRaiseLoyaltyOnceForAVigil (33f3): a vigil raises the named living
// companions' loyalty, never above the cap, once per operation, durably;
// a failed save changes nothing.
func TestRaiseLoyaltyOnceForAVigil(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			withDisposition(domain.Companion{ID: 1, Name: "Bran", MobTemplateID: 58}, 0, 40),
			withDisposition(domain.Companion{ID: 2, Name: "Cara", MobTemplateID: 58}, 0, 59),
			withDisposition(domain.Companion{ID: 3, Name: "Dain", MobTemplateID: 58}, 0, 80),
			withDisposition(domain.Companion{ID: 4, Name: "Esk", MobTemplateID: 58}, 0, 10),
		}},
	}}, &fakeRuntime{})
	store := module.store.(*fakeStore)
	loyalty := func(id int) int {
		r, _ := module.registry.Get(7)
		c, _ := findCompanion(r, id)
		return c.Disposition.Loyalty
	}

	raised, err := module.RaiseLoyaltyOnce(7, "rest-1:vigil", []int{1, 2, 3}, 3, 60)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bran", "Cara"}, raised)
	assert.Equal(t, 43, loyalty(1))
	assert.Equal(t, 60, loyalty(2), "held to the cap")
	assert.Equal(t, 80, loyalty(3), "above the cap: never lowered")
	assert.Equal(t, 10, loyalty(4), "not at the vigil")
	saved, _ := store.saved.Get(7)
	assert.True(t, saved.HasApplied("rest-1:vigil"), "durable with the change")

	raised, err = module.RaiseLoyaltyOnce(7, "rest-1:vigil", []int{1}, 3, 60)
	require.NoError(t, err)
	assert.Empty(t, raised)
	assert.Equal(t, 43, loyalty(1), "never twice")

	store.saveErr = assert.AnError
	_, err = module.RaiseLoyaltyOnce(7, "rest-2:vigil", []int{1}, 3, 60)
	assert.Error(t, err)
	assert.Equal(t, 43, loyalty(1), "rolled back")
}

// TestAppliedOpsSurviveTheRealDecoder (33f3 review finding 1): a vigil's
// operation round-trips through the company save format, so a retry after
// a restart still finds it.
func TestAppliedOpsSurviveTheRealDecoder(t *testing.T) {
	registry := domain.NewRegistry()
	record := domain.Record{LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, Name: "Bran", MobTemplateID: 58}, 0, 40)}}
	record.MarkApplied("rest-1:vigil")
	registry.Put(record)
	data, err := yaml.Marshal(registry)
	require.NoError(t, err)
	loaded := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, loaded))
	got, ok := loaded.Get(7)
	require.True(t, ok)
	assert.True(t, got.HasApplied("rest-1:vigil"))
}
