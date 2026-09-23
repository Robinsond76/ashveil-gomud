package usercommands

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type TrainingOption struct {
	Name          string
	CurrentStatus string
	Cost          int
	Message       string
}
type TrainingOptions struct {
	TrainingPoints int
	Options        []TrainingOption
}

func Train(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(room.SkillTraining) == 0 {
		user.SendText(`You must find a trainer to perform training.`)
		return false, nil
	}

	trainingData := TrainingOptions{
		TrainingPoints: user.Character.TrainingPoints,
		Options:        []TrainingOption{}, // name to option
	}

	trainables := []string{}
	maxed := []string{}
	untrainables := []string{}

	trainOpts := map[string]TrainingOption{}

	for skillName, trainingRange := range room.SkillTraining {
		// Skip skills that no longer exist (e.g. deleted from the datafiles).
		if !skills.SkillExists(skillName) {
			continue
		}
		maxLevel := skills.MaxSkillLevel(skillName)
		currentLevel := user.Character.GetSkillLevel(skillName)
		opt := TrainingOption{
			Name: skillName,
		}

		if currentLevel >= maxLevel {
			opt.CurrentStatus = "Maximum"
			opt.Message = ""
			opt.Cost = 0
		} else if currentLevel >= trainingRange.Max {
			opt.CurrentStatus = fmt.Sprintf("Level %d", currentLevel)
			opt.Message = "Seek additional training elsewhere."
			opt.Cost = 0
		} else if currentLevel < trainingRange.Min-1 {
			opt.CurrentStatus = fmt.Sprintf("Level %d", currentLevel)
			opt.Message = fmt.Sprintf("You aren't ready yet. See me for level %d.", trainingRange.Min)
			opt.Cost = 0
		} else if allowed, _ := archetypes.CanTrain(user.UserId, skillName); !allowed {
			// Archetype-claimed skill this character may not train (Phase 17).
			opt.CurrentStatus = fmt.Sprintf("Level %d", currentLevel)
			opt.Message = archetypeOnlyLabel(skillName)
			opt.Cost = 0
		} else if currentLevel >= trainingRange.Min-1 && currentLevel < trainingRange.Max {
			opt.Cost = user.Character.GetSkillLevelCost(currentLevel + 1)
			if currentLevel == 0 {
				opt.CurrentStatus = `<ansi fg="cyan-bold">Unknown</ansi>`
				opt.Message = fmt.Sprintf(`<ansi fg="cyan-bold">Learn this skill here for </ansi><ansi fg="yellow">%d training points</ansi><ansi fg="cyan-bold">.</ansi>`, opt.Cost)
			} else {
				opt.CurrentStatus = fmt.Sprintf(`<ansi fg="cyan-bold">Level %d</ansi>`, currentLevel)
				opt.Message = fmt.Sprintf(`<ansi fg="cyan-bold">Upgrade this skill here for </ansi><ansi fg="yellow">%d training points</ansi><ansi fg="cyan-bold">.</ansi>`, opt.Cost)
			}

			opt.Cost = currentLevel + 1
		}

		trainOpts[skillName] = opt
		if currentLevel >= maxLevel {
			maxed = append(maxed, skillName)
		} else if opt.Cost > 0 {
			trainables = append(trainables, skillName)
		} else {
			untrainables = append(untrainables, skillName)
		}
	}

	sort.Strings(maxed)
	sort.Strings(trainables)
	sort.Strings(untrainables)

	for _, skillName := range trainables {
		trainingData.Options = append(trainingData.Options, trainOpts[skillName])
	}
	for _, skillName := range maxed {
		trainingData.Options = append(trainingData.Options, trainOpts[skillName])
	}
	for _, skillName := range untrainables {
		trainingData.Options = append(trainingData.Options, trainOpts[skillName])
	}

	if rest == "" {
		user.SendText(buildTrainPanel(trainingData))
	} else {

		user.Character.CancelBuffsWithFlag("hidden")

		allSkills := []string{}
		for skillName, _ := range room.SkillTraining {
			// Skip skills that no longer exist (e.g. deleted from the datafiles).
			if !skills.SkillExists(skillName) {
				continue
			}
			allSkills = append(allSkills, skillName)
		}

		match, closeMatch := util.FindMatchIn(rest, allSkills...)
		if match == "" {
			match = closeMatch
		}

		trainingRange, ok := room.SkillTraining[match]

		currentLevel := user.Character.GetSkillLevel(match)

		if !ok || !skills.SkillExists(match) { // If it's not something that can be learned here
			user.SendText(`The trainer pokes you on your chest, "I think you're in the wrong place, pal."`)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> looks a little confused.`, user.Character.Name),
				user.UserId)
		} else if currentLevel >= skills.MaxSkillLevel(match) { // Max level
			user.SendText(`The trainer chuckles, "I admire your ambition, but you have already mastered that skill!"`)
			room.SendText(
				fmt.Sprintf(`The trainer chuckles and says something you can't quite make out to <ansi fg="username">%s</ansi>`, user.Character.Name),
				user.UserId)
		} else if allowed, reason := archetypes.CanTrain(user.UserId, match); !allowed { // Archetype-claimed skill (Phase 17)
			user.SendText(fmt.Sprintf(`The trainer shakes his head. "%s"`, reason))
		} else if currentLevel < trainingRange.Min-1 { // Not high enough level
			user.SendText(`The trainer shakes his head, "You aren't ready to train here."`)
		} else {

			requiredTrainingPoints := user.Character.GetSkillLevelCost(currentLevel + 1)

			if user.Character.TrainingPoints < requiredTrainingPoints {
				user.SendText(`The trainer pulls you close and says quietly, "You aren't ready yet. Return when you have more experience."`)
				room.SendText(
					fmt.Sprintf(`The trainer pulls <ansi fg="username">%s</ansi> close and mumbles something in their ear.`, user.Character.Name),
					user.UserId)
			} else {

				// Take away the cost
				user.Character.TrainingPoints -= requiredTrainingPoints
				// Upgrade their skill level
				skillName := match
				newLevel := user.Character.TrainSkill(match)

				skillData := struct {
					SkillName  string
					SkillLevel int
				}{
					SkillName:  skillName,
					SkillLevel: newLevel,
				}

				skillUpTxt, _ := templates.Process("character/skillup", skillData, user.UserId)

				user.SendText("The trainer grimly considers you for a moment, and then his demeanor changes dramatically.")
				user.SendText(skillUpTxt)
				user.SendText(`"Congratulations!", the trainer exclaims. You are now a little more prepared for the world.`)
				room.SendText(
					fmt.Sprintf(`The trainer shakes <ansi fg="username">%ss</ansi> hand while congratulating them. Must be nice.`, user.Character.Name),
					user.UserId)

				if match == `tame` {
					if newLevel == 1 {
						user.Character.MobMastery.SetTame(1, 1)
						user.SendText(`You've learned how to tame a <ansi fg="mobname">rat</ansi>!`)
					}
				}
			}

		}

	}

	return true, nil
}

// archetypeOnlyLabel names the archetypes that may train a skill, for the
// train panel ("Cleric or Wizard only.").
func archetypeOnlyLabel(skillName string) string {
	names := archetypes.ClaimantNames(skillName)
	if names == "" {
		return "Not for your archetype."
	}
	return names + " only."
}
