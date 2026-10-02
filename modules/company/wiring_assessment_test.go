package company

import (
	"fmt"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"slices"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/assessment"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33i1 wiring: the company assessment through the real scout and
// consider commands, and the battle view's outlook, on the brawl world
// (Aria, her four companions, and the five bandits on the road).

// bandits is the bandits' group on the road and what a player types for it.
func (b *brawl) banditGroup() (enemyparty.Group, string) {
	b.t.Helper()
	groups := enemyparty.Groups(b.road)
	for _, g := range groups {
		if !g.Solo() {
			return g, usercommands.GroupKeyword(b.road, g)
		}
	}
	if len(groups) > 0 { // a lone bandit left
		return groups[0], usercommands.GroupKeyword(b.road, groups[0])
	}
	b.t.Fatal("no bandit group on the road")
	return enemyparty.Group{}, ""
}

func (b *brawl) report() assessment.Report {
	b.t.Helper()
	g, _ := b.banditGroup()
	rep, ok := assessment.Gather(b.aria, b.road, g)
	require.True(b.t, ok)
	return rep
}

// snapshot is what an assessment must never change.
type worldSnapshot struct {
	round, turn uint64
	health      map[int]int
	aria        int
	aggro       bool
}

func (b *brawl) snapshot() worldSnapshot {
	s := worldSnapshot{round: util.GetRoundCount(), turn: util.GetTurnCount(), health: map[int]int{}, aria: b.aria.Character.Health, aggro: b.aria.Character.Aggro != nil}
	for _, id := range mobs.GetAllMobInstanceIds() {
		if m := mobs.GetInstance(id); m != nil {
			s.health[id] = m.Character.Health
		}
	}
	return s
}

func TestScoutAndConsiderAssessTheCompany(t *testing.T) {
	b := newBrawl(t)
	_, kw := b.banditGroup()
	before := b.snapshot()

	got := b.cmd("scout", kw)
	assert.Contains(t, got, "as they stand")
	assert.Contains(t, got, "Assessment: ")
	assert.Contains(t, got, " for your company; ")
	assert.Contains(t, got, "Counted: you, Tamsin Reed, Brother Oswin, Garrick Vane and Ysolde.")
	assert.Contains(t, got, "Not judged: spells, healing, guards and abilities, hidden foes, and anyone yet to come.")
	assert.NotRegexp(t, `\d+%`, got, "never a percentage")

	con := b.cmd("consider", kw)
	assert.Contains(t, con, "You consider ")
	assert.Contains(t, con, "Counted: you, Tamsin Reed, Brother Oswin, Garrick Vane and Ysolde.")
	assert.Contains(t, con, "Type scout "+kw+" to see how they stand.")
	assert.NotContains(t, con, "front [", "consider draws no grid")
	headline := b.report().Headline()
	assert.Contains(t, got, headline)
	assert.Contains(t, con, headline)

	// One member's name considers its whole group.
	assert.Contains(t, b.cmd("consider", "bandit captain"), headline)

	after := b.snapshot()
	assert.Equal(t, before, after, "assessing changes nothing and spends no time")
	_, inBattle := battle.Current(7)
	assert.False(t, inBattle, "and starts no fight")
}

func TestAssessmentFollowsTheCompanysState(t *testing.T) {
	b := newBrawl(t)
	whole := b.report()
	require.Len(t, whole.Counted, 5)
	assert.Empty(t, whole.Hurt)

	// Wounds: everyone near death reads worse, and says who is hurt.
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Health = 1
	}
	b.aria.Character.Health = 1
	hurt := b.report()
	assert.Less(t, hurt.Ratio, whole.Ratio)
	assert.Equal(t, assessment.Hopeless, hurt.Risk)
	assert.Contains(t, hurt.Text(), "Hurt: you (near death), Tamsin Reed (near death)")
	for id := 1; id <= 4; id++ {
		c := b.companion(id)
		c.Character.Health = c.Character.HealthMax.Value
	}
	b.aria.Character.Health = b.aria.Character.HealthMax.Value

	// A companion who walked off isn't counted, and is named as away.
	ysolde := b.companion(4)
	b.road.RemoveMob(ysolde.InstanceId)
	verge := rooms.LoadRoom(920102)
	verge.AddMob(ysolde.InstanceId)
	ysolde.Character.RoomId = verge.RoomId
	apart := b.report()
	assert.Len(t, apart.Counted, 4)
	assert.Contains(t, apart.Text(), "Not with you: Ysolde (away).")
	assert.Less(t, apart.Ratio, whole.Ratio, "one fewer fighter")
	verge.RemoveMob(ysolde.InstanceId)
	b.road.AddMob(ysolde.InstanceId)
	ysolde.Character.RoomId = b.road.RoomId

	// Burden lowers a member's dodge: the foes land more.
	tamsin := b.companion(1)
	tamsin.Character.StoreItem(items.New(10013))
	tamsin.Character.StoreItem(items.New(10013))
	laden := b.report()
	assert.Contains(t, laden.Text(), "Burdened among you: Tamsin Reed (heavily burdened).")
	assert.Less(t, laden.Ratio, whole.Ratio)

	// A dismissed companion is no longer in the company at all.
	b.cmd("company", "dismiss ysolde")
	gone := b.report()
	assert.Len(t, gone.Counted, 4)
	assert.NotContains(t, gone.Text(), "Ysolde", "a dismissed companion is no longer in the company")
}

func TestAssessmentReachFollowsTheFormation(t *testing.T) {
	b := newBrawl(t)
	b.unplaced()
	assert.Empty(t, b.report().NoReach, "no one placed: every blow lands, as in combat")
	assert.Empty(t, b.report().OutOfReach)

	// Aria alone, placed, bare-handed: the bruiser and the slinger stand in
	// the bandits' second rank, out of her reach; scout's marks agree.
	b.cmd("company", "dismiss all")
	_, kw := b.banditGroup()
	for _, spot := range []string{"me 2 2"} {
		require.Contains(t, b.cmd("formation", "move "+spot), "Placed", spot)
		scouted := b.cmd("scout", kw)
		rep := b.report()
		assert.Equal(t, []string{"you"}, rep.Counted)
		assert.Equal(t, strings.Contains(scouted, "You can reach none of them from your place"), slices.Contains(rep.NoReach, "you"), spot)
		assert.Contains(t, rep.OutOfReach, "the bandit bruiser", spot)
		assert.Contains(t, rep.OutOfReach, "the bandit slinger", spot)
		assert.Contains(t, scouted, "Out of your company's reach: ", spot)
		assert.NotContains(t, scouted, "*bandit bruiser", "scout marks it out of reach too")
	}
	alone := b.report()

	// Solo clear is refused and assessment still uses the authoritative center.
	_, err := usercommands.TryCommand("formation", "clear me", b.aria.UserId, 0)
	assert.ErrorIs(t, err, domain.ErrSoloFormation)
	assert.Equal(t, alone.Ratio, b.report().Ratio)

}

func TestAssessmentSeesWhatScoutSees(t *testing.T) {
	b := newBrawl(t)
	g, kw := b.banditGroup()
	all := b.report()

	// A hidden bandit isn't counted, named, or reached for.
	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93301, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93301) })
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	require.NoError(t, captain.Character.AddBuff(93301, true))
	captain.Character.Validate()
	require.True(t, captain.Character.HasBuffFlag("hidden"))
	require.Len(t, g.Visible(), 4)
	veiled := b.report()
	assert.Greater(t, veiled.Ratio, all.Ratio, "the hidden captain isn't weighed")
	got := b.cmd("consider", kw)
	assert.NotContains(t, got, "captain")
	assert.Contains(t, b.cmd("consider", "bandit captain"), `You see no enemy called "bandit captain" here.`)

	// In the dark, nothing.
	biome := b.road.Biome
	b.road.Biome = "cave"
	t.Cleanup(func() { b.road.Biome = biome })
	assert.Contains(t, b.cmd("consider", kw), "It's too dark to make them out.")
	assert.NotContains(t, b.cmd("scout", kw), "Assessment")
}

func TestConsiderRefusesWhatIsNoEnemy(t *testing.T) {
	b := newBrawl(t)
	b.bystander()
	assert.Contains(t, b.cmd("consider", "bystander"), "Bystander is no enemy of yours.")
	assert.Contains(t, b.cmd("consider", "tamsin"), "Tamsin Reed travels with you.", "a companion is no enemy")
	assert.Contains(t, b.cmd("consider", "dragon"), `You see no enemy called "dragon" here.`)
	assert.Contains(t, b.cmd("consider", ""), "Consider whom?")
}

func TestAssessmentNotesAnAlly(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(parties.UseMemoryForTest())
	b.bystander()
	assert.False(t, b.report().Allies)
	p := parties.New(7)
	require.NotNil(t, p)
	p.InvitePlayer(8)
	p.AcceptInvite(8)
	rep := b.report()
	assert.True(t, rep.Allies)
	assert.Len(t, rep.Counted, 5, "the ally isn't counted")
	assert.Contains(t, rep.Text(), "Allies here aren't counted.")
}

// TestBattleViewShowsTheOutlook: Company.Battle carries the same headline
// scout gives, in words, and none in the dark.
func TestBattleViewShowsTheOutlook(t *testing.T) {
	b := newBrawl(t)
	views := battleViews(t)
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.refresh(7)
	view := lastView(views, 7)
	require.NotEmpty(t, view)
	outlook, ok := view["outlook"].(map[string]any)
	require.True(t, ok, "the outlook is sent: %v", view)
	assert.Contains(t, []any{"easy", "fair", "hard", "grave", "hopeless"}, outlook["risk"])
	assert.Contains(t, outlook["text"], " for your company; ")
	_, kw := b.banditGroup()
	assert.Contains(t, b.cmd("scout", kw), outlook["text"].(string), "the same words as scout")

	biome := b.road.Biome
	b.road.Biome = "cave"
	t.Cleanup(func() { b.road.Biome = biome })
	b.refresh(7)
	view = lastView(views, 7)
	assert.Equal(t, true, view["dark"])
	assert.Nil(t, view["outlook"], "no outlook in the dark")
}

// 33i1 review findings 1 and 2: members standing here who can't fight are
// named with the reason, and a company with nobody able to fight reads as
// hopeless, not as a coin toss.
func TestAssessmentOfACompanyThatCantFight(t *testing.T) {
	b := newBrawl(t)
	tamsin := b.companion(1)
	tamsin.Character.CombatWithdrawn = true
	t.Cleanup(func() { tamsin.Character.CombatWithdrawn = false })
	rep := b.report()
	assert.Len(t, rep.Counted, 4)
	assert.Contains(t, rep.Text(), "Not with you: Tamsin Reed (out of the fight).")
	assert.NotContains(t, rep.Text(), "(away)", "she stands here")

	b.cmd("company", "dismiss all")
	b.aria.Character.CombatWithdrawn = true
	t.Cleanup(func() { b.aria.Character.CombatWithdrawn = false })
	_, kw := b.banditGroup()
	got := b.cmd("consider", kw)
	assert.Contains(t, got, "Hopeless for your company; the odds look clear.")
	assert.Contains(t, got, "Counted: nobody who can fight. Not with you: you (out of the fight).")
}

// 33i1 review finding 3: a hidden namesake never shadows a visible enemy.
func TestConsiderFindsTheVisibleNamesake(t *testing.T) {
	b := newBrawl(t)
	cutthroats := b.bandits["bandit cutthroat"]
	require.Len(t, cutthroats, 2)
	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93302, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93302) })
	first := mobs.GetInstance(cutthroats[0])
	require.NoError(t, first.Character.AddBuff(93302, true))
	first.Character.Validate()
	_, mobId := b.road.FindByName("bandit cutthroat")
	require.Equal(t, cutthroats[0], mobId, "the room's own lookup lands on the hidden one first")

	got := b.cmd("consider", "bandit cutthroat")
	assert.Contains(t, got, "Assessment:", "the visible cutthroat is found")
	assert.Contains(t, b.cmd("consider", fmt.Sprintf("#%d", cutthroats[0])), "You see no enemy called", "nor by its number")
	assert.Contains(t, b.cmd("consider", fmt.Sprintf("#%d", cutthroats[1])), "Assessment:")
}

// 33i1 review finding 5: a lone creature that means no harm is no enemy.
func TestConsiderRefusesAHarmlessLoner(t *testing.T) {
	b := newBrawl(t)
	fence := mobs.NewMobById(mobs.MobId(9105), b.road.RoomId)
	require.NotNil(t, fence)
	b.road.AddMob(fence.InstanceId)
	t.Cleanup(func() { b.road.RemoveMob(fence.InstanceId); mobs.DestroyInstance(fence.InstanceId) })
	g, ok := enemyparty.GroupOf(b.road, fence.InstanceId)
	require.True(t, ok)
	require.True(t, g.Solo(), "on its own, not in the bandits' spawn group")
	assert.Contains(t, b.cmd("consider", "bandit fence"), "The bandit fence is no enemy of yours.")
}

// Phase 33i2: the assessment says how the group fights together and the
// roles its visible members show: by level (the bandits, level 1 to 4,
// are a rabble), by a template's coordination, or, once fighting it, by
// the battle's tier. A hidden member's role isn't named.
func TestAssessmentNamesTheGroupsCoordination(t *testing.T) {
	b := newBrawl(t)
	_, kw := b.banditGroup()
	assert.Contains(t, b.cmd("scout", kw), "They fight as a rabble.")
	assert.Contains(t, b.cmd("consider", kw), "They fight as a rabble.")

	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	slinger := mobs.GetInstance(b.bandits["bandit slinger"][0])
	captain.Coordination = 3
	captain.Role = "guardian"
	slinger.Role = "healer"
	assert.Contains(t, b.cmd("scout", kw), "They fight as a drilled company: a healer and a guardian among them.")

	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93303, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93303) })
	require.NoError(t, slinger.Character.AddBuff(93303, true))
	slinger.Character.Validate()
	got := b.cmd("consider", kw)
	assert.Contains(t, got, "They fight as a drilled company: a guardian among them.")
	assert.NotContains(t, got, "healer", "a hidden healer isn't named")
}

// Phase 33i2: in a battle the assessment and the battle view's outlook
// give the battle's tier, fixed when it began.
func TestBattleViewShowsTheCoordination(t *testing.T) {
	b := newBrawl(t)
	views := battleViews(t)
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	battle.SetCoordination(7, 4)
	b.refresh(7)
	outlook, ok := lastView(views, 7)["outlook"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "They fight as a veteran company.", outlook["coordination"])
	_, kw := b.banditGroup()
	assert.Contains(t, b.cmd("scout", kw), "They fight as a veteran company.")
}

// 33i2 review finding 5: a hidden member's level doesn't lift the tier
// the assessment names.
func TestAssessmentTierIgnoresHiddenMembers(t *testing.T) {
	b := newBrawl(t)
	_, kw := b.banditGroup()
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	captain.Character.Level = 60 // (60+1+1+1+1)/5 = 12: a band
	assert.Contains(t, b.cmd("scout", kw), "They fight as a band.")
	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93305, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93305) })
	require.NoError(t, captain.Character.AddBuff(93305, true))
	captain.Character.Validate()
	assert.Contains(t, b.cmd("scout", kw), "They fight as a rabble.")
}
