
const nouns = ["quest", "book", "books", "library", "spell", "spells", "worried", "worry"];

/**
 * Called when a user asks the mob a question.
 * @param {ActorObject} mob - The mob.
 * @param {RoomObject} room - The room the mob is in.
 * @param {AskEventDetails} eventDetails - Details about the ask event.
 * @returns {boolean} Return true if the event was handled.
 */
function onAsk(mob, room, eventDetails) {
    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    if ( user.HasQuest("6-end") ) {
        mob.Command("say Oh, it's nothing.");
        return true;
    }

    match = UtilFindMatchIn(eventDetails.askText, nouns);
    if ( match.found ) {

        mob.Command("emote thinks for a moment.");
        mob.Command("say I took a book called The History of Frostfang out with me weeks ago and can't remember where I left it.");
        mob.Command("say If you can find it for me, I'll teach you a useful spell.");

        user.GetParty().GiveQuest("6-start");

        return true;
    }

    return false;
}

/**
 * Called when a user gives the mob an item or gold.
 * @param {ActorObject} mob - The mob.
 * @param {RoomObject} room - The room the mob is in.
 * @param {GiveEventDetails} eventDetails - Details about the give event.
 * @returns {boolean} Return true if the event was handled.
 */
function onGive(mob, room, eventDetails) {

    if (eventDetails.sourceType == "mob") {
        return false;
    }

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }
    
    if ( eventDetails.gold > 0 ) {
        mob.Command("say I'll use this money to buy more books!");
        return true;
    }

    if (eventDetails.item) {
        if (eventDetails.item.ItemId != 10) {
            mob.Command("look !"+String(eventDetails.item.ItemId));
            mob.Command("drop !"+String(eventDetails.item.ItemId), UtilGetSecondsToTurns(5));
            return true;
        }
    }

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    if ( user.HasQuest("6-end") ) {
        // Illuminate is a wizard spell (Ashveil Phase 17). Someone who finished
        // the quest before becoming a wizard can come back and learn it.
        if ( !user.HasSpell("illum") && user.LearnSpell("illum") ) {
            mob.Command("say Ah, you've taken up the arcane since we last spoke. Let me show you <ansi fg=\"spell-helpful\">Illuminate</ansi> after all.");
            mob.Command("emote Shows you some useful gestures.");
            return true;
        }
        mob.Command("say Please don't borrow books from the library without permission!");
        return true;
    }

    if ( user.HasQuest("6-start") ) {

        user.GetParty().GiveQuest("6-end");

        mob.Command("say Thank you! It is such an interesting history. For example, Frostfang used to be called DragonsFang!");
        mob.Command("say I'll teach you the <ansi fg=\"spell-helpful\">Illuminate</ansi> spell. It's useful in dark places.");
        mob.Command("emote Shows you some useful gestures.");
        mob.Command("say Check your <ansi fg=\"command\">spellbook</ansi>.");

        user.GetParty().LearnSpell("illum");
        
        return true;
    }

}


/**
 * Called each round when the mob is idle.
 * @param {ActorObject} mob - The mob.
 * @param {RoomObject} room - The room the mob is in.
 * @returns {boolean} Return true if the event was handled.
 */
// Invoked once every round if mob is idle
function onIdle(mob, room) {

    noQuest = room.MissingQuest("6-start");
    if ( noQuest.length < 1 ) {
        return false;
    }

    round = UtilGetRoundNumber();

    action = round % 6;

    if ( action == 0 ) {
        mob.Command("say now where did I leave that book?");
        return true;
    }

    return false;
}
