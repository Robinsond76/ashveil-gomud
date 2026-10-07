package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
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

// Phase 62: an attack round's strikes ride its event, and the fight leader's
// roll log keeps them for `why` until the next fight begins.
func TestAttackStrikesRideTheEventAndTheRollLog(t *testing.T) {
	stream := combatstream.New()
	t.Cleanup(combatstream.UseForTest(stream))
	log := combatstream.DefaultRollLog()
	leader := combatstream.Ref{UserId: 7, Name: "Aria", LeaderUserId: 7, MemberKey: "leader"}
	foe := combatstream.Ref{MobInstanceId: 21, Name: "bandit captain"}
	t.Cleanup(func() { log.Clear(7) })
	log.Clear(7)

	stream.Open(5, 100, "p", leader, nil, []combatstream.Ref{foe})
	strikes := []combatstream.Strike{{Chance: 55, Base: 55, Roll: 10, Hit: true, Raw: 7, Armor: 3, Reduced: 2, Damage: 5}}
	result := combat.AttackResult{Hit: true, DamageToTarget: 5, Strikes: strikes}
	emitAttack(leader, foe, 100, &characters.Character{}, result)
	emitAttack(foe, leader, 100, &characters.Character{}, combat.AttackResult{Strikes: []combatstream.Strike{{Chance: 40, Base: 40, Roll: 90}}})
	emitAttack(foe, leader, 100, &characters.Character{}, combat.AttackResult{}) // no strikes recorded: nothing to explain

	rolls := log.Recent(7, 0)
	require.Len(t, rolls, 2, "only rounds with strikes are kept")
	assert.False(t, rolls[0].Ours, "the foe's round, newest first")
	assert.True(t, rolls[1].Ours)
	assert.Equal(t, strikes, rolls[1].Strikes, "the event carries the engine's strikes unchanged")
	assert.Equal(t, 5, rolls[1].Damage)

	// A new fight for the same leader starts the log afresh.
	log.Clear(7)
	assert.Empty(t, log.Recent(7, 0))
}

func TestSigilNoteNamesWhatHeldOverTheBattle(t *testing.T) {
	assert.Equal(t, "", sigilNote(999999), "no battle, no sigil")
}

// Review: `why` never explains a round against a foe the leader can't make
// out (its armor and chances would give it away), and only keeps rounds of
// the leader's own company.
func TestRollLogSkipsUnseenFoesAndOthersRounds(t *testing.T) {
	stream := combatstream.New()
	t.Cleanup(combatstream.UseForTest(stream))
	log := combatstream.DefaultRollLog()
	const leaderId, hiddenBuff = 8, 99621
	t.Cleanup(func() { log.Clear(leaderId) })
	log.Clear(leaderId)

	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: hiddenBuff, Name: "test hiding", Flags: []string{"hidden"}, RoundInterval: 1, TriggerCount: 100})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(hiddenBuff) })
	lurker := &mobs.Mob{InstanceId: 8621}
	lurker.Character.Health = 10
	lurker.Character.Buffs = buffs.New()
	lurker.Character.Buffs.AddBuff(hiddenBuff, true)
	mobs.SetTestInstance(lurker)
	t.Cleanup(func() { mobs.RemoveTestInstance(lurker.InstanceId) })
	require.True(t, lurker.Character.HasBuffFlag("hidden"))

	leaderUser := &users.UserRecord{UserId: leaderId, Character: &characters.Character{Name: "Aria"}}
	users.SetTestUser(leaderUser)
	t.Cleanup(func() { users.RemoveTestUser(leaderId) })

	leader := combatstream.Ref{UserId: leaderId, Name: "Aria", LeaderUserId: leaderId, MemberKey: "leader"}
	seen := combatstream.Ref{MobInstanceId: 8622, Name: "bandit"}
	hidden := combatstream.Ref{MobInstanceId: lurker.InstanceId, Name: "lurker"}
	stream.Open(5, 0, "p", leader, nil, []combatstream.Ref{seen, hidden})
	strikes := []combatstream.Strike{{Chance: 50, Base: 50, Roll: 80}}
	emitAttack(hidden, leader, 0, &characters.Character{}, combat.AttackResult{Strikes: strikes})
	emitAttack(leader, hidden, 0, &characters.Character{}, combat.AttackResult{Strikes: strikes})
	emitAttack(seen, hidden, 0, &characters.Character{}, combat.AttackResult{Strikes: strikes}) // not the company's round
	emitAttack(seen, leader, 0, &characters.Character{}, combat.AttackResult{Strikes: strikes})

	rolls := log.Recent(leaderId, 0)
	require.Len(t, rolls, 1, "only the seen foe's blow at the company is kept")
	assert.Equal(t, "bandit", rolls[0].Source.Name)
}
