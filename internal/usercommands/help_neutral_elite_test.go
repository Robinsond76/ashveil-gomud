package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39i: every neutral elite has a help page that lists its ranks as the
// class table does, is indexed with aliases, and is linked from its route
// page and from help elite.
func TestNeutralEliteHelpPages(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	routes := map[string]string{
		"reaper": "halberdier-routes", "linebreaker": "halberdier-routes", "tempest-lancer": "halberdier-routes",
		"sword-saint": "samurai-routes", "shogun": "samurai-routes", "kenshi": "samurai-routes",
		"tempest-lord": "shaman-routes", "veil-mother": "shaman-routes", "mountain-speaker": "shaman-routes",
		"grand-puppeteer": "dollmaster-routes", "golem-lord": "dollmaster-routes", "string-sovereign": "dollmaster-routes",
		// Phase 39i2
		"packlord": "beasttamer-routes", "beastlord": "beasttamer-routes", "dragon-lord": "beasttamer-routes",
		"gryphon-lord": "gryphon-rider-routes", "falcon-marshal": "gryphon-rider-routes", "wyvern-lord": "gryphon-rider-routes",
		"panacean": "alchemist-routes", "grenadier": "alchemist-routes", "transmuter": "alchemist-routes",
		"siege-master": "arbalist-routes", "deadeye": "arbalist-routes", "bastion": "arbalist-routes",
	}
	indexed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		indexed[topic.Command] = true
	}
	elite, err := GetHelpContents("elite")
	require.NoError(t, err)
	for id, hub := range routes {
		class, ok := classes.Get(id)
		require.True(t, ok, id)
		assert.True(t, indexed[id], "help index lists %s", id)
		text, err := GetHelpContents(id)
		require.NoError(t, err, id)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for "+id)
		for _, r := range class.Ranks {
			assert.Contains(t, plain, r.Name, "%s page lists rank %s", id, r.Name)
		}
		assert.Contains(t, text, "help "+hub, "%s links to its route page", id)
		assert.Contains(t, elite, "help "+id, "help elite links to %s", id)
		route, err := GetHelpContents(hub)
		require.NoError(t, err)
		assert.Contains(t, route, "help "+id, "%s links to %s", hub, id)
		assert.NotContains(t, strings.ToLower(route), "still to come", hub)
	}
	// The page names answer to their spaced and plural aliases.
	for alias, topic := range map[string]string{"sword saint": "sword-saint", "tempest lord": "tempest-lord", "veil mother": "veil-mother", "twin-draw": "sword-saint", "stone-cloak": "mountain-speaker", "third-doll": "grand-puppeteer", "stormstruck": "tempest-lancer", "dragon lord": "dragon-lord", "falcon marshal": "falcon-marshal", "wyvern lord": "wyvern-lord", "gryphon lord": "gryphon-lord", "siege master": "siege-master", "ballista-bolt": "siege-master", "covering-shot": "bastion", "elixir": "panacean", "grenado": "grenadier", "twin-mutagen": "transmuter"} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
}
