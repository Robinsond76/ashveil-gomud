
/**
 * Called when the buff is first applied to the actor.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You lay your head down and immediately doze off.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is getting some rest.', actor.UserId());
    actor.SetAdjective("sleeping", true);
}

/**
 * Called each round while the buff is active.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onTrigger(actor, triggersLeft) {
    healAmt = actor.AddHealth(UtilDiceRoll(3, 8));

    SendUserMessage(actor.UserId(),     'ZZzzz...');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' snores loudly.', actor.UserId());
}

/**
 * Called when the buff expires or is removed.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You wake up!');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' wakes up.', actor.UserId());
    actor.SetAdjective("sleeping", false);
    actor.GiveBuff(16, "sleep"); // Refreshed
}
