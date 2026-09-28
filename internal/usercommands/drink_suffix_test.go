package usercommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
)

// Phase 32a: drinking for yourself always ends with where your thirst
// stands; a band crossed, or a companion watered, reads as before.
func TestDrinkSuffix(t *testing.T) {
	full := survival.FullNeeds()
	same := survival.ProvisionResult{Name: "Aria", Needs: full}
	if got, want := drinkSuffix(same, false), " Thirst: "+survival.ThirstLabel(full.Thirst)+"."; got != want {
		t.Fatalf("no band crossed = %q, want %q", got, want)
	}
	crossed := same
	crossed.Thirst = survival.Change{Before: survival.BandLow, After: survival.BandFull}
	if got := drinkSuffix(crossed, false); got != provisionSuffix(crossed, false) || got == "" {
		t.Fatalf("band crossed = %q", got)
	}
	if got := drinkSuffix(same, true); got != provisionSuffix(same, true) {
		t.Fatalf("companion watered = %q", got)
	}
}

// Phase 32a: the shipped Hydrated buff (the waterskin's) still cancels
// Thirsty but says nothing of its own.
func TestHydratedBuffHasNoFlourish(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "buffs", "34-hydrated.js"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	if strings.Contains(script, "SendUserMessage") || strings.Contains(script, "Nectar") {
		t.Fatal("the Hydrated buff must not message the player")
	}
	if !strings.Contains(script, `CancelBuffWithFlag("thirsty")`) {
		t.Fatal("the Hydrated buff must still cancel Thirsty")
	}
}
