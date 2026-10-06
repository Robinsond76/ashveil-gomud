package usercommands

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
	keywords.LoadAliases() // help and aliases read the world's keywords.yaml
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

// Phase 38c1 review: the status sheet names the class, its tier, and a
// promotion that is ready.
func TestStatusShowsTheClass(t *testing.T) {
	useWorld(t, "default")
	s := sampleSummary()
	s.Leader.Archetype, s.Leader.ClassName, s.Leader.ClassTier, s.Leader.ClassRank = "Warrior", "Mercenary", "advanced", 25
	s.Leader.Promotion = "ready"
	useSummary(t, s)
	text := statusText(t, users.NewUserRecord(7, 1), "")
	assert.Regexp(t, `(Cls|Class): +Mercenary \(advanced\)`, text)
	assert.Regexp(t, `(Prm|Promote): +ready \(class\)`, text)
	s.Leader.ClassName, s.Leader.ClassTier, s.Leader.Promotion = "Warlord", "elite", ""
	useSummary(t, s)
	text = statusText(t, users.NewUserRecord(7, 1), "")
	assert.Regexp(t, `(Cls|Class): +Warlord \(elite\)`, text)
	assert.NotRegexp(t, `(Prm|Promote): `, text)
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

// TestStatusShowsBurden (Phase 30g3): the Vitals panel names how burdened
// the character's own worn and carried load leaves them, as a word.
func TestStatusShowsBurden(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.AgilityBaseKg, gameplay.Combat.AgilityStrengthKg, gameplay.Combat.AgilityFreeLoad = 15, 0.5, 0.35
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	const anvilID = 99611
	items.SetTestItemSpec(&items.ItemSpec{ItemId: anvilID, Name: "test anvil", Type: items.Object, Weight: 40000})
	t.Cleanup(func() { items.RemoveTestItemSpec(anvilID) })

	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Wren"
	text := statusText(t, user, "")
	assert.Regexp(t, `Burden: +Unburdened`, text)

	user.Character.Items = append(user.Character.Items, items.New(anvilID))
	text = statusText(t, user, "")
	assert.Regexp(t, `Burden: +Heavily burdened`, text)
	assert.NotContains(t, text, "40.0", "a word, never the weight or a ratio")
}

// TestStatusShowsSkillAndBulk (Phase 35a2): status shows Attack, Evasion
// and the armor bulk, marked when heavier than the class's training.
func TestStatusShowsSkillAndBulk(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	const plateID = 99612
	plate := &items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, Subtype: items.Wearable, Weight: 12000, DamageReduction: 10}
	require.NoError(t, plate.Validate())
	items.SetTestItemSpec(plate)
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })

	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Wren"
	user.Character.Level = 12
	text := statusText(t, user, "")
	assert.Regexp(t, `Attack: +12 +Evasion: 12`, text, "enemies' default rates without a class")
	assert.Regexp(t, `Bulk: +Light`, text)

	user.Character.Equipment.Body = items.New(plateID)
	text = statusText(t, user, "")
	assert.Regexp(t, `Bulk: +Heavy`, text)
	assert.NotContains(t, text, "untrained", "no class: trained for anything")
}

// rogueGear is a provider with a rogue's 35a2 profile: trained for light
// armor only.
type rogueGear struct{}

func (rogueGear) CanTrain(int, string) (bool, string)      { return true, "" }
func (rogueGear) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (rogueGear) Exists(string) bool                       { return true }
func (rogueGear) ArchetypeName(string) (string, bool)      { return "Rogue", true }
func (rogueGear) PlayerArchetype(int) (string, bool)       { return "rogue", true }
func (rogueGear) CombatProfile(id string) (archetypes.Profile, bool) {
	return archetypes.Profile{Name: "Rogue", AttackRate: 0.9, EvasionRate: 1.1, ArmorTraining: items.BulkLight,
		ShieldSizes: []string{"none"}}, id == "rogue"
}

// TestUntrainedArmorThroughEquipAndStatus (35a2 review): a rogue who
// puts on heavy armor with plain equip (no company) is warned, and status
// marks the bulk untrained and shows Attack and Evasion 10 lower.
func TestUntrainedArmorThroughEquipAndStatus(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	races.LoadDataFiles()
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.UntrainedSkillLoss, cfg.Combat.UntrainedChantRounds = 10, 1
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	const plateID = 99613
	plate := &items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, Subtype: items.Wearable, Weight: 12000, DamageReduction: 10, Bulk: items.BulkHeavy}
	require.NoError(t, plate.Validate())
	items.SetTestItemSpec(plate)
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })
	archetypes.SetProvider(rogueGear{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })

	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Wren"
	user.Character.Level = 20
	user.Character.RaceId = 1 // human: every armor slot
	require.True(t, user.Character.StoreItem(items.New(plateID)))
	text := statusText(t, user, "")
	assert.Regexp(t, `Attack: +18 +Evasion: 22`, text, "a rogue's rates at level 20")

	messages := captureLookMessages(t)
	_, err := Equip("test plate", user, &rooms.Room{RoomId: 1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	said := tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	assert.Contains(t, said, "You wear your test plate.")
	assert.Contains(t, said, "You are trained for light armor, not heavy.")
	assert.Equal(t, plateID, user.Character.Equipment.Body.ItemId, "allowed, with a warning")

	text = statusText(t, user, "")
	assert.Regexp(t, `Bulk: +Heavy, untrained`, text)
	assert.Regexp(t, `Attack: +8 +Evasion: 12`, text, "10 lower in untrained armor")
}

// TestStatusShowsClassesAndCompanyRoster (Phase 45): the sheet names the
// leader's path and promoted class (38c1's Class row), lists each companion's class and
// level, and shows a gather in progress as what the company is doing.
func TestStatusShowsClassesAndCompanyRoster(t *testing.T) {
	useWorld(t, "default")
	sum := sampleSummary()
	sum.Leader.Class = "knight"
	sum.Leader.Archetype = "Warrior"
	sum.Leader.ClassName, sum.Leader.ClassTier = "Knight", "advanced"
	sum.Companions = []companyview.Member{
		{ID: 1, Name: "Oswin", Level: 6, Archetype: "Cleric", Class: "priest", Status: company.MemberPresent},
		{ID: 2, Name: "Brant", Level: 4, Archetype: "Warrior", Status: company.MemberPresent},
		{ID: 3, Name: "Ysolde", Level: 5, Archetype: "Mage", Status: company.MemberDead},
	}
	sum.Activity = companyview.Activity{Kind: companyview.Gathering, Detail: "gathering herbs", Percent: 40, Remaining: 12 * time.Second}
	useSummary(t, sum)
	text := statusText(t, users.NewUserRecord(7, 1), "")
	for _, want := range []string{
		"Path:", "Warrior", "Knight (advanced)", "Oswin:", "Priest (Cleric), Lv 6", "Brant:", "Warrior, Lv 4", "Ysolde:", "Mage, Lv 5 (fallen)",
		"Doing:", "Gathering herbs 40%, 12s left",
	} {
		assert.Contains(t, text, want)
	}
}

// TestStatusHelpMentionsClasses (Phase 45): the page says the sheet shows
// the promoted class and each companion's class.
func TestStatusHelpMentionsClasses(t *testing.T) {
	useWorld(t, "default")
	text, err := GetHelpContents("status")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "each companion's class and level")
}
