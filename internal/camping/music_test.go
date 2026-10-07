package camping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func player(key string, f Family, level, tier int) Player {
	return Player{Key: key, Name: key, Family: f, Level: level, Tier: tier}
}

func TestMusicSkillLevelsFromPractice(t *testing.T) {
	for songs, want := range map[int]int{0: 1, 9: 1, 10: 2, 24: 2, 25: 3, 49: 3, 50: 4, 500: 4} {
		assert.Equal(t, want, MusicSkill{Family: FamilyStrings, Songs: songs}.Level(), "songs %d", songs)
	}
	_, more := MusicSkill{Family: FamilyStrings, Songs: 50}.SongsToNext()
	assert.False(t, more, "level 4 is the top")
	left, more := MusicSkill{Family: FamilyStrings, Songs: 4}.SongsToNext()
	assert.True(t, more)
	assert.Equal(t, 6, left)
}

func TestPlayerStrengthRanges(t *testing.T) {
	assert.Equal(t, 2, player("a", FamilyStrings, 1, InstrumentCrude).Strength())
	assert.Equal(t, 8, player("a", FamilyStrings, 4, InstrumentMasterwork).Strength())
	assert.Equal(t, 2, player("v", FamilyVoice, 1, 0).Strength(), "voice is level plus one")
	assert.Equal(t, 5, player("v", FamilyVoice, 4, 0).Strength())
}

func TestPlanSongPicksBestPlayerPerFamilyAndAllPractise(t *testing.T) {
	song := PlanSong([]Player{
		player("a", FamilyStrings, 1, InstrumentCommon),
		player("b", FamilyStrings, 3, InstrumentFine),
		player("c", FamilyDrums, 2, InstrumentCrude),
		player("d", FamilyWinds, 0, InstrumentCrude), // level 0 cannot play
	}, false)
	assert.Len(t, song.Plays, 2)
	assert.Equal(t, 6, song.Strength(FamilyStrings))
	assert.Equal(t, 3, song.Strength(FamilyDrums))
	assert.Len(t, song.Players, 3, "everyone who played practises")
	assert.Empty(t, PlanSong(nil, false).Plays)
}

func TestWetHalvesStringsAndWindsButNotDrumsOrVoice(t *testing.T) {
	players := []Player{
		player("a", FamilyStrings, 2, InstrumentFine),
		player("b", FamilyWinds, 2, InstrumentFine),
		player("c", FamilyDrums, 2, InstrumentFine),
		player("d", FamilyVoice, 3, 0),
	}
	dry := PlanSong(players, false)
	wet := PlanSong(players, true)
	assert.Equal(t, 5, dry.Strength(FamilyStrings))
	assert.Equal(t, 3, wet.Strength(FamilyStrings))
	assert.Equal(t, 3, wet.Strength(FamilyWinds))
	assert.Equal(t, dry.Strength(FamilyDrums), wet.Strength(FamilyDrums))
	assert.Equal(t, dry.Strength(FamilyVoice), wet.Strength(FamilyVoice))
	assert.True(t, Wet("Heavy Rain"))
	assert.True(t, Wet("snow"))
	assert.False(t, Wet("clear"))
}

func TestRainproofInstrumentShrugsOffWeather(t *testing.T) {
	p := player("a", FamilyStrings, 2, InstrumentMasterwork)
	p.Quirk = QuirkRainproof
	assert.Equal(t, 6, PlanSong([]Player{p}, true).Strength(FamilyStrings))
}

func TestEffectCapsAndEnsembleThresholds(t *testing.T) {
	max4 := func(f Family) Player {
		if f == FamilyVoice {
			return player("v", f, 4, 0)
		}
		return player(string(f), f, 4, InstrumentMasterwork)
	}
	full := PlanSong([]Player{max4(FamilyStrings), max4(FamilyWinds), max4(FamilyDrums), max4(FamilyVoice)}, false)
	assert.True(t, full.Ensemble())
	assert.True(t, full.Full())
	assert.Equal(t, 50, full.RestedPct(), "40% capped, a quarter more")
	assert.Equal(t, 10, full.FatigueBonus())
	assert.Equal(t, 5, full.DrumSpeed())
	assert.Equal(t, 31, full.AilmentPct(), "voice 5 gives 25%, a quarter more is 31")
}

func TestEnsembleNeedsThreeStrongFamilies(t *testing.T) {
	strong := func(key string, f Family) Player { return player(key, f, 2, InstrumentFine) } // strength 5
	weak := player("w", FamilyVoice, 1, 0)                                                   // strength 2
	three := PlanSong([]Player{strong("a", FamilyStrings), strong("b", FamilyWinds), strong("c", FamilyDrums), weak}, false)
	assert.Equal(t, 3, three.StrongFamilies())
	assert.True(t, three.Ensemble())
	assert.False(t, three.Full())
	two := PlanSong([]Player{strong("a", FamilyStrings), strong("b", FamilyWinds), weak}, false)
	assert.False(t, two.Ensemble())
	// A wet night drops strings and winds under four: no ensemble.
	assert.False(t, PlanSong([]Player{player("a", FamilyStrings, 1, InstrumentCommon), player("b", FamilyWinds, 1, InstrumentCommon), strong("c", FamilyDrums), strong("d", FamilyVoice)}, true).Ensemble())
}

func TestSingleFamilyEffectsScaleWithStrength(t *testing.T) {
	s := PlanSong([]Player{player("a", FamilyStrings, 1, InstrumentCommon)}, false) // 3
	assert.Equal(t, 15, s.RestedPct())
	assert.Equal(t, 0, s.FatigueBonus())
	assert.Equal(t, 0, s.DrumSpeed())
	assert.Equal(t, 0, s.AilmentPct())
	w := PlanSong([]Player{player("a", FamilyWinds, 2, InstrumentFine)}, false) // 5
	assert.Equal(t, 5, w.FatigueBonus())
	d := PlanSong([]Player{player("a", FamilyDrums, 1, InstrumentFine)}, false) // 4
	assert.Equal(t, 2, d.DrumSpeed())
	v := PlanSong([]Player{player("a", FamilyVoice, 3, 0)}, false) // 4
	assert.Equal(t, 20, v.AilmentPct())
}

func TestRaidPctScalesPerFamilyDrumsDouble(t *testing.T) {
	one := PlanSong([]Player{player("a", FamilyStrings, 1, InstrumentCrude)}, false)
	assert.Equal(t, 110, one.RaidPct())
	drums := PlanSong([]Player{player("a", FamilyDrums, 1, InstrumentCrude)}, false)
	assert.Equal(t, 120, drums.RaidPct())
	all := PlanSong([]Player{player("a", FamilyStrings, 1, InstrumentCrude), player("b", FamilyWinds, 1, InstrumentCrude), player("c", FamilyDrums, 1, InstrumentCrude), player("d", FamilyVoice, 1, 0)}, false)
	assert.Equal(t, 150, all.RaidPct())
	assert.Equal(t, 100, Song{}.RaidPct(), "no song, no change")
}

func TestHushedAndMuffledMasterworksCutTheRaidCost(t *testing.T) {
	hushed := player("a", FamilyWinds, 1, InstrumentMasterwork)
	hushed.Quirk = QuirkHushed
	assert.Equal(t, 100, PlanSong([]Player{hushed}, false).RaidPct())
	muffled := player("a", FamilyDrums, 1, InstrumentMasterwork)
	muffled.Quirk = QuirkMuffled
	assert.Equal(t, 110, PlanSong([]Player{muffled}, false).RaidPct())
}

func TestFadedBattles(t *testing.T) {
	assert.Equal(t, 0, FadedBattles(0, 40))
	assert.Equal(t, 3, FadedBattles(3, 0))
	assert.Equal(t, 2, FadedBattles(3, 10), "any pct shortens by at least one")
	assert.Equal(t, 0, FadedBattles(1, 5))
	assert.Equal(t, 3, FadedBattles(5, 40))
}

func TestDrumBuffIDs(t *testing.T) {
	assert.Equal(t, 0, DrumBuffFor(0))
	assert.Equal(t, 9401, DrumBuffFor(1))
	assert.Equal(t, 9405, DrumBuffFor(9))
	assert.True(t, IsDrumBuff(9403))
	assert.False(t, IsDrumBuff(9300))
}

func TestGigWindow(t *testing.T) {
	for hour, want := range map[int]bool{18: false, 19: true, 20: true, 21: false, 3: false} {
		assert.Equal(t, want, InGigWindow(hour), "hour %d", hour)
	}
}

func twoFamilies() Song {
	return PlanSong([]Player{player("a", FamilyStrings, 2, InstrumentCommon), player("b", FamilyDrums, 1, InstrumentCrude)}, false)
}

func TestGigPayScalesWithBandAndStrengthAndEnsemble(t *testing.T) {
	song := twoFamilies() // 4 + 2
	low := GigPay(5, song)
	assert.Greater(t, GigPay(15, song), low)
	big := PlanSong([]Player{player("a", FamilyStrings, 4, InstrumentMasterwork), player("b", FamilyDrums, 3, InstrumentFine)}, false)
	assert.Greater(t, GigPay(5, big), low)
	assert.Equal(t, GigPay(DefaultGigBand, song), GigPay(0, song), "no band pays the default")
	ens := PlanSong([]Player{player("a", FamilyStrings, 2, InstrumentFine), player("b", FamilyWinds, 2, InstrumentFine), player("c", FamilyDrums, 2, InstrumentFine)}, false)
	assert.Greater(t, GigPay(5, ens), (5+2)*ens.Strength(FamilyStrings)*3/2, "ensemble bonus")
}

func TestGigRefusalLimits(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	song := twoFamilies()
	var log GigLog
	assert.Equal(t, "", log.GigRefusal(100, 19, 5, now, song))
	assert.Contains(t, log.GigRefusal(100, 12, 5, now, song), "19:00 to 21:00")
	assert.Contains(t, log.GigRefusal(100, 19, 5, now, PlanSong([]Player{player("a", FamilyStrings, 1, InstrumentCrude)}, false)), "at least 2 families")

	log = log.Started(Gig{RoomID: 100, StartedAtUTC: now, Pay: 10, Song: song, Day: 5})
	assert.Contains(t, log.GigRefusal(100, 19, 5, now, song), "already playing")
	log = log.Paid(now)
	assert.Contains(t, log.GigRefusal(200, 19, 5, now.Add(time.Hour), song), "not long ago", "real-time cooldown")
	assert.Contains(t, log.GigRefusal(100, 19, 5, now.Add(4*time.Hour), song), "this evening", "once per inn per evening")
	assert.Equal(t, "", log.GigRefusal(200, 19, 5, now.Add(4*time.Hour), song), "another inn is fine")
	assert.Equal(t, "", log.GigRefusal(100, 19, 6, now.Add(4*time.Hour), song), "the next evening is fine")
}

func TestGigDueAndRemaining(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	g := Gig{RoomID: 1, StartedAtUTC: start, Pay: 5}
	assert.False(t, g.Due(start.Add(GigDuration-time.Second)))
	assert.True(t, g.Due(start.Add(GigDuration)))
	assert.Equal(t, GigDuration, g.Remaining(start))
	assert.Equal(t, time.Duration(0), g.Remaining(start.Add(2*GigDuration)))
}
