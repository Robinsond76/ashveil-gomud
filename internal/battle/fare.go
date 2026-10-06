package battle

// Phase 50: the battle condition each member of the player's company began
// the battle in (its needs and meal buff), as one line per member key. It is
// fixed at the battle's start and runtime state only, like the battle.

// SetFare records the members' battle conditions under a player's battle:
// member key to its summary. Members with none are left out.
func SetFare(userId int, fare map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		b.Fare = fare
	}
}

// FareOf is a copy of the members' battle conditions in the player's battle.
func FareOf(userId int) map[string]string {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok || len(b.Fare) == 0 {
		return nil
	}
	out := make(map[string]string, len(b.Fare))
	for k, v := range b.Fare {
		out[k] = v
	}
	return out
}
