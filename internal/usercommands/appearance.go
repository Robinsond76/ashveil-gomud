package usercommands

// Ashveil Phase 72a commands: appearance (and appearance edit), lifestory
// (and lifestory choose), and creation, the one-time offer to existing
// characters.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Appearance shows how you look, or with `edit` (at an inn) describes
// yourself again.
func Appearance(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	sub := strings.ToLower(strings.TrimSpace(rest))

	if sub == `edit` {
		d := appearance.Current()
		if d == nil {
			user.SendText(`Looks are not set up in this world.`)
			return true, nil
		}
		if room == nil || !room.HasTag(innTag) {
			if p := user.GetPrompt(); p != nil && p.Command == `appearance` {
				user.ClearPrompt()
				creation.Clear(user.UserId)
			}
			user.SendText(`You can only change your looks at an inn, where there's a mirror and a quiet corner.`)
			return true, nil
		}
		cmdPrompt, isNew := user.StartPrompt(`appearance`, `edit`)
		if isNew {
			user.SendText(``)
			user.SendText(`You settle in front of the mirror. It costs nothing, and your life story stays as it is.`)
		}
		runCreation(cmdPrompt, user, creationModeEdit)
		return true, nil
	}

	c := user.Character
	user.SendText(buildDescriptionPanel(c))
	d := appearance.Current()
	if d == nil || !c.HasLooks() {
		user.SendText(`You haven't described yourself yet. At an inn, <ansi fg="command">appearance edit</ansi> lets you choose your looks.`)
		return true, nil
	}
	user.SendText(looksSummaryText(d, appearance.Looks(c.Looks)))
	user.SendText(`<ansi fg="command">appearance edit</ansi> at an inn changes your looks for free (<ansi fg="command">help appearance</ansi>).`)
	return true, nil
}

// looksSummaryText lists the picks in words.
func looksSummaryText(d *appearance.Data, l appearance.Looks) string {
	var b strings.Builder
	for _, id := range appearance.SingleTraits {
		o, ok := d.Option(id, l[id])
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("  <ansi fg=\"black-bold\">%-10s</ansi> %s\n", id+`:`, o.Name))
	}
	marks := []string{}
	for _, key := range []string{appearance.TraitMark, appearance.KeyMark2} {
		if o, ok := d.Option(appearance.TraitMark, l[key]); ok && o.ID != appearance.NoMark {
			marks = append(marks, o.Name)
		}
	}
	if len(marks) > 0 {
		b.WriteString(fmt.Sprintf("  <ansi fg=\"black-bold\">%-10s</ansi> %s\n", `marks:`, strings.Join(marks, `, `)))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Lifestory shows your backstory (or a player's at a glance), or with
// `choose` writes it, once.
func Lifestory(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	arg := strings.TrimSpace(rest)
	c := user.Character
	d := lifestory.Current()
	if d == nil {
		user.SendText(`Life stories are not set up in this world.`)
		return true, nil
	}

	switch strings.ToLower(arg) {
	case `choose`:
		if c.HasLifeStory() {
			user.SendText(`Your life story is already written, and it stays as it is.`)
			if p := user.GetPrompt(); p != nil && p.Command == `lifestory` {
				user.ClearPrompt()
				creation.Clear(user.UserId)
			}
			return true, nil
		}
		cmdPrompt, isNew := user.StartPrompt(`lifestory`, `choose`)
		if isNew {
			user.SendText(``)
			user.SendText(`Your life story can be written once. It gives a little: a few points of stat, a keepsake, perhaps a skill.`)
		}
		runCreation(cmdPrompt, user, creationModeStory)
		return true, nil
	case ``:
		if !c.HasLifeStory() {
			user.SendText(`Your life story hasn't been written. <ansi fg="command">lifestory choose</ansi> writes it, once (<ansi fg="command">help lifestory</ansi>).`)
			return true, nil
		}
		_, sp := c.LooksPronouns()
		user.SendText(``)
		user.SendText(`  ` + d.Backstory(lifestory.Picks(c.LifeStory), c.Name, sp))
		user.SendText(`  ` + lifeStoryEffectsLine(d, lifestory.Picks(c.LifeStory)))
		user.SendText(``)
		return true, nil
	}

	// lifestory <player>: who they are at a glance.
	if room != nil {
		if playerId, _ := room.FindByName(arg, rooms.FindAll); playerId > 0 {
			if other := users.GetByUserId(playerId); other != nil {
				user.SendText(lifestoryGlance(d, other.Character.Name, lifestory.Picks(other.Character.LifeStory)))
				return true, nil
			}
		}
	}
	user.SendText(fmt.Sprintf(`You don't see "%s" here. <ansi fg="command">lifestory</ansi> shows your own.`, arg))
	return true, nil
}

func lifestoryGlance(d *lifestory.Data, name string, picks lifestory.Picks) string {
	if len(picks) == 0 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> keeps their past to themselves.`, name)
	}
	parts := []string{}
	for _, id := range lifestory.Stages {
		if o, ok := d.Option(id, picks[id]); ok {
			parts = append(parts, o.Name)
		}
	}
	return fmt.Sprintf(`<ansi fg="username">%s</ansi>: %s.`, name, strings.Join(parts, `, `))
}

// Creation offers an existing character the looks and life story steps
// (the login offer sends it, and a player may type it).
func Creation(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user.Character.RoomId == -1 {
		user.SendText(`Finish creating your character first.`)
		return true, nil
	}
	cmdPrompt, isNew := user.StartPrompt(`creation`, ``)
	if isNew {
		user.SendText(``)
	}
	if runCreation(cmdPrompt, user, creationModeLegacy) == creationUnavailable {
		user.ClearPrompt()
		user.SendText(`There is nothing left to write. <ansi fg="command">appearance</ansi> and <ansi fg="command">lifestory</ansi> show what you chose.`)
	}
	return true, nil
}
