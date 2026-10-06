// Phase 39g: Antidote. A thrown flask takes an ally's poison and bleeding off it.
SPELL_ID = 'antidote';
SPELL_NAME = 'Antidote';

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You uncork a pale flask.');
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S uncorks a pale flask.'), sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
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

function onMagic(sourceActor, target) {
    var cured = [];
    if (target.CancelBuffWithFlag('poison')) {
        cured.push('poison');
    }
    if (target.CancelBuffWithFlag('bleeding')) {
        cured.push('bleeding');
    }
    var suffix = cured.length > 0 ? ' (' + cured.join(' and ') + ' cured)' : ' (no effect)';
    if (sourceActor.UserId() == target.UserId() && sourceActor.InstanceId() == target.InstanceId()) {
        tell(sourceActor, target, 'You drink the antidote.' + suffix, '', fill(sourceActor, '%S drinks an antidote.') + suffix);
        return;
    }
    tell(sourceActor, target,
        'You throw the antidote to ' + target.GetCombatName(false) + '.' + suffix,
        fill(sourceActor, '%S throws you an antidote.') + suffix,
        fill(sourceActor, '%S throws an antidote to ') + target.GetCombatName(false) + '.' + suffix);
}
