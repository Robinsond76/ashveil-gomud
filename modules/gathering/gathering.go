// Package gathering is Phase 40a2: a company spends real time and effort to
// gather herbs, firewood, fish and game from rooms that offer them. Each
// room's resource has a pool that depletes and regrows in real time and is
// saved with the world, so a restart or copyover never refills it early.
//
// The work is a timed action kept in memory only: a restart or copyover
// simply cancels it, with nothing gained or lost. It runs on the game loop
// (the NewRound listener) and uses real time only; it never advances the
// world's clock. The rules and rolls live in internal/gathering.
package gathering

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/gathering"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const gatherUsage = `Usage: gather | gather herbs | gather firewood | fish | hunt. See <ansi fg="command">help gathering</ansi>.`

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*gathering.Ledger) error
	Save(gathering.Ledger) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(l *gathering.Ledger) error {
	data, err := s.plug.ReadBytes("pools")
	if errors.Is(err, os.ErrNotExist) {
		*l = gathering.NewLedger()
		return nil
	}
	if err != nil {
		return err
	}
	var wire gathering.Ledger
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := gathering.NewLedger()
	for room, pools := range wire.Rooms {
		if room <= 0 {
			continue
		}
		for kind, pool := range pools {
			if kind.Valid() {
				if loaded.Rooms[room] == nil {
					loaded.Rooms[room] = map[gathering.Kind]gathering.Pool{}
				}
				loaded.Rooms[room][kind] = pool
			}
		}
	}
	*l = loaded
	return nil
}

func (s pluginStore) Save(l gathering.Ledger) error { return s.plug.WriteStruct("pools", l) }

// action is one company's work in progress.
type action struct {
	Kind    gathering.Kind
	RoomID  int
	Started time.Time
	Due     time.Time
}

// GatheringModule is the module's state.
type GatheringModule struct {
	plug *plugins.Plugin

	mu       sync.Mutex
	store    Store
	loadErr  error
	ledger   gathering.Ledger
	settings gathering.Settings
	actions  map[int]*action

	// Seams; natives by default.
	clock       func() time.Time
	rng         gathering.Rand
	lookupUser  func(userID int) *users.UserRecord
	loadRoom    func(roomID int) *rooms.Room
	inBattle    func(u *users.UserRecord) bool
	resting     func(userID int) bool
	travelling  func(userID int) bool
	follower    func(userID int) bool
	weatherIn   func(zone string) (weather.Condition, bool)
	dark        func(u *users.UserRecord, room *rooms.Room) bool
	specialist  func(leaderUserID int, utility string, roomIDs ...int) (archetypes.Specialist, bool)
	members     func(u *users.UserRecord) []*characters.Character
	scribeRank  func(c *characters.Character) int
	itemCount   func(leaderUserID, itemID int) int
	spendItem   func(leaderUserID, itemID int) bool
	effort      func(userID, roomID, pct int)
	encounter   func(userID, roomID, bonusPct int) bool
	wouldExceed func(leaderUserID, addGrams int) bool
	deposit     func(leaderUserID int, op string, stacks []encumbrance.CargoStack) error
	giveItem    func(u *users.UserRecord, itemID int)
}

// module is the registered instance (tests reach the real wiring through it).
var module *GatheringModule

func init() {
	m := newModule()
	module = m
	m.plug = plugins.New("gathering", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("gathering: save", "error", err)
		}
	})
	m.plug.AddUserCommand("gather", m.gatherCommand, false, false)
	m.plug.AddUserCommand("fish", m.fishCommand, false, false)
	m.plug.AddUserCommand("hunt", m.huntCommand, false, false)
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	events.RegisterListener(events.Input{}, m.onInput)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	events.RegisterListener(events.PlayerDespawn{}, m.onDespawn)
	rooms.SetDepletedCheck(m.isDepleted)
}

func newModule() *GatheringModule {
	return &GatheringModule{
		ledger:   gathering.NewLedger(),
		settings: gathering.DefaultSettings(),
		actions:  map[int]*action{},
		clock:    time.Now,
		rng:      util.Rand,
		lookupUser: func(userID int) *users.UserRecord {
			return users.GetByUserId(userID)
		},
		loadRoom: rooms.LoadRoom,
		inBattle: actionpolicy.InBattle,
		resting: func(userID int) bool {
			blocked, _ := camping.MovementBlocked(userID)
			return blocked
		},
		travelling: func(userID int) bool {
			blocked, _ := expedition.MovementBlocked(userID)
			return blocked
		},
		follower: func(userID int) bool {
			p := parties.Get(userID)
			return p != nil && !p.IsLeader(userID)
		},
		weatherIn: weather.CurrentCondition,
		dark: func(u *users.UserRecord, room *rooms.Room) bool {
			return room.VisibilityForUser(u) < 1
		},
		specialist: archetypes.BestSpecialist,
		members:    presentMembers,
		scribeRank: func(c *characters.Character) int { return c.GetSkillLevel(loot.ScribeSkill) },
		itemCount:  company.CompanyItemCount,
		spendItem:  company.SpendCompanyItem,
		effort:     walking.Effort,
		encounter:  encounters.Attempt,
		wouldExceed: func(leaderUserID, addGrams int) bool {
			_, full := encumbrance.WouldExceed(leaderUserID, addGrams)
			return full
		},
		deposit: encumbrance.DepositCargo,
		giveItem: func(u *users.UserRecord, itemID int) {
			itm := items.New(itemID)
			if u.Character.StoreItem(itm) {
				events.AddToQueue(events.ItemOwnership{UserId: u.UserId, Item: itm, Gained: true})
			}
		},
	}
}

// presentMembers is the leader's character and each living, able companion
// standing in the leader's room.
func presentMembers(u *users.UserRecord) []*characters.Character {
	out := []*characters.Character{u.Character}
	for _, ref := range survival.CurrentRoster(u.UserId) {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok {
			continue
		}
		instanceID, ok := company.InstanceFor(u.UserId, companionID)
		if !ok {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || mob.Character.RoomId != u.Character.RoomId || mob.Character.IsDisabled() {
			continue
		}
		out = append(out, &mob.Character)
	}
	return out
}

func (m *GatheringModule) load() {
	if m.store == nil {
		return
	}
	loaded := gathering.NewLedger()
	if err := m.store.Load(&loaded); err != nil {
		m.mu.Lock()
		m.loadErr = err
		m.mu.Unlock()
		mudlog.Error("gathering: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ledger = loaded
	m.loadErr = nil
	if m.plug != nil {
		m.settings = parseSettings(m.plug.Config.Get)
	}
	m.ledger.Prune(m.settings.Rules, m.clock())
}

func (m *GatheringModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *GatheringModule) saveLocked() error {
	if m.store == nil {
		return nil
	}
	m.ledger.Prune(m.settings.Rules, m.clock())
	return m.store.Save(m.ledger.Clone())
}

// Settings is a copy of the current numbers (for tests and other views).
func (m *GatheringModule) cfg() gathering.Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

// --- pools ---

// isDepleted is rooms' depleted check: the room's resource has no charge
// left right now. Ephemeral (tutorial) rooms are never depleted.
func (m *GatheringModule) isDepleted(roomID int, resource string) bool {
	kind := gathering.Kind(resource)
	if !kind.Valid() || rooms.IsEphemeralRoomId(roomID) {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rule, ok := m.settings.Rules[kind]
	if !ok {
		return false
	}
	return m.ledger.Charges(roomID, kind, rule, m.clock()) < 1
}

// untilNextCharge is how long until a picked-clean pool regrows one charge.
func (m *GatheringModule) untilNextCharge(roomID int, kind gathering.Kind) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	rule := m.settings.Rules[kind]
	p, ok := m.ledger.Rooms[roomID][kind]
	if !ok {
		return 0
	}
	grown, _ := p.Regrown(rule, m.clock())
	wait := rule.Regrow - m.clock().Sub(grown.LastRegrowUTC)
	if wait < 0 {
		return 0
	}
	return wait
}

// --- commands ---

func (m *GatheringModule) gatherCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	if len(args) == 0 {
		user.SendText(m.offers(user, room))
		return true, nil
	}
	switch args[0] {
	case "herbs", "herb", "plants":
		user.SendText(m.start(user, room, gathering.Herbs))
	case "firewood", "wood", "kindling":
		user.SendText(m.start(user, room, gathering.Firewood))
	case "fish", "fishing":
		user.SendText(m.start(user, room, gathering.Fishing))
	case "game", "hunt":
		user.SendText(m.start(user, room, gathering.Game))
	default:
		user.SendText(gatherUsage)
	}
	return true, nil
}

func (m *GatheringModule) fishCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.start(user, room, gathering.Fishing))
	return true, nil
}

func (m *GatheringModule) huntCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.start(user, room, gathering.Game))
	return true, nil
}

// verbs are what each kind is called to the player.
var verbs = map[gathering.Kind]struct{ present, noun, command string }{
	gathering.Herbs:    {"gathering herbs", "herbs", "gather herbs"},
	gathering.Firewood: {"gathering firewood", "firewood", "gather firewood"},
	gathering.Fishing:  {"fishing", "fishing", "fish"},
	gathering.Game:     {"hunting", "game", "hunt"},
}

// shoreBiomes are the biomes where a water room can be fished.
var shoreBiomes = map[string]bool{"shore": true, "water": true}

// offered reports whether the room offers the kind. Fishing also works from
// a water room on the shore or in the water.
func offered(room *rooms.Room, kind gathering.Kind) bool {
	if room == nil {
		return false
	}
	if room.HasResource(string(kind)) {
		return true
	}
	if kind == gathering.Fishing && room.HasResource(rooms.ResourceWater) {
		if biome := room.GetBiome(); biome != nil && shoreBiomes[strings.ToLower(biome.BiomeId)] {
			return true
		}
	}
	return false
}

// offers is a bare `gather`: what the room offers and what the company can do.
func (m *GatheringModule) offers(user *users.UserRecord, room *rooms.Room) string {
	cfg := m.cfg()
	var lines []string
	now := m.clock()
	for _, kind := range gathering.Kinds {
		if !offered(room, kind) {
			continue
		}
		rule := cfg.Rules[kind]
		m.mu.Lock()
		charges := m.ledger.Charges(room.RoomId, kind, rule, now)
		m.mu.Unlock()
		if rooms.IsEphemeralRoomId(room.RoomId) {
			charges = rule.PoolMax
		}
		state := fmt.Sprintf("%d of %d ready", charges, rule.PoolMax)
		if charges < 1 {
			state = "picked clean for now, regrows in " + waitText(m.untilNextCharge(room.RoomId, kind))
		}
		line := fmt.Sprintf(`  <ansi fg="command">%-15s</ansi> %s, %s`, verbs[kind].command, state, durationText(rule.Duration))
		if note := m.needNote(user, kind, cfg); note != "" {
			line += "; " + note
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return `There is nothing here to gather. Rooms with herbs, firewood, fishing or game say so on the "Here:" line when you <ansi fg="command">look</ansi>. See <ansi fg="command">help gathering</ansi>.`
	}
	return "Here your company can gather:\n" + strings.Join(lines, "\n")
}

// needNote is what a kind is missing, or how it will go.
func (m *GatheringModule) needNote(user *users.UserRecord, kind gathering.Kind, cfg gathering.Settings) string {
	switch kind {
	case gathering.Fishing:
		if m.itemCount(user.UserId, cfg.Items.FishingLine) < 1 {
			return "needs a fishing line"
		}
	case gathering.Game:
		if !m.hasTool(user, toolBow) {
			return "no bow or sling, so snares at half the chance"
		}
	}
	return ""
}

func durationText(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%d min", int(d/time.Minute))
	}
	return fmt.Sprintf("%d s", int(d.Round(time.Second)/time.Second))
}

func waitText(d time.Duration) string {
	mins := int((d + time.Minute - 1) / time.Minute)
	if mins < 1 {
		mins = 1
	}
	if mins == 1 {
		return "about a minute"
	}
	return fmt.Sprintf("about %d minutes", mins)
}

// refuse is why the company can't start the work, or "".
func (m *GatheringModule) refuse(user *users.UserRecord, room *rooms.Room, kind gathering.Kind, cfg gathering.Settings) string {
	switch {
	case user == nil || user.Character == nil || room == nil:
		return "You can't do that now."
	case user.Character.Health < 1:
		return "You are in no state to gather anything."
	case m.follower(user.UserId):
		return "Only the leader of the party gives that order."
	case m.inBattle(user):
		return "You can't gather in the middle of a battle."
	case m.resting(user.UserId):
		return "You can't gather while you are resting."
	case m.travelling(user.UserId):
		return "You can't gather while you are travelling."
	}
	if !offered(room, kind) {
		return map[gathering.Kind]string{
			gathering.Herbs:    "There are no herbs worth picking here.",
			gathering.Firewood: "There is no deadfall to gather here.",
			gathering.Fishing:  "There is nowhere to fish here.",
			gathering.Game:     "There is no game to hunt here.",
		}[kind]
	}
	if kind == gathering.Fishing && m.itemCount(user.UserId, cfg.Items.FishingLine) < 1 {
		return `You need a fishing line to fish. Markets and the old fisherman at Frost Lake sell them (<ansi fg="command">help gathering</ansi>).`
	}
	return ""
}

// start begins the timed action and returns what the leader is told.
func (m *GatheringModule) start(user *users.UserRecord, room *rooms.Room, kind gathering.Kind) string {
	cfg := m.cfg()
	if text := m.refuse(user, room, kind, cfg); text != "" {
		return text
	}
	rule := cfg.Rules[kind]
	now := m.clock()

	m.mu.Lock()
	if m.loadErr != nil {
		m.mu.Unlock()
		return "Gathering is unavailable until the world data reloads."
	}
	if cur, busy := m.actions[user.UserId]; busy {
		m.mu.Unlock()
		return fmt.Sprintf("Your company is already %s here.", verbs[cur.Kind].present)
	}
	charges := m.ledger.Charges(room.RoomId, kind, rule, now)
	if rooms.IsEphemeralRoomId(room.RoomId) {
		charges = rule.PoolMax
	}
	m.mu.Unlock()
	if charges < 1 {
		return fmt.Sprintf("The %s here is picked clean for now. It regrows in %s.", verbs[kind].noun, waitText(m.untilNextCharge(room.RoomId, kind)))
	}

	m.mu.Lock()
	m.actions[user.UserId] = &action{Kind: kind, RoomID: room.RoomId, Started: now, Due: now.Add(rule.Duration)}
	m.mu.Unlock()

	text := fmt.Sprintf("Your company sets about %s. (%s)", verbs[kind].present, durationText(rule.Duration))
	if kind == gathering.Game && !m.hasTool(user, toolBow) {
		text = "No one has a bow, crossbow or sling, so your company sets snares instead. (" + durationText(rule.Duration) + ")"
	}
	return text + "\nAny command other than look or conditions stops the work."
}

// --- cancelling ---

// keepsWorking are the typed commands that never interrupt the work.
var keepsWorking = map[string]bool{
	"look": true, "l": true, "conditions": true, "gather": true, "fish": true, "hunt": true,
}

// onInput cancels the work when its leader types anything else.
func (m *GatheringModule) onInput(e events.Event) events.ListenerReturn {
	in, ok := e.(events.Input)
	if !ok || in.UserId <= 0 || in.MobInstanceId > 0 {
		return events.Continue
	}
	fields := strings.Fields(strings.ToLower(in.InputText))
	if len(fields) == 0 || keepsWorking[fields[0]] {
		return events.Continue
	}
	m.cancel(in.UserId, "You stop what you were doing; the work is abandoned.")
	return events.Continue
}

// cancel ends a user's work with nothing gained, telling them why.
func (m *GatheringModule) cancel(userID int, why string) bool {
	m.mu.Lock()
	_, busy := m.actions[userID]
	delete(m.actions, userID)
	m.mu.Unlock()
	if busy && why != "" {
		if user := m.lookupUser(userID); user != nil {
			user.SendText(why)
		}
	}
	return busy
}

func (m *GatheringModule) onUserPurged(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.UserPurged); ok {
		m.cancel(evt.UserId, "")
	}
	return events.Continue
}

func (m *GatheringModule) onDespawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerDespawn); ok {
		m.cancel(evt.UserId, "")
	}
	return events.Continue
}

// Active reports whether the user's company is working (tests, views).
func (m *GatheringModule) Active(userID int) (gathering.Kind, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.actions[userID]
	if !ok {
		return "", false
	}
	return a.Kind, true
}

// --- the tick ---

func (m *GatheringModule) onNewRound(e events.Event) events.ListenerReturn {
	m.tick()
	return events.Continue
}

// tick ends work that can no longer go on and finishes work that is due.
func (m *GatheringModule) tick() {
	m.mu.Lock()
	ids := make([]int, 0, len(m.actions))
	for id := range m.actions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	sort.Ints(ids)
	now := m.clock()
	for _, id := range ids {
		m.mu.Lock()
		a, ok := m.actions[id]
		var snapshot action
		if ok {
			snapshot = *a
		}
		m.mu.Unlock()
		if !ok {
			continue
		}
		user := m.lookupUser(id)
		if user == nil || user.Character == nil {
			m.cancel(id, "")
			continue
		}
		switch {
		case user.Character.Health < 1:
			m.cancel(id, "You are in no state to go on gathering.")
			continue
		case user.Character.RoomId != snapshot.RoomID:
			m.cancel(id, "You have moved on; the work is abandoned.")
			continue
		case m.inBattle(user):
			m.cancel(id, "The fight cuts the work short; you gain nothing.")
			continue
		case m.resting(id), m.travelling(id):
			m.cancel(id, "You can't go on gathering; the work is abandoned.")
			continue
		}
		if now.Before(snapshot.Due) {
			continue
		}
		m.mu.Lock()
		delete(m.actions, id)
		m.mu.Unlock()
		if room := m.loadRoom(snapshot.RoomID); room != nil {
			m.finish(user, room, snapshot)
		}
	}
}

// --- finishing ---

type tool int

const (
	toolBow   tool = iota // a bow, crossbow or sling
	toolAxe               // a hatchet or axe
	toolKnife             // a bladed weapon
)

// weaponSubtype reads a worn weapon's subtype.
func weaponSubtype(itm items.Item) string {
	if itm.ItemId < 1 {
		return ""
	}
	spec := itm.GetSpec()
	if spec.Type != items.Weapon {
		return ""
	}
	return strings.ToLower(string(spec.Subtype))
}

// hasTool reports whether a present member wields a weapon that works as the tool.
func (m *GatheringModule) hasTool(user *users.UserRecord, t tool) bool {
	for _, c := range m.members(user) {
		for _, itm := range []items.Item{c.Equipment.Weapon, c.Equipment.Offhand} {
			sub := weaponSubtype(itm)
			switch t {
			case toolBow:
				if sub == "shooting" {
					return true
				}
			case toolAxe:
				if sub == "cleaving" {
					return true
				}
			case toolKnife:
				if camping.Bladed(sub) {
					return true
				}
			}
		}
	}
	return false
}

func (m *GatheringModule) hasScribe(user *users.UserRecord) bool {
	for _, c := range m.members(user) {
		if m.scribeRank(c) > 0 {
			return true
		}
	}
	return false
}

func (m *GatheringModule) forager(user *users.UserRecord, roomID int) (archetypes.Specialist, bool) {
	if m.specialist == nil {
		return archetypes.Specialist{}, false
	}
	return m.specialist(user.UserId, archetypes.UtilityForage, roomID)
}

func itemName(id int) string {
	if spec := items.GetItemSpec(id); spec != nil {
		return spec.Name
	}
	return fmt.Sprintf("item %d", id)
}

func named(id, count int) string {
	name := itemName(id)
	if count > 1 {
		name = fmt.Sprintf("%d %s", count, mobparty.Plural(name))
	}
	return fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, name)
}

// finish resolves one completed attempt: spend a charge, roll the yield,
// charge the effort, hand over the haul, then roll the danger.
func (m *GatheringModule) finish(user *users.UserRecord, room *rooms.Room, a action) {
	cfg := m.cfg()
	rule := cfg.Rules[a.Kind]
	now := m.clock()

	// Tutorial copies are never picked clean, and nothing is recorded for
	// them (their ids are short-lived).
	spent, saveErr := true, error(nil)
	if !rooms.IsEphemeralRoomId(room.RoomId) {
		m.mu.Lock()
		if spent = m.ledger.Spend(room.RoomId, a.Kind, rule, now); spent {
			saveErr = m.saveLocked()
		}
		m.mu.Unlock()
	}
	if saveErr != nil {
		mudlog.Error("gathering: save pools", "error", saveErr)
	}
	if !spent {
		user.SendText(fmt.Sprintf("While your company worked, another company picked the %s here clean. You find nothing.", verbs[a.Kind].noun))
		return
	}

	sp, hasForager := m.forager(user, room.RoomId)
	var drops []gathering.Drop
	var lines []string
	switch a.Kind {
	case gathering.Herbs:
		ctx := gathering.HerbContext{
			Zone: room.Zone, ForageLevel: sp.Level, HasForaging: hasForager,
			Knife: m.hasTool(user, toolKnife), Dark: m.dark(user, room), Scribe: m.hasScribe(user),
		}
		var weed bool
		drops, weed = cfg.RollHerbs(ctx, m.rng)
		switch {
		case weed:
			lines = append(lines, "Nobody in your company knows good leaves from bad, and the pick turns out to be a bitter weed.")
		case ctx.Dark:
			lines = append(lines, "It is too dark to pick carefully, so the haul is thin.")
		}
	case gathering.Firewood:
		wet := false
		if condition, ok := m.weatherIn(room.Zone); ok && !room.IsIndoor() {
			wet = cfg.IsWet(condition.Name)
		}
		hasFS := false
		if m.specialist != nil {
			_, hasFS = m.specialist(user.UserId, archetypes.UtilityFieldSmith, room.RoomId)
		}
		dry, damp := cfg.RollFirewood(gathering.FirewoodContext{Axe: m.hasTool(user, toolAxe), FieldSmith: hasFS, Wet: wet})
		drops = gathering.Add(drops, cfg.Items.Firewood, dry)
		drops = gathering.Add(drops, cfg.Items.DampFirewood, damp)
		if damp > 0 {
			lines = append(lines, "The rain has soaked much of the wood; some of it comes back damp.")
		}
	case gathering.Fishing:
		drops = cfg.RollFish(gathering.FishContext{Zone: room.Zone, Ranger: hasForager}, m.rng)
		if len(drops) == 0 {
			lines = append(lines, "The water gives up nothing.")
		}
	case gathering.Game:
		ctx := gathering.HuntContext{Zone: room.Zone, ForageLevel: sp.Level, HasForaging: hasForager, Snares: !m.hasTool(user, toolBow)}
		drops = cfg.RollHunt(ctx, m.rng)
		if len(drops) == 0 {
			if ctx.Snares {
				lines = append(lines, "The snares come up empty.")
			} else {
				lines = append(lines, "The game slips away before a shot can be taken.")
			}
		}
	}

	m.effort(user.UserId, room.RoomId, rule.EffortPc)

	if len(drops) > 0 {
		op := fmt.Sprintf("gather-%d-%d", user.UserId, a.Started.UnixNano())
		got, left := m.deliver(user, drops, op)
		if len(got) > 0 {
			who := "Your company gathers"
			switch a.Kind {
			case gathering.Fishing:
				who = "Your company lands"
			case gathering.Game:
				who = "Your company brings back"
			}
			line := fmt.Sprintf("%s %s.", who, strings.Join(got, ", "))
			if hasForager && !sp.IsLeader && (a.Kind == gathering.Herbs || a.Kind == gathering.Game || a.Kind == gathering.Fishing) {
				line += fmt.Sprintf(" %s's woodcraft helps.", sp.Name)
			}
			lines = append(lines, line)
		}
		if left {
			lines = append(lines, "There was more, but your company can carry no more.")
		}
	}
	if a.Kind == gathering.Fishing && m.settingsLineBreaks(user, cfg) {
		lines = append(lines, "A fishing line snaps and is lost.")
	}
	rest := m.chargesLeft(room, a.Kind, rule)
	if rest < 1 {
		lines = append(lines, fmt.Sprintf("The %s here is picked clean for now.", verbs[a.Kind].noun))
		events.AddToQueue(events.RoomResourcesChanged{RoomId: room.RoomId}) // clients redraw the marker
	}
	user.SendText(strings.Join(lines, "\n"))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s company finishes %s.`, user.Character.Name, verbs[a.Kind].present), user.UserId)

	bonus := 0
	if a.Kind == gathering.Game {
		bonus = cfg.HuntEncounterBonus
	}
	m.encounter(user.UserId, room.RoomId, bonus)
}

// settingsLineBreaks rolls whether the fishing line breaks, spending one.
func (m *GatheringModule) settingsLineBreaks(user *users.UserRecord, cfg gathering.Settings) bool {
	if !m.rng.Pct(cfg.LineBreakPct) {
		return false
	}
	return m.spendItem(user.UserId, cfg.Items.FishingLine)
}

func (m *GatheringModule) chargesLeft(room *rooms.Room, kind gathering.Kind, rule gathering.Rule) int {
	if rooms.IsEphemeralRoomId(room.RoomId) {
		return rule.PoolMax
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ledger.Charges(room.RoomId, kind, rule, m.clock())
}

// deliver hands the haul to the company: the cargo, or the leader's pack
// when there is no cargo, as much as the company can carry. It returns the
// named items taken and whether anything was left behind.
func (m *GatheringModule) deliver(user *users.UserRecord, drops []gathering.Drop, op string) (names []string, left bool) {
	taken := map[int]int{}
	var order []int
	grams := 0
	for _, d := range drops {
		weight := 0
		if spec := items.GetItemSpec(d.ItemID); spec != nil {
			weight = spec.Weight
		}
		for i := 0; i < d.Count; i++ {
			if m.wouldExceed(user.UserId, grams+weight) {
				left = true
				continue
			}
			grams += weight
			if taken[d.ItemID] == 0 {
				order = append(order, d.ItemID)
			}
			taken[d.ItemID]++
		}
	}
	if len(order) == 0 {
		return nil, left
	}
	stacks := make([]encumbrance.CargoStack, 0, len(order))
	for _, id := range order {
		stacks = append(stacks, encumbrance.CargoStack{ItemId: id, Count: taken[id]})
	}
	if err := m.deposit(user.UserId, op, stacks); err != nil {
		if !errors.Is(err, encumbrance.ErrNoCargo) {
			mudlog.Warn("gathering: deposit", "leader", user.UserId, "error", err)
			return nil, true
		}
		for _, id := range order { // no cargo: the leader's own pack
			for i := 0; i < taken[id]; i++ {
				m.giveItem(user, id)
			}
		}
	}
	for _, id := range order {
		names = append(names, named(id, taken[id]))
	}
	return names, left
}
