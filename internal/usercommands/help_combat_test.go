package usercommands

import (
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCombatHelpTopics: every topic in the shipped help index's combat
// category renders, and the Ashveil combat pages answer to their aliases
// (help reach, help whetstone, ...).
func TestCombatHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" && !topic.AdminOnly {
			combat = append(combat, topic.Command)
		}
	}
	for _, want := range []string{"combat", "formation", "targeting", "strategy", "chemistry", "sharpen", "light", "battle-summary", "resurrect", "narration", "combatpace", "battlescreen", "orders"} {
		assert.Contains(t, combat, want, "help index lists %s under combat", want)
	}
	for _, topic := range combat {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for", topic)
	}

	aliases := map[string]string{
		"battle": "combat", "fighting": "combat",
		"reach": "formation", "interception": "formation",
		"target": "targeting", "whetstone": "sharpen", "darkness": "light",
		"battlesummary": "battle-summary", "resurrection": "resurrect",
		"critical": "narration", "crit": "narration", "healed": "narration", "chanting": "narration",
		"battle-screen": "battlescreen", "battle-map": "battlescreen",
		"order": "orders", "battle-orders": "orders", "when-do": "orders",
		"pace": "combatpace", "pacing": "combatpace", "combat-pace": "combatpace",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	// Phase 29c: the narration page explains the parentheses, and the
	// pages that quoted combat lines quote the new voice.
	text, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, want := range []string{"(5 damage)", "(critical hit, 9 damage)", "absorbed)", "healed)", "(chanting: "} {
		assert.Contains(t, text, want)
	}
	for _, topic := range []string{"combat", "targeting", "formation", "attack"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.NotContains(t, text, "turn on the", topic)
		assert.NotContains(t, text, "turns on the", topic)
	}
	for _, topic := range []string{"combat", "damage"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help narration", "%s links to help narration", topic)
	}

	// The death page reflects Ashveil's death rules (Phase 25a).
	text, err = GetHelpContents("death")
	require.NoError(t, err)
	assert.Contains(t, text, "one level")
	assert.Contains(t, text, "church")
}

// Phase 30g2: help defense renders with its configured numbers, answers
// to its aliases, and is linked from the combat hub; the pages the phase
// changed no longer describe the old rules.
func TestDefenseHelp(t *testing.T) {
	// The shipped numbers (Phase 35b retuned them), as the page's own
	// examples use.
	t.Chdir(filepath.Join("..", ".."))
	old := configs.GetGamePlayConfig()
	t.Cleanup(configs.SetTestGamePlayConfig(old))
	require.NoError(t, configs.ReloadConfig())
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("defense")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for defense", "block", "parry", "dodge", "28% plus the shield's own armor", "between 8% and 55%", "8% when evenly matched", "from 3% to 40%", "sword parries 13%", "an iron shield\n38%", "help evasion", "help shields",
		"swords, staves, and long polearms", "daggers", "Bows and slings can't parry", "No dodge if the block fails", "no defense at all"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "{{", "every config number rendered")
	for _, alias := range []string{"parry", "dodge", "parrying", "defence"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help defense", alias)
	}
	// The rest reads a blank gameplay config, not whatever an earlier test (or
	// the first pass of -count=2) left loaded.
	defaults := configs.GamePlay{}
	defaults.Validate()
	configs.SetTestGamePlayConfig(defaults)

	for _, topic := range []string{"combat", "armor", "interrupts", "statuses", "battle-summary"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, page, "help defense", "%s links to help defense", topic)
	}

	armor, err := GetHelpContents("armor")
	require.NoError(t, err)
	assert.Contains(t, armor, "(5 damage, 2 absorbed)")
	assert.NotContains(t, armor, "Phase", "no phase numbers in player help")
	assert.NotContains(t, armor, "50%")

	interrupts, err := GetHelpContents("interrupts")
	require.NoError(t, err)
	plain = tagPattern.ReplaceAllString(interrupts, "")
	assert.Contains(t, plain, "5% to 20%")
	assert.Contains(t, plain, "A plain miss is never countered")
	assert.NotContains(t, plain, "50%")
	assert.NotContains(t, plain, "{{")

	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	assert.NotContains(t, narration, "2 blocked)")
}

func TestCombatHelpPronounsAndOrdinals(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, alias := range []string{"pronouns", "ordinals"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, narration, got)
	}
	for _, want := range []string{"he", "she", "they", "it", "second cutthroat", "third cutthroat", "restart", "you", "your"} {
		assert.Contains(t, narration, want)
	}
	for _, topic := range []string{"narration", "combat", "targeting", "battle-summary"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, text, "second", topic)
		assert.NotContains(t, text, "set pronouns", topic)
	}
	targeting, err := GetHelpContents("targeting")
	require.NoError(t, err)
	assert.Contains(t, targeting, "attack [group]")
}

func TestPainReactionHelpExplainsCriticalAndLethalCases(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, want := range []string{"pain reaction", "critical hit", "stays standing", "death line"} {
		assert.Contains(t, narration, want)
	}
	combat, err := GetHelpContents("combat")
	require.NoError(t, err)
	assert.Contains(t, combat, "pain reaction")
	assert.Contains(t, combat, "help narration")
	critical, err := GetHelpContents("critical")
	require.NoError(t, err)
	assert.Equal(t, narration, critical)
	pain, err := GetHelpContents("pain")
	require.NoError(t, err)
	assert.Equal(t, narration, pain)
}

// TestCombatPaceHelp (Phase 29f): the pacing page gives the four paces and
// their timings, the screen-reader default, the catch-up and flush rules,
// and what is never held back; combat, set, and narration point to it.
func TestCombatPaceHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	page, err := GetHelpContents("combatpace")
	require.NoError(t, err)
	page = tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{
		"set combatpace fast", "set combatpace normal", "set combatpace slow", "set combatpace off",
		"0.4 seconds", "done in 6 seconds", "about 7 seconds",
		"screen reader", "the default is off",
		"every 8 seconds", "before the next begins",
		"pain reaction or a death line",
		"Your prompt", "battle view",
		"Nothing is lost",
		"never held back", "tells",
	} {
		assert.Contains(t, page, want)
	}

	combat, err := GetHelpContents("combat")
	require.NoError(t, err)
	combat = tagPattern.ReplaceAllString(combat, "")
	for _, want := range []string{"every 8 seconds", "up to 8 seconds", "help combatpace", "set combatpace"} {
		assert.Contains(t, combat, want)
	}
	assert.NotContains(t, combat, "a round is 4 seconds", "the old round length is gone")

	set, err := GetHelpContents("set")
	require.NoError(t, err)
	assert.Contains(t, set, "combatpace")

	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(narration, ""), "A short pause comes before a pain reaction")
}

func TestMoraleAndMercyHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	for _, topic := range []string{"morale", "mercy", "surrender", "nerve"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, text, "Help for")
	}
	for topic, word := range map[string]string{"combat": "help morale", "alignment": "help mercy", "chemistry": "help morale"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, text, word)
	}
}

// Phase 30g3: help burden renders with its configured numbers, answers to
// its aliases, and is linked from the combat hub and the pages burden
// changes; the stale Strength line about item counts is gone.
func TestBurdenHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("burden")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for burden", "15 kg, plus 0.5 kg for each point of Strength", "The first 0.35",
		"unburdened", "lightly burdened", "burdened", "heavily burdened", "two fifths of your dodge",
		"A shield's block and a weapon's parry are not", "never make you lighter", "status", "look [someone]", "scout [group]"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "{{", "every config number rendered")
	for _, alias := range []string{"burdened", "agility", "unburdened", "personal-load"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help burden", alias)
	}

	for _, topic := range []string{"combat", "defense", "perception", "strength", "cargo", "encumbrance", "scout"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, page, "help burden", "%s links to help burden", topic)
	}
	strength, err := GetHelpContents("strength")
	require.NoError(t, err)
	assert.NotContains(t, strength, "how many items you can carry")
	status, err := GetHelpContents("status")
	require.NoError(t, err)
	assert.Contains(t, status, "burden")
}

func TestFormationDefaultsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("formation")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"row 2, column 2", "vacant", "deliberate clear", "once"} {
		assert.Contains(t, plain, want)
	}
	text, err = GetHelpContents("company-inventory")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "every member's equipment slots")
}

// Phase 38a: the Witch and hexes pages render, are indexed (hexes under
// combat, witch under character), answer to their aliases, and the pages
// the sixth class touched name it.
func TestWitchHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	category := map[string]string{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		category[topic.Command] = topic.Category
	}
	assert.NotEmpty(t, category["witch"], "witch is indexed")
	assert.Equal(t, "combat", category["hexes"])

	for topic, aliases := range map[string][]string{
		"witch":    {"witches", "hexcraft"},
		"hexes":    {"hex", "slumber", "binding", "blight", "dread"},
		"statuses": {"asleep", "paralyzed", "blighted"},
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(want, ""), "Help for", topic)
		for _, alias := range aliases {
			got, err := GetHelpContents(alias)
			require.NoError(t, err, alias)
			assert.Equal(t, want, got, "help %s is help %s", alias, topic)
		}
	}

	for topic, wants := range map[string][]string{
		"hexes":     {"Slumber", "Earthbind", "Binding Hex", "Blight", "level 30", "(resisted)"},
		"witch":     {"controller", "help hexes"},
		"statuses":  {"Asleep", "Paralyzed", "Blighted"},
		"combat":    {"help hexes"},
		"archetype": {"Witch"},
		"strategy":  {"witches hex"},
		"health":    {"Witch"},
		"growth":    {"Witch"},
		"armor":     {"Witch"},
		"shields":   {"Witch"},
		"evasion":   {"Witch"},
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		for _, want := range wants {
			assert.Contains(t, text, want, "help %s mentions %s", topic, want)
		}
	}
}

// TestClassHelpTopics: Phase 38b's class pages are indexed, answer to their
// aliases and render, and the old "help class" now reaches the class hub.
func TestClassHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	pages := map[string][]string{
		"classes":           {"Ranks", "class promote", "Hierarch"},
		"promotion":         {"Level 10", "class promote [class] confirm"},
		"talents":           {"Mending Hands", "talent pick [talent] confirm"},
		"cleric-routes":     {"Greater Heal", "Rejuvenation", "Siphon"},
		"warrior-routes":    {"Lay on Hands", "Blood Oath", "Divine shield"},
		"halberdier":        {"Sweep", "Brace", "Hook", "crowded"},
		"halberdier-routes": {"Sweeper", "Vanguard", "Valkyrie", "Charged Sweep", "Hold the line"},
		"dollmaster":        {"Puppet Strike", "Guard String", "Tangle", "Emergency Splice", "doll parts"},
		"dollmaster-routes": {"Puppeteer", "Golemancer", "Marionettist", "Two dolls", "Golem"},
		"doll":              {"doll wield", "doll mend", "doll name", "doll remove"},
		"beasttamer":        {"Sic", "Rally", "Pack Sense", "bonded beast", "wounded"},
		"beasttamer-routes": {"Houndmaster", "Bearward", "Dragon Tamer", "Warhound", "War bear", "Drake hatchling"},
		"beast":             {"beast name", "beast [member]", "wounded"},
		"summoning":         {"Call the Host", "Bind the Fiend", "Hellfire", "Mercy"},
		"elite":             {"Warlord", "Paladin", "Dread Knight", "Promotion ready", "Elite talents", "Routes are final"},
		"warlord":           {"Marked for Ruin", "Battle Cry", "Sunder", "Relentless", "Warlord's Command", "Iron Hide"},
		// Phase 38c3: the wizard and witch elites.
		"archon":        {"Counterspell", "Twin ward", "Mana Shield", "Reflection", "Archon's Aegis", "Focused Will"},
		"archmage":      {"Overchannel", "Quick casting", "Arcane Barrage", "Archmage's Storm", "Deep Reserves"},
		"necromancer":   {"Raise the Fallen", "Deeper drain", "Grave Chill", "Death's Harvest", "Lich's Bargain", "help thrall"},
		"thrall":        {"risen", "60%", "never saved", "help necromancer"},
		"wizard-routes": {"Theurgist", "Arcanist", "Warlock", "Sorcerer", "Arcane Lance", "Life Drain", "help archon", "help high-sorcerer"},
		// Phase 38d: the Sorcerer's elite.
		"high-sorcerer": {"High Lance", "Gathered chant", "Unbroken chant", "Twin Lance", "Searing lance", "Bottomless well", "Instant Lance", "Deep Reserves"},
		"wise-one":      {"Hearthward", "Deep Slumber", "Mend Charm", "Cleansing ward", "Hearth's Peace", "Ward of Life"},
		"coven-mother":  {"Coven's Will", "Lasting hexes", "Cheaper hexes", "Breaking the boss", "Twin Hex", "Coven Circle"},
		"crone-of-ash":  {"Ashen Curse", "Rotting Miasma", "Quick curses", "Lingering Curse", "Soul Rot", "Crone's Doom"},
		"witch-routes":  {"Hedge Witch", "Coven Sage", "Hag", "help wise-one", "help crone-of-ash"},
	}
	for topic, wants := range pages {
		assert.Contains(t, listed, topic, "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for", topic)
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	aliases := map[string]string{
		"class": "classes", "routes": "classes", "promote": "promotion", "talent": "talents",
		"priest": "cleric-routes", "druid": "cleric-routes", "paladin": "warrior-routes",
		"knight": "warrior-routes", "blackguard": "warrior-routes", "angel": "summoning", "demon": "summoning",
		"hierarch": "summoning", "demonologist": "summoning",
		"elites": "elite", "elite-class": "elite", "warlords": "warlord", "battle-cry": "warlord",
		"sweep": "halberdier", "valkyrie": "halberdier-routes", "sweeper": "halberdier-routes",
		"counterspell": "archon", "overchannel": "archmage", "raise-the-fallen": "necromancer", "thralls": "thrall",
		"theurgist": "wizard-routes", "warlock": "wizard-routes", "sorcerer": "wizard-routes", "arcane-lance": "wizard-routes", "twin-lance": "high-sorcerer", "instant-lance": "high-sorcerer", "hearthward": "wise-one", "coven-circle": "coven-mother",
		"crone": "crone-of-ash", "soul-rot": "crone-of-ash", "hag": "witch-routes", "coven-sage": "witch-routes",
		"dolls": "dollmaster", "tangle": "dollmaster", "puppeteer": "dollmaster-routes", "marionettist": "dollmaster-routes",
		"mend-doll": "doll",
		"sic":       "beasttamer", "rally": "beasttamer", "bonded-beast": "beasttamer", "houndmaster": "beasttamer-routes", "bearward": "beasttamer-routes", "dragon-tamer": "beasttamer-routes",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for _, topic := range []string{"archetype", "progression", "strategy", "combat", "warrior", "witch", "halberdier", "dollmaster", "beasttamer", "promotion", "classes", "elite", "interrupts", "summoning", "talents"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "help ", topic)
	}
}

// TestBattleScreenHelp (40f): the page says what the picture shows, links
// the topics it relies on, and the combat and web client hubs point to it.
func TestBattleScreenHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("battlescreen")
	require.NoError(t, err)
	text = tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for battlescreen", "Minimise", "Open automatically", "never as numbers", "retreat", "company tactics focus [rule]", "help strategy", "Animation", "reduced", "keeps pace", "latest blow", "Allies", "company faltering", "+N more", "K controller", "setting", "hovering names the class", "help promotion", "watch that company at full size", "your band"} {
		assert.Contains(t, text, want)
	}
	for _, hub := range []string{"combat", "webclient"} {
		hubText, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, tagPattern.ReplaceAllString(hubText, ""), "help battlescreen", "help %s links the battle screen", hub)
	}
}

// TestRogueRangerEliteHelp (38c2): the rogue and ranger route pages and the
// six elite pages are indexed, render with their ranks, answer to their
// aliases, and the hubs link them.
func TestRogueRangerEliteHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	pages := map[string][]string{
		"rogue-routes":     {"Scout (good)", "Duelist (neutral)", "Assassin (evil)", "Pathfinder", "Swordmaster", "Nightblade"},
		"ranger-routes":    {"Warden (good)", "Hunter (neutral)", "Stalker (evil)", "Sentinel", "Marksman", "Ravager"},
		"elite-pathfinder": {"Pathfinder's Eye", "Ambush Master", "Trailwise", "Shadow Footing"},
		"swordmaster":      {"Blade Dance", "Twin ripostes", "Perfect Parry", "Unbroken Guard"},
		"nightblade":       {"Death Mark", "Shadowstep", "Killing Spree", "Coup de Grace"},
		"sentinel":         {"Overwatch", "Guardian Arrow", "Long Draw"},
		"marksman":         {"Called Shot", "Second Nock", "Perfect Shot"},
		"ravager":          {"Hunt Down", "Harrow", "Apex", "Eagle Eye"},
	}
	for topic, wants := range pages {
		assert.Contains(t, listed, topic, "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for", topic)
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	aliases := map[string]string{
		"assassin": "rogue-routes", "warden": "ranger-routes", "stalker": "ranger-routes",
		"pathfinder-class": "elite-pathfinder", "trailwise": "elite-pathfinder", "death-mark": "nightblade",
		"overwatch": "sentinel", "second-nock": "marksman", "hunt-down": "ravager", "blade-dance": "swordmaster",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	// The walking skill keeps its name and points at the elite page.
	skill, err := GetHelpContents("pathfinder")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(skill, ""), "elite-pathfinder")
	for _, hub := range []string{"elite", "classes", "promotion"} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "rogue-routes", hub)
		assert.Contains(t, plain, "ranger-routes", hub)
	}
	elite, err := GetHelpContents("elite")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(elite, "")
	for _, name := range []string{"Pathfinder", "Swordmaster", "Nightblade", "Sentinel", "Marksman", "Ravager"} {
		assert.Contains(t, plain, name)
	}
	assert.NotContains(t, plain, "Pathfinder, Swordmaster, Nightblade,\nSentinel", "no longer listed as still to come")
}

// Phase 61: help orders renders, names every condition and action the
// orders package offers, and the hub pages point to it.
func TestOrdersHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("orders")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for", "up to 3 orders", "as ordered", "ally [n]", "self [n]", "chanting", "boss", "first", "foe [kind]",
		"heal", "break", "guard", "strongest", "hold", "orders [who] add [condition] then [action]", "orders [who] preset", "orders [who] clear"} {
		assert.Contains(t, plain, want, "help orders mentions %s", want)
	}
	for _, topic := range []string{"combat", "strategy"} {
		hub, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, hub, "help orders", "help %s links to help orders", topic)
	}
}

// Phase 60: help events renders, answers to its aliases, and is linked
// from the adventure hub and the pages about the road and camp.
func TestEventsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var road []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "road" && !topic.AdminOnly {
			road = append(road, topic.Command)
		}
	}
	assert.Contains(t, road, "events", "help index lists events under the road")

	want, err := GetHelpContents("events")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"Help for", "choose 2", "event", "(closed: needs a rogue)", "chancy", "Nothing here moves the world's clock"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"event", "scene", "scenes", "story events", "choose"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help events", alias)
	}
	for _, hub := range []string{"adventure", "travel", "camp"} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, text, "help events", "%s links to help events", hub)
	}
}
