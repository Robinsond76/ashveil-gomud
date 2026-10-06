package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
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

// named is a tagged name as narration prints it: a mob's with its article,
// a player's (userTag) as typed, whatever its case.
func named(tagged string) string {
	if strings.HasPrefix(tagged, `<ansi fg="username">`) {
		return tagged
	}
	return util.Article(tagged)
}

// turnsToward is the line for a fighter taking a new target: "The bandit
// cutthroat turns toward Garrick Vane." who and target are tagged names;
// who may be "You".
func turnsToward(who, target string) string {
	if who == `You` {
		return fmt.Sprintf(`You turn toward %s.`, named(target))
	}
	return util.CapitalizeFirst(fmt.Sprintf(`%s turns toward %s.`, named(who), named(target)))
}

// healerMarked is the line a company on its healers default says when it
// turns on an enemy healer (Phase 35e): the player sees why the company went
// for that foe, not the usual "you turn toward".
func healerMarked(target string) string {
	return fmt.Sprintf(`Your company marks %s as a healer and goes for it first.`, named(target))
}

// Opener and closing pools, keyed by a mob group ("" is the generic pool).
// A fight's lines come from its enemies' first group with a pool.
// "bandits" is for the road bandits of the play-test roadmap (32c) and the
// company tests; "slum-ruffians" and "practice-squad" are shipped groups.
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
		"slum-ruffians": {
			`They fan out across the alley, and close in.`,
			`Someone in the slum whistles, and they all come on at once.`,
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
		"slum-ruffians": {
			`The last of them goes down, and the alley is quiet.`,
			`The last of them falls. No one in the slums comes to help.`,
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

// shieldBreaksOwnerLine is what a fighter is told when their shield breaks.
func shieldBreaksOwnerLine(item string) string {
	return fmt.Sprintf(`<ansi fg="214">Your <ansi fg="item">%s</ansi> cracks apart and falls away.</ansi>`, item)
}

// shieldBreaksRoomLine is what the room sees; owner is a tagged name.
func shieldBreaksRoomLine(item, owner string) string {
	return fmt.Sprintf(`<ansi fg="214">%s</ansi>`, util.CapitalizeFirst(fmt.Sprintf(`%s's <ansi fg="item">%s</ansi> cracks apart and falls away.`, named(owner), item)))
}

// groupInOtherBattle reports whether a player other than userId is in a
// battle against partyID in roomId: the room already has, or still has, a
// fight against that group, so it hears no second opener or closing.
func groupInOtherBattle(userId, roomId int, partyID string) bool {
	for _, uid := range battle.Players() {
		if uid == userId {
			continue
		}
		if b, ok := battle.Current(uid); ok && b.RoomId == roomId && b.PartyID == partyID {
			return true
		}
	}
	return false
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

// deathNoticedKey marks a mob whose death line was printed, so a later
// round that reports it again, before its queued suicide runs, is silent.
const deathNoticedKey = `29c.deathNoticed`

// mobDeathNotice prints a mob's death (or, a practice foe, beaten) line
// to its room at once, so it lands in the round's order, before the
// fight's closing line; its queued "suicide quiet" then skips it.
func mobDeathNotice(mob *mobs.Mob) {
	if mob.GetTempData(deathNoticedKey) != nil {
		return
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return
	}
	mob.SetTempData(deathNoticedKey, true)
	name := mobTag(mobName(mob.InstanceId))
	line := combat.DeathLine(name)
	if mob.Practice {
		line = combat.BeatenLine(name)
	}
	room.SendText(line)
}
