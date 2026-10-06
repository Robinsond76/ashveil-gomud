package camping

import "strings"

// Phase 43a camp supplies (help campsupplies): finished supplies a company
// prepares at its camp. Each is an ordinary item the company carries; the
// camp module spends one when its benefit starts.
const (
	BrothItemID   = 30040 // fortifying broth: +5% health limit, queued for the rest
	WarmingItemID = 30041 // warming draught: a quarter less cold taken on
	CoolingItemID = 30042 // cooling salve: a quarter less heat taken on
	IncenseItemID = 30043 // watch incense: +10 points to a watch's spotting
	WarmingBuffID = 74
	CoolingBuffID = 75
	SupplyMinutes = 15 // how long a personal supply lasts, in real minutes
	IncenseBonus  = 10 // percentage points added to a watch's chance to spot raiders
	IncenseCapPct = 90 // incense cannot raise the chance past this
	BrothPctOfMax = 5  // the health limit a broth adds, as a percent of the maximum (rounded up)
)

// BrothBuffs are the four sizes of the Fortified buff, smallest first; a
// member gets the one that is closest to 5% of their maximum health.
var BrothBuffs = []BrothTier{
	{BuffID: 70, Bonus: 2, UpTo: 40},
	{BuffID: 71, Bonus: 5, UpTo: 100},
	{BuffID: 72, Bonus: 10, UpTo: 200},
	{BuffID: 73, Bonus: 20, UpTo: 0},
}

// BrothTier is one size of the Fortified buff: members whose maximum health
// is at most UpTo (0: anyone larger) get it.
type BrothTier struct {
	BuffID int
	Bonus  int
	UpTo   int
}

// BrothTierFor is the size of broth buff for a member with this maximum
// health.
func BrothTierFor(healthMax int) BrothTier {
	for _, t := range BrothBuffs {
		if t.UpTo == 0 || healthMax <= t.UpTo {
			return t
		}
	}
	return BrothBuffs[len(BrothBuffs)-1]
}

// IsBrothBuff reports whether buffID is one of the Fortified sizes.
func IsBrothBuff(buffID int) bool {
	for _, t := range BrothBuffs {
		if t.BuffID == buffID {
			return true
		}
	}
	return false
}

// Supply is one kind of camp supply.
type Supply struct {
	Key      string
	Name     string
	ItemID   int
	Personal bool // one personal benefit per member (broth, warming, cooling)
	Aliases  []string
}

// Supplies is every camp supply, in the order they are listed.
var Supplies = []Supply{
	{Key: "broth", Name: "fortifying broth", ItemID: BrothItemID, Personal: true, Aliases: []string{"fortifying", "fortify"}},
	{Key: "warming", Name: "warming draught", ItemID: WarmingItemID, Personal: true, Aliases: []string{"warm", "draught"}},
	{Key: "cooling", Name: "cooling salve", ItemID: CoolingItemID, Personal: true, Aliases: []string{"cool", "salve"}},
	{Key: "incense", Name: "watch incense", ItemID: IncenseItemID, Aliases: []string{"watch"}},
}

// FindSupply matches a word to a supply.
func FindSupply(word string) (Supply, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	for _, s := range Supplies {
		if word == s.Key || word == strings.ToLower(s.Name) {
			return s, true
		}
		for _, a := range s.Aliases {
			if word == a {
				return s, true
			}
		}
	}
	return Supply{}, false
}

// Prepared is what a camp has queued for its next rest (Phase 43a). Broth
// holds the member keys it will fortify; Incense that a bundle will burn.
// The supplies themselves are spent when the rest starts, not when queued.
type Prepared struct {
	Broth   []string `yaml:"broth,omitempty"`
	Incense bool     `yaml:"incense,omitempty"`
}

// Empty reports whether nothing is queued.
func (p *Prepared) Empty() bool {
	return p == nil || (len(p.Broth) == 0 && !p.Incense)
}

// HasBroth reports whether a member's broth is queued.
func (p *Prepared) HasBroth(member string) bool {
	if p == nil {
		return false
	}
	for _, key := range p.Broth {
		if key == member {
			return true
		}
	}
	return false
}

// WithBroth is a copy with a member's broth queued.
func (p *Prepared) WithBroth(member string) *Prepared {
	out := Prepared{}
	if p != nil {
		out = Prepared{Broth: append([]string(nil), p.Broth...), Incense: p.Incense}
	}
	out.Broth = append(out.Broth, member)
	return &out
}

// WithIncense is a copy with incense queued.
func (p *Prepared) WithIncense() *Prepared {
	out := Prepared{Incense: true}
	if p != nil {
		out.Broth = append([]string(nil), p.Broth...)
	}
	return &out
}

// Cleared is a copy with a member's broth (all of it, for ""), or with
// everything for "all", removed; nil when nothing is left.
func (p *Prepared) Cleared(member string) *Prepared {
	if p == nil {
		return nil
	}
	if member == "all" {
		return nil
	}
	out := Prepared{Incense: p.Incense}
	for _, key := range p.Broth {
		if key != member {
			out.Broth = append(out.Broth, key)
		}
	}
	if out.Empty() {
		return nil
	}
	return &out
}

// IncenseChance is a watch's chance to spot raiders (a percent) with
// incense burning: ten points more, capped at 90, and never lower than it
// already was, so a watch that already spots them for certain keeps it.
func IncenseChance(chance int) int {
	return max(chance, min(IncenseCapPct, chance+IncenseBonus))
}
