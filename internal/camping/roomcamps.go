package camping

import "sync"

// RoomCamp is one camp pitched in a room, as look shows it (Phase 32a).
type RoomCamp struct {
	LeaderUserID int
	FireLit      bool
}

var (
	roomCampsMu     sync.RWMutex
	roomCampsReader func(roomID int) []RoomCamp
)

// SetRoomCampsReader registers modules/camping's lock-free snapshot of the
// camps in each room. Passing nil clears it.
func SetRoomCampsReader(f func(roomID int) []RoomCamp) {
	roomCampsMu.Lock()
	defer roomCampsMu.Unlock()
	roomCampsReader = f
}

// RoomCamps returns the camps pitched in roomID, or none without a reader.
// It never waits on the camping module's own lock.
func RoomCamps(roomID int) []RoomCamp {
	roomCampsMu.RLock()
	f := roomCampsReader
	roomCampsMu.RUnlock()
	if f == nil {
		return nil
	}
	return f(roomID)
}

// CampLines is what look shows for the camps in a room: one line each.
// The viewer's own camp, or one whose leader has no name to give, reads
// "A camp"; anyone else's names its leader.
func CampLines(camps []RoomCamp, viewerUserID int, nameOf func(userID int) string) []string {
	lines := make([]string, 0, len(camps))
	for _, camp := range camps {
		whose := "A camp"
		if camp.LeaderUserID != viewerUserID && nameOf != nil {
			if name := nameOf(camp.LeaderUserID); name != "" {
				whose = name + "'s camp"
			}
		}
		if camp.FireLit {
			lines = append(lines, whose+" is pitched here: bedrolls around a crackling campfire.")
		} else {
			lines = append(lines, whose+" is pitched here, around a cold fire pit.")
		}
	}
	return lines
}

// UseRoomCampsReaderForTest registers f and returns a func that restores
// the reader it replaced.
func UseRoomCampsReaderForTest(f func(roomID int) []RoomCamp) (restore func()) {
	roomCampsMu.Lock()
	previous := roomCampsReader
	roomCampsReader = f
	roomCampsMu.Unlock()
	return func() { SetRoomCampsReader(previous) }
}
