package gmcp

// Phase 32g: "Company.Camp", the web client's Camp tab: the leader's camp
// seen from the room they stand in, whether a camp can be made here, and
// whether an inn is here. Sent by the company feed only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/parties"
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
	Embers      bool   `json:"embers"`
	Tent        bool   `json:"tent"`
	RestPercent int    `json:"rest_percent"`
	RestSeconds int    `json:"rest_seconds"`
	CanCamp     bool   `json:"can_camp"`
	Inn         bool   `json:"inn"`
	// Phase 40b: the camp's room ID, and the camps of the other leaders in
	// the viewer's party. Never a camp of a company outside the party.
	RoomID      int          `json:"room_id"`
	AlliedCamps []alliedCamp `json:"allied_camps"`
}

// alliedCamp is a party member's camp, drawn on the map as an ally tent.
type alliedCamp struct {
	RoomID  int    `json:"room_id"`
	Leader  string `json:"leader"`
	FireLit bool   `json:"fire_lit"`
	Resting bool   `json:"resting"`
}

func campPayloadOf(s camping.CampState) campPayload {
	return campPayload{AlliedCamps: []alliedCamp{}, RoomID: s.RoomID, HasCamp: s.HasCamp, Here: s.Here, Room: s.RoomTitle, FireLit: s.FireLit, Resting: s.Resting, Rested: s.Rested,
		Embers: s.Embers, Tent: s.Tent, RestPercent: s.RestPercent, RestSeconds: s.RestSeconds, CanCamp: s.CanCamp, Inn: s.Inn}
}

// campExtra is the feed's Company.Camp message; nothing is sent while no
// camping provider can report.
func campExtra(state func(leaderUserID, roomID int, tags []string) (camping.CampState, bool), allies func(user *users.UserRecord) []alliedCamp) companyExtra {
	return companyExtra{module: "Company.Camp", build: func(user *users.UserRecord) []byte {
		var tags []string
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			tags = room.Tags
		}
		s, ok := state(user.UserId, user.Character.RoomId, tags)
		if !ok {
			return nil
		}
		p := campPayloadOf(s)
		if allies != nil {
			if a := allies(user); a != nil {
				p.AlliedCamps = a
			}
		}
		data, err := json.Marshal(p)
		if err != nil {
			return nil
		}
		return data
	}}
}

// partyCamps lists the camps of the other members of the user's party
// (Phase 40b), by their own leader state. Only the party's members are ever
// looked up, so a camp of a company outside the party, even one in the same
// room, is never sent.
func partyCamps(user *users.UserRecord) []alliedCamp {
	return alliedCampsOf(user, camping.CampStateOf)
}

func alliedCampsOf(user *users.UserRecord, state func(leaderUserID, roomID int, tags []string) (camping.CampState, bool)) []alliedCamp {
	party := parties.Get(user.UserId)
	if party == nil {
		return nil
	}
	var out []alliedCamp
	for _, uid := range party.GetMembers() {
		if uid == user.UserId {
			continue
		}
		s, ok := state(uid, 0, nil)
		if !ok || !s.HasCamp {
			continue
		}
		out = append(out, alliedCamp{RoomID: s.RoomID, Leader: users.CharacterName(uid), FireLit: s.FireLit, Resting: s.Resting})
	}
	return out
}
