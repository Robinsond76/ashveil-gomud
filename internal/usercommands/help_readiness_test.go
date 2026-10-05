package usercommands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestReadinessHelp: Phase 33h2's page renders, answers to its aliases,
// sits in the company category, and the pages it changes link it and no
// longer promise a free refill.
func TestReadinessHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"readiness": {"still on 8 when you next log in", "last", "automatic save", "keeps the health and mana it had", "never comes back below 1", "starts full once",
			"Nobody recovers while you're logged out", "full health and mana", "Nobody regains mana this way", "up to half their", "help patch", "half its", "as hurt as when it ran"},
		"company":   {"Readiness", "help readiness", "keeps the health and mana it had"},
		"health":    {"help readiness", "only up to half your", "restore you all to full"},
		"inn":       {"full health and mana", "help readiness"},
		"camp":      {"full mana, once per rest", "help readiness"},
		"resurrect": {"half their health and half their mana", "help readiness"},
		"wounds":    {"at half health", "help readiness"},
		"heal":      {"help readiness"},
		"quit":      {"help readiness"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	company, err := GetHelpContents("company")
	require.NoError(t, err)
	assert.NotContains(t, tagPattern.ReplaceAllString(company, ""), "the companion is healed", "a level no longer heals")
	wounds, err := GetHelpContents("wounds")
	require.NoError(t, err)
	assert.NotContains(t, tagPattern.ReplaceAllString(wounds, ""), "comes back whole")

	for _, alias := range []string{"recovery", "vitals", "companion health", "regen"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for readiness", alias)
	}
	var category []string
	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.Category == "company" {
			category = append(category, info.Command)
		}
	}
	assert.Contains(t, category, "readiness")
}

// Review finding (33h2): "recover" is the brawling skill's topic; a second
// topic claiming the alias made `help recover` depend on map order.
func TestHelpRecoverStaysBrawling(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	for i := 0; i < 5; i++ {
		keywords.LoadAliases()
		text, err := GetHelpContents("recover")
		require.NoError(t, err)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for brawling")
	}
}

// Every help alias names one topic: the alias table is built from a map,
// so an alias under two topics resolves at random.
func TestHelpAliasesAreUnique(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "keywords.yaml"))
	require.NoError(t, err)
	var parsed struct {
		HelpAliases map[string][]string `yaml:"help-aliases"`
	}
	require.NoError(t, yaml.Unmarshal(data, &parsed))
	owner := map[string]string{}
	for topic, aliases := range parsed.HelpAliases {
		for _, alias := range aliases {
			alias = strings.ToLower(alias)
			if prior, ok := owner[alias]; ok && prior != topic {
				t.Errorf("help alias %q names both %q and %q", alias, prior, topic)
			}
			owner[alias] = topic
		}
	}
}
