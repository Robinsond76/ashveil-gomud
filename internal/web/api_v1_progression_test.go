package web

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"math"
	"net/http/httptest"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
)

func TestXPTLWithCfgClampsConsistentlyWithCharacter(t *testing.T) {
	const highLevel = 200_000_000

	cfg := configs.GetProgressionConfig()
	previewXP := xpTLWithCfg(highLevel, cfg)
	engineXP := characters.New().XPTL(highLevel)

	assert.Equal(t, math.MaxInt, previewXP)
	assert.Equal(t, engineXP, previewXP)
}

func TestProgressionPreviewSmoothStatsAndPoints(t *testing.T) {
	for _, smooth := range []bool{false, true} {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/api/v1/progression/preview?SmoothStatGrowth=%t&StatPointsEveryNLevels=2&StatPointsPerLevel=1&MaxLevel=200", smooth), nil)
		rec := httptest.NewRecorder()
		apiV1GetProgressionPreview(rec, req)
		var got APIResponse[progressionPreviewData]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.True(t, got.Success)
		cfg := configs.GetProgressionConfig()
		cfg.SmoothStatGrowth = configs.ConfigBool(smooth)
		cfg.StatPointsEveryNLevels = 2
		cfg.StatPointsPerLevel = 1
		cfg.Validate()
		for i, level := range got.Data.Levels {
			assert.Equal(t, cfg.RacialForLevel(level, 3), got.Data.StatGains["base3"][i])
			assert.Equal(t, cfg.StatPointsAt(level), got.Data.StatPoints[i], "downsampled charts use cumulative rule")
			assert.Equal(t, cfg.HealthAtLevel(level, cfg.CompressStat(cfg.RacialForLevel(level, 3)), float64(cfg.DefaultHPPerLevel), 0), got.Data.HP["base3"][i])
		}
	}
}
