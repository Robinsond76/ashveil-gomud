package mobs

import (
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Idle chatter limits (world style bible, 2026-10-07): a town NPC speaks
// rarely and never repeats itself to the same player for a long while.
//
// The engine gives every idle mob a turn each round, and both the idle
// command list and onIdle scripts used to fire a say or emote every few
// seconds. While a mob runs its idle turn (BeginIdle/EndIdle around it), each
// say, sayto, shout or emote it queues goes through one decision for that
// turn: it may speak if a player is there to hear it, its cooldown has passed,
// and the turn's first line is new to at least one player in the room (from
// this kind of mob, within the memory window). One veteran in the room never
// silences a line a newcomer has yet to hear, such as a quest hint. A turn
// that may speak lets every line of that turn through, so a scripted speech
// is never cut in half; a turn that may not drops them all. Other commands
// (wander, pathto, lookfortrouble) and anything queued outside an idle turn
// (combat, replies to players, conversations already under way) pass as
// before.
//
// The memory is cosmetic and kept in memory only: a restart forgets it.

// chatterVerbs lists the speaking commands and their keyword aliases
// (keywords.yaml command-aliases), so `yell` or `.` can't slip past.
var chatterVerbs = map[string]bool{
	`say`:    true,
	`.`:      true,
	`sayto`:  true,
	`shout`:  true,
	`yell`:   true,
	`scream`: true,
	`holler`: true,
	`emote`:  true,
}

type idleChatter struct {
	listeners []int // players in the room when the idle turn began
	decided   bool
	allowed   bool
}

type chatterLimitValues struct {
	cooldown uint64 // rounds between a mob's idle chatter (0 = no cooldown)
	memory   uint64 // rounds a player remembers a line (0 = no memory)
}

// chatterLimitsOverride lets tests fix the limits without loading config.
var chatterLimitsOverride *chatterLimitValues

func chatterLimits() chatterLimitValues {
	if chatterLimitsOverride != nil {
		return *chatterLimitsOverride
	}
	gp := configs.GetGamePlayConfig()
	return chatterLimitValues{
		cooldown: uint64(max(0, int(gp.MobChatterCooldownRounds))),
		memory:   uint64(max(0, int(gp.MobChatterMemoryRounds))),
	}
}

var chatterMemory = struct {
	sync.Mutex
	heard     map[int]map[string]uint64 // userId -> line key -> round heard
	lastSweep uint64                    // round the whole map was last pruned
}{heard: map[int]map[string]uint64{}}

// BeginIdle marks the start of the mob's idle turn; listeners are the players
// who would hear it.
func (m *Mob) BeginIdle(listeners []int) {
	m.idle = &idleChatter{listeners: listeners}
}

// EndIdle ends the idle turn begun by BeginIdle.
func (m *Mob) EndIdle() {
	m.idle = nil
}

// ChatterReady reports whether the mob's idle chatter cooldown has passed.
// Conversations between mobs start only when it has.
func (m *Mob) ChatterReady() bool {
	cooldown := chatterLimits().cooldown
	return cooldown == 0 || m.lastChatter == 0 || util.GetRoundCount()-m.lastChatter >= cooldown
}

// MarkChatter starts the mob's chatter cooldown now.
func (m *Mob) MarkChatter() {
	m.lastChatter = max(util.GetRoundCount(), 1)
}

func isChatterCommand(cmd string) bool {
	verb, _, _ := strings.Cut(strings.TrimSpace(cmd), ` `)
	return chatterVerbs[strings.ToLower(verb)]
}

func (m *Mob) chatterKey(cmd string) string {
	return strconv.Itoa(int(m.MobId)) + `|` + strings.ToLower(strings.TrimSpace(cmd))
}

// idleChatterAllowed decides whether a queued command runs. Only chatter
// queued during an idle turn is ever held back.
func (m *Mob) idleChatterAllowed(cmd string) bool {
	if m.idle == nil || !isChatterCommand(cmd) {
		return true
	}

	limits := chatterLimits()
	now := util.GetRoundCount()
	key := m.chatterKey(cmd)

	chatterMemory.Lock()
	defer chatterMemory.Unlock()

	if !m.idle.decided {
		m.idle.decided = true
		sweepChatterMemoryLocked(now, limits.memory)
		m.idle.allowed = len(m.idle.listeners) > 0 && m.ChatterReady() && !everyoneHeardLocked(m.idle.listeners, key, now, limits.memory)
		if m.idle.allowed {
			m.MarkChatter()
		}
	}

	if !m.idle.allowed {
		return false
	}

	if limits.memory > 0 {
		for _, userId := range m.idle.listeners {
			lines := chatterMemory.heard[userId]
			if lines == nil {
				lines = map[string]uint64{}
				chatterMemory.heard[userId] = lines
			}
			lines[key] = now
		}
	}
	return true
}

// everyoneHeardLocked reports whether every listener heard the line within
// the memory window, forgetting their older lines as it goes. chatterMemory
// must be held.
func everyoneHeardLocked(listeners []int, key string, now uint64, memory uint64) bool {
	if memory == 0 || len(listeners) == 0 {
		return false
	}
	for _, userId := range listeners {
		lines := chatterMemory.heard[userId]
		pruneLinesLocked(userId, lines, now, memory)
		if _, ok := lines[key]; !ok {
			return false
		}
	}
	return true
}

func pruneLinesLocked(userId int, lines map[string]uint64, now uint64, memory uint64) {
	for k, round := range lines {
		if now-round >= memory {
			delete(lines, k)
		}
	}
	if len(lines) == 0 {
		delete(chatterMemory.heard, userId)
	}
}

// sweepChatterMemoryLocked prunes every player once per memory window, so
// players who left town or logged out don't keep their lines until restart.
func sweepChatterMemoryLocked(now uint64, memory uint64) {
	if memory == 0 || now-chatterMemory.lastSweep < memory {
		return
	}
	chatterMemory.lastSweep = now
	for userId, lines := range chatterMemory.heard {
		pruneLinesLocked(userId, lines, now, memory)
	}
}

// ForgetChatterForTest clears every player's chatter memory.
func ForgetChatterForTest() {
	chatterMemory.Lock()
	defer chatterMemory.Unlock()
	clear(chatterMemory.heard)
	chatterMemory.lastSweep = 0
}
