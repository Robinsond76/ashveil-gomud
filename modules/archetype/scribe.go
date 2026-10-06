package archetype

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 36a Scribe. The skill (ranks 1-4, a caster skill; a companion
// learns it with `company train`) identifies Rare and better gear: at a
// completed camp rest automatically (autoskill scribe, no mana), and in
// the field with the `scribe` command (mana, never in battle). The
// company's best living Scribe present does the work, ties to the leader
// and then the lowest companion ID, and is named.

var _ archetypes.ScribeProvider = (*ArchetypeModule)(nil)

const (
	utilityScribe = "scribe"
	scribeUsage   = `Usage: scribe | scribe [item] | scribe [member] [item]. See <ansi fg="command">help scribe</ansi>.`
)

// scribeHolder is a company member who carries gear and may read it.
type scribeHolder struct {
	name     string
	isLeader bool
	id       int // companion ID; 0 for the leader
	ch       *characters.Character
	rank     int
}

func (h scribeHolder) subject() string {
	if h.isLeader {
		return "You"
	}
	return h.name
}

func (h scribeHolder) verb(you, other string) string {
	if h.isLeader {
		return you
	}
	return other
}

// scribeHolders is the leader and the living, able companions standing in
// roomIDs, the leader first then by companion ID.
func scribeHolders(user *users.UserRecord, roomIDs ...int) []scribeHolder {
	var out []scribeHolder
	for _, mb := range companyMembers(user, roomIDs...) {
		h := scribeHolder{name: mb.Name, isLeader: mb.user != nil, id: mb.CompanionID}
		if mb.user != nil {
			h.ch = mb.user.Character
		} else {
			h.ch = &mb.mob.Character
		}
		h.rank = h.ch.GetSkillLevel(loot.ScribeSkill)
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].isLeader != out[j].isLeader {
			return out[i].isLeader
		}
		return out[i].id < out[j].id
	})
	return out
}

// itemRef points at an item a character carries or wears, so identifying
// it changes the real instance.
type itemRef struct {
	*items.Item
	worn bool
}

// itemRefs lists every item a character carries, and wears when equipped.
func itemRefs(c *characters.Character, equipped bool) []itemRef {
	var out []itemRef
	for i := range c.Items {
		out = append(out, itemRef{Item: &c.Items[i]})
	}
	if equipped {
		for _, slot := range characters.AllSlots() {
			if itm := c.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
				out = append(out, itemRef{Item: itm, worn: true})
			}
		}
	}
	return out
}

// bestScribe is the holder with the highest rank among those eligible
// (rank > 0): the first on a tie, and holders arrive leader first.
func bestScribe(holders []scribeHolder, eligible func(scribeHolder) bool) (scribeHolder, bool) {
	var best scribeHolder
	for _, h := range holders {
		if h.rank < 1 || (eligible != nil && !eligible(h)) {
			continue
		}
		if best.rank < h.rank {
			best = h
		}
	}
	return best, best.rank > 0
}

// CampIdentify implements archetypes.ScribeProvider. It runs on the game
// loop when a camp rest completes. A worn item reveals itself with the rest
// (no Scribe needed, and no autoskill switch); then the best Scribe reads
// every unidentified item in the members' packs (the company's cargo is the
// leader's pack) that its rank covers, if autoskill scribe is on.
func (m *ArchetypeModule) CampIdentify(leaderUserID int) []string {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return nil
	}
	holders := scribeHolders(user, user.Character.RoomId)
	var lines []string

	for _, h := range holders {
		var worn []string
		for _, slot := range characters.AllSlots() {
			itm := h.ch.Equipment.Get(slot)
			if itm != nil && itm.ItemId > 0 && itm.Identify() {
				worn = append(worn, itm.DisplayName())
			}
		}
		if len(worn) > 0 {
			who := "Your"
			if !h.isLeader {
				who = h.name + "'s"
			}
			lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">By the fire, %s worn gear gives up its secrets: %s.</ansi>`, strings.ToLower(who), strings.Join(worn, ", ")))
			h.ch.Validate(true)
		}
	}

	if !m.autoskillOn(leaderUserID, utilityScribe) {
		return lines
	}
	best, ok := bestScribe(holders, nil)
	if !ok {
		return lines
	}
	var read []string
	for _, h := range holders {
		for _, ref := range itemRefs(h.ch, false) {
			if ref.IsIdentified() || !loot.ScribeCovers(best.rank, ref.RollRarity()) {
				continue
			}
			if ref.Identify() {
				read = append(read, ref.DisplayName())
			}
		}
	}
	if len(read) > 0 {
		lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">%s %s the company's new finds: %s.</ansi>`, best.subject(), best.verb("read", "reads"), strings.Join(read, ", ")))
	}
	return lines
}

func (m *ArchetypeModule) registerScribe() {
	m.plug.AddUserCommand("scribe", m.scribeCommand, false, false)
}

// scribeTarget is an unidentified item and whose it is.
type scribeTarget struct {
	holder scribeHolder
	item   *items.Item
	worn   bool
}

func unidentified(holders []scribeHolder) []scribeTarget {
	var out []scribeTarget
	for _, h := range holders {
		for _, ref := range itemRefs(h.ch, true) {
			if !ref.IsIdentified() {
				out = append(out, scribeTarget{h, ref.Item, ref.worn})
			}
		}
	}
	return out
}

func (m *ArchetypeModule) scribeCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.scribe(rest, user, room))
	return true, nil
}

func (m *ArchetypeModule) scribe(rest string, user *users.UserRecord, room *rooms.Room) string {
	if usercommands.InBattle(user) {
		return usercommands.BattleUnderWay
	}
	if _, busy := battle.Current(user.UserId); busy {
		return usercommands.BattleUnderWay
	}
	holders := scribeHolders(user, room.RoomId)
	args := strings.Fields(rest)

	if len(args) == 0 {
		return scribeList(holders)
	}

	// "scribe [member] [item]": a leading word that names a present
	// companion Scribe picks who reads.
	var chosen *scribeHolder
	if len(args) > 1 {
		for i := range holders {
			if !holders[i].isLeader && strings.EqualFold(holders[i].name, args[0]) {
				chosen = &holders[i]
				args = args[1:]
				break
			}
		}
	}
	name := strings.Join(args, " ")

	var target *scribeTarget
	for _, t := range unidentified(holders) {
		if part, _ := t.item.NameMatch(name, true); part {
			target = &t
			break
		}
	}
	if target == nil {
		for _, h := range holders {
			for _, ref := range itemRefs(h.ch, true) {
				if part, _ := ref.NameMatch(name, true); part {
					return fmt.Sprintf(`The <ansi fg="item">%s</ansi> holds no secrets: it needs no reading.`, ref.DisplayName())
				}
			}
		}
		return fmt.Sprintf(`Nobody in your company carries a "%s" to read.`, name)
	}
	rarity := target.item.RollRarity()
	need := loot.ScribeRankFor(rarity)
	cost := loot.ScribeManaCost(rarity)

	var reader scribeHolder
	if chosen != nil {
		reader = *chosen
		switch {
		case reader.rank < 1:
			return fmt.Sprintf("%s has no Scribe training.", reader.name)
		case !loot.ScribeCovers(reader.rank, rarity):
			return fmt.Sprintf("%s's Scribe (rank %d) can't read a %s item: it takes rank %d.", reader.name, reader.rank, rarity, need)
		case reader.ch.Mana < cost:
			return fmt.Sprintf("%s lacks the mana: reading a %s item takes %d (%d left).", reader.name, rarity, cost, reader.ch.Mana)
		}
	} else {
		var ok bool
		reader, ok = bestScribe(holders, func(h scribeHolder) bool {
			return loot.ScribeCovers(h.rank, rarity) && h.ch.Mana >= cost
		})
		if !ok {
			if best, any := bestScribe(holders, nil); !any {
				return "Nobody in your company has Scribe training. See <ansi fg=\"command\">help identify</ansi> for other ways to learn what gear can do."
			} else if !loot.ScribeCovers(best.rank, rarity) {
				return fmt.Sprintf("A %s item takes Scribe rank %d; the best here is rank %d (%s).", rarity, need, best.rank, best.name)
			}
			return fmt.Sprintf("Reading a %s item takes %d mana, and no Scribe here can spare it.", rarity, cost)
		}
	}

	reader.ch.Mana -= cost
	target.item.Identify()
	if target.worn {
		target.holder.ch.Validate(true) // a worn item's affixes change its wearer's numbers
	}
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	return fmt.Sprintf(`%s %s the <ansi fg="item">%s</ansi> (%d mana).`, reader.subject(), reader.verb("read", "reads"), target.item.DisplayName(), cost) +
		"\n" + target.item.RolledDescription(reader.rank)
}

// scribeList says what is waiting to be read and who can read it.
func scribeList(holders []scribeHolder) string {
	waiting := unidentified(holders)
	best, hasScribe := bestScribe(holders, nil)
	var b strings.Builder
	if hasScribe {
		fmt.Fprintf(&b, "Your best Scribe here is %s (rank %d).\n", map[bool]string{true: "you", false: best.name}[best.isLeader], best.rank)
	} else {
		b.WriteString("Nobody here has Scribe training.\n")
	}
	if len(waiting) == 0 {
		b.WriteString("Nothing the company carries is waiting to be read.")
		return b.String()
	}
	b.WriteString("Waiting to be read:\n")
	for _, t := range waiting {
		owner := "you"
		if !t.holder.isLeader {
			owner = t.holder.name
		}
		covered := ""
		if hasScribe && !loot.ScribeCovers(best.rank, t.item.RollRarity()) {
			covered = fmt.Sprintf(" (needs rank %d)", loot.ScribeRankFor(t.item.RollRarity()))
		}
		fmt.Fprintf(&b, "  %s, carried by %s%s\n", t.item.DisplayName(), owner, covered)
	}
	b.WriteString(`Use <ansi fg="command">scribe [item]</ansi>; a camp rest reads them for free.`)
	return b.String()
}
