package battle

import "testing"

func TestGuardCounts(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("no battle, no guards: %d", got)
	}
	if _, ok := SpendGuard(1, "companion:1"); ok {
		t.Error("no battle: nothing to spend")
	}
	Begin(1, 10, 5, "p", []int{100})
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("a battle starts with %d guards, got %d", MaxGuards, got)
	}
	if left, ok := SpendGuard(1, "companion:1"); !ok || left != 1 {
		t.Errorf("spend: %d %v", left, ok)
	}
	if left, ok := SpendGuard(1, "companion:1"); !ok || left != 0 {
		t.Errorf("spend the last: %d %v", left, ok)
	}
	if _, ok := SpendGuard(1, "companion:1"); ok {
		t.Error("none left")
	}
	if got := GuardsLeft(1, "leader"); got != MaxGuards {
		t.Errorf("each guardian has its own: %d", got)
	}

	// One back per GuardRefillRounds combat rounds, up to MaxGuards.
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("one round: %d", got)
	}
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("two rounds: %d", got)
	}
	TickGuards()
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 2 {
		t.Errorf("four rounds: %d", got)
	}
	TickGuards()
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("capped: %d", got)
	}

	// A charge doesn't carry over a full count: spent again, it takes the
	// whole wait.
	SpendGuard(1, "companion:1")
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("one round after spending from full: %d", got)
	}

	// Copies out: a caller's copy can't touch the battle.
	b, _ := Current(1)
	b.Guards["companion:1"] = Guard{Left: 9}
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("Current is a copy: %d", got)
	}

	// Gone with the battle; a new one starts full.
	End(1)
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("ended: %d", got)
	}
	Begin(1, 10, 6, "q", []int{101})
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("a new battle: %d", got)
	}
}
