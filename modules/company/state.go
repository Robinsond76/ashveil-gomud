package company

// Phase 22b: durable companion level and gear. The record's State is the
// source of truth; the live mob is rebuilt from it on every restore and
// snapshotted back at the seams below. See modules/company/AGENTS.md.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ensureState returns the companion's state, initializing a legacy
// companion's from its template first. The upgrade is saved before it is
// returned; a failed save restores the old record and returns the error,
// so no template gear is handed out without a durable record of it. A nil
// state with no error means the template is unavailable.
func (m *CompanyModule) ensureState(leaderUserID int, companion domain.Companion) (*domain.MemberState, error) {
	if companion.State != nil {
		s := companion.State.Clone()
		return &s, nil
	}
	state, ok := m.runtime.TemplateState(companion.MobTemplateID)
	if !ok {
		return nil, nil
	}
	before, _ := m.registry.Get(leaderUserID)
	if err := m.registry.SetState(leaderUserID, companion.ID, state); err != nil {
		return nil, err
	}
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return nil, err
	}
	return &state, nil
}

// refreshSnapshot copies a tracked companion's live mob into its record.
// It reports whether anything was recorded; the caller saves.
func (m *CompanyModule) refreshSnapshot(leaderUserID, companionID int) bool {
	instanceID, tracked := m.instance(leaderUserID, companionID)
	if !tracked || !m.runtime.IsLive(instanceID) {
		return false
	}
	state, ok := m.runtime.Snapshot(instanceID)
	if !ok {
		return false
	}
	return m.registry.SetState(leaderUserID, companionID, state) == nil
}

// refreshAll snapshots every live tracked companion (plugin OnSave:
// autosave, shutdown, copyover).
func (m *CompanyModule) refreshAll() {
	if m.persistenceAvailable() != nil {
		return
	}
	for leaderUserID, byCompanion := range m.instances {
		for companionID := range byCompanion {
			m.refreshSnapshot(leaderUserID, companionID)
		}
	}
}

// onItemOwnership records a companion's gear right after it changes.
func (m *CompanyModule) onItemOwnership(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.ItemOwnership)
	if !ok || evt.MobInstanceId <= 0 || m.persistenceAvailable() != nil {
		return events.Continue
	}
	leaderUserID, companionID, found := m.companionForInstance(evt.MobInstanceId)
	if !found {
		return events.Continue
	}
	if m.refreshSnapshot(leaderUserID, companionID) {
		if err := m.save(); err != nil {
			mudlog.Error("company: save companion gear", "leader", leaderUserID, "companion", companionID, "error", err)
		}
	}
	return events.Continue
}

// onPlayerDespawn records a leaving leader's companions and removes their
// live mobs. Left behind, a reverted companion could be killed for gear
// that its record would restore again.
func (m *CompanyModule) onPlayerDespawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok || m.persistenceAvailable() != nil {
		return events.Continue
	}
	byCompanion := m.instances[evt.UserId]
	if len(byCompanion) == 0 {
		return events.Continue
	}
	changed := false
	for companionID := range byCompanion {
		if m.refreshSnapshot(evt.UserId, companionID) {
			changed = true
		}
	}
	if changed {
		if err := m.save(); err != nil {
			// The snapshot stays in memory for the next autosave.
			mudlog.Error("company: save on leader leave", "leader", evt.UserId, "error", err)
		}
	}
	for companionID, instanceID := range byCompanion {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(evt.UserId, instanceID)
		}
		m.clearInstance(evt.UserId, companionID)
	}
	return events.Continue
}

// recordCompanionDeath clears a dead companion's gear (it dropped into the
// corpse or was lost with the body) and keeps its level, so the next
// restore can't bring the gear back.
func (m *CompanyModule) recordCompanionDeath(leaderUserID, companionID, level int) {
	if m.persistenceAvailable() != nil {
		return
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	for _, c := range record.Companions {
		if c.ID != companionID || c.State == nil {
			continue
		}
		state := c.State.Clone()
		state.ClearGear()
		if level > 0 {
			state.Level = level
		}
		if err := m.registry.SetState(leaderUserID, companionID, state); err != nil {
			return
		}
		if err := m.save(); err != nil {
			mudlog.Error("company: save companion death", "leader", leaderUserID, "companion", companionID, "error", err)
		}
		return
	}
}

func companionLevel(c domain.Companion) string {
	if c.State == nil || c.State.Level < 1 {
		return "level ?"
	}
	return "level " + strconv.Itoa(c.State.Level)
}

// gearView lists a companion's recorded gear.
func (m *CompanyModule) gearView(leaderUserID int, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions."
	}
	c, ok := resolveCompanion(record, selector)
	if !ok {
		return fmt.Sprintf(`No companion matches "%s".`, strings.TrimSpace(selector))
	}
	// Show the live mob's gear when it is out, so a change the mob made
	// itself (equipping from its pack) is visible at once.
	if m.refreshSnapshot(leaderUserID, c.ID) {
		record, _ = m.registry.Get(leaderUserID)
		c, _ = resolveCompanion(record, "#"+strconv.Itoa(c.ID))
	}
	name := templateName(c.MobTemplateID, strconv.Itoa(c.MobTemplateID))
	lines := []string{fmt.Sprintf("#%d %s, %s", c.ID, name, companionLevel(c))}
	if c.State == nil {
		lines = append(lines, "  Gear is recorded the next time this companion is restored.")
		return strings.Join(lines, "\n")
	}
	worn := []string{}
	for _, slot := range characters.AllSlots() {
		if itm := c.State.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			worn = append(worn, fmt.Sprintf("  %-8s %s", characters.SlotLabel(slot), itemName(*itm)))
		}
	}
	if len(worn) == 0 {
		lines = append(lines, "  Wearing nothing.")
	} else {
		lines = append(lines, "  Wearing:")
		lines = append(lines, worn...)
	}
	if len(c.State.Items) == 0 {
		lines = append(lines, "  Carrying nothing.")
	} else {
		names := make([]string, 0, len(c.State.Items))
		for _, itm := range c.State.Items {
			names = append(names, itemName(itm))
		}
		lines = append(lines, "  Carrying: "+strings.Join(names, ", "))
	}
	return strings.Join(lines, "\n")
}

func itemName(itm items.Item) string {
	if spec := items.GetItemSpec(itm.ItemId); spec != nil {
		return itm.DisplayName()
	}
	return fmt.Sprintf("item %d", itm.ItemId)
}
