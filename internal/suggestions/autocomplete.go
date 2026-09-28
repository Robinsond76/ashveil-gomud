package suggestions

import (
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// AutoCompleteRequest is the value threaded through OnAutoComplete handlers.
// Cmd is the resolved command (after alias expansion). Parts is the original
// whitespace-split input slice. UserId identifies the requesting player.
// Results is the accumulated completion suffix list; handlers append to it.
type AutoCompleteRequest struct {
	UserId  int
	Cmd     string
	Parts   []string
	Results []string
}

// OnAutoComplete is fired after the built-in autocomplete logic has run.
// Modules register handlers here to contribute completions for their own
// commands. Each handler receives the current request (including any results
// already accumulated by earlier handlers), may append to Results, and must
// return the modified value.
//
// Example registration from a module:
//
//	suggestions.OnAutoComplete.Register(func(r suggestions.AutoCompleteRequest) suggestions.AutoCompleteRequest {
//	    if r.Cmd != "mycommand" {
//	        return r
//	    }
//	    // ... append to r.Results ...
//	    return r
//	})
var OnAutoComplete util.Hook[AutoCompleteRequest]

// GetAutoComplete returns a list of completion suffixes for the given partial
// input text. Each entry is the text that should be appended to inputText to
// form a complete suggestion. The results are sorted shortest-first.
func GetAutoComplete(userId int, inputText string) []string {

	result := []string{}

	user := users.GetByUserId(userId)
	if user == nil {
		return result
	}

	// Ashveil 32h: never complete (or show) a password being typed, from
	// the terminal or the web client.
	if connections.InputMasked(connections.ConnectionId(user.ConnectionId())) {
		return result
	}

	// If engaged in a prompt, try to match an option
	if promptInfo := user.GetPrompt(); promptInfo != nil {
		if qInfo := promptInfo.GetNextQuestion(); qInfo != nil {

			if len(qInfo.Options) > 0 {

				for _, opt := range qInfo.Options {

					if inputText == `` {
						result = append(result, opt)
						continue
					}

					s1 := strings.ToLower(opt)
					s2 := strings.ToLower(inputText)
					if s1 != s2 && strings.HasPrefix(s1, s2) {
						result = append(result, s1[len(s2):])
					}
				}

				return result
			}
		}
	}

	if inputText == `` {
		return result
	}

	isAdmin := user.Role == users.RoleAdmin
	parts := strings.Split(inputText, ` `)

	// If only one part, probably a command
	if len(parts) < 2 {

		result = append(result, usercommands.GetCmdSuggestions(parts[0], isAdmin)...)

		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			for exitName, exitInfo := range room.Exits {
				if exitInfo.Secret {
					continue
				}
				if strings.HasPrefix(strings.ToLower(exitName), strings.ToLower(parts[0])) {
					result = append(result, exitName[len(parts[0]):])
				}
			}
		}
	} else {

		cmd := keywords.TryCommandAlias(parts[0])
		targetName := strings.ToLower(strings.Join(parts[1:], ` `))
		targetNameLen := len(targetName)

		itemList := []items.Item{}
		itemTypeSearch := []items.ItemType{}
		itemSubtypeSearch := []items.ItemSubType{}

		req := OnAutoComplete.Fire(AutoCompleteRequest{
			UserId:  userId,
			Cmd:     cmd,
			Parts:   parts,
			Results: result,
		})

		if len(req.Results) > 0 {
			result = req.Results
		} else if cmd == `help` {

			result = append(result, usercommands.GetHelpSuggestions(targetName, isAdmin)...)

		} else if cmd == `look` {

			itemList = user.Character.GetAllBackpackItems()

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for exitName, exitInfo := range room.Exits {
					if exitInfo.Secret {
						continue
					}
					if strings.HasPrefix(strings.ToLower(exitName), targetName) {
						result = append(result, exitName[targetNameLen:])
					}
				}

				for containerName := range room.Containers {
					if strings.HasPrefix(strings.ToLower(containerName), targetName) {
						result = append(result, containerName[targetNameLen:])
					}
				}

				for _, corpse := range room.Corpses {
					if corpse.Prunable {
						continue
					}
					corpseName := corpse.Character.Name + ` corpse`
					if strings.HasPrefix(strings.ToLower(corpseName), targetName) {
						result = append(result, corpseName[targetNameLen:])
					}
				}
			}

		} else if cmd == `drop` || cmd == `trash` || cmd == `sell` || cmd == `store` || cmd == `inspect` || cmd == `enchant` || cmd == `appraise` || cmd == `give` || cmd == `stash` || cmd == `offer` {

			itemList = user.Character.GetAllBackpackItems()

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for containerName := range room.Containers {
					if strings.HasPrefix(strings.ToLower(containerName), targetName) {
						result = append(result, containerName[targetNameLen:])
					}
				}
			}

		} else if cmd == `equip` {

			itemList = user.Character.GetAllBackpackItems()
			itemSubtypeSearch = append(itemSubtypeSearch, items.Wearable)
			itemTypeSearch = append(itemTypeSearch, items.Weapon)

		} else if cmd == `remove` {

			itemList = user.Character.GetAllWornItems()

		} else if cmd == `get` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				itemList = room.GetAllFloorItems(false)
			}

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				if room.Gold > 0 {
					goldName := `gold`
					if strings.HasPrefix(goldName, targetName) {
						result = append(result, goldName[targetNameLen:])
					}
				}
				for containerName, containerInfo := range room.Containers {
					if containerInfo.Lock.IsLocked() {
						continue
					}

					for _, item := range containerInfo.Items {
						iSpec := item.GetSpec()
						if strings.HasPrefix(strings.ToLower(iSpec.Name), targetName) {
							result = append(result, iSpec.Name[targetNameLen:]+` from `+containerName)
						}
					}

					if containerInfo.Gold > 0 {
						goldName := `gold from ` + containerName
						if strings.HasPrefix(goldName, targetName) {
							result = append(result, goldName[targetNameLen:])
						}
					}
				}

				if configs.GetGamePlayConfig().Death.CorpseItems {
					for _, corpse := range room.Corpses {
						if corpse.Prunable {
							continue
						}
						corpseName := strings.ToLower(corpse.Character.Name + ` corpse`)

						for _, item := range corpse.Items {
							iSpec := item.GetSpec()
							fullSuggestion := strings.ToLower(iSpec.Name) + ` from ` + corpseName
							if strings.HasPrefix(fullSuggestion, targetName) {
								result = append(result, fullSuggestion[targetNameLen:])
							}
						}

						for _, item := range corpse.Character.GetAllWornItems() {
							iSpec := item.GetSpec()
							fullSuggestion := strings.ToLower(iSpec.Name) + ` from ` + corpseName
							if strings.HasPrefix(fullSuggestion, targetName) {
								result = append(result, fullSuggestion[targetNameLen:])
							}
						}

						if corpse.Gold > 0 {
							goldSuggestion := `gold from ` + corpseName
							if strings.HasPrefix(goldSuggestion, targetName) {
								result = append(result, goldSuggestion[targetNameLen:])
							}
						}

						allSuggestion := `all ` + corpseName
						if strings.HasPrefix(allSuggestion, targetName) {
							result = append(result, allSuggestion[targetNameLen:])
						}
					}
				}
			}

		} else if cmd == `eat` {

			itemList = user.Character.GetAllBackpackItems()
			itemSubtypeSearch = append(itemSubtypeSearch, items.Edible)

		} else if cmd == `drink` {

			itemList = user.Character.GetAllBackpackItems()
			itemSubtypeSearch = append(itemSubtypeSearch, items.Drinkable)

		} else if cmd == `use` {

			itemList = user.Character.GetAllBackpackItems()
			itemSubtypeSearch = append(itemSubtypeSearch, items.Usable)

		} else if cmd == `throw` {

			itemList = user.Character.GetAllBackpackItems()
			itemSubtypeSearch = append(itemSubtypeSearch, items.Throwable)

		} else if cmd == `picklock` || cmd == `unlock` || cmd == `lock` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for exitName, exitInfo := range room.Exits {
					if exitInfo.Secret || !exitInfo.HasLock() {
						continue
					}
					if strings.HasPrefix(strings.ToLower(exitName), targetName) {
						result = append(result, exitName[targetNameLen:])
					}
				}

				for containerName, containerInfo := range room.Containers {
					if containerInfo.HasLock() {
						if strings.HasPrefix(strings.ToLower(containerName), targetName) {
							result = append(result, containerName[targetNameLen:])
						}
					}
				}
			}

		} else if cmd == `attack` || cmd == `consider` || cmd == `backstab` || cmd == `pickpocket` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {

				mobNameTracker := map[string]int{}

				for _, mobInstId := range room.GetMobs() {
					if mob := mobs.GetInstance(mobInstId); mob != nil {

						if mob.Character.IsCharmed() && (mob.Character.Aggro == nil || mob.Character.Aggro.UserId != userId) {
							continue
						}

						if targetName == `` {
							result = append(result, mob.Character.Name)
							continue
						}

						if strings.HasPrefix(strings.ToLower(mob.Character.Name), targetName) {
							name := mob.Character.Name[targetNameLen:]

							mobNameTracker[name] = mobNameTracker[name] + 1

							if mobNameTracker[name] > 1 {
								name += `#` + strconv.Itoa(mobNameTracker[name])
							}
							result = append(result, name)
						}
					}
				}
			}

		} else if cmd == `buy` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for _, mobInstId := range room.GetMobs(rooms.FindMerchant) {

					mob := mobs.GetInstance(mobInstId)
					if mob == nil {
						continue
					}

					for _, stockInfo := range mob.Character.Shop.GetInstock() {
						item := items.New(stockInfo.ItemId)
						if item.ItemId > 0 {
							itemList = append(itemList, item)
						}
					}
				}
			}

		} else if cmd == `set` {

			options := []string{
				`description`,
				`prompt`,
				`fprompt`,
				`tinymap`,
			}

			for _, opt := range options {
				if strings.HasPrefix(opt, targetName) {
					result = append(result, opt[len(targetName):])
				}
			}

		} else if cmd == `spawn` {

			if len(inputText) >= len(`spawn item `) && inputText[0:len(`spawn item `)] == `spawn item ` {
				targetName := inputText[len(`spawn item `):]
				for _, itemName := range items.GetAllItemNames() {
					for _, testName := range util.BreakIntoParts(itemName) {
						if strings.HasPrefix(testName, targetName) {
							result = append(result, testName[len(targetName):])
						}
					}
				}
			} else if len(inputText) >= len(`spawn mob `) && inputText[0:len(`spawn mob `)] == `spawn mob ` {
				targetName := inputText[len(`spawn mob `):]
				for _, mobName := range mobs.GetAllMobNames() {
					for _, testName := range util.BreakIntoParts(mobName) {
						if strings.HasPrefix(testName, targetName) {
							result = append(result, testName[len(targetName):])
						}
					}
				}
			} else if len(inputText) >= len(`spawn gold `) && inputText[0:len(`spawn gold `)] == `spawn gold ` {
				result = append(result, "50", "100", "500", "1000", "5000")
			} else {
				options := []string{`mob`, `gold`, `item`}
				for _, opt := range options {
					if strings.HasPrefix(opt, targetName) {
						result = append(result, opt[len(targetName):])
					}
				}
			}

		} else if cmd == `locate` {

			ids := users.GetOnlineUserIds()
			for _, id := range ids {
				if id == user.UserId {
					continue
				}
				if u := users.GetByUserId(id); u != nil {
					if strings.HasPrefix(strings.ToLower(u.Character.Name), targetName) {
						result = append(result, u.Character.Name[targetNameLen:])
					}
				}
			}

		} else if cmd == `cast` {

			for spellName, casts := range user.Character.GetSpells() {
				if casts < 0 {
					continue
				}
				if strings.HasPrefix(spellName, targetName) {
					result = append(result, spellName[len(targetName):])
				}
			}

		} else if cmd == `whisper` {

			ids := users.GetOnlineUserIds()
			for _, id := range ids {
				if id == user.UserId {
					continue
				}
				if u := users.GetByUserId(id); u != nil {
					if strings.HasPrefix(strings.ToLower(u.Character.Name), targetName) {
						result = append(result, u.Character.Name[targetNameLen:])
					}
				}
			}

		} else if cmd == `put` {

			itemList = user.Character.GetAllBackpackItems()

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for containerName := range room.Containers {
					if strings.HasPrefix(strings.ToLower(containerName), targetName) {
						result = append(result, containerName[targetNameLen:])
					}
				}
			}

		} else if cmd == `bank` {

			for _, opt := range []string{`deposit`, `withdraw`} {
				if strings.HasPrefix(opt, targetName) {
					result = append(result, opt[targetNameLen:])
				}
			}

		} else if cmd == `train` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for skillName := range room.SkillTraining {
					if strings.HasPrefix(strings.ToLower(skillName), targetName) {
						result = append(result, skillName[targetNameLen:])
					}
				}
			}

		} else if cmd == `aid` {

			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				for _, uid := range room.GetPlayers(rooms.FindDowned) {
					if uid == user.UserId {
						continue
					}
					if u := users.GetByUserId(uid); u != nil {
						if strings.HasPrefix(strings.ToLower(u.Character.Name), targetName) {
							result = append(result, u.Character.Name[targetNameLen:])
						}
					}
				}
			}

		} else if cmd == `formset` {

			if len(parts) == 2 {
				if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
					for _, uid := range room.GetPlayers() {
						if u := users.GetByUserId(uid); u != nil {
							if strings.HasPrefix(strings.ToLower(u.Character.Name), targetName) {
								result = append(result, u.Character.Name[targetNameLen:])
							}
						}
					}
					for _, mobInstId := range room.GetMobs() {
						if mob := mobs.GetInstance(mobInstId); mob != nil {
							if strings.HasPrefix(strings.ToLower(mob.Character.Name), targetName) {
								result = append(result, mob.Character.Name[targetNameLen:])
							}
						}
					}
				}
			} else if len(parts) >= 3 {
				racePart := strings.ToLower(strings.Join(parts[2:], ` `))
				racePartLen := len(racePart)
				for _, r := range races.GetRaces() {
					rName := strings.ToLower(r.Name)
					if strings.HasPrefix(rName, racePart) {
						result = append(result, r.Name[racePartLen:])
					}
				}
				for _, opt := range []string{`revert`, `short`, `medium`, `long`} {
					if strings.HasPrefix(opt, racePart) {
						result = append(result, opt[racePartLen:])
					}
				}
			}

		}

		if len(itemList) > 0 {

			bpItemTracker := map[string]int{}

			typeSearchCt := len(itemTypeSearch)
			subtypeSearchCt := len(itemSubtypeSearch)

			for _, item := range itemList {
				iSpec := item.GetSpec()

				skip := false
				if typeSearchCt > 0 || subtypeSearchCt > 0 {
					skip = true

					for i := 0; i < typeSearchCt; i++ {
						if iSpec.Type == itemTypeSearch[i] {
							skip = false
						}
					}

					for i := 0; i < subtypeSearchCt; i++ {
						if iSpec.Subtype == itemSubtypeSearch[i] {
							skip = false
						}
					}

					if skip {
						continue
					}
				}

				if targetName == `` {

					name := iSpec.Name

					bpItemTracker[name] = bpItemTracker[name] + 1

					if bpItemTracker[name] > 1 {
						name += `#` + strconv.Itoa(bpItemTracker[name])
					}
					result = append(result, name)

					continue
				}

				for _, testName := range util.BreakIntoParts(iSpec.Name) {
					if strings.HasPrefix(strings.ToLower(testName), targetName) {
						name := testName[targetNameLen:]

						bpItemTracker[name] = bpItemTracker[name] + 1

						if bpItemTracker[name] > 1 {
							name += `#` + strconv.Itoa(bpItemTracker[name])
						}
						result = append(result, name)
					}
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return len(result[i]) < len(result[j])
	})

	return result
}
