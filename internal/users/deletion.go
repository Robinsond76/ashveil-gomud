package users

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"

	"gopkg.in/yaml.v2"
)

// Ashveil Phase 32h: deleting a character, keeping the login. The
// `delete character` command flags the record (Deleting) and saves it, the
// player leaves the world with a hand-off despawn, every module drops the
// user's state on events.UserPurged{KeepAccount: true}, and then
// ResetDeletedCharacter gives the record a new character in the Void. The
// flag is durable, so a crash anywhere in between is finished by the boot
// sweep (hooks.SweepDeletions).

// DeletingLoginRefusal is what a login of a flagged record is told.
const DeletingLoginRefusal = `That character is being deleted. Try again in a moment.`

// ResetDeletedCharacter replaces an offline flagged user's character with
// a new one in the Void, frees the old name in the character index, clears
// the flag, and saves. The account (username, password, role, settings,
// macros, aliases, tips) stays. An unflagged or missing record is left as
// it is, so the reset can run twice.
func ResetDeletedCharacter(userId int) error {
	if GetByUserId(userId) != nil {
		return fmt.Errorf("user %d is online", userId)
	}
	u, err := loadUserById(userId)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !u.Deleting {
		return nil
	}
	if u.Character != nil && u.Character.Name != `` {
		if owner, ok := GetCharacterIndex().Find(u.Character.Name); ok && owner == userId {
			GetCharacterIndex().Remove(u.Character.Name)
		}
	}
	u.Character = characters.New()
	u.Character.SetUserId(u.UserId)
	u.Character.Validate()
	u.Deleting = false
	return SaveUser(*u)
}

// DeletingUserIds lists the users whose record on file is flagged for
// deletion, online or not (the boot sweep).
func DeletingUserIds() []int {
	dir := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []int
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, `.yaml`) {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSuffix(name, `.yaml`))
		if err != nil || id >= ReplayUserIdBase {
			continue
		}
		data, err := os.ReadFile(util.FilePath(dir, `/`, name))
		if err != nil {
			continue
		}
		var flag struct {
			Deleting bool `yaml:"deleting"`
		}
		if yaml.Unmarshal(data, &flag) != nil || !flag.Deleting {
			continue
		}
		out = append(out, id)
	}
	return out
}
