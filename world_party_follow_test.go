package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func partyWorldData(t *testing.T) {
	t.Helper()
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	dir, err := filepath.Abs("_datafiles/world/default")
	require.NoError(t, err)
	flat["FilePaths.DataFiles"] = dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	keywords.LoadAliases()
	rooms.LoadBiomeDataFiles()
}

func TestQueuedPartyFollowRevalidatesBeforeExecuting(t *testing.T) {
	partyWorldData(t)
	for _, scenario := range []string{"valid", "valid-immediate", "off", "off-immediate", "off-on", "leave-rejoin", "promote", "manual-move", "prompt", "disband-recreate"} {
		t.Run(scenario, func(t *testing.T) {
			origin := &rooms.Room{RoomId: 934001, Title: "Origin", Exits: map[string]exit.RoomExit{"north": {RoomId: 934002}}}
			dest := &rooms.Room{RoomId: 934002, Title: "Destination", Exits: map[string]exit.RoomExit{}}
			rooms.SetTestRoom(origin)
			rooms.SetTestRoom(dest)
			t.Cleanup(func() { rooms.RemoveTestRoom(origin.RoomId); rooms.RemoveTestRoom(dest.RoomId) })
			leader := users.NewUserRecord(93401, 0)
			follower := users.NewUserRecord(93402, 0)
			for _, u := range []*users.UserRecord{leader, follower} {
				u.Password = "$2a$fixture"
				u.Character.RoomId = origin.RoomId
				u.Character.Health = 20
				u.Character.ActionPoints = 100
				users.SetTestUser(u)
				origin.AddPlayer(u.UserId)
				t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
			}
			p := parties.New(leader.UserId)
			require.NotNil(t, p)
			t.Cleanup(func() {
				if current := parties.Get(leader.UserId); current != nil {
					current.Disband()
				}
				if current := parties.Get(follower.UserId); current != nil {
					current.Disband()
				}
			})
			p.InvitePlayer(follower.UserId)
			p.AcceptInvite(follower.UserId)
			p.SetFollow(follower.UserId, true)
			var queued []events.Input
			id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
				input := e.(events.Input)
				if input.UserId == follower.UserId && input.PartyFollow != nil {
					queued = append(queued, input)
				}
				return events.Cancel
			}, events.First)
			_, err := usercommands.Go("north", leader, origin, 0)
			require.NoError(t, err)
			events.ProcessEvents()
			events.UnregisterListener(events.Input{}, id)
			require.Len(t, queued, 1)
			input := queued[0]
			w := &World{ignoreInput: map[int]uint64{}, userInputEventTracker: map[int]struct{}{follower.UserId: {}}, mobInputEventTracker: map[int]struct{}{}}
			input.ReadyTurn = util.GetTurnCount() + 1 // force the real delayed-input path
			assert.Equal(t, events.CancelAndRequeue, w.HandleInputEvents(input))
			switch strings.TrimSuffix(scenario, "-immediate") {
			case "off":
				p.SetFollow(follower.UserId, false)
			case "off-on":
				p.SetFollow(follower.UserId, false)
				p.SetFollow(follower.UserId, true)
			case "leave-rejoin":
				p.Leave(follower.UserId)
				p.InvitePlayer(follower.UserId)
				p.AcceptInvite(follower.UserId)
				p.SetFollow(follower.UserId, true)
			case "promote":
				p.Promote(follower.UserId)
			case "prompt":
				follower.StartPrompt("delete", "character")
			case "manual-move":
				follower.Character.RoomId = dest.RoomId
			case "disband-recreate":
				p.Disband()
				p = parties.New(leader.UserId)
				p.InvitePlayer(follower.UserId)
				p.AcceptInvite(follower.UserId)
				p.SetFollow(follower.UserId, true)
			}
			delete(w.userInputEventTracker, follower.UserId)
			util.IncrementTurnCount()
			if strings.HasSuffix(scenario, "-immediate") {
				input.ReadyTurn = 0
			} else {
				input.ReadyTurn = util.GetTurnCount()
			}
			result := w.HandleInputEvents(input)
			if strings.TrimSuffix(scenario, "-immediate") == "valid" {
				assert.Equal(t, events.Continue, result)
				assert.Equal(t, dest.RoomId, follower.Character.RoomId)
				assert.Equal(t, 90, follower.Character.ActionPoints)
			} else {
				assert.Equal(t, events.Cancel, result)
				assert.Equal(t, 100, follower.Character.ActionPoints, "a stale command must never execute")
				if scenario != "manual-move" {
					assert.Equal(t, origin.RoomId, follower.Character.RoomId)
				}
			}
		})
	}
}

func TestProductionHooksInstallAlliedSupportProvider(t *testing.T) {
	partyWorldData(t)
	previous := effecttargets.SetAlliedLeaders(nil)
	t.Cleanup(func() { effecttargets.SetAlliedLeaders(previous); events.ClearListeners() })
	hooks.RegisterListeners()
	spells.LoadSpellFiles()
	sp := spells.GetSpell("healall")
	require.NotNil(t, sp)
	original := sp.Scope
	sp.Scope = spells.ScopeAllied
	t.Cleanup(func() { sp.Scope = original })
	p := parties.New(93411)
	require.NotNil(t, p)
	t.Cleanup(p.Disband)
	p.InvitePlayer(93412)
	p.AcceptInvite(93412)
	for _, id := range []int{93411, 93412} {
		u := users.NewUserRecord(id, 0)
		u.Character.RoomId = 934101
		u.Character.Health = 20
		users.SetTestUser(u)
		p.SetSupport(id, true)
		t.Cleanup(func() { users.RemoveTestUser(id) })
	}
	info := effecttargets.Resolve(93411, 0, characters.SpellAggroInfo{SpellId: "healall"})
	assert.Contains(t, info.TargetUserIds, 93412, "actual hook registration must wire the consent provider")
	p.SetSupport(93412, false)
	info = effecttargets.Resolve(93411, 0, info)
	assert.NotContains(t, info.TargetUserIds, 93412)
}

func TestQueuedPartyAttackRevalidatesAtWorldExecution(t *testing.T) {
	partyWorldData(t)
	for _, immediate := range []bool{false, true} {
		for _, scenario := range []string{"valid", "off-on", "leave-rejoin", "promote", "separated", "gone", "withdrawn", "prompt", "different-battle"} {
			t.Run(fmt.Sprintf("%s/%t", scenario, immediate), func(t *testing.T) {
				t.Cleanup(parties.UseMemoryForTest())
				battle.Reset()
				t.Cleanup(battle.Reset)
				room := &rooms.Room{RoomId: 934101, Title: "Battle", Exits: map[string]exit.RoomExit{}}
				rooms.SetTestRoom(room)
				t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
				u := users.NewUserRecord(93412, 0)
				u.Password = "$2a$fixture"
				u.Character.Name = "Follower"
				u.Character.RoomId = room.RoomId
				u.Character.Health = 20
				users.SetTestUser(u)
				room.AddPlayer(u.UserId)
				t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
				p := parties.New(93411)
				p.InvitePlayer(u.UserId)
				require.True(t, p.AcceptInvite(u.UserId))
				p.SetAutoAttack(u.UserId, true)
				m := &mobs.Mob{InstanceId: 934121, Character: *characters.New()}
				m.Character.Name = "foe"
				m.Character.Health = 100
				m.Character.RoomId = room.RoomId
				mobs.SetTestInstance(m)
				room.AddMob(m.InstanceId)
				t.Cleanup(func() { mobs.RemoveTestInstance(m.InstanceId) })
				in := events.Input{UserId: u.UserId, InputText: fmt.Sprintf("attack #%d", m.InstanceId), PartyAttack: &events.PartyAttackOrder{LeaderUserId: 93411, OriginRoomId: room.RoomId, ConsentToken: p.AutoAttackToken(u.UserId), TargetMobInstanceId: m.InstanceId}}
				w := &World{ignoreInput: map[int]uint64{}, userInputEventTracker: map[int]struct{}{u.UserId: {}}, mobInputEventTracker: map[int]struct{}{}}
				in.ReadyTurn = util.GetTurnCount() + 1
				require.Equal(t, events.CancelAndRequeue, w.HandleInputEvents(in))
				switch scenario {
				case "off-on":
					p.SetAutoAttack(u.UserId, false)
					p.SetAutoAttack(u.UserId, true)
				case "leave-rejoin":
					p.Leave(u.UserId)
					p.InvitePlayer(u.UserId)
					p.AcceptInvite(u.UserId)
					p.SetAutoAttack(u.UserId, true)
				case "promote":
					p.Promote(u.UserId)
				case "separated":
					u.Character.RoomId++
				case "gone":
					mobs.RemoveTestInstance(m.InstanceId)
				case "withdrawn":
					m.Character.CombatWithdrawn = true
				case "prompt":
					u.StartPrompt("delete", "character")
				case "different-battle":
					battle.Begin(u.UserId, room.RoomId, 1, "other", []int{99999})
				}
				delete(w.userInputEventTracker, u.UserId)
				util.IncrementTurnCount()
				in.ReadyTurn = util.GetTurnCount()
				if immediate {
					in.ReadyTurn = 0
				}
				result := w.HandleInputEvents(in)
				if scenario == "valid" {
					assert.Equal(t, events.Continue, result)
					require.NotNil(t, u.Character.Aggro)
					assert.Equal(t, m.InstanceId, u.Character.Aggro.MobInstanceId)
				} else {
					assert.Equal(t, events.Cancel, result)
					assert.Nil(t, u.Character.Aggro)
				}
			})
		}
	}
}
