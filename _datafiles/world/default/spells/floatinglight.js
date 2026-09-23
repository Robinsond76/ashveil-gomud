/**
 * Called when the casting is initialized.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onCast(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You cup your hands and begin to chant.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' cups their hands and begins to chant.', sourceActor.UserId());
    return true;
}

/**
 * Called each round while the spell is being cast.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onWait(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'A spark kindles between your palms...');
    SendRoomMessage(sourceActor.GetRoomId(), 'A spark kindles between '+sourceActor.GetCharacterName(true)+'\'s palms...', sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt. The light always floats
 * above the caster and lights the way for the caster's allies.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You release a floating light, and it rises to hover above you.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' releases a floating light that rises to hover above them.', sourceActor.UserId());

    sourceActor.GiveBuff(1000, "spell");
}
