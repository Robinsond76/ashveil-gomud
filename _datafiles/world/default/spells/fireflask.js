// Phase 39g: Fire Flask. A flask of fire bursts on the foe aimed at and the foe beside it (the
// Bombardier's route: a hotter flask, one more foe, and foes left burning). The size is its power block
// (fireflask.yaml, read through SpellPower), times the caster's SpellFactor against each target.
SPELL_ID = 'fireflask';
SPELL_NAME = 'Fire Flask';
BURNING_BUFF = 1105; // Phase 30a: the burning status

function harmRoll(sourceActor, targetActor) {
    var pct = 100 + sourceActor.ClassEffect('flaskfire');
    return Math.max(1, Math.floor(sourceActor.SpellPower('fireflask') * sourceActor.SpellFactor(targetActor) * pct / 100));
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    SendUserMessage(sourceActor.UserId(), 'You uncork a red flask and hurl it.');
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S uncorks a red flask and hurls it.'), sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
}

function onMagic(sourceActor, targetActors) {
    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var burn = sourceActor.ClassEffect('flaskburn') > 0;

    SendUserMessage(sourceUserId, 'The flask bursts into flame.');
    SendRoomMessage(roomId, 'The flask bursts into flame.', sourceUserId);

    for (var i = 0; i < targetActors.length; i++) {
        var target = targetActors[i];
        var targetUserId = target.UserId();
        var dealt = -target.AddHealth(-harmRoll(sourceActor, target));
        var suffix = ' (' + dealt + ' damage';
        if (burn) {
            target.GiveBuff(BURNING_BUFF, 'spell');
            suffix += ', burning';
        }
        suffix += ')';
        SendUserMessage(sourceUserId, '    Fire washes over ' + target.GetCombatName(false) + '.' + suffix);
        SendRoomMessage(roomId, '    Fire washes over ' + target.GetCombatName(false) + '.' + suffix, sourceUserId, targetUserId);
        SendUserMessage(targetUserId, '    Fire washes over you.' + suffix);
    }
}
