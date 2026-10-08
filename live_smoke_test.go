package main

// Phase 44: the live smoke playtest. It builds the real server, runs it
// against a copy of the shipped world, and plays a new character through the
// core loop over telnet: login, character creation, the whole tutorial (a
// recruited company, a formation, camp, a real battle), the help index,
// copyover, a restart and a second player. Unit and wiring tests miss
// start-up, persistence and multi-player seams; this finds them.
//
// It takes a few minutes of real time (a camp rest is a real minute), so it
// only runs when asked:
//
//	make smoke        (or: ASHVEIL_LIVE_SMOKE=1 go test -run TestLiveSmoke -timeout 20m -v .)
//
// ASHVEIL_SMOKE_KEEP=1 keeps the server's data copy and transcripts.

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

var (
	ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	iacRE  = regexp.MustCompile(`(?s)\xff\xfa.*?\xff\xf0|\xff[\xfb-\xfe].`)
)

// mudClient is a telnet or web player: it keeps everything the server says,
// and waits for patterns in what it has not consumed yet.
type mudClient struct {
	t    *testing.T
	name string
	conn net.Conn        // a telnet player
	ws   *websocket.Conn // or a web one

	mu         sync.Mutex
	pending    string // unconsumed, ANSI and telnet negotiation stripped
	transcript strings.Builder
	closed     bool
	lastSend   time.Time
}

// dialWeb is a web-client player: the same reader over the /ws socket the
// browser client uses, one text frame per line typed (Phase 79).
func dialWeb(t *testing.T, name string, httpPort int) *mudClient {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d/ws", httpPort), nil)
	if err != nil {
		t.Fatalf("%s: dial the websocket: %v", name, err)
	}
	m := &mudClient{t: t, name: name, ws: ws}
	go m.readLoop()
	return m
}

func dialMud(t *testing.T, name string, port int) *mudClient {
	t.Helper()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 10*time.Second)
	if err != nil {
		t.Fatalf("%s: dial: %v", name, err)
	}
	m := &mudClient{t: t, name: name, conn: conn}
	go m.readLoop()
	t.Cleanup(func() { conn.Close() })
	return m
}

func (m *mudClient) readLoop() {
	buf := make([]byte, 64*1024)
	for {
		var n int
		var err error
		if m.ws != nil {
			var frame []byte
			_, frame, err = m.ws.ReadMessage()
			n = copy(buf, frame)
		} else {
			n, err = m.conn.Read(buf)
		}
		if n > 0 {
			text := string(iacRE.ReplaceAll(buf[:n], nil))
			text = ansiRE.ReplaceAllString(text, "")
			m.mu.Lock()
			m.pending += text
			m.transcript.WriteString(text)
			m.mu.Unlock()
		}
		if err != nil {
			m.mu.Lock()
			m.closed = true
			m.mu.Unlock()
			return
		}
	}
}

func (m *mudClient) send(line string) {
	m.t.Helper()
	// The server drops a line that arrives within a turn (Timing.TurnMs) of
	// the last, so pace like a person typing.
	if wait := 200*time.Millisecond - time.Since(m.lastSend); wait > 0 {
		time.Sleep(wait)
	}
	m.lastSend = time.Now()
	if m.ws != nil {
		// The browser client sends each line as one text frame, no newline.
		if err := m.ws.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			m.t.Fatalf("%s: send %q: %v", m.name, line, err)
		}
		return
	}
	if _, err := m.conn.Write([]byte(line + "\n")); err != nil {
		m.t.Fatalf("%s: send %q: %v", m.name, line, err)
	}
}

// tryExpect waits for a regexp in the unconsumed output, consumes through
// the match and returns everything up to and including it.
func (m *mudClient) tryExpect(pattern string, timeout time.Duration) (string, bool) {
	return m.match(pattern, timeout, true)
}

// match is tryExpect that may leave the match unconsumed.
func (m *mudClient) match(pattern string, timeout time.Duration, consume bool) (string, bool) {
	re := regexp.MustCompile(`(?s)` + pattern)
	deadline := time.Now().Add(timeout)
	for {
		m.mu.Lock()
		if loc := re.FindStringIndex(m.pending); loc != nil {
			got := m.pending[:loc[1]]
			if consume {
				m.pending = m.pending[loc[1]:]
			}
			m.mu.Unlock()
			return got, true
		}
		tail := m.pending
		closed := m.closed
		m.mu.Unlock()
		if closed || time.Now().After(deadline) {
			if len(tail) > 2500 {
				tail = tail[len(tail)-2500:]
			}
			return tail, false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// expect is tryExpect that fails the test on a miss.
func (m *mudClient) expect(pattern string, timeout time.Duration) string {
	m.t.Helper()
	got, ok := m.tryExpect(pattern, timeout)
	if !ok {
		why := "timed out"
		if m.isClosed() {
			why = "connection closed"
		}
		m.t.Fatalf("%s: %s waiting for /%s/. Unread output:\n%s", m.name, why, pattern, got)
	}
	return got
}

// answer replies to a prompt and waits for the next one. A server-wide
// event (the first sunrise of a fresh world) can redraw a prompt and drop
// what was typed, so an unanswered prompt is asked again.
func (m *mudClient) answer(prompt, reply, next string) {
	m.t.Helper()
	for attempt := 0; attempt < 4; attempt++ {
		m.expect(prompt, 20*time.Second)
		m.send(reply)
		if _, ok := m.match(next, 6*time.Second, false); ok {
			return
		}
	}
	m.t.Fatalf("%s: answered %q to /%s/ four times and never reached /%s/", m.name, reply, prompt, next)
}

// do sends a command and returns the output up to the first match of want.
func (m *mudClient) do(line, want string) string {
	m.t.Helper()
	m.send(line)
	return m.expect(want, 30*time.Second)
}

// doAfterBattle is do for a command a just-finished battle may still refuse
// ("The battle is under way"): the battle is cleared a round after its last
// line, so it is tried again until it takes.
func (m *mudClient) doAfterBattle(line, want string) string {
	m.t.Helper()
	for attempt := 0; attempt < 10; attempt++ {
		m.send(line)
		got, ok := m.tryExpect(`(`+want+`)|The battle is under way`, 30*time.Second)
		if !ok {
			m.t.Fatalf("%s: no answer to %q", m.name, line)
		}
		if !strings.Contains(got, "The battle is under way") {
			return got
		}
		time.Sleep(2 * time.Second)
	}
	m.t.Fatalf("%s: %q was refused as a battle under way ten times", m.name, line)
	return ""
}

// drain waits for quiet and returns what arrived.
func (m *mudClient) drain(quiet time.Duration) string {
	m.t.Helper()
	last := -1
	for {
		time.Sleep(quiet)
		m.mu.Lock()
		n := len(m.pending)
		m.mu.Unlock()
		if n == last {
			break
		}
		last = n
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.pending
	m.pending = ""
	return out
}

func (m *mudClient) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mudClient) text() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.transcript.String()
}

// login signs into an existing account.
func (m *mudClient) login(user, pass string) {
	m.t.Helper()
	m.expect(`Find out more`, 20*time.Second)
	m.send("")
	m.expect(`username \(or "new"\)`, 20*time.Second)
	m.send(user)
	m.expect(`password:`, 10*time.Second)
	m.send(pass)
}

// register makes a new account and character: race human, the given
// archetype. It answers the tutorial question and stops there.
func (m *mudClient) register(user, pass, character, archetypeName string, skipTutorial bool) {
	m.t.Helper()
	m.registerArriving(user, pass, character, archetypeName, skipTutorial, `Town Square`)
}

// registerArriving is register for a world whose start room is not the Town
// Square: a skipped tutorial waits for the named room.
func (m *mudClient) registerArriving(user, pass, character, archetypeName string, skipTutorial bool, arrival string) {
	m.t.Helper()
	m.expect(`Find out more`, 20*time.Second)
	m.send("")
	m.expect(`username \(or "new"\)`, 20*time.Second)
	m.send("new")
	m.expect(`Choose your username`, 10*time.Second)
	m.send(user)
	m.expect(`Choose your password`, 10*time.Second)
	m.send(pass)
	m.expect(`Repeat your password`, 10*time.Second)
	m.send(pass)
	m.expect(`Email Address`, 10*time.Second)
	m.send("")
	m.expect(`screen reader`, 10*time.Second)
	m.send("n")
	m.expect(`create a new user`, 10*time.Second)
	m.send("y")
	m.expect(`type look or start`, 20*time.Second)
	m.send("start")
	m.answer(`Which race will you be`, "human", `known as \(name\)`)
	m.answer(`known as \(name\)`, character, `Choose the name `+character+`\? \[yes/no\]`)
	m.answer(`Choose the name `+character+`\? \[yes/no\]`, "yes", `Which archetype will you follow`)
	// By name: the menu numbers shift whenever a lineage is added.
	m.answer(`Which archetype will you follow`, strings.ToLower(archetypeName), `Become a `+archetypeName+`\?`)
	m.answer(`Become a `+archetypeName+`\?`, "yes", `How will others speak of you\?`)
	m.createLooksAndStory()
	if skipTutorial {
		m.answer(`skip the tutorial\? \[yes/no\]`, "yes", arrival)
	} else {
		m.answer(`skip the tutorial\? \[yes/no\]`, "no", `Tutorial, stage 1 of 8: Your character`)
	}
}

// createLooksAndStory answers the looks and life-story steps (phase 72a)
// with each list's first option and no line of its own, checks the summary,
// confirms it, takes the standard (not Iron) option, and stops at the
// tutorial question.
func (m *mudClient) createLooksAndStory() {
	m.t.Helper()
	const (
		question = `Answer with a number or a name\.`
		ownLine  = `press Enter for none\.`
		summary  = `Keep this character as described\?`
		tutorial = `skip the tutorial\? \[yes/no\]`
	)
	sawSummary := false
	for asked := 0; asked < 40; asked++ {
		// Peek at whichever comes first without consuming the tutorial
		// question, which the caller answers.
		got, ok := m.match(question+`|`+ownLine+`|`+tutorial, 20*time.Second, false)
		if !ok {
			m.t.Fatalf("%s: no creation question or tutorial question. Unread output:\n%s", m.name, got)
		}
		switch {
		case strings.HasSuffix(got, "[yes/no]"):
			if !sawSummary {
				m.t.Fatalf("%s: creation reached the tutorial without a summary:\n%s", m.name, got)
			}
			return
		case strings.HasSuffix(got, "for none."):
			m.expect(ownLine, time.Second)
			m.send("")
		// The question after the summary (phase 77's Iron option) echoes
		// the summary's answer line, so only the first match is the summary.
		case !sawSummary && regexp.MustCompile(summary).MatchString(got):
			m.expect(question, time.Second)
			for _, want := range []string{`You look like this`, `Your story`, `\+1 `} {
				if !regexp.MustCompile(`(?i)` + want).MatchString(got) {
					m.t.Errorf("%s: the creation summary lacks /%s/:\n%s", m.name, want, got)
				}
			}
			sawSummary = true
			m.send("confirm")
		default:
			m.expect(question, time.Second)
			m.send("1")
		}
	}
	m.t.Fatalf("%s: creation asked over 40 questions", m.name)
}

// smokeServer is a real server process on a private copy of the world.
type smokeServer struct {
	t        *testing.T
	bin      string
	dir      string
	port     int
	httpPort int // the web client's port (Phase 79: the websocket login)
	cmd      *exec.Cmd
	log      string
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// smokeOptions shape a smoke server's private copy of the world.
type smokeOptions struct {
	extraConfig string               // appended to the overrides file (YAML)
	patch       func(dataDir string) // edits the copied world before it starts
}

func newSmokeServer(t *testing.T) *smokeServer {
	t.Helper()
	return newSmokeServerWith(t, smokeOptions{})
}

func newSmokeServerWith(t *testing.T, opts smokeOptions) *smokeServer {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var dir string
	if os.Getenv("ASHVEIL_SMOKE_KEEP") != "" {
		dir, err = os.MkdirTemp("", "ashveil-smoke-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("keeping the server's files in %s", dir)
	} else {
		dir = t.TempDir()
	}
	s := &smokeServer{t: t, dir: dir, port: freePort(t), httpPort: freePort(t), bin: filepath.Join(dir, "gomud-server"), log: filepath.Join(dir, "server.log")}

	if out, err := exec.Command("go", "build", "-o", s.bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build the server: %v\n%s", err, out)
	}
	if out, err := exec.Command("cp", "-r", filepath.Join(root, "_datafiles"), filepath.Join(dir, "_datafiles")).CombinedOutput(); err != nil {
		t.Fatalf("copy the world: %v\n%s", err, out)
	}
	// Fresh state: no users, instances or round count from a dev checkout.
	usersDir := filepath.Join(dir, "_datafiles", "world", "default", "users")
	_ = os.RemoveAll(usersDir)
	if err := os.MkdirAll(usersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A dev checkout's saved room instances would shadow the shipped rooms.
	_ = os.RemoveAll(filepath.Join(dir, "_datafiles", "world", "default", "rooms.instances"))
	if opts.patch != nil {
		opts.patch(filepath.Join(dir, "_datafiles", "world", "default"))
	}
	overrides := fmt.Sprintf("Network:\n  TelnetPort: [%d]\n  LocalPort: 0\n  HttpPort: %d\n  HttpsPort: 0\n  SSHPort: 0\nTiming:\n  RoundSeconds: 2\n", s.port, s.httpPort) + opts.extraConfig
	if err := os.WriteFile(filepath.Join(dir, "overrides.yaml"), []byte(overrides), 0o644); err != nil {
		t.Fatal(err)
	}
	return s
}

func (s *smokeServer) start() {
	s.t.Helper()
	logFile, err := os.OpenFile(s.log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		s.t.Fatal(err)
	}
	cmd := exec.Command(s.bin)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(),
		"CONFIG_PATH="+filepath.Join(s.dir, "overrides.yaml"),
		"LOG_LEVEL=LOW",
		"LOG_PATH="+filepath.Join(s.dir, "game.log"),
		"LOG_NOCOLOR=1",
	)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		s.t.Fatalf("start the server: %v", err)
	}
	s.cmd = cmd
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", s.port), time.Second); err == nil {
			c.Close()
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	s.t.Fatalf("the server never listened on %d:\n%s", s.port, s.tailLog())
}

func (s *smokeServer) tailLog() string {
	b, _ := os.ReadFile(s.log)
	if len(b) > 3000 {
		b = b[len(b)-3000:]
	}
	return string(b)
}

// stop asks for a clean shutdown (which saves) and waits for it.
func (s *smokeServer) stop() {
	s.t.Helper()
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	_ = s.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _ = s.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		_ = s.cmd.Process.Kill()
		<-done
		s.t.Fatalf("the server ignored SIGTERM for a minute")
	}
	s.cmd = nil
}

func (s *smokeServer) kill() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
		s.cmd = nil
	}
}

// helpTopics lists every topic and alias the world indexes for `help`.
func helpTopics(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("_datafiles", "world", "default", "keywords.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Help        map[string]any      `yaml:"help"`
		HelpAliases map[string][]string `yaml:"help-aliases"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			seen[x] = true
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	for category, topics := range doc.Help {
		if category != "admin" { // admin pages answer only to admins
			walk(topics)
		}
	}
	admin := map[string]bool{}
	if topics, ok := doc.Help["admin"]; ok {
		before := seen
		seen = map[string]bool{}
		walk(topics)
		admin, seen = seen, before
	}
	for topic, aliases := range doc.HelpAliases {
		if admin[topic] {
			continue
		}
		seen[topic] = true
		for _, a := range aliases {
			seen[a] = true
		}
	}
	var topics []string
	for topic := range seen {
		if topic != "" && !strings.Contains(topic, " ") {
			topics = append(topics, topic)
		}
	}
	return topics
}

const (
	smokePrompt = `HP:-?\d+/\d+ MP:\d+/\d+\]`
)

func TestLiveSmoke(t *testing.T) {
	if os.Getenv("ASHVEIL_LIVE_SMOKE") == "" {
		t.Skip("set ASHVEIL_LIVE_SMOKE=1 (or run `make smoke`) to play a live server")
	}
	if testing.Short() {
		t.Skip("live smoke playtest skipped in -short mode")
	}

	// An existing character from before phase 72a: the shipped admin, with
	// a hashed password so login skips the forced password change.
	srv := newSmokeServerWith(t, smokeOptions{patch: func(dataDir string) {
		b, err := os.ReadFile(filepath.Join("_datafiles", "world", "default", "users", "1.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		legacy := strings.Replace(string(b), "password: password\n", "password: "+util.Hash("password")+"\n", 1)
		if legacy == string(b) {
			t.Fatal("the shipped admin's password line moved")
		}
		if err := os.WriteFile(filepath.Join(dataDir, "users", "1.yaml"), []byte(legacy), 0o644); err != nil {
			t.Fatal(err)
		}
	}})
	t.Cleanup(func() {
		srv.kill()
		if t.Failed() {
			t.Logf("server output:\n%s", srv.tailLog())
		}
	})
	srv.start()

	var p1 *mudClient
	// Steps share one top-level t: a failed step is fatal, so later steps
	// never run against a broken game.
	step := func(name string, fn func()) {
		t.Helper()
		t.Logf("== %s", name)
		fn()
	}

	// No script, test or player command may ever come back as unrecognized
	// at login, or a mud-side error; checked against everything p1 saw.
	t.Cleanup(func() {
		if p1 == nil {
			return
		}
		txt := p1.text()
		for _, bad := range []string{"not recognized", "looks a little confused", "panic:", "runtime error"} {
			if strings.Contains(txt, bad) {
				t.Errorf("player 1 saw %q:\n%s", bad, snippetAround(txt, bad))
			}
		}
	})

	step("create a warrior and take the tutorial", func() {
		p1 = dialMud(t, "smoker1", srv.port)
		p1.register("smoker1", "smokepass1", "Torvald", "Warrior", false)
		p1.expect(`Tutorial, stage 1 of 8: Your character`, 30*time.Second)

		p1.drain(3 * time.Second) // the tutorial hand-off drops input typed during it
		p1.do("status", `More: company status`)
		p1.do("inventory", `Cargo:`)
		p1.do("experience", `Next:`)
		p1.do("conditions", `Conditions`)
		p1.expect(`Well done\. You know yourself well enough`, 10*time.Second)

		p1.do("east", `Tutorial, stage 2 of 8: Your company`)
		p1.do("company recruit tamsin", `Tamsin Reed joins your company`)
		p1.do("company recruit oswin", `Brother Oswin joins your company`)
		p1.expect(`Head east for the next lesson: Formation`, 10*time.Second)

		p1.do("east", `Tutorial, stage 3 of 8: Formation`)
		p1.do("formation move tamsin 1 2", `Placed Tamsin Reed`)
		p1.do("formation move oswin 2 1", `Placed Brother Oswin`)
		p1.expect(`Head east for the next lesson: Survival`, 10*time.Second)

		p1.do("east", `Tutorial, stage 4 of 8: Survival`)
		p1.do("eat sandwich", `You eat`)
		p1.do("drink waterskin", `You drink`)
		p1.do("weather", `Temperature here`)
		p1.do("temperature", `Air temperature`)
		p1.do("strain", `Walking here|strain|fatigue`)
		p1.do("cargo", `Capacity:`)
		p1.expect(`Head east for the next lesson: Camp`, 10*time.Second)

		p1.do("east", `Tutorial, stage 5 of 8: Camp`)
		p1.do("camp", `You make camp here`)
		p1.do("camp fire", `campfire`)
		p1.do("camp rest", `settle in by the fire`)
		p1.expect(`Your company is Rested`, 120*time.Second)
		p1.expect(`Head east for the next lesson: Combat`, 20*time.Second)

		p1.do("east", `Tutorial, stage 6 of 8: Combat`)
		p1.do("set combatpace off", `Combat pace`)
		p1.do("formation move me 3 2", `Placed Torvald`)
		// Phase 61: a battle order, set before the fight.
		p1.do("orders oswin add ally 50 then heal", `Order 1 for Brother Oswin`)
		scout := p1.do("scout squad", `Assessment:`)
		if !strings.Contains(scout, "straw archer") {
			t.Errorf("scout squad did not show the archer:\n%s", scout)
		}
		p1.send("attack squad")
		fight := p1.expect(`The fight with the straw squad is over`, 5*time.Minute)
		// Phase 44: the straw soldiers "can't hurt anyone"; once they hit
		// for the flat damage floor and killed the company.
		for _, bad := range []string{"has fallen", "crumples and does not rise", "company is beaten"} {
			if strings.Contains(fight, bad) {
				t.Errorf("the harmless practice squad hurt the company (%q):\n%s", bad, snippetAround(fight, bad))
			}
		}
		summary := p1.expect(`Company\s+You \d+/\d+.*Brother Oswin \d+/\d+`, 20*time.Second)
		if !strings.Contains(summary, "Enemies 0") {
			t.Errorf("practice foes dealt damage: %s", summary)
		}
		p1.expect(`Head east for the next lesson: Alignment`, 20*time.Second)
		p1.drain(time.Second)
		// Phase 62: `why` explains the fight's latest blows from the engine's roll.
		p1.do("why", `Round \d+: .*\n.*(To hit: \d+ in 100, rolled \d+|No roll needed)`)

		p1.doAfterBattle("east", `Tutorial, stage 7 of 8: Alignment`)
		p1.do("company alignment", `Company alignment`)
		p1.do("company inspect corvin", `won't join a company`)
		p1.do("standing", `Elsewhere`)
		p1.expect(`Head east for the next lesson: Departure`, 10*time.Second)

		p1.do("east", `Tutorial, stage 8 of 8: Departure`)
		p1.do("gate", `Town Square`)
		p1.expect(`Your journey begins`, 20*time.Second)
	})

	step("the world after the gate", func() {
		p1.drain(2 * time.Second)
		out := p1.do("status", `More: company status`)
		if !strings.Contains(out, "Warrior") || strings.Contains(out, "scrub") {
			t.Errorf("status should call the class Warrior, not a legacy profession:\n%s", out)
		}
		p1.do("company status", `Brother Oswin`)
		out = p1.do("look guardsman", `Tier \d`)
		if !strings.Contains(out, "sword") {
			t.Errorf("a shipped weapon should show its tier and family: %s", out)
		}
		p1.do("loot", `No eligible battle loot`)
		p1.do("company patch", `patch threshold|patches itself`)
		p1.do("formation", `back\s+\[`)
	})

	step("every indexed help topic renders", func() {
		topics := helpTopics(t)
		if len(topics) < 100 {
			t.Fatalf("only %d help topics found in keywords.yaml", len(topics))
		}
		var empty []string
		for _, topic := range topics {
			p1.send("help " + topic)
			// The command is echoed ahead of its output; wait for that,
			// then for the prompt that closes it, so an idle prompt tick
			// from before cannot end the read early.
			p1.expect(regexp.QuoteMeta("help "+topic), 20*time.Second)
			out := p1.expect(smokePrompt, 20*time.Second)
			if strings.Contains(out, "No help found") || len(strings.TrimSpace(out)) < 60 {
				empty = append(empty, topic)
			}
		}
		if len(empty) > 0 {
			t.Errorf("%d of %d help topics render nothing: %v", len(empty), len(topics), empty)
		}
	})

	step("copyover keeps the player and the company", func() {
		p1.drain(time.Second)
		if err := srv.cmd.Process.Signal(syscall.SIGUSR1); err != nil {
			t.Fatal(err)
		}
		p1.expect(`Copyover complete`, 90*time.Second)
		p1.drain(2 * time.Second)
		out := p1.do("company status", `Brother Oswin`)
		if !strings.Contains(out, "Tamsin Reed") {
			t.Errorf("the company did not survive copyover:\n%s", out)
		}
		p1.do("look", `Exits:`)
	})

	step("two players share a room", func() {
		p2 := dialMud(t, "smoker2", srv.port)
		p2.register("smoker2", "smokepass2", "Mirelle", "Witch", true)
		p2.expect(`Town Square`, 40*time.Second)
		p2.drain(2 * time.Second)

		out := p2.do("spells", `Slumber`)
		if !strings.Contains(out, "Slumber") {
			t.Errorf("a new Witch should know Slumber:\n%s", out)
		}
		p2.do("status", `More: company status`)

		out = p1.do("who", `Mirelle|Also here`)
		_ = out
		p2.send("say the smoke test was here")
		p1.expect(`the smoke test was here`, 20*time.Second)
		p2.send("quit")
		p2.expect(`meditation`, 10*time.Second)
	})

	step("the web client logs in after an early GMCP request", func() {
		// Phase 79: the web client's windows ask for their pages
		// (!!GMCP(Char.Creation)) as soon as the socket is up, before the
		// player has typed a username. Over a websocket that request used
		// to be taken as the username and the login failed.
		w := dialWeb(t, "web-smoker2", srv.httpPort)
		w.expect(`username \(or "new"\)`, 20*time.Second)
		w.send("!!GMCP(Char.Creation)")
		w.send("smoker2")
		w.expect(`password:`, 10*time.Second)
		w.send("smokepass2")
		// The telnet player's quit may still be closing; take the seat.
		out, _ := w.tryExpect(`Welcome to the Mud|Kick them\?`, 30*time.Second)
		if strings.Contains(out, "Kick them?") {
			w.send("y")
			out += w.expect(`Welcome to the Mud`, 30*time.Second)
		}
		for _, bad := range []string{"try again", "Invalid login"} {
			if strings.Contains(out, bad) {
				t.Errorf("the web login printed %q after an early GMCP request:\n%s", bad, out)
			}
		}
		w.drain(2 * time.Second)
		w.do("status", `More: company status`)
		w.send("quit")
		w.expect(`meditation`, 10*time.Second)
	})

	step("an existing character is offered looks and a story once", func() {
		pa := dialMud(t, "legacy", srv.port)
		pa.login("admin", "password")
		pa.expect(`Write them now, or later\?`, 40*time.Second)
		pa.send("later")
		pa.expect(`You put it off`, 20*time.Second)
		pa.drain(time.Second)
		pa.send("quit")
		pa.expect(`meditation`, 10*time.Second)
	})

	step("log out, restart the server and come back", func() {
		p1.send("quit")
		deadline := time.Now().Add(90 * time.Second)
		for !p1.isClosed() {
			if time.Now().After(deadline) {
				t.Fatalf("quit never closed the connection:\n%s", p1.drain(time.Second))
			}
			time.Sleep(200 * time.Millisecond)
		}
		srv.stop()
		srv.start()

		p1b := dialMud(t, "smoker1-again", srv.port)
		p1b.login("smoker1", "smokepass1")
		p1b.expect(`Welcome to the Mud`, 30*time.Second)
		p1b.drain(2 * time.Second)
		out := p1b.do("company status", `Brother Oswin`)
		if !strings.Contains(out, "Tamsin Reed") {
			t.Errorf("the company was lost across a restart:\n%s", out)
		}
		out = p1b.do("formation", `back\s+\[`)
		if !strings.Contains(out, "Tamsin Reed(#1)") {
			t.Errorf("the formation was lost across a restart:\n%s", out)
		}
		p1b.do("orders oswin", `1\. When an ally is below 50% health, heal that ally first\.`) // Phase 61: orders persist
		p1b.do("chronicle joined", `Tamsin Reed joined the company`)                           // Phase 63: the chronicle persists
		out = p1b.do("status", `More: company status`)
		if !strings.Contains(out, "Torvald") {
			t.Errorf("status after restart: %s", out)
		}
		if bad := "not recognized"; strings.Contains(p1b.text(), bad) {
			t.Errorf("login after restart printed %q:\n%s", bad, snippetAround(p1b.text(), bad))
		}
		out = p1b.do("appearance", `changes your looks for free|haven't described yourself`)
		if strings.Contains(out, "haven't described") {
			t.Errorf("the chosen looks were lost across a restart:\n%s", out)
		}
		out = p1b.do("lifestory", `It gives you|hasn't been written`)
		if strings.Contains(out, "hasn't been written") {
			t.Errorf("the life story was lost across a restart:\n%s", out)
		}

		// The put-off offer is remembered across the restart.
		pa := dialMud(t, "legacy-again", srv.port)
		pa.login("admin", "password")
		pa.expect(`Welcome to the Mud`, 30*time.Second)
		pa.drain(6 * time.Second)
		if strings.Contains(pa.text(), "Write them now, or later") {
			t.Errorf("an existing character was offered the creation steps twice")
		}
	})
}

// snippetAround shows a few lines of text around the first match.
func snippetAround(text, needle string) string {
	i := strings.Index(text, needle)
	if i < 0 {
		return ""
	}
	lo, hi := max(0, i-400), min(len(text), i+400)
	return text[lo:hi]
}
