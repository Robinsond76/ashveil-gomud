package company

// Phase 49: companion banter. At a camp rest, and now and then after a
// won battle, two or three companions trade a line or two, drawn by their
// archetype, personality, and alignment (internal/banter). It is flavor
// only: nothing here changes a stat, saves anything but the personality
// rolled at recruitment, or advances the clock. A player turns it off with
// `set banter off`.

import (
	"github.com/GoMudEngine/GoMud/internal/modconfig"
	"math/rand"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var _ domain.BanterProvider = (*CompanyModule)(nil)

const (
	// banterWindow is how many of a member's latest lines are not repeated.
	banterWindow = 12
	// Default chances, in percent. Config: BanterBattlePercent,
	// BanterCampStartPercent, BanterRestedPercent.
	defaultBanterBattlePercent    = 33
	defaultBanterCampStartPercent = 100
	defaultBanterRestedPercent    = 50
	defaultBanterSongPercent      = 60 // camp music: a song gets a comment more often than not
	// A member at or under this share of its health at the battle's end
	// makes it a close call; every member over the flawless share makes it
	// a flawless win.
	banterCloseShare    = 0.30
	banterFlawlessShare = 0.90
)

// banterState is the module's banter memory: in memory only, since an
// exchange is flavor and a restart may forget it.
type banterState struct {
	mu     sync.Mutex
	pool   *banter.Pool
	err    error
	loaded bool
	rng    *rand.Rand
	// recent is each leader's companions' latest line ids, oldest first.
	recent map[int]map[int][]string
	last   map[int][]banter.Said
	// fell is each leader's companions who died since the last battle
	// ended (their ids, in order), noted when the death is recorded.
	fell map[int][]int
}

func (b *banterState) poolLocked() *banter.Pool {
	if !b.loaded {
		b.loaded = true
		b.pool, b.err = banter.Load()
		if b.err != nil {
			mudlog.Error("company: banter data", "error", b.err)
		}
	}
	return b.pool
}

func (b *banterState) forget(leader int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.recent, leader)
	delete(b.last, leader)
	delete(b.fell, leader)
}

// noteFall remembers that a companion died, for the battle's end.
func (b *banterState) noteFall(leader, companionID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fell == nil {
		b.fell = map[int][]int{}
	}
	b.fell[leader] = append(b.fell[leader], companionID)
}

func (m *CompanyModule) banterPercent(key string, fallback int) int {
	if m.plug != nil {
		if n, ok := modconfig.Int(m.plug.Config.Get(key)); ok && n >= 0 && n <= 100 {
			return n
		}
	}
	return fallback
}

// rollPersonality gives a new companion its temperament (once).
func (m *CompanyModule) rollPersonality(leaderUserID int, companion domain.Companion) {
	if companion.Personality != "" {
		return
	}
	m.banter.mu.Lock()
	if m.banter.rng == nil {
		m.banter.rng = rand.New(rand.NewSource(rand.Int63()))
	}
	p := banter.Personalities[m.banter.rng.Intn(len(banter.Personalities))]
	m.banter.mu.Unlock()
	if err := m.registry.SetCompanionPersonality(leaderUserID, companion.ID, p); err != nil {
		mudlog.Warn("company: set personality", "leader", leaderUserID, "companion", companion.ID, "error", err)
	}
}

// personalityOf is a companion's personality, derived from its ID when
// none was ever saved.
func personalityOf(c domain.Companion) string {
	if banter.ValidPersonality(c.Personality) {
		return c.Personality
	}
	return banter.PersonalityFor(c.ID)
}

// banterEnabled reports whether the leader wants banter.
func banterEnabled(leaderUserID int) (*users.UserRecord, bool) {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return nil, false
	}
	return user, banter.Enabled(user.GetConfigOption(banter.OptionKey))
}

// banterMember is a companion who may speak.
func (m *CompanyModule) banterMember(c domain.Companion) banter.Member {
	return banter.Member{ID: c.ID, Name: companionName(c), Archetype: c.Archetype, Personality: personalityOf(c), Alignment: m.companionAlignment(c)}
}

// campMembers are the companions with the camp: not dead, away, or fled.
func (m *CompanyModule) campMembers(leaderUserID int) []banter.Member {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	var out []banter.Member
	for _, c := range record.Companions {
		if c.Dead() || c.Separated() || c.PendingReturn || creatures.Is(c.Archetype) { // 38e review: a hound or a golem doesn't talk
			continue
		}
		out = append(out, m.banterMember(c))
	}
	return out
}

// exchange runs one exchange for the leader, remembers it, and returns it.
// Friends and rivals who talk about each other move their bond a point
// (Phase 65).
func (m *CompanyModule) exchange(user *users.UserRecord, members []banter.Member, contexts []string, fallen string, night bool) []banter.Said {
	said := m.exchangeRaw(user, members, contexts, fallen, night)
	m.bondsFromTalk(user.UserId, said)
	return said
}

func (m *CompanyModule) exchangeRaw(user *users.UserRecord, members []banter.Member, contexts []string, fallen string, night bool) []banter.Said {
	if len(members) < 2 {
		return nil
	}
	b := &m.banter
	b.mu.Lock()
	defer b.mu.Unlock()
	pool := b.poolLocked()
	if pool == nil {
		return nil
	}
	if b.rng == nil {
		b.rng = rand.New(rand.NewSource(rand.Int63()))
	}
	recent := map[int]map[string]bool{}
	for id, ids := range b.recent[user.UserId] {
		set := map[string]bool{}
		for _, line := range ids {
			set[line] = true
		}
		recent[id] = set
	}
	said := pool.Exchange(b.rng, banter.Request{Contexts: contexts, Members: members, Leader: user.Character.Name, Fallen: fallen, Night: night, Recent: recent, Bond: m.bondSigns(user.UserId)})
	if len(said) == 0 {
		return nil
	}
	if b.recent == nil {
		b.recent = map[int]map[int][]string{}
		b.last = map[int][]banter.Said{}
	}
	if b.recent[user.UserId] == nil {
		b.recent[user.UserId] = map[int][]string{}
	}
	for _, s := range said {
		ids := append(b.recent[user.UserId][s.Member], s.LineID)
		if len(ids) > banterWindow {
			ids = ids[len(ids)-banterWindow:]
		}
		b.recent[user.UserId][s.Member] = ids
	}
	b.last[user.UserId] = said
	return said
}

func (b *banterState) roll(percent int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.rng == nil {
		b.rng = rand.New(rand.NewSource(rand.Int63()))
	}
	return b.rng.Intn(100) < percent
}

// CampBanter implements company.BanterProvider.
func (m *CompanyModule) CampBanter(leaderUserID int, context string) []banter.Said {
	var percent int
	switch context {
	case banter.CtxCamp:
		percent = m.banterPercent("BanterCampStartPercent", defaultBanterCampStartPercent)
	case banter.CtxRested:
		percent = m.banterPercent("BanterRestedPercent", defaultBanterRestedPercent)
	case banter.CtxSong: // camp music: a comment on the song, at its own chance
		percent = m.banterPercent("BanterSongPercent", defaultBanterSongPercent)
	default:
		return nil
	}
	if context == banter.CtxRested && m.persistenceAvailable() == nil {
		m.bondsCampRest(leaderUserID) // Phase 65: a shared camp, whether or not they talk
	}
	user, on := banterEnabled(leaderUserID)
	if !on || m.persistenceAvailable() != nil || !m.banter.roll(percent) {
		return nil
	}
	return m.exchange(user, m.campMembers(leaderUserID), []string{context}, "", gametime.IsNight())
}

// LastBanter implements company.BanterProvider.
func (m *CompanyModule) LastBanter(leaderUserID int) []banter.Said {
	m.banter.mu.Lock()
	defer m.banter.mu.Unlock()
	return append([]banter.Said(nil), m.banter.last[leaderUserID]...)
}

// battleBanter is the exchange after a won battle, or "" when none is due.
// It runs at the battle's end, before the patch-up heals anyone, so the
// health it reads is how the fight left the company.
func (m *CompanyModule) battleBanter(user *users.UserRecord, outcome string) string {
	// A fall is noticed even when nothing is said.
	fallen := m.newlyFallen(user)
	if outcome != combatstream.OutcomeVictory || !banter.Enabled(user.GetConfigOption(banter.OptionKey)) {
		return ""
	}
	if !m.banter.roll(m.banterPercent("BanterBattlePercent", defaultBanterBattlePercent)) {
		return ""
	}
	views, ok := domain.CompanyMembers(user.UserId)
	if !ok {
		return ""
	}
	record, _ := m.registry.Get(user.UserId)
	byID := map[int]domain.Companion{}
	for _, c := range record.Companions {
		byID[c.ID] = c
	}
	var members []banter.Member
	lowest := 1.0
	note := func(hp, max int) {
		if max > 0 {
			if share := float64(hp) / float64(max); share < lowest {
				lowest = share
			}
		}
	}
	note(user.Character.Health, user.Character.HealthMax.Value)
	for _, v := range views {
		if v.Status != domain.MemberPresent || v.HP < 1 {
			if v.Status == domain.MemberPresent {
				note(0, 1)
			}
			continue
		}
		note(v.HP, v.HPMax)
		if c, ok := byID[v.ID]; ok && !creatures.Is(c.Archetype) { // 38e review: creatures don't talk
			members = append(members, m.banterMember(c))
		}
	}
	contexts := banterContexts(fallen != "", lowest)
	said := m.exchange(user, members, contexts, fallen, gametime.IsNight())
	return banter.Format(said)
}

// newlyFallen is the name of a companion that died since the last battle
// ended and is still dead ("" when none did), and forgets the deaths, so
// each is mourned once. A companion dead from before (one awaiting
// resurrection, or any dead across a restart) is not a new fall.
func (m *CompanyModule) newlyFallen(user *users.UserRecord) string {
	b := &m.banter
	b.mu.Lock()
	ids := b.fell[user.UserId]
	delete(b.fell, user.UserId)
	b.mu.Unlock()
	record, _ := m.registry.Get(user.UserId)
	for _, id := range ids {
		if c, ok := findCompanion(record, id); ok && c.Dead() {
			return banter.ShortName(companionName(c))
		}
	}
	return ""
}

// banterContexts are the contexts to draw from after a won battle, most
// specific first: someone fell, it was a close call (someone at or under
// banterCloseShare of their health), or it was flawless (everyone at or
// over banterFlawlessShare). Anything else is an ordinary win.
func banterContexts(fell bool, lowestShare float64) []string {
	switch {
	case fell:
		return []string{banter.CtxFall, banter.CtxWin}
	case lowestShare <= banterCloseShare:
		return []string{banter.CtxClose, banter.CtxWin}
	case lowestShare >= banterFlawlessShare:
		return []string{banter.CtxFlawless, banter.CtxWin}
	}
	return []string{banter.CtxWin}
}
