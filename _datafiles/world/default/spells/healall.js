
HEAL_DICE_QTY = 2;
HEAL_DICE_SIDES = 3;

SPELL_NAME = 'Minor Heal All';
WAIT_ROUNDS = 2; // healall.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

/**
 * Called when the casting is initialized.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onCast(sourceActor, targetActors) {

    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You begin a low prayer for your companions, and warmth gathers around you.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' begins a low prayer.' + rounds, sourceActor.UserId());
    return true;
}

/**
 * Called each round while the spell is being cast.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onWait(sourceActor, targetActors) {

    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep praying. A soft light spreads from you.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps praying, and a soft light spreads through the room.' + rounds, sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt: one cast line, then one
 * indented line naming everyone healed, each viewer as "you".
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActors) {

    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();

    // Apply every heal first, and report what each mended.
    var healed = [];
    for (var i = 0; i < targetActors.length; i++) {
        var amount = targetActors[i].AddHealth(UtilDiceRoll(HEAL_DICE_QTY, HEAL_DICE_SIDES));
        healed.push({ userId: targetActors[i].UserId(), name: targetActors[i].GetCombatName(false), amount: amount });
    }

    // listFor is the healed line as viewerUserId sees it (0: the room).
    function listFor(viewerUserId) {
        var entries = [];
        for (var i = 0; i < healed.length; i++) {
            var who = (viewerUserId != 0 && healed[i].userId == viewerUserId) ? 'you' : healed[i].name;
            entries.push(who + ' (' + healed[i].amount + ' healed)');
        }
        return '    ' + entries.join(' · ');
    }

    SendUserMessage(sourceUserId, 'You open your hands, and the light settles over your companions.');
    SendUserMessage(sourceUserId, listFor(sourceUserId));

    var exclude = [sourceUserId];
    for (var i = 0; i < healed.length; i++) {
        if (healed[i].userId == 0 || healed[i].userId == sourceUserId) {
            continue;
        }
        exclude.push(healed[i].userId);
        SendUserMessage(healed[i].userId, sourceActor.GetCombatName(true) + ' opens ' + sourceActor.GetCombatPronoun('possessive') + ' hands, and the light settles over you.');
        SendUserMessage(healed[i].userId, listFor(healed[i].userId));
    }

    // SendRoomMessage's exclusions are variadic: spread the list.
    SendRoomMessage.apply(null, [roomId, sourceActor.GetCombatName(true) + ' opens ' + sourceActor.GetCombatPronoun('possessive') + ' hands, and the light settles over the company.'].concat(exclude));
    SendRoomMessage.apply(null, [roomId, listFor(0)].concat(exclude));
}
