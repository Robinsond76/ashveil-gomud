package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
)

// Phase 55: a company that finishes a camp rest Frozen wakes with a chill.

type coldProvider struct{ exposure map[string]int }

func (coldProvider) AirTemperatureIn(int) (int, bool) { return 0, false }
func (c coldProvider) ExposureOf(_ int, key string) (int, bool) {
	return c.exposure[key], true
}

type chillRecorder struct{ caught []string }

func (c *chillRecorder) CatchAilment(_ int, key survival.MemberKey, kind string) (bool, error) {
	c.caught = append(c.caught, string(key)+":"+kind)
	return true, nil
}
func (c *chillRecorder) CureAilment(int, survival.MemberKey, string) (bool, error) {
	return false, nil
}

func restWhileCold(t *testing.T, exposure int, inn bool) ([]string, string) {
	t.Helper()
	e, user := heroEnv(t)
	messages := captureMessages(t)
	rec := &chillRecorder{}
	survival.SetAilmentService(rec)
	climate.SetProvider(coldProvider{exposure: map[string]int{string(survival.LeaderMemberKey): exposure}})
	t.Cleanup(func() {
		survival.SetAilmentService(nil)
		climate.SetProvider(nil)
	})
	if inn {
		e.completeInn(t, user)
	} else {
		e.completeCamp(t, user)
	}
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()
	return rec.caught, strings.Join(*messages, "\n")
}

func TestARestInTheColdGivesAChill(t *testing.T) {
	caught, text := restWhileCold(t, -60, false)
	assert.Equal(t, []string{"leader:chill"}, caught)
	assert.Contains(t, text, "has caught a chill")
}

func TestAComfortableRestGivesNoChill(t *testing.T) {
	caught, text := restWhileCold(t, -20, false)
	assert.Empty(t, caught)
	assert.NotContains(t, text, "caught a chill")
}

func TestAnInnStayNeverGivesAChill(t *testing.T) {
	caught, _ := restWhileCold(t, -90, true)
	assert.Empty(t, caught, "a bed under a roof is not a night in the cold")
}
