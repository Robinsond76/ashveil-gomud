# Help for ~flee~

The ~flee~ command attempts to run away from combat into another area. In a
battle it is an emergency personal escape; ~retreat [exit]~ orders a company
withdrawal (see ~help retreat~). A
flight that gets away ends your battle at once.

If successful, foes will not chase you. You must get past every foe of your
battle's group that is attacking you; a group waiting its turn doesn't block
you. Able companions follow you out. Blocked living companions separate using
the saved return system, then rejoin when the battle ends for 5 loyalty.
If separation cannot be saved, the escape stops; already saved flights stay
pending until the battle ends. Dead companions still require resurrection.

Some hurts keep you from fleeing at all: being hobbled by a lash
(see ~help statuses~), hamstrung, tackled, or winded. While one lasts, ~flee~
is refused, and a flight you had already begun is held. No-go restrictions,
locks and room permissions also prevent escape.

The chance of success is **30 + (fleeSpeed / (fleeSpeed+attackerSpeed) * 70)**

## Usage:

  ~flee~