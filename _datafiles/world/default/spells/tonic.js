// Phase 39g: Bracing Tonic. One ally gains +5 Attack and +5 Evasion for 3 combat rounds. One tonic per ally at
// a time. A Mutagenist's tonic also hardens the ally (armor for the battle) at the price of some of its health.
SPELL_ID = 'tonic';
SPELL_NAME = 'Bracing Tonic';

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You uncork an amber flask.');
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S uncorks an amber flask.'), sourceActor.UserId());
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
    var rounds = 3 + sourceActor.ClassEffect('toniclong');
    if (!target.GrantBless(rounds)) {
        tell(sourceActor, target, target.GetCombatName(false) + ' is already braced. (no effect)', '', '');
        return;
    }
    var suffix = ' (+5 Attack and Evasion, ' + rounds + ' rounds';

    // Mutagen: armor for the battle, for a share of the ally's health (never its last point).
    var armor = sourceActor.ClassEffect('mutagenarmr');
    if (armor > 0 && target.GrantBark(armor, 0)) {
        suffix += ', +' + armor + ' armor for the battle';
        if (sourceActor.ClassEffect('mutagenfree') == 0) {
            var cost = Math.floor(target.GetHealthLimit() * sourceActor.ClassEffect('mutagencost') / 100);
            cost = Math.min(cost, target.GetHealth() - 1);
            if (cost > 0) {
                target.AddHealth(-cost);
                suffix += ', ' + cost + ' health paid';
            }
        }
    }
    suffix += ')';

    if (sourceActor.UserId() == target.UserId() && sourceActor.InstanceId() == target.InstanceId()) {
        tell(sourceActor, target, 'You drink a bracing tonic.' + suffix, '', fill(sourceActor, '%S drinks a bracing tonic.') + suffix);
        return;
    }
    tell(sourceActor, target,
        'You throw a bracing tonic to ' + target.GetCombatName(false) + '.' + suffix,
        fill(sourceActor, '%S throws you a bracing tonic.') + suffix,
        fill(sourceActor, '%S throws a bracing tonic to ') + target.GetCombatName(false) + '.' + suffix);
}
