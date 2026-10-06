package usercommands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/telemetry"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Help(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var helpTxt string
	var err error = nil

	args := util.SplitButRespectQuotes(rest)

	if isHelpIndexRequest(args) {

		type helpCommand struct {
			Command string
			Type    string
			Missing bool
		}

		type commandLists struct {
			Commands map[string][]helpCommand
			Skills   map[string][]helpCommand
			Admin    map[string][]helpCommand
		}

		helpCommandList := commandLists{
			Commands: make(map[string][]helpCommand),
			Skills:   make(map[string][]helpCommand),
			Admin:    make(map[string][]helpCommand),
		}

		for _, command := range keywords.GetAllHelpTopicInfo() {

			category := command.Category
			if category == `all` {
				category = ``
			}

			templateFile := `help/` + keywords.TryHelpAlias(command.Command)

			if command.AdminOnly {
				if user.HasRolePermission(command.Command, true) {
					helpCommandList.Admin[category] = append(
						helpCommandList.Admin[category],
						helpCommand{Command: command.Command, Type: "command-admin", Missing: false},
					)
				}
				continue
			}

			hlpCmd := helpCommand{Command: command.Command, Type: command.Type, Missing: !templates.Exists(templateFile)}

			if command.Type == `skill` {
				helpCommandList.Skills[category] = append(helpCommandList.Skills[category], hlpCmd)
				continue
			}

			helpCommandList.Commands[category] = append(helpCommandList.Commands[category], hlpCmd)

		}

		helpTxt, err = templates.Process("help/help", helpCommandList, user.UserId)
		if err != nil {
			helpTxt = err.Error()
		}
	} else {

		helpTxt, err = GetHelpContents(rest)
		if err != nil {
			// Ashveil: an admin command's own page answers `help [command]`
			// for those who may use it; anyone else sees no such topic.
			if adminTxt, ok := adminHelpContents(rest, user); ok {
				user.SendText(adminTxt)
				return true, nil
			}
			user.SendText(fmt.Sprintf(`No help found for "%s"`, rest))
			return true, err
		}

		telemetry.TrackFull(telemetry.CatHelpTopic, "", 0, 0, 0, 0, resolveHelpTopic(rest))

	}

	user.SendText(helpTxt)

	return true, nil
}

func getRaceOptions(raceRequest string) []races.Race {

	allRaces := races.GetRaces()
	sort.Slice(allRaces, func(i, j int) bool {
		return allRaces[i].RaceId < allRaces[j].RaceId
	})

	raceNames := strings.Split(raceRequest, ` `)

	getAllRaces := false
	if raceRequest == `all` {
		getAllRaces = true
	}

	raceOptions := []races.Race{}
	for _, race := range allRaces {

		if len(raceRequest) == 0 {
			if !race.Selectable && !getAllRaces {
				continue
			}
		} else if len(raceNames) > 0 {
			lowerName := strings.ToLower(race.Name)
			found := false
			for _, rName := range raceNames {
				if strings.Contains(lowerName, strings.ToLower(rName)) {
					found = true
					break
				}
			}
			if !getAllRaces && !found {
				continue
			}
		}
		raceOptions = append(raceOptions, race)
	}

	return raceOptions
}

func GetHelpContents(input string) (string, error) {

	args := util.SplitButRespectQuotes(input)

	helpName := args[0]
	helpRest := ``

	args = args[1:]
	if len(args) > 0 {
		helpRest = strings.Join(args, ` `)
	}

	// replace any non alpha/numeric characters in "rest"
	if fullSearchAlias := keywords.TryHelpAlias(input); fullSearchAlias != input {
		helpName = fullSearchAlias
	} else {
		helpName = regexp.MustCompile(`[^a-zA-Z0-9\\-]+`).ReplaceAllString(helpName, ``)
		helpName = keywords.TryHelpAlias(helpName)
	}

	var helpVars any = nil

	if helpName == `emote` {
		helpVars = emoteAliases
	}

	if helpName == `races` {
		helpVars = getRaceOptions(helpRest)
	}

	if helpName == `spell` {
		sData := spells.GetSpell(helpRest)
		if sData == nil {
			sData = spells.FindSpellByName(helpRest)
		}

		if sData == nil {
			helpName = `spells`
		} else {
			helpVars = *sData
		}
	}

	returnText, err := templates.Process("help/"+helpName, helpVars, 0)

	result := OnGetHelpContents.Fire(HelpContentsResult{Text: returnText, Err: err})
	return result.Text, result.Err
}

// HelpContentsResult is the value threaded through OnGetHelpContents handlers.
// Text is the fully rendered help content. Err is the error from template
// processing, or nil on success. Handlers may replace Text, clear Err to
// suppress a template error and supply fallback content, or set Err to
// signal a failure.
type HelpContentsResult struct {
	Text string
	Err  error
}

// OnGetHelpContents is fired at the end of GetHelpContents with the fully
// rendered help text and any template error. Modules register handlers here
// to modify or augment help content before it is returned to the caller.
//
// Example registration from a module:
//
//	usercommands.OnGetHelpContents.Register(func(r usercommands.HelpContentsResult) usercommands.HelpContentsResult {
//	    r.Text += "\nSee also: mycustomtopic"
//	    return r
//	})
var OnGetHelpContents util.Hook[HelpContentsResult]

// resolveHelpTopic returns the canonical help topic name for a raw input string,
// applying the same alias resolution used by GetHelpContents.
func resolveHelpTopic(input string) string {
	if fullSearchAlias := keywords.TryHelpAlias(input); fullSearchAlias != input {
		return fullSearchAlias
	}
	helpName := util.SplitButRespectQuotes(input)[0]
	helpName = regexp.MustCompile(`[^a-zA-Z0-9\\-]+`).ReplaceAllString(helpName, ``)
	return keywords.TryHelpAlias(helpName)
}

// isHelpIndexRequest reports whether `help` should list its index: bare, or
// asked about itself. `help help` once answered "No help found" (Phase 44).
func isHelpIndexRequest(args []string) bool {
	return len(args) == 0 || (len(args) == 1 && strings.EqualFold(args[0], "help"))
}

// adminHelpContents is the page of an admin command (admincommands/help/
// command.[name]) for a user allowed to run it, when the help topic is
// listed under admin and has no player page.
func adminHelpContents(input string, user *users.UserRecord) (string, bool) {
	fields := strings.Fields(strings.ToLower(input))
	if len(fields) == 0 {
		return "", false
	}
	name := keywords.TryHelpAlias(fields[0])
	admin := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == name {
			admin = topic.AdminOnly
			break
		}
	}
	if !admin || !user.HasRolePermission(name, true) {
		return "", false
	}
	text, err := templates.Process("admincommands/help/command."+name, nil, user.UserId)
	if err != nil {
		return "", false
	}
	return text, true
}
