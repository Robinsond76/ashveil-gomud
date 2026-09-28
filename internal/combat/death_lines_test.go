package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestDeathLinesVoice (Phase 29c): every death notice is in the narration
// voice and takes one name.
func TestDeathLinesVoice(t *testing.T) {
	for _, pool := range [][]string{DeathLines, BeatenLines, PlayerDeathLines} {
		assert.NotEmpty(t, pool)
		for _, line := range pool {
			assert.Empty(t, util.NarrationVoiceProblem(line), line)
			assert.Equal(t, 1, strings.Count(line, "%s"), line)
		}
	}
}

func TestDeathLineArticles(t *testing.T) {
	for i := 0; i < 20; i++ {
		assert.True(t, strings.HasPrefix(DeathLine(`<ansi fg="mobname">rat</ansi>`), `The <ansi fg="mobname">rat</ansi> `))
		assert.True(t, strings.HasPrefix(DeathLine("bandit captain"), "The bandit captain "))
		assert.True(t, strings.HasPrefix(PlayerDeathLine("Garrick Vane"), "Garrick Vane "))
	}
	assert.Equal(t, "The straw footman is beaten and yields the field.", BeatenLine("straw footman"))
}
