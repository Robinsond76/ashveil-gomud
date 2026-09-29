package mobs

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/races"
)

func TestMobRejectsIncompletePainReactionOverride(t *testing.T) {
	mob := &Mob{Character: *characters.New(), PainReactions: []races.PainReaction{{ToRoom: "{name} reels."}}}
	if err := mob.Validate(); err == nil {
		t.Fatal("accepted an NPC override with no victim line")
	}
}

func TestSpiderQueenLoadsHerOwnPainReaction(t *testing.T) {
	loadShippedPronounData(t)
	queen := GetMobSpec(37)
	if queen == nil || len(queen.PainReactions) == 0 || !strings.Contains(queen.PainReactions[0].ToRoom, "crown") {
		t.Fatalf("spider queen's authored override did not load: %+v", queen)
	}
}
