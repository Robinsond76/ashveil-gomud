// Phase 35b: the spell's size is its power block (hex.yaml, read through
// SpellPower); skill decides how well it lands: the roll is multiplied by
// the caster's SpellFactor against the target (0.5 to 1.5, its Attack
// against the target's Evasion).
function harmRoll(sourceActor, targetActor) {
    return Math.max(1, Math.floor(sourceActor.SpellPower('hex') * sourceActor.SpellFactor(targetActor)));
}

SPELL_NAME = 'Withering Hex';
WAIT_ROUNDS = 1; // hex.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
// Phase 30d1: a blow that draws blood breaks the chant (help interrupts).
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

/**
 * Called when the casting is initialized.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onCast(sourceActor, targetActor) {

    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You fix your eyes on ' + targetActor.GetCombatName(false) + ' and begin to croak a hex.' + rounds);
    SendUserMessage(targetActor.UserId(), sourceActor.GetCombatName(true) + ' fixes ' + sourceActor.GetCombatPronoun('possessive') + ' eyes on you and begins to croak a hex.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' fixes ' + sourceActor.GetCombatPronoun('possessive') + ' eyes on ' + targetActor.GetCombatName(false) + ' and begins to croak a hex.' + rounds, sourceActor.UserId(), targetActor.UserId());
    return true;
}

/**
 * Called each round while the spell is being cast.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onWait(sourceActor, targetActor) {

    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep croaking. The words sour in your mouth.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps croaking the hex. The words sour the air.' + rounds, sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActor) {

    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var targetUserId = targetActor.UserId();

    // Apply the harm first, and report what it took.
    var dealt = -targetActor.AddHealth(-harmRoll(sourceActor, targetActor));
    var suffix = ' (' + dealt + ' damage)';

    SendUserMessage(sourceUserId, 'You spit the last word, and ' + targetActor.GetCombatName(false) + ' withers under it.' + suffix);
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' spits the last word of the hex, and ' + targetActor.GetCombatName(false) + ' withers under it.' + suffix, sourceUserId, targetUserId);
    SendUserMessage(targetUserId, sourceActor.GetCombatName(true) + ' spits the last word of the hex, and something in you withers.' + suffix);
}
