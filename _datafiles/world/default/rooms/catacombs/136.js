

const nouns = ["sarcophagus", "tomb"];
const verbs = ["touch", "push", "hit", "kick", "open", "pry"];


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

    SendUserMessage(user.UserId(), "The room begins to tremble, and a trap door opens beneath your feet! You fall into the room below!");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" has triggered some sort of trap! the room begins to tremble and a trap door opens beneath your feet. You fall into the darkness below.", user.UserId());


    players = room.GetPlayers();
    for( var i in players ) {
        SendRoomMessage(138, players[i].GetCharacterName(true)+" falls into the room from above.", players[i].UserId());
        players[i].MoveRoom(138);
    }
    
    return true;
}


      