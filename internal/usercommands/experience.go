package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Experience(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) > 0 && args[0] == `chart` {

		args = args[1:]

		startLevel := 1
		endLevel := 25

		chartRace := user.Character.GetRaceId()

		// xp chart elf 50
		if len(args) > 1 {

			if lvl, err := strconv.Atoi(args[len(args)-1]); err == nil {
				endLevel = lvl
				args = args[:len(args)-1]
			} else if lvl, err := strconv.Atoi(args[0]); err == nil {
				endLevel = lvl
				args = args[1:]
			}

			raceName := strings.Join(args, ` `)
			if raceInfo, found := races.FindRace(raceName); found {
				chartRace = raceInfo.RaceId
			}
		} else if len(args) == 1 {

			if lvl, err := strconv.Atoi(args[0]); err == nil {
				endLevel = lvl
			} else {
				if raceInfo, found := races.FindRace(args[0]); found {
					chartRace = raceInfo.RaceId
				}
			}

		}

		if endLevel < 2 {
			endLevel = 2
		}
		if endLevel > 200 {
			endLevel = 200
		}
		if endLevel-startLevel > 25 {
			startLevel = endLevel - 25
		}
		if startLevel < 1 {
			startLevel = 1
		}

		mockChar := characters.New()
		mockChar.RaceId = chartRace
		mockChar.Validate()

		headers := []string{`Level`, `Experience`, `Str`, `Spd`, `Smt`, `Vit`, `Mys`, `Per`, `ALL`}
		rows := [][]string{}

		formatting := []string{
			`<ansi fg="white-bold">%s</ansi>`,
			`<ansi fg="red">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
		}

		zeroStr := ``

		stats := []string{`str`, `spd`, `smt`, `vit`, `mys`, `per`}

		// adjForLevel returns the effective (ValueAdj) stat value at a given level
		// for the mock character, using Recalculate so the cap is applied.
		adjForLevel := func(lvl int) map[string]int {
			if lvl < 1 {
				lvl = 1
			}
			mockChar.Stats.Strength.Recalculate(lvl)
			mockChar.Stats.Speed.Recalculate(lvl)
			mockChar.Stats.Smarts.Recalculate(lvl)
			mockChar.Stats.Vitality.Recalculate(lvl)
			mockChar.Stats.Mysticism.Recalculate(lvl)
			mockChar.Stats.Perception.Recalculate(lvl)
			return map[string]int{
				`str`: mockChar.Stats.Strength.ValueAdj,
				`spd`: mockChar.Stats.Speed.ValueAdj,
				`smt`: mockChar.Stats.Smarts.ValueAdj,
				`vit`: mockChar.Stats.Vitality.ValueAdj,
				`mys`: mockChar.Stats.Mysticism.ValueAdj,
				`per`: mockChar.Stats.Perception.ValueAdj,
			}
		}

		oldG := adjForLevel(startLevel - 1)

		newG := map[string]int{}
		totalG := map[string]int{}
		for _, stat := range stats {
			totalG[stat] = oldG[stat]
		}

		for i := startLevel; i <= endLevel; i++ {

			newG = adjForLevel(i)

			tnlXP := mockChar.XPTL(i) - mockChar.XPTL(i-1)

			if i == 1 {
				tnlXP = 0
			}

			if i > 1 {

				row := []string{fmt.Sprintf(`%d`, i), fmt.Sprintf(`%d`, tnlXP)}
				gainStr := zeroStr
				all := 0
				for _, stat := range stats {
					gain := newG[stat] - oldG[stat]
					all += gain
					totalG[stat] += gain

					if gain > 0 {
						gainStr = fmt.Sprintf(`%d`, gain)
					} else {
						gainStr = zeroStr
					}

					row = append(row, gainStr)
				}

				if all > 0 {
					row = append(row, fmt.Sprintf(`%d`, all))
				} else {
					row = append(row, zeroStr)
				}

				rows = append(rows, row)
			}

			for _, stat := range stats {
				oldG[stat] = newG[stat]
			}
		}

		row := []string{`Total`, fmt.Sprintf(`%d`, mockChar.XPTL(endLevel)-1500)}
		all := 0
		for _, stat := range stats {
			gain := totalG[stat]
			all += gain
			gainStr := ``
			if gain > 0 {
				gainStr = fmt.Sprintf(`%d`, gain)
			}

			row = append(row, gainStr)
		}

		if all > 0 {
			row = append(row, fmt.Sprintf(`%d`, all))
		} else {
			row = append(row, zeroStr)
		}

		rows = append(rows, row)

		raceInfo := races.GetRace(mockChar.RaceId)
		searchResultsTable := templates.GetTable(fmt.Sprintf(`Experience Chart for %s`, raceInfo.Name), headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText(tplTxt)

		return true, nil
	}

	realXPNow, realXPTNL := user.Character.XPTNLActual()
	xpInfo := map[string]int{
		"Level": user.Character.Level,
		"Exp":   realXPNow,
		"Tnl":   realXPTNL,
		"Tp":    user.Character.TrainingPoints,
		"Sp":    user.Character.StatPoints,
	}

	tplTxt, _ := templates.Process("character/experience", xpInfo, user.UserId)
	// Ashveil (Phase 26a): the last level lost to death.
	tplTxt += lastLossLine(user)
	user.SendText(tplTxt)

	return true, nil
}
