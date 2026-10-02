package buffs

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Flags are string identifiers attached to buffs that modify or gate behavior.
//
// All flags are lowercase. Flag definitions live in YAML data files under
// the "buffs-flags" directory (one <flag>.yaml file per flag) and describe the
// flag's display name, description, and whether it is locked from editing.

// All is a special sentinel flag that matches any buff. It is never persisted
// as a flag data file and always passes validation.
const All string = ``

var (
	flagSpecs map[string]*FlagSpec = make(map[string]*FlagSpec)
)

// FlagSpec describes a single buff flag definition loaded from a YAML file.
type FlagSpec struct {
	Flag        string `yaml:"flag"`        // The flag identifier, e.g. "perma-gear"
	Name        string `yaml:"name"`        // Plain text name, e.g. "Unremovable Gear"
	Description string `yaml:"description"` // One sentence describing what the flag represents
	Locked      bool   `yaml:"locked"`      // If true, the flag cannot be edited or removed
	// Harmful and Helpful say what the flag does to its bearer, for player
	// displays; a flag that is neither (a marker such as combat-status) says
	// nothing. Phase 34 review.
	Harmful bool `yaml:"harmful,omitempty"`
	Helpful bool `yaml:"helpful,omitempty"`
}

// Id implements the fileloader.Loadable interface.
func (f *FlagSpec) Id() string {
	return f.Flag
}

// Filename returns the on-disk filename for this flag.
func (f *FlagSpec) Filename() string {
	return fmt.Sprintf("%s.yaml", strings.ToLower(f.Flag))
}

// Filepath implements the fileloader.LoadableSimple interface.
func (f *FlagSpec) Filepath() string {
	return f.Filename()
}

// Validate implements the fileloader.LoadableSimple interface.
func (f *FlagSpec) Validate() error {
	f.Flag = strings.ToLower(strings.TrimSpace(f.Flag))
	if f.Flag == `` {
		return fmt.Errorf("flag identifier cannot be empty")
	}
	if f.Name == `` {
		f.Name = f.Flag
	}
	return nil
}

// GetFlagSpec returns the flag spec for the given flag identifier, or nil if
// the flag is not defined.
func GetFlagSpec(flag string) *FlagSpec {
	return flagSpecs[strings.ToLower(flag)]
}

// GetAllFlagSpecs returns a copy of all loaded flag specs keyed by flag id.
func GetAllFlagSpecs() map[string]*FlagSpec {
	result := make(map[string]*FlagSpec, len(flagSpecs))
	for k, v := range flagSpecs {
		result[k] = v
	}
	return result
}

// GetAllFlagSpecsSorted returns a copy of every loaded flag spec, sorted by
// flag identifier.
func GetAllFlagSpecsSorted() []FlagSpec {
	result := make([]FlagSpec, 0, len(flagSpecs))
	for _, f := range flagSpecs {
		result = append(result, *f)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Flag < result[j].Flag
	})
	return result
}

// IsValidFlag reports whether the given flag is a known, defined flag. The
// special All sentinel ("") is always considered valid.
func IsValidFlag(flag string) bool {
	if flag == All {
		return true
	}
	_, ok := flagSpecs[strings.ToLower(flag)]
	return ok
}

// LoadFlagDataFiles loads all buff flag definitions from disk and merges any
// plugin-provided flags. It self-loads via the boot sequence in main.go.
func LoadFlagDataFiles() {

	start := time.Now()

	tmpFlags, err := fileloader.LoadAllFlatFiles[string, *FlagSpec](string(configs.GetFilePathsConfig().DataFiles) + `/buffs-flags`)
	if err != nil {
		panic(err)
	}

	flagSpecs = tmpFlags

	loadPluginFlags(flagSpecs)

	mudlog.Info("buffs.LoadFlagDataFiles()", "loadedCount", len(flagSpecs), "Time Taken", time.Since(start))
}
