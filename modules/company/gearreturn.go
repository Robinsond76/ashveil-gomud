package company

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

// Companion gear return: when a companion leaves the company for good
// (dismissal, desertion, or a death whose rescue time ran out), the gear the
// leader gave it goes back to the leader's cargo. Its default gear, the
// items its template minted, stays with the template so a recruit cannot be
// farmed for loot.

// defaultGearCounts is how many of each item a fresh recruit of the template
// holds on its own: its worn and carried template gear, plus the starter
// pack the company hands every member.
func (m *CompanyModule) defaultGearCounts(templateID int) map[int]int {
	counts := map[int]int{}
	if spec := mobs.GetMobSpec(mobs.MobId(templateID)); spec != nil {
		for _, slot := range characters.AllSlots() {
			if itm := spec.Character.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
				counts[itm.ItemId]++
			}
		}
		for _, itm := range spec.Character.Items {
			if itm.ItemId > 0 {
				counts[itm.ItemId]++
			}
		}
	}
	if id := m.starterPackID(); id > 0 {
		counts[id]++
	}
	return counts
}

// currentState is a companion's gear state right now: the live mob's when it
// is attached, else the saved one.
func (m *CompanyModule) currentState(leaderID int, c domain.Companion) *domain.MemberState {
	if inst, ok := m.instance(leaderID, c.ID); ok && !c.Dead() && m.runtime.IsLive(inst) && !m.runtime.CharmedByOther(leaderID, inst) {
		if st, ok := m.runtime.Snapshot(inst); ok {
			return &st
		}
	}
	return c.State
}

// returnableGear is what a companion holds beyond its defaults, and its
// gold. Each default item is set aside once by item id, so a given piece of
// the same kind as a default is the one kept with the template; any worn
// or carried extra, and every doll's gear but its cudgel, goes back.
func (m *CompanyModule) returnableGear(leaderID int, c domain.Companion) ([]items.Item, int) {
	st := m.currentState(leaderID, c)
	if st == nil {
		return nil, 0
	}
	defaults := m.defaultGearCounts(c.MobTemplateID)
	var out []items.Item
	take := func(itm items.Item) {
		if itm.ItemId <= 0 {
			return
		}
		if defaults[itm.ItemId] > 0 {
			defaults[itm.ItemId]--
			return
		}
		out = append(out, itm)
	}
	for _, slot := range characters.AllSlots() {
		if itm := st.Equipment.Get(slot); itm != nil {
			take(*itm)
		}
	}
	for _, itm := range st.Items {
		take(itm)
	}
	for _, d := range st.Dolls {
		skipCudgel := true
		for _, slot := range characters.AllSlots() {
			itm := d.Equipment.Get(slot)
			if itm == nil || itm.ItemId <= 0 {
				continue
			}
			if skipCudgel && itm.ItemId == characters.DollStarterWeapon {
				skipCudgel = false
				continue
			}
			out = append(out, *itm)
		}
	}
	cloned := make([]items.Item, len(out))
	for i, itm := range out {
		cloned[i] = domain.MemberState{Items: []items.Item{itm}}.Clone().Items[0]
	}
	return cloned, st.Gold
}

// stageGearReturn puts returned gear and gold into the leader's cargo as an
// asset operation on the registry record, so the next m.save writes it in
// the same file as the roster change that frees it: a crash either keeps the
// companion with its gear or has the gear in the cargo, never both and
// never neither. finishGearReturn then installs it on the leader's file.
// Cargo capacity never blocks the return (an over-full company still
// walks), so the report says when it leaves the company over capacity.
func (m *CompanyModule) stageGearReturn(u *users.UserRecord, returned []items.Item, gold int) {
	rec, _ := m.registry.Get(u.UserId)
	rec.LeaderUserID = u.UserId
	cargo := append(domain.MemberState{Items: u.Character.Items}.Clone().Items, returned...)
	rec.AssetOperation = &domain.AssetOperation{ID: uuid.New().String(), Items: cargo, Equipment: u.Character.Equipment, Gold: u.Character.Gold + gold}
	m.registry.Put(rec)
}

// finishGearReturn installs a staged return on the leader's own file. A
// failure leaves the operation pending, and the leader's next command
// finishes it (PrepareAssets).
func (m *CompanyModule) finishGearReturn(u *users.UserRecord) {
	if err := m.finishAssets(u); err != nil {
		mudlog.Error("company: gear return awaits recovery", "leader", u.UserId, "error", err)
	}
}

// gearReturnLeader is the leader whose cargo can take returned gear: online
// and on shared cargo. Any other leader gets nothing staged.
func (m *CompanyModule) gearReturnLeader(leaderID int) *users.UserRecord {
	u := users.GetByUserId(leaderID)
	if u == nil || u.Character == nil || !u.Character.CompanyCargo {
		return nil
	}
	return u
}

// gearReturnLine is the leader's report of what came back.
func gearReturnLine(leaderID int, name string, returned []items.Item, gold int) string {
	if len(returned) == 0 && gold == 0 {
		return ""
	}
	names := make([]string, 0, len(returned))
	for _, itm := range returned {
		names = append(names, itemName(itm))
	}
	parts := []string{}
	if len(names) > 0 {
		parts = append(parts, strings.Join(names, ", "))
	}
	if gold > 0 {
		parts = append(parts, fmt.Sprintf("%d gold", gold))
	}
	line := fmt.Sprintf("%s's gear returns to your cargo: %s.", name, strings.Join(parts, " and "))
	if load, ok := encumbrance.CurrentLoad(leaderID); ok && load.TotalGrams() > load.CapacityGrams {
		line += " Your company is now over capacity; walking stays possible, but drop or sell cargo, or give gear to a pack horse."
	}
	return line
}

// gearStatLine is a member's gear in one line for company status and
// company gear: what it wields and wears, and what that does (Phase 48).
// With a live character it reports damage, protection and burden.
func gearStatLine(c *characters.Character, equipment characters.Worn) string {
	names := []string{}
	for _, slot := range characters.AllSlots() {
		if itm := equipment.Get(slot); itm != nil && itm.ItemId > 0 && slot != items.Pack {
			names = append(names, itemName(*itm))
		}
	}
	line := "nothing worn"
	if len(names) > 0 {
		line = strings.Join(names, ", ")
	}
	if c != nil {
		st := equipmentStats(c, encumbrance.Load{})
		line += fmt.Sprintf("; damage %s, protection %d, %s, dodge %d%%", st.Damage, st.Defense, st.Burden, st.DodgePct)
	}
	return line
}

// statChanges describes what a gear change does to a member's numbers, as
// "Protection 3 -> 7, damage 1d4 -> 1d6+1", or "" when nothing moved.
func statChanges(before, after *characters.Character) string {
	b, a := equipmentStats(before, encumbrance.Load{}), equipmentStats(after, encumbrance.Load{})
	parts := []string{}
	add := func(label, was, now string) {
		if was != now {
			parts = append(parts, fmt.Sprintf("%s %s -> %s", label, was, now))
		}
	}
	itoa := func(n int) string { return fmt.Sprintf("%d", n) }
	add("damage", b.Damage, a.Damage)
	if b.OffhandDamage != a.OffhandDamage {
		add("offhand damage", b.OffhandDamage+"", a.OffhandDamage+"")
	}
	add("protection", itoa(b.Defense), itoa(a.Defense))
	add("dodge", itoa(b.DodgePct)+"%", itoa(a.DodgePct)+"%")
	add("burden", b.Burden, a.Burden)
	add("health", itoa(b.HealthMax), itoa(a.HealthMax))
	add("mana", itoa(b.ManaMax), itoa(a.ManaMax))
	for _, name := range []string{"strength", "speed", "smarts", "vitality", "mysticism", "perception"} {
		add(name, itoa(b.Stats[name]), itoa(a.Stats[name]))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// statusGearLine is a companion's gear line for company status: from the
// live mob when it is out (with its numbers), else from the saved record.
func (m *CompanyModule) statusGearLine(leaderID int, c domain.Companion) string {
	if c.Dead() {
		return ""
	}
	if inst, ok := m.instance(leaderID, c.ID); ok {
		if live := mobs.GetInstance(inst); live != nil {
			return gearStatLine(&live.Character, live.Character.Equipment)
		}
	}
	if c.State == nil {
		return ""
	}
	return gearStatLine(nil, c.State.Equipment)
}
