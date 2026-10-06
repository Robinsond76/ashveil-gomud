package main

// Phase 44b: the world smoke playtest. Phase 44's TestLiveSmoke never leaves
// the tutorial, so none of the world's loops ran against a real server. This
// plays a skipped-tutorial Warrior through them over telnet: recruiting at
// the Waymark Inn, the Old Kings Road journey (with its fallen tree), a
// random encounter against road brigands, battle loot, gathering, a camp,
// a restart, salvage at the Frostfang armorer, a market sale and a night at
// the Dunmar inn.
//
// It takes a few minutes of real time (a journey, a fight, two timed gathers
// and an inn stay), so it only runs when asked:
//
//	make smoke-world   (or: ASHVEIL_LIVE_SMOKE=1 go test -run TestLiveSmokeWorld -timeout 20m -v .)
//
// Three things about the shipped world are bent for it, in the test's own
// copy only (see worldSmokeOptions): new characters start at Dunmar's West
// Gate (the one shipped route starts there), the Old Kings Road gets an
// encounter table of two level-1 road brigands that always spring, and
// the character is made an admin so it can teleport to Frostfang, which no
// shipped road reaches yet.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// worldSmokeOptions returns the server options for the world smoke.
func worldSmokeOptions(t *testing.T) smokeOptions {
	return smokeOptions{
		extraConfig: "SpecialRooms:\n  StartRoom: 2001\n",
		patch: func(world string) {
			road := filepath.Join(world, "rooms", "old_kings_road")
			appendFile(t, filepath.Join(road, "zone-config.yaml"), `
# Phase 44b smoke fixture: a pair of level-1 brigands on every eligible entry.
encounters:
  band:
    low: 1
    high: 1
  entrychance: 100
  tables:
    road:
      - id: brigand-pair
        weight: 100
        text: Brigands step out of the pines onto the road.
        members:
          - mobid: 86
            count: 2
`)
			// Unarmed, so a lucky critical cannot fell a level-1 companion and
			// make the run depend on dice; the fight itself is still real.
			brigand := filepath.Join(world, "mobs", "old_kings_road", "86-road_brigand.yaml")
			b, err := os.ReadFile(brigand)
			if err != nil {
				t.Fatal(err)
			}
			const cudgel = "    weapon:\n      itemid: 10015\n"
			if !strings.Contains(string(b), cudgel) {
				t.Fatalf("the road brigand no longer carries a cudgel to take away:\n%s", b)
			}
			if err := os.WriteFile(brigand, []byte(strings.Replace(string(b), cudgel, "", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			appendFile(t, filepath.Join(road, "2002.yaml"), `
encounter:
  enabled: true
  table: road
  chance: 100
`)
		},
	}
}

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

// makeAdmin promotes a saved account while the server is stopped.
func makeAdmin(t *testing.T, srv *smokeServer, userID string) {
	t.Helper()
	path := filepath.Join(srv.dir, "_datafiles", "world", "default", "users", userID+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "role: user") {
		t.Fatalf("%s has no `role: user` line:\n%.300s", path, b)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), "role: user", "role: admin", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// doAll sends a command and returns everything it printed, up to the prompt
// that follows its echo (do stops at the first match of a pattern, which
// leaves the rest of a multi-line answer unread).
func doAll(m *mudClient, line string) string {
	m.t.Helper()
	m.drain(300 * time.Millisecond) // earlier output must not satisfy the echo
	m.send(line)
	m.expect(regexp.QuoteMeta(line), 20*time.Second)
	return m.expect(smokePrompt, 30*time.Second)
}

// assertNoServerErrors fails on text a player must never see.
func assertNoServerErrors(t *testing.T, who, txt string) {
	t.Helper()
	for _, bad := range []string{"not recognized", "looks a little confused", "panic:", "runtime error"} {
		if strings.Contains(txt, bad) {
			t.Errorf("%s saw %q:\n%s", who, bad, snippetAround(txt, bad))
		}
	}
}

func TestLiveSmokeWorld(t *testing.T) {
	if os.Getenv("ASHVEIL_LIVE_SMOKE") == "" {
		t.Skip("set ASHVEIL_LIVE_SMOKE=1 (or run `make smoke-world`) to play a live server")
	}
	if testing.Short() {
		t.Skip("live smoke playtest skipped in -short mode")
	}

	srv := newSmokeServerWith(t, worldSmokeOptions(t))
	t.Cleanup(func() {
		srv.kill()
		if t.Failed() {
			t.Logf("server output:\n%s", srv.tailLog())
		}
	})
	srv.start()

	var p *mudClient
	step := func(name string, fn func()) {
		t.Helper()
		t.Logf("== %s", name)
		fn()
	}
	// Whoever is signed in at the end, and every earlier connection, is
	// checked for text a player must never see.
	var clients []*mudClient
	t.Cleanup(func() {
		for _, c := range clients {
			assertNoServerErrors(t, c.name, c.text())
		}
	})
	dial := func(name string) *mudClient {
		c := dialMud(t, name, srv.port)
		clients = append(clients, c)
		return c
	}

	step("a new warrior wakes at Dunmar's West Gate", func() {
		p = dial("world1")
		p.registerArriving("world1", "worldpass1", "Torvald", "4", "Warrior", true, `Dunmar West Gate`)
		p.drain(2 * time.Second)
		// A fresh character is at full health, not a fraction of it.
		out := doAll(p, "status")
		if m := regexp.MustCompile(`Health:\s+(\d+)/(\d+)`).FindStringSubmatch(out); m == nil {
			t.Errorf("status shows no health:\n%s", out)
		} else if m[1] != m[2] {
			// Phase 44b: a new character used to start at the engine's flat
			// 10 health, under a maximum its archetype raised (11/59).
			t.Errorf("a new character should start at full health, got %s/%s", m[1], m[2])
		}
		p.do("weather", `It is \d`)
	})

	step("recruit at the Waymark Inn", func() {
		p.do("west", `The Waymark Inn`)
		out := doAll(p, "inn")
		if !strings.Contains(out, "5 gold") {
			t.Errorf("the inn should quote 5 gold a member:\n%s", out)
		}
		p.do("company recruit", `Your company: 0/4 companions`)
		p.do("company recruit tamsin", `Tamsin Reed joins your company`)
		p.do("company recruit oswin", `Brother Oswin joins your company`)
		p.do("company status", `Brother Oswin`)
		p.do("east", `Dunmar West Gate`)
	})

	step("the Old Kings Road journey", func() {
		// The setting-out line and the route report come in two turns.
		out := p.do("north", `Route: oak-road`)
		if !strings.Contains(out, "step onto the Old King's Road") {
			t.Errorf("setting out should say so:\n%s", out)
		}
		p.do("travel status", `Progress: \d+%`)
		// The oak-road's fallen tree pauses the company halfway.
		// Bad weather stretches a route, so the waits are generous.
		p.expect(`A fallen tree blocks the oak-road route`, 3*time.Minute)
		out = doAll(p, "travel status")
		if !strings.Contains(out, "Paused at the fallen tree") {
			t.Errorf("travel status should say the company is paused:\n%s", out)
		}
		p.do("travel resume", `You resume travel`)
		p.expect(`You have reached Fork at the Black Oak`, 4*time.Minute)
		p.drain(time.Second)
		p.do("look", `Fork at the Black Oak`)
		out = doAll(p, "weather")
		if !strings.Contains(out, "Temperature here") || !strings.Contains(out, "Sky:") {
			t.Errorf("a forest road has a sky:\n%s", out)
		}
		out = doAll(p, "scout")
		if !strings.Contains(out, "fight could find you") || !strings.Contains(out, "levels 1 to 1") {
			t.Errorf("scout should name the zone's level band:\n%s", out)
		}
	})

	var battleOver string
	step("a random encounter on the road", func() {
		// The grace after arriving (two entries, thirty real seconds) lets the
		// first steps through; the room then springs the brigands every time.
		sprang := false
		for cycle := 0; cycle < 25 && !sprang; cycle++ {
			p.send("east")
			p.expect(`Trappers' Post`, 20*time.Second)
			p.send("west")
			if _, ok := p.tryExpect(`Brigands step out of the pines`, 4*time.Second); ok {
				sprang = true
			}
		}
		if !sprang {
			t.Fatalf("no encounter in 25 trips through the Fork:\n%s", p.drain(time.Second))
		}
		// The fight plays itself out (no mid-battle input) and ends on its
		// closing line and the summary.
		battleOver = p.expect(`The fight with [^\n]*is over`, 4*time.Minute)
		for _, bad := range []string{"company is beaten", "has fallen"} {
			if strings.Contains(battleOver, bad) {
				t.Errorf("two unarmed level-1 brigands beat the company (%q):\n%s", bad, snippetAround(battleOver, bad))
			}
		}
		p.expect(`Enemies\s+[^\n]*slain`, 20*time.Second)
	})

	step("battle loot and the company's levels", func() {
		// Open item (phase 44 follow-up): "The battle is under way" lingers for
		// a moment after the summary. Ask at once, the way a player does, and
		// log how long it took rather than fail on it.
		started := time.Now()
		var out string
		for attempt := 0; attempt < 15; attempt++ {
			p.send("loot")
			got, ok := p.tryExpect(`You take|No eligible battle loot|The battle is under way`, 20*time.Second)
			if !ok {
				t.Fatalf("no answer to loot:\n%s", got)
			}
			if !strings.Contains(got, "The battle is under way") {
				out = got
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Logf("loot was answered %s after the fight's summary", time.Since(started).Round(100*time.Millisecond))
		if !strings.Contains(out, "You take") {
			t.Errorf("the brigands' spoils should be lootable, got:\n%s", out)
		}
		p.drain(time.Second)
		// Both companions levelled in that fight: the roster must say so now,
		// not at the next autosave (44b fix).
		out = doAll(p, "company status")
		if !regexp.MustCompile(`Tamsin Reed, level [2-9]`).MatchString(out) {
			t.Errorf("company status should show Tamsin's new level:\n%s", out)
		}
		p.do("experience", `Tamsin Reed Lvl: [2-9]`)
	})

	step("gather firewood and make a camp", func() {
		p.do("gather", `gather firewood\s+\d+ of \d+ ready`)
		p.send("gather firewood")
		p.expect(`Your company sets about gathering firewood`, 10*time.Second)
		out := p.expect(`Your company gathers [^\n]*firewood bundle`, 60*time.Second)
		if regexp.MustCompile(`gathers [2-9]\d* firewood bundle[^s]`).MatchString(out) {
			t.Errorf("a haul of several firewood bundles should be plural:\n%s", out)
		}
		out = doAll(p, "camp")
		if !strings.Contains(out, "You make camp here") || !strings.Contains(out, "deadfall") {
			t.Errorf("the Fork's deadfall should feed the fire for nothing:\n%s", out)
		}
		p.do("camp fire", `crackling campfire`)
		p.do("camp status", `The campfire is lit`)
	})

	step("restart with the camp, cargo and levels intact", func() {
		p.send("quit")
		deadline := time.Now().Add(90 * time.Second)
		for !p.isClosed() {
			if time.Now().After(deadline) {
				t.Fatalf("quit never closed the connection:\n%s", p.drain(time.Second))
			}
			time.Sleep(200 * time.Millisecond)
		}
		srv.stop()
		makeAdmin(t, srv, "1")
		srv.start()

		p = dial("world1-again")
		p.login("world1", "worldpass1")
		p.expect(`Welcome to the Mud`, 30*time.Second)
		p.drain(2 * time.Second)
		out := doAll(p, "company status")
		if !regexp.MustCompile(`Tamsin Reed, level [2-9]`).MatchString(out) {
			t.Errorf("a level gained in the fight was lost across a restart:\n%s", out)
		}
		p.do("camp status", `Camp at Fork at the Black Oak`)
		out = doAll(p, "inventory")
		if !strings.Contains(out, "firewood bundle") {
			t.Errorf("the gathered firewood was lost across a restart:\n%s", out)
		}
		p.do("camp break", `You break camp`)
	})

	step("salvage at the Frostfang armorer and sell at the Dunmar market", func() {
		// No road joins Dunmar to Frostfang yet, so an admin's teleport.
		p.do("teleport 63", `Steelwhisper Armory`)
		p.do("sell junk", `junk`) // sold, or "No one here wants your junk": never "not recognized"
		p.do("salvage", `break it down|break .* down|salvage \[item\]`)
		p.do("remove guardsman's broadsword", `equipment updated`)
		out := doAll(p, "salvage guardsman's broadsword")
		if !strings.Contains(out, "breaks your guardsman's broadsword down into") || !strings.Contains(out, "scrap iron") {
			t.Errorf("a tier-1 sword should salvage to scrap iron:\n%s", out)
		}
		p.do("teleport 2004", `Dunmar Market Square`)
		out = doAll(p, "market")
		if !strings.Contains(out, "Market prices in Dunmar") || !strings.Contains(out, "scrap iron") {
			t.Errorf("the Dunmar market should trade scrap iron:\n%s", out)
		}
		p.do("market sell scrap iron", `You sell the scrap iron at the market for \d+ gold`)
		// Gathered goods are not for resale: firewood is sold, never bought back.
		p.do("market sell firewood", `won't take that firewood bundle`)
	})

	step("a night at the Dunmar inn", func() {
		p.do("teleport 2003", `The Waymark Inn`)
		out := doAll(p, "inn")
		if !strings.Contains(out, "15 gold") {
			t.Errorf("a room for three should cost 15 gold:\n%s", out)
		}
		p.do("inn rest", `settles in to rest`)
		p.expect(`Your company wakes rested and refreshed`, 90*time.Second)
		p.expect(`Your company feels well rested`, 30*time.Second)
		out = doAll(p, "status")
		if !strings.Contains(out, "Well Rested") {
			t.Errorf("status should show Well Rested after the inn:\n%s", out)
		}
	})

	step("a clean shutdown", func() {
		p.send("quit")
		deadline := time.Now().Add(90 * time.Second)
		for !p.isClosed() {
			if time.Now().After(deadline) {
				t.Fatalf("quit never closed the connection:\n%s", p.drain(time.Second))
			}
			time.Sleep(200 * time.Millisecond)
		}
		srv.stop()
	})
}
