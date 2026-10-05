// Phase 35a2: 8 + 2d4 + level/8 + Mysticism/12, about 1.2x Magic
// Missile (35b's ratio).
HARM_BASE = 8;
HARM_DICE_QTY = 2;
HARM_DICE_SIDES = 4;
HARM_LEVEL_DIV = 8;
HARM_MYSTICISM_DIV = 12;

// Phase 35a2: a spell lands about one weapon hit, and skill decides how
// well: the roll is multiplied by the caster's SpellFactor against the
// target (0.5 to 1.5, its Attack against the target's Evasion).
function harmRoll(sourceActor, targetActor) {
    var roll = HARM_BASE + UtilDiceRoll(HARM_DICE_QTY, HARM_DICE_SIDES) +
        Math.floor(sourceActor.GetLevel() / HARM_LEVEL_DIV) +
        Math.floor(sourceActor.GetStat('mysticism') / HARM_MYSTICISM_DIV);
    return Math.max(1, Math.floor(roll * sourceActor.SpellFactor(targetActor)));
}

SPELL_NAME = 'Withering Hex';
WAIT_ROUNDS = 1; // hex.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
// Phase 30d1: a blow that draws blood breaks the chant (help interrupts).
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
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
