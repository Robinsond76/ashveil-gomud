package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// The skill schedule (2026-10-12): the Skills tab shows no ability before
// the level it comes at, so a level-1 brawler has no Tackle card.
func TestCapabilitiesHideAbilitiesBeforeTheirLevel(t *testing.T) {
	old := skills.GetAllSkills()
	restore := []*skills.Skill{}
	for i := range old {
		restore = append(restore, &old[i])
	}
	t.Cleanup(func() { skills.SetTestData(restore, nil) })
	skills.SetTestData([]*skills.Skill{{SkillId: "brawling", MaxLevel: 4}}, nil)
	u := users.NewUserRecord(7, 1)
	u.Character.Skills = map[string]int{"brawling": 1}

	names := func() []string {
		var out []string
		for _, c := range buildCapabilities(u).Automatic {
			out = append(out, c.Name)
		}
		return out
	}
	u.Character.Level = 1
	assert.NotContains(t, names(), "Tackle", "a level-1 warrior has no Tackle yet")
	u.Character.Level = 2
	assert.Contains(t, names(), "Tackle", "Tackle shows from level 2")
}
