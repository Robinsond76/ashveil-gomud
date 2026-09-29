package races

import (
	"fmt"
	"regexp"
	"strings"
)

// PainReaction pairs what the wounded character hears with what others see.
type PainReaction struct {
	ToVictim string `yaml:"tovictim"`
	ToRoom   string `yaml:"toroom"`
}

var painToken = regexp.MustCompile(`\{[^{}]*\}`)

// ValidatePainReactions rejects incomplete or unrenderable authored lines.
func ValidatePainReactions(pairs []PainReaction) error {
	for i, pair := range pairs {
		if strings.TrimSpace(pair.ToVictim) == "" || strings.TrimSpace(pair.ToRoom) == "" {
			return fmt.Errorf("pain reaction %d needs both victim and room lines", i+1)
		}
		for _, line := range []string{pair.ToVictim, pair.ToRoom} {
			for _, token := range painToken.FindAllString(line, -1) {
				switch token {
				case "{name}", "{he}", "{him}", "{his}":
				default:
					return fmt.Errorf("pain reaction %d has unsupported token %s", i+1, token)
				}
			}
			withoutKnown := strings.NewReplacer("{name}", "", "{he}", "", "{him}", "", "{his}", "").Replace(line)
			if strings.ContainsAny(withoutKnown, "{}") {
				return fmt.Errorf("pain reaction %d has an incomplete token", i+1)
			}
		}
	}
	return nil
}
