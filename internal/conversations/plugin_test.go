package conversations

import (
	"io/fs"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

type fakeFS struct {
	files map[string][]byte
}

func newFakeFS(files map[string][]byte) *fakeFS {
	return &fakeFS{files: files}
}

func (f *fakeFS) ReadFile(name string) ([]byte, error) {
	if b, ok := f.files[name]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeFS) Open(name string) (fs.File, error) { return nil, fs.ErrNotExist }

func (f *fakeFS) KnownPaths() []string {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	return paths
}

func (f *fakeFS) AllFileSubSystems(yield func(fs.ReadFileFS) bool) { yield(f) }

func resetPluginState() {
	pluginFileSystems = nil
	converseCheckCache = map[string]bool{}
	conversations = map[int]*Conversation{}
	conversationCounter = map[string]int{}
	conversationUniqueId = 0
}

const sampleConversation = `-
  Supported:
    "*": ["*"]
  Conversation:
    - ["#1 say hello"]
    - ["#2 say hi"]
`

func TestHasConverseFile_FindsPluginFile(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	if !HasConverseFile(9001, "TestZone") {
		t.Fatalf("expected plugin conversation file to be found")
	}
	// Second call exercises the cache path.
	if !HasConverseFile(9001, "TestZone") {
		t.Fatalf("expected cached plugin conversation file to be found")
	}
}

func TestHasConverseFile_MissingReturnsFalse(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	if HasConverseFile(9002, "TestZone") {
		t.Fatalf("expected missing conversation to return false")
	}
}

func TestAttemptConversation_UsesPluginFile(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	convId := AttemptConversation(9001, 1, "goblin", 2, "rat", "TestZone")
	if convId == 0 {
		t.Fatalf("expected a non-zero conversation id from plugin file")
	}

	c := getConversation(convId)
	if c == nil {
		t.Fatalf("expected conversation to be stored")
	}
	if len(c.ActionList) != 2 {
		t.Fatalf("expected 2 conversation actions, got %d", len(c.ActionList))
	}
}

func TestReadPluginConversationFile_KeyFormat(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/frostfang/42.yaml`: []byte(sampleConversation),
	}))

	if _, ok := readPluginConversationFile("frostfang", 42); !ok {
		t.Fatalf("expected plugin conversation file at frostfang/42")
	}
	if _, ok := readPluginConversationFile("frostfang", 43); ok {
		t.Fatalf("did not expect a file for mob 43")
	}
}

// A conversation not yet stepped has no LastRound; the maintenance sweep
// must age it from its start, not prune it at once whenever the round
// count is past ten (37c: this failed in shuffled runs).
func TestFreshConversationSurvivesMaintenance(t *testing.T) {
	resetPluginState()
	defer resetPluginState()
	prev := util.GetRoundCount()
	defer util.SetRoundCount(prev)
	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	util.SetRoundCount(500)
	convId := AttemptConversation(9001, 1, "goblin", 2, "rat", "TestZone")
	if convId == 0 {
		t.Fatal("no conversation")
	}
	util.SetRoundCount(503)
	defer util.UseRandForTest(func(int) int { return 0 })() // every call sweeps
	if getConversation(convId) == nil {
		t.Fatal("pruned three rounds after it began, never stepped")
	}
	util.SetRoundCount(520)
	if getConversation(convId) != nil {
		t.Fatal("twenty idle rounds should be pruned")
	}
}
