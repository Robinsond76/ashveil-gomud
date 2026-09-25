package companyview

// Phase 26a prompt tokens. The engine builds prompts on connection
// goroutines as well as on the game loop, outside the world lock, and the
// company module has no mutex, so a prompt never reads game state. The
// token values are worked out on the game loop (Refresh, every NewRound and
// after every command) and kept in a mutex-guarded cache that the
// OnBuildPrompt handler only reads.

import (
	"strconv"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Tokens are the prompt tokens this package fills. An unknown value renders
// as nothing, so a token never shows literally.
var Tokens = []string{
	"{hunger}", "{thirst}", "{fatigue}",
	"{hungerv}", "{thirstv}", "{fatiguev}",
	"{light}", "{warmth}", "{load}", "{company}",
	"{activity}", "{warn}",
}

var tokenSet = func() map[string]bool {
	set := map[string]bool{}
	for _, t := range Tokens {
		set[t] = true
	}
	return set
}()

var (
	cacheMu sync.RWMutex
	cache   = map[int]map[string]string{}
)

func needValue(n Need) string {
	if !n.Known {
		return ""
	}
	return strconv.Itoa(n.Value)
}

func needLabel(n Need) string {
	if !n.Known {
		return ""
	}
	return n.Label
}

// PromptValues are a summary's token values. {warn} and {activity} lead
// with a space when they have anything to say, so they can follow other
// text directly; the rest are bare.
func PromptValues(s Summary) map[string]string {
	values := map[string]string{
		"{hunger}":   needLabel(s.Leader.Hunger),
		"{thirst}":   needLabel(s.Leader.Thirst),
		"{fatigue}":  needLabel(s.Leader.Fatigue),
		"{hungerv}":  needValue(s.Leader.Hunger),
		"{thirstv}":  needValue(s.Leader.Thirst),
		"{fatiguev}": needValue(s.Leader.Fatigue),
		"{warmth}":   s.Leader.Warmth,
		"{load}":     s.LoadLabel,
	}
	if s.CompanyKnown {
		values["{company}"] = CompanyCount(s.Alive, s.Dead)
	}
	if s.LightKnown {
		values["{light}"] = LightLabel(s.Light)
	}
	if label := s.Activity.Label(); label != "" {
		values["{activity}"] = ` <ansi fg="cyan">` + label + `</ansi>`
	}
	if warn := WarnCluster(s.WarnWords()); warn != "" {
		values["{warn}"] = ` <ansi fg="yellow">` + warn[1:] + `</ansi>`
	}
	return values
}

// Refresh recomputes a user's prompt values. Call it on the game loop.
func Refresh(user *users.UserRecord) {
	if user == nil || user.Character == nil {
		return
	}
	values := PromptValues(For(user))
	cacheMu.Lock()
	cache[user.UserId] = values
	cacheMu.Unlock()
}

// RefreshUser is Refresh by user ID; it does nothing for a user who isn't
// online.
func RefreshUser(userID int) {
	Refresh(users.GetByUserId(userID))
}

// refreshAll recomputes every online user's values and drops the rest.
func refreshAll() {
	online := users.GetAllActiveUsers()
	live := make(map[int]bool, len(online))
	for _, u := range online {
		live[u.UserId] = true
		Refresh(u)
	}
	cacheMu.Lock()
	for id := range cache {
		if !live[id] {
			delete(cache, id)
		}
	}
	cacheMu.Unlock()
}

// fillPrompt is the OnBuildPrompt handler. It reads only the cache.
func fillPrompt(d users.PromptData) users.PromptData {
	if d.User == nil {
		return d
	}
	cacheMu.RLock()
	values := cache[d.User.UserId]
	cacheMu.RUnlock()
	for i, t := range d.Tokens {
		if tokenSet[t.Tag] {
			d.Tokens[i].Value = values[t.Tag]
		}
	}
	return d
}

func onNewRound(events.Event) events.ListenerReturn {
	refreshAll()
	return events.Continue
}

func init() {
	users.OnBuildPrompt.Register(fillPrompt)
	events.RegisterListener(events.NewRound{}, onNewRound)
}
