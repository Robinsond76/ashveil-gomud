package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alliedBrawl(t *testing.T) (*brawl, *users.UserRecord, *mobs.Mob) {
	t.Helper()
	b := newBrawl(t)
	t.Cleanup(parties.UseMemoryForTest())
	ally := users.NewUserRecord(8, 0)
	ally.Username = "borin"
	ally.Password = "$2a$test"
	ally.Character.Name = "Borin"
	ally.Character.RoomId = b.road.RoomId
	ally.Character.RaceId = 1
	ally.Character.Level = 3
	ally.Character.Validate()
	ally.Character.Health = ally.Character.HealthMax.Value
	users.SetTestUser(ally)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	_, err := usercommands.TryCommand("company", "summon tamsin reed", 8, events.CmdSkipScripts)
	require.NoError(t, err)
	events.ProcessEvents()
	instance, ok := module.instance(8, 1)
	require.True(t, ok)
	companion := mobs.GetInstance(instance)
	require.NotNil(t, companion)
	p := parties.New(7)
	require.NotNil(t, p)
	p.InvitePlayer(8)
	p.AcceptInvite(8)
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		in := e.(events.Input)
		if in.PartyAttack != nil && usercommands.ValidPartyFollow(in) {
			cmd, rest, _ := strings.Cut(in.InputText, " ")
			_, err := usercommands.TryCommand(cmd, rest, in.UserId, events.CmdSkipScripts)
			require.NoError(t, err)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })
	return b, ally, companion
}

func TestAlliedCompaniesSharedBattleAuthorityAndReward(t *testing.T) {
	b, ally, companion := alliedBrawl(t)
	p := parties.Get(7)
	p.SetAutoAttack(8, true)
	// Both companies may place their own first companion in the same grid cell.
	b.cmd("formation", "move tamsin 1 1")
	_, err := usercommands.TryCommand("formation", "move tamsin 1 1", 8, events.CmdSkipScripts)
	require.NoError(t, err)
	before7, _ := module.registry.Get(7)
	before8, _ := module.registry.Get(8)
	assert.Equal(t, before7.Formation, before8.Formation)
	for _, ids := range b.bandits {
		for _, id := range ids {
			m := mobs.GetInstance(id)
			m.Character.HealthMax.Value = 1000
			m.Character.Health = 1000
		}
	}
	b.toughen()
	ally.Character.HealthMax.Value = 10000
	ally.Character.Health = 10000
	companion.Character.HealthMax.Value = 10000
	companion.Character.Health = 10000
	b.aimAt("bandit captain")
	b.fight()
	battle7, ok := battle.Current(7)
	require.True(t, ok)
	battle8, ok := battle.Current(8)
	require.True(t, ok)
	assert.NotEqual(t, battle7.FightID, battle8.FightID)
	for id := range battle7.Enemies {
		assert.True(t, battle8.Has(id))
	}
	// Independent focus and assets: one player's order does not alter the ally.
	b.cmd("company", "tactics focus strongest")
	after, _ := battle.Current(8)
	assert.Equal(t, battle8.Focus, after.Focus)
	owner, key, attached := domain.LeaderAndKeyForInstance(companion.InstanceId)
	assert.True(t, attached)
	assert.Equal(t, 8, owner)
	assert.Equal(t, domain.CompanionMemberKey(1), key)
	foe := mobs.GetInstance(b.bandits["bandit captain"][0])
	// The real combat entry points maintain contribution; ensure both companies
	// have hit the specific enemy, not merely another member of its group.
	for i := 0; i < 30 && (foe.Character.PlayerDamage[7] <= 0 || foe.Character.PlayerDamage[8] <= 0); i++ {
		b.aria.Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
		ally.Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
		companion.Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
		b.fight()
	}
	require.Positive(t, foe.Character.PlayerDamage[7])
	require.Positive(t, foe.Character.PlayerDamage[8])
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	xp7, xp8, xpc := b.aria.Character.Experience, ally.Character.Experience, companion.Character.Experience
	foe.Character.Level = 10
	foe.Character.TNLScale = 1
	foe.Character.Health = 0
	_, err = mobcommands.Suicide("quiet", foe, b.road)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Greater(t, b.aria.Character.Experience, xp7)
	assert.Greater(t, ally.Character.Experience, xp8)
	assert.Greater(t, companion.Character.Experience, xpc)
	require.NotEmpty(t, b.road.Corpses)
	claim := b.road.Corpses[len(b.road.Corpses)-1].ClaimUserId
	assert.Contains(t, []int{7, 8}, claim)
	xp7 = b.aria.Character.Experience
	_, err = mobcommands.Suicide("quiet", foe, b.road)
	require.NoError(t, err)
	assert.Equal(t, xp7, b.aria.Character.Experience)
	// One company's ordered retreat never changes the ally's current battle.
	b.cmd("retreat", "east")
	if ally.Character.Aggro != nil {
		assert.Nil(t, ally.Character.Aggro.RetreatInfo)
	}
	still, ok := battle.Current(8)
	require.True(t, ok)
	assert.Equal(t, battle8.FightID, still.FightID)
}

func TestAllianceDoesNotJoinSeparateBattleWithoutConsent(t *testing.T) {
	b, ally, _ := alliedBrawl(t)
	foe := b.bandits["bandit captain"][0]
	b.cmd("attack", fmt.Sprintf("#%d", foe))
	assert.Nil(t, ally.Character.Aggro, "accepted membership defaults autoattack off")
	assert.Zero(t, mobs.GetInstance(foe).Character.PlayerDamage[8])
}

func TestAlliedFinalEnemyPaysAfterCombatClosesBattle(t *testing.T) {
	b, ally, _ := alliedBrawl(t)
	p := parties.Get(7)
	p.SetAutoAttack(8, true)
	b.toughen()
	ally.Character.HealthMax.Value = 10000
	ally.Character.Health = 10000
	b.aimAt("bandit captain")
	b.fight()
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	finalID := b.bandits["bandit captain"][0]
	for _, m := range b.livingBandits() {
		if m.InstanceId != finalID {
			_, err := mobcommands.Suicide("vanish", m, b.road)
			require.NoError(t, err)
			continue
		}
		m.Character.Health = 1
		m.Character.Level = 10
		m.Character.TNLScale = 1
		m.Character.TrackPlayerDamage(7, 1)
		m.Character.TrackPlayerDamage(8, 1)
	}
	b.road.Corpses = nil
	battle.Begin(7, b.road.RoomId, b.round, "final", []int{finalID})
	battle.Begin(8, b.road.RoomId, b.round, "final", []int{finalID})
	b.aria.Character.SetAggro(0, finalID, characters.DefaultAttack)
	ally.Character.SetAggro(0, finalID, characters.DefaultAttack)
	before7, before8 := b.aria.Character.Experience, ally.Character.Experience
	for i := 0; i < 50; i++ {
		b.fight()
		if _, active := battle.Current(7); !active {
			break
		}
	}
	_, active := battle.Current(7)
	require.False(t, active)
	assert.Greater(t, b.aria.Character.Experience, before7)
	assert.Greater(t, ally.Character.Experience, before8)
	require.Len(t, b.road.Corpses, 1)
	assert.Contains(t, []int{7, 8}, b.road.Corpses[0].ClaimUserId)
}

func TestAlliedSupportRespectsIntegratedBattleBoundary(t *testing.T) {
	for _, change := range []string{"shared", "revoked", "separate", "idle"} {
		t.Run(change, func(t *testing.T) {
			b, ally, companion := alliedBrawl(t)
			p := parties.Get(7)
			p.SetAutoAttack(8, true)
			p.SetSupport(7, true)
			p.SetSupport(8, true)
			previous := effecttargets.SetAlliedLeaders(parties.AlliedLeaders)
			t.Cleanup(func() { effecttargets.SetAlliedLeaders(previous) })
			b.toughen()
			for _, m := range b.livingBandits() {
				m.Character.Health = 1000
				m.Character.HealthMax.Value = 1000
			}
			ally.Character.Health = 1000
			ally.Character.HealthMax.Value = 1000
			companion.Character.Health = 1000
			companion.Character.HealthMax.Value = 1000
			sp := spells.GetSpell("healall")
			original := sp.Scope
			sp.Scope = spells.ScopeAllied
			t.Cleanup(func() { sp.Scope = original })
			info := friendlyCast(t, b)
			require.Contains(t, info.TargetUserIds, 8)
			require.Contains(t, info.TargetMobInstanceIds, companion.InstanceId)
			b.aimAt("bandit captain")
			b.fight()
			require.False(t, effecttargets.OtherBattle(7, 0, 8, 0), "consenting companies share the same enemies")
			ally.Character.Health = 1
			companion.Character.Health = 1
			switch change {
			case "revoked":
				p.SetSupport(8, false)
			case "separate":
				battle.Begin(8, b.road.RoomId, b.round, "other", []int{999999})
			case "idle":
				battle.End(7)
			}
			_, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
			require.NoError(t, err)
			if change == "shared" {
				assert.Greater(t, ally.Character.Health, 1)
				assert.Greater(t, companion.Character.Health, 1)
			} else {
				assert.Equal(t, 1, ally.Character.Health)
				assert.Equal(t, 1, companion.Character.Health)
			}
		})
	}
}
