package hooks

import "testing"

// Phase 61 review: "first" is the first round a battle's orders are read,
// not its StartRound, so a battle that begins at a round's end (the next
// group stepping up) still has a first round.
func TestFirstOrdersRoundIsTheFirstRoundRead(t *testing.T) {
	before := combatRound.Load()
	t.Cleanup(func() { combatRound.Store(before); clear(fightOpened) })
	clear(fightOpened)
	combatRound.Store(40)
	if !firstOrdersRound(9001) || !firstOrdersRound(9001) {
		t.Fatal("the first round read is the first round, for every member")
	}
	combatRound.Store(41)
	if firstOrdersRound(9001) {
		t.Fatal("a later round is not the first")
	}
	if !firstOrdersRound(9002) {
		t.Fatal("a battle begun since has its own first round")
	}
}
