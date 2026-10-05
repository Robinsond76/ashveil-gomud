package web

import (
	"encoding/json"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"testing"
)

type previewArchetypes struct{}

func (previewArchetypes) CanTrain(int, string) (bool, string)      { return true, "" }
func (previewArchetypes) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (previewArchetypes) Exists(string) bool                       { return true }
func (previewArchetypes) ArchetypeName(id string) (string, bool)   { return id, true }
func (previewArchetypes) PlayerArchetype(int) (string, bool)       { return "warrior", true }
func (previewArchetypes) HealthPerLevel(id string) (float64, bool) {
	r, ok := map[string]float64{"warrior": 6, "wizard": 3}[id]
	return r, ok
}
func (previewArchetypes) HealthArchetypes() map[string]float64 {
	return map[string]float64{"warrior": 6, "wizard": 3}
}

func TestProgressionPreviewParityAndDownsampledXP(t *testing.T) {
	archetypes.SetProvider(previewArchetypes{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	cfg := configs.GetProgressionConfig()
	for _, query := range []string{"", "?MaxLevel=300", "?StatStepLevels=10&HPFullLevels=10&HPAfterFull=2&XPKneeLevel=20&XPKneeGrowth=1.2"} {
		rec := httptest.NewRecorder()
		apiV1GetProgressionPreview(rec, httptest.NewRequest("GET", "/admin/api/v1/progression/preview"+query, nil))
		require.Equal(t, 200, rec.Code)
		if path := os.Getenv("PROGRESSION_PREVIEW_CAPTURE"); path != "" && query == "" {
			require.NoError(t, os.WriteFile(path, rec.Body.Bytes(), 0600))
		}
		var envelope APIResponse[progressionPreviewData]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
		require.True(t, envelope.Success)
		d := envelope.Data
		local := cfg
		if query != "" && query != "?MaxLevel=300" {
			local.StatStepLevels = 10
			local.HPFullLevels = 10
			local.HPAfterFull = 2
			local.XPKneeLevel = 20
			local.XPKneeGrowth = 1.2
		}
		for i, level := range d.Levels {
			assert.Equal(t, local.XPThreshold(level, 1), d.XPCumulative[i])
			prev := 0
			if level > 1 {
				prev = local.XPThreshold(level-1, 1)
			}
			assert.Equal(t, local.XPThreshold(level, 1)-prev, d.XPPerLevel[i])
			v := applyCapWithCfg(gainsForLevelWithCfg(level, 3, local), local)
			assert.Equal(t, local.HealthAtLevel(level, v, 6, 0), d.HP["warrior"][i])
			assert.Equal(t, local.HealthAtLevel(level, v, 3, 0), d.HP["wizard"][i])
			gameplay := configs.GetGamePlayConfig()
			gameplay.Progression = local
			restore := configs.SetTestGamePlayConfig(gameplay)
			manaCharacter := characters.New()
			manaCharacter.Level = level
			manaCharacter.Stats.Mysticism.Base = 3
			manaCharacter.RecalculateStats()
			assert.Equal(t, manaCharacter.ManaMax.Value, d.Mana["base3"][i])
			restore()
			if query == "" {
				stat := stats.StatInfo{Base: 3}
				stat.Recalculate(level)
				assert.Equal(t, stat.Value, d.StatGains["base3"][i])
				c := characters.New()
				c.SetUserId(7)
				c.Level = level
				c.Stats.Vitality.Base = 3
				c.Stats.Mysticism.Base = 3
				c.RecalculateStats()
				assert.Equal(t, c.HealthMax.Value, d.HP["warrior"][i])
				assert.Equal(t, c.XPTL(level), d.XPCumulative[i])
				assert.Equal(t, c.ManaMax.Value, d.Mana["base3"][i])
			}
		}
	}
	assert.Equal(t, cfg, configs.GetProgressionConfig(), "preview never mutates live progression")
}
