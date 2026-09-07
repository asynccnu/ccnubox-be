package grpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeedServiceServer struct {
	feedv1.UnimplementedFeedServiceServer
	feedEventService       service.FeedEventService
	feedUserConfigService  service.FeedUserConfigService
	muxiOfficialMSGService service.MuxiOfficialMSGService
	pushService            service.PushService
	l                      logger.Logger
	metrics                *metricsx.FeedDeliveryMetrics
}

func NewFeedServiceServer(
	feedEventService service.FeedEventService,
	feedUserConfigService service.FeedUserConfigService,
	muxiOfficialMSGService service.MuxiOfficialMSGService,
	pushService service.PushService,
	metricSet *metricsx.Metrics,
	l logger.Logger,
) *FeedServiceServer {
	var feedMetrics *metricsx.FeedDeliveryMetrics
	if metricSet != nil {
		feedMetrics = metricSet.Feed
	}
	return &FeedServiceServer{
		feedEventService:       feedEventService,
		feedUserConfigService:  feedUserConfigService,
		muxiOfficialMSGService: muxiOfficialMSGService,
		pushService:            pushService,
		metrics:                feedMetrics,
		l:                      l,
	}
}

func (g *FeedServiceServer) GetFeedEvents(ctx context.Context, req *feedv1.GetFeedEventsReq) (*feedv1.GetFeedEventsResp, error) {
	if req == nil || strings.TrimSpace(req.GetStudentId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}
	feedEvents, fail, err := g.feedEventService.GetFeedEvents(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}

	//如果有失败消息的话就尝试进行消息推送
	if len(fail) > 0 {
		// 获取消息
		go func() {
			errs := g.pushService.PushMSGS(context.Background(), fail)
			if len(errs) > 0 {
				g.l.Info(
					fmt.Sprintf("原失败消息数量:%d,推送发生错误数量:%d,首条错误消息%s", len(fail), len(errs), errs[0].Err.Error()),
					logger.Error(errs[0].Err),
				)
			}
		}()
	}
	return &feedv1.GetFeedEventsResp{
		FeedEvents: convFeedEventsVOFromDomainToGRPC(feedEvents),
	}, nil
}

func (g *FeedServiceServer) ChangeFeedAllowList(ctx context.Context, req *feedv1.ChangeFeedAllowListReq) (*feedv1.ChangeFeedAllowListResp, error) {
	if req == nil || req.AllowList == nil {
		return nil, status.Error(codes.InvalidArgument, "allow_list is required")
	}
	if !tool.IsValidStudentID(req.AllowList.GetStudentId()) {
		return nil, status.Error(codes.InvalidArgument, "valid student_id is required")
	}
	err := g.feedUserConfigService.ChangeAllowList(ctx, convAllowListFromGRPCToDomain(req.AllowList))
	if err != nil {
		return nil, err
	}
	return &feedv1.ChangeFeedAllowListResp{}, nil
}

func (g *FeedServiceServer) FindOrCreateAllowList(ctx context.Context, req *feedv1.FindOrCreateAllowListReq) (*feedv1.FindOrCreateAllowListResp, error) {
	if req == nil || !tool.IsValidStudentID(req.GetStudentId()) {
		return nil, status.Error(codes.InvalidArgument, "valid student_id is required")
	}
	list, err := g.feedUserConfigService.FindOrCreateAllowList(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}
	return &feedv1.FindOrCreateAllowListResp{AllowList: convAllowListFromDomainToGRPC(&list)}, nil
}

func (g *FeedServiceServer) ClearFeedEvent(ctx context.Context, req *feedv1.ClearFeedEventReq) (*feedv1.ClearFeedEventResp, error) {
	if req == nil || strings.TrimSpace(req.GetStudentId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}
	err := g.feedEventService.ClearFeedEvent(ctx, req.GetStudentId(), req.GetFeedId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	return &feedv1.ClearFeedEventResp{}, nil
}

func (g *FeedServiceServer) ReadFeedEvent(ctx context.Context, req *feedv1.ReadFeedEventReq) (*feedv1.ReadFeedEventResp, error) {
	if req == nil || req.GetFeedId() <= 0 || strings.TrimSpace(req.GetStudentId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_id and student_id are required")
	}
	err := g.feedEventService.ReadFeedEvent(ctx, req.GetStudentId(), req.GetFeedId())
	if err != nil {
		return nil, err
	}
	return &feedv1.ReadFeedEventResp{}, nil
}

func (g *FeedServiceServer) SaveFeedToken(ctx context.Context, req *feedv1.SaveFeedTokenReq) (*feedv1.SaveFeedTokenResp, error) {
	err := g.feedUserConfigService.SaveFeedToken(ctx, req.GetStudentId(), req.GetToken())
	if err != nil {
		return nil, err
	}
	return &feedv1.SaveFeedTokenResp{}, nil
}

func (g *FeedServiceServer) RemoveFeedToken(ctx context.Context, req *feedv1.RemoveFeedTokenReq) (*feedv1.RemoveFeedTokenResp, error) {
	err := g.feedUserConfigService.RemoveFeedToken(ctx, req.GetStudentId(), req.GetToken())
	if err != nil {
		return nil, err
	}
	return &feedv1.RemoveFeedTokenResp{}, nil
}

func (g *FeedServiceServer) PublicMuxiOfficialMSG(ctx context.Context, req *feedv1.PublicMuxiOfficialMSGReq) (*feedv1.PublicMuxiOfficialMSGResp, error) {

	err := g.muxiOfficialMSGService.PublicMuxiOfficialMSG(ctx, convMuxiMSGFromGRPCTODomain(req.MuxiOfficialMSG))
	if err != nil {
		return nil, err
	}

	return &feedv1.PublicMuxiOfficialMSGResp{}, nil
}

func (g *FeedServiceServer) StopMuxiOfficialMSG(ctx context.Context, req *feedv1.StopMuxiOfficialMSGReq) (*feedv1.StopMuxiOfficialMSGResp, error) {
	err := g.muxiOfficialMSGService.StopMuxiOfficialMSG(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &feedv1.StopMuxiOfficialMSGResp{}, nil
}

func (g *FeedServiceServer) GetToBePublicOfficialMSG(ctx context.Context, req *feedv1.GetToBePublicOfficialMSGReq) (*feedv1.GetToBePublicOfficialMSGResp, error) {
	msgs, err := g.muxiOfficialMSGService.GetToBePublicOfficialMSG(ctx, false)
	if err != nil {
		return nil, err
	}

	resp := make([]*feedv1.MuxiOfficialMSG, len(msgs))
	for i := range msgs {
		resp[i] = convMuxiMSGFromDomainTOGRPC(&msgs[i])
	}

	return &feedv1.GetToBePublicOfficialMSGResp{MsgList: resp}, nil
}

// 微服务内部调用
func (g *FeedServiceServer) PublicFeedEvent(ctx context.Context, req *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
	if err := validatePublicFeedEventRequest(req); err != nil {
		g.recordLibraryPublish(req, "invalid")
		return nil, err
	}
	publishStatus, err := g.feedEventService.PublicFeedEvent(ctx, req.GetIsAll(), domainFeedEventFromRequest(req))
	if err != nil {
		g.recordLibraryPublish(req, "error")
		return nil, err
	}
	g.recordLibraryPublish(req, strings.ToLower(publishStatus.String()))
	return &feedv1.PublicFeedEventResp{Status: publishStatus}, nil
}

func (g *FeedServiceServer) recordLibraryPublish(req *feedv1.PublicFeedEventReq, result string) {
	if g.metrics == nil || req == nil || req.GetEvent() == nil || req.GetEvent().GetType() != feedv1.FeedEventType_LIBRARY {
		return
	}
	g.metrics.LibraryPublishTotal.WithLabelValues(result).Inc()
}

func (g *FeedServiceServer) ListLibraryPreferenceChanges(ctx context.Context, req *feedv1.ListLibraryPreferenceChangesReq) (*feedv1.ListLibraryPreferenceChangesResp, error) {
	if req == nil || req.GetAfterRevision() < 0 || req.GetLimit() < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid preference change cursor")
	}
	changes, next, err := g.feedUserConfigService.ListLibraryPreferenceChanges(ctx, req.GetAfterRevision(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	items := make([]*feedv1.LibraryPreferenceChange, len(changes))
	for i := range changes {
		items[i] = &feedv1.LibraryPreferenceChange{
			Revision:  changes[i].Revision,
			StudentId: changes[i].StudentID,
			Enabled:   changes[i].Enabled,
			ChangedAt: changes[i].ChangedAt,
		}
	}
	return &feedv1.ListLibraryPreferenceChangesResp{Changes: items, NextRevision: next}, nil
}

func (g *FeedServiceServer) ListLibraryReminderUsers(ctx context.Context, req *feedv1.ListLibraryReminderUsersReq) (*feedv1.ListLibraryReminderUsersResp, error) {
	if req == nil || req.GetAfterId() < 0 || req.GetLimit() < 0 || req.GetSnapshotRevision() < 0 || (req.GetAfterId() > 0 && req.GetSnapshotRevision() == 0) {
		return nil, status.Error(codes.InvalidArgument, "invalid library user cursor")
	}
	users, next, snapshotRevision, err := g.feedUserConfigService.ListLibraryReminderUsers(ctx, req.GetAfterId(), req.GetSnapshotRevision(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	items := make([]*feedv1.LibraryReminderUser, len(users))
	for i := range users {
		items[i] = &feedv1.LibraryReminderUser{Id: users[i].ID, StudentId: users[i].StudentID, Revision: users[i].Revision}
	}
	return &feedv1.ListLibraryReminderUsersResp{Users: items, NextId: next, SnapshotRevision: snapshotRevision}, nil
}

func domainFeedEventFromRequest(req *feedv1.PublicFeedEventReq) domain.FeedEvent {
	event := req.GetEvent()
	return domain.FeedEvent{
		ID:           event.GetId(),
		StudentId:    req.GetStudentId(),
		Type:         strings.ToLower(event.GetType().String()),
		Title:        event.GetTitle(),
		Content:      event.GetContent(),
		Url:          event.GetUrl(),
		ExtendFields: event.GetExtendFields(),
		CreatedAt:    event.GetCreatedAt(),
		DedupeKey:    event.GetDedupeKey(),
		Source:       event.GetSource(),
		OccurredAt:   event.GetOccurredAt(),
	}
}

func validatePublicFeedEventRequest(req *feedv1.PublicFeedEventReq) error {
	if req == nil || req.GetEvent() == nil {
		return status.Error(codes.InvalidArgument, "event is required")
	}
	studentID := strings.TrimSpace(req.GetStudentId())
	if !req.GetIsAll() && (studentID == "" || len(studentID) > 64) {
		return status.Error(codes.InvalidArgument, "valid student_id is required")
	}
	event := req.GetEvent()
	switch event.GetType() {
	case feedv1.FeedEventType_GRADE,
		feedv1.FeedEventType_HOLIDAY,
		feedv1.FeedEventType_MUXI,
		feedv1.FeedEventType_ENERGY,
		feedv1.FeedEventType_FEEDBACK,
		feedv1.FeedEventType_LIBRARY:
	default:
		return status.Error(codes.InvalidArgument, "unsupported feed event type")
	}
	if len(event.GetTitle()) > 1024 || len(event.GetContent()) > 8192 || len(event.GetUrl()) > domain.MaxFeedEventURLBytes {
		return status.Error(codes.InvalidArgument, "feed event field is too long")
	}
	if len(event.GetDedupeKey()) > 255 || len(event.GetSource()) > 64 || len(event.GetExtendFields()) > 32 {
		return status.Error(codes.InvalidArgument, "feed event metadata is too large")
	}
	for key, value := range event.GetExtendFields() {
		lowerKey := strings.ToLower(key)
		if len(key) > 64 || len(value) > 1024 || strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "jwt") ||
			strings.Contains(lowerKey, "hmac") || strings.Contains(lowerKey, "secret") {
			return status.Error(codes.InvalidArgument, "invalid extend_fields")
		}
	}
	if event.GetType() == feedv1.FeedEventType_LIBRARY {
		if req.GetIsAll() || event.GetDedupeKey() == "" || event.GetSource() != "library" || event.GetOccurredAt() <= 0 {
			return status.Error(codes.InvalidArgument, "library event requires dedupe_key, source=library and occurred_at")
		}
		allowedFields := map[string]struct{}{
			"notification_type": {},
			"reservation_id":    {},
			"seat_id":           {},
			"seat_label":        {},
			"location":          {},
			"start_at":          {},
			"end_at":            {},
			"target_at":         {},
			"episode_version":   {},
			"deep_link":         {},
		}
		for key := range event.GetExtendFields() {
			if _, allowed := allowedFields[key]; !allowed {
				return status.Error(codes.InvalidArgument, "unsupported library extend_field")
			}
		}
	}
	return nil
}

func (g *FeedServiceServer) Register(server grpc.ServiceRegistrar) {
	feedv1.RegisterFeedServiceServer(server, g)
}
