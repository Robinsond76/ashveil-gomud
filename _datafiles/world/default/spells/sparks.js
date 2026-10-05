
// Phase 35a2: 4 + 1d4 + level/15 + Mysticism/25 a target, about 0.6x
// Magic Missile (35b's ratio).
HARM_BASE = 4;
HARM_DICE_QTY = 1;
HARM_DICE_SIDES = 4;
HARM_LEVEL_DIV = 15;
HARM_MYSTICISM_DIV = 25;

// Phase 35a2: a spell lands about one weapon hit, and skill decides how
// well: the roll is multiplied by the caster's SpellFactor against the
// target (0.5 to 1.5, its Attack against the target's Evasion).
function harmRoll(sourceActor, targetActor) {
    var roll = HARM_BASE + UtilDiceRoll(HARM_DICE_QTY, HARM_DICE_SIDES) +
        Math.floor(sourceActor.GetLevel() / HARM_LEVEL_DIV) +
        Math.floor(sourceActor.GetStat('mysticism') / HARM_MYSTICISM_DIV);
    return Math.max(1, Math.floor(roll * sourceActor.SpellFactor(targetActor)));
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
