package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/tutorial"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 27d: the Tutorial package.

func sampleView() tutorial.View {
	return tutorial.View{
		Stage: 2, Stages: 8, ID: "company", Title: "Your company",
		Goal:      "Recruit two companions.",
		Checklist: []tutorial.Check{{Label: "two companions (1 of 2)", Done: false}},
		Hints:     []string{"company recruit shows who is hiring here."},
	}
}

func testTutorialFeed(views map[int]tutorial.View) (*tutorialFeed, *[]sent) {
	out := &[]sent{}
	f := newTutorialFeed()
	f.view = func(userID int) (tutorial.View, bool) {
		v, ok := views[userID]
		return v, ok
	}
	f.accepting = func(int) bool { return true }
	f.send = func(userID int, module string, payload []byte) {
		var body map[string]any
		_ = json.Unmarshal(payload, &body)
		*out = append(*out, sent{userID, module, body})
	}
	return f, out
}

func TestTutorialPayloadShape(t *testing.T) {
	body, err := json.Marshal(tutorialPayloadOf(sampleView(), true))
	require.NoError(t, err)
	assert.JSONEq(t, `{"active":true,"stage":2,"stages":8,"id":"company","title":"Your company",
		"goal":"Recruit two companions.","checklist":[{"label":"two companions (1 of 2)","done":false}],
		"hints":["company recruit shows who is hiring here."]}`, string(body))
	assert.Nil(t, tutorialPayloadOf(tutorial.View{}, false), "not in the course (the feed sends {})")
	body, _ = json.Marshal(tutorialPayloadOf(tutorial.View{Stage: 8, Stages: 8, ID: "departure", Title: "Departure"}, true))
	assert.Contains(t, string(body), `"checklist":[]`, "never null")
	assert.Contains(t, string(body), `"hints":[]`)
}

func TestTutorialSentOnChangeOnly(t *testing.T) {
	views := map[int]tutorial.View{7: sampleView()}
	f, out := testTutorialFeed(views)
	f.update(7)
	require.Len(t, *out, 1)
	assert.Equal(t, "Tutorial", (*out)[0].module)
	assert.Equal(t, "Your company", (*out)[0].body["title"])
	f.update(7)
	assert.Len(t, *out, 1, "unchanged: nothing sent")

	v := sampleView()
	v.Checklist[0] = tutorial.Check{Label: "two companions (2 of 2)", Done: true}
	views[7] = v
	f.update(7)
	require.Len(t, *out, 2, "a checklist change")

	delete(views, 7) // graduated or skipped
	f.update(7)
	require.Len(t, *out, 3)
	assert.Empty(t, (*out)[2].body, "{} once")
	f.update(7)
	assert.Len(t, *out, 3)
}

func TestTutorialNotInCourseSendsEmptyOnce(t *testing.T) {
	f, out := testTutorialFeed(map[int]tutorial.View{})
	f.update(8)
	f.update(8)
	require.Len(t, *out, 1)
	assert.Empty(t, (*out)[0].body)
}

func TestTutorialResentAfterForget(t *testing.T) {
	f, out := testTutorialFeed(map[int]tutorial.View{7: sampleView()})
	f.update(7)
	f.forget(7)
	f.update(7)
	assert.Len(t, *out, 2)
}

func TestTutorialNothingForNonGMCPConnections(t *testing.T) {
	f, out := testTutorialFeed(map[int]tutorial.View{7: sampleView()})
	accepting := false
	f.accepting = func(int) bool { return accepting }
	f.update(7)
	assert.Empty(t, *out)
	accepting = true
	f.update(7)
	assert.Len(t, *out, 1)
}

func TestTutorialWebRequestAndDespawn(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(43, 4343)
	users.SetTestUser(u)
	var got []int
	id := events.RegisterListener(GMCPTutorialRequest{}, func(e events.Event) events.ListenerReturn {
		got = append(got, e.(GMCPTutorialRequest).UserId)
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(GMCPTutorialRequest{}, id) })
	assert.True(t, gmcpModule.HandleWebGMCP(u.ConnectionId(), []byte("!!GMCP(Tutorial)")))
	events.ProcessEvents()
	assert.Equal(t, []int{43}, got)

	tutorialFeeds.mu.Lock()
	tutorialFeeds.last[44] = "{}"
	tutorialFeeds.mu.Unlock()
	events.AddToQueue(events.PlayerDespawn{UserId: 44})
	events.ProcessEvents()
	tutorialFeeds.mu.Lock()
	_, has := tutorialFeeds.last[44]
	tutorialFeeds.mu.Unlock()
	assert.False(t, has)
}
