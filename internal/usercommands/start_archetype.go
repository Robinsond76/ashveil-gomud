package usercommands

// Ashveil Phase 22a: the archetype step of character creation. It only
// uses the internal/archetypes seam; the archetype module owns the choice,
// its grants, and the starter kit.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	archetypeQuestion   = `Which archetype will you follow?`
	archetypeSkippedKey = `archetype-skipped`
)

// startArchetypeStep runs the archetype step inside the start prompt.
// done is true when Start should return now (waiting on an answer); restart
// additionally asks Start to clear the prompt and begin again. It does
// nothing without a provider that creates, when no archetypes are
// configured, once the character has an archetype, or once it has run in
// this prompt (after a commit or a failed one).
func startArchetypeStep(cmdPrompt *prompt.Prompt, user *users.UserRecord) (restart, done bool) {
	if _, skipped := cmdPrompt.Recall(archetypeSkippedKey); skipped {
		return false, false
	}
	choices := archetypes.CreationChoices(user.UserId)
	if len(choices) == 0 {
		return false, false
	}

	question := cmdPrompt.Ask(archetypeQuestion, []string{})
	if !question.Done {
		user.SendText(archetypeChoicesText(choices))
		return false, true
	}

	picked, ok := matchArchetypeChoice(choices, question.Response)
	if !ok {
		question.RejectResponse()
		user.SendText(`That isn't one of the archetypes. Answer with its number or name.`)
		user.SendText(archetypeChoicesText(choices))
		return false, true
	}

	confirm := cmdPrompt.Ask(fmt.Sprintf(`Become a %s? The choice is permanent.`, picked.Name), []string{`yes`, `no`}, `no`)
	if !confirm.Done {
		return false, true
	}
	if confirm.Response != `yes` {
		// Same as declining a name: start the prompt over. Race and name
		// are already on the character, so it resumes at this step.
		return true, true
	}

	text, committed := archetypes.ChooseAtCreation(user.UserId, picked.ID)
	if text != `` {
		user.SendText(text)
	}
	if !committed {
		user.SendText(`You can choose later with "archetype choose <name>".`)
	}
	// Either way the step is over for this prompt. The questions above are
	// cached by text, so re-running it would replay the old answers.
	cmdPrompt.Store(archetypeSkippedKey, true)
	user.SendText(``)
	return false, false
}

func archetypeChoicesText(choices []archetypes.Choice) string {
	var b strings.Builder
	b.WriteString("\n")
	for i, c := range choices {
		b.WriteString(fmt.Sprintf("  <ansi fg=\"yellow-bold\">%d)</ansi> <ansi fg=\"white-bold\">%s</ansi> %s\n", i+1, c.Name, c.Description))
		if len(c.Skills) > 0 {
			b.WriteString("       Skills: " + strings.Join(c.Skills, ", ") + "\n")
		}
		if len(c.Kit) > 0 {
			b.WriteString("       Starter kit: " + strings.Join(c.Kit, ", ") + "\n")
		}
	}
	b.WriteString("  Answer with a number or a name.\n")
	return b.String()
}

// matchArchetypeChoice accepts a 1-based number, an id, or a name.
func matchArchetypeChoice(choices []archetypes.Choice, response string) (archetypes.Choice, bool) {
	response = strings.TrimSpace(response)
	if n, err := strconv.Atoi(response); err == nil {
		if n >= 1 && n <= len(choices) {
			return choices[n-1], true
		}
		return archetypes.Choice{}, false
	}
	for _, c := range choices {
		if strings.EqualFold(c.ID, response) || strings.EqualFold(c.Name, response) {
			return c, true
		}
	}
	return archetypes.Choice{}, false
}
