// Phase 38c3: Raise the Fallen. Raises the strongest foe that fell in this battle as the caster's thrall; the chant is its rising.
SPELL_ID = 'raisefallen';
SPELL_NAME = 'Raise the Fallen';
WAIT_ROUNDS = 1; // raisefallen.yaml's waitrounds

function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You speak over the fallen, and the cold in the room leans toward it.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S speaks over the fallen, and the cold in the room leans toward it.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. The body shudders.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps chanting. The body shudders.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    var name = sourceActor.Raise();
    if (name != '') {
        SendUserMessage(sourceActor.UserId(), 'The fallen rises, empty-eyed, and takes up its weapon for you. (thrall: ' + name + ')');
        SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S raises the fallen, and it rises, empty-eyed, to fight for the company.') + ' (thrall: ' + name + ')', sourceActor.UserId());
    } else {
        SendUserMessage(sourceActor.UserId(), 'Nothing answers the rite. (no effect)');
    }
}
