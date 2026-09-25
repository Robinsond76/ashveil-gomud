package expedition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeProgress struct{ fakeMovementProvider }

func (fakeProgress) JourneyProgress(int) (Progress, bool) {
	return Progress{Percent: 42, Remaining: time.Minute, Route: "oak-road"}, true
}

func TestProgressProviderNone(t *testing.T) {
	SetMovementProvider(nil)
	_, ok := JourneyProgress(7)
	assert.False(t, ok)
	SetMovementProvider(&fakeMovementProvider{})
	t.Cleanup(func() { SetMovementProvider(nil) })
	_, ok = JourneyProgress(7)
	assert.False(t, ok, "a movement provider without progress")
	SetMovementProvider(&fakeProgress{})
	p, ok := JourneyProgress(7)
	assert.True(t, ok)
	assert.Equal(t, 42, p.Percent)
}
