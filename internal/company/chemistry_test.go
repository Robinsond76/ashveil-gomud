package company

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestTierBoundaries(t *testing.T) {
	rules := DefaultChemistryRules()
	cases := []struct{ rounds, tier, bonus int }{
		{0, TierNone, 0}, {899, TierNone, 0}, {900, TierFamiliar, 2},
		{2699, TierFamiliar, 2}, {2700, TierTrusted, 4}, {6299, TierTrusted, 4},
		{6300, TierSworn, 6}, {100000, TierSworn, 6},
	}
	for _, c := range cases {
		tier := rules.Tier(c.rounds)
		if tier != c.tier || rules.Bonus(tier) != c.bonus {
			t.Errorf("rounds %d: tier %d bonus %d, want %d %d", c.rounds, tier, rules.Bonus(tier), c.tier, c.bonus)
		}
	}
	if rules.Bonus(TierNone) != 0 || rules.Bonus(9) != 0 {
		t.Error("out-of-range tiers must give no bonus")
	}
}

func TestRulesValid(t *testing.T) {
	if !DefaultChemistryRules().Valid() {
		t.Fatal("defaults must be valid")
	}
	bad := []ChemistryRules{
		{TierRounds: [3]int{0, 2, 3}, TierBonus: [3]int{1, 2, 3}},
		{TierRounds: [3]int{5, 5, 6}, TierBonus: [3]int{1, 2, 3}},
		{TierRounds: [3]int{1, 2, 3}, TierBonus: [3]int{3, 2, 4}},
		{TierRounds: [3]int{1, 2, 3}, TierBonus: [3]int{1, 2, 11}},
		{TierRounds: [3]int{1, 2, 3}, TierBonus: [3]int{-1, 2, 3}},
	}
	for i, r := range bad {
		if r.Valid() {
			t.Errorf("case %d should be invalid: %+v", i, r)
		}
	}
}

func TestBondPairSorted(t *testing.T) {
	a, b := BondPair(CompanionMemberKey(2), LeaderMemberKey)
	c, d := BondPair(LeaderMemberKey, CompanionMemberKey(2))
	if a != c || b != d || a > b {
		t.Fatalf("pair not canonical: %s %s / %s %s", a, b, c, d)
	}
}

func TestChargeOncePerRound(t *testing.T) {
	var r Record
	c1 := CompanionMemberKey(1)
	if before, after, ok := r.ChargeBond(LeaderMemberKey, c1, 10); !ok || before != 0 || after != 1 {
		t.Fatalf("first charge: %d %d %v", before, after, ok)
	}
	if _, _, ok := r.ChargeBond(c1, LeaderMemberKey, 10); ok {
		t.Fatal("same round charged twice")
	}
	if _, _, ok := r.ChargeBond(LeaderMemberKey, c1, 9); ok {
		t.Fatal("a replayed earlier round was charged")
	}
	if _, after, ok := r.ChargeBond(LeaderMemberKey, c1, 11); !ok || after != 2 {
		t.Fatalf("next round: %d %v", after, ok)
	}
	if _, _, ok := r.ChargeBond(c1, c1, 12); ok || len(r.Bonds) != 1 {
		t.Fatal("a member can't bond with itself")
	}
	bond, ok := r.FindBond(c1, LeaderMemberKey)
	if !ok || bond.Rounds != 2 || bond.LastRound != 11 {
		t.Fatalf("bond %+v", bond)
	}
}

func TestChargeResumesAfterCounterReset(t *testing.T) {
	r := Record{Bonds: []Bond{{A: CompanionMemberKey(1), B: LeaderMemberKey, Rounds: 50, LastRound: 1_400_000}}}
	// A few rounds behind (a crash between counter saves): wait.
	if _, _, ok := r.ChargeBond(LeaderMemberKey, CompanionMemberKey(1), 1_399_990); ok {
		t.Fatal("a slightly earlier round was charged")
	}
	// Far behind (a reset counter file): resume.
	if _, after, ok := r.ChargeBond(LeaderMemberKey, CompanionMemberKey(1), 1_314_000); !ok || after != 51 {
		t.Fatalf("after a reset: %d %v", after, ok)
	}
}

func TestBestBondHighestPresentOnly(t *testing.T) {
	rules := DefaultChemistryRules()
	c1, c2, c3 := CompanionMemberKey(1), CompanionMemberKey(2), CompanionMemberKey(3)
	bonds := []Bond{
		{A: c1, B: LeaderMemberKey, Rounds: 1000},
		{A: c2, B: LeaderMemberKey, Rounds: 7000},
		{A: c3, B: LeaderMemberKey, Rounds: 3000},
		{A: c1, B: c2, Rounds: 9000},
	}
	best, tier, ok := BestBond(bonds, LeaderMemberKey, nil, rules)
	if !ok || tier != TierSworn || best.A != c2 {
		t.Fatalf("best %+v tier %d", best, tier)
	}
	notC2 := func(k MemberKey) bool { return k != c2 }
	best, tier, ok = BestBond(bonds, LeaderMemberKey, notC2, rules)
	if !ok || tier != TierTrusted || best.A != c3 {
		t.Fatalf("without c2: %+v tier %d", best, tier)
	}
	if _, _, ok := BestBond(bonds, LeaderMemberKey, func(MemberKey) bool { return false }, rules); ok {
		t.Fatal("no present partner must mean no bond")
	}
	if _, _, ok := BestBond(bonds, CompanionMemberKey(9), nil, rules); ok {
		t.Fatal("an unbonded member has no bond")
	}
}

func TestProgressPercent(t *testing.T) {
	rules := DefaultChemistryRules()
	cases := []struct{ rounds, next, pct int }{
		{0, TierFamiliar, 0}, {450, TierFamiliar, 50}, {900, TierTrusted, 0},
		{1800, TierTrusted, 50}, {6299, TierSworn, 99}, {6300, TierNone, 100},
	}
	for _, c := range cases {
		next, pct := rules.Progress(c.rounds)
		if next != c.next || pct != c.pct {
			t.Errorf("rounds %d: next %d %d%%, want %d %d%%", c.rounds, next, pct, c.next, c.pct)
		}
	}
}

func TestGetCopiesBonds(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 1, Companions: []Companion{{ID: 1, MobTemplateID: 5}}, Bonds: []Bond{{A: CompanionMemberKey(1), B: LeaderMemberKey, Rounds: 3}}})
	got, _ := r.Get(1)
	got.Bonds[0].Rounds = 99
	again, _ := r.Get(1)
	if again.Bonds[0].Rounds != 3 {
		t.Fatal("Get must copy bonds")
	}
	clone := r.Clone()
	clone.Companies[1].Bonds[0].Rounds = 77
	if again, _ := r.Get(1); again.Bonds[0].Rounds != 3 {
		t.Fatal("Clone must copy bonds")
	}
}

func TestPutPrunesBondsOfRemovedMembers(t *testing.T) {
	r := NewRegistry()
	c1, c2 := CompanionMemberKey(1), CompanionMemberKey(2)
	r.Put(Record{
		LeaderUserID: 1,
		Companions:   []Companion{{ID: 1, MobTemplateID: 5}, {ID: 2, MobTemplateID: 5}},
		Bonds: []Bond{
			{A: c1, B: LeaderMemberKey, Rounds: 10},
			{A: c2, B: LeaderMemberKey, Rounds: 20},
			{A: c1, B: c2, Rounds: 30},
			{A: CompanionMemberKey(7), B: LeaderMemberKey, Rounds: 40},
		},
	})
	record, _ := r.Get(1)
	if len(record.Bonds) != 3 {
		t.Fatalf("a bond with an unknown member should be pruned: %+v", record.Bonds)
	}
	r.Dismiss(1, 1)
	record, _ = r.Get(1)
	if len(record.Bonds) != 1 || record.Bonds[0].A != c2 || record.Bonds[0].Rounds != 20 {
		t.Fatalf("dismissal must end exactly companion 1's bonds: %+v", record.Bonds)
	}
	r.DismissAll(1)
	record, _ = r.Get(1)
	if len(record.Bonds) != 0 {
		t.Fatalf("dismiss all ends every bond: %+v", record.Bonds)
	}
}

func TestBondsYAMLRoundTrip(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 1, Companions: []Companion{{ID: 1, MobTemplateID: 5}}, Bonds: []Bond{{A: CompanionMemberKey(1), B: LeaderMemberKey, Rounds: 901, LastRound: 1314123}}})
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Registry
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	bond, ok := back.Companies[1].FindBond(LeaderMemberKey, CompanionMemberKey(1))
	if !ok || bond.Rounds != 901 || bond.LastRound != 1314123 {
		t.Fatalf("round trip lost the bond: %+v\n%s", bond, data)
	}
}
