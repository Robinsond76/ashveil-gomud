// Phase 35b: the spell's size is its power block (sparks.yaml, read through
// SpellPower); skill decides how well it lands: the roll is multiplied by
// the caster's SpellFactor against the target (0.5 to 1.5, its Attack
// against the target's Evasion).
function harmRoll(sourceActor, targetActor) {
    return Math.max(1, Math.floor(sourceActor.SpellPower('sparks') * sourceActor.SpellFactor(targetActor)));
}

SPELL_NAME = 'Shower of Sparks';
OVERLOADED_BUFF = 1106; // Phase 30a: the sparks scramble whatever they sear
WAIT_ROUNDS = 1; // sparks.yaml's waitrounds

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
    SendUserMessage(sourceActor.UserId(), 'You begin to chant, and the air around your hands starts to crackle.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' begins to chant, and the air starts to crackle.' + rounds, sourceActor.UserId());
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
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. Sparks jump between your fingers.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps chanting. Sparks jump and snap in the air.' + rounds, sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt: one cast line, then an
 * indented line per target.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActors) {

    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();

    SendUserMessage(sourceUserId, 'You fling your hands open, and a shower of sparks bursts from them.');
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' flings open ' + sourceActor.GetCombatPronoun('possessive') + ' hands, and a shower of sparks bursts out.', sourceUserId);

    for (var i = 0; i < targetActors.length; i++) {

        var target = targetActors[i];
        var targetUserId = target.UserId();

        // Apply the harm first, and report what it took.
        var dealt = -target.AddHealth(-harmRoll(sourceActor, target));
        // Phase 30a: the charge leaves its target overloaded.
        target.GiveBuff(OVERLOADED_BUFF, 'spell');
        var suffix = ' (' + dealt + ' damage, overloaded)';

        if (sourceUserId != 0 && sourceUserId == targetUserId) {
            SendUserMessage(sourceUserId, '    Sparks sear your own skin.' + suffix);
            SendRoomMessage(roomId, '    Sparks sear ' + sourceActor.GetCombatName(false) + '.' + suffix, sourceUserId);
            continue;
        }

        SendUserMessage(sourceUserId, '    Sparks sear ' + target.GetCombatName(false) + '.' + suffix);
        SendRoomMessage(roomId, '    Sparks sear ' + target.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
        SendUserMessage(targetUserId, '    Sparks sear you.' + suffix);
    }
}
