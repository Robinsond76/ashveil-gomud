package items

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

//
// Item is used for item instances
// Flat specs are found by loading the spec of the item id.
// Anything in this struct is mutable.
//

var (
	ItemDisabledSlot = Item{ItemId: -1}

	// -short suffix should also be defined in case shorthand symbols are preferred
	adjectiveSwaps = map[string]string{
		// Is the item exploding?
		`exploding`:       `<ansi fg="red">!!!Exploding!!!</ansi>`,
		`exploding-short`: `<ansi fg="red">!!!/ansi>`,
	}
)

const (
	// TODO: Centralize these types somewhere, eventually?
	UUIDItem = uuid.IDType(0b00000001)
)

// Instance properties that may change
type Item struct {
	ItemId            int       `yaml:"itemid,omitempty"`
	UUID              uuid.UUID `yaml:"uuid,omitempty"`
	Blob              string    `yaml:"blob,omitempty"`          // Does this item have a blob? Should be base64 encoded.
	Uses              int       `yaml:"uses,omitempty"`          // How many uses it has left
	LastUsedRound     uint64    `yaml:"lastusedround,omitempty"` // Last round this item was used
	Spec              *ItemSpec `yaml:"overrides,omitempty"`
	Uncursed          bool      `yaml:"uncursed,omitempty"`          // Is this item uncursed?
	Enchantments      uint8     `yaml:"enchantments,omitempty"`      // Is this item enchanted?
	Adjectives        []string  `yaml:"adjectives,omitempty"`        // Decorative text for the name of the item (e.g. "exploding")
	StashedBy         int       `yaml:"stashedby,omitempty"`         // userid of whoever stashed this item
	CanNeverBeRemoved bool      `yaml:"canneverberemoved,omitempty"` // If true, this item can never be unequipped once worn
	// Phase 23b: a whetstone's edge. SharpBonus is added to each successful
	// strike until SharpStrikes run out. Plain values, never a pointer:
	// items are copied by value, and a shared pointer would alias one edge
	// across copies.
	SharpBonus   int `yaml:"sharpbonus,omitempty"`
	SharpStrikes int `yaml:"sharpstrikes,omitempty"`
	// Phase 43b: a poison coating on a blade. CoatKind is the poison's id,
	// CoatExpires an absolute Unix time (real time passes offline), and
	// CoatContacts the damaging contacts left. Plain values for the same
	// reason as the edge.
	CoatKind     string `yaml:"coatkind,omitempty"`
	CoatExpires  int64  `yaml:"coatexpires,omitempty"`
	CoatContacts int    `yaml:"coatcontacts,omitempty"`
	// Phase 36a: a generated item's roll. A value, replaced whole by
	// Identify, never edited through a shared slice.
	Loot Rolled `yaml:"loot,omitempty"`
	// Phase 36c: the player marked this item as junk for `sell junk`. A
	// plain value, so it follows the item wherever it goes.
	Junk bool `yaml:"junk,omitempty"`
	// Phase 67: a relic's awakening progress, one count per awakening in
	// its spec. Replaced whole on each change, never edited in place.
	Awaken []int `yaml:"awaken,omitempty"`
	// Phase 71: the trophy (an item id) an enchanter worked into this item.
	// It adds the trophy's gear effects while the item is worn and changes
	// nothing else: not the spec, the value or the roll.
	Trophy        int            `yaml:"trophy,omitempty"`
	tempDataStore map[string]any // Temporary data store for this item. Not saved to disk.
}

func New(itemId int) Item {
	itemSpec := GetItemSpec(itemId)

	newItm := Item{}
	if itemSpec != nil {
		newItm.UUID = uuid.New(UUIDItem)
		newItm.ItemId = itemId
		if itemSpec.Uses > 0 {
			newItm.Uses = itemSpec.Uses
		}
	}

	newItm.Validate()

	return newItm
}

func (i *Item) GetScript() string {
	return i.GetSpec().GetScript()
}

func (i *Item) HasAdjective(adj string) bool {
	if i.Adjectives == nil {
		return false
	}

	for _, a := range i.Adjectives {
		if a == adj {
			return true
		}
	}

	return false
}

func (i *Item) SetAdjective(adj string, addToList bool) {
	if i.Adjectives == nil {
		i.Adjectives = []string{}
	}
	for idx, a := range i.Adjectives {
		if a == adj {
			if addToList {
				return
			} else {
				i.Adjectives = append(i.Adjectives[:idx], i.Adjectives[idx+1:]...)
				return
			}
		}
	}
	if addToList {
		i.Adjectives = append(i.Adjectives, adj)
	}
}

// performs a break test and returns true if the item breaks
// Pass a uint8 to increase the chance of breaking.
func (i *Item) BreakTest(increaseChance ...int) bool {
	bc := i.GetSpec().BreakChance
	if bc < 1 {
		return false
	}
	randNum := uint8(util.Rand(100))
	if len(increaseChance) > 0 {
		if uint8(increaseChance[0]) >= randNum {
			randNum = 0
		} else {
			randNum -= uint8(increaseChance[0])
		}
	}
	return bc > randNum
}

func (i *Item) SetTempData(key string, value any) {

	if i.tempDataStore == nil {
		i.tempDataStore = make(map[string]any)
	}

	if value == nil {
		delete(i.tempDataStore, key)
		return
	}
	i.tempDataStore[key] = value
}

func (i *Item) GetTempData(key string) any {

	if i.tempDataStore == nil {
		i.tempDataStore = make(map[string]any)
	}

	if value, ok := i.tempDataStore[key]; ok {
		return value
	}
	return nil
}

func (i Item) IsDisabled() bool {
	return i.ItemId < 0
}

// Refilled is the item after it is filled back to uses (Phase 43a): an
// empty container becomes its full item, anything else keeps its identity
// with the new uses.
func (i Item) Refilled(uses int) Item {
	if filled := i.GetSpec().FilledItemId; filled > 0 && GetItemSpec(filled) != nil {
		return New(filled)
	}
	i.Uses = uses
	return i
}

func (i *Item) Validate() {
	if i.ItemId < 1 {
		return
	}

	// Make sure has a uid
	if i.UUID.IsNil() {
		i.UUID = uuid.New(UUIDItem)
	}

	iSpec := i.GetSpec()
	if iSpec.ItemId > 0 {
		if i.Uses == 0 && iSpec.Uses > 0 {
			i.Uses = iSpec.Uses
		}
	}

}

func (i *Item) GetLongDescription() string {
	return i.GetLongDescriptionFor(0)
}

// GetLongDescriptionFor is the long description as seen by a viewer with
// the given Scribe rank (Phase 36a): rank 4 sees each affix's tier and range
// and where a rolled item came from.
func (i *Item) GetLongDescriptionFor(scribeRank int) string {

	iSpec := i.GetSpec()

	longDesc := strings.Builder{}

	longDesc.WriteString(iSpec.Description)

	if iSpec.Type == Readable {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - You should probably <ansi fg="command">read</ansi> this.`)

	} else if iSpec.Subtype == Drinkable {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - You could probably <ansi fg="command">drink</ansi> this.`)

	} else if iSpec.Subtype == Edible {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - You could probably <ansi fg="command">eat</ansi> this.`)

	} else if iSpec.Type == Lockpicks {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - These are used with the <ansi fg="command">picklock</ansi> command.`)

	} else if iSpec.Type == Key {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - When you find the right door, keys are added to your <ansi fg="command">keyring</ansi> automatically.`)

	} else if iSpec.Subtype == Wearable {

		longDesc.WriteString("\n")
		longDesc.WriteString(fmt.Sprintf(`- It looks like wearable %s equipment.`, iSpec.Type))

	} else if iSpec.Type == Weapon {

		longDesc.WriteString("\n")
		longDesc.WriteString(fmt.Sprintf(`- It looks like a %d-Handed weapon.`, iSpec.Hands))

		if iSpec.Subtype == Claws {

			longDesc.WriteString("\n")
			longDesc.WriteString(`- It looks like a claws weapon. These can be dual wielded without training.`)

		} else if iSpec.Subtype == Shooting {

			longDesc.WriteString("\n")
			longDesc.WriteString(`- This can fired into adjacent areas. (<ansi fg="command">help shoot</ansi>)`)

		}

		if iSpec.WaitRounds > 0 {

			longDesc.WriteString("\n")
			longDesc.WriteString(fmt.Sprintf(`- It requires an extra %d round(s) between attacks.`, iSpec.WaitRounds))

		}

		if i.Sharpened() {

			longDesc.WriteString("\n")
			longDesc.WriteString(fmt.Sprintf(`- Its edge is honed: +%d damage for its next %d strikes.`, i.SharpBonus, i.SharpStrikes))

		}

	} else if iSpec.Subtype == Usable {

		longDesc.WriteString("\n")
		longDesc.WriteString(` - You could probably <ansi fg="command">use</ansi> this.`)

	}

	if iSpec.Relic != nil {
		longDesc.WriteString("\n")
		longDesc.WriteString(i.RelicDescription())
	} else if lines := i.TrophyLines(); len(lines) > 0 {
		longDesc.WriteString("\n")
		longDesc.WriteString(strings.Join(lines, "\n"))
	}
	if i.IsRolled() {
		longDesc.WriteString("\n")
		longDesc.WriteString(i.RolledDescription(scribeRank))
	} else if iSpec.Tier > 0 {
		longDesc.WriteString("\n")
		longDesc.WriteString(iSpec.TierDescription())
	}

	if iSpec.IsGoods() {
		longDesc.WriteString("\n")
		longDesc.WriteString(iSpec.GoodsDescription())
	}

	return longDesc.String()
}

// Sharpened reports whether the item has an edge left.
func (i *Item) Sharpened() bool {
	return i.SharpStrikes > 0 && i.SharpBonus > 0
}

// Sharpen gives the item an edge of bonus damage for strikes successful
// strikes. An existing edge is never stacked or reset. It reports whether
// the item was sharpened.
func (i *Item) Sharpen(bonus, strikes int) bool {
	if i.Sharpened() || bonus <= 0 || strikes <= 0 {
		return false
	}
	i.SharpBonus = bonus
	i.SharpStrikes = strikes
	return true
}

// SpendEdge spends n strikes of the edge; the edge ends at zero.
func (i *Item) SpendEdge(n int) {
	if n <= 0 {
		return
	}
	i.SharpStrikes -= n
	if i.SharpStrikes <= 0 {
		i.SharpStrikes = 0
		i.SharpBonus = 0
	}
}

// EdgeLabel is the weapon-condition suffix shown in inventories, or "":
// the edge (Phase 23b) and a live poison coating (Phase 43b).
func (i *Item) EdgeLabel() string {
	var parts []string
	if i.Sharpened() {
		parts = append(parts, fmt.Sprintf(`<ansi fg="item-bonus-damage">(sharp: %d)</ansi>`, i.SharpStrikes))
	}
	if coat := i.CoatLabel(time.Now()); coat != `` {
		parts = append(parts, coat)
	}
	// Phase 71: a trophy enchant is told apart from a plain copy in lists.
	if spec := i.TrophySpecOf(); spec != nil {
		parts = append(parts, fmt.Sprintf(`<ansi fg="magenta">(enchanted: %s)</ansi>`, spec.Name))
	}
	return strings.Join(parts, ` `)
}

// Coated reports whether the item carries a live poison coating at now: it
// names a poison, has contacts left, and has not run out of time. Expiry is
// checked at every read, so an offline weapon never delivers stale poison.
func (i *Item) Coated(now time.Time) bool {
	return i.CoatKind != `` && i.CoatContacts > 0 && now.Unix() < i.CoatExpires
}

// Coat puts a poison coating on the item. A live coating is never replaced
// or refreshed (clear it first); it reports whether the item was coated.
func (i *Item) Coat(kind string, expires time.Time, contacts int, now time.Time) bool {
	if i.Coated(now) || kind == `` || contacts <= 0 || !expires.After(now) {
		return false
	}
	i.CoatKind = kind
	i.CoatExpires = expires.Unix()
	i.CoatContacts = contacts
	return true
}

// ClearCoat removes any coating (live or lapsed) and reports whether the
// item held one.
func (i *Item) ClearCoat() bool {
	had := i.CoatKind != ``
	i.CoatKind, i.CoatExpires, i.CoatContacts = ``, 0, 0
	return had
}

// SpendCoat spends n contacts of the coating; it ends at zero.
func (i *Item) SpendCoat(n int) {
	if n <= 0 || i.CoatKind == `` {
		return
	}
	i.CoatContacts -= n
	if i.CoatContacts <= 0 {
		i.ClearCoat()
	}
}

// CoatSummary is the coating without markup ("Bitterleaf, 9 min, 8 hits"),
// or "" without a live one.
func (i *Item) CoatSummary(now time.Time) string {
	if !i.Coated(now) {
		return ``
	}
	name := i.CoatKind
	if p, ok := PoisonByID(i.CoatKind); ok {
		name = p.Name
	}
	mins := int((time.Unix(i.CoatExpires, 0).Sub(now) + time.Minute - 1) / time.Minute)
	return fmt.Sprintf(`%s, %d min, %d hits`, name, mins, i.CoatContacts)
}

// CoatLabel is the coating suffix shown in inventories, or "" without a
// live one: the poison, the minutes left and the contacts left.
func (i *Item) CoatLabel(now time.Time) string {
	if !i.Coated(now) {
		return ``
	}
	name := i.CoatKind
	if p, ok := PoisonByID(i.CoatKind); ok {
		name = p.Name
	}
	left := time.Unix(i.CoatExpires, 0).Sub(now)
	mins := int((left + time.Minute - 1) / time.Minute)
	return fmt.Sprintf(`<ansi fg="item-bonus-damage">(%s: %d min, %d hits)</ansi>`, name, mins, i.CoatContacts)
}

func (i *Item) IsBetterThan(otherItm Item) bool {

	if otherItm.ItemId < 1 {
		return i.ItemId > 0 // As long as the other item isn't also zero, it's better.
	}
	// Whichever is higher value is better
	return i.GetSpec().Value > otherItm.GetSpec().Value
}

func (i *Item) GetSpec() ItemSpec {
	if i.Spec != nil {
		spec := *i.Spec
		// Old enchanted pack instances retain their identity and custom fields,
		// while their shipped container type follows the current base data.
		if base := GetItemSpec(i.ItemId); base != nil && base.Type == Pack {
			spec.Type, spec.Subtype = Pack, Wearable
		}
		return spec
	}
	iSpec := GetItemSpec(i.ItemId)
	if iSpec == nil {
		iSpec = &ItemSpec{}
	}
	return *iSpec
}

func (i *Item) AddWornBuff(buffId int) {
	if i.Spec == nil {
		specCopy := *GetItemSpec(i.ItemId)
		i.Spec = &specCopy
	}

	i.Spec.WornBuffIds = append(i.Spec.WornBuffIds, buffId)
}

func (i *Item) Rename(newName string, displayNameOrStyle ...string) {
	if i.Spec == nil {
		specCopy := *GetItemSpec(i.ItemId)
		i.Spec = &specCopy
	}

	i.Spec.Name = newName

	if len(displayNameOrStyle) > 0 {
		// Just in case color short tags are being used...
		i.Spec.DisplayName = util.ConvertColorShortTags(displayNameOrStyle[0])

	} else {
		i.Spec.DisplayName = ``
	}
}

func (i *Item) Redescribe(newDescription string) {
	if i.Spec == nil {
		specCopy := *GetItemSpec(i.ItemId)
		i.Spec = &specCopy
	}

	i.Spec.Description = newDescription
}

func (i *Item) IsEnchanted() bool {
	return i.Enchantments > 0
}

func (i *Item) UnEnchant() {
	if i.IsEnchanted() {
		i.Spec = nil
		i.Enchantments = 0
		// Phase 36a: a rolled item's numbers live in its Spec override.
		i.rebuildRolledSpec()
	}
}

// enchantmentLevel is 0-100. If 0(zero) remove any enchantments.
func (i *Item) Enchant(damageBonus int, defenseBonus int, statBonus map[string]int, cursed bool) {

	var newSpec ItemSpec

	if i.Spec == nil {
		specCopy := *GetItemSpec(i.ItemId)
		newSpec = specCopy
	} else {
		newSpec = *i.Spec
	}

	// Phase 36a: a rolled item's value carries its quality, which the auto
	// value doesn't know, so only the enchantment's own worth is added.
	priorValue, priorAuto := newSpec.Value, 0
	if i.IsRolled() {
		before := newSpec
		before.AutoCalculateValue()
		priorAuto = before.Value
		newSpec.StatMods = cloneStatMods(newSpec.StatMods)
	}

	newSpec.Damage.BonusDamage += damageBonus
	newSpec.DamageReduction += defenseBonus

	// Permanently add new statmods
	for statName, statBonusAmt := range statBonus {
		newSpec.StatMods.Add(statName, statBonusAmt)
	}

	i.Enchantments++

	newSpec.Cursed = cursed

	newSpec.Damage.FormatDiceRoll()
	if i.IsRolled() {
		// Phase 36a: keep the roll's quality-scaled value and add only what
		// the enchantment itself is worth.
		newSpec.AutoCalculateValue()
		newSpec.Value = priorValue + max(0, newSpec.Value-priorAuto)
	} else {
		newSpec.AutoCalculateValue()
	}

	i.Spec = &newSpec
}

func (i *Item) Uncurse() {
	i.Uncursed = true
}

func (i *Item) IsCursed() bool {
	return i.GetSpec().Cursed && !i.Uncursed
}

func (i *Item) IsRemoveLocked() bool {
	return i.CanNeverBeRemoved
}

// Gets the specifics of the item damage
// Considers overrides
func (i *Item) GetDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	if i.ItemId < 1 {
		return 1, 1, 3, 0, []int{} // Default Damages
	}
	dmg := i.GetDamage()
	return dmg.Attacks, dmg.DiceCount, dmg.SideCount, dmg.BonusDamage, dmg.CritBuffIds
}

func (i *Item) IsSpecial() bool {
	iSpec := i.GetSpec()
	if len(i.Blob) > 0 {
		return true
	}
	if iSpec.Uses > 0 && iSpec.Uses != i.Uses {
		return true
	}
	if i.Spec != nil || i.IsRolled() {
		return true
	}

	return false
}

func (i *Item) GetDamage() Damage {
	return i.GetSpec().Damage
}

// Returns a random number up to the total possible reduction for this item.
func (i *Item) GetDefense() int {
	itemInfo := i.GetSpec()
	return itemInfo.DamageReduction
}

func (i *Item) Equals(b Item) bool {
	return i.ItemId == b.ItemId && i.UUID == b.UUID
}

func (i *Item) IsValid() bool {

	if itemInfo := GetItemSpec(i.ItemId); itemInfo != nil {
		return true
	}
	return false
}

func (i *Item) GetBlob() string {
	if len(i.Blob) == 0 {
		return ``
	}

	decoded := util.Decode(i.Blob)
	return string(util.Decompress(decoded))
}

func (i *Item) SetBlob(blob string) {
	compressed := util.Compress([]byte(blob))
	i.Blob = util.Encode(compressed)
}

func (i *Item) AttrString() string {

	flags := []string{}

	if i.IsCursed() {
		flags = append(flags, `<ansi fg="item-cursed">c</ansi>`)
	}
	if i.IsEnchanted() {
		flags = append(flags, `<ansi fg="item-enchanted">e</ansi>`)
	}
	if i.Junk {
		flags = append(flags, `<ansi fg="item-flags">j</ansi>`)
	}

	if len(flags) == 0 {
		return ``
	}

	return fmt.Sprintf(`<ansi fg="item-flags">[%s]</ansi>`, strings.Join(flags, ``))
}

func (i *Item) DisplayName() string {
	if i.ItemId < 1 { // Used to represent item slots that are disabled
		if i.ItemId == 0 { // Used to represent item slots that are empty
			return `<ansi fg="item-nothing">-nothing-</ansi>`
		} else {
			return `<ansi fg="item-nothing">***disabled***</ansi>`
		}
	}

	prefix := ``
	if i.GetSpec().QuestToken != `` {
		prefix = `<ansi fg="questflag">★</ansi>`
	}

	suffix := ``
	if adjLen := len(i.Adjectives); adjLen > 0 {
		suffix += ` <ansi fg="black-bold">(`
		for i, adj := range i.Adjectives {
			if newAdj, ok := adjectiveSwaps[adj]; ok {
				suffix += newAdj
			} else {
				suffix += adj
			}
			if i < adjLen-1 {
				suffix += `|`
			}
		}
		suffix += `)</ansi>`
	}

	spec := i.GetSpec()
	if spec.DisplayName != `` {
		if spec.DisplayName[0:1] == `:` {
			return prefix + colorpatterns.ApplyColorPattern(spec.Name, spec.DisplayName[1:]) + suffix
		} else {
			return prefix + spec.DisplayName + suffix
		}
	}
	// Phase 36a: rolled gear shows its quality, affixes and rarity colour.
	if i.IsRolled() {
		name := i.RollName(spec.Name)
		if i.Loot.Rarity != RarityCommon && i.Loot.Rarity != `` {
			name = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, i.Loot.Rarity.Colour(), name)
		}
		if label := i.RollLabel(); label != `` {
			suffix = ` ` + label + suffix
		}
		return prefix + name + suffix
	}
	return prefix + spec.Name + suffix
}

func (i *Item) Name() string {

	if i.ItemId < 1 { // Used to represent item slots that are disabled
		if i.ItemId == 0 { // Used to represent item slots that are empty
			return `-nothing-`
		} else {
			return `***disabled***`
		}
	}

	return i.GetSpec().Name
}

func (i *Item) ShorthandId() string {
	if i.ItemId < 1 { // Used to represent item slots that are disabled
		return ``
	}

	return fmt.Sprintf(`!%d:%s`, i.ItemId, i.UUID.String())
}

func (i *Item) NameSimple() string {

	if i.ItemId < 1 { // Used to represent item slots that are disabled
		if i.ItemId == 0 { // Used to represent item slots that are empty
			return `-nothing-`
		} else {
			return `***disabled***`
		}
	}

	return i.GetSpec().NameSimple
}

func (i *Item) NameComplex() string {

	if i.ItemId < 1 { // Used to represent item slots that are disabled
		if i.ItemId == 0 { // Used to represent item slots that are empty
			return `<ansi fg="item-nothing">-nothing-</ansi>`
		} else {
			return `<ansi fg="item-nothing">***disabled***</ansi>`
		}
	}

	nm := i.DisplayName()

	if i.GetSpec().Damage.BonusDamage > 0 {
		nm = fmt.Sprintf(`%s <ansi fg="item-bonus-damage">+%d</ansi>`, nm, i.GetSpec().Damage.BonusDamage)
	}
	if edge := i.EdgeLabel(); edge != `` {
		nm = fmt.Sprintf(`%s %s`, nm, edge)
	}
	flagsStr := i.AttrString()
	if flagsStr != `` {
		nm = fmt.Sprintf(`%s %s`, flagsStr, nm)
	}
	return nm
}

func (i *Item) NameMatch(input string, allowContains bool) (partialMatch bool, fullMatch bool) {

	if i.ItemId < 1 { // Used to represent item slots that are empty
		return false, false
	}

	input = strings.ToLower(input)
	simpleName := strings.ToLower(i.Name())

	// Phase 36a: an identified rolled item answers to its generated name.
	if i.IsRolled() && i.Loot.Identified && i.Loot.Name != `` {
		rolledName := strings.ToLower(i.Loot.Name)
		if rolledName == input {
			return true, true
		}
		if strings.HasPrefix(rolledName, input) {
			return true, false
		}
	}

	if allowContains {
		for _, word := range strings.Fields(simpleName) {
			if strings.HasPrefix(word, input) {
				if simpleName == input {
					return true, true
				}
				return true, false
			}
		}
	}

	if strings.HasPrefix(simpleName, input) {
		if simpleName == input {
			return true, true
		}
		return true, false
	}

	// Phase 36a: rolled gear also answers to the words its shown name adds
	// (quality, and an identified item's affix words), which hold nothing
	// hidden.
	if i.IsRolled() {
		shown := strings.ToLower(strings.ReplaceAll(i.RollName(simpleName), ",", ""))
		if shown == input {
			return true, true
		}
		if strings.HasPrefix(shown, input) {
			return true, false
		}
		if allowContains {
			for _, word := range strings.Fields(shown) {
				if strings.HasPrefix(word, input) {
					return true, false
				}
			}
		}
	}

	return false, false
}

func (i *Item) StatMod(statName ...string) int {

	if i.ItemId < 1 {
		return 0
	}

	itemInfo := i.GetSpec()

	return itemInfo.StatMods.Get(statName...)
}

func startsWithVowel(s string) bool {
	if len(s) == 0 {
		return false
	}

	firstChar := unicode.ToLower(rune(s[0]))
	return firstChar == 'a' || firstChar == 'e' || firstChar == 'i' || firstChar == 'o' || firstChar == 'u'
}

// Provided a name and a list of items, find the first item that matches the name
// Will first provide a pair of starts-width and exact matches,
// and if not found then a contains.
func FindMatchIn(itemName string, items ...Item) (pMatch Item, fMatch Item) {

	if len(itemName) > 1 {
		if itemName[0] == '!' { // Special meaning to specify an item

			var itemIdMatch int = 0
			var itemUUIDMatch uuid.UUID = uuid.UUID{}

			parts := strings.Split(itemName[1:], `:`)
			itemIdMatch, _ = strconv.Atoi(parts[0])

			if len(parts) > 1 {
				itemUUIDMatch, _ = uuid.FromString(parts[1])
			}

			for _, itm := range items {

				// If a uid was included, it takes priority over qualifying/disqualifying
				if !itemUUIDMatch.IsNil() {
					if itm.UUID != itemUUIDMatch {
						continue
					}
					return itm, itm
				}

				if itemIdMatch > 0 {
					if itm.ItemId != itemIdMatch {
						continue
					}
					return itm, itm
				}
			}
			return Item{}, Item{}
		}
	}

	itemName, itemNumber := util.GetMatchNumber(itemName)

	var matchItem Item
	var closeMatchItem Item

	var matchItemCt int = 0
	var closeMatchItemCt int = 0

	for _, i := range items {

		part, full := i.NameMatch(itemName, false)

		if part {
			closeMatchItemCt++
			if closeMatchItemCt == itemNumber {
				closeMatchItem = i
			}
		}

		if full {
			matchItemCt++
			if matchItemCt == itemNumber {
				matchItem = i
				break
			}
		}

	}

	// If no "starts with" or "exact" matches are found, try and find the first items that contain the supplied name
	// Note: Can't have an exact match if there was never a close match
	if closeMatchItem.ItemId == 0 {
		closeMatchItemCt = 0
		for _, i := range items {
			part, _ := i.NameMatch(itemName, true)

			if part {
				closeMatchItemCt++
				if closeMatchItemCt == itemNumber {
					closeMatchItem = i
					break
				}
			}

		}

	}

	if matchItem.ItemId > 0 {
		return Item{}, matchItem
	}

	if closeMatchItem.ItemId > 0 {
		return closeMatchItem, Item{}
	}

	return Item{}, Item{}
}

// Weight is the item's encumbrance weight in grams (Phase 28), from its
// base data: an item carrying its own spec copy (taken from cargo,
// enchanted, renamed, or saved before weights were authored) would
// otherwise keep a stale weight for good. An item with no base data uses
// its own spec.
func (i *Item) Weight() int {
	if base := GetItemSpec(i.ItemId); base != nil {
		// Phase 36a: an identified roll's weight affixes lighten the base.
		return i.Loot.weight(base.Weight)
	}
	if i.Spec != nil {
		return i.Spec.Weight
	}
	return 0
}

// CarryBonusGrams is a pack's added carrying capacity (Phase 32f), read
// from base data like Weight.
func (i *Item) CarryBonusGrams() int {
	if base := GetItemSpec(i.ItemId); base != nil {
		return base.CarryBonus
	}
	if i.Spec != nil {
		return i.Spec.CarryBonus
	}
	return 0
}

// SaddleKind is the kind of horse the item fits as a saddle (Phase 32f),
// or "" for anything that isn't one. Read from base data like Weight.
func (i *Item) SaddleKind() SaddleKind {
	if base := GetItemSpec(i.ItemId); base != nil {
		return base.Saddle
	}
	if i.Spec != nil {
		return i.Spec.Saddle
	}
	return ""
}

// TierDescription is the catalog line for look and inspect: the item's tier
// and family ("Tier 2 (Steel) glaive"). Empty for an untiered item.
func (i ItemSpec) TierDescription() string {
	if i.Tier < 1 {
		return ""
	}
	line := fmt.Sprintf("Tier %d (%s)", i.Tier, TierName(i.Tier))
	if i.Family != "" {
		line += " " + i.Family
	}
	return line + ". See help equipmenttiers."
}

// GoodsDescription is the trade line for look and inspect: the category and
// what a kilogram of the item is worth.
func (i ItemSpec) GoodsDescription() string {
	if !i.IsGoods() {
		return ""
	}
	return fmt.Sprintf("Trade good (%s): worth about %.0f gold a kg. See help goods.", i.Goods, i.GoodsValuePerKg())
}
