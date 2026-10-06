package items

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type ItemType string
type ItemSubType string
type Element string
type Intensity string
type TokenName string

type WeaponHands = int

var (
	items map[int]*ItemSpec = make(map[int]*ItemSpec)
)

type ItemTypeInfo struct {
	Type        string
	Description string
	Count       int
	MinItemId   int
	MaxItemId   int
}

// Returns key=type and value=description
func ItemTypes() []ItemTypeInfo {
	return []ItemTypeInfo{
		// Equipment
		// Equipment - Weapons
		{string(Weapon), `This can be wielded as a weapon.`, 0, 10000, 19999},
		// Equipment - Armor
		{string(Offhand), `This can be worn in the offhand.`, 0, 20000, 29999},
		{string(Head), `This can be worn in the players head equipment slot.`, 0, 20000, 29999},
		{string(Neck), `This can be worn in the players neck equipment slot.`, 0, 20000, 29999},
		{string(Body), `This can be worn in the players body equipment slot.`, 0, 20000, 29999},
		{string(Belt), `This can be worn in the players belt equipment slot.`, 0, 20000, 29999},
		{string(Gloves), `This can be worn in the players gloves equipment slot.`, 0, 20000, 29999},
		{string(Ring), `This can be worn in the players ring equipment slot.`, 0, 20000, 29999},
		{string(Legs), `This can be worn in the players legs equipment slot.`, 0, 20000, 29999},
		{string(Feet), `This can be worn in the players feet equipment slot.`, 0, 20000, 29999},
		{string(Pack), `Assigned container supplying company cargo capacity.`, 0, 0, 9999},
		// Consumables
		{string(Potion), `This is a magic potion.`, 0, 30000, 39999},
		{string(Food), `This is food.`, 0, 30000, 39999},
		{string(Drink), `This is a drink.`, 0, 30000, 39999},
		{string(Scroll), `This is a scroll.`, 0, 0, 9999},
		{string(Grenade), `This is an explosive object.`, 0, 0, 9999},
		{string(Junk), `This is garbage.`, 0, 0, 9999},
		// Other
		{string(Readable), `This can be read.`, 0, 0, 9999},
		{string(Key), `This is a key that opens a locked container or door.`, 0, 0, 9999},
		{string(Object), `This is a catch-all generic object without pre-defined special behaviors.`, 0, 0, 9999},
		{string(Gemstone), `This is a gemstone.`, 0, 0, 9999},
		{string(Lockpicks), `This allows use of the picklock skill.`, 0, 0, 9999},
		{string(Botanical), `This is an herb.`, 0, 30000, 39999},
		{string(Commodity), `A raw trade good or crafting ingredient.`, 0, 0, 9999},
	}
}

// Returns key=subtype and value=description
func ItemSubtypes() []ItemTypeInfo {
	return []ItemTypeInfo{
		// Miscellaneous
		{string(Wearable), `Can be targetted with the equip/wear/wield command.`, 0, 0, 0},
		{string(Drinkable), `Can be targetted with the drink command.`, 0, 0, 0},
		{string(Edible), `Can be targetted with the eat command.`, 0, 0, 0},
		{string(Usable), `Can be targetted with the use command.`, 0, 0, 0},
		{string(Throwable), `Can be targetted with the throw command.`, 0, 0, 0},
		{string(Mundane), `No special behavior built in.`, 0, 0, 0},
		// Weapons
		{string(Generic), `Any weapon that doesn't get assigned an actual weapon subcategory.`, 0, 0, 0},
		{string(Bludgeoning), `A blunt weapon.`, 0, 0, 0},
		{string(Cleaving), `A hacking/chopping weapon.`, 0, 0, 0},
		{string(Stabbing), `A piercing weapon.`, 0, 0, 0},
		{string(Slashing), `A slicing and slashing weapon.`, 0, 0, 0},
		{string(Shooting), `A ranged weapon.`, 0, 0, 0},
		{string(Claws), `A slashing weapon worn on the hands.`, 0, 0, 0},
		{string(Whipping), `A whipping weapon.`, 0, 0, 0},
		// Miscellaneous data
		{string(BlobContent), `Can store blob content in the item data.`, 0, 0, 0},
	}
}

const (
	Unknown ItemType = ""

	// Equipment
	Weapon  ItemType = "weapon"
	Offhand ItemType = "offhand"
	Head    ItemType = "head"
	Neck    ItemType = "neck"
	Body    ItemType = "body"
	Belt    ItemType = "belt"
	Gloves  ItemType = "gloves"
	Ring    ItemType = "ring"
	Legs    ItemType = "legs"
	Feet    ItemType = "feet"
	Pack    ItemType = "pack"
	// Consumables
	Potion  ItemType = "potion"
	Food    ItemType = "food"
	Drink   ItemType = "drink"
	Scroll  ItemType = "scroll"
	Grenade ItemType = "grenade" // Expected to be thrown
	Junk    ItemType = "junk"

	// Other
	Readable  ItemType = "readable"  // Something with writing to reveal when read
	Key       ItemType = "key"       // A key for a door
	Object    ItemType = "object"    // A mundane object
	Gemstone  ItemType = "gemstone"  // A gem
	Lockpicks ItemType = "lockpicks" // Used for lockpicking
	Botanical ItemType = "botanical" // A plant, herb, etc.
	Commodity ItemType = "commodity" // A raw trade good or crafting ingredient
	Service   ItemType = "service"   // Possibly a ticket,action, or favor being purchased

	// Subtypes for wearables
	Wearable  ItemSubType = "wearable"
	Drinkable ItemSubType = "drinkable"
	Edible    ItemSubType = "edible"
	Usable    ItemSubType = "usable"
	Throwable ItemSubType = "throwable" // If dropped/thrown, triggers buff effects on room and is lost
	Mundane   ItemSubType = "mundane"

	// Subtypes for weapons, chooses attack messages.
	Generic     ItemSubType = "generic"
	Bludgeoning ItemSubType = "bludgeoning"
	Cleaving    ItemSubType = "cleaving"
	Stabbing    ItemSubType = "stabbing"
	Slashing    ItemSubType = "slashing"
	Shooting    ItemSubType = "shooting" // bows, crossbows, guns, etc.
	Claws       ItemSubType = "claws"
	Whipping    ItemSubType = "whipping"

	BlobContent ItemSubType = "blobcontent"

	OneHanded WeaponHands = 1
	TwoHanded WeaponHands = 2

	Fire        Element = "fire"
	Water       Element = "water"
	Ice         Element = "ice"
	Electricity Element = "electricity"
	Acid        Element = "acid"
	Life        Element = "life"
	Death       Element = "death"

	// Intensity of the attack
	Prepare  Intensity = "prepare"
	Wait     Intensity = "wait"
	Miss     Intensity = "miss"
	Weak     Intensity = "weak"
	Normal   Intensity = "normal"
	Heavy    Intensity = "heavy"
	Critical Intensity = "critical"

	// Tokens
	TokenItemName     TokenName = "{itemname}"
	TokenSource       TokenName = "{source}"
	TokenSourceType   TokenName = "{sourcetype}" // will be 'user' or 'mob'
	TokenTarget       TokenName = "{target}"
	TokenTargetType   TokenName = "{targettype}" // will be 'user' or 'mob'
	TokenUsesLeft     TokenName = "{usesleft}"
	TokenDamage       TokenName = "{damage}"
	TokenEntranceName TokenName = "{entrancename}"
	TokenExitName     TokenName = "{exitname}"
	TokenSourceHe     TokenName = "{sourcehe}"
	TokenSourceHim    TokenName = "{sourcehim}"
	TokenSourceHis    TokenName = "{sourcehis}"
	TokenTargetHe     TokenName = "{targethe}"
	TokenTargetHim    TokenName = "{targethim}"
	TokenTargetHis    TokenName = "{targethis}"

	POVUser  = 0
	POVOther = 1
)

type Damage struct {
	Attacks     int    `yaml:"attacks,omitempty"` // How many attacks this weapon gets (usually 1)
	DiceRoll    string // 1d6, etc.
	CritBuffIds []int  `yaml:"critbuffids,omitempty"` // If this damage is a crit, what buffs does it apply?
	DiceCount   int    `yaml:"dicecount,omitempty"`   // how many dice to roll for this weapons damage
	SideCount   int    `yaml:"sidecount,omitempty"`   // how many sides per dice roll
	BonusDamage int    `yaml:"bonusdamage,omitempty"` // flat damage bonus, so for example 1d6+1
}

type ItemMessage string

// Attack messages
type AttackMessageOptions []ItemMessage
type AttackEffects map[Intensity]AttackMessageOptions
type AttackMessages map[ItemSubType]AttackEffects

// The blueprint for an item
type ItemSpec struct {
	ItemId          int
	Value           int
	Tier            int         `yaml:"tier,omitempty"`            // Phase 36a: material and power budget 1-6; 0 means unset (tier 1)
	Uses            int         `yaml:"uses,omitempty"`            // How many uses it starts with
	Refillable      string      `yaml:"refillable,omitempty"`      // Phase 40a: what it can be refilled with ("water"), back to Uses, at a room with that resource
	EmptyItemId     int         `yaml:"emptyitemid,omitempty"`     // Phase 43a: what is left when its last use is spent (an empty waterskin), instead of nothing
	FilledItemId    int         `yaml:"filleditemid,omitempty"`    // Phase 43a: on a refillable empty container, the full item a fill turns it into
	BuffIds         []int       `yaml:"buffids,omitempty"`         // What buffs it can apply (if used)
	WornBuffIds     []int       `yaml:"wornbuffids,omitempty"`     // BuffId's that are applied while worn, and expired when removed.
	DamageReduction int         `yaml:"damagereduction,omitempty"` // % of damage it reduces when it blocks attacks
	WaitRounds      int         `yaml:"waitrounds,omitempty"`      // How many extra rounds each combat requires
	Hands           WeaponHands `yaml:"hands"`                     // How many hands it takes to wield
	Name            string
	DisplayName     string `yaml:"displayname,omitempty"` // Name that is typically displayed to the user
	NameSimple      string // A simpler name for the item, for example "Golden Battleaxe" should be "Battleaxe" or "Axe" for simple
	Description     string
	QuestToken      string `yaml:"questtoken,omitempty"` // Grants this quest if given/picked up
	Type            ItemType
	Subtype         ItemSubType
	Damage          Damage
	Element         Element           `yaml:"element,omitempty"`
	StatMods        statmods.StatMods `yaml:"statmods,omitempty"`    // What stats it modifies when equipped
	BreakChance     uint8             `yaml:"breakchance,omitempty"` // Chance in 100 that the item will break when used, or when the character is hit with it equipped, or if it is in the characters inventory during an explosion, etc.
	Cursed          bool              `yaml:"cursed,omitempty"`      // Can't be removed once equipped
	KeyLockId       string            `yaml:"keylockid,omitempty"`   // Example: `778-north` - If it's a key, what lock does it open? roomid-exitname etc.
	Nutrition       int               `yaml:"nutrition,omitempty"`   // Survival hunger benefit when eaten; zero keeps ordinary food behavior
	Hydration       int               `yaml:"hydration,omitempty"`   // Survival thirst benefit when eaten or drunk; zero keeps ordinary drink behavior
	Recipe          int               `yaml:"recipe,omitempty"`      // Phase 56: a recipe page teaches this dish (an item ID) when used; never resold
	Meal            string            `yaml:"meal,omitempty"`        // Phase 50: a cooked meal's buff kind (survival.MealKinds), given when eaten
	Ailment         string            `yaml:"ailment,omitempty"`     // Phase 55: an ailment kind (survival.AilmentKinds) eating this gives (raw game meat's Gut-ache)
	Weight          int               `yaml:"weight,omitempty"`      // Encumbrance weight in grams; zero means unweighted (no load contribution)
	Reach           bool              `yaml:"reach,omitempty"`       // Polearm-class weapon: extends melee reach to a column's frontmost-or-one-behind occupant (see Phase 11c)
	Sling           bool              `yaml:"sling,omitempty"`       // cold delays this weapon, never inferred from its name
	Parry           int               `yaml:"parry,omitempty"`       // Added to the subtype's parry modifier, in percent (Phase 30g2: a staff +5)
	WarmthBonus     int               `yaml:"warmthbonus,omitempty"` // Phase 36a: affix warmth added on top of Warmth (or the slot default)
	Warmth          int               `yaml:"warmth,omitempty"`      // Insulation when worn (Phase 15); 0 uses the exposure module's per-slot default, negative means none
	CarryBonus      int               `yaml:"carrybonus,omitempty"`  // A pack's added carrying capacity in grams (Phase 32f); a member counts only their largest
	Saddle          SaddleKind        `yaml:"saddle,omitempty"`      // A saddle's kind (Phase 32f): fits a horse of the same kind
	Bulk            string            `yaml:"bulk,omitempty"`        // Armor bulk (Phase 35a2): light, medium or heavy; defaulted from weight at load
	ShieldSize      string            `yaml:"shieldsize,omitempty"`  // A shield's size (Phase 35a2): buckler, shield or tower; defaulted to shield
	WeaponClass     string            `yaml:"weaponclass,omitempty"` // A weapon's class (Phase 35a2), e.g. mace, staff, rod, club, improvised
	Family          string            `yaml:"family,omitempty"`      // Phase 36b: catalog family, e.g. glaive, leather, kite shield; shown on look, never inferred from the name
	Goods           string            `yaml:"goods,omitempty"`       // Phase 36b: trade goods category (trophy, salvage, material, valuable, provision, curio)
	WornBy          []string          `yaml:"wornby,omitempty"`      // Phase 38e: creature species (archetype ids) this gear is cut for; only they wear it, and they wear nothing else
	Relic           *RelicSpec        `yaml:"relic,omitempty"`       // Phase 36d: an authored Legendary or Set piece: its signature or set, item level and boss
}

// Trade goods categories (Phase 36b). Goods are sold, not worn: they make
// pack capacity a choice.
const (
	GoodsTrophy    = "trophy"
	GoodsSalvage   = "salvage"
	GoodsMaterial  = "material"
	GoodsValuable  = "valuable"
	GoodsProvision = "provision"
	GoodsCurio     = "curio"
)

// GoodsCategories lists every valid goods category in display order.
func GoodsCategories() []string {
	return []string{GoodsTrophy, GoodsSalvage, GoodsMaterial, GoodsValuable, GoodsProvision, GoodsCurio}
}

// MaxTier is the highest equipment tier (Relic).
const MaxTier = 6

var tierNames = [MaxTier + 1]string{"Common", "Common", "Steel", "Tempered", "Masterwork", "Runeforged", "Relic"}

// TierName is a tier's name: 1 Common, 2 Steel, 3 Tempered, 4 Masterwork,
// 5 Runeforged, 6 Relic. An unset tier (0) is Common.
func TierName(tier int) string {
	if tier < 0 || tier > MaxTier {
		return ""
	}
	return tierNames[tier]
}

// IsGoods reports whether the spec is a trade good.
func (i ItemSpec) IsGoods() bool { return i.Goods != "" }

// GoodsValuePerKg is a good's value for each kilogram it weighs, as the
// player sees it when deciding what a full pack should hold. Zero for an
// item with no weight.
func (i ItemSpec) GoodsValuePerKg() float64 {
	if i.Weight <= 0 {
		return 0
	}
	return float64(i.Value) * 1000 / float64(i.Weight)
}

// Armor bulk (Phase 35a2) and the weights that set it when an item names
// none.
const (
	BulkLight  = "light"
	BulkMedium = "medium"
	BulkHeavy  = "heavy"

	BulkHeavyGrams  = 6000
	BulkMediumGrams = 2500

	ShieldBuckler = "buckler"
	ShieldNormal  = "shield"
	ShieldTower   = "tower"
)

// IsArmor reports whether the spec is worn armor: a wearable that is not
// a pack. Weapons have no bulk.
func (i ItemSpec) IsArmor() bool {
	return i.Type != Weapon && i.Type != Pack && i.Subtype == Wearable
}

// IsShield reports whether the spec is a shield: an off-hand wearable that
// reduces damage (a lantern or holy symbol, with no armor, is not).
func (i ItemSpec) IsShield() bool {
	return i.Type == Offhand && i.Subtype == Wearable && i.DamageReduction > 0
}

// BulkForWeight is the bulk of armor that names none: 6 kg or more is
// heavy, 2.5 kg or more medium, else light.
func BulkForWeight(grams int) string {
	switch {
	case grams >= BulkHeavyGrams:
		return BulkHeavy
	case grams >= BulkMediumGrams:
		return BulkMedium
	}
	return BulkLight
}

// SaddleKind is the kind of horse a saddle fits (Phase 32f).
type SaddleKind string

const (
	SaddlePack   SaddleKind = "pack"
	SaddleRiding SaddleKind = "riding"
)

// AllEquipSlots returns every equipment slot ItemType in canonical display order.
// This is the single authoritative definition used by the characters, races,
// and combat packages to avoid per-package slot-list duplication.
func AllEquipSlots() []ItemType {
	return []ItemType{
		Weapon,
		Offhand,
		Head,
		Neck,
		Body,
		Belt,
		Gloves,
		Ring,
		Legs,
		Feet,
		Pack,
	}
}

// WeaponSlots returns the slots that hold weapons.
func WeaponSlots() []ItemType {
	return []ItemType{Weapon, Offhand}
}

// ArmorSlots returns every equipment slot except Weapon — the slots that hold
// armor and wearable items providing passive protection or stat benefits.
func ArmorSlots() []ItemType {
	return []ItemType{
		Offhand,
		Head,
		Neck,
		Body,
		Belt,
		Gloves,
		Ring,
		Legs,
		Feet,
	}
}

func (i Element) String() string {
	return string(i)
}

func (i ItemType) String() string {
	return string(i)
}

func (i ItemSubType) String() string {
	return string(i)
}

func (d *Damage) String() string {
	if d.DiceRoll == "" {
		return "N/A"
	}
	return d.DiceRoll
}

func (d *Damage) FormatDiceRoll() string {

	d.DiceRoll = util.FormatDiceRoll(d.Attacks, d.DiceCount, d.SideCount, d.BonusDamage, d.CritBuffIds)
	if d.DiceRoll == "0@0d0" {
		d.DiceRoll = ""
	}
	return d.DiceRoll
}

func (d *Damage) InitDiceRoll(dRoll string) {
	// If diceroll is specified, it overrides whatever stats are already there
	if len(dRoll) < 1 {
		return
	}

	d.Attacks, d.DiceCount, d.SideCount, d.BonusDamage, _ = util.ParseDiceRoll(dRoll)
}

func FindItem(nameOrId string) int {
	if itemId, err := strconv.Atoi(nameOrId); err == nil {
		if itm := New(itemId); itm.ItemId != 0 {
			return itm.ItemId
		}
	}

	return FindItemByName(nameOrId)
}

func FindKeyByLockId(lockId string) int {

	for _, item := range items {
		if item.Type != Key {
			continue
		}
		if item.KeyLockId == lockId {
			return item.ItemId
		}
	}

	return 0
}

func FindItemByName(name string) int {
	name = strings.ToLower(name)

	for _, item := range items {
		if strings.ToLower(item.Name) == name {
			return item.ItemId
		}
	}

	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item.Name), name) {
			return item.ItemId
		}
	}

	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), name) {
			return item.ItemId
		}
	}

	return 0
}

func GetAllItemSpecs() []ItemSpec {

	itemSpecs := []ItemSpec{}
	for _, item := range items {
		itemSpecs = append(itemSpecs, *item)
	}
	return itemSpecs
}

func GetAllItemNames() []string {

	itemNames := []string{}
	for _, item := range items {
		itemNames = append(itemNames, item.Name)
	}
	return itemNames
}

// Presumably to ensure the datafile hasn't messed something up.
func (i *ItemSpec) Id() int {
	return i.ItemId
}

func CanBackstab(iSubType ItemSubType) bool {
	if iSubType == Cleaving || iSubType == Stabbing || iSubType == Slashing || iSubType == Claws {
		return true
	}
	return false
}

// GetBuffSummary returns the count and aggregate GetValue() of all WornBuffIds
// on the item spec. It is used by the admin API to surface passive buff info
// in the item list without requiring callers to load each buff separately.
func GetBuffSummary(itemId int) (count int, value int) {
	spec, ok := items[itemId]
	if !ok || spec == nil {
		return 0, 0
	}
	for _, bId := range spec.WornBuffIds {
		if bs := buffs.GetBuffSpec(bId); bs != nil {
			count++
			value += bs.GetValue()
		}
	}
	return count, value
}

func (i *ItemSpec) AutoCalculateValue() {

	val := 5 // base value of 5

	// Weapon damage valuation: expected damage per attack = DiceCount*(SideCount+1)/2 + BonusDamage
	if i.Damage.DiceCount > 0 && i.Damage.SideCount > 0 {
		expectedDmg := i.Damage.DiceCount*(i.Damage.SideCount+1)/2 + i.Damage.BonusDamage
		val += expectedDmg * i.Damage.Attacks * 15
	}
	// Armor damage reduction valuation
	val += i.DamageReduction * 17

	// Get the value of any buff it applies on use
	for _, buffId := range i.BuffIds {
		if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
			val += buffSpec.GetValue()
		}
	}

	// Get the value of any buff applied while worn
	for _, buffId := range i.WornBuffIds {
		if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
			val += buffSpec.GetValue()
		}
	}

	for _, statMod := range i.StatMods {
		val += statMod * 11
	}

	// Consumables scale with number of uses
	if i.Uses > 1 {
		val *= i.Uses
	}

	if i.Type == Lockpicks {
		val *= 2
	}

	if i.Hands > 1 {
		val = int(math.Ceil(float64(val) * 1.25))
	}

	if i.Type == Ring {
		// rings are automatically worth more, since they are jewelry
		val *= 2
	}

	i.Value = val
}

func (i *ItemSpec) ItemFolder(baseonly ...bool) string {
	if i.ItemId >= 50000 && i.ItemId < 60000 {
		return `relics-50000`
	} else if i.ItemId >= 40000 {
		return ``
	} else if i.ItemId >= 30000 {
		return `consumables-30000`
	} else if i.ItemId >= 20000 {
		if len(baseonly) > 0 && baseonly[0] {
			return `armor-20000`
		}
		return `armor-20000/` + string(i.Type)
	} else if i.ItemId >= 10000 {
		return `weapons-10000`
	}
	return `other-0`
}

func (i *ItemSpec) Filepath() string {
	folder := i.ItemFolder()
	if folder == `` {
		return i.Filename()
	}
	return folder + `/` + i.Filename()
}

// Presumably to ensure the datafile hasn't messed something up.
func (i *ItemSpec) Validate() error {
	if i.Sling && (i.Type != Weapon || i.Subtype != Shooting) {
		return fmt.Errorf("sling capability requires a shooting weapon")
	}

	if i.Name == `` {
		return fmt.Errorf("item has no name")
	}

	// Phase 35a2: bulk and shield size, checked and defaulted.
	i.Bulk = strings.ToLower(strings.TrimSpace(i.Bulk))
	i.ShieldSize = strings.ToLower(strings.TrimSpace(i.ShieldSize))
	i.WeaponClass = strings.ToLower(strings.TrimSpace(i.WeaponClass))
	switch i.Bulk {
	case ``, BulkLight, BulkMedium, BulkHeavy:
	default:
		return fmt.Errorf("unknown bulk %q", i.Bulk)
	}
	if i.Bulk != `` && !i.IsArmor() {
		return fmt.Errorf("bulk requires wearable armor")
	}
	if i.IsArmor() && i.Bulk == `` {
		i.Bulk = BulkForWeight(i.Weight)
	}
	switch i.ShieldSize {
	case ``:
		if i.IsShield() {
			i.ShieldSize = ShieldNormal
		}
	case ShieldBuckler, ShieldNormal, ShieldTower:
		if !i.IsShield() {
			return fmt.Errorf("shield size requires a shield")
		}
	default:
		return fmt.Errorf("unknown shield size %q", i.ShieldSize)
	}
	if i.WeaponClass != `` && i.Type != Weapon {
		return fmt.Errorf("weapon class requires a weapon")
	}

	// Phase 36b: tier, family and goods category, checked.
	if i.Tier < 0 || i.Tier > MaxTier {
		return fmt.Errorf("tier %d is outside 0 to %d", i.Tier, MaxTier)
	}
	i.Family = strings.ToLower(strings.TrimSpace(i.Family))
	i.Goods = strings.ToLower(strings.TrimSpace(i.Goods))
	for n, who := range i.WornBy {
		i.WornBy[n] = strings.ToLower(strings.TrimSpace(who))
	}
	if i.Goods != `` {
		valid := false
		for _, c := range GoodsCategories() {
			valid = valid || c == i.Goods
		}
		if !valid {
			return fmt.Errorf("unknown goods category %q", i.Goods)
		}
		if i.Type == Weapon || i.Type == Pack || i.IsArmor() {
			return fmt.Errorf("goods cannot be equipment")
		}
	}

	if i.Relic != nil {
		if err := i.Relic.validate(i); err != nil {
			return fmt.Errorf("relic: %w", err)
		}
	}

	if i.CarryBonus < 0 || (i.Type == Pack && (i.CarryBonus <= 0 || i.Subtype != Wearable || len(i.StatMods) > 0 || len(i.WornBuffIds) > 0 || i.DamageReduction != 0)) {
		return fmt.Errorf("pack must be wearable with positive capacity and no combat modifiers")
	}

	if i.Nutrition < 0 {
		return fmt.Errorf("item nutrition cannot be negative")
	}

	if i.Hydration < 0 {
		return fmt.Errorf("item hydration cannot be negative")
	}

	if i.Recipe < 0 || (i.Recipe > 0 && i.Subtype != Usable) {
		return fmt.Errorf("recipe page must be usable and name a dish")
	}

	if i.Type == Weapon {
		if i.Hands == 0 {
			i.Hands = 1
		}
		if i.Hands > TwoHanded {
			i.Hands = TwoHanded
		}
		if i.Damage.Attacks < 1 {
			i.Damage.Attacks = 1
		}
	}

	if i.DamageReduction < 0 {
		i.DamageReduction = 0
	} else if i.DamageReduction > 100 {
		i.DamageReduction = 100
	}

	if i.BreakChance > 100 {
		i.BreakChance = 100
	}

	if i.NameSimple == `` {
		i.NameSimple = i.Name
	}

	if i.DisplayName != `` {
		i.DisplayName = util.ConvertColorShortTags(i.DisplayName)
	}

	i.Damage.InitDiceRoll(i.Damage.DiceRoll)
	i.Damage.FormatDiceRoll()

	if i.Value < 1 {
		i.AutoCalculateValue()
	}

	return nil
}

func (i *ItemSpec) Filename() string {
	filename := util.ConvertForFilename(i.Name)
	return fmt.Sprintf("%d-%s.yaml", i.ItemId, filename)
}

func (i ItemSpec) GetScript() string {

	// Check plugin-registered scripts first.
	if script := getPluginScript(i.ItemId); script != `` {
		return script
	}

	scriptPath := i.GetScriptPath()

	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := util.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}

	return ``
}

func (i *ItemSpec) GetScriptPath() string {
	// Load any script for the item (prefers .js, falls back to .lua)
	return util.ResolveScriptPath(string(configs.GetFilePathsConfig().DataFiles) + `/items/` + i.Filepath())
}

func GetItemSpec(itemId int) *ItemSpec {
	if itemId > 0 {
		spec, ok := items[itemId]
		if ok {
			return spec
		}
	}
	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpItems, err := fileloader.LoadAllFlatFiles[int, *ItemSpec](string(configs.GetFilePathsConfig().DataFiles) + `/items`)
	if err != nil {
		panic(err)
	}

	items = tmpItems

	// Merge items from plugin file systems.
	loadPluginItems(items)

	tmpAttackMessages, err := fileloader.LoadAllFlatFiles[ItemSubType, *WeaponAttackMessageGroup](string(configs.GetFilePathsConfig().DataFiles) + `/combat-messages`)
	if err != nil {
		panic(err)
	}

	attackMessages = tmpAttackMessages

	mudlog.Info("itemspec.LoadDataFiles()", "itemLoadedCount", len(items), "attackMessageCount", len(attackMessages), "Time Taken", time.Since(start))

}
