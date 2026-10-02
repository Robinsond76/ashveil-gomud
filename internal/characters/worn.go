package characters

import "github.com/GoMudEngine/GoMud/internal/items"

type Worn struct {
	Pack    items.Item `yaml:"pack,omitempty"`
	Weapon  items.Item `yaml:"weapon,omitempty"`
	Offhand items.Item `yaml:"offhand,omitempty"`
	Head    items.Item `yaml:"head,omitempty"`
	Neck    items.Item `yaml:"neck,omitempty"`
	Body    items.Item `yaml:"body,omitempty"`
	Belt    items.Item `yaml:"belt,omitempty"`
	Gloves  items.Item `yaml:"gloves,omitempty"`
	Ring    items.Item `yaml:"ring,omitempty"`
	Legs    items.Item `yaml:"legs,omitempty"`
	Feet    items.Item `yaml:"feet,omitempty"`
}

// Get returns a pointer to the item in the given slot, or nil for an
// unrecognized slot type. This is the single place that maps an ItemType to a
// Worn struct field; add new slots here and nowhere else.
func (w *Worn) Get(slot items.ItemType) *items.Item {
	switch slot {
	case items.Pack:
		return &w.Pack
	case items.Weapon:
		return &w.Weapon
	case items.Offhand:
		return &w.Offhand
	case items.Head:
		return &w.Head
	case items.Neck:
		return &w.Neck
	case items.Body:
		return &w.Body
	case items.Belt:
		return &w.Belt
	case items.Gloves:
		return &w.Gloves
	case items.Ring:
		return &w.Ring
	case items.Legs:
		return &w.Legs
	case items.Feet:
		return &w.Feet
	}
	return nil
}

// Set places item into the given slot. Does nothing for an unrecognized slot.
func (w *Worn) Set(slot items.ItemType, item items.Item) {
	switch slot {
	case items.Pack:
		w.Pack = item
	case items.Weapon:
		w.Weapon = item
	case items.Offhand:
		w.Offhand = item
	case items.Head:
		w.Head = item
	case items.Neck:
		w.Neck = item
	case items.Body:
		w.Body = item
	case items.Belt:
		w.Belt = item
	case items.Gloves:
		w.Gloves = item
	case items.Ring:
		w.Ring = item
	case items.Legs:
		w.Legs = item
	case items.Feet:
		w.Feet = item
	}
}

// AllSlots returns every equipment slot in canonical display order.
// Delegates to items.AllEquipSlots() so the single source of truth lives
// alongside the ItemType constants.
func AllSlots() []items.ItemType {
	return items.AllEquipSlots()
}

// WeaponSlots returns the slots that hold weapons.
func WeaponSlots() []items.ItemType {
	return items.WeaponSlots()
}

// ArmorSlots returns every equipment slot except Weapon.
func ArmorSlots() []items.ItemType {
	return items.ArmorSlots()
}

// SlotLabel returns the short display label (with trailing colon) for a slot,
// e.g. items.Head -> "Head:". Used by UI code so label strings are not
// scattered across rendering packages.
func SlotLabel(slot items.ItemType) string {
	switch slot {
	case items.Pack:
		return "Pack:"
	case items.Weapon:
		return "Weapon:"
	case items.Offhand:
		return "Offhand:"
	case items.Head:
		return "Head:"
	case items.Neck:
		return "Neck:"
	case items.Body:
		return "Body:"
	case items.Belt:
		return "Belt:"
	case items.Gloves:
		return "Gloves:"
	case items.Ring:
		return "Ring:"
	case items.Legs:
		return "Legs:"
	case items.Feet:
		return "Feet:"
	}
	return string(slot) + ":"
}

// GetAllSlotTypes returns all slot names as strings.
// Kept for backward compatibility; prefer AllSlots() for typed access.
func GetAllSlotTypes() []string {
	slots := AllSlots()
	out := make([]string, len(slots))
	for i, s := range slots {
		out[i] = string(s)
	}
	return out
}

func (w *Worn) StatMod(stat ...string) int {
	total := 0
	for _, slot := range AllSlots() {
		if slot != items.Pack {
			total += w.Get(slot).StatMod(stat...)
		}
	}
	return total
}

func (w *Worn) EnableAll() {
	for _, slot := range AllSlots() {
		if w.Get(slot).ItemId < 0 {
			w.Set(slot, items.Item{})
		}
	}
}

func (w *Worn) GetAllItems() []items.Item {
	out := []items.Item{}
	for _, slot := range AllSlots() {
		if itm := w.Get(slot); itm.ItemId > 0 {
			out = append(out, *itm)
		}
	}
	return out
}
