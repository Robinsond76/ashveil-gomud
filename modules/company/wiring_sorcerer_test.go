package company

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38d wiring: the Sorcerer and High Sorcerer through the real strategy
// pass, the shipped Arcane Lance script and the combat round (the brawl
// world; Aria's blows never land).

func lanceRounds(b *brawl, rounds int) string {
	var out string
	for i := 0; i < rounds; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	return out
}

func TestSorcererLoosesALanceAfterAChantAndPaysItsMana(t *testing.T) {
	b := eliteCaster(t, "sorcerer", 10, "arcanelance", "mm")
	stream := b.listen()
	b.startWitchFight()
	assert.Equal(t, 300-18, b.aria.Character.Mana, "the chant's mana is spent as it begins")
	out := lanceRounds(b, 6)
	assert.Contains(t, out, "chanting: Arcane Lance, 3 rounds")
	assert.Regexp(t, `You hurl the lance, and it drives into .* \(\d+ damage\)`, out)
	assert.Positive(t, castEvents(*stream, combatstream.CastStart, "Aria"))
}

func TestSorcererLanceHitsAboutTwiceAMagicMissile(t *testing.T) {
	dmg := func(spell string) int {
		b := eliteCaster(t, "sorcerer", 10, spell)
		b.startWitchFight()
		out := lanceRounds(b, 6)
		re := regexp.MustCompile(`\((\d+) damage\)`)
		best := 0
		for _, m := range re.FindAllStringSubmatch(out, -1) {
			n, _ := strconv.Atoi(m[1])
			best = max(best, n)
		}
		return best
	}
	lance, missile := dmg("arcanelance"), dmg("mm")
	assert.Greater(t, lance, missile*3/2, "a lance (%d) well over a missile (%d)", lance, missile)
}

func TestSorcererLanceNeedsTheRankToBeKnown(t *testing.T) {
	// Below level 10 the rank has not taught it: an unknown spell is never cast.
	b := eliteCaster(t, "sorcerer", 9, "arcanelance")
	b.startWitchFight()
	assert.Equal(t, 300, b.aria.Character.Mana, "nothing is cast")
}

func TestSorcererFallsBackToMagicMissileWhenTheLanceIsOutOfReach(t *testing.T) {
	b := eliteCaster(t, "sorcerer", 10, "arcanelance", "mm")
	b.aria.Character.Mana = 10
	b.startWitchFight()
	out := lanceRounds(b, 4)
	assert.NotContains(t, out, "chanting: Arcane Lance")
	assert.Contains(t, out, "You release the light", "Magic Missile instead")
}

func TestHighSorcererTwinLanceStrikesASecondFoe(t *testing.T) {
	b := eliteCaster(t, "high-sorcerer", 45, "arcanelance")
	b.startWitchFight()
	out := lanceRounds(b, 8)
	assert.Contains(t, out, "twin lance")
	assert.Contains(t, out, "A second lance of light strikes")
}

func TestHighSorcererInstantLanceNeedsNoChantOnceABattle(t *testing.T) {
	b := eliteCaster(t, "high-sorcerer", 60, "arcanelance")
	b.startWitchFight()
	assert.True(t, b.aria.Character.RT.LanceFreed, "the opening Lance was the instant one")
	assert.Contains(t, strings.Join(*b.messages, "\n"), "(instant lance, no chant)")
	out := lanceRounds(b, 8)
	assert.NotContains(t, out, "instant lance", "once a battle")
}

func TestHighSorcererGatheredChantIsShorterThanTheSorcerers(t *testing.T) {
	count := func(class string, level int) int {
		b := eliteCaster(t, class, level, "arcanelance")
		stream := b.listen()
		b.startWitchFight()
		lanceRounds(b, 10)
		return castEvents(*stream, combatstream.CastStart, "Aria")
	}
	slow, quick := count("sorcerer", 25), count("high-sorcerer", 35)
	assert.Greater(t, quick, slow, "more Lances in the same rounds")
}

func TestHighSorcererLanceCostsLessAtRank50(t *testing.T) {
	b := eliteCaster(t, "high-sorcerer", 50, "arcanelance")
	require.Equal(t, 300, b.aria.Character.Mana)
	b.startWitchFight()
	assert.Equal(t, 300-12, b.aria.Character.Mana)
}
