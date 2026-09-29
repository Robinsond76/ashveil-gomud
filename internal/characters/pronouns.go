package characters

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/races"
)

// PronounForms are the third-person forms used by combat narration.
type PronounForms struct {
	Subject    string
	Object     string
	Possessive string
}

var pronounForms = map[string]PronounForms{
	"he":   {Subject: "he", Object: "him", Possessive: "his"},
	"she":  {Subject: "she", Object: "her", Possessive: "her"},
	"they": {Subject: "they", Object: "them", Possessive: "their"},
	"it":   {Subject: "it", Object: "it", Possessive: "its"},
}

// PronounFormsFor returns forms for a supported pronoun value. Unknown values
// safely use the neutral they forms.
func PronounFormsFor(value string) PronounForms {
	if forms, ok := pronounForms[strings.ToLower(strings.TrimSpace(value))]; ok {
		return forms
	}
	return pronounForms["they"]
}

func validPronoun(value string) bool {
	_, ok := pronounForms[strings.ToLower(strings.TrimSpace(value))]
	return ok
}

// CombatPronouns returns an explicit character setting or the effective race's
// default. It does not materialize either fallback on the character.
func (c *Character) CombatPronouns() PronounForms {
	if validPronoun(c.Pronouns) {
		return PronounFormsFor(c.Pronouns)
	}
	if race := races.GetRace(c.GetRaceId()); race != nil {
		return PronounFormsFor(race.DefaultPronouns)
	}
	return PronounFormsFor("")
}
