package util

import "testing"

func TestArticle(t *testing.T) {
	cases := map[string]string{
		"bandit captain":                    "the bandit captain",
		"Garrick Vane":                      "Garrick Vane",
		`<ansi fg="mobname">rat</ansi>`:     `the <ansi fg="mobname">rat</ansi>`,
		`<ansi fg="mobname">Ysolde</ansi>`:  `<ansi fg="mobname">Ysolde</ansi>`,
		"":                                  "",
		"the rat":                           "the rat",
		`<ansi fg="mobname">the rat</ansi>`: `<ansi fg="mobname">the rat</ansi>`,
		"theodric's hound":                  "the theodric's hound",
	}
	for in, want := range cases {
		if got := Article(in); got != want {
			t.Errorf("Article(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCapitalizeFirst(t *testing.T) {
	cases := map[string]string{
		"the rat bites.":                       "The rat bites.",
		`<ansi fg="x">the</ansi> rat`:          `<ansi fg="x">The</ansi> rat`,
		"":                                     "",
		"<ansi>":                               "<ansi>",
		"You swing.":                           "You swing.",
		`<ansi fg="mobname">élan</ansi> falls`: `<ansi fg="mobname">Élan</ansi> falls`,
	}
	for in, want := range cases {
		if got := CapitalizeFirst(in); got != want {
			t.Errorf("CapitalizeFirst(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNarrationVoiceProblem(t *testing.T) {
	cases := map[string]string{
		`The <ansi fg="mobname">rat</ansi> bites you. (5 damage)`: "",
		`<ansi fg="ALERT">A</ansi> quiet line.`:                   "", // markup is not text
		`You hit it!`:                                             "exclamation mark",
		`A CRITICAL blow.`:                                        "ALL-CAPS word CRITICAL",
	}
	for line, want := range cases {
		if got := NarrationVoiceProblem(line); got != want {
			t.Errorf("NarrationVoiceProblem(%q) = %q, want %q", line, got, want)
		}
	}
}
