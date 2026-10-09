// Phase 38b: Call the Host. Calls the caster's summon for the battle, once; the chant is its arrival.
SPELL_ID = 'callhost';
SPELL_NAME = 'Call the Host';
WAIT_ROUNDS = 2; // callhost.yaml's waitrounds

function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You lift your holy symbol and call upon the Host.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S lifts %P holy symbol and calls upon the Host.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep calling. Light gathers overhead.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps calling. Light gathers overhead.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    if (sourceActor.Summon()) {
        SendUserMessage(sourceActor.UserId(), 'An Angel descends in a column of light and takes its place beside you.');
        SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S calls, and an Angel descends in a column of light.'), sourceActor.UserId());
    } else {
        SendUserMessage(sourceActor.UserId(), 'Nothing answers the call. (no effect)');
    }
}
