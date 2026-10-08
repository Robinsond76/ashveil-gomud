// Phase 39c: Lightning, a Shaman's heavy bolt at one foe. In Rain it strikes 50% harder; a Stormcaller's
// bolt also strikes a second foe for a share of its damage (the 'chain' class effect). The spell's size is its
// power block (lightning.yaml, read through SpellPower); skill decides how well it lands (SpellFactor).
SPELL_ID = 'lightning';
SPELL_NAME = 'Lightning';
WAIT_ROUNDS = 1; // lightning.yaml's waitrounds

// Phase 29c narration voice: mechanics in lowercase parentheses at the end.
function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' turn)' : ' turns)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You lift your hands, and the air crackles.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S lifts %P hands, and the air crackles.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. Hair stands on end.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps chanting. The air hums.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, targetActors) {
    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var rain = sourceActor.Weather() == 'rain';
    var chain = Math.max(0, sourceActor.ClassEffect('chain'));
    // Phase 39i: a Tempest Lord's Storm wall chains through the whole row it was aimed along.
    var reach = sourceActor.ClassEffect('chainrow') > 0 ? targetActors.length : 2;
    for (var i = 0; i < targetActors.length && i < reach; i++) {
        var foe = targetActors[i];
        var pct = i == 0 ? 100 : chain;
        if (pct <= 0) { break; }
        var base = sourceActor.SpellPower(SPELL_ID) * sourceActor.SpellFactor(foe);
        if (rain) { base = base * 1.5; }
        var dealt = -foe.AddHealth(-Math.max(1, Math.floor(base * pct / 100)));
        var suffix = ' (' + dealt + ' damage' + (rain ? ', rain' : '') + (i > 0 ? ', chained' : '') + ')';
        var name = foe.GetCombatName(false);
        if (i == 0) {
            SendUserMessage(sourceUserId, 'You loose the lightning, and it strikes ' + name + '.' + suffix);
            SendRoomMessage(roomId, fill(sourceActor, '%S looses lightning, and it strikes ') + name + '.' + suffix, sourceUserId, foe.UserId());
            SendUserMessage(foe.UserId(), fill(sourceActor, '%S looses lightning, and it strikes you.') + suffix);
        } else {
            SendUserMessage(sourceUserId, 'The bolt leaps on to ' + name + '.' + suffix);
            SendRoomMessage(roomId, 'The bolt leaps on to ' + name + '.' + suffix, sourceUserId, foe.UserId());
            SendUserMessage(foe.UserId(), 'The bolt leaps on and strikes you.' + suffix);
        }
    }
}
