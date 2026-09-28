package items

import "testing"

// Phase 29c: only a real critical hit draws the critical pool; a
// non-critical roll over 100% (a sharpened top roll) caps at heavy.
func TestGetAttackMessagePoolChoice(t *testing.T) {
	const st ItemSubType = "test-pools"
	pool := func(marker string) AttackOptions {
		return AttackOptions{Together: TogetherMessages{ToAttacker: MessageOptions{ItemMessage(marker)}}}
	}
	group := &WeaponAttackMessageGroup{OptionId: st, Options: AttackTypes{}}
	for _, in := range []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical} {
		group.Options[in] = pool(string(in))
	}
	attackMessages[st] = group
	t.Cleanup(func() { delete(attackMessages, st) })

	cases := []struct {
		pct  int
		crit bool
		want Intensity
	}{
		{0, false, Miss},
		{10, false, Weak},
		{50, false, Normal},
		{80, false, Heavy},
		{150, false, Heavy},
		{40, true, Critical},
		{150, true, Critical},
	}
	for _, c := range cases {
		got := string(GetAttackMessage(st, c.pct, c.crit).Together.ToAttacker.Get(1))
		if got != string(c.want) {
			t.Errorf("GetAttackMessage(%d, %v) = %s, want %s", c.pct, c.crit, got, c.want)
		}
	}
}
