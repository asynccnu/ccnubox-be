package server

import (
	"strings"
	"testing"

	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
)

func TestInvitationRequestLogsAreRedacted(t *testing.T) {
	for _, req := range []any{
		&libraryv1.NotifyTeamInvitationRequest{OperatorStudentId: "operator-private", TeamId: "123456", StudentIds: []string{"recipient-private"}},
		&feedv1.PublicFeedEventReq{StudentId: "recipient-private", Event: &feedv1.FeedEvent{Type: feedv1.FeedEventType_LIBRARY, ExtendFields: map[string]string{"notification_type": "TEAM_INVITATION", "team_id": "123456"}}},
	} {
		value := requestLogValue(req)
		if !strings.HasPrefix(value, "[REDACTED:") || strings.Contains(value, "private") || strings.Contains(value, "123456") {
			t.Fatalf("unsafe request log: %s", value)
		}
	}
	if requestLogValue("ordinary") != "ordinary" {
		t.Fatal("ordinary request logging changed")
	}
}
