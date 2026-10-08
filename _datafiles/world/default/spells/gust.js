// Phase 39c: Gust, a Shaman's small wind spell. The spell's size is its power block (gust.yaml, read through
// SpellPower); skill decides how well it lands (SpellFactor).
function harmRoll(sourceActor, targetActor) {
    return Math.max(1, Math.floor(sourceActor.SpellPower('gust') * sourceActor.SpellFactor(targetActor)));
}

SPELL_NAME = 'Gust';
WAIT_ROUNDS = 1; // gust.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function onCast(sourceActor, targetActor) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You begin to chant, and the air stirs about your hands.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' begins to chant, and the air stirs about them.' + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. The wind rises.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCombatName(true) + ' keeps chanting. The wind rises.' + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, targetActor) {
    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var targetUserId = targetActor.UserId();

    // Apply the harm first, and report what it took.
    var dealt = -targetActor.AddHealth(-harmRoll(sourceActor, targetActor));
    var suffix = ' (' + dealt + ' damage)';

    SendUserMessage(sourceUserId, 'You loose a sharp gust, and it slams into ' + targetActor.GetCombatName(false) + '.' + suffix);
    SendRoomMessage(roomId, sourceActor.GetCombatName(true) + ' looses a sharp gust into ' + targetActor.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
    SendUserMessage(targetUserId, sourceActor.GetCombatName(true) + ' looses a sharp gust, and it slams into you.' + suffix);
}
