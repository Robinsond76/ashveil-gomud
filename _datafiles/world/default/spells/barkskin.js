// Phase 38b: Barkskin. One ally (or a row, from Wide Bark) gains armor for the battle; Thornhide hurts a foe that strikes it.
SPELL_ID = 'barkskin';
SPELL_NAME = 'Barkskin';
WAIT_ROUNDS = 1; // barkskin.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You press your palm to the bark of the world, and speak.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S presses a palm to the ground and speaks.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep speaking. Bark creeps over skin.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps speaking. Bark creeps over skin.') + rounds, sourceActor.UserId());
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
    var targetActors = sourceActor.ClassEffect('barkrow') > 0 ? single.RowAllies(1) : [single];
    for (var i = 0; i < targetActors.length; i++) {
        var target = targetActors[i];
        var armor = Math.max(1, sourceActor.ClassEffect('barkskin'));
        var thorns = sourceActor.ClassEffect('thornhide');
        var suffix = ' (+' + armor + ' armor' + (thorns > 0 ? ', thorns ' + thorns : '') + ')';
        if (!target.GrantBark(armor, thorns)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' already wears bark. (no effect)', '', '');
            continue;
        }
        tell(sourceActor, target,
            'Bark creeps over ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, 'Bark creeps over your skin at %S\'s word.') + suffix,
            'Bark creeps over ' + target.GetCombatName(false) + '.' + suffix);
    }
}
