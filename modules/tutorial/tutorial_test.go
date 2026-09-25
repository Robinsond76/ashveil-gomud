package tutorial

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

// course is a module on fake rooms: templates 900..905, copies at +1000 on
// each placement (+1000 again for a second placement).
type course struct {
	m         *TutorialModule
	user      *users.UserRecord
	rooms     map[int]*rooms.Room
	original  map[int]int
	members   []company.MemberView
	form      company.Formation
	claimed   map[int]bool
	given     []int
	looked    []int
	relocated []int
	messages  *[]string
	next      int
	// Phase 27b.
	food, drink bool
	tier        camping.Tier
	reporting   bool
	struck      int
	survivalUp  bool
	strikeErr   error
}

func newCourse(t *testing.T) *course {
	t.Helper()
	c := &course{rooms: map[int]*rooms.Room{}, original: map[int]int{}, claimed: map[int]bool{}, next: 1000, reporting: true, survivalUp: true}
	m := newModule()
	m.roomIDs = func() []int { return []int{900, 901, 902, 903, 904, 905} }
	m.copyRooms = func(ids ...int) (map[int]int, error) {
		out := map[int]int{}
		for _, id := range ids {
			copyID := id + c.next
			c.rooms[copyID] = &rooms.Room{RoomId: copyID, Title: "copy"}
			c.original[copyID] = id
			out[id] = copyID
		}
		c.next += 1000
		return out, nil
	}
	m.loadRoom = func(id int) *rooms.Room { return c.rooms[id] }
	m.originalRoom = func(id int) int {
		if o, ok := c.original[id]; ok {
			return o
		}
		return id
	}
	m.moveTo = func(userID, roomID int) error {
		from := c.user.Character.RoomId
		c.user.Character.RoomId = roomID
		m.onRoomChange(events.RoomChange{UserId: userID, FromRoomId: from, ToRoomId: roomID})
		return nil
	}
	m.look = func(_ *users.UserRecord, roomID int) { c.looked = append(c.looked, roomID) }
	m.relocate = func(_, roomID int) int { c.relocated = append(c.relocated, roomID); return 0 }
	m.lookupUser = func(id int) *users.UserRecord {
		if c.user != nil && c.user.UserId == id {
			return c.user
		}
		return nil
	}
	m.members = func(int) ([]company.MemberView, bool) { return c.members, true }
	m.formation = func(int) (company.Formation, bool) { return c.form, true }
	m.hasClaimed = func(_ int, id int) bool { return c.claimed[id] }
	m.giveItem = func(_ *users.UserRecord, id int) bool { c.given = append(c.given, id); return true }
	m.registered = func(string) bool { return true }
	m.carries = func(_ *users.UserRecord, drink bool) bool {
		if drink {
			return c.drink
		}
		return c.food
	}
	m.restTier = func(int) (camping.Tier, bool) { return c.tier, true }
	m.restReporting = func() bool { return c.reporting }
	m.abandonCamp = func(int) error {
		if c.strikeErr != nil {
			return c.strikeErr
		}
		c.struck++
		return nil
	}
	m.survivalUp = func() bool { return c.survivalUp }
	c.m = m

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	c.user = users.NewUserRecord(7, 1)
	c.user.Character.RoomId = -1
	users.SetTestUser(c.user)
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == 7 {
			messages = append(messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.ProcessEvents(); events.UnregisterListener(events.Message{}, id) })
	c.messages = &messages
	return c
}

func (c *course) text() string {
	events.ProcessEvents()
	out := tagPattern.ReplaceAllString(strings.Join(*c.messages, "\n"), "")
	*c.messages = nil
	return out
}

func (c *course) run(cmd string) {
	c.m.onCommandDone(usercommands.CommandDone{UserId: 7, Command: cmd})
	c.m.check(c.user)
}

func (c *course) exit(roomID int, name string) (int, bool) {
	r := c.rooms[roomID]
	if r == nil {
		return 0, false
	}
	e, ok := r.ExitsTemp[name]
	return e.RoomId, ok
}

func (c *course) stage() StageID { return progressOf(c.user.Character).Stage }

// at puts the player at a stage, as if walked there.
func (c *course) at(id StageID) {
	p := progressOf(c.user.Character)
	p.Stage = id
	p.save(c.user.Character)
}

func inspectionsOf(id StageID) []string { return stages[stageIndex(id)].Inspections }

func TestBeginPlacesAtTheFirstStage(t *testing.T) {
	c := newCourse(t)
	require.True(t, c.m.Begin(7))
	assert.Equal(t, 1900, c.user.Character.RoomId)
	assert.Equal(t, []int{1900}, c.looked, "placement shows the room")
	assert.Equal(t, []int{1900}, c.relocated, "the company comes along")
	assert.Equal(t, -1, c.user.Character.RoomIdOnReset)
	assert.Equal(t, StageCharacter, c.stage())
	assert.Equal(t, stateActive, progressOf(c.user.Character).State)
	assert.Contains(t, c.text(), "stage 1 of 6: Your character")
	_, open := c.exit(1900, "east")
	assert.False(t, open, "the way on is closed until the stage is passed")
}

func TestAdvanceUnlocksNextRoom(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.text()
	for _, cmd := range inspectionsOf(StageCharacter)[:3] {
		c.run(cmd)
	}
	assert.Equal(t, StageCharacter, c.stage())
	c.run("look") // not an inspection
	c.run("conditions")
	assert.Equal(t, StageCompany, c.stage())
	to, open := c.exit(1900, "east")
	require.True(t, open)
	assert.Equal(t, 1901, to)
	assert.Contains(t, c.text(), "Head east for the next lesson: Your company")

	// Walking in shows the stage.
	require.NoError(t, c.m.moveTo(7, 1901))
	assert.Contains(t, c.text(), "stage 2 of 6: Your company")

	// Two living companions pass Company; a formation passes Formation.
	c.members = []company.MemberView{{ID: 1, Status: company.MemberPresent}, {ID: 2, Status: company.MemberPresent}}
	c.m.check(c.user)
	assert.Equal(t, StageFormation, c.stage())
	require.NoError(t, c.form.Place(company.CompanionMemberKey(1), 0, 1))
	require.NoError(t, c.form.Place(company.CompanionMemberKey(2), 2, 1))
	c.m.check(c.user)
	assert.Equal(t, StageSurvival, c.stage())
	to, open = c.exit(1902, "east")
	require.True(t, open)
	assert.Equal(t, 1904, to, "the Weather Yard follows the Drill Ground")

	// Survival: fed, watered, and the four inspections.
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Nutrition: 35}})
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Hydration: 40}})
	for _, cmd := range inspectionsOf(StageSurvival) {
		assert.Equal(t, StageSurvival, c.stage())
		c.run(cmd)
	}
	assert.Equal(t, StageCamp, c.stage())
	to, open = c.exit(1904, "east")
	require.True(t, open)
	assert.Equal(t, 1905, to)

	// Camp: Rested passes it, and the camp is struck.
	struck := c.struck
	c.m.check(c.user)
	assert.Equal(t, StageCamp, c.stage(), "not rested yet")
	c.tier = camping.TierRested
	c.m.check(c.user)
	assert.Equal(t, StageDeparture, c.stage())
	assert.Equal(t, struck+1, c.struck, "the course camp is struck")
	to, open = c.exit(1905, "east")
	require.True(t, open)
	assert.Equal(t, 1903, to, "on to the Gate")
	to, open = c.exit(1903, "gate")
	require.True(t, open)
	assert.Equal(t, rooms.StartRoomIdAlias, to)
	c.m.check(c.user)
	assert.Equal(t, StageDeparture, c.stage(), "departure ends at the gate")
}

func TestGraduateOnce(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	p := progressOf(c.user.Character)
	p.Stage = StageDeparture
	p.save(c.user.Character)
	c.user.Character.RoomId = 1903
	require.NoError(t, c.m.moveTo(7, 1))
	assert.Equal(t, []int{defaultGraduationItem}, c.given)
	assert.Equal(t, 1, c.relocated[len(c.relocated)-1], "the company leaves with the player")
	assert.Equal(t, stateGraduated, progressOf(c.user.Character).State)
	assert.Zero(t, c.user.Character.RoomIdOnReset)
	assert.Contains(t, c.text(), "You have finished your training")

	// Walking back through anything later gives nothing more.
	c.user.Character.RoomId = 1903
	require.NoError(t, c.m.moveTo(7, 1))
	assert.Len(t, c.given, 1)
}

func TestLeavingEarlyIsASkip(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	require.NoError(t, c.m.moveTo(7, 1)) // e.g. a portal or recall
	assert.Equal(t, stateSkipped, progressOf(c.user.Character).State)
	assert.Empty(t, c.given)
}

func TestSkipGivesNothing(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.text()
	_, _ = c.m.command("skip", c.user, nil, 0)
	assert.Contains(t, c.text(), "tutorial skip yes")
	assert.Equal(t, stateActive, progressOf(c.user.Character).State, "asks first")
	_, _ = c.m.command("skip yes", c.user, nil, 0)
	assert.Equal(t, stateSkipped, progressOf(c.user.Character).State)
	assert.Equal(t, rooms.StartRoomIdAlias, c.user.Character.RoomId)
	assert.Equal(t, rooms.StartRoomIdAlias, c.relocated[len(c.relocated)-1], "companions leave with the player")
	assert.Empty(t, c.given)
	_, _ = c.m.command("", c.user, nil, 0)
	assert.Contains(t, c.text(), "You left the tutorial")
}

func TestResumeRebuildsAtStage(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	p := progressOf(c.user.Character)
	p.Stage = StageFormation
	p.save(c.user.Character)
	// Logged out: the engine sent them to the Void; the copies are gone.
	c.user.Character.RoomId = -1
	delete(c.m.copies, 7)
	c.m.resume(7)
	assert.Equal(t, 2902, c.user.Character.RoomId, "the Drill Ground, in fresh copies")
	to, open := c.exit(2900, "east")
	require.True(t, open)
	assert.Equal(t, 2901, to)
	_, open = c.exit(2901, "east")
	assert.True(t, open, "every passed stage's way is open")
	_, open = c.exit(2902, "east")
	assert.False(t, open, "the current one isn't")
	assert.Contains(t, c.text(), "back where your training left off")

	c.m.resume(7)
	assert.Equal(t, 2902, c.user.Character.RoomId, "already there: nothing moves")
}

func TestNextOnlyWhenAllowed(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	p := progressOf(c.user.Character)
	p.Stage = StageCompany
	p.save(c.user.Character)
	c.text()
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageCompany, c.stage())
	assert.Contains(t, c.text(), "Not yet")
	c.claimed[61], c.claimed[62] = true, true
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageFormation, c.stage(), "both recruits already claimed: waived")

	// Formation: waived only while the company is short and can't refill
	// from the course's recruits (e.g. one was dismissed after Company).
	c.members = []company.MemberView{view(1, company.MemberPresent), view(2, company.MemberPresent)}
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageFormation, c.stage(), "two companions: set the formation instead")
	c.members = []company.MemberView{view(1, company.MemberPresent)}
	c.claimed[62] = false
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageFormation, c.stage(), "a recruit is still claimable")
	c.claimed[62] = true
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageSurvival, c.stage(), "short and nothing left to claim: waived")

	// Survival: waived only once supplied and with nothing left to eat or
	// drink that's still needed.
	c.food, c.drink = true, true
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageSurvival, c.stage(), "not supplied yet")
	p = progressOf(c.user.Character)
	p.Supplied = true
	p.save(c.user.Character)
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageSurvival, c.stage(), "food and drink in the pack")
	c.drink = false
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Hydration: 10}})
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageSurvival, c.stage(), "out of drink, but already watered")
	c.food = false
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageSurvival, c.stage(), "the meal is waived, the inspections aren't")
	assert.True(t, progressOf(c.user.Character).Seen[seenFed])
	for _, cmd := range inspectionsOf(StageSurvival) {
		c.run(cmd)
	}
	assert.Equal(t, StageCamp, c.stage())

	// Camp: waived only when nothing can camp.
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageCamp, c.stage())
	c.reporting = false
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageDeparture, c.stage(), "no camping: waived")
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageDeparture, c.stage(), "departure is walked, not waived")
}

func TestTutorialViewChecklist(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.run("status")
	c.text()
	_, _ = c.m.command("", c.user, nil, 0)
	out := c.text()
	assert.Contains(t, out, "[x] status")
	assert.Contains(t, out, "[ ] inventory")
	assert.Contains(t, out, "Goal:")
}

// TestBeginAgainKeepsProgress: a second Begin (start typed in the Void
// before resume) carries on at the saved stage, and a finished course is
// never restarted.
func TestBeginAgainKeepsProgress(t *testing.T) {
	c := newCourse(t)
	require.True(t, c.m.Begin(7))
	for _, cmd := range inspectionsOf(StageCharacter) {
		c.run(cmd)
	}
	c.m.check(c.user)
	require.Equal(t, StageCompany, c.stage())
	c.user.Character.RoomId = -1
	require.True(t, c.m.Begin(7))
	assert.Equal(t, StageCompany, c.stage(), "not reset to the first stage")
	assert.Equal(t, 901+c.next-1000, c.user.Character.RoomId, "placed in the stage's room of fresh copies")

	progress{State: stateGraduated, Stage: StageDeparture}.save(c.user.Character)
	c.user.Character.RoomId = -1
	c.given = nil
	require.True(t, c.m.Begin(7))
	assert.Equal(t, rooms.StartRoomIdAlias, c.user.Character.RoomId, "out to the start room")
	assert.Equal(t, stateGraduated, progressOf(c.user.Character).State)
	assert.Empty(t, c.given)
}

// --- Phase 27b ---

func TestInspectionsCountOnlyInTheirStage(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.run("weather")
	assert.False(t, progressOf(c.user.Character).Seen["weather"], "a Survival inspection doesn't count at Character")
	c.at(StageSurvival)
	c.run("weather")
	c.run("status")
	p := progressOf(c.user.Character)
	assert.True(t, p.Seen["weather"])
	assert.False(t, p.Seen["status"], "nor a Character one at Survival")
}

func TestProvisionCountsOnlyInSurvival(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Nutrition: 35, Hydration: 5}})
	assert.False(t, progressOf(c.user.Character).Seen[seenFed])
	c.at(StageSurvival)
	c.m.onProvision(survival.Provisioned{LeaderUserID: 8, Benefit: survival.Benefit{Nutrition: 35}})
	assert.False(t, progressOf(c.user.Character).Seen[seenFed], "someone else's meal")
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Nutrition: 35}})
	p := progressOf(c.user.Character)
	assert.True(t, p.Seen[seenFed])
	assert.False(t, p.Seen[seenWatered], "food alone")

	for _, cmd := range inspectionsOf(StageSurvival) {
		c.run(cmd)
	}
	assert.Equal(t, StageSurvival, c.stage(), "still thirsty work")
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Hydration: 40}})
	c.m.check(c.user)
	assert.Equal(t, StageCamp, c.stage())
}

func TestMissingInspectionCommandIsNotAskedFor(t *testing.T) {
	c := newCourse(t)
	c.m.registered = func(cmd string) bool { return cmd != "strain" }
	c.m.Begin(7)
	c.at(StageSurvival)
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Nutrition: 1, Hydration: 1}})
	for _, cmd := range []string{"weather", "temperature", "cargo"} {
		c.run(cmd)
	}
	assert.Equal(t, StageCamp, c.stage())
}

// walkIn puts the player at stage id's previous stage and walks them into
// its room, as the east exit would.
func (c *course) walkIn(t *testing.T, id StageID) {
	t.Helper()
	c.at(id)
	at := stageIndex(id)
	from := c.m.copyOf(7, stages[at-1].Room)
	c.user.Character.RoomId = from
	require.NoError(t, c.m.moveTo(7, c.m.copyOf(7, stages[at].Room)))
}

func TestSuppliesOnceAndOnlyWhatsMissing(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.food = true // the kit's sandwich is still in the pack
	c.walkIn(t, StageSurvival)
	assert.Equal(t, []int{defaultWaterItem}, c.given, "only the missing drink")
	assert.True(t, progressOf(c.user.Character).Supplied)
	assert.Contains(t, c.text(), "stage 4 of 6: Survival")

	c.food, c.drink = false, false
	c.walkIn(t, StageSurvival)
	c.m.resume(7)
	delete(c.m.copies, 7)
	c.user.Character.RoomId = -1
	c.m.resume(7)
	assert.Equal(t, []int{defaultWaterItem}, c.given, "never again, by walking in or by resume")
}

func TestSuppliesOnResumeAtSurvival(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.at(StageSurvival)
	delete(c.m.copies, 7)
	c.user.Character.RoomId = -1
	c.m.resume(7)
	assert.Equal(t, 2904, c.user.Character.RoomId)
	assert.Equal(t, []int{defaultRationItem, defaultWaterItem}, c.given, "an empty pack gets both")
}

func TestCampStruckOnPlaceSkipAndLeave(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	assert.Equal(t, 1, c.struck, "placement strikes any stale course camp")
	c.at(StageCamp)
	delete(c.m.copies, 7)
	c.user.Character.RoomId = -1
	c.m.resume(7)
	assert.Equal(t, 2, c.struck, "and so does resume")
	_, _ = c.m.command("skip yes", c.user, nil, 0)
	assert.Equal(t, 3, c.struck, "and skipping")

	d := newCourse(t)
	d.m.Begin(7)
	d.at(StageCamp)
	require.NoError(t, d.m.moveTo(7, 1)) // walked out some other way
	assert.Equal(t, 2, d.struck, "and leaving")
	assert.Equal(t, stateSkipped, progressOf(d.user.Character).State)
}

func TestSurvivalAndCampViews(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.at(StageSurvival)
	c.m.onProvision(survival.Provisioned{LeaderUserID: 7, Benefit: survival.Benefit{Nutrition: 35}})
	c.run("cargo")
	c.text()
	_, _ = c.m.command("", c.user, nil, 0)
	out := c.text()
	assert.Contains(t, out, "stage 4 of 6: Survival")
	assert.Contains(t, out, "[x] eat something")
	assert.Contains(t, out, "[ ] drink something")
	assert.Contains(t, out, "[x] cargo")
	assert.Contains(t, out, "[ ] weather")

	c.at(StageCamp)
	_, _ = c.m.command("", c.user, nil, 0)
	out = c.text()
	assert.Contains(t, out, "[ ] Rested")
	assert.Contains(t, out, "camp rest")
}

// --- Phase 27b review ---

func TestSurvivalWaiverWithoutSurvival(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.at(StageSurvival)
	c.food, c.drink = true, true // the kit is in the pack, but eating can't count
	c.survivalUp = false
	_, _ = c.m.command("next", c.user, nil, 0)
	p := progressOf(c.user.Character)
	assert.True(t, p.Seen[seenFed] && p.Seen[seenWatered], "the meal is waived")
	assert.Equal(t, StageSurvival, c.stage())
	for _, cmd := range inspectionsOf(StageSurvival) {
		c.run(cmd)
	}
	assert.Equal(t, StageCamp, c.stage())
	_, _ = c.m.command("next", c.user, nil, 0)
	assert.Equal(t, StageDeparture, c.stage(), "no rest without survival: Camp waived")
}

func TestFailedStrikeIsRetried(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.at(StageCamp)
	c.strikeErr = assert.AnError
	struck := c.struck
	_, _ = c.m.command("skip yes", c.user, nil, 0)
	assert.Equal(t, stateSkipped, progressOf(c.user.Character).State)
	assert.Equal(t, "yes", c.user.Character.GetMiscData(keyStrike), "a retry is owed")
	c.m.check(c.user)
	assert.Equal(t, struck, c.struck, "still failing")
	c.strikeErr = nil
	c.m.check(c.user)
	assert.Equal(t, struck+1, c.struck, "retried after the course")
	assert.Nil(t, c.user.Character.GetMiscData(keyStrike))
	c.m.check(c.user)
	assert.Equal(t, struck+1, c.struck, "once")
}

func TestLogoutInCourseStrikesTheCamp(t *testing.T) {
	c := newCourse(t)
	c.m.Begin(7)
	c.at(StageCamp)
	struck := c.struck
	c.m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.Equal(t, struck+1, c.struck)

	progress{State: stateGraduated, Stage: StageDeparture}.save(c.user.Character)
	c.m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.Equal(t, struck+1, c.struck, "a real-world camp is left alone")
}

func TestEdibleWaterCountsAsDrink(t *testing.T) {
	assert.True(t, provisionKind(items.ItemSpec{Subtype: items.Edible, Hydration: 5}, true))
	assert.True(t, provisionKind(items.ItemSpec{Subtype: items.Drinkable, Hydration: 5}, true))
	assert.False(t, provisionKind(items.ItemSpec{Subtype: items.Drinkable}, true))
	assert.True(t, provisionKind(items.ItemSpec{Subtype: items.Edible, Nutrition: 5}, false))
	assert.False(t, provisionKind(items.ItemSpec{Subtype: items.Drinkable, Hydration: 5}, false))
}
