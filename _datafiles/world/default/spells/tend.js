TEND_DICE_QTY = 2;
TEND_DICE_SIDES = 3;

SPELL_NAME = 'Tend Wounds';
WAIT_ROUNDS = 1; // tend.yaml's waitrounds

// Phase 30b: a cleric's tending closes part of a lasting wound, raising
// the wound limit. It heals no health itself; heal does that, up to the
// new limit. The dice match internal/wounds.DefaultRules.
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

    // Review fix: no mana spent on someone with nothing to tend.
    if (!targetActor.HasLastingWound()) {
        SendUserMessage(sourceActor.UserId(), 'You find no wound there to tend.');
        return false;
    }

    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You murmur a prayer of mending over clean linen.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' murmurs a prayer of mending.' + rounds, sourceActor.UserId());
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
    SendUserMessage(sourceActor.UserId(), 'You keep murmuring.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps murmuring.' + rounds, sourceActor.UserId());
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

    var tended = targetActor.TendWound(UtilDiceRoll(TEND_DICE_QTY, TEND_DICE_SIDES));
    if (tended.closed == 0) {
        SendUserMessage(sourceUserId, 'You find no wound there to tend.');
        return;
    }
    var suffix = ' (wound treated, limit ' + tended.limit + ' of ' + tended.max + ')';

    if (sourceUserId != 0 && sourceUserId == targetUserId) {
        SendUserMessage(sourceUserId, 'You press your hands to ' + tended.wound + ', and it draws closed a little.' + suffix);
        SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' tends ' + sourceActor.GetCombatPronoun('possessive') + ' own wound.' + suffix, sourceUserId);
        return;
    }

    SendUserMessage(sourceUserId, 'You tend ' + targetActor.GetCombatName(false) + ': ' + tended.wound + ' draws closed a little.' + suffix);
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' tends ' + targetActor.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
    SendUserMessage(targetUserId, sourceActor.GetCombatName(true) + ' tends you: ' + tended.wound + ' draws closed a little.' + suffix);
}
