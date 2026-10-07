package company

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/bonds"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 65: bonds through the real module, the real camp and battle
// entries, the Phase 64 opinion seam and the views.

func companyOf(t *testing.T) domain.Record {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	return record
}

func TestACampRestDrawsSuitedCompanionsCloserAndClashingOnesApart(t *testing.T) {
	b := newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	banterPercents(t, 0, 0, 0) // no talk: the bonds move regardless
	opine(t, map[int]opinionSetup{
		1: {"stoic", 0, 50}, 2: {"devout", 0, 50}, // suit
		3: {"cheerful", 0, 50}, 4: {"grim", 0, 50}, // clash
	})
	require.Nil(t, domain.CampBanter(7, banter.CtxRested))
	assert.Equal(t, 2, bondOf(t, 1, 2).Value, "a stoic and a devout one suit")
	assert.Equal(t, -1, bondOf(t, 3, 4).Value, "a cheerful and a grim one clash")
	assert.Equal(t, 1, bondOf(t, 1, 3).Value, "the rest are one point")
	assert.Equal(t, 2, bondOf(t, 2, 3).Value, "cheerful and devout suit")

	// Not again inside the cooldown, so a camp loop is no farm.
	domain.CampBanter(7, banter.CtxRested)
	assert.Equal(t, 2, bondOf(t, 1, 2).Value)
	now = now.Add(bonds.CooldownOf(bonds.Camp) + time.Minute)
	domain.CampBanter(7, banter.CtxRested)
	assert.Equal(t, 4, bondOf(t, 1, 2).Value)
	// The start of a rest is not a shared camp yet.
	now = now.Add(time.Hour)
	domain.CampBanter(7, banter.CtxCamp)
	assert.Equal(t, 4, bondOf(t, 1, 2).Value)
	_ = b
}

func TestTimeTogetherNeverLiftsABondPastFiftyOrDropsOnePastWary(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	banterPercents(t, 0, 0, 0)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 50}, 2: {"devout", 0, 50}, 3: {"cheerful", 0, 50}, 4: {"grim", 0, 50}})
	setBond(t, 1, 2, 49)
	setBond(t, 3, 4, -24)
	domain.CampBanter(7, banter.CtxRested)
	assert.Equal(t, 50, bondOf(t, 1, 2).Value)
	assert.Equal(t, -25, bondOf(t, 3, 4).Value)
	now = now.Add(time.Hour)
	domain.CampBanter(7, banter.CtxRested)
	assert.Equal(t, 50, bondOf(t, 1, 2).Value, "camps stop at close")
	assert.Equal(t, -25, bondOf(t, 3, 4).Value, "and at wary: camping alone never makes rivals (Phase 65 review)")
	assert.False(t, bonds.IsRival(bondOf(t, 3, 4).Value))
}

func TestAWonBattleSideBySideIsOnePointUnlessTheyRub(t *testing.T) {
	b := newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	banterPercents(t, 0, 0, 0)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 50}, 2: {"devout", 0, 50}, 3: {"cheerful", 0, 50}, 4: {"grim", 0, 50}})
	battleWon(b, combatstream.OutcomeVictory)
	assert.Equal(t, 1, bondOf(t, 1, 2).Value)
	assert.Equal(t, 0, bondOf(t, 3, 4).Value, "cheerful and grim win beside each other without warming")
	battleWon(b, combatstream.OutcomeVictory)
	assert.Equal(t, 1, bondOf(t, 1, 2).Value, "ten minutes between")
	now = now.Add(bonds.CooldownOf(bonds.Battle) + time.Minute)
	battleWon(b, combatstream.OutcomeVictory)
	assert.Equal(t, 2, bondOf(t, 1, 2).Value)
	now = now.Add(time.Hour)
	battleWon(b, combatstream.OutcomeDefeat)
	assert.Equal(t, 2, bondOf(t, 1, 2).Value, "a lost battle is nothing to bond over")
}

func TestAFallenOrFledCompanionIsNotBondedWith(t *testing.T) {
	b := newBrawl(t)
	banterPercents(t, 0, 0, 0)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 50}, 2: {"devout", 0, 50}, 3: {"stoic", 0, 50}, 4: {"stoic", 0, 50}})
	record := companyOf(t)
	for i := range record.Companions {
		if record.Companions[i].ID == 2 {
			record.Companions[i].Death = &domain.CompanionDeath{}
		}
	}
	module.registry.Put(record)
	battleWon(b, combatstream.OutcomeVictory)
	domain.CampBanter(7, banter.CtxRested)
	assert.Equal(t, 0, bondOf(t, 1, 2).Value)
	assert.Positive(t, bondOf(t, 1, 3).Value)
}

func TestAgreeingOverAChoiceDrawsCompanionsTogetherAndSplittingPullsThemApart(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"grim", 0, 50}, 2: {"grim", 0, 50}, // both like a rough camp
		3: {"cheerful", 0, 50}, // dislikes it
	})
	_, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Op: "camp-1", Witnesses: []int{1, 2, 3}})
	require.NoError(t, err)
	assert.Equal(t, 1, bondOf(t, 1, 2).Value, "both liked it")
	assert.Equal(t, -1, bondOf(t, 1, 3).Value, "one liked it and one did not")
	assert.Equal(t, -1, bondOf(t, 2, 3).Value)
	assert.Equal(t, 0, bondOf(t, 1, 4).Value, "Ysolde saw nothing")

	// Splitting over a choice is a real clash: it takes a pair past wary,
	// which time together never does (Phase 65 review).
	setBond(t, 1, 3, -50)
	now = now.Add(3 * time.Hour) // past the opinions' own cooldown too
	_, err = module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Op: "camp-2", Witnesses: []int{1, 3}})
	require.NoError(t, err)
	assert.Equal(t, -51, bondOf(t, 1, 3).Value)
}

func TestARescueAndARefusalAreOncePerHourPerPair(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	a, b := domain.CompanionMemberKey(1), domain.CompanionMemberKey(2)
	module.BondEvent(7, a, b, bonds.Rescue)
	module.BondEvent(7, b, a, bonds.Rescue)
	assert.Equal(t, 3, bondOf(t, 1, 2).Value, "the pair is one bond, rescued once this hour")
	now = now.Add(61 * time.Minute)
	module.BondEvent(7, b, a, bonds.Rescue)
	assert.Equal(t, 6, bondOf(t, 1, 2).Value)
	module.BondEvent(7, a, b, bonds.Refusal)
	assert.Equal(t, 4, bondOf(t, 1, 2).Value)
	module.BondEvent(7, a, domain.LeaderMemberKey, bonds.Rescue)
	module.BondEvent(7, a, a, bonds.Rescue)
	module.BondEvent(7, a, b, bonds.Camp) // not an event a battle reports
	record := companyOf(t)
	require.Len(t, record.Bonds, 1, "no bond with the leader or oneself")
}

func TestARivalryIsWarnedThenOneLeaves(t *testing.T) {
	b := newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 60}, 3: {"stoic", 0, 20}})
	setBond(t, 1, 3, -84)
	k1, k3 := domain.CompanionMemberKey(1), domain.CompanionMemberKey(3)
	*b.messages = nil
	module.BondEvent(7, k1, k3, bonds.Refusal) // -86: warned
	events.ProcessEvents()
	told := companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
	assert.Contains(t, told, "can hardly stand to share a company. If it goes on, one of them will leave.")
	assert.True(t, bondOf(t, 1, 3).Warned)
	assert.Contains(t, told, "cannot bear each other", "and it says how far it has come")
	_, stillThere := findCompanion(companyOf(t), 3)
	require.True(t, stillThere)

	for i := 0; i < 7; i++ { // -88 ... -100
		now = now.Add(61 * time.Minute)
		module.BondEvent(7, k1, k3, bonds.Refusal)
	}
	record := companyOf(t)
	_, thereOne := findCompanion(record, 1)
	_, thereThree := findCompanion(record, 3)
	assert.True(t, thereOne, "the more loyal one stays")
	assert.False(t, thereThree, "the one with less loyalty has had enough")
	assert.Empty(t, record.Bonds, "a departed companion's bonds go with it")
	events.ProcessEvents()
	assert.Contains(t, companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), ""), "has had enough of")
}

func TestAMendedRivalryIsForgivenItsWarning(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	record := companyOf(t)
	record.SetBond(domain.Bond{A: 1, B: 3, Value: -62, Warned: true})
	module.registry.Put(record)
	module.BondEvent(7, domain.CompanionMemberKey(1), domain.CompanionMemberKey(3), bonds.Rescue) // -59
	assert.False(t, bondOf(t, 1, 3).Warned)
	assert.Equal(t, -59, bondOf(t, 1, 3).Value)
}

func TestAWarnedRivalryNeverLosesAnyoneBeforeItIsWarned(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	setBond(t, 1, 3, -99) // a hand-edited file
	module.BondEvent(7, domain.CompanionMemberKey(1), domain.CompanionMemberKey(3), bonds.Refusal)
	_, there := findCompanion(companyOf(t), 3)
	assert.True(t, there, "the warning comes first")
	assert.True(t, bondOf(t, 1, 3).Warned)
}

func TestBondsAreDroppedWhenACompanionLeavesTheRecord(t *testing.T) {
	newBrawl(t)
	setBond(t, 1, 2, 30)
	setBond(t, 2, 3, 40)
	record := companyOf(t)
	kept := record.Companions[:0:0]
	for _, c := range record.Companions {
		if c.ID != 2 {
			kept = append(kept, c)
		}
	}
	record.Companions = kept
	module.registry.Put(record)
	assert.Empty(t, companyOf(t).Bonds)
}

func TestBondsAreShownInWordsAndOnTheInspectLine(t *testing.T) {
	newBrawl(t)
	setBond(t, 1, 2, 62)
	setBond(t, 1, 3, -60)
	view := module.bondsView(7, "")
	assert.Contains(t, view, "Tamsin Reed and Brother Oswin are close. (bond +62)")
	assert.Contains(t, view, "Each steps in once a battle for the other when hurt")
	assert.Contains(t, view, "Tamsin Reed and Garrick Vane can't stand each other. (bond -60)")
	assert.Contains(t, view, "Won't guard each other: set a guardian's ward to someone else")
	assert.Contains(t, view, "are still getting to know each other", "a pair with no bond yet says so")
	filtered := module.bondsView(7, "oswin")
	assert.Contains(t, filtered, "Oswin")
	assert.NotContains(t, filtered, "Garrick Vane can't stand")
	assert.Contains(t, module.bondsView(7, "nobody"), "No companion called")

	panel, ok := domain.BondsOf(7)
	require.True(t, ok)
	require.Len(t, panel.Pairs, 6, "four companions make six pairs")
	assert.Equal(t, 62, panel.Pairs[0].Value, "the strongest feeling first")
	var tamsin domain.BondMember
	for _, m := range panel.Members {
		if m.ID == 1 {
			tamsin = m
		}
	}
	words := []string{}
	for _, f := range tamsin.Feelings {
		words = append(words, f.Words)
	}
	assert.Contains(t, words, "is close to Brother Oswin")
	assert.Contains(t, words, "can't stand Garrick Vane")

	inspect, _ := module.inspectMember(7, "tamsin", false)
	assert.Contains(t, inspect, "Bonds: is close to Brother Oswin")
}

func TestOnlyOnePersonHasNoBondsToShow(t *testing.T) {
	newBrawl(t)
	record := companyOf(t)
	record.Companions = record.Companions[:1]
	module.registry.Put(record)
	assert.Contains(t, module.bondsView(7, ""), "You need at least two")
}

func TestFriendsAndRivalsTalkAboutEachOtherAndItMovesTheirBond(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	banterPercents(t, 0, 100, 0)
	setBond(t, 1, 2, 60)
	talked := false
	for i := 0; i < 80 && !talked; i++ {
		for _, s := range domain.CampBanter(7, banter.CtxCamp) {
			talked = talked || s.Ctx == banter.CtxFriend
		}
	}
	require.True(t, talked, "friends sometimes talk about each other")
	assert.Greater(t, bondOf(t, 1, 2).Value, 60-1, "a friend line warms the pair")

	// A rival line sours them, once a half hour.
	setBond(t, 1, 2, -20)
	module.bondsFromTalk(7, []banter.Said{{Member: 1, Ctx: banter.CtxRival}, {Member: 2, Ctx: banter.CtxRival}})
	assert.Equal(t, -21, bondOf(t, 1, 2).Value)
	module.bondsFromTalk(7, []banter.Said{{Member: 2, Ctx: banter.CtxRival}, {Member: 1, Ctx: banter.CtxRival}})
	assert.Equal(t, -21, bondOf(t, 1, 2).Value, "the same half hour")
	module.bondsFromTalk(7, []banter.Said{{Member: 1, Ctx: banter.CtxCamp}, {Member: 2, Ctx: banter.CtxCamp}})
	assert.Equal(t, -21, bondOf(t, 1, 2).Value, "ordinary talk moves nothing")
}

// Phase 65 review: a rivalry that reaches its end in the middle of a fight
// marks the leaver, but no one leaves until the fight is over; the next
// round's retry sends them away.
func TestARivalryEndingMidFightWaitsForTheFightToEnd(t *testing.T) {
	b := newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 60}, 3: {"stoic", 0, 20}})
	setBond(t, 1, 3, -86)
	record := companyOf(t)
	bond, _ := record.BondOf(1, 3)
	bond.Warned = true
	record.SetBond(bond)
	module.registry.Put(record)

	b.aria.Character.Aggro = &characters.Aggro{}
	k1, k3 := domain.CompanionMemberKey(1), domain.CompanionMemberKey(3)
	for i := 0; i < 7; i++ { // -88 ... -100
		now = now.Add(61 * time.Minute)
		module.BondEvent(7, k1, k3, bonds.Refusal)
	}
	leaver, there := findCompanion(companyOf(t), 3)
	require.True(t, there, "no one leaves in the middle of a fight")
	assert.True(t, leaver.MoraleDesert, "but the leaver is marked, and saved")

	b.aria.Character.Aggro = nil
	module.retryReturns()
	_, there = findCompanion(companyOf(t), 3)
	assert.False(t, there, "the fight over, they go")
	_, there = findCompanion(companyOf(t), 1)
	assert.True(t, there)
}

// Phase 65 review: a bond change whose save fails leaves nothing behind: no
// value, no cooldown stamp, so the same event counts once the save works.
func TestAFailedBondSaveLeavesTheRegistryAsItWas(t *testing.T) {
	m := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 3, MobTemplateID: 58}},
			Bonds: []domain.Bond{{A: 1, B: 3, Value: -10}}},
	}}, &fakeRuntime{})
	now := time.Unix(1_800_000_000, 0)
	m.clock = func() time.Time { return now }
	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	k1, k3 := domain.CompanionMemberKey(1), domain.CompanionMemberKey(3)
	m.BondEvent(7, k1, k3, bonds.Refusal)
	record, _ := m.registry.Get(7)
	bond, _ := record.BondOf(1, 3)
	assert.Equal(t, -10, bond.Value)
	assert.Empty(t, bond.At, "no cooldown stamp that was never saved")

	store.saveErr = nil
	m.BondEvent(7, k1, k3, bonds.Refusal)
	record, _ = m.registry.Get(7)
	bond, _ = record.BondOf(1, 3)
	assert.Equal(t, -12, bond.Value, "the refusal counts once the save works")
}

// Phase 65 review: a warned pair already at the bottom does not lose anyone
// to time together (a clashing camp that cannot move the bond); only a real
// clash sends one away.
func TestAWarnedPairAtTheBottomLeavesOnlyOnARealClash(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	banterPercents(t, 0, 0, 0)
	opine(t, map[int]opinionSetup{1: {"cheerful", 0, 60}, 2: {"stoic", 0, 50}, 3: {"stoic", 0, 50}, 4: {"grim", 0, 20}})
	setBond(t, 1, 4, -100)
	record := companyOf(t)
	bond, _ := record.BondOf(1, 4)
	bond.Warned = true
	record.SetBond(bond)
	module.registry.Put(record)

	domain.CampBanter(7, banter.CtxRested) // cheerful and grim clash: -1, held at -100
	_, there := findCompanion(companyOf(t), 4)
	require.True(t, there, "a camp is not a clash")

	module.BondEvent(7, domain.CompanionMemberKey(1), domain.CompanionMemberKey(4), bonds.Refusal)
	_, there = findCompanion(companyOf(t), 4)
	assert.False(t, there, "a refused guard is")
}
