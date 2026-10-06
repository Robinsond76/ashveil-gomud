package camping

import "sync"

// RoomCamp is one camp pitched in a room, as look shows it (Phase 32a).
type RoomCamp struct {
	LeaderUserID int
	FireLit      bool
	Damp         bool     // lit with damp wood: no warmth (Phase 40a2)
	Embers       bool     // burned down after a rest (Phase 40a3)
	Tent         bool     // a tent is pitched (Phase 40a3)
	TentKind     TentKind // which one (Phase 52); empty is canvas
	Resting      bool     // a rest is under way
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
		pitch := "bedrolls"
		if camp.Tent {
			pitch = TentOf(camp.TentKind).WithArticle() + " and bedrolls"
		}
		switch {
		case camp.FireLit && camp.Damp:
			lines = append(lines, whose+" is pitched here: "+pitch+" around a smoky, sullen fire of damp wood.")
		case camp.FireLit:
			lines = append(lines, whose+" is pitched here: "+pitch+" around a crackling campfire.")
		case camp.Embers:
			lines = append(lines, whose+" is pitched here: "+pitch+" around the glowing embers of a banked fire.")
		case camp.Tent:
			lines = append(lines, whose+" is pitched here: an oiled canvas tent beside a cold fire pit.")
		default:
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
