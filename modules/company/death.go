package company

// Phase 25b: companion death and the rescue allowance. A dead companion
// stays on the roster, never spawns, and is charged the leader's online
// time until it is resurrected (resurrect.go) or lost. See
// docs/superpowers/specs/2026-09-24-phase-25b-companion-death-design.md.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

const (
	// defaultAllowanceDays is the rescue allowance in game days.
	defaultAllowanceDays = 3
	// maxChargeStep caps one charge, in seconds: a game loop stalled longer
	// than this isn't spent.
	maxChargeStep = 60
	// warnAtSeconds is when the leader is told time is short.
	warnAtSeconds = 30 * 60
)

func yamlMarshal(v any) ([]byte, error) { return yaml.Marshal(v) }

func (m *CompanyModule) now() time.Time {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

// allowanceSeconds is the rescue allowance granted at a death: the
// configured game days (default 3) of real time, as RoundsPerDay ×
// RoundSeconds. It is snapshotted into each death, so a later calendar
// change doesn't touch it.
func (m *CompanyModule) allowanceSeconds() int {
	if m.allowanceForTest > 0 {
		return m.allowanceForTest
	}
	days := defaultAllowanceDays
	if m.plug != nil {
		if n, ok := configInt(m.plug.Config.Get("ResurrectionAllowanceDays")); ok && n >= 1 {
			days = n
		}
	}
	perDay := gametime.GetDate().RoundsPerDay
	if perDay < 1 {
		perDay = 900
	}
	return days * perDay * int(configs.GetTimingConfig().RoundSeconds)
}

// formatAllowance renders seconds as "2h 41m" (or "45s" under a minute).
func formatAllowance(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", max(seconds, 0))
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, seconds%3600/60)
}

func companionName(c domain.Companion) string {
	return templateName(c.MobTemplateID, "#"+strconv.Itoa(c.ID))
}

// keptState is the companion's state after its death: its level and
// experience, and only the gear the engine's drop rules left on the body.
func keptState(c domain.Companion, evt events.MobDeath) domain.MemberState {
	state := domain.MemberState{Level: 1}
	if c.State != nil {
		state = c.State.Clone()
	}
	if evt.Level > 0 {
		state.Level = evt.Level
	}
	state.Equipment = characters.Worn{}
	for slot, itm := range evt.KeptWorn {
		state.Equipment.Set(slot, itm)
	}
	state.Items = append([]items.Item(nil), evt.KeptItems...)
	state.Gold = evt.KeptGold
	return state.Clone()
}

// recordCompanionDeath marks a tracked companion dead: its kept gear and
// level, out of the formation, with the allowance snapshotted, in one
// save. A failed save leaves the death in memory for the next save.
func (m *CompanyModule) recordCompanionDeath(leaderUserID, companionID int, evt events.MobDeath) {
	if m.persistenceAvailable() != nil {
		return
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	c, ok := findCompanion(record, companionID)
	if !ok || c.Dead() {
		return
	}
	allowance := m.allowanceSeconds()
	death := domain.CompanionDeath{
		OpID:      fmt.Sprintf("cdeath-%d-%d-%d", leaderUserID, companionID, util.GetRoundCount()),
		Allowance: allowance,
		Remaining: allowance,
		Warned:    allowance <= warnAtSeconds,
	}
	_ = m.registry.SetState(leaderUserID, companionID, keptState(c, evt))
	if err := m.registry.MarkDead(leaderUserID, companionID, death); err != nil {
		mudlog.Error("company: mark companion dead", "leader", leaderUserID, "companion", companionID, "error", err)
		return
	}
	if _, anchored := m.anchors[leaderUserID]; !anchored && m.chemistryWorld().LeaderOnline(leaderUserID) {
		m.startAnchor(leaderUserID)
	}
	mudlog.Info("company: companion died", "leader", leaderUserID, "companion", companionID, "op", death.OpID, "allowance", allowance)
	if err := m.save(); err != nil {
		mudlog.Error("company: save companion death", "leader", leaderUserID, "companion", companionID, "error", err)
	}
	m.chemistryWorld().Tell(leaderUserID, fmt.Sprintf(`<ansi fg="red">%s has fallen.</ansi> You have %s of your own time to bring your company to a church or a village shaman and <ansi fg="command">resurrect</ansi> them.`,
		companionName(c), formatAllowance(allowance)))
}

// --- the allowance clock ---

// startAnchor starts counting a leader's online time from now (login,
// copyover, or a death while signed in).
func (m *CompanyModule) startAnchor(leaderUserID int) {
	if m.anchors == nil {
		m.anchors = map[int]time.Time{}
	}
	m.anchors[leaderUserID] = m.now()
}

func hasDead(record domain.Record) bool {
	return slices.ContainsFunc(record.Companions, domain.Companion.Dead)
}

// chargeAllowances runs once a round: every online leader with a dead
// companion is charged the real time since their anchor. Offline leaders'
// anchors are dropped, so their time away is never charged.
func (m *CompanyModule) chargeAllowances() {
	if m.persistenceAvailable() != nil {
		return
	}
	leaders := make([]int, 0, len(m.anchors))
	for leaderUserID, record := range m.registry.Companies {
		if hasDead(record) {
			leaders = append(leaders, leaderUserID)
		}
	}
	for leaderUserID := range m.anchors {
		if record, ok := m.registry.Get(leaderUserID); !ok || !hasDead(record) {
			delete(m.anchors, leaderUserID)
		}
	}
	slices.Sort(leaders)
	for _, leaderUserID := range leaders {
		if !m.chemistryWorld().LeaderOnline(leaderUserID) {
			delete(m.anchors, leaderUserID)
			continue
		}
		m.chargeLeader(leaderUserID)
	}
}

// chargeLeader charges a leader's dead companions the time since the
// anchor, at most maxChargeStep, and moves the anchor on. A leader with no
// anchor gets one and is charged nothing. It reports whether anything was
// charged. Companions who reach zero are lost.
func (m *CompanyModule) chargeLeader(leaderUserID int) bool {
	anchor, ok := m.anchors[leaderUserID]
	if !ok {
		m.startAnchor(leaderUserID)
		return false
	}
	now := m.now()
	elapsed := int(now.Sub(anchor) / time.Second)
	switch {
	case elapsed < 1:
		return false // keep the fraction for the next charge
	case elapsed > maxChargeStep:
		elapsed = maxChargeStep
		m.anchors[leaderUserID] = now
	default:
		m.anchors[leaderUserID] = anchor.Add(time.Duration(elapsed) * time.Second)
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return false
	}
	var expired []domain.Companion
	for _, c := range record.Companions {
		if !c.Dead() {
			continue
		}
		remaining := c.Death.Remaining - elapsed
		warned := c.Death.Warned
		if !warned && remaining > 0 && remaining <= warnAtSeconds {
			warned = true
			m.chemistryWorld().Tell(leaderUserID, fmt.Sprintf(`<ansi fg="yellow">Time is short:</ansi> %s can be raised for only %s more.`, companionName(c), formatAllowance(remaining)))
		}
		_ = m.registry.SetRemaining(leaderUserID, c.ID, remaining, warned)
		if remaining <= 0 {
			expired = append(expired, c)
		}
	}
	for _, c := range expired {
		if err := m.expire(leaderUserID, c.ID); err != nil {
			mudlog.Error("company: companion expiry", "leader", leaderUserID, "companion", c.ID, "error", err)
		}
	}
	return true
}

// expire loses a dead companion whose allowance is spent: it leaves the
// roster through the dismissal path, and is remembered among the lost in
// the same save.
func (m *CompanyModule) expire(leaderUserID, companionID int) error {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	c, ok := findCompanion(record, companionID)
	if !ok || !c.Dead() {
		return nil
	}
	lost := domain.LostCompanion{ID: c.ID, MobTemplateID: c.MobTemplateID, Name: companionName(c), OpID: c.Death.OpID}
	if c.State != nil {
		lost.Level = c.State.Level
	}
	if err := m.dropCompanion(leaderUserID, record, c, &lost); err != nil {
		return err
	}
	mudlog.Info("company: companion lost", "leader", leaderUserID, "companion", companionID, "op", lost.OpID)
	m.chemistryWorld().Tell(leaderUserID, fmt.Sprintf(`<ansi fg="red">%s is lost to you.</ansi> Their name is carved among your fallen.`, lost.Name))
	return nil
}

// remindDead tells a leader, at login, of each dead companion and its time.
func (m *CompanyModule) remindDead(leaderUserID int) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	for _, c := range record.Companions {
		if c.Dead() {
			m.chemistryWorld().Tell(leaderUserID, fmt.Sprintf("%s lies fallen: %s of your time remain to raise them at a church or a village shaman.", companionName(c), formatAllowance(c.Death.Remaining)))
		}
	}
}

// deadStatus is a dead companion's state in company status.
func deadStatus(c domain.Companion) string {
	return fmt.Sprintf("fallen, %s left to raise", formatAllowance(c.Death.Remaining))
}

// lostLine lists the lost for company status, or "" when there are none.
func lostLine(record domain.Record) string {
	if len(record.Lost) == 0 {
		return ""
	}
	names := make([]string, 0, len(record.Lost))
	for _, l := range record.Lost {
		names = append(names, fmt.Sprintf("#%d %s (level %d)", l.ID, l.Name, l.Level))
	}
	return "Lost: " + strings.Join(names, ", ")
}

// deathOnSpawn restarts the leader's allowance clock at login or copyover
// and reminds them of the dead.
func (m *CompanyModule) deathOnSpawn(leaderUserID int) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || !hasDead(record) {
		return
	}
	m.startAnchor(leaderUserID)
	m.remindDead(leaderUserID)
}

// deathOnDespawn charges a leaving leader up to now and drops the anchor.
// It reports whether anything changed that needs saving.
func (m *CompanyModule) deathOnDespawn(leaderUserID int) bool {
	if _, anchored := m.anchors[leaderUserID]; !anchored {
		return false
	}
	charged := m.chargeLeader(leaderUserID)
	delete(m.anchors, leaderUserID)
	return charged
}
