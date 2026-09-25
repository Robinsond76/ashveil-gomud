package companyview

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/stretchr/testify/assert"
)

func TestWarmthLabel(t *testing.T) {
	for exposure, want := range map[int]string{
		0: "", 24: "", -24: "",
		-25: "Chilled", -50: "Frostbitten", -75: "Hypothermic", -100: "Freezing to Death",
		25: "Overheated", 50: "Heatstricken", 75: "Heat Exhausted", 100: "Heatstroke",
	} {
		assert.Equal(t, want, WarmthLabel(exposure), exposure)
	}
}

func TestLightLabel(t *testing.T) {
	assert.Equal(t, "Dark", LightLabel(0))
	assert.Equal(t, "Dim", LightLabel(1))
	assert.Equal(t, "", LightLabel(2))
}

func TestLoadLabel(t *testing.T) {
	neutral := encumbrance.LoadBand{TravelDurationPct: 100, FatiguePct: 100}
	slowed := encumbrance.LoadBand{MinRatio: 0.75, TravelDurationPct: 120, FatiguePct: 130}
	assert.Equal(t, "Unburdened", LoadLabel(encumbrance.Load{PersonalGrams: 1000, CapacityGrams: 10000}, neutral))
	assert.Equal(t, "Burdened", LoadLabel(encumbrance.Load{PersonalGrams: 8000, CapacityGrams: 10000}, slowed))
	assert.Equal(t, "Overloaded", LoadLabel(encumbrance.Load{PersonalGrams: 6000, CargoGrams: 4000, CapacityGrams: 10000}, slowed))
	assert.Equal(t, "Overloaded", LoadLabel(encumbrance.Load{PersonalGrams: 1, CapacityGrams: 0}, neutral), "any load over no capacity")
	assert.Equal(t, "Unburdened", LoadLabel(encumbrance.Load{}, neutral))
}

func TestActivityLabel(t *testing.T) {
	for _, tc := range []struct {
		a    Activity
		want string
	}{
		{Activity{}, ""},
		{Activity{Kind: Travelling, Percent: 42}, "Travelling 42%"},
		{Activity{Kind: Stopped, Percent: 60}, "Stopped"},
		{Activity{Kind: CampRest, Remaining: 12*time.Minute + 5*time.Second}, "Resting 12m"},
		{Activity{Kind: InnStay, Remaining: 5 * time.Minute}, "At inn 5m"},
		{Activity{Kind: Camped}, "Camped"},
	} {
		assert.Equal(t, tc.want, tc.a.Label(), tc.want)
	}
}

func TestFormatRemaining(t *testing.T) {
	assert.Equal(t, "45s", FormatRemaining(45*time.Second))
	assert.Equal(t, "12m", FormatRemaining(12*time.Minute+30*time.Second))
	assert.Equal(t, "1h 5m", FormatRemaining(65*time.Minute))
	assert.Equal(t, "0s", FormatRemaining(-time.Second))
}

func TestCompanyCount(t *testing.T) {
	assert.Equal(t, "1", CompanyCount(1, 0))
	assert.Equal(t, "3, 1 dead", CompanyCount(3, 1))
}

func TestWarnCluster(t *testing.T) {
	assert.Equal(t, "", WarnCluster(nil))
	assert.Equal(t, " Hungry Tired", WarnCluster([]string{"Hungry", "", "Tired"}))
	assert.Equal(t, " A B C D", WarnCluster([]string{"A", "B", "C", "D", "E"}), "at most four words")
}

func TestNeedWarns(t *testing.T) {
	assert.False(t, Need{Known: true, Value: 80}.Warns())
	assert.False(t, Need{Known: true, Value: 51}.Warns())
	assert.True(t, Need{Known: true, Value: 50}.Warns())
	assert.True(t, Need{Known: true, Value: 0}.Warns())
	assert.False(t, Need{Value: 0}.Warns(), "unknown never warns")
}
