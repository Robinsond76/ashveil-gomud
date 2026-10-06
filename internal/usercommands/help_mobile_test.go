package usercommands

import (
	"os"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40i: `help mobile` renders, is indexed under the configuration hub
// and answers to its aliases; the pages about the web client, click-to-walk
// and the battle screen point at it; the web client page ships the manifest
// and scripts the page describes.
func TestMobileHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "mobile" && !topic.AdminOnly {
			listed = true
			assert.Equal(t, "configuration", topic.Category)
		}
	}
	assert.True(t, listed, "the help index lists mobile")

	want, err := GetHelpContents("mobile")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	assert.Contains(t, plain, "Help for mobile")
	for _, phrase := range []string{"Game", "Map", "Here", "Company", "touch bar", "Walk to...", "Stop walking", "pennants", "Retreat",
		"pinch", "Install app", "Add to Home Screen", "does not play offline"} {
		assert.Contains(t, plain, phrase)
	}
	assert.NotContains(t, want, "</ ", "no broken tags")
	for _, alias := range []string{"phone", "touch", "touchscreen", "touch bar", "install", "install app", "add to home screen", "pwa", "web app"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help mobile", alias)
	}
	for _, topic := range []string{"webclient", "walkto", "battlescreen"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help mobile", "help %s points at help mobile", topic)
	}
}

// The page promises an installable app: the manifest names it, starts at
// the client, and its icons exist; the client page links the manifest, the
// viewport, the phone stylesheet and script.
func TestWebClientShipsTheInstallableManifest(t *testing.T) {
	const pub = "../../_datafiles/html/public/"
	manifest, err := os.ReadFile(pub + "static/images/site.webmanifest")
	require.NoError(t, err)
	for _, want := range []string{`"name": "Ashveil"`, `"display": "standalone"`, `"start_url": "/webclient-pure.html"`, `"purpose": "maskable"`} {
		assert.Contains(t, string(manifest), want)
	}
	for _, icon := range []string{"icon-192.png", "icon-512.png", "icon-maskable-512.png"} {
		assert.Contains(t, string(manifest), "../sprites/app/"+icon)
		_, err := os.Stat(pub + "static/sprites/app/" + icon)
		assert.NoError(t, err, icon)
	}
	page, err := os.ReadFile(pub + "webclient-pure.html")
	require.NoError(t, err)
	for _, want := range []string{`name="viewport" content="width=device-width`, `site.webmanifest`, `css/mobile.css`, `js/mobile.js`} {
		assert.True(t, strings.Contains(string(page), want), "webclient-pure.html has %s", want)
	}
}
