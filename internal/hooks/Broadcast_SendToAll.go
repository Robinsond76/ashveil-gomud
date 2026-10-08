package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Checks whether their level is too high for a guide
func Broadcast_SendToAll(e events.Event) events.ListenerReturn {

	broadcast, typeOk := e.(events.Broadcast)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "Broadcast", "Actual Type", e.Type())
		return events.Continue
	}

	textOut := ``
	if len(broadcast.Text) > 0 {
		textOut = templates.AnsiParse(broadcast.Text)
	}

	textOutSR := ``
	if len(broadcast.TextScreenReader) > 0 {
		textOutSR = templates.AnsiParse(broadcast.TextScreenReader)
	}

	for _, u := range users.GetAllActiveUsers() {

		// Ambient news (a sunset) would break into the battle log, so it
		// waits for the battle to end.
		if broadcast.HoldInBattle {
			if _, inBattle := battle.Current(u.UserId); inBattle {
				text := broadcast.Text
				if u.ScreenReader && len(broadcast.TextScreenReader) > 0 {
					text = broadcast.TextScreenReader
				}
				heldAmbient[u.UserId] = append(heldAmbient[u.UserId], text)
				continue
			}
		}

		// Ashveil Phase 29f: a broadcast a combat round caused ("X has
		// DIED!", a level gained by a kill) waits its turn among a pacing
		// player's held lines. A line-refresh-free broadcast (a prompt, a
		// countdown) never waits.
		if !broadcast.SkipLineRefresh {
			text := broadcast.Text
			if u.ScreenReader && len(broadcast.TextScreenReader) > 0 {
				text = broadcast.TextScreenReader
			}
			if len(text) > 0 && holdBehindCombat(u, text, broadcast.IsCommunication) {
				continue
			}
		}

		events.AddToQueue(events.RedrawPrompt{UserId: u.UserId}, 100)

		if u.ScreenReader {

			if len(textOutSR) > 0 {

				if broadcast.SkipLineRefresh {
					connections.SendTo(
						[]byte(textOutSR),
						u.ConnectionId(),
					)
				} else {
					connections.SendTo(
						[]byte(term.AnsiMoveCursorColumn.String()+term.AnsiEraseLine.String()+textOutSR),
						u.ConnectionId(),
					)
				}

				continue
			}

		}

		if broadcast.SkipLineRefresh {
			connections.SendTo(
				[]byte(textOut),
				u.ConnectionId(),
			)
		} else {
			connections.SendTo(
				[]byte(term.AnsiMoveCursorColumn.String()+term.AnsiEraseLine.String()+textOut),
				u.ConnectionId(),
			)
		}
	}

	return events.Continue
}

// heldAmbient is the ambient broadcasts waiting for each player's battle to end.
var heldAmbient = map[int][]string{}

// releaseAmbient tells a player what the world did while they fought.
func releaseAmbient(userId int) {
	held := heldAmbient[userId]
	delete(heldAmbient, userId)
	if len(held) == 0 {
		return
	}
	if u := users.GetByUserId(userId); u != nil {
		for _, text := range held {
			u.SendText(text)
		}
	}
}
