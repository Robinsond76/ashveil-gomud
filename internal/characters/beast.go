package characters

// Phase 39e: a Beast Tamer's bonded beast. Like a doll it is a durable record
// kept on its Tamer (a player's saved character, a companion's company
// state) and a live mob only for a battle (internal/beasts). Unlike a doll it
// is alive: it takes its own turn, and a fall leaves it wounded instead of
// destroyed.

// BeastState is the bonded beast's durable record.
type BeastState struct {
	Name string `yaml:"name,omitempty"`
	// Damage is the health the beast has lost; rest takes it back.
	Damage int `yaml:"damage,omitempty"`
	// Wounded is a beast that fell in battle: it stays out of fights until
	// the company rests.
	Wounded bool `yaml:"wounded,omitempty"`
}

// BeastInfo is what a live beast is: its Tamer, its kind and the gifts the
// Tamer's route gave it when it stood. It lives for one battle.
type BeastInfo struct {
	Kind                string // "wolf", "warhound", "bear" or "drake"
	OwnerUser, OwnerMob int    // the Tamer's user id, or its mob instance
	OwnerKey            string // the Tamer's company member key
	HPPct               int    // percent of a warrior's health at its level
	Dice, Sides         int    // its natural weapon's dice
	TempoPct            int    // percent of a normal tempo
	Damage              int    // damage added to each of its blows
	Hobble              bool   // its bites hobble a wounded foe
	Guards, GuardsUsed  int    // Guard uses a battle and spent (the bear)
	BreathEvery         int    // rounds between Breaths (the drake), 0 for none
	BreathNext          uint64 // the combat round of its next Breath
	Rallied             int    // Rally heals the Tamer has given it this battle
}

// NewBeastState is a fresh beast: named, whole.
func NewBeastState(name string) BeastState { return BeastState{Name: name} }

// EnsureBeast makes the character's beast record if it has none, and
// returns it.
func (c *Character) EnsureBeast(name string) *BeastState {
	if c.Beast == nil {
		b := NewBeastState(name)
		c.Beast = &b
	}
	return c.Beast
}

// HasBeast reports whether the character has a bonded beast.
func (c *Character) HasBeast() bool { return c.Beast != nil }

// CloneBeast copies a beast record.
func CloneBeast(in *BeastState) *BeastState {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
