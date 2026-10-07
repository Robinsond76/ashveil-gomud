package usercommands

// Ashveil Phase 77: the Iron (Hardcore) option and the account blessings,
// both settled as creation ends. `start` asks the Iron question after the
// looks and life story, once; the answer is final. The web client's
// creation panel shows the same question (a creation.View of step "iron")
// and answers with the same input a telnet player types.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	ironTitle   = `Take the Iron option?`
	ironConfirm = `Iron is for this character's whole life and can't be changed. Take it?`
)

func ironOptions() []creation.Option {
	return []creation.Option{
		{ID: `standard`, Name: `standard`, Text: `A defeat costs one level, and you may be rescued, captured or robbed.`},
		{ID: `iron`, Name: `Iron`, Text: `A defeat always costs two levels (never rescue or capture). Foes are no stronger. You wear the Iron badge, and Iron characters can earn extra blessings.`},
	}
}

// startIronStep asks the Iron question inside `start`: it reports true
// while a question is open (Start returns), false once it is answered.
func startIronStep(cmdPrompt *prompt.Prompt, user *users.UserRecord) bool {
	c := user.Character
	if c.IronOffered() {
		return false
	}
	r := &creationRun{p: cmdPrompt, user: user, mode: creationModeNew, app: appearance.Current(), story: lifestory.Current(), st: creationStateOf(cmdPrompt)}

	if picked, _ := cmdPrompt.Recall(`iron-pick`); picked != true {
		id, status := r.choose(`iron`, `iron`, ironTitle, 0, ironOptions(), false)
		if status != askPicked {
			return true
		}
		if id == `standard` {
			finishIron(r, false)
			return false
		}
		cmdPrompt.Store(`iron-pick`, true)
	}

	opts := []creation.Option{{ID: `yes`, Name: `yes, take Iron`}, {ID: `no`, Name: `no, standard`}}
	id, status := r.choose(`iron`, `iron-confirm`, ironConfirm, 0, opts, false)
	if status != askPicked {
		return true
	}
	if id == `no` {
		finishIron(r, false)
		return false
	}
	finishIron(r, true)
	return false
}

func finishIron(r *creationRun, iron bool) {
	c := r.user.Character
	c.SetIron(iron)
	c.MarkIronOffered()
	r.p.Store(`iron-pick`, false)
	creation.Clear(r.user.UserId)
	if iron {
		r.user.SendText(`<ansi fg="yellow-bold">You take the Iron.</ansi> Your badge will show on <ansi fg="command">online</ansi> and your character sheet. See <ansi fg="command">help hardcore</ansi>.`)
	}
}

// giveBlessings gives a new character the account's earned blessings, once:
// the blessings it carries are kept on the character, so a repeat gives
// nothing twice. It tells the player what they got.
func giveBlessings(user *users.UserRecord) {
	given := blessings.Apply(user.Character, blessings.EarnedFor(user.UserId))
	if len(given) == 0 {
		return
	}
	lines := make([]string, 0, len(given))
	for _, g := range given {
		lines = append(lines, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>: %s`, g.Blessing.Name, g.Blessing.PerkText()))
	}
	user.SendText(`<ansi fg="yellow-bold">Your earlier road follows you:</ansi> ` + strings.Join(lines, `; `) + `. <ansi fg="black-bold">(blessings)</ansi>`)
}
