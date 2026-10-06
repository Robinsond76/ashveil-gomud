package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

// Phase 50: a member's needs and meal buff set its battle condition when a
// battle begins (internal/survival's ConditionFor holds the numbers; help
// survival and help cooking). Hunger cuts the damage it deals, thirst makes
// it take more, and fatigue keeps its hit-chance cut (read per blow); a meal
// buff adds its own and counts one battle off. The condition is fixed for the
// battle: eating mid-battle is refused anyway.

// startFare gives each member standing in a battle that has just begun its
// battle condition, restores a meal's mana, counts a battle off the meal
// buffs, and names what is not neutral in one line.
func (sd side) startFare(room *rooms.Room) {
	uid := sd.user.UserId
	needs := map[survival.MemberKey]survival.MemberNeeds{}
	for _, n := range survival.CompanyNeeds(uid) {
		needs[n.Key] = n
	}
	fare := map[string]string{}
	var spent []survival.MemberKey
	var notes []string
	for _, a := range sideActors(sd.user, room) {
		rt := a.char.RTState()
		n, ok := needs[a.key]
		if !ok {
			rt.FareDamage, rt.FareGuard = 0, 0
			continue
		}
		c := survival.ConditionFor(n.Needs)
		rt.FareDamage, rt.FareGuard = c.DamagePct, c.GuardPct
		if c.ManaPct > 0 && a.char.Mana < a.char.ManaMax.Value {
			a.char.Mana = min(a.char.ManaMax.Value, a.char.Mana+max(1, a.char.ManaMax.Value*c.ManaPct/100))
		}
		if c.Meal.Kind != "" {
			spent = append(spent, a.key)
		}
		if sum := c.Summary(n.Needs.MealBattles); sum != "" {
			fare[string(a.key)] = sum
			notes = append(notes, a.char.Name+" "+sum)
		}
	}
	battle.SetFare(uid, fare)
	if err := survival.SpendMealBattle(uid, spent); err != nil {
		mudlog.Warn("fare: meal battle not counted", "leader", uid, "error", err)
	}
	if len(notes) > 0 {
		sd.user.SendText(fmt.Sprintf(`<ansi fg="yellow">Going in:</ansi> %s.`, strings.Join(notes, "; ")))
	}
}
