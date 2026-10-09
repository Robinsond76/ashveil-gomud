// Phase 38b: Greater Heal. A Priest's heal of about two average hits. A Hierarch's heal also removes one harmful status, once per patient per battle.
SPELL_ID = 'greaterheal';
SPELL_NAME = 'Greater Heal';
WAIT_ROUNDS = 2; // greaterheal.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You begin a deep prayer, and light pools in your hands.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S begins a deep prayer, and light pools in %P hands.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep praying. The light burns bright.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps praying. The light burns bright.') + rounds, sourceActor.UserId());
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
        var rolled = Math.max(1, Math.floor(sourceActor.SpellPower(SPELL_ID) * sourceActor.HealFactor()));
        var healed = target.AddHealth(rolled);
        var suffix = ' (' + healed + ' healed' + target.WoundNote(rolled, healed);
        if (sourceActor.ClassEffect('cleanseheal') > 0) {
            var word = target.CleanseOne('heal');
            if (word != '') { suffix += ', ' + word + ' removed'; }
        }
        suffix += ')';
        tell(sourceActor, target,
            'Your light floods ' + target.GetCombatName(false) + ', and deep wounds close.' + suffix,
            fill(sourceActor, '%S\'s light floods you, and deep wounds close.') + suffix,
            fill(sourceActor, '%S\'s light floods ') + target.GetCombatName(false) + '.' + suffix);
    }
}
