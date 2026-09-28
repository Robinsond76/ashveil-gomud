package tutorial

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// replayCourse is a course whose replay seams are recorded: the events a
// replay queues, the throwaway it builds, the archetype chosen.
type replayCourse struct {
	*course
	queued    []events.Event
	throwaway *users.UserRecord
	chosen    []string
	stale     *users.UserRecord
	offline   []int
}

func newReplayCourse(t *testing.T) *replayCourse {
	t.Helper()
	rc := &replayCourse{course: newCourse(t)}
	rc.m.seams = replaySeams{
		newReplay: func(real *users.UserRecord) (*users.UserRecord, error) {
			u := users.NewUserRecord(users.ReplayUserIdBase, 0)
			u.ReplayOf = real.UserId
			u.Character.Name = real.Character.Name
			rc.throwaway = u
			return u, nil
		},
		saveUser:        func(*users.UserRecord) error { return nil },
		queue:           func(e events.Event) { rc.queued = append(rc.queued, e) },
		playerArchetype: func(int) (string, bool) { return "rogue", true },
		chooseArchetype: func(_ int, id string) (string, bool) {
			rc.chosen = append(rc.chosen, id)
			return "You receive a rogue's kit.", true
		},
		onlineReplayOf: func(int) *users.UserRecord { return rc.stale },
		offlineReplays: func() []int { return rc.offline },
	}
	return rc
}

// asReplay makes the course's player a throwaway of user 70.
func (rc *replayCourse) asReplay() {
	rc.user.ReplayOf = 70
}

func handOffs(queued []events.Event) []events.UserHandOff {
	var out []events.UserHandOff
	for _, e := range queued {
		if h, ok := e.(events.UserHandOff); ok {
			out = append(out, h)
		}
	}
	return out
}

func TestReplayRefusals(t *testing.T) {
	for name, set := range map[string]func(rc *replayCourse){
		"inside a replay": func(rc *replayCourse) { rc.asReplay() },
		"mid-fight":       func(rc *replayCourse) { rc.user.Character.Aggro = &characters.Aggro{MobInstanceId: 1} },
		"no connection": func(rc *replayCourse) {
			u := users.NewUserRecord(7, 0)
			rc.user = u
			users.SetTestUser(u)
		},
		"the course closed": func(rc *replayCourse) { delete(rc.rooms, 907) },
		"not confirmed":     func(*replayCourse) {},
	} {
		t.Run(name, func(t *testing.T) {
			rc := newReplayCourse(t)
			set(rc)
			rest := "replay yes"
			if name == "not confirmed" {
				rest = "replay"
			}
			_, err := rc.m.command(rest, rc.user, nil, 0)
			require.NoError(t, err)
			assert.Empty(t, rc.queued, "nothing starts")
			assert.Nil(t, rc.throwaway)
			assert.NotEmpty(t, rc.text())
		})
	}
}

// TestReplayStartsAHandOff: confirmed, a throwaway is built with the real
// archetype remembered, and the real character leaves with a hand-off to
// it on the same connection.
func TestReplayStartsAHandOff(t *testing.T) {
	rc := newReplayCourse(t)
	_, err := rc.m.command("replay yes", rc.user, nil, 0)
	require.NoError(t, err)
	require.NotNil(t, rc.throwaway)
	assert.Equal(t, "rogue", rc.throwaway.Character.GetMiscData(keyReplayArchetype))
	require.Len(t, rc.queued, 2)
	despawn, ok := rc.queued[0].(events.PlayerDespawn)
	require.True(t, ok)
	assert.Equal(t, 7, despawn.UserId)
	assert.True(t, despawn.HandOff)
	assert.Equal(t, events.UserHandOff{ConnectionId: 1, FromUserId: 7, ToUserId: users.ReplayUserIdBase}, rc.queued[1])
}

// TestReplayFirstSpawnBeginsTheCourse: the throwaway's first spawn
// chooses the remembered archetype (its kit) and starts the course.
func TestReplayFirstSpawnBeginsTheCourse(t *testing.T) {
	rc := newReplayCourse(t)
	rc.asReplay()
	rc.user.Character.SetMiscData(keyReplayArchetype, "rogue")
	rc.m.resume(7)
	assert.Equal(t, []string{"rogue"}, rc.chosen)
	assert.Nil(t, rc.user.Character.GetMiscData(keyReplayArchetype))
	assert.Equal(t, StageCharacter, rc.stage())
	assert.Equal(t, 1900, rc.user.Character.RoomId)
	assert.Contains(t, rc.text(), "rogue's kit")

	// A real character's spawn with no course does nothing of the kind.
	rc2 := newReplayCourse(t)
	rc2.m.resume(7)
	assert.Empty(t, rc2.chosen)
	assert.Equal(t, stateNone, progressOf(rc2.user.Character).State)
}

// TestReplayEndsAtEveryWayOut: skipping, graduating, walking out early
// (a death's move), and the course closing hand a replay back, once; a
// real character's are untouched.
func TestReplayEndsAtEveryWayOut(t *testing.T) {
	ways := map[string]func(rc *replayCourse){
		"skip": func(rc *replayCourse) {
			_, err := rc.m.command("skip yes", rc.user, nil, 0)
			require.NoError(t, err)
		},
		"graduate": func(rc *replayCourse) {
			rc.at(StageDeparture)
			rc.m.moveTo(7, 1)
		},
		"walked out early": func(rc *replayCourse) { rc.m.moveTo(7, 1) },
		"the course closed": func(rc *replayCourse) {
			delete(rc.rooms, 907)
			rc.m.resume(7)
		},
	}
	for name, way := range ways {
		t.Run(name, func(t *testing.T) {
			rc := newReplayCourse(t)
			rc.asReplay()
			require.True(t, rc.m.Begin(7))
			if name == "the course closed" {
				rc.user.Character.RoomId = -1 // logged out and back
				delete(rc.m.copies, 7)
			}
			rc.text()
			way(rc)
			hand := handOffs(rc.queued)
			require.Len(t, hand, 1, "handed back")
			assert.Equal(t, events.UserHandOff{ConnectionId: 1, FromUserId: 7, ToUserId: 70}, hand[0])
			assert.Contains(t, rc.text(), "You set the practice character aside.")
			rc.m.handBack(rc.user)
			assert.Len(t, handOffs(rc.queued), 1, "only once")
		})
		t.Run(name+", a real character", func(t *testing.T) {
			rc := newReplayCourse(t)
			require.True(t, rc.m.Begin(7))
			way(rc)
			assert.Empty(t, handOffs(rc.queued))
		})
	}
}

// TestReplaySkipStaysOutOfTheStartRoom: a replay's skip hands back from
// where it is; only a real character goes to the start room.
func TestReplaySkipStaysOutOfTheStartRoom(t *testing.T) {
	rc := newReplayCourse(t)
	rc.asReplay()
	require.True(t, rc.m.Begin(7))
	at := rc.user.Character.RoomId
	_, err := rc.m.command("skip yes", rc.user, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, at, rc.user.Character.RoomId)
}

// TestReplayDespawnQueuesThePurge: a throwaway leaving by any way is
// purged; a real character never is.
func TestReplayDespawnQueuesThePurge(t *testing.T) {
	rc := newReplayCourse(t)
	rc.m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.Empty(t, rc.queued)
	rc.asReplay()
	rc.m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.Equal(t, []events.Event{events.UserPurged{UserId: 7}}, rc.queued)
}

// TestRealLoginEndsAStaleReplay: the real character coming in on another
// client ends their replay, whose connection closes as on logout.
func TestRealLoginEndsAStaleReplay(t *testing.T) {
	rc := newReplayCourse(t)
	rc.stale = users.NewUserRecord(users.ReplayUserIdBase, 2)
	rc.stale.ReplayOf = 7
	rc.m.onReplaySpawn(rc.user)
	require.Len(t, rc.queued, 1)
	despawn, ok := rc.queued[0].(events.PlayerDespawn)
	require.True(t, ok)
	assert.Equal(t, users.ReplayUserIdBase, despawn.UserId)
	assert.False(t, despawn.HandOff, "no hand-back: the real one is online elsewhere")
	assert.True(t, rc.m.handingBack[users.ReplayUserIdBase])

	// A throwaway's own spawn checks nothing.
	rc.queued = nil
	rc.m.onReplaySpawn(rc.stale)
	assert.Empty(t, rc.queued)
}

// TestReplaySweepAndPurge: the boot sweep purges each replay left behind;
// the purge drops the course's runtime state.
func TestReplaySweepAndPurge(t *testing.T) {
	rc := newReplayCourse(t)
	rc.offline = []int{900000003, 900000004}
	rc.m.sweepReplays()
	assert.Equal(t, []events.Event{events.UserPurged{UserId: 900000003}, events.UserPurged{UserId: 900000004}}, rc.queued)

	require.True(t, rc.m.Begin(7))
	rc.at(StageCombat)
	rc.m.raiseSquad(rc.user)
	rc.m.handingBack[7] = true
	require.NotNil(t, rc.m.copies[7])
	rc.m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Nil(t, rc.m.copies[7])
	assert.Nil(t, rc.m.fights[7])
	assert.False(t, rc.m.handingBack[7])
}

// TestFirstLessonMentionsReplay (32b help): the first lesson tells a new
// player the course can be replayed later.
func TestFirstLessonMentionsReplay(t *testing.T) {
	found := false
	for _, h := range stages[0].Hints {
		if strings.Contains(h, "tutorial replay") {
			found = true
		}
	}
	assert.True(t, found)
}
