// Phase 38b: Rejuvenation. One ally heals over the class's Rejuvenation rounds for its share of a Minor Heal; Wild Growth also cures poison.
SPELL_ID = 'rejuvenation';
SPELL_NAME = 'Rejuvenation';
WAIT_ROUNDS = 1; // rejuvenation.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You hum a slow green song.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S hums a slow green song.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep humming. Green light gathers.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps humming. Green light gathers.') + rounds, sourceActor.UserId());
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
        var rounds = Math.max(1, sourceActor.ClassEffect('rejuvenation'));
        var pct = Math.max(1, sourceActor.ClassEffect('rejuvpct'));
        var total = Math.max(1, Math.floor(sourceActor.SpellPower('heal') * sourceActor.HealFactor() * pct / 100));
        var suffix = ' (' + total + ' over ' + rounds + ' rounds)';
        if (sourceActor.ClassEffect('wildgrowth') > 0 && target.HasBuff(13)) {
            target.RemoveBuff(13);
            suffix = suffix.slice(0, -1) + ', poison cured)';
        }
        if (!target.StartRejuv(rounds, total)) {
            tell(sourceActor, target, target.GetCombatName(false) + ' is already rejuvenating. (no effect)', '', '');
            continue;
        }
        tell(sourceActor, target,
            'Green light settles on ' + target.GetCombatName(false) + '.' + suffix,
            fill(sourceActor, 'Green light from %S settles on you.') + suffix,
            fill(sourceActor, 'Green light from %S settles on ') + target.GetCombatName(false) + '.' + suffix);
    }
}
