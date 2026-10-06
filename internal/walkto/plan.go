// Package walkto plans the Phase 40d click-to-walk route: a breadth-first
// search over rooms the player has already visited, through exits they can
// use, capped at MaxSteps. It is pure over a Graph; modules/walkto supplies
// the live world and drives the steps, and modules/gmcp reads the active
// route for the web map. Nothing here advances game time.
package walkto

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MaxSteps is the longest route walkto will plan (approved 2026-10-05).
const MaxSteps = 60

// Block says why an exit is not part of a route.
type Block int

const (
	BlockNone    Block = iota
	BlockSecret        // a secret exit the player has not found or used
	BlockJourney       // a travel-profile exit: a real journey, never a step
	BlockMessage       // an exit with an exit message and delay
)

// Exit is one way out of a room, as the player sees it.
type Exit struct {
	Name   string
	To     int
	Block  Block
	Locked bool // locked now
	HasKey bool // the player can open it with a key they carry
}

// Node is a room as the planner needs it.
type Node struct {
	ID      int
	Zone    string
	Title   string
	Legend  string
	Visited bool
	Exits   []Exit
}

// Graph loads a room; false when it does not exist.
type Graph func(roomID int) (Node, bool)

// Step is one move on a route.
type Step struct {
	Exit string
	To   int
}

// Route is a planned walk. Path starts at the room the player stands in.
type Route struct {
	Target int
	Path   []int
	Steps  []Step
}

var (
	ErrHere         = errors.New("you are already there")
	ErrUnvisited    = errors.New("you have not been there")
	ErrNoPath       = errors.New("no known way there")
	ErrLocked       = errors.New("a locked door is in the way")
	ErrUnknownPlace = errors.New("no visited place by that name")
)

// TooFarError reports a route longer than MaxSteps.
type TooFarError struct{ Steps int }

func (e TooFarError) Error() string {
	return fmt.Sprintf("that is %d steps away; walkto goes at most %d at a time", e.Steps, MaxSteps)
}

type hop struct {
	from int
	exit string
}

// Tree is the result of one search: the steps to every reachable visited room.
type Tree struct {
	Start int
	Dist  map[int]int
	via   map[int]hop
}

// Search walks outward from start through visited rooms. With ignoreLocks a
// locked exit counts as open (used to explain a refusal). It does not stop at
// MaxSteps, so the caller can say how far a room is.
func Search(g Graph, start int, ignoreLocks bool) Tree {
	t := Tree{Start: start, Dist: map[int]int{start: 0}, via: map[int]hop{}}
	queue := []int{start}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		node, ok := g(id)
		if !ok {
			continue
		}
		exits := append([]Exit(nil), node.Exits...)
		sort.Slice(exits, func(i, j int) bool { return exits[i].Name < exits[j].Name })
		for _, e := range exits {
			if e.Block != BlockNone || (e.Locked && !e.HasKey && !ignoreLocks) {
				continue
			}
			if _, seen := t.Dist[e.To]; seen {
				continue
			}
			next, ok := g(e.To)
			if !ok || !next.Visited {
				continue
			}
			t.Dist[e.To] = t.Dist[id] + 1
			t.via[e.To] = hop{from: id, exit: e.Name}
			queue = append(queue, e.To)
		}
	}
	return t
}

// RouteTo builds the route to a reached room.
func (t Tree) RouteTo(target int) (Route, bool) {
	if _, ok := t.Dist[target]; !ok {
		return Route{}, false
	}
	var steps []Step
	for at := target; at != t.Start; {
		h := t.via[at]
		steps = append([]Step{{Exit: h.exit, To: at}}, steps...)
		at = h.from
	}
	path := []int{t.Start}
	for _, s := range steps {
		path = append(path, s.To)
	}
	return Route{Target: target, Path: path, Steps: steps}, true
}

// Plan routes from one room to another. The target must be a visited room.
func Plan(g Graph, from, to int) (Route, error) {
	if from == to {
		return Route{}, ErrHere
	}
	node, ok := g(to)
	if !ok || !node.Visited {
		return Route{}, ErrUnvisited
	}
	if route, ok := Search(g, from, false).RouteTo(to); ok {
		if len(route.Steps) > MaxSteps {
			return Route{}, TooFarError{Steps: len(route.Steps)}
		}
		return route, nil
	}
	if _, ok := Search(g, from, true).RouteTo(to); ok {
		return Route{}, ErrLocked
	}
	return Route{}, ErrNoPath
}

// Resolve turns what the player typed into a visited room: a room number, or
// a landmark word matched against the legend and then the title of visited
// rooms in the zone they stand in. The nearest match wins.
func Resolve(g Graph, from int, query string) (int, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0, ErrUnknownPlace
	}
	here, ok := g(from)
	if !ok {
		return 0, ErrUnknownPlace
	}
	strict, loose := Search(g, from, false), Search(g, from, true)
	best := func(match func(Node) bool) (int, bool) {
		for _, tree := range []Tree{strict, loose} {
			id, bestDist := 0, -1
			for room, dist := range tree.Dist {
				if room == from {
					continue
				}
				node, ok := g(room)
				if !ok || node.Zone != here.Zone || !match(node) {
					continue
				}
				if bestDist < 0 || dist < bestDist || (dist == bestDist && room < id) {
					id, bestDist = room, dist
				}
			}
			if bestDist >= 0 {
				return id, true
			}
		}
		return 0, false
	}
	if id, ok := best(func(n Node) bool { return strings.ToLower(n.Legend) == query }); ok {
		return id, nil
	}
	if id, ok := best(func(n Node) bool { return strings.HasPrefix(strings.ToLower(n.Title), query) }); ok {
		return id, nil
	}
	if id, ok := best(func(n Node) bool { return strings.Contains(strings.ToLower(n.Title), query) }); ok {
		return id, nil
	}
	return 0, ErrUnknownPlace
}
