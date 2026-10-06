package company

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

func equipmentEdge(itm items.Item) (int, int) {
	if itm.GetSpec().Type == items.Weapon && itm.Sharpened() {
		return itm.SharpBonus, itm.SharpStrikes
	}
	return 0, 0
}

func equipmentStats(c *characters.Character, load encumbrance.Load) domain.EquipmentStats {
	weapon, offhand := c.Equipment.Weapon.GetDamage(), c.Equipment.Offhand.GetDamage()
	edgeBonus, edgeStrikes := equipmentEdge(c.Equipment.Weapon)
	offhandEdgeBonus, offhandEdgeStrikes := equipmentEdge(c.Equipment.Offhand)
	offhandDamage := ""
	if c.Equipment.Offhand.GetSpec().Type == items.Weapon {
		offhandDamage = offhand.DiceRoll
	}
	return domain.EquipmentStats{EdgeBonus: edgeBonus, EdgeStrikes: edgeStrikes, OffhandEdgeBonus: offhandEdgeBonus, OffhandEdgeStrikes: offhandEdgeStrikes, WeaponCoat: c.Equipment.Weapon.CoatSummary(time.Now()), OffhandCoat: c.Equipment.Offhand.CoatSummary(time.Now()), Hands: c.HandsRequired(c.Equipment.Weapon), Reach: c.Equipment.Weapon.GetSpec().Reach, Shield: c.HasShield(), HealthMax: c.HealthMax.Value, ManaMax: c.ManaMax.Value, Damage: weapon.DiceRoll, OffhandDamage: offhandDamage, Defense: c.GetDefense(), WornG: c.PersonalGrams(), Burden: c.BurdenWord(), DodgePct: combat.DodgeRetentionPct(c), PackCapacityG: c.Equipment.Pack.CarryBonusGrams(), CapacityG: load.CapacityGrams, CargoG: load.TotalGrams(), Stats: map[string]int{
		"strength": c.Stats.Strength.ValueAdj, "speed": c.Stats.Speed.ValueAdj, "smarts": c.Stats.Smarts.ValueAdj, "vitality": c.Stats.Vitality.ValueAdj, "mysticism": c.Stats.Mysticism.ValueAdj, "perception": c.Stats.Perception.ValueAdj}}
}

// previewKey is c without what changes as rounds pass but no preview reads:
// vitals (availability is keyed separately), cooldowns, timers, the record
// of play, and each buff's tick counters (its presence, stacks and
// permanence remain). The pet is keyed by its buffs and presence instead.
// A ticking field missed here costs a rebuild, never a stale preview.
func previewKey(c *characters.Character) characters.Character {
	k := *c
	k.Health, k.Mana, k.ActionPoints = 0, 0, 0
	k.Experience, k.Gold, k.Bank = 0, 0, 0
	k.RoomId, k.Zone = 0, ""
	k.Cooldowns, k.Timers, k.KD = nil, nil, characters.KDStats{}
	k.MiscData, k.QuestProgress, k.ZonesVisited, k.Settings = nil, nil, nil, nil
	k.Pet = pets.Pet{}
	list := make([]*buffs.Buff, 0, len(c.Buffs.List))
	for _, b := range c.Buffs.List {
		if b == nil {
			continue
		}
		kb := *b
		kb.RoundCounter = 0
		if !kb.Expired() {
			kb.TriggersLeft = 0
		}
		list = append(list, &kb)
	}
	k.Buffs.List = list
	return k
}

func equipmentFits(loadBefore, loadAfter encumbrance.Load) bool {
	return max(0, loadAfter.TotalGrams()-loadAfter.CapacityGrams) <= max(0, loadBefore.TotalGrams()-loadBefore.CapacityGrams)
}

// equipmentViewRefreshRounds bounds how long a cached view may be reused,
// so a change outside its key (such as reloaded item data) still shows.
const equipmentViewRefreshRounds = 15

type equipmentViewEntry struct {
	key   [sha256.Size]byte
	round uint64
	view  domain.EquipmentView
}

// equipmentViewCache holds each leader's last view. The company feed asks
// for one every round, and every preview copies the character, so an
// unchanged leader reuses the last result instead of rebuilding every
// preview. builds counts rebuilds for tests.
type equipmentViewCache struct {
	mu     sync.Mutex
	byUser map[int]equipmentViewEntry
	builds int
}

func (m *CompanyModule) forgetEquipmentView(id int) {
	m.equipmentViews.mu.Lock()
	delete(m.equipmentViews.byUser, id)
	m.equipmentViews.mu.Unlock()
}

// EquipmentView never prepares assets, saves or mutates an item. Catalogue
// choices are exact instances owned by this leader; commands re-resolve them.
// The result is reused while the leader's character (equipment, cargo,
// stats and effects), company load and availability are unchanged, and is
// shared with the cache: callers must not mutate it.
func (m *CompanyModule) EquipmentView(id int) domain.EquipmentView {
	return m.EquipmentViewFor(id, "")
}

// EquipmentViewFor builds previews only for the focus slot, or for every
// slot when focus is "": a rebuild costs a copy of the character per
// preview, and the editor shows one slot's choices at a time.
func (m *CompanyModule) EquipmentViewFor(id int, focus string) domain.EquipmentView {
	return m.EquipmentViewForMember(id, "me", focus)
}

// equipmentMembers lists who the Gear editor can show: the leader, then each
// living companion, with whether each can change gear now.
func (m *CompanyModule) equipmentMembers(u *users.UserRecord) []domain.EquipmentMember {
	out := []domain.EquipmentMember{{Ref: "me", Name: u.Character.Name, Ready: true}}
	rec, _ := m.registry.Get(u.UserId)
	for _, c := range rec.Companions {
		ref := fmt.Sprintf("#%d", c.ID)
		_, _, err := m.equipmentActor(u, ref)
		out = append(out, domain.EquipmentMember{Ref: ref, Name: companionName(c), Ready: err == nil})
	}
	return out
}

// memberCharacter is a member's character for display only, whether or not
// the member can change gear now.
func (m *CompanyModule) memberCharacter(u *users.UserRecord, member string) *characters.Character {
	key, err := m.resolveMemberKey(u.UserId, member)
	if err != nil || key == domain.LeaderMemberKey {
		return u.Character
	}
	id, _ := domain.CompanionIDFromMemberKey(key)
	if inst, ok := m.instance(u.UserId, id); ok {
		if live := mobs.GetInstance(inst); live != nil {
			return &live.Character
		}
	}
	return nil
}

// EquipmentViewForMember is EquipmentViewFor for one named member ("me" or
// "#N"): the same slots, exact cargo choices and Current -> After previews,
// run against that member's gear and the leader's shared cargo.
func (m *CompanyModule) EquipmentViewForMember(id int, member, focus string) domain.EquipmentView {
	member = strings.ToLower(strings.TrimSpace(member))
	if member == "" || member == "leader" || member == "self" {
		member = "me"
	}
	out := domain.EquipmentView{Member: member, Slots: []domain.EquipmentSlot{}}
	u := users.GetByUserId(id)
	if u == nil || u.Character == nil {
		out.Reason = "Character unavailable."
		return out
	}
	if !u.Character.CompanyCargo {
		out.Reason = "Shared cargo is not ready."
		return out
	}
	out.Members = m.equipmentMembers(u)
	load, known := encumbrance.CurrentLoad(id)
	_, actor, err := m.equipmentActor(u, member)
	if err != nil {
		out.Reason = err.Error()
	} else if err = m.persistenceAvailable(); err != nil {
		out.Reason = err.Error()
	} else {
		if rec, ok := m.registry.Get(id); ok && rec.AssetOperation != nil {
			out.Reason = "Equipment operation awaits recovery."
		} else if !known {
			out.Reason = "Cargo capacity unavailable."
		} else {
			out.Available = true
		}
	}
	if actor == nil {
		actor = m.memberCharacter(u, member)
	}
	if actor == nil {
		out.Available, out.Reason = false, "That member is not here."
		return out
	}
	out.Current = equipmentStats(actor, load)
	// A member's items are the shared cargo, which every proposal takes
	// from u instead.
	// The cache is keyed on the character without them or its round ticks,
	// plus the cargo; a rebuild marshals it once more, in full, as the source
	// of every preview's clone.
	bare := *actor
	bare.Items = nil
	keyed := previewKey(&bare)
	keyRaw, keyErr := yaml.Marshal(&keyed)
	cargoRaw, cargoErr := yaml.Marshal(u.Character.Items)
	if keyErr != nil || cargoErr != nil {
		out.Available, out.Reason = false, "Equipment preview unavailable."
		return out
	}
	hpPerLevel := bare.HealthGainPerLevel()
	archetype := bare.ArchetypeID() // 35a2: the class decides the gear rules
	key := sha256.Sum256(append(append(keyRaw, cargoRaw...), fmt.Sprintf("|%+v|%t|%t|%s|%t|%v|%s|%g|%s|%s|%v", load, known, out.Available, out.Reason, bare.Pet.Exists() && !bare.Pet.IsMissing(), bare.Pet.GetBuffs(), focus, hpPerLevel, archetype, member, out.Members)...))
	round := util.GetRoundCount()
	m.equipmentViews.mu.Lock()
	cached, hit := m.equipmentViews.byUser[id]
	m.equipmentViews.mu.Unlock()
	if hit && cached.key == key && round-cached.round < equipmentViewRefreshRounds {
		return cached.view
	}
	// Only a rebuild needs the full character, as every preview's source.
	raw, err := yaml.Marshal(&bare)
	if err != nil {
		out.Available, out.Reason = false, "Equipment preview unavailable."
		return out
	}
	clone := func() (*characters.Character, error) { return characterFrom(raw, hpPerLevel, archetype) }
	preview := func(verb, slot string, itm items.Item) domain.EquipmentChoice {
		choice := domain.EquipmentChoice{Ref: itm.ShorthandId(), Label: domain.PlainLabel(itm)}
		args := []string{verb, member, itm.ShorthandId(), slot}
		if verb == "remove" {
			args = []string{verb, member, slot, itm.ShorthandId()}
		}
		choice.Command = "company " + strings.Join(args, " ")
		proposed, cargo, _, displaced, err := proposalFrom(u, actor, args, clone)
		if err != nil {
			choice.Reason = err.Error()
			return choice
		}
		before, after, ok := equipmentLoadFrom(load, known, actor, proposed, u.Character.Items, cargo)
		for _, old := range displaced {
			if old.ItemId > 0 {
				choice.Returned = append(choice.Returned, domain.PlainLabel(old))
			}
		}
		if verb == "remove" {
			choice.Returned = append(choice.Returned, domain.PlainLabel(itm))
		}
		stats := equipmentStats(proposed, after)
		choice.After = &stats
		if !out.Available {
			choice.Reason = out.Reason
		} else if !ok {
			choice.Reason = "Cargo capacity unavailable."
		} else if !(verb == "remove" && slot == string(items.Pack)) && !equipmentFits(before, after) {
			choice.Reason = "That change would exceed company cargo capacity. Drop or sell cargo, or assign a larger pack first."
		} else {
			choice.Allowed = true
		}
		return choice
	}
	for _, slot := range characters.AllSlots() {
		entry := domain.EquipmentSlot{Slot: string(slot), Label: strings.TrimSuffix(characters.SlotLabel(slot), ":"), Choices: []domain.EquipmentChoice{}}
		entry.Pending = focus != "" && focus != string(slot)
		if itm := actor.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			equipped := domain.EquipmentChoice{Ref: itm.ShorthandId(), Label: domain.PlainLabel(*itm)}
			entry.Equipped = &equipped
			if !entry.Pending {
				removal := preview("remove", string(slot), *itm)
				entry.Remove = &removal
			}
		}
		if entry.Pending {
			out.Slots = append(out.Slots, entry)
			continue
		}
		for _, itm := range u.Character.Items {
			spec := itm.GetSpec()
			if spec.Type == slot || (slot == items.Offhand && spec.Type == items.Weapon) {
				entry.Choices = append(entry.Choices, preview("equip", string(slot), itm))
			}
		}
		out.Slots = append(out.Slots, entry)
	}
	online := map[int]bool{id: true}
	for _, uid := range users.GetOnlineUserIds() {
		online[uid] = true
	}
	m.equipmentViews.mu.Lock()
	if m.equipmentViews.byUser == nil {
		m.equipmentViews.byUser = map[int]equipmentViewEntry{}
	}
	// A missed despawn must not keep an offline leader's view: prune on
	// every rebuild, as the GMCP feed prunes each round.
	for uid := range m.equipmentViews.byUser {
		if !online[uid] {
			delete(m.equipmentViews.byUser, uid)
		}
	}
	m.equipmentViews.byUser[id] = equipmentViewEntry{key: key, round: round, view: out}
	m.equipmentViews.builds++
	m.equipmentViews.mu.Unlock()
	return out
}
