package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 32f: packs and saddles are item data.
func TestLogisticsFieldsDecode(t *testing.T) {
	var spec ItemSpec
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 31\nname: satchel\ncarrybonus: 5000\n"), &spec))
	assert.Equal(t, 5000, spec.CarryBonus)
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 34\nname: pack saddle\nsaddle: pack\n"), &spec))
	assert.Equal(t, SaddlePack, spec.Saddle)
}

func TestCarryBonusAndSaddleReadBaseData(t *testing.T) {
	SetTestItemSpec(&ItemSpec{ItemId: 988110, Name: "satchel", CarryBonus: 5000})
	SetTestItemSpec(&ItemSpec{ItemId: 988111, Name: "saddle", Saddle: SaddleRiding})
	t.Cleanup(func() { RemoveTestItemSpec(988110); RemoveTestItemSpec(988111) })

	frozen := Item{ItemId: 988110, Spec: &ItemSpec{ItemId: 988110, Name: "old satchel"}}
	assert.Equal(t, 5000, frozen.CarryBonusGrams(), "an old spec copy doesn't hide the bonus")
	assert.Equal(t, SaddleRiding, (&Item{ItemId: 988111}).SaddleKind())
	orphan := Item{ItemId: 988112, Spec: &ItemSpec{ItemId: 988112, CarryBonus: 70, Saddle: SaddlePack}}
	assert.Equal(t, 70, orphan.CarryBonusGrams())
	assert.Equal(t, SaddlePack, orphan.SaddleKind())
	assert.Zero(t, (&Item{ItemId: 988113}).CarryBonusGrams())
	assert.Equal(t, SaddleKind(""), (&Item{ItemId: 988113}).SaddleKind())
}
