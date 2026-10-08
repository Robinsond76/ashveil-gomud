// Phase 38b: Grove. Rejuvenation on a whole formation row at the Grove's share each, plus half of it at once.
SPELL_ID = 'grove';
SPELL_NAME = 'Grove';
WAIT_ROUNDS = 1; // grove.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You spread your arms, and the ground begins to green.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S spreads %P arms, and the ground begins to green.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep singing. Roots stir beneath your feet.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps singing. Roots stir.') + rounds, sourceActor.UserId());
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
    var targetActors = single.RowAllies(Math.max(1, sourceActor.ClassEffect('grove')));
    var rounds = Math.max(1, sourceActor.ClassEffect('rejuvenation'));
    var pct = Math.max(1, sourceActor.ClassEffect('rejuvpct'));
    var grove = Math.max(1, sourceActor.ClassEffect('grovepct'));
    var total = Math.max(1, Math.floor(sourceActor.SpellPower('heal') * sourceActor.HealFactor() * pct / 100 * grove / 100));
    var names = [];
    for (var i = 0; i < targetActors.length; i++) {
        if (targetActors[i].StartRejuv(rounds, total)) {
            // Phase 38c1 review: the grove blooms, healing half its share at once.
            var bloom = targetActors[i].AddHealth(Math.floor(total / 2));
            names.push(targetActors[i].GetCombatName(false) + ' (' + bloom + ' now, ' + total + ' over ' + rounds + ' rounds)');
            SendUserMessage(targetActors[i].UserId(), fill(sourceActor, 'A grove of green light from %S settles over you.'));
        }
    }
    var line = names.length ? '    ' + names.join(' · ') : '    No one needs the grove. (no effect)';
    SendUserMessage(sourceActor.UserId(), 'The ground greens, and the grove settles over the row.');
    SendUserMessage(sourceActor.UserId(), line);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, 'The ground greens around %S, and a grove settles over the row.'), sourceActor.UserId());
    SendRoomMessage(sourceActor.GetRoomId(), line, sourceActor.UserId());
}
