package parties

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// Only membership/leadership is durable. Invitations, ranks and consent are
// deliberately omitted: recovery never restores authority to act automatically.
type savedParty struct {
	Leader  int   `json:"leader"`
	Members []int `json:"members"`
}
type savedAlliances struct {
	Version int          `json:"version"`
	Parties []savedParty `json:"parties"`
}

var storagePath string
var storageErr error

func ClearError() { storageErr = nil }

func LastError() error { return storageErr }

// ConfigureStorage loads the alliance file before serving commands. Missing
// files migrate old runtime-only parties to an empty registry. Bad data fails
// closed, preserving the previous registry and never overwriting the file.
func ConfigureStorage(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		storagePath = path
		partyMap = map[int]*Party{}
		storageErr = nil
		return nil
	}
	if err != nil {
		return err
	}
	var saved savedAlliances
	if err = json.Unmarshal(data, &saved); err != nil {
		return err
	}
	if saved.Version != 1 {
		return fmt.Errorf("unsupported alliance version %d", saved.Version)
	}
	loaded := map[int]*Party{}
	for _, entry := range saved.Parties {
		if entry.Leader <= 0 || !slices.Contains(entry.Members, entry.Leader) {
			return fmt.Errorf("invalid alliance leader")
		}
		p := &Party{LeaderUserId: entry.Leader, UserIds: append([]int(nil), entry.Members...), Position: map[int]string{}}
		for _, id := range entry.Members {
			if id <= 0 || loaded[id] != nil {
				return fmt.Errorf("duplicate or invalid alliance member %d", id)
			}
			loaded[id] = p
		}
	}
	partyMap = loaded
	storagePath = path
	storageErr = nil
	return nil
}

// checkpoint preserves pointer identity on rollback, since commands/GMCP hold
// party pointers. The game loop owns these mutations and the storage operation.
func checkpoint() func() {
	index := make(map[int]*Party, len(partyMap))
	copies := map[*Party]Party{}
	for id, p := range partyMap {
		index[id] = p
		copy := *p
		copy.UserIds = slices.Clone(p.UserIds)
		copy.InviteUserIds = slices.Clone(p.InviteUserIds)
		copy.AutoAttackers = slices.Clone(p.AutoAttackers)
		copy.Followers = slices.Clone(p.Followers)
		copy.Supporters = slices.Clone(p.Supporters)
		copy.Position = map[int]string{}
		for k, v := range p.Position {
			copy.Position[k] = v
		}
		copy.autoTokens = map[int]uint64{}
		for k, v := range p.autoTokens {
			copy.autoTokens[k] = v
		}
		copy.followTokens = map[int]uint64{}
		for k, v := range p.followTokens {
			copy.followTokens[k] = v
		}
		copies[p] = copy
	}
	return func() {
		partyMap = index
		for p, copy := range copies {
			*p = copy
		}
	}
}

func persist(undo func()) bool {
	if storagePath == "" {
		return true
	}
	saved := savedAlliances{Version: 1, Parties: []savedParty{}}
	seen := map[*Party]bool{}
	for _, p := range partyMap {
		if !seen[p] {
			seen[p] = true
			saved.Parties = append(saved.Parties, savedParty{Leader: p.LeaderUserId, Members: slices.Clone(p.UserIds)})
		}
	}
	sort.Slice(saved.Parties, func(i, j int) bool { return saved.Parties[i].Leader < saved.Parties[j].Leader })
	data, err := json.MarshalIndent(saved, "", "  ")
	if err == nil {
		err = writeAllianceFile(storagePath, data)
	}
	if err != nil {
		undo()
		storageErr = fmt.Errorf("alliance change was not saved and was rolled back: %w", err)
		return false
	}
	storageErr = nil
	return true
}

func writeAllianceFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".alliances-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// UseMemoryForTest isolates persistence and registry without touching world files.
func UseMemoryForTest() func() {
	oldMap, oldPath, oldErr := partyMap, storagePath, storageErr
	partyMap = map[int]*Party{}
	storagePath = ""
	storageErr = nil
	return func() { partyMap = oldMap; storagePath = oldPath; storageErr = oldErr }
}
