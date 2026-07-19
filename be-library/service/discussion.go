package service

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type DiscussionService interface {
	GetDiscussion(ctx context.Context, req *libraryv1.GetDiscussionRequest) (*libraryv1.GetDiscussionResponse, error)
	ReserveDiscussion(ctx context.Context, req *libraryv1.ReserveDiscussionRequest) (*libraryv1.ReserveDiscussionResponse, error)
}

type discussionService struct {
	crawler    *crawler.Crawler
	userClient userv1.UserServiceClient
	l          logger.Logger
}

var (
	ErrGetDiscussion = errorx.FormatErrorFunc(libraryv1.ErrorGetDiscussionError("获取研讨间失败"))
)

func NewDiscussionService(userClient userv1.UserServiceClient, libCrawler *crawler.Crawler, l logger.Logger) DiscussionService {
	return &discussionService{
		crawler:    libCrawler,
		userClient: userClient,
		l:          l,
	}
}

func (s *discussionService) GetDiscussion(ctx context.Context, req *libraryv1.GetDiscussionRequest) (*libraryv1.GetDiscussionResponse, error) {
	if req == nil || req.RoomType == "" || req.VenueId == "" || req.Date == "" {
		return nil, ErrGetDiscussion(errorx.New("room_type, venue_id and date are required"))
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return nil, ErrGetDiscussion(errorx.Errorf("invalid date %q", req.Date))
	}
	token, err := s.getDiscussionToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	ds, err := s.crawler.GetDiscussion(ctx, token, req.RoomType, req.VenueId, req.Date)
	if err != nil {
		return nil, ErrGetDiscussion(errorx.Errorf("get discussion failed, stuId: %s, err: %w", req.StuId, err))
	}
	resp := &libraryv1.GetDiscussionResponse{
		Discussions: make([]*libraryv1.Discussion, 0, len(ds)),
	}
	for _, d := range ds {
		if d == nil {
			continue
		}
		ts := make([]*libraryv1.DisableTime, 0, len(d.DisableList))
		for _, t := range d.DisableList {
			if t == nil {
				continue
			}
			ts = append(ts, &libraryv1.DisableTime{
				Start: t.Start,
				End:   t.End,
			})
		}
		resp.Discussions = append(resp.Discussions, &libraryv1.Discussion{
			RoomId:      d.RoomID,
			RoomType:    d.RoomType,
			Name:        d.Name,
			VenueId:     d.VenueID,
			Address:     d.Address,
			DisableList: ts,
		})
	}
	return resp, nil
}

func (s *discussionService) ReserveDiscussion(ctx context.Context, req *libraryv1.ReserveDiscussionRequest) (*libraryv1.ReserveDiscussionResponse, error) {
	if req == nil {
		return nil, ErrGetDiscussion(errorx.New("request is required"))
	}
	token, err := s.getDiscussionToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	msg, err := s.crawler.ReserveDiscussion(ctx, token, req.DevId, req.LabId, req.KindId, req.Title, req.Start, req.End, req.List)
	if err != nil {
		return nil, ErrGetDiscussion(errorx.Errorf("reserve discussion failed, stuId: %s, err: %w", req.StuId, err))
	}
	return &libraryv1.ReserveDiscussionResponse{Message: msg}, nil
}

func (s *discussionService) getDiscussionToken(ctx context.Context, studentID string) (string, error) {
	if studentID == "" {
		return "", ErrGetToken(errorx.New("student id is required"))
	}
	tokenResp, err := s.userClient.GetLibraryDiscussionToken(ctx, &userv1.GetLibraryTokenRequest{StudentId: studentID})
	if err != nil {
		return "", ErrGetToken(errorx.Errorf("get token failed, stuId: %s, err: %w", studentID, err))
	}
	if tokenResp == nil || tokenResp.Token == "" {
		return "", ErrGetToken(errorx.Errorf("empty token returned, stuId: %s", studentID))
	}
	return tokenResp.Token, nil
}
