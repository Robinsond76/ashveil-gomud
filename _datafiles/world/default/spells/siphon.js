// Phase 38b: Siphon, a Blood Priest's drain. It strikes a foe for a Magic
// Missile's worth, scaled by skill, and heals the most hurt ally for what it
// dealt. Twin Siphon also strikes a second foe and heals a second ally, at
// 60%. With Blood Ward, healing beyond an ally's full health becomes a ward.
SPELL_ID = 'siphon';
SPELL_NAME = 'Siphon';
WAIT_ROUNDS = 1; // siphon.yaml's waitrounds

function chanting(rounds) {
    return ' (chanting: ' + SPELL_NAME + ', ' + rounds + (rounds == 1 ? ' round)' : ' rounds)');
}

function fill(sourceActor, text) {
    return text.replace('%S', sourceActor.GetCombatName(true)).replace('%P', sourceActor.GetCombatPronoun('possessive'));
}

function onCast(sourceActor, targetActors) {
    var rounds = chanting(WAIT_ROUNDS + 1);
    SendUserMessage(sourceActor.UserId(), 'You begin to chant, and the foe\'s life shows red in the air.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S begins to chant, and a red thread spins from the foe.') + rounds, sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    var rounds = chanting(sourceActor.ChantRoundsLeft());
    SendUserMessage(sourceActor.UserId(), 'You keep chanting. The thread pulls taut.' + rounds);
    SendRoomMessage(sourceActor.GetRoomId(), fill(sourceActor, '%S keeps chanting. The thread pulls taut.') + rounds, sourceActor.UserId());
}

function onMagic(sourceActor, targetActors) {
    var roomId = sourceActor.GetRoomId();
    var sourceUserId = sourceActor.UserId();
    var strikes = Math.max(1, Math.min(sourceActor.ClassEffect('siphon'), 2));
    var used = [];
    for (var i = 0; i < targetActors.length && i < strikes; i++) {
        var foe = targetActors[i];
        var pct = i == 0 ? 100 : 60;
        var roll = Math.max(1, Math.floor(sourceActor.SpellPower(SPELL_ID) * sourceActor.SpellFactor(foe) * pct / 100));
        var dealt = -foe.AddHealth(-roll);
        var ally = sourceActor.MostHurtAlly(false);
        var suffix = ' (' + dealt + ' damage';
        // Phase 38c3: a Necromancer's Deeper drain heals more than it takes,
        // and Grave Chill hobbles the first foe.
        var drain = sourceActor.ClassEffect('drainpct');
        var give = drain > 0 ? Math.floor(dealt * drain / 100) : dealt;
        if (i == 0 && dealt > 0 && sourceActor.ClassEffect('gravechill') > 0) {
            foe.GiveStatus(1108, 1);
            suffix += ', hobbled';
        }
        if (ally != null && dealt > 0) {
            var healed = ally.AddHealth(give);
            suffix += ', ' + healed + ' healed';
            var over = give - healed;
            if (over > 0 && sourceActor.ClassEffect('bloodward') > 0) {
                var cap = Math.max(1, Math.floor(sourceActor.SpellPower('ward') / 2));
                if (ally.GrantWard(cap, 1)) { suffix += ', ward ' + cap; }
            }
            SendUserMessage(ally.UserId(), 'The stolen life flows into you.' + suffix + ')');
        }
        suffix += ')';
        var foeName = foe.GetCombatName(false);
        SendUserMessage(sourceUserId, 'You pull the life out of ' + foeName + '.' + suffix);
        SendRoomMessage(roomId, fill(sourceActor, '%S pulls the life out of ') + foeName + '.' + suffix, sourceUserId, foe.UserId());
        SendUserMessage(foe.UserId(), fill(sourceActor, '%S pulls the life out of you.') + suffix);
    }
}
