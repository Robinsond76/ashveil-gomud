// Phase 39g: Healing Draught. A thrown flask heals an ally for the spell's power block
// (draught.yaml, read through SpellPower), the Alchemist's route making it stronger
// (Potent draughts), splashing the next most hurt ally (Splash draught) and cleaning the
// patient (Clean draught). No chant: it lands as it is thrown.
SPELL_ID = 'draught';
SPELL_NAME = 'Healing Draught';

function healRoll(sourceActor) {
    var pct = 100 + sourceActor.ClassEffect('flaskheal');
    return Math.max(1, Math.floor(sourceActor.SpellPower('draught') * sourceActor.HealFactor() * pct / 100));
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You uncork a flask from your satchel.');
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S uncorks a flask from %P satchel.'), sourceActor.UserId());
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

function same(a, b) {
    return a.UserId() == b.UserId() && a.InstanceId() == b.InstanceId();
}

function onMagic(sourceActor, target) {
    var rolled = healRoll(sourceActor);
    var healed = target.AddHealth(rolled);
    var suffix = ' (' + healed + ' healed' + target.WoundNote(rolled, healed);

    // Clean draught: one harmful status comes off the patient, once a battle for each ally.
    if (sourceActor.ClassEffect('flaskclean') > 0) {
        var word = target.CleanseOne('draught');
        if (word != '') {
            suffix += ', ' + word + ' cleared';
        }
    }
    suffix += ')';

    if (same(sourceActor, target)) {
        tell(sourceActor, target,
            'You drink a healing draught, and your wounds close.' + suffix, '',
            fill(sourceActor, '%S drinks a healing draught.') + suffix);
    } else {
        tell(sourceActor, target,
            'You throw a healing draught to ' + target.GetCombatName(false) + ', and the wounds close.' + suffix,
            fill(sourceActor, '%S throws you a healing draught, and your wounds close.') + suffix,
            fill(sourceActor, '%S throws a healing draught to ') + target.GetCombatName(false) + '.' + suffix);
    }

    // Splash draught: the next most hurt ally takes a share of it.
    var splash = sourceActor.ClassEffect('flasksplash');
    if (splash > 0) {
        var hurt = sourceActor.HurtAllies(false);
        for (var i = 0; i < hurt.length; i++) {
            if (same(hurt[i], target)) {
                continue;
            }
            var share = Math.max(1, Math.floor(rolled * splash / 100));
            var got = hurt[i].AddHealth(share);
            var note = ' (' + got + ' healed' + hurt[i].WoundNote(share, got) + ')';
            tell(sourceActor, hurt[i],
                'The spray from the flask reaches ' + hurt[i].GetCombatName(false) + '.' + note,
                'The spray from the flask reaches you.' + note,
                'The spray from the flask reaches ' + hurt[i].GetCombatName(false) + '.' + note);
            break;
        }
    }
}
