package usercommands

import (
	"strings"
	"testing"

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
