package service

import (
	"context"
	"strings"
	"sync"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

type pushService struct {
	pushClient        jpush.PushClient // 用于推送的客户端
	userFeedConfigDAO dao.FeedUserConfigDAO
	feedTokenDAO      dao.FeedTokenDAO
}

type PushService interface {
	GetPushCID(context.Context) (string, error)
	PreparePush(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, error)
	PreparePushForDelivery(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, string, error)
	PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error
	PushMSG(ctx context.Context, pushData *domain.FeedEvent) error
	PushMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, cid string) error
	PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData
}

// PreparedPush 保存已通过 Token 和推送开关检查的接收目标。
type PreparedPush struct {
	tokens []string
}

type ErrWithData struct {
	FeedEvent *domain.FeedEvent `json:"feed_event"`
	Err       error             `json:"err"`
}

func NewPushService(
	pushClient jpush.PushClient,
	userFeedConfigDAO dao.FeedUserConfigDAO,
	feedTokenDAO dao.FeedTokenDAO,
) PushService {
	return &pushService{
		pushClient:        pushClient,
		userFeedConfigDAO: userFeedConfigDAO,
		feedTokenDAO:      feedTokenDAO,
	}
}

func (s *pushService) PushMSGS(ctx context.Context, pushDatas []domain.FeedEvent) []ErrWithData {
	errs := make([]ErrWithData, 0)
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护 errs 切片的并发安全

	for _, pushData := range pushDatas {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(data domain.FeedEvent) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := s.PushMSG(ctx, &data)
			if err != nil {
				mu.Lock()
				errs = append(errs, ErrWithData{
					FeedEvent: &data,
					Err:       err,
				})
				mu.Unlock()
			}
		}(pushData)
	}
	wg.Wait()

	return errs
}

func (s *pushService) GetPushCID(ctx context.Context) (string, error) {
	cid, err := s.pushClient.GetCID(ctx)
	if err != nil {
		return "", errorx.Errorf("service: get jpush cid failed: %w", err)
	}
	return cid, nil
}

// 推送单条消息
func (s *pushService) PushMSG(ctx context.Context, pushData *domain.FeedEvent) error {
	return s.PushMSGWithCID(ctx, pushData, "")
}

func (s *pushService) PushMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, cid string) error {
	prepared, err := s.PreparePush(ctx, pushData)
	if err != nil || prepared == nil {
		return err
	}
	return s.PushPreparedMSGWithCID(ctx, pushData, prepared, cid)
}

func (s *pushService) PreparePush(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, error) {
	prepared, _, err := s.PreparePushForDelivery(ctx, pushData)
	return prepared, err
}

func (s *pushService) PreparePushForDelivery(ctx context.Context, pushData *domain.FeedEvent) (*PreparedPush, string, error) {
	if !tool.IsValidStudentID(pushData.StudentId) {
		return nil, "suppressed_no_target_or_disabled", nil
	}
	config, err := s.userFeedConfigDAO.GetPushConfig(ctx, pushData.StudentId)
	if err != nil {
		return nil, "", errorx.Errorf("service: get push config failed, sid: %s, err: %w", pushData.StudentId, err)
	}
	if !pushAllowed(pushData.Type, config) {
		return nil, suppressionReason(pushData.Type), nil
	}
	tokens, err := s.feedTokenDAO.GetTokens(ctx, pushData.StudentId)
	if err != nil {
		return nil, "", errorx.Errorf("service: get tokens failed for push, sid: %s, err: %w", pushData.StudentId, err)
	}
	if len(tokens) == 0 {
		return nil, "suppressed_no_target_or_disabled", nil
	}
	return &PreparedPush{tokens: tokens}, "", nil
}

func suppressionReason(label string) string {
	if strings.EqualFold(label, "library") {
		return "suppressed_by_allow_list"
	}
	return "suppressed_no_target_or_disabled"
}

func pushAllowed(label string, config dao.PushConfigSnapshot) bool {
	pos, exists := configMap[label]
	if !exists || config.Deleted {
		return false
	}
	if !config.Exists {
		if strings.EqualFold(label, "library") {
			return false
		}
		config.Config = model.DefaultPushConfig
	}
	return config.Config&(1<<pos) != 0
}

func (s *pushService) PushPreparedMSGWithCID(ctx context.Context, pushData *domain.FeedEvent, prepared *PreparedPush, cid string) error {
	if prepared == nil || len(prepared.tokens) == 0 {
		return nil
	}
	err := s.pushClient.Push(ctx, prepared.tokens, jpush.PushData{
		ContentType: pushData.Type,
		Extras:      pushData.ExtendFields,
		MsgContent:  pushData.Content,
		Title:       pushData.Title,
		Cid:         cid,
	})
	if err != nil {
		return errorx.Errorf("service: jpush client call failed, sid: %s, tokens_count: %d, err: %w", pushData.StudentId, len(prepared.tokens), err)
	}
	return nil
}
