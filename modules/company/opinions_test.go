package company

import (
	"strings"
	"testing"
	"time"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 64: companion opinions, through the real module, the real mercy
// answer and the real feed of the web panel.

// opine sets each companion's personality, alignment and loyalty.
func opine(t *testing.T, set map[int]struct {
	personality string
	alignment   int
	loyalty     int
}) {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	for i, c := range record.Companions {
		s, ok := set[c.ID]
		if !ok {
			continue
		}
		record.Companions[i].Personality = s.personality
		record.Companions[i].Disposition = &domain.Disposition{Alignment: s.alignment, Loyalty: s.loyalty}
	}
	module.registry.Put(record)
}

type opinionSetup = struct {
	personality string
	alignment   int
	loyalty     int
}

func loyaltyOf(t *testing.T, id int) int {
	t.Helper()
	record, _ := module.registry.Get(7)
	for _, c := range record.Companions {
		if c.ID == id {
			return c.Disposition.Loyalty
		}
	}
	t.Fatalf("no companion %d", id)
	return 0
}

func withClock(t *testing.T, now *time.Time) {
	t.Helper()
	module.clock = func() time.Time { return *now }
	t.Cleanup(func() { module.clock = nil })
}

func TestAPersonalityReactsToAChoiceAndLoyaltyMoves(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"grim", 0, 50},     // likes a rough camp
		2: {"cheerful", 0, 50}, // dislikes it
		3: {"devout", 0, 50},   // has no view on it
	})
	bystander := loyaltyOf(t, 4)
	said, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Subject: "Verge", Witnesses: []int{1, 2, 3}})
	require.NoError(t, err)
	require.Len(t, said, 2, "the devout one has no opinion of a rough camp")
	joined := strings.Join(said, "\n")
	assert.Contains(t, joined, "(loyalty +2)")
	assert.Contains(t, joined, "(loyalty -2)")
	assert.Equal(t, 52, loyaltyOf(t, 1))
	assert.Equal(t, 48, loyaltyOf(t, 2))
	assert.Equal(t, 50, loyaltyOf(t, 3))
	assert.Equal(t, bystander, loyaltyOf(t, 4), "a companion nobody named did not witness it")
}

func TestTwoCompanionsNeverSayTheSameWordsAboutAChoice(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"boastful", 0, 50},
		2: {"boastful", 0, 50},
		3: {"stoic", 0, 50},
	})
	said, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Subject: "Verge", Witnesses: []int{1, 2, 3}})
	require.NoError(t, err)
	require.Len(t, said, 3)
	joined := strings.Join(said, "\n")
	assert.Equal(t, 1, strings.Count(joined, "I have standards"), "the second boastful voice does not repeat the first: %q", joined)
	assert.Equal(t, 3, strings.Count(joined, "(loyalty"), "every change still shows, quietly")
	assert.Contains(t, joined, `black-bold">(loyalty`, "the loyalty aside is dim")
	assert.NotContains(t, joined, "claps", "agreeing with a complaint is no friendly clap on the shoulder")
	for _, line := range said {
		assert.NotRegexp(t, `^[^"]* (says|boasts|declares|murmurs), "`, line, "not the old 'Name says,' template")
	}
}

func TestAnOpinionIsGivenOncePerChoiceAndOncePerCooldown(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"grim", 0, 50}})
	choice := opinions.Choice{Kind: opinions.Rough, Op: "camp-1", Witnesses: []int{1}}
	said, err := module.Opinion(7, choice)
	require.NoError(t, err)
	require.Len(t, said, 1)
	again, err := module.Opinion(7, choice)
	require.NoError(t, err)
	assert.Empty(t, again, "the same choice reported twice changes nothing")
	choice.Op = "camp-2"
	again, err = module.Opinion(7, choice)
	require.NoError(t, err)
	assert.Empty(t, again, "the same kind inside its cooldown is not farmed")
	assert.Equal(t, 52, loyaltyOf(t, 1))
	now = now.Add(opinions.Kinds[3].Cooldown + time.Minute) // rough camp
	again, err = module.Opinion(7, choice)
	require.NoError(t, err)
	assert.Len(t, again, 1, "after the cooldown it speaks again")
	assert.Equal(t, 54, loyaltyOf(t, 1))
}

func TestOpinionsStayInsideTheCeilingAndFloor(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"grim", 0, 79},     // liked: one point of room under the ceiling
		2: {"cheerful", 0, 31}, // disliked: one point of room above the floor
		3: {"grim", 0, 80},     // already at the ceiling
		4: {"cheerful", 0, 30}, // already at the floor
	})
	said, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Witnesses: []int{1, 2, 3, 4}})
	require.NoError(t, err)
	assert.Len(t, said, 4, "they still say their piece")
	assert.Equal(t, opinions.Ceiling, loyaltyOf(t, 1))
	assert.Equal(t, opinions.Floor, loyaltyOf(t, 2))
	assert.Equal(t, opinions.Ceiling, loyaltyOf(t, 3))
	assert.Equal(t, opinions.Floor, loyaltyOf(t, 4))
	assert.GreaterOrEqual(t, opinions.Floor, 25, "the floor is above the battle nerve line, so opinions never make a companion hesitate")
	assert.NotContains(t, strings.Join(said, "\n"), "loyalty +0")
}

func TestTheDeadAndTheSeparatedHaveNoOpinion(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"grim", 0, 50}, 2: {"grim", 0, 50}})
	record, _ := module.registry.Get(7)
	for i, c := range record.Companions {
		if c.ID == 1 {
			record.Companions[i].Death = &domain.CompanionDeath{}
		}
	}
	module.registry.Put(record)
	said, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Witnesses: []int{1, 2}})
	require.NoError(t, err)
	assert.Len(t, said, 1)
	assert.Equal(t, 50, loyaltyOf(t, 1))
}

func TestEveryoneWithTheLeaderWitnessesWhenNoOneIsNamed(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"grim", 0, 50}, 2: {"grim", 0, 50}, 3: {"grim", 0, 50}, 4: {"grim", 0, 50},
	})
	said, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough})
	require.NoError(t, err)
	assert.Len(t, said, 4, "all four companions walk with the leader in the brawl")
}

func TestObserversHearTheReactions(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"grim", 0, 50}, 2: {"cheerful", 0, 50}})
	var heard []opinions.Reaction
	opinions.Observe(func(leader int, c opinions.Choice, rs []opinions.Reaction) {
		if leader == 7 && c.Op == "observed" {
			heard = append(heard, rs...)
		}
	})
	_, err := module.Opinion(7, opinions.Choice{Kind: opinions.Rough, Op: "observed", Witnesses: []int{1, 2}})
	require.NoError(t, err)
	require.Len(t, heard, 2)
	assert.Equal(t, opinions.Likes, heard[0].Verdict)
	assert.Equal(t, opinions.Dislikes, heard[1].Verdict)
}

func TestTheMercyAnswerLetsPersonalitiesReact(t *testing.T) {
	b, _ := yieldingBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{
		1: {"devout", 0, 50},  // likes mercy; alignment is silent
		2: {"grim", 0, 50},    // dislikes it
		3: {"devout", 60, 50}, // alignment already approves, so no second reaction
		4: {"wry", 0, 50},     // no view on mercy
	})
	hooks.MercyTick(events.NewTurn{})
	p := b.aria.GetPrompt()
	require.NotNil(t, p)
	token := p.Rest
	p.Questions[0].Answer("yes")
	out := b.cmd("mercy", token)
	assert.Contains(t, out, "(loyalty +2)")
	assert.Contains(t, out, "(loyalty -2)")
	assert.Equal(t, 52, loyaltyOf(t, 1))
	assert.Equal(t, 48, loyaltyOf(t, 2))
	assert.Equal(t, 52, loyaltyOf(t, 3), "morale's own approval of mercy (+2), not a second one")
	assert.Equal(t, morale.Reaction(60, true), 2)
	assert.Equal(t, 50, loyaltyOf(t, 4))
	record, _ := module.registry.Get(7)
	var oswin domain.Companion
	for _, c := range record.Companions {
		if c.ID == 1 {
			oswin = c
		}
	}
	require.NotNil(t, oswin.Opinions)
	require.Len(t, oswin.Opinions.Notes, 1)
	assert.Equal(t, string(opinions.Spare), oswin.Opinions.Notes[0].Kind)
}

func TestTheOpinionPanelNamesLikesDislikesAndWhatTheySaid(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"devout", 0, 85}, 2: {"grim", 0, 10}})
	_, err := module.Opinion(7, opinions.Choice{Kind: opinions.SellRelic, Subject: "the Pale Crown", Witnesses: []int{1}})
	require.NoError(t, err)
	panel, ok := domain.OpinionsOf(7)
	require.True(t, ok)
	require.NotEmpty(t, panel.Members)
	var row domain.OpinionRow
	for _, r := range panel.Members {
		if r.ID == 1 {
			row = r
		}
	}
	assert.Equal(t, "devout", row.Personality)
	assert.Equal(t, "devoted", row.Mood)
	assert.Contains(t, row.Likes, "mercy to the beaten")
	assert.Contains(t, row.Dislikes, "selling relics")
	require.Len(t, row.Notes, 1)
	assert.Equal(t, -1, row.Notes[0].Verdict)
	assert.Equal(t, "the Pale Crown", row.Notes[0].Subject)
	for _, r := range panel.Members {
		if r.ID == 2 {
			assert.Equal(t, "close to leaving", r.Mood)
		}
	}
}

func TestTheOpinionsCommandReadsInWords(t *testing.T) {
	b := newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"stoic", 0, 62}})
	out := b.cmd("opinions", "")
	assert.Contains(t, out, "What your companions think of your choices:")
	assert.Contains(t, out, "stoic: loyal (loyalty 62).")
	assert.Contains(t, out, "Likes: a rough camp")
	assert.Contains(t, out, "Dislikes: a bed at an inn")
	out = b.cmd("company", "opinions")
	assert.Contains(t, out, "stoic: loyal")
	assert.Contains(t, b.cmd("opinions", "nobody-here"), "No companion called")
}

func TestSharingYourOwnFoodWithACompanionIsAChoice(t *testing.T) {
	module, prov, user := mealSetup(t)
	now := time.Unix(1_800_000_000, 0)
	withClockOn(t, module, &now)
	record, _ := module.registry.Get(7)
	record.Companions[0].Personality = "cheerful"
	record.Companions[0].Disposition = &domain.Disposition{Loyalty: 50}
	module.registry.Put(record)
	meal := spec(t, items.ItemSpec{ItemId: 989301, Name: "travel bread", Uses: 3, Subtype: items.Edible, Nutrition: 90})
	meal.UUID = uuid.New(items.UUIDItem)
	user.Character.Items = []items.Item{meal}
	useCargo(t, &fakeCargo{})
	out := module.mealView(user, rooms.NewEmptyRoom(), mealEat)
	assert.NotEmpty(t, prov.fed)
	assert.Contains(t, out, "(loyalty +2)", "the leader's own bread fed a companion, and the cheerful one liked it")
	after, _ := module.registry.Get(7)
	assert.Equal(t, 52, after.Companions[0].Disposition.Loyalty)
}

func TestTheCompanysCargoIsNotYoursToShare(t *testing.T) {
	module, _, user := mealSetup(t)
	now := time.Unix(1_800_000_000, 0)
	withClockOn(t, module, &now)
	record, _ := module.registry.Get(7)
	record.Companions[0].Personality = "cheerful"
	record.Companions[0].Disposition = &domain.Disposition{Loyalty: 50}
	module.registry.Put(record)
	useCargo(t, &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: 989302, Count: 5}}})
	spec(t, items.ItemSpec{ItemId: 989302, Name: "cargo jerky", Subtype: items.Edible, Nutrition: 90})
	out := module.mealView(user, rooms.NewEmptyRoom(), mealEat)
	assert.NotContains(t, out, "loyalty")
}

// withClockOn is withClock for a module the test built.
func withClockOn(t *testing.T, m *CompanyModule, now *time.Time) {
	t.Helper()
	m.clock = func() time.Time { return *now }
}

func TestInspectingACompanionShowsItsTemperamentInWords(t *testing.T) {
	b := newBrawl(t)
	opine(t, map[int]opinionSetup{1: {"boastful", 0, 45}})
	record, _ := module.registry.Get(7)
	name := ""
	for _, c := range record.Companions {
		if c.ID == 1 {
			name = companionName(c)
		}
	}
	out := b.cmd("company", "inspect "+strings.ToLower(name))
	assert.Contains(t, out, "Temperament: boastful, steady (loyalty 45).")
	assert.Contains(t, out, "Likes executing the beaten, a bed at an inn")
	assert.Contains(t, out, "dislikes a rough camp")
}

// Phase 64 review: observers (Phase 65 bonds) hear who saw a choice, silent
// witnesses included, even when the source named no one; a companion
// already deserting has no say.
func TestObserversHearEveryWitnessAndDesertersStayQuiet(t *testing.T) {
	newBrawl(t)
	now := time.Unix(1_800_000_000, 0)
	withClock(t, &now)
	opine(t, map[int]opinionSetup{1: {"grim", 0, 50}, 2: {"wry", 0, 50}, 3: {"stoic", 0, 50}, 4: {"cheerful", 0, 50}})
	record, _ := module.registry.Get(7)
	for i := range record.Companions {
		if record.Companions[i].ID == 3 {
			record.Companions[i].MoraleDesert = true
		}
	}
	module.registry.Put(record)
	var witnesses []int
	var reacted []int
	opinions.Observe(func(leader int, c opinions.Choice, rs []opinions.Reaction) {
		if leader == 7 && c.Op == "witnessed" {
			witnesses = c.Witnesses
			for _, r := range rs {
				reacted = append(reacted, r.CompanionID)
			}
		}
	})
	_, err := module.Opinion(7, opinions.Choice{Kind: opinions.Prudence, Op: "witnessed"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2, 4}, witnesses, "everyone with the leader saw it, the wry and cheerful shrug included; the deserter is gone")
	assert.Equal(t, []int{1}, reacted, "only the grim has a view on the careful way")
}
