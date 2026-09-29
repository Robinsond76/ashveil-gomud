package battle

import (
	"fmt"
	"sort"
	"strings"
)

// EnemyName is the immutable combat-narration identity captured for an enemy
// instance while a battle is active.
type EnemyName struct {
	InstanceId  int
	BaseName    string
	Noun        string
	DisplayName string
}

func cloneEnemyNames(names map[int]EnemyName) map[int]EnemyName {
	clone := make(map[int]EnemyName, len(names))
	for id, snapshot := range names {
		clone[id] = snapshot
	}
	return clone
}

// AssignEnemyNames adds snapshots for members seen in the user's battle. It
// shares snapshots with the complete, same-room component of overlapping
// battles, without consulting live game state while the battle mutex is held.
func AssignEnemyNames(userId int, members []EnemyName) {
	mu.Lock()
	defer mu.Unlock()

	start, ok := battles[userId]
	if !ok {
		return
	}
	component := overlappingBattles(start)
	known := make(map[int]EnemyName)
	for _, b := range component {
		for id, snapshot := range b.EnemyNames {
			if _, exists := known[id]; !exists {
				known[id] = snapshot
			}
		}
	}

	newMembers := make(map[int]EnemyName)
	for _, member := range members {
		if member.InstanceId == 0 {
			continue
		}
		if _, exists := known[member.InstanceId]; exists {
			continue
		}
		newMembers[member.InstanceId] = normalizeEnemyName(member)
	}
	assignNewEnemyNames(known, newMembers)

	for _, b := range component {
		b.EnemyNames = cloneEnemyNames(known)
	}
}

// EnemyDisplayName returns a captured label for an active battle, falling
// back to the caller's current name when no snapshot remains.
func EnemyDisplayName(instanceId int, fallback string) string {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range battles {
		if snapshot, ok := b.EnemyNames[instanceId]; ok {
			return snapshot.DisplayName
		}
	}
	return fallback
}

func overlappingBattles(start *Battle) []*Battle {
	component := []*Battle{start}
	seen := map[*Battle]bool{start: true}
	for i := 0; i < len(component); i++ {
		for _, candidate := range battles {
			if seen[candidate] || candidate.RoomId != start.RoomId || !sharesEnemy(component[i], candidate) {
				continue
			}
			seen[candidate] = true
			component = append(component, candidate)
		}
	}
	return component
}

func sharesEnemy(a, b *Battle) bool {
	for id := range a.Enemies {
		if b.Enemies[id] {
			return true
		}
	}
	return false
}

func normalizeEnemyName(name EnemyName) EnemyName {
	name.BaseName = strings.TrimSpace(name.BaseName)
	if name.BaseName == "" {
		name.BaseName = "creature"
	}
	name.Noun = strings.TrimSpace(name.Noun)
	if name.Noun == "" {
		name.Noun = defaultNoun(name.BaseName)
	}
	return name
}

func assignNewEnemyNames(known, fresh map[int]EnemyName) {
	if len(fresh) == 0 {
		return
	}
	all := make(map[int]EnemyName, len(known)+len(fresh))
	for id, snapshot := range known {
		all[id] = snapshot
	}
	for id, snapshot := range fresh {
		all[id] = snapshot
	}
	cohorts := map[string][]EnemyName{}
	for _, snapshot := range all {
		cohorts[baseKey(snapshot.BaseName)] = append(cohorts[baseKey(snapshot.BaseName)], snapshot)
	}
	for _, cohort := range cohorts {
		sort.Slice(cohort, func(i, j int) bool { return cohort[i].InstanceId < cohort[j].InstanceId })
	}

	used := make(map[string]bool, len(known))
	for _, snapshot := range known {
		used[displayKey(snapshot.DisplayName)] = true
	}
	knownDisplay := make(map[string]bool, len(used))
	for display := range used {
		knownDisplay[display] = true
	}
	keys := make([]string, 0, len(cohorts))
	for key, cohort := range cohorts {
		keys = append(keys, key)
		if len(cohort) == 1 {
			if _, isFresh := fresh[cohort[0].InstanceId]; isFresh {
				used[displayKey(cohort[0].BaseName)] = true
			}
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		cohort := cohorts[key]
		freshCohort := make([]EnemyName, 0, len(cohort))
		for _, snapshot := range cohort {
			if _, exists := fresh[snapshot.InstanceId]; exists {
				freshCohort = append(freshCohort, snapshot)
			}
		}
		if len(freshCohort) == 0 {
			continue
		}
		sharedNoun := cohort[0].Noun
		if nounCollides(key, sharedNoun, cohorts) {
			sharedNoun = strippedBaseName(cohort[0].BaseName)
		}
		sort.Slice(freshCohort, func(i, j int) bool { return freshCohort[i].InstanceId < freshCohort[j].InstanceId })
		knownCount := len(cohort) - len(freshCohort)
		for i, snapshot := range freshCohort {
			rank := knownCount + i + 1
			if len(cohort) == 1 {
				if knownDisplay[displayKey(snapshot.BaseName)] {
					snapshot.DisplayName = uniqueDisplayName(snapshot.BaseName, snapshot.InstanceId, used)
				} else {
					snapshot.DisplayName = snapshot.BaseName
				}
			} else {
				snapshot.Noun = sharedNoun
				snapshot.DisplayName = uniqueOrdinalName(rank, sharedNoun, strippedBaseName(snapshot.BaseName), snapshot.InstanceId, used)
			}
			fresh[snapshot.InstanceId] = snapshot
			used[displayKey(snapshot.DisplayName)] = true
		}
	}
	for id, snapshot := range fresh {
		known[id] = snapshot
	}
}

func nounCollides(base, noun string, cohorts map[string][]EnemyName) bool {
	for otherBase, cohort := range cohorts {
		if otherBase != base && strings.EqualFold(strings.TrimSpace(cohort[0].Noun), strings.TrimSpace(noun)) {
			return true
		}
	}
	return false
}

func defaultNoun(base string) string {
	words := strings.Fields(strippedBaseName(base))
	if len(words) == 0 {
		return "creature"
	}
	return words[len(words)-1]
}

func strippedBaseName(base string) string {
	words := strings.Fields(strings.TrimSpace(base))
	if len(words) > 1 {
		switch strings.ToLower(words[0]) {
		case "a", "an", "the":
			words = words[1:]
		}
	}
	if len(words) == 0 {
		return "creature"
	}
	return strings.Join(words, " ")
}

func baseKey(base string) string { return strings.ToLower(strings.TrimSpace(base)) }
func displayKey(display string) string {
	return strings.ToLower(strings.TrimSpace(display))
}

func uniqueDisplayName(base string, instanceID int, used map[string]bool) string {
	if !used[displayKey(base)] {
		return base
	}
	for discriminator := 1; ; discriminator++ {
		candidate := fmt.Sprintf("%s (%d)", base, instanceID)
		if discriminator > 1 {
			candidate = fmt.Sprintf("%s (%d, %d)", base, instanceID, discriminator)
		}
		if !used[displayKey(candidate)] {
			return candidate
		}
	}
}

func uniqueOrdinalName(rank int, noun, fullBase string, instanceID int, used map[string]bool) string {
	candidate := ordinal(rank) + " " + noun
	if !used[displayKey(candidate)] {
		return candidate
	}
	candidate = ordinal(rank) + " " + fullBase
	if !used[displayKey(candidate)] {
		return candidate
	}
	return uniqueDisplayName(candidate, instanceID, used)
}

func ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"}
	if n >= 1 && n <= 10 {
		return words[n]
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
