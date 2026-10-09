package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	commontool "github.com/asynccnu/ccnubox-be/common/tool"
)

type InvitationTeam struct {
	ReminderTeam
	OperatorStudentID string
	IsMasterUser      bool
	Members           map[string]int
}

// 邀请采用独立严格模型，不改变组队成功扫描的最小契约。
func (c *ReminderHTTPClient) GetCurrentTeamForInvitation(ctx context.Context, token string) (*InvitationTeam, error) {
	var team *InvitationTeam
	err := c.getCurrentTeamResponse(ctx, token, func(raw json.RawMessage) error {
		var err error
		team, err = decodeInvitationTeam(raw)
		return err
	})
	return team, err
}

func decodeInvitationTeam(raw json.RawMessage) (*InvitationTeam, error) {
	invalid := func() (*InvitationTeam, error) {
		return nil, fmt.Errorf("%w: invalid invitation team data", ErrUpstreamStateUnknown)
	}
	team, err := decodeReminderTeam(raw)
	if err != nil || team == nil {
		return nil, err
	}
	if !teamNumericID.MatchString(team.ID) || team.ExpirationTime == nil {
		return invalid()
	}
	var data struct {
		Username     *string         `json:"username"`
		IsMasterUser *bool           `json:"isMasterUser"`
		Members      json.RawMessage `json:"userInfoList"`
	}
	if json.Unmarshal(raw, &data) != nil || data.Username == nil || !commontool.IsValidStudentID(*data.Username) || data.IsMasterUser == nil {
		return invalid()
	}
	membersRaw := bytes.TrimSpace(data.Members)
	if len(membersRaw) == 0 || membersRaw[0] != '[' {
		return invalid()
	}
	var members []struct {
		Username *string `json:"username"`
		Status   *int    `json:"status"`
	}
	if json.Unmarshal(membersRaw, &members) != nil {
		return invalid()
	}
	result := &InvitationTeam{ReminderTeam: *team, OperatorStudentID: *data.Username, IsMasterUser: *data.IsMasterUser, Members: map[string]int{}}
	for _, member := range members {
		if member.Username == nil || !commontool.IsValidStudentID(*member.Username) || member.Status == nil {
			return invalid()
		}
		if previous, exists := result.Members[*member.Username]; exists && previous != *member.Status {
			return invalid()
		}
		result.Members[*member.Username] = *member.Status
	}
	return result, nil
}
