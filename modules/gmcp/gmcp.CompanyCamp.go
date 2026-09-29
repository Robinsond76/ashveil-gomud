package gmcp

// Phase 32g: "Company.Camp", the web client's Camp tab: the leader's camp
// seen from the room they stand in, whether a camp can be made here, and
// whether an inn is here. Sent by the company feed only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type campPayload struct {
	HasCamp     bool   `json:"has_camp"`
	Here        bool   `json:"here"`
	Room        string `json:"room"`
	FireLit     bool   `json:"fire_lit"`
	Resting     bool   `json:"resting"`
	Rested      bool   `json:"rested"`
	RestPercent int    `json:"rest_percent"`
	RestSeconds int    `json:"rest_seconds"`
	CanCamp     bool   `json:"can_camp"`
	Inn         bool   `json:"inn"`
}

func campPayloadOf(s camping.CampState) campPayload {
	return campPayload{HasCamp: s.HasCamp, Here: s.Here, Room: s.RoomTitle, FireLit: s.FireLit, Resting: s.Resting, Rested: s.Rested,
		RestPercent: s.RestPercent, RestSeconds: s.RestSeconds, CanCamp: s.CanCamp, Inn: s.Inn}
}

// campExtra is the feed's Company.Camp message; nothing is sent while no
// camping provider can report.
func campExtra(state func(leaderUserID, roomID int, tags []string) (camping.CampState, bool)) companyExtra {
	return companyExtra{module: "Company.Camp", build: func(user *users.UserRecord) []byte {
		var tags []string
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			tags = room.Tags
		}
		s, ok := state(user.UserId, user.Character.RoomId, tags)
		if !ok {
			return nil
		}
		data, err := json.Marshal(campPayloadOf(s))
		if err != nil {
			return nil
		}
		return data
	}}
}
