package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30b wiring: `heal` and `heal wounds` through the real command, in
// the brawl world (Oswin is a cleric), out of any fight.

// answer answers the open prompt question as the world's input loop does,
// and runs the command it belongs to.
func (b *brawl) answer(response string) string {
	b.t.Helper()
	p := b.aria.GetPrompt()
	require.NotNil(b.t, p, "a question is open")
	q := p.GetNextQuestion()
	require.NotNil(b.t, q)
	q.Answer(response)
	return b.cmd(p.Command, p.Rest)
}

func (b *brawl) quietSaves() *int {
	saved := 0
	old := module.saveUser
	module.saveUser = func(*users.UserRecord) error { saved++; return nil }
	b.t.Cleanup(func() { module.saveUser = old })
	return &saved
}

func TestHealWoundsIsRefusedInAFight(t *testing.T) {
	b := newBrawl(t)
	b.aimAt("bandit captain")
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 2}}
	assert.Contains(t, b.cmd("heal", "wounds"), "You can't tend wounds in the middle of a fight.")
	assert.Len(t, b.aria.Character.Wounds, 1)
	assert.Nil(t, b.aria.GetPrompt())
}

func TestHealListsWithoutChangingAnything(t *testing.T) {
	b := newBrawl(t)
	tamsin := b.companion(1)
	tamsin.Character.HealthMax.Value, tamsin.Character.Health = 100, 30
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 40}}
	out := b.cmd("heal", "")
	assert.Contains(t, out, "Tamsin Reed: 30/100 (wound limit 60); a broken arm (holds back 40)")
	assert.Contains(t, out, "heal wounds")
	assert.Equal(t, 30, tamsin.Character.Health)
	assert.Len(t, tamsin.Character.Wounds, 1)
	assert.Contains(t, b.cmd("heal", "sideways"), "Usage:")
}

// The cleric first (tend, then heal), then splints and bandages from the
// cargo; the clock never moves.
func TestHealWoundsClericThenItems(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: splintItemID, Count: 1}, {ItemId: bandageItemID, Count: 3}}}
	useCargo(t, cargo)
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 8
	tamsin := b.companion(1)
	tamsin.Character.HealthMax.Value, tamsin.Character.Health = 100, 30
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 40}}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	out := b.cmd("heal", "wounds")
	assert.Contains(t, out, "Brother Oswin kneels beside Tamsin Reed first, the worst hurt.")
	assert.Regexp(t, `Brother Oswin tends Tamsin Reed's broken arm, and it draws closed\. \(wound treated, limit \d+ of 100\)`, out)
	assert.Equal(t, 0, oswin.Character.Mana, "two tends at 4 mana each")
	assert.Contains(t, out, "sways, pale and spent. (mana 0 of 20)")
	assert.Regexp(t, `You set Tamsin Reed's broken arm against a splint and bind it with linen\. \(wound treated, limit \d+ of 100; 0 splints left\)`, out)
	assert.Contains(t, cargo.consumed, splintItemID)
	require.Len(t, tamsin.Character.Wounds, 1)
	closed := 40 - tamsin.Character.Wounds[0].Points
	assert.GreaterOrEqual(t, closed, 4+4, "two tends (at least 2 each) and a splint (4)")
	assert.LessOrEqual(t, closed, 12+4)
	assert.Contains(t, cargo.consumed, bandageItemID, "under half her limit: bandaged")
	assert.Contains(t, out, "You wrap Tamsin Reed's hurts.")
	assert.Contains(t, out, "Still hurt: Tamsin Reed")
	assert.Contains(t, out, "A camp rest or an inn will do the rest.")
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

func TestHealWoundsWithNoOneHurt(t *testing.T) {
	b := newBrawl(t)
	assert.Contains(t, b.cmd("heal", "wounds"), "No one in your company is hurt.")
	assert.Nil(t, b.aria.GetPrompt())
}

// The physician: no keeps the gold; yes takes it and closes every wound;
// too little gold is refused.
func TestHealWoundsPhysician(t *testing.T) {
	b := newBrawl(t)
	saves := b.quietSaves()
	module.physiciansForTest = map[int]physician{b.road.RoomId: {RoomID: b.road.RoomId, Name: "the physician", PricePerWound: 15}}
	t.Cleanup(func() { module.physiciansForTest = nil })
	b.companion(2).Character.Mana = 0 // no cleric to help
	tamsin := b.companion(1)
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}}
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 1}}
	b.aria.Character.Gold = 100

	out := b.cmd("heal", "wounds")
	assert.Contains(t, out, `"30 gold, and they'll all be closed by morning."`)
	require.NotNil(t, b.aria.GetPrompt())
	assert.Contains(t, b.answer("no"), "You thank the physician and keep your gold.")
	assert.Equal(t, 100, b.aria.Character.Gold)
	assert.Len(t, tamsin.Character.Wounds, 1)
	assert.Nil(t, b.aria.GetPrompt())

	b.cmd("heal", "wounds")
	out = b.answer("yes")
	assert.Contains(t, out, "You pay 30 gold.")
	assert.Contains(t, out, "(2 wounds healed)")
	assert.Equal(t, 70, b.aria.Character.Gold)
	assert.Empty(t, tamsin.Character.Wounds)
	assert.Empty(t, b.aria.Character.Wounds)
	assert.Equal(t, 1, *saves, "the user is saved with the payment")
	assert.Nil(t, b.aria.GetPrompt())

	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}}
	b.aria.Character.Gold = 10
	b.cmd("heal", "wounds")
	assert.Contains(t, b.answer("yes"), "You don't have 15 gold.")
	assert.Equal(t, 10, b.aria.Character.Gold)
	assert.Len(t, tamsin.Character.Wounds, 1)
}
