package usercommands

// Ashveil Phase 72a: the looks and life story steps of character creation.
//
// The steps run inside a prompt (start, creation, appearance edit or
// lifestory choose), asking one numbered question at a time. Answers are held
// in the prompt's state until the player confirms the summary, so a redo
// changes nothing and nothing is granted twice. Each question is also
// published to internal/creation so the web client's creation panel can show
// the same step; the panel answers with the same input a telnet player types.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// creationMode says which command is running the steps.
type creationMode string

const (
	creationModeNew    creationMode = `new`    // start: looks, then life story
	creationModeLegacy creationMode = `legacy` // creation: what an existing character is missing, skippable
	creationModeEdit   creationMode = `edit`   // appearance edit: looks only, at an inn
	creationModeStory  creationMode = `story`  // lifestory choose: life story only, once

	creationStateKey = `creation-state`
	// innTag is the room tag appearance edit needs. It is the camping
	// module's default inn tag.
	innTag = `inn`
)

type creationOutcome int

const (
	creationWaiting     creationOutcome = iota // a question is open; the command returns now
	creationDone                               // the steps committed
	creationSkipped                            // an existing character put the steps off
	creationUnavailable                        // nothing to ask (no data, or nothing missing)
)

// saveUserFn persists a user once the steps commit outside `start` (which
// the normal save covers). Tests replace it so they never write user files.
var saveUserFn = func(u *users.UserRecord) error { return users.SaveUser(*u) }

type creationState struct {
	Opened bool
	Looks  appearance.Looks
	Story  lifestory.Picks
}

func creationStateOf(p *prompt.Prompt) *creationState {
	if v, ok := p.Recall(creationStateKey); ok {
		if st, ok := v.(*creationState); ok {
			return st
		}
	}
	st := &creationState{Looks: appearance.Looks{}, Story: lifestory.Picks{}}
	p.Store(creationStateKey, st)
	return st
}

type askStatus int

const (
	askPending askStatus = iota
	askPicked
	askBack
	askSkip
)

// creationRun is one pass through the steps (the command re-runs on every
// answer, picking up from the prompt's state).
type creationRun struct {
	p       *prompt.Prompt
	user    *users.UserRecord
	mode    creationMode
	app     *appearance.Data
	story   *lifestory.Data
	doLooks bool
	doStory bool
	st      *creationState
	total   int
}

// runCreation asks the next unanswered question, or commits once the
// summary is confirmed. Unavailable means there is nothing to do.
func runCreation(p *prompt.Prompt, user *users.UserRecord, mode creationMode) creationOutcome {
	r := &creationRun{p: p, user: user, mode: mode, app: appearance.Current(), story: lifestory.Current()}
	c := user.Character
	switch mode {
	case creationModeNew:
		if c.CreationOffered() { // `start` runs again on every later answer
			return creationUnavailable
		}
		r.doLooks, r.doStory = r.app != nil, r.story != nil
	case creationModeLegacy:
		r.doLooks, r.doStory = r.app != nil && !c.HasLooks(), r.story != nil && !c.HasLifeStory()
	case creationModeEdit:
		r.doLooks = r.app != nil
	case creationModeStory:
		r.doStory = r.story != nil && !c.HasLifeStory()
	}
	if !r.doLooks && !r.doStory {
		return creationUnavailable
	}
	r.st = creationStateOf(p)
	r.total = r.questionCount()

	if r.mode == creationModeLegacy && !r.st.Opened {
		out, waiting := r.opener()
		if waiting {
			return creationWaiting
		}
		if out != creationWaiting {
			return out
		}
	}

	for {
		if r.doLooks && !r.looksComplete() {
			switch r.looksStep() {
			case askPending:
				return creationWaiting
			case askSkip:
				return r.skip()
			}
			continue
		}
		if r.doStory && !r.storyComplete() {
			switch r.storyStep() {
			case askPending:
				return creationWaiting
			case askSkip:
				return r.skip()
			}
			continue
		}
		switch r.summaryStep() {
		case askPending:
			return creationWaiting
		case askSkip:
			return r.skip()
		case askBack:
			continue
		}
		// A commit that fails clears the picks that failed; asking again
		// starts that step over.
		if out := r.commit(); out != creationWaiting {
			return out
		}
	}
}

// questionCount is how many questions the steps ask, for "n of N".
func (r *creationRun) questionCount() int {
	n := 0
	if r.doLooks {
		n += len(appearance.SingleTraits) + 2 // marks and the free line
	}
	if r.doStory {
		n += len(lifestory.Stages)
	}
	return n
}

func (r *creationRun) skip() creationOutcome {
	if r.mode != creationModeLegacy {
		// appearance edit and lifestory choose: leave with nothing changed.
		r.user.SendText(`You leave things as they were.`)
		r.finish()
		return creationSkipped
	}
	r.user.Character.MarkCreationOffered()
	r.user.SendText(`You put it off. Type <ansi fg="command">appearance edit</ansi> at an inn to describe yourself, or <ansi fg="command">lifestory choose</ansi> to write your story.`)
	r.finish()
	return creationSkipped
}

// leaving reports an answer that puts the steps off (an existing
// character's skip) or cancels appearance edit or lifestory choose. A new
// character must finish them.
func (r *creationRun) leaving(response string) bool {
	if r.mode == creationModeNew {
		return false
	}
	return strings.EqualFold(response, `skip`) || strings.EqualFold(response, `cancel`)
}

func (r *creationRun) finish() {
	// `start` goes on to ask the tutorial question in the same prompt, so
	// only the stand-alone commands end it.
	if r.mode != creationModeNew {
		r.user.ClearPrompt()
	}
	creation.Clear(r.user.UserId)
}

// opener is the first question an existing character is asked.
func (r *creationRun) opener() (creationOutcome, bool) {
	const title = `Your character has no looks or life story yet. Write them now, or later?`
	opts := []creation.Option{{ID: `now`, Name: `now`}, {ID: `later`, Name: `later`}}
	// No prompt options: the web panel answers with the option's number, as
	// it does for every other question, and matchCreationOption reads it.
	show := func() {
		v := r.view(creation.View{Step: `looks`, Key: `opener`, Kind: creation.KindConfirm, Title: title, Options: opts})
		r.user.SendText(r.menuText(v))
	}
	q := r.p.Ask(title, []string{}, `now`)
	if !q.Done {
		r.user.SendText(``)
		r.user.SendText(`  Ashveil now lets you describe your character and choose a life story. You can do it now, or put it off and use <ansi fg="command">appearance edit</ansi> at an inn later.`)
		show()
		return creationWaiting, true
	}
	answer := strings.TrimSpace(q.Response)
	r.p.Forget(title)
	if r.leaving(answer) {
		answer = `later`
	}
	o, ok := matchCreationOption(opts, answer)
	if !ok {
		r.user.SendText(`That isn't one of the options.`)
		r.p.Ask(title, []string{}, `now`)
		show()
		return creationWaiting, true
	}
	r.st.Opened = true
	if o.ID == `later` {
		return r.skip(), false
	}
	return creationWaiting, false
}

// ---------------------------------------------------------------- asking

func (r *creationRun) view(v creation.View) creation.View {
	v.Mode = string(r.mode)
	v.Total = r.total
	v.Picks = r.picksView()
	if r.doLooks && r.app != nil {
		p, _ := r.pronouns()
		v.Preview = r.app.Compose(r.st.Looks, p)
		v.Skin, v.Hair = r.app.Colors(r.st.Looks)
	}
	if lineage, ok := archetypes.PlayerArchetype(r.user.UserId); ok {
		v.Lineage = lineage
	}
	if r.doStory && r.story != nil && r.storyComplete() {
		_, sp := r.pronouns()
		v.Backstory = r.story.Backstory(r.st.Story, r.user.Character.Name, sp)
	}
	v.CanSkip = r.mode != creationModeNew
	creation.Publish(r.user.UserId, v)
	return v
}

func (r *creationRun) picksView() map[string]string {
	out := map[string]string{}
	for k, v := range r.st.Looks {
		if v != `` {
			out[k] = v
		}
	}
	for k, v := range r.st.Story {
		if v != `` {
			out[`story.`+k] = v
		}
	}
	return out
}

// pronouns are the forms for the pronouns picked so far (race default
// before the first answer).
func (r *creationRun) pronouns() (appearance.Pronouns, lifestory.Pronouns) {
	pick := r.st.Looks[appearance.TraitPronouns]
	if pick == `` {
		return r.user.Character.LooksPronouns()
	}
	f := characters.PronounFormsFor(pick)
	return appearance.Pronouns{Subject: f.Subject, Object: f.Object, Possessive: f.Possessive},
		lifestory.Pronouns{Subject: f.Subject, Object: f.Object, Possessive: f.Possessive}
}

func matchCreationOption(options []creation.Option, response string) (creation.Option, bool) {
	response = strings.TrimSpace(response)
	if response == `` {
		return creation.Option{}, false
	}
	if n, err := strconv.Atoi(response); err == nil {
		if n >= 1 && n <= len(options) {
			return options[n-1], true
		}
		return creation.Option{}, false
	}
	for _, o := range options {
		if strings.EqualFold(o.ID, response) || strings.EqualFold(o.Name, response) {
			return o, true
		}
	}
	return creation.Option{}, false
}

func (r *creationRun) menuText(v creation.View) string {
	var b strings.Builder
	b.WriteString("\n")
	if v.Number > 0 && v.Total > 0 {
		b.WriteString(fmt.Sprintf("  <ansi fg=\"black-bold\">(%d of %d)</ansi> ", v.Number, v.Total))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(fmt.Sprintf("<ansi fg=\"white-bold\">%s</ansi>\n", v.Title))
	for i, o := range v.Options {
		b.WriteString(fmt.Sprintf("  <ansi fg=\"yellow-bold\">%2d)</ansi> <ansi fg=\"white-bold\">%s</ansi>", i+1, o.Name))
		if len(o.Stats) > 0 {
			b.WriteString(fmt.Sprintf(" <ansi fg=\"black-bold\">(+1 %s)</ansi>", strings.Join(o.Stats, ` or `)))
		}
		b.WriteString("\n")
		if o.Text != `` {
			b.WriteString("      " + o.Text + "\n")
		}
	}
	hints := []string{`Answer with a number or a name.`}
	if v.CanBack {
		hints = append(hints, `<ansi fg="command">back</ansi> starts this part over.`)
	}
	if v.CanSkip {
		hints = append(hints, r.leaveHint())
	}
	b.WriteString("  " + strings.Join(hints, ` `) + "\n")
	return b.String()
}

// leaveHint names the answer that leaves the steps in this mode.
func (r *creationRun) leaveHint() string {
	if r.mode == creationModeLegacy {
		return `<ansi fg="command">skip</ansi> puts it all off.`
	}
	return `<ansi fg="command">cancel</ansi> leaves things as they were.`
}

// choose asks one numbered question. It reports pending while waiting, and
// the picked option's id once answered. back and skip answers are returned
// for the caller to act on.
func (r *creationRun) choose(step, key, title string, number int, opts []creation.Option, canBack bool) (string, askStatus) {
	show := func() {
		v := r.view(creation.View{Step: step, Key: key, Kind: creation.KindChoice, Title: title,
			Number: number, Options: opts, CanBack: canBack})
		r.user.SendText(r.menuText(v))
	}
	q := r.p.Ask(title, []string{})
	if !q.Done {
		show()
		return ``, askPending
	}
	response := strings.TrimSpace(q.Response)
	r.p.Forget(title)
	if strings.EqualFold(response, `back`) && canBack {
		return ``, askBack
	}
	if r.leaving(response) {
		return ``, askSkip
	}
	if o, ok := matchCreationOption(opts, response); ok {
		return o.ID, askPicked
	}
	r.user.SendText(`That isn't one of the options.`)
	r.p.Ask(title, []string{})
	show()
	return ``, askPending
}

// ---------------------------------------------------------------- looks

func creationOptions(options []appearance.Option) []creation.Option {
	out := make([]creation.Option, 0, len(options))
	for _, o := range options {
		out = append(out, creation.Option{ID: o.ID, Name: o.Name, Color: o.Color})
	}
	return out
}

func (r *creationRun) looksComplete() bool {
	for _, id := range appearance.SingleTraits {
		if r.st.Looks[id] == `` {
			return false
		}
	}
	if r.st.Looks[appearance.TraitMark] == `` {
		return false
	}
	if r.needsSecondMark() && r.st.Looks[appearance.KeyMark2] == `` {
		return false
	}
	_, asked := r.st.Looks[appearance.KeyLine]
	return asked
}

func (r *creationRun) needsSecondMark() bool {
	m := r.st.Looks[appearance.TraitMark]
	return m != `` && m != appearance.NoMark && r.app.Marks() >= 2
}

func (r *creationRun) looksStep() askStatus {
	race := r.user.Character.RaceKey()
	n := 0
	for _, id := range appearance.SingleTraits {
		n++
		if r.st.Looks[id] != `` {
			continue
		}
		return r.lookPick(id, id, n, r.app.Options(id, race))
	}
	n++
	if r.st.Looks[appearance.TraitMark] == `` {
		return r.lookPick(appearance.TraitMark, appearance.TraitMark, n, r.app.Options(appearance.TraitMark, race))
	}
	if r.needsSecondMark() && r.st.Looks[appearance.KeyMark2] == `` {
		first := r.st.Looks[appearance.TraitMark]
		var rest []appearance.Option
		for _, o := range r.app.Options(appearance.TraitMark, race) {
			if o.ID != first {
				rest = append(rest, o)
			}
		}
		return r.lookPick(appearance.KeyMark2, appearance.TraitMark, n, rest)
	}
	return r.lineQuestion(n + 1)
}

func (r *creationRun) lookPick(key, traitID string, number int, options []appearance.Option) askStatus {
	title := r.app.Trait(traitID).Title
	if key == appearance.KeyMark2 {
		title = `Do you carry a second mark?`
	}
	id, status := r.choose(`looks`, key, title, number, creationOptions(options), len(r.st.Looks) > 0)
	switch status {
	case askBack:
		r.st.Looks = appearance.Looks{}
		return askBack
	case askPicked:
		r.st.Looks[key] = id
		return askPicked
	}
	return status
}

func (r *creationRun) lineQuestion(number int) askStatus {
	title := fmt.Sprintf(`Add a line of your own (up to %d characters), or press Enter for none.`, r.app.Line())
	q := r.p.Ask(title, []string{}, `none`)
	if !q.Done {
		r.view(creation.View{Step: `looks`, Key: appearance.KeyLine, Kind: creation.KindText, Title: title, Number: number, CanBack: true})
		r.user.SendText(``)
		hint := ` <ansi fg="command">back</ansi> starts your looks over.`
		if r.mode != creationModeNew {
			hint += ` ` + r.leaveHint()
		}
		r.user.SendText(`  <ansi fg="black-bold">(` + fmt.Sprint(number) + ` of ` + fmt.Sprint(r.total) + `)</ansi> This line follows the description others read when they look at you.` + hint)
		return askPending
	}
	response := strings.TrimSpace(q.Response)
	r.p.Forget(title)
	if strings.EqualFold(response, `back`) {
		r.st.Looks = appearance.Looks{}
		return askBack
	}
	if r.leaving(response) {
		return askSkip
	}
	if strings.EqualFold(response, `none`) {
		r.st.Looks[appearance.KeyLine] = ``
		return askPicked
	}
	line, err := appearance.CleanLine(response, r.app.Line(), configs.GetConfig().IsBannedName)
	if err != nil {
		r.user.SendText(`That line isn't allowed: ` + err.Error() + `.`)
		r.p.Ask(title, []string{}, `none`)
		r.view(creation.View{Step: `looks`, Key: appearance.KeyLine, Kind: creation.KindText, Title: title, Number: number, CanBack: true})
		return askPending
	}
	r.st.Looks[appearance.KeyLine] = line
	return askPicked
}

// ---------------------------------------------------------------- story

func (r *creationRun) storyComplete() bool {
	for _, id := range lifestory.Stages {
		o, ok := r.story.Option(id, r.st.Story[id])
		if !ok {
			return false
		}
		if r.st.Story[lifestory.StatKey(id)] == `` && len(lifestory.Offered(o, r.st.Story)) > 1 {
			return false
		}
	}
	return true
}

func (r *creationRun) storyStep() askStatus {
	base := 0
	if r.doLooks {
		base = len(appearance.SingleTraits) + 2
	}
	for i, id := range lifestory.Stages {
		stage := r.story.Stage(id)
		picked, havePick := r.story.Option(id, r.st.Story[id])
		if !havePick {
			opts := make([]creation.Option, 0, len(stage.Options))
			_, sp := r.pronouns()
			for _, o := range stage.Options {
				opts = append(opts, creation.Option{
					ID: o.ID, Name: o.Name,
					Text:  storyPreview(r.story, id, o.ID, r.user.Character.Name, sp),
					Stats: lifestory.Offered(o, r.st.Story),
				})
			}
			optID, status := r.choose(`story`, id, stage.Title, base+i+1, opts, len(r.st.Story) > 0)
			switch status {
			case askBack:
				r.st.Story = lifestory.Picks{}
				return askBack
			case askPicked:
				r.st.Story[id] = optID
				// A single stat that can still take the +1 is taken at once.
				chosen, _ := r.story.Option(id, optID)
				if offered := lifestory.Offered(chosen, r.st.Story); len(offered) == 1 {
					r.st.Story[lifestory.StatKey(id)] = offered[0]
				}
				return askPicked
			}
			return status
		}
		if r.st.Story[lifestory.StatKey(id)] == `` {
			offered := lifestory.Offered(picked, r.st.Story)
			if len(offered) > 1 {
				opts := make([]creation.Option, 0, len(offered))
				for _, st := range offered {
					opts = append(opts, creation.Option{ID: st, Name: st})
				}
				title := fmt.Sprintf(`%s: which does it sharpen?`, picked.Name)
				statID, status := r.choose(`story`, lifestory.StatKey(id), title, base+i+1, opts, true)
				switch status {
				case askBack:
					r.st.Story = lifestory.Picks{}
					return askBack
				case askPicked:
					r.st.Story[lifestory.StatKey(id)] = statID
					return askPicked
				}
				return status
			}
		}
	}
	return askPicked
}

func storyPreview(d *lifestory.Data, stage, option, name string, p lifestory.Pronouns) string {
	return d.Backstory(lifestory.Picks{stage: option}, name, p)
}

// ---------------------------------------------------------------- summary

func (r *creationRun) summaryStep() askStatus {
	const title = `Keep this character as described?`
	opts := []creation.Option{{ID: `confirm`, Name: `confirm`}}
	if r.doLooks {
		opts = append(opts, creation.Option{ID: `looks`, Name: `redo looks`})
	}
	if r.doStory {
		opts = append(opts, creation.Option{ID: `story`, Name: `redo story`})
	}
	q := r.p.Ask(title, []string{})
	if !q.Done {
		v := r.view(creation.View{Step: `summary`, Key: `summary`, Kind: creation.KindConfirm, Title: title, Options: opts})
		r.user.SendText(r.summaryText())
		r.user.SendText(r.menuText(v))
		return askPending
	}
	response := strings.TrimSpace(q.Response)
	r.p.Forget(title)
	if r.leaving(response) {
		return askSkip
	}
	o, ok := matchCreationOption(opts, response)
	if !ok {
		r.user.SendText(`That isn't one of the options.`)
		r.p.Ask(title, []string{})
		v := r.view(creation.View{Step: `summary`, Key: `summary`, Kind: creation.KindConfirm, Title: title, Options: opts})
		r.user.SendText(r.menuText(v))
		return askPending
	}
	switch o.ID {
	case `looks`:
		r.st.Looks = appearance.Looks{}
		return askBack
	case `story`:
		r.st.Story = lifestory.Picks{}
		return askBack
	}
	return askPicked
}

func (r *creationRun) summaryText() string {
	var b strings.Builder
	b.WriteString("\n")
	if r.doLooks {
		ap, _ := r.pronouns()
		b.WriteString(`  <ansi fg="white-bold">You look like this:</ansi>` + "\n")
		b.WriteString("  " + r.app.Compose(r.st.Looks, ap) + "\n\n")
	}
	if r.doStory {
		_, sp := r.pronouns()
		b.WriteString(`  <ansi fg="white-bold">Your story:</ansi>` + "\n")
		b.WriteString("  " + r.story.Backstory(r.st.Story, r.user.Character.Name, sp) + "\n")
		b.WriteString("  " + lifeStoryEffectsLine(r.story, r.st.Story) + "\n")
	}
	return b.String()
}

// lifeStoryEffectsLine says what a set of picks gives: stats, the trade's
// keepsake and skill.
func lifeStoryEffectsLine(d *lifestory.Data, picks lifestory.Picks) string {
	parts := []string{}
	bonus := lifestory.StatBonus(picks)
	for _, st := range lifestory.StatNames {
		if bonus[st] > 0 {
			parts = append(parts, fmt.Sprintf(`+%d %s`, bonus[st], st))
		}
	}
	if trade, ok := d.Trade(picks); ok {
		if trade.Keepsake > 0 {
			if spec := items.GetItemSpec(trade.Keepsake); spec != nil {
				parts = append(parts, `a keepsake: `+spec.Name)
			}
		}
		if trade.Skill != `` {
			name := trade.Skill
			if info := skills.GetSkill(trade.Skill); info != nil {
				name = info.Name
			}
			parts = append(parts, name+` familiarity`)
		}
	}
	if len(parts) == 0 {
		return `<ansi fg="black-bold">It gives nothing but the telling.</ansi>`
	}
	return `<ansi fg="black-bold">It gives you:</ansi> ` + strings.Join(parts, `, `) + `.`
}

// ---------------------------------------------------------------- commit

func (r *creationRun) commit() creationOutcome {
	c := r.user.Character
	if r.doLooks {
		if err := c.SetLooks(r.app, r.st.Looks); err != nil {
			mudlog.Warn("creation", "looks", err.Error(), "userId", r.user.UserId)
			r.user.SendText(`Those looks can't be used (` + err.Error() + `). Let's choose them again.`)
			r.st.Looks = appearance.Looks{}
			return creationWaiting
		}
	}
	keepsake := ``
	if r.doStory {
		k, err := c.ApplyLifeStory(r.story, r.st.Story)
		if err != nil {
			mudlog.Warn("creation", "story", err.Error(), "userId", r.user.UserId)
			r.user.SendText(`That story can't be used (` + err.Error() + `). Let's choose it again.`)
			r.st.Story = lifestory.Picks{}
			return creationWaiting
		}
		keepsake = k
		c.RecalculateStats()
	}
	c.MarkCreationOffered()

	if r.doStory {
		line := `Your story is written.`
		if keepsake != `` {
			line += ` A ` + keepsake + ` goes into your pack.`
		}
		r.user.SendText(line)
	} else {
		r.user.SendText(`You look as you have chosen.`)
	}
	if r.mode != creationModeNew {
		if err := saveUserFn(r.user); err != nil {
			mudlog.Warn("creation", "save", err.Error(), "userId", r.user.UserId)
		}
	}
	events.AddToQueue(events.PlayerChanged{UserId: r.user.UserId})
	r.finish()
	return creationDone
}

// startCreationStep runs the steps inside `start`: it reports true while
// a question is open (Start returns), false once done or when there is
// nothing to ask.
func startCreationStep(cmdPrompt *prompt.Prompt, user *users.UserRecord) bool {
	return runCreation(cmdPrompt, user, creationModeNew) == creationWaiting
}
