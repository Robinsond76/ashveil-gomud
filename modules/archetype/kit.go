package archetype

// Phase 22a: the archetype step of character creation and starter kits.
//
// A committed choice records the kit it owes (Registry.Kits) in the same
// save. The kit itself is granted into the backpack together with a claim
// marker on the character (MiscData), so the items and the marker are
// always persisted by the same user save. The grant runs after the choice
// and again on every spawn; it only gives when a kit is owed and the
// marker is absent, so each character receives its kit exactly once.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// kitMarkerKey is the character MiscData key recording the archetype whose
// kit this character received.
const kitMarkerKey = "archetype-kit"

func nativeItemName(itemID int) (string, bool) {
	spec := items.GetItemSpec(itemID)
	if spec == nil {
		return "", false
	}
	return spec.Name, true
}

func nativeSaveUser(user *users.UserRecord) error {
	return users.SaveUser(*user)
}

// resolveKit drops kit item ids that don't name a loaded item.
func (m *ArchetypeModule) resolveKit(a archetypes.Archetype) []int {
	if m.itemName == nil {
		return a.Kit
	}
	out := make([]int, 0, len(a.Kit))
	for _, id := range a.Kit {
		if _, ok := m.itemName(id); !ok {
			mudlog.Warn("archetype: unknown kit item", "archetype", a.ID, "item", id)
			continue
		}
		out = append(out, id)
	}
	return out
}

// kitNames lists a kit's item names in order, folding repeats into
// "name (x2)". It must be called outside the module lock.
func (m *ArchetypeModule) kitNames(a archetypes.Archetype) []string {
	counts := map[int]int{}
	order := []int{}
	for _, id := range a.Kit {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}
	out := make([]string, 0, len(order))
	for _, id := range order {
		name := fmt.Sprintf("item %d", id)
		if m.itemName != nil {
			if n, ok := m.itemName(id); ok {
				name = n
			}
		}
		if counts[id] > 1 {
			name = fmt.Sprintf("%s (x%d)", name, counts[id])
		}
		out = append(out, name)
	}
	return out
}

// kitLine is the one-line kit preview, or "" for an empty kit.
func (m *ArchetypeModule) kitLine(a archetypes.Archetype) string {
	if len(a.Kit) == 0 {
		return ""
	}
	return "Starter kit: " + strings.Join(m.kitNames(a), ", ") + "."
}

// grantKit gives the user their owed starter kit unless they already
// received one, then saves the user. It returns player-facing text, or ""
// when nothing was granted. It runs on the game loop and never holds the
// module lock while touching the user.
func (m *ArchetypeModule) grantKit(user *users.UserRecord) string {
	if user == nil {
		return ""
	}
	m.mu.Lock()
	owed, isOwed := m.registry.Kits[user.UserId]
	a, known := m.table.Get(owed)
	m.mu.Unlock()
	if !isOwed || !known || len(a.Kit) == 0 {
		return ""
	}
	if user.Character.GetMiscData(kitMarkerKey) != nil {
		return ""
	}
	for _, id := range a.Kit {
		itm := items.New(id)
		if itm.ItemId < 1 {
			mudlog.Warn("archetype: kit item unavailable", "archetype", a.ID, "item", id)
			continue
		}
		user.Character.StoreItem(itm)
	}
	user.Character.SetMiscData(kitMarkerKey, a.ID)
	if m.saveUser != nil {
		// The items and the marker are both in memory and travel together
		// in the next autosave, logout, or copyover save if this one fails.
		if err := m.saveUser(user); err != nil {
			mudlog.Error("archetype: save after kit", "user", user.UserId, "error", err)
		}
	}
	return fmt.Sprintf(`You receive the %s starter kit: %s. Use "equip <item>" to ready it.`, a.Name, strings.Join(m.kitNames(a), ", "))
}

// --- archetypes.Creator -------------------------------------------------------

var _ archetypes.Creator = (*ArchetypeModule)(nil)

// CreationChoices lists every configured archetype with its kit preview.
func (m *ArchetypeModule) CreationChoices() []archetypes.Choice {
	m.mu.Lock()
	list := m.table.List()
	m.mu.Unlock()
	out := make([]archetypes.Choice, 0, len(list))
	for _, a := range list {
		out = append(out, archetypes.Choice{
			ID:          a.ID,
			Name:        a.Name,
			Description: a.Description,
			Skills:      append([]string(nil), a.Skills...),
			Kit:         m.kitNames(a),
		})
	}
	return out
}

// ChooseAtCreation commits a confirmed choice for an online user.
func (m *ArchetypeModule) ChooseAtCreation(userID int, archetypeID string) (string, bool) {
	user := users.GetByUserId(userID)
	if user == nil {
		return "", false
	}
	return m.chooseResult(user, archetypeID, true)
}
