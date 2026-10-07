package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 63: the mercy answer, through the real prompt and command, writes
// the company's deed, once, whichever way it went.
func TestMercyAnswerIsWrittenInTheChronicle(t *testing.T) {
	for _, c := range []struct {
		answer string
		kind   chronicle.Kind
		other  chronicle.Kind
	}{{"yes", chronicle.Spared, chronicle.Executed}, {"no", chronicle.Executed, chronicle.Spared}} {
		t.Run(c.answer, func(t *testing.T) {
			mem := useChronicle(t)
			b, _ := yieldingBrawl(t)
			hooks.MercyTick(events.NewTurn{})
			p := b.aria.GetPrompt()
			require.NotNil(t, p)
			token := p.Rest
			assert.Empty(t, deeds(mem, c.kind), "asking is not a deed")
			p.Questions[0].Answer(c.answer)
			b.cmd("mercy", token)
			got := deeds(mem, c.kind)
			require.Len(t, got, 1)
			assert.NotEmpty(t, got[0].Subject, "it names the foe")
			assert.Contains(t, got[0].Ref, "mob:")
			assert.Empty(t, deeds(mem, c.other))

			require.NoError(t, morale.AnswerMercy(7, token, c.answer))
			assert.Len(t, deeds(mem, c.kind), 1, "a repeated answer writes nothing")
		})
	}
}
