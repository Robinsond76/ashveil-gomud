package company

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"regexp"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
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
	assert.Contains(t, out, "An inn will do the rest, or a camp rest with a splint or bandage for each wound.")
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

	events.ProcessEvents() // drain what earlier steps queued
	gold := []int{}
	listener := events.RegisterListener(events.EquipmentChange{}, func(e events.Event) events.ListenerReturn {
		gold = append(gold, e.(events.EquipmentChange).GoldChange)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.EquipmentChange{}, listener) })
	b.cmd("heal", "wounds")
	out = b.answer("yes")
	events.ProcessEvents()
	assert.Equal(t, []int{-30}, gold, "the Worth panel refreshes on the physician's fee")
	assert.Contains(t, out, "You pay 30 gold.")
	assert.Contains(t, out, "(2 wounds healed)")
	assert.Equal(t, 70, b.aria.Character.Gold)
	assert.Empty(t, tamsin.Character.Wounds)
	assert.Empty(t, b.aria.Character.Wounds)
	record, _ := module.registry.Get(7)
	for _, c := range record.Companions {
		if c.ID == 1 {
			require.NotNil(t, c.State)
			assert.Empty(t, c.State.Wounds, "the company record is refreshed with the payment")
		}
	}
	assert.Equal(t, 1, *saves, "the user is saved with the payment")
	assert.Nil(t, b.aria.GetPrompt())

	// Review fix: too little gold, no question.
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}}
	b.aria.Character.Gold = 10
	assert.Contains(t, b.cmd("heal", "wounds"), "You have only 10 gold.")
	assert.Nil(t, b.aria.GetPrompt())

	// Gold spent between the offer and the answer is checked again.
	b.aria.Character.Gold = 100
	b.cmd("heal", "wounds")
	b.aria.Character.Gold = 10
	assert.Contains(t, b.answer("yes"), "You don't have 15 gold.")
	assert.Equal(t, 10, b.aria.Character.Gold)
	assert.Len(t, tamsin.Character.Wounds, 1)

	// Wounds that change between the offer and the answer are priced again.
	b.aria.Character.Gold = 100
	b.cmd("heal", "wounds")
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 1}}
	assert.Contains(t, b.answer("yes"), "wounds have changed since the price was named")
	assert.Equal(t, 100, b.aria.Character.Gold)
	assert.Nil(t, b.aria.GetPrompt())

	// A fight begun between the offer and the answer refuses, and clears the
	// question.
	b.cmd("heal", "wounds")
	b.aimAt("bandit captain")
	assert.Contains(t, b.answer("yes"), "You can't tend wounds in the middle of a fight.")
	assert.Equal(t, 100, b.aria.Character.Gold)
	assert.Nil(t, b.aria.GetPrompt())
}

// A companion's fight alone refuses heal wounds.
func TestHealWoundsIsRefusedWhileACompanionFights(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.Aggro = nil
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 2}}
	tamsin := b.companion(1)
	tamsin.Character.Aggro = &characters.Aggro{MobInstanceId: b.captain().InstanceId}
	assert.Contains(t, b.cmd("heal", "wounds"), "You can't tend wounds in the middle of a fight.")
	assert.Len(t, b.aria.Character.Wounds, 1)
}

// A player cleric heals to the limit with Minor Heal, spending their own
// mana; tending themselves reads in the second person.
func TestHealWoundsALeaderClericHealsToTheLimit(t *testing.T) {
	b := newBrawl(t)
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Mana = 0
	}
	c := b.aria.Character
	c.SetSkill(`cast`, 1)
	c.SpellBook["heal"] = 1
	c.SpellBook["tend"] = 1
	c.Level = 20 // the heal's level bonus outweighs any dice roll
	c.ManaMax.Value, c.Mana = 40, 40
	c.HealthMax.Value, c.Health = 100, 50
	c.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	out := b.cmd("heal", "wounds")
	assert.Contains(t, out, "You tend your own cut hand, and it draws closed.")
	assert.NotContains(t, out, "first, the worst hurt", "a healer who starts on themselves is not 'first' beside anyone")
	assert.Regexp(t, `You lay glowing hands on yourself\. \(\d+ healed\)`, out)
	// 35a2: each heal is 8 + 2d4 + level/6, as Minor Heal is in battle.
	m := regexp.MustCompile(`You lay glowing hands on yourself\. \((\d+) healed\)`).FindStringSubmatch(out)
	require.NotNil(t, m)
	healed, _ := strconv.Atoi(m[1])
	assert.GreaterOrEqual(t, healed, min(8+2+c.Level/6, c.HealthLimit()-50), "8 + 2d4 + level/6 at level %d", c.Level)
	assert.Empty(t, c.Wounds)
	assert.Less(t, c.Mana, 40, "the leader's own mana is spent")
	assert.Greater(t, c.Health, 50)
	assert.LessOrEqual(t, c.Health, c.HealthLimit())
}

// A cleric with no mana left is named as spent, not as missing.
func TestHealWoundsSpentHealers(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.companion(2).Character.Mana = 0
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 2}}
	assert.Contains(t, b.cmd("heal", "wounds"), "Your healers have no mana left.")
}

// A camp rest spends bandages and splints through company.SpendSupply:
// the cargo first, then the packs; none left reports false.
func TestSpendSupplyThroughTheProvider(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.CompanyCargo = false // the injected provider models legacy split containers
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: bandageItemID, Count: 1}}}
	useCargo(t, cargo)
	b.aria.Character.StoreItem(items.New(bandageItemID))
	b.aria.Character.StoreItem(items.New(splintItemID))
	assert.True(t, domain.SpendSupply(7, wounds.Bandage))
	assert.Equal(t, []int{bandageItemID}, cargo.consumed, "the cargo first")
	assert.True(t, domain.SpendSupply(7, wounds.Bandage), "then the leader's pack")
	assert.False(t, domain.SpendSupply(7, wounds.Bandage), "no bandages left")
	assert.True(t, domain.SpendSupply(7, wounds.Splint), "a splint from the pack")
	assert.False(t, domain.SpendSupply(7, wounds.Splint), "no splints left")
	for _, itm := range b.aria.Character.Items {
		assert.NotContains(t, []int{bandageItemID, splintItemID}, itm.ItemId)
	}
}

// Phase 40a2: the camp fire's firewood and a fishing line come from the
// company through company.CompanyItemCount / SpendCompanyItem.
func TestItemSupplyThroughTheProvider(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.CompanyCargo = false
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: bandageItemID, Count: 1}}}
	useCargo(t, cargo)
	b.aria.Character.StoreItem(items.New(bandageItemID))
	assert.Equal(t, 2, domain.CompanyItemCount(7, bandageItemID), "the cargo and the pack")
	assert.Zero(t, domain.CompanyItemCount(7, 12345))
	assert.True(t, domain.SpendCompanyItem(7, bandageItemID))
	assert.Equal(t, []int{bandageItemID}, cargo.consumed, "the cargo first")
	assert.True(t, domain.SpendCompanyItem(7, bandageItemID), "then the pack")
	assert.False(t, domain.SpendCompanyItem(7, bandageItemID))
	assert.Zero(t, domain.CompanyItemCount(99, bandageItemID), "an unknown leader has nothing")
}
