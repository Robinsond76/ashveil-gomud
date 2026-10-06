package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSkillsFeedHonorsRetiredEntriesAndRealRankCaps(t *testing.T) {
	old := skills.GetAllSkills()
	restore := []*skills.Skill{}
	for i := range old {
		restore = append(restore, &old[i])
	}
	t.Cleanup(func() { skills.SetTestData(restore, nil) })
	skills.SetTestData([]*skills.Skill{{SkillId: "protection", Name: "Protection", Description: "Protect your allies.", MaxLevel: 3}, {SkillId: "brawling", MaxLevel: 4}, {SkillId: "search", MaxLevel: 4}}, nil)
	u := inventoryUser(t)
	u.Character.Skills = map[string]int{"protection": 4, "brawling": 2, "search": 4, "orphan": 4}
	node, name := (&GMCPCharModule{}).GetCharNode(u, "Char.Skills")
	assert.Equal(t, "Char.Skills", name)
	list, ok := node.([]GMCPCharModule_Payload_Skill)
	require.True(t, ok)
	require.Len(t, list, 2)
	for _, skill := range list {
		if skill.Name == "protection" {
			assert.Equal(t, 3, skill.Level)
			assert.Equal(t, 3, skill.MaxLevel)
			assert.True(t, skill.Maximum)
			assert.Equal(t, "Protection", skill.Title, "the panel shows the skill's display name")
			assert.Equal(t, "Protect your allies.", skill.Description, "the panel explains what the skill does")
		}
	}
	assert.Equal(t, 4, u.Character.Skills["protection"], "preview does not migrate or mutate save state")
}
