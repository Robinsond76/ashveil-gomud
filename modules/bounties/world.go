package bounties

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/bounty"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BoardTag is the room tag that makes a room a bounty board.
const BoardTag = "bounty-board"

// board is the board room a leader stands at.
type board struct {
	Key   string // unique per board room: the seed of its list
	Title string
	Band  bounty.Band
}

// world is what the module reads outside its own state, so tests run
// without the game.
type world interface {
	// Board is the board room the user stands in.
	Board(userID int) (board, bool)
	// Candidates is every lair boss and named group the zones hold.
	Candidates() []bounty.Target
	// Level is the user's level (0 when unknown), for the rating words.
	Level(userID int) int
	// Place is the title of the room the user stands in.
	Place(userID int) string
	// Busy reports whether the user is in a battle.
	Busy(userID int) bool
	// Pay adds gold to the user.
	Pay(userID, gold int) bool
	Online(userID int) bool
	Push(userID int, namespace string, payload any)
}

type liveWorld struct{}

func (liveWorld) Board(userID int) (board, bool) {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return board{}, false
	}
	room := rooms.LoadRoom(u.Character.RoomId)
	if room == nil || !room.HasTag(BoardTag) {
		return board{}, false
	}
	b := board{Key: room.Zone + ":" + strconv.Itoa(room.RoomId), Title: room.Title}
	if cfg := rooms.GetZoneConfig(room.Zone); cfg != nil {
		b.Band = bounty.Band{Low: cfg.Encounters.Band.Low, High: cfg.Encounters.Band.High}
	}
	return b, true
}

func lookup(mobID int) (encounters.Template, bool) {
	spec := mobs.GetMobSpec(mobs.MobId(mobID))
	if spec == nil {
		return encounters.Template{}, false
	}
	return encounters.Template{Solitary: spec.Solitary, Healer: spec.Role == "healer"}, true
}

// Candidates reads each zone's validated encounter tables: a boss
// composition is a lair (its first member is the master), an ordinary one
// is a named group.
func (liveWorld) Candidates() []bounty.Target {
	var out []bounty.Target
	names := rooms.GetAllZoneNames()
	sort.Strings(names)
	for _, zone := range names {
		cfg := rooms.GetZoneConfig(zone)
		if cfg == nil || len(cfg.Encounters.Tables) == 0 {
			continue
		}
		valid, _ := cfg.Encounters.Validate(lookup)
		if !valid.Band.Valid() {
			continue
		}
		for _, table := range valid.Tables {
			for _, c := range table {
				t := bounty.Target{Zone: zone, Low: valid.Band.Low, High: valid.Band.High}
				if c.Boss {
					spec := mobs.GetMobSpec(mobs.MobId(c.Members[0].MobID))
					if spec == nil {
						continue
					}
					t.Kind, t.Ref, t.Name = bounty.Boss, "mob:"+strconv.Itoa(c.Members[0].MobID), spec.Character.Name
				} else {
					t.Kind, t.Ref, t.Name = bounty.Group, "group:"+c.ID, c.Title()
				}
				out = append(out, t)
			}
		}
	}
	return out
}

func (liveWorld) Level(userID int) int {
	if u := users.GetByUserId(userID); u != nil && u.Character != nil {
		return u.Character.Level
	}
	return 0
}

func (liveWorld) Place(userID int) string {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return ""
	}
	if r := rooms.LoadRoom(u.Character.RoomId); r != nil {
		return r.Title
	}
	return ""
}

func (liveWorld) Busy(userID int) bool {
	u := users.GetByUserId(userID)
	return u != nil && actionpolicy.InBattle(u)
}

func (liveWorld) Pay(userID, gold int) bool {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return false
	}
	u.Character.Gold += gold
	events.AddToQueue(events.EquipmentChange{UserId: userID, GoldChange: gold})
	return true
}

func (liveWorld) Online(userID int) bool { return users.GetByUserId(userID) != nil }

func (liveWorld) Push(userID int, namespace string, payload any) {
	f, ok := usercommands.GetExportedFunction("SendGMCPEvent")
	if !ok {
		return
	}
	if send, ok := f.(func(int, string, any)); ok {
		if _, err := json.Marshal(payload); err != nil {
			mudlog.Error("bounties: payload", "error", err)
			return
		}
		send(userID, namespace, payload)
	}
}
