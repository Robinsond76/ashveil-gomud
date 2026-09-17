package company

import (
	"errors"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeCompaniesMigratesLegacySingleCompanion(t *testing.T) {
	legacy := []byte("companies:\n  2:\n    leader_user_id: 2\n    companion:\n      mob_template_id: 58\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(legacy, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 1, record.Companions[0].ID)
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
}

func TestDecodeCompaniesReadsRosterAndFormation(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    companions:\n      - id: 1\n        mob_template_id: 58\n    formation:\n      - [\"leader\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(1, 1))
}

func TestDecodeCompaniesRejectsMalformedData(t *testing.T) {
	registry := domain.NewRegistry()
	registry.Put(domain.Record{
		LeaderUserID: 2,
		Companions:   []domain.Companion{{ID: 1, MobTemplateID: 58}},
	})

	assert.Error(t, decodeCompanies([]byte("companies: [this is: not valid"), registry))

	record, ok := registry.Get(2)
	require.True(t, ok, "malformed data must not replace the active registry")
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
}

func TestDecodeCompaniesPreservesNonzeroLegacyID(t *testing.T) {
	legacy := []byte("companies:\n  2:\n    leader_user_id: 2\n    companion:\n      id: 5\n      mob_template_id: 58\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(legacy, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, 5, record.Companions[0].ID)
}

func TestDecodeCompaniesAssignsNextCompanionIDAfterLegacyLoad(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    companions:\n      - id: 3\n        mob_template_id: 58\n      - id: 7\n        mob_template_id: 58\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	assert.Equal(t, 8, record.NextCompanionID, "legacy data without the field must resume above the highest ID")
}

func TestDecodeCompaniesReadsPersistedNextCompanionID(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    companions:\n      - id: 1\n        mob_template_id: 58\n    next_companion_id: 6\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	assert.Equal(t, 6, record.NextCompanionID)
}

func TestDecodeCompaniesKeepsEmptyHighWaterMarkRecord(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    next_companion_id: 5\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	assert.Empty(t, record.Companions)
	assert.Equal(t, 5, record.NextCompanionID)
}

func TestDecodeCompaniesPrefersRosterOverLegacyCompanion(t *testing.T) {
	data := []byte("companies:\n  2:\n    leader_user_id: 2\n    companions:\n      - id: 1\n        mob_template_id: 58\n      - id: 2\n        mob_template_id: 59\n    companion:\n      id: 9\n      mob_template_id: 99\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	require.Len(t, record.Companions, 2, "the legacy companion must not be appended when a roster exists")
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
	assert.Equal(t, 59, record.Companions[1].MobTemplateID)
}

func formationModule(t *testing.T) (*CompanyModule, *fakeStore) {
	t.Helper()
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	store := module.store.(*fakeStore)
	return module, store
}

func TestFormationCommandMovesLeaderAndPersists(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	handled, err := module.formationCommand("move leader 1 1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, 1, store.saveCalls)
	saved := store.saved.Companies[7]
	assert.Equal(t, domain.LeaderMemberKey, saved.Formation.At(0, 0))
}

func TestFormationCommandRejectsOccupiedSlotAndUnknownMember(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	_, err := module.formationCommand("move leader 1 1", user, nil, 0)
	require.NoError(t, err)
	_, err = module.formationCommand("move #1 1 1", user, nil, 0)
	assert.ErrorIs(t, err, domain.ErrSlotOccupied)
	_, err = module.formationCommand("move #9 2 2", user, nil, 0)
	assert.Error(t, err)
	assert.Equal(t, 1, store.saveCalls, "rejected moves must not persist")
}

func TestFormationCommandSwapAndClear(t *testing.T) {
	module, _ := formationModule(t)
	user := users.NewUserRecord(7, 1)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	require.NoError(t, module.registry.PlaceMember(7, domain.CompanionMemberKey(1), 2, 2))

	handled, err := module.formationCommand("swap leader #1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.CompanionMemberKey(1), record.Formation.At(0, 0))

	handled, err = module.formationCommand("clear #1", user, nil, 0)
	require.True(t, handled)
	require.NoError(t, err)
	record, _ = module.registry.Get(7)
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 0))
}

func TestFormationCommandSaveFailureRollsBack(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	store.saveErr = errors.New("disk full")
	_, err := module.formationCommand("move leader 1 1", user, nil, 0)
	assert.ErrorIs(t, err, store.saveErr)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 0), "failed save must not keep the move")
}

func TestFormationCommandSaveFailureRollsBackNewRecord(t *testing.T) {
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	user := users.NewUserRecord(7, 1)
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")

	_, err := module.formationCommand("move leader 1 1", user, nil, 0)
	assert.ErrorIs(t, err, store.saveErr)
	_, ok := module.registry.Get(7)
	assert.False(t, ok, "a failed first move must not leave a company record")
}

func TestFormationCommandSwapSaveFailureRollsBack(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	require.NoError(t, module.registry.PlaceMember(7, domain.CompanionMemberKey(1), 2, 2))
	before := cloneRegistry(module.registry)

	store.saveErr = errors.New("disk full")
	_, err := module.formationCommand("swap leader #1", user, nil, 0)
	assert.ErrorIs(t, err, store.saveErr)
	assert.Equal(t, before, module.registry, "failed swap must not persist")
}

func TestFormationCommandClearSaveFailureRollsBack(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	before := cloneRegistry(module.registry)

	store.saveErr = errors.New("disk full")
	_, err := module.formationCommand("clear leader", user, nil, 0)
	assert.ErrorIs(t, err, store.saveErr)
	assert.Equal(t, before, module.registry, "failed clear must not persist")
}

func TestFormationCommandRejectsUnknownCompanionID(t *testing.T) {
	module, store := formationModule(t)
	user := users.NewUserRecord(7, 1)
	_, err := module.formationCommand("move #9 2 2", user, nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no company member matches")
	assert.Equal(t, 0, store.saveCalls)
}

func TestFormationCommandRendersGridAndUnplaced(t *testing.T) {
	module, _ := formationModule(t)
	require.NoError(t, module.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0))
	text := module.renderFormation(7)
	assert.Contains(t, text, "front")
	assert.Contains(t, text, "leader")
	assert.Contains(t, text, "#1", "unplaced companions are listed by id")
	assert.Contains(t, text, "#2", "unplaced companions are listed by id")
}

func TestParseSlotRejectsOutOfRange(t *testing.T) {
	_, err := parseSlot("0")
	assert.Error(t, err)
	_, err = parseSlot("4")
	assert.Error(t, err)
	n, err := parseSlot("2")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestDecodeCompaniesKeepsFirstDuplicateFormationOccupant(t *testing.T) {
	data := []byte("companies:\n  2:\n    formation:\n      - [\"leader\", \"leader\", \"\"]\n      - [\"\", \"\", \"\"]\n      - [\"\", \"\", \"\"]\n")
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, registry))

	record, ok := registry.Get(2)
	require.True(t, ok)
	assert.Equal(t, domain.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, domain.MemberKey(""), record.Formation.At(0, 1))
}
