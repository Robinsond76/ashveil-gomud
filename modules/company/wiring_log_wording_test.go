package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 88: a real fight's log names the kind of weapon ("Oswin's mace"),
// not the item ("Oswin's acolyte's mace"), has no unannounced "rage
// subsides".
func TestFightLogWording(t *testing.T) {
	b := newBrawl(t)
	b.aimAt("bandit cutthroat")
	log := companyTagPattern.ReplaceAllString(b.fightAll(80), "")
	assert.NotContains(t, log, "acolyte's mace")
	assert.NotContains(t, log, "crude cudgel")
	assert.NotContains(t, log, "guardsman's broadsword")
	assert.Regexp(t, `Oswin's mace `, log)
	assert.NotContains(t, log, "rage subsides")
}
