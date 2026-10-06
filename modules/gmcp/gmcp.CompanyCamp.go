package gmcp

// Phase 32g: "Company.Camp", the web client's Camp tab: the leader's camp
// seen from the room they stand in, whether a camp can be made here, and
// whether an inn is here. Sent by the company feed only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type campPayload struct {
	HasCamp bool   `json:"has_camp"`
	Here    bool   `json:"here"`
	Room    string `json:"room"`
	FireLit bool   `json:"fire_lit"`
	Resting bool   `json:"resting"`
	Rested  bool   `json:"rested"`
	Embers  bool   `json:"embers"`
	Tent    bool   `json:"tent"`
	// Phase 52: which tent is pitched (its name and effect), and the tents
	// carried, for the picker.
	TentKind string    `json:"tent_kind,omitempty"`
	TentName string    `json:"tent_name,omitempty"`
	TentNote string    `json:"tent_note,omitempty"`
	Tents    []tentRow `json:"tents"`
	// Phase 40a4: the camp gear the company carries, one label each.
	Gear []string `json:"gear"`
	// Phase 43a: camp supplies carried, and what is queued for the next rest.
	Supplies    []string `json:"supplies"`
	Prepared    []string `json:"prepared"`
	TheftRisk   bool     `json:"theft_risk"`
	RestPercent int      `json:"rest_percent"`
	RestSeconds int      `json:"rest_seconds"`
	CanCamp     bool     `json:"can_camp"`
	Inn         bool     `json:"inn"`
	// Phase 40b: the camp's room ID, and the camps of the other leaders in
	// the viewer's party. Never a camp of a company outside the party.
	RoomID      int          `json:"room_id"`
	AlliedCamps []alliedCamp `json:"allied_camps"`
	// Phase 49: the company's latest exchange (camp talk or after a
	// battle), shown on the Camp tab; absent before the first.
	Banter []banterLine `json:"banter,omitempty"`
	// Phase 51: the rest duty of each member at the camp, and whether a
	// running rest has fixed them.
	Duties       []dutyRow `json:"duties"`
	DutiesLocked bool      `json:"duties_locked"`
	// Phase 56: the dishes the leader has learned, one line each.
	Recipes []string `json:"recipes"`
}

// dutyRow is one member's rest duty for the Camp tab's picker. Command is
// the word "camp duties" takes for the member.
type dutyRow struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Duty    string   `json:"duty"`
	Options []string `json:"options"`
}

// tentRow is one carried tent for the Camp tab's picker. Command is what
// pitches it.
type tentRow struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Effect  string `json:"effect"`
	Pitched bool   `json:"pitched"`
	Command string `json:"command"`
}

type banterLine struct {
	Name string `json:"name"`
	Verb string `json:"verb"`
	Text string `json:"text"`
}

// alliedCamp is a party member's camp, drawn on the map as an ally tent.
type alliedCamp struct {
	RoomID  int    `json:"room_id"`
	Leader  string `json:"leader"`
	FireLit bool   `json:"fire_lit"`
	Resting bool   `json:"resting"`
	Embers  bool   `json:"embers"` // Phase 40c: banked embers, drawn when the fire is not lit
	Tent    bool   `json:"tent"`   // Phase 40c: pitched with a tent; a rough camp without one is drawn otherwise
}

func campPayloadOf(s camping.CampState) campPayload {
	gear := s.Gear
	if gear == nil {
		gear = []string{}
	}
	supplies, prepared := s.Supplies, s.Prepared
	if supplies == nil {
		supplies = []string{}
	}
	if prepared == nil {
		prepared = []string{}
	}
	duties := make([]dutyRow, 0, len(s.Duties))
	for _, d := range s.Duties {
		duties = append(duties, dutyRow{Key: d.Key, Name: d.Name, Command: d.Command, Duty: d.Duty, Options: d.Options})
	}
	recipes := s.Recipes
	if recipes == nil {
		recipes = []string{}
	}
	tents := make([]tentRow, 0, len(s.Tents))
	for _, t := range s.Tents {
		tents = append(tents, tentRow{Kind: string(t.Kind), Name: t.Name, Effect: t.Effect, Pitched: t.Pitched, Command: "camp tent " + camping.TentOf(t.Kind).Short})
	}
	var tentKind, tentName string
	if s.Tent {
		spec := camping.TentOf(s.TentKind)
		tentKind, tentName = string(spec.Kind), spec.Name
	}
	return campPayload{Recipes: recipes, Tents: tents, TentKind: tentKind, TentName: tentName, TentNote: s.TentNote, Duties: duties, DutiesLocked: s.DutiesLocked, Gear: gear, Supplies: supplies, Prepared: prepared, TheftRisk: s.TheftRisk, AlliedCamps: []alliedCamp{}, RoomID: s.RoomID, HasCamp: s.HasCamp, Here: s.Here, Room: s.RoomTitle, FireLit: s.FireLit, Resting: s.Resting, Rested: s.Rested,
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
		for _, l := range company.LastBanter(user.UserId) {
			p.Banter = append(p.Banter, banterLine{Name: l.Name, Verb: l.Verb, Text: l.Text})
		}
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
		out = append(out, alliedCamp{RoomID: s.RoomID, Leader: users.CharacterName(uid), FireLit: s.FireLit, Resting: s.Resting, Embers: s.Embers, Tent: s.Tent})
	}
	return out
}
