package buffs

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

const (
	TriggersLeftExpired   = 0 // When it hits this number it will be pruned ASAP
	TriggersLeftUnlimited = 1000000000
)

type Buff struct {
	BuffId         int    // Which buff template does it refer to?
	Source         string `yaml:"source,omitempty"`         // Optional source identifier for where this buff originated. Example: spell, item, area
	OnStartWaiting bool   `yaml:"onstartwaiting,omitempty"` // Is the onstart event waiting to trigger?
	PermaBuff      bool   `yaml:"permabuff,omitempty"`      // Is this buff from a worn item or race?
	// Need to instance track the following:
	RoundCounter    int `yaml:"roundcounter,omitempty"`    // How many rounds have passed. Triggers on (RoundCounter%RoundInterval == 0)
	TriggersLeft    int `yaml:"triggersleft,omitempty"`    // How many times it triggers
	TriggersInitial int `yaml:"triggersinitial,omitempty"` // The trigger count when the buff was first applied (may differ from spec if overridden)
	Stacks          int `yaml:"stacks,omitempty"`          // Phase 30a: how many times it has stacked (0 reads as 1)
}

func (b *Buff) StatMod(statName string) int {
	if b.Expired() {
		return 0
	}
	if buffInfo := GetBuffSpec(b.BuffId); buffInfo != nil {
		return buffInfo.StatMods.Get(statName)
	}
	return 0
}

func (b *Buff) Expired() bool {
	return b.TriggersLeft <= TriggersLeftExpired
}

// A list of applied buffs
type Buffs struct {
	List      []*Buff
	buffFlags map[string][]int // a map of buff flags to the index of the buff
	buffIds   map[int]int      // a map of a buffId to it position in buffList
}

func New() Buffs {
	return Buffs{
		List:      []*Buff{},
		buffFlags: make(map[string][]int),
		buffIds:   make(map[int]int),
	}
}

func (bs *Buffs) Validate(forceRebuild ...bool) {
	if bs.buffFlags == nil {
		bs.buffFlags = make(map[string][]int)
	}
	if bs.buffIds == nil {
		bs.buffIds = make(map[int]int)
	}

	if (len(bs.List) != len(bs.buffIds)) || (len(forceRebuild) > 0 && forceRebuild[0]) {
		// Rebuild
		bs.buffIds = make(map[int]int)
		bs.buffFlags = make(map[string][]int)

		for idx, b := range bs.List {
			bs.buffIds[b.BuffId] = idx
			bSpec := GetBuffSpec(b.BuffId)
			if bSpec == nil {
				mudlog.Warn("buffs.Validate()", "buffId", b.BuffId, "error", "invalid character buffId")
				continue
			}
			for _, flag := range bSpec.Flags {
				if _, ok := bs.buffFlags[flag]; !ok {
					bs.buffFlags[flag] = []int{}
				}
				bs.buffFlags[flag] = append(bs.buffFlags[flag], idx)
			}
		}
	}
}

func (bs *Buffs) StatMod(statName string) int {
	buffAmt := 0
	for _, b := range bs.List {
		buffAmt += b.StatMod(statName)
	}
	return buffAmt
}

func (bs *Buff) Name() string {
	if sp := GetBuffSpec(bs.BuffId); sp != nil {
		return sp.Name
	}
	return ""
}

func (bs *Buffs) RemoveBuff(buffId int) bool {
	if index, ok := bs.buffIds[buffId]; ok {
		bs.List[index].TriggersLeft = TriggersLeftExpired
		return true
	}
	return false
}

func (bs *Buffs) TriggersLeft(buffId int) int {
	if idx, ok := bs.buffIds[buffId]; ok {
		return bs.List[idx].TriggersLeft
	}
	return 0
}

func (bs *Buffs) GetBuffIdsWithFlag(action string) []int {
	if action != All && !IsValidFlag(action) {
		mudlog.Warn("buffs.GetBuffIdsWithFlag()", "flag", action, "error", "unknown buff flag")
	}
	buffIds := []int{}
	for _, idx := range bs.buffFlags[action] {
		buffIds = append(buffIds, bs.List[idx].BuffId)
	}
	return buffIds
}

func (bs *Buffs) HasFlag(action string, expire bool) bool {

	if action != All {
		if !IsValidFlag(action) {
			mudlog.Warn("buffs.HasFlag()", "flag", action, "error", "unknown buff flag")
		}
		if _, ok := bs.buffFlags[action]; !ok {
			return false
		}
	}

	found := false
	for index, b := range bs.List {
		bSpec := GetBuffSpec(b.BuffId)
		for _, p := range bSpec.Flags {

			if b.Expired() {
				continue
			}
			if p == action || action == All {
				found = true

				// If expire is set, need to check the rest of the buffs to possibly expire them too.
				if expire {
					// Buff zero is special, and if force cancelled, it will be removed from the list
					if b.BuffId == 0 {
						bs.List = append(bs.List[:index], bs.List[index+1:]...)
					} else {
						b.TriggersLeft = TriggersLeftExpired
						bs.List[index] = b
					}
					break
				}

				// Otherwise just return found
				return found

			}
		}

	}

	return found
}

func (bs *Buffs) HasBuff(buffId int) bool {
	if _, ok := bs.buffIds[buffId]; ok {
		return true
	}
	return false
}

func (bs *Buffs) Started(buffId int) {
	if idx, ok := bs.buffIds[buffId]; ok {
		bs.List[idx].OnStartWaiting = false
	}
}

func (bs *Buffs) AddBuff(buffId int, isPermanent bool, triggerCountOverride ...int) bool {
	if buffInfo := GetBuffSpec(buffId); buffInfo != nil {

		newBuff := Buff{
			BuffId:          buffInfo.BuffId,
			RoundCounter:    0,
			PermaBuff:       false,
			TriggersLeft:    buffInfo.TriggerCount,
			TriggersInitial: buffInfo.TriggerCount,
			Stacks:          1,
		}

		if len(triggerCountOverride) > 0 && triggerCountOverride[0] > 0 {
			newBuff.TriggersLeft = triggerCountOverride[0]
			newBuff.TriggersInitial = triggerCountOverride[0]
		}

		if isPermanent {
			newBuff.TriggersLeft = TriggersLeftUnlimited
			newBuff.TriggersInitial = TriggersLeftUnlimited
			newBuff.PermaBuff = true
		}

		if idx, ok := bs.buffIds[buffId]; ok {
			// Phase 30a: an expired buff still awaiting the pruner starts
			// over, not from its old stacks and round count; a live one
			// gains a stack, up to its maximum.
			revived := bs.List[idx].Expired()
			bs.List[idx].TriggersLeft = newBuff.TriggersLeft
			bs.List[idx].TriggersInitial = newBuff.TriggersInitial
			bs.List[idx].PermaBuff = newBuff.PermaBuff
			switch {
			case revived:
				bs.List[idx].Stacks = 1
				bs.List[idx].RoundCounter = 0
			case bs.List[idx].Stacks < 1:
				bs.List[idx].Stacks = 1
				fallthrough
			default:
				if bs.List[idx].Stacks < buffInfo.MaxStacks {
					bs.List[idx].Stacks++
				}
			}
			return true
		}

		bs.List = append(bs.List, &newBuff)
		listIndex := len(bs.List) - 1
		bs.buffIds[buffId] = listIndex
		for _, flag := range buffInfo.Flags {
			if _, ok := bs.buffFlags[flag]; !ok {
				bs.buffFlags[flag] = []int{}
			}
			bs.buffFlags[flag] = append(bs.buffFlags[flag], listIndex)
		}

		return true
	}

	return false
}

// Returns what buffs were triggered
func (bs *Buffs) Trigger(buffId ...int) (triggeredBuffs []*Buff) {

	for idx, b := range bs.List {

		// Special case where 1 or more specific buffId's were expectred to trigger (ONLY!)
		// This might happen if a buff needs to trigger before a round begins
		if len(buffId) > 0 {
			for _, id := range buffId {
				if b.BuffId != id {
					continue
				}
			}
		}

		if buffInfo := GetBuffSpec(b.BuffId); buffInfo != nil {

			// Combat statuses are advanced only by the combat pass.
			if buffInfo.CombatRounds {
				continue
			}

			// If there's no more life left to it, prune it
			// We do this first so that it's the first thing that happens AFTER a full round has already passed.
			if b.TriggersLeft > 0 {
				b.RoundCounter++
				if b.RoundCounter%buffInfo.RoundInterval == 0 {
					// It cannot be pruned unless it is triggered
					triggeredBuffs = append(triggeredBuffs, b)
					if b.TriggersLeft != TriggersLeftUnlimited {
						b.TriggersLeft--
					} else {
						// If unimited, reset the counter to prevent some future overflow
						b.RoundCounter = 0
					}
				}
				bs.List[idx] = b
			}

		}

	}

	return triggeredBuffs
}

func (bs *Buffs) GetBuffs(buffId ...int) []*Buff {
	retBuffs := []*Buff{}
	for _, b := range bs.List {
		if !b.Expired() {

			if len(buffId) > 0 {
				for _, id := range buffId {
					if b.BuffId != id {
						continue
					}
					retBuffs = append(retBuffs, b)
				}
			} else {
				retBuffs = append(retBuffs, b)
			}

		}
	}
	return retBuffs
}

func (bs *Buffs) Prune() (prunedBuffs []*Buff) {

	if len(bs.List) == 0 {
		return prunedBuffs
	}

	var prune bool = false
	var didPrune bool = false
	for i := len(bs.List) - 1; i >= 0; i-- {

		prune = false

		b := bs.List[i] // Get a ptr to the data within the slice

		buffInfo := GetBuffSpec(b.BuffId)

		if buffInfo == nil {
			prune = true
		} else {
			// If there's no more life left to it, prune it
			// We do this first so that it's the first thing that happens AFTER a full round has already passed.
			if b.Expired() {
				prune = true
			}
		}

		if prune {
			prunedBuffs = append(prunedBuffs, b)
			// remove the buff
			bs.List = append(bs.List[:i], bs.List[i+1:]...)
			didPrune = true
		}
	}

	// Since pruning occured, rebuild the lookups
	if didPrune {
		bs.Validate(true)
	}

	return prunedBuffs
}

func GetDurations(buff *Buff, spec *BuffSpec) (roundsLeft int, totalRounds int) {

	// Phase 30a: a combat-round buff's count is combat rounds, and
	// game-round triggering never applies to it. Since Phase 82c a battle
	// round lasts as long as its actions need, so the count is reported as
	// battle rounds, not converted to game rounds.
	if spec.CombatRounds {
		initial := buff.TriggersInitial
		if initial <= 0 {
			initial = spec.TriggerCount
		}
		return max(buff.TriggersLeft, 0), initial
	}

	if spec.RoundInterval <= 0 {
		return 0, 0
	}

	initialTriggers := buff.TriggersInitial
	if initialTriggers <= 0 {
		initialTriggers = spec.TriggerCount
	}
	totalRounds = initialTriggers * spec.RoundInterval

	if buff.TriggersLeft <= 0 {
		return 0, totalRounds
	}

	roundsIntoInterval := buff.RoundCounter % spec.RoundInterval
	roundsUntilNextTrigger := spec.RoundInterval - roundsIntoInterval
	roundsLeft = (buff.TriggersLeft-1)*spec.RoundInterval + roundsUntilNextTrigger

	return roundsLeft, totalRounds
}
