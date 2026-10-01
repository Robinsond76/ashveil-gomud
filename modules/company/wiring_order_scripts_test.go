package company

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// houndScript answers through the real onAsk entry point with replies outside
// ask's order list, and tries a hostile command when asked about a fight.
const houndScript = `function onAsk(mob, room, eventDetails) {
    if (eventDetails.askText.indexOf("fight") >= 0) {
        mob.Command("attack Aria");
        return true;
    }
    mob.Command("say thanks friend");
    mob.Command("saytoonly @" + String(eventDetails.sourceId) + " psst a secret");
    mob.Command("nods");
    return true;
}
`

// scriptedHound is a follower with a conversation script, charmed to owner.
func (b *brawl) scriptedHound(owner int) *mobs.Mob {
	b.t.Helper()
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "mobs", "brawl")
	require.NoError(b.t, os.MkdirAll(filepath.Join(dir, "scripts"), 0755))
	require.NoError(b.t, os.WriteFile(filepath.Join(dir, "9110-hound.yaml"), []byte(banditMob(9110, "hound", 1)), 0600))
	require.NoError(b.t, os.WriteFile(filepath.Join(dir, "scripts", "9110-hound.js"), []byte(houndScript), 0600))
	mobs.LoadDataFiles()
	scripting.ClearMobVMs()
	b.t.Cleanup(scripting.ClearMobVMs)
	hound := mobs.NewMobById(9110, b.road.RoomId)
	require.NotNil(b.t, hound)
	hound.Character.Charm(owner, -1, "")
	b.road.AddMob(hound.InstanceId)
	return hound
}

func TestScriptRepliesOfAnotherPlayersFollowerAreNotOrders(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(8)
	var tagged []events.Input
	listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in := e.(events.Input); in.MemberOrder != nil {
			tagged = append(tagged, in)
		}
		return events.Continue
	}, events.First)
	defer events.UnregisterListener(events.Input{}, listener)

	text := b.cmd("ask", "hound about bones")
	assert.Contains(t, text, "thanks friend", "the follower's own reply runs")
	assert.NotContains(t, text, "no longer command", "Aria is not told she lost a member she never had")
	assert.NotContains(t, text, "not available")
	assert.Empty(t, tagged, "another player's follower is never ordered by Aria")
	assert.Equal(t, 8, hound.Character.Charmed.UserId)
}

func TestScriptRepliesOfOwnFollowerUseFullConversation(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(b.aria.UserId)
	var tagged []events.Input
	listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in := e.(events.Input); in.MemberOrder != nil {
			tagged = append(tagged, in)
		}
		return events.Continue
	}, events.First)
	defer events.UnregisterListener(events.Input{}, listener)

	text := b.cmd("ask", "hound about bones")
	assert.Contains(t, text, "thanks friend", "conversation subjects reach onAsk")
	assert.Contains(t, text, "psst a secret", "a private reply is not refused")
	assert.NotContains(t, text, "not available")
	require.NotEmpty(t, tagged)
	for _, in := range tagged {
		assert.True(t, in.MemberOrder.Scripted, in.InputText)
		assert.Equal(t, b.aria.UserId, in.MemberOrder.UserID)
	}

	text = b.cmd("ask", "hound about fighting")
	assert.Nil(t, hound.Character.Aggro, "a script still cannot make the member attack")
	assert.NotContains(t, text, "Members fight automatically", "a refused script action is silent")
}
