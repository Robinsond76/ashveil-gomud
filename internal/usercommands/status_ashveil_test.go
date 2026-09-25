package usercommands

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// useWorld points the data files at a shipped world ("default" or
// "empty") for the test.
func useWorld(t *testing.T, world string) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", world)
	set := func(value string) error {
		flat := configs.Flatten(configs.GetOverrides())
		flat["FilePaths.DataFiles"] = value
		return configs.RestoreOverrides(flat)
	}
	previous := configs.GetFilePathsConfig().DataFiles.String()
	require.NoError(t, set(dir))
	t.Cleanup(func() { require.NoError(t, set(previous)) })
}

// useSummary makes the read model return s.
func useSummary(t *testing.T, s companyview.Summary) {
	t.Helper()
	summaryFor = func(*users.UserRecord) companyview.Summary { return s }
	t.Cleanup(func() { summaryFor = companyview.For })
}

// sampleSummary: hungry and tired, chilled, in dim light, with two living
// companions and one fallen, burdened, resting at camp, Rested, and a
// checkpoint.
func sampleSummary() companyview.Summary {
	need := func(v int, label string) companyview.Need {
		return companyview.Need{Known: true, Value: v, Label: label}
	}
	return companyview.Summary{
		Leader: companyview.Member{Leader: true, Archetype: "Ranger", ArchetypeKnown: true, Hunger: need(40, "Hungry"), Thirst: need(90, "Hydrated"),
			Fatigue: need(45, "Tired"), Warmth: "Chilled", WarmthKnown: true},
		CompanyKnown: true,
		Companions:   []companyview.Member{{ID: 1}, {ID: 2}, {ID: 3, Status: company.MemberDead}},
		Alive:        3, Dead: 1,
		LoadKnown: true, Load: encumbrance.Load{PersonalGrams: 5000, CargoGrams: 3000, CapacityGrams: 10000}, LoadLabel: "Burdened",
		ActivityKnown: true, Activity: companyview.Activity{Kind: companyview.CampRest, Remaining: 12 * time.Minute},
		RestKnown: true, RestTier: camping.TierRested, RestLeft: time.Hour,
		LightKnown: true, Light: 1,
		Alignment:  40,
		Checkpoint: "The Chapel of the Wayfarer",
	}
}

func statusText(t *testing.T, user *users.UserRecord, rest string) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Status(rest, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
}

// TestStatusAshveilSheet (Phase 26a): the shipped layout's character,
// vitals, and company panels, from the read model.
func TestStatusAshveilSheet(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Wren"
	text := statusText(t, user, "")
	for _, want := range []string{
		"Character", "Vitals", "Company", "Attributes", "Strength",
		"Path:", "Ranger", "Align:", "40 (",
		"Health:", "Hunger:", "Hungry (40)", "Fatigue:", "Tired (45)", "Warmth:", "Chilled", "Light:", "Dim",
		"Members:", "3 alive, 1 fallen", "Load:", "Burdened (8.0/10.0 kg)", "Doing:", "Resting 12m",
		"Rest:", "Rested (1h 0m left)", "Wake at:", "The Chapel of the Wayfarer",
		"More: company status, formation, conditions, survival",
	} {
		assert.Contains(t, text, want)
	}
}

// TestStatusUnknownsAreLeftOut: with nothing known, no survival or company
// row pretends to a value.
func TestStatusUnknownsAreLeftOut(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, companyview.Summary{Alive: 1})
	text := statusText(t, users.NewUserRecord(7, 1), "")
	for _, absent := range []string{"Hunger:", "Warmth:", "Light:", "Load:", "Doing:", "Rest:", "Wake at:"} {
		assert.NotContains(t, text, absent)
	}
	assert.Regexp(t, `Members: +unknown`, text)
	assert.NotContains(t, text, "Path:", "no archetype provider: left out")
}

// TestStatusEmptyWorldLayoutUnchanged: the upstream empty world's layout
// has no Ashveil panels and keeps the engine's sheet, with no errors.
func TestStatusEmptyWorldLayoutUnchanged(t *testing.T) {
	useWorld(t, "empty")
	useSummary(t, sampleSummary())
	text := statusText(t, users.NewUserRecord(7, 1), "")
	assert.Contains(t, text, "Health:")
	assert.NotContains(t, text, "Vitals")
	assert.NotContains(t, text, "Hunger:")
	assert.NotContains(t, text, "More:")
}

// TestStatusTrainUnchanged: stat training still works.
func TestStatusTrainUnchanged(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	assert.Contains(t, statusText(t, users.NewUserRecord(7, 1), "train"), "points left to spend")
}
