package tutorial

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fake struct{ calls []int }

func (f *fake) Begin(userID int) bool { f.calls = append(f.calls, userID); return true }

func TestBeginWithoutProvider(t *testing.T) {
	SetProvider(nil)
	assert.False(t, Active())
	assert.False(t, Begin(7))
	f := &fake{}
	SetProvider(f)
	assert.True(t, Active())
	t.Cleanup(func() { SetProvider(nil) })
	assert.True(t, Begin(7))
	assert.Equal(t, []int{7}, f.calls)
}

type viewing struct{ fake }

func (viewing) TutorialView(userID int) (View, bool) {
	return View{Stage: 2, Stages: 8, Title: "Your company"}, userID == 7
}

// Phase 27d: ViewOf reads a provider that is also a Viewer.
func TestViewOf(t *testing.T) {
	SetProvider(nil)
	_, ok := ViewOf(7)
	assert.False(t, ok, "no provider")
	SetProvider(&fake{})
	t.Cleanup(func() { SetProvider(nil) })
	_, ok = ViewOf(7)
	assert.False(t, ok, "a provider that can't show a view")
	SetProvider(&viewing{})
	v, ok := ViewOf(7)
	assert.True(t, ok)
	assert.Equal(t, "Your company", v.Title)
	_, ok = ViewOf(8)
	assert.False(t, ok, "not in the course")
}

func TestChangedFiresOnChanged(t *testing.T) {
	var got []int
	OnChanged.Register(func(userID int) int {
		if userID == 4401 {
			got = append(got, userID)
		}
		return userID
	})
	Changed(4401)
	assert.Equal(t, []int{4401}, got)
}
