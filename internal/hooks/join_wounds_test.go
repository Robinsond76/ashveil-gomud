package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 30b review: a player's light wounds close at login (no fight
// survives one); lasting wounds stay.
func TestJoinClosesLightWounds(t *testing.T) {
	newHandOffWorld(t)
	u := saveUser(t, 932101, "Scarred")
	u.Character.RoomId = 932001
	u.Character.Wounds = []wounds.Wound{{Kind: wounds.Bruise, Points: 1, Light: true}, {Kind: wounds.Cut, Place: "arm", Points: 2}}
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	HandleJoin(events.PlayerSpawn{UserId: u.UserId, RoomId: 932001})
	assert.Equal(t, []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 2}}, u.Character.Wounds)
}

// Phase 30b review: a lasting wound survives the real user save and load.
func TestLastingWoundSurvivesUserSaveAndLoad(t *testing.T) {
	newHandOffWorld(t)
	u := saveUser(t, 932102, "Mended")
	u.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 3}}
	if err := users.SaveUser(*u); err != nil {
		t.Fatal(err)
	}
	back, err := users.LoadUserFile(u.UserId)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, u.Character.Wounds, back.Character.Wounds)
}
