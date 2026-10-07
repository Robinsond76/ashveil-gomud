package company

// Phase 70: errands. From an inn a companion who is not in the formation can
// be sent on an escort, a hunt or a scouting job of a chosen real-time
// length (internal/errands). It leaves the map, saved with the errand's
// return time, and comes back when the time is up, the leader is online and
// free, and a saved seed has been resolved into modest gold, an item, a
// rumour of a lair, or a wound. Decisions and numbers:
// docs/plans/2026-10-07-phase-70-errands.md.
//
// Real time only. The errand is written (company file) before the mob is
// removed, and cleared in the same save that records its wound, so a crash
// leaves the companion away or back, never lost. The leader's gold and item
// are written after, so a crash between loses a reward and never doubles one.

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/errands"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

var _ domain.ErrandViewer = (*CompanyModule)(nil)

// errandRoomTag is the room tag a companion can be sent from (an inn).
const errandRoomTag = "inn"

// errandPlace is where the leader stands, as errands see it.
type errandPlace struct {
	RoomID int
	Zone   string
	// Town is true in a room companions can be sent from.
	Town bool
	// BandLow and BandHigh are the zone's recommended band (0 when none).
	BandLow, BandHigh int
}

// lair is a boss the zone's encounter tables name.
type lair struct {
	MobID int
	Name  string
}

// errandWorld is what errands need from the running game, behind a seam so
// unit tests need no world.
type errandWorld interface {
	// Place is where the leader stands; false when they are not in the world.
	Place(leaderUserID int) (errandPlace, bool)
	// Free is the leader's room when they may take companions back (alive,
	// out of any fight, journey or camp rest).
	Free(leaderUserID int) (roomID int, free bool)
	// Lairs are the bosses the zone's tables name, by mob id.
	Lairs(zone string) []lair
	// ItemValue is an item's merchant base value; false for an unknown item.
	ItemValue(itemID int) (int, bool)
	// Pay gives the leader gold and, when item is set, an item, and saves
	// them. It reports whether the leader could be paid.
	Pay(leaderUserID, gold int, item *items.Item) bool
	// Busy says why a leader cannot send anyone now ("" when they can).
	Busy(user *users.UserRecord) string
}

type nativeErrandWorld struct{ m *CompanyModule }

func (w nativeErrandWorld) Place(leaderUserID int) (errandPlace, bool) {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return errandPlace{}, false
	}
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return errandPlace{}, false
	}
	p := errandPlace{RoomID: room.RoomId, Zone: room.Zone, Town: room.HasTag(errandRoomTag)}
	if cfg := rooms.GetZoneConfig(room.Zone); cfg != nil && cfg.Encounters.Band.Valid() {
		p.BandLow, p.BandHigh = cfg.Encounters.Band.Low, cfg.Encounters.Band.High
	}
	return p, true
}

func (w nativeErrandWorld) Free(leaderUserID int) (int, bool) { return nativeLeaderFree(leaderUserID) }

func (w nativeErrandWorld) Lairs(zone string) []lair {
	cfg := rooms.GetZoneConfig(zone)
	if cfg == nil {
		return nil
	}
	seen := map[int]bool{}
	var out []lair
	for _, table := range cfg.Encounters.Tables {
		for _, comp := range table {
			if !comp.Boss || len(comp.Members) == 0 || seen[comp.Members[0].MobID] {
				continue
			}
			spec := mobs.GetMobSpec(mobs.MobId(comp.Members[0].MobID))
			if spec == nil {
				continue
			}
			seen[comp.Members[0].MobID] = true
			out = append(out, lair{MobID: comp.Members[0].MobID, Name: spec.Character.Name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MobID < out[j].MobID })
	return out
}

func (w nativeErrandWorld) ItemValue(itemID int) (int, bool) {
	if itemID < 1 {
		return 0, false
	}
	spec := items.GetItemSpec(itemID)
	if spec == nil {
		return 0, false
	}
	return spec.Value, true
}

func (w nativeErrandWorld) Pay(leaderUserID, gold int, item *items.Item) bool {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return false
	}
	if gold > 0 {
		user.Character.Gold += gold
		events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: gold})
	}
	if item != nil {
		user.Character.StoreItem(*item)
	}
	if w.m != nil && w.m.saveUser != nil {
		if err := w.m.saveUser(user); err != nil {
			mudlog.Error("company: save after errand", "user", leaderUserID, "error", err)
		}
	}
	return true
}

func (w nativeErrandWorld) Busy(user *users.UserRecord) string {
	if battleBusy(user) {
		return "Not in the middle of a battle. Send someone once the fighting is done."
	}
	if _, free := nativeLeaderFree(user.UserId); !free {
		return "Not while you are fighting, travelling or resting. Send someone once that is over."
	}
	return ""
}

func battleBusy(user *users.UserRecord) bool {
	_, busy := battle.Current(user.UserId)
	return busy
}

func (m *CompanyModule) errandWorld() errandWorld {
	if m.errandSeam != nil {
		return m.errandSeam
	}
	return nativeErrandWorld{m: m}
}

// errandItemPool is the item ids an errand may bring back: gathered goods
// of small value, set in config. An item is only ever paid when its value is
// no more than the gold the same errand would have paid, so a merchant never
// pays more for it than the gold it stands for.
func (m *CompanyModule) errandItemPool() []int {
	var raw any
	if m.plug != nil {
		raw = m.plug.Config.Get("ErrandItemIds")
	}
	ids := make([]int, 0)
	for id := range allowedTemplateIDs(raw) {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// errandSeed is a fresh seed for an errand's rolls, from the module's source.
func (m *CompanyModule) errandSeed() int64 {
	if m.errandRng == nil {
		m.errandRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return m.errandRng.Int63()
}

func hasErrand(record domain.Record) bool {
	for _, c := range record.Companions {
		if c.OnErrand() {
			return true
		}
	}
	return false
}

// errandState is the one-line state of a companion away on its errand.
func (m *CompanyModule) errandState(e errands.Errand) string {
	job := e.Kind.Info().Gerund
	left := e.Remaining(m.now().Unix())
	if left == 0 {
		return fmt.Sprintf("away %s; due back, once you are free", job)
	}
	return fmt.Sprintf("away %s; back in %s", job, errands.Span(left))
}

// errandRefusal says why a companion cannot be sent, or "" when it can.
func (m *CompanyModule) errandRefusal(leaderUserID int, record domain.Record, c domain.Companion) string {
	name := nameOf(c, "That companion")
	switch {
	case c.Dead():
		return fmt.Sprintf("%s has fallen and must be raised first.", name)
	case c.PendingReturn:
		return fmt.Sprintf("%s has fled the fight and has not come back yet.", name)
	case c.OnErrand():
		return fmt.Sprintf("%s is already on an errand (%s).", name, strings.TrimPrefix(m.errandState(*c.Errand), "away "))
	case c.Separated():
		return fmt.Sprintf("%s is separated from the company and must rejoin first.", name)
	case creatures.Is(c.Archetype):
		return fmt.Sprintf("%s is a creature; it has no errands to run.", name)
	}
	instanceID, tracked := m.instance(leaderUserID, c.ID)
	if !tracked || !m.runtime.IsLive(instanceID) || !m.runtime.IsAttached(leaderUserID, instanceID) || !m.runtime.WithLeader(leaderUserID, instanceID) {
		return fmt.Sprintf("%s isn't here with you.", name)
	}
	if _, fighting, ok := m.runtime.Standing(instanceID); !ok || fighting {
		return fmt.Sprintf("%s is busy fighting.", name)
	}
	return ""
}

// startErrand sends a companion away: the errand is saved, then its mob is
// removed. A failed save changes nothing.
func (m *CompanyModule) startErrand(leaderUserID int, c domain.Companion, kind errands.Kind, length errands.Length, place errandPlace) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	before, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", domain.ErrUnknownMember
	}
	instanceID, tracked := m.instance(leaderUserID, c.ID)
	if !tracked || !m.runtime.IsLive(instanceID) {
		return "", domain.ErrUnknownMember
	}
	_, hpMax, _ := m.runtime.Vitals(instanceID)
	m.refreshSnapshot(leaderUserID, c.ID)
	record, _ := m.registry.Get(leaderUserID)
	fresh, found := findCompanion(record, c.ID)
	if !found {
		return "", domain.ErrUnknownMember
	}
	level := companionLevelNumber(fresh)
	errand := errands.New(kind, length, place.Zone, m.now().Unix(), level, place.BandLow, place.BandHigh, hpMax, m.errandSeed())
	record.SendAway(c.ID, errand)
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return "", err
	}
	m.runtime.Detach(leaderUserID, instanceID)
	m.clearInstance(leaderUserID, c.ID)
	if m.strays != nil {
		delete(m.strays[leaderUserID], c.ID)
	}
	name := nameOf(fresh, "Your companion")
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sets out %s for %s. They will be back in %s, whether or not you are here; <ansi fg="command">errands</ansi> shows how they fare.`,
		name, kind.Info().Gerund, errandWhere(errand), length.Info().Label), nil
}

func errandWhere(e errands.Errand) string {
	if e.Zone == "" {
		return "the roads"
	}
	return e.Zone
}

// recallErrand brings a companion home early with nothing: the errand is
// dropped in one save and the companion spawns beside the leader.
func (m *CompanyModule) recallErrand(leaderUserID int, c domain.Companion) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	roomID, free := m.errandWorld().Free(leaderUserID)
	if !free {
		return "Not while you are fighting, travelling or resting. Call them back once that is over.", nil
	}
	before, _ := m.registry.Get(leaderUserID)
	record, _ := m.registry.Get(leaderUserID)
	record.BringBack(c.ID)
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return "", err
	}
	if err := m.restoreForLeader(leaderUserID, roomID); err != nil {
		mudlog.Warn("company: recall errand", "leader", leaderUserID, "companion", c.ID, "error", err)
	}
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is called back from the errand and returns with nothing to show for it.`, nameOf(c, "Your companion")), nil
}

// returnedErrand is one finished errand, ready to tell.
type returnedErrand struct {
	c       domain.Companion
	errand  errands.Errand
	outcome errands.Outcome
	// gold and item are what the leader is paid; what is the outcome in
	// plain words (the chronicle's), and told the same with colour.
	gold int
	item *items.Item
	what string
	told string
}

// tickErrands brings home every errand that is due once its leader is
// online and free. Called each round; it reads real time only.
func (m *CompanyModule) tickErrands() {
	if m.persistenceAvailable() != nil {
		return
	}
	now := m.now().Unix()
	leaders := []int{}
	for leaderUserID, record := range m.registry.Companies {
		for _, c := range record.Companions {
			if c.OnErrand() && c.Errand.Due(now) {
				leaders = append(leaders, leaderUserID)
				break
			}
		}
	}
	slices.Sort(leaders)
	for _, leaderUserID := range leaders {
		if !m.chemistryWorld().LeaderOnline(leaderUserID) {
			continue
		}
		roomID, free := m.errandWorld().Free(leaderUserID)
		if !free {
			continue
		}
		if err := m.finishErrands(leaderUserID, roomID, now); err != nil {
			mudlog.Warn("company: finish errands", "leader", leaderUserID, "error", err)
		}
	}
}

// finishErrands resolves every due errand of one leader: one company save
// clears them and records any wound, then the companions spawn beside the
// leader, then the leader is paid and told and the deeds recorded.
func (m *CompanyModule) finishErrands(leaderUserID, roomID int, now int64) error {
	before, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	record, _ := m.registry.Get(leaderUserID)
	world := m.errandWorld()
	pool := m.errandItemPool()
	var done []returnedErrand
	for i, c := range record.Companions {
		if !c.OnErrand() || !c.Errand.Due(now) {
			continue
		}
		r := m.resolveErrand(leaderUserID, world, pool, c)
		if r.outcome.Kind == errands.Wound {
			if state := withWound(c, r.errand, r.outcome); state != nil {
				record.Companions[i].State = state
			}
		}
		record.BringBack(c.ID)
		done = append(done, r)
	}
	if len(done) == 0 {
		return nil
	}
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	err := m.restoreForLeader(leaderUserID, roomID)
	for _, r := range done {
		paid := true
		if r.gold > 0 || r.item != nil {
			paid = world.Pay(leaderUserID, r.gold, r.item)
		}
		name := nameOf(r.c, "Your companion")
		line := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is back from %s in %s %s.`, name, r.errand.Kind.Info().Label, errandWhere(r.errand), r.told)
		if !paid {
			line = fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is back from %s.`, name, r.errand.Kind.Info().Label)
		}
		m.chemistryWorld().Tell(leaderUserID, line)
		chronicle.Record(leaderUserID, chronicle.Entry{
			Kind: chronicle.Errand, Members: []string{companionName(r.c)}, Keys: []string{string(domain.CompanionMemberKey(r.c.ID))},
			Subject: r.errand.Kind.Info().Label, Detail: r.what, Place: errandWhere(r.errand), Ref: "errand:" + string(r.errand.Kind),
		})
	}
	return err
}

// resolveErrand turns one due errand into what the leader gets. An item is
// chosen only from the pool and only when its value is within the gold the
// errand paid; with none that fits, the errand pays that gold.
func (m *CompanyModule) resolveErrand(leaderUserID int, world errandWorld, pool []int, c domain.Companion) returnedErrand {
	e := *c.Errand
	lairs := world.Lairs(e.Zone)
	outcome := errands.Resolve(e, len(lairs) > 0)
	r := returnedErrand{c: c, errand: e, outcome: outcome}
	if outcome.Kind == errands.Item {
		var fits []int
		values := map[int]int{}
		for _, id := range pool {
			if value, ok := world.ItemValue(id); ok && value <= outcome.Gold {
				fits = append(fits, id)
				values[id] = value
			}
		}
		if len(fits) == 0 {
			r.outcome.Kind = errands.Gold
		} else {
			itm := items.New(fits[outcome.Roll%len(fits)])
			if itm.ItemId < 1 {
				r.outcome.Kind = errands.Gold
			} else {
				// Phase 70 review: the find is part of the pay at its full
				// worth and the rest comes in coin, so a hunt's finds don't
				// pay less than an escort's gold.
				r.item = &itm
				r.gold = outcome.Gold - values[itm.ItemId]
			}
		}
	}
	switch r.outcome.Kind {
	case errands.Gold:
		r.gold = outcome.Gold
		r.what = fmt.Sprintf("with %d gold", r.gold)
		r.told = fmt.Sprintf(`with <ansi fg="gold">%d gold</ansi>`, r.gold)
	case errands.Item:
		r.what = "with " + r.item.DisplayName()
		r.told = fmt.Sprintf(`with <ansi fg="itemname">%s</ansi>`, r.item.DisplayName())
		if r.gold > 0 {
			r.what += fmt.Sprintf(" and %d gold", r.gold)
			r.told += fmt.Sprintf(` and <ansi fg="gold">%d gold</ansi>`, r.gold)
		}
	case errands.Rumour:
		name, said := rumourOf(lairs, outcome.Roll, lairSlain(leaderUserID))
		r.what = "with word of a lair: " + name + " " + said
		r.told = fmt.Sprintf(`with word of a lair: <ansi fg="mobname">%s</ansi> %s`, name, said)
	case errands.Wound:
		r.what = "with a wound: " + wounds.Describe(errandWound(e, outcome))
		r.told = r.what
	}
	return r
}

// lairSlain says whether the company has already slain a lair's master
// (the chronicle's boss deeds, by mob).
func lairSlain(leaderUserID int) func(lair) bool {
	return func(l lair) bool {
		return chronicle.Has(leaderUserID, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Boss}, Ref: fmt.Sprintf("mob:%d", l.MobID)})
	}
}

// rumourOf names a lair worth hearing of, one the company has not slain
// when there is one, and what is said of it.
func rumourOf(lairs []lair, roll int, slain func(lair) bool) (name, said string) {
	var open []lair
	for _, l := range lairs {
		if !slain(l) {
			open = append(open, l)
		}
	}
	if len(open) == 0 {
		open = lairs
	}
	if len(open) == 0 {
		return "no one", "worth the telling"
	}
	l := open[roll%len(open)]
	return l.Name, "is said to hold court nearby"
}

// errandWound is the wound an errand leaves, sized from the maximum health
// saved when the companion left, and placed by the errand's own seed.
func errandWound(e errands.Errand, o errands.Outcome) wounds.Wound {
	maxHealth := e.MaxHealth
	if maxHealth <= 0 {
		maxHealth = 20 + 5*e.Level
	}
	rng := rand.New(rand.NewSource(e.Seed ^ int64(o.Roll)))
	return wounds.Beaten(maxHealth, o.WoundPct, rng.Intn)
}

// withWound is the companion's saved state with the errand's wound added.
func withWound(c domain.Companion, e errands.Errand, o errands.Outcome) *domain.MemberState {
	if c.State == nil {
		// Phase 70 review: an empty state would spawn it bare; the send
		// always takes a snapshot, so this only guards a broken record.
		return nil
	}
	state := c.State.Clone()
	state.Wounds = append(state.Wounds, errandWound(e, o))
	return &state
}

// --- views and commands ---

// ErrandPanel implements company.ErrandViewer: the errands tab and command.
func (m *CompanyModule) ErrandPanel(leaderUserID int) (domain.ErrandPanel, bool) {
	if m.persistenceAvailable() != nil {
		return domain.ErrandPanel{}, false
	}
	record, _ := m.registry.Get(leaderUserID)
	panel := domain.ErrandPanel{Now: m.now().Unix(), Rows: []domain.ErrandRow{}, Options: domain.ErrandOptions(), Recent: []string{}}
	place, inWorld := m.errandWorld().Place(leaderUserID)
	switch {
	case !inWorld:
		panel.Where = "You are not in the world."
	case !place.Town:
		panel.Where = "Companions can be sent from an inn. Find one in a town."
	default:
		panel.Here = true
		panel.Where = "You can send companions from here."
		// Phase 70 review: not while a fight, journey or rest holds the leader.
		if user := users.GetByUserId(leaderUserID); user != nil {
			if busy := m.errandWorld().Busy(user); busy != "" {
				panel.Here, panel.Where = false, busy
			}
		}
	}
	if inWorld {
		panel.Zone = place.Zone
		if place.BandLow > 0 {
			panel.Band = fmt.Sprintf("%d-%d", place.BandLow, place.BandHigh)
		}
	}
	now := m.now().Unix()
	for _, c := range record.Companions {
		row := domain.ErrandRow{ID: c.ID, Name: nameOf(c, "A companion"), Level: companionLevelNumber(c)}
		switch {
		case c.OnErrand():
			e := c.Errand
			row.State = "away"
			row.Kind, row.KindLabel, row.Length, row.Zone = string(e.Kind), e.Kind.Info().Label, string(e.Length), e.Zone
			row.ReturnsAt, row.Remaining, row.Due = e.ReturnsAt, e.Remaining(now), e.Due(now)
			if row.Due {
				row.Waiting = "waiting for you to be out of any fight, journey or rest"
			}
		default:
			if why := m.errandRefusal(leaderUserID, record, c); why != "" {
				row.State, row.Why = "busy", why
			} else {
				row.State = "ready"
			}
		}
		panel.Rows = append(panel.Rows, row)
	}
	for _, e := range chronicle.Query(leaderUserID, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Errand}, Limit: 5}) {
		panel.Recent = append(panel.Recent, chronicle.Prose(e))
	}
	return panel, true
}

// errandsView is the `errands` command's text.
func (m *CompanyModule) errandsView(leaderUserID int) string {
	panel, ok := m.ErrandPanel(leaderUserID)
	if !ok {
		return "Your company's errands can't be read right now."
	}
	if len(panel.Rows) == 0 {
		return "You have no companions to send on errands. Recruit some first (help errands)."
	}
	lines := []string{"Company errands:"}
	for _, r := range panel.Rows {
		switch r.State {
		case "away":
			state := fmt.Sprintf("away on %s in %s; back in %s", errandKindLabel(r.Kind), orRoads(r.Zone), errands.Span(r.Remaining))
			if r.Due {
				state = fmt.Sprintf("away on %s in %s; due back, %s", errandKindLabel(r.Kind), orRoads(r.Zone), r.Waiting)
			}
			lines = append(lines, fmt.Sprintf("  #%d %s, level %d: %s", r.ID, r.Name, r.Level, state))
		case "busy":
			lines = append(lines, fmt.Sprintf("  #%d %s, level %d: not free (%s)", r.ID, r.Name, r.Level, r.Why))
		default:
			lines = append(lines, fmt.Sprintf("  #%d %s, level %d: free to send", r.ID, r.Name, r.Level))
		}
	}
	lines = append(lines, "", panel.Where)
	if panel.Here {
		lines = append(lines, errandBandLine(panel))
		lines = append(lines, "Send one with errand send [member] [escort|hunt|scout] [short|medium|long]; short is half an hour, medium two hours, long eight.")
	}
	lines = append(lines, "Call one back early, with nothing, with errand recall [member]. See help errands.")
	if len(panel.Recent) > 0 {
		lines = append(lines, "", "Lately:")
		for _, s := range panel.Recent {
			lines = append(lines, "  "+s)
		}
	}
	return strings.Join(lines, "\n")
}

func errandBandLine(p domain.ErrandPanel) string {
	if p.Band == "" {
		return fmt.Sprintf("%s names no level band, so an errand here pays by the companion's own level.", orRoads(p.Zone))
	}
	return fmt.Sprintf("Errands from here run in %s, a zone for levels %s: the pay follows the band, and a companion under it risks a wound.", orRoads(p.Zone), p.Band)
}

func orRoads(zone string) string {
	if zone == "" {
		return "the roads"
	}
	return zone
}

func errandKindLabel(kind string) string {
	return errands.Kind(kind).Info().Label
}

// errandCommand is `errand` and `errands`.
func (m *CompanyModule) errandCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	if len(args) == 0 || args[0] == "list" || args[0] == "status" {
		user.SendText(m.errandsView(user.UserId))
		return true, nil
	}
	switch args[0] {
	case "send", "dispatch":
		text, err := m.errandSend(user, args[1:])
		if err != nil {
			return true, err
		}
		user.SendText(text)
	case "recall", "return":
		text, err := m.errandRecall(user, args[1:])
		if err != nil {
			return true, err
		}
		user.SendText(text)
	default:
		user.SendText("Usage: errands | errand send [member] [escort|hunt|scout] [short|medium|long] | errand recall [member]. See help errands.")
	}
	return true, nil
}

const errandUsage = "Usage: errand send [member] [escort|hunt|scout] [short|medium|long]. See help errands."

// errandSend parses and runs `errand send <member> <kind> [length]`.
func (m *CompanyModule) errandSend(user *users.UserRecord, args []string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error(), nil
	}
	if len(args) < 2 {
		return errandUsage, nil
	}
	length := errands.Short
	if l, ok := errands.LengthByWord(args[len(args)-1]); ok && len(args) >= 3 {
		length, args = l, args[:len(args)-1]
	}
	kind, ok := errands.KindByWord(args[len(args)-1])
	if !ok {
		return "Choose an errand: escort, hunt or scout. " + errandUsage, nil
	}
	selector := strings.Join(args[:len(args)-1], " ")
	record, ok := m.registry.Get(user.UserId)
	if !ok || len(record.Companions) == 0 {
		return "You have no companions to send on errands.", nil
	}
	c, found := resolveCompanion(record, selector)
	if !found {
		if _, ambiguous := ambiguousCompanion(record, selector); ambiguous {
			return "More than one companion answers to that; use their number (#N).", nil
		}
		return "You have no companion like that.", nil
	}
	world := m.errandWorld()
	place, inWorld := world.Place(user.UserId)
	if !inWorld || !place.Town {
		return "Companions can be sent on errands from an inn. Find one in a town.", nil
	}
	if busy := world.Busy(user); busy != "" {
		return busy, nil
	}
	if why := m.errandRefusal(user.UserId, record, c); why != "" {
		return why, nil
	}
	return m.startErrand(user.UserId, c, kind, length, place)
}

// errandRecall parses and runs `errand recall <member>`.
func (m *CompanyModule) errandRecall(user *users.UserRecord, args []string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error(), nil
	}
	selector := strings.Join(args, " ")
	record, ok := m.registry.Get(user.UserId)
	if !ok {
		return "You have no companions.", nil
	}
	c, found := resolveCompanion(record, selector)
	if !found {
		if _, ambiguous := ambiguousCompanion(record, selector); ambiguous {
			return "More than one companion answers to that; use their number (#N).", nil
		}
		return "You have no companion like that.", nil
	}
	if !c.OnErrand() {
		return fmt.Sprintf("%s is not away on an errand.", nameOf(c, "That companion")), nil
	}
	return m.recallErrand(user.UserId, c)
}
