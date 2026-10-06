package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestTargetLostNoticeSkipsAFoeThatJustFell(t *testing.T) {
	const fallen = 987001
	assert.True(t, targetLostNotice(fallen), "a target that simply vanished is reported")
	mobs.TrackRecentDeath(fallen)
	assert.False(t, targetLostNotice(fallen), "a foe that was just beaten ended the fight; no error line")
	assert.True(t, targetLostNotice(0))
}
