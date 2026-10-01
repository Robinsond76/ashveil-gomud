package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Mercy(rest string, u *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	p := u.GetPrompt()
	if p == nil || p.Command != "mercy" || p.Rest != rest || morale.AnswerMercy == nil {
		return true, nil
	}
	if len(p.Questions) != 1 {
		return true, nil
	}
	q := p.Questions[0]
	if !q.Done {
		return true, nil
	}
	return true, morale.AnswerMercy(u.UserId, rest, q.Response)
}
