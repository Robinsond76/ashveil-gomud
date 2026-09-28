package tutorial

// Phase 32b: tutorial replays. A player types "tutorial replay yes" and
// their real character leaves the world as on quit; a throwaway user with
// the same name and race, level 1 with nothing, takes over the connection
// and runs the course. Any end of the course hands the connection back to
// the real character, exactly as it was, and purges the throwaway. See
// docs/superpowers/specs/2026-09-28-phase-32b-tutorial-replay-design.md.

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// keyReplayArchetype is the real character's archetype, kept on the
// throwaway until its first spawn chooses it (and grants its kit).
const keyReplayArchetype = "tutorial-replay-archetype"

// replaySeams are the world calls a replay makes; natives by default.
type replaySeams struct {
	newReplay       func(real *users.UserRecord) (*users.UserRecord, error)
	saveUser        func(u *users.UserRecord) error
	queue           func(e events.Event)
	playerArchetype func(userID int) (string, bool)
	chooseArchetype func(userID int, archetypeID string) (string, bool)
	onlineReplayOf  func(realUserID int) *users.UserRecord
	offlineReplays  func() []int
	// inFight: the player is fighting, in a battle, or set on by a foe
	// (a replay leaves the world at once, so it waits for peace).
	inFight func(user *users.UserRecord) bool
}

func nativeReplaySeams() replaySeams {
	return replaySeams{
		newReplay:       users.NewReplayUser,
		saveUser:        func(u *users.UserRecord) error { return users.SaveUser(*u) },
		queue:           func(e events.Event) { events.AddToQueue(e) },
		playerArchetype: archetypes.PlayerArchetype,
		chooseArchetype: archetypes.ChooseAtCreation,
		onlineReplayOf:  users.OnlineReplayOf,
		offlineReplays:  users.OfflineReplayUserIds,
		inFight:         nativeInFight,
	}
}

func nativeInFight(user *users.UserRecord) bool {
	if user.Character.Aggro != nil {
		return true
	}
	if _, ok := battle.Current(user.UserId); ok {
		return true
	}
	if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
		for _, id := range room.GetMobs() {
			if mob := mobs.GetInstance(id); mob != nil && mob.Character.Aggro != nil && mob.Character.Aggro.UserId == user.UserId {
				return true
			}
		}
	}
	return false
}

// replay is "tutorial replay [yes]".
func (m *TutorialModule) replay(user *users.UserRecord, confirmed bool) {
	switch {
	case user.IsReplay():
		user.SendText(`You're already replaying the tutorial. <ansi fg="command">tutorial skip yes</ansi> sets this practice character aside.`)
		return
	case m.seams.inFight(user):
		user.SendText("You're too busy to leave right now!")
		return
	case user.ConnectionId() == 0:
		user.SendText("You can't replay the tutorial from here.")
		return
	case !m.available():
		user.SendText("The training grounds are closed right now.")
		return
	case !confirmed:
		user.SendText(`Replay the tutorial as a new, level 1 character with nothing? Your character will step out of the world until you leave the course, and nothing from the replay is kept. Type <ansi fg="command">tutorial replay yes</ansi> to begin.`)
		return
	}

	throwaway, err := m.seams.newReplay(user)
	if err != nil {
		mudlog.Error("tutorial: replay", "user", user.UserId, "error", err)
		user.SendText("The training grounds can't take you right now. Try again in a moment.")
		return
	}
	if id, ok := m.seams.playerArchetype(user.UserId); ok && id != "" {
		throwaway.Character.SetMiscData(keyReplayArchetype, id)
		if err := m.seams.saveUser(throwaway); err != nil {
			mudlog.Error("tutorial: replay archetype", "user", user.UserId, "error", err)
		}
	}

	mudlog.Info("tutorial: replay", "user", user.UserId, "replay", throwaway.UserId)
	user.SendText(`<ansi fg="magenta">You step out of the world and into the training grounds, as a new recruit.</ansi>`)
	m.handOff(user, throwaway.UserId)
}

// handOff takes a user out of the world, keeping their connection, and
// gives the connection to another.
func (m *TutorialModule) handOff(from *users.UserRecord, toUserID int) {
	m.seams.queue(events.PlayerDespawn{
		UserId:        from.UserId,
		RoomId:        from.Character.RoomId,
		Username:      from.Username,
		CharacterName: from.Character.Name,
		TimeOnline:    from.GetOnlineInfo().OnlineTimeStr,
		HandOff:       true,
	})
	m.seams.queue(events.UserHandOff{ConnectionId: from.ConnectionId(), FromUserId: from.UserId, ToUserId: toUserID})
}

// handBack ends a replay: the throwaway leaves and the real character
// comes back on the same connection. It does nothing for a real
// character, and only once per replay. The throwaway's despawn queues its
// purge (onPlayerDespawn).
func (m *TutorialModule) handBack(user *users.UserRecord) bool {
	if !user.IsReplay() {
		return false
	}
	if m.handingBack[user.UserId] {
		return true
	}
	m.handingBack[user.UserId] = true
	user.SendText(`<ansi fg="magenta">You set the practice character aside.</ansi>`)
	m.handOff(user, user.ReplayOf)
	return true
}

// startReplay chooses the throwaway's archetype (granting its kit, as
// creation does) and begins the course, on its first spawn.
func (m *TutorialModule) startReplay(user *users.UserRecord) {
	if id, _ := user.Character.GetMiscData(keyReplayArchetype).(string); id != "" {
		if text, _ := m.seams.chooseArchetype(user.UserId, id); text != "" {
			user.SendText(text)
		}
		user.Character.SetMiscData(keyReplayArchetype, nil)
	}
	if !m.Begin(user.UserId) {
		user.SendText("The training grounds are unavailable right now.")
		m.handBack(user)
	}
}

// onReplaySpawn runs for every spawn: a real character logging in while
// their replay is still online (another client) ends that replay, whose
// connection closes as on logout.
func (m *TutorialModule) onReplaySpawn(user *users.UserRecord) {
	if user.IsReplay() {
		return
	}
	stale := m.seams.onlineReplayOf(user.UserId)
	if stale == nil {
		return
	}
	stale.SendText(`<ansi fg="magenta">Your character has come back into the world elsewhere, so the practice character is set aside.</ansi>`)
	m.handingBack[stale.UserId] = true
	m.seams.queue(events.PlayerDespawn{
		UserId:        stale.UserId,
		RoomId:        stale.Character.RoomId,
		Username:      stale.Username,
		CharacterName: stale.Character.Name,
		TimeOnline:    stale.GetOnlineInfo().OnlineTimeStr,
	})
}

// sweepReplays purges every replay left by a restart or crash. Copyover
// restores online replays before plugins load, so those are never swept.
func (m *TutorialModule) sweepReplays() {
	for _, id := range m.seams.offlineReplays() {
		mudlog.Info("tutorial: sweeping replay", "replay", id)
		m.seams.queue(events.UserPurged{UserId: id})
	}
}

// onUserPurged drops a purged user's course: copies, squad, and hand-back
// mark.
func (m *TutorialModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.clearSquad(evt.UserId)
	delete(m.copies, evt.UserId)
	delete(m.handingBack, evt.UserId)
	return events.Continue
}
