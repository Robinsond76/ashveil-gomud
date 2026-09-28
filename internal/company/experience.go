package company

import "fmt"

// LevelLine is the notice a leader gets when a companion levels up from a
// kill (Phase 32e).
func LevelLine(name string, to int) string {
	return fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi> reached level %d!`, name, to)
}
