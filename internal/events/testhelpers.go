package events

// ClearQueueForTest drops every queued and requeued event without
// dispatching it, so a test starts without what an earlier test enqueued and
// never processed. It leaves registered listeners alone: package init
// functions register the production ones.
func ClearQueueForTest() {
	qLock.Lock()
	defer qLock.Unlock()
	globalQueue = globalQueue[:0]
	requeues = requeues[:0]
	clear(uniqueMap)
}
