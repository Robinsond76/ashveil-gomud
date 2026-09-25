// Package tutorial is the Ashveil tutorial (Phase 27a): a course of stages
// in per-player copies of the tutorial rooms, each passed by a real result
// (inspections run, companions recruited, a formation set), with durable
// progress on the character, resume after logout, restart, or copyover, a
// skip, and a once-only graduation reward. See
// docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md.
// Phase 27b adds the Survival and Camp lessons:
// docs/superpowers/specs/2026-09-25-phase-27b-tutorial-survival-camp-design.md.
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

	graduationItem int
	rationItem     int
	waterItem      int
	recruits       []int

	// copies maps each player in the course to their room copies, template
	// room ID to copy ID. In memory only: resume makes new ones.
	copies map[int]map[int]int
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
	survival.OnProvision.Register(func(p survival.Provisioned) survival.Provisioned {
		m.onProvision(p)
		return p
	})
	events.RegisterListener(events.RoomChange{}, m.onRoomChange)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
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
	if at < 0 || stages[at].Room >= len(ids) {
		return false
	}
	copies, err := m.copyRooms(ids...)
	if err != nil {
		mudlog.Error("tutorial: room copies", "user", user.UserId, "error", err)
		return false
	}
	m.copies[user.UserId] = copies
	// No course camp outlives its room copies: a rest cut short by a
	// logout or restart is made again in the new ones.
	m.strikeCamp(user.UserId)
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
	for _, cmd := range stages[at].Inspections {
		if d.Command == cmd && !p.Seen[cmd] {
			p.Seen[cmd] = true
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
// (with nutrition) or, with drink, to drink (with hydration).
func carriesProvision(user *users.UserRecord, drink bool) bool {
	for _, item := range user.Character.GetAllBackpackItems() {
		spec := item.GetSpec()
		switch {
		case drink && spec.Subtype == items.Drinkable && spec.Hydration > 0:
			return true
		case !drink && spec.Subtype == items.Edible && spec.Nutrition > 0:
			return true
		}
	}
	return false
}

// strikeCamp removes a player's camp: in the course it can only be a
// course camp, in a room copy that won't outlast the course.
func (m *TutorialModule) strikeCamp(userID int) {
	if err := m.abandonCamp(userID); err != nil {
		mudlog.Error("tutorial: strike camp", "user", userID, "error", err)
	}
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
	}
	return false // Departure ends at the gate
}

// check runs on every refresh (each round and after each command): a met
// gate advances the course.
func (m *TutorialModule) check(user *users.UserRecord) {
	if user == nil || user.Character == nil {
		return
	}
	p := progressOf(user.Character)
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
	if stages[at].ID == StageCamp {
		m.strikeCamp(user.UserId) // rested; the camp's work is done
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
	m.strikeCamp(user.UserId)
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
	// Survival: the supplies are gone (eaten by a companion, dropped) and
	// the player has nothing left to eat or drink.
	if p.Stage == StageSurvival && p.Supplied &&
		((!p.Seen[seenFed] && !m.carries(user, false)) || (!p.Seen[seenWatered] && !m.carries(user, true))) {
		user.SendText("You have nothing left to eat or drink, so this lesson is waived. Buy food and water at a market before a long road.")
		m.advance(user, p)
		return
	}
	// Camp: nothing here can camp or rest.
	if p.Stage == StageCamp && !m.restReporting() {
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
	m.strikeCamp(user.UserId) // before moving: a rest in progress holds the player
	user.SendText(`<ansi fg="magenta">You leave the training grounds behind.</ansi>`)
	if err := m.travel(user, rooms.StartRoomIdAlias); err != nil {
		mudlog.Error("tutorial: skip", "user", user.UserId, "error", err)
	}
}
