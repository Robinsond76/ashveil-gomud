// Phase 39c: Rain, a Shaman's weather. The weather itself lives in Go (ScriptActor.CallWeather puts it on
// the caster's battle and its mark on the foes); this script tells it.
SPELL_ID = 'rain';
SPELL_NAME = 'Rain';
WAIT_ROUNDS = 1; // rain.yaml's waitrounds
KIND = 'rain';

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You raise your arms and call the rain.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S raises %P arms and calls the rain.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep calling. The clouds darken.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps calling. The clouds darken.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    var result = sourceActor.CallWeather(KIND);
    if (result.reason == 'same') {
        SendUserMessage(sourceActor.UserId(), 'The rain already hangs over the battle. (no effect)');
        return;
    }
    if (!result.landed) {
        SendUserMessage(sourceActor.UserId(), 'The sky does not answer. (no effect)');
        return;
    }
    var suffix = ' (' + result.rounds + ' rounds, Lightning 50% stronger)';
    var replaced = result.replaced != '' ? ' It replaces the ' + (result.replaced == 'chill' ? 'chill wind' : result.replaced) + '.' : '';
    SendUserMessage(sourceActor.UserId(), 'Rain drives down over the battle.' + replaced + suffix);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S calls, and rain drives down over the battle.') + replaced + suffix, sourceActor.UserId());
}
