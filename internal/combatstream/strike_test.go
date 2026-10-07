package combatstream

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExplainAMiss(t *testing.T) {
	got := Strike{Chance: 38, Base: 45, Modifier: -7, Roll: 61}.Explain()
	assert.Equal(t, []string{"To hit: 38 in 100, rolled 62 (it missed). skill and speed gave 45, -7 from darkness or a second weapon."}, got)
}

func TestExplainADefendedStrike(t *testing.T) {
	got := Strike{Chance: 60, Base: 60, Roll: 10, Hit: true, Defense: "dodged", DefenseChance: 31}.Explain()
	assert.Equal(t, "To hit: 60 in 100, rolled 11 (it landed).", got[0])
	assert.Equal(t, "Then it was dodged, a 31 in 100 chance; no damage.", got[1])
	assert.Len(t, got, 2)
}

func TestExplainALandedBlow(t *testing.T) {
	got := strings.Join(Strike{
		Chance: 70, Base: 65, Bonus: 5, Roll: 20, Hit: true, DefenseChance: 25,
		Quality: "telling", Crit: true, Rolled: 6, Raw: 12, Armor: 4, Reduced: 3, Damage: 9,
		Notes: []string{"Sharpened edge added 2"},
	}.Explain(), "\n")
	for _, want := range []string{
		"To hit: 70 in 100, rolled 21 (it landed). skill and speed gave 65, +5 from company chemistry.",
		"Defence: 25 in 100 to turn it aside, and it was not.",
		"A telling blow.",
		"A critical hit.",
		"Damage: 12 before armor, armor 4 took 3, 9 got through.",
		"Sharpened edge added 2.",
	} {
		assert.Contains(t, got, want)
	}
}

func TestExplainAutomaticStrikes(t *testing.T) {
	assert.Equal(t, []string{"No roll needed (perfect shot)."}, Strike{Auto: "perfect shot", Roll: -1}.Explain()[:1])
	assert.Contains(t, Strike{Auto: "harmless", Roll: -1}.Explain(), "It has nothing to hurt with.")
}

func TestHeadline(t *testing.T) {
	assert.Equal(t, "", Headline(nil))
	assert.Equal(t, "missed, 38 in 100; dodged, 31 in 100; 9 damage after armor 4",
		Headline([]Strike{{Chance: 38}, {Hit: true, Defense: "dodged", DefenseChance: 31}, {Hit: true, Damage: 9, Armor: 4}}))
}
