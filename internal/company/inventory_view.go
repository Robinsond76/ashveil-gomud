package company

// Phase 32g: a company member's gear as data for the web client's
// Inventory tab (modules/gmcp's Company.Inventory). The text view is
// modules/company's `company inventory`.

import (
	"regexp"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

var markup = regexp.MustCompile(`<[^>]*>`)

// PlainLabel is ItemLabel without its markup, for the web client, which
// shows server strings as text.
func PlainLabel(itm items.Item) string {
	return markup.ReplaceAllString(ItemLabel(itm), "")
}

// InventoryItem is one item as the Inventory tab shows it.
type InventoryItem struct {
	// Ref names exactly this item to a command (items.FindMatchIn's
	// "!<id>:<uuid>" form), so two same-named items are never confused.
	Ref string
	// Name is the item's plain name, as a command names it (look, equip,
	// eat); Label is how players see it (an edge, a quest mark), with the
	// markup taken out (32g review finding 3).
	Name    string
	Label   string
	Grams   int // one item's weight
	Count   int
	Uses    int
	UsesMax int // the spec's uses; 0 or 1 for an item without uses
	Type    string
	Subtype string
	// Slot is the worn slot, "" for a carried item.
	Slot string
}

// InventoryMember is one member's gear.
type InventoryMember struct {
	Available bool // Alive, present and controlled; exact management is possible.
	Key       MemberKey
	Name      string
	Fallen    bool
	// Unrecorded is a companion whose gear isn't known yet.
	Unrecorded     bool
	Grams          int
	Pack           string // the pack that counts, "" for none
	PackBonusGrams int
	Worn, Carried  []InventoryItem
}

// ItemLabel is an item's name as players see it, with its edge.
func ItemLabel(itm items.Item) string {
	if edge := itm.EdgeLabel(); edge != "" {
		return itm.DisplayName() + " " + edge
	}
	return itm.DisplayName()
}

func inventoryItem(itm items.Item, slot string) InventoryItem {
	out := InventoryItem{Ref: itm.ShorthandId(), Name: itm.Name(), Label: PlainLabel(itm), Grams: itm.Weight(), Count: 1, Uses: itm.Uses, Slot: slot}
	if spec := items.GetItemSpec(itm.ItemId); spec != nil {
		out.UsesMax, out.Type, out.Subtype = spec.Uses, string(spec.Type), string(spec.Subtype)
	}
	return out
}

// InventoryMemberOf is a member's gear from its state.
func InventoryMemberOf(key MemberKey, name string, s MemberState) InventoryMember {
	m := InventoryMember{Key: key, Name: name, Worn: []InventoryItem{}, Carried: []InventoryItem{}}
	for _, slot := range characters.AllSlots() {
		if itm := s.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			m.Worn = append(m.Worn, inventoryItem(*itm, string(slot)))
			m.Grams += itm.Weight()
		}
	}
	for _, itm := range s.Items {
		if itm.ItemId > 0 {
			m.Carried = append(m.Carried, inventoryItem(itm, ""))
			m.Grams += itm.Weight()
		}
	}
	if pack, grams := BestPack(s.Items); grams > 0 {
		m.Pack, m.PackBonusGrams = PlainLabel(pack), grams
	}
	return m
}

// InventoryProvider is optionally implemented by the registered
// FormationProvider (Phase 32g). Game loop only; it never writes.
type InventoryProvider interface {
	// CompanyInventory lists the leader's companions' gear by ID; ok is
	// false when the company can't be read.
	CompanyInventory(leaderUserID int) ([]InventoryMember, bool)
}

// CompanyInventory is a leader's companions' gear; ok is false without a
// provider or while the company can't be read.
func CompanyInventory(leaderUserID int) ([]InventoryMember, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	ip, ok := p.(InventoryProvider)
	if !ok {
		return nil, false
	}
	return ip.CompanyInventory(leaderUserID)
}
