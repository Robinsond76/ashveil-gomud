package combat

import (
	"hash/fnv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var humanoidPainReactions = []races.PainReaction{{
	ToVictim: "Pain flashes through you. You stagger but stay on your feet.",
	ToRoom:   "{name} staggers and draws a ragged breath, but stays on {his} feet.",
}}

var creaturePainReactions = []races.PainReaction{{
	ToVictim: "Pain jolts through you. You recoil but stay on your feet.",
	ToRoom:   "{name} recoils from the blow, then braces to fight on.",
}}

// painReactionFor selects authored NPC text, then race text, then a safe
// fallback. Its deterministic variant choice does not consume combat RNG.
func painReactionFor(target *characters.Character, targetType SourceTarget, targetMob *mobs.Mob, remainingHealth, strikeOrdinal int) (victim, room string) {
	pool := humanoidPainReactions
	if targetType == Mob {
		if race := races.GetRace(target.GetRaceId()); race != nil {
			if race.DefaultPronouns == "it" {
				pool = creaturePainReactions
			}
			if len(race.PainReactions) > 0 {
				pool = race.PainReactions
			}
		}
		if targetMob != nil && len(targetMob.PainReactions) > 0 {
			pool = targetMob.PainReactions
		}
	}

	nameHash := fnv.New32a()
	_, _ = nameHash.Write([]byte(strings.ToLower(target.Name)))
	pair := pool[(uint32(remainingHealth+strikeOrdinal)+nameHash.Sum32())%uint32(len(pool))]
	name := target.Name
	if targetType == Mob {
		name = util.Article(name)
	}
	pronouns := combatPronouns(target, targetType)
	replacer := strings.NewReplacer("{name}", name, "{he}", pronouns.Subject, "{him}", pronouns.Object, "{his}", pronouns.Possessive)
	return util.CapitalizeFirst(replacer.Replace(pair.ToVictim)), util.CapitalizeFirst(replacer.Replace(pair.ToRoom))
}
