
const lantern = ["lantern", "light"];
const verbs = ["touch", "fix", "pull", "light", "take", "get", "move", "adjust", "turn", "rub", "clean", "dust", "wipe", "polish", "repair", "break", "smash", "ignite", "light"];


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

    matches = UtilFindMatchIn(rest, lantern);
    if ( !matches.found ) {
        return false;
    }

    // Ashveil Phase 33h3: a move the player asks for waits until the fight is over.
    if ( user.InBattle() ) {
        SendUserMessage(user.UserId(), "Not while your company is fighting.");
        return true;
    }

    SendUserMessage(user.UserId(), "You adjust the latern, and a passage way reveals itself in the wall. Just as you step through the passage, it closes behind you.");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" does something with the lantern, and a secret passage opens! He disappears through the passage just as it closes.", user.UserId());
    user.MoveRoom(50);

    return true;
}



