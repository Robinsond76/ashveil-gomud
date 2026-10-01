
/**
 * Called when a user issues a command in the room.
 * @param {string} cmd - The command issued.
 * @param {string} rest - The arguments following the command.
 * @param {ActorObject} user - The user issuing the command.
 * @param {RoomObject} room - The room where the command was issued.
 * @returns {boolean} Return true if the command was handled.
 */
function onCommand(cmd, rest, user, room) {

    if ( rest.substr(rest.length - 2) != "ob" ) {
        return false;
    }

    if ( cmd == "look" ) {

        SendUserMessage(user.UserId(), "The obelisk crackles with subtle dark energy. It seems like it could be dangerous to the touch.");
        SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" looks at the obelisk.", user.UserId());
        
        return true;
    }

    if ( cmd == "touch" ) {
      
        SendUserMessage(user.UserId(), "You reach out and touch the obelisk.");
        SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" reaches out and touches the obelisk.", user.UserId());

        // Ashveil 33f1: the obelisk no longer teaches Portal.
        SendUserMessage(user.UserId(), "The stone is cold. Whatever power it held has long since faded.");
        
        return true;
    }
    
    return false;
}