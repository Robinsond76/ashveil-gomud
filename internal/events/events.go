package events

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/util"
)

type EventType string

var (
	qLock     = sync.Mutex{}
	allQueues = map[string]*Queue[Event]{}

	requeues = []requeue{}

	globalQueue  priorityQueue
	orderCounter uint64                      // global counter to maintain insertion order.
	uniqueMap    = make(map[string]struct{}) // map to enforce uniqueness

	eventDebugging bool
)

type requeue struct {
	evt      Event
	priority int
	cause    uint64
	slot     int
	typed    bool
}

// Event is the common interface for events.
type Event interface {
	Type() string
}

// Generic events are mostly used for plugins
type GenericEvent interface {
	Event
	Data(name string) any
}

// prioritizedEvent wraps an Event with a priority and an order field.
type prioritizedEvent struct {
	event    Event
	priority int    // Lower numbers indicate higher priority. Default is 0.
	order    uint64 // Used to preserve FIFO order among events with the same priority.
	cause    uint64 // The combat round that caused it (Phase 29f), or 0.
	slot     int    // The round's turn slot that caused it (Phase 82c), or 0.
	typed    bool   // It came from what a player typed (Phase 29f).
}

// UniqueEvent is implemented by events that should be unique in the queue.
type uniqueEvent interface {
	Event
	UniqueID() string
}

// PriorityQueue implements heap.Interface for *PrioritizedEvent.
type priorityQueue []*prioritizedEvent

func (pq priorityQueue) Len() int { return len(pq) }

// Less returns true if element i has a higher priority than element j.
// Here, "higher priority" means a lower numeric value. If priorities are equal,
// the one with the lower order (i.e. inserted earlier) is considered higher.
func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].priority == pq[j].priority {
		return pq[i].order < pq[j].order
	}
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*prioritizedEvent)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// Enqueue adds an event to the global queue.
// The caller can optionally pass a priority value.
// If omitted, the default priority is 0.
func AddToQueue(e Event, priority ...int) {
	add(e, currentCause.Load(), currentSlot.Load(), currentTyped.Load(), priority...)
}

// add queues an event with its cause and typed marks (Phase 29f).
func add(e Event, cause uint64, slot int64, typed bool, priority ...int) {

	qLock.Lock()
	defer qLock.Unlock()

	prio := 0
	if len(priority) > 0 {
		prio = priority[0]
	}

	// Check for uniqueness if the event implements UniqueEvent.
	if ue, ok := e.(uniqueEvent); ok {
		uid := ue.UniqueID()
		// If we already have an entry for this uniqueID, skip it!
		if _, exists := uniqueMap[uid]; exists {
			return
		}

		uniqueMap[ue.UniqueID()] = struct{}{}
	}

	orderCounter++
	pe := &prioritizedEvent{
		event:    e,
		priority: prio,
		order:    orderCounter,
		cause:    cause,
		slot:     int(slot),
		typed:    typed,
	}

	if eventDebugging {
		fmt.Println(`events.AddToQueue`, "type:", e.Type(), `priority`, prio, `order`, orderCounter)
	}

	heap.Push(&globalQueue, pe)
}

// Same as AddToQueue but avoids a mutex lock for optimization purposes
// Should only be used when a mutex lock is already held
func reAddToQueue(e Event, cause uint64, slot int, typed bool, priority ...int) {

	prio := 0
	if len(priority) > 0 {
		prio = priority[0]
	}

	// Check for uniqueness if the event implements UniqueEvent.
	if ue, ok := e.(uniqueEvent); ok {
		uid := ue.UniqueID()
		// If we already have an entry for this uniqueID, skip it!
		if _, exists := uniqueMap[uid]; exists {
			return
		}

		uniqueMap[ue.UniqueID()] = struct{}{}
	}

	orderCounter++
	pe := &prioritizedEvent{
		event:    e,
		priority: prio,
		order:    orderCounter,
		cause:    cause,
		slot:     slot,
		typed:    typed,
	}

	if eventDebugging {
		fmt.Println(`events.reAddToQueue`, "type:", e.Type(), `priority:`, prio, `order:`, orderCounter)
	}

	heap.Push(&globalQueue, pe)
}

func addToRequeue(e Event, cause uint64, slot int, typed bool, priority ...int) {
	qLock.Lock()
	defer qLock.Unlock()

	prio := 0
	if len(priority) > 0 {
		prio = priority[0]
	}
	requeues = append(requeues, requeue{
		evt:      e,
		priority: prio,
		cause:    cause,
		slot:     slot,
		typed:    typed,
	})
}

// ProcessEvents runs the event loop until the queue is empty.
// It processes events one at a time in order of priority.
// Any events enqueued (even from within a handler) will be picked up in order.
func ProcessEvents() {

	// Since this is intended to run frequently and quickly
	// Only sample the runtime 1 in 100 times
	eventCounter := 0
	if eventDebugging || rand.Intn(100) == 0 {
		start := time.Now()
		defer func() {
			util.TrackTime(`events.ProcessEvents()`, time.Since(start).Seconds())
			if eventDebugging {
				if time.Since(start).Seconds() > 0.00125 {
					fmt.Println(`events.ProcessEvents`, "events handled:", eventCounter, "time taken:", time.Since(start).Seconds())
				}
			}
		}()
	}

	qLock.Lock()

	// Requeues are a special group that has been deferred to the next processevents loop
	// They are added back into the event queue at the top of the process events function
	for _, itm := range requeues {
		reAddToQueue(itm.evt, itm.cause, itm.slot, itm.typed, itm.priority)
	}
	requeues = requeues[:0]

	var evtResult ListenerReturn
	for {

		if globalQueue.Len() < 1 {
			break
		}

		pe := heap.Pop(&globalQueue).(*prioritizedEvent)

		if eventDebugging {
			eventCounter++
			fmt.Println(`events.ProcessEvents`, "type:", pe.event.Type(), `remain:`, globalQueue.Len())
		}

		// If this is a unique event, remove it from the uniqueMap.
		if ue, ok := pe.event.(uniqueEvent); ok {
			delete(uniqueMap, ue.UniqueID())
		}

		qLock.Unlock()

		// Phase 29f: while a caused event is dispatched, what its
		// listeners queue inherits its cause.
		prevCause := currentCause.Swap(pe.cause)
		prevSlot := currentSlot.Swap(int64(pe.slot))
		prevTyped := currentTyped.Swap(pe.typed)
		evtResult = DoListeners(pe.event)
		currentCause.Store(prevCause)
		currentSlot.Store(prevSlot)
		currentTyped.Store(prevTyped)
		if evtResult == CancelAndRequeue {
			addToRequeue(pe.event, pe.cause, pe.slot, pe.typed, pe.priority)
		}

		qLock.Lock()

	}

	qLock.Unlock()
}

func SetDebug(on bool) {
	qLock.Lock()
	defer qLock.Unlock()
	eventDebugging = on
}

// Initialize the priority queue.
func init() {
	heap.Init(&globalQueue)
}
