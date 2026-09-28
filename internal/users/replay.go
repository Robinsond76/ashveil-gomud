package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/util"

	"gopkg.in/yaml.v2"
)

// Ashveil Phase 32b: tutorial replays. A replay is a throwaway user, with
// its own id, that stands in for a real character on the same connection
// while the player runs the course again. It is in neither the user nor
// the character index, so it can't be logged into, and its id comes from
// a range the index never hands out.

// ReplayUserIdBase is the first id a replay user can have. Real accounts
// take their ids from the user index, counting up from 1.
const ReplayUserIdBase = 900_000_000

// replayUsernamePrefix starts every replay username; no account name can
// contain a colon.
const replayUsernamePrefix = `replay:`

// IsReplay reports whether this user is a tutorial replay.
func (u *UserRecord) IsReplay() bool {
	return u != nil && u.ReplayOf != 0
}

// userFilePath is where a user's record is saved.
func userFilePath(userId int) string {
	return util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(userId)+`.yaml`)
}

// NewReplayUser builds a replay of a real user: a new, level-1 character
// with the real one's name and race and nothing else, flagged with the
// real user's id. The player's own settings (screen reader, options,
// aliases, macros, role) come along. It is saved to its file but not
// logged in or indexed.
func NewReplayUser(real *UserRecord) (*UserRecord, error) {
	if real == nil || real.Character == nil {
		return nil, errors.New("no character to replay")
	}
	if real.IsReplay() {
		return nil, errors.New("already a replay")
	}
	id, err := nextReplayUserId()
	if err != nil {
		return nil, err
	}

	u := NewUserRecord(id, 0)
	u.ReplayOf = real.UserId
	u.Username = replayUsernamePrefix + strconv.Itoa(id)
	// Never a plaintext password: that would hold the player at the
	// "change your password" prompt. Nobody can log in to it anyway.
	u.Password = real.Password
	u.Role = real.Role
	u.Permissions = append([]string(nil), real.Permissions...)
	u.ScreenReader = real.ScreenReader
	u.IsAI = real.IsAI
	u.EmailAddress = ``
	for k, v := range real.ConfigOptions {
		u.ConfigOptions[k] = v
	}
	for k, v := range real.Macros {
		u.Macros[k] = v
	}
	if len(real.Aliases) > 0 {
		u.Aliases = map[string]string{}
		for k, v := range real.Aliases {
			u.Aliases[k] = v
		}
	}
	u.TipsComplete = map[string]bool{}
	for k, v := range real.TipsComplete {
		u.TipsComplete[k] = v
	}

	c := characters.New()
	c.Name = real.Character.Name
	c.RaceId = real.Character.RaceId
	if r := races.GetRace(c.RaceId); r != nil {
		c.Alignment = r.DefaultAlignment
	}
	c.ExtraLives = int(configs.GetGamePlayConfig().LivesStart)
	// Placed by the tutorial once it spawns; the Void until then.
	c.RoomId = -1
	c.Zone = real.Character.Zone
	c.Validate()
	u.Character = c

	if err := SaveUser(*u); err != nil {
		return nil, err
	}
	return u, nil
}

// nextReplayUserId is the lowest id from ReplayUserIdBase that has neither
// a user file nor an online user.
func nextReplayUserId() (int, error) {
	for id := ReplayUserIdBase; id < ReplayUserIdBase+1_000_000; id++ {
		if GetByUserId(id) != nil {
			continue
		}
		if _, err := os.Stat(userFilePath(id)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return 0, err
		}
		return id, nil
	}
	return 0, errors.New("no replay ids left")
}

// OnlineReplayOf is the online replay of a real user, if any.
func OnlineReplayOf(realUserId int) *UserRecord {
	userManagerMu.RLock()
	defer userManagerMu.RUnlock()
	for _, u := range userManager.Users {
		if u.ReplayOf == realUserId && realUserId != 0 {
			return u
		}
	}
	return nil
}

// OfflineReplayUserIds lists the replay users whose files remain but who
// aren't online: left behind by a restart or crash mid-replay. Copyover
// restores the online ones first, so they are never listed.
func OfflineReplayUserIds() []int {
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
		if err != nil || id < ReplayUserIdBase {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var flag struct {
			ReplayOf int `yaml:"replayof"`
		}
		if yaml.Unmarshal(data, &flag) != nil || flag.ReplayOf == 0 {
			continue
		}
		if GetByUserId(id) != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

// LoadUserFile loads a user straight from their file by id, without the
// user index: the only way to load a replay user.
func LoadUserFile(userId int) (*UserRecord, error) {
	u, err := loadUserById(userId)
	if err != nil {
		return nil, err
	}
	u.connectionTime = time.Now() // as LoadUser: time online starts now
	return u, nil
}

// RemoveUserFile removes an offline user's file. Removing one that is
// already gone succeeds, so a purge can run twice.
func RemoveUserFile(userId int) error {
	if GetByUserId(userId) != nil {
		return fmt.Errorf("user %d is online", userId)
	}
	if err := os.Remove(userFilePath(userId)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
