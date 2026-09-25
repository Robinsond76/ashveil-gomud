package companyview

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setCache(t *testing.T, userID int, values map[string]string) {
	t.Helper()
	cacheMu.Lock()
	cache[userID] = values
	cacheMu.Unlock()
	t.Cleanup(func() {
		cacheMu.Lock()
		delete(cache, userID)
		cacheMu.Unlock()
	})
}

func TestPromptValues(t *testing.T) {
	v := PromptValues(fullSources().summary(testUser()))
	assert.Equal(t, "Hungry", v["{hunger}"])
	assert.Equal(t, "40", v["{hungerv}"])
	assert.Equal(t, "Hydrated", v["{thirst}"])
	assert.Equal(t, "Dim", v["{light}"])
	assert.Equal(t, "Chilled", v["{warmth}"])
	assert.Equal(t, "Burdened", v["{load}"])
	assert.Equal(t, "3, 1 dead", v["{company}"])
	assert.Equal(t, ` <ansi fg="cyan">Resting 12m</ansi>`, v["{activity}"])
	assert.Equal(t, ` <ansi fg="yellow">Hungry Tired Dim Chilled</ansi>`, v["{warn}"])

	quiet := PromptValues(noSources().summary(testUser()))
	assert.Empty(t, quiet["{warn}"])
	assert.Empty(t, quiet["{activity}"])
	assert.Empty(t, quiet["{hunger}"], "unknown renders as nothing")
	assert.Empty(t, quiet["{company}"], "unknown company: nothing, not a count")
}

func TestPromptTokens(t *testing.T) {
	u := testUser()
	setCache(t, u.UserId, map[string]string{"{fatigue}": "Tired", "{company}": "3, 1 dead", "{warn}": " Tired"})
	out := u.ProcessPromptString("[{fatigue}|{company}]{warn}{light}{nosuch}:")
	assert.Equal(t, "[Tired|3, 1 dead] Tired{nosuch}:", out, "an unknown value is empty; an unknown token stays literal")
}

func TestQuietDefaultPromptUnchanged(t *testing.T) {
	u := testUser()
	old := `[HP:{hp}/{HP} MP:{mp}/{MP}]{h}:`
	now := `[HP:{hp}/{HP} MP:{mp}/{MP}]{warn}{activity}{h}:`
	assert.Equal(t, u.ProcessPromptString(old), u.ProcessPromptString(now), "no cache yet")
	setCache(t, u.UserId, PromptValues(noSources().summary(u)))
	assert.Equal(t, u.ProcessPromptString(old), u.ProcessPromptString(now), "nothing to warn")
}

// TestPromptCacheRace: prompts built off the game loop while it refreshes
// the cache (run under -race).
func TestPromptCacheRace(t *testing.T) {
	u := testUser()
	other := users.NewUserRecord(8, 1)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = other.ProcessPromptString("{warn}{activity}{hunger}")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			values := PromptValues(fullSources().summary(u))
			cacheMu.Lock()
			cache[other.UserId] = values
			cacheMu.Unlock()
		}
	}()
	wg.Wait()
	cacheMu.Lock()
	delete(cache, other.UserId)
	cacheMu.Unlock()
}

// TestRealRefreshRace (review coverage): the real game-loop refresh against
// prompts built on another goroutine, under -race.
func TestRealRefreshRace(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(31, 1)
	users.SetTestUser(u)
	t.Cleanup(func() {
		cacheMu.Lock()
		delete(cache, 31)
		cacheMu.Unlock()
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = u.ProcessPromptString("{warn}{company}")
		}
	}()
	for i := 0; i < 200; i++ {
		refreshAll()
	}
	<-done
}

func TestOnRefreshFiresWithSummary(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(32, 1)
	u.Character.Name = "Hook"
	users.SetTestUser(u)
	var got []Refreshed
	OnRefresh.Register(func(r Refreshed) Refreshed {
		if r.User.UserId == 32 {
			got = append(got, r)
		}
		return r
	})
	RefreshUser(32)
	RefreshUser(999) // not online: nothing
	require.Len(t, got, 1)
	assert.Equal(t, "Hook", got[0].Summary.Leader.Name)
	cacheMu.Lock()
	delete(cache, 32)
	cacheMu.Unlock()
}
