package parties

import "slices"

type Party struct {
	LeaderUserId  int
	UserIds       []int
	InviteUserIds []int
	AutoAttackers []int
	Position      map[int]string
	Followers     []int
	Supporters    []int
	followTokens  map[int]uint64
	autoTokens    map[int]uint64
}

var (
	followSequence uint64
	partyMap       = map[int]*Party{} // key is leader user id, value is party
)

func New(userId int) *Party {
	storageErr = nil
	if userId <= 0 {
		return nil
	}
	undo := checkpoint()
	if _, ok := partyMap[userId]; ok {
		return nil
	}
	p := &Party{
		LeaderUserId:  userId,
		UserIds:       []int{userId},
		InviteUserIds: []int{},
		AutoAttackers: []int{},
		Position:      map[int]string{},
	}
	partyMap[userId] = p
	if !persist(undo) {
		return nil
	}
	return p
}

func Get(userId int) *Party {
	if party, ok := partyMap[userId]; ok {
		return party
	}
	return nil
}

func (p *Party) ChanceToBeTargetted(userId int) int {
	// Legacy ranks are display-only: independently formed companies have
	// equal initial targeting weight. Formation governs interception.
	return 1
}

func (p *Party) GetRank(userId int) string {
	if val, ok := p.Position[userId]; ok {
		return val
	}
	return `middle`
}

func (p *Party) SetRank(userId int, rank string) {
	if !p.IsMember(userId) {
		return
	}
	if rank == `front` || rank == `back` {
		p.Position[userId] = rank
		return
	}

	delete(p.Position, userId)
}

func (p *Party) IsLeader(userId int) bool {
	return p.LeaderUserId == userId
}

func (p *Party) New(userId int) *Party {
	if party, ok := partyMap[userId]; ok {
		return party
	}
	return nil
}

func (p *Party) SetAutoAttack(userId int, on bool) bool {
	if !p.IsMember(userId) {
		return false
	}

	if on {
		for _, id := range p.AutoAttackers {
			if id == userId {
				return true
			}
		}
		p.AutoAttackers = append(p.AutoAttackers, userId)
		if p.autoTokens == nil {
			p.autoTokens = map[int]uint64{}
		}
		followSequence++
		p.autoTokens[userId] = followSequence
		return false
	}

	for i, id := range p.AutoAttackers {
		if id == userId {
			p.AutoAttackers = append(p.AutoAttackers[:i], p.AutoAttackers[i+1:]...)
			delete(p.autoTokens, userId)
			return true
		}
	}
	return false
}

func (p *Party) GetAutoAttackUserIds() []int {
	var result []int
	for _, id := range p.AutoAttackers {
		if p.IsMember(id) {
			result = append(result, id)
		}
	}
	return result
}

func (p *Party) Leave(userId int) bool {
	storageErr = nil
	undo := checkpoint()
	if p.Invited(userId) {
		return p.DeclineInvite(userId)
	}
	if !p.IsMember(userId) {
		return false
	}
	if p.IsLeader(userId) && len(p.UserIds) == 1 {
		return p.TryDisband()
	}
	p.SetAutoAttack(userId, false)
	p.SetFollow(userId, false)
	p.SetSupport(userId, false)
	delete(p.Position, userId)
	if p.IsLeader(userId) {
		if len(p.UserIds) == 1 {
			return p.TryDisband()
		}

		for _, id := range p.UserIds {
			if id != userId {
				p.setLeader(id)
				break
			}
		}
	}

	for i, id := range p.UserIds {
		if id == userId {
			p.UserIds = append(p.UserIds[:i], p.UserIds[i+1:]...)
			break
		}
	}

	delete(partyMap, userId)

	return persist(undo)
}

func (p *Party) IsMember(userId int) bool {
	for _, id := range p.UserIds {
		if id == userId {
			return true
		}
	}
	return false
}

func (p *Party) Invited(userId int) bool {
	for _, id := range p.InviteUserIds {
		if id == userId {
			return true
		}
	}
	return false
}

func (p *Party) InvitePlayer(userId int) bool {
	if _, ok := partyMap[userId]; ok {
		return false
	}
	p.InviteUserIds = append(p.InviteUserIds, userId)
	partyMap[userId] = p

	return true
}

func (p *Party) AcceptInvite(userId int) bool {
	storageErr = nil
	undo := checkpoint()
	if !p.Invited(userId) {
		return false
	}

	p.UserIds = append(p.UserIds, userId)

	for idx, uid := range p.InviteUserIds {
		if uid == userId {
			p.InviteUserIds = append(p.InviteUserIds[:idx], p.InviteUserIds[idx+1:]...)
			break
		}
	}
	return persist(undo)
}

func (p *Party) DeclineInvite(userId int) bool {
	if !p.Invited(userId) {
		return false
	}

	for idx, uid := range p.InviteUserIds {
		if uid == userId {
			p.InviteUserIds = append(p.InviteUserIds[:idx], p.InviteUserIds[idx+1:]...)
			break
		}
	}

	delete(partyMap, userId)

	return true
}

func (p *Party) Disband() { p.TryDisband() }

func (p *Party) TryDisband() bool {
	storageErr = nil
	undo := checkpoint()
	p.AutoAttackers = nil
	p.autoTokens = nil
	p.Followers = nil
	p.followTokens = nil
	p.Supporters = nil
	for _, userId := range p.UserIds {
		delete(partyMap, userId)
	}
	for _, userId := range p.InviteUserIds {
		delete(partyMap, userId)
	}
	p.UserIds = nil
	p.InviteUserIds = nil
	return persist(undo)
}

func (p *Party) GetMembers() []int {
	return append([]int{}, p.UserIds...)
}

func (p *Party) GetInvited() []int {
	return append([]int{}, p.InviteUserIds...)
}

// Consent is runtime-only; accepted membership is durable. Joining grants no movement,
// combat or support authority. These methods run on the owning game loop.
func (p *Party) SetFollow(userId int, on bool) bool {
	if !p.IsMember(userId) {
		return false
	}
	wasOn := p.setConsent(&p.Followers, userId, on)
	if on && !wasOn {
		followSequence++
		if p.followTokens == nil {
			p.followTokens = map[int]uint64{}
		}
		p.followTokens[userId] = followSequence
	}
	if !on {
		delete(p.followTokens, userId)
	}
	return wasOn
}

func (p *Party) SetSupport(userId int, on bool) bool {
	return p.setConsent(&p.Supporters, userId, on)
}

func (p *Party) setConsent(ids *[]int, userId int, on bool) bool {
	if !p.IsMember(userId) {
		return false
	}
	wasOn := slices.Contains(*ids, userId)
	if on && !wasOn {
		*ids = append(*ids, userId)
	}
	if !on {
		*ids = slices.DeleteFunc(*ids, func(id int) bool { return id == userId })
	}
	return wasOn
}

func (p *Party) Follows(userId int) bool {
	return p.IsMember(userId) && !p.IsLeader(userId) && slices.Contains(p.Followers, userId)
}

func (p *Party) Supports(userId int) bool {
	return p.IsMember(userId) && slices.Contains(p.Supporters, userId)
}

// Promote validates membership and clears follow consent: consent to follow
// one leader is never inherited by their replacement.
func (p *Party) Promote(userId int) bool {
	storageErr = nil
	undo := checkpoint()
	if !p.setLeader(userId) {
		return false
	}
	return persist(undo)
}

func (p *Party) setLeader(userId int) bool {
	if !p.IsMember(userId) {
		return false
	}
	if p.LeaderUserId != userId {
		p.Followers = nil
		p.followTokens = nil
		p.AutoAttackers = nil
		p.autoTokens = nil
	}
	p.LeaderUserId = userId
	return true
}

// AlliedLeaders is the helpful-effect provider. Both owners must opt in;
// invitations and stale consent cannot broaden a company's target scope.
// The effect resolver checks room, life, company ownership and chant snapshots.
func AlliedLeaders(userId int) []int {
	p := Get(userId)
	if p == nil || !p.Supports(userId) {
		return nil
	}
	var result []int
	for _, id := range p.GetMembers() {
		if id != userId && p.Supports(id) {
			result = append(result, id)
		}
	}
	return result
}

// FollowToken identifies this specific consent, invalidating queued moves even
// if a player leaves/rejoins or turns following off and back on before execution.
func (p *Party) FollowToken(userId int) uint64 {
	if !p.Follows(userId) {
		return 0
	}
	return p.followTokens[userId]
}

// Suspend removes session-only consent without removing durable membership.
func Suspend(userId int) {
	if p := Get(userId); p != nil {
		if p.Invited(userId) {
			p.DeclineInvite(userId)
			return
		}
		p.SetFollow(userId, false)
		p.SetSupport(userId, false)
		p.SetAutoAttack(userId, false)
	}
}

func (p *Party) AutoAttackToken(userId int) uint64 {
	if !p.IsMember(userId) || !slices.Contains(p.AutoAttackers, userId) {
		return 0
	}
	return p.autoTokens[userId]
}
