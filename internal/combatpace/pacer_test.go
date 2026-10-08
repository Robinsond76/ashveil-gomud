package combatpace

import (
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func at(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }

func TestParseAndDefaults(t *testing.T) {
	for _, s := range []string{"fast", "Normal", " slow ", "OFF"} {
		if _, ok := Parse(s); !ok {
			t.Errorf("Parse(%q) rejected", s)
		}
	}
	if _, ok := Parse("quick"); ok {
		t.Error("Parse accepted an unknown pace")
	}
	if For(nil, false) != Normal || For(nil, true) != Off {
		t.Error("unset pace should default to normal, or off for a screen reader")
	}
	if For("slow", true) != Slow {
		t.Error("a chosen pace wins over the screen-reader default")
	}
	if For("bogus", false) != Normal || For(3, false) != Normal {
		t.Error("a bad saved value falls back to the default")
	}
	if Off.Spec() != (Spec{}) {
		t.Error("off has no gaps")
	}
}

func TestForRoundClampsTheWindow(t *testing.T) {
	if got := Slow.ForRound(8 * time.Second).Window; got != 7200*time.Millisecond {
		t.Fatalf("slow window in an 8s round = %v, want 7.2s", got)
	}
	if got := Normal.ForRound(8 * time.Second).Window; got != 7200*time.Millisecond {
		t.Fatalf("normal window in an 8s round = %v, want 7.2s", got)
	}
	if got := Normal.ForRound(4 * time.Second).Window; got != 3600*time.Millisecond {
		t.Fatalf("normal window in a 4s round = %v, want 3.6s", got)
	}
}

// fixedSpec is the fixed-cadence timing the pacer tests were written
// against (a Gap of 800ms), independent of the shipped paces' values.
func fixedSpec(round time.Duration) Spec {
	s := Spec{Gap: 800 * time.Millisecond, Dramatic: 1400 * time.Millisecond, Quick: 250 * time.Millisecond, Window: 6 * time.Second}
	if limit := round * 9 / 10; limit > 0 && s.Window > limit {
		s.Window = limit
	}
	return s
}

// Phase 87: what was "slow" is the normal pace now, and slow is one and a
// half times that; fast and off are as they were.
func TestPhase87PacesAreSlower(t *testing.T) {
	if got := Normal.Beats().Beat; got != 1500*time.Millisecond {
		t.Errorf("normal beat = %v, want 1.5s", got)
	}
	if got := Slow.Beats().Beat; got != 2250*time.Millisecond {
		t.Errorf("slow beat = %v, want 2.25s", got)
	}
	if got := Fast.Beats().Beat; got != 600*time.Millisecond {
		t.Errorf("fast beat = %v, want 0.6s", got)
	}
	n, s := Normal.Beats(), Slow.Beats()
	for name, pair := range map[string][2]time.Duration{
		"follow": {n.Follow, s.Follow}, "extra": {n.Extra, s.Extra}, "tail": {n.Tail, s.Tail},
		"gap": {Normal.Spec().Gap, Slow.Spec().Gap}, "dramatic": {Normal.Spec().Dramatic, Slow.Spec().Dramatic},
	} {
		if pair[1] != pair[0]*3/2 {
			t.Errorf("slow %s = %v, want 1.5 x normal (%v)", name, pair[1], pair[0])
		}
	}
	if Off.Spec() != (Spec{}) {
		t.Error("off is untouched")
	}
}

// release steps the clock in 50ms turns up to ms and records when each line
// went out.
func release(p *Pacer, until int) map[string]int {
	out := map[string]int{}
	for ms := 0; ms <= until; ms += 50 {
		rel, _ := p.Due(at(ms))
		for _, r := range rel {
			out[r.Text] = ms
		}
	}
	return out
}

func TestLinesGoOutInOrderAtThePacesGap(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	for _, l := range []string{"a\n", "b\n", "c\n"} {
		p.Hold(1, 2, 0, l, spec, at(0))
	}
	if !p.Busy(1) {
		t.Fatal("held lines should make the player busy")
	}
	got := release(p, 3000)
	want := map[string]int{"a\n": 0, "b\n": 800, "c\n": 1600}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("release times %v, want %v", got, want)
	}
	if p.Busy(1) {
		t.Fatal("drained player still busy")
	}
}

func TestGapsShrinkToFitTheWindow(t *testing.T) {
	p := New()
	spec := Fast.ForRound(8 * time.Second) // 0.4s gap, 3s window
	for i := 0; i < 21; i++ {              // 20 gaps = 8s unscaled
		p.Hold(1, 2, 0, string(rune('a'+i)), spec, at(0))
	}
	var last time.Time
	var order []string
	for ms := 0; ms <= 4000; ms += 50 {
		rel, _ := p.Due(at(ms))
		for _, r := range rel {
			order = append(order, r.Text)
			last = at(ms)
		}
	}
	if len(order) != 21 || order[0] != "a" || order[20] != "u" {
		t.Fatalf("order %v", order)
	}
	if last.Sub(t0) > 3*time.Second {
		t.Fatalf("last line at %v, past the 3s window", last.Sub(t0))
	}
}

func TestDramaticAndQuickBeats(t *testing.T) {
	p := New()
	p.Mark("The bandit howls.") // marks match without the trailing newline
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 2, 0, "You strike the bandit. (critical hit, 9 damage)\n", spec, at(0))
	p.Hold(1, 2, 0, "The bandit howls.\n", spec, at(0))
	p.Hold(1, 2, 0, "The spell bursts.\n", spec, at(0))
	p.Hold(1, 2, 0, "  the rat (3 damage)\n", spec, at(0))
	got := release(p, 4000)
	want := map[string]int{
		"You strike the bandit. (critical hit, 9 damage)\n": 0,
		"The bandit howls.\n":                               1400,
		"The spell bursts.\n":                               2200,
		"  the rat (3 damage)\n":                            2450,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("release times %v, want %v", got, want)
	}

	p.StartRound()
	p.Hold(1, 4, 0, "The bandit howls.\n", spec, at(0))
	p.Hold(1, 4, 0, "The bandit howls.\n", spec, at(0))
	if got := release(p, 4000); got["The bandit howls.\n"] != 800 {
		t.Fatalf("a mark outlived its round: %v", got)
	}
}

func TestAnOlderRoundIsFlushedAheadOfANewOne(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 2, 0, "old1", spec, at(0))
	p.Hold(1, 2, 0, "old2", spec, at(0))
	p.Due(at(0)) // old1 out
	flushed := p.Hold(1, 4, 0, "new1", spec, at(8000))
	if !reflect.DeepEqual(flushed, []Release{{UserId: 1, Text: "old2"}}) {
		t.Fatalf("flushed %v, want [old2]", flushed)
	}
	rel, drained := p.Due(at(8000))
	if len(rel) != 1 || rel[0].Text != "new1" || !reflect.DeepEqual(drained, []int{1}) {
		t.Fatalf("new round release %v drained %v", rel, drained)
	}
}

func TestFlushAndFlushAll(t *testing.T) {
	p := New()
	spec := Slow.ForRound(8 * time.Second)
	p.Hold(2, 2, 0, "b1", spec, at(0))
	p.Hold(1, 2, 0, "a1", spec, at(0))
	p.Hold(1, 2, 0, "a2", spec, at(0))
	p.Hold(3, 2, 0, "c1", spec, at(0))
	if got, ended := p.Flush(3); !reflect.DeepEqual(got, []Release{{UserId: 3, Text: "c1"}}) || !ended {
		t.Fatalf("Flush(3) = %v %v", got, ended)
	}
	if got, ended := p.Flush(3); p.Busy(3) || got != nil || ended {
		t.Fatal("a flushed player holds nothing")
	}
	rel, drained := p.FlushAll()
	want := []Release{{UserId: 1, Text: "a1"}, {UserId: 1, Text: "a2"}, {UserId: 2, Text: "b1"}}
	if !reflect.DeepEqual(rel, want) || !reflect.DeepEqual(drained, []int{1, 2}) {
		t.Fatalf("FlushAll = %v %v", rel, drained)
	}
	if p.Busy(1) || p.Busy(2) {
		t.Fatal("FlushAll left a player busy")
	}
}

func TestUseForTestRestores(t *testing.T) {
	orig := Default()
	mine := New()
	restore := UseForTest(mine)
	if Default() != mine {
		t.Fatal("UseForTest did not install the pacer")
	}
	restore()
	if Default() != orig {
		t.Fatal("restore did not put the original back")
	}
}

func TestFollowJoinsHeldLinesOnly(t *testing.T) {
	p := New()
	if p.Follow(1, "alone") {
		t.Fatal("Follow held a line for a player with nothing held")
	}
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 2, 0, "r1", spec, at(0))
	p.Hold(1, 2, 0, "r2", spec, at(0))
	if !p.Follow(1, "after") {
		t.Fatal("Follow did not join the held lines")
	}
	got := release(p, 3000)
	want := map[string]int{"r1": 0, "r2": 800, "after": 1600}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("release times %v, want %v", got, want)
	}
}

func TestAnOpenRoundIsBusyUntilDrained(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	p.StartRound(1, 2, 3)
	for _, id := range []int{1, 2, 3} {
		if !p.Busy(id) {
			t.Fatalf("player %d not busy as the round opens", id)
		}
	}
	p.Hold(1, 2, 0, "l1", spec, at(0))
	p.Hold(1, 2, 0, "l2", spec, at(0))
	// The first turn: player 2 and 3 were sent nothing, and their round
	// ends; player 1 still has a line held.
	_, drained := p.Due(at(0))
	if !reflect.DeepEqual(drained, []int{2, 3}) || !p.Busy(1) || p.Busy(2) {
		t.Fatalf("drained %v, busy1 %v busy2 %v", drained, p.Busy(1), p.Busy(2))
	}
	_, drained = p.Due(at(800))
	if !reflect.DeepEqual(drained, []int{1}) || p.Busy(1) {
		t.Fatalf("drained %v, busy1 %v", drained, p.Busy(1))
	}

	// Flush ends an open round even with nothing held.
	p.StartRound(4)
	if lines, ended := p.Flush(4); len(lines) != 0 || !ended || p.Busy(4) {
		t.Fatalf("Flush of an open round: %v %v", lines, ended)
	}
	// FlushAll ends every open round.
	p.StartRound(5)
	if _, drained := p.FlushAll(); !reflect.DeepEqual(drained, []int{5}) || p.Busy(5) {
		t.Fatalf("FlushAll drained %v", drained)
	}
}

func TestALateLineOfAnOlderRoundJoinsTheNewer(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 4, 0, "n1", spec, at(0))
	p.Hold(1, 4, 0, "n2", spec, at(0))
	if flushed := p.Hold(1, 2, 0, "late", spec, at(0)); flushed != nil {
		t.Fatalf("a late older line flushed the newer round: %v", flushed)
	}
	got := release(p, 3000)
	want := map[string]int{"n1": 0, "n2": 800, "late": 1600}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("release times %v, want %v", got, want)
	}
}

// TestABusyRoundStillFillsItsWindow: 30 lines at the normal pace would take
// 23.2s; they are squeezed into the 6s window, evenly, not into a fraction
// of a second (a regression: the scaling overflowed int64 nanoseconds).
func TestABusyRoundStillFillsItsWindow(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	for i := 0; i < 30; i++ {
		p.Hold(1, 2, 0, string(rune('A'+i)), spec, at(0))
	}
	got := release(p, 7000)
	if len(got) != 30 {
		t.Fatalf("released %d of 30", len(got))
	}
	last := got[string(rune('A'+29))]
	if last < 5900 || last > 6000 {
		t.Fatalf("last line at %dms, want the end of the 6s window", last)
	}
	if mid := got[string(rune('A'+15))]; mid < 3000 || mid > 3400 {
		t.Fatalf("line 16 at %dms, want it about halfway", mid)
	}
}

// Phase 40e: data entries ride the queue without a beat of their own.

func TestDataGoesOutWithTheNextTextLine(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 2, 0, "line one", spec, at(0))
	p.HoldData(1, 2, "d1", spec, at(0))
	p.HoldData(1, 2, "d2", spec, at(0))
	p.Hold(1, 2, 0, "line two", spec, at(0))
	p.HoldData(1, 2, "d3", spec, at(0))

	var order []string
	var times []int
	for ms := 0; ms <= 4000; ms += 50 {
		rel, _ := p.Due(at(ms))
		for _, r := range rel {
			if r.IsData {
				order = append(order, r.Data.(string))
			} else {
				order = append(order, r.Text)
			}
			times = append(times, ms)
		}
	}
	want := []string{"line one", "d1", "d2", "line two", "d3"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order %v, want %v", order, want)
	}
	// d1 and d2 are no earlier than line one and no later than line two;
	// d3, with no line after it, goes out with the last line.
	if times[1] != times[3] || times[2] != times[3] || times[4] != times[3] || times[0] >= times[3] {
		t.Fatalf("release times %v: data should share line two's moment", times)
	}
}

func TestDataAloneAndFlushedWithText(t *testing.T) {
	p := New()
	spec := Slow.ForRound(8 * time.Second)
	p.HoldData(1, 2, "only", spec, at(0))
	if !p.Busy(1) {
		t.Fatal("held data leaves a player busy")
	}
	rel, drained := p.Due(at(0))
	if len(rel) != 1 || !rel[0].IsData || !reflect.DeepEqual(drained, []int{1}) {
		t.Fatalf("data-only release %v drained %v", rel, drained)
	}

	p.Hold(2, 2, 0, "t1", spec, at(0))
	p.HoldData(2, 2, "d1", spec, at(0))
	got, ended := p.Flush(2)
	want := []Release{{UserId: 2, Text: "t1"}, {UserId: 2, Data: "d1", IsData: true}}
	if !reflect.DeepEqual(got, want) || !ended {
		t.Fatalf("Flush = %v %v, want %v", got, ended, want)
	}
}

func TestOlderRoundDataIsFlushedAheadOfANewRound(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	p.Hold(1, 2, 0, "old", spec, at(0))
	p.HoldData(1, 2, "od", spec, at(0))
	flushed := p.HoldData(1, 4, "nd", spec, at(8000))
	want := []Release{{UserId: 1, Text: "old"}, {UserId: 1, Data: "od", IsData: true}}
	if !reflect.DeepEqual(flushed, want) {
		t.Fatalf("flushed %v, want %v", flushed, want)
	}
}

func TestFollowDataWaitsOnlyBehindHeldEntries(t *testing.T) {
	p := New()
	spec := fixedSpec(8 * time.Second)
	if p.FollowData(1, "x") {
		t.Fatal("nothing held: data goes out at once")
	}
	p.Hold(1, 2, 0, "l1", spec, at(0))
	if !p.FollowData(1, "x") {
		t.Fatal("held lines: data follows them")
	}
}

func TestReportLinesGoOutWithoutAWait(t *testing.T) {
	p := New()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	spec := Normal.ForRound(6 * time.Second)
	p.Hold(1, 1, 1, "blow", spec, now)
	p.Hold(1, 1, 1, "header", spec, now)
	p.StartReport(1)
	p.Hold(1, 1, 0, "row one", spec, now)
	p.Follow(1, "row two")
	out, _ := p.Due(now.Add(spec.Gap))
	if len(out) != 4 || out[3].Text != "row two" {
		t.Fatalf("report rows waited behind the header: %v", out)
	}
}
