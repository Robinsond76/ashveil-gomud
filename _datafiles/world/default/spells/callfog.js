// Phase 39c: Call Fog, a Shaman's weather. The weather itself lives in Go (ScriptActor.CallWeather puts it on
// the caster's battle and its mark on the foes); this script tells it.
SPELL_ID = 'callfog';
SPELL_NAME = 'Call Fog';
WAIT_ROUNDS = 0; // callfog.yaml's waitrounds
KIND = 'fog';

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You raise your hands and call the fog down.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S raises %P hands and calls the fog down.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep calling. The air grows damp and gray.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps calling. The air grows damp and gray.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, single) {
    var result = sourceActor.CallWeather(KIND);
    if (result.reason == 'same') {
        SendUserMessage(sourceActor.UserId(), 'The fog already hangs over the battle. (no effect)');
        return;
    }
    if (!result.landed) {
        SendUserMessage(sourceActor.UserId(), 'The sky does not answer. (no effect)');
        return;
    }
    var suffix = ' (' + result.rounds + ' rounds, foe ranged attacks and spells weaker)';
    var replaced = result.replaced != '' ? ' It replaces the ' + (result.replaced == 'chill' ? 'chill wind' : result.replaced) + '.' : '';
    SendUserMessage(sourceActor.UserId(), 'A gray fog rolls over the battle.' + replaced + suffix);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S calls, and a gray fog rolls over the battle.') + replaced + suffix, sourceActor.UserId());
}
