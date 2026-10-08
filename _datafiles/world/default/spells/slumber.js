// Phase 38a: Slumber, a Witch's hex. The mechanics live in Go
// (ScriptActor.HexTargets reaches the foes a Witch's level allows;
// ScriptActor.CastHex rolls the resist, applies the status and keeps the
// no-lock-loop immunity); this script tells it.
SPELL_ID = 'slumber';
SPELL_NAME = 'Slumber';
WAIT_ROUNDS = 1; // slumber.yaml's waitrounds
STATUS_WORD = 'asleep';

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

/**
 * Called when the casting is initialized.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onCast(sourceActor, targetActors) {

    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You begin to chant a low, drowsy rhyme.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S begins to chant a low, drowsy rhyme.') + rounds, sourceActor.UserId());
    return true;
}

/**
 * Called each round while the spell is being cast.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onWait(sourceActor, targetActors) {

    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. Your voice drops to a murmur.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps chanting a drowsy rhyme.') + rounds, sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt: one cast line, then an
 * indented line per target.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject[]} targetActors - The targets of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActors) {

    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var targets = sourceActor.HexTargets(targetActors);

    SendUserMessage(sourceUserId, 'You sing the last line, and a heavy hush rolls out over the foe.');
    SendRoomMessage(roomId, fill(sourceActor, '%S sings out the last line, and a heavy hush rolls over the foe.'), sourceUserId);

    for (var i = 0; i < targets.length; i++) {

        var target = targets[i];
        var targetUserId = target.UserId();
        var name = target.GetCombatName(false);
        var result = sourceActor.CastHex(SPELL_ID, target);
        var line = '';

        if (result.reason == 'landed') {
            var length = result.rounds > 0 ? ', ' + result.rounds + (result.rounds == 1 ? ' round' : ' rounds') : '';
            var suffix = ' (' + STATUS_WORD + length + ')';
            line = '    ' + name + ' slumps, fast asleep.' + suffix;
            SendUserMessage(targetUserId, '    A heavy hush closes over you, and you slump into sleep.' + suffix);
        } else if (result.reason == 'resisted') {
            line = '    ' + name + ' shrugs the hex off. (resisted)';
            SendUserMessage(targetUserId, '    You shrug the hex off. (resisted)');
        } else if (result.reason == 'immune') {
            line = '    ' + name + ' is past hexing for the moment. (immune)';
        } else {
            line = '    ' + name + ' is already ' + STATUS_WORD + '. (no effect)';
        }

        SendUserMessage(sourceUserId, line);
        SendRoomMessage(roomId, line, sourceUserId, targetUserId);
    }
}
