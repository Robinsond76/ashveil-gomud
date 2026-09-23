package light

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

func TestParsePenalty(t *testing.T) {
	assert.Equal(t, 25, parsePenalty(25, 40))
	assert.Equal(t, 40, parsePenalty(nil, 40), "missing config uses the default")
	assert.Equal(t, 0, parsePenalty(0, 40), "an explicit 0 disables the penalty")
	assert.Equal(t, 40, parsePenalty(-3, 40), "a negative value uses the default")
}

func TestLightReportDarkWithoutLight(t *testing.T) {
	out := strings.Join(lightReport(rooms.LightConditions{Night: true}, 0, 0, viewerLight{}, 40), "\n")
	assert.Contains(t, out, "Ambient light here: dark.")
	assert.Contains(t, out, "no moon")
	assert.Contains(t, out, "You carry no light.")
	assert.Contains(t, out, "You can see nothing.")
	assert.Contains(t, out, "-40 to hit")
}

func TestLightReportTorchAndPartyLight(t *testing.T) {
	out := strings.Join(lightReport(rooms.LightConditions{Night: true, Moonlight: 2}, 1, 2, viewerLight{Own: true, Party: true}, 0), "\n")
	assert.Contains(t, out, "Ambient light here: dim.")
	assert.Contains(t, out, "You carry your own light.")
	assert.Contains(t, out, "An ally's light")
	assert.Contains(t, out, "this room and its exits")
	assert.NotContains(t, out, "to hit")
}

func TestLightReportNightVision(t *testing.T) {
	out := strings.Join(lightReport(rooms.LightConditions{DarkBiome: true}, 0, 2, viewerLight{NightVision: true}, 0), "\n")
	assert.Contains(t, out, "see clearly in the dark")
}
