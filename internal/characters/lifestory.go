package characters

// Ashveil Phase 72a: a player's looks and life story. The picks live in
// Character.Looks and Character.LifeStory; the description `look` shows is
// composed from the looks, and the life story's stat bonus is derived from
// its picks on every read (see lifestory.StatBonus), so it can't apply twice.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/races"
)

// creationOfferedKey marks a character that has been through (or declined)
// the looks and life story steps. A character without it is legacy: it is
// offered the steps once, at login.
const creationOfferedKey = `creation-offered`

// ErrLifeStorySet is returned when a life story is chosen a second time.
var ErrLifeStorySet = errors.New("the life story is already written")

func (c *Character) lifeStoryStatMod(statName string) int {
	if len(c.LifeStory) == 0 {
		return 0
	}
	return lifestory.StatBonus(c.LifeStory)[statName]
}

// HasLooks reports whether the character has chosen looks.
func (c *Character) HasLooks() bool { return len(c.Looks) > 0 }

// HasLifeStory reports whether the character has a life story.
func (c *Character) HasLifeStory() bool { return len(c.LifeStory) > 0 }

// Background is the character's trade option id (the id phase 72 checks),
// or "" without a life story.
func (c *Character) Background() string { return lifestory.Background(c.LifeStory) }

// CreationOffered reports whether the looks and life story steps were
// finished or declined.
func (c *Character) CreationOffered() bool {
	v, _ := c.GetMiscData(creationOfferedKey).(bool)
	return v
}

// MarkCreationOffered records that the steps need not be offered again.
func (c *Character) MarkCreationOffered() { c.SetMiscData(creationOfferedKey, true) }

// RaceKey is the lowercase name of the character's true race, which the
// looks bands that name races are matched against.
func (c *Character) RaceKey() string {
	if r := races.GetRace(c.RaceId); r != nil {
		return strings.ToLower(r.Name)
	}
	return ``
}

// LooksPronouns are the forms the composers fill text with: the picked
// pronouns, else the race default.
func (c *Character) LooksPronouns() (appearance.Pronouns, lifestory.Pronouns) {
	f := c.CombatPronouns()
	return appearance.Pronouns{Subject: f.Subject, Object: f.Object, Possessive: f.Possessive},
		lifestory.Pronouns{Subject: f.Subject, Object: f.Object, Possessive: f.Possessive}
}

// SetLooks checks the picks against the data and the character's race,
// stores them, sets the pronouns, and writes the composed description.
// Looks have no other effect.
func (c *Character) SetLooks(d *appearance.Data, l appearance.Looks) error {
	if d == nil {
		return errors.New("looks are not available")
	}
	if err := d.Check(l, c.RaceKey()); err != nil {
		return err
	}
	stored := make(map[string]string, len(l))
	for k, v := range l {
		if v != `` {
			stored[k] = v
		}
	}
	c.Looks = stored
	c.Pronouns = stored[appearance.TraitPronouns]
	c.RecomposeDescription(d)
	return nil
}

// RecomposeDescription rewrites the description from the stored looks.
func (c *Character) RecomposeDescription(d *appearance.Data) {
	if d == nil || len(c.Looks) == 0 {
		return
	}
	p, _ := c.LooksPronouns()
	c.Description = d.Compose(appearance.Looks(c.Looks), p)
}

// ApplyLifeStory records the life story, grants the trade's keepsake and
// skill familiarity, and reports the keepsake's name ("" when none). The
// story is written once: a second call returns ErrLifeStorySet and changes
// nothing. The stat bonus needs no grant; it is read from the picks.
func (c *Character) ApplyLifeStory(d *lifestory.Data, picks lifestory.Picks) (keepsake string, err error) {
	if d == nil {
		return ``, errors.New("life stories are not available")
	}
	if c.HasLifeStory() {
		return ``, ErrLifeStorySet
	}
	if err := d.Check(picks); err != nil {
		return ``, err
	}
	stored := make(map[string]string, len(picks))
	for k, v := range picks {
		if v != `` {
			stored[k] = v
		}
	}
	trade, _ := d.Trade(picks)
	if trade.Keepsake > 0 {
		item := items.New(trade.Keepsake)
		if item.ItemId == 0 {
			return ``, fmt.Errorf("keepsake item %d does not exist", trade.Keepsake)
		}
		c.StoreItem(item)
		keepsake = item.DisplayName()
	}
	if trade.Skill != `` && c.GetSkillLevel(trade.Skill) < 1 {
		c.SetSkill(trade.Skill, 1)
	}
	c.LifeStory = stored
	return keepsake, nil
}

// LookColors are the skin and hair colours (#rrggbb) of the character's
// looks, which the web client paints on the player's sprites; "" without
// looks (or looks data).
func (c *Character) LookColors() (skin, hair string) {
	if len(c.Looks) == 0 {
		return ``, ``
	}
	d := appearance.Current()
	if d == nil {
		return ``, ``
	}
	return d.Colors(appearance.Looks(c.Looks))
}
