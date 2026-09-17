package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
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
