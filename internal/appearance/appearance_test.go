package appearance

import (
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shipped(t *testing.T) *Data {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "looks.yaml"))
	require.NoError(t, err)
	require.NoError(t, LoadBytes(raw))
	t.Cleanup(func() { SetData(nil) })
	return Current()
}

var (
	he   = Pronouns{Subject: "he", Object: "him", Possessive: "his"}
	she  = Pronouns{Subject: "she", Object: "her", Possessive: "her"}
	they = Pronouns{Subject: "they", Object: "them", Possessive: "their"}
)

func pronounsFor(id string) Pronouns {
	switch id {
	case "he":
		return he
	case "she":
		return she
	}
	return they
}

func baseline(d *Data) Looks {
	l := Looks{}
	for _, id := range SingleTraits {
		l[id] = d.Trait(id).Options[0].ID
	}
	l[TraitMark] = NoMark
	return l
}

func TestShippedLooksValidate(t *testing.T) {
	d := shipped(t)
	for _, id := range append(append([]string{}, SingleTraits...), TraitMark) {
		require.NotNil(t, d.Trait(id), id)
	}
	assert.Equal(t, 160, d.Line())
	assert.Equal(t, 2, d.Marks())
}

// The composer's description has to read as prose for every band: a capital,
// a closing full stop, no stray placeholders or doubled punctuation.
func TestComposeIsWellFormedForEveryBand(t *testing.T) {
	d := shipped(t)
	bad := regexp.MustCompile(`\{|\}|\s{2,}|,\s*,|\s,|,\.|\.\.`)
	check := func(l Looks) {
		text := d.Compose(l, pronounsFor(l[TraitPronouns]))
		require.NotEmpty(t, text)
		assert.Regexp(t, `^An? [a-z]`, text, text)
		assert.True(t, strings.HasSuffix(text, "."), text)
		assert.False(t, bad.MatchString(text), text)
	}
	// every band against the baseline
	for _, tr := range d.Traits {
		for _, o := range tr.Options {
			l := baseline(d)
			if tr.ID == TraitMark {
				l[TraitMark] = o.ID
			} else {
				l[tr.ID] = o.ID
			}
			check(l)
		}
	}
	// and a thousand random combinations with up to two marks
	rng := rand.New(rand.NewSource(72))
	for i := 0; i < 1000; i++ {
		l := Looks{}
		for _, tr := range d.Traits {
			l[tr.ID] = tr.Options[rng.Intn(len(tr.Options))].ID
		}
		if l[TraitMark] != NoMark {
			l[KeyMark2] = d.Trait(TraitMark).Options[rng.Intn(len(d.Trait(TraitMark).Options))].ID
		}
		check(l)
	}
}

func TestComposeExample(t *testing.T) {
	d := shipped(t)
	l := baseline(d)
	l[TraitPronouns], l[TraitHeight], l[TraitBuild] = "she", "tall", "gaunt"
	l[TraitAge], l[TraitFace], l[TraitEyes] = "forty", "plain", "grey"
	l[TraitSkin], l[TraitHairColor], l[TraitHairStyle] = "olive", "black", "cropped"
	l[TraitVoice], l[TraitMark] = "low", "burn"
	l[KeyLine] = "She never speaks of the fire."
	assert.Equal(t,
		"A tall, gaunt woman past forty, grey-eyed, olive-skinned, her black hair cropped close. Her voice is low. A burn scar runs across the back of her left hand. She never speaks of the fire.",
		d.Compose(l, she))
	// they/them uses their, and an adjective starting with a vowel takes an
	l[TraitPronouns], l[TraitHeight], l[TraitBuild] = "they", "middling", "wiry"
	l[TraitFace], l[TraitAge] = "plain", "young"
	assert.True(t, strings.HasPrefix(d.Compose(l, they), "A wiry, young person"), d.Compose(l, they))
	l[TraitBuild] = "heavy"
	l[TraitHeight] = "short"
	l[TraitAge] = "old"
	assert.True(t, strings.HasPrefix(d.Compose(l, they), "A short, heavyset, old person"))
	l[TraitHeight], l[TraitBuild], l[TraitAge] = "middling", "lean", "old"
	l[TraitBuild] = "gaunt"
	assert.True(t, strings.HasPrefix(d.Compose(l, they), "A gaunt, old person"))
	l[TraitBuild] = "broad"
	l[TraitHeight] = "very-tall"
	assert.True(t, strings.HasPrefix(d.Compose(l, he), "A towering"))
	l[TraitHeight], l[TraitAge], l[TraitFace], l[TraitBuild] = "middling", "prime", "plain", "sturdy"
	l[TraitPronouns] = "he"
	assert.True(t, strings.HasPrefix(d.Compose(l, he), "A sturdy man in his prime"), d.Compose(l, he))
}

func TestComposeAnBeforeVowel(t *testing.T) {
	d := &Data{Traits: []Trait{
		{ID: TraitPronouns, Options: []Option{{ID: "he", Noun: "man"}}},
		{ID: TraitAge, Options: []Option{{ID: "old", Adj: "old"}}},
	}}
	assert.Equal(t, "An old man.", d.Compose(Looks{TraitPronouns: "he", TraitAge: "old"}, he))
}

func TestComposePartialPicksStillReads(t *testing.T) {
	d := shipped(t)
	assert.Equal(t, "A person.", d.Compose(Looks{}, they))
	assert.Equal(t, "A woman.", d.Compose(Looks{TraitPronouns: "she"}, she))
}

func TestRaceLimitsOptions(t *testing.T) {
	d := shipped(t)
	has := func(race string) bool {
		for _, o := range d.Options(TraitAge, race) {
			if o.ID == "ageless" {
				return true
			}
		}
		return false
	}
	assert.True(t, has("elf"))
	assert.True(t, has("Elf"))
	assert.False(t, has("human"))
	assert.False(t, has(""))
	assert.Len(t, d.Options(TraitAge, "human"), len(d.Trait(TraitAge).Options)-1)
}

func TestCheckRejectsBadPicks(t *testing.T) {
	d := shipped(t)
	l := baseline(d)
	require.NoError(t, d.Check(l, "human"))

	missing := Looks{}
	for k, v := range l {
		missing[k] = v
	}
	delete(missing, TraitEyes)
	assert.Error(t, d.Check(missing, "human"))

	l[TraitAge] = "ageless"
	assert.Error(t, d.Check(l, "human"), "an elf-only band is closed to humans")
	assert.NoError(t, d.Check(l, "elf"))

	l = baseline(d)
	l[TraitHeight] = "gigantic"
	assert.Error(t, d.Check(l, "human"))

	l = baseline(d)
	l[KeyMark2] = "burn"
	assert.Error(t, d.Check(l, "human"), "a second mark needs a first")
	l[TraitMark] = "cut"
	assert.NoError(t, d.Check(l, "human"))

	l[KeyLine] = strings.Repeat("x", 161)
	assert.Error(t, d.Check(l, "human"))
}

func TestMatch(t *testing.T) {
	opts := []Option{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "past forty"}}
	for _, in := range []string{"2", "b", "PAST FORTY", " 2 "} {
		o, ok := Match(opts, in)
		require.True(t, ok, in)
		assert.Equal(t, "b", o.ID)
	}
	for _, in := range []string{"", "0", "3", "zzz"} {
		_, ok := Match(opts, in)
		assert.False(t, ok, in)
	}
}

func TestCleanLine(t *testing.T) {
	banned := func(s string) (string, bool) {
		if strings.Contains(strings.ToLower(s), "badword") {
			return "*badword*", true
		}
		return "", false
	}
	got, err := CleanLine("  A  <ansi fg=\"red\">scar</ansi>\tover {the} eye\n", 160, banned)
	require.NoError(t, err)
	assert.Equal(t, `A ansi fg="red"scar/ansi over the eye`, got)
	assert.NotContains(t, got, "<")

	_, err = CleanLine(strings.Repeat("a", 161), 160, banned)
	assert.Error(t, err)
	_, err = CleanLine("a BADWORD here", 160, banned)
	assert.Error(t, err)
	got, err = CleanLine("   ", 160, banned)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestValidateRejectsMalformedData(t *testing.T) {
	good := func() *Data {
		raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "looks.yaml"))
		require.NoError(t, err)
		var d Data
		require.NoError(t, yamlUnmarshal(raw, &d))
		return &d
	}
	require.NoError(t, good().Validate())

	d := good()
	d.Traits = d.Traits[1:]
	assert.Error(t, d.Validate(), "missing pronouns trait")

	d = good()
	d.Trait(TraitSkin).Options[0].Color = "red"
	assert.Error(t, d.Validate())

	d = good()
	d.Trait(TraitVoice).Options[0].Phrase = "{Hisser} voice."
	assert.Error(t, d.Validate())

	d = good()
	d.Trait(TraitMark).Options = d.Trait(TraitMark).Options[1:]
	assert.Error(t, d.Validate(), "the mark trait needs none")
}

// Review regression: a free line CleanLine accepts (counted in characters)
// must pass Check too; accented letters take two bytes each.
func TestCheckCountsTheFreeLineInCharacters(t *testing.T) {
	d := shipped(t)
	line, err := CleanLine(strings.Repeat("é", d.Line()), d.Line(), nil)
	require.NoError(t, err)
	l := baseline(d)
	l[KeyLine] = line
	assert.NoError(t, d.Check(l, "human"))
	l[KeyLine] = line + "é"
	assert.Error(t, d.Check(l, "human"))
}
