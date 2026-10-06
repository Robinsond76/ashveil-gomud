package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// setOutput runs `set <rest>` for user and returns what it told them, and
// the settings it reported changed.
func setOutput(t *testing.T, user *users.UserRecord, rest string) (string, []string) {
	t.Helper()
	events.ProcessEvents()
	var told []string
	var changed []string
	freshEvents(t)
	msg := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == user.UserId {
			told = append(told, tagPattern.ReplaceAllString(m.Text, ""))
		}
		return events.Continue
	})
	setting := events.RegisterListener(events.UserSettingChanged{}, func(e events.Event) events.ListenerReturn {
		changed = append(changed, e.(events.UserSettingChanged).Name)
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, msg)
	defer events.UnregisterListener(events.UserSettingChanged{}, setting)
	handled, err := Set(rest, user, nil, 0)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	return strings.Join(told, ""), changed
}

func TestSetCombatPace(t *testing.T) {
	user := users.NewUserRecord(4201, 1)

	out, _ := setOutput(t, user, "")
	assert.Contains(t, out, "combatpace: normal (default)", "set lists the pace and its default")

	out, changed := setOutput(t, user, "combatpace slow")
	assert.Contains(t, out, "Combat pace set to slow.")
	assert.Equal(t, "slow", user.GetConfigOption(combatpace.OptionKey))
	assert.Equal(t, []string{combatpace.OptionKey}, changed, "the change is announced so held lines are sent")

	out, changed = setOutput(t, user, "combatpace sideways")
	assert.Contains(t, out, "must be one of: fast, normal, slow, or off")
	assert.Equal(t, "slow", user.GetConfigOption(combatpace.OptionKey), "a bad value changes nothing")
	assert.Empty(t, changed)

	out, _ = setOutput(t, user, "combatpace")
	assert.Contains(t, out, "combatpace: slow")
	assert.Contains(t, out, "set combatpace fast|normal|slow|off")

	// The choice is saved with the user.
	data, err := yaml.Marshal(user)
	require.NoError(t, err)
	var loaded users.UserRecord
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	assert.Equal(t, combatpace.Slow, combatpace.For(loaded.GetConfigOption(combatpace.OptionKey), false))

	// A screen reader's default is off.
	reader := users.NewUserRecord(4202, 1)
	reader.ScreenReader = true
	out, _ = setOutput(t, reader, "")
	assert.Contains(t, out, "combatpace: off (default)")
}

// Phase 49: `set banter` toggles company banter, which is on when unset.
func TestSetBanter(t *testing.T) {
	user := users.NewUserRecord(4202, 1)

	out, _ := setOutput(t, user, "")
	assert.Contains(t, out, "banter: ON", "set lists banter, on by default")

	out, changed := setOutput(t, user, "banter off")
	assert.Contains(t, out, "Company banter toggled OFF.")
	assert.Equal(t, false, user.GetConfigOption(banter.OptionKey))
	assert.Equal(t, []string{banter.OptionKey}, changed)
	out, _ = setOutput(t, user, "")
	assert.Contains(t, out, "banter: OFF")

	out, _ = setOutput(t, user, "banter")
	assert.Contains(t, out, "Company banter toggled ON.", "bare set banter toggles")
	assert.Equal(t, true, user.GetConfigOption(banter.OptionKey))
	out, _ = setOutput(t, user, "banter off")
	out, _ = setOutput(t, user, "banter on")
	assert.Contains(t, out, "toggled ON.")

	out, changed = setOutput(t, user, "banter loudly")
	assert.Contains(t, out, "Usage: set banter on|off")
	assert.Empty(t, changed)
}
