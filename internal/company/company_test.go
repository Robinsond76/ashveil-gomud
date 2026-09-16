package company_test

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrySummonStoresAllowedTemplate(t *testing.T) {
	registry := company.NewRegistry()
	err := registry.Summon(7, 58, map[int]struct{}{58: {}})
	require.NoError(t, err)

	got, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, 7, got.LeaderUserID)
	assert.Equal(t, 58, got.Companion.MobTemplateID)
}

func TestRegistrySummonRejectsSecondCompanion(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.Summon(7, 58, map[int]struct{}{58: {}}))

	assert.ErrorIs(t, registry.Summon(7, 58, map[int]struct{}{58: {}}), company.ErrCompanionAlreadyPresent)
	got, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, 58, got.Companion.MobTemplateID)
}

func TestRegistrySummonValidatesIDsAndAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		leaderID int
		template int
		allowed  map[int]struct{}
		wantErr  error
	}{
		{name: "zero leader", leaderID: 0, template: 58, allowed: map[int]struct{}{58: {}}, wantErr: company.ErrInvalidLeader},
		{name: "negative leader", leaderID: -1, template: 58, allowed: map[int]struct{}{58: {}}, wantErr: company.ErrInvalidLeader},
		{name: "zero template", leaderID: 7, template: 0, allowed: map[int]struct{}{0: {}}, wantErr: company.ErrInvalidTemplate},
		{name: "negative template", leaderID: 7, template: -1, allowed: map[int]struct{}{-1: {}}, wantErr: company.ErrInvalidTemplate},
		{name: "disallowed template", leaderID: 7, template: 58, allowed: map[int]struct{}{59: {}}, wantErr: company.ErrTemplateNotAllowed},
		{name: "nil allowlist", leaderID: 7, template: 58, allowed: nil, wantErr: company.ErrTemplateNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, company.NewRegistry().Summon(tt.leaderID, tt.template, tt.allowed), tt.wantErr)
		})
	}
}

func TestRegistryGetReportsUnknownLeader(t *testing.T) {
	got, ok := company.NewRegistry().Get(7)
	assert.False(t, ok)
	assert.Equal(t, company.Record{}, got)
}

func TestRegistryDismissRemovesRecordAndIsIdempotent(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.Summon(7, 58, map[int]struct{}{58: {}}))

	assert.True(t, registry.Dismiss(7))
	assert.False(t, registry.Dismiss(7))
	_, ok := registry.Get(7)
	assert.False(t, ok)
}

func TestRegistrySentinelsAreDistinct(t *testing.T) {
	assert.False(t, errors.Is(company.ErrInvalidLeader, company.ErrInvalidTemplate))
	assert.False(t, errors.Is(company.ErrInvalidTemplate, company.ErrTemplateNotAllowed))
	assert.False(t, errors.Is(company.ErrTemplateNotAllowed, company.ErrCompanionAlreadyPresent))
}
