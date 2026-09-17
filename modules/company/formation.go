package company

import (
	"fmt"
	"strconv"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const formationUsage = "Usage: formation | formation move <member> <row> <col> | formation swap <member-a> <member-b> | formation clear <member>"

var rowLabels = [domain.FormationRows]string{"front", "mid  ", "back "}

func (m *CompanyModule) renderFormation(leaderUserID int) string {
	record, _ := m.registry.Get(leaderUserID)
	lines := []string{"Company formation (3x3, row 1 = front):"}
	lines = append(lines, "         col 1        col 2        col 3")
	placed := map[domain.MemberKey]bool{}
	for r := 0; r < domain.FormationRows; r++ {
		cells := make([]string, 0, domain.FormationCols)
		for c := 0; c < domain.FormationCols; c++ {
			key := record.Formation.At(r, c)
			if key == "" {
				cells = append(cells, fmt.Sprintf("%-12s", "------"))
				continue
			}
			placed[key] = true
			cells = append(cells, fmt.Sprintf("%-12s", m.memberName(leaderUserID, key)))
		}
		lines = append(lines, fmt.Sprintf("%s [ %s ]", rowLabels[r], strings.Join(cells, " | ")))
	}

	unplaced := []string{}
	if !placed[domain.LeaderMemberKey] {
		unplaced = append(unplaced, "leader")
	}
	for _, companion := range record.Companions {
		if !placed[domain.CompanionMemberKey(companion.ID)] {
			unplaced = append(unplaced, fmt.Sprintf("%s (#%d)", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID))
		}
	}
	if len(unplaced) > 0 {
		lines = append(lines, "Unplaced: "+strings.Join(unplaced, ", "))
	}
	lines = append(lines, formationUsage)
	return strings.Join(lines, "\n")
}

func (m *CompanyModule) memberName(leaderUserID int, key domain.MemberKey) string {
	if key == domain.LeaderMemberKey {
		if user := users.GetByUserId(leaderUserID); user != nil {
			return user.Character.Name
		}
		return "leader"
	}
	record, _ := m.registry.Get(leaderUserID)
	for _, companion := range record.Companions {
		if domain.CompanionMemberKey(companion.ID) == key {
			return fmt.Sprintf("%s(#%d)", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID)
		}
	}
	return string(key)
}

func (m *CompanyModule) resolveMemberKey(leaderUserID int, selector string) (domain.MemberKey, error) {
	s := strings.ToLower(strings.TrimSpace(selector))
	if s == "" {
		return "", fmt.Errorf("company: a member is required")
	}
	if s == "leader" || s == "me" || s == "self" {
		return domain.LeaderMemberKey, nil
	}
	if strings.HasPrefix(s, "#") {
		id, err := strconv.Atoi(strings.TrimPrefix(s, "#"))
		if err != nil {
			return "", fmt.Errorf("company: invalid companion id %q", selector)
		}
		return domain.CompanionMemberKey(id), nil
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", fmt.Errorf("company: no company member matches %q", selector)
	}
	companion, ok := resolveCompanion(record, s)
	if !ok {
		return "", fmt.Errorf("company: no company member matches %q", selector)
	}
	return domain.CompanionMemberKey(companion.ID), nil
}

func parseSlot(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > domain.FormationCols {
		return 0, fmt.Errorf("company: slot %q must be a number 1-%d", raw, domain.FormationCols)
	}
	return n, nil
}

func (m *CompanyModule) formationCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	if err := m.persistenceAvailable(); err != nil {
		return true, err
	}
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) == 0 {
		user.SendText(m.renderFormation(user.UserId))
		return true, nil
	}
	switch args[0] {
	case "move":
		if len(args) != 4 {
			user.SendText(formationUsage)
			return true, nil
		}
		key, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		row, err := parseSlot(args[2])
		if err != nil {
			return true, err
		}
		col, err := parseSlot(args[3])
		if err != nil {
			return true, err
		}
		before, existed := m.registry.Get(user.UserId)
		if err := m.registry.PlaceMember(user.UserId, key, row-1, col-1); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			if existed {
				m.registry.Put(before)
			} else {
				m.registry.Put(domain.Record{LeaderUserID: user.UserId})
			}
			return true, err
		}
		user.SendText(fmt.Sprintf("Placed %s at row %d, column %d.", m.memberName(user.UserId, key), row, col))
	case "swap":
		if len(args) != 3 {
			user.SendText(formationUsage)
			return true, nil
		}
		a, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		b, err := m.resolveMemberKey(user.UserId, args[2])
		if err != nil {
			return true, err
		}
		before, _ := m.registry.Get(user.UserId)
		if err := m.registry.SwapMembers(user.UserId, a, b); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			m.registry.Put(before)
			return true, err
		}
		user.SendText("Formation positions swapped.")
	case "clear":
		if len(args) != 2 {
			user.SendText(formationUsage)
			return true, nil
		}
		key, err := m.resolveMemberKey(user.UserId, args[1])
		if err != nil {
			return true, err
		}
		before, _ := m.registry.Get(user.UserId)
		if err := m.registry.ClearMember(user.UserId, key); err != nil {
			return true, err
		}
		if err := m.save(); err != nil {
			m.registry.Put(before)
			return true, err
		}
		user.SendText(fmt.Sprintf("Removed %s from the formation.", m.memberName(user.UserId, key)))
	default:
		user.SendText(formationUsage)
	}
	return true, nil
}
