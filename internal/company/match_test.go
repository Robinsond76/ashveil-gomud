package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitNameMatchesSeparatesExactFromSubstring(t *testing.T) {
	names := []string{"Bram", "Brammel", "Old Bram", "Ysolde", ""}
	id := func(s string) string { return s }

	exact, partial := SplitNameMatches(names, "bram", id)
	assert.Equal(t, []string{"Bram"}, exact, "case-insensitive exact")
	assert.Equal(t, []string{"Brammel", "Old Bram"}, partial, "substring hits in order")

	exact, partial = SplitNameMatches(names, "sold", id)
	assert.Empty(t, exact)
	assert.Equal(t, []string{"Ysolde"}, partial)

	exact, partial = SplitNameMatches(names, "nobody", id)
	assert.Empty(t, exact)
	assert.Empty(t, partial)
}
