package company

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rites"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ritesLoss is leader 7 with companions #2 and #3 (loyalty 60) who both
// trusted #1 (bond 40), and #1 lost for good. It leaves the rites queued.
func ritesLoss(t *testing.T) (*CompanyModule, *fakeChemWorld, *chronicle.Memory) {
	t.Helper()
	module, world, _, _ := newDeathModule(t)
	mem := useChronicle(t)
	record, _ := module.registry.Get(7)
	record.Companions = append(record.Companions, domain.Companion{ID: 3, MobTemplateID: deathTemplate, State: &domain.MemberState{Level: 2}, Disposition: &domain.Disposition{Alignment: 10, Loyalty: 60}})
	record.Companions[1].Disposition.Loyalty = 60
	record.SetBond(domain.Bond{A: 1, B: 2, Value: 40})
	record.SetBond(domain.Bond{A: 1, B: 3, Value: 40})
	record.NextCompanionID = 4
	module.registry.Put(record)
	killOne(module)
	require.NoError(t, module.expire(7, 1))
	module.riteSeam = func(*users.UserRecord) string { return "" }
	return module, world, mem
}

func loyalty(t *testing.T, module *CompanyModule, id int) int {
	t.Helper()
	return companion(t, module, id).Disposition.Loyalty
}

func TestALostCompanionLeavesRitesInTheSameSaveAsTheExpiry(t *testing.T) {
	module, world, _ := ritesLoss(t)
	record, _ := module.registry.Get(7)
	require.Len(t, record.Rites, 1)
	rite := record.Rites[0]
	assert.Equal(t, 1, rite.Companion)
	assert.Equal(t, "lost", rite.Cause)
	assert.Equal(t, 5, rite.Level)
	assert.Equal(t, []int{2, 3}, rite.Close, "both trusted them")
	assert.Empty(t, rite.Band)
	assert.False(t, rite.Offered)
	saved := module.store.(*fakeStore).saved.Companies[7]
	require.Len(t, saved.Rites, 1, "the file that lost the companion carries the rites")
	assert.Contains(t, told(world), "You can hold rites for them at your next camp or inn (rites)")
}

func TestAFailedExpiryLeavesNoRiteBehind(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	require.Error(t, module.expire(7, 1))
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Rites, "the companion is still here, so nothing is mourned")
	module.store.(*fakeStore).saveErr = nil
	require.NoError(t, module.expire(7, 1))
	require.NoError(t, module.expire(7, 1), "a retry after the loss does nothing twice")
	record, _ = module.registry.Get(7)
	assert.Len(t, record.Rites, 1)
}

func TestDesertionMournsOnlyAfterLongService(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	record, _ := module.registry.Get(7)
	record.Service = []domain.Service{{Member: domain.CompanionMemberKey(1), Rounds: rites.LongServiceRounds}, {Member: domain.CompanionMemberKey(2), Rounds: rites.LongServiceRounds - 1}}
	module.registry.Put(record)
	record, _ = module.registry.Get(7)
	c1, _ := findCompanion(record, 1)
	require.NoError(t, module.removeCompanion(7, record, c1))
	record, _ = module.registry.Get(7)
	require.Len(t, record.Rites, 1)
	assert.Equal(t, "left", record.Rites[0].Cause)
	assert.Equal(t, []int{2}, record.Rites[0].Band, "no one trusted them, so the rest of the band mourns")

	c2, _ := findCompanion(record, 2)
	require.NoError(t, module.removeCompanion(7, record, c2))
	record, _ = module.registry.Get(7)
	assert.Len(t, record.Rites, 1, "a short service is not mourned")
}

func TestDismissalIsNotMourned(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	record, _ := module.registry.Get(7)
	record.Service = []domain.Service{{Member: domain.CompanionMemberKey(1), Rounds: 10 * rites.LongServiceRounds}}
	module.registry.Put(record)
	_, err := module.dismiss(7, "#1")
	require.NoError(t, err)
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Rites, "the leader chose to send them away")
}

func TestHoldingRitesSteadiesTheCloseDrawsThemTogetherAndIsRecorded(t *testing.T) {
	module, _, mem := ritesLoss(t)
	text := module.ritesAct(&users.UserRecord{UserId: 7}, "#1", true)
	assert.Contains(t, text, "hold rites for #1")
	assert.Contains(t, text, "(loyalty +3)")
	assert.Equal(t, 63, loyalty(t, module, 2))
	assert.Equal(t, 63, loyalty(t, module, 3))
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Rites)
	bond, _ := record.BondOf(2, 3)
	assert.Equal(t, rites.BondGain, bond.Value)
	deeds := deeds(mem, chronicle.Rites)
	require.Len(t, deeds, 1)
	assert.Equal(t, "The company held rites for #1 (lost for good).", chronicle.Prose(deeds[0]))
	assert.Empty(t, module.store.(*fakeStore).saved.Companies[7].Rites)
	assert.Equal(t, "No one is waiting to be mourned.", module.ritesAct(&users.UserRecord{UserId: 7}, "", true))
}

func TestSkippingRitesCostsLoyaltyButNeverBelowTheFloor(t *testing.T) {
	module, _, mem := ritesLoss(t)
	record, _ := module.registry.Get(7)
	for i, c := range record.Companions {
		if c.ID == 3 {
			record.Companions[i].Disposition.Loyalty = rites.Floor + 2 // #3 is nearly at the floor
		}
	}
	module.registry.Put(record)
	text := module.ritesAct(&users.UserRecord{UserId: 7}, "", false)
	assert.Contains(t, text, "go without a word")
	assert.Equal(t, 55, loyalty(t, module, 2), "-5 for a companion who trusted them")
	assert.Equal(t, rites.Floor, loyalty(t, module, 3), "never below the floor")
	deeds := deeds(mem, chronicle.Rites)
	require.Len(t, deeds, 1)
	assert.Equal(t, "The company let #1 go without a word (lost for good).", chronicle.Prose(deeds[0]))
}

func TestRitesNeedACampOrInnAndAreNeverHeldTwice(t *testing.T) {
	module, _, _ := ritesLoss(t)
	module.riteSeam = nil // the real check: no camp or inn stay here
	text := module.ritesAct(&users.UserRecord{UserId: 7}, "", true)
	assert.Contains(t, text, "Rites are held at a camp or an inn")
	record, _ := module.registry.Get(7)
	assert.Len(t, record.Rites, 1, "nothing was held or skipped")
	assert.Equal(t, 60, loyalty(t, module, 2))
}

func TestRitesPickByNumberNameAndAll(t *testing.T) {
	record := domain.Record{Rites: []domain.Rite{{Op: "a", Companion: 1, Name: "Hild"}, {Op: "b", Companion: 2, Name: "Hilda"}, {Op: "c", Companion: 3, Name: "Brann"}}}
	got, why := riteMatches(record, "#3")
	require.Empty(t, why)
	assert.Equal(t, "Brann", got[0].Name)
	got, _ = riteMatches(record, "hild")
	require.Len(t, got, 1, "an exact name beats a prefix")
	assert.Equal(t, 1, got[0].Companion)
	_, why = riteMatches(record, "hil")
	assert.Contains(t, why, "More than one")
	got, _ = riteMatches(record, "all")
	assert.Len(t, got, 3)
	_, why = riteMatches(record, "")
	assert.Contains(t, why, "More than one is waiting")
	_, why = riteMatches(record, "9")
	assert.Contains(t, why, "No one by that number")
}

func TestACampOffersThenLetsAnUnansweredRitePassAtTheNext(t *testing.T) {
	module, _, mem := ritesLoss(t)
	text := module.OfferRites(7)
	assert.Contains(t, text, "has not yet mourned")
	assert.Contains(t, text, "#1 (lost for good)")
	record, _ := module.registry.Get(7)
	require.Len(t, record.Rites, 1)
	assert.True(t, record.Rites[0].Offered)
	assert.Equal(t, 60, loyalty(t, module, 2), "an offer costs nothing")
	assert.Contains(t, module.ritesView(7), "it passes at the next camp or inn")

	// The next camp finds it unanswered: it passes, at its cost.
	text = module.OfferRites(7)
	assert.Contains(t, text, "No rites were ever held for #1, and now the moment has passed", "the leader is told the cost came from silence")
	assert.Equal(t, 55, loyalty(t, module, 2))
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Rites)
	require.Len(t, deeds(mem, chronicle.Rites), 1)
	assert.Equal(t, "", module.OfferRites(7), "nothing waits")
}

func TestARiteHeldAfterTheOfferIsNotLetPass(t *testing.T) {
	module, _, mem := ritesLoss(t)
	module.OfferRites(7)
	module.ritesAct(&users.UserRecord{UserId: 7}, "all", true)
	assert.Equal(t, "", module.OfferRites(7))
	assert.Equal(t, 63, loyalty(t, module, 2))
	assert.Equal(t, "rite:held", deeds(mem, chronicle.Rites)[0].Ref)
}

func TestARiteSaveFailureChangesNothing(t *testing.T) {
	module, _, mem := ritesLoss(t)
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	text := module.ritesAct(&users.UserRecord{UserId: 7}, "", true)
	assert.Contains(t, text, "could not be kept")
	record, _ := module.registry.Get(7)
	assert.Len(t, record.Rites, 1)
	assert.Equal(t, 60, loyalty(t, module, 2))
	assert.Empty(t, deeds(mem, chronicle.Rites))
}

func TestRitesAndBondsSurviveARestart(t *testing.T) {
	module, _, _ := ritesLoss(t)
	module.OfferRites(7)
	record, _ := module.registry.Get(7)
	record.SetBond(domain.Bond{A: 2, B: 3, Value: 31})
	module.registry.Put(record)
	data, err := yamlMarshal(module.registry)
	require.NoError(t, err)
	loaded := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, loaded))
	got, _ := loaded.Get(7)
	require.Len(t, got.Rites, 1)
	assert.True(t, got.Rites[0].Offered)
	assert.Equal(t, []int{2, 3}, got.Rites[0].Close)
	bond, ok := got.BondOf(2, 3)
	assert.True(t, ok, "Phase 65's bonds were written but not read back (the load dropped them)")
	assert.Equal(t, 31, bond.Value)
}

func TestTheRitePanelShowsWhatWaits(t *testing.T) {
	module, _, _ := ritesLoss(t)
	panel, ok := module.RitePanel(7)
	require.True(t, ok)
	require.Len(t, panel.Rows, 1)
	row := panel.Rows[0]
	assert.Equal(t, 1, row.ID)
	assert.Equal(t, "lost for good", row.Cause)
	assert.Len(t, row.Close, 2)
	assert.False(t, row.Offered)
}

// The rules' limits match the opinions they borrow from.
func TestRitesLoyaltyLimitsMatchOpinions(t *testing.T) {
	assert.Equal(t, opinions.Floor, rites.Floor)
	assert.Equal(t, opinions.Ceiling, rites.Ceiling)
	assert.Equal(t, domain.DefaultChemistryRules().TierRounds[domain.TierTrusted-1], rites.LongServiceRounds)
	assert.False(t, strings.Contains(rites.Phrase(rites.Lost), "god"), "no creed in the words")
}

// Phase 74's folded-in check: a companion brought down in a won fight does
// not live on with a lasting wound. At zero health they die (and are raised
// or mourned), and a death clears the wounds they carried; lasting wounds
// come from critical blows, defeats, errands and scenes, not from falling.
func TestAFallenCompanionKeepsNoWounds(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	record, _ := module.registry.Get(7)
	state := record.Companions[0].State.Clone()
	state.Wounds = []wounds.Wound{{Points: 4}}
	require.NoError(t, module.registry.SetState(7, 1, state))
	killOne(module)
	c := companion(t, module, 1)
	require.True(t, c.Dead())
	assert.Empty(t, c.State.Wounds, "death clears wounds; raising them heals whole")
}
