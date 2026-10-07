package usercommands

// Ashveil Phase 72a: the looks and life story steps, driven through the real
// start, appearance, lifestory and creation commands.

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/tutorial"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const tutorialQuestion = `Would you like to skip the tutorial?`

// creationWorld loads the shipped world's looks, life stories, items,
// skills and races, and stops saves reaching disk.
func creationWorld(t *testing.T) {
	t.Helper()
	useWorld(t, "default")
	races.LoadDataFiles()
	items.LoadDataFiles()
	skills.LoadDataFiles()
	require.NoError(t, appearance.Load())
	require.NoError(t, lifestory.Load())
	require.True(t, appearance.Available())
	require.True(t, lifestory.Available())
	tutorial.SetProvider(nil)
	previous := saveUserFn
	saveUserFn = func(*users.UserRecord) error { return nil }
	t.Cleanup(func() {
		saveUserFn = previous
		appearance.SetData(nil)
		lifestory.SetData(nil)
	})
}

type creationUser struct {
	t     *testing.T
	u     *users.UserRecord
	room  *rooms.Room
	sent  []string
	asked []string
}

func newCreationUser(t *testing.T, id, raceID int) *creationUser {
	t.Helper()
	u := users.NewUserRecord(id, 1)
	u.Username = "acct" + str(id)
	u.Character.Name = "Mara"
	u.Character.RaceId = raceID
	u.Character.RoomId = -1
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(id); creation.Clear(id) })
	freshEvents(t)
	cu := &creationUser{t: t, u: u, room: &rooms.Room{RoomId: -1, Title: "The Void"}}
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == id {
			cu.sent = append(cu.sent, m.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lid) })
	return cu
}

func (c *creationUser) text() string { return strings.Join(c.sent, "\n") }

// start runs `start` once.
func (c *creationUser) start() {
	c.t.Helper()
	_, err := Start("", c.u, c.room, 0)
	require.NoError(c.t, err)
	events.ProcessEvents()
}

// pending is the open question's text.
func (c *creationUser) pending() string {
	p := c.u.GetPrompt()
	if p == nil {
		return ``
	}
	if q := p.GetNextQuestion(); q != nil {
		return q.Question
	}
	return ``
}

// answer types a response to the open question and re-runs the command the
// way the world does (a prompt re-invokes its command).
func (c *creationUser) answer(command, rest, response string) {
	c.t.Helper()
	p := c.u.GetPrompt()
	require.NotNil(c.t, p, "no open prompt for %q", response)
	q := p.GetNextQuestion()
	require.NotNil(c.t, q, "no open question for %q", response)
	c.asked = append(c.asked, q.Question)
	q.Answer(response)
	c.sent = nil
	require.Equal(c.t, `start`, command, "only start is driven through answer")
	c.start()
}

// runTo answers every question with the given responses (by question text
// prefix, else "1"/blank) until the open question is stop or the prompt ends.
func (c *creationUser) runTo(command, rest, stop string, responses map[string]string) {
	c.t.Helper()
	for i := 0; i < 80; i++ {
		q := c.pending()
		if q == `` || q == stop {
			return
		}
		resp := `1`
		for prefix, r := range responses {
			if strings.HasPrefix(q, prefix) {
				resp = r
			}
		}
		c.answer(command, rest, resp)
	}
	c.t.Fatalf("flow did not settle; open question %q", c.pending())
}

func TestStartAsksLooksThenLifeStoryThenTutorial(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7201, 1)
	cu.start()

	// pronouns is the first looks question, with the numbered list shown
	assert.Equal(t, "How will others speak of you?", cu.pending())
	assert.Contains(t, cu.text(), "he/him")
	view, ok := creation.Get(7201)
	require.True(t, ok)
	assert.Equal(t, "looks", view.Step)
	assert.Equal(t, "pronouns", view.Key)
	assert.Equal(t, creation.KindChoice, view.Kind)
	assert.Len(t, view.Options, 3)
	assert.Equal(t, "new", view.Mode)
	assert.Equal(t, 1, view.Number)
	assert.Equal(t, 15, view.Total, "ten traits, marks, the free line, three stages")

	cu.runTo("start", "", `Keep this character as described?`, map[string]string{
		"How will others":       "2", // she
		"How tall":              "tall",
		"Do you carry a mark?":  "burn scar",
		"Do you carry a second": "none",
		"Add a line":            "She never speaks of the fire.",
	})
	assert.Equal(t, "Keep this character as described?", cu.pending())
	// the order of questions: ten traits, mark, second mark, line, three stages
	require.GreaterOrEqual(t, len(cu.asked), 13)
	assert.Equal(t, "How will others speak of you?", cu.asked[0])
	assert.Equal(t, "How old are you?", cu.asked[1])
	assert.Equal(t, "Do you carry a mark?", cu.asked[10])
	assert.Equal(t, "Do you carry a second mark?", cu.asked[11])
	assert.True(t, strings.HasPrefix(cu.asked[12], "Add a line of your own"))
	assert.Equal(t, "Where were you born?", cu.asked[13])

	// nothing is granted before the summary is confirmed
	c := cu.u.Character
	assert.False(t, c.HasLooks())
	assert.False(t, c.HasLifeStory())
	assert.Empty(t, c.Items)
	assert.Contains(t, cu.text(), "You look like this")
	assert.Contains(t, cu.text(), "She never speaks of the fire.")
	assert.Contains(t, cu.text(), "It gives you:")
	sumView, _ := creation.Get(7201)
	assert.Equal(t, "summary", sumView.Step)
	assert.Contains(t, sumView.Preview, "woman")
	assert.NotEmpty(t, sumView.Backstory)

	cu.answer("start", "", "confirm")
	assert.Equal(t, tutorialQuestion, cu.pending(), "the tutorial question follows the summary")
	assert.True(t, c.HasLooks())
	assert.True(t, c.HasLifeStory())
	assert.True(t, c.CreationOffered())
	assert.Equal(t, "she", c.Pronouns)
	assert.Equal(t, "she", c.CombatPronouns().Subject)
	assert.Contains(t, c.Description, "tall")
	assert.Contains(t, c.Description, "woman")
	assert.Contains(t, c.Description, "A burn scar runs across the back of her left hand.")
	assert.True(t, strings.HasSuffix(c.Description, "She never speaks of the fire."))
	_, still := creation.Get(7201)
	assert.False(t, still, "the creation view ends with the steps")
}

func TestLifeStoryEffectsApplyOnce(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7202, 1)
	cu.start()
	before := cu.u.Character.Stats
	cu.runTo("start", "", `Keep this character as described?`, map[string]string{
		"Where were you born": "river-farms", // vitality or strength
		"The river farms":     "vitality",
		"How did you grow":    "farmhand", // strength or vitality
		"A farmhand":          "strength",
		"What did you do":     "soldier", // strength or vitality, keepsake
		"A soldier":           "vitality",
		"Add a line":          "",
	})
	c := cu.u.Character
	assert.Empty(t, c.Items, "nothing is granted before confirming")
	assert.Equal(t, 0, c.StatMod("vitality"))

	// redo the story: answers reset, still nothing granted
	cu.answer("start", "", "redo story")
	assert.Equal(t, "Where were you born?", cu.pending())
	assert.Empty(t, c.Items)
	cu.runTo("start", "", `Keep this character as described?`, map[string]string{
		"Where were you born": "river-farms", "The river farms": "vitality",
		"How did you grow": "farmhand", "A farmhand": "strength",
		"What did you do": "soldier", "A soldier": "vitality",
	})
	cu.answer("start", "", "confirm")

	assert.Equal(t, "soldier", c.Background())
	require.Len(t, c.Items, 1, "one keepsake, however often the story was redone")
	assert.Equal(t, 30400, c.Items[0].ItemId)
	assert.Contains(t, cu.text(), "goes into your pack")
	assert.Equal(t, 2, c.StatMod("vitality"))
	assert.Equal(t, 1, c.StatMod("strength"))
	assert.Equal(t, 0, c.StatMod("speed"))
	assert.Equal(t, 3, c.StatMod("vitality")+c.StatMod("strength")+c.StatMod("smarts")+c.StatMod("speed")+c.StatMod("mysticism")+c.StatMod("perception"))
	c.RecalculateStats()
	assert.Equal(t, before.Vitality.Training, c.Stats.Vitality.Training, "no training points are spent or added")
	assert.Equal(t, 2, c.Stats.Vitality.Mods)

	// a second application is refused and changes nothing
	_, err := c.ApplyLifeStory(lifestory.Current(), lifestory.Picks(c.LifeStory))
	assert.ErrorIs(t, err, characters.ErrLifeStorySet)
	assert.Len(t, c.Items, 1)
	assert.Equal(t, 2, c.StatMod("vitality"))

	// relog and restart: the saved character reads back the same
	raw, err := yaml.Marshal(c)
	require.NoError(t, err)
	var back characters.Character
	require.NoError(t, yaml.Unmarshal(raw, &back))
	assert.Equal(t, c.Looks, back.Looks)
	assert.Equal(t, c.LifeStory, back.LifeStory)
	assert.Equal(t, c.Description, back.Description)
	assert.Equal(t, "soldier", back.Background())
	assert.Equal(t, 2, back.StatMod("vitality"))
	assert.True(t, back.CreationOffered())
	assert.Len(t, back.Items, 1)
}

func TestTradeSkillFamiliarity(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7203, 1)
	cu.start()
	cu.runTo("start", "", `Keep this character as described?`, map[string]string{
		"What did you do": "A labourer",
	})
	cu.answer("start", "", "confirm")
	c := cu.u.Character
	assert.Equal(t, 1, c.GetSkillLevel("cooking"))
	require.Len(t, c.Items, 1)
	assert.Equal(t, 30404, c.Items[0].ItemId)
	assert.Equal(t, 1, items.GetItemSpec(30404).Value, "keepsakes have a merchant-floor value only")
}

func TestRedoLooksChangesDescription(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7204, 1)
	cu.start()
	cu.runTo("start", "", `Keep this character as described?`, map[string]string{"How will others": "he"})
	cu.answer("start", "", "redo looks")
	assert.Equal(t, "How will others speak of you?", cu.pending())
	cu.runTo("start", "", `Keep this character as described?`, map[string]string{"How will others": "they"})
	cu.answer("start", "", "confirm")
	c := cu.u.Character
	assert.Equal(t, "they", c.Pronouns)
	assert.Contains(t, c.Description, "person")
	assert.NotContains(t, c.Description, " man")
}

func TestBackRedoesTheStepAndBadAnswersAreAskedAgain(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7205, 1)
	cu.start()
	cu.answer("start", "", "she") // pronouns
	cu.answer("start", "", "7")   // not an age
	assert.Equal(t, "How old are you?", cu.pending())
	assert.Contains(t, cu.text(), "That isn't one of the options.")
	cu.answer("start", "", "forty")
	assert.Equal(t, "How tall are you?", cu.pending())
	// back wipes the looks picks, so the step begins again
	cu.answer("start", "", "back")
	assert.Equal(t, "How will others speak of you?", cu.pending())
	view, _ := creation.Get(7205)
	assert.Empty(t, view.Picks)
	assert.False(t, view.CanBack, "nothing to go back over at the first question")
	// back with nothing picked is just a bad answer
	cu.answer("start", "", "back")
	assert.Equal(t, "How will others speak of you?", cu.pending())
}

func TestFreeLineFiltering(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7207, 1)
	cu.start()
	var lineQuestion string
	for i := 0; i < 20 && lineQuestion == ``; i++ {
		if q := cu.pending(); strings.HasPrefix(q, "Add a line") {
			lineQuestion = q
			break
		}
		cu.answer("start", "", "1")
	}
	require.NotEmpty(t, lineQuestion)

	cu.answer("start", "", strings.Repeat("a", 200))
	assert.Equal(t, lineQuestion, cu.pending(), "a line over 160 characters is asked again")
	assert.Contains(t, cu.text(), "isn't allowed")

	cu.answer("start", "", `Scarred <ansi fg="red">badly</ansi>.`)
	assert.Equal(t, "Where were you born?", cu.pending())
	view, _ := creation.Get(7207)
	assert.Equal(t, `Scarred ansi fg="red"badly/ansi.`, view.Picks["line"])
	assert.NotContains(t, view.Preview, "<")
}

func TestElvesSeeTheirOwnAgeBand(t *testing.T) {
	creationWorld(t)
	has := func(raceID int, id int) bool {
		cu := newCreationUser(t, id, raceID)
		cu.start()
		cu.answer("start", "", "1")
		view, _ := creation.Get(id)
		require.Equal(t, "age", view.Key)
		for _, o := range view.Options {
			if o.ID == "ageless" {
				return true
			}
		}
		return false
	}
	assert.False(t, has(1, 7208), "a human")
	elf := races.GetRace(2)
	require.NotNil(t, elf)
	assert.True(t, has(2, 7209), "an elf")
}

func TestStartSkipsLooksAndStoryWithoutData(t *testing.T) {
	appearance.SetData(nil)
	lifestory.SetData(nil)
	tutorial.SetProvider(nil)
	cu := newCreationUser(t, 7210, 1)
	cu.start()
	assert.Equal(t, tutorialQuestion, cu.pending(), "a world without looks.yaml goes straight on")
}

func TestStartDoesNotAskAgainAfterConfirm(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7211, 1)
	cu.start()
	cu.runTo("start", "", `Keep this character as described?`, nil)
	cu.answer("start", "", "confirm")
	require.Equal(t, tutorialQuestion, cu.pending())
	// answering the tutorial question re-runs start; creation must not return
	cu.u.GetPrompt().GetNextQuestion().Answer("no")
	cu.start()
	_, open := creation.Get(7211)
	assert.False(t, open)
	assert.NotContains(t, cu.pending(), "Keep this character")
}

func innRoom() *rooms.Room {
	return &rooms.Room{RoomId: 72001, Title: "The Wayhouse", Tags: []string{"inn"}}
}

func TestAppearanceEditOnlyAtAnInn(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7212, 1)
	c := cu.u.Character
	c.RoomId = 72002
	require.NoError(t, c.SetLooks(appearance.Current(), func() appearance.Looks {
		l := appearance.Looks{}
		for _, id := range appearance.SingleTraits {
			l[id] = appearance.Current().Trait(id).Options[0].ID
		}
		l[appearance.TraitMark] = appearance.NoMark
		return l
	}()))
	original := c.Description
	c.LifeStory = map[string]string{lifestory.StageTrade: "scholar"}

	// refused away from an inn
	street := &rooms.Room{RoomId: 72002, Title: "A Street"}
	_, err := Appearance("edit", cu.u, street, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.text(), "only change your looks at an inn")
	assert.Nil(t, cu.u.GetPrompt())

	// allowed at an inn: runs the looks steps only, never the life story
	cu.sent = nil
	_, err = Appearance("edit", cu.u, innRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	require.Equal(t, "How will others speak of you?", cu.pending())
	view, _ := creation.Get(7212)
	assert.Equal(t, "edit", view.Mode)
	assert.Equal(t, 12, view.Total, "looks only")

	room := innRoom()
	for i := 0; i < 40 && cu.pending() != `Keep this character as described?`; i++ {
		resp := "1"
		if strings.HasPrefix(cu.pending(), "How will others") {
			resp = "they"
		}
		cu.u.GetPrompt().GetNextQuestion().Answer(resp)
		_, err = Appearance("edit", cu.u, room, 0)
		require.NoError(t, err)
		events.ProcessEvents()
	}
	require.Equal(t, "Keep this character as described?", cu.pending())
	assert.Equal(t, original, c.Description, "nothing changes before the summary is confirmed")
	cu.u.GetPrompt().GetNextQuestion().Answer("confirm")
	_, err = Appearance("edit", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()

	assert.NotEqual(t, original, c.Description)
	assert.Equal(t, "they", c.Pronouns)
	assert.Equal(t, "scholar", c.Background(), "the life story stays as it was")
	assert.Nil(t, cu.u.GetPrompt())
	assert.Empty(t, c.Items, "an edit grants nothing")

	// leaving the inn mid-edit ends it
	_, err = Appearance("edit", cu.u, room, 0)
	require.NoError(t, err)
	require.NotNil(t, cu.u.GetPrompt())
	cu.u.GetPrompt().GetNextQuestion().Answer("1")
	cu.sent = nil
	_, err = Appearance("edit", cu.u, street, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Nil(t, cu.u.GetPrompt())
	assert.Contains(t, cu.text(), "only change your looks at an inn")
}

func TestAppearanceShowsLooksInWords(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7213, 1)
	c := cu.u.Character
	c.RoomId = 72002
	d := appearance.Current()
	l := appearance.Looks{}
	for _, id := range appearance.SingleTraits {
		l[id] = d.Trait(id).Options[0].ID
	}
	l[appearance.TraitMark] = "burn"
	require.NoError(t, c.SetLooks(d, l))
	_, err := Appearance("", cu.u, &rooms.Room{RoomId: 72002}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	text := tagPattern.ReplaceAllString(cu.text(), "")
	assert.Contains(t, text, "sharp-featured man, grey-eyed, pale-skinned")
	assert.Contains(t, text, "A burn scar runs across the")
	assert.Contains(t, text, "age:")
	assert.Contains(t, text, "marks:     burn scar")
	assert.Contains(t, text, "appearance edit")
}

func TestLifestoryChooseWritesOnceAndShowsTheStory(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7214, 1)
	c := cu.u.Character
	c.RoomId = 72002
	room := &rooms.Room{RoomId: 72002}

	_, err := Lifestory("", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.text(), "hasn't been written")

	_, err = Lifestory("choose", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	require.Equal(t, "Where were you born?", cu.pending())
	view, _ := creation.Get(7214)
	assert.Equal(t, "story", view.Mode)
	assert.Equal(t, 3, view.Total)

	for i := 0; i < 30 && cu.pending() != `Keep this character as described?`; i++ {
		resp := "1"
		if q := cu.pending(); strings.Contains(q, "sharpen") {
			resp = "1"
		}
		cu.u.GetPrompt().GetNextQuestion().Answer(resp)
		_, err = Lifestory("choose", cu.u, room, 0)
		require.NoError(t, err)
		events.ProcessEvents()
	}
	require.Equal(t, "Keep this character as described?", cu.pending())
	cu.u.GetPrompt().GetNextQuestion().Answer("confirm")
	_, err = Lifestory("choose", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()

	assert.True(t, c.HasLifeStory())
	assert.False(t, c.HasLooks(), "the story step doesn't touch looks")
	assert.Nil(t, cu.u.GetPrompt())
	require.Len(t, c.Items, 1)

	// shown, and locked afterwards
	cu.sent = nil
	_, err = Lifestory("", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.text(), "Mara was born")
	cu.sent = nil
	_, err = Lifestory("choose", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.text(), "already written")
	assert.Nil(t, cu.u.GetPrompt())
	assert.Len(t, c.Items, 1)
}

func TestLifestoryGlanceAtAnotherPlayer(t *testing.T) {
	creationWorld(t)
	d := lifestory.Current()
	text := lifestoryGlance(d, "Mara", lifestory.Picks{
		lifestory.StageHomeland: "coast", lifestory.StageUpbringing: "street-child", lifestory.StageTrade: "outlaw",
	})
	assert.Contains(t, text, "The coast, A street child, An outlaw")
	assert.Contains(t, lifestoryGlance(d, "Mara", nil), "keeps their past to themselves")
}

// An existing character is asked once; "later" is remembered.
func TestCreationOfferToExistingCharacter(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7215, 1)
	c := cu.u.Character
	c.RoomId = 72002
	room := &rooms.Room{RoomId: 72002}
	assert.False(t, c.CreationOffered())

	_, err := Creation("", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.pending(), "no looks or life story yet")
	view, _ := creation.Get(7215)
	assert.True(t, view.CanSkip)
	assert.Equal(t, "legacy", view.Mode)

	cu.u.GetPrompt().GetNextQuestion().Answer("later")
	_, err = Creation("", cu.u, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.True(t, c.CreationOffered(), "putting it off is remembered")
	assert.Nil(t, cu.u.GetPrompt())
	assert.False(t, c.HasLooks())
	assert.Contains(t, cu.text(), "appearance edit")
	assert.Contains(t, cu.text(), "lifestory choose")

	// remembered across a restart
	raw, err := yaml.Marshal(c)
	require.NoError(t, err)
	var back characters.Character
	require.NoError(t, yaml.Unmarshal(raw, &back))
	assert.True(t, back.CreationOffered())
}

func TestCreationSkipAnswerAtAnyQuestionAndFullFlow(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7216, 1)
	c := cu.u.Character
	c.RoomId = 72002
	room := &rooms.Room{RoomId: 72002}

	run := func(answer string) {
		_, err := Creation("", cu.u, room, 0)
		require.NoError(t, err)
		events.ProcessEvents()
		_ = answer
	}
	run("")
	cu.u.GetPrompt().GetNextQuestion().Answer("now")
	run("")
	require.Equal(t, "How will others speak of you?", cu.pending())
	cu.u.GetPrompt().GetNextQuestion().Answer("skip")
	run("")
	assert.True(t, c.CreationOffered())
	assert.Nil(t, cu.u.GetPrompt())
	assert.False(t, c.HasLooks())

	// the full flow for a legacy character commits looks and story and saves
	saved := 0
	saveUserFn = func(*users.UserRecord) error { saved++; return nil }
	run("")
	cu.u.GetPrompt().GetNextQuestion().Answer("now")
	for i := 0; i < 60 && cu.pending() != `Keep this character as described?`; i++ {
		run("")
		if cu.pending() == `Keep this character as described?` {
			break
		}
		cu.u.GetPrompt().GetNextQuestion().Answer("1")
	}
	run("")
	require.Equal(t, "Keep this character as described?", cu.pending())
	cu.u.GetPrompt().GetNextQuestion().Answer("confirm")
	run("")
	assert.True(t, c.HasLooks())
	assert.True(t, c.HasLifeStory())
	assert.Equal(t, 1, saved, "a stand-alone commit saves the user")
	assert.Nil(t, cu.u.GetPrompt())

	// nothing is left to write
	cu.sent = nil
	run("")
	assert.Contains(t, cu.text(), "nothing left to write")
}

func TestCreationRefusedInTheVoid(t *testing.T) {
	creationWorld(t)
	cu := newCreationUser(t, 7217, 1)
	_, err := Creation("", cu.u, &rooms.Room{RoomId: -1}, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, cu.text(), "Finish creating your character first")
	assert.Nil(t, cu.u.GetPrompt())
}

func TestBackgroundHelperForOtherPhases(t *testing.T) {
	c := &characters.Character{}
	assert.Equal(t, "", c.Background())
	c.LifeStory = lifestory.PicksWithBackground("noble")
	assert.Equal(t, "noble", c.Background())
}
