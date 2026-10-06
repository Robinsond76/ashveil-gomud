package company

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/modconfig"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"

	domain "github.com/GoMudEngine/GoMud/internal/company"
)

var _ domain.RelocationProvider = (*CompanyModule)(nil)

// strayRounds is how many rounds in a row a companion may stand away from
// its leader before it is separated (Phase 33h3): long enough for an
// ordinary follow to land.
const strayRounds = 2

// RelocateCompany implements company.RelocationProvider (Phase 25a; 33h3):
// after the leader was moved from originRoomID into roomID by anything but
// an ordinary exit, every living companion whose mob stands in either room
// moves into roomID, out of any fight; every other living one out in the
// world is separated. Records, formation, gear, alignment, and service are
// unchanged for those who move. The dead (no live mob) and the fled are
// untouched. Companions are handled by number.
func (m *CompanyModule) RelocateCompany(leaderUserID, originRoomID, roomID int) int {
	byCompanion := m.instances[leaderUserID]
	companionIDs := make([]int, 0, len(byCompanion))
	for companionID := range byCompanion {
		companionIDs = append(companionIDs, companionID)
	}
	slices.Sort(companionIDs)
	moved := 0
	for _, companionID := range companionIDs {
		instanceID := byCompanion[companionID]
		if !m.runtime.IsAttached(leaderUserID, instanceID) {
			continue
		}
		at, _, standing := m.runtime.Standing(instanceID)
		if !standing {
			continue // dying: its death settles it
		}
		if at == originRoomID || at == roomID {
			if m.runtime.Relocate(instanceID, roomID) {
				moved++
			}
			continue
		}
		if err := m.separate(leaderUserID, companionID, domain.SeparatedByMove); err != nil {
			mudlog.Warn("company: separate on move", "leader", leaderUserID, "companion", companionID, "error", err)
		}
	}
	return moved
}

var _ domain.DefeatProvider = (*CompanyModule)(nil)

// WoundCompany implements company.DefeatProvider (Phase 53): a defeat that
// leaves the company for dead leaves each living companion who is with the
// leader a lasting wound. The wound lives on the companion's own character,
// so it shows in the panels and heals like any other; the snapshot is
// refreshed so a restart keeps it.
func (m *CompanyModule) WoundCompany(leaderUserID, pct int, roll func(n int) int) []string {
	byCompanion := m.instances[leaderUserID]
	companionIDs := make([]int, 0, len(byCompanion))
	for companionID := range byCompanion {
		companionIDs = append(companionIDs, companionID)
	}
	slices.Sort(companionIDs)
	var names []string
	for _, companionID := range companionIDs {
		instanceID := byCompanion[companionID]
		if !m.runtime.IsAttached(leaderUserID, instanceID) {
			continue
		}
		if _, _, standing := m.runtime.Standing(instanceID); !standing {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil {
			continue
		}
		mob.Character.AddWound(wounds.Beaten(mob.Character.HealthMax.Value, pct, roll))
		m.refreshSnapshot(leaderUserID, companionID)
		names = append(names, mob.Character.Name)
	}
	return names
}

// separate takes a living companion off the map (Phase 33h3): its gear,
// wounds, and vitals are snapshotted and saved with its separation before
// its mob is removed, as morale flight does. A failed save leaves it where
// it stands, to be tried again.
func (m *CompanyModule) separate(leaderUserID, companionID int, reason string) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	before, ok := m.registry.Get(leaderUserID)
	if !ok {
		return domain.ErrUnknownMember
	}
	c, ok := findCompanion(before, companionID)
	if !ok || c.Dead() || c.PendingReturn || c.Separated() {
		return domain.ErrUnknownMember
	}
	instanceID, tracked := m.instance(leaderUserID, companionID)
	if !tracked || !m.runtime.IsLive(instanceID) {
		return domain.ErrUnknownMember
	}
	m.refreshSnapshot(leaderUserID, companionID)
	record, _ := m.registry.Get(leaderUserID)
	rounds := m.separationRounds()
	for i := range record.Companions {
		if record.Companions[i].ID == companionID {
			record.Companions[i].Separation = &domain.Separation{Reason: reason, RoundsLeft: rounds}
		}
	}
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	m.runtime.Detach(leaderUserID, instanceID)
	m.clearInstance(leaderUserID, companionID)
	if m.strays != nil {
		delete(m.strays[leaderUserID], companionID)
	}
	m.chemistryWorld().Tell(leaderUserID, separationLine(nameOf(c, m.companionLabel(leaderUserID, companionID)), reason, rounds))
	return nil
}

// separationLine tells the leader who was separated, why, and when to
// expect them.
func separationLine(name, reason string, rounds int) string {
	why := "was not with you and is separated from the company"
	if reason == domain.SeparatedStray {
		why = "has lost sight of you and is separated from the company"
	}
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi> %s, and will find the way back to you in %s, once you are out of any fight (<ansi fg="command">help separation</ansi>).`,
		name, why, roundsText(rounds))
}

// roundsText says a span of rounds in game-loop seconds, roughly.
func roundsText(rounds int) string {
	seconds := rounds * int(configs.GetTimingConfig().RoundSeconds)
	switch {
	case seconds <= 0:
		return "a moment"
	case seconds < 90:
		return fmt.Sprintf("about %d seconds", seconds)
	default:
		return fmt.Sprintf("about %d minutes", (seconds+30)/60)
	}
}

// separationRounds is the configured catch-up time.
func (m *CompanyModule) separationRounds() int {
	if m.plug != nil {
		if n, ok := modconfig.Int(m.plug.Config.Get("SeparationRounds")); ok && n >= 1 && n <= 10000 {
			return n
		}
	}
	return domain.DefaultSeparationRounds
}

// sweepStrays separates any living companion that has stood away from its
// online leader, out of a fight, for strayRounds rounds in a row (Phase
// 33h3): a door it could not pass, an admin's move, anything a relocation
// did not settle. Runs once a round.
func (m *CompanyModule) sweepStrays() {
	if m.persistenceAvailable() != nil {
		return
	}
	if m.strays == nil {
		m.strays = map[int]map[int]int{}
	}
	leaders := make([]int, 0, len(m.instances))
	for leaderUserID := range m.instances {
		leaders = append(leaders, leaderUserID)
	}
	slices.Sort(leaders)
	for leaderUserID := range m.strays {
		if _, ok := m.instances[leaderUserID]; !ok {
			delete(m.strays, leaderUserID)
		}
	}
	for _, leaderUserID := range leaders {
		if !m.chemistryWorld().LeaderOnline(leaderUserID) {
			delete(m.strays, leaderUserID)
			continue
		}
		byCompanion := m.instances[leaderUserID]
		companionIDs := make([]int, 0, len(byCompanion))
		for companionID := range byCompanion {
			companionIDs = append(companionIDs, companionID)
		}
		slices.Sort(companionIDs)
		for _, companionID := range companionIDs {
			instanceID := byCompanion[companionID]
			_, fighting, standing := m.runtime.Standing(instanceID)
			away := standing && !fighting && m.runtime.IsAttached(leaderUserID, instanceID) && !m.runtime.WithLeader(leaderUserID, instanceID)
			if !away {
				delete(m.strays[leaderUserID], companionID)
				continue
			}
			if m.strays[leaderUserID] == nil {
				m.strays[leaderUserID] = map[int]int{}
			}
			m.strays[leaderUserID][companionID]++
			if m.strays[leaderUserID][companionID] < strayRounds {
				continue
			}
			if err := m.separate(leaderUserID, companionID, domain.SeparatedStray); err != nil {
				mudlog.Warn("company: separate stray", "leader", leaderUserID, "companion", companionID, "error", err)
			}
		}
	}
}

// tickSeparations counts each online leader's separated companions down
// one round and brings back those whose time is up once the leader is free
// (Phase 33h3). The count changes in memory and reaches disk with the next
// company save, so a crash can only lengthen it. It never reads or
// advances the world clock.
func (m *CompanyModule) tickSeparations() {
	if m.persistenceAvailable() != nil {
		return
	}
	leaders := []int{}
	for leaderUserID, record := range m.registry.Companies {
		if hasSeparated(record) {
			leaders = append(leaders, leaderUserID)
		}
	}
	slices.Sort(leaders)
	for _, leaderUserID := range leaders {
		if !m.chemistryWorld().LeaderOnline(leaderUserID) {
			continue
		}
		record, _ := m.registry.Get(leaderUserID)
		due := false
		for i, c := range record.Companions {
			if !c.Separated() {
				continue
			}
			if c.Separation.RoundsLeft > 0 {
				record.Companions[i].Separation.RoundsLeft--
			}
			due = due || record.Companions[i].Separation.RoundsLeft == 0
		}
		m.registry.Put(record)
		if !due {
			continue
		}
		roomID, free := m.leaderFreeFor(leaderUserID)
		if !free {
			continue
		}
		if err := m.rejoin(leaderUserID, roomID); err != nil {
			mudlog.Warn("company: rejoin", "leader", leaderUserID, "error", err)
		}
	}
}

// rejoin clears every due separation in one save and only then spawns the
// returning companions beside the leader. A failed save changes nothing.
// One whose spawn fails is put back as separated and due (saved with the
// next company save), so the next free round tries again; only those who
// arrive are announced.
func (m *CompanyModule) rejoin(leaderUserID, roomID int) error {
	before, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	record, _ := m.registry.Get(leaderUserID)
	var due []domain.Companion
	for i, c := range record.Companions {
		if c.Separated() && c.Separation.RoundsLeft == 0 {
			due = append(due, c)
			record.Companions[i].Separation = nil
		}
	}
	if len(due) == 0 {
		return nil
	}
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	err := m.restoreForLeader(leaderUserID, roomID)
	current, _ := m.registry.Get(leaderUserID)
	retry := false
	for _, c := range due {
		if _, tracked := m.instance(leaderUserID, c.ID); tracked {
			m.chemistryWorld().Tell(leaderUserID, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> finds the way back and rejoins you.`, nameOf(c, m.companionLabel(leaderUserID, c.ID))))
			continue
		}
		for i := range current.Companions {
			if current.Companions[i].ID == c.ID && !current.Companions[i].Dead() {
				current.Companions[i].Separation = &domain.Separation{Reason: c.Separation.Reason}
				retry = true
			}
		}
	}
	if retry {
		m.registry.Put(current)
	}
	return err
}

func hasSeparated(record domain.Record) bool {
	for _, c := range record.Companions {
		if c.Separated() {
			return true
		}
	}
	return false
}

// leaderFreeFor reports the leader's room and whether a separated
// companion may rejoin them there: online, alive, and not in a battle, on
// a journey, or resting at camp.
func (m *CompanyModule) leaderFreeFor(leaderUserID int) (int, bool) {
	if m.leaderFree != nil {
		return m.leaderFree(leaderUserID)
	}
	return nativeLeaderFree(leaderUserID)
}

func nativeLeaderFree(leaderUserID int) (int, bool) {
	u := users.GetByUserId(leaderUserID)
	if u == nil || u.Character == nil || u.Character.Health < 1 || u.Character.Aggro != nil {
		return 0, false
	}
	if _, busy := battle.Current(leaderUserID); busy {
		return 0, false
	}
	if blocked, _ := expedition.MovementBlocked(leaderUserID); blocked {
		return 0, false
	}
	if blocked, _ := camping.MovementBlocked(leaderUserID); blocked {
		return 0, false
	}
	return u.Character.RoomId, rooms.LoadRoom(u.Character.RoomId) != nil
}

// RelocateWithdrawal preflights every selected member before moving any. It
// owns no durable separation: dead and already-fled members are not selected.
func (m *CompanyModule) RelocateWithdrawal(uid, origin, destination int, ids []int) error {
	if rooms.LoadRoom(destination) == nil {
		return domain.ErrUnknownMember
	}
	for _, id := range ids {
		leader, _, ok := m.LeaderAndKeyForInstance(id)
		mob := mobs.GetInstance(id)
		if !ok || leader != uid || mob == nil || mob.Character.RoomId != origin || mob.Character.Health <= 0 || !m.runtime.IsAttached(uid, id) {
			return domain.ErrUnknownMember
		}
	}
	moved := []int{}
	for _, id := range ids {
		if !m.runtime.Relocate(id, destination) {
			for _, previous := range moved {
				m.runtime.Relocate(previous, origin)
			}
			return domain.ErrUnknownMember
		}
		moved = append(moved, id)
	}
	return nil
}
