package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bestiary"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ashveil Phase 66: `bestiary` reads what the leader has learned of each
// kind of creature by beating it (internal/bestiary). It only reads: free,
// instant, and it names only kinds the leader has fought, so it never
// tells anything of a foe not yet met.

func Bestiary(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	entries := bestiary.Known(bestiary.KillsOf(user.Character))
	if len(entries) == 0 {
		user.SendText(`Your bestiary is empty. Beat a creature and it is entered: its lore at the first kill, its defences at the third and its habits at the sixth (a boss teaches faster). <ansi fg="command">help bestiary</ansi> has the details.`)
		return true, nil
	}
	arg := strings.TrimSpace(rest)
	if arg == `` {
		user.SendText(BestiaryList(entries, ``))
		return true, nil
	}
	// A zone named in full is that zone's list ("bestiary frost lake"),
	// even when a creature's name also contains the words.
	for _, e := range entries {
		if e.Zone != `` && strings.EqualFold(e.Zone, arg) {
			user.SendText(BestiaryList(entries, arg))
			return true, nil
		}
	}
	if e, ok := bestiary.Find(entries, arg); ok {
		// Same-named kinds from different zones are separate entries:
		// show each, so none is hidden behind the better-known one.
		var texts []string
		for _, o := range entries {
			if strings.EqualFold(o.Name, e.Name) {
				texts = append(texts, BestiaryEntry(o))
			}
		}
		user.SendText(strings.Join(texts, "\n\n"))
		return true, nil
	}
	if text := BestiaryList(entries, arg); text != `` {
		user.SendText(text)
		return true, nil
	}
	user.SendText(fmt.Sprintf(`Nothing called "%s" is in your bestiary. <ansi fg="command">bestiary</ansi> lists what you know; a creature is entered when you beat it.`, arg))
	return true, nil
}

// BestiaryList lists the known kinds by zone; a zone filter keeps the
// zones whose name contains it (empty text when none does).
func BestiaryList(entries []bestiary.Entry, zone string) string {
	zone = strings.ToLower(zone)
	var lines []string
	last := "\x00"
	count := 0
	for _, e := range entries {
		if zone != `` && !strings.Contains(strings.ToLower(e.Zone), zone) {
			continue
		}
		count++
		if e.Zone != last {
			last = e.Zone
			name := e.Zone
			if name == `` {
				name = `Elsewhere`
			}
			lines = append(lines, fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, name))
		}
		lines = append(lines, fmt.Sprintf(`  <ansi fg="mobname">%-24s</ansi> %3d %-5s %s`, e.Name, e.Kills, kills(e.Kills), e.Tier.Name()))
	}
	if count == 0 {
		return ``
	}
	head := fmt.Sprintf(`<ansi fg="yellow-bold">Your bestiary</ansi>: %s. <ansi fg="command">bestiary [name]</ansi> reads an entry.`, plural(count, `kind`, `kinds`))
	return head + "\n" + strings.Join(lines, "\n")
}

func kills(n int) string {
	if n == 1 {
		return `kill`
	}
	return `kills`
}

// BestiaryEntry is one entry in words.
func BestiaryEntry(e bestiary.Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<ansi fg="yellow-bold">%s</ansi>`, e.Name)
	if e.Zone != `` {
		fmt.Fprintf(&b, `, %s`, e.Zone)
	}
	fmt.Fprintf(&b, "\n  Known: %s (%d %s); %s.\n", e.Tier.Name(), e.Kills, kills(e.Kills), e.Progress())
	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		fmt.Fprintf(&b, `<ansi fg="yellow">%s</ansi>`+"\n", title)
		for _, l := range lines {
			fmt.Fprintf(&b, "  %s\n", l)
		}
	}
	section(`Lore`, e.Lore)
	section(`Defences`, e.Defences)
	section(`Habits and weaknesses`, e.Habits)
	return strings.TrimRight(b.String(), "\n")
}
