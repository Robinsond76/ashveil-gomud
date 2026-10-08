// Phase 35b: the spell's size is its power block (mm.yaml, read through
// SpellPower); skill decides how well it lands: the roll is multiplied by
// the caster's SpellFactor against the target (0.5 to 1.5, its Attack
// against the target's Evasion).
function harmRoll(sourceActor, targetActor) {
    return Math.max(1, Math.floor(sourceActor.SpellPower('mm') * sourceActor.SpellFactor(targetActor)));
}

SPELL_NAME = 'Magic Missile';
WAIT_ROUNDS = 1; // mm.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
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
    SendUserMessage(sourceActor.UserId(), 'You begin to chant, and a point of cold light gathers at your fingertips.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' begins to chant, and a point of cold light gathers in the air.' + rounds, sourceActor.UserId());
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
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. The light brightens and begins to turn.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps chanting. The light brightens and begins to turn.' + rounds, sourceActor.UserId());
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

    SendUserMessage(sourceUserId, 'You release the light, and it streaks into ' + targetActor.GetCombatName(false) + '.' + suffix);
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' releases a streak of cold light into ' + targetActor.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
    SendUserMessage(targetUserId, sourceActor.GetCombatName(true) + ' releases a streak of cold light, and it strikes you.' + suffix);

    // Phase 38c3: an Archmage's Arcane Barrage strikes a second foe at full
    // damage.
    if (sourceActor.ClassEffect('barrage') > 0) {
        var all = sourceActor.CastTargets();
        for (var i = 0; i < all.length; i++) {
            var other = all[i];
            if (other.UserId() == targetUserId && other.InstanceId() == targetActor.InstanceId()) { continue; }
            var more = -other.AddHealth(-harmRoll(sourceActor, other));
            var tail = ' (barrage, ' + more + ' damage)';
            SendUserMessage(sourceUserId, 'A second streak of cold light arcs into ' + other.GetCombatName(false) + '.' + tail);
            SendRoomMessage(roomId, 'A second streak of cold light arcs from ' + sourceActor.GetCombatName(true) + ' into ' + other.GetCombatName(false) + '.' + tail, sourceUserId, other.UserId());
            break;
        }
    }
}
