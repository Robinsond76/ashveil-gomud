package combatstream

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExplainAMiss(t *testing.T) {
	got := Strike{Chance: 38, Base: 45, Modifier: -7, Roll: 61}.Explain()
	assert.Equal(t, []string{"To hit: 38 in 100, rolled 62 (it missed). Skill and speed gave 45, -7 from darkness or a second weapon."}, got)
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
		Quality: "telling", Crit: true, Rolled: 6, Raw: 12, Armor: 4, ArmorTook: 3, Reduced: 3, Damage: 9,
		Notes: []string{"Sharpened edge added 2"},
	}.Explain(), "\n")
	for _, want := range []string{
		"To hit: 70 in 100, rolled 21 (it landed). Skill and speed gave 65, +5 from company chemistry.",
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

// Review: armor's own share is shown apart from a ward's, so "armor took"
// never counts what a ward or aura took (they have their own lines).
func TestExplainArmorApartFromAWard(t *testing.T) {
	got := strings.Join(Strike{Chance: 60, Base: 60, Roll: 1, Hit: true, Raw: 20, Armor: 10, ArmorTook: 2, Reduced: 8, Damage: 12,
		Notes: []string{"A ward took 6"}}.Explain(), "\n")
	assert.Contains(t, got, "Damage: 20 before armor, armor 10 took 2, 12 got through.")
	assert.Contains(t, got, "A ward took 6.")
}

// Review: a chance the limits held says so, so the parts add up.
func TestExplainAClampedChance(t *testing.T) {
	got := Strike{Chance: 95, Base: 98, Bonus: 5, Roll: 3, Hit: true}.Explain()[0]
	assert.Equal(t, "To hit: 95 in 100, rolled 4 (it landed). Skill and speed gave 98, +5 from company chemistry, held to 95 by the limits on any chance to hit.", got)
}

// Review: a Marksman's critical shot through a shield is not "not blocked".
func TestExplainACritThroughAShield(t *testing.T) {
	got := Strike{Chance: 60, Base: 60, Roll: 1, Hit: true, DefenseChance: 40, ThroughShield: true, Crit: true, Raw: 9, Damage: 9}.Explain()
	assert.Contains(t, got, "Defence: 40 in 100 to block it, and it was blocked, but a critical shot goes through a shield.")
}

// Review: a pet's bite says what it did; a Perfect Parry needed no roll.
func TestExplainAPetAndAPerfectParry(t *testing.T) {
	assert.Equal(t, []string{"The wolf joined in: 6 before armor, armor 20 took 1, 5 got through."},
		Strike{Pet: "wolf", Hit: true, Roll: -1, Raw: 6, Armor: 20, ArmorTook: 1, Reduced: 1, Damage: 5}.Explain())
	got := Strike{Chance: 60, Base: 60, Roll: 1, Hit: true, Defense: "parried"}.Explain()
	assert.Equal(t, "Then it was parried outright by a Perfect Parry, no roll needed; no damage.", got[1])
}

// Review: a round's heading calls it critical only when a critical strike
// got damage through, and a pet's bite is not the member's own swing.
func TestRoundHeadingCritAndPetSwings(t *testing.T) {
	r := Roll{Round: 2, Source: Ref{Name: "Aria"}, Target: Ref{Name: "bandit"}, Damage: 4, Crit: true,
		Strikes: []Strike{{Hit: true, Crit: true, Raw: 5, Reduced: 5}, {Hit: true, Raw: 4, Damage: 4}}}
	assert.Equal(t, "Round 2: Aria -> bandit, hit for 4.", r.Describe(0)[0])
	r.Strikes[0].Damage = 1
	assert.Equal(t, "Round 2: Aria -> bandit, landed a critical hit for 4.", r.Describe(0)[0])

	tl := newTally()
	tl.noteSwings(Event{Source: Ref{Name: "Aria", UserId: 3}, Strikes: []Strike{{Chance: 10, Roll: 50}, {Pet: "wolf", Hit: true, Damage: 3}}})
	sw := tl.swings[Ref{Name: "Aria", UserId: 3}.Key()]
	assert.Equal(t, 1, sw.Thrown)
	assert.Equal(t, 1, sw.Missed)
}
