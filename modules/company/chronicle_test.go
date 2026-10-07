package company

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useChronicle installs an in-memory chronicle for the test.
func useChronicle(t *testing.T) *chronicle.Memory {
	t.Helper()
	mem := chronicle.NewMemory()
	chronicle.SetProvider(mem)
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	return mem
}

func deeds(mem *chronicle.Memory, kind chronicle.Kind) []chronicle.Entry {
	return mem.Log(7).Query(chronicle.Filter{Kinds: []chronicle.Kind{kind}})
}

// Phase 63: each way the roster changes is written in the chronicle by the
// real command or listener that changed it.

func TestARecruitJoiningIsWritten(t *testing.T) {
	mem := useChronicle(t)
	m, _, user, _ := newRecruitModule(t, 50)
	_, err := m.recruit(user, hiringRoom, "tamsin")
	require.NoError(t, err)
	got := deeds(mem, chronicle.Joined)
	require.Len(t, got, 1)
	assert.Equal(t, "mob:961", got[0].Ref)
	// The test world defines no mob 961, so the name falls back; a real
	// template names itself.
	assert.Equal(t, []string{"A companion"}, got[0].Members)
	assert.Contains(t, chronicle.Prose(got[0]), "joined the company")

	// A recruit that fails writes nothing.
	_, err = m.recruit(user, hiringRoom, "tamsin")
	require.NoError(t, err)
	assert.Len(t, deeds(mem, chronicle.Joined), 1, "a refused recruit is not a deed")
}

func TestDismissingOneOrAllIsWritten(t *testing.T) {
	mem := useChronicle(t)
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, Name: "Oswin"}, {ID: 2, MobTemplateID: 58, Name: "Tamsin"}, {ID: 3, MobTemplateID: 58, Name: "Ysolde"}}},
	}}, &fakeRuntime{})
	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	got := deeds(mem, chronicle.Dismissed)
	require.Len(t, got, 1)
	assert.Equal(t, []string{"Oswin"}, got[0].Members)
	assert.Equal(t, []string{"companion:1"}, got[0].Keys, "the key tells apart companions who share a name")

	_, err = module.dismiss(7, "all")
	require.NoError(t, err)
	got = deeds(mem, chronicle.Dismissed)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"Tamsin", "Ysolde"}, got[0].Members, "one deed names everyone sent away")
	assert.Equal(t, []string{"companion:2", "companion:3"}, got[0].Keys)
	assert.Equal(t, "Tamsin and Ysolde were sent away.", chronicle.Prose(got[0]))

	_, err = module.dismiss(7, "all")
	require.NoError(t, err)
	assert.Len(t, deeds(mem, chronicle.Dismissed), 2, "dismissing no one writes nothing")
}

func TestDesertionIsWrittenAsADesertionNotADismissal(t *testing.T) {
	mem := useChronicle(t)
	world := newFakeWorld()
	world.leaders[7] = 100
	useFakeLifecycle(t, &fakeLifecycle{})
	module, runtime := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			withDisposition(domain.Companion{ID: 1, MobTemplateID: 58, Name: "Oswin"}, -100, 28),
			withDisposition(domain.Companion{ID: 2, MobTemplateID: 58}, 100, 90),
		}},
	}}, world)
	module.setInstance(7, 1, 99)
	runtime.live = map[int]bool{99: true}
	runRounds(module, defaultDriftEveryRounds*6)
	got := deeds(mem, chronicle.Deserted)
	require.Len(t, got, 1)
	assert.Equal(t, []string{"Oswin"}, got[0].Members)
	assert.Empty(t, deeds(mem, chronicle.Dismissed))
}

func TestACompanionsDeathResurrectionAndLossAreWritten(t *testing.T) {
	mem := useChronicle(t)
	module, _, _, clock := newDeathModule(t)
	killOne(module)
	fell := deeds(mem, chronicle.Fell)
	require.Len(t, fell, 1)
	assert.Equal(t, "mob:"+strconv.Itoa(deathTemplate), fell[0].Ref)
	assert.Equal(t, []string{"companion:1"}, fell[0].Keys)
	module.onMobDeath(eventsMobDeath(101))
	assert.Len(t, deeds(mem, chronicle.Fell), 1, "a second death event for the same companion adds nothing")

	_, err := module.ResurrectCompanion(7, "#1", 2007)
	require.NoError(t, err)
	assert.Len(t, deeds(mem, chronicle.Raised), 1)
	assert.Contains(t, chronicle.Prose(deeds(mem, chronicle.Raised)[0]), "was raised from the dead")

	// #2 falls and its allowance runs out.
	module.onMobDeath(eventsMobDeath(102))
	require.Len(t, deeds(mem, chronicle.Fell), 2)
	for i := 0; i < 1000 && len(deeds(mem, chronicle.Lost)) == 0; i++ {
		clock.advance(4 * 1e9)
		module.chargeAllowances()
	}
	require.Len(t, deeds(mem, chronicle.Lost), 1, "an allowance that runs out writes the loss")
	assert.Contains(t, chronicle.Prose(deeds(mem, chronicle.Lost)[0]), "lost for good")
}

func TestAPromotionIsWrittenForThePlayerAndForACompanion(t *testing.T) {
	mem := useChronicle(t)
	w, _ := classBrawl(t, 10, 45)
	w.cmd("class", "promote priest")
	assert.Empty(t, deeds(mem, chronicle.Promoted), "a preview is not a deed")
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "You are now a Priest.")
	got := deeds(mem, chronicle.Promoted)
	require.Len(t, got, 1)
	assert.Equal(t, "class:priest", got[0].Ref)
	assert.Equal(t, "Priest", got[0].Subject)
	assert.Equal(t, []string{w.aria.Character.Name}, got[0].Members)
	assert.Equal(t, []string{"leader"}, got[0].Keys)
	w.cmd("class", "promote priest confirm")
	assert.Len(t, deeds(mem, chronicle.Promoted), 1, "a repeated confirmation writes nothing")
}

func TestACompanionsPromotionIsWrittenWithItsKey(t *testing.T) {
	mem := useChronicle(t)
	w, _ := classBrawl(t, 3, 0)
	w.companion(2).Character.Level = 10
	setCompanionAlignment(t, 2, 40)
	w.respawn()
	require.Contains(t, w.cmd("class", "promote #2 priest confirm"), "Brother Oswin is now a Priest.")
	got := deeds(mem, chronicle.Promoted)
	require.Len(t, got, 1)
	assert.Equal(t, []string{"Brother Oswin"}, got[0].Members)
	assert.Equal(t, []string{"companion:2"}, got[0].Keys)
	assert.Contains(t, chronicle.Prose(got[0]), "Brother Oswin became a Priest")
}

func eventsMobDeath(instance int) events.MobDeath {
	return events.MobDeath{InstanceId: instance, Level: 5}
}
