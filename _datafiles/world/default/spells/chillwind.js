// Phase 39c: Chill Wind, a Shaman's weather. The weather itself lives in Go (ScriptActor.CallWeather puts it on
// the caster's battle and its mark on the foes); this script tells it.
SPELL_ID = 'chillwind';
SPELL_NAME = 'Chill Wind';
WAIT_ROUNDS = 1; // chillwind.yaml's waitrounds
KIND = 'chill';

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You turn your face to the north and call a cold wind.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S turns to the north and calls a cold wind.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep calling. The wind bites.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps calling. The wind bites.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    var result = sourceActor.CallWeather(KIND);
    if (result.reason == 'same') {
        SendUserMessage(sourceActor.UserId(), 'The cold wind already hangs over the battle. (no effect)');
        return;
    }
    if (!result.landed) {
        SendUserMessage(sourceActor.UserId(), 'The sky does not answer. (no effect)');
        return;
    }
    var suffix = ' (' + result.rounds + ' rounds, foe chants and sling shots slowed)';
    var replaced = result.replaced != '' ? ' It replaces the ' + (result.replaced == 'chill' ? 'chill wind' : result.replaced) + '.' : '';
    SendUserMessage(sourceActor.UserId(), 'A bitter wind sweeps across the battle.' + replaced + suffix);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S calls, and a bitter wind sweeps across the battle.') + replaced + suffix, sourceActor.UserId());
}
