package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookRendersTravelViewBeforeRoom(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)

	viewer := &fakeViewer{handled: true}
	expedition.SetViewProvider(viewer)
	t.Cleanup(func() { expedition.SetViewProvider(nil) })

	// A nil room proves the travel view short-circuits before ordinary
	// room rendering, which would otherwise dereference the room.
	handled, err := Look("", user, nil, 0)
	require.NoError(t, err)
	assert.True(t, handled)
	require.Equal(t, 1, viewer.calls)
	assert.Equal(t, 7, viewer.last)
}

func TestLookPropagatesTravelViewError(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)

	viewer := &fakeViewer{handled: true, err: assert.AnError}
	expedition.SetViewProvider(viewer)
	t.Cleanup(func() { expedition.SetViewProvider(nil) })

	handled, err := Look("", user, nil, 0)
	require.ErrorIs(t, err, assert.AnError)
	assert.True(t, handled)
}
