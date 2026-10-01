package spells

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHelpfulScopeCompatibilityAndValidation(t *testing.T) {
	for _, tc := range []struct {
		typ  SpellType
		want EffectScope
	}{{HelpSingle, ScopeMember}, {HelpMulti, ScopeCompany}, {HelpArea, ScopeArea}, {HarmMulti, ""}} {
		sp := &SpellData{Type: tc.typ}
		assert.NoError(t, sp.Validate())
		assert.Equal(t, tc.want, sp.FriendlyScope())
	}
	for _, tc := range []struct {
		typ   SpellType
		scope EffectScope
		valid bool
	}{{HelpMulti, ScopeAllied, true}, {HelpSingle, ScopeMember, true}, {HelpMulti, ScopeMember, false}, {HelpSingle, ScopeCompany, false}, {HarmMulti, ScopeCompany, false}, {HelpMulti, "bogus", false}} {
		err := (&SpellData{Type: tc.typ, Scope: tc.scope}).Validate()
		if tc.valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}
