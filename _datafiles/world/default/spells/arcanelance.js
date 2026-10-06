// Phase 38d: Arcane Lance, a Sorcerer's burst. One heavy, costly bolt at one foe after a chant (blows can break
// it). The rank table's effects change it: Gathered power and High Lance add damage ('lancepct'), Twin Lance
// strikes a second foe for a share ('lancetwin'). The spell's size is its power block (arcanelance.yaml, read
// through SpellPower); skill decides how well it lands (SpellFactor).
SPELL_ID = 'arcanelance';
SPELL_NAME = 'Arcane Lance';
WAIT_ROUNDS = 1; // arcanelance.yaml's waitrounds

function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    // A High Sorcerer's Gathered chant and Instant Lance shorten some chants after this line; onWait gives
    // the true count, so the opening line names none for it (Phase 38d review).
    var shortened = sourceActor.ClassEffect('lancetrim') > 0 || sourceActor.ClassEffect('lancefree') > 0;
    var rounds = shortened ? ' (chanting: ' + SPELL_NAME + ')' : chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You raise your hands, and a spear of white light begins to form.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S raises %P hands, and a spear of white light begins to form.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. The lance lengthens and hums.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps chanting. The lance of light lengthens.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, targetActors) {
    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var twin = Math.max(0, sourceActor.ClassEffect('lancetwin'));
    var more = 1 + Math.max(0, sourceActor.ClassEffect('lancepct')) / 100;
    for (var i = 0; i < targetActors.length && i < 2; i++) {
        var foe = targetActors[i];
        var pct = i == 0 ? 100 : twin;
        if (pct <= 0) { break; }
        var roll = Math.max(1, Math.floor(sourceActor.SpellPower(SPELL_ID) * sourceActor.SpellFactor(foe) * more * pct / 100));
        var dealt = -foe.AddHealth(-roll);
        var suffix = ' (' + dealt + ' damage' + (i > 0 ? ', twin lance' : '') + ')';
        var name = foe.GetCombatName(false);
        if (i == 0) {
            SendUserMessage(sourceUserId, 'You hurl the lance, and it drives into ' + name + '.' + suffix);
            SendRoomMessage(roomId, fill(sourceActor, '%S hurls a lance of light, and it drives into ') + name + '.' + suffix, sourceUserId, foe.UserId());
            SendUserMessage(foe.UserId(), fill(sourceActor, '%S hurls a lance of light, and it drives into you.') + suffix);
        } else {
            SendUserMessage(sourceUserId, 'A second lance forms and strikes ' + name + '.' + suffix);
            SendRoomMessage(roomId, 'A second lance of light strikes ' + name + '.' + suffix, sourceUserId, foe.UserId());
            SendUserMessage(foe.UserId(), 'A second lance of light strikes you.' + suffix);
        }
    }
}
