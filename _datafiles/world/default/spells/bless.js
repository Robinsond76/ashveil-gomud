// Phase 38b: Bless. One ally gains +5 Attack and +5 Evasion for 3 combat rounds. One Bless per ally at a time.
SPELL_ID = 'bless';
SPELL_NAME = 'Bless';
WAIT_ROUNDS = 1; // bless.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You murmur a blessing.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S murmurs a blessing.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep murmuring. The blessing warms.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps murmuring a blessing.') + rounds, sourceActor.UserId());
}

// tell sends a line to the caster, the target and the room, each once.
function tell(sourceActor, target, toSource, toTarget, toRoom) {
    var sourceUserId = sourceActor.UserId();
    var targetUserId = target.UserId();
    SendUserMessage(sourceUserId, toSource);
    if (targetUserId != 0 && targetUserId != sourceUserId) {
        SendUserMessage(targetUserId, toTarget);
    }
    SendRoomMessage(sourceActor.GetRoomId(), toRoom, sourceUserId, targetUserId);
}

function onMagic(sourceActor, single) {
    var targetActors = [single];
    for (var i = 0; i < targetActors.length; i++) {
        var target = targetActors[i];
        var suffix = ' (+5 Attack and Evasion, 3 rounds)';
        if (!target.GrantBless(3)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' is already blessed. (no effect)', '', '');
            continue;
        }
        tell(sourceActor, target,
            'You bless ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, '%S blesses you.') + suffix,
            fill(sourceActor, '%S blesses ') + target.GetCombatName(false) + '.' + suffix);
    }
}
