package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAilmentsHelpIsIndexedAndMatchesTheRules (Phase 55): the page renders
// through help, is listed under the road category, answers to its aliases,
// and quotes the numbers survival's rules hold.
func TestAilmentsHelpIsIndexedAndMatchesTheRules(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var road []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "road" && !topic.AdminOnly {
			road = append(road, topic.Command)
		}
	}
	assert.Contains(t, road, "ailments")

	text, err := GetHelpContents("ailments")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for ailments")
	for _, a := range survival.Ailments() {
		assert.Contains(t, plain, a.Name, a.Kind)
		assert.Contains(t, plain, a.RemedyName, a.Kind)
		assert.Contains(t, plain, a.Effect()[:3], a.Kind)
		for _, ing := range a.Remedy {
			assert.Contains(t, plain, ing.Name, a.Kind)
		}
	}
	assert.Contains(t, plain, "-10% damage dealt")
	assert.Contains(t, plain, "+10% damage taken")
	assert.Contains(t, plain, "-15% damage dealt")

	for _, alias := range []string{"chill", "gut-ache", "fever", "remedy", "sickness"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help ailments", alias)
	}
	for _, topic := range []string{"survival", "wounds", "campsupplies", "cooking", "conditions", "combat"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, page, "ailments", "help %s points to the new page", topic)
	}
}

// TestAilmentRowsNameTheCureAndTheBattlesLeft (Phase 55): the conditions
// rows and the numbers they quote come from the rules.
func TestAilmentRowsNameTheCureAndTheBattlesLeft(t *testing.T) {
	rows := ailmentRows([]string{"Chill (3 battles)", "Fever (1 battle)", "Plague (2 battles)"})
	require.Len(t, rows, 2, "an unknown ailment is skipped")
	assert.Equal(t, "Chill", rows[0].name)
	assert.Equal(t, "-10% damage; 3 battles left. Treat it with camp prepare remedy.", rows[0].description)
	assert.Contains(t, rows[1].description, "1 battle left")
}
