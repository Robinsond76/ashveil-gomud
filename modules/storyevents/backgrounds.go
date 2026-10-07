package storyevents

// Phase 72 backgrounds: a character's life story gives member tags that
// story-event requirements (`tag:`) and town lines (`member_tag:`) read.
// Only the leader has a life story; companions carry none.

import (
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() { storyevents.RegisterTagSource(lifeStoryTags) }

func lifeStoryTags(leaderUserID int, memberKey string) []string {
	if memberKey != string(survival.LeaderMemberKey) {
		return nil
	}
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return nil
	}
	return lifestory.Tags(user.Character.LifeStory)
}
