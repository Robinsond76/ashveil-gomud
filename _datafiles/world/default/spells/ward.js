// Phase 38b: Ward. One ally's next blows are absorbed, each up to one average hit of the caster's level (the ward spell's power block). The class's WardBlows says how many blows. One ward per ally.
SPELL_ID = 'ward';
SPELL_NAME = 'Ward';
WAIT_ROUNDS = 1; // ward.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You raise your hands and begin a warding prayer.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S raises %P hands and begins a warding prayer.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep warding. The air thickens around your ally.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps warding. The air thickens.') + rounds, sourceActor.UserId());
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
        var cap = Math.max(1, Math.floor(sourceActor.SpellPower(SPELL_ID)));
        var blows = Math.max(1, sourceActor.ClassEffect('wardblows'));
        var suffix = ' (ward, absorbs up to ' + cap + (blows > 1 ? ' of each of ' + blows + ' blows)' : ')');
        if (!target.GrantWard(cap, blows)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' already stands under a ward. (no effect)', '', '');
            continue;
        }
        tell(sourceActor, target,
            'You lay a ward over ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, '%S lays a ward over you.') + suffix,
            fill(sourceActor, '%S lays a ward over ') + target.GetCombatName(false) + '.' + suffix);
    }
}
