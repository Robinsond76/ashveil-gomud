package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCasterPowerHelp: Phase 35b's pages render through help with the
// numbers that matter, answer to their aliases, are indexed, and the hub
// pages link them; the pages the phase made stale no longer say mana comes
// back on its own or that spells fizzle in a battle.
func TestCasterPowerHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	spells.LoadSpellFiles()
	expected := map[string][]string{
		"patch": {"Your company patches itself up.", "Brother Oswin lays glowing hands on Garrick Vane. (13 healed)",
			"back to your feet", "a companion still has a foe", "healing threshold", "mana reserve", "company patch", "10 to 16", "no wounds are tended", "help draughts"},
		"draughts":         {"minor mana draught", "25%", "lesser mana draught", "40%", "greater mana draught", "60%", "out of battle only", "Moilyn"},
		"wounds":           {"3 or more levels below you", "a light wound of a"},
		"friendly-effects": {"Minor Heal takes one round of chanting, Minor Heal All two."},
		"camp":             {"a rest that\n  raiders break restores neither"},
		"mana":             {"coming back from death", "does not come back on its own", "camp rest", "help draughts", "help patch", "40 mana plus 10 a level", "36 plus 9 a level", "doesn't refill"},
		"health":           {"only up to half your"},
		"combat":           {"help patch", "help draughts", "3 or more levels"},
		"cast":             {"never fizzles", "100% chance never does"},
		"spells":           {"never fizzles in a battle", "level 3"},
		"strategy":         {"never fizzles", "Minor Heal All at level 3", "a draught refills it"},
		"interrupts":       {"fifth of its difficulty", "Magic Missile +15"},
		"abilities":        {"1 more for\n    every 3 levels", "a round longer", "help patch"},
		"guardian":         {"1 more for every 10 levels", "as the battle begins"},
		"tactics":          {"help patch", "every 10 levels"},
		"progression":      {"Spells and abilities grow", "Shower of Sparks", "second class option (coming)"},
		"heal":             {"help patch", "only up to half"},
		"spell mm":         {"Power:", "7 + 1d6, +1 per 10 levels, +1 per 15 Mysticism"},
		"spell healall":    {"55% per patient"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	for _, topic := range []string{"mana", "strategy", "health", "readiness", "cast", "combat", "friendly-effects"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, stale := range []string{"regain a little every few seconds", "can fizzle", "Mana comes back out of combat", "regain health and mana", "health and mana recovery pauses", "Both take two rounds"} {
			assert.NotContains(t, plain, stale, topic)
		}
	}
	for alias, topic := range map[string]string{"mana potion": "draughts", "draught": "draughts", "company patch": "patch", "patching": "patch"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for "+topic, alias)
	}
	categories := map[string]string{}
	for _, info := range keywords.GetAllHelpTopicInfo() {
		categories[info.Command] = info.Category
	}
	assert.Equal(t, "company", categories["patch"])
	assert.Equal(t, "items", categories["draughts"])
}
