package users

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// This file contains vars/receiver methods for the UserRecord struct dealing with the prmopt.
// This just makes it easier to find and make adjustments to. It got annoying searching userrecord.go
// NOTE: NOT to be confused with an interactive question/answer prompt.
//
// Prompt Helpfile: templates/help/set-prompt.template
//

var (
	promptDefaultCompiled = ``
)

// PromptToken is one segment of a parsed prompt string.
// Tag holds the original brace-delimited token (e.g. "{hp}"), or an empty
// string for literal text segments that appear between tokens.
// Value holds the resolved output for this segment. For known tokens Value is
// the rendered string. For unknown tokens Value defaults to Tag so that
// unrecognised tokens round-trip unchanged. For literal segments Value is the
// literal text.
type PromptToken struct {
	Tag   string
	Value string
}

// PromptData is the value threaded through OnBuildPrompt handlers.
// User is the player whose prompt is being rendered.
// Tokens is the fully-resolved token slice. Handlers may modify Value on any
// token (or append/remove tokens) before the final string is assembled.
type PromptData struct {
	User   *UserRecord
	Tokens []PromptToken
}

// OnBuildPrompt is fired inside ProcessPromptString after all built-in tokens
// have been resolved and before the final prompt string is assembled.
// Modules register handlers here to add, replace, or remove prompt tokens.
//
// Example registration from a module:
//
//	users.OnBuildPrompt.Register(func(d users.PromptData) users.PromptData {
//	    for i, t := range d.Tokens {
//	        if t.Tag == "{mytoken}" {
//	            d.Tokens[i].Value = computeMyValue(d.User)
//	        }
//	    }
//	    return d
//	})
var OnBuildPrompt util.Hook[PromptData]

func (u *UserRecord) GetCommandPrompt() string {

	promptOut := ``

	if u.activePrompt != nil {

		if activeQuestion := u.activePrompt.GetNextQuestion(); activeQuestion != nil {
			promptOut = activeQuestion.String()
		}
	}

	goAhead := ``
	if connections.GetClientSettings(u.ConnectionId()).SendTelnetGoAhead {
		goAhead = term.TelnetGoAhead.String()
	}

	if len(promptOut) == 0 {

		if promptDefaultCompiled == `` {
			promptDefaultCompiled = util.ConvertColorShortTags(configs.GetTextFormatsConfig().Prompt.String())
		}

		var customPrompt any = nil
		var inCombat bool = u.Character.Aggro != nil

		if inCombat {
			customPrompt = u.GetConfigOption(`fprompt-compiled`)
		}

		// No other custom prompts? try the default setting
		if customPrompt == nil {
			customPrompt = u.GetConfigOption(`prompt-compiled`)
		}

		if customPrompt != nil {
			if ansiPrompt, ok := customPrompt.(string); ok {
				promptOut = u.ProcessPromptString(ansiPrompt)
			}
		}

		// Still nothing? Default to ... default
		if len(promptOut) == 0 {
			promptOut = u.ProcessPromptString(promptDefaultCompiled)
		}

	}

	unsent, suggested := u.GetUnsentText()
	// Ashveil 32h: a password being typed is redrawn as stars, with no
	// suggestion (the connection's mask is lock-guarded, so this is safe on
	// a connection goroutine).
	if connections.InputMasked(u.connectionId) {
		unsent, suggested = strings.Repeat(`*`, utf8.RuneCountInString(unsent)), ``
	}
	if len(suggested) > 0 {
		suggested = `<ansi fg="suggested-text">` + suggested + `</ansi>`
	}
	return term.AnsiMoveCursorColumn.String() + term.AnsiEraseLine.String() + promptOut + unsent + suggested + goAhead

}

func (u *UserRecord) ProcessPromptString(promptStr string) string {

	var currentXP, tnlXP int = -1, -1
	var hpPct, mpPct int = -1, -1
	var hpClass, mpClass string

	promptLen := len(promptStr)
	tagStartPos := -1

	tokens := []PromptToken{}
	litBuf := strings.Builder{}

	flushLiteral := func() {
		if litBuf.Len() > 0 {
			tokens = append(tokens, PromptToken{Tag: ``, Value: litBuf.String()})
			litBuf.Reset()
		}
	}

	for i := 0; i < promptLen; i++ {
		if promptStr[i] == '{' {
			flushLiteral()
			tagStartPos = i
			continue
		}
		if promptStr[i] == '}' {
			tag := promptStr[tagStartPos : i+1]
			var value string

			switch tag {

			case `{\n}`:
				value = "\n"

			case `{hp}`:
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, hpClass, u.Character.Health)

			case `{hp:-}`:
				value = strconv.Itoa(u.Character.Health)

			case `{HP}`:
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, hpClass, u.Character.HealthMax.Value)

			case `{HP:-}`:
				value = strconv.Itoa(u.Character.HealthMax.Value)

			case `{hp%}`:
				if hpPct == -1 {
					hpPct = int(math.Floor(float64(u.Character.Health) / float64(u.Character.HealthMax.Value) * 100))
				}
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, hpClass, hpPct)

			case `{hp%:-}`:
				if hpPct == -1 {
					hpPct = int(math.Floor(float64(u.Character.Health) / float64(u.Character.HealthMax.Value) * 100))
				}
				value = strconv.Itoa(hpPct) + `%`

			case `{mp}`:
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Mana, u.Character.ManaMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, mpClass, u.Character.Mana)

			case `{mp:-}`:
				value = strconv.Itoa(u.Character.Mana)

			case `{MP}`:
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Mana, u.Character.ManaMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, mpClass, u.Character.ManaMax.Value)

			case `{MP:-}`:
				value = strconv.Itoa(u.Character.ManaMax.Value)

			case `{mp%}`:
				if mpPct == -1 {
					mpPct = int(math.Floor(float64(u.Character.Mana) / float64(u.Character.ManaMax.Value) * 100))
				}
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Mana, u.Character.ManaMax.Value))
				}
				value = fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, mpClass, mpPct)

			case `{mp%:-}`:
				if mpPct == -1 {
					mpPct = int(math.Floor(float64(u.Character.Mana) / float64(u.Character.ManaMax.Value) * 100))
				}
				value = strconv.Itoa(mpPct) + `%`

			case `{ap}`:
				value = strconv.Itoa(u.Character.ActionPoints)

			case `{xp}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				value = strconv.Itoa(currentXP)

			case `{XP}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				value = strconv.Itoa(tnlXP)

			case `{xpn}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				value = strconv.Itoa(tnlXP - currentXP)

			case `{xp%}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				value = strconv.Itoa(int(math.Floor(float64(currentXP)/float64(tnlXP)*100))) + `%`

			case `{h}`:
				if u.Character.HasBuffFlag("hidden") {
					value = `H`
				}

			case `{a}`:
				alignClass := u.Character.AlignmentName()
				value = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignClass, alignClass[:1])

			case `{A}`:
				alignClass := u.Character.AlignmentName()
				value = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignClass, alignClass)

			case `{g}`:
				value = strconv.Itoa(u.Character.Gold)

			case `{tp}`:
				value = strconv.Itoa(u.Character.TrainingPoints)

			case `{sp}`:
				value = strconv.Itoa(u.Character.StatPoints)

			case `{i}`:
				value = strconv.Itoa(len(u.Character.Items))

			case `{I}`:
				value = strconv.Itoa(u.Character.CarryCapacity())

			case `{lvl}`:
				value = strconv.Itoa(u.Character.Level)

			case `{w}`:
				if u.Character.Aggro != nil {
					value = strconv.Itoa(u.Character.Aggro.RoundsWaiting)
				} else {
					value = `0`
				}

			case `{t}`:
				value = gametime.GetDate().String(true)

			case `{T}`:
				value = gametime.GetDate().String()

			default:
				value = tag
			}

			tokens = append(tokens, PromptToken{Tag: tag, Value: value})
			tagStartPos = -1
			continue
		}

		if tagStartPos == -1 {
			litBuf.WriteByte(promptStr[i])
		}
	}

	// Flush any trailing literal text (also covers an unclosed '{' at end of
	// string — tagStartPos would be set but we have nothing to emit for the
	// incomplete tag, so we just flush whatever accumulated before it).
	flushLiteral()

	data := OnBuildPrompt.Fire(PromptData{User: u, Tokens: tokens})

	out := strings.Builder{}
	for _, t := range data.Tokens {
		out.WriteString(t.Value)
	}
	return out.String()
}
