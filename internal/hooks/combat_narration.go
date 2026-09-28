package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 29c: the narration voice's engagement, opener, closing, and death
// lines.

func mobTag(name string) string  { return fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, name) }
func userTag(name string) string { return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, name) }

// turnsToward is the line for a fighter taking a new target: "The bandit
// cutthroat turns toward Garrick Vane." who and target are tagged names;
// who may be "You".
func turnsToward(who, target string) string {
	if who == `You` {
		return fmt.Sprintf(`You turn toward %s.`, util.Article(target))
	}
	return util.CapitalizeFirst(fmt.Sprintf(`%s turns toward %s.`, util.Article(who), util.Article(target)))
}

// Opener and closing pools, keyed by a mob group ("" is the generic pool).
// A fight's lines come from its enemies' first group with a pool.
var (
	fightOpeners = map[string][]string{
		"": {
			`Weapons come up, and the fight is on.`,
			`The air goes tight. Someone moves first, and then everyone does.`,
			`No more words. The fight begins.`,
		},
		"bandits": {
			`The bandits spread out, blades low, and close in.`,
			`A sharp whistle, and the bandits come on.`,
			`The bandits trade a look, and draw.`,
		},
		"practice-squad": {
			`A drill horn sounds, and the straw squad creaks into line.`,
			`The straw soldiers lurch forward on their frames. The drill begins.`,
		},
	}
	fightClosings = map[string][]string{
		"": {
			`The last of them falls. The fight is over.`,
			`Nothing else moves. It is done.`,
			`Silence, then only your own breathing. The fight is over.`,
		},
		"bandits": {
			`The last bandit goes down, and no one is left to run.`,
			`The last bandit falls, and it is quiet again.`,
		},
		"practice-squad": {
			`The last straw soldier topples. The drill is done.`,
		},
	}
)

// narrationPool picks pools' entry for the first of groups that has one,
// else the generic pool.
func narrationPool(pools map[string][]string, groups []string) []string {
	for _, g := range groups {
		if pool, ok := pools[g]; ok && g != "" {
			return pool
		}
	}
	return pools[""]
}

// fightOpener is a fight's first line, for enemies of groups.
func fightOpener(groups []string) string {
	pool := narrationPool(fightOpeners, groups)
	return pool[util.Rand(len(pool))]
}

// fightClosing is a won fight's last line, for enemies of groups.
func fightClosing(groups []string) string {
	pool := narrationPool(fightClosings, groups)
	return pool[util.Rand(len(pool))]
}

// sendFightClosing sends a won fight's closing line to its room.
func sendFightClosing(roomId int, enemies []combatstream.Ref) {
	if room := rooms.LoadRoom(roomId); room != nil {
		room.SendText(fightClosing(enemyGroups(enemies)))
	}
}

// enemyGroups are the mob groups of a fight's first enemy, from its
// template (the instance may be gone by the fight's end).
func enemyGroups(enemies []combatstream.Ref) []string {
	for _, r := range enemies {
		if m := mobs.GetInstance(r.MobInstanceId); m != nil {
			return m.Groups
		}
		if spec := mobs.GetMobSpec(mobs.MobId(r.MobId)); spec != nil {
			return spec.Groups
		}
	}
	return nil
}

// mobDeathNotice prints a mob's death (or, a practice foe, beaten) line
// to its room at once, so it lands in the round's order, before the
// fight's closing line; its queued "suicide quiet" then skips it.
func mobDeathNotice(mob *mobs.Mob) {
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return
	}
	name := mobTag(mob.Character.Name)
	if mob.Practice {
		room.SendText(combat.BeatenLine(name))
		return
	}
	room.SendText(combat.DeathLine(name))
}
