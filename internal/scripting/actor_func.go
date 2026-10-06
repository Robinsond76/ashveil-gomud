package scripting

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

func setActorFunctions(vm registrar) {
	vm.Set(`GetUser`, GetUser)
	vm.Set(`GetMob`, GetMob)
	vm.Set(`ActorNames`, ActorNames)
}

type ScriptActor struct {
	userId          int
	mobInstanceId   int
	userRecord      *users.UserRecord
	mobRecord       *mobs.Mob
	characterRecord *characters.Character // Lets us bypass the user/mob check in many cases
}

func (a ScriptActor) UserId() int {
	return a.userId
}

func (a ScriptActor) InstanceId() int {
	return a.mobInstanceId
}

func (a ScriptActor) MobTypeId() int {
	if a.mobRecord != nil {
		return int(a.mobRecord.MobId)
	}
	return 0
}

func (a ScriptActor) GetRace() string {
	return a.characterRecord.Race()
}

func (a ScriptActor) GetTrueRace() string {
	if r := races.GetRace(a.characterRecord.RaceId); r != nil {
		return r.Name
	}
	return `Ghostly Spirit`
}

func (a ScriptActor) IsFormChanged() bool {
	return a.characterRecord.IsFormChanged()
}

func (a ScriptActor) ApplyFormChange(raceId int) bool {
	if races.GetRace(raceId) == nil {
		return false
	}
	a.characterRecord.ApplyFormChange(raceId)
	return true
}

func (a ScriptActor) RevertFormChange() bool {
	if !a.characterRecord.IsFormChanged() {
		return false
	}
	a.characterRecord.RevertFormChange()
	return true
}

func (a ScriptActor) GetSize() string {
	if r := races.GetRace(a.characterRecord.GetRaceId()); r != nil {
		return string(r.Size)
	}
	return string(races.Medium)
}

func (a ScriptActor) SendText(msg string) {
	if a.userRecord == nil {
		return
	}

	msg = userTextWrap.Wrap(msg)

	a.userRecord.SendText(msg)
}

func (a ScriptActor) GetLevel() int {
	return a.characterRecord.Level
}

func (a ScriptActor) GetStat(statName string) int {

	statName = strings.ToLower(statName)

	if strings.HasPrefix(statName, "st") {
		return a.characterRecord.Stats.Strength.ValueAdj
	}

	if strings.HasPrefix(statName, "sp") {
		return a.characterRecord.Stats.Speed.ValueAdj
	}

	if strings.HasPrefix(statName, "sm") {
		return a.characterRecord.Stats.Smarts.ValueAdj
	}

	if strings.HasPrefix(statName, "vi") {
		return a.characterRecord.Stats.Vitality.ValueAdj
	}

	if strings.HasPrefix(statName, "my") {
		return a.characterRecord.Stats.Mysticism.ValueAdj
	}

	if strings.HasPrefix(statName, "pe") {
		return a.characterRecord.Stats.Perception.ValueAdj
	}

	return 0
}

func (a ScriptActor) SetResetRoomId(roomId int) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.Character.RoomIdOnReset = roomId
}

func (a ScriptActor) SetTempData(key string, value any) {

	if a.userRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.userRecord = nil
			value = userValue
		}
		a.userRecord.SetTempData(key, value)
		return
	}

	if a.mobRecord != nil {
		if userValue, ok := value.(ScriptActor); ok { // Don't store pointer to user data.
			userValue.mobRecord = nil
			value = userValue
		}
		a.mobRecord.SetTempData(key, value)
		return
	}
}

func (a ScriptActor) GetTempData(key string) any {

	if a.userRecord != nil {
		if value := a.userRecord.GetTempData(key); value != nil {
			if userValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(userValue.userId, 0)
			}
			return value
		}
	} else if a.mobRecord != nil {
		if value := a.mobRecord.GetTempData(key); value != nil {
			if mobValue, ok := value.(ScriptActor); ok { // If it was userdata we need to reload the whole thing in case the user isn't around anymore.
				value = GetActor(0, mobValue.mobInstanceId)
			}
			return value
		}
	}
	return nil
}

func (a ScriptActor) SetMiscCharacterData(key string, value any) {

	if _, ok := value.(ScriptActor); ok { // Don't store actor data.
		return
	}
	a.characterRecord.SetMiscData(key, value)
}

func (a ScriptActor) GetMiscCharacterData(key string) any {
	if value := a.characterRecord.GetMiscData(key); value != nil {
		return value
	}
	return nil
}

func (a ScriptActor) GetMiscCharacterDataKeys(prefixMatches ...string) []string {
	return a.characterRecord.GetMiscDataKeys(prefixMatches...)
}

// GetCombatName is the actor's coloured name as combat narration prints it
// (Phase 29c): a mob's with its article ("the bandit captain"), capitalised
// when startOfLine; a player's as it is.
func (a ScriptActor) GetCombatName(startOfLine bool) string {
	name := a.GetCharacterName(true)
	if a.mobRecord != nil {
		name = `<ansi fg="mobname">` + battle.EnemyDisplayName(a.mobInstanceId, a.characterRecord.Name) + `</ansi>`
		name = util.Article(name)
	}
	if startOfLine {
		name = util.CapitalizeFirst(name)
	}
	return name
}

// GetCombatPronoun returns subject, object, or possessive combat pronouns.
// Players always use they forms, including while polymorphed.
func (a ScriptActor) GetCombatPronoun(form string) string {
	if a.characterRecord == nil {
		return ""
	}
	pronouns := a.characterRecord.CombatPronouns()
	if a.userRecord != nil {
		pronouns = characters.PronounFormsFor("they")
	}
	switch form {
	case "subject":
		return pronouns.Subject
	case "object":
		return pronouns.Object
	case "possessive":
		return pronouns.Possessive
	}
	return ""
}

// ChantRoundsLeft is how many rounds from now the actor's spell is
// released, the release round included (Phase 29c): 1 means it lands next
// round. 0 when the actor isn't casting.
func (a ScriptActor) ChantRoundsLeft() int {
	if aggro := a.characterRecord.Aggro; aggro != nil && aggro.Type == characters.SpellCast {
		return aggro.RoundsWaiting + 1
	}
	return 0
}

func (a ScriptActor) GetCharacterName(wrapInTags bool) string {

	if wrapInTags {
		if a.userRecord != nil {
			return `<ansi fg="username">` + a.characterRecord.Name + `</ansi>`
		} else if a.mobRecord != nil {
			return `<ansi fg="mobname">` + a.characterRecord.Name + `</ansi>`
		}
	}

	return a.characterRecord.Name
}

func (a ScriptActor) SetCharacterName(newName string) {
	a.characterRecord.Name = newName
}

func (a ScriptActor) GetRoomId() int {
	return a.characterRecord.RoomId
}

func (a ScriptActor) HasQuest(questId string) bool {
	return a.characterRecord.HasQuest(questId)
}

func (a ScriptActor) GiveQuest(questId string) {

	events.AddToQueue(events.Quest{
		UserId:     a.UserId(),
		QuestToken: questId,
	})

}

// Gets a party object that indexes ALL members of the party
func (a ScriptActor) GetParty(excludeSelf ...bool) ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: true,
		includeMissing: true,
		includeSelf:    len(excludeSelf) == 0 || !excludeSelf[0],
	}
}

// Gets a party object that indexes only party members in the same room
func (a ScriptActor) GetPartyPresent(excludeSelf ...bool) ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: true,
		includeMissing: false,
		includeSelf:    len(excludeSelf) == 0 || !excludeSelf[0],
	}
}

// Gets a party object that indexes only members NOT in the same room.
func (a ScriptActor) GetPartyMissing() ScriptParty {
	return ScriptParty{
		actor:          &a,
		includePresent: false,
		includeMissing: true,
		includeSelf:    false,
	}
}

func (a ScriptActor) AddGold(amt int, bankAmt ...int) {
	if !a.prepareUserAssets() {
		return
	}
	a.characterRecord.Gold += amt
	if a.characterRecord.Gold < 0 {
		a.characterRecord.Gold = 0
	}
	if len(bankAmt) > 0 {
		a.characterRecord.Bank += bankAmt[0]
		if a.characterRecord.Bank < 0 {
			a.characterRecord.Bank = 0
		}
	}
}

func (a ScriptActor) AddHealth(amt int) int {
	ret := a.characterRecord.ApplyHealthChange(amt)

	// Phase 38a: the first damage wakes a sleeper.
	if ret < 0 && status.Wake(a.characterRecord) {
		a.wakeLines()
	}

	if ret != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}

	return ret
}

func (a ScriptActor) AddMana(amt int) int {
	ret := a.characterRecord.ApplyManaChange(amt)

	if ret != 0 && a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}

	return ret
}

func (a ScriptActor) Sleep(seconds int) {
	if a.userId == 0 {
		a.mobRecord.Sleep(seconds)
	}
}

func (a ScriptActor) Command(cmd string, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.Command(cmd, waitSeconds[0])
	} else {
		a.mobRecord.CommandScripted(events.Requester(), cmd, waitSeconds[0])
	}
}

func (a ScriptActor) CommandFlagged(cmd string, flags events.EventFlag, waitSeconds ...float64) {
	if len(waitSeconds) < 1 {
		waitSeconds = append(waitSeconds, 0)
	}
	if a.userId > 0 {
		a.userRecord.CommandFlagged(cmd, flags, waitSeconds[0])
	} else {
		a.mobRecord.CommandScripted(events.Requester(), cmd, waitSeconds[0])
	}
}

func (a ScriptActor) TrainSkill(skillName string, skillLevel int) bool {

	if a.userRecord == nil {
		return false
	}

	skillName = strings.ToLower(skillName)

	if !skills.SkillExists(skillName) {
		return false
	}

	currentLevel := a.characterRecord.GetSkillLevel(skillName)

	// Archetype-claimed skills need the right archetype (Phase 17).
	if currentLevel < skillLevel {
		if allowed, reason := archetypes.CanTrain(a.userId, skillName); !allowed {
			if reason != `` {
				a.userRecord.SendText(reason)
			}
			return false
		}
	}

	if currentLevel < skillLevel {
		newLevel := a.characterRecord.TrainSkill(skillName, skillLevel)

		skillData := struct {
			SkillName  string
			SkillLevel int
		}{
			SkillName:  skillName,
			SkillLevel: newLevel,
		}
		skillUpTxt, _ := templates.Process("character/skillup", skillData, a.userRecord.UserId)
		a.SendText(skillUpTxt)

		return true

	}
	return false
}

func (a ScriptActor) GetSkillLevel(skillName string) int {
	return a.characterRecord.GetSkillLevel(skillName)
}

func (a ScriptActor) MoveRoom(destRoomId int) {

	if a.userRecord != nil {

		originRoomId := a.characterRecord.RoomId
		rmNow := rooms.LoadRoom(originRoomId)

		if rmNext := rooms.LoadRoom(destRoomId); rmNext != nil {

			if err := rooms.MoveToRoom(a.userId, destRoomId); err != nil {
				return
			}
			// MoveToRoom can map a special room into the player's own copy.
			destRoomId = a.characterRecord.RoomId
			rmNext = rooms.LoadRoom(destRoomId)

			// Ashveil Phase 33h3: the company comes along (or is separated,
			// and told) through the one shared step; other charmed mobs
			// standing with the player follow as before.
			for _, mobInstId := range a.characterRecord.GetCharmIds() {
				if _, _, companion := company.LeaderAndKeyForInstance(mobInstId); companion {
					continue
				}
				if mob := mobs.GetInstance(mobInstId); mob == nil || rmNow == nil || rmNext == nil || mob.Character.RoomId != originRoomId {
					continue
				}
				rmNow.RemoveMob(mobInstId)
				rmNext.AddMob(mobInstId)
			}
			if company.RelocateCompany(a.userId, originRoomId, destRoomId) > 0 {
				a.userRecord.SendText(company.CompanyFollows)
			}

			if doLook, err := TryRoomScriptEvent(`onEnter`, a.userRecord.UserId, destRoomId); err != nil || doLook {
				a.userRecord.CommandFlagged(`look`, events.CmdSecretly) // Do a secret look.
			}
		}

	} else if a.mobRecord != nil {

		// Ashveil Phase 33h3: a companion moves only with its leader.
		if _, _, companion := company.LeaderAndKeyForInstance(a.mobInstanceId); companion {
			return
		}

		if mobRoom := rooms.LoadRoom(a.characterRecord.RoomId); mobRoom != nil {
			if destRoom := rooms.LoadRoom(destRoomId); destRoom != nil {
				mobRoom.RemoveMob(a.mobInstanceId)
				destRoom.AddMob(a.mobInstanceId)
			}
		}

	}
}

func (a ScriptActor) UpdateItem(itm ScriptItem) {
	if !a.prepareUserAssets() {
		return
	}
	a.userRecord.Character.UpdateItem(itm.originalItem, *itm.itemRecord)
}

func (a ScriptActor) AddEventLog(category string, message string) {
	if a.userRecord != nil {
		a.userRecord.EventLog.Add(category, message)
	}
}

// MarkVisitedRoom marks one or more rooms as visited for this actor.
// Only applies to user actors; mobs do not track room visits.
// Each roomId argument is recorded under the zone that room belongs to.
func (a ScriptActor) MarkVisitedRoom(roomIds ...int) {
	if a.userRecord == nil {
		return
	}
	for _, roomId := range roomIds {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		zCfg := rooms.GetZoneConfig(room.Zone)
		var validRoomIds map[int]struct{}
		if zCfg != nil {
			validRoomIds = zCfg.RoomIds
		}
		a.characterRecord.MarkVisitedRoom(roomId, room.Zone, validRoomIds)
	}
}

// MarkVisitedZone marks every room in the named zone as visited for this actor.
// Only applies to user actors; mobs do not track room visits.
// Uses FindZoneName for partial/best-match zone name resolution.
func (a ScriptActor) MarkVisitedZone(zoneName string) {
	if a.userRecord == nil {
		return
	}
	resolvedZone := rooms.FindZoneName(zoneName)
	if resolvedZone == `` {
		return
	}
	zCfg := rooms.GetZoneConfig(resolvedZone)
	if zCfg == nil {
		return
	}
	for roomId := range zCfg.RoomIds {
		a.characterRecord.MarkVisitedRoom(roomId, resolvedZone, nil)
	}
}

func (a ScriptActor) GiveItem(itm any) {
	if !a.prepareUserAssets() {
		return
	}

	var sItem *ScriptItem

	if itmScriptItem, ok := itm.(*ScriptItem); ok {
		sItem = itmScriptItem
	} else if itmInt, ok := itm.(int); ok {
		sItem = CreateItem(itmInt)
	} else if itmInt64, ok := itm.(int64); ok {
		sItem = CreateItem(int(itmInt64))
	} else if itmInt32, ok := itm.(int32); ok {
		sItem = CreateItem(int(itmInt32))
	} else if itmFloat64, ok := itm.(float64); ok {
		sItem = CreateItem(int(itmFloat64))
	}

	if sItem != nil {
		iRecord := sItem.itemRecord
		if a.characterRecord.StoreItem(*iRecord) {
			if a.userId > 0 {

				events.AddToQueue(events.ItemOwnership{
					UserId: a.userId,
					Item:   *iRecord,
					Gained: true,
				})

			}
		}
	}

}

func (a ScriptActor) TakeItem(itm ScriptItem) {
	if !a.prepareUserAssets() {
		return
	}
	if a.characterRecord.RemoveItem(*itm.itemRecord) {
		if a.userId > 0 {

			events.AddToQueue(events.ItemOwnership{
				UserId: a.userId,
				Item:   *itm.itemRecord,
				Gained: false,
			})

		}
	}
}

func (a ScriptActor) HasBuff(buffId int) bool {
	return a.characterRecord.HasBuff(buffId)
}

func (a ScriptActor) GiveBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		UserId:        a.userId,
		MobInstanceId: a.mobInstanceId,
		BuffId:        buffId,
		Source:        source,
	})

}

func (a ScriptActor) GetStatMod(statModName string) int {
	return a.characterRecord.StatMod(statModName)
}

func (a ScriptActor) HasBuffFlag(buffFlag string) bool {
	return a.characterRecord.HasBuffFlag(buffFlag)
}

func (a ScriptActor) CancelBuffWithFlag(buffFlag string) bool {

	found := false

	for _, buffId := range a.characterRecord.Buffs.GetBuffIdsWithFlag(strings.ToLower(buffFlag)) {
		found = found || a.RemoveBuff(buffId)
	}

	return found
}

// Remove a buff silently
func (a ScriptActor) RemoveBuff(buffId int) bool {

	if !configs.GetGamePlayConfig().AllowItemBuffRemoval {
		buffList := a.characterRecord.GetBuffs(buffId)
		if len(buffList) > 0 {
			if buffList[0].PermaBuff {
				return false
			}
		}
	}

	return a.characterRecord.Buffs.RemoveBuff(buffId)

}

func (a ScriptActor) HasItemId(itemId int, excludeWorn ...bool) bool {
	for _, itm := range a.characterRecord.GetAllBackpackItems() {
		if itm.ItemId == itemId {
			return true
		}
	}
	if len(excludeWorn) == 0 || !excludeWorn[0] {
		for _, itm := range a.characterRecord.GetAllWornItems() {
			if itm.ItemId == itemId {
				return true
			}
		}
	}
	return false
}

func (a ScriptActor) GetBackpackItems() []ScriptItem {
	if !a.prepareUserAssets() {
		return nil
	}
	itms := make([]ScriptItem, 0, 5)
	for _, item := range a.characterRecord.GetAllBackpackItems() {
		itms = append(itms, newScriptItem(item))
	}
	return itms
}

func (a ScriptActor) GetAlignment() int {
	return int(a.characterRecord.Alignment)
}

func (a ScriptActor) GetAlignmentName() string {
	return a.characterRecord.AlignmentName()
}

func (a ScriptActor) ChangeAlignment(alignmentChange int) {
	a.characterRecord.UpdateAlignment(alignmentChange)
}

func (a ScriptActor) HasSpell(spellId string) bool {
	return a.characterRecord.HasSpell(spellId)
}

// LearnSpell teaches a spell. For players it first consults the archetype
// gate (Phase 17): a spell from a school their archetype doesn't claim is
// refused with an explanation, and false is returned. Mobs are not gated.
func (a ScriptActor) LearnSpell(spellId string) bool {
	if a.userId > 0 && !a.characterRecord.HasSpell(spellId) {
		if allowed, reason := archetypes.CanLearnSpell(a.userId, spellId); !allowed {
			if a.userRecord != nil && reason != `` {
				a.userRecord.SendText(reason)
			}
			return false
		}
	}
	return a.characterRecord.LearnSpell(spellId)
}

func (a ScriptActor) IsAggro(actor ScriptActor) bool {
	return a.characterRecord.IsAggro(actor.UserId(), actor.InstanceId())
}

func (a ScriptActor) GetMobKills(mobId int) int {
	return a.characterRecord.KD.GetMobKills(mobId)
}

func (a ScriptActor) GetRaceKills(race string) int {

	raceKills := map[string]int{}

	for mid, kCt := range a.characterRecord.KD.Kills {
		if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {
			if raceInfo := races.GetRace(mobSpec.Character.RaceId); raceInfo != nil {
				raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
			}
		}
	}

	return raceKills[race]
}

func (a ScriptActor) SetHealth(amt int) {
	c := a.characterRecord
	if c.CombatWithdrawn && amt < c.Health {
		return
	}
	if amt > c.Health {
		// Phase 30b: raising health stops at the wound limit.
		amt = c.CapHealing(c.Health, amt)
	}
	c.Health = amt
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}
}

// GetHealthLimit is how far healing can restore the actor: its max health
// less what its wounds hold back (Phase 30b).
func (a ScriptActor) GetHealthLimit() int {
	return a.characterRecord.HealthLimit()
}

// SpellFactor is what a damage spell's roll is multiplied by against the
// target (Phase 35a2): 1 + 0.5 × the caster's skill edge (its Attack
// against the target's Evasion), from 0.5 to 1.5.
func (a ScriptActor) SpellFactor(target ScriptActor) float64 {
	if a.characterRecord == nil || target.characterRecord == nil {
		return 1
	}
	factor := 1 + 0.5*characters.SkillEdge(a.characterRecord.AttackSkill(), target.characterRecord.Evasion())
	factor *= 1 + float64(a.characterRecord.ClassEffects().Int(classes.SpellPct))/100
	// Phase 38c3: an Archmage's Overchannel, and an Archon's Mana Shield on
	// the one the spell strikes.
	if agg := a.characterRecord.Aggro; agg != nil && agg.Type == characters.SpellCast && agg.SpellInfo.Over > 0 {
		factor *= 1 + float64(agg.SpellInfo.Over)/100
	}
	if r := target.characterRecord.Aura.SpellResolve; r > 0 {
		factor *= 1 - float64(min(r, 100))/100
	}
	// Phase 39c: a fogbound caster's spells hit weaker.
	if len(a.characterRecord.GetBuffs(status.Fogbound)) > 0 {
		factor *= float64(100-stormcraft.FogSpellPct) / 100
	}
	return factor
}

// SpellPower is one roll of a spell's size for this caster (Phase 35b),
// from the spell's `power` block (internal/spellpower), before skill or
// gear: base + dice + level and Mysticism bonuses, times its share. It is
// 0 for a spell without one. Scripts multiply it by SpellFactor or
// HealFactor and round down.
func (a ScriptActor) SpellPower(spellId string) float64 {
	sp := spells.GetSpell(spellId)
	if a.characterRecord == nil || sp == nil || sp.Power == nil {
		return 0
	}
	return sp.Power.Raw(a.characterRecord.Level, a.characterRecord.Stats.Mysticism.ValueAdj, util.Rand)
}

// HealFactor is what the actor's heals are multiplied by (Phase 35a2): 1
// plus its gear's healing percent (a holy symbol's 5).
func (a ScriptActor) HealFactor() float64 {
	if a.characterRecord == nil {
		return 1
	}
	return 1 + float64(a.characterRecord.HealingBonusPct())/100
}

// WoundNote is the text a heal adds when the actor's wound limit held some
// of it back (Phase 30b): ", wound limit 10 of 16", else "". rolled is what
// the heal tried; healed what AddHealth returned.
func (a ScriptActor) WoundNote(rolled, healed int) string {
	c := a.characterRecord
	limit := c.HealthLimit()
	if rolled <= healed || limit >= c.HealthMax.Value || c.Health < limit {
		return ``
	}
	return fmt.Sprintf(`, wound limit %d of %d`, limit, c.HealthMax.Value)
}

// HasLastingWound reports whether the actor has a wound tend can close
// (Phase 30b): a lasting one.
func (a ScriptActor) HasLastingWound() bool {
	return len(wounds.Lasting(a.characterRecord.Wounds)) > 0
}

// TendWound closes up to points of the actor's worst lasting wound (Phase
// 30b, the tend spell). It returns {closed, wound, limit, max}: the points
// closed (0 when there was no wound to tend), the wound as it was
// described ("a broken arm"), and the limit and max health after.
func (a ScriptActor) TendWound(points int) map[string]any {
	c := a.characterRecord
	ws, was, closed, ok := wounds.Close(c.Wounds, points)
	out := map[string]any{"closed": 0, "wound": ``, "limit": c.HealthLimit(), "max": c.HealthMax.Value}
	if !ok {
		return out
	}
	c.Wounds = ws
	out["closed"] = closed
	out["wound"] = wounds.Describe(was)
	out["limit"] = c.HealthLimit()
	if a.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.userId})
	}
	return out
}

func (a ScriptActor) GetHealth() int {
	return a.characterRecord.Health
}

func (a ScriptActor) GetHealthMax() int {
	return a.characterRecord.HealthMax.Value
}

func (a ScriptActor) GetHealthPct() float64 {
	return float64(a.characterRecord.Health) / float64(a.characterRecord.HealthMax.Value)
}

func (a ScriptActor) GetMana() int {
	return a.characterRecord.Mana
}

func (a ScriptActor) GetManaMax() int {
	return a.characterRecord.ManaMax.Value
}

func (a ScriptActor) GetManaPct() float64 {
	return float64(a.characterRecord.Mana) / float64(a.characterRecord.ManaMax.Value)
}

func (a ScriptActor) SetAdjective(adj string, addIt bool) {
	a.characterRecord.SetAdjective(adj, addIt)
}

func (a ScriptActor) IsHome() bool {
	if a.mobRecord != nil {
		return a.mobRecord.HomeRoomId == a.characterRecord.RoomId
	}
	return false
}

func (a ScriptActor) GetTrainingPoints() int {
	return a.characterRecord.TrainingPoints
}

func (a ScriptActor) GiveTrainingPoints(ct int) {
	if ct < 1 {
		return
	}
	a.characterRecord.TrainingPoints += ct
}

func (a ScriptActor) GetStatPoints() int {
	return a.characterRecord.StatPoints
}

func (a ScriptActor) GiveStatPoints(ct int) {
	if ct < 1 {
		return
	}
	a.characterRecord.StatPoints += ct
}

func (a ScriptActor) GiveExtraLife() {
	c := configs.GetGamePlayConfig()
	a.characterRecord.ExtraLives += 1
	if a.characterRecord.ExtraLives > int(c.LivesMax) {
		a.characterRecord.ExtraLives = int(c.LivesMax)
	}
}

func (a ScriptActor) Uncurse() []*ScriptItem {

	retList := []*ScriptItem{}

	for _, itm := range a.characterRecord.Uncurse() {
		retList = append(retList, GetItem(itm))
	}

	return retList
}

func (a ScriptActor) GetPet() *ScriptPet {
	if !a.characterRecord.Pet.Exists() {
		return nil
	}
	return GetPet(&a.characterRecord.Pet, a.userId)
}

func (a ScriptActor) GrantXP(xpAmt int, reason string) {
	if a.mobInstanceId > 0 {
		return
	}
	a.userRecord.GrantXP(xpAmt, reason)
}

func (a ScriptActor) TimerSet(name string, period string) {
	a.characterRecord.TimerSet(name, period)
}

func (a ScriptActor) TimerExpired(name string) bool {
	return a.characterRecord.TimerExpired(name)
}

func (a ScriptActor) TimerExists(name string) bool {
	return a.characterRecord.TimerExists(name)
}

// ////////////////////////////////////////////////////////
//
// Functions only really useful for mobs
//
// ////////////////////////////////////////////////////////

// Returns true if a mob is charmed by/friendly to a player.
// If userId is ommitted, it will return true if the mob is charmed by any player.
func (a ScriptActor) IsCharmed(userId ...int) bool {
	if len(userId) < 1 {
		return a.characterRecord.IsCharmed()
	}
	return a.characterRecord.IsCharmed(userId[0])
}

func (a ScriptActor) GetCharmedUserId() int {
	return a.characterRecord.GetCharmedUserId()
}

func (a ScriptActor) getScript() string {
	if a.mobRecord != nil {
		return a.mobRecord.GetScript()
	}
	return ""
}

func (a ScriptActor) getScriptTag() string {
	if a.mobRecord != nil {
		return a.mobRecord.ScriptTag
	}
	return ""
}

func (a ScriptActor) ShorthandId() string {

	if a.userRecord != nil {

		return a.userRecord.ShorthandId()

	} else if a.mobRecord != nil {

		return a.mobRecord.ShorthandId()

	}

	return ``
}

func (a ScriptActor) GetLastInputRound() uint64 {
	if a.userRecord != nil {
		return a.userRecord.GetLastInputRound()
	}
	return 0
}

// PlaySound sends a sound event to this actor. Only affects user actors.
// soundId is the identifier defined in audio.yaml (e.g. "levelup").
// category groups related sounds for client-side volume control (e.g. "combat", "other").
func (a ScriptActor) PlaySound(soundId string, category string) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.PlaySound(soundId, category)
}

// PlayMusic sends a music-change event to this actor. Only affects user actors.
// musicFileOrId may be a filepath (e.g. "static/audio/music/intro.mp3") or a sound
// identifier from audio.yaml whose filepath resolves to a music file.
// Pass "Off" to stop music.
func (a ScriptActor) PlayMusic(musicFileOrId string) {
	if a.userRecord == nil {
		return
	}
	a.userRecord.PlayMusic(musicFileOrId)
}

func (a ScriptActor) Pathing() bool {
	if a.mobRecord == nil {
		return false
	}
	return a.mobRecord.Path.Current() != nil || a.mobRecord.Path.Len() > 0
}

func (a ScriptActor) PathingAtWaypoint() bool {
	if a.mobRecord == nil {
		return false
	}
	pathStep := a.mobRecord.Path.Current()
	return pathStep != nil && pathStep.Waypoint()
}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////
func GetActor(userId int, mobInstanceId int) *ScriptActor {

	if userId > 0 {
		if user := users.GetByUserId(userId); user != nil {
			return &ScriptActor{
				userId:          userId,
				userRecord:      user,
				characterRecord: user.Character,
			}
		}
	} else if mobInstanceId > 0 {
		if mob := mobs.GetInstance(mobInstanceId); mob != nil {
			return &ScriptActor{
				mobInstanceId:   mobInstanceId,
				mobRecord:       mob,
				characterRecord: &mob.Character,
			}
		}
	}

	return nil
}

func GetUser(userId int) *ScriptActor {
	return GetActor(userId, 0)
}

func GetMob(mobInstanceId int) *ScriptActor {
	return GetActor(0, mobInstanceId)
}

func ActorNames(actorList []*ScriptActor) string {

	sBuilder := strings.Builder{}
	listSize := len(actorList)

	for i := 0; i < listSize; i++ {

		sBuilder.WriteString(actorList[i].GetCharacterName(true))

		if i < listSize-2 {
			sBuilder.WriteString(`, `)
		} else if i == listSize-2 {
			sBuilder.WriteString(`and `)
		}
	}

	return sBuilder.String()
}

// ////////////////////////////////////////////////////////
//
// New ScriptActor methods
//
// ////////////////////////////////////////////////////////

func (a ScriptActor) GetDescription() string {
	return a.characterRecord.GetDescription()
}

func (a ScriptActor) GetGold() int {
	if !a.prepareUserAssets() {
		return 0
	}
	return a.characterRecord.Gold
}

func (a ScriptActor) GetBank() int {
	return a.characterRecord.Bank
}

func (a ScriptActor) GetWornItems() []ScriptItem {
	itms := make([]ScriptItem, 0)
	for _, itm := range a.characterRecord.GetAllWornItems() {
		itms = append(itms, newScriptItem(itm))
	}
	return itms
}

func (a ScriptActor) GetWornItem(slot string) *ScriptItem {
	itm := a.characterRecord.Equipment.Get(items.ItemType(slot))
	if itm == nil || itm.ItemId <= 0 || itm.IsDisabled() {
		return nil
	}
	si := newScriptItem(*itm)
	return &si
}

func (a ScriptActor) FindInBackpack(itemName string) *ScriptItem {
	if !a.prepareUserAssets() {
		return nil
	}
	itm, found := a.characterRecord.FindInBackpack(itemName)
	if !found {
		return nil
	}
	si := newScriptItem(itm)
	return &si
}

// Scripted rewards/transfers can target a player other than the command issuer.
func (a ScriptActor) prepareUserAssets() bool {
	if a.userId > 0 {
		if err := company.PrepareAssets(a.userId); err != nil {
			if a.userRecord != nil {
				a.userRecord.SendText("Company assets await recovery; the asset action was refused.")
			}
			return false
		}
	}
	return true
}

func (a ScriptActor) FindOnBody(itemName string) *ScriptItem {
	itm, found := a.characterRecord.FindOnBody(itemName)
	if !found {
		return nil
	}
	si := newScriptItem(itm)
	return &si
}

func (a ScriptActor) IsQuestDone(questToken string) bool {
	return a.characterRecord.IsQuestDone(questToken)
}

func (a ScriptActor) ClearQuestToken(questToken string) {
	a.characterRecord.ClearQuestToken(questToken)
}

func (a ScriptActor) GetAllSkills() map[string]int {
	return a.characterRecord.GetAllSkillRanks()
}

func (a ScriptActor) GetSpells() map[string]int {
	return a.characterRecord.GetSpells()
}

func (a ScriptActor) UnLearnSpell(spellId string) bool {
	return a.characterRecord.UnLearnSpell(spellId)
}

func (a ScriptActor) DisableSpell(spellId string) bool {
	return a.characterRecord.DisableSpell(spellId)
}

func (a ScriptActor) EnableSpell(spellId string) bool {
	return a.characterRecord.EnableSpell(spellId)
}

func (a ScriptActor) GetCooldown(tag string) int {
	return a.characterRecord.GetCooldown(tag)
}

func (a ScriptActor) TryCooldown(tag string, period string) bool {
	return a.characterRecord.TryCooldown(tag, period)
}

func (a ScriptActor) GetSetting(name string) string {
	return a.characterRecord.GetSetting(name)
}

func (a ScriptActor) SetSetting(name string, value string) {
	a.characterRecord.SetSetting(name, value)
}

func (a ScriptActor) GetExperience() int {
	return a.characterRecord.Experience
}

func (a ScriptActor) GetExtraLives() int {
	return a.characterRecord.ExtraLives
}

func (a ScriptActor) IsInCombat() bool {
	return a.characterRecord.Aggro != nil
}

// InBattle reports whether the actor is fighting: in a battle of their
// own company's, or with aggro (Ashveil Phase 33h3). Scripts that move a
// player at the player's own request refuse while it is true.
func (a ScriptActor) InBattle() bool {
	if a.userRecord != nil {
		if _, ok := battle.Current(a.userId); ok {
			return true
		}
	}
	return a.characterRecord.Aggro != nil
}

func (a ScriptActor) IsDowned() bool {
	return a.characterRecord.Health < 1
}

func (a ScriptActor) GetDefense() int {
	return a.characterRecord.GetDefense()
}

func (a ScriptActor) GetGearValue() int {
	return a.characterRecord.GetGearValue()
}

func (a ScriptActor) GetCarryCapacity() int {
	return a.characterRecord.CarryCapacity()
}

func (a ScriptActor) GetZoneVisitProgress(zoneName string) map[string]any {
	resolvedZone := rooms.FindZoneName(zoneName)
	if resolvedZone == `` {
		return map[string]any{`visited`: 0, `total`: 0, `percent`: 0}
	}
	zCfg := rooms.GetZoneConfig(resolvedZone)
	var validRoomIds map[int]struct{}
	if zCfg != nil {
		validRoomIds = zCfg.RoomIds
	}
	visited, total := a.characterRecord.ZoneVisitProgress(resolvedZone, validRoomIds)
	percent := 0
	if total > 0 {
		percent = int(float64(visited) / float64(total) * 100)
	}
	return map[string]any{`visited`: visited, `total`: total, `percent`: percent}
}

func (a ScriptActor) GetAdjectives() []string {
	return a.characterRecord.GetAdjectives()
}

func (a ScriptActor) HasAdjective(adj string) bool {
	return a.characterRecord.HasAdjective(adj)
}

func (a ScriptActor) GetActionPoints() int {
	return a.characterRecord.ActionPoints
}

func (a ScriptActor) GetActionPointsMax() int {
	return a.characterRecord.ActionPointsMax.Value
}

func (a ScriptActor) GetHealthAppearance() string {
	if a.characterRecord.HealthMax.Value < 1 {
		return a.GetCharacterName(true) + ` is in perfect health.`
	}
	nameTag := `username`
	if a.mobRecord != nil {
		nameTag = `mobname`
	}
	pct := int(float64(a.characterRecord.Health) / float64(a.characterRecord.HealthMax.Value) * 100)
	className := util.HealthClass(a.characterRecord.Health, a.characterRecord.HealthMax.Value)
	name := `<ansi fg="` + nameTag + `">` + a.characterRecord.Name + `</ansi>`
	switch {
	case pct < 15:
		return name + ` looks like they're <ansi fg="` + className + `">about to die!</ansi>`
	case pct < 50:
		return name + ` looks to be in <ansi fg="` + className + `">pretty bad shape.</ansi>`
	case pct < 80:
		return name + ` has some <ansi fg="` + className + `">cuts and bruises.</ansi>`
	case pct < 100:
		return name + ` has <ansi fg="` + className + `">a few scratches.</ansi>`
	}
	return name + ` is in <ansi fg="` + className + `">perfect health.</ansi>`
}

// wakeLines tells the holder and the room that a sleeper woke.
func (a ScriptActor) wakeLines() {
	spec := status.Get(status.Asleep)
	if a.userRecord != nil {
		a.userRecord.SendText(spec.EndYou)
	}
	if room := rooms.LoadRoom(a.characterRecord.RoomId); room != nil {
		name := a.GetCombatName(true)
		if a.userId > 0 {
			room.SendText(fmt.Sprintf(spec.EndOther, name), a.userId)
		} else {
			room.SendText(fmt.Sprintf(spec.EndOther, name))
		}
	}
}

// HexTargets is the foes a hex reaches (Phase 38a): the first of targets,
// as many as the caster's level allows (hexes.Reach).
func (a ScriptActor) HexTargets(targets []*ScriptActor) []*ScriptActor {
	reach := hexes.Reach(a.characterRecord.Level) + a.characterRecord.ClassEffects().Int(classes.HexReach)
	if reach >= len(targets) {
		return targets
	}
	return targets[:reach]
}

// hexRoll rolls a hex's resist (0..n-1); tests replace it.
var hexRoll = util.Rand

// UseHexRollForTest replaces the hex resist roll and returns its restore.
func UseHexRollForTest(roll func(int) int) (restore func()) {
	prev := hexRoll
	hexRoll = roll
	return func() { hexRoll = prev }
}

// holderKey names an actor for the hex immunity ledger.
func (a ScriptActor) holderKey() string {
	if a.userId > 0 {
		return fmt.Sprintf(`u%d`, a.userId)
	}
	return fmt.Sprintf(`m%d`, a.mobInstanceId)
}

// CastHex lands a hex on target (Phase 38a), and returns {landed, reason,
// rounds}: reason is "landed", "already" (it carries the status),
// "immune" (a hex of this status landed on it lately), "resisted" or
// "invalid". The caster's Mysticism is set against the target's Mysticism
// (Vitality for Binding Hex): 65 in 100 at even stats, held to 25..90, and
// a boss resists 25 more and holds a status half as long. The status is
// queued as an ordinary buff, so it lands only on someone still in a
// fight. rounds is the status's length in combat rounds (0 for a hex that
// shakes morale only).
func (a ScriptActor) CastHex(spellId string, target ScriptActor) map[string]any {
	out := map[string]any{`landed`: false, `reason`: `invalid`, `rounds`: 0}
	h, ok := hexes.For(spellId)
	if !ok || a.characterRecord == nil || target.characterRecord == nil {
		return out
	}
	tc := target.characterRecord
	if tc.Health < 1 {
		return out // fell while the hex was chanted
	}
	buff := h.Buff
	if h.Morale {
		buff = -1
	}
	if h.Buff > 0 && tc.HasBuff(h.Buff) {
		out[`reason`] = `already`
		return out
	}
	holder := target.holderKey()
	if hexes.Default.Immune(holder, buff) {
		out[`reason`] = `immune`
		return out
	}
	boss := target.mobRecord != nil && target.mobRecord.Boss
	mine, theirs := a.characterRecord.Stats.Mysticism.ValueAdj, tc.Stats.Mysticism.ValueAdj
	if h.Resist == hexes.Vitality {
		theirs = tc.Stats.Vitality.ValueAdj
	}
	fx := a.characterRecord.ClassEffects()
	resist := 0
	if boss {
		resist = hexes.BossResist
		if fx.Has(classes.BossHalf) {
			resist /= 2 // Phase 38c3: Breaking the boss
		}
	}
	if hexRoll(100) >= hexes.LandChanceWith(combat.StatEdge(mine, theirs), resist, fx.Int(classes.HexLand)) {
		// Phase 38c3: Coven Circle lets the first resisted hex of a battle land.
		if rt := a.characterRecord.RTState(); fx.Has(classes.Circle) && !rt.CircleUsed {
			rt.CircleUsed = true
			out[`circle`] = true
			line := `    The coven's circle closes round the hex, and it lands anyway. (coven circle)`
			if a.userId > 0 {
				SendUserMessage(a.userId, line)
			}
			SendRoomMessage(a.characterRecord.RoomId, line, a.userId)
		} else {
			out[`reason`] = `resisted`
			return out
		}
	}
	out[`landed`], out[`reason`] = true, `landed`
	a.hexWard()
	a.curseFoe(target, h)
	if h.Morale {
		leader := a.userId
		if a.mobInstanceId > 0 {
			leader, _, _ = company.LeaderAndKeyForInstance(a.mobInstanceId)
		}
		events.AddToQueue(events.MoraleCheck{LeaderUserId: leader, MobInstanceId: target.mobInstanceId})
		hexes.Default.Land(holder, buff, 0)
		return out
	}
	spec := buffs.GetBuffSpec(h.Buff)
	if spec == nil {
		out[`landed`], out[`reason`] = false, `invalid`
		return out
	}
	// A status counts one trigger more than it lasts; poison, which counts
	// game-round triggers, is its trigger count.
	own, extra := spec.TriggerCount, 0
	if spec.CombatRounds {
		own, extra = spec.TriggerCount-1, 1
	}
	rounds := h.RoundsAt(a.characterRecord.Level, boss, own)
	if h.Buff == buffAsleepID && rounds > 0 && !boss {
		rounds += fx.Int(classes.SlumberLong)
	}
	longer := 0
	if !boss {
		longer = fx.Int(classes.HexLong) // Phase 38c3: Lasting hexes
	}
	evt := events.Buff{UserId: target.userId, MobInstanceId: target.mobInstanceId, BuffId: h.Buff, Source: `spell`}
	if rounds > 0 {
		rounds += longer
		evt.Triggers = rounds + extra
	} else {
		rounds = own
		if longer > 0 {
			rounds += longer
			evt.Triggers = rounds + extra
		}
	}
	a.twinHex(target)
	a.lingerCurse(target, rounds)
	events.AddToQueue(evt)
	hexes.Default.Land(holder, buff, rounds)
	out[`rounds`] = rounds
	return out
}

// buffAsleepID is Slumber's status, for the Witch's longer sleeps.
const buffAsleepID = status.Asleep

// hexWard is a Witch's Warding hex: a landed hex shields the most hurt ally
// from the next blows, up to an average hit of her level. A Wise One's
// Hearthward (Phase 38c3) wards the three, then four, most hurt allies
// without a ward, each up to 2 average hits.
func (a ScriptActor) hexWard() {
	fx := a.characterRecord.ClassEffects()
	n := fx.Int(classes.HexWard)
	if n < 1 {
		return
	}
	pct := max(100, fx.Int(classes.HexWardCap))
	cap := max(1, int(a.SpellPower("ward"))*pct/100)
	granted := 0
	for _, ally := range a.HurtAllies(false) {
		if granted >= n {
			break
		}
		if ally.GrantWard(cap, max(1, fx.Int(classes.WardBlows))) {
			a.WardGifts(ally)
			granted++
		}
	}
}

// curseFoe marks a foe a hex landed on for the Crone's riders (Phase
// 38c3): allies' Attack against it, the poison that doubles, and the fall
// that frightens its group; it also remembers whose hex it carries, and
// that the hexed streak of Crone's Doom starts here.
func (a ScriptActor) curseFoe(target ScriptActor, h hexes.Hex) {
	fx := a.characterRecord.ClassEffects()
	if target.characterRecord == nil || target.mobRecord == nil {
		return
	}
	rt := target.characterRecord.RTState()
	if h.Buff > 0 && !h.Morale {
		if rt.HexBuffs == nil {
			rt.HexBuffs = map[int]bool{}
		}
		rt.HexBuffs[h.Buff] = true
	}
	// Crone's Doom follows the Crone: another witch's hex doesn't take it.
	if rt.CurseBy == nil || fx.Has(classes.Doom) || !rt.CurseBy.ClassEffects().Has(classes.Doom) {
		rt.CurseBy = a.characterRecord
	}
	if n := fx.Int(classes.CurseAtk); n > 0 {
		rt.CurseAtk = n
	}
	if n := fx.Int(classes.CurseDmg); n > 0 {
		rt.CurseDmg = n // Ashen Curse: every ally's blows bite deeper
	}
	if fx.Has(classes.SoulRot) {
		rt.SoulRot = true
	}
	if fx.Has(classes.PoisonX2) && h.Spell == `miasma` {
		rt.PoisonX2 = true
	}
}

// twinHex counts a landed hex, and on every third leaves its target
// exposed for a round (Phase 38c3).
func (a ScriptActor) twinHex(target ScriptActor) {
	fx := a.characterRecord.ClassEffects()
	rt := a.characterRecord.RTState()
	rt.HexLands++
	if fx.Has(classes.TwinHex) && rt.HexLands%3 == 0 {
		target.GiveStatus(status.Exposed, 1)
	}
}

// lingerCurse notes the round a hex ends, so the foe is left exposed a
// round after it (Lingering Curse, Phase 38c3).
func (a ScriptActor) lingerCurse(target ScriptActor, rounds int) {
	if a.characterRecord.ClassEffects().Has(classes.Linger) && target.characterRecord != nil && rounds > 0 {
		target.characterRecord.RTState().LingerAt = uint64(hexes.Default.Round() + rounds + 1)
	}
}
