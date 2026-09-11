package service

import (
	"context"

	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
)

type LibraryPreferenceChange struct {
	Revision  int64
	StudentID string
	Enabled   bool
	ChangedAt int64
}

type LibraryReminderUser struct {
	ID        int64
	StudentID string
	Revision  int64
}

type PublishResult int

const (
	PublishAccepted PublishResult = iota
	PublishDuplicate
	PublishSuppressed
)

type FeedGateway interface {
	PreferenceChanges(context.Context, int64, int32) ([]LibraryPreferenceChange, int64, error)
	ReminderUsers(context.Context, int64, int64, int32) ([]LibraryReminderUser, int64, int64, error)
	Publish(context.Context, string, *feedv1.FeedEvent) (PublishResult, error)
}

type grpcFeedGateway struct{ client feedv1.FeedServiceClient }

func NewFeedGateway(client feedv1.FeedServiceClient) FeedGateway {
	return &grpcFeedGateway{client: client}
}

func (g *grpcFeedGateway) PreferenceChanges(ctx context.Context, after int64, limit int32) ([]LibraryPreferenceChange, int64, error) {
	resp, err := g.client.ListLibraryPreferenceChanges(ctx, &feedv1.ListLibraryPreferenceChangesReq{AfterRevision: after, Limit: limit})
	if err != nil {
		return nil, after, err
	}
	result := make([]LibraryPreferenceChange, 0, len(resp.GetChanges()))
	for _, row := range resp.GetChanges() {
		if row == nil {
			continue
		}
		result = append(result, LibraryPreferenceChange{Revision: row.GetRevision(), StudentID: row.GetStudentId(), Enabled: row.GetEnabled(), ChangedAt: row.GetChangedAt()})
	}
	return result, resp.GetNextRevision(), nil
}

func (g *grpcFeedGateway) ReminderUsers(ctx context.Context, after, snapshotRevision int64, limit int32) ([]LibraryReminderUser, int64, int64, error) {
	resp, err := g.client.ListLibraryReminderUsers(ctx, &feedv1.ListLibraryReminderUsersReq{AfterId: after, Limit: limit, SnapshotRevision: snapshotRevision})
	if err != nil {
		return nil, after, snapshotRevision, err
	}
	result := make([]LibraryReminderUser, 0, len(resp.GetUsers()))
	for _, row := range resp.GetUsers() {
		if row == nil {
			continue
		}
		result = append(result, LibraryReminderUser{ID: row.GetId(), StudentID: row.GetStudentId(), Revision: row.GetRevision()})
	}
	return result, resp.GetNextId(), resp.GetSnapshotRevision(), nil
}

func (g *grpcFeedGateway) Publish(ctx context.Context, studentID string, event *feedv1.FeedEvent) (PublishResult, error) {
	resp, err := g.client.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{StudentId: studentID, Event: event})
	if err != nil {
		return PublishAccepted, err
	}
	switch resp.GetStatus() {
	case feedv1.PublishStatus_DUPLICATE:
		return PublishDuplicate, nil
	case feedv1.PublishStatus_SUPPRESSED_BY_ALLOW_LIST:
		return PublishSuppressed, nil
	default:
		return PublishAccepted, nil
	}
}
