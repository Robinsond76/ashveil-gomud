package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// TestBlockChanceRange verifies block chance is within the configured range.
func TestBlockChanceRange(t *testing.T) {
	cfg := configs.GetCombatConfig()
	minBlock := int(cfg.BlockChanceMin)
	maxBlock := int(cfg.BlockChanceMax)

	// Test edge cases
	tests := []struct {
		name        string
		shieldArmor int
		defStr      int
		atkStr      int
	}{
		{"equal stats", 5, 50, 50},
		{"defender stronger", 5, 75, 50},
		{"attacker stronger", 5, 25, 50},
		{"max shield armor", 10, 50, 50},
		{"min shield armor", 1, 50, 50},
	}

	for _, tt := range tests {
		got := blockChance(tt.shieldArmor, tt.defStr, tt.atkStr)
		if got < minBlock || got > maxBlock {
			t.Errorf("%s: blockChance = %d, want in range [%d, %d]", tt.name, got, minBlock, maxBlock)
		}
	}
}

// TestBlockChanceIncreaseWithStrength verifies shield block improves with Strength advantage.
func TestBlockChanceIncreaseWithStrength(t *testing.T) {
	// Same armor, different strength
	weak := blockChance(5, 50, 50)     // equal
	stronger := blockChance(5, 75, 50) // +25 strength

	if stronger <= weak {
		t.Errorf("blockChance should increase with strength: weak=%d, stronger=%d", weak, stronger)
	}
}

// TestParryChanceRange verifies parry chance is within the configured range.
func TestParryChanceRange(t *testing.T) {
	cfg := configs.GetCombatConfig()
	minParry := int(cfg.ParryChanceMin)
	maxParry := int(cfg.ParryChanceMax)

	// Test various speed deltas and weapon types
	tests := []struct {
		name       string
		defSpeed   int
		atkSpeed   int
		weaponType items.ItemSubType
	}{
		{"equal stats, slashing", 50, 50, items.Slashing},
		{"faster defender, slashing", 75, 50, items.Slashing},
		{"slower defender, stabbing", 25, 50, items.Stabbing},
		{"equal stats, bludgeoning", 50, 50, items.Bludgeoning},
		{"faster defender, generic", 75, 50, items.Generic},
	}

	for _, tt := range tests {
		got := parryChance(tt.defSpeed, tt.atkSpeed, tt.weaponType)
		if got < minParry || got > maxParry {
			t.Errorf("%s: parryChance = %d, want in range [%d, %d]", tt.name, got, minParry, maxParry)
		}
	}
}

// TestParryWeaponModifiers verifies weapon-specific parry modifiers.
func TestParryWeaponModifiers(t *testing.T) {
	baseChance := 20

	tests := []struct {
		name         string
		weaponType   items.ItemSubType
		wantModifier int
	}{
		{"slashing gets +5", items.Slashing, 5},
		{"stabbing gets +5", items.Stabbing, 5},
		{"whipping gets +5", items.Whipping, 5},
		{"bludgeoning gets 0", items.Bludgeoning, 0},
		{"cleaving gets 0", items.Cleaving, 0},
		{"generic gets -5", items.Generic, -5},
	}

	for _, tt := range tests {
		got := applyParryWeaponModifier(baseChance, tt.weaponType)
		want := baseChance + tt.wantModifier
		if got != want {
			t.Errorf("%s: applyParryWeaponModifier = %d, want %d", tt.name, got, want)
		}
	}
}

// TestBashChanceRange verifies bash chance is within the configured range.
func TestBashChanceRange(t *testing.T) {
	cfg := configs.GetCombatConfig()
	minBash := int(cfg.BashChanceMin)
	maxBash := int(cfg.BashChanceMax)

	tests := []struct {
		name   string
		defStr int
		atkStr int
	}{
		{"equal stats", 50, 50},
		{"defender stronger", 75, 50},
		{"attacker stronger", 25, 50},
		{"extreme difference", 100, 0},
	}

	for _, tt := range tests {
		got := BashChance(tt.defStr, tt.atkStr)
		if got < minBash || got > maxBash {
			t.Errorf("%s: BashChance = %d, want in range [%d, %d]", tt.name, got, minBash, maxBash)
		}
	}
}
