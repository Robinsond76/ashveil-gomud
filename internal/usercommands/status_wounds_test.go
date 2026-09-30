package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 30b: the character sheet's health shows the wound limit while
// wounded, and nothing extra while not.
func TestStatusShowsTheWoundLimit(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Wren"
	user.Character.HealthMax.Value, user.Character.Health = 16, 12
	assert.NotContains(t, statusText(t, user, ""), "limit")

	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}}
	assert.Contains(t, statusText(t, user, ""), "12/16 (limit 13)")
}
