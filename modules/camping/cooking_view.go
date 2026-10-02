package camping

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"strings"
)

func (m *CampingModule) CookingCapability(userID int) (camping.CookingView, bool) {
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return camping.CookingView{}, false
	}
	v := camping.CookingView{Rank: u.Character.GetSkillLevel("cooking")}
	recipes := m.campSettings().Recipes
	descriptions := []string{}
	for _, r := range recipes {
		need := "no trained skill required"
		if r.Skill != "" {
			need = fmt.Sprintf("%s rank %d", r.Skill, r.MinLevel)
		}
		descriptions = append(descriptions, itemName(r.Output)+" requires "+need)
	}
	v.Description = "Manual camp cook at your own lit campfire, outside battle, with ingredients and cargo space. " + strings.Join(descriptions, "; ")
	chosen, blocked := selectCampRecipe(u, recipes)
	m.mu.Lock()
	camp, has := m.camps[userID]
	m.mu.Unlock()
	switch {
	case m.persistenceAvailable() != nil:
		v.Reason = "Camp service unavailable"
	case len(recipes) == 0:
		v.Reason = "No configured camp recipes"
	case chosen == nil && blocked != nil:
		v.Reason = fmt.Sprintf("%s needs %s rank %d", itemName(blocked.Output), blocked.Skill, blocked.MinLevel)
	case chosen == nil:
		v.Reason = "Missing recipe ingredients or trained ranks"
	case !has || camp.RoomID != u.Character.RoomId:
		v.Reason = "Requires your own camp here"
	case !camp.FireLit:
		v.Reason = "Requires a lit campfire"
	case m.inBattle != nil && m.inBattle(userID):
		v.Reason = "Unavailable in battle"
	default:
		delta := 0
		if spec := items.GetItemSpec(chosen.Output); spec != nil {
			delta = spec.Weight
		}
		for _, id := range chosen.Inputs {
			if spec := items.GetItemSpec(id); spec != nil {
				delta -= spec.Weight
			}
		}
		if _, full := encumbrance.WouldExceed(userID, delta); full {
			v.Reason = "Finished meal exceeds cargo capacity"
		}
	}
	v.Ready = v.Reason == ""
	return v, true
}
