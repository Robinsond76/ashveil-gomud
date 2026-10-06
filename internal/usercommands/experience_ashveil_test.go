package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func experienceText(t *testing.T, user *users.UserRecord, rest string) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Experience(rest, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
}

// TestExperienceShowsLastLoss (Phase 26a).
func TestExperienceShowsLastLoss(t *testing.T) {
	useWorld(t, "default")
	user := users.NewUserRecord(7, 1)
	assert.NotContains(t, experienceText(t, user, ""), "last death")
	user.Character.SetMiscData(death.LastLossKey, death.LastLossValue(6, 5))
	text := experienceText(t, user, "")
	assert.Contains(t, text, "Your last death cost you a level: 6 to 5.")
	user.Character.SetMiscData(death.LastLossKey, death.LastLossValue(1, 1))
	assert.Contains(t, experienceText(t, user, ""), "progress toward level 2")
}

// fakeMembers is a company provider whose members carry live progress.
type fakeMembers struct {
	fakeChemistryStanding
	views []company.MemberView
}

func (f fakeMembers) CompanyMembers(int) ([]company.MemberView, bool) { return f.views, true }

// TestExperienceListsTheCompany drives the real command (Phase 32e): each
// companion's level and progress under the leader's block, the fallen
// marked, and a solo player's output unchanged.
func TestExperienceListsTheCompany(t *testing.T) {
	useWorld(t, "default")
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	user := users.NewUserRecord(7, 1)

	company.SetFormationProvider(fakeMembers{views: []company.MemberView{}})
	solo := experienceText(t, user, "")
	assert.NotContains(t, solo, "Your company")

	company.SetFormationProvider(fakeMembers{views: []company.MemberView{
		{ID: 1, Name: "Corvin", Level: 4, Status: company.MemberPresent, ExpInto: 50, ExpTNL: 200, ExpKnown: true},
		{ID: 2, Name: "Mira", Level: 2, Status: company.MemberAwaiting},
		{ID: 3, Name: "Tobb", Level: 3, Status: company.MemberDead},
	}})
	text := experienceText(t, user, "")
	assert.True(t, strings.HasPrefix(text, strings.Split(solo, "\n")[0]), "the leader's block still leads")
	assert.Contains(t, text, "Your company:")
	assert.Contains(t, text, "Corvin Lvl: 4 XP: 50/200 (25%)")
	assert.Contains(t, text, "Mira Lvl: 2")
	assert.Contains(t, text, "Tobb level 3, fallen")
}

// Phase 35a: exercise the actual command and shipped template, including the
// end of the announced milestone schedule. Companion lines remain unchanged.
func TestExperienceShowsNextMilestone(t *testing.T) {
	useWorld(t, "default")
	user := users.NewUserRecord(7, 1)
	for _, tc := range []struct {
		level int
		text  string
	}{{1, "level 3: second class option (coming)"}, {9, "level 10: class promotion"}, {29, "level 30: elite promotion (coming)"}, {30, "level 35: elite rank and talent (coming)"}, {60, "No upcoming milestone announced."}} {
		user.Character.Level = tc.level
		user.Character.Validate()
		text := experienceText(t, user, "")
		assert.Contains(t, text, "Next: "+tc.text)
		assert.NotContains(t, text, "<no value>")
	}
}
