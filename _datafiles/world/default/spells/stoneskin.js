// Phase 39c: Stoneskin, an Earthspeaker's Barkskin of the earth. One ally gains armor for the battle.
SPELL_ID = 'stoneskin';
SPELL_NAME = 'Stoneskin';
WAIT_ROUNDS = 0; // stoneskin.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You press your palm to the stone of the world, and speak.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S presses a palm to the stone and speaks.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep speaking. The ground grumbles.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps speaking. The ground grumbles.') + rounds, sourceActor.UserId());
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
        var armor = Math.max(1, sourceActor.ClassEffect('stoneskin'));
        var suffix = ' (+' + armor + ' armor)';
        if (!target.GrantBark(armor, 0)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' already wears stone. (no effect)', '', '');
            continue;
        }
        tell(sourceActor, target,
            'Stone hardens over ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, 'Stone hardens over your skin at %S\'s word.') + suffix,
            'Stone hardens over ' + target.GetCombatName(false) + '.' + suffix);
    }
}
