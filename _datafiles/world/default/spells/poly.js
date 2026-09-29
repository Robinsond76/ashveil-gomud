POLYMORPH_RACE_IDS = [5, 7, 14, 18];

POLYMORPH_BUFF_ID = 42;

/**
 * Called when the casting is initialized.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'You begin weaving threads of transmutation magic...');
    SendRoomMessage(sourceActor.GetRoomId(),
        sourceActor.GetCharacterName(true) + ' begins weaving threads of transmutation magic...',
        sourceActor.UserId());
    return true;
}

/**
 * Called each round while the spell is being cast.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to abort the cast.
 */
function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'The transmutation magic intensifies...');
    SendRoomMessage(sourceActor.GetRoomId(),
        sourceActor.GetCharacterName(true) + ' continues weaving transmutation magic...',
        sourceActor.UserId());
}

/**
 * Called when the spell succeeds its cast attempt.
 * @param {ActorObject} sourceActor - The actor casting the spell.
 * @param {ActorObject} targetActor - The target of the spell.
 * @returns {boolean} Return false to prevent default post-cast behavior.
 */
function onMagic(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();
    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);
    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    raceId = POLYMORPH_RACE_IDS[UtilDiceRoll(1, POLYMORPH_RACE_IDS.length) - 1];

    targetActor.ApplyFormChange(raceId);

    targetActor.GiveBuff(POLYMORPH_BUFF_ID, "spell");

    newRace = targetActor.GetRace();

    if (sourceUserId != targetUserId) {

        SendUserMessage(sourceUserId,
            'You unleash a bolt of transmutation magic at ' + targetName +
            ', transforming them into a <ansi fg="race">' + newRace + '</ansi>.');

        SendRoomMessage(roomId,
            sourceName + ' unleashes a bolt of transmutation magic at ' + targetName +
            ', transforming them.',
            sourceUserId, targetUserId);

        SendUserMessage(targetUserId,
            sourceName + ' hits you with a bolt of transmutation magic. ' +
            'Your body twists and reshapes - you are now a <ansi fg="race">' + newRace + '</ansi>.');

    } else {

        SendUserMessage(sourceUserId,
            'You unleash the transmutation magic on yourself. ' +
            'Your body twists and reshapes - you are now a <ansi fg="race">' + newRace + '</ansi>.');

        SendRoomMessage(roomId,
            sourceName + ' unleashes transmutation magic on themselves, transforming into a <ansi fg="race">' + newRace + '</ansi>.',
            sourceUserId);
    }
}
