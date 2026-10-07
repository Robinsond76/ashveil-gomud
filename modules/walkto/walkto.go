// Package walkto is Phase 40d click-to-walk: the `walkto` command and the web
// map's click on a visited tile. The leader walks a route planned over rooms
// they have already visited (internal/walkto), one ordinary `go` per step,
// StepMillis of real time apart, so strain, encounters, followers, hunger
// and weather all apply as for a typed move. Walking never advances the game
// clock and is not persisted: a restart or copyover just ends it.
//
// Timers only queue an event (walktoTimerDue); every step runs on the game
// loop. The plan state lives in internal/walkto so the GMCP feed can read it.
package walkto

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walkto"
)

//go:embed files/*
var files embed.FS

const (
	defaultStepMillis = 1500
	minStepMillis     = 250
	maxStepMillis     = 10000
)

// keepsWalking are the typed commands that never interrupt a walk.
var keepsWalking = map[string]bool{"look": true, "l": true, "map": true, "walkto": true, "autowalk": true}

// WalktoModule drives click-to-walk.
type WalktoModule struct {
	plug *plugins.Plugin

	mu     sync.Mutex
	issued map[int]string // the move text a walk just queued, so the Input listener lets it pass
	tried  map[int]tried  // the step a walk last queued, so a refused move stops the walk

	stepEvery time.Duration

	// seams, replaced by tests
	lookupUser func(id int) *users.UserRecord
	loadRoom   func(id int) *rooms.Room
	after      func(d time.Duration, f func())
	move       func(userID int, text string)
	inBattle   func(*users.UserRecord) bool
	summary    func(*users.UserRecord) companyview.Summary
	hostileIn  func(room *rooms.Room, userID int) string
}

var module *WalktoModule

func init() {
	m := newModule()
	module = m
	m.plug = plugins.New("walkto", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.AddUserCommand("walkto", m.command, true, false)
	events.RegisterListener(events.Input{}, m.onInput)
	events.RegisterListener(events.RoomChange{}, m.onRoomChange)
	events.RegisterListener(events.UserPurged{}, m.onUserGone)
	events.RegisterListener(events.PlayerDespawn{}, m.onDespawn)
	events.RegisterListener(timerDue{}, onTimerDue)
}

func newModule() *WalktoModule {
	return &WalktoModule{
		issued:     map[int]string{},
		tried:      map[int]tried{},
		stepEvery:  defaultStepMillis * time.Millisecond,
		lookupUser: users.GetByUserId,
		loadRoom:   rooms.LoadRoom,
		after: func(d time.Duration, f func()) {
			time.AfterFunc(d, func() { events.AddToQueue(timerDue{run: f}) })
		},
		move: func(userID int, text string) {
			events.AddToQueue(events.Input{UserId: userID, InputText: text})
		},
		inBattle:  actionpolicy.InBattle,
		summary:   companyview.For,
		hostileIn: hostileMob,
	}
}

// tried is the walk generation and path index whose step was last queued.
type tried struct {
	gen uint64
	idx int
}

// timerDue carries a step timer's callback onto the game loop.
type timerDue struct{ run func() }

func (timerDue) Type() string { return `WalktoTimerDue` }

func onTimerDue(e events.Event) events.ListenerReturn {
	if due, ok := e.(timerDue); ok && due.run != nil {
		due.run()
	}
	return events.Continue
}

func (m *WalktoModule) load() {
	if m.plug == nil {
		return
	}
	ms := defaultStepMillis
	switch v := m.plug.Config.Get("StepMillis").(type) {
	case int:
		ms = v
	case int64:
		ms = int(v)
	case float64:
		ms = int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			ms = n
		}
	}
	if ms < minStepMillis || ms > maxStepMillis {
		ms = defaultStepMillis
	}
	m.stepEvery = time.Duration(ms) * time.Millisecond
}

// Active reports whether a user is walking (tests, views).
func (m *WalktoModule) Active(userID int) bool {
	_, ok := walkto.Current(userID)
	return ok
}

// --- the world as a graph ---

// graphFor reads the live rooms the way this player sees them.
func (m *WalktoModule) graphFor(user *users.UserRecord) walkto.Graph {
	memo := map[int]*walkto.Node{}
	return func(id int) (walkto.Node, bool) {
		if n, ok := memo[id]; ok {
			if n == nil {
				return walkto.Node{}, false
			}
			return *n, true
		}
		room := m.loadRoom(id)
		if room == nil {
			memo[id] = nil
			return walkto.Node{}, false
		}
		node := walkto.Node{
			ID: id, Zone: room.Zone, Title: room.Title, Legend: room.MapLegend,
			Visited: id == user.Character.RoomId || user.Character.HasVisitedRoom(id, room.Zone),
		}
		for name, info := range room.Exits {
			e := walkto.Exit{Name: name, To: info.RoomId}
			switch {
			case info.TravelProfile != "":
				e.Block = walkto.BlockJourney
			case info.ExitMessage != "":
				e.Block = walkto.BlockMessage
			case info.Secret:
				dest := m.loadRoom(info.RoomId)
				if dest == nil || !user.Character.SeesSecretExit(id, name, info.RoomId, dest.Zone) {
					e.Block = walkto.BlockSecret
				}
			}
			if info.Lock.IsLocked() {
				e.Locked = true
				e.HasKey, _ = user.Character.HasKey(fmt.Sprintf(`%d-%s`, id, name), int(info.Lock.Difficulty))
			}
			node.Exits = append(node.Exits, e)
		}
		memo[id] = &node
		return node, true
	}
}

// --- the command ---

const usage = "Usage: <ansi fg=\"command\">walkto [place]</ansi> walks to a place you have visited: a room number from the map, or a word like <ansi fg=\"command\">inn</ansi>. <ansi fg=\"command\">walkto stop</ansi> halts. See <ansi fg=\"command\">help walkto</ansi>."

func (m *WalktoModule) command(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	arg := strings.TrimSpace(rest)
	lower := strings.ToLower(arg)

	switch lower {
	case "stop", "halt", "cancel", "end":
		if !m.stop(user.UserId, "You stop walking.") {
			user.SendText("You are not walking anywhere.")
		}
		return true, nil
	case "":
		if a, ok := walkto.Current(user.UserId); ok {
			user.SendText(m.statusLine(user, a))
		} else {
			user.SendText(usage)
		}
		return true, nil
	}

	if refusal := m.cannotWalk(user); refusal != "" {
		user.SendText(refusal)
		return true, nil
	}

	g := m.graphFor(user)
	from := user.Character.RoomId
	var target int
	if n, err := strconv.Atoi(arg); err == nil {
		target = n
	} else {
		id, err := walkto.Resolve(g, from, arg)
		if err != nil {
			user.SendText(fmt.Sprintf("You don't know a place called %q here. Try a room from the map you have visited, or a word like <ansi fg=\"command\">inn</ansi>.", arg))
			return true, nil
		}
		target = id
	}

	route, err := walkto.Plan(g, from, target)
	if err != nil {
		user.SendText(planRefusal(err))
		return true, nil
	}

	// A walk already under way is replaced by the new one.
	m.dropIssued(user.UserId)
	a := walkto.Begin(user.UserId, route, m.warns(user))
	dest, _ := g(target)
	user.SendText(fmt.Sprintf("You set out for <ansi fg=\"room-title\">%s</ansi>, %s away. Type <ansi fg=\"command\">walkto stop</ansi> to halt; any other command takes over.",
		dest.Title, stepsWord(len(route.Steps))))
	m.step(user.UserId, a.Gen)
	return true, nil
}

func planRefusal(err error) string {
	switch err {
	case walkto.ErrHere:
		return "You are already there."
	case walkto.ErrUnvisited:
		return "You haven't been there, so you can't walk there. Walkto only goes to places you have visited."
	case walkto.ErrLocked:
		return "A locked door is in the way, and you carry no key for it."
	case walkto.ErrNoPath:
		return "You know no way there from here: walkto only follows rooms you have visited and exits you can use."
	}
	if far, ok := err.(walkto.TooFarError); ok {
		return far.Error() + ". Walk part of the way, then try again."
	}
	return err.Error()
}

func stepsWord(n int) string {
	if n == 1 {
		return "1 step"
	}
	return fmt.Sprintf("%d steps", n)
}

func (m *WalktoModule) statusLine(user *users.UserRecord, a walkto.Active) string {
	dest := "your destination"
	if room := m.loadRoom(a.Target); room != nil {
		dest = fmt.Sprintf("<ansi fg=\"room-title\">%s</ansi>", room.Title)
	}
	return fmt.Sprintf("You are walking to %s, %s to go. <ansi fg=\"command\">walkto stop</ansi> halts.", dest, stepsWord(len(a.Path)-1-a.Idx))
}

// cannotWalk is why this user may not start a walk, "" when they may.
func (m *WalktoModule) cannotWalk(user *users.UserRecord) string {
	if user.Character.IsDisabled() {
		return "You are unable to do that while downed."
	}
	if user.Character.Aggro != nil || m.inBattle(user) {
		return "You can't walk anywhere in the middle of a fight."
	}
	if party := parties.Get(user.UserId); party != nil && !party.IsLeader(user.UserId) {
		return "Only your party's leader can set the walk; you follow the leader."
	}
	if blocked, msg := expedition.MovementBlocked(user.UserId); blocked {
		return msg
	}
	if blocked, msg := camping.MovementBlocked(user.UserId); blocked {
		return msg
	}
	if blocked, msg := storyevents.MovementBlocked(user.UserId); blocked {
		return msg
	}
	return ""
}

// --- walking ---

// warns records which of the leader's needs already warn, so only a new
// crossing stops the walk.
func (m *WalktoModule) warns(user *users.UserRecord) [3]bool {
	s := m.summary(user)
	return [3]bool{s.Leader.Hunger.Warns(), s.Leader.Thirst.Warns(), s.Leader.Fatigue.Warns()}
}

// step runs on the game loop: reconcile where the walker is, stop for any
// reason that applies, else queue the next move and arm the next timer.
func (m *WalktoModule) step(userID int, gen uint64) {
	a, ok := walkto.Current(userID)
	if !ok || a.Gen != gen {
		return
	}
	user := m.lookupUser(userID)
	if user == nil {
		walkto.Clear(userID)
		return
	}
	cur := user.Character.RoomId
	idx := a.Idx
	switch {
	case cur == a.Path[idx]:
		// The last step was queued from here and the walker never left: the
		// move was refused (a script, a buff, no action points), so stop
		// rather than retry it every step.
		m.mu.Lock()
		last, ok := m.tried[userID]
		m.mu.Unlock()
		if ok && last.gen == gen && last.idx == idx {
			m.stop(userID, "Something keeps you from going on, so you stop walking.")
			return
		}
	case idx+1 < len(a.Path) && cur == a.Path[idx+1]:
		idx++
		walkto.Advance(userID, gen, idx)
	default:
		m.stop(userID, "You are no longer on your route, so you stop walking.")
		return
	}
	if idx == len(a.Path)-1 {
		m.arrive(user, a)
		return
	}
	if reason := m.stopReason(user, a, idx); reason != "" {
		m.stop(userID, reason)
		return
	}
	next := a.Steps[idx]
	m.mu.Lock()
	m.issued[userID] = "go " + next.Exit
	m.tried[userID] = tried{gen: gen, idx: idx}
	m.mu.Unlock()
	m.move(userID, "go "+next.Exit)
	m.after(m.stepEvery, func() { m.step(userID, gen) })
}

// stopReason is why the walk must halt before the next step, "" to go on.
func (m *WalktoModule) stopReason(user *users.UserRecord, a walkto.Active, idx int) string {
	if user.Character.IsDisabled() {
		return "You are down, so you stop walking."
	}
	if user.Character.Aggro != nil || m.inBattle(user) {
		return "A fight breaks out, so you stop walking."
	}
	room := m.loadRoom(user.Character.RoomId)
	if room == nil {
		return "You stop walking."
	}
	if who := m.hostileIn(room, user.UserId); who != "" {
		return fmt.Sprintf("%s is here and means trouble, so you stop walking.", who)
	}
	next := a.Steps[idx]
	info, ok := room.Exits[next.Exit]
	if !ok || info.RoomId != next.To || info.TravelProfile != "" || info.ExitMessage != "" {
		return "The way ahead has changed, so you stop walking."
	}
	if info.Lock.IsLocked() {
		if has, _ := user.Character.HasKey(fmt.Sprintf(`%d-%s`, room.RoomId, next.Exit), int(info.Lock.Difficulty)); !has {
			return "The door ahead is locked and you have no key, so you stop walking."
		}
	}
	s := m.summary(user)
	needs := [3]struct {
		warns bool
		word  string
	}{{s.Leader.Hunger.Warns(), s.Leader.Hunger.Label}, {s.Leader.Thirst.Warns(), s.Leader.Thirst.Label}, {s.Leader.Fatigue.Warns(), s.Leader.Fatigue.Label}}
	for i, n := range needs {
		if n.warns && !a.Warn[i] {
			return fmt.Sprintf("You are %s, so you stop walking.", strings.ToLower(n.word))
		}
	}
	if s.Leader.Fatigue.Known && s.Leader.Fatigue.Value <= 0 {
		return "You are too exhausted to go on, so you stop walking."
	}
	return ""
}

func (m *WalktoModule) arrive(user *users.UserRecord, a walkto.Active) {
	if !walkto.Clear(user.UserId) {
		return
	}
	m.dropIssued(user.UserId)
	title := "your destination"
	if room := m.loadRoom(a.Target); room != nil {
		title = fmt.Sprintf("<ansi fg=\"room-title\">%s</ansi>", room.Title)
	}
	user.SendText(fmt.Sprintf("You arrive at %s.", title))
}

// stop ends a walk and tells the player why; it reports whether one ran.
func (m *WalktoModule) stop(userID int, why string) bool {
	if !walkto.Clear(userID) {
		return false
	}
	m.dropIssued(userID)
	if user := m.lookupUser(userID); user != nil && why != "" {
		user.SendText(why)
	}
	return true
}

func (m *WalktoModule) dropIssued(userID int) {
	m.mu.Lock()
	delete(m.issued, userID)
	delete(m.tried, userID)
	m.mu.Unlock()
}

// --- listeners ---

// onInput ends a walk when its leader types anything but a look, the map or
// walkto itself. The step's own queued move passes, and so do orders given to
// followers or companions.
func (m *WalktoModule) onInput(e events.Event) events.ListenerReturn {
	in, ok := e.(events.Input)
	if !ok || in.UserId <= 0 || in.MobInstanceId > 0 {
		return events.Continue
	}
	if in.PartyFollow != nil || in.PartyAttack != nil || in.MemberOrder != nil {
		return events.Continue
	}
	if _, walking := walkto.Current(in.UserId); !walking {
		return events.Continue
	}
	text := strings.TrimSpace(in.InputText)
	m.mu.Lock()
	issued := m.issued[in.UserId]
	if issued != "" && strings.EqualFold(text, issued) {
		delete(m.issued, in.UserId)
		m.mu.Unlock()
		return events.Continue
	}
	m.mu.Unlock()
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 0 || keepsWalking[fields[0]] {
		return events.Continue
	}
	m.stop(in.UserId, "You stop walking.")
	return events.Continue
}

// onRoomChange tells the map as soon as each step lands, and ends the walk
// the moment it reaches its target instead of a step-delay later.
func (m *WalktoModule) onRoomChange(e events.Event) events.ListenerReturn {
	rc, ok := e.(events.RoomChange)
	if !ok || rc.UserId <= 0 || rc.MobInstanceId > 0 {
		return events.Continue
	}
	a, walking := walkto.Current(rc.UserId)
	if !walking || a.Idx+1 >= len(a.Path) || a.Path[a.Idx+1] != rc.ToRoomId {
		return events.Continue
	}
	walkto.Advance(rc.UserId, a.Gen, a.Idx+1)
	if a.Idx+1 == len(a.Path)-1 {
		if user := m.lookupUser(rc.UserId); user != nil {
			m.arrive(user, a)
		}
	}
	return events.Continue
}

func (m *WalktoModule) onUserGone(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.UserPurged); ok {
		m.stop(evt.UserId, "")
	}
	return events.Continue
}

func (m *WalktoModule) onDespawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerDespawn); ok {
		m.stop(evt.UserId, "")
	}
	return events.Continue
}

// hostileMob names a mob in the room that attacks the player on sight.
func hostileMob(room *rooms.Room, userID int) string {
	for _, id := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.IsCharmed() {
			continue
		}
		if parties.ReservedFrom(mob.EncounterOwner, userID) {
			continue
		}
		hostile := mob.Hostile
		if !hostile {
			for _, group := range mob.Groups {
				if mobs.IsHostile(group, userID) {
					hostile = true
					break
				}
			}
		}
		if hostile {
			return fmt.Sprintf("<ansi fg=\"mobname\">%s</ansi>", mob.Character.Name)
		}
	}
	return ""
}
