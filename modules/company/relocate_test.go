package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func relocationModule(t *testing.T) (*CompanyModule, *fakeRuntime) {
	t.Helper()
	runtime := &fakeRuntime{live: map[int]bool{101: true, 102: true, 103: true, 104: true}}
	record := domain.Record{LeaderUserID: 7, Companions: []domain.Companion{
		{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}, {ID: 3, MobTemplateID: 58}, {ID: 4, MobTemplateID: 58},
	}}
	require.NoError(t, record.Formation.Place(domain.CompanionMemberKey(1), 1, 1))
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{7: record}}, runtime)
	for companionID, instanceID := range map[int]int{1: 101, 2: 102, 3: 103, 4: 104} {
		module.setInstance(7, companionID, instanceID)
	}
	return module, runtime
}

func TestRelocateCompanyMovesLiveAttached(t *testing.T) {
	module, runtime := relocationModule(t)
	before, _ := module.registry.Get(7)

	moved := module.RelocateCompany(7, 18)

	assert.Equal(t, 4, moved)
	assert.Equal(t, map[int]int{101: 18, 102: 18, 103: 18, 104: 18}, runtime.relocated)
	after, _ := module.registry.Get(7)
	assert.Equal(t, before, after, "only live mobs move")
	assert.Zero(t, runtime.detachCalls)
	assert.Zero(t, runtime.spawnCalls)
	instanceID, tracked := module.instance(7, 1)
	assert.True(t, tracked, "still tracked")
	assert.Equal(t, 101, instanceID)
}

func TestRelocateCompanySkipsDeadDetachedAndUnattached(t *testing.T) {
	module, runtime := relocationModule(t)
	runtime.dying = map[int]bool{101: true}  // alive no longer
	runtime.stolen = map[int]bool{102: true} // charmed away
	delete(runtime.live, 103)                // gone
	module.clearInstance(7, 4)               // died earlier: untracked

	moved := module.RelocateCompany(7, 18)

	assert.Zero(t, moved)
	assert.Empty(t, runtime.relocated)
	assert.Zero(t, module.RelocateCompany(8, 18), "a leader with no company")
}
