// Package testarea is the admin test area: a closed set of rooms an admin
// teleports into as their own character to try combat, camping, classes,
// gear, horses and weather, and a way back that leaves the character exactly
// as it was. It never runs for anyone but an admin, and nothing earned, lost
// or changed inside it survives the return.
//
// A trip is a Session: the whole user record plus every module's per-user
// state (internal/userstate), saved the moment the trip starts. The rooms
// (zones Test Area and Test Area Road, rooms 90001-90010) have no exit to
// the rest of the world, so no walk, route or journey leaves them.
//
// Nothing here advances the world's clock: weather is the zone's own, and
// the shared day and night are never touched.
package testarea

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/userstate"
)

const (
	// HubRoom is where a trip begins.
	HubRoom = 90001
	// FirstRoom and LastRoom bound the test rooms' ids.
	FirstRoom = 90001
	LastRoom  = 90010
)

// Zones are the test area's zones.
var Zones = []string{"Test Area", "Test Area Road"}

// roomNames are the words that name a test room.
var roomNames = map[string]int{
	"hub": 90001, "yard": 90002, "combat": 90002, "camp": 90003, "armory": 90004, "armoury": 90004,
	"stable": 90005, "weather": 90006, "deck": 90006, "cellar": 90007, "dark": 90007,
	"pass": 90008, "narrow": 90008, "thicket": 90009, "ambush": 90009, "road": 90010, "thieves": 90010,
}

// roomList is what `testarea rooms` shows, in order.
var roomList = []struct {
	ID   int
	Word string
	Use  string
}{
	{90001, "hub", "the sign and the way to everything else"},
	{90002, "yard", "combat: call foes with testarea fight"},
	{90008, "pass", "combat on narrow ground"},
	{90009, "thicket", "combat with an ambush"},
	{90003, "camp", "a quiet camp with water, firewood and gathering"},
	{90010, "road", "a camp thieves and raiders always visit"},
	{90004, "armory", "free gear: testarea kit, give, items"},
	{90005, "stable", "horses for sale"},
	{90006, "weather", "weather controls for the zone"},
	{90007, "cellar", "darkness, for light sources"},
}

// Module is the test area's state.
type Module struct {
	plug *plugins.Plugin

	mu       sync.Mutex
	store    Store
	sessions map[int]Session
	loadErr  error
}

var registered *Module

func init() {
	m := &Module{plug: plugins.New("testarea", "1.0"), sessions: map[int]Session{}}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("testarea", m.command, true, true)
	m.plug.Callbacks.SetOnLoad(m.load)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	registered = m
}

func (m *Module) load() {
	sessions, err := m.store.Load()
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.loadErr = err
		mudlog.Error("testarea: load", "error", err)
		return
	}
	m.sessions, m.loadErr = sessions, nil
}

func (m *Module) persistLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("the saved trips are unreadable (%v); fix the file and reload", m.loadErr)
	}
	return m.store.Save(m.sessions)
}

func (m *Module) session(userID int) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[userID]
	return s, ok
}

// onPlayerSpawn reminds an admin who logs in mid-trip.
func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	if _, in := m.session(evt.UserId); in {
		if user := users.GetByUserId(evt.UserId); user != nil {
			user.SendText(`<ansi fg="alert-3">You are on a test area trip: nothing here is kept. Type </ansi><ansi fg="command">testarea return</ansi><ansi fg="alert-3"> to go back to where you were.</ansi>`)
		}
	}
	return events.Continue
}

// onUserPurged forgets a purged user's trip.
func (m *Module) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, held := m.sessions[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.sessions, evt.UserId)
	if err := m.persistLocked(); err != nil {
		mudlog.Error("testarea: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}

// IsAreaRoom reports whether a room id is one of the test rooms.
func IsAreaRoom(roomID int) bool { return roomID >= FirstRoom && roomID <= LastRoom }

func isAdmin(user *users.UserRecord) bool {
	return user.Role == users.RoleAdmin || user.HasRolePermission("testarea")
}

func say(user *users.UserRecord, text string) {
	if text != "" {
		user.SendText(text)
	}
}

func cmd(s string) string { return `<ansi fg="command">` + s + `</ansi>` }

// command is the `testarea` command.
func (m *Module) command(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !isAdmin(user) {
		say(user, "You don't have permission to use testarea.")
		return true, nil
	}
	fields := strings.Fields(rest)
	sub := ""
	if len(fields) > 0 {
		sub = strings.ToLower(fields[0])
	}
	args := []string{}
	if len(fields) > 1 {
		args = fields[1:]
	}
	switch sub {
	case "help", "?":
		out, _ := templates.Process("admincommands/help/command.testarea", nil, user.UserId)
		say(user, out)
	case "return", "back", "leave":
		m.returnBack(user)
	case "status":
		m.status(user)
	case "rooms":
		say(user, roomsText())
	case "", "enter":
		say(user, m.enter(user, "hub"))
	default:
		if _, known := m.tool(sub); known {
			say(user, m.runTool(user, sub, args))
		} else {
			say(user, m.enter(user, sub))
		}
	}
	return true, nil
}

func roomsText() string {
	var b strings.Builder
	b.WriteString("Test rooms (testarea <word> goes to one):\n")
	for _, r := range roomList {
		fmt.Fprintf(&b, "  %s (%d): %s\n", cmd(r.Word), r.ID, r.Use)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Module) status(user *users.UserRecord) {
	s, in := m.session(user.UserId)
	if !in {
		say(user, "You are not on a test area trip. "+cmd("testarea")+" starts one.")
		return
	}
	say(user, fmt.Sprintf("You are on a test area trip begun %s. %s returns you to room %d exactly as you were.",
		s.Taken.Format("15:04:05"), cmd("testarea return"), s.Room))
}

// target resolves a room word or id to a test room.
func target(word string) (int, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	if id, ok := roomNames[word]; ok {
		return id, true
	}
	if id, err := strconv.Atoi(word); err == nil && IsAreaRoom(id) {
		return id, true
	}
	return 0, false
}

// enter starts a trip, or moves within one, and describes the arrival.
func (m *Module) enter(user *users.UserRecord, word string) string {
	roomID, ok := target(word)
	if !ok {
		return fmt.Sprintf("There is no test room called %q. %s lists them.", word, cmd("testarea rooms"))
	}
	if rooms.LoadRoom(roomID) == nil {
		return "This world has no test area: its rooms (90001-90010) are missing."
	}
	if actionpolicy.InBattle(user) {
		return actionpolicy.BattleUnderWay
	}
	if _, in := m.session(user.UserId); !in {
		if blocked, why := camping.MovementBlocked(user.UserId); blocked {
			return why
		}
		if parties.Get(user.UserId) != nil {
			return "Leave your party before you start a test trip."
		}
		snap, err := takeSession(user)
		if err != nil {
			return "The trip did not start: " + err.Error()
		}
		m.mu.Lock()
		m.sessions[user.UserId] = snap
		err = m.persistLocked()
		if err != nil {
			delete(m.sessions, user.UserId)
		}
		m.mu.Unlock()
		if err != nil {
			return "The trip did not start, because it could not be saved: " + err.Error()
		}
		say(user, `<ansi fg="alert-3">Your state is saved. Nothing you gain, lose or change here is kept; </ansi>`+cmd("testarea return")+`<ansi fg="alert-3"> puts you back.</ansi>`)
	}
	if err := m.move(user, roomID); err != nil {
		return err.Error()
	}
	return ""
}

// move puts the user and their company in a room and looks.
func (m *Module) move(user *users.UserRecord, roomID int) error {
	from := user.Character.RoomId
	if from == roomID {
		_, _ = usercommands.TryCommand("look", "", user.UserId, 0)
		return nil
	}
	if err := rooms.MoveToRoom(user.UserId, roomID); err != nil {
		return err
	}
	if company.RelocateCompany(user.UserId, from, user.Character.RoomId) > 0 {
		say(user, company.CompanyFollows)
	}
	_, _ = usercommands.TryCommand("look", "", user.UserId, 0)
	return nil
}

// returnBack ends the trip: the record and every module's state come back
// as saved, and the user stands where they started.
func (m *Module) returnBack(user *users.UserRecord) {
	snap, in := m.session(user.UserId)
	if !in {
		say(user, "You are not on a test area trip.")
		return
	}
	if actionpolicy.InBattle(user) {
		say(user, actionpolicy.BattleUnderWay+" "+cmd("testarea clear")+" sends the foes away.")
		return
	}
	restored, err := snap.restoredRecord(user)
	if err != nil {
		say(user, "The return failed: "+err.Error()+" Your trip is kept; try again.")
		return
	}
	dest := snap.Room
	if rooms.LoadRoom(dest) == nil {
		dest = rooms.StartRoomIdAlias
	}
	if err := rooms.MoveToRoom(user.UserId, dest); err != nil {
		say(user, "The return failed: "+err.Error()+" Your trip is kept; try again.")
		return
	}
	restored.Character.RoomId = user.Character.RoomId
	users.UpdateOnlineUser(*restored)
	errs := userstate.RestoreAll(user.UserId, user.Character.RoomId, snap.stateBytes())
	if len(errs) > 0 {
		for _, e := range errs {
			mudlog.Error("testarea: restore", "user", user.UserId, "error", e)
		}
		say(user, fmt.Sprintf("The return was not complete (%v). Your trip is kept; try %s again.", errs[0], cmd("testarea return")))
		return
	}
	if err := users.SaveUser(*user); err != nil {
		mudlog.Error("testarea: save user after return", "user", user.UserId, "error", err)
	}
	m.mu.Lock()
	delete(m.sessions, user.UserId)
	saveErr := m.persistLocked()
	m.mu.Unlock()
	if saveErr != nil {
		mudlog.Error("testarea: save after return", "user", user.UserId, "error", saveErr)
	}
	say(user, `<ansi fg="alert-3">You are back as you were. Nothing from the test area was kept.</ansi>`)
	_, _ = usercommands.TryCommand("look", "", user.UserId, 0)
}

// sortedKeys is a helper for stable listings.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
