package camping

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

const innUsage = "Usage: inn | inn status | inn rest"

// innSettings is the Phase 16 inn configuration.
type innSettings struct {
	RoomTag          string
	PricePerMember   int
	RestDuration     time.Duration
	FatigueRecovery  int
	WellRestedBuffId int
	// Phase 23a rest tiers: the camp tier's buff, and each tier's real-time
	// duration (converted to rounds when granted).
	RestedBuffId       int
	RestedDuration     time.Duration
	WellRestedDuration time.Duration
	// Phase 23b whetstones: the stone's item ID, and the edge one use
	// gives each dull blade (bonus damage for that many strikes).
	WhetstoneItemId  int
	SharpenedBonus   int
	SharpenedStrikes int
}

func defaultInnSettings() innSettings {
	return innSettings{
		RoomTag:          "inn",
		PricePerMember:   5,
		RestDuration:     60 * time.Second,
		FatigueRecovery:  60,
		WellRestedBuffId: 1030,

		RestedBuffId:       1033,
		RestedDuration:     15 * time.Minute,
		WellRestedDuration: 30 * time.Minute,

		WhetstoneItemId:  30,
		SharpenedBonus:   1,
		SharpenedStrikes: 20,
	}
}

// parseInnSettings reads config through get, keeping the default for any
// value that is missing or out of range.
func parseInnSettings(get func(string) any) innSettings {
	s := defaultInnSettings()
	if tag := strings.TrimSpace(configString(get("InnRoomTag"))); tag != "" {
		s.RoomTag = tag
	}
	if n, ok := configInt(get("PricePerMember")); ok && n >= 0 {
		s.PricePerMember = n
	}
	if d, ok := configSeconds(get("InnRestDuration")); ok && d > 0 {
		s.RestDuration = d
	}
	if n, ok := configInt(get("InnFatigueRecovery")); ok && n > 0 {
		s.FatigueRecovery = n
	}
	if n, ok := configInt(get("WellRestedBuffId")); ok && n > 0 {
		s.WellRestedBuffId = n
	}
	if n, ok := configInt(get("RestedBuffId")); ok && n > 0 {
		s.RestedBuffId = n
	}
	if d, ok := configSeconds(get("RestedDuration")); ok && d > 0 {
		s.RestedDuration = d
	}
	if d, ok := configSeconds(get("WellRestedDuration")); ok && d > 0 {
		s.WellRestedDuration = d
	}
	if n, ok := configInt(get("WhetstoneItemId")); ok && n > 0 {
		s.WhetstoneItemId = n
	}
	if n, ok := configInt(get("SharpenedBonus")); ok && n > 0 {
		s.SharpenedBonus = n
	}
	if n, ok := configInt(get("SharpenedStrikes")); ok && n > 0 {
		s.SharpenedStrikes = n
	}
	return s
}

func configInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		return n, err == nil
	}
	return 0, false
}

// configSeconds reads a duration as a Go duration string ("60s") or a
// number of seconds.
func configSeconds(raw any) (time.Duration, bool) {
	if text, ok := raw.(string); ok {
		if d, err := time.ParseDuration(strings.TrimSpace(text)); err == nil {
			return d, true
		}
	}
	if n, ok := configInt(raw); ok {
		return time.Duration(n) * time.Second, true
	}
	return 0, false
}

func (m *CampingModule) innSettings() innSettings {
	if !m.innCfgLoaded {
		return defaultInnSettings()
	}
	return m.innCfg
}

func (m *CampingModule) resetInnState() {
	m.stays = map[int]camping.InnStay{}
	m.innRecoveryApplied = map[int]bool{}
	m.wellRestedPending = map[int]bool{}
	m.restedPending = map[int]bool{}
	m.owed = map[int]map[int]camping.OwedGrant{}
	m.autoSharpen = map[int]bool{}
	m.innTimers = map[int]Timer{}
	m.innTimerGeneration = map[int]uint64{}
}

// --- native seams ---

func (m *CampingModule) userByID(userID int) *users.UserRecord {
	if m.lookupUser != nil {
		return m.lookupUser(userID)
	}
	return users.GetByUserId(userID)
}

func (m *CampingModule) companyMembers(leaderUserID int) int {
	if m.companySize != nil {
		return m.companySize(leaderUserID)
	}
	if roster := survival.CurrentRoster(leaderUserID); len(roster) > 0 {
		return len(roster)
	}
	return 1
}

func (m *CampingModule) isTravelling(leaderUserID int) bool {
	if m.travelling != nil {
		return m.travelling(leaderUserID)
	}
	blocked, _ := expedition.MovementBlocked(leaderUserID)
	return blocked
}

func (m *CampingModule) currentWeather(zone string) (weather.Condition, bool) {
	if m.weatherIn != nil {
		return m.weatherIn(zone)
	}
	return weather.CurrentCondition(zone)
}

// campRecovery is the fatigue a camp rest in room restores: FatigueRecovery
// scaled by the zone's weather RestRecoveryPct (rounded half up, at least
// 1). scaled reports a weather penalty worth mentioning.
func (m *CampingModule) campRecovery(room *rooms.Room) (int, weather.Condition, bool) {
	recovery := camping.FatigueRecovery
	if room == nil {
		return recovery, weather.Condition{}, false
	}
	condition, ok := m.currentWeather(room.Zone)
	if !ok || condition.RestRecoveryPct <= 0 {
		return recovery, weather.Condition{}, false
	}
	scaled := (recovery*condition.RestRecoveryPct + 50) / 100
	if scaled < 1 {
		scaled = 1
	}
	return scaled, condition, condition.RestRecoveryPct < 100
}

// --- inn stays ---

func innOperationID(stay camping.InnStay) string {
	return fmt.Sprintf("inn-rest-%d-%d-%d", stay.LeaderUserID, stay.RoomID, stay.Rest.StartedAtUTC.UnixNano())
}

func (m *CampingModule) innPrice(leaderUserID int, pricing standing.Standing) (price, members int) {
	members = m.companyMembers(leaderUserID)
	if members < 1 {
		members = 1
	}
	return pricing.InnPrice(m.innSettings().PricePerMember * members), members
}

// innStanding is the company's Phase 21b standing in the room's settlement
// (the zero Standing, normal prices, when there is none). It calls into
// other modules, so it is read before taking m.mu.
func innStanding(leaderUserID int, room *rooms.Room) standing.Standing {
	if room == nil {
		return standing.Standing{}
	}
	s, ok := standing.For(leaderUserID, room.Zone)
	if !ok {
		return standing.Standing{}
	}
	return s
}

func innRefusal(room *rooms.Room) string {
	return fmt.Sprintf("%s won't give your company a room.", roomTitle(room.RoomId))
}

func (m *CampingModule) innCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	switch {
	case len(args) == 0 || args[0] == "status":
		user.SendText(m.innStatus(user, room))
	case args[0] == "rest":
		user.SendText(m.innRest(user, room))
	default:
		user.SendText(innUsage)
	}
	return true, nil
}

// innStatus shows any stay in progress, or this inn's price.
func (m *CampingModule) innStatus(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	pricing := innStanding(user.UserId, room)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.stays[user.UserId]; ok {
		if err := m.syncStayLocked(user.UserId); err != nil {
			mudlog.Warn("camping: inn status sync", "leader", user.UserId, "error", err)
		}
		if _, still := m.stays[user.UserId]; still {
			return m.innStatusTextLocked(user.UserId)
		}
	}
	if room == nil || !room.HasTag(m.innSettings().RoomTag) {
		return "There is no inn here."
	}
	if pricing.InnRefused() {
		return innRefusal(room)
	}
	price, members := m.innPrice(user.UserId, pricing)
	perMember := fmt.Sprintf("%d per member", m.innSettings().PricePerMember)
	if pricing.InnMarkupPct > 0 {
		perMember += fmt.Sprintf(", +%d%%", pricing.InnMarkupPct)
	}
	text := fmt.Sprintf("%s offers your company of %d a room for the night: %d gold (%s). You have %d gold.\nUse \"inn rest\" to pay and rest (%s).",
		roomTitle(room.RoomId), members, price, perMember, user.Character.Gold, m.innSettings().RestDuration)
	if pricing.InnMarkupPct > 0 {
		text += fmt.Sprintf("\nYour company is %s here, so the room costs %d%% more.", pricing.Tier, pricing.InnMarkupPct)
	}
	return text
}

// innRest takes the company's gold and starts a durable real-time stay.
func (m *CampingModule) innRest(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if err := m.survival.Available(); err != nil {
		return err.Error()
	}
	// The travel check runs before taking m.mu (camping never holds its lock
	// while calling another module). Both it and a route start run on the
	// game loop, so nothing can slip in between.
	if m.isTravelling(user.UserId) {
		return "You can't take a room while travelling."
	}
	pricing := innStanding(user.UserId, room)
	m.mu.Lock()
	defer m.mu.Unlock()
	// Config is written by load() under m.mu, so read it under m.mu too.
	settings := m.innSettings()
	if room == nil || !room.HasTag(settings.RoomTag) {
		return "There is no inn here."
	}
	if camp, ok := m.camps[user.UserId]; ok && camp.Rest != nil && camp.Rest.State == camping.Resting {
		return "You are already resting at camp."
	}
	if _, ok := m.stays[user.UserId]; ok {
		if err := m.syncStayLocked(user.UserId); err != nil {
			mudlog.Warn("camping: inn rest sync", "leader", user.UserId, "error", err)
		}
		if stay, still := m.stays[user.UserId]; still {
			if stay.Resting() {
				return "You are already resting at the inn."
			}
			return "Your company has only just woken. Give it a moment."
		}
	}
	if pricing.InnRefused() {
		return innRefusal(room)
	}
	price, _ := m.innPrice(user.UserId, pricing)
	if user.Character.Gold < price {
		return fmt.Sprintf("A room for your company costs %d gold, and you have only %d.", price, user.Character.Gold)
	}
	stay, err := camping.StartInnStay(user.UserId, room.RoomId, price, m.clock().UTC(), settings.RestDuration, settings.FatigueRecovery)
	if err != nil {
		return "You can't rest here."
	}
	user.Character.Gold -= price
	m.stays[user.UserId] = stay
	delete(m.innRecoveryApplied, user.UserId)
	if err := m.saveLocked(); err != nil {
		// Roll back: nothing changes, and the gold comes back.
		delete(m.stays, user.UserId)
		user.Character.Gold += price
		return err.Error()
	}
	m.scheduleStayLocked(stay)
	return fmt.Sprintf("You pay %d gold and your company settles in to rest. (%s)", price, settings.RestDuration)
}

// syncStayLocked completes a due stay and applies its recovery, or retries
// the recovery of a completed stay not yet marked applied. Like syncLocked
// it is idempotent across crashes: the survival operation is ledgered by a
// deterministic ID. It never touches characters or buffs, since it can run
// on a timer goroutine: Well Rested is only marked pending here.
func (m *CampingModule) syncStayLocked(leaderUserID int) error {
	stay, ok := m.stays[leaderUserID]
	if !ok {
		return nil
	}
	if stay.Resting() {
		now := m.clock().UTC()
		if !stay.Due(now) {
			return nil
		}
		completed, err := stay.Complete(now)
		if err != nil {
			return err
		}
		m.stays[leaderUserID] = completed
		if err := m.saveLocked(); err != nil {
			m.stays[leaderUserID] = stay
			return err
		}
		m.stopInnTimerLocked(leaderUserID)
		return m.applyInnRecoveryLocked(leaderUserID, completed, true)
	}
	if m.innRecoveryApplied[leaderUserID] {
		return nil
	}
	return m.applyInnRecoveryLocked(leaderUserID, stay, false)
}

func (m *CampingModule) applyInnRecoveryLocked(leaderUserID int, stay camping.InnStay, announce bool) error {
	if _, err := m.survival.ApplyCompanyRestRecovery(leaderUserID, innOperationID(stay), stay.Rest.RecoveryAmount()); err != nil {
		return err
	}
	m.innRecoveryApplied[leaderUserID] = true
	m.wellRestedPending[leaderUserID] = true
	if err := m.saveLocked(); err != nil {
		// Keep the in-memory markers so a same-process retry cannot call
		// survival twice; the next save persists them.
		return err
	}
	if announce {
		m.sendToLeader(leaderUserID, "Your company wakes rested and refreshed.")
	}
	return nil
}

func (m *CampingModule) scheduleStayLocked(stay camping.InnStay) {
	leaderUserID := stay.LeaderUserID
	if !stay.Resting() {
		m.stopInnTimerLocked(leaderUserID)
		return
	}
	remaining := stay.RemainingAt(m.clock().UTC())
	m.innTimerGeneration[leaderUserID]++
	generation := m.innTimerGeneration[leaderUserID]
	m.stopInnTimerLocked(leaderUserID)
	m.innTimers[leaderUserID] = m.scheduler.AfterFunc(remaining, func() {
		m.onInnTimer(leaderUserID, generation)
	})
}

func (m *CampingModule) stopInnTimerLocked(leaderUserID int) {
	if timer, ok := m.innTimers[leaderUserID]; ok {
		timer.Stop()
		delete(m.innTimers, leaderUserID)
	}
}

// onInnTimer runs off the game loop. It only persists state.
func (m *CampingModule) onInnTimer(leaderUserID int, generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.innTimerGeneration[leaderUserID] != generation {
		return
	}
	if _, ok := m.innTimers[leaderUserID]; !ok {
		return
	}
	delete(m.innTimers, leaderUserID)
	stay, ok := m.stays[leaderUserID]
	if !ok || !stay.Resting() {
		return
	}
	if err := m.syncStayLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: inn timer sync", "leader", leaderUserID, "error", err)
	}
}

// recoverStaysLocked reconciles stays on load or copyover: an active stay
// reschedules or completes once if overdue; a completed one retries only
// its idempotent recovery. Invalid stays are kept for operator repair.
func (m *CampingModule) recoverStaysLocked() {
	for leaderUserID, stay := range m.stays {
		if err := stay.Validate(); err != nil {
			mudlog.Warn("camping: recovery invalid inn stay", "leader", leaderUserID, "error", err)
			continue
		}
		if err := m.syncStayLocked(leaderUserID); err != nil {
			mudlog.Warn("camping: recovery inn sync", "leader", leaderUserID, "error", err)
			continue
		}
		if refreshed, ok := m.stays[leaderUserID]; ok && refreshed.Resting() {
			m.scheduleStayLocked(refreshed)
		}
	}
}

func (m *CampingModule) innStatusTextLocked(leaderUserID int) string {
	stay, ok := m.stays[leaderUserID]
	if !ok {
		return "You have no room at an inn."
	}
	lines := []string{fmt.Sprintf("Your company has a room at %s.", roomTitle(stay.RoomID))}
	if stay.Resting() {
		now := m.clock().UTC()
		lines = append(lines, fmt.Sprintf("Resting: %d%% complete (%s remaining).", int(stay.ProgressAt(now)*100), stay.RemainingAt(now).Round(time.Second)))
	} else {
		lines = append(lines, "Your company has rested here.")
	}
	lines = append(lines, "Company:")
	for _, member := range m.survival.CompanyNeeds(leaderUserID) {
		lines = append(lines, fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			member.Name,
			member.Needs.Hunger, survival.HungerLabel(member.Needs.Hunger),
			member.Needs.Thirst, survival.ThirstLabel(member.Needs.Thirst),
			member.Needs.Fatigue, survival.FatigueLabel(member.Needs.Fatigue)))
	}
	return strings.Join(lines, "\n")
}
