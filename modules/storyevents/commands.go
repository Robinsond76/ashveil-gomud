package storyevents

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// eventCommand shows the page waiting on the company again.
func (m *Module) eventCommand(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	if !m.reshow(user.UserId, "") {
		user.SendText("No scene is waiting on your company. Scenes open as you travel: at a ruin, a stranger's fire, a cliff. See <ansi fg=\"command\">help events</ansi>.")
	}
	return true, nil
}

// chooseCommand answers the waiting page.
func (m *Module) chooseCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	rest = strings.TrimSpace(rest)
	if _, _, waiting := m.waiting(user.UserId); !waiting {
		user.SendText("No scene is waiting on your company.")
		return true, nil
	}
	// The web client adds the page token after the number.
	fields := strings.Fields(rest)
	page := ""
	if len(fields) == 2 {
		page = fields[1]
	}
	n, err := 0, error(nil)
	if len(fields) == 0 || len(fields) > 2 {
		err = strconv.ErrSyntax
	} else {
		n, err = strconv.Atoi(fields[0])
	}
	if err != nil {
		user.SendText("Usage: choose <number>. <ansi fg=\"command\">event</ansi> shows the choices again.")
		return true, nil
	}
	m.choose(user.UserId, n, page)
	return true, nil
}
