package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttackEvents(t *testing.T) {
	src := combatstream.Ref{UserId: 7, Name: "Aria"}
	tgt := combatstream.Ref{MobInstanceId: 21, Name: "bandit captain"}
	unarmed := &characters.Character{}

	cases := []struct {
		name    string
		result  combat.AttackResult
		outcome string
		crit    bool
	}{
		{"a miss", combat.AttackResult{}, combatstream.OutcomeMiss, false},
		{"a hit", combat.AttackResult{Hit: true, DamageToTarget: 4}, combatstream.OutcomeHit, false},
		{"a critical hit", combat.AttackResult{Hit: true, Crit: true, DamageToTarget: 9}, combatstream.OutcomeCrit, true},
		{"a backstab flag with no hit is a miss", combat.AttackResult{Crit: true}, combatstream.OutcomeMiss, false},
		{"a blocked strike is a miss with its defense", combat.AttackResult{Defenses: []string{combat.DefenseBlocked}}, combatstream.OutcomeMiss, false},
	}
	for _, c := range cases {
		got := attackEvents(src, tgt, 100, unarmed, c.result)
		require.Len(t, got, 1, c.name)
		e := got[0]
		assert.Equal(t, combatstream.Attack, e.Kind, c.name)
		assert.Equal(t, c.outcome, e.Outcome, c.name)
		assert.Equal(t, c.crit, e.Crit, c.name)
		assert.Equal(t, c.result.DamageToTarget, e.Damage, c.name)
		assert.Equal(t, src, e.Source, c.name)
		assert.Equal(t, tgt, e.Target, c.name)
		assert.Equal(t, 100, e.RoomId, c.name)
		assert.Equal(t, "", e.WeaponType, c.name+": unarmed")
		assert.Equal(t, c.result.Defenses, e.Defenses, c.name)
	}

	// Buffs the blow applies follow the attack, on the target, then on the
	// attacker.
	got := attackEvents(src, tgt, 100, unarmed, combat.AttackResult{Hit: true, BuffTarget: []int{901}, BuffSource: []int{902}})
	require.Len(t, got, 3)
	assert.Equal(t, combatstream.StatusApplied, got[1].Kind)
	assert.Equal(t, tgt, got[1].Target)
	assert.Equal(t, 901, got[1].BuffId)
	assert.Equal(t, src, got[2].Target)
	assert.Equal(t, 902, got[2].BuffId)

	// Phase 30a: a stab's two stacks of bleeding are one status on the blow.
	got = attackEvents(src, tgt, 100, unarmed, combat.AttackResult{Hit: true, Crit: true, BuffTarget: []int{1100, 1100}})
	require.Len(t, got, 2)
	assert.Equal(t, 1100, got[1].BuffId)
}

func TestSpellDeltaEvents(t *testing.T) {
	caster := combatstream.Ref{UserId: 7, Name: "Aria"}
	refs := []combatstream.Ref{
		{MobInstanceId: 21, Name: "bandit captain"},
		{UserId: 7, Name: "Aria"},
		{MobInstanceId: 22, Name: "bandit slinger"},
		{MobInstanceId: 23, Name: "gone"},
	}
	got := spellDeltaEvents(caster, "spark", 100, refs, []int{10, 5, 8, 4}, []int{6, 9, 8, 0}, []bool{true, true, true, false})
	require.Len(t, got, 2, "no change and a gone target report nothing")
	assert.Equal(t, combatstream.SpellHit, got[0].Kind)
	assert.Equal(t, 4, got[0].Damage)
	assert.Equal(t, refs[0], got[0].Target)
	assert.Equal(t, "spark", got[0].SpellId)
	assert.Equal(t, combatstream.Heal, got[1].Kind)
	assert.Equal(t, 4, got[1].Amount)
	assert.Equal(t, refs[1], got[1].Target)
	assert.Equal(t, caster, got[1].Source)
}

func TestWeaponType(t *testing.T) {
	assert.Equal(t, "", weaponType(nil))
	assert.Equal(t, "", weaponType(&characters.Character{}))
	c := &characters.Character{}
	c.Equipment.Weapon = items.Item{ItemId: 999999} // no spec loaded
	assert.Equal(t, "", weaponType(c))
}

func TestGroupTurnsToward(t *testing.T) {
	assert.Equal(t, "They turn toward", groupTurnsToward(mobparty.Party{}))
	a := engagementMob(t, 8601, 5, 1)
	a.Character.Name = "ruffian"
	assert.Equal(t, `The <ansi fg="mobname">ruffian</ansi> turns toward`, groupTurnsToward(mobparty.Party{Members: []int{8601}}))
	b := engagementMob(t, 8602, 5, 1)
	b.Character.Name = "big rat"
	assert.Equal(t, `The <ansi fg="mobname">ruffian</ansi> and the others turn toward`, groupTurnsToward(mobparty.Party{Members: []int{8601, 8602}}))
}

func TestTurnsToward(t *testing.T) {
	assert.Equal(t, `The <ansi fg="mobname">bandit cutthroat</ansi> turns toward <ansi fg="username">Aria</ansi>.`,
		turnsToward(mobTag("bandit cutthroat"), userTag("Aria")))
	assert.Equal(t, `You turn toward the <ansi fg="mobname">bandit captain</ansi>.`, turnsToward(`You`, mobTag("bandit captain")))
	assert.Equal(t, `<ansi fg="mobname">Garrick Vane</ansi> turns toward the <ansi fg="mobname">bandit bruiser</ansi>.`,
		turnsToward(mobTag("Garrick Vane"), mobTag("bandit bruiser")))
	// Review fix: a player's lowercase name takes no article.
	assert.Equal(t, `The <ansi fg="mobname">bandit captain</ansi> turns toward <ansi fg="username">bob</ansi>.`,
		turnsToward(mobTag("bandit captain"), userTag("bob")))
}
