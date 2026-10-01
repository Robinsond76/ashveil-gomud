package mobcommands

import (
	"fmt"
	"math/rand"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// PracticeBeaten is a practice foe beaten (Phase 27c).
type PracticeBeaten struct {
	InstanceId int
	MobId      int
	RoomId     int
}

// OnPracticeBeaten fires on the game loop when a practice mob
// (mobs.Mob.Practice) is beaten. The tutorial's practice fight counts it.
var OnPracticeBeaten util.Hook[PracticeBeaten]

// mobNameTag is a mob's name as the death notices print it.
func mobNameTag(mob *mobs.Mob) string {
	return `<ansi fg="mobname">` + battle.EnemyDisplayName(mob.InstanceId, mob.Character.Name) + `</ansi>`
}

// Suicide kills mob. rest is "" (a death, with its notice), "quiet" (the
// same, but the room's death or beaten notice was already printed: Phase
// 29c's combat deaths print it in the round, in order), or "vanish" (gone
// without a trace or reward).
func Suicide(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if mob.DeathProcessed {
		return true, nil
	}
	if mob.Character.CombatWithdrawn && rest != "mercy" && rest != "vanish" {
		return true, nil
	}

	currentRound := util.GetRoundCount()
	config := configs.GetGamePlayConfig()

	if rest != `vanish` && rest != "mercy" && mob.Character.HasBuffFlag("revive-on-death") {

		mob.Character.Health = mob.Character.HealthMax.Value

		room.SendText(`<ansi fg="mobname">` + mob.Character.Name + `</ansi> is suddenly revived in a shower of sparks!`)

		mob.Character.CancelBuffsWithFlag("revive-on-death")

		return true, nil
	}

	mob.DeathProcessed = true

	// Useful to know sometimes
	mobs.TrackRecentDeath(mob.InstanceId)

	mudlog.Debug(`Mob Death`, `name`, mob.Character.Name, `rest`, rest)

	// Make sure to clean up any charm stuff if it's being removed
	if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
		if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
			charmedUser.Character.TrackCharmed(mob.InstanceId, false)
		}
	}

	// Ashveil (Phase 27c): a practice foe is beaten, not killed. It leaves
	// as a vanish does: no XP, alignment, kills, taming, drops, loot, gold,
	// or MobDeath. Its attackers keep their aim, so the next round
	// re-targets them.
	if rest != `vanish` && mob.Practice {
		if rest != `quiet` {
			room.SendText(combat.BeatenLine(mobNameTag(mob)))
		}
		OnPracticeBeaten.Fire(PracticeBeaten{InstanceId: mob.InstanceId, MobId: int(mob.MobId), RoomId: room.RoomId})
		rest = `vanish`
	}

	// vanish is meant to remove the mob without any rewards/drops/etc.
	if rest == `vanish` {

		// Destroy any record of this mob.
		mobs.DestroyInstance(mob.InstanceId)

		// Clean up mob from room...
		if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
			r.CleanupMobSpawns(false)
		}

		// Remove from current room
		room.RemoveMob(mob.InstanceId)

		return true, nil
	}

	// Send a death msg to everyone in the room. Phase 29c: a combat death
	// ("suicide quiet") has already printed it, in order, from the round.
	if rest != `quiet` && rest != "mercy" {
		room.SendText(combat.DeathLine(mobNameTag(mob)))
	}

	// Special handling of "The Guide"
	// Mark this moment to prevent an immediate respawn
	if mob.MobId == 38 {
		if mob.Character.Charmed != nil {
			if tmpU := users.GetByUserId(mob.Character.Charmed.UserId); tmpU != nil {
				tmpU.SetTempData(`lastGuideRound`, currentRound)
			}
		}
	}

	mobXP := mob.Character.XPTL(mob.Character.Level - 1)

	// Phase 25b: the worn-item drops are rolled here, before the death is
	// announced, so the event can say what stays on the body.
	permaGear := mob.Character.HasBuffFlag("perma-gear")
	body := bodyKeeps(mob, permaGear)

	killedByUsers := make([]int, 0, len(mob.Character.PlayerDamage))
	for uId := range mob.Character.PlayerDamage {
		killedByUsers = append(killedByUsers, uId)
	}

	events.AddToQueue(events.MobDeath{
		MobId:         int(mob.MobId),
		InstanceId:    mob.InstanceId,
		RoomId:        room.RoomId,
		CharacterName: mob.Character.Name,
		Level:         mob.Character.Level,
		PlayerDamage:  mob.Character.PlayerDamage,
		KilledByUsers: killedByUsers,
		KeptWorn:      body.keptWorn,
		KeptItems:     body.keptItems,
		KeptGold:      body.keptGold,
	})

	contributors := eligibleContributors(mob, room.RoomId, rest == "mercy")
	shares := rewardShares(mob, mobXP, contributors)
	for _, uid := range contributors {
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		share := shares[uid]
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId == mob.InstanceId {
			user.Character.Aggro = nil
			events.AddToQueue(events.AggroChanged{UserId: uid, RoomId: room.RoomId})
		}
		scripting.TryMobScriptEvent("onDie", mob.InstanceId, uid, "user", map[string]any{"attackerCount": len(contributors)})
		if mob.Character.Zone != `Training` { // Don't track any kills in the training zone
			user.Character.KD.AddMobKill(int(mob.MobId))
			if mob.IsElite {
				user.Character.KD.AddEliteKill(int(mob.MobId), mob.Character.Name)
			}
		}

		user.GrantXP(share, `combat`)
		for _, line := range awardCompanyXP(user.UserId, user.Character, share, room.RoomId) {
			user.SendText(line)
		}

		// Apply alignment changes
		alignmentBefore := user.Character.AlignmentName()
		alignmentAdj := combat.AlignmentChange(user.Character.Alignment, mob.Character.Alignment)
		if rest == "mercy" {
			alignmentAdj = 0
		}
		user.Character.UpdateAlignment(alignmentAdj)
		alignmentAfter := user.Character.AlignmentName()

		mudlog.Debug("Alignment", "user Alignment", user.Character.Alignment, "mob Alignment", mob.Character.Alignment, `alignmentAdj`, alignmentAdj, `alignmentBefore`, alignmentBefore, `alignmentAfter`, alignmentAfter)

		if alignmentBefore != alignmentAfter {
			before := fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentBefore, alignmentBefore)
			after := fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignmentAfter, alignmentAfter)
			updateTxt := fmt.Sprintf(`<ansi fg="231">Your alignment has shifted from %s to %s!</ansi>`, before, after)
			user.SendText(updateTxt)
			events.AddToQueue(events.AlignmentChanged{
				UserId:       user.UserId,
				AlignmentOld: alignmentBefore,
				AlignmentNew: alignmentAfter,
			})
		}

	}

	// Shared kills use a fixed claim, not the first pickup command. A corpse is
	// used even in worlds configured for floor drops, so gold/items cannot leak
	// through an unowned floor path. The corpse owns the physical loot once.
	claimCorpse := len(contributors) > 1 && !permaGear
	claimOwner := 0
	if claimCorpse {
		claimOwner = lootClaimant(mob, contributors)
		room.SendText(fmt.Sprintf(`Battle loot from %s is claimed by <ansi fg="username">%s</ansi>.`, mobNameTag(mob), users.CharacterName(claimOwner)))
	}

	if !permaGear {

		corpseItems := []items.Item{}
		corpseGold := 0

		// Check for any dropped loot...
		for _, item := range mob.Character.Items {
			if bool(config.Death.CorpseItems && config.Death.CorpsesEnabled) || claimCorpse {
				corpseItems = append(corpseItems, item)
			} else {
				msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
				room.SendText(msg)
				events.AddToQueue(events.MobItemDrop{
					MobId:  int(mob.MobId),
					RoomId: room.RoomId,
					Zone:   mob.Character.Zone,
					ItemId: item.ItemId,
				})
				room.AddItem(item, false)
			}
		}

		for _, item := range body.dropWorn {

			if bool(config.Death.CorpseItems && config.Death.CorpsesEnabled) || claimCorpse {
				corpseItems = append(corpseItems, item)
			} else {
				msg := fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName())
				room.SendText(msg)
				events.AddToQueue(events.MobItemDrop{
					MobId:  int(mob.MobId),
					RoomId: room.RoomId,
					Zone:   mob.Character.Zone,
					ItemId: item.ItemId,
				})
				room.AddItem(item, false)
			}
		}

		if mob.LootCategory != "" {
			if table, found := loot.GetTable(mob.LootCategory); found {
				roll := rand.Uint64()
				if entry, ok := table.Resolve(roll); ok && items.GetItemSpec(entry.ItemID) != nil {
					count := entry.RollCount(rand.Uint64())
					mudlog.Debug("Category Loot Roll", "category", mob.LootCategory, "roll", roll, "itemID", entry.ItemID, "count", count)
					for i := 0; i < count; i++ {
						item := items.New(entry.ItemID)
						if bool(config.Death.CorpseItems && config.Death.CorpsesEnabled) || claimCorpse {
							corpseItems = append(corpseItems, item)
						} else {
							room.SendText(fmt.Sprintf(`<ansi fg="item">%s</ansi> drops to the ground.`, item.DisplayName()))
							events.AddToQueue(events.MobItemDrop{MobId: int(mob.MobId), RoomId: room.RoomId, Zone: mob.Character.Zone, ItemId: item.ItemId})
							room.AddItem(item, false)
						}
					}
				}
			}
		}

		if mob.Character.Gold > 0 {
			if bool(config.Death.CorpseItems && config.Death.CorpsesEnabled) || claimCorpse {
				corpseGold = mob.Character.Gold
			} else {
				msg := fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi> drops to the ground.`, mob.Character.Gold)
				room.SendText(msg)
				room.Gold += mob.Character.Gold
			}
		}

		// Destroy any record of this mob.
		mobs.DestroyInstance(mob.InstanceId)

		// Clean up mob from room...
		if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
			r.CleanupMobSpawns(false)
		}

		// Remove from current room
		room.RemoveMob(mob.InstanceId)

		if bool(config.Death.CorpsesEnabled) || claimCorpse {
			c := rooms.Corpse{
				ClaimUserId:  claimOwner,
				MobId:        int(mob.MobId),
				Character:    mob.Character,
				RoundCreated: currentRound,
			}
			if bool(config.Death.CorpseItems) || claimCorpse {
				c.Items = corpseItems
				c.Gold = corpseGold
			}
			// Rolled drops already live in c.Items (or on the floor); remove
			// them from the corpse's worn view so one item never has two owners.
			for _, item := range body.dropWorn {
				c.Character.RemoveFromBody(item)
			}
			room.AddCorpse(c)
		}

		return true, nil
	}

	// Destroy any record of this mob.
	mobs.DestroyInstance(mob.InstanceId)

	// Clean up mob from room...
	if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
		r.CleanupMobSpawns(false)
	}

	// Remove from current room
	room.RemoveMob(mob.InstanceId)

	if config.Death.CorpsesEnabled {
		room.AddCorpse(rooms.Corpse{
			MobId:        int(mob.MobId),
			Character:    mob.Character,
			RoundCreated: currentRound,
		})
	}

	return true, nil
}

// deathBody is what a dying mob's drop rules leave on its body and what they
// drop from its worn gear.
type deathBody struct {
	dropWorn  []items.Item
	keptWorn  map[items.ItemType]items.Item
	keptItems []items.Item
	keptGold  int
}

// bodyKeeps rolls a dying mob's worn-item drops: each worn item that isn't
// remove-locked drops with ItemDropChance percent. A perma-gear mob drops
// nothing, keeping its carried items and gold too; any other mob always
// drops those.
func bodyKeeps(mob *mobs.Mob, permaGear bool) deathBody {
	body := deathBody{}
	for _, slot := range items.AllEquipSlots() {
		item := mob.Character.Equipment.Get(slot)
		if item == nil || item.ItemId <= 0 {
			continue
		}
		if !permaGear && !item.IsRemoveLocked() {
			roll := util.Rand(100)
			util.LogRoll(`Drop Item`, roll, mob.ItemDropChance)
			if roll < mob.ItemDropChance {
				body.dropWorn = append(body.dropWorn, *item)
				continue
			}
		}
		if body.keptWorn == nil {
			body.keptWorn = map[items.ItemType]items.Item{}
		}
		body.keptWorn[slot] = *item
	}
	if permaGear {
		body.keptItems = append([]items.Item(nil), mob.Character.Items...)
		body.keptGold = mob.Character.Gold
	}
	return body
}
