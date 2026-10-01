# Skills

## Data files (data-driven)

Skills and professions are loaded from YAML datafiles, not hardcoded:

* `_datafiles/world/default/skills/<skillid>.yaml` — one file per skill (`skillid`, `name`, `description`, `maxlevel`). The filename is the raw `skillid` (e.g. `dual-wield.yaml`).
* `_datafiles/world/default/professions/<professionid>.yaml` — one file per profession (`professionid`, `name`, `description`, `skills: [...]`). The filename runs the spaced id through `util.ConvertForFilename` (e.g. `treasure hunter` → `treasure_hunter.yaml`).

Skills are keyed by lowercase string ids matching `Character.Skills map[string]int`, so no save migration was needed. `maxlevel` defaults to 4 when unset.

Loaded at boot in `main.go` (`skills.LoadDataFiles()` then `skills.LoadProfessionDataFiles()` — skills first so profession cross-references can warn). Hot-reload in-game with `reload skills`. Manage via the admin pages (`/admin/skills`, `/admin/skills-professions`), the CRUD API (`/admin/api/v1/skills`, `/admin/api/v1/professions`), and the API docs at `/admin/skills-api`.

Validation policy is **reject writes, zero reads**: `SetSkill`/`TrainSkill` no-op (with a mudlog warning) for unknown skill ids; `GetSkillLevel` returns 0 for unknown skills, so orphaned save entries stay inert. Deleting a skill that a profession references is rejected.

Skills are referenced everywhere as plain lowercase string ids (e.g. `"cast"`, `"dual-wield"`); there are no compiled-in skill-name constants. The YAML files are the source of truth for which skills exist at runtime.

TODO: Improve this documentation. For now, documenting philosophy/approach to skill implementation

## Training

Skills can be acquired via any source, but generally can be expected to be learned at TRAINING CENTERS

## Levels

SKills range from level 1 to 4, with increasing cost at each level:
* Level 1 - 1 Skillpoint
* Level 2 - 2 Skillpoints
* Level 3 - 3 Skillpoints
* Level 4 - 4 Skillpoints

The effect of this is:
* To level a single skill to the maximum (Level 4) would require 10 Skillpoints, AKA 10 experience levels.
* Those same 10 skillpoints could be used to learn 10 different unique skills at level 1
* Players can go `deep and narrow` or `shallow and wide`

## Philosophy

Goal of skills design should be as follows:

* Skill Levels (1-4) determines CAPABILITIES - actual verbs or features of the skill. The motivation to levelling up a skill should be increase of abilities.
  * Exception: Level MAY also modify some things stats cannot effect, such as spell level effecting spell proficiency learning speed, or mappable area.
* Stat points (0-200+) determines effectiveness - success rates, degree/duration of effect, etc.
  * Exception: Sometimes the SCALE of a skills effect or capability may be best determined by skill level, such as how many tamed creatures one can have.
  * Equipment can modify stat points, but cannot modify skill levels
* Players who level an individual skill to level 4 shouldn't become somehow greatly more powerful, but instead get new capabilities that their stat points will drive
* Synergies (skills that combine with other skills) should ideally occur at the highest level of one of the skills (level 4). 
  * For example, a ranger's `track` skill at Level 4 reads the trail two rooms out (Ashveil 33f2).
