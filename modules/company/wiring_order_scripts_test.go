package company

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// houndScript answers through the real onAsk entry point with replies outside
// ask's order list, tries hostile commands (one by alias) about a fight, drops
// its gear when asked about it, and leaves other subjects unanswered.
const houndScript = `function onAsk(mob, room, eventDetails) {
    if (eventDetails.askText.indexOf("nothing") >= 0) {
        return false;
    }
    if (eventDetails.askText.indexOf("fighting") >= 0) {
        mob.Command("attack Aria");
        mob.Command("a Aria");
        return true;
    }
    if (eventDetails.askText.indexOf("gear") >= 0) {
        mob.Command("drop all");
        return true;
    }
    mob.Command("say thanks friend");
    mob.Command("saytoonly @" + String(eventDetails.sourceId) + " psst a secret");
    mob.Command("bow");
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

// scriptOrders records the follower orders queued while a test runs.
func scriptOrders(t *testing.T) *[]events.Input {
	var orders []events.Input
	freshEvents(t)
	listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in := e.(events.Input); in.MemberOrder != nil {
			orders = append(orders, in)
		}
		return events.Continue
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, listener) })
	return &orders
}

func TestScriptRepliesOfAnotherPlayersFollowerAnswerToItsOwner(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(8)
	orders := scriptOrders(t)

	text := b.cmd("ask", "hound about bones")
	assert.Contains(t, text, "thanks friend", "the follower's own reply runs")
	assert.NotContains(t, text, "no longer command", "Aria is not told she lost a member she never had")
	assert.NotContains(t, text, "not available")
	require.NotEmpty(t, *orders)
	for _, in := range *orders {
		assert.True(t, in.MemberOrder.Scripted, in.InputText)
		assert.Equal(t, 8, in.MemberOrder.UserID, "a follower's script answers to its owner, not Aria")
	}

	text = b.cmd("ask", "hound about fighting")
	assert.Nil(t, hound.Character.Aggro, "Aria cannot make another player's follower attack through its script")
	assert.NotContains(t, text, "Members fight automatically", "a refused script action is silent")
	assert.Equal(t, 8, hound.Character.Charmed.UserId)
}

func TestScriptRepliesOfOwnFollowerUseFullConversation(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(b.aria.UserId)
	orders := scriptOrders(t)

	text := b.cmd("ask", "hound about bones")
	assert.Contains(t, text, "thanks friend", "conversation subjects reach onAsk")
	assert.Contains(t, text, "psst a secret", "a private reply is not refused")
	assert.Contains(t, text, "bows gracefully", "an emote shortcut is not refused")
	assert.NotContains(t, text, "not available")
	require.NotEmpty(t, *orders)
	for _, in := range *orders {
		assert.True(t, in.MemberOrder.Scripted, in.InputText)
		assert.Equal(t, b.aria.UserId, in.MemberOrder.UserID)
	}

	text = b.cmd("ask", "hound about fighting")
	assert.Nil(t, hound.Character.Aggro, "a script cannot make the member attack, by name or alias")
	assert.NotContains(t, text, "Members fight automatically", "a refused script action is silent")

	text = b.cmd("ask", "hound about nothing")
	assert.Contains(t, text, "help ask", "an unanswered subject still points the owner to the orders")

	assert.Contains(t, b.cmd("ask", "hound to attack Aria"), "Members fight automatically", "a typed attack order is still refused aloud")
	assert.Nil(t, hound.Character.Aggro)
}

func TestScriptedGearChangeLockedInOwnersBattle(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(b.aria.UserId)
	sword := items.New(10010)
	require.NotZero(t, sword.ItemId)
	require.True(t, hound.Character.StoreItem(sword))

	battle.Begin(b.aria.UserId, b.road.RoomId, 1, "test", []int{99})
	text := b.cmd("ask", "hound about gear")
	assert.Len(t, hound.Character.Items, 1, "a script cannot change a member's load in battle")
	assert.NotContains(t, text, actionpolicy.BattleUnderWay, "a refused script action is silent")

	battle.End(b.aria.UserId)
	b.cmd("ask", "hound about gear")
	assert.Empty(t, hound.Character.Items, "outside battle the script's drop runs")
}

func TestScriptedOrderSurvivesTheMemberMoving(t *testing.T) {
	b := newBrawl(t)
	hound := b.scriptedHound(b.aria.UserId)
	orders := scriptOrders(t)
	restore := events.WithRequester(b.aria.UserId)
	scripting.GetActor(0, hound.InstanceId).Command("say later")
	restore()
	events.ProcessEvents()
	require.Len(t, *orders, 1)
	order := *(*orders)[0].MemberOrder

	b.road.RemoveMob(hound.InstanceId)
	hound.Character.RoomId++
	assert.Empty(t, actionpolicy.Member(order, hound, "say"), "a delayed script reply still runs after the follower moves")
	assert.NotEmpty(t, actionpolicy.Member(order, hound, "attack"))
	hound.Character.Charm(8, -1, "")
	assert.NotEmpty(t, actionpolicy.Member(order, hound, "say"), "a new charm voids the old script order")
}
