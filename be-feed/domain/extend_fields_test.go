package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestExtendFieldsGetMissingKey(t *testing.T) {
	const key = "missing"

	value := ExtendFields{}.Get(key)
	if !errors.Is(value.Err, errKeyNotFound) {
		t.Fatalf("Get(%q) error = %v, want %v", key, value.Err, errKeyNotFound)
	}
	if !strings.Contains(value.Err.Error(), key) {
		t.Fatalf("Get(%q) error %q does not contain key", key, value.Err)
	}
}

func TestInvitationStorageValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*FeedEvent)
		valid  bool
	}{
		{"expired_history", func(e *FeedEvent) {}, true},
		{"missing_team", func(e *FeedEvent) { delete(e.ExtendFields, "team_id") }, false},
		{"bad_team", func(e *FeedEvent) { e.ExtendFields["team_id"] = "abc" }, false},
		{"missing_expiry", func(e *FeedEvent) { delete(e.ExtendFields, "expires_at") }, false},
		{"bad_expiry", func(e *FeedEvent) { e.ExtendFields["expires_at"] = "invalid" }, false},
		{"overflow_expiry", func(e *FeedEvent) { e.ExtendFields["expires_at"] = "9223372036854775808" }, false},
		{"equal_expiry", func(e *FeedEvent) { e.ExtendFields["expires_at"] = "1" }, false},
		{"unknown_field", func(e *FeedEvent) { e.ExtendFields["member_name"] = "sensitive" }, false},
		{"url", func(e *FeedEvent) { e.Url = "https://example.com" }, false},
		{"missing_source", func(e *FeedEvent) { e.Source = "" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := FeedEvent{StudentId: "A", Type: "library", Source: "library", DedupeKey: "key", OccurredAt: 1, ExtendFields: map[string]string{"notification_type": "TEAM_INVITATION", "team_id": "123", "expires_at": "2"}}
			tc.mutate(&e)
			if err := ValidateFeedEventForStorage(e); (err == nil) != tc.valid {
				t.Fatalf("err=%v valid=%v", err, tc.valid)
			}
		})
	}
}
