package company

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 49 wiring: a recruit is rolled a personality; after a won battle,
// and at a camp rest, companions talk; `set banter off` silences them.

// spoken matches a quoted line of speech (the quote holds no markup).
var spoken = regexp.MustCompile(`"[^"<>]{8,}"`)

func banterPercents(t *testing.T, battle, camp, rested int) {
	t.Helper()
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["Modules.company.BanterBattlePercent"] = battle
	flat["Modules.company.BanterCampStartPercent"] = camp
	flat["Modules.company.BanterRestedPercent"] = rested
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
}

func battleWon(b *brawl, outcome string) string {
	*b.messages = nil
	events.AddToQueue(events.BattleEnded{UserId: b.aria.UserId, Outcome: outcome})
	events.ProcessEvents()
	return companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
}

func TestRecruitsRollAPersonality(t *testing.T) {
	b := newBrawl(t)
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	require.Len(t, record.Companions, 4)
	for _, c := range record.Companions {
		assert.True(t, banter.ValidPersonality(c.Personality), "companion %d rolled %q", c.ID, c.Personality)
	}
	// It is saved once and kept.
	first := record.Companions[0].Personality
	require.NoError(t, module.registry.SetCompanionPersonality(7, record.Companions[0].ID, "grim"))
	after, _ := module.registry.Get(7)
	assert.Equal(t, first, after.Companions[0].Personality)
	_ = b
}

func TestLegacyCompanionPersonalityIsStable(t *testing.T) {
	c := domain.Companion{ID: 3}
	assert.Equal(t, personalityOf(c), personalityOf(c))
	assert.True(t, banter.ValidPersonality(personalityOf(c)))
	c.Personality = "boastful"
	assert.Equal(t, "boastful", personalityOf(c))
}

func TestWonBattleCompanionsTalk(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 100, 100, 100)
	out := battleWon(b, combatstream.OutcomeVictory)
	require.Regexp(t, spoken, out, "companions spoke after the victory")
	// Count spoken lines, not quotes: since camp talk reads like a book
	// (#203) a line can break its speech around the speaker ("Not one
	// wound," Garrick announces. "A masterpiece."), two quotes on one line.
	lines := 0
	for _, line := range strings.Split(out, "\n") {
		if spoken.MatchString(line) {
			lines++
		}
	}
	assert.GreaterOrEqual(t, lines, 2)
	assert.LessOrEqual(t, lines, 4, out)
	record, _ := module.registry.Get(7)
	named := 0
	for _, c := range record.Companions {
		if strings.Contains(out, companionName(c)) {
			named++
		}
	}
	assert.GreaterOrEqual(t, named, 2, "at least two members spoke")
	assert.NotEmpty(t, module.LastBanter(7), "the web client's Camp tab reads the latest exchange")
}

func TestOnlyAVictoryDrawsBanter(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 100, 100, 100)
	for _, outcome := range []string{combatstream.OutcomeDefeat, combatstream.OutcomeBrokenOff, "fled"} {
		assert.NotRegexp(t, spoken, battleWon(b, outcome), outcome)
	}
}

func TestBattleBanterIsOccasional(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 0, 100, 100)
	for i := 0; i < 20; i++ {
		assert.NotRegexp(t, spoken, battleWon(b, combatstream.OutcomeVictory))
	}
	banterPercents(t, 33, 100, 100)
	talked := 0
	for i := 0; i < 300; i++ {
		if spoken.MatchString(battleWon(b, combatstream.OutcomeVictory)) {
			talked++
		}
	}
	assert.Greater(t, talked, 50, "roughly a third of 300 victories")
	assert.Less(t, talked, 160)
}

func TestBanterOffSilencesTheCompany(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 100, 100, 100)
	b.aria.SetConfigOption(banter.OptionKey, false)
	assert.NotRegexp(t, spoken, battleWon(b, combatstream.OutcomeVictory))
	assert.Nil(t, module.CampBanter(7, banter.CtxCamp))
	b.aria.SetConfigOption(banter.OptionKey, true)
	assert.Regexp(t, spoken, battleWon(b, combatstream.OutcomeVictory))
}

func TestCampBanterThroughTheProvider(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 0, 100, 0)
	said := domain.CampBanter(7, banter.CtxCamp)
	require.GreaterOrEqual(t, len(said), 2)
	assert.Equal(t, said, domain.LastBanter(7))
	assert.Nil(t, domain.CampBanter(7, banter.CtxRested), "the rested roll is 0%")
	assert.Nil(t, domain.CampBanter(7, banter.CtxWin), "only camp contexts")
}

func TestFallenAndAbsentMembersDoNotTalk(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 0, 100, 100)
	record, _ := module.registry.Get(7)
	for i := range record.Companions[1:] {
		record.Companions[i+1].Death = &domain.CompanionDeath{}
	}
	module.registry.Put(record)
	assert.Nil(t, module.CampBanter(7, banter.CtxCamp), "one member left cannot talk to anyone")
}

func TestBanterContexts(t *testing.T) {
	assert.Equal(t, []string{"fall", "win"}, banterContexts(true, 0.1))
	assert.Equal(t, []string{"close", "win"}, banterContexts(false, 0.25))
	assert.Equal(t, []string{"close", "win"}, banterContexts(false, 0.3))
	assert.Equal(t, []string{"win"}, banterContexts(false, 0.6))
	assert.Equal(t, []string{"flawless", "win"}, banterContexts(false, 0.9))
	assert.Equal(t, []string{"flawless", "win"}, banterContexts(false, 1))
}

func TestFallenCompanionIsNoticedOnce(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	record, _ := module.registry.Get(7)
	user := b.aria
	assert.Equal(t, "", module.newlyFallen(user))
	// Review: one already dead (awaiting resurrection, or across a
	// restart) is not a new fall.
	record.Companions[1].Death = &domain.CompanionDeath{}
	module.registry.Put(record)
	assert.Equal(t, "", module.newlyFallen(user), "an old death is not mourned again")
	// A death recorded in the fight is.
	module.recordCompanionDeath(7, record.Companions[0].ID, events.MobDeath{})
	assert.Equal(t, banter.ShortName(companionName(record.Companions[0])), module.newlyFallen(user), "a new death")
	assert.Equal(t, "", module.newlyFallen(user), "the same death is not a new fall")
	// One raised again before the battle ends is not mourned.
	module.recordCompanionDeath(7, record.Companions[2].ID, events.MobDeath{})
	again, _ := module.registry.Get(7)
	again.Companions[2].Death = nil
	module.registry.Put(again)
	assert.Equal(t, "", module.newlyFallen(user))
}

func TestPurgeForgetsBanter(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 0, 100, 0)
	require.NotNil(t, module.CampBanter(7, banter.CtxCamp))
	module.banter.forget(7)
	assert.Empty(t, module.LastBanter(7))
}

// Review: a companion killed in the fight is mourned at the victory's end
// with fall lines, never with a plain win line.
func TestCompanionKilledInTheFightDrawsFallLines(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	banterPercents(t, 100, 100, 100)
	pool, err := banter.Load()
	require.NoError(t, err)
	byID := map[string]banter.Line{}
	for _, l := range pool.Lines() {
		byID[l.ID] = l
	}
	record, _ := module.registry.Get(7)
	module.recordCompanionDeath(7, record.Companions[0].ID, events.MobDeath{})
	out := battleWon(b, combatstream.OutcomeVictory)
	require.Regexp(t, spoken, out)
	said := module.LastBanter(7)
	require.NotEmpty(t, said)
	for _, s := range said {
		assert.NotEqual(t, record.Companions[0].ID, s.Member, "the dead do not speak")
		l := byID[s.LineID]
		if l.Reply == "" {
			assert.Contains(t, l.Ctx, banter.CtxFall, "%q is not a fall line", s.Text)
		}
	}
}

// 38e review: a hound and a stone golem keep the company but never speak at
// the fire.
func TestCreaturesDoNotTalk(t *testing.T) {
	creatureBrawl(t)
	banterPercents(t, 0, 100, 0)
	for range 20 {
		for _, s := range module.CampBanter(7, banter.CtxCamp) {
			assert.NotContains(t, []int{5, 6}, s.Member, "%s spoke", s.Name)
		}
	}
}
