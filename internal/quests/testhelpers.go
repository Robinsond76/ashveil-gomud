package quests

// SetTestQuest registers a quest directly, replacing any with the same id.
// For testing only.
func SetTestQuest(q *Quest) {
	quests[q.QuestId] = q
}

// RemoveTestQuest removes a quest registered with SetTestQuest. For testing
// only.
func RemoveTestQuest(questId int) {
	delete(quests, questId)
}
