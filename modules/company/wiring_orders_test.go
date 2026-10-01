package company

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemberOrdersUseAutomaticBattlePolicy(t *testing.T) {
	b := newBrawl(t)
	m := b.companion(1)
	round := util.GetRoundCount()
	for _, cmd := range []string{"attack bandit", "throw sword", "cast mm"} {
		assert.Contains(t, b.cmd("ask", "Tamsin to "+cmd), "Members fight automatically")
		assert.Nil(t, m.Character.Aggro)
	}
	assert.Contains(t, b.cmd("ask", "Tamsin to say ready"), "ready")
	assert.NotContains(t, b.cmd("ask", "Tamsin to look"), "not available")
	battle.Begin(b.aria.UserId, b.road.RoomId, 1, "test", []int{90001})
	for _, cmd := range []string{"equip sword", "remove all", "eat bread", "drink water", "get sword", "drop sword", "give sword Aria"} {
		assert.Contains(t, b.cmd("ask", "Tamsin to "+cmd), actionpolicy.BattleUnderWay)
	}
	for _, cmd := range []string{"aid Tamsin"} {
		c, rest, _ := strings.Cut(cmd, " ")
		assert.Contains(t, b.cmd(c, rest), actionpolicy.BattleUnderWay)
	}
	assert.Contains(t, b.cmd("ask", "Tamsin say still here"), "still here")
	assert.Equal(t, round, util.GetRoundCount(), "orders do not advance shared time")
}

func TestQueuedMemberOrdersRevalidate(t *testing.T) {
	for _, change := range []string{"battle", "charm", "charm back", "expire", "dismiss", "move", "death", "leader death", "leader logout"} {
		t.Run(change, func(t *testing.T) {
			b := newBrawl(t)
			m := b.companion(1)
			var queued *events.Input
			listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
				in := e.(events.Input)
				if in.MemberOrder != nil {
					copy := in
					queued = &copy
					return events.Cancel
				}
				return events.Continue
			}, events.First)
			b.cmd("ask", "Tamsin say requested")
			events.UnregisterListener(events.Input{}, listener)
			require.NotNil(t, queued)
			require.NotNil(t, queued.MemberOrder)
			switch change {
			case "battle":
				battle.Begin(7, b.road.RoomId, 1, "new", []int{99})
				queued.InputText = "remove all"
			case "charm":
				m.Character.Charm(8, -1, "")
			case "charm back":
				m.Character.Charm(8, -1, "")
				m.Character.CharmAsCompanion(7, -2, "")
			case "expire":
				m.Character.Charmed.Expire()
			case "dismiss":
				b.cmd("company", "dismiss Tamsin")
			case "move":
				b.road.RemoveMob(m.InstanceId)
				m.Character.RoomId++
			case "death":
				m.Character.Health = 0
			case "leader death":
				b.aria.Character.Health = 0
			case "leader logout":
				users.ResetActiveUsers()
			}
			*b.messages = nil
			cmd, rest, _ := strings.Cut(queued.InputText, " ")
			_, err := mobcommands.TryCommand(cmd, rest, m.InstanceId, queued.MemberOrder)
			require.NoError(t, err)
			events.ProcessEvents()
			assert.NotContains(t, strings.Join(*b.messages, "\n"), "says: requested")
			if change != "leader logout" {
				assert.NotEmpty(t, *b.messages, "a refused order explains why")
			}
		})
	}
}

func TestScriptedFollowerOrdersCarryRequester(t *testing.T) {
	b := newBrawl(t)
	m := b.companion(1)
	var orders []events.Input
	listener := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		in := e.(events.Input)
		if in.MemberOrder != nil {
			orders = append(orders, in)
			return events.Cancel
		}
		return events.Continue
	}, events.First)
	defer events.UnregisterListener(events.Input{}, listener)
	restore := events.WithRequester(7)
	scripting.GetActor(0, m.InstanceId).Command("say ready;attack Aria")
	restore()
	events.ProcessEvents()
	require.Len(t, orders, 2)
	assert.Equal(t, 7, orders[0].MemberOrder.UserID)
	assert.Equal(t, "companion:1", orders[0].MemberOrder.MemberKey)
	_, err := mobcommands.TryCommand("attack", "Aria", m.InstanceId, orders[1].MemberOrder)
	require.NoError(t, err)
	assert.Nil(t, m.Character.Aggro, "semicolon cannot smuggle an attack")
	battle.Begin(7, b.road.RoomId, 1, "test", []int{99})
	worn := m.Character.Equipment
	_, err = mobcommands.TryCommand("remove", "all", m.InstanceId)
	require.NoError(t, err)
	assert.Equal(t, worn, m.Character.Equipment, "autonomous follower gear is also locked")
}

func TestLegacySkillsDoNotOverwriteBattleActions(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.SetAggro(0, 123, characters.DefaultAttack)
	before := b.aria.Character.Aggro
	_, err := usercommands.Aid("someone", b.aria, b.road, 0)
	require.NoError(t, err)
	assert.Same(t, before, b.aria.Character.Aggro)
}

func TestLegalOutOfBattleGearAndTemporaryFollower(t *testing.T) {
	b := newBrawl(t)
	m := b.companion(1)
	worn := len(m.Character.Equipment.GetAllItems())
	require.Positive(t, worn)
	original := m.Character.Equipment
	b.cmd("ask", "Tamsin remove all")
	assert.Len(t, m.Character.Equipment.GetAllItems(), worn, "shared members use company commands")
	b.cmd("ask", "Tamsin equip all")
	assert.Len(t, m.Character.Equipment.GetAllItems(), worn)
	for _, slot := range characters.AllSlots() {
		if original.Get(slot).ItemId > 0 {
			b.cmd("company", "remove #1 "+string(slot))
		}
	}
	assert.Empty(t, m.Character.Equipment.GetAllItems())
	for _, itm := range original.GetAllItems() {
		b.cmd("company", "equip #1 "+itm.ShorthandId())
	}
	assert.Len(t, m.Character.Equipment.GetAllItems(), worn)

	follower := mobs.NewMobById(9101, b.road.RoomId)
	require.NotNil(t, follower)
	follower.Character.Name = "Follower"
	follower.Character.Charm(7, -1, "")
	b.road.AddMob(follower.InstanceId)
	assert.Contains(t, b.cmd("ask", "Follower say ready"), "ready")
	assert.Contains(t, b.cmd("ask", "Follower attack Aria"), "Members fight automatically")
	assert.Nil(t, follower.Character.Aggro)
	assert.Nil(t, m.Character.Aggro)
}

func TestMemberPolicyRunsBeforeScriptsAndExpandsAliases(t *testing.T) {
	b := newBrawl(t)
	path := scripting.UserScriptPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(`function onCommand(cmd, rest, user, room) { user.SendText("INTERCEPTED"); return true; }`), 0600))
	scripting.ClearUserVM()
	t.Cleanup(func() { scripting.ClearUserVM() })
	battle.Begin(7, b.road.RoomId, 1, "test", []int{99})
	b.aria.Aliases = map[string]string{"dress": "wear sword", "fightthem": "a bandit", "see": "look"}
	for _, cmd := range []string{"dress", "wear", "fightthem", "aid"} {
		*b.messages = nil
		_, err := usercommands.TryCommand(cmd, "", 7, 0)
		require.NoError(t, err)
		events.ProcessEvents()
		text := strings.Join(*b.messages, "\n")
		assert.Contains(t, text, actionpolicy.BattleUnderWay, cmd)
		assert.NotContains(t, text, "INTERCEPTED", cmd)
	}
	*b.messages = nil
	_, err := usercommands.TryCommand("see", "", 7, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*b.messages, "\n"), "INTERCEPTED", "observation reaches scripts")
}

func TestLegacySkillExecutionRechecksBattleAndOwnership(t *testing.T) {
	b := newBrawl(t)
	aid := characters.SpellAggroInfo{SpellId: "aidskill", TargetUserIds: []int{7}}
	battle.Begin(7, b.road.RoomId, 1, "new battle", []int{99})
	handled, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, aid)
	require.NoError(t, err)
	assert.True(t, handled, "aid does not finish after a battle starts")
	battle.End(7)
	follower := mobs.NewMobById(9101, b.road.RoomId)
	require.NotNil(t, follower)
	follower.Character.Name = "Follower"
	follower.Character.Charm(7, -1, "")
	b.road.AddMob(follower.InstanceId)
	battle.Begin(7, b.road.RoomId, 2, "own battle", []int{99})
	assert.Contains(t, b.cmd("ask", "Follower remove all"), actionpolicy.BattleUnderWay)
}
