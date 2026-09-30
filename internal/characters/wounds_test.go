package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/wounds"
	"gopkg.in/yaml.v2"
)

func woundedChar(health, maxHealth int, ws ...wounds.Wound) *Character {
	c := New()
	c.HealthMax.Value = maxHealth
	c.Health = health
	c.Wounds = ws
	return c
}

func TestHealthLimit(t *testing.T) {
	c := woundedChar(5, 16, wounds.Wound{Kind: wounds.Cut, Points: 3})
	if c.HealthLimit() != 13 || !c.Wounded() {
		t.Fatalf("limit %d wounded %v", c.HealthLimit(), c.Wounded())
	}
	c.Wounds = nil
	if c.HealthLimit() != 16 || c.Wounded() {
		t.Fatal("unwounded limit is max health")
	}
}

func TestHealStopsAtWoundLimit(t *testing.T) {
	c := woundedChar(10, 16, wounds.Wound{Kind: wounds.Cut, Points: 3})
	if hp, _ := c.Heal(10, 0); hp != 3 || c.Health != 13 {
		t.Fatalf("Heal restored %d to %d, want 3 to 13", hp, c.Health)
	}
	c.Health = 10
	if got := c.ApplyHealthChange(10); got != 3 || c.Health != 13 {
		t.Fatalf("ApplyHealthChange restored %d to %d, want 3 to 13", got, c.Health)
	}
	// Damage is untouched by the limit.
	if got := c.ApplyHealthChange(-4); got != -4 || c.Health != 9 {
		t.Fatalf("damage %d to %d", got, c.Health)
	}
}

func TestHealthAboveLimitIsKept(t *testing.T) {
	c := woundedChar(15, 16, wounds.Wound{Kind: wounds.Cut, Points: 3})
	if hp, _ := c.Heal(5, 0); hp != 0 || c.Health != 15 {
		t.Fatalf("Heal changed health above the limit: %d to %d", hp, c.Health)
	}
	if got := c.ApplyHealthChange(5); got != 0 || c.Health != 15 {
		t.Fatalf("ApplyHealthChange changed health above the limit: %d to %d", got, c.Health)
	}
	c.AddWound(wounds.Wound{Kind: wounds.Cut, Points: 5})
	if c.Health != 15 {
		t.Fatal("a new wound must not take health away")
	}
}

func TestLevelUpRefillStopsAtLimit(t *testing.T) {
	c := New()
	c.Level = 1
	c.Validate()
	c.Experience = c.XPTNL()
	c.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Points: 4}}
	c.Health = 1
	if ok, _ := c.LevelUp(); !ok {
		t.Fatal("level up expected")
	}
	if c.Health != c.HealthLimit() || c.Health >= c.HealthMax.Value {
		t.Fatalf("health %d limit %d max %d", c.Health, c.HealthLimit(), c.HealthMax.Value)
	}
}

func TestWoundsRoundTripYAML(t *testing.T) {
	c := woundedChar(5, 16, wounds.Wound{Kind: wounds.Fracture, Place: "arm", Points: 3})
	data, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back Character
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Wounds) != 1 || back.Wounds[0] != c.Wounds[0] {
		t.Fatalf("wounds after round trip: %+v", back.Wounds)
	}
}
