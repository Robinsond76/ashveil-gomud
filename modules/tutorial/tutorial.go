// Package tutorial is the Ashveil tutorial (Phase 27a): a course of stages
// in per-player copies of the tutorial rooms, each passed by a real result
// (inspections run, companions recruited, a formation set), with durable
// progress on the character, resume after logout, restart, or copyover, a
// skip, and a once-only graduation reward. See
// docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md.
// Phase 27b adds the Survival and Camp lessons:
// docs/superpowers/specs/2026-09-25-phase-27b-tutorial-survival-camp-design.md.
// Phase 27c adds the practice fight:
// docs/superpowers/specs/2026-09-25-phase-27c-tutorial-practice-fight-design.md.
package tutorial

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	domain "github.com/GoMudEngine/GoMud/internal/tutorial"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//go:embed files/*
var files embed.FS

const (
	defaultGraduationItem = 20043
	// The Survival stage's supplies (27b): a cheese sandwich and a
	// waterskin, as in the starter kits.
	defaultRationItem = 30004
	defaultWaterItem  = 30015
	// exitLifetime keeps an opened way open as long as anyone could need it.
	exitLifetime = "7 real days"
)

var defaultRecruits = []int{61, 62}

// defaultSquad is the practice fight's foes (27c): three straw footmen and
// a straw archer, who stands behind them.
var defaultSquad = []int{67, 67, 67, 68}

// fight is a player's practice squad: its room copy, the foes still
// standing, and how many were raised. In memory only, like the copies:
// resume raises a fresh squad.
type fight struct {
	room     int
	standing map[int]bool
	raised   int
}

// TutorialModule runs the course.
type TutorialModule struct {
	plug *plugins.Plugin

	// Seams; natives by default.
	lookupUser   func(userID int) *users.UserRecord
	roomIDs      func() []int
	copyRooms    func(roomIDs ...int) (map[int]int, error)
	loadRoom     func(roomID int) *rooms.Room
	moveTo       func(userID, roomID int) error
	look         func(user *users.UserRecord, roomID int)
	relocate     func(leaderUserID, roomID int) int
	originalRoom func(roomID int) int
	members      func(leaderUserID int) ([]company.MemberView, bool)
	formation    func(leaderUserID int) (company.Formation, bool)
	hasClaimed   func(leaderUserID, templateID int) bool
	giveItem     func(user *users.UserRecord, itemID int) bool
	// Phase 27b.
	registered    func(command string) bool
	carries       func(user *users.UserRecord, drink bool) bool
	restTier      func(userID int) (camping.Tier, bool)
	restReporting func() bool
	abandonCamp   func(leaderUserID int) error
	survivalUp    func() bool
	// Phase 27c.
	spawnFoe  func(mobID, roomID int) (instanceID int, ok bool)
	removeFoe func(instanceID int)
	foeHere   func(instanceID, roomID int) bool

	graduationItem int
	rationItem     int
	waterItem      int
	recruits       []int
	squad          []int

	// copies maps each player in the course to their room copies, template
	// room ID to copy ID. In memory only: resume makes new ones.
	copies map[int]map[int]int
	// fights maps each player at the practice fight to their squad.
	fights map[int]*fight
}

var module *TutorialModule

var _ domain.Provider = (*TutorialModule)(nil)

func newModule() *TutorialModule {
	return &TutorialModule{
		lookupUser: users.GetByUserId,
		roomIDs:    configuredRooms,
		copyRooms:  rooms.CreateEphemeralRoomIds,
		loadRoom:   rooms.LoadRoom,
		moveTo:     func(userID, roomID int) error { return rooms.MoveToRoom(userID, roomID) },
		look: func(user *users.UserRecord, roomID int) {
			if room := rooms.LoadRoom(roomID); room != nil {
				usercommands.Look(``, user, room, events.CmdSecretly)
			}
		},
		originalRoom: rooms.GetOriginalRoom,
		relocate:     company.RelocateCompany,
		members:      company.CompanyMembers,
		formation:    company.FormationFor,
		hasClaimed:   company.HasClaimed,
		giveItem: func(user *users.UserRecord, itemID int) bool {
			if items.GetItemSpec(itemID) == nil {
				return false
			}
			return user.Character.StoreItem(items.New(itemID))
		},
		registered: usercommands.IsRegistered,
		carries:    carriesProvision,
		restTier: func(userID int) (camping.Tier, bool) {
			tier, _, ok := camping.RestTierOf(userID)
			return tier, ok
		},
		restReporting:  camping.RestReporting,
		abandonCamp:    camping.AbandonCamp,
		survivalUp:     survival.CompanyServiceAvailable,
		spawnFoe:       spawnPracticeFoe,
		removeFoe:      removePracticeFoe,
		foeHere:        practiceFoeHere,
		squad:          defaultSquad,
		fights:         map[int]*fight{},
		graduationItem: defaultGraduationItem,
		rationItem:     defaultRationItem,
		waterItem:      defaultWaterItem,
		recruits:       defaultRecruits,
		copies:         map[int]map[int]int{},
	}
}

func init() {
	m := newModule()
	m.plug = plugins.New("tutorial", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.AddUserCommand("tutorial", m.command, true, false)
	companyview.OnRefresh.Register(func(r companyview.Refreshed) companyview.Refreshed {
		m.check(r.User)
		return r
	})
	usercommands.OnCommandDone.Register(func(d usercommands.CommandDone) usercommands.CommandDone {
		m.onCommandDone(d)
		return d
	})
	mobcommands.OnPracticeBeaten.Register(func(b mobcommands.PracticeBeaten) mobcommands.PracticeBeaten {
		m.onPracticeBeaten(b)
		return b
	})
	survival.OnProvision.Register(func(p survival.Provisioned) survival.Provisioned {
		m.onProvision(p)
		return p
	})
	events.RegisterListener(events.RoomChange{}, m.onRoomChange)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.PlayerDespawn{}, m.onPlayerDespawn)
	events.RegisterListener(resumeTutorial{}, m.onResume)
	domain.SetProvider(m)
	module = m
}

func configuredRooms() []int {
	var out []int
	for _, s := range configs.GetSpecialRoomsConfig().TutorialRooms {
		if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

func (m *TutorialModule) load() {
	if n, ok := configInt(m.plug.Config.Get("GraduationItemId")); ok && n > 0 {
		m.graduationItem = n
	}
	if n, ok := configInt(m.plug.Config.Get("RationItemId")); ok && n > 0 {
		m.rationItem = n
	}
	if n, ok := configInt(m.plug.Config.Get("WaterItemId")); ok && n > 0 {
		m.waterItem = n
	}
	if list, ok := m.plug.Config.Get("PracticeSquad").([]any); ok {
		var ids []int
		for _, raw := range list {
			if n, ok := configInt(raw); ok && n > 0 {
				ids = append(ids, n)
			}
		}
		if len(ids) > 0 {
			m.squad = ids
		}
	}
	if list, ok := m.plug.Config.Get("TutorialRecruits").([]any); ok {
		var ids []int
		for _, raw := range list {
			if n, ok := configInt(raw); ok && n > 0 {
				ids = append(ids, n)
			}
		}
		if len(ids) > 0 {
			m.recruits = ids
		}
	}
	if len(m.roomIDs()) < len(stages) {
		mudlog.Error("tutorial: fewer TutorialRooms than stages", "rooms", len(m.roomIDs()), "stages", len(stages))
	}
}

// --- placement ---

// place makes fresh room copies for a player in the course, opens the way
// up to their current stage's room, and moves them there.
func (m *TutorialModule) place(user *users.UserRecord, p progress) bool {
	ids := m.roomIDs()
	at := stageIndex(p.Stage)
	if at < 0 || !m.available() {
		return false
	}
	copies, err := m.copyRooms(ids...)
	if err != nil {
		mudlog.Error("tutorial: room copies", "user", user.UserId, "error", err)
		return false
	}
	m.copies[user.UserId] = copies
	// Fresh copies, fresh squad: the old one stood in the old copies.
	m.clearSquad(user.UserId)
	// No course camp outlives its room copies. A logout strikes it already;
	// this catches a crash or restart, where a rest that ran its minute
	// meanwhile has completed and still grants Rested.
	m.strikeCamp(user)
	for i := 0; i < at; i++ {
		m.openAfter(user.UserId, i)
	}
	if stages[at].ID == StageDeparture {
		m.openGate(user.UserId)
	}
	// Leaving mid-course sends the engine's player to the Void; resume
	// brings them back.
	user.Character.RoomIdOnReset = -1
	target := copies[ids[stages[at].Room]]
	if err := m.travel(user, target); err != nil {
		mudlog.Error("tutorial: place", "user", user.UserId, "room", target, "error", err)
		return false
	}
	m.sendStage(user, at)
	m.enterStage(user, p)
	return true
}

// available reports whether every stage has its room: a deployment whose
// TutorialRooms lacks one can't run the course (27d review).
func (m *TutorialModule) available() bool {
	ids := m.roomIDs()
	for _, s := range stages {
		if s.Room < 0 || s.Room >= len(ids) {
			return false
		}
	}
	return true
}

// closeCourse lets a player in the course out when it can't run: skipped,
// with no reward, wherever the engine put them.
func (m *TutorialModule) closeCourse(user *users.UserRecord, p progress) {
	p.State = stateSkipped
	p.save(user.Character)
	delete(m.copies, user.UserId)
	user.Character.RoomIdOnReset = 0
	m.strikeCamp(user)
	m.clearSquad(user.UserId)
	mudlog.Error("tutorial: course unavailable", "user", user.UserId, "rooms", len(m.roomIDs()), "stages", len(stages))
	user.SendText(`The training grounds are closed, so your training ends here. Your journey begins.`)
}

// travel moves a player (and their living companions, which only follow
// on foot) into a room and shows it.
func (m *TutorialModule) travel(user *users.UserRecord, roomID int) error {
	if err := m.moveTo(user.UserId, roomID); err != nil {
		return err
	}
	m.relocate(user.UserId, user.Character.RoomId)
	m.look(user, user.Character.RoomId)
	return nil
}

// copyOf is a player's copy of the tutorial room at index i; 0 when none.
func (m *TutorialModule) copyOf(userID, i int) int {
	ids := m.roomIDs()
	if i < 0 || i >= len(ids) {
		return 0
	}
	return m.copies[userID][ids[i]]
}

// openAfter opens the way east from stage i's room to stage i+1's.
func (m *TutorialModule) openAfter(userID, i int) {
	if i+1 >= len(stages) {
		return
	}
	from := m.loadRoom(m.copyOf(userID, stages[i].Room))
	to := m.copyOf(userID, stages[i+1].Room)
	if from == nil || to == 0 {
		return
	}
	from.AddTemporaryExit("east", exit.TemporaryRoomExit{RoomId: to, Title: "east", UserId: userID, Expires: exitLifetime})
	if stages[i+1].ID == StageDeparture {
		m.openGate(userID)
	}
}

// openGate opens the last room's gate to the start room.
func (m *TutorialModule) openGate(userID int) {
	last := stages[len(stages)-1]
	if room := m.loadRoom(m.copyOf(userID, last.Room)); room != nil {
		room.AddTemporaryExit("gate", exit.TemporaryRoomExit{RoomId: rooms.StartRoomIdAlias, Title: "gate", UserId: userID, Expires: exitLifetime})
	}
}

// inCourse reports whether a room is one of the tutorial rooms (a copy's
// template is).
func (m *TutorialModule) inCourse(roomID int) bool {
	original := m.originalRoom(roomID)
	for _, id := range m.roomIDs() {
		if id == original {
			return true
		}
	}
	return false
}

// --- the course ---

// Begin implements internal/tutorial.Provider: a new character starts at
// the first stage.
func (m *TutorialModule) Begin(userID int) bool {
	user := m.lookupUser(userID)
	if user == nil || user.Character == nil {
		return false
	}
	switch prior := progressOf(user.Character); prior.State {
	case stateActive:
		if !m.available() {
			m.closeCourse(user, prior)
			return false
		}
		// Already in the course (e.g. "start" typed in the Void before
		// resume ran): carry on at their stage, never from the top.
		if stageIndex(prior.Stage) < 0 {
			prior.Stage = stages[0].ID
			prior.save(user.Character)
		}
		return m.place(user, prior)
	case stateSkipped, stateGraduated:
		// The course is done and doesn't restart: out to the start room.
		return m.travel(user, rooms.StartRoomIdAlias) == nil
	}
	p := progress{State: stateActive, Stage: stages[0].ID, Seen: map[string]bool{}}
	p.save(user.Character)
	if !m.place(user, p) {
		progress{}.save(user.Character)
		return false
	}
	return true
}

func (m *TutorialModule) sendStage(user *users.UserRecord, at int) {
	s := stages[at]
	lines := []string{
		"",
		fmt.Sprintf(`<ansi fg="yellow-bold">Tutorial, stage %d of %d: %s</ansi>`, at+1, len(stages), s.Title),
		s.Intro,
		fmt.Sprintf(`<ansi fg="yellow">Goal:</ansi> %s`, s.Goal),
		`Type <ansi fg="command">tutorial</ansi> at any time for your goal and hints, or <ansi fg="command">tutorial skip</ansi> to leave the course.`,
	}
	user.SendText(strings.Join(lines, "\n"))
}

func (m *TutorialModule) onCommandDone(d usercommands.CommandDone) {
	user := m.lookupUser(d.UserId)
	if user == nil || user.Character == nil {
		return
	}
	p := progressOf(user.Character)
	at := stageIndex(p.Stage)
	if p.State != stateActive || at < 0 {
		return
	}
	for _, key := range stages[at].Inspections {
		if inspectionMatches(key, d.Command, d.Rest) && !p.Seen[key] {
			p.Seen[key] = true
			p.save(user.Character)
			return
		}
	}
}

// onProvision counts a real meal and drink in the Survival stage, whoever
// in the company was fed.
func (m *TutorialModule) onProvision(e survival.Provisioned) {
	user := m.lookupUser(e.LeaderUserID)
	if user == nil || user.Character == nil {
		return
	}
	p := progressOf(user.Character)
	if p.State != stateActive || p.Stage != StageSurvival {
		return
	}
	changed := false
	if e.Benefit.Nutrition > 0 && !p.Seen[seenFed] {
		p.Seen[seenFed], changed = true, true
	}
	if e.Benefit.Hydration > 0 && !p.Seen[seenWatered] {
		p.Seen[seenWatered], changed = true, true
	}
	if changed {
		p.save(user.Character)
	}
}

// enterStage readies the stage a player has just reached (walked into or
// been placed in): Survival's supplies, once per character and only for
// what they lack.
func (m *TutorialModule) enterStage(user *users.UserRecord, p progress) {
	if p.Stage == StageCombat {
		m.raiseSquad(user)
		return
	}
	if p.Stage != StageSurvival || p.Supplied {
		return
	}
	p.Supplied = true
	p.save(user.Character)
	for _, supply := range []struct {
		drink  bool
		itemID int
	}{{false, m.rationItem}, {true, m.waterItem}} {
		if m.carries(user, supply.drink) || !m.giveItem(user, supply.itemID) {
			continue
		}
		if spec := items.GetItemSpec(supply.itemID); spec != nil {
			user.SendText(fmt.Sprintf(`A warden presses a <ansi fg="item">%s</ansi> into your hands.`, spec.Name))
		}
	}
}

// carriesProvision reports whether a character carries something to eat
// or, with drink, to drink.
func carriesProvision(user *users.UserRecord, drink bool) bool {
	for _, item := range user.Character.GetAllBackpackItems() {
		if provisionKind(item.GetSpec(), drink) {
			return true
		}
	}
	return false
}

// provisionKind: food is edible with nutrition; drink is anything eaten or
// drunk with hydration, as eat and drink provision it.
func provisionKind(spec items.ItemSpec, drink bool) bool {
	if drink {
		return (spec.Subtype == items.Drinkable || spec.Subtype == items.Edible) && spec.Hydration > 0
	}
	return spec.Subtype == items.Edible && spec.Nutrition > 0
}

// strikeCamp removes a player's camp: in the course it can only be a
// course camp, in a room copy that won't outlast the course. A failure
// (camping can't save) is owed as a retry, which check makes every round
// until it succeeds, even after the player has left the course.
func (m *TutorialModule) strikeCamp(user *users.UserRecord) {
	if err := m.abandonCamp(user.UserId); err != nil {
		mudlog.Error("tutorial: strike camp", "user", user.UserId, "error", err)
		user.Character.SetMiscData(keyStrike, "yes")
		return
	}
	user.Character.SetMiscData(keyStrike, nil)
}

// retryStrike makes an owed strike.
func (m *TutorialModule) retryStrike(user *users.UserRecord) {
	if owed, _ := user.Character.GetMiscData(keyStrike).(string); owed == "yes" {
		m.strikeCamp(user)
	}
}

// onPlayerDespawn strikes a course camp on logout: its room copy won't be
// there to come back to, and a rest cut short is made again on resume.
// This runs before the engine's own leave handling.
func (m *TutorialModule) onPlayerDespawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}
	user := m.lookupUser(evt.UserId)
	if user == nil || user.Character == nil || progressOf(user.Character).State != stateActive {
		return events.Continue
	}
	m.strikeCamp(user)
	// The squad goes too: the copies it stands in are freed, and their IDs
	// may be handed back on resume.
	m.clearSquad(user.UserId)
	return events.Continue
}

// --- the practice fight (27c) ---

// raiseSquad stands a practice squad up in the player's copy of the Combat
// room, unless the whole of it still stands there. A squad in an older
// copy, or one missing a foe that wasn't beaten, is removed first.
func (m *TutorialModule) raiseSquad(user *users.UserRecord) {
	room := m.copyOf(user.UserId, stages[stageIndex(StageCombat)].Room)
	if room == 0 {
		return
	}
	if f := m.fights[user.UserId]; f != nil {
		if f.room == room && m.intact(f) {
			return
		}
		m.clearSquad(user.UserId)
	}
	f := &fight{room: room, standing: map[int]bool{}}
	for _, mobID := range m.squad {
		if id, ok := m.spawnFoe(mobID, room); ok {
			f.standing[id] = true
			f.raised++
		}
	}
	m.fights[user.UserId] = f
	if f.raised == 0 {
		mudlog.Error("tutorial: no practice squad", "user", user.UserId, "squad", m.squad)
		user.SendText(`The practice yard is empty today. Type <ansi fg="command">tutorial next</ansi> to go on.`)
	}
}

// intact: every foe not yet beaten still stands in the squad's room.
func (m *TutorialModule) intact(f *fight) bool {
	for id := range f.standing {
		if !m.foeHere(id, f.room) {
			return false
		}
	}
	return true
}

// clearSquad removes a player's foes still standing.
func (m *TutorialModule) clearSquad(userID int) {
	f := m.fights[userID]
	if f == nil {
		return
	}
	for id := range f.standing {
		m.removeFoe(id)
	}
	delete(m.fights, userID)
}

// onPracticeBeaten counts a beaten foe for the player whose squad it was.
func (m *TutorialModule) onPracticeBeaten(b mobcommands.PracticeBeaten) {
	for _, f := range m.fights {
		if f.standing[b.InstanceId] {
			delete(f.standing, b.InstanceId)
			return
		}
	}
}

// squadBeaten: the Combat gate, every foe raised was beaten.
func (m *TutorialModule) squadBeaten(userID int) bool {
	f := m.fights[userID]
	return f != nil && f.raised > 0 && len(f.standing) == 0
}

func spawnPracticeFoe(mobID, roomID int) (int, bool) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, false
	}
	mob := mobs.NewMobByIdNoElite(mobs.MobId(mobID), roomID, 0)
	if mob == nil {
		return 0, false
	}
	room.AddMob(mob.InstanceId)
	return mob.InstanceId, true
}

func practiceFoeHere(instanceID, roomID int) bool {
	mob := mobs.GetInstance(instanceID)
	return mob != nil && mob.Character.RoomId == roomID
}

func removePracticeFoe(instanceID int) {
	if mob := mobs.GetInstance(instanceID); mob != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			room.RemoveMob(instanceID)
		}
	}
	mobs.DestroyInstance(instanceID)
}

// rested: the Camp gate, a rest tier held.
func (m *TutorialModule) rested(userID int) bool {
	tier, ok := m.restTier(userID)
	return ok && tier >= camping.TierRested
}

// passed reports whether the player's current stage's gate is met.
func (m *TutorialModule) passed(user *users.UserRecord, p progress) bool {
	switch p.Stage {
	case StageCharacter:
		return characterDone(p, m.registered)
	case StageCompany:
		members, ok := m.members(user.UserId)
		return ok && companyDone(members)
	case StageFormation:
		members, ok := m.members(user.UserId)
		f, fok := m.formation(user.UserId)
		return ok && fok && formationDone(f, members)
	case StageSurvival:
		return survivalDone(p, m.registered)
	case StageCamp:
		return m.rested(user.UserId)
	case StageCombat:
		return m.squadBeaten(user.UserId)
	case StageAlignment:
		return inspected(stages[stageIndex(StageAlignment)], p, m.registered)
	}
	return false // Departure ends at the gate
}

// check runs on every refresh (each round and after each command): a met
// gate advances the course.
func (m *TutorialModule) check(user *users.UserRecord) {
	if user == nil || user.Character == nil {
		return
	}
	m.retryStrike(user)
	p := progressOf(user.Character)
	// A foe gone without being beaten (despawned, torn down) is made good
	// while the player is in the yard.
	if p.State == stateActive && p.Stage == StageCombat && user.Character.RoomId == m.copyOf(user.UserId, stages[stageIndex(StageCombat)].Room) {
		m.raiseSquad(user)
	}
	if p.State != stateActive || !m.passed(user, p) {
		return
	}
	m.advance(user, p)
}

// advance moves a player on to the next stage and opens the way to it.
func (m *TutorialModule) advance(user *users.UserRecord, p progress) {
	at := stageIndex(p.Stage)
	if at < 0 || at+1 >= len(stages) {
		return
	}
	p.Stage = stages[at+1].ID
	p.save(user.Character)
	switch stages[at].ID {
	case StageCamp:
		m.strikeCamp(user) // rested; the camp's work is done
	case StageCombat:
		m.clearSquad(user.UserId) // beaten, or waived
	}
	m.openAfter(user.UserId, at)
	done := stages[at].Done
	if done == "" {
		done = "Stage passed."
	}
	user.SendText(fmt.Sprintf(`<ansi fg="green">Well done.</ansi> %s Head <ansi fg="exit">east</ansi> for the next lesson: %s.`, done, stages[at+1].Title))
}

// onRoomChange shows a stage when its room is entered, and ends the
// course when a player walks out of it.
func (m *TutorialModule) onRoomChange(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId <= 0 {
		return events.Continue
	}
	user := m.lookupUser(evt.UserId)
	if user == nil || user.Character == nil {
		return events.Continue
	}
	p := progressOf(user.Character)
	if p.State != stateActive {
		return events.Continue
	}
	from, to := m.inCourse(evt.FromRoomId), m.inCourse(evt.ToRoomId)
	switch {
	case from && !to:
		m.leave(user, p)
	case to:
		if at := stageIndex(p.Stage); at >= 0 && m.copyOf(user.UserId, stages[at].Room) == evt.ToRoomId && evt.FromRoomId != evt.ToRoomId && from {
			m.sendStage(user, at)
			m.enterStage(user, p)
		}
	}
	return events.Continue
}

// leave ends the course: graduation, with its reward once, when the player
// reached Departure; otherwise it counts as skipped.
//
// Any move out counts, not only the gate: a death respawn or an admin
// teleport also ends the course, as a skip without confirmation. No 27a
// room can kill a player (the rooms spawn nothing); 27c's practice fight
// must decide this before death is reachable here.
func (m *TutorialModule) leave(user *users.UserRecord, p progress) {
	delete(m.copies, user.UserId)
	user.Character.RoomIdOnReset = 0
	m.strikeCamp(user)
	m.clearSquad(user.UserId)
	// Companions follow on foot a moment later; bring them now, so none
	// is left behind in the course's copies.
	m.relocate(user.UserId, user.Character.RoomId)
	if p.Stage != StageDeparture {
		p.State = stateSkipped
		p.save(user.Character)
		mudlog.Info("tutorial: left early", "user", user.UserId, "stage", p.Stage)
		return
	}
	p.State = stateGraduated
	p.save(user.Character)
	mudlog.Info("tutorial: graduated", "user", user.UserId)
	text := `<ansi fg="green-bold">You have finished your training.</ansi>`
	if m.giveItem(user, m.graduationItem) {
		if spec := items.GetItemSpec(m.graduationItem); spec != nil {
			text += fmt.Sprintf(` You receive a <ansi fg="item">%s</ansi>.`, spec.Name)
		}
	}
	user.SendText(text + ` Your journey begins.`)
}

// --- resume ---

// resumeTutorial is queued by PlayerSpawn, so it runs after the engine has
// placed the player.
type resumeTutorial struct{ UserId int }

func (resumeTutorial) Type() string { return "TutorialResume" }

// The engine's own PlayerSpawn handling runs first: a player saved in the
// Void with RoomIdOnReset -1 lands in the start room for a moment, and the
// queued resume then brings them back to the course. That hop is expected.
func (m *TutorialModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		events.AddToQueue(resumeTutorial{UserId: evt.UserId})
	}
	return events.Continue
}

func (m *TutorialModule) onResume(e events.Event) events.ListenerReturn {
	evt, ok := e.(resumeTutorial)
	if !ok {
		return events.Continue
	}
	m.resume(evt.UserId)
	return events.Continue
}

// resume brings a player in the course back to their stage when they are
// no longer in its rooms (logout, restart, copyover).
func (m *TutorialModule) resume(userID int) {
	user := m.lookupUser(userID)
	if user == nil || user.Character == nil {
		return
	}
	p := progressOf(user.Character)
	if p.State != stateActive {
		return
	}
	if m.inCourse(user.Character.RoomId) && m.copies[userID] != nil {
		return
	}
	if stageIndex(p.Stage) < 0 {
		p.Stage = stages[0].ID
		p.save(user.Character)
	}
	if !m.available() {
		m.closeCourse(user, p)
		return
	}
	user.SendText(`<ansi fg="magenta">You find yourself back where your training left off.</ansi>`)
	m.place(user, p)
}

// --- the command ---

func (m *TutorialModule) command(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	p := progressOf(user.Character)
	args := strings.Fields(strings.ToLower(rest))
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch {
	case p.State == stateGraduated:
		user.SendText("You finished the tutorial. See <ansi fg=\"command\">help</ansi> for any command.")
	case p.State == stateSkipped:
		user.SendText("You left the tutorial. See <ansi fg=\"command\">help</ansi> for any command.")
	case p.State != stateActive:
		user.SendText("You aren't in the tutorial.")
	case sub == "skip":
		m.skip(user, p, len(args) > 1 && args[1] == "yes")
	case sub == "next":
		m.next(user, p)
	default:
		user.SendText(m.view(user, p))
	}
	return true, nil
}

// view is the current stage, goal, checklist, and hints.
func (m *TutorialModule) view(user *users.UserRecord, p progress) string {
	at := stageIndex(p.Stage)
	if at < 0 {
		return "Your place in the tutorial was lost; type tutorial skip yes to leave it."
	}
	s := stages[at]
	lines := []string{
		fmt.Sprintf(`<ansi fg="yellow-bold">Tutorial, stage %d of %d: %s</ansi>`, at+1, len(stages), s.Title),
		fmt.Sprintf(`<ansi fg="yellow">Goal:</ansi> %s`, s.Goal),
	}
	check := func(done bool, what string) {
		mark := "[ ]"
		if done {
			mark = "[x]"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", mark, what))
	}
	switch s.ID {
	case StageSurvival:
		check(p.Seen[seenFed], "eat something")
		check(p.Seen[seenWatered], "drink something")
	case StageCamp:
		check(m.rested(user.UserId), "Rested")
	case StageCombat:
		beaten, raised := 0, len(m.squad)
		if f := m.fights[user.UserId]; f != nil {
			beaten, raised = f.raised-len(f.standing), f.raised
		}
		check(m.squadBeaten(user.UserId), fmt.Sprintf("beat the straw soldiers (%d of %d)", beaten, raised))
	}
	for _, cmd := range required(s, m.registered) {
		check(p.Seen[cmd], cmd)
	}
	for _, h := range s.Hints {
		lines = append(lines, "  - "+h)
	}
	lines = append(lines, `<ansi fg="black-bold">tutorial next (when a stage can't be done), tutorial skip (leave the course, without its reward)</ansi>`)
	return strings.Join(lines, "\n")
}

// next lets a stuck player through a stage they can no longer finish:
// Company or Formation short of companions with the course's recruits
// claimed, Survival with nothing left to eat or drink, Camp without camping.
func (m *TutorialModule) next(user *users.UserRecord, p progress) {
	// Company and Formation both need two living companions. Once both
	// free recruits are claimed (TutorialRecruits, which must match the
	// company module's Muster Yard candidates) a player short of two, say
	// after a dismissal, can't get them back here, so the lesson is waived.
	if p.Stage == StageCompany || p.Stage == StageFormation {
		members, _ := m.members(user.UserId)
		claimedAll := len(m.recruits) > 0
		for _, id := range m.recruits {
			if !m.hasClaimed(user.UserId, id) {
				claimedAll = false
			}
		}
		if !companyDone(members) && claimedAll {
			user.SendText("You've already claimed this course's recruits and have too few companions for this lesson, so it is waived.")
			m.advance(user, p)
			return
		}
	}
	// Survival: the meal alone is waived when it can't be had (no
	// survival, or the supplies are gone and nothing is left); the
	// inspections are still asked for.
	if p.Stage == StageSurvival {
		up := m.survivalUp()
		waived := false
		for _, part := range []struct {
			seen  string
			drink bool
		}{{seenFed, false}, {seenWatered, true}} {
			if !p.Seen[part.seen] && (!up || (p.Supplied && !m.carries(user, part.drink))) {
				p.Seen[part.seen], waived = true, true
			}
		}
		if waived {
			p.save(user.Character)
			user.SendText("You can't eat or drink here now, so that part of the lesson is waived. Buy food and water at a market before a long road.")
			if m.passed(user, p) {
				m.advance(user, p)
			} else {
				user.SendText(`Check the weather, temperature, strain, and cargo to finish. Type <ansi fg="command">tutorial</ansi> for the checklist.`)
			}
			return
		}
	}
	// Camp: nothing here can camp or rest.
	// Combat: there's no yard, or no squad could be raised in it.
	if f := m.fights[user.UserId]; p.Stage == StageCombat &&
		(m.copyOf(user.UserId, stages[stageIndex(StageCombat)].Room) == 0 || (f != nil && f.raised == 0)) {
		user.SendText("There's no one to practice against, so this lesson is waived.")
		m.advance(user, p)
		return
	}
	if p.Stage == StageCamp && (!m.restReporting() || !m.survivalUp()) {
		user.SendText("Camping isn't available right now, so this lesson is waived.")
		m.advance(user, p)
		return
	}
	if at := stageIndex(p.Stage); at >= 0 {
		user.SendText(fmt.Sprintf(`Not yet. <ansi fg="yellow">Goal:</ansi> %s Type <ansi fg="command">tutorial</ansi> for hints.`, stages[at].Goal))
	}
}

// skip leaves the course for the start room, with no graduation reward.
func (m *TutorialModule) skip(user *users.UserRecord, p progress, confirmed bool) {
	if !confirmed {
		user.SendText(`Leave the tutorial now? You won't get its graduation reward. Type <ansi fg="command">tutorial skip yes</ansi> to leave.`)
		return
	}
	p.State = stateSkipped
	p.save(user.Character)
	delete(m.copies, user.UserId)
	user.Character.RoomIdOnReset = 0
	m.strikeCamp(user) // before moving: a rest in progress holds the player
	m.clearSquad(user.UserId)
	user.SendText(`<ansi fg="magenta">You leave the training grounds behind.</ansi>`)
	if err := m.travel(user, rooms.StartRoomIdAlias); err != nil {
		mudlog.Error("tutorial: skip", "user", user.UserId, "error", err)
	}
}
