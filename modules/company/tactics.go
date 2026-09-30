package company

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 30c: company tactics. The company-wide focus (a target rule every
// member aims by instead of its own) and the healing threshold, stored by
// modules/strategy (internal/strategy.SaveTactics). In a battle only the
// focus may change, for that battle only, one order a round
// (internal/battle); the next round's upkeep turns everyone at once.

const tacticsUsage = `Usage: company tactics | company tactics focus <none|leader|casters|nearest|weakest|strongest|wounded|default> | company tactics healing <10-90> | company tactics default`

// tacticsStillTurning answers a second order before the first is carried
// out, or one given before the battle has begun.
const tacticsStillTurning = `Your company is still turning; try again next round.`

// tacticsNotYet answers an order given after attack but before the battle
// has begun (the next round begins it).
const tacticsNotYet = `The battle hasn't begun yet; call a focus once it has, next round.`

// tacticsOnlyFocus answers any other tactics change in a battle.
const tacticsOnlyFocus = `In the middle of a battle you can only call a new focus (<ansi fg="command">company tactics focus [rule]</ansi>).`

// tacticsCommand is the "tactics" shorthand for "company tactics".
func (m *CompanyModule) tacticsCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.tactics(user, room, strings.Fields(strings.ToLower(rest))))
	return true, nil
}

// tactics runs company tactics with its arguments and returns the answer.
func (m *CompanyModule) tactics(user *users.UserRecord, room *rooms.Room, args []string) string {
	if len(args) == 0 {
		return m.tacticsView(user)
	}
	inBattle := usercommands.InBattle(user)
	saved := strategy.TacticsFor(user.UserId)
	switch args[0] {
	case "focus", "target", "on":
		if len(args) != 2 {
			return tacticsUsage
		}
		if inBattle {
			return m.orderFocus(user, room, args[1])
		}
		focus := strategy.NoFocus
		if args[1] != "default" {
			f, ok := strategy.ParseFocus(args[1])
			if !ok {
				return fmt.Sprintf("%q is no focus. %s", args[1], focusChoices())
			}
			focus = f
		}
		saved.Focus = focus
		if err := strategy.SaveTactics(user.UserId, saved); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Your company's focus is now %s: %s.", focus, strategy.DescribeFocus(focus))
	case "healing", "heal":
		if inBattle {
			return tacticsOnlyFocus
		}
		if len(args) != 2 {
			return tacticsUsage
		}
		h, ok := strategy.ParseHealing(args[1])
		if !ok {
			return "The healing threshold is a percentage from 10 to 90, in tens: e.g. company tactics healing 70."
		}
		saved.Healing = h
		if err := strategy.SaveTactics(user.UserId, saved); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Your healers now heal anyone below %d%% of their health.", h)
	case "default", "reset":
		if inBattle {
			return tacticsOnlyFocus
		}
		if err := strategy.SaveTactics(user.UserId, strategy.Tactics{}); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Your company's tactics are back to the defaults: focus none, healing below %d%%.", strategy.DefaultHealing)
	}
	return tacticsUsage
}

func focusChoices() string {
	names := make([]string, len(strategy.FocusRules))
	for i, r := range strategy.FocusRules {
		names[i] = string(r)
	}
	return "Choose one of: " + strings.Join(names, ", ") + ", or default."
}

// orderFocus calls a focus for this battle only.
func (m *CompanyModule) orderFocus(user *users.UserRecord, room *rooms.Room, value string) string {
	saved := strategy.TacticsFor(user.UserId)
	var err error
	rule := saved.Focus
	if value == "default" {
		err = battle.ClearFocus(user.UserId)
	} else {
		f, ok := strategy.ParseFocus(value)
		if !ok {
			return fmt.Sprintf("%q is no focus. %s", value, focusChoices())
		}
		rule = f
		err = battle.SetFocus(user.UserId, string(f))
	}
	if errors.Is(err, battle.ErrNoBattle) {
		return tacticsNotYet
	}
	if errors.Is(err, battle.ErrFocusPending) {
		return tacticsStillTurning
	}
	if err != nil {
		return err.Error()
	}
	you, them := callLines(user, room, rule)
	if room != nil {
		room.SendText(them, user.UserId)
	}
	return you
}

// callLines are the order as the player and the room hear it: the focus's
// choice among every foe of the battle standing, or the rule when it has
// none.
func callLines(user *users.UserRecord, room *rooms.Room, rule strategy.Rule) (you, them string) {
	name := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, user.Character.Name)
	their := user.Character.CombatPronouns().Possessive
	if rule == "" || rule == strategy.NoFocus {
		return `You let each of your company choose their own foe.`,
			fmt.Sprintf(`%s lets each of %s company choose their own foe.`, name, their)
	}
	onto := focusWords(rule)
	if b, ok := battle.Current(user.UserId); ok && room != nil {
		if p, found := enemyparty.BattleParty(b, enemyparty.Parties(room)); found {
			att := enemyparty.PlayerAttacker(user)
			att.Rule, att.Spell = rule, true // among every foe standing, reach aside
			if id, ok := enemyparty.RuleChoice(enemyparty.Group{Party: p}, att, 0); ok {
				if mob := mobs.GetInstance(id); mob != nil {
					onto = util.Article(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, battle.EnemyDisplayName(id, mob.Character.Name)))
				}
			}
		}
	}
	return fmt.Sprintf(`You call the company onto %s.`, onto),
		fmt.Sprintf(`%s calls %s company onto %s.`, name, their, onto)
}

// focusWords names what a focus goes for when no one foe answers it.
func focusWords(rule strategy.Rule) string {
	switch rule {
	case strategy.Leader:
		return "their leader"
	case strategy.Casters:
		return "their spell-casters"
	case strategy.Nearest:
		return "the nearest of them"
	case strategy.Strongest:
		return "their strongest"
	case strategy.Wounded:
		return "their most wounded"
	}
	return "their weakest"
}

// tacticsView shows the tactics, this battle's order, and each member's
// own strategy.
func (m *CompanyModule) tacticsView(user *users.UserRecord) string {
	t := strategy.TacticsFor(user.UserId)
	var b strings.Builder
	b.WriteString("Company tactics:\n")
	fmt.Fprintf(&b, "  Focus:   %s (%s)\n", t.Focus, strategy.DescribeFocus(t.Focus))
	fmt.Fprintf(&b, "  Healing: your healers heal anyone below %d%% of their health\n", t.Healing)
	if _, inBattle := battle.Current(user.UserId); inBattle {
		if r, set := battle.Focus(user.UserId); set {
			fmt.Fprintf(&b, "  In this battle: focus %s, until it ends", r)
		} else {
			fmt.Fprintf(&b, "  In this battle: your saved focus")
		}
		if battle.FocusReady(user.UserId) {
			b.WriteString(" (ready for an order)\n")
		} else {
			b.WriteString(" (turning next round)\n")
		}
	}
	_, focused := enemyparty.Focus(user.UserId)
	b.WriteString("Each member's own strategy")
	if focused {
		b.WriteString(" (the focus overrides whom they go for; roles stay)")
	}
	b.WriteString(":\n")
	for _, mb := range m.tacticsMembers(user) {
		s := enemyparty.MemberStrategy(user.UserId, mb.key)
		fmt.Fprintf(&b, "  %-16s %s, %s\n", mb.name, s.Role, s.Rule)
	}
	b.WriteString(`Change them with <ansi fg="command">company tactics focus [rule]</ansi>, <ansi fg="command">company tactics healing [percent]</ansi>, or <ansi fg="command">company tactics default</ansi>. See <ansi fg="command">help tactics</ansi>.`)
	return b.String()
}

type tacticsMember struct {
	key  domain.MemberKey
	name string
}

// tacticsMembers are the player and their living companions, in roster
// order.
func (m *CompanyModule) tacticsMembers(user *users.UserRecord) []tacticsMember {
	out := []tacticsMember{{key: domain.LeaderMemberKey, name: "You"}}
	record, ok := m.registry.Get(user.UserId)
	if !ok {
		return out
	}
	for _, c := range record.Companions {
		if c.Death != nil {
			continue
		}
		out = append(out, tacticsMember{key: domain.CompanionMemberKey(c.ID), name: nameOf(c, templateName(c.MobTemplateID, "a companion"))})
	}
	return out
}
