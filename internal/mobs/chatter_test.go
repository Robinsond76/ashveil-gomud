package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// queuedCommands runs fn and returns the commands it queued for the mob.
func queuedCommands(t *testing.T, instanceId int, fn func()) []string {
	t.Helper()
	events.ClearQueueForTest()
	var got []string
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId == instanceId {
			got = append(got, in.InputText)
		}
		return events.Continue
	})
	defer events.UnregisterListener(events.Input{}, id)
	fn()
	events.ProcessEvents()
	return got
}

func useChatterLimits(t *testing.T, cooldown, memory uint64) {
	t.Helper()
	chatterLimitsOverride = &chatterLimitValues{cooldown: cooldown, memory: memory}
	ForgetChatterForTest()
	prevRound := util.GetRoundCount()
	t.Cleanup(func() {
		chatterLimitsOverride = nil
		ForgetChatterForTest()
		util.SetRoundCount(prevRound)
		events.ClearQueueForTest()
	})
}

func idleTurn(m *Mob, listeners []int, cmds ...string) {
	m.BeginIdle(listeners)
	defer m.EndIdle()
	for _, c := range cmds {
		m.Command(c)
	}
}

func TestIdleChatterWaitsForItsCooldown(t *testing.T) {
	useChatterLimits(t, 60, 0)
	util.SetRoundCount(1000)
	m := &Mob{InstanceId: 9101, MobId: 9101}

	got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say The roads are bad.") })
	if len(got) != 1 {
		t.Fatalf("the first idle line is spoken, got %v", got)
	}

	util.SetRoundCount(1030)
	got = queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Taxes again.", "emote spits.") })
	if len(got) != 0 {
		t.Fatalf("no chatter inside the cooldown, got %v", got)
	}

	util.SetRoundCount(1060)
	got = queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Taxes again.") })
	if len(got) != 1 {
		t.Fatalf("chatter returns once the cooldown has passed, got %v", got)
	}
}

func TestAnIdleSpeechIsNeverCutInHalf(t *testing.T) {
	useChatterLimits(t, 60, 900)
	util.SetRoundCount(500)
	m := &Mob{InstanceId: 9102, MobId: 9102}

	got := queuedCommands(t, m.InstanceId, func() {
		idleTurn(m, []int{1}, "emote squints at the lake.", "say I crashed my boat out there.;say Never again.")
	})
	if len(got) != 3 {
		t.Fatalf("every line of an allowed turn is spoken, got %v", got)
	}
}

func TestIdleLinesDoNotRepeatToTheSamePlayer(t *testing.T) {
	useChatterLimits(t, 1, 900)
	util.SetRoundCount(100)
	m := &Mob{InstanceId: 9103, MobId: 9103}
	twin := &Mob{InstanceId: 9104, MobId: 9103} // another of the same kind

	if got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Coin for the poor?") }); len(got) != 1 {
		t.Fatalf("first time heard, got %v", got)
	}

	util.SetRoundCount(200)
	if got := queuedCommands(t, twin.InstanceId, func() { idleTurn(twin, []int{1}, "say Coin for the poor?") }); len(got) != 0 {
		t.Fatalf("the same line from the same kind of mob stays unsaid to player 1, got %v", got)
	}
	if got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Bless you, stranger.") }); len(got) != 1 {
		t.Fatalf("a fresh line is spoken, got %v", got)
	}

	util.SetRoundCount(300)
	if got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{2}, "say Coin for the poor?") }); len(got) != 1 {
		t.Fatalf("a player who never heard it hears it, got %v", got)
	}

	util.SetRoundCount(1200)
	if got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Coin for the poor?") }); len(got) != 1 {
		t.Fatalf("after the memory window the line may come back, got %v", got)
	}
}

func TestOnlyIdleChatterIsHeldBack(t *testing.T) {
	useChatterLimits(t, 60, 900)
	util.SetRoundCount(2000)
	m := &Mob{InstanceId: 9105, MobId: 9105}
	m.MarkChatter() // on cooldown

	got := queuedCommands(t, m.InstanceId, func() {
		idleTurn(m, []int{1}, "say Hello.", "wander", "lookfortrouble")
	})
	if len(got) != 2 || got[0] != "wander" || got[1] != "lookfortrouble" {
		t.Fatalf("non-chatter idle commands still run, got %v", got)
	}

	got = queuedCommands(t, m.InstanceId, func() {
		m.Command("say Stand and fight.")
		m.CommandScripted(0, "emote draws steel.")
	})
	if len(got) != 2 {
		t.Fatalf("chatter outside an idle turn is never held back, got %v", got)
	}

	// Scripts call CommandScripted; inside an idle turn it is held back too.
	got = queuedCommands(t, m.InstanceId, func() {
		m.BeginIdle([]int{1})
		m.CommandScripted(0, "say I really don't trust the arch-bishop.")
		m.EndIdle()
	})
	if len(got) != 0 {
		t.Fatalf("a script's idle chatter obeys the cooldown, got %v", got)
	}
}

func TestChatterLimitsCanBeTurnedOff(t *testing.T) {
	useChatterLimits(t, 0, 0)
	util.SetRoundCount(10)
	m := &Mob{InstanceId: 9106, MobId: 9106}
	for i := 0; i < 3; i++ {
		if got := queuedCommands(t, m.InstanceId, func() { idleTurn(m, []int{1}, "say Same again.") }); len(got) != 1 {
			t.Fatalf("with both limits at 0 every turn speaks, turn %d got %v", i, got)
		}
	}
}
