// Phase 39c: Stoneskin, an Earthspeaker's Barkskin of the earth. One ally gains armor for the battle.
SPELL_ID = 'stoneskin';
SPELL_NAME = 'Stoneskin';
WAIT_ROUNDS = 0; // stoneskin.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
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
    // Phase 39i: a Mountain Speaker's Stone cloak covers the target's whole row.
    var targetActors = sourceActor.ClassEffect('stonerow') > 0 ? single.RowAllies(1) : [single];
    var landed = false;
    for (var i = 0; i < targetActors.length; i++) {
        var target = targetActors[i];
        var armor = Math.max(1, sourceActor.ClassEffect('stoneskin'));
        var suffix = ' (+' + armor + ' armor)';
        if (!target.GrantBark(armor, 0)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' already wears stone. (no effect)', '', '');
            continue;
        }
        landed = true;
        tell(sourceActor, target,
            'Stone hardens over ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, 'Stone hardens over your skin at %S\'s word.') + suffix,
            'Stone hardens over ' + target.GetCombatName(false) + '.' + suffix);
    }
    // Phase 39i: a Mountain Speaker's Tremor. The earth it spoke to shakes under the foes' front row.
    var tremor = sourceActor.ClassEffect('tremor');
    if (landed && tremor > 0) {
        var felled = sourceActor.Tremor(tremor);
        var line;
        if (felled.length > 0) {
            line = 'The ground shakes under the foes\' front row, and ' + felled.join(', ') + (felled.length == 1 ? ' goes' : ' go') + ' down. (tremor, knocked down)';
        } else {
            line = 'The ground shakes under the foes\' front row, and they keep their feet. (tremor)';
        }
        SendUserMessage(sourceActor.UserId(), line);
        SendRoomMessage(sourceActor.GetRoomId(), line, sourceActor.UserId());
    }
}
