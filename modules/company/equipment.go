package company

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"gopkg.in/yaml.v2"
)

const equipmentUsage = "Usage: company equipment | company treasury | company equip [member] [item] | company remove [member] [slot] | company compare [member] [item]"

// PrepareAssets finishes an interrupted write before any further player command.
// It also pools a living companion's old carried items/gold exactly once.
func (m *CompanyModule) PrepareAssets(id int) error {
	u := users.GetByUserId(id)
	if u == nil || u.Character == nil {
		return nil
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.finishAssets(u); err != nil {
		return err
	}
	if err := encumbrance.UnifyCargo(id); err != nil {
		return err
	}
	// The feature is active only when the cargo provider supports shared cargo.
	if !u.Character.CompanyCargo {
		return nil
	}
	rec, _ := m.registry.Get(id)
	rec.LeaderUserID = id
	before, _ := m.registry.Get(id)
	cargo := append([]items.Item(nil), u.Character.Items...)
	gold, changed := u.Character.Gold, false
	for i, c := range rec.Companions {
		if c.Dead() {
			continue
		}
		if inst, tracked := m.instance(id, c.ID); tracked && m.runtime.IsLive(inst) {
			if m.runtime.CharmedByOther(id, inst) {
				continue
			}
			if state, ok := m.runtime.Snapshot(inst); ok {
				c.State = &state
			}
		}
		if c.State == nil {
			continue
		}
		rec.Companions[i].State = c.State
		if len(c.State.Items) == 0 && c.State.Gold == 0 {
			continue
		}
		st := c.State.Clone()
		cargo = append(cargo, st.Items...)
		gold += st.Gold
		st.Items, st.Gold = nil, 0
		rec.Companions[i].State = &st
		changed = true
	}
	worn := u.Character.Equipment
	if id := m.starterPackID(); id > 0 {
		packChanged, err := m.migratePacks(&rec, &cargo, &worn, id)
		if err != nil {
			return err
		}
		changed = changed || packChanged
	}
	if !changed {
		return nil
	}
	return m.commitAssets(u, before, rec, cargo, worn, gold)
}

// commitAssets writes the company gear and resulting leader assets together.
// The company operation is authoritative until the user file acknowledges it.
func (m *CompanyModule) commitAssets(u *users.UserRecord, before, rec domain.Record, cargo []items.Item, worn characters.Worn, gold int) error {
	rec.AssetOperation = &domain.AssetOperation{ID: uuid.New().String(), Items: cargo, Equipment: worn, Gold: gold}
	rec.LeaderUserID = u.UserId
	m.registry.Put(rec)
	if err := m.save(); err != nil {
		if before.LeaderUserID == 0 {
			m.registry.Remove(u.UserId)
		} else {
			m.registry.Put(before)
		}
		return err
	}
	m.installLiveGear(u.UserId, rec)
	return m.finishAssets(u)
}

func (m *CompanyModule) installLiveGear(id int, rec domain.Record) {
	for _, c := range rec.Companions {
		if c.State == nil || c.Dead() {
			continue
		}
		inst, ok := m.instance(id, c.ID)
		if !ok || m.runtime.CharmedByOther(id, inst) {
			continue
		}
		if live := mobs.GetInstance(inst); live != nil {
			st := c.State.Clone()
			live.Character.Equipment, live.Character.Items, live.Character.Gold = st.Equipment, st.Items, st.Gold
			live.Character.Validate(true)
		}
	}
}

func (m *CompanyModule) finishAssets(u *users.UserRecord) error {
	rec, ok := m.registry.Get(u.UserId)
	if !ok || rec.AssetOperation == nil {
		return nil
	}
	op := rec.AssetOperation
	if m.saveUser == nil {
		return fmt.Errorf("company: no user asset store")
	}
	if u.Character.CompanyAssetOp != op.ID {
		before := domain.MemberState{Items: u.Character.Items, Equipment: u.Character.Equipment, Gold: u.Character.Gold}.Clone()
		oldMode, oldOp := u.Character.CompanyCargo, u.Character.CompanyAssetOp
		final := domain.MemberState{Items: op.Items, Equipment: op.Equipment}.Clone()
		u.Character.Items, u.Character.Equipment, u.Character.Gold = final.Items, final.Equipment, op.Gold
		u.Character.CompanyCargo, u.Character.CompanyAssetOp = true, op.ID
		if err := m.saveUser(u); err != nil {
			u.Character.Items, u.Character.Equipment, u.Character.Gold = before.Items, before.Equipment, before.Gold
			u.Character.CompanyCargo, u.Character.CompanyAssetOp = oldMode, oldOp
			return fmt.Errorf("company: equipment operation awaits recovery: %w", err)
		}
		u.Character.Validate(true)
		events.AddToQueue(events.CompanyAssetsChanged{UserId: u.UserId})
	}
	rec.AssetOperation = nil
	m.registry.Put(rec)
	if err := m.save(); err != nil {
		rec.AssetOperation = op
		m.registry.Put(rec)
		return fmt.Errorf("company: equipment acknowledgement awaits recovery: %w", err)
	}
	return nil
}

// equipmentActor resolves the durable company key, then verifies its live owner,
// life, location and battle state. Names alone never authorize an operation.
func (m *CompanyModule) equipmentActor(u *users.UserRecord, selector string) (domain.MemberKey, *characters.Character, error) {
	key, err := m.resolveMemberKey(u.UserId, selector)
	if err != nil {
		return "", nil, err
	}
	if u.Character.Health < 1 || actionpolicy.InBattle(u) {
		return "", nil, fmt.Errorf(actionpolicy.BattleUnderWay)
	}
	if key == domain.LeaderMemberKey {
		return key, u.Character, nil
	}
	id, _ := domain.CompanionIDFromMemberKey(key)
	rec, _ := m.registry.Get(u.UserId)
	for _, c := range rec.Companions {
		if c.ID != id {
			continue
		}
		inst, tracked := m.instance(u.UserId, id)
		live := mobs.GetInstance(inst)
		if c.Dead() || c.PendingReturn || !tracked || live == nil || live.Character.Health < 1 || !m.runtime.WithLeader(u.UserId, inst) || !m.runtime.IsAttached(u.UserId, inst) || !live.Character.IsCharmed(u.UserId) || m.runtime.CharmedByOther(u.UserId, inst) {
			return "", nil, fmt.Errorf("That member must be alive and present with your company.")
		}
		if live.Character.Aggro != nil {
			return "", nil, fmt.Errorf(actionpolicy.BattleUnderWay)
		}
		return key, &live.Character, nil
	}
	return "", nil, domain.ErrUnknownMember
}

func cloneCharacter(c *characters.Character) (*characters.Character, error) {
	raw, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	var out characters.Character
	if err = yaml.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	out.Validate(true)
	return &out, nil
}

func (m *CompanyModule) equipmentCommand(u *users.UserRecord, args []string) string {
	if err := m.PrepareAssets(u.UserId); err != nil {
		return err.Error()
	}
	if !u.Character.CompanyCargo {
		return "Shared company cargo is unavailable."
	}
	if args[0] == "treasury" {
		return fmt.Sprintf("Company treasury: %d gold. Purchases and trading use this pool; banked gold stays in the bank.", u.Character.Gold)
	}
	if args[0] == "equipment" {
		lines := []string{equipmentUsage, "Company cargo (exact item references):"}
		for _, itm := range u.Character.Items {
			lines = append(lines, fmt.Sprintf("  %s  %s (%.1f kg)", itm.ShorthandId(), itemName(itm), float64(itm.Weight())/1000))
		}
		return strings.Join(lines, "\n")
	}
	if len(args) < 3 {
		return equipmentUsage
	}
	key, actor, err := m.equipmentActor(u, args[1])
	if err != nil {
		return err.Error()
	}
	proposed, err := cloneCharacter(actor)
	if err != nil {
		return err.Error()
	}
	cargo := append([]items.Item(nil), u.Character.Items...)
	var itm items.Item
	var displaced []items.Item
	if args[0] == "remove" {
		slot := items.ItemType(args[2])
		ptr := proposed.Equipment.Get(slot)
		if ptr == nil || ptr.ItemId < 1 {
			return "That member has no equipment in that slot."
		}
		itm = *ptr
		if itm.IsRemoveLocked() || itm.IsCursed() {
			return "That equipment is bound or cursed and cannot be removed."
		}
		if !proposed.RemoveFromBody(itm) {
			return "That equipment cannot be removed."
		}
		cargo = append(cargo, itm)
	} else {
		ref := strings.Join(args[2:], " ")
		close, exact := items.FindMatchIn(ref, cargo...)
		itm = exact
		if itm.ItemId < 1 {
			itm = close
		}
		if itm.ItemId < 1 {
			return "That item is no longer in your company cargo."
		}
		// Refuse ambiguous names rather than silently assign the wrong enchanted copy.
		if strings.Contains(ref, ":") && strings.HasPrefix(ref, "!") && ref != itm.ShorthandId() {
			return "That exact cargo item is no longer available."
		}
		if !strings.Contains(ref, ":") {
			matches := 0
			for _, item := range cargo {
				a, b := items.FindMatchIn(ref, item)
				if a.ItemId > 0 || b.ItemId > 0 {
					matches++
				}
			}
			if matches > 1 {
				return "Several cargo items match. Use the exact reference from company equipment."
			}
		}
		// Cursed offhands must not be displaced through the two-hand path.
		for _, slot := range characters.AllSlots() {
			old := proposed.Equipment.Get(slot)
			if old.ItemId > 0 && old.IsCursed() {
				// Wear's ordinary guards handle other slots; guard displaced offhands too.
				if slot == items.Offhand && itm.GetSpec().Type == items.Weapon && proposed.HandsRequired(itm) == 2 {
					return "The offhand is cursed and cannot be removed."
				}
			}
		}
		var success bool
		var reason string
		displaced, success, reason = proposed.Wear(itm)
		if success {
			for _, old := range displaced {
				if old.ItemId > 0 && old.IsCursed() {
					return "That equipment is cursed and cannot be removed."
				}
			}
		}
		if !success {
			return reason
		}
		for i, item := range cargo {
			if item.Equals(itm) {
				cargo = append(cargo[:i], cargo[i+1:]...)
				break
			}
		}
		for _, old := range displaced {
			if old.ItemId > 0 {
				cargo = append(cargo, old)
			}
		}
	}
	proposed.Validate(true)
	loadBefore, loadAfter, hasLoad := equipmentLoad(u.UserId, actor, proposed, u.Character.Items, cargo)
	lossAllowed := args[0] == "remove" && itm.GetSpec().Type == items.Pack
	if args[0] != "compare" && hasLoad && !lossAllowed && max(0, loadAfter.TotalGrams()-loadAfter.CapacityGrams) > max(0, loadBefore.TotalGrams()-loadBefore.CapacityGrams) {
		return "That change would exceed company cargo capacity. Drop or sell cargo, or assign a larger pack first."
	}
	if args[0] == "compare" {
		w := itm.GetSpec()
		shield := func(c *characters.Character) bool {
			return c.Equipment.Offhand.ItemId > 0 && c.Equipment.Offhand.GetSpec().Type == items.Offhand
		}
		role := "fighter"
		if st, ok := m.strategyForEquipment(u.UserId, key); ok {
			role = st
		}
		lines := []string{fmt.Sprintf("%s: %s; role %s.", actor.Name, itemName(itm), role), fmt.Sprintf("Weapon: %s, %d hands, reach %t; price %d gold.", w.Damage.DiceRoll, proposed.HandsRequired(itm), w.Reach, w.Value), fmt.Sprintf("Protection: %d -> %d. Shield: %t -> %t.", actor.GetDefense(), proposed.GetDefense(), shield(actor), shield(proposed)), fmt.Sprintf("Worn load: %.1f -> %.1f kg; burden: %s -> %s.", float64(actor.PersonalGrams())/1000, float64(proposed.PersonalGrams())/1000, actor.BurdenWord(), proposed.BurdenWord())}
		if hasLoad {
			lines = append(lines, fmt.Sprintf("Company capacity: %.1f -> %.1f kg; cargo: %.1f -> %.1f kg.", float64(loadBefore.CapacityGrams)/1000, float64(loadAfter.CapacityGrams)/1000, float64(loadBefore.TotalGrams())/1000, float64(loadAfter.TotalGrams())/1000))
			if loadAfter.TotalGrams() > loadAfter.CapacityGrams {
				lines = append(lines, "This would leave the company over capacity; walking remains possible.")
			}
		}
		lines = append(lines, "This is a comparison, not an automatic upgrade. Check reach, shield and burden for the member's role.")
		return strings.Join(lines, "\n")
	}
	rec, _ := m.registry.Get(u.UserId)
	rec.LeaderUserID = u.UserId
	before, _ := m.registry.Get(u.UserId)
	for i, c := range rec.Companions {
		if inst, ok := m.instance(u.UserId, c.ID); ok && !c.Dead() && !m.runtime.CharmedByOther(u.UserId, inst) {
			if st, ok := m.runtime.Snapshot(inst); ok {
				rec.Companions[i].State = &st
			}
		}
	}
	worn := u.Character.Equipment
	if key == domain.LeaderMemberKey {
		worn = proposed.Equipment
	} else {
		id, _ := domain.CompanionIDFromMemberKey(key)
		for i, c := range rec.Companions {
			if c.ID != id {
				continue
			}
			st, ok := m.runtime.Snapshot(m.instances[u.UserId][id])
			if !ok {
				return "That member is no longer available."
			}
			st.Equipment = proposed.Equipment
			rec.Companions[i].State = &st
		}
	}
	if err := m.commitAssets(u, before, rec, cargo, worn, u.Character.Gold); err != nil {
		return err.Error()
	}
	actor.CancelBuffsWithFlag("hidden")
	evt := events.EquipmentChange{UserId: u.UserId, ItemsWorn: []items.Item{itm}, ItemsRemoved: displaced}
	if args[0] == "remove" {
		evt.ItemsWorn = nil
		evt.ItemsRemoved = []items.Item{itm}
	}
	if key != domain.LeaderMemberKey {
		id, _ := domain.CompanionIDFromMemberKey(key)
		evt.UserId = 0
		evt.MobInstanceId = m.instances[u.UserId][id]
	}
	if key == domain.LeaderMemberKey && args[0] != "remove" && len(itm.GetSpec().WornBuffIds) > 0 {
		for _, buff := range actor.Buffs.List {
			if buff.OnStartWaiting {
				if _, err := scripting.TryBuffScriptEvent("onStart", u.UserId, 0, buff.BuffId); err == nil {
					actor.TrackBuffStarted(buff.BuffId)
				}
			}
		}
	}
	events.AddToQueue(evt)
	return fmt.Sprintf("%s's equipment updated. Shared cargo retains every displaced item.", actor.Name)
}

// strategyForEquipment keeps the comparison's role tied to the saved strategy.
func (m *CompanyModule) strategyForEquipment(id int, key domain.MemberKey) (string, bool) {
	archetype := ""
	if key == domain.LeaderMemberKey {
		if u := users.GetByUserId(id); u != nil {
			archetype, _ = archetypes.PlayerArchetype(id)
		}
	} else {
		rec, _ := m.registry.Get(id)
		cid, _ := domain.CompanionIDFromMemberKey(key)
		for _, c := range rec.Companions {
			if c.ID == cid {
				archetype = c.Archetype
			}
		}
	}
	s := strategy.For(id, string(key), archetype)
	return string(s.Role), true
}

func (m *CompanyModule) ManageEquipment(id int, verb, rest string) (bool, string) {
	u := users.GetByUserId(id)
	if u == nil || !u.Character.CompanyCargo {
		return false, ""
	}
	if verb == "equip" && rest == "all" || verb == "gearup" {
		return true, "Assign equipment deliberately with company equip; compare choices with company compare."
	}
	if verb == "remove" {
		if rest == "all" {
			return false, ""
		}
		itm, found := u.Character.FindOnBody(rest)
		if !found {
			return true, "You are not wearing that item."
		}
		if strings.HasPrefix(rest, "!") && strings.Contains(rest, ":") && rest != itm.ShorthandId() {
			return true, "That exact worn item is no longer available."
		}
		for _, slot := range characters.AllSlots() {
			if u.Character.Equipment.Get(slot).Equals(itm) {
				return true, m.equipmentCommand(u, []string{"remove", "leader", string(slot)})
			}
		}
	}
	return true, m.equipmentCommand(u, []string{verb, "leader", rest})
}

func (m *CompanyModule) queueAutoLoot(round uint64) {
	for _, u := range users.GetAllActiveUsers() {
		if !u.Character.AutoLoot || actionpolicy.InBattle(u) || u.Character.Health < 1 {
			continue
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		if room == nil {
			continue
		}
		ready := false
		for i := range room.Corpses {
			c := &room.Corpses[i]
			if c.ClaimUserId != u.UserId || !c.CanLoot(u.UserId, round) {
				continue
			}
			if c.Gold > 0 {
				ready = true
			}
			for _, itm := range c.Items {
				if _, full := encumbrance.WouldExceed(u.UserId, encumbrance.AddedGrams(u.UserId, u.Character.Items, itm)); !full {
					ready = true
				}
			}
		}
		if ready {
			u.Command("loot own", -1)
		}
	}
}
