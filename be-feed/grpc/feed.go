package grpc

import (
	"context"
	"errors"
	"fmt"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type FeedServiceServer struct {
	feedv1.UnimplementedFeedServiceServer
	feedEventService       service.FeedEventService
	feedUserConfigService  service.FeedUserConfigService
	muxiOfficialMSGService service.MuxiOfficialMSGService
	pushService            service.PushService
	l                      logger.Logger
}

func NewFeedServiceServer(
	feedEventService service.FeedEventService,
	feedUserConfigService service.FeedUserConfigService,
	muxiOfficialMSGService service.MuxiOfficialMSGService,
	pushService service.PushService,
	l logger.Logger,
) *FeedServiceServer {
	return &FeedServiceServer{
		feedEventService:       feedEventService,
		feedUserConfigService:  feedUserConfigService,
		muxiOfficialMSGService: muxiOfficialMSGService,
		pushService:            pushService,
		l:                      l,
	}
}

func (g *FeedServiceServer) GetFeedEvents(ctx context.Context, req *feedv1.GetFeedEventsReq) (*feedv1.GetFeedEventsResp, error) {
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
					logger.FormatLog("push", errors.New("消息推送发生错误"))...,
				)
			}
		}()
	}
	return &feedv1.GetFeedEventsResp{
		FeedEvents: convFeedEventsVOFromDomainToGRPC(feedEvents),
	}, nil
}

func (g *FeedServiceServer) ChangeFeedAllowList(ctx context.Context, req *feedv1.ChangeFeedAllowListReq) (*feedv1.ChangeFeedAllowListResp, error) {
	err := g.feedUserConfigService.ChangeAllowList(ctx, convAllowListFromGRPCToDomain(req.AllowList))
	if err != nil {
		return nil, err
	}
	return &feedv1.ChangeFeedAllowListResp{}, nil
}

func (g *FeedServiceServer) GetFeedAllowList(ctx context.Context, req *feedv1.GetFeedAllowListReq) (*feedv1.GetFeedAllowListResp, error) {
	list, err := g.feedUserConfigService.GetFeedAllowList(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}
	return &feedv1.GetFeedAllowListResp{AllowList: convAllowListFromDomainToGRPC(&list)}, nil
}

func (g *FeedServiceServer) ClearFeedEvent(ctx context.Context, req *feedv1.ClearFeedEventReq) (*feedv1.ClearFeedEventResp, error) {
	err := g.feedEventService.ClearFeedEvent(ctx, req.GetStudentId(), req.GetFeedId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	return &feedv1.ClearFeedEventResp{}, nil
}

func (g *FeedServiceServer) ReadFeedEvent(ctx context.Context, req *feedv1.ReadFeedEventReq) (*feedv1.ReadFeedEventResp, error) {
	err := g.feedEventService.ReadFeedEvent(ctx, req.GetFeedId())
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
	msgs, err := g.muxiOfficialMSGService.GetToBePublicOfficialMSG(ctx)
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
	go func() {
		//此处进行异步,为什么异步呢,主要是内部调用也有上下文取消时间这在推送给所有人的时候将会非常致命
		ctx = context.Background()
		feedEvent := domain.FeedEvent{
			StudentId:    req.GetStudentId(),
			Type:         req.GetEvent().GetType(),
			Title:        req.GetEvent().GetTitle(),
			Content:      req.GetEvent().GetContent(),
			ExtendFields: req.GetEvent().GetExtendFields(),
		}

		err := g.feedEventService.PublicFeedEvent(ctx, req.GetIsAll(), feedEvent)
		if err != nil {
			g.l.Error("推送失败", logger.FormatLog("push", err)...)
		}
		return
	}()

	return &feedv1.PublicFeedEventResp{}, nil
}

func (g *FeedServiceServer) Register(server *grpc.Server) {
	feedv1.RegisterFeedServiceServer(server, g)
}
