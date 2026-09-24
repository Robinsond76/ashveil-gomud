package archetypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wizard() Archetype {
	return Archetype{
		ID:              "wizard",
		Name:            "Wizard",
		Skills:          []string{"cast", "enchant"},
		Schools:         []string{"illusion", "conjuration"},
		GrantSkills:     map[string]int{"cast": 1},
		GrantSpells:     []string{"floatinglight"},
		Utility:         []string{"light"},
		CompanionLevels: []int{1, 10, 20, 30},
	}
}

func cleric() Archetype {
	return Archetype{
		ID:              "cleric",
		Name:            "Cleric",
		Skills:          []string{"cast", "protection"},
		Schools:         []string{"restoration"},
		GrantSkills:     map[string]int{"cast": 1, "protection": 1},
		CompanionLevels: []int{1, 10, 20, 30},
	}
}

func rogue() Archetype {
	return Archetype{
		ID:              "rogue",
		Name:            "Rogue",
		Skills:          []string{"skulduggery", "peep"},
		GrantSkills:     map[string]int{"skulduggery": 1},
		Utility:         []string{"traps"},
		CompanionLevels: []int{1, 10, 20, 30},
	}
}

func TestArchetypeValidate(t *testing.T) {
	ok := wizard()
	require.NoError(t, ok.Validate())

	cases := map[string]func(a *Archetype){
		"empty id":             func(a *Archetype) { a.ID = " " },
		"bad id":               func(a *Archetype) { a.ID = "Wiz ard" },
		"empty name":           func(a *Archetype) { a.Name = "" },
		"no skills":            func(a *Archetype) { a.Skills = nil; a.GrantSkills = nil },
		"grant unlisted":       func(a *Archetype) { a.GrantSkills = map[string]int{"brawling": 1} },
		"grant level zero":     func(a *Archetype) { a.GrantSkills = map[string]int{"cast": 0} },
		"levels too short":     func(a *Archetype) { a.CompanionLevels = []int{1, 10, 20} },
		"levels not ascending": func(a *Archetype) { a.CompanionLevels = []int{1, 20, 10, 30} },
		"levels below one":     func(a *Archetype) { a.CompanionLevels = []int{0, 10, 20, 30} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			a := wizard()
			mutate(&a)
			assert.Error(t, a.Validate())
		})
	}
}

func TestArchetypeValidateNormalizes(t *testing.T) {
	a := wizard()
	a.ID = " Wizard "
	a.Skills = []string{"Cast", "cast", " enchant "}
	a.Schools = []string{"Illusion"}
	a.Utility = []string{"LIGHT"}
	require.NoError(t, a.Validate())
	assert.Equal(t, "wizard", a.ID)
	assert.Equal(t, []string{"cast", "enchant"}, a.Skills)
	assert.Equal(t, []string{"illusion"}, a.Schools)
	assert.Equal(t, []string{"light"}, a.Utility)
}

func TestValidateGrantedSpells(t *testing.T) {
	schools := map[string]string{"floatinglight": "illusion", "heal": "restoration", "plain": ""}
	schoolOf := func(id string) (string, bool) {
		s, ok := schools[id]
		return s, ok
	}
	a := wizard()
	require.NoError(t, a.ValidateGrantedSpells(schoolOf))

	a.GrantSpells = []string{"heal"}
	assert.Error(t, a.ValidateGrantedSpells(schoolOf), "outside claimed schools")

	a.GrantSpells = []string{"nosuch"}
	assert.Error(t, a.ValidateGrantedSpells(schoolOf), "unknown spell")

	a.GrantSpells = []string{"plain"}
	assert.NoError(t, a.ValidateGrantedSpells(schoolOf), "open spell may be granted")
}

func TestNewTableSkipsInvalidAndDuplicates(t *testing.T) {
	bad := wizard()
	bad.Name = ""
	dup := rogue()
	table, errs := NewTable([]Archetype{wizard(), cleric(), rogue(), bad, dup})
	assert.Len(t, errs, 2)
	assert.Equal(t, []string{"cleric", "rogue", "wizard"}, ids(table.List()))
}

func ids(list []Archetype) []string {
	out := []string{}
	for _, a := range list {
		out = append(out, a.ID)
	}
	return out
}

func TestClaims(t *testing.T) {
	table, errs := NewTable([]Archetype{wizard(), cleric(), rogue()})
	require.Empty(t, errs)

	assert.Equal(t, []string{"cleric", "wizard"}, table.SkillClaimants("cast"), "shared claim")
	assert.Equal(t, []string{"rogue"}, table.SkillClaimants("Skulduggery"))
	assert.Empty(t, table.SkillClaimants("search"), "trade skill")
	assert.True(t, table.SkillClaimed("cast"))
	assert.False(t, table.SkillClaimed("search"))

	assert.Equal(t, []string{"wizard"}, table.SchoolClaimants("illusion"))
	assert.Empty(t, table.SchoolClaimants(""))
	assert.Empty(t, table.SchoolClaimants("necromancy"))
}

func TestCanTrain(t *testing.T) {
	table, _ := NewTable([]Archetype{wizard(), cleric(), rogue()})

	ok, _ := table.CanTrain("", "search")
	assert.True(t, ok, "unchosen may train a trade skill")

	ok, reason := table.CanTrain("", "cast")
	assert.False(t, ok, "unchosen must choose first")
	assert.Contains(t, reason, "archetype")

	ok, _ = table.CanTrain("wizard", "cast")
	assert.True(t, ok)
	ok, _ = table.CanTrain("cleric", "cast")
	assert.True(t, ok, "shared claim")

	ok, reason = table.CanTrain("rogue", "cast")
	assert.False(t, ok)
	assert.Contains(t, reason, "Cleric")
	assert.Contains(t, reason, "Wizard")

	ok, _ = table.CanTrain("wizard", "search")
	assert.True(t, ok, "trade skills stay open")

	ok, _ = table.CanTrain("gone", "cast")
	assert.False(t, ok, "unknown archetype is treated as unchosen")
}

func TestCanLearnSchool(t *testing.T) {
	table, _ := NewTable([]Archetype{wizard(), cleric(), rogue()})

	ok, _ := table.CanLearnSchool("", "")
	assert.True(t, ok, "schoolless spell is open")
	ok, _ = table.CanLearnSchool("", "necromancy")
	assert.True(t, ok, "unclaimed school is open")

	ok, reason := table.CanLearnSchool("", "restoration")
	assert.False(t, ok)
	assert.Contains(t, reason, "archetype")

	ok, _ = table.CanLearnSchool("cleric", "Restoration")
	assert.True(t, ok)
	ok, reason = table.CanLearnSchool("wizard", "restoration")
	assert.False(t, ok)
	assert.Contains(t, reason, "Cleric")
}

func TestCompanionUtilityLevel(t *testing.T) {
	a := wizard()
	for level, want := range map[int]int{0: 0, 1: 1, 9: 1, 10: 2, 19: 2, 20: 3, 29: 3, 30: 4, 99: 4} {
		assert.Equal(t, want, a.CompanionSkillLevel(level), "character level %d", level)
	}
	a.CompanionLevels = nil
	assert.Equal(t, 0, a.CompanionSkillLevel(50), "no table, no level")
}

func TestHasUtilityAndSkill(t *testing.T) {
	a := wizard()
	require.NoError(t, a.Validate())
	assert.True(t, a.HasUtility("Light"))
	assert.False(t, a.HasUtility("traps"))
	assert.True(t, a.ListsSkill("CAST"))
	assert.False(t, a.ListsSkill("peep"))
}

func TestValidateDropsNonPositiveKitIDs(t *testing.T) {
	a := Archetype{ID: "warrior", Name: "Warrior", Skills: []string{"brawling"}, CompanionLevels: []int{1, 2, 3, 4}, Kit: []int{10002, 0, -3, 30001, 30001}}
	assert.NoError(t, a.Validate())
	assert.Equal(t, []int{10002, 30001, 30001}, a.Kit, "invalid ids dropped, repeats kept")
}
