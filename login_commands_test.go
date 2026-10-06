package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
)

// TestEveryLoginCommandExists: found live (Phase 44), every player's login
// printed "inbox not recognized" because the shipped OnLoginCommands still
// ran a command from a module this fork does not have.
func TestEveryLoginCommandExists(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	loadAllDataFiles(false)

	for _, line := range configs.GetConfig().Server.OnLoginCommands {
		word := strings.ToLower(strings.Fields(line + " x")[0])
		if alias := keywords.TryCommandAlias(word); alias != word {
			word = strings.Fields(alias)[0]
		}
		if !usercommands.IsRegistered(word) && !moduleRegisters(t, word) {
			t.Errorf("OnLoginCommands runs %q, which is not a registered command", line)
		}
	}
}

// moduleRegisters reports whether a module's source registers the user
// command (plugins add theirs when loaded, which this test does not run).
func moduleRegisters(t *testing.T, word string) bool {
	t.Helper()
	found := false
	needle := `AddUserCommand("` + word + `"`
	_ = filepath.Walk("modules", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), needle) {
			found = true
		}
		return nil
	})
	return found
}
