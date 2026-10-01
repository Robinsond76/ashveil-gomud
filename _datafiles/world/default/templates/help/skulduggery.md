# Help for ~skulduggery~ (skill)

The ~skulduggery~ skill is actuall several skills you learn.

Each skill is improved by your stats and level as well.

## Usage:

(Lvl 1) ~sneak [direction/exit]~ Remain hidden for a period of time, even when moving between areas.  
(Lvl 2) ~bump [enemy]~ Bump into a player or NPC, causing a fraction of their coins to drop to the ground.  
(Lvl 3) ~backstab [enemy]~ Guarenteed critical on successful attack. A backstab doesn't start a fight with a creature, and can't be used once a battle is under way: start fights with ~attack [group]~. In a battle, any level of skulduggery strikes an opening on its own instead (~help abilities~).  
(Lvl 4) ~pickpocket [enemy]~ Gain ability to steal from players and NPC's while hidden.

Odds of success on a bump attempt: **attackStrength / 2 - max(0, targetLevel - attackerLevel)**  
On success up to 25% of their gold will drop to the floor.

Odds of success on a pickpocket attempt: **(attackSpeed+attackSmarts+attackPerception)/3 - defendPerception - max(0, targetLevel - attackerLevel)**  
You gain a +15% chance of success if you are sneaking at the time of a pickpocket attempt.  
On success you steal at least 25% of their money and 1 item.