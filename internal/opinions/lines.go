package opinions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/banter"
)

// lines are what each personality says to a kind it has a verdict on. A
// line is one short sentence in the companion's own voice.
var lines = map[string]map[Kind]string{
	"stoic": {
		Rough:     "A cold camp keeps a body honest.",
		Prudence:  "Careful. That is how we live.",
		Share:     "Fair. A band that eats together lasts.",
		Reverence: "The dead are owed that much.",
		Inn:       "Beds are for the soft. We lose our edge.",
		Greed:     "Coin is weight. It slows a march.",
		Cunning:   "A lie is a debt. It comes due.",
	},
	"cheerful": {
		Spare:    "Ha! Good. Let the poor fool run.",
		Inn:      "A real bed! Now we're talking.",
		Share:    "That's the way. Nobody goes hungry on my watch.",
		Kindness: "That was kind. I like us better for it.",
		Courage:  "Bold! I'd follow you off a cliff, mostly.",
		Execute:  "Did we have to? Sigh. Well.",
		Greed:    "Is that all we are, a bag and a purse?",
		Rough:    "Mud again. Lovely. Just lovely.",
	},
	"grim": {
		Execute:   "Dead things don't come back for revenge.",
		Rough:     "Dirt and a wall of dark. It's honest.",
		Prudence:  "Wise. The ones who rush are buried first.",
		SellRelic: "Sell it. Sentiment never stopped a blade.",
		Spare:     "You'll meet that one again. Mark me.",
		Inn:       "Walls and a roof make us soft, and then dead.",
		Kindness:  "Kindness is how the weak get killed.",
		Courage:   "Courage is what's left when you've no sense.",
	},
	"boastful": {
		Courage:   "That's the stuff of songs. Mine, mostly.",
		Inn:       "Finally, quarters fit for someone of my record.",
		Greed:     "Spoils to the victors. And I am a victor.",
		SellRelic: "Gold! Let the trinket rot in a shop.",
		Execute:   "A clean end. They'll tell it right at the fire.",
		Prudence:  "Run? Me? I broke no oath to hide.",
		Rough:     "Sleeping in dirt? I have standards.",
		Reverence: "Ghosts and oaths. I make my own luck.",
	},
	"wry": {
		Cunning:   "Oh, nicely done. I'd have lied worse.",
		Greed:     "Finally, a sensible motive.",
		Inn:       "A bed. How decadent of us.",
		SellRelic: "Practical. The relic keeps nothing warm.",
		Courage:   "Heroics. Lovely. Someone write it on a stone.",
		Reverence: "The dead have had their say. We haven't eaten.",
		Kindness:  "How terribly generous. And how expensive.",
		Rough:     "Ah, the great outdoors. My favourite place to freeze.",
	},
	"devout": {
		Spare:     "Mercy is a prayer said with the hands.",
		Kindness:  "You carry a light. Keep it close.",
		Reverence: "The old ones will remember this.",
		Share:     "Bread shared is a blessing kept.",
		Execute:   "A life ended in cold blood. I will pray for it.",
		SellRelic: "Those were someone's hallows. It sits ill with me.",
		Greed:     "Gold is a poor god. It will own you.",
		Cunning:   "A lie spoils the vessel it was poured into.",
	},
}

// LineFor is what a personality of a given name says about a kind, or ""
// when it has no verdict on it, set down as prose by n so a run of
// reactions varies its shape and never repeats a voice's words.
func LineFor(n *banter.Narrator, name, personality string, k Kind) string {
	text, ok := lines[strings.ToLower(personality)][k]
	if !ok || Verdict(personality, k) == Neutral {
		return ""
	}
	return n.Say(name, strings.ToLower(personality), string(k), text)
}
