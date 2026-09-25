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
