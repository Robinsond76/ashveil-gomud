package blessings

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type memStore struct {
	saved   []byte
	saves   int
	failing bool
	loadErr error
}

func (s *memStore) Load(r *Registry) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	*r = Registry{Accounts: map[int][]Earned{}}
	if s.saved == nil {
		return nil
	}
	return yaml.Unmarshal(s.saved, r)
}

func (s *memStore) Save(r Registry) error {
	if s.failing {
		return errors.New("disk full")
	}
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	s.saved = data
	s.saves++
	return nil
}

type pushed struct {
	user      int
	namespace string
	payload   any
}

type fakeWorld struct {
	chars  map[int]who
	users  map[int]*users.UserRecord
	told   []string
	pushes []pushed
}

func (w *fakeWorld) Character(id int) (who, bool) { c, ok := w.chars[id]; return c, ok }
func (w *fakeWorld) User(id int) (*users.UserRecord, bool) {
	u, ok := w.users[id]
	return u, ok
}
func (w *fakeWorld) Tell(_ int, text string) { w.told = append(w.told, text) }
func (w *fakeWorld) Push(id int, namespace string, payload any) {
	w.pushes = append(w.pushes, pushed{id, namespace, payload})
}

// env is the real module (so the chronicle's deed seam reaches it) with a
// fake world and an in-memory store, the shipped blessings, and an
// in-memory chronicle.
func env(t *testing.T) (*Module, *fakeWorld, *memStore) {
	t.Helper()
	require.NoError(t, blessings.LoadBytes(shippedBlessings(t)))
	prevW, prevStore, prevAcc := module.w, module.store, module.accounts
	w := &fakeWorld{chars: map[int]who{}, users: map[int]*users.UserRecord{}}
	store := &memStore{}
	module.w, module.store = w, store
	module.mu.Lock()
	module.accounts = map[int][]Earned{}
	module.loadErr = nil
	module.mu.Unlock()
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() {
		module.w, module.store, module.accounts = prevW, prevStore, prevAcc
		blessings.SetData(nil)
		chronicle.SetProvider(nil)
	})
	return module, w, store
}

func signIn(w *fakeWorld, id int, name string, iron bool) *users.UserRecord {
	u := users.NewUserRecord(id, 1)
	u.Character = &characters.Character{Name: name, Hardcore: iron}
	w.chars[id] = who{Name: name, Iron: iron}
	w.users[id] = u
	return u
}

func deed(id int, kind chronicle.Kind, n int) {
	for i := 0; i < n; i++ {
		chronicle.Record(id, chronicle.Entry{Kind: kind, Members: []string{"Mara"}})
	}
}

func TestADeedEarnsABlessingOnceAndTellsThePlayer(t *testing.T) {
	m, w, store := env(t)
	signIn(w, 11, "Mara", false)

	deed(11, chronicle.Boss, 1)
	assert.Equal(t, []string{"road-tested"}, m.Earned(11), "a boss kill, through the real chronicle seam")
	require.Len(t, w.told, 1)
	assert.Contains(t, w.told[0], "Road-tested")
	assert.Contains(t, w.told[0], "Your next character will")
	require.NotEmpty(t, w.pushes)
	assert.Equal(t, "Char.Blessings", w.pushes[len(w.pushes)-1].namespace)
	assert.Equal(t, 1, store.saves, "saved at once")

	deed(11, chronicle.Boss, 2)
	assert.Equal(t, []string{"road-tested"}, m.Earned(11), "never earned twice (three bosses, standard: no Iron blessing)")
	assert.Len(t, w.told, 1)
	deed(11, chronicle.Boss, 2)
	assert.Equal(t, []string{"road-tested", "old-hand"}, m.Earned(11), "five bosses earn the next, in order earned")
}

func TestIronBlessingsOnlyCountForAnIronCharacter(t *testing.T) {
	m, w, _ := env(t)
	signIn(w, 12, "Standard", false)
	signIn(w, 13, "Ferro", true)
	deed(12, chronicle.Boss, 3)
	deed(13, chronicle.Boss, 3)
	assert.NotContains(t, m.Earned(12), "iron-tested")
	assert.Contains(t, m.Earned(13), "iron-tested")
	deed(13, chronicle.Promoted, 1)
	assert.Contains(t, m.Earned(13), "iron-oath")
}

func TestBlessingsSurviveACharacterDeletionButNotAnAccountPurge(t *testing.T) {
	m, w, store := env(t)
	signIn(w, 14, "Mara", false)
	deed(14, chronicle.Boss, 1)
	require.Equal(t, []string{"road-tested"}, m.Earned(14))

	// `delete character` purges with the account kept.
	m.onUserPurged(events.UserPurged{UserId: 14, KeepAccount: true})
	assert.Equal(t, []string{"road-tested"}, m.Earned(14), "the account keeps its blessings")

	// They are on disk too, and reload.
	m2 := newModule()
	m2.store = store
	m2.load()
	assert.Equal(t, []string{"road-tested"}, m2.Earned(14))

	// A purge that removes the whole account takes them.
	m.onUserPurged(events.UserPurged{UserId: 14})
	assert.Empty(t, m.Earned(14))
	m3 := newModule()
	m3.store = store
	m3.load()
	assert.Empty(t, m3.Earned(14))
}

func TestAFailedSaveKeepsTheBlessingInMemory(t *testing.T) {
	m, w, store := env(t)
	signIn(w, 15, "Mara", false)
	store.failing = true
	deed(15, chronicle.Boss, 1)
	assert.Equal(t, []string{"road-tested"}, m.Earned(15), "an earned blessing is never refused")
	store.failing = false
	require.NoError(t, m.save())
	m2 := newModule()
	m2.store = store
	m2.load()
	assert.Equal(t, []string{"road-tested"}, m2.Earned(15), "saved with the next save")
}

func TestAnUnreadableFileIsNeverOverwritten(t *testing.T) {
	m, w, store := env(t)
	signIn(w, 16, "Mara", false)
	store.loadErr = errors.New("corrupt")
	m.load()
	deed(16, chronicle.Boss, 1)
	assert.Zero(t, store.saves, "no save while a load error is outstanding")
}

func TestLoginCatchesUpOnDeedsRecordedWhileOffline(t *testing.T) {
	m, w, _ := env(t)
	deed(17, chronicle.Relic, 1) // nobody signed in
	assert.Empty(t, m.Earned(17))
	signIn(w, 17, "Mara", false)
	m.onPlayerSpawn(events.PlayerSpawn{UserId: 17})
	assert.Equal(t, []string{"treasure-eye"}, m.Earned(17))
}

func TestPanelShowsCarriedWaitingAndProgress(t *testing.T) {
	m, w, _ := env(t)
	u := signIn(w, 18, "Mara", false)
	u.Character.Blessings = []string{"company-keeper"}
	deed(18, chronicle.Boss, 1) // earns road-tested: waiting
	deed(18, chronicle.Boss, 2) // three bosses: old-hand needs five
	p := m.panelFor(18, u)
	assert.False(t, p.Iron)
	assert.Equal(t, 5, p.Discount)
	require.Len(t, p.Carried, 1)
	assert.Equal(t, "company-keeper", p.Carried[0].ID)
	require.Len(t, p.Waiting, 1)
	assert.Equal(t, "road-tested", p.Waiting[0].ID)
	var oldHand *panelBlessing
	for i := range p.Next {
		if p.Next[i].ID == "old-hand" {
			oldHand = &p.Next[i]
		}
		assert.NotEqual(t, "road-tested", p.Next[i].ID, "an earned blessing is not still to earn")
	}
	require.NotNil(t, oldHand)
	assert.Equal(t, 3, oldHand.Have)
	assert.Equal(t, 5, oldHand.Need)

	text := m.render(18, u)
	assert.Contains(t, text, "You carry:")
	assert.Contains(t, text, "waiting for your next character")
	assert.Contains(t, text, "Still to earn:")
	assert.Contains(t, text, "(3 of 5)")
	assert.Contains(t, text, "(Iron characters only)")
	assert.Contains(t, text, "take 5% off every recruit")
	assert.False(t, strings.Contains(text, "This world has no blessings"))
}

func TestIronCharacterSeesItsStatusAndThePanelIsPushed(t *testing.T) {
	m, w, _ := env(t)
	u := signIn(w, 19, "Mara", true)
	deed(19, chronicle.Boss, 3)
	text := m.render(19, u)
	assert.Contains(t, text, "You are an Iron character")
	assert.NotContains(t, text, "(Iron characters only)", "an Iron character can earn them")
	m.push(19)
	last := w.pushes[len(w.pushes)-1]
	assert.Equal(t, "Char.Blessings", last.namespace)
	assert.True(t, last.payload.(panel).Iron)
}

func shippedBlessings(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "blessings.yaml"))
	require.NoError(t, err)
	return data
}

func TestTheModuleIsTheProviderTheCreationStepReads(t *testing.T) {
	m, w, _ := env(t)
	signIn(w, 20, "Mara", false)
	deed(20, chronicle.Boss, 1)
	assert.Equal(t, []string{"road-tested"}, blessings.EarnedFor(20), "usercommands.Start reads through the seam")
	assert.Equal(t, m.Earned(20), blessings.EarnedFor(20))
	assert.Empty(t, blessings.EarnedFor(99))
}

// Review fix: the panel is pushed at spawn (blessings waiting), before
// `start` gives them; giving them refreshes it so they show as carried.
func TestGivingBlessingsRefreshesThePanel(t *testing.T) {
	_, w, _ := env(t)
	u := signIn(w, 21, "Mara", false)
	deed(21, chronicle.Boss, 1)
	u.Character = &characters.Character{Name: "Tamsin"}
	blessings.Apply(u.Character, blessings.EarnedFor(21))
	before := len(w.pushes)
	blessings.NotifyGiven(21)
	require.Len(t, w.pushes, before+1)
	p := w.pushes[before].payload.(panel)
	require.Len(t, p.Carried, 1)
	assert.Equal(t, "road-tested", p.Carried[0].ID)
	assert.Empty(t, p.Waiting)
}
