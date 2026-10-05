
// Phase 35a2: one average weapon hit, 8 + 2d4 + level/6 (about 13 / 15 /
// 18 at levels 1 / 10 / 30), times the healer's HealFactor (a holy
// symbol's +5%).
HEAL_BASE = 8;
HEAL_DICE_QTY = 2;
HEAL_DICE_SIDES = 4;
HEAL_LEVEL_DIV = 6;

// healRoll is one patient's heal before any wound limit.
function healRoll(sourceActor) {
    var roll = HEAL_BASE + UtilDiceRoll(HEAL_DICE_QTY, HEAL_DICE_SIDES) + Math.floor(sourceActor.GetLevel() / HEAL_LEVEL_DIV);
    return Math.floor(roll * sourceActor.HealFactor());
}

SPELL_NAME = 'Minor Heal';
WAIT_ROUNDS = 2; // heal.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
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
    SendUserMessage(sourceActor.UserId(), 'You begin a low prayer, and warmth gathers in your palms.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' begins a low prayer.' + rounds, sourceActor.UserId());
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
    SendUserMessage(sourceActor.UserId(), 'You keep praying. Your hands begin to glow.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps praying, and a soft glow grows in ' + sourceActor.GetCombatPronoun('possessive') + ' hands.' + rounds, sourceActor.UserId());
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

    // Apply the heal first, and report what it mended.
    // Phase 30b: a wound limit holds some of it back, and says so.
    var rolled = healRoll(sourceActor);
    var healed = targetActor.AddHealth(rolled);
    var suffix = ' (' + healed + ' healed' + targetActor.WoundNote(rolled, healed) + ')';

    if (sourceUserId != 0 && sourceUserId == targetUserId) {
        SendUserMessage(sourceUserId, 'You press your glowing hands to your own wounds, and they close a little.' + suffix);
        SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' presses glowing hands to ' + sourceActor.GetCombatPronoun('possessive') + ' own wounds.' + suffix, sourceUserId);
        return;
    }

    // Phase 32d: a companion healer tending its own wounds.
    if (sourceUserId == 0 && targetUserId == 0 && sourceActor.InstanceId() == targetActor.InstanceId()) {
        SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' presses glowing hands to ' + sourceActor.GetCombatPronoun('possessive') + ' own wounds.' + suffix);
        return;
    }

    SendUserMessage(sourceUserId, 'You lay your glowing hands on ' + targetActor.GetCombatName(false) + ', and the wounds close a little.' + suffix);
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' lays glowing hands on ' + targetActor.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
    SendUserMessage(targetUserId, sourceActor.GetCombatName(true) + ' lays glowing hands on you, and your wounds close a little.' + suffix);
}
