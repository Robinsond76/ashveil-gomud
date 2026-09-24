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

func TestChargeOncePerRound(t *testing.T) {
	var r Record
	c1 := CompanionMemberKey(1)
	if !r.ChargeService(c1, 10) {
		t.Fatal("first charge")
	}
	if r.ChargeService(c1, 10) {
		t.Fatal("same round charged twice")
	}
	if r.ChargeService(c1, 9) {
		t.Fatal("a replayed earlier round was charged")
	}
	if !r.ChargeService(c1, 11) || !r.ChargeService(LeaderMemberKey, 11) {
		t.Fatal("next round")
	}
	s, ok := r.FindService(c1)
	if !ok || s.Rounds != 2 || s.LastRound != 11 || s.Saved != 0 {
		t.Fatalf("service %+v", s)
	}
}

func TestChargeResumesAfterCounterReset(t *testing.T) {
	r := Record{Service: []Service{{Member: LeaderMemberKey, Rounds: 50, LastRound: 1_400_000}}}
	if r.ChargeService(LeaderMemberKey, 1_399_990) {
		t.Fatal("a slightly earlier round was charged")
	}
	if !r.ChargeService(LeaderMemberKey, 1_314_000) {
		t.Fatal("after a reset the counter should resume")
	}
}

func TestBandAverageDilutesAndUsesSavedRounds(t *testing.T) {
	rules := DefaultChemistryRules()
	c1, c2, c3, c4 := CompanionMemberKey(1), CompanionMemberKey(2), CompanionMemberKey(3), CompanionMemberKey(4)
	r := Record{Service: []Service{
		{Member: LeaderMemberKey, Rounds: 7000, Saved: 7000},
		{Member: c1, Rounds: 7000, Saved: 7000},
		{Member: c2, Rounds: 7000, Saved: 7000},
		{Member: c3, Rounds: 7000, Saved: 6300},
	}}
	veterans := []MemberKey{LeaderMemberKey, c1, c2, c3}
	if tier := r.BandTier(veterans, rules); tier != TierSworn {
		t.Fatalf("four veterans: %d", tier)
	}
	// A fresh recruit (no entry) dilutes 4 veterans to about Trusted.
	withRecruit := append(append([]MemberKey(nil), veterans...), c4)
	if avg, _ := r.BandAverage(withRecruit, true, 0); avg != (7000*3+6300)/5 {
		t.Fatalf("average %d", avg)
	}
	if tier := r.BandTier(withRecruit, rules); tier != TierTrusted {
		t.Fatalf("with a recruit: %d", tier)
	}
	// Only saved rounds count toward a tier.
	fresh := Record{Service: []Service{{Member: LeaderMemberKey, Rounds: 900}, {Member: c1, Rounds: 900}}}
	if tier := fresh.BandTier([]MemberKey{LeaderMemberKey, c1}, rules); tier != TierNone {
		t.Fatalf("unsaved rounds gave tier %d", tier)
	}
	if avg, _ := fresh.BandAverage([]MemberKey{LeaderMemberKey, c1}, false, 0); avg != 900 {
		t.Fatalf("current average %d", avg)
	}
	fresh.MarkServiceSaved()
	if tier := fresh.BandTier([]MemberKey{LeaderMemberKey, c1}, rules); tier != TierFamiliar {
		t.Fatalf("saved rounds tier %d", tier)
	}
	// A lone member is no band.
	if _, ok := r.BandAverage([]MemberKey{LeaderMemberKey}, true, 0); ok {
		t.Fatal("a lone member is no band")
	}
	if r.BandTier([]MemberKey{LeaderMemberKey}, rules) != TierNone {
		t.Fatal("a lone member gets no tier")
	}
}

func TestGetCopiesService(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 1, Companions: []Companion{{ID: 1, MobTemplateID: 5}}, Service: []Service{{Member: CompanionMemberKey(1), Rounds: 3, Saved: 3}}})
	got, _ := r.Get(1)
	got.Service[0].Rounds = 99
	again, _ := r.Get(1)
	if again.Service[0].Rounds != 3 || again.Service[0].Saved != 3 {
		t.Fatalf("Get must copy service: %+v", again.Service)
	}
	clone := r.Clone()
	clone.Companies[1].Service[0].Rounds = 77
	if again, _ := r.Get(1); again.Service[0].Rounds != 3 {
		t.Fatal("Clone must copy service")
	}
}

func TestPutPrunesServiceOfRemovedMembers(t *testing.T) {
	r := NewRegistry()
	c1, c2 := CompanionMemberKey(1), CompanionMemberKey(2)
	r.Put(Record{
		LeaderUserID: 1,
		Companions:   []Companion{{ID: 1, MobTemplateID: 5}, {ID: 2, MobTemplateID: 5}},
		Service: []Service{
			{Member: LeaderMemberKey, Rounds: 10},
			{Member: c1, Rounds: 20},
			{Member: c2, Rounds: 30},
			{Member: CompanionMemberKey(7), Rounds: 40},
		},
	})
	record, _ := r.Get(1)
	if len(record.Service) != 3 {
		t.Fatalf("an unknown member's service should be pruned: %+v", record.Service)
	}
	r.Dismiss(1, 1)
	record, _ = r.Get(1)
	if _, ok := record.FindService(c1); ok || len(record.Service) != 2 {
		t.Fatalf("dismissal must end exactly companion 1's service: %+v", record.Service)
	}
	r.DismissAll(1)
	record, _ = r.Get(1)
	if _, ok := record.FindService(LeaderMemberKey); !ok || len(record.Service) != 1 {
		t.Fatalf("dismiss all keeps only the leader's own service: %+v", record.Service)
	}
}

func TestPutNormalizesService(t *testing.T) {
	r := NewRegistry()
	c1 := CompanionMemberKey(1)
	r.Put(Record{
		LeaderUserID: 1,
		Companions:   []Companion{{ID: 1, MobTemplateID: 5}},
		Service: []Service{
			{Member: c1, Rounds: 40, LastRound: 9, Saved: 40},
			{Member: c1, Rounds: 30, LastRound: 12, Saved: 30}, // a duplicate
			{Member: LeaderMemberKey, Rounds: -5, Saved: 7},
		},
	})
	record, _ := r.Get(1)
	want := []Service{{Member: c1, Rounds: 40, LastRound: 12, Saved: 40}, {Member: LeaderMemberKey}}
	if len(record.Service) != 2 || record.Service[0] != want[0] || record.Service[1] != want[1] {
		t.Fatalf("service %+v, want %+v", record.Service, want)
	}
}

func TestServiceYAMLRoundTrip(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 1, Companions: []Companion{{ID: 1, MobTemplateID: 5}}, Service: []Service{{Member: CompanionMemberKey(1), Rounds: 901, LastRound: 1314123, Saved: 901}}})
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Registry
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	s, ok := back.Companies[1].FindService(CompanionMemberKey(1))
	if !ok || s.Rounds != 901 || s.LastRound != 1314123 {
		t.Fatalf("round trip lost the service: %+v\n%s", s, data)
	}
	if s.Saved != 0 {
		t.Fatal("Saved is in-memory only; the loader marks loaded service saved")
	}
}

// Review finding (band rework) 1: long service can't hide a new recruit.
func TestBandTierCapsEachMembersService(t *testing.T) {
	rules := DefaultChemistryRules()
	c1 := CompanionMemberKey(1)
	r := Record{Service: []Service{{Member: LeaderMemberKey, Rounds: 27000, Saved: 27000}}}
	pair := []MemberKey{LeaderMemberKey, c1}
	if avg, _ := r.BandAverage(pair, true, 0); avg != 13500 {
		t.Fatalf("uncapped average %d", avg)
	}
	if avg, _ := r.BandAverage(pair, true, 6300); avg != 3150 {
		t.Fatalf("capped average %d", avg)
	}
	if tier := r.BandTier(pair, rules); tier != TierTrusted {
		t.Fatalf("a 30-day veteran and a recruit: tier %d, want Trusted", tier)
	}
}
