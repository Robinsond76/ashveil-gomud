// Phase 35b: a greater mana draught restores 60% of the drinker's mana
// pool at once.
DRAUGHT_PCT = 60;

/**
 * Called when the buff is first applied to the actor.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onStart(actor, triggersLeft) {
    var restored = actor.AddMana(Math.floor(actor.GetManaMax() * DRAUGHT_PCT / 100));
    SendUserMessage(actor.UserId(), 'The draught settles cold in your chest, and your mind clears. (' + restored + ' mana)');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' drinks down a blue draught.', actor.UserId());
}
