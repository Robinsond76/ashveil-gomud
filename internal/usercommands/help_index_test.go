package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
)

// TestStorageHelpAnswersToItsCommands: found by the live playtest (Phase
// 44), `help bid`, `help store` and `help unstore` answered "No help found":
// the index listed a command that does not exist and two aliases of the
// storage page as pages of their own. (The live smoke test renders every
// indexed topic, with the modules' own pages loaded.)
func TestStorageHelpAnswersToItsCommands(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		listed = append(listed, topic.Command)
	}
	assert.Contains(t, listed, "storage")
	assert.NotContains(t, listed, "bid", "there is no bid command and no page for it")

	want, err := GetHelpContents("storage")
	assert.NoError(t, err)
	assert.NotEmpty(t, want)
	for _, alias := range []string{"store", "unstore"} {
		got, err := GetHelpContents(alias)
		assert.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help storage", alias)
	}
}

// `help help` lists the index like a bare `help`; it answered "No help
// found" before.
func TestHelpAboutHelpListsTheIndex(t *testing.T) {
	assert.True(t, isHelpIndexRequest(nil))
	assert.True(t, isHelpIndexRequest([]string{"help"}))
	assert.True(t, isHelpIndexRequest([]string{"HELP"}))
	assert.False(t, isHelpIndexRequest([]string{"mana"}))
	assert.False(t, isHelpIndexRequest([]string{"help", "mana"}))
}
