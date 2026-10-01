package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
)

type Corpse struct {
	ClaimUserId  int // Shared battle loot belongs to this deterministic claimant; 0 is unclaimed.
	UserId       int
	MobId        int
	Character    characters.Character
	RoundCreated uint64
	Prunable     bool         // Whether it can be removed
	Items        []items.Item // Held items when CorpseItems config is enabled
	Gold         int          // Held gold when CorpseItems config is enabled
}

func (c *Corpse) Update(roundNow uint64, decayRate string) {

	if c.Prunable {
		return
	}

	if decayRate == `` {
		decayRate = `1 week`
	}

	gd := gametime.GetDate(c.RoundCreated)
	decayRound := gd.AddPeriod(decayRate)

	// Has enough time passed to do the respawn?
	if roundNow >= decayRound {
		c.Prunable = true
	}

}

func (c *Corpse) AddItem(i items.Item) {
	c.Items = append(c.Items, i)
}

func (c *Corpse) RemoveItem(i items.Item) {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			break
		}
	}
}

func (c *Corpse) FindItem(itemName string) (items.Item, bool) {
	closeMatch, matchItem := items.FindMatchIn(itemName, c.Items...)
	if matchItem.ItemId != 0 {
		return matchItem, true
	}
	if closeMatch.ItemId != 0 {
		return closeMatch, true
	}
	return items.Item{}, false
}

func (c *Corpse) HasItems() bool {
	return len(c.Items) > 0 || c.Gold > 0
}
