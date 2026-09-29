package races

import "testing"

func TestPainReactionPairsRequireBothViewpointsAndKnownTokens(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pair  PainReaction
		valid bool
	}{
		{"paired", PainReaction{ToVictim: "You stagger.", ToRoom: "{name} staggers against {his} shield."}, true},
		{"missing victim", PainReaction{ToRoom: "{name} staggers."}, false},
		{"missing room", PainReaction{ToVictim: "You stagger."}, false},
		{"unknown token", PainReaction{ToVictim: "You stagger.", ToRoom: "{who} staggers."}, false},
		{"unclosed token", PainReaction{ToVictim: "You stagger.", ToRoom: "{name staggers."}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePainReactions([]PainReaction{tc.pair})
			if (err == nil) != tc.valid {
				t.Fatalf("ValidatePainReactions(%+v) error = %v, valid %v", tc.pair, err, tc.valid)
			}
		})
	}
}
