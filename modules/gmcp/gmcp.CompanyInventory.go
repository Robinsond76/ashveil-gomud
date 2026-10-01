package gmcp

// Phase 32g: "Company.Inventory", the web client's Inventory tab: the load
// split, every member's worn and carried items (the player first), the
// horses, and the cargo, as `company inventory` shows them. Each item
// carries the reference a command resolves to exactly that item. Built on
// the game loop, sent by the company feed only when it changes. See
// docs/designs/2026-09-29-phase-32g-company-dock-design.md.

import (
	"encoding/json"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type inventoryItem struct {
	Ref     string `json:"ref"`
	Name    string `json:"name"`  // plain, as a command names it
	Label   string `json:"label"` // as players see it, without markup
	Grams   int    `json:"grams"`
	Count   int    `json:"count"`
	Uses    int    `json:"uses"`
	UsesMax int    `json:"uses_max"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Slot    string `json:"slot,omitempty"`
}

type inventoryMember struct {
	Available  bool            `json:"available"`
	Key        string          `json:"key"`
	Name       string          `json:"name"`
	Fallen     bool            `json:"fallen"`
	Unrecorded bool            `json:"unrecorded"`
	Grams      int             `json:"grams"`
	Pack       string          `json:"pack"`
	PackBonusG int             `json:"pack_bonus_g"`
	Worn       []inventoryItem `json:"worn"`
	Carried    []inventoryItem `json:"carried"`
}

type inventoryHorse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Saddle    string `json:"saddle"`
	CapacityG int    `json:"capacity_g"`
	// Rides is a saddled riding horse, which carries a rider.
	Rides bool `json:"rides"`
}

type inventoryLoad struct {
	TotalG          int `json:"total_g"`
	CapacityG       int `json:"capacity_g"`
	MemberCapacityG int `json:"member_capacity_g"`
	MountCapacityG  int `json:"mount_capacity_g"`
	CargoG          int `json:"cargo_g"`
}

type inventoryPayload struct {
	Shared   bool              `json:"shared"`
	Treasury int               `json:"treasury"`
	AutoLoot bool              `json:"autoloot"`
	Load     *inventoryLoad    `json:"load"`
	Members  []inventoryMember `json:"members"`
	// CompanionsKnown is false when the company can't be read; Members
	// then holds the player only.
	CompanionsKnown bool             `json:"companions_known"`
	Horses          []inventoryHorse `json:"horses"`
	Cargo           []inventoryItem  `json:"cargo"`
}

// inventorySources are what the payload reads; natives unless a test
// swaps them.
type inventorySources struct {
	companions func(leaderUserID int) ([]company.InventoryMember, bool)
	herd       func(leaderUserID int) []mount.HorseView
	cargo      func(leaderUserID int) []encumbrance.CargoStack
	load       func(leaderUserID int) (encumbrance.Load, bool)
}

func nativeInventorySources() inventorySources {
	return inventorySources{
		companions: company.CompanyInventory,
		herd:       mount.HerdOf,
		cargo:      encumbrance.CargoContents,
		load:       encumbrance.CurrentLoad,
	}
}

func inventoryItemOf(i company.InventoryItem) inventoryItem {
	return inventoryItem{Ref: i.Ref, Name: i.Name, Label: i.Label, Grams: i.Grams, Count: i.Count, Uses: i.Uses, UsesMax: i.UsesMax,
		Type: i.Type, Subtype: i.Subtype, Slot: i.Slot}
}

func inventoryItems(in []company.InventoryItem) []inventoryItem {
	out := make([]inventoryItem, 0, len(in))
	for _, i := range in {
		out = append(out, inventoryItemOf(i))
	}
	return out
}

func inventoryMemberOf(m company.InventoryMember) inventoryMember {
	return inventoryMember{Available: m.Available, Key: string(m.Key), Name: m.Name, Fallen: m.Fallen, Unrecorded: m.Unrecorded, Grams: m.Grams,
		Pack: m.Pack, PackBonusG: m.PackBonusGrams, Worn: inventoryItems(m.Worn), Carried: inventoryItems(m.Carried)}
}

// cargoItem is a cargo stack. Its reference is "!<item id>", which
// `cargo take` matches (it prefers a partly used stack of that item).
func cargoItem(s encumbrance.CargoStack) inventoryItem {
	itm := items.Item{ItemId: s.ItemId}
	out := inventoryItem{Ref: "!" + strconv.Itoa(s.ItemId), Name: itm.Name(), Label: company.PlainLabel(itm), Count: s.Count, Uses: s.Uses}
	if spec := items.GetItemSpec(s.ItemId); spec != nil {
		out.Grams, out.UsesMax, out.Type, out.Subtype = spec.Weight, spec.Uses, string(spec.Type), string(spec.Subtype)
		if out.Uses == 0 {
			out.Uses = spec.Uses // a full stack
		}
	}
	return out
}

func buildInventoryPayload(user *users.UserRecord, src inventorySources) inventoryPayload {
	uid := user.UserId
	leader := company.InventoryMemberOf(company.LeaderMemberKey, user.Character.Name,
		company.MemberState{Items: user.Character.Items, Equipment: user.Character.Equipment})
	p := inventoryPayload{Shared: user.Character.CompanyCargo, Treasury: user.Character.Gold, AutoLoot: user.Character.AutoLoot, Members: []inventoryMember{inventoryMemberOf(leader)}, Horses: []inventoryHorse{}, Cargo: []inventoryItem{}}
	if load, ok := src.load(uid); ok {
		p.Load = &inventoryLoad{TotalG: load.TotalGrams(), CapacityG: load.CapacityGrams, MemberCapacityG: load.MemberCapacityGrams,
			MountCapacityG: load.MountCapacityGrams, CargoG: load.CargoGrams}
	}
	if companions, ok := src.companions(uid); ok {
		p.CompanionsKnown = true
		for _, m := range companions {
			p.Members = append(p.Members, inventoryMemberOf(m))
		}
	}
	for _, h := range src.herd(uid) {
		p.Horses = append(p.Horses, inventoryHorse{ID: h.ID, Name: h.Name, Kind: string(h.Kind), Saddle: h.Saddle,
			CapacityG: h.CapacityGrams, Rides: h.Kind == mount.KindRiding && h.Saddle != ""})
	}
	if p.Shared {
		p.Cargo = inventoryItems(leader.Carried)
		leader.Carried = []company.InventoryItem{}
		leader.Grams = user.Character.PersonalGrams()
		p.Members[0] = inventoryMemberOf(leader)
	} else {
		for _, s := range src.cargo(uid) {
			p.Cargo = append(p.Cargo, cargoItem(s))
		}
	}
	return p
}

// inventoryExtra is the feed's Company.Inventory message.
func inventoryExtra() companyExtra {
	return companyExtra{module: "Company.Inventory", build: func(user *users.UserRecord) []byte {
		data, err := json.Marshal(buildInventoryPayload(user, nativeInventorySources()))
		if err != nil {
			return nil
		}
		return data
	}}
}
