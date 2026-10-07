package company

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blessed installs a 10% blessing and registers the user, as a signed-in
// player is, carrying it.
func blessed(t *testing.T, user *users.UserRecord) {
	t.Helper()
	blessings.SetData(&blessings.Data{Blessings: []blessings.Blessing{
		{ID: "keeper", Name: "Keeper", Text: "t", Deed: "joined", Count: 1, Perk: blessings.Perk{Discount: 10}},
	}})
	user.Character.Blessings = []string{"keeper"}
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(user.UserId); blessings.SetData(nil) })
}

// Phase 77: an account blessing's recruit discount shows in the listing and
// the notice, and is what the hire charges. The candidate's own price (and
// a tutorial recruit's freedom) are untouched.
func TestBlessedRecruitPaysLessForAuthoredCandidates(t *testing.T) {
	m, _, user, _ := newRecruitModule(t, 150)
	blessed(t, user)

	assert.Contains(t, m.listing(7, testRecruiters()[hiringRoom]), "Price: 108 gold", "120 less 10%")
	assert.Contains(t, strings.Join(m.RecruiterLines(7, hiringRoom), "\n"), "(108 gold)")
	text, err := m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Contains(t, text, "You pay 108 gold.")
	assert.Equal(t, 42, user.Character.Gold)
	assert.Equal(t, 120, testRecruiters()[hiringRoom].Candidates[1].Price, "the table keeps the full price")

	// It decides whether the buyer can afford the hire.
	user.Character.Gold = 107
	text, err = m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Equal(t, "garrick asks 108 gold, and you have 107.", text)
}

func TestUnblessedRecruitPaysFullPrice(t *testing.T) {
	m, _, user, _ := newRecruitModule(t, 150)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(user.UserId) })
	text, err := m.recruit(user, hiringRoom, "garrick")
	require.NoError(t, err)
	assert.Contains(t, text, "You pay 120 gold.")
}

func TestBlessedRecruitPaysLessForGeneratedCandidates(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 500)
	blessed(t, user)
	m.RecruiterLines(user.UserId, hiringRoom)
	before := rosterOf(t, m, 7)
	c := before.Candidates[0]
	require.Greater(t, c.Price, 10)
	want := blessings.RecruitPrice(user.Character, c.Price)
	require.Less(t, want, c.Price)

	assert.Contains(t, m.inspectAt(7, hiringRoom, c.Key), "They ask "+strconv.Itoa(want)+" gold.")
	assert.Contains(t, strings.Join(m.RecruiterLines(7, hiringRoom), "\n"), c.Name+"</ansi> ("+strconv.Itoa(want)+" gold)")
	text, err := m.recruit(user, hiringRoom, c.Key)
	require.NoError(t, err)
	assert.Contains(t, text, "You pay "+strconv.Itoa(want)+" gold.")
	assert.Equal(t, 500-want, user.Character.Gold)
}
