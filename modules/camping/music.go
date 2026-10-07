package camping

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/modtimer"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Camp music (help music; spec docs/plans/2026-10-07-camp-music.md). A
// member learns Music in one family and, when a camp rest begins, the
// company plays a short song:
//
//   - The song is planned and locked on the rest when it starts
//     (RestSession.Song), with its cost to the raid and thief roll, so a
//     restart never re-plans it. Nothing in the song runs in battle.
//   - Its effects are granted when the rest finishes, to the leader and the
//     live companions, and only for an unspoiled rest: strings and the
//     ensemble through the Rested grant (grantPendingTiers), winds in the
//     rest's own Fatigue (applyRestRecoveryLocked), drums as a Drumbeat buff
//     (ended by the next battle), voice by fading ailments.
//   - Every player practises: songs played raise the skill at 10, 25 and 50.
//   - The pending song is saved with the pending grant (restedSongs), so
//     breaking camp or resting again before the grant neither drops it nor
//     swaps it.
//   - Gigs (inn gig) reuse the planned song for coin, with a window on the
//     world clock and two real-time limits (camping.GigLog).

// musicTeacherTag marks a room where a music teacher teaches level 1.
const musicTeacherTag = "music-teacher"

// musicMember is a company member at the camp with the skill they hold.
type musicMember struct {
	target prepTarget
	skill  camping.MusicSkill
	has    bool
}

func (m *CampingModule) skillOf(leaderUserID int, key string) (camping.MusicSkill, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	skill, ok := m.musicSkills[leaderUserID][key]
	return skill, ok
}

// musicMembers lists the leader and the live companions in the leader's
// room with their Music skills. Call it before taking m.mu is not needed:
// it takes m.mu itself, and calls the company module only through
// prepTargets (which doesn't hold the lock).
func (m *CampingModule) musicMembers(user *users.UserRecord) []musicMember {
	targets, _ := m.prepTargets(user)
	out := make([]musicMember, 0, len(targets))
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range targets {
		skill, ok := m.musicSkills[user.UserId][t.key]
		out = append(out, musicMember{target: t, skill: skill, has: ok})
	}
	return out
}

// carriedInstruments is the best instrument the company carries in each
// instrument family (read before m.mu: it calls the company module).
func (m *CampingModule) carriedInstruments(leaderUserID int) map[camping.Family]camping.Instrument {
	out := map[camping.Family]camping.Instrument{}
	for _, f := range []camping.Family{camping.FamilyStrings, camping.FamilyWinds, camping.FamilyDrums} {
		if in, ok := camping.BestInstrument(f, func(id int) bool { return m.gearCount(leaderUserID, id) > 0 }); ok {
			out[f] = in
		}
	}
	return out
}

// playersOf turns the members who can play into song players: a member
// plays when they have the skill and, but for the voice, the company
// carries an instrument of their family.
func playersOf(members []musicMember, instruments map[camping.Family]camping.Instrument) []camping.Player {
	var out []camping.Player
	for _, mm := range members {
		if !mm.has {
			continue
		}
		p := camping.Player{Key: mm.target.key, Name: mm.target.name, Family: mm.skill.Family, Level: mm.skill.Level()}
		if mm.skill.Family != camping.FamilyVoice {
			in, ok := instruments[mm.skill.Family]
			if !ok {
				continue
			}
			p.Tier, p.Quirk = in.Tier, in.Quirk
		}
		out = append(out, p)
	}
	return out
}

func (m *CampingModule) isMusicOff(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.musicOff[leaderUserID]
}

// planSong is the song the company would play now at room. tent is whether
// a tent is pitched (it keeps the weather off the instruments). It reads
// the company's packs, so call it before taking m.mu.
func (m *CampingModule) planSong(user *users.UserRecord, room *rooms.Room, tent bool) camping.Song {
	if m.isMusicOff(user.UserId) {
		return camping.Song{}
	}
	members := m.musicMembers(user)
	players := playersOf(members, m.carriedInstruments(user.UserId))
	wet := false
	if room != nil && !tent {
		if cond, ok := m.currentWeather(room.Zone); ok {
			wet = camping.Wet(cond.Name)
		}
	}
	return camping.PlanSong(players, wet)
}

// --- the camp song: start, text ---

// songLine is what one family's play looks like.
func songLine(p camping.FamilyPlay) string {
	who := p.Player
	if who == "" {
		who = "Someone"
	}
	switch p.Family {
	case camping.FamilyStrings:
		return fmt.Sprintf("%s picks out a tune on the %s.", who, p.Instrument)
	case camping.FamilyWinds:
		return fmt.Sprintf("%s lifts the %s and a thin melody rises with the smoke.", who, p.Instrument)
	case camping.FamilyDrums:
		return fmt.Sprintf("%s taps a slow beat on the %s.", who, p.Instrument)
	}
	return fmt.Sprintf("%s sings, low and steady.", who)
}

// songText is the song's report: a few short lines, then what it will do
// when the rest ends.
func songText(song camping.Song) string {
	lines := make([]string, 0, len(song.Plays)+2)
	for _, p := range song.Plays {
		lines = append(lines, songLine(p))
	}
	lines = append(lines, "The song will lend the rest: "+strings.Join(effectsShort(song), "; ")+".")
	return strings.Join(lines, "\n")
}

// songRoomText is what the others in the room see and hear.
func songRoomText(song camping.Song) string {
	lines := make([]string, 0, len(song.Plays))
	for _, p := range song.Plays {
		lines = append(lines, songLine(p))
	}
	return strings.Join(lines, "\n")
}

func effectsShort(song camping.Song) []string {
	var out []string
	for _, p := range song.Plays {
		switch p.Family {
		case camping.FamilyStrings:
			out = append(out, fmt.Sprintf("Rested lasts %d%% longer", song.RestedPct()))
		case camping.FamilyWinds:
			out = append(out, fmt.Sprintf("+%d Fatigue", song.FatigueBonus()))
		case camping.FamilyDrums:
			out = append(out, fmt.Sprintf("+%d speed in the next battle", song.DrumSpeed()))
		case camping.FamilyVoice:
			out = append(out, fmt.Sprintf("ailments fade %d%% faster", song.AilmentPct()))
		}
	}
	if song.Ensemble() {
		out = append(out, "the sleepers wake Well Rested")
	}
	return out
}

// startSongText is added to a rest's start report; it also shows the room
// the performance and runs the company's banter for it. It runs after the
// rest is saved, outside m.mu.
func (m *CampingModule) startSongText(user *users.UserRecord, room *rooms.Room, song camping.Song) string {
	if song.Empty() {
		return ""
	}
	if room != nil {
		room.SendText(songRoomText(song), user.UserId)
	}
	text := songText(song)
	if m.onSong != nil {
		m.onSong(user.UserId, song)
	} else if said := company.CampBanter(user.UserId, banter.CtxSong); len(said) > 0 {
		text += "\n\n" + banter.Format(said)
	}
	return text
}

// --- the song's effects ---

// pendingSong is the song saved with the leader's pending Rested grant.
func (m *CampingModule) pendingSong(leaderUserID int) (camping.Song, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	song, ok := m.restedSongs[leaderUserID]
	return song, ok && !song.Empty()
}

// restBuffFor is the tier and length of the Rested buff a camp rest grants:
// the tent sets it (a large tent's is Well Rested, a camouflaged tent's is
// shorter), the ensemble upgrades the sleepers' to Well Rested as the large
// tent does (they do not stack: either gives the one upgrade), and the
// strings stretch it.
func restBuffFor(settings innSettings, tier camping.Tier, tent *camping.Tent, song *camping.Song) (camping.Tier, time.Duration) {
	buffTier, duration := tier, settings.tierDuration(tier)
	if tier == camping.TierRested {
		tentPct := 100
		if tent != nil {
			tentPct = max(tent.RestedPct, 1)
			if tent.WellRested {
				buffTier = camping.TierWellRested
			}
		}
		if song != nil && song.Ensemble() {
			buffTier = camping.TierWellRested
		}
		duration = settings.tierDuration(buffTier)
		if tent == nil || !tent.WellRested {
			duration = duration * time.Duration(tentPct) / 100
		}
	}
	if song != nil && tier == camping.TierRested {
		duration = duration * time.Duration(100+song.RestedPct()) / 100
	}
	return buffTier, duration
}

// grantDrums gives the leader and the live companions a Drumbeat buff that
// lasts as long as their rest buff or until their next battle ends.
func (m *CampingModule) grantDrums(user *users.UserRecord, live map[int]*characters.Character, speed, rounds int) bool {
	id := camping.DrumBuffFor(speed)
	if id == 0 {
		return false
	}
	given := false
	grant := func(c *characters.Character) {
		if c == nil || c.Health < 1 {
			return
		}
		if err := m.addBuff(c, id, rounds); err != nil {
			mudlog.Warn("camping: grant drumbeat", "buff", id, "error", err)
			return
		}
		given = true
	}
	grant(user.Character)
	for _, cid := range sortedIDs(live) {
		grant(live[cid])
	}
	return given
}

// onBattleEndedMusic ends the Drumbeat after the first battle.
func (m *CampingModule) onBattleEndedMusic(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.BattleEnded)
	if !ok {
		return events.Continue
	}
	user := m.userByID(evt.UserId)
	if user == nil || user.Character == nil {
		return events.Continue
	}
	m.endDrums(user.Character)
	live, _ := m.companions(evt.UserId)
	for _, id := range sortedIDs(live) {
		m.endDrums(live[id])
	}
	return events.Continue
}

func (m *CampingModule) endDrums(c *characters.Character) {
	for _, id := range camping.DrumBuffIDs {
		if m.holdsBuff(c, id) {
			m.dropBuff(c, id)
		}
	}
}

// fadeAilments shortens each member's ailments by the voice's percent, one
// line for the company.
func (m *CampingModule) fadeAilments(user *users.UserRecord, pct int) string {
	if pct <= 0 {
		return ""
	}
	faded := 0
	for _, n := range survival.CompanyNeeds(user.UserId) {
		for _, a := range survival.ActiveAilments(n.Needs) {
			left := survival.AilmentBattles(n.Needs, a.Kind)
			after := camping.FadedBattles(left, pct)
			if after >= left {
				continue
			}
			if after <= 0 {
				if ok, err := survival.CureAilment(user.UserId, n.Key, a.Kind); err == nil && ok {
					faded++
				}
				continue
			}
			if ok, err := survival.FadeAilment(user.UserId, n.Key, a.Kind, after); err == nil && ok {
				faded++
			}
		}
	}
	if faded == 0 {
		return ""
	}
	return fmt.Sprintf("The company's singing eases %s.", plural(faded, "ailment"))
}

// practiceSong counts one song for each player and says who gained a
// level. It saves before it reports.
func (m *CampingModule) practiceSong(leaderUserID int, players []camping.SongPlayer) []string {
	if len(players) == 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.musicSkills[leaderUserID] == nil {
		return nil
	}
	before := make(map[string]camping.MusicSkill, len(players))
	var lines []string
	for _, p := range players {
		skill, ok := m.musicSkills[leaderUserID][p.Key]
		if !ok || skill.Family != p.Family {
			continue // they changed family since the song
		}
		before[p.Key] = skill
		next := skill.Practised(1)
		m.musicSkills[leaderUserID][p.Key] = next
		if next.Level() > skill.Level() {
			who := p.Name
			if who == "" {
				who = "A musician"
			}
			lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">%s's %s playing reaches level %d.</ansi>`, who, strings.ToLower(camping.FamilyName(p.Family)), next.Level()))
		}
	}
	if len(before) == 0 {
		return nil
	}
	if err := m.saveLocked(); err != nil {
		for key, skill := range before {
			m.musicSkills[leaderUserID][key] = skill
		}
		mudlog.Warn("camping: practice save", "leader", leaderUserID, "error", err)
		return nil
	}
	return lines
}

// onSongGranted runs the song's effects that the grant pass owns: the
// drums, the voice and practice. It returns the lines to show the leader.
func (m *CampingModule) onSongGranted(user *users.UserRecord, live map[int]*characters.Character, song camping.Song, rounds int) []string {
	var lines []string
	if speed := song.DrumSpeed(); speed > 0 && m.grantDrums(user, live, speed, rounds) {
		lines = append(lines, fmt.Sprintf("The drums are still in your blood: +%d speed in your next battle.", speed))
	}
	if line := m.fadeAilments(user, song.AilmentPct()); line != "" {
		lines = append(lines, line)
	}
	return append(lines, m.practiceSong(user.UserId, song.Players)...)
}

// --- commands ---

const musicUsage = "Usage: music | music on | music off | music learn [family] [member] [confirm] | music craft [instrument] | camp music [on|off]"

func (m *CampingModule) musicCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	user.SendText(m.musicArgs(user, room, args))
	return true, nil
}

func (m *CampingModule) musicArgs(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if len(args) == 0 || args[0] == "status" {
		return m.musicStatus(user, room)
	}
	switch args[0] {
	case "on", "off":
		return m.setMusicOn(user, args[0] == "on")
	case "learn":
		return m.musicLearn(user, room, args[1:])
	case "craft", "make":
		return m.musicCraft(user, room, args[1:])
	}
	return musicUsage
}

// setMusicOn silences or restores the camp song.
func (m *CampingModule) setMusicOn(user *users.UserRecord, on bool) string {
	m.mu.Lock()
	prev, had := m.musicOff[user.UserId]
	if on {
		delete(m.musicOff, user.UserId)
	} else {
		m.musicOff[user.UserId] = true
	}
	if err := m.saveLocked(); err != nil {
		if had {
			m.musicOff[user.UserId] = prev
		} else {
			delete(m.musicOff, user.UserId)
		}
		m.mu.Unlock()
		return err.Error()
	}
	m.mu.Unlock()
	if on {
		return "Your company will play at camp again."
	}
	return "Your company will rest in silence. (music on brings the song back.)"
}

// musicStatus is `music`: who plays what, the families covered, and what
// the next song would give.
func (m *CampingModule) musicStatus(user *users.UserRecord, room *rooms.Room) string {
	members := m.musicMembers(user)
	instruments := m.carriedInstruments(user.UserId)
	off := m.isMusicOff(user.UserId)
	lines := []string{`<ansi fg="yellow-bold">Your company's music</ansi> (help music)`}
	for _, mm := range members {
		switch {
		case !mm.has:
			lines = append(lines, fmt.Sprintf("  %s: no music yet.", mm.target.name))
		case mm.skill.Family == camping.FamilyVoice:
			lines = append(lines, fmt.Sprintf("  %s: %s (sings).", mm.target.name, mm.skill.Label()))
		default:
			if in, ok := instruments[mm.skill.Family]; ok {
				lines = append(lines, fmt.Sprintf("  %s: %s, on the %s.", mm.target.name, mm.skill.Label(), in.Name))
			} else {
				lines = append(lines, fmt.Sprintf("  %s: %s, but the company carries no instrument for it.", mm.target.name, mm.skill.Label()))
			}
		}
	}
	carried := make([]string, 0, 3)
	for _, f := range []camping.Family{camping.FamilyStrings, camping.FamilyWinds, camping.FamilyDrums} {
		if in, ok := instruments[f]; ok {
			carried = append(carried, fmt.Sprintf("%s (%s)", in.Name, camping.InstrumentTierName(in.Tier)))
		}
	}
	if len(carried) == 0 {
		lines = append(lines, "Instruments carried: none (help instruments).")
	} else {
		lines = append(lines, "Instruments carried: "+strings.Join(carried, ", ")+".")
	}
	song := camping.PlanSong(playersOf(members, instruments), false)
	switch {
	case off:
		lines = append(lines, "The camp song is off (music on).")
	case song.Empty():
		lines = append(lines, "Nobody can play yet: learn a family from a music teacher (music learn) and carry its instrument.")
	default:
		lines = append(lines, fmt.Sprintf("Families covered: %s. The next camp rest's song:", song.Covered()))
		for _, l := range song.Effects() {
			lines = append(lines, "  "+l)
		}
		if room != nil && m.roomWet(room) {
			lines = append(lines, "  (Rain or snow halves strings and winds at camp unless a tent is pitched.)")
		}
	}
	if room != nil && room.HasTag(musicTeacherTag) {
		lines = append(lines, fmt.Sprintf("A music teacher is here: music learn [family] [member] costs %d gold.", camping.MusicTeachPrice))
	}
	return strings.Join(lines, "\n")
}

func (m *CampingModule) roomWet(room *rooms.Room) bool {
	cond, ok := m.currentWeather(room.Zone)
	return ok && camping.Wet(cond.Name)
}

// musicLearn teaches level 1 of a family to a member, for gold, at a music
// teacher. Changing family drops a member back to level 1 of the new one
// and is paid again, so it needs "confirm" when it would lose practice.
func (m *CampingModule) musicLearn(user *users.UserRecord, room *rooms.Room, args []string) string {
	if room == nil || !room.HasTag(musicTeacherTag) {
		return "There is no music teacher here. (Alderbrook's green has one; help music.)"
	}
	confirm := false
	if n := len(args); n > 0 && args[n-1] == "confirm" {
		confirm, args = true, args[:n-1]
	}
	if len(args) == 0 {
		lines := []string{fmt.Sprintf("The teacher offers level 1 of any family for %d gold:", camping.MusicTeachPrice)}
		for _, info := range camping.Families {
			lines = append(lines, fmt.Sprintf("  %s: needs %s.", info.Name, info.Needs))
		}
		lines = append(lines, "music learn [family] [member]  (a member's name; none means you). Changing family costs the practice you have.")
		return strings.Join(lines, "\n")
	}
	family, ok := camping.ParseFamily(args[0])
	if !ok {
		return "The teacher knows strings, winds, drums and voice. " + musicUsage
	}
	targets, _ := m.prepTargets(user)
	who, refusal := resolveTargets(strings.Join(args[1:], " "), targets)
	if refusal != "" {
		return refusal
	}
	if len(who) != 1 {
		return "Teach one member at a time."
	}
	t := who[0]
	current, has := m.skillOf(user.UserId, t.key)
	switch {
	case has && current.Family == family:
		return fmt.Sprintf("%s already plays %s: %s. Practice at camp is the only teacher now.", t.name, strings.ToLower(camping.FamilyName(family)), current.Label())
	case has && current.Songs > 0 && !confirm:
		return fmt.Sprintf("%s plays %s at level %d. Learning %s means starting over at level 1 and paying %d gold. Say \"music learn %s %s confirm\" to go ahead.",
			t.name, strings.ToLower(camping.FamilyName(current.Family)), current.Level(), strings.ToLower(camping.FamilyName(family)), camping.MusicTeachPrice, family, memberWord(t, user))
	}
	if user.Character.Gold < camping.MusicTeachPrice {
		return fmt.Sprintf("The teacher asks %d gold, and you have %d.", camping.MusicTeachPrice, user.Character.Gold)
	}
	m.mu.Lock()
	if m.musicSkills[user.UserId] == nil {
		m.musicSkills[user.UserId] = map[string]camping.MusicSkill{}
	}
	prev, hadPrev := m.musicSkills[user.UserId][t.key]
	m.musicSkills[user.UserId][t.key] = camping.MusicSkill{Family: family}
	if err := m.saveLocked(); err != nil {
		if hadPrev {
			m.musicSkills[user.UserId][t.key] = prev
		} else {
			delete(m.musicSkills[user.UserId], t.key)
		}
		m.mu.Unlock()
		return err.Error()
	}
	m.mu.Unlock()
	user.Character.Gold -= camping.MusicTeachPrice
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -camping.MusicTeachPrice})
	needs := ""
	for _, info := range camping.Families {
		if info.Family == family && family != camping.FamilyVoice {
			needs = " They need " + info.Needs + " in the company's packs to play it (help instruments)."
		}
	}
	return fmt.Sprintf("You pay %d gold. The teacher drills %s in %s until the first tune comes.%s", camping.MusicTeachPrice, t.name, strings.ToLower(camping.FamilyName(family)), needs)
}

func memberWord(t prepTarget, user *users.UserRecord) string {
	if t.key == string(survival.LeaderMemberKey) {
		return "me"
	}
	return strings.ToLower(t.name)
}

// --- crafting ---

// instrumentRecipe is what a company makes an instrument from. Crude ones
// are common knowledge and made from gathered goods; a fine one needs its
// recipe page (a recipe book entry, cookbook.Learn) and a bought common
// instrument and planks.
type instrumentRecipe struct {
	Output int
	Inputs []int
}

const (
	ashwoodPlankItemID = 223
	elkAntlersItemID   = 201
	boarTusksItemID    = 202
	wolfHideItemID     = 28
	rawGameMeatItemID  = 29
)

var instrumentRecipes = []instrumentRecipe{
	{3200, []int{elkAntlersItemID, rawGameMeatItemID}},
	{3203, []int{boarTusksItemID, boarTusksItemID}},
	{3206, []int{wolfHideItemID, wolfHideItemID, elkAntlersItemID}},
	{3202, []int{3201, ashwoodPlankItemID, ashwoodPlankItemID}},
	{3205, []int{3204, ashwoodPlankItemID, ashwoodPlankItemID}},
	{3208, []int{3207, ashwoodPlankItemID, ashwoodPlankItemID}},
}

func (r instrumentRecipe) tier() int {
	in, _ := camping.InstrumentOf(r.Output)
	return in.Tier
}

// knownTo reports whether the character can make it: crude is common
// knowledge, fine needs a learned page.
func (r instrumentRecipe) knownTo(c *characters.Character) bool {
	if r.tier() <= camping.InstrumentCrude {
		return true
	}
	return cookbook.Knows(c, cookbook.Recipe{Output: r.Output, Inputs: r.Inputs, MinLevel: cookbook.BasicLevel + 1})
}

func recipeNeeds(inputs []int) string {
	counts := map[int]int{}
	var order []int
	for _, id := range inputs {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}
	parts := make([]string, 0, len(order))
	for _, id := range order {
		if counts[id] > 1 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[id], itemName(id)))
		} else {
			parts = append(parts, itemName(id))
		}
	}
	return strings.Join(parts, ", ")
}

// musicCraft lists what the leader can make, or makes one instrument from
// the company's packs and cargo at the leader's camp.
func (m *CampingModule) musicCraft(user *users.UserRecord, room *rooms.Room, args []string) string {
	if len(args) == 0 {
		lines := []string{"Instruments you can make at your camp (music craft [name]):"}
		for _, r := range instrumentRecipes {
			if !r.knownTo(user.Character) {
				continue
			}
			lines = append(lines, fmt.Sprintf("  %s (%s): %s.", itemName(r.Output), camping.InstrumentTierName(r.tier()), recipeNeeds(r.Inputs)))
		}
		lines = append(lines, "Fine instruments are made from a bought common one and ashwood planks, once you have read their recipe page (help instruments).")
		return strings.Join(lines, "\n")
	}
	if m.inBattle != nil && m.inBattle(user.UserId) {
		return "You can't work at an instrument in the middle of a fight."
	}
	word := strings.Join(args, " ")
	var chosen *instrumentRecipe
	for i := range instrumentRecipes {
		r := &instrumentRecipes[i]
		name := strings.ToLower(itemName(r.Output))
		if name == word || strings.Contains(name, word) {
			chosen = r
			break
		}
	}
	if chosen == nil {
		return fmt.Sprintf("There is no instrument called %q to make. See music craft.", word)
	}
	if !chosen.knownTo(user.Character) {
		return fmt.Sprintf("You don't know how to make the %s: it needs a recipe page (help instruments).", itemName(chosen.Output))
	}
	m.mu.Lock()
	camp, has := m.camps[user.UserId]
	m.mu.Unlock()
	if !has || room == nil || camp.RoomID != room.RoomId {
		return "Instruments are made at your camp, by the fire: make camp first."
	}
	need := map[int]int{}
	for _, id := range chosen.Inputs {
		need[id]++
	}
	for _, id := range sortedKeys(need) {
		if have := m.gearCount(user.UserId, id); have < need[id] {
			return fmt.Sprintf("You need %s for the %s, and the company has %d.", recipeNeeds(chosen.Inputs), itemName(chosen.Output), have)
		}
	}
	inGrams := 0
	for _, id := range chosen.Inputs {
		if spec := items.GetItemSpec(id); spec != nil {
			inGrams += spec.Weight
		}
	}
	outGrams := 0
	if spec := items.GetItemSpec(chosen.Output); spec != nil {
		outGrams = spec.Weight
	}
	if text, refuse := encumbrance.TooMuchToCarry(user.UserId, outGrams-inGrams); refuse {
		return text
	}
	for _, id := range chosen.Inputs {
		if !m.spendOne(user.UserId, id) {
			return "The materials couldn't be gathered right now."
		}
	}
	made := items.New(chosen.Output)
	if user.Character.CompanyCargo {
		if err := encumbrance.DepositCargo(user.UserId, "", []encumbrance.CargoStack{{ItemId: chosen.Output, Count: 1}}); err != nil {
			room.AddItem(made, false)
			return fmt.Sprintf(`You make a <ansi fg="itemname">%s</ansi> and set it down by the fire.`, itemName(chosen.Output))
		}
		return fmt.Sprintf(`You make a <ansi fg="itemname">%s</ansi>; it goes into the company's cargo.`, itemName(chosen.Output))
	}
	if !user.Character.StoreItem(made) {
		room.AddItem(made, false)
		return fmt.Sprintf(`You make a <ansi fg="itemname">%s</ansi> and set it down by the fire.`, itemName(chosen.Output))
	}
	return fmt.Sprintf(`You make a <ansi fg="itemname">%s</ansi>.`, itemName(chosen.Output))
}

// --- views ---

// MusicLabelOf implements camping.MusicProvider: a member's Music in a few
// words for the Company panel ("" with none).
func (m *CampingModule) MusicLabelOf(leaderUserID int, memberKey string) string {
	skill, ok := m.skillOf(leaderUserID, memberKey)
	if !ok {
		return ""
	}
	return skill.Label()
}

// musicState is the Camp tab's Music block for the leader's company at
// the room they stand in.
func (m *CampingModule) musicState(leaderUserID int, room *rooms.Room) camping.MusicState {
	user := m.userByID(leaderUserID)
	if user == nil || user.Character == nil {
		return camping.MusicState{}
	}
	members := m.musicMembers(user)
	instruments := m.carriedInstruments(leaderUserID)
	state := camping.MusicState{Known: true, Off: m.isMusicOff(leaderUserID)}
	for _, mm := range members {
		row := camping.MusicRow{Key: mm.target.key, Name: mm.target.name}
		if mm.has {
			row.Family, row.Level = string(mm.skill.Family), mm.skill.Level()
			row.Label = mm.skill.Label()
			if mm.skill.Family != camping.FamilyVoice {
				if in, ok := instruments[mm.skill.Family]; ok {
					row.Instrument = in.Name
				}
			}
		}
		state.Players = append(state.Players, row)
	}
	wet := room != nil && m.roomWet(room)
	song := camping.PlanSong(playersOf(members, instruments), wet && !m.tentPitched(leaderUserID))
	state.Covered = song.Covered()
	if !state.Off {
		state.Effects = song.Effects()
	}
	state.Teacher = room != nil && room.HasTag(musicTeacherTag)
	return state
}

func (m *CampingModule) tentPitched(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	camp, ok := m.camps[leaderUserID]
	return ok && camp.Tent
}

// --- inn gigs ---

func (m *CampingModule) clockNow() (int, uint64) {
	if m.worldClock != nil {
		return m.worldClock()
	}
	d := gametime.GetDate()
	return d.Hour24, d.DayNumber
}

func (m *CampingModule) bandHigh(zone string) int {
	if m.zoneBand != nil {
		return m.zoneBand(zone)
	}
	if cfg := rooms.GetZoneConfig(zone); cfg != nil && cfg.Encounters.Band.Valid() {
		return cfg.Encounters.Band.High
	}
	return 0
}

// gigNotice is the inn's gig board for this company: the window, what the
// company can field, and what it would pay. It reads state only.
func (m *CampingModule) gigNotice(user *users.UserRecord, room *rooms.Room) (camping.GigNotice, camping.Song) {
	members := m.musicMembers(user)
	song := camping.PlanSong(playersOf(members, m.carriedInstruments(user.UserId)), false)
	hour, day := m.clockNow()
	m.mu.Lock()
	log := m.gigLogs[user.UserId]
	m.mu.Unlock()
	notice := camping.GigNotice{Window: camping.GigWindowText(), Open: camping.InGigWindow(hour)}
	notice.Reason = log.GigRefusal(room.RoomId, hour, day, m.clock(), song)
	notice.Ready = notice.Reason == ""
	if !song.Empty() {
		notice.Pay = camping.GigPay(m.bandHigh(room.Zone), song)
	}
	notice.Families = len(song.Plays)
	return notice, song
}

// gigStatus is `inn gig status`: the notice board.
func (m *CampingModule) gigStatus(user *users.UserRecord, room *rooms.Room) string {
	if room == nil || !room.HasTag(m.innSettings().RoomTag) {
		return "There is no inn here."
	}
	notice, _ := m.gigNotice(user, room)
	lines := []string{fmt.Sprintf("On the notice board: musicians wanted, %s each evening (help gigs).", notice.Window)}
	switch {
	case notice.Ready:
		lines = append(lines, fmt.Sprintf("Your company can play now: %d families, about %d gold. Say \"inn gig\".", notice.Families, notice.Pay))
	default:
		lines = append(lines, notice.Reason)
		if notice.Families >= camping.GigMinFamilies {
			lines = append(lines, fmt.Sprintf("When it can, your %d families would earn about %d gold.", notice.Families, notice.Pay))
		}
	}
	return strings.Join(lines, "\n")
}

// innGig starts a gig: the company plays for the room until it ends, and
// the pay is credited once, on the game loop.
func (m *CampingModule) innGig(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if room == nil || !room.HasTag(m.innSettings().RoomTag) {
		return "There is no inn here."
	}
	if m.isTravelling(user.UserId) {
		return "You can't take the stage while travelling."
	}
	if m.inBattle != nil && m.inBattle(user.UserId) {
		return "You can't play in the middle of a fight."
	}
	notice, song := m.gigNotice(user, room)
	if !notice.Ready {
		return notice.Reason
	}
	_, day := m.clockNow()
	m.mu.Lock()
	defer m.mu.Unlock()
	if camp, ok := m.camps[user.UserId]; ok && camp.Rest != nil && camp.Rest.State == camping.Resting {
		return "You are resting at camp."
	}
	if stay, ok := m.stays[user.UserId]; ok && stay.Resting() {
		return "You are resting at the inn."
	}
	log := m.gigLogs[user.UserId]
	if log.Current != nil && !log.Current.Settled {
		return "Your company is already playing."
	}
	gig := camping.Gig{RoomID: room.RoomId, StartedAtUTC: m.clock().UTC(), Pay: notice.Pay, Song: song, Day: day}
	prev, had := m.gigLogs[user.UserId]
	m.gigLogs[user.UserId] = log.Started(gig)
	if err := m.saveLocked(); err != nil {
		if had {
			m.gigLogs[user.UserId] = prev
		} else {
			delete(m.gigLogs, user.UserId)
		}
		return err.Error()
	}
	m.scheduleGigLocked(user.UserId, gig)
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s company takes the floor.`+"\n%s", user.Character.Name, songRoomText(song)), user.UserId)
	return fmt.Sprintf("Your company takes the floor at %s. (%s)\n%s", roomTitle(room.RoomId), camping.GigDuration, songRoomText(song))
}

func (m *CampingModule) scheduleGigLocked(leaderUserID int, gig camping.Gig) {
	if gig.Done {
		modtimer.Stop(m.gigTimers, leaderUserID)
		return
	}
	m.gigGeneration = modtimer.Arm(m.gigTimers, m.gigGeneration, leaderUserID, m.scheduler, gig.Remaining(m.clock()), func(generation uint64) {
		m.onGigTimer(leaderUserID, generation)
	})
}

// onGigTimer runs off the game loop: it only marks the gig finished and
// saves. The pay is credited on the game loop (settleGigs).
func (m *CampingModule) onGigTimer(leaderUserID int, generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !modtimer.Claim(m.gigTimers, m.gigGeneration, leaderUserID, generation) {
		return
	}
	if err := m.syncGigLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: gig timer sync", "leader", leaderUserID, "error", err)
	}
}

// syncGigLocked marks a due gig finished.
func (m *CampingModule) syncGigLocked(leaderUserID int) error {
	log, ok := m.gigLogs[leaderUserID]
	if !ok || log.Current == nil || log.Current.Done || !log.Current.Due(m.clock()) {
		return nil
	}
	done := *log.Current
	done.Done = true
	next := log
	next.Current = &done
	m.gigLogs[leaderUserID] = next
	if err := m.saveLocked(); err != nil {
		m.gigLogs[leaderUserID] = log
		return err
	}
	modtimer.Stop(m.gigTimers, leaderUserID)
	return nil
}

// recoverGigsLocked reschedules or finishes the gigs found on load.
func (m *CampingModule) recoverGigsLocked() {
	for leaderUserID, log := range m.gigLogs {
		if log.Current == nil || log.Current.Done {
			continue
		}
		if err := m.syncGigLocked(leaderUserID); err != nil {
			mudlog.Warn("camping: recovery gig sync", "leader", leaderUserID, "error", err)
			continue
		}
		if cur := m.gigLogs[leaderUserID].Current; cur != nil && !cur.Done {
			m.scheduleGigLocked(leaderUserID, *cur)
		}
	}
}

// gigBlock reports a gig in progress, for the movement block.
func (m *CampingModule) gigBlockLocked(leaderUserID int) (bool, string) {
	log, ok := m.gigLogs[leaderUserID]
	if !ok || log.Current == nil || log.Current.Settled {
		return false, ""
	}
	if log.Current.Done {
		return true, "Your company is finishing its gig. Give it a moment."
	}
	return true, fmt.Sprintf("Your company is playing at %s (%s remaining). The crowd won't let you leave mid-song.", roomTitle(log.Current.RoomID), log.Current.Remaining(m.clock()).Round(time.Second))
}

// settleGigs (game loop) pays each finished gig once to its online leader:
// it saves the gig as settled, with the evening and the cooldown spent,
// before crediting the gold, so a restart can never pay it twice.
func (m *CampingModule) settleGigs() {
	m.mu.Lock()
	var leaders []int
	for leaderUserID, log := range m.gigLogs {
		if log.Current != nil && log.Current.Done && !log.Current.Settled {
			leaders = append(leaders, leaderUserID)
		}
	}
	m.mu.Unlock()
	sort.Ints(leaders)
	for _, leaderUserID := range leaders {
		user := m.userByID(leaderUserID)
		if user == nil || user.Character == nil {
			continue // owed until the leader is back online
		}
		m.mu.Lock()
		log := m.gigLogs[leaderUserID]
		if log.Current == nil || !log.Current.Done || log.Current.Settled {
			m.mu.Unlock()
			continue
		}
		gig := *log.Current
		m.gigLogs[leaderUserID] = log.Paid(m.clock())
		if err := m.saveLocked(); err != nil {
			m.gigLogs[leaderUserID] = log
			m.mu.Unlock()
			mudlog.Warn("camping: settle gig", "leader", leaderUserID, "error", err)
			continue
		}
		m.mu.Unlock()
		user.Character.Gold += gig.Pay
		events.AddToQueue(events.EquipmentChange{UserId: leaderUserID, GoldChange: gig.Pay})
		text := fmt.Sprintf("%s\nThe hat comes back with %d gold.", camping.CrowdLine(gig.Song), gig.Pay)
		lines := m.practiceSong(leaderUserID, gig.Song.Players)
		user.SendText(strings.Join(append([]string{text}, lines...), "\n"))
		if room := rooms.LoadRoom(gig.RoomID); room != nil {
			room.SendText(camping.CrowdLine(gig.Song), leaderUserID)
		}
	}
}
