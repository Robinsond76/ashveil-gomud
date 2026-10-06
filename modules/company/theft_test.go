package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// thiefCargo adds the cargo keeper's withdraw to the fake cargo.
type thiefCargo struct {
	*fakeCargo
	withdrawn []int
}

func (k *thiefCargo) DepositCargo(int, string, []encumbrance.CargoStack) error { return nil }
func (k *thiefCargo) WithdrawCargo(_, itemID, count int) error {
	for i, s := range k.stacks {
		if s.ItemId == itemID && s.Count >= count {
			k.withdrawn = append(k.withdrawn, itemID)
			if k.stacks[i].Count -= count; k.stacks[i].Count == 0 {
				k.stacks = append(k.stacks[:i], k.stacks[i+1:]...)
			}
			return nil
		}
	}
	return encumbrance.ErrInsufficientCargo
}

// Phase 40a4: camp thieves take loose goods from the cargo and the
// companions' packs, never the leader's pack, equipment, quest items, keys
// or protected gear.
func TestCampTheftTakesLooseGoodsAndSparesTheRest(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.CompanyCargo = false
	meat := spec(t, items.ItemSpec{ItemId: 989101, Name: "raw game meat", Weight: 300, Type: items.Food})
	token := spec(t, items.ItemSpec{ItemId: 989102, Name: "oath token", Weight: 10, Type: items.Junk, QuestToken: "oath"})
	key := spec(t, items.ItemSpec{ItemId: 989103, Name: "iron key", Weight: 10, Type: items.Key})
	tent := spec(t, items.ItemSpec{ItemId: 989104, Name: "oiled canvas tent", Weight: 9000, Type: items.Junk})
	mine := spec(t, items.ItemSpec{ItemId: 989105, Name: "leader's lucky coin", Weight: 10, Type: items.Junk})
	cargo := &thiefCargo{fakeCargo: &fakeCargo{stacks: []encumbrance.CargoStack{
		{ItemId: meat.ItemId, Count: 8}, {ItemId: token.ItemId, Count: 1}, {ItemId: tent.ItemId, Count: 1}}}}
	useThieves(t, cargo)
	b.aria.Character.StoreItem(mine)
	tamsin := b.companion(1)
	tamsin.Character.Items = append(tamsin.Character.Items, key, items.Item{ItemId: meat.ItemId})

	protect := func(id int) bool { return id == tent.ItemId }
	losses := domain.CampTheft(7, 100, 50, func(int) int { return 0 }, protect)
	require.NotEmpty(t, losses)
	for _, l := range losses {
		assert.Equal(t, meat.ItemId, l.ItemID, "only the meat is loose, unprotected and not a key or token")
		assert.Equal(t, "raw game meat", l.Name)
	}
	total := 0
	for _, l := range losses {
		total += l.Count
	}
	assert.Equal(t, 9, total, "all 100%: the cargo's 8 and the companion's 1")
	assert.Equal(t, []int{meat.ItemId}, uniq(cargo.withdrawn))
	for _, s := range cargo.stacks {
		assert.NotEqual(t, meat.ItemId, s.ItemId)
	}
	assert.Equal(t, 1, module.CompanyItemCount(7, token.ItemId), "the quest token is still there")
	assert.Equal(t, 1, module.CompanyItemCount(7, tent.ItemId), "so is the tent")
	assert.Equal(t, 1, module.CompanyItemCount(7, mine.ItemId), "and the leader's pack")
	for _, itm := range tamsin.Character.Items {
		assert.NotEqual(t, meat.ItemId, itm.ItemId, "the companion's meat is gone")
	}
	var kept bool
	for _, itm := range tamsin.Character.Items {
		kept = kept || itm.ItemId == key.ItemId
	}
	assert.True(t, kept, "the companion keeps the key")
}

func TestCampTheftIsASmallShareAndCapped(t *testing.T) {
	b := newBrawl(t)
	_ = b
	meat := spec(t, items.ItemSpec{ItemId: 989111, Name: "raw game meat", Weight: 300, Type: items.Food})
	cargo := &thiefCargo{fakeCargo: &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: meat.ItemId, Count: 30}}}}
	useThieves(t, cargo)

	losses := domain.CampTheft(7, 10, 4, func(int) int { return 0 }, nil)
	require.Len(t, losses, 1)
	assert.Equal(t, 3, losses[0].Count, "10% of 30")
	assert.Equal(t, 27, cargo.stacks[0].Count)

	cargo.stacks = []encumbrance.CargoStack{{ItemId: meat.ItemId, Count: 100}}
	losses = domain.CampTheft(7, 10, 4, func(int) int { return 0 }, nil)
	assert.Equal(t, 4, losses[0].Count, "capped at four")

	cargo.stacks = []encumbrance.CargoStack{{ItemId: meat.ItemId, Count: 2}}
	losses = domain.CampTheft(7, 10, 4, func(int) int { return 0 }, nil)
	assert.Equal(t, 1, losses[0].Count, "at least one")

	cargo.stacks = nil
	assert.Empty(t, domain.CampTheft(7, 10, 4, func(int) int { return 0 }, nil), "nothing to take")
	assert.Empty(t, domain.CampTheft(99, 10, 4, func(int) int { return 0 }, nil), "an unknown leader")
}

func uniq(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func useThieves(t *testing.T, c *thiefCargo) {
	t.Helper()
	encumbrance.SetProvider(c)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })
}
