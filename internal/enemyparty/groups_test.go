package enemyparty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthWord(t *testing.T) {
	cases := []struct {
		h, max int
		want   string
	}{
		{10, 10, "unhurt"}, {8, 10, "scratched"}, {5, 10, "wounded"}, {3, 10, "badly wounded"},
		{1, 10, "near death"}, {0, 10, "down"}, {5, 0, "unhurt"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, HealthWord(c.h, c.max), "%d/%d", c.h, c.max)
	}
}
