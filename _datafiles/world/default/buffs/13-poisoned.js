
/**
 * Called when the buff is first applied to the actor.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'You begin to feel sick.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is looking sickly.', actor.UserId());
}

/**
 * Called each round while the buff is active.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onTrigger(actor, triggersLeft) {
    // Phase 38c3: a Crone of Ash's Rotting Miasma doubles the poison it leaves.
    dmgAmt = Math.abs(Math.abs(actor.AddHealth(UtilDiceRoll(1, 8)*actor.PoisonScale()*-1)));

    SendUserMessage(actor.UserId(),     'The poison hurts you for <ansi fg="damage">'+String(dmgAmt)+' damage</ansi>!');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' convulses under the effects of a poison.', actor.UserId());
}

/**
 * Called when the buff expires or is removed.
 * @param {ActorObject} actor - The actor the buff is applied to.
 * @param {number} triggersLeft - How many trigger rounds remain.
 * @returns {void}
 */
function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(),     'The poison wears off.');
    SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' looks a bit more normal.', actor.UserId());
}
