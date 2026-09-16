// Package company contains the durable company and companion domain model.
package company

import "errors"

var (
	ErrInvalidLeader           = errors.New("invalid leader user ID")
	ErrInvalidTemplate         = errors.New("invalid mob template ID")
	ErrTemplateNotAllowed      = errors.New("mob template is not allowed")
	ErrCompanionAlreadyPresent = errors.New("leader already has a companion")
)

type Companion struct {
	MobTemplateID int `yaml:"mob_template_id"`
}

type Record struct {
	LeaderUserID int       `yaml:"leader_user_id"`
	Companion    Companion `yaml:"companion"`
}

type Registry struct {
	Companies map[int]Record `yaml:"companies"`
}

func NewRegistry() *Registry {
	return &Registry{Companies: make(map[int]Record)}
}

func (r *Registry) Get(leaderUserID int) (Record, bool) {
	if r == nil {
		return Record{}, false
	}
	record, ok := r.Companies[leaderUserID]
	return record, ok
}

func (r *Registry) Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}) error {
	if leaderUserID <= 0 {
		return ErrInvalidLeader
	}
	if mobTemplateID <= 0 {
		return ErrInvalidTemplate
	}
	if _, ok := allowed[mobTemplateID]; !ok {
		return ErrTemplateNotAllowed
	}
	if _, ok := r.Companies[leaderUserID]; ok {
		return ErrCompanionAlreadyPresent
	}
	if r.Companies == nil {
		r.Companies = make(map[int]Record)
	}
	r.Companies[leaderUserID] = Record{
		LeaderUserID: leaderUserID,
		Companion:    Companion{MobTemplateID: mobTemplateID},
	}
	return nil
}

func (r *Registry) Dismiss(leaderUserID int) bool {
	if r == nil {
		return false
	}
	if _, ok := r.Companies[leaderUserID]; !ok {
		return false
	}
	delete(r.Companies, leaderUserID)
	return true
}
