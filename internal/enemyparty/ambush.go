package enemyparty

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// SpawnAmbush spawns a hostile encounter of a mob template in roomID that
// attacks the leader: a named pair of its kind, or one if it is solitary
// (Phase 29b2: no lone enemies). It returns the first foe's instance ID.
// Travel ambushes (Phase 12c) and camp raids (Phase 33f3) both use it. It
// must run on the game loop.
var ambushRoll = util.Rand

// UseAmbushRollForTest supplies seeded encounter detection without changing combat dice.
func UseAmbushRollForTest(roll func(int) int) func() {
	prev := ambushRoll
	ambushRoll = roll
	return func() { ambushRoll = prev }
}

func SpawnAmbush(roomID, mobTemplateID, leaderUserID int) (int, error) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("enemyparty: room %d is unavailable", roomID)
	}
	mob := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
	if mob == nil {
		return 0, fmt.Errorf("enemyparty: encounter mob template %d is unavailable", mobTemplateID)
	}
	foes := []*mobs.Mob{mob}
	if !mob.Solitary {
		if second := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID); second != nil {
			foes = append(foes, second)
		}
	}
	group, groupName := "", ""
	if len(foes) > 1 {
		group = EncounterGroup(roomID, mob.InstanceId)
		// Phase 32c: the pair is named as it forms ("a band of ruffians")
		// and keeps the name while it stands.
		summaries := make([]mobparty.MobSummary, len(foes))
		for i, foe := range foes {
			summaries[i] = rooms.GroupSummary(foe)
		}
		groupName = mobparty.Generate(summaries).Name
	}
	engage(roomID, room, foes, leaderUserID, group, groupName, true, 0)
	return mob.InstanceId, nil
}

// engage rolls the company's detection of the foes, then sets the foes upon
// the leader: hostile, unmoving, in their spawn group. Camp raids, travel
// ambushes and random room encounters (Phase 37) all end here.
// A room encounter (surprise false) is a sudden appearance, not an ambush:
// neither side loses its opening round. delay holds the foes' attack back that
// many seconds, so a company walking in behind its leader arrives first.
func engage(roomID int, room *rooms.Room, foes []*mobs.Mob, leaderUserID int, group, groupName string, surprise bool, delay float64) {
	observer, perception, visibility := "you", 0, room.GetVisibility()
	// Phase 39f: a Skyscout watches from the air, in the open only.
	watch := func(c *characters.Character) int {
		n := c.Stats.Perception.ValueAdj
		if !room.IsIndoor() {
			n += c.ClassEffects().Int(classes.SkyEye)
		}
		return n
	}
	if u := users.GetByUserId(leaderUserID); u != nil && u.Character != nil {
		found := u.Character.Health > 0 && u.Character.RoomId == roomID && !u.Character.CombatWithdrawn
		if found {
			perception, visibility = watch(u.Character), room.VisibilityForUser(u)
		}
		// Roster order supplies stable ties, including deliberately unplaced observers.
		if members, ok := company.CompanyMembers(leaderUserID); ok {
			for _, member := range members {
				id, ok := company.InstanceFor(leaderUserID, member.ID)
				if !ok {
					continue
				}
				m := mobs.GetInstance(id)
				if m == nil || m.Character.Health < 1 || m.Character.RoomId != roomID || m.Character.CombatWithdrawn {
					continue
				}
				if sees := watch(&m.Character); !found || sees > perception {
					found = true
					observer, perception, visibility = m.Character.Name, sees, room.VisibilityForMob(m)
				}
			}
		}

	}
	stealth := 0
	for _, foe := range foes {
		n := foe.Character.Stats.Perception.ValueAdj
		if foe.Stealth != nil {
			n = *foe.Stealth
		}
		stealth = max(stealth, n)
	}
	cover := 0
	if room.HasTag("ambush-cover") {
		cover = 10
	}
	advantage := 0
	if surprise {
		_, advantage = formationcombat.Detection(perception, stealth, visibility, cover, ambushRoll(100), false)
		// Phase 38c2: a Pathfinder halves the ambushes on the company, and its
		// Ambush Master turns one around.
		if advantage < 0 {
			if _, scout := CompanyEffect(leaderUserID, classes.AmbushFlip); scout != "" {
				advantage, observer = 1, scout
			} else if _, scout := CompanyEffect(leaderUserID, classes.PathEye); scout != "" && ambushRoll(100) < 50 {
				advantage, observer = 0, scout
			}
		}
	}
	for _, foe := range foes {
		foe.AmbushOwner, foe.AmbushAdvantage, foe.AmbushObserver = leaderUserID, advantage, observer
		foe.Hostile = true
		foe.MaxWander = 0
		foe.SpawnGroup = group
		foe.GroupName = groupName
		room.AddMob(foe.InstanceId)
		if delay > 0 {
			foe.Command(fmt.Sprintf("attack @%d", leaderUserID), delay)
		} else {
			foe.Command(fmt.Sprintf("attack @%d", leaderUserID))
		}
	}
}

// EncounterGroup names an encounter's spawn group after its first foe.
func EncounterGroup(roomID, firstInstanceID int) string {
	return fmt.Sprintf("encounter:%d:%d", roomID, firstInstanceID)
}

// WatchAmbush keeps an already successful Camp Watch check: it cannot
// turn a detected raid into enemy surprise. Called on the game loop.
func WatchAmbush(roomID, first, owner int) {
	r := rooms.LoadRoom(roomID)
	if r == nil {
		return
	}
	p, ok := PartyOf(r, first)
	if !ok {
		return
	}
	for _, id := range p.Members {
		if m := mobs.GetInstance(id); m != nil && m.AmbushOwner == owner && m.AmbushAdvantage < 0 {
			m.AmbushAdvantage = 0
		}
	}
}

// CompanyEffect is the best value of a class effect among a leader's
// standing company in the leader's room: the leader and each living
// companion that is with them (Phase 38c2: an elite's company-wide gifts).
// name is the holder that gives it, "" when none does.
func CompanyEffect(leaderUserID int, key string) (value int, name string) {
	u := users.GetByUserId(leaderUserID)
	if u == nil || u.Character == nil {
		return 0, ""
	}
	consider := func(c *characters.Character) {
		if c.Health < 1 || c.CombatWithdrawn || c.RoomId != u.Character.RoomId {
			return
		}
		if v := c.ClassEffects().Int(key); v > value {
			value, name = v, c.Name
		}
	}
	consider(u.Character)
	members, _ := company.CompanyMembers(leaderUserID)
	for _, member := range members {
		id, ok := company.InstanceFor(leaderUserID, member.ID)
		if !ok {
			continue
		}
		if m := mobs.GetInstance(id); m != nil {
			consider(&m.Character)
		}
	}
	return value, name
}
