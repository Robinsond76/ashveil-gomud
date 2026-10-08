package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

// 33d review: a last enemy finished by a damage-over-time tick died through
// MobRoundTick's queued suicide, after the same round's battle pass had
// already ended the battle, so its killer was paid nothing.
// The same holds for a death found when a buff triggers on application.
func TestRoundTickDeathAfterBattlePassStillPaysXP(t *testing.T) {
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 99331, Name: "review venom", TriggerNow: true, RoundInterval: 1, TriggerCount: 1})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(99331) })
	for name, dies := range map[string]func(b *brawl, captain *mobs.Mob){
		"round tick": func(b *brawl, captain *mobs.Mob) {
			hooks.MobRoundTick(events.NewRound{RoundNumber: b.round})
		},
		"buff applied": func(b *brawl, captain *mobs.Mob) {
			hooks.ApplyBuffs(events.Buff{MobInstanceId: captain.InstanceId, BuffId: 99331})
		},
	} {
		t.Run(name, func(t *testing.T) { roundTickDeath(t, dies) })
	}
}

func roundTickDeath(t *testing.T, dies func(b *brawl, captain *mobs.Mob)) {
	b := newBrawl(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	b.toughen()
	b.aimAt("bandit captain")
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	hardTo(&captain.Character, 1000)
	for i := 0; i < 30 && captain.Character.PlayerDamage[7] <= 0; i++ {
		b.toughen()
		b.fight()
	}
	require.Positive(t, captain.Character.PlayerDamage[7], "Aria has hit the captain")
	for _, m := range b.livingBandits() {
		if m.InstanceId != captain.InstanceId {
			mobcommands.Suicide("vanish", m, rooms.LoadRoom(m.Character.RoomId))
		}
	}
	events.ProcessEvents()
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	xp := b.aria.Character.Experience

	// One game round, in its listeners' order: the round tick finds the
	// captain dead (a poison tick) and queues its death; combat's battle
	// pass then ends the battle; the queued death runs last.
	captain.Character.Health = 0
	b.round++
	dies(b, captain)
	hooks.DoCombat(events.NewRound{RoundNumber: b.round})
	_, inBattle = battle.Current(7)
	require.False(t, inBattle, "the battle pass ended the battle before the death ran")
	events.ProcessEvents()

	require.Nil(t, mobs.GetInstance(captain.InstanceId), "the captain's death ran")
	require.Greater(t, b.aria.Character.Experience, xp, "the killer is paid")
}
