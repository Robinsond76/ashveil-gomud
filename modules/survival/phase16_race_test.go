package survival

import (
	"strconv"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Real modules: walking registers the step provider and a NewRound
	// band tick; exposure registers the climate provider (whose exposure
	// read walking takes under the exposure lock) and its NewRound tick.
	_ "github.com/GoMudEngine/GoMud/modules/exposure"
	_ "github.com/GoMudEngine/GoMud/modules/walking"
)

// TestPhase16ConcurrentStepsTimersAndTicks runs, at once, what Phase 16 can
// run concurrently against the real survival module: the game loop taking
// walking steps (walking -> exposure read -> survival drain) and processing
// NewRound ticks (exposure and walking), while timer goroutines apply camp
// and inn rest recovery and expedition checkpoint exertion. Run under -race.
func TestPhase16ConcurrentStepsTimersAndTicks(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	domain.SetCompanyService(m)
	domain.SetMemberDrainService(m)
	t.Cleanup(func() {
		domain.SetCompanyService(nil)
		domain.SetMemberDrainService(nil)
	})

	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "snow", Name: "Snow", Symbol: "*"})
	for _, id := range []int{93101, 93102} {
		rooms.SetTestRoom(&rooms.Room{RoomId: id, Zone: "RaceTundra", Biome: "snow"})
	}
	t.Cleanup(func() {
		rooms.RemoveTestBiome("snow")
		rooms.RemoveTestRoom(93101)
		rooms.RemoveTestRoom(93102)
	})
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 7)
	user.Character.Name = "Hero"
	user.Character.RoomId = 93102
	user.Character.Validate()
	users.SetTestUser(user)

	const n = 200
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { // the game loop
		defer wg.Done()
		for i := 0; i < n; i++ {
			walking.Stepped(7, 93101, 93102)
			if i%5 == 0 {
				events.AddToQueue(events.NewRound{RoundNumber: uint64(i)})
				events.ProcessEvents()
			}
		}
	}()
	go func() { // camp and inn rest timers
		defer wg.Done()
		for i := 0; i < n; i++ {
			_, _ = m.ApplyCompanyRestRecovery(7, "camp-rest-race-"+strconv.Itoa(i), 1)
			_, _ = m.ApplyCompanyRestRecovery(7, "inn-rest-race-"+strconv.Itoa(i), 1)
		}
	}()
	go func() { // expedition checkpoint timers
		defer wg.Done()
		for i := 0; i < n; i++ {
			_, _ = m.ApplyCompanyExertion(7, "expedition-race-"+strconv.Itoa(i), domain.Exertion{Hunger: 1})
		}
	}()
	wg.Wait()

	needs := m.CompanyNeeds(7)
	require.NotEmpty(t, needs)
	assert.Less(t, needs[0].Needs.Hunger, 100, "expedition exertion landed")
}
