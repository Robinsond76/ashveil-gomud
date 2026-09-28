package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
)

// Phase 32a: a recruiter room's notice as look shows it, per viewer.
func TestRecruiterLines(t *testing.T) {
	plain := func(lines []string) []string {
		out := make([]string, len(lines))
		for i, l := range lines {
			out[i] = companyTagPattern.ReplaceAllString(l, "")
		}
		return out
	}
	hint := "  Type company recruit to see them, or company inspect [name]."

	t.Run("free and priced", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		assert.Equal(t, []string{"On the hiring slate: tamsin (free), garrick (120 gold).", hint}, plain(m.RecruiterLines(7, hiringRoom)))
	})
	t.Run("claimed once-only offer left off", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		_ = m.registry.Claim(7, 961)
		assert.Equal(t, []string{"On the hiring slate: garrick (120 gold).", hint}, plain(m.RecruiterLines(7, hiringRoom)))
		assert.Contains(t, plain(m.RecruiterLines(8, hiringRoom))[0], "tamsin (free)", "another player's list is their own")
	})
	t.Run("in the company left off", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		m.registry.Put(domain.Record{LeaderUserID: 7, NextCompanionID: 2, Companions: []domain.Companion{{ID: 1, MobTemplateID: 963}}})
		assert.Equal(t, []string{"On the hiring slate: tamsin (free).", hint}, plain(m.RecruiterLines(7, hiringRoom)))
	})
	t.Run("refused", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		w := m.world.(*fakeWorld)
		w.templates[963] = 90
		w.leaders[7] = -60
		assert.Equal(t, []string{"On the hiring slate: tamsin (free), garrick (won't join you).", hint}, plain(m.RecruiterLines(7, hiringRoom)))
	})
	t.Run("none left", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		_ = m.registry.Claim(7, 961)
		m.registry.Put(domain.Record{LeaderUserID: 7, NextCompanionID: 2, Claimed: []int{961}, Companions: []domain.Companion{{ID: 1, MobTemplateID: 963}}})
		assert.Equal(t, []string{"No one on the hiring slate is looking for work with you now."}, plain(m.RecruiterLines(7, hiringRoom)))
	})
	t.Run("not a recruiter room", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		assert.Empty(t, m.RecruiterLines(7, 1))
		_, ok := m.LookCandidate(7, 1, "tamsin")
		assert.False(t, ok)
	})
	t.Run("look at a candidate", func(t *testing.T) {
		m, _, _, _ := newRecruitModule(t, 0)
		text, ok := m.LookCandidate(7, hiringRoom, "garrick")
		assert.True(t, ok)
		assert.Contains(t, companyTagPattern.ReplaceAllString(text, ""), "company inspect garrick")
		_, ok = m.LookCandidate(7, hiringRoom, "nobody")
		assert.False(t, ok)
	})
}
