
const verbs = ["touch", "push", "press", "take", "rub", "polish"];
const nouns = ["raven", "eyes", "bird"];

/**
 * Called when a user issues a command in the room.
 * @param {string} cmd - The command issued.
 * @param {string} rest - The arguments following the command.
 * @param {ActorObject} user - The user issuing the command.
 * @param {RoomObject} room - The room where the command was issued.
 * @returns {boolean} Return true if the command was handled.
 */
function onCommand(cmd, rest, user, room) {

    if ( !verbs.includes(cmd) ) {
        return false;
    }
    
    matches = UtilFindMatchIn(rest, nouns);
    if ( !matches.found ) {
        return false;
    }

    // Ashveil Phase 33h3: a move the player asks for waits until the fight is over.
    if ( user.InBattle() ) {
        SendUserMessage(user.UserId(), "Not while your company is fighting.");
        return true;
    }

    SendUserMessage(user.UserId(), "You press the eyes of the raven, and follow a secret entrance to the west!");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" presses in the eyes of the raven, and falls through into a room to the west!", user.UserId());

    user.GetParty().GiveQuest("2-investigate");
    user.MoveRoom(31);
        
    return true;
}

