// Phase 38b: Bind the Fiend. Calls the caster's summon for the battle, once; the chant is its arrival.
SPELL_ID = 'bindfiend';
SPELL_NAME = 'Bind the Fiend';
WAIT_ROUNDS = 2; // bindfiend.yaml's waitrounds

function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You trace a binding circle and speak a name that is not yours.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S traces a binding circle and speaks a name.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep binding. The air smells of sulphur.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps binding. The air smells of sulphur.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    if (sourceActor.Summon()) {
        SendUserMessage(sourceActor.UserId(), 'A Demon tears through the circle, snarling, and bows to your will.');
        SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S binds, and a Demon tears through the circle.'), sourceActor.UserId());
    } else {
        SendUserMessage(sourceActor.UserId(), 'Nothing answers the binding. (no effect)');
    }
}
