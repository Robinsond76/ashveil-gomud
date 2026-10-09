package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39i2: the Beast Tamer, Gryphon Rider, Alchemist and Arbalist elites
// are open, ungated and carry the signature effects the hooks read.
func TestRemainingNeutralElitesAreOpen(t *testing.T) {
	for lineage, routes := range map[string]map[string]string{
		"beasttamer":    {"houndmaster": "packlord", "bearward": "beastlord", "dragon-tamer": "dragon-lord"},
		"gryphon-rider": {"gryphon-knight": "gryphon-lord", "skyscout": "falcon-marshal", "wyvern-rider": "wyvern-lord"},
		"alchemist":     {"apothecary": "panacean", "bombardier": "grenadier", "mutagenist": "transmuter"},
		"arbalist":      {"siegebreaker": "siege-master", "sharpshooter": "deadeye", "warden-of-the-wall": "bastion"},
	} {
		for adv, want := range routes {
			elite, ok := Elite(adv)
			require.True(t, ok, adv)
			assert.Equal(t, want, elite.ID)
			assert.False(t, elite.Planned, want)
			assert.Equal(t, GateAny, elite.Gate, want)
			assert.Equal(t, lineage, elite.Lineage, want)
			require.Len(t, elite.Ranks, 7, want)
			for i, r := range elite.Ranks {
				assert.Equal(t, 30+5*i, r.Level, "%s rank %d", want, i)
			}
			for _, align := range []int{-100, 0, 100} {
				_, err := Check(lineage, adv, want, 30, align)
				assert.NoError(t, err, "%s at alignment %d", want, align)
			}
			_, err := Check(lineage, adv, want, 29, 0)
			assert.Error(t, err, want)
		}
	}
}

func TestRemainingNeutralEliteEffects(t *testing.T) {
	at := func(id string, level int) Effects { return EffectsFor(id, level, nil) }

	// Beast Tamer.
	assert.True(t, at("packlord", 30).Has(BeastHunt))
	assert.Equal(t, 75, at("packlord", 45).Int(BeastHobbleAt))
	assert.Equal(t, 50, at("packlord", 60).Int(BeastOpen))
	assert.Zero(t, at("packlord", 55).Int(BeastOpen))
	assert.Equal(t, 50, at("beastlord", 30).Int(BeastSwipe))
	assert.Equal(t, 75, at("beastlord", 50).Int(BeastSwipe))
	assert.Equal(t, 50, at("beastlord", 60).Int(BeastRise))
	assert.Zero(t, at("beastlord", 55).Int(BeastRise))
	assert.True(t, at("dragon-lord", 30).Has(BreathBurn))
	assert.Equal(t, 4, at("dragon-lord", 45).Int(BreathFoes))
	assert.True(t, at("dragon-lord", 60).Has(BreathFirst))

	// Gryphon Rider.
	assert.Equal(t, 1, at("gryphon-lord", 30).Int(DiveCD))
	assert.False(t, at("gryphon-lord", 55).Has(DiveQuake))
	assert.True(t, at("gryphon-lord", 60).Has(DiveQuake))
	assert.Equal(t, 5, at("falcon-marshal", 30).Int(DiveMark))
	assert.Equal(t, 8, at("falcon-marshal", 45).Int(DiveMark))
	assert.Equal(t, 12, at("falcon-marshal", 60).Int(DiveMark))
	assert.Equal(t, 25, at("wyvern-lord", 30).Int(DivePoisX))
	assert.Equal(t, 40, at("wyvern-lord", 45).Int(DivePoisX))
	assert.Zero(t, at("wyvern-lord", 55).Int(DiveTail))
	assert.Equal(t, 50, at("wyvern-lord", 60).Int(DiveTail))

	// Alchemist.
	assert.Equal(t, 25, at("panacean", 30).Int(ElixirSave))
	assert.Equal(t, 40, at("panacean", 50).Int(ElixirSave))
	assert.Equal(t, 1, at("panacean", 55).Int(ElixirUses))
	assert.Equal(t, 2, at("panacean", 60).Int(ElixirUses))
	assert.Equal(t, 2, at("grenadier", 30).Int(FlaskReach), "two more than the two a flask reaches: four foes")
	assert.Equal(t, 3, at("grenadier", 45).Int(FlaskReach), "five foes")
	assert.Positive(t, at("transmuter", 30).Int(MutagenRow))
	assert.Equal(t, 5, at("transmuter", 60).Int(MutagenRow))

	// Arbalist.
	assert.Equal(t, 1, at("siege-master", 30).Int(BoltThrough))
	assert.Equal(t, 2, at("siege-master", 45).Int(BoltThrough))
	assert.Equal(t, 4, at("siege-master", 60).Int(BoltThrough))
	assert.Equal(t, 80, at("siege-master", 55).Int(BoltThroughPct))
	assert.True(t, at("deadeye", 30).Has(ReloadCrit))
	assert.False(t, at("deadeye", 55).Has(ReloadKill))
	assert.True(t, at("deadeye", 60).Has(ReloadKill))
	assert.Equal(t, 2, at("bastion", 30).Int(ColumnShot))
	assert.Equal(t, 3, at("bastion", 45).Int(ColumnShot))
	assert.Equal(t, 100, at("bastion", 55).Int(ColumnShotPct))
}

func TestRemainingNeutralEliteTalents(t *testing.T) {
	names := func(lineage string) (out []string) {
		for _, talent := range EliteTalentsFor(lineage) {
			out = append(out, talent.Name)
		}
		return out
	}
	assert.ElementsMatch(t, []string{"Iron Hide", "Pack Bond", "Savage Jaws"}, names("beasttamer"))
	assert.ElementsMatch(t, []string{"Iron Hide", "Veteran's Edge", "Shadow Footing"}, names("gryphon-rider"))
	assert.ElementsMatch(t, []string{"Iron Hide", "Master Brewer", "Potent Flasks"}, names("alchemist"))
	assert.ElementsMatch(t, []string{"Long Draw", "Eagle Eye", "Quick Nock"}, names("arbalist"))
	// An elite talent opens at 35, and only for an elite character.
	assert.ErrorIs(t, CanPick("arbalist", "sharpshooter", nil, 35, "quick-nock"), ErrEliteTalent)
	assert.NoError(t, CanPick("arbalist", "deadeye", nil, 35, "quick-nock"))
}
