package gmcp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEquipmentExtraLeavesLegacyGearAvailable(t *testing.T) {
	u := inventoryUser(t)
	assert.Nil(t, equipmentExtra().build(u), "legacy Gear must not be replaced by an empty shared catalogue")
	u.Character.CompanyCargo = true
	assert.NotNil(t, equipmentExtra().build(u), "shared Gear reports service availability honestly")
}
