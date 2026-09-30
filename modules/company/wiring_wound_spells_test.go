package company

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// castOnSelf has Aria cast spell on herself out of a fight, through the
// real cast command and round, until it lands or fizzles out of tries.
func (b *brawl) castOnSelf(spell string, landed func(string) bool) string {
	b.t.Helper()
	c := b.aria.Character
	c.Aggro = nil
	c.SetSkill(`cast`, 4)
	c.SpellBook[spell] = 5000
	c.Stats.Mysticism.ValueAdj = 1000
	var transcript string
	for try := 0; try < 20; try++ {
		c.ManaMax.Value, c.Mana = 100, 100
		transcript += b.cmd("cast", spell+" aria") + "\n"
		for i := 0; i < 4; i++ {
			out := b.fight()
			transcript += out + "\n"
			if landed(out) {
				return transcript
			}
		}
	}
	return transcript
}

var woundLimitNote = regexp.MustCompile(`\((\d+) healed, wound limit (\d+) of (\d+)\)`)

// A heal the wound limit holds back says so, through a real cast.
func TestAHeldBackHealNamesTheWoundLimit(t *testing.T) {
	b := newBrawl(t)
	c := b.aria.Character
	c.HealthMax.Value, c.Health = 100, 49
	c.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 50}} // limit 50
	out := b.castOnSelf("heal", func(s string) bool { return strings.Contains(s, "healed") })
	m := woundLimitNote.FindStringSubmatch(out)
	require.NotNil(t, m, "the held-back heal names the limit: %s", out)
	assert.Equal(t, "1", m[1])
	assert.Equal(t, "50", m[2])
	assert.Equal(t, "100", m[3])
	assert.Equal(t, 50, c.Health, "healed to the limit, no further")
}

// Tend closes wound points, through a real cast, and restores no health.
func TestTendClosesAWoundThroughARealCast(t *testing.T) {
	b := newBrawl(t)
	c := b.aria.Character
	c.HealthMax.Value, c.Health = 100, 40
	c.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 20}}
	out := b.castOnSelf("tend", func(s string) bool { return strings.Contains(s, "wound treated") })
	require.Contains(t, out, "a broken arm", out)
	assert.Regexp(t, `\(wound treated, limit \d+ of 100\)`, out)
	require.Len(t, c.Wounds, 1)
	closed := 20 - c.Wounds[0].Points
	assert.GreaterOrEqual(t, closed, 2)
	assert.LessOrEqual(t, closed, 6)
	assert.Equal(t, 40, c.Health, "tending heals no health itself")
}
