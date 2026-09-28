package characters

import (
	"slices"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Phase 32a: a company member's name carries no ♥friend tag; any other
// charmed mob keeps it.
func TestCompanionCharmHidesCharmedAdjective(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	companion := New()
	companion.Name = "Tamsin Reed"
	companion.Health = 10
	companion.CharmAsCompanion(7, CharmPermanent, CharmExpiredRevert)
	if !companion.IsCharmed(7) || !companion.IsCompanion() {
		t.Fatalf("companion charm: charmed=%v companion=%v", companion.IsCharmed(7), companion.IsCompanion())
	}
	if adj := companion.GetMobName(7).Adjectives; slices.Contains(adj, `charmed`) {
		t.Fatalf("companion adjectives = %v, want no charmed", adj)
	}

	pet := New()
	pet.Name = "a stray dog"
	pet.Health = 10
	pet.Charm(7, CharmPermanent, CharmExpiredRevert)
	if pet.IsCompanion() {
		t.Fatal("an ordinary charm must not be a companion")
	}
	if adj := pet.GetMobName(7).Adjectives; !slices.Contains(adj, `charmed`) {
		t.Fatalf("charmed mob adjectives = %v, want charmed", adj)
	}

	// Befriended away by someone else: an ordinary charm, tag back.
	companion.Charm(9, CharmPermanent, CharmExpiredRevert)
	if companion.IsCompanion() || !slices.Contains(companion.GetMobName(9).Adjectives, `charmed`) {
		t.Fatal("a companion charmed by another must show ♥friend again")
	}

	companion.CharmAsCompanion(7, CharmPermanent, CharmExpiredRevert)
	companion.RemoveCharm()
	if companion.IsCompanion() || slices.Contains(companion.GetMobName(7).Adjectives, `charmed`) {
		t.Fatal("RemoveCharm must clear the companion charm and the tag")
	}
}
