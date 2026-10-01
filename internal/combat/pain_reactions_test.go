package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestPainReactionSelectionAndViewpoints(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()

	human := characters.New()
	human.Name = "Garrick Vane"
	human.RaceId = 1
	gotVictim, gotRoom := painReactionFor(human, User, nil, 4, 1)
	if !strings.HasPrefix(gotVictim, "Pain flashes through you") || !strings.Contains(gotRoom, "Garrick Vane") || strings.Contains(gotRoom, "the Garrick") {
		t.Fatalf("human fallback: victim %q, room %q", gotVictim, gotRoom)
	}

	beast := characters.New()
	beast.Name = "timber wolf"
	beast.RaceId = 11
	gotVictim, gotRoom = painReactionFor(beast, Mob, nil, 4, 1)
	if !strings.Contains(gotRoom, "The timber wolf") || !strings.Contains(gotVictim, "you") {
		t.Fatalf("beast reaction: victim %q, room %q", gotVictim, gotRoom)
	}

	override := &mobs.Mob{PainReactions: []races.PainReaction{{ToVictim: "You grip the wound.", ToRoom: "{name} grips {his} wound."}}}
	beast.Pronouns = "she"
	gotVictim, gotRoom = painReactionFor(beast, Mob, override, 4, 1)
	if gotVictim != "You grip the wound." || gotRoom != "The timber wolf grips her wound." {
		t.Fatalf("NPC override: victim %q, room %q", gotVictim, gotRoom)
	}

	golem := characters.New()
	golem.Name = "stone golem"
	golem.RaceId = 16 // it, with no authored set
	gotVictim, gotRoom = painReactionFor(golem, Mob, nil, 4, 1)
	if !strings.HasPrefix(gotVictim, "Pain jolts through you") || !strings.Contains(gotRoom, "recoils from the blow") {
		t.Fatalf("generic creature fallback: victim %q, room %q", gotVictim, gotRoom)
	}
}

func TestPainVariantUsesVictimNameWithoutConsumingCombatRNG(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()
	a := characters.New()
	a.Name, a.RaceId = "A", 11
	b := characters.New()
	b.Name, b.RaceId = "B", 11
	aVictim, _ := painReactionFor(a, Mob, nil, 4, 1)
	bVictim, _ := painReactionFor(b, Mob, nil, 4, 1)
	if aVictim == bVictim {
		t.Fatalf("distinct victim names selected the same reaction at the same health and strike: %q", aVictim)
	}
}

func TestLaterLethalStrikeDoesNotReactAfterEarlierCritical(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	gameplay.Combat.CritMultMin, gameplay.Combat.CritMultMax = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	const doubleStrikeID = 99241
	items.SetTestItemSpec(&items.ItemSpec{ItemId: doubleStrikeID, Name: "test double strike", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Damage: items.Damage{DiceRoll: "1d1", Attacks: 2, DiceCount: 1, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(doubleStrikeID) })
	source := edgeFighter(90231)
	source.Name = "Aria"
	source.Equipment.Weapon = items.New(doubleStrikeID)
	target := edgeFighter(90231)
	target.Name = "timber wolf"
	target.RaceId = 11
	target.Health = 2
	result := calculateCombat(*source, *target, User, Mob, 0, 0)
	if result.DamageToTarget != 2 || len(result.MessagesToTarget) != 3 {
		t.Fatalf("two critical hits, only first surviving, need hit/reaction/hit: %+v", result)
	}
	if !strings.Contains(result.MessagesToTarget[0], "critical hit") || !strings.Contains(result.MessagesToTarget[2], "critical hit") || !strings.Contains(result.MessagesToTarget[1], "you") {
		t.Fatalf("wrong per-strike order: %q", result.MessagesToTarget)
	}
}

func TestOrdinaryHitHasNoPainReaction(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	target := edgeFighter(90231)
	target.Health = 100
	result := calculateCombat(*source, *target, User, Mob, 0, 0)
	if result.DamageToTarget <= 0 || len(result.MessagesToTarget) != 1 {
		t.Fatalf("ordinary hit produced a reaction or failed to hit: %+v", result)
	}
}

func TestDodgedStrikeHasNoPainReaction(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	target := edgeFighter(90231)
	result := calculateCombat(*source, *target, User, Mob, 0, 0)
	if result.DamageToTarget != 0 || len(result.MessagesToTarget) != 1 || !strings.Contains(result.MessagesToTarget[0], "twist aside") {
		t.Fatalf("dodge caused damage or a pain reaction: %+v", result)
	}
}

func TestFullyBlockedCriticalHasNoPainReaction(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	const plateID = 99242
	items.SetTestItemSpec(&items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, DamageReduction: 100000})
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })
	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	target := edgeFighter(90231)
	target.Equipment.Body = items.New(plateID)
	for i := 0; i < 200; i++ {
		result := calculateCombat(*source, *target, User, Mob, 0, 0)
		if result.DamageToTarget != 0 {
			continue
		}
		if !result.Crit || len(result.MessagesToTarget) != 1 {
			t.Fatalf("absorbed critical announced pain: %+v", result)
		}
		return
	}
	t.Fatal("no fully absorbed critical strike in 200 attempts")
}

func TestSeparateRoomCriticalSendsWitnessReactionToBothRooms(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	otherRoom := &rooms.Room{RoomId: 90232, Zone: "Test", Biome: "city", Tags: []string{rooms.TagLit}}
	rooms.SetTestRoom(otherRoom)
	t.Cleanup(func() { rooms.RemoveTestRoom(otherRoom.RoomId) })
	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	target := edgeFighter(90232)
	target.Name = "timber wolf"
	target.RaceId = 11
	result := calculateCombat(*source, *target, User, Mob, 0, 0)
	if len(result.MessagesToSourceRoom) != 2 || len(result.MessagesToTargetRoom) == 0 || result.MessagesToSourceRoom[1] != result.MessagesToTargetRoom[len(result.MessagesToTargetRoom)-1] {
		t.Fatalf("separate-room witnesses did not share one reaction: %+v", result)
	}
}

func TestSurvivingCriticalStrikeGetsImmediatePainLineForEachViewer(t *testing.T) {
	edgeSpecs(t)
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	source := edgeFighter(90231)
	source.Name = "Aria"
	source.Equipment.Weapon = items.New(edgeSwordID)
	source.SetAggro(0, 1, characters.BackStab)
	target := edgeFighter(90231)
	target.Name = "timber wolf"
	target.RaceId = 11
	target.Health = 100
	for i := 0; i < 200; i++ {
		result := calculateCombat(*source, *target, User, Mob, 0, 0)
		if !result.Crit || result.DamageToTarget <= 0 {
			continue
		}
		if len(result.MessagesToTarget) != 2 || len(result.MessagesToSource) != 2 || len(result.MessagesToSourceRoom) != 2 {
			t.Fatalf("a critical hit needs a hit then one reaction for victim, attacker, and room: %+v", result)
		}
		if !strings.Contains(result.MessagesToTarget[0], "critical hit") || !strings.Contains(result.MessagesToTarget[1], "you") {
			t.Fatalf("victim did not get second-person pain after the hit: %q", result.MessagesToTarget)
		}
		if !strings.Contains(result.MessagesToSource[1], "The timber wolf") || result.MessagesToSource[1] != result.MessagesToSourceRoom[1] {
			t.Fatalf("witness reaction differs or loses target label: %q, %q", result.MessagesToSource, result.MessagesToSourceRoom)
		}
		// Phase 29f: both viewpoints of the pain line wait the longer gap;
		// the hit itself does not.
		pacer := combatpace.Default()
		if !pacer.Marked(result.MessagesToTarget[1]) || !pacer.Marked(result.MessagesToSource[1]) {
			t.Fatalf("pain lines not marked dramatic: %q, %q", result.MessagesToTarget[1], result.MessagesToSource[1])
		}
		if pacer.Marked(result.MessagesToTarget[0]) || pacer.Marked(result.MessagesToSource[0]) {
			t.Fatalf("the critical hit line itself was marked")
		}
		return
	}
	t.Fatal("no damaging critical hit in 200 attempts")
}

func TestLethalCriticalStrikeHasNoPainLine(t *testing.T) {
	edgeSpecs(t)
	source := edgeFighter(90231)
	source.Name = "Aria"
	source.Equipment.Weapon = items.New(edgeSwordID)
	source.SetAggro(0, 1, characters.BackStab)
	target := edgeFighter(90231)
	target.Name = "timber wolf"
	target.RaceId = 11
	target.Health = 1
	for i := 0; i < 200; i++ {
		result := calculateCombat(*source, *target, User, Mob, 0, 0)
		if !result.Crit || result.DamageToTarget <= 0 {
			continue
		}
		if len(result.MessagesToTarget) != 1 || len(result.MessagesToSource) != 1 || len(result.MessagesToSourceRoom) != 1 {
			t.Fatalf("lethal critical hit announced pain: %+v", result)
		}
		return
	}
	t.Fatal("no damaging critical hit in 200 attempts")
}

func TestShippedBeastRacesHaveDistinctPainReactions(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()
	seen := map[string]int{}
	for _, raceID := range []int{7, 8, 10, 11, 14, 18, 21} {
		race := races.GetRace(raceID)
		if race == nil || len(race.PainReactions) == 0 {
			t.Errorf("race %d has no authored pain reaction", raceID)
			continue
		}
		for _, pair := range race.PainReactions {
			if err := races.ValidatePainReactions([]races.PainReaction{pair}); err != nil {
				t.Errorf("race %d: %v", raceID, err)
			}
			if other := seen[pair.ToRoom]; other != 0 {
				t.Errorf("races %d and %d share a supposedly distinct line", other, raceID)
			}
			seen[pair.ToRoom] = raceID
		}
	}
}
