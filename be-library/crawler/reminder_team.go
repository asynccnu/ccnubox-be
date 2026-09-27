package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/tool"
)

var teamNumericID = regexp.MustCompile(`^[0-9]+$`)

type ReminderTeam struct {
	ID             string
	Status         int
	OnDate         string
	Total          int
	ExpirationTime *time.Time
}

// GetCurrentTeam 使用讨论间裸 Token，不进入座位签名/HMAC 路径。
func (c *ReminderHTTPClient) GetCurrentTeam(parent context.Context, token string) (*ReminderTeam, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%w: empty token", ErrUpstreamStateUnknown)
	}
	ctx, cancel := context.WithTimeout(parent, c.requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+reminderTeamPath, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: create team request", ErrUpstreamStateUnknown)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	raw, err := c.doResponse(req, true)
	if err != nil {
		return nil, err
	}
	return decodeReminderTeam(raw)
}

func decodeReminderTeam(raw json.RawMessage) (*ReminderTeam, error) {
	invalid := func() (*ReminderTeam, error) {
		return nil, fmt.Errorf("%w: invalid current team data", ErrUpstreamStateUnknown)
	}
	data := bytes.TrimSpace(raw)
	if bytes.Equal(data, []byte(`""`)) {
		return nil, nil
	}
	if len(data) == 0 || data[0] != '{' {
		return invalid()
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || len(fields) == 0 {
		return invalid()
	}
	idRaw, okID := fields["id"]
	statusRaw, okStatus := fields["status"]
	if !okID || !okStatus || bytes.Equal(bytes.TrimSpace(statusRaw), []byte("null")) {
		return invalid()
	}
	id, err := stringOrNumber(idRaw)
	if err != nil || strings.TrimSpace(id) == "" || len(id) > 128 || (len(idRaw) > 0 && idRaw[0] != '"' && !teamNumericID.MatchString(id)) {
		return invalid()
	}
	var status int
	if json.Unmarshal(statusRaw, &status) != nil || status < -3 || status > 2 {
		return invalid()
	}
	team := &ReminderTeam{ID: id, Status: status}
	if value, exists := fields["onDate"]; exists {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &team.OnDate) != nil {
			return invalid()
		}
		if team.OnDate != "" {
			date, err := time.ParseInLocation("2006-01-02", team.OnDate, tool.GetLocation())
			if err != nil || date.Format("2006-01-02") != team.OnDate {
				return invalid()
			}
		}
	}
	if value, exists := fields["total"]; exists {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &team.Total) != nil || team.Total < 0 {
			return invalid()
		}
	}
	if value, exists := fields["expirationTime"]; exists {
		var text string
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &text) != nil {
			return invalid()
		}
		if text != "" {
			parsed, err := time.ParseInLocation("2006-01-02 15:04:05", text, tool.GetLocation())
			if err != nil || parsed.Format("2006-01-02 15:04:05") != text {
				return invalid()
			}
			team.ExpirationTime = &parsed
		}
	}
	return team, nil
}
